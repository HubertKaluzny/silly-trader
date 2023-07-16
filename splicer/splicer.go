package splicer

import (
	"errors"

	"github.com/hubertkaluzny/silly-trader/record"
)

type Splice struct {
	Data      []record.Market
	StartTime int64
	EndTime   int64
	Result    float64
}

func SpliceData(data []record.Market, period, resultN int) ([]Splice, error) {
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
