package model

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/4kills/go-libdeflate/v2"

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
	RomanEncoding      CompressionEncodingType = "roman"
)

type CombineStrategy string

const (
	InterleaveCombine CombineStrategy = "interleave"
	ConcatCombine     CombineStrategy = "concat"
)

func ToCombineStrategy(input string) (CombineStrategy, error) {
	switch input {
	case string(InterleaveCombine):
		return InterleaveCombine, nil
	case string(ConcatCombine):
		return ConcatCombine, nil
	}
	return InterleaveCombine, errors.New("invalid combine strategy specified")
}

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
	case string(RomanEncoding):
		return RomanEncoding, nil
	}
	return SimpleEncoding, errors.New("invalid encoding type specified")
}

type CompressionItem struct {
	Data           record.Model `json:"model"`
	CompressedSize int          `json:"compressed_length"`
	Result         float64      `json:"result"`
}

type CompressionModel struct {
	SpliceOptions     splicer.SpliceOptions   `json:"splice_options"`
	Items             []CompressionItem       `json:"items"`
	EncodingType      CompressionEncodingType `json:"encoding_type"`
	CachedDistanceMap [][]float64             `json:"distance_map"`
	CombineStrategy   CombineStrategy         `json:"combine_strategy"`
}

func NewCompressionModel(spliceOpts splicer.SpliceOptions, encodingType CompressionEncodingType, combineStrat CombineStrategy) *CompressionModel {
	return &CompressionModel{
		SpliceOptions:   spliceOpts,
		EncodingType:    encodingType,
		CombineStrategy: combineStrat,
	}
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
	c, err := libdeflate.NewCompressorLevel(libdeflate.MaxCompressionLevel)
	if err != nil {
		return err
	}
	newItems := make([]CompressionItem, len(splices))
	for i, s := range splices {
		modelData := record.MarketToModel(s.Data)
		item, err := CompressModelData(c, modelData, model.EncodingType)
		item.Result = s.Result
		if err != nil {
			return err
		}
		newItems[i] = *item
	}
	model.Items = append(model.Items, newItems...)
	return nil
}

