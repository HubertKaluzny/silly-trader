package eval

import (
	"math"
	"sort"

	"github.com/hubertkaluzny/silly-trader/model"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func CompressionHeatMap(model *model.CompressionModel, downSampleBy int) (*charts.HeatMap, error) {
	hmap := charts.NewHeatMap()

	similarityMap, err := model.DistanceMap()
	if err != nil {
		return nil, err
	}
	mapLength := len(similarityMap) / downSampleBy
	hmData := make([]opts.HeatMapData, mapLength*mapLength)
	inserted := 0
	min := math.MaxFloat64
	max := math.SmallestNonzeroFloat64

	for i := 0; i < mapLength; i++ {
		for j := 0; j < mapLength; j++ {
			minLocalValue := math.MaxFloat64

			// local loop
			if downSampleBy > 1 {
				for x := 0; x < downSampleBy; x++ {
					for y := 0; y < downSampleBy; y++ {
						ix := (downSampleBy * i) + x
						jy := (downSampleBy * j) + y
						minLocalValue = math.Min(minLocalValue, similarityMap[ix][jy])
					}
				}
				hmData[inserted] = opts.HeatMapData{Value: [3]interface{}{i, j, minLocalValue}}
			} else {
				minLocalValue = similarityMap[i][j]
				hmData[inserted] = opts.HeatMapData{Value: [3]interface{}{i, j, similarityMap[i][j]}}
			}

			inserted += 1
			if minLocalValue > max {
				max = minLocalValue
			} else if minLocalValue < min {
				min = minLocalValue
			}
		}
	}

	hmap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Compression Model Distance Heatmap",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			XAxisIndex: 0,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			YAxisIndex: 0,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "value",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "value",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
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

	hmap.AddSeries("Distance", hmData)

	return hmap, nil
}

func CompressionDstVarHistogram(model *model.CompressionModel) (*charts.Bar, error) {
	const bucketSize = float64(0.001)

	bar := charts.NewBar()

	buckets, err := model.DistanceVarianceHistogram(bucketSize)
	if err != nil {
		return nil, err
	}

	barData := make([]opts.BarData, len(buckets))
	axis := make([]int, len(buckets))

	i := 0
	for bucket := range buckets {
		axis[i] = bucket
		i++
	}
	sort.Ints(axis)
	for i, bucket := range axis {
		barData[i] = opts.BarData{Value: buckets[bucket]}
	}

	bar.SetXAxis(axis).AddSeries("Distance", barData)

	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Compression Model Distance Result variance",
		}),
	)

	return bar, nil
}

func CompressionSizeHistogram(model *model.CompressionModel) (*charts.Bar, error) {
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
