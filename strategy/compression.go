package strategy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/hubertkaluzny/silly-trader/record"
	"github.com/hubertkaluzny/silly-trader/splicer"
)

type CompressionItem struct {
	Splice         splicer.Splice `json:"splice"`
	CompressedSize int            `json:"compressed_length"`
}

type CompressionModel struct {
	SpliceOptions splicer.SpliceOptions `json:"splice_options"`
	Items         []CompressionItem     `json:"items"`
}

func NewCompressionModel(spliceOpts splicer.SpliceOptions) *CompressionModel {
	return &CompressionModel{SpliceOptions: spliceOpts}
}

func LoadCompressionModelFromFile(file string) (*CompressionModel, error) {
	modelFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer modelFile.Close()
	reader, err := gzip.NewReader(modelFile)
	if err != nil {
		return nil, err
	}
	var model *CompressionModel
	decoder := json.NewDecoder(reader)
	err = decoder.Decode(model)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (model *CompressionModel) AddMarketData(data []record.Market) error {
	splices, err := splicer.SpliceData(data, model.SpliceOptions)
	if err != nil {
		return err
	}
	newItems := make([]CompressionItem, len(splices))
	for i, s := range splices {
		item, err := CompressSplice(s)
		if err != nil {
			return err
		}
		newItems[i] = *item
	}
	model.Items = append(model.Items, newItems...)
	return nil
}

func (model *CompressionModel) PredictResult(observation []record.Market) (*CompressionItem, error) {
	compressedObservation, err := CompressMarketData(observation)
	if err != nil {
		return nil, err
	}
	Cx1 := float64(len(compressedObservation))
	bestDist := math.MaxFloat32
	bestCandidate := -1
	for i, item := range model.Items {
		concatted := append(item.Splice.Data, observation...)
		compressedConcatted, err := CompressMarketData(concatted)
		if err != nil {
			return nil, err
		}
		Cx1x2 := float64(len(compressedConcatted))
		Cx2 := float64(item.CompressedSize)
		distance := (Cx1x2 - math.Min(Cx1, Cx2)) / math.Max(Cx1, Cx2)
		if distance < bestDist {
			bestDist = distance
			bestCandidate = i
		}
	}
	if bestCandidate == -1 {
		return nil, errors.New("no candidates found")
	}
	return &model.Items[bestCandidate], nil
}

func DistanceBetween(x1 CompressionItem, x2 CompressionItem) (float64, error) {
	Cx1 := float64(x1.CompressedSize)
	Cx2 := float64(x2.CompressedSize)

	concatted := append(x1.Splice.Data, x2.Splice.Data...)
	compressedConcatted, err := CompressMarketData(concatted)
	if err != nil {
		return math.MaxFloat64, err
	}

	Cx1x2 := float64(len(compressedConcatted))

	return (Cx1x2 - math.Min(Cx1, Cx2)) / math.Max(Cx1, Cx2), nil
}

func (model *CompressionModel) SimilarityMap() ([][]float64, error) {
	res := make([][]float64, len(model.Items))
	for i := range model.Items {
		res[i] = make([]float64, len(model.Items))
	}
	for i, itemI := range model.Items {
		for j, itemJ := range model.Items[i:] {
			canonicalJ := j + i
			distance, err := DistanceBetween(itemI, itemJ)
			if err != nil {
				return nil, err
			}
			res[i][canonicalJ] = distance
			res[canonicalJ][i] = distance
		}
	}
	return res, nil
}

func (model *CompressionModel) SizeResultBuckets() map[int][]float64 {
	buckets := make(map[int][]float64)
	for _, item := range model.Items {
		bucket := item.CompressedSize
		buckets[bucket] = append(buckets[bucket], item.Splice.Result)
	}
	return buckets
}

func (model *CompressionModel) SaveToFile(file string) error {
	modelFile, err := os.Create(file)
	defer modelFile.Close()
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(modelFile)
	encoder := json.NewEncoder(modelFile)
	err = encoder.Encode(model)
	if err != nil {
		return err
	}
	err = gzipWriter.Flush()
	return err
}

func CompressMarketData(data []record.Market) ([]byte, error) {
	var b strings.Builder
	for _, rec := range data {
		b.WriteString(fmt.Sprintf(`%.6f,%.6f,%.6f,%.6f,%6.f,%6.f-`,
			rec.Open,
			rec.High,
			rec.Low,
			rec.Close,
			rec.Volume,
			rec.VWAP))
	}

	var buffBytes []byte
	buff := bytes.NewBuffer(buffBytes)
	writer := gzip.NewWriter(buff)
	_, err := writer.Write([]byte(b.String()))
	if err != nil {
		return nil, err
	}
	err = writer.Flush()
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func CompressSplice(s splicer.Splice) (*CompressionItem, error) {
	compressed, err := CompressMarketData(s.Data)
	if err != nil {
		return nil, err
	}
	return &CompressionItem{
		Splice:         s,
		CompressedSize: len(compressed),
	}, nil
}
