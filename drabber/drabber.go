package drabber

import (
	"encoding/csv"
	"time"
)

type DataGrabber struct {
	Fetcher *Fetcher
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

func NewDrabber(fetcher *Fetcher) *DataGrabber {
	return &DataGrabber{Fetcher: fetcher}
}
