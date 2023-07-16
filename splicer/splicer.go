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

type Splicer struct {
	Period  int
	ResultN int
}

func NewSplicer(period, resultN int) *Splicer {
	return &Splicer{
		Period:  period,
		ResultN: resultN,
	}
}

func (s *Splicer) Splice(data []record.Market) ([]Splice, error) {
	if len(data) < s.Period+s.ResultN {
		return nil, errors.New("insufficient data length provided for provided params")
	}

	// can pre-allocate this if we're not too lazy
	// to do the maths
	var splices []Splice
	for i, _ := range data[s.Period : len(data)-s.ResultN] {
		spliceData := data[i:(i + s.Period)]
		startTime := spliceData[0].Timestamp
		endTime := spliceData[s.Period-1].Timestamp
		result := data[i+s.Period+s.ResultN].Close

		splices = append(splices, Splice{
			Data:      spliceData,
			StartTime: startTime,
			EndTime:   endTime,
			Result:    result,
		})
	}

	return splices, nil
}
