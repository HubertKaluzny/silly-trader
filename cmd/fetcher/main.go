package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hubertkaluzny/silly-trader/fetcher"
)

func main() {
	printUsageAndExit := func() {
		fmt.Println("Usage: data-grabber <stock|crypto> <ticker> <from-date yyyy-mm-dd> <to-date yyyy-mm-dd> <destination file>")
		fmt.Println("Example: data-grabber stock AAPL 2020-01-01 2020-12-31 data.csv")
		os.Exit(1)
	}

	args := os.Args
	if len(args) != 6 {
		printUsageAndExit()
	}

	polygonKey, hasPolygonKey := os.LookupEnv("POLYGON_API_KEY")
	if !hasPolygonKey {
		fmt.Println("POLYGON_API_KEY environment variable is not set")
		printUsageAndExit()
	}

	polygonFetcher := fetcher.NewPolygonFetcher(polygonKey)
	grabber := fetcher.NewDataGrabber(polygonFetcher)

	target := fetcher.GrabTarget{}
	if args[1] == "stock" {
		target.MarketType = fetcher.Stock
	} else if args[1] == "crypto" {
		target.MarketType = fetcher.Crypto
	} else {
		printUsageAndExit()
	}

	target.Ticker = strings.ToUpper(args[2])

	fromDate, err := time.Parse("2006-01-02", args[3])
	if err != nil {
		fmt.Println(err)
		printUsageAndExit()
	}
	target.From = fromDate

	toDate, err := time.Parse("2006-01-02", args[4])
	if err != nil {

		fmt.Println(err)
		printUsageAndExit()
	}
	target.To = toDate

	err = grabber.Grab(target, args[5])

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
