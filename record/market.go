package record

import (
	"errors"
	"strconv"
)

type Market struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	VWAP      float64
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
