package model

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/hubertkaluzny/silly-trader/record"
	"github.com/hubertkaluzny/silly-trader/splicer"
)

type Neighbour struct {
	Distance float64
	Item     CompressionItem
}

type CompressionEncodingType string

const (
	SimpleEncoding     CompressionEncodingType = "simple"
	ExpandedEncoding   CompressionEncodingType = "expanded"
	SFExpandedEncoding CompressionEncodingType = "expanded_sf"
	CharVarLength      CompressionEncodingType = "char_var"
)

type PredictionStrategy string

const (
	DiscreteWNN  PredictionStrategy = "wnn"
	ContinousWNN PredictionStrategy = "cwnn"
	TopDog       PredictionStrategy = "top"
)

type PredictionOpts struct {
	Strategy PredictionStrategy
	NearestN int
}

func ToCompressionEncodingType(input string) (CompressionEncodingType, error) {
	switch input {
	case string(SimpleEncoding):
		return SimpleEncoding, nil
	case string(ExpandedEncoding):
		return ExpandedEncoding, nil
	case string(SFExpandedEncoding):
		return SFExpandedEncoding, nil
	case string(CharVarLength):
		return CharVarLength, nil
	}
	return SimpleEncoding, errors.New("invalid encoding type specified")
}

type CompressionItem struct {
	Splice         splicer.Splice `json:"splice"`
	CompressedSize int            `json:"compressed_length"`
}

type CompressionModel struct {
	SpliceOptions     splicer.SpliceOptions   `json:"splice_options"`
	Items             []CompressionItem       `json:"items"`
	EncodingType      CompressionEncodingType `json:"encoding_type"`
	CachedDistanceMap [][]float64             `json:"distance_map"`
}

