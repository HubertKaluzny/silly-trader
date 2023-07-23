package record

import (
	"errors"
	"math"
	"strconv"
)

type Market struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	VWAP      float64 `json:"vwap"`
}

type NormalisationType string

const (
	None   NormalisationType = "none"
	ZScore NormalisationType = "z_score"
)

func ToNormalisationType(input string) (NormalisationType, error) {
	switch input {
	case string(None):
		return None, nil
	case string(ZScore):
		return ZScore, nil
	}
	return None, errors.New("invalid normalisation type specified")
}

func NormaliseToZScore(data []Market) []Market {
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
	res := make([]Market, len(data))
	for i, rec := range data {
		res[i] = Market{
			Timestamp: rec.Timestamp,
			Open:      (rec.Open - mean["open"]) / std["open"],
			High:      (rec.High - mean["high"]) / std["high"],
			Low:       (rec.Low - mean["low"]) / std["low"],
			Close:     (rec.Close - mean["close"]) / std["close"],
			Volume:    (rec.Volume - mean["volume"]) / std["volume"],
			VWAP:      (rec.VWAP - mean["VWAP"]) / std["VWAP"],
		}
	}
	return res
}

func SerializeMarket(r Market) []string {
	return []string{
		strconv.FormatInt(r.Timestamp, 10),
		strconv.FormatFloat(r.Open, 'G', -1, 64),
		strconv.FormatFloat(r.High, 'G', -1, 64),
		strconv.FormatFloat(r.Low, 'G', -1, 64),
		strconv.FormatFloat(r.Close, 'G', -1, 64),
		strconv.FormatFloat(r.Volume, 'G', -1, 64),
		strconv.FormatFloat(r.VWAP, 'G', -1, 64),
	}
}

func UnserialiseMarket(fields []string) (*Market, error) {
	if len(fields) != 7 {
		return nil, errors.New("incorrect number of fields in record")
	}
	ts, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return nil, err
	}
	o, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, err
	}
	h, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return nil, err
	}
	l, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		return nil, err
	}
	c, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return nil, err
	}
	v, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return nil, err
	}
	vwap, err := strconv.ParseFloat(fields[6], 64)
	if err != nil {
		return nil, err
	}

	return &Market{
		Timestamp: ts,
		Open:      o,
		High:      h,
		Low:       l,
		Close:     c,
		Volume:    v,
		VWAP:      vwap,
	}, nil
}
