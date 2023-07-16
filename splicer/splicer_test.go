package splicer

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hubertkaluzny/silly-trader/record"
)

func randomMarketData(length int) []record.Market {
	res := make([]record.Market, length)
	for i := 0; i < length; i++ {
		res[i] = record.Market{
			Timestamp: int64(i),
			Open:      rand.Float64(),
			High:      rand.Float64(),
			Low:       rand.Float64(),
			Close:     rand.Float64(),
			Volume:    rand.Float64(),
			VWAP:      rand.Float64(),
		}
	}
	return res
}

func TestSpliceData(t *testing.T) {
	runTest := func(t *testing.T, dataSize, period, resultN int) {
		testData := randomMarketData(dataSize)

		splices, err := SpliceData(testData, SpliceOptions{Period: period, ResultN: resultN})
		assert.NoError(t, err)

		expectedLength := dataSize - (resultN + period)
		assert.Equal(t, expectedLength, len(splices))

		fstSplice := splices[0]
		assert.Equal(t, period, len(fstSplice.Data))
		assert.ElementsMatch(t, testData[:period], fstSplice.Data)

		lastSplice := splices[expectedLength-1]
		assert.Equal(t, period, len(lastSplice.Data))
		end := dataSize - resultN - 1
		start := end - period
		expectedLastSpliceData := testData[start:end]
		assert.ElementsMatch(t, expectedLastSpliceData, lastSplice.Data)
	}

	t.Run("single period splices", func(t *testing.T) {
		runTest(t, 36, 1, 1)
	})

	t.Run("uneven period splices", func(t *testing.T) {
		runTest(t, 36, 3, 1)
	})

	t.Run("uneven period and resultN splices", func(t *testing.T) {
		runTest(t, 36, 3, 2)
	})
}
