package eval

import (
	"fmt"
	"math"
	"sort"

	"github.com/hubertkaluzny/silly-trader/strategy"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func CompressionHeatMap(model *strategy.CompressionModel) (*charts.HeatMap, error) {
	hmap := charts.NewHeatMap()

	similarityMap, err := model.SimilarityMap()
	if err != nil {
		return nil, err
	}
	hmData := make([]opts.HeatMapData, len(similarityMap)*len(similarityMap))
	inserted := 0
	min := math.MaxFloat64
	max := math.SmallestNonzeroFloat64
	for i, js := range similarityMap {
		for j, val := range js {
			hmData[inserted] = opts.HeatMapData{Value: [3]interface{}{i, j, val}}
			fmt.Print(val)
			fmt.Print(",")
			inserted += 1
			if val > max {
				max = val
			} else if val < min {
				min = val
			}
		}
	}

	fmt.Printf("min: %f, max: %f\n", math.Floor(min), math.Ceil(max))

	hmap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Compression Model Similarity Heatmap",
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: true,
			Min:        float32(math.Floor(min)),
			Max:        float32(math.Ceil(max)),
			InRange: &opts.VisualMapInRange{
				Color: []string{"#50a3ba", "#eac736", "#d94e5d"},
			},
		}),
	)

	hmap.AddSeries("Similarity", hmData, charts.WithHeatMapChartOpts(opts.HeatMapChart{
		XAxisIndex: 0,
		YAxisIndex: 0,
	}))

	return hmap, nil
}

func CompressionSizeHistogram(model *strategy.CompressionModel) (*charts.Bar, error) {
	bar := charts.NewBar()

	buckets := model.SizeResultBuckets()

	sizes := make([]int, len(buckets))
	i := 0
	for size := range buckets {
		sizes[i] = size
		i++
	}
	sort.Ints(sizes)

	barData := make([]opts.BarData, len(sizes))
	for i, bucket := range sizes {
		count := len(buckets[bucket])
		barData[i] = opts.BarData{Value: count}
	}

	bar.SetXAxis(sizes).AddSeries("Frequency", barData)

	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Compression Model Size Histogram",
		}),
	)

	return bar, nil
}
