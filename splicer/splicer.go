package splicer

import (
	"errors"

	"github.com/hubertkaluzny/silly-trader/record"
)

type Splice struct {
	Data      []record.Market `json:"data"`
	StartTime int64           `json:"start_time"`
	EndTime   int64           `json:"end_time"`
	Result    float64         `json:"result"`
}

type SpliceOptions struct {
	Period  int `json:"period"`
	ResultN int `json:"result_n"`
}

func SpliceData(data []record.Market, opts SpliceOptions) ([]Splice, error) {
	period := opts.Period
	resultN := opts.ResultN
	if len(data) < period+resultN {
		return nil, errors.New("insufficient data length provided for provided params")
	}

	// can pre-allocate this if we're not too lazy
	// to do the maths
	var splices []Splice
	for i, _ := range data[period : len(data)-resultN] {
		spliceData := data[i:(i + period)]
		startTime := spliceData[0].Timestamp
		endTime := spliceData[period-1].Timestamp
		result := data[i+period+resultN].Close

		splices = append(splices, Splice{
			Data:      spliceData,
			StartTime: startTime,
			EndTime:   endTime,
			Result:    result,
		})
	}

	return splices, nil
}
