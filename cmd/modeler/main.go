package main

import (
	"encoding/csv"
	"os"

	"github.com/hubertkaluzny/silly-trader/record"
)

func main() {
	// in file, out file
	// interval, ahead count, normalization

	/*
			We need to normalise the data here then
			splice according to the provided interval
		    and calculate result for each splice ahead by some
		    time period (that is also provided)
	*/

	printUsageAndExit := func() {
		os.Exit(1)
	}

	args := os.Args
	if len(args) != 3 {
		printUsageAndExit()
	}

	inF, err := os.Open("inputfile")
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(inF)
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	parsedRecs := make([]record.Market, len(records)-1)
	for i, rec := range records[1:] {
		parsed, err := record.UnserialiseMarket(rec)
		if err != nil {
			panic(err)
		}
		parsedRecs[i] = *parsed
	}

	// read the data into memory and work on it babes
	// but how do we normalize new data?
	// we could do percentage point increases?
	// would need to multiply it by largish number
	// so that character count can affect the compression maybe

	outF, err := os.Create("outputfile")
	if err != nil {
		panic(err)
	}
}