func (model *CompressionModel) PredictResults(observation record.Model, opts PredictionOpts) (int, error) {
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
		if res.Item.Result > 1 {
			buyFreq += 1 / res.Distance
		} else if res.Item.Result < 1 {
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
func (model *CompressionModel) GetClosestNeighbours(observation record.Model, nearestN int) ([]*Neighbour, error) {

	c, err := libdeflate.NewCompressorLevel(libdeflate.MaxCompressionLevel)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	compressedObservation, err := CompressModelData(c, observation, model.EncodingType)
	if err != nil {
		return nil, err
	}
	Cx1 := float64(compressedObservation.CompressedSize)
	results := make([]*Neighbour, nearestN)
	for _, item := range model.Items {
		item := item

		var combined record.Model
		switch model.CombineStrategy {
		case InterleaveCombine:
			interleaved, err := record.InterleaveModels(item.Data, observation)
			if err != nil {
				return nil, err
			}
			combined = *interleaved
		case ConcatCombine:
			combined = record.ConcatModels(item.Data, observation)
		}

		compressedCombined, err := GetCompressedLength(c, combined, model.EncodingType)
		if err != nil {
			return nil, err
		}
		Cx1x2 := float64(compressedCombined)
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

func DistanceBetween(c libdeflate.Compressor, x1 CompressionItem, x2 CompressionItem, encodingType CompressionEncodingType, combineStrat CombineStrategy) (float64, error) {
	Cx1 := float64(x1.CompressedSize)
	Cx2 := float64(x2.CompressedSize)

	var combined record.Model
	switch combineStrat {
	case InterleaveCombine:
		interleaved, err := record.InterleaveModels(x1.Data, x2.Data)
		if err != nil {
			return math.MaxFloat64, err
		}
		combined = *interleaved
	case ConcatCombine:
		combined = record.ConcatModels(x1.Data, x2.Data)
	}
	compressedCombined, err := GetCompressedLength(c, combined, encodingType)
	if err != nil {
		return math.MaxFloat64, err
	}

	Cx1x2 := float64(compressedCombined)

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
				//continue
			}
			destinationBucket := int(math.Floor(dist / bucketSize))
			if destinationBucket > largestBucket {
				largestBucket = destinationBucket
			}
			if destinationBucket < smallestBucket {
				smallestBucket = destinationBucket
			}
			resultBuckets[destinationBucket] = append(resultBuckets[destinationBucket], model.Items[i].Result)
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
			c, err := libdeflate.NewCompressor()
			if err != nil {
				panic(err)
			}
			defer c.Close()
			for j, itemJ := range model.Items[i:] {
				canonicalJ := j + i
				distance, err := DistanceBetween(c, itemI, itemJ, model.EncodingType, model.CombineStrategy)
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
		buckets[bucket] = append(buckets[bucket], item.Result)
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

func EncodeToCharVarLength(b *strings.Builder, records []float64) {
	convertFloat := func(f float64) {
		val := int(math.Round(f * 100))
		negative := false
		if val < 0 {
			negative = true
			val = -val
		}
		if val == 0 {
			b.WriteRune('0')
			return
		}
		resStr := make([]byte, val, val)
		if negative {
			resStr[0] = 'N'
		} else {
			resStr[0] = 'P'
		}
		for i := 1; i < val; i *= 2 {
			copy(resStr[i:], resStr[:i])
		}
		b.Grow(val)
		b.Write(resStr)
	}

	for _, rec := range records {
		convertFloat(rec)
		b.WriteRune(',')
	}
}

func EncodeToRomanNumerals(b *strings.Builder, records []float64) {
	conversions := map[int]string{
		1000: "M",
		900:  "CM",
		500:  "D",
		400:  "CD",
		100:  "C",
		90:   "XC",
		50:   "L",
		40:   "XL",
		10:   "X",
		9:    "IX",
		5:    "V",
		4:    "IV",
		1:    "I",
	}
	convertFloat := func(f float64) {
		val := int(math.Round(f * 1000))
		negative := false
		if val < 0 {
			negative = true
			val = -val
		}
		if val == 0 {
			b.WriteRune('0')
			return
		}
		if negative {
			b.WriteRune('-')
		}
		for romanVal, romanDigit := range conversions {
			for val >= romanVal {
				b.WriteString(romanDigit)
				val -= romanVal
			}
		}
	}
	for _, rec := range records {
		convertFloat(rec)
		b.WriteRune(',')
	}
}

func EncodeToSimpleString(b *strings.Builder, records []float64) {
	for _, rec := range records {
		b.WriteString(fmt.Sprintf("%.6f,", rec))
	}
}

// EncodeToExpandedString repeats each digit in string by its value
// e.g 1.2345 -> 1.22333444455555
func EncodeToExpandedString(b *strings.Builder, records []float64) {
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
	for _, rec := range records {
		b.WriteString(convertFloat(rec))
		b.WriteRune(',')
	}
}

// EncodeToSFExpandedString repeats each digit in string by its value,
// count is reduced by number of preceding significant digits
// e.g 3.4 -> 333.444
func EncodeToSFExpandedString(b *strings.Builder, records []float64) {
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
	for _, rec := range records {
		b.WriteString(convertFloat(rec))
		b.WriteRune(',')
	}
}

type EncodingFunc func(*strings.Builder, []float64)

func GetCompressedLength(c libdeflate.Compressor, data record.Model, encodingType CompressionEncodingType) (int, error) {
	var encodingFunc EncodingFunc
	switch encodingType {
	case SimpleEncoding:
		encodingFunc = EncodeToSimpleString
	case ExpandedEncoding:
		encodingFunc = EncodeToExpandedString
	case SFExpandedEncoding:
		encodingFunc = EncodeToSFExpandedString
	case CharVarLength:
		encodingFunc = EncodeToCharVarLength
	case RomanEncoding:
		encodingFunc = EncodeToRomanNumerals
	}

	var b strings.Builder
	calcSize := func(input []float64) (int, error) {
		b.Reset()
		encodingFunc(&b, input)
		var compBuffer = make([]byte, b.Len(), b.Len())
		size, _, err := c.Compress([]byte(b.String()), compBuffer, libdeflate.ModeGzip)
		if err != nil {
			return -1, err
		}
		return size, nil
	}

	oSize, err := calcSize(data.Opens)
	if err != nil {
		return -1, err
	}
	hSize, err := calcSize(data.Highs)
	if err != nil {
		return -1, err
	}
	lSize, err := calcSize(data.Lows)
	if err != nil {
		return -1, err
	}
	cSize, err := calcSize(data.Closes)
	if err != nil {
		return -1, err
	}
	vSize, err := calcSize(data.Volumes)
	if err != nil {
		return -1, err
	}
	vwapSize, err := calcSize(data.VWAPs)
	if err != nil {
		return -1, err
	}

	return oSize + hSize + lSize + cSize + vSize + vwapSize, nil
}

func CompressModelData(c libdeflate.Compressor, m record.Model, encodingType CompressionEncodingType) (*CompressionItem, error) {
	compressed, err := GetCompressedLength(c, m, encodingType)
	if err != nil {
		return nil, err
	}
	return &CompressionItem{
		Data:           m,
		CompressedSize: compressed,
	}, nil
}
