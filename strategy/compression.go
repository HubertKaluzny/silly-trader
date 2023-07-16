package strategy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"

	"github.com/hubertkaluzny/silly-trader/record"
	"github.com/hubertkaluzny/silly-trader/splicer"
)

type CompressedItem struct {
	splicer.Splice
	CompressedData []byte `json:"compressed_data"`
}

type CompressionModel struct {
	SpliceOptions splicer.SpliceOptions
	Items         []CompressedItem `json:"items"`
}

func NewCompressionModel(spliceOpts splicer.SpliceOptions) *CompressionModel {
	return &CompressionModel{SpliceOptions: splicer.SpliceOptions{}}
}

func (model *CompressionModel) AddMarketData(data []record.Market) error {
	splices, err := splicer.SpliceData(data, model.SpliceOptions)
	if err != nil {
		return err
	}
	newItems := make([]CompressedItem, len(splices))
	for i, s := range splices {
		item, err := CompressSplice(s)
		if err != nil {
			return err
		}
		newItems[i] = *item
	}
	model.Items = append(model.Items, newItems...)
	return nil
}

func CompressMarketData(data []record.Market) ([]byte, error) {
	var b strings.Builder
	for _, rec := range data {
		b.WriteString(fmt.Sprintf(`%.6f,%.6f,%.6f,%.6f,%6.f,%6.f-`,
			rec.Open,
			rec.High,
			rec.Low,
			rec.Close,
			rec.Volume,
			rec.VWAP))
	}

	var buffBytes []byte
	buff := bytes.NewBuffer(buffBytes)
	writer := gzip.NewWriter(buff)
	_, err := writer.Write([]byte(b.String()))
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func CompressSplice(s splicer.Splice) (*CompressedItem, error) {
	compressed, err := CompressMarketData(s.Data)
	if err != nil {
		return nil, err
	}
	return &CompressedItem{
		Splice:         s,
		CompressedData: compressed,
	}, nil
}
