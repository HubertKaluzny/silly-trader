package model

import (
	"fmt"
	"math"

	"github.com/hubertkaluzny/silly-trader/record"
)

type ModelType string

const (
	Compression ModelType = "compression"
	Cosine      ModelType = "cosine"
)

type Model interface {
	SaveToFile(file string) error
	AddMarketData(data []record.Market) error
	SizeResultBuckets() map[int][]float64
	DistanceMap() ([][]float64, error)
}

func DistanceVarianceHistogram(model Model, bucketSize float64) (map[int]float64, error) {
	distanceMap, err := model.DistanceMap()
	if err != nil {
		return nil, err
	}

	resultBuckets := make(map[int][]float64)
	largestBucket := math.MinInt
	smallestBucket := math.MaxInt
	for i, js := range distanceMap {
		for j, dist := range js {
			if i == j {
				//continue
			}
			destinationBucket := int(math.Floor(dist / bucketSize))
			if destinationBucket > largestBucket {
				largestBucket = destinationBucket
			}
			if destinationBucket < smallestBucket {
				smallestBucket = destinationBucket
			}

			resultBuckets[destinationBucket] = append(resultBuckets[destinationBucket], math.Abs(model.Items[i].Result-model.Items[j].Result))
		}
	}

	fmt.Printf("Largest bucket: %d\n", largestBucket)

	varianceResults := make(map[int]float64)
	for bucket := smallestBucket; bucket <= largestBucket; bucket++ {
		results, hasBucket := resultBuckets[bucket]
		if !hasBucket {
			varianceResults[bucket] = float64(0)
			continue
		}
		N := float64(len(results))
		sum := float64(0)
		for _, result := range results {
			sum += result
		}
		mean := sum / N
		variance := float64(0)
		for _, result := range results {
			variance += math.Pow(result-mean, 2)
		}
		variance /= N
		varianceResults[bucket] = variance
	}

	fmt.Printf("varianceResults: %+v\n", varianceResults)

	return varianceResults, nil
}
