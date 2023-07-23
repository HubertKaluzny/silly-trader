package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/urfave/cli/v2"

	"github.com/hubertkaluzny/silly-trader/eval"
	"github.com/hubertkaluzny/silly-trader/model"
	"github.com/hubertkaluzny/silly-trader/record"
	"github.com/hubertkaluzny/silly-trader/splicer"
)

func main() {

	const PeriodFlag = "period"
	const ResultNFlag = "resultn"
	const SkipNFlag = "skipn"
	const NormalisationFlag = "normalisation"
	const CompressionEncodingFlag = "cencoding"
	const ModelCombineStrategyFlag = "combine"
	const DownsampleFlag = "downsample"

	app := &cli.App{
		Name: "model",
		Commands: []*cli.Command{
			{
				Name: "create",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  PeriodFlag,
						Value: 24 * 7,
					},
					&cli.IntFlag{
						Name:  ResultNFlag,
						Value: 24,
					},
					&cli.IntFlag{
						Name:  SkipNFlag,
						Value: 3,
					},
					&cli.StringFlag{
						Name:  NormalisationFlag,
						Value: string(record.ZScore),
					},
					&cli.StringFlag{
						Name:  CompressionEncodingFlag,
						Value: string(model.RomanEncoding),
					},
					&cli.StringFlag{
						Name:  ModelCombineStrategyFlag,
						Value: string(record.InterleaveCombine),
					},
				},
				Action: func(ctx *cli.Context) error {
					normalisationType, err := record.ToNormalisationType(ctx.String(NormalisationFlag))
					if err != nil {
						return err
					}
					dataFilePath := ctx.Args().Get(0)

					dataFile, err := os.Open(dataFilePath)
					if err != nil {
						return err
					}

					encodingType, err := model.ToCompressionEncodingType(ctx.String(CompressionEncodingFlag))
					if err != nil {
						return err
					}

					combineStrat, err := record.ToCombineStrategy(ctx.String(ModelCombineStrategyFlag))
					if err != nil {
						return err
					}

					fmt.Println("Opening data file...")
					reader := csv.NewReader(dataFile)
					records, err := reader.ReadAll()
					if err != nil {
						return err
					}

					parsedRecs := make([]record.Market, len(records)-1)
					for i, rec := range records[1:] {
						parsed, err := record.UnserialiseMarket(rec)
						if err != nil {
							return err
						}
						parsedRecs[i] = *parsed
					}
					fmt.Printf("Parsed %d records.\n", len(parsedRecs))
					opts := splicer.SpliceOptions{
						Period:            ctx.Int(PeriodFlag),
						ResultN:           ctx.Int(ResultNFlag),
						SkipN:             ctx.Int(SkipNFlag),
						NormalisationType: normalisationType,
					}
					fmt.Printf("Splicing data with options: %+v\n", opts)
					importedModel := model.NewCompressionModel(opts, encodingType, combineStrat)

					err = importedModel.AddMarketData(parsedRecs)
					if err != nil {
						return err
					}

					fmt.Println("Data added to model, saving...")
					outputFilePath := ctx.Args().Get(1)
					return importedModel.SaveToFile(outputFilePath)
				},
			},
			{
				Name: "add",
				Action: func(ctx *cli.Context) error {
					dataFilePath := ctx.Args().Get(0)
					modelFilePath := ctx.Args().Get(1)

					importedModel, err := model.LoadCompressionModelFromFile(modelFilePath)
					if err != nil {
						return err
					}
					fmt.Printf("Loaded model with %d records.\n", len(importedModel.Items))

					dataFile, err := os.Open(dataFilePath)
					if err != nil {
						return err
					}

					fmt.Println("Opening data file...")
					reader := csv.NewReader(dataFile)
					records, err := reader.ReadAll()
					if err != nil {
						return err
					}

					parsedRecs := make([]record.Market, len(records)-1)
					for i, rec := range records[1:] {
						parsed, err := record.UnserialiseMarket(rec)
						if err != nil {
							return err
						}
						parsedRecs[i] = *parsed
					}
					fmt.Printf("Parsed %d records.\n", len(parsedRecs))

					err = importedModel.AddMarketData(parsedRecs)
					if err != nil {
						return err
					}

					fmt.Println("Data added to model, saving...")

					return importedModel.SaveToFile(modelFilePath)
				},
			},
			{
				Name: "eval",
				Subcommands: []*cli.Command{
					{
						Name: "heatmap",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:  DownsampleFlag,
								Value: 8,
							},
						},
						Action: func(ctx *cli.Context) error {
							modelFilePath := ctx.Args().Get(0)
							outputFilePath := ctx.Args().Get(1)
							downsampleBy := ctx.Int(DownsampleFlag)

							importedModel, err := model.LoadCompressionModelFromFile(modelFilePath)
							if err != nil {
								return err
							}
							fmt.Printf("Loaded model with %d records.\n", len(importedModel.Items))

							hmap, err := eval.CompressionHeatMap(importedModel, downsampleBy)
							if err != nil {
								return err
							}
							// save model distance map if it was calculated
							err = importedModel.SaveToFile(modelFilePath)
							if err != nil {
								return err
							}

							fmt.Println("Heatmap generated, rendering output.")
							outputFile, err := os.Create(outputFilePath)
							if err != nil {
								return err
							}

							page := components.NewPage()
							page.SetLayout(components.PageCenterLayout)
							page.AddCharts(hmap)

							err = page.Render(outputFile)
							if err != nil {
								return err
							}

							return outputFile.Close()
						},
					},
					{
						Name: "dstvar",
						Action: func(ctx *cli.Context) error {
							modelFilePath := ctx.Args().Get(0)
							outputFilePath := ctx.Args().Get(1)

							importedModel, err := model.LoadCompressionModelFromFile(modelFilePath)
							if err != nil {
								return err
							}
							fmt.Printf("Loaded model with %d records.\n", len(importedModel.Items))

							hmap, err := eval.CompressionDstVarHistogram(importedModel)
							if err != nil {
								return err
							}
							// save model distance map if it was calculated
							err = importedModel.SaveToFile(modelFilePath)
							if err != nil {
								return err
							}

							fmt.Println("Distance result variance graph generated, rendering output.")
							outputFile, err := os.Create(outputFilePath)
							if err != nil {
								return err
							}

							page := components.NewPage()
							page.SetLayout(components.PageCenterLayout)
							page.AddCharts(hmap)

							err = page.Render(outputFile)
							if err != nil {
								return err
							}

							return outputFile.Close()
						},
					},
					{
						Name: "histogram",
						Action: func(ctx *cli.Context) error {
							modelFilePath := ctx.Args().Get(0)
							outputFilePath := ctx.Args().Get(1)

							importedModel, err := model.LoadCompressionModelFromFile(modelFilePath)
							if err != nil {
								return err
							}
							fmt.Printf("Loaded model with %d records.\n", len(importedModel.Items))
							histogram, err := eval.CompressionSizeHistogram(importedModel)
							if err != nil {
								return err
							}

							fmt.Println("Histogram generated, rendering output.")
							outputFile, err := os.Create(outputFilePath)
							if err != nil {
								return err
							}

							err = histogram.Render(outputFile)
							if err != nil {
								return err
							}

							return outputFile.Close()
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
