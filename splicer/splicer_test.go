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
	runTest := func(t *testing.T, dataSize int, opts SpliceOptions) {
		testData := randomMarketData(dataSize)
		period := opts.Period
		resultN := opts.ResultN
		skipN := opts.SkipN
		splices, err := SpliceData(testData, opts)
		assert.NoError(t, err)

		iterations := (dataSize - period - resultN + 1) / (1 + skipN)
		remainder := (dataSize - period - resultN + 1) % (1 + skipN)
		expectedLength := iterations
		if remainder > 0 {
			expectedLength++
		}

		assert.Equal(t, expectedLength, len(splices))

		fstSplice := splices[0]
		assert.Equal(t, period, len(fstSplice.Data))
		assert.ElementsMatch(t, testData[:period], fstSplice.Data)

		lastSplice := splices[expectedLength-1]
		assert.Equal(t, period, len(lastSplice.Data))
		start := (iterations - 1) * (1 + skipN)
		if remainder > 0 {
			start += 1 + skipN
		}
		end := start + period
		expectedLastSpliceData := testData[start:end]
		assert.ElementsMatch(t, expectedLastSpliceData, lastSplice.Data)
	}

	t.Run("single period splices", func(t *testing.T) {
		runTest(t, 36, SpliceOptions{
			Period:            1,
			ResultN:           1,
			NormalisationType: None,
		})
	})

	t.Run("uneven period splices", func(t *testing.T) {
		runTest(t, 36, SpliceOptions{
			Period:            3,
			ResultN:           1,
			NormalisationType: None,
		})
	})

	t.Run("uneven period and resultN splices", func(t *testing.T) {
		runTest(t, 36, SpliceOptions{
			Period:            3,
			ResultN:           2,
			NormalisationType: None,
		})
	})

	t.Run("skipping odd number of splices", func(t *testing.T) {
		runTest(t, 36, SpliceOptions{
			Period:            1,
			ResultN:           1,
			SkipN:             3,
			NormalisationType: None,
		})
	})

	t.Run("skipping odd number of splices, and periods", func(t *testing.T) {
		runTest(t, 36, SpliceOptions{
			Period:            2,
			ResultN:           1,
			SkipN:             3,
			NormalisationType: None,
		})
	})

	t.Run("Odd sized dataset", func(t *testing.T) {
		runTest(t, 37, SpliceOptions{
			Period:            5,
			ResultN:           1,
			SkipN:             3,
			NormalisationType: None,
		})
		runTest(t, 731, SpliceOptions{
			Period:            18,
			ResultN:           5,
			SkipN:             7,
			NormalisationType: None,
		})
	})
}
