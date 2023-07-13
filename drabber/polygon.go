package drabber

import (
	"context"
	"encoding/csv"
	"strconv"
	"time"

	pio "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
)

type Polygon struct {
	client *pio.Client
}

var _ Fetcher = (*Polygon)(nil)

func NewPolygonFetcher(apiKey string) *Polygon {
	c := pio.New(apiKey)

	return &Polygon{c}
}

func (p Polygon) Fetch(target GrabTarget, w csv.Writer) error {

	ticker := target.Ticker
	if target.MarketType == Crypto {
		ticker = "X:" + ticker
	}

	err := w.Write([]string{"timestamp", "open", "high", "low", "close", "volume", "vwap"})
	if err != nil {
		return err
	}

	params := models.ListAggsParams{
		Ticker:     ticker,
		Multiplier: 1,
		Timespan:   "hour",
		From:       models.Millis(target.From),
		To:         models.Millis(target.To),
	}.WithOrder(models.Desc).WithLimit(50000).WithAdjusted(true)

	iter := p.client.ListAggs(context.TODO(), params)

	for iter.Next() {
		i := iter.Item()
		record := []string{
			strconv.FormatInt(time.Time(i.Timestamp).UnixMilli(), 10),
			strconv.FormatFloat(i.Open, 'G', -1, 64),
			strconv.FormatFloat(i.High, 'G', -1, 64),
			strconv.FormatFloat(i.Low, 'G', -1, 64),
			strconv.FormatFloat(i.Close, 'G', -1, 64),
			strconv.FormatFloat(i.Volume, 'G', -1, 64),
			strconv.FormatFloat(i.VWAP, 'G', -1, 64),
		}
		err := w.Write(record)
		if err != nil {
			return err
		}

	}
	err = iter.Err()
	if err != nil {
		return err
	}

	return nil
}
