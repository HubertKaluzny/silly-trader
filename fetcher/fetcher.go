package fetcher

import (
	"encoding/csv"
	"os"
	"time"
)

type Fetcher struct {
	Source Source
}

type MarketType string

const (
	Stock  MarketType = "stock"
	Crypto MarketType = "crypto"
)

type FetchTarget struct {
	MarketType MarketType
	From       time.Time
	To         time.Time
	Ticker     string
}

type Source interface {
	Fetch(target FetchTarget, w csv.Writer) error
}

func NewFetcher(source Source) *Fetcher {
	return &Fetcher{Source: source}
}

func (f Fetcher) Fetch(target FetchTarget, dst string) error {
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	ws := csv.NewWriter(dstFile)
	err = f.Source.Fetch(target, *ws)
	if err != nil {
		return err
	}
	ws.Flush()
	return nil
}
