package fetcher

import (
	"context"
	"encoding/csv"
	"time"

	pio "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"

	"github.com/hubertkaluzny/silly-trader/record"
)

type Polygon struct {
	client *pio.Client
}

var _ Source = (*Polygon)(nil)

func NewPolygonSource(apiKey string) *Polygon {
	c := pio.New(apiKey)

	return &Polygon{c}
}

func (p Polygon) Fetch(target FetchTarget, w csv.Writer) error {
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
		rec := record.Market{
			Timestamp: time.Time(i.Timestamp).UnixMilli(),
			Open:      i.Open,
			High:      i.High,
			Low:       i.Low,
			Close:     i.Close,
			Volume:    i.Volume,
			VWAP:      i.VWAP,
		}
		err := w.Write(record.SerializeMarket(rec))
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
