package splicer

import (
	"errors"
	"fmt"
	"math"

	"github.com/hubertkaluzny/silly-trader/record"
)

type Splice struct {
	Data              []record.Market   `json:"data"`
	StartTime         int64             `json:"start_time"`
	EndTime           int64             `json:"end_time"`
	Result            float64           `json:"result"`
	NormalisationType NormalisationType `json:"normalisation_type"`
}

type NormalisationType string

const (
	None              NormalisationType = "none"
	PercentageChanges NormalisationType = "percentage"
	ZScore            NormalisationType = "z_score"
)

func ToNormalisationType(input string) (NormalisationType, error) {
	switch input {
	case string(None):
		return None, nil
	case string(PercentageChanges):
		return PercentageChanges, nil
	case string(ZScore):
		return ZScore, nil
	}
	return None, errors.New("invalid normalisation type specified")
}

type SpliceOptions struct {
	Period            int               `json:"period"`
	ResultN           int               `json:"result_n"`
	SkipN             int               `json:"skip_n"`
	NormalisationType NormalisationType `json:"normalisation_type"`
}

func normaliseToPercentages(data []record.Market) []record.Market {
	if len(data) == 0 {
		return nil
	}
	if len(data) == 1 {
		return []record.Market{
			{
				Timestamp: data[0].Timestamp,
				Open:      1,
				High:      1,
				Low:       1,
				Close:     1,
				Volume:    1,
				VWAP:      1,
			},
		}
	}
	prev := data[0]
	rest := data[1:]
	res := make([]record.Market, len(rest))
	for i, data := range rest {
		res[i] = record.Market{
			Timestamp: data.Timestamp,
			Open:      data.Open / prev.Open,
			High:      data.High / prev.High,
			Low:       data.Low / prev.Low,
			Close:     data.Close / prev.Close,
			Volume:    data.Volume / prev.Volume,
			VWAP:      data.VWAP / prev.VWAP,
		}
		prev = data
	}
	return res
}

func normaliseToZScore(data []record.Market) []record.Market {
	if len(data) == 0 {
		return nil
	}
	N := len(data)

	// calculate means
	mean := map[string]float64{
		"open":   0,
		"high":   0,
		"low":    0,
		"close":  0,
		"volume": 0,
		"VWAP":   0,
	}
	for _, rec := range data {
		mean["open"] += rec.Close
		mean["high"] += rec.High
		mean["low"] += rec.Low
		mean["close"] += rec.Close
		mean["volume"] += rec.Volume
		mean["VWAP"] += rec.VWAP
	}
	mean["open"] /= float64(N)
	mean["high"] /= float64(N)
	mean["low"] /= float64(N)
	mean["close"] /= float64(N)
	mean["volume"] /= float64(N)
	mean["VWAP"] /= float64(N)

	// calculate standard deviations
	std := map[string]float64{
		"open":   0,
		"high":   0,
		"low":    0,
		"close":  0,
		"volume": 0,
		"VWAP":   0,
	}
	for _, rec := range data {
		std["open"] += math.Pow(rec.Open-mean["open"], 2)
		std["high"] += math.Pow(rec.High-mean["high"], 2)
		std["low"] += math.Pow(rec.Low-mean["low"], 2)
		std["close"] += math.Pow(rec.Close-mean["close"], 2)
		std["volume"] += math.Pow(rec.Volume-mean["volume"], 2)
		std["VWAP"] += math.Pow(rec.VWAP-mean["VWAP"], 2)
	}
	std["open"] /= float64(N)
	std["high"] /= float64(N)
	std["low"] /= float64(N)
	std["close"] /= float64(N)
	std["volume"] /= float64(N)
	std["VWAP"] /= float64(N)

	std["open"] = math.Sqrt(std["open"])
	std["high"] = math.Sqrt(std["high"])
	std["low"] = math.Sqrt(std["low"])
	std["close"] = math.Sqrt(std["close"])
	std["volume"] = math.Sqrt(std["volume"])
	std["VWAP"] = math.Sqrt(std["VWAP"])

	// calculate z-scores
	res := make([]record.Market, len(data))
	for i, rec := range data {
		res[i] = record.Market{
			Timestamp: rec.Timestamp,
			Open:      (rec.Open - mean["open"]) / std["open"],
			High:      (rec.High - mean["high"]) / std["high"],
			Low:       (rec.Low - mean["low"]) / std["low"],
			Close:     (rec.Close - mean["close"]) / std["close"],
			Volume:    (rec.Volume - mean["volume"]) / std["volume"],
			VWAP:      (rec.VWAP - mean["volume"]) / std["volume"],
		}
	}
	return res
}

func SpliceData(data []record.Market, opts SpliceOptions) ([]Splice, error) {
	period := opts.Period
	resultN := opts.ResultN
	if len(data) < period+resultN {
		return nil, errors.New("insufficient data length provided for provided params")
	}

	switch opts.NormalisationType {
	case PercentageChanges:
		data = normaliseToPercentages(data)
	case ZScore:
		data = normaliseToZScore(data)
	}

	var splices []Splice
	for i := 0; i+period+resultN-1 < len(data); i += 1 + opts.SkipN {
		fmt.Printf("i: %d, period: %d, skip: %d, result: %d\n", i, period, opts.SkipN, opts.ResultN)
		spliceData := data[i:(i + period)]

		startTime := spliceData[0].Timestamp
		endTime := spliceData[period-1].Timestamp

		priceAtClose := spliceData[period-1].Close

		priceAtResult := data[i+period+resultN-1].Open

		result := priceAtResult - priceAtClose

		splices = append(splices, Splice{
			Data:      spliceData,
			StartTime: startTime,
			EndTime:   endTime,
			Result:    result,
		})
	}

	return splices, nil
}