func NewCompressionModel(spliceOpts splicer.SpliceOptions, encodingType CompressionEncodingType) *CompressionModel {
	return &CompressionModel{SpliceOptions: spliceOpts, EncodingType: encodingType}
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
	var model CompressionModel
	decoder := json.NewDecoder(reader)
	err = decoder.Decode(&model)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (model *CompressionModel) AddMarketData(data []record.Market) error {
	splices, err := splicer.SpliceData(data, model.SpliceOptions)
	if err != nil {
		return err
	}
	newItems := make([]CompressionItem, len(splices))
	for i, s := range splices {
		item, err := CompressSplice(s, model.EncodingType)
		if err != nil {
			return err
		}
		newItems[i] = *item
	}
	model.Items = append(model.Items, newItems...)
	return nil
}

func (model *CompressionModel) PredictResults(observation []record.Market, opts PredictionOpts) (int, error) {
	results, err := model.GetClosestNeighbours(observation, opts.NearestN)
	if err != nil {
		return 0, err
	}

	// assuming item results are z-scores
	// weighted result by distance
	buyFreq := float64(0)
	sellFreq := float64(0)
	neitherFreq := float64(0)
	for _, res := range results {
		if res.Item.Splice.Result > 1 {
			buyFreq += 1 / res.Distance
		} else if res.Item.Splice.Result < 1 {
			sellFreq += 1 / res.Distance
		} else {
			neitherFreq += 1 / res.Distance
		}
	}

	if buyFreq > sellFreq && buyFreq > neitherFreq {
		return 1, nil
	} else if sellFreq > buyFreq && sellFreq > neitherFreq {
		return -1, nil
	} else {
		return 0, nil
	}
}

// GetClosestNeighbours expects data to come pre-normalised
func (model *CompressionModel) GetClosestNeighbours(observation []record.Market, nearestN int) ([]*Neighbour, error) {
	compressedObservation, err := CompressMarketData(observation, model.EncodingType)
	if err != nil {
		return nil, err
	}
	Cx1 := float64(len(compressedObservation))
	results := make([]*Neighbour, nearestN)
	for _, item := range model.Items {
		item := item
		combined, err := record.CombineInterleaved(item.Splice.Data, observation)
		if err != nil {
			return nil, err
		}
		compressedCombined, err := CompressMarketData(combined, model.EncodingType)
		if err != nil {
			return nil, err
		}
		Cx1x2 := float64(len(compressedCombined))
		Cx2 := float64(item.CompressedSize)
		distance := (Cx1x2 - math.Min(Cx1, Cx2)) / math.Max(Cx1, Cx2)

		insertIndex := -1
		for i, res := range results {
			i := i
			if res == nil {
				insertIndex = i
				break
			}
			if res.Distance > distance {
				insertIndex = i
				break
			}
		}
		if insertIndex != -1 {
			results[insertIndex] = &Neighbour{
				Distance: distance,
				Item:     item,
			}
		}
	}
	return results, nil
}

func DistanceBetween(x1 CompressionItem, x2 CompressionItem, encodingType CompressionEncodingType) (float64, error) {
	Cx1 := float64(x1.CompressedSize)
	Cx2 := float64(x2.CompressedSize)

	combined, err := record.CombineInterleaved(x1.Splice.Data, x2.Splice.Data)
	if err != nil {
		return math.MaxFloat64, err
	}
	compressedCombined, err := CompressMarketData(combined, encodingType)
	if err != nil {
		return math.MaxFloat64, err
	}

	Cx1x2 := float64(len(compressedCombined))

	return (Cx1x2 - math.Min(Cx1, Cx2)) / math.Max(Cx1, Cx2), nil
}

func (model *CompressionModel) DistanceVarianceHistogram(bucketSize float64) (map[int]float64, error) {
	distanceMap, err := model.DistanceMap()
	if err != nil {
		return nil, err
	}

	resultBuckets := make(map[int][]float64)
	largestBucket := math.MinInt
	smallestBucket := math.MaxInt
	for i, js := range distanceMap {
		for j, dist := range js {
			if i == j {
				continue
			}
			destinationBucket := int(math.Floor(dist / bucketSize))
			if destinationBucket > largestBucket {
				largestBucket = destinationBucket
			}
			if destinationBucket < smallestBucket {
				smallestBucket = destinationBucket
			}
			resultBuckets[destinationBucket] = append(resultBuckets[destinationBucket], model.Items[i].Splice.Result)
		}
	}

	fmt.Printf("Largest bucket: %d\n", largestBucket)

	varianceResults := make(map[int]float64)
	for bucket := smallestBucket; bucket <= largestBucket; bucket++ {
		results, hasBucket := resultBuckets[bucket]
		if !hasBucket {
			varianceResults[bucket] = float64(0)
			continue
		}
		N := float64(len(results))
		sum := float64(0)
		for _, result := range results {
			sum += result
		}
		mean := sum / N
		variance := float64(0)
		for _, result := range results {
			variance += math.Pow(result-mean, 2)
		}
		variance /= N
		varianceResults[bucket] = variance
	}

	fmt.Printf("varianceResults: %+v\n", varianceResults)

	return varianceResults, nil
}

func (model *CompressionModel) DistanceMap() ([][]float64, error) {
	// if we already have a distance map, return it
	if model.CachedDistanceMap != nil && len(model.CachedDistanceMap) == len(model.Items) {
		return model.CachedDistanceMap, nil
	}

	res := make([][]float64, len(model.Items))
	for i := range model.Items {
		res[i] = make([]float64, len(model.Items))
	}
	var wg sync.WaitGroup
	for i, itemI := range model.Items {
		wg.Add(1)
		go func(i int, itemI CompressionItem) {
			defer wg.Done()
			for j, itemJ := range model.Items[i:] {
				canonicalJ := j + i
				distance, err := DistanceBetween(itemI, itemJ, model.EncodingType)
				if err != nil {
					panic(err)
				}
				res[i][canonicalJ] = distance
				res[canonicalJ][i] = distance
			}
		}(i, itemI)
	}
	wg.Wait()
	model.CachedDistanceMap = res
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
	encoder := json.NewEncoder(gzipWriter)
	err = encoder.Encode(model)
	if err != nil {
		return err
	}
	err = gzipWriter.Flush()
	return nil
}

func EncodeToCharVarLength(records []record.Market, field string) string {
	convertFloat := func(f float64) string {
		val := int(math.Floor(f * 1000))
		negative := false
		if val < 0 {
			negative = true
			val = -val
		}
		if val == 0 {
			return "0"
		}
		resStr := make([]rune, val, val)
		if negative {
			resStr[0] = 'N'
		} else {
			resStr[0] = 'P'
		}
		for i := 1; i < val; i *= 2 {
			copy(resStr[i:], resStr[:i])
		}
		return string(resStr)
	}
	var b strings.Builder
	for _, rec := range records {
		switch field {
		case "open":
			b.WriteString(convertFloat(rec.Open))
		case "high":
			b.WriteString(convertFloat(rec.High))
		case "low":
			b.WriteString(convertFloat(rec.Low))
		case "close":
			b.WriteString(convertFloat(rec.Close))
		case "volume":
			b.WriteString(convertFloat(rec.Volume))
		case "vwap":
			b.WriteString(convertFloat(rec.VWAP))
		}
		b.WriteRune(',')
	}

	return b.String()
}

func EncodeToSimpleString(records []record.Market, field string) string {
	var b strings.Builder
	for _, rec := range records {
		switch field {
		case "open":
			b.WriteString(fmt.Sprintf("%.6f", rec.Open))
		case "high":
			b.WriteString(fmt.Sprintf("%.6f", rec.High))
		case "low":
			b.WriteString(fmt.Sprintf("%.6f", rec.Low))
		case "close":
			b.WriteString(fmt.Sprintf("%.6f", rec.Close))
		case "volume":
			b.WriteString(fmt.Sprintf("%.6f", rec.Volume))
		case "vwap":
			b.WriteString(fmt.Sprintf("%.6f", rec.VWAP))
		}
	}

	return b.String()
}

// EncodeToExpandedString repeats each digit in string by its value
// e.g 1.2345 -> 1.22333444455555
func EncodeToExpandedString(records []record.Market, field string) string {
	// happy to hear from anyone that has a less
	// silly way of doing this
	convertFloat := func(f float64) string {
		var newStr []rune
		str := fmt.Sprintf("%.6f", f)
		for _, c := range str {
			if c < '0' || c > '9' {
				newStr = append(newStr, c)
				continue
			}
			cVal := c - '0'
			for i := int32(1); i <= cVal; i++ {
				newStr = append(newStr, c)
			}
		}
		return string(newStr)
	}
	var b strings.Builder
	for _, rec := range records {
		switch field {
		case "open":
			b.WriteString(convertFloat(rec.Open))
		case "high":
			b.WriteString(convertFloat(rec.High))
		case "low":
			b.WriteString(convertFloat(rec.Low))
		case "close":
			b.WriteString(convertFloat(rec.Close))
		case "volume":
			b.WriteString(convertFloat(rec.Volume))
		case "vwap":
			b.WriteString(convertFloat(rec.VWAP))
		}
		b.WriteRune(',')
	}

	return b.String()
}

// EncodeToSFExpandedString repeats each digit in string by its value,
// count is reduced by number of preceding significant digits
// e.g 3.4 -> 333.444
func EncodeToSFExpandedString(records []record.Market, field string) string {
	convertFloat := func(f float64) string {
		var newStr []rune
		str := fmt.Sprintf("%.6f", f)
		for i, c := range str {
			if c < '0' || c > '9' {
				newStr = append(newStr, c)
				continue
			}
			cVal := c - '0'
			for r := int32(0); r < cVal-int32(i); r++ {
				newStr = append(newStr, c)
			}
		}
		return string(newStr)
	}
	var b strings.Builder
	for _, rec := range records {
		switch field {
		case "open":
			b.WriteString(convertFloat(rec.Open))
		case "high":
			b.WriteString(convertFloat(rec.High))
		case "low":
			b.WriteString(convertFloat(rec.Low))
		case "close":
			b.WriteString(convertFloat(rec.Close))
		case "volume":
			b.WriteString(convertFloat(rec.Volume))
		case "vwap":
			b.WriteString(convertFloat(rec.VWAP))
		}
		b.WriteRune(',')
	}

	return b.String()
}

func CompressMarketData(data []record.Market, encodingType CompressionEncodingType) ([]byte, error) {
	var buffBytes []byte
	buff := bytes.NewBuffer(buffBytes)
	writer := zlib.NewWriter(buff)

	// not my proudest work :(
	switch encodingType {
	case SimpleEncoding:
		writer.Write([]byte(EncodeToSimpleString(data, "open")))
		writer.Write([]byte(EncodeToSimpleString(data, "high")))
		writer.Write([]byte(EncodeToSimpleString(data, "low")))
		writer.Write([]byte(EncodeToSimpleString(data, "close")))
		writer.Write([]byte(EncodeToSimpleString(data, "volume")))
		writer.Write([]byte(EncodeToSimpleString(data, "vwap")))
	case ExpandedEncoding:
		writer.Write([]byte(EncodeToExpandedString(data, "open")))
		writer.Write([]byte(EncodeToExpandedString(data, "high")))
		writer.Write([]byte(EncodeToExpandedString(data, "low")))
		writer.Write([]byte(EncodeToExpandedString(data, "close")))
		writer.Write([]byte(EncodeToExpandedString(data, "volume")))
		writer.Write([]byte(EncodeToExpandedString(data, "vwap")))
	case SFExpandedEncoding:
		writer.Write([]byte(EncodeToSFExpandedString(data, "open")))
		writer.Write([]byte(EncodeToSFExpandedString(data, "high")))
		writer.Write([]byte(EncodeToSFExpandedString(data, "low")))
		writer.Write([]byte(EncodeToSFExpandedString(data, "close")))
		writer.Write([]byte(EncodeToSFExpandedString(data, "volume")))
		writer.Write([]byte(EncodeToSFExpandedString(data, "vwap")))
	case CharVarLength:
		writer.Write([]byte(EncodeToCharVarLength(data, "open")))
		writer.Write([]byte(EncodeToCharVarLength(data, "high")))
		writer.Write([]byte(EncodeToCharVarLength(data, "low")))
		writer.Write([]byte(EncodeToCharVarLength(data, "close")))
		writer.Write([]byte(EncodeToCharVarLength(data, "volume")))
		writer.Write([]byte(EncodeToCharVarLength(data, "vwap")))
	}

	err := writer.Flush()
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func CompressSplice(s splicer.Splice, encodingType CompressionEncodingType) (*CompressionItem, error) {
	compressed, err := CompressMarketData(s.Data, encodingType)
	if err != nil {
		return nil, err
	}
	return &CompressionItem{
		Splice:         s,
		CompressedSize: len(compressed),
	}, nil
}
