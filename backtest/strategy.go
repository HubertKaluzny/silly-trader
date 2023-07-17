package backtest

import (
	"github.com/hubertkaluzny/silly-trader/model"
	"github.com/hubertkaluzny/silly-trader/record"
)

// EvaluateBuyHold expects history to come pre-normalised
// returns whether to buy :)
func EvaluateBuyHold(m *model.CompressionModel, curHistory []record.Market) (bool, error) {
	observation := curHistory[m.SpliceOptions.Period:]
	results, err := m.PredictResult(observation, 9)
	if err != nil {
		return false, err
	}

	// assuming item results are z-scores
	// weighted result by distance
	buyFreq := float64(0)
	sellFreq := float64(0)
	for _, res := range results {
		if res.Item.Splice.Result > 1 {
			buyFreq += 1 / res.Distance
		} else if res.Item.Splice.Result < 1 {
			sellFreq += 1 / res.Distance
		}
	}

	return buyFreq > sellFreq, nil
}
