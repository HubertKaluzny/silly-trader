package model

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"math"
	"os"
	"sync"

	"github.com/hubertkaluzny/silly-trader/record"
	"github.com/hubertkaluzny/silly-trader/splicer"
)

type CosineItem struct {
	Data   record.Model `json:"data"`
	Result float64      `json:"result"`
}

type CosineModel struct {
	SpliceOptions     splicer.SpliceOptions
	CombineStrategy   record.CombineStrategy
	Items             []CosineItem
	CachedDistanceMap [][]float64
}

func NewCosineModel(spliceOpts splicer.SpliceOptions, combineStrat record.CombineStrategy) *CosineModel {
	return &CosineModel{
		SpliceOptions:   spliceOpts,
		CombineStrategy: combineStrat,
	}
}

func (m *CosineModel) AddMarketData(data []record.Market) error {
	splices, err := splicer.SpliceData(data, m.SpliceOptions)
	if err != nil {
		return err
	}
	items := make([]CosineItem, len(splices))
	for i, splice := range splices {
		items[i].Data = record.MarketToModel(splice.Data)
		items[i].Result = splice.Result
	}
	m.Items = append(m.Items, items...)
	return nil
}

func cosineDistanceBetween(x1s, x2s []float64) (float64, error) {
	if len(x1s) != len(x2s) {
		return math.MaxFloat64, errors.New("vectors must have the same length")
	}

	var numerator, sumx1s, sumx2s float64
	for i, x1 := range x1s {
		sumx1s += math.Pow(x1s[i], 2)
		sumx2s += math.Pow(x2s[i], 2)
		for _, x2 := range x2s {
			numerator += x1 * x2
		}
	}
	denominator := math.Sqrt(sumx1s * sumx2s)

	return numerator / denominator, nil
}

func CosineDistanceBetween(x1, x2 CosineItem) (float64, error) {
	// calculate cosine distances between each array for ohlcv + vwap
	// return distance average or sum?
	opens, err := cosineDistanceBetween(x1.Data.Opens, x2.Data.Opens)
	if err != nil {
		return math.MaxFloat64, err
	}
	highs, err := cosineDistanceBetween(x1.Data.Highs, x2.Data.Highs)
	if err != nil {
		return math.MaxFloat64, err
	}
	lows, err := cosineDistanceBetween(x1.Data.Lows, x2.Data.Lows)
	if err != nil {
		return math.MaxFloat64, err
	}
	closes, err := cosineDistanceBetween(x1.Data.Closes, x2.Data.Closes)
	if err != nil {
		return math.MaxFloat64, err
	}
	volumes, err := cosineDistanceBetween(x1.Data.Volumes, x2.Data.Volumes)
	if err != nil {
		return math.MaxFloat64, err
	}
	vwaps, err := cosineDistanceBetween(x1.Data.VWAPs, x2.Data.VWAPs)
	if err != nil {
		return math.MaxFloat64, err
	}

	// I think technically here we should be combining ohlcv + vwaps
	// as one large vector rather than individual vectors?
	sum := opens + highs + lows + closes + volumes + vwaps
	return sum / 6, nil
}

func (m *CosineModel) DistanceMap() ([][]float64, error) {
	if m.CachedDistanceMap != nil && len(m.CachedDistanceMap) == len(m.Items) {
		return m.CachedDistanceMap, nil
	}
	res := make([][]float64, len(m.Items))
	for i := range m.Items {
		res[i] = make([]float64, len(m.Items))
	}
	var wg sync.WaitGroup
	for i, itemI := range m.Items {
		wg.Add(1)
		go func(i int, itemI CosineItem) {
			defer wg.Done()
			for j, itemJ := range m.Items[i:] {
				canonicalJ := j + i
				distance, err := CosineDistanceBetween(itemI, itemJ)
				if err != nil {
					panic(err)
				}
				res[i][canonicalJ] = distance
				res[canonicalJ][i] = distance
			}

		}(i, itemI)
	}
	wg.Wait()
	m.CachedDistanceMap = res
	return res, nil
}

func (m *CosineModel) SaveToFile(file string) error {
	modelFile, err := os.Create(file)
	defer modelFile.Close()
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(modelFile)
	encoder := json.NewEncoder(gzipWriter)
	err = encoder.Encode(m)
	if err != nil {
		return err
	}
	err = gzipWriter.Flush()
	return nil
}
