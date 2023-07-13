package drabber

import (
	"encoding/csv"
	"os"
	"time"
)

type DataGrabber struct {
	Fetcher Fetcher
}

type MarketType string

const (
	Stock  MarketType = "stock"
	Crypto MarketType = "crypto"
)

type GrabTarget struct {
	MarketType MarketType
	From       time.Time
	To         time.Time
	Ticker     string
}

type Fetcher interface {
	Fetch(target GrabTarget, w csv.Writer) error
}

func NewDataGrabber(fetcher Fetcher) *DataGrabber {
	return &DataGrabber{Fetcher: fetcher}
}

func (d DataGrabber) Grab(target GrabTarget, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}

	ws := csv.NewWriter(f)
	err = d.Fetcher.Fetch(target, *ws)
	if err != nil {
		return err
	}
	ws.Flush()
	return nil
}
