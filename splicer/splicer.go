package splicer

import (
	"errors"

	"github.com/hubertkaluzny/silly-trader/record"
)

type Splice struct {
	Data              []record.Market          `json:"data"`
	StartTime         int64                    `json:"start_time"`
	EndTime           int64                    `json:"end_time"`
	Result            float64                  `json:"result"`
	NormalisationType record.NormalisationType `json:"normalisation_type"`
}

type SpliceOptions struct {
	Period            int                      `json:"period"`
	ResultN           int                      `json:"result_n"`
	SkipN             int                      `json:"skip_n"`
	NormalisationType record.NormalisationType `json:"normalisation_type"`
}

func SpliceData(data []record.Market, opts SpliceOptions) ([]Splice, error) {
	period := opts.Period
	resultN := opts.ResultN
	if len(data) < period+resultN {
		return nil, errors.New("insufficient data length provided for provided params")
	}

	var splices []Splice
	for i := 0; i+period+resultN-1 < len(data); i += 1 + opts.SkipN {
		curPeriodData := data[i:(i + period + resultN)]
		switch opts.NormalisationType {
		case record.ZScore:
			curPeriodData = record.NormaliseToZScore(curPeriodData)
		}
		spliceData := curPeriodData[0:period]

		startTime := spliceData[0].Timestamp
		endTime := spliceData[period-1].Timestamp

		priceAtClose := spliceData[period-1].Close
		priceAtResult := curPeriodData[period+resultN-1].Open
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
