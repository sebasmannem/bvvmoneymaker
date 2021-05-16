package internal

import (
	"fmt"
	"github.com/bitvavo/go-bitvavo-api"
	"github.com/sebasmannem/bvvmoneymaker/pkg/moving_average"
	"github.com/shopspring/decimal"
	"sort"
)

type MABuckets []MABucket

type MABucket struct {
	timestamp int
	close     decimal.Decimal
	high      decimal.Decimal
	low       decimal.Decimal
	open      decimal.Decimal
	volume    decimal.Decimal
}

type MAHandler struct {
	market      *BvvMarket
	interval    string
	limit       int64
	buckets     MABuckets
	ema         *moving_average.EMA
}

func NewMAHandler(market *BvvMarket, config bvvMAConfig) (mah *MAHandler, err error) {
	config.SetDefaults()
	ema, err := moving_average.NewEMA(config.Window)
	if err != nil {
		return mah, err
	}
	mah = &MAHandler{
		market: market,
		interval: config.Interval,
		limit: config.Limit,
		ema: ema,
	}
	err = mah.initFromCandles()
	if err != nil {
		return nil, err
	}
	return mah, nil
}

func newMABucket(candle bitvavo.Candle) (bucket MABucket, err error) {
	lowVal, err := decimal.NewFromString(candle.Low)
	if err != nil {
		return bucket, err
	}
	openVal, err := decimal.NewFromString(candle.Open)
	if err != nil {
		return bucket, err
	}
	closeVal, err := decimal.NewFromString(candle.Close)
	if err != nil {
		return bucket, err
	}
	highVal, err := decimal.NewFromString(candle.High)
	if err != nil {
		return bucket, err
	}
	volume, err := decimal.NewFromString(candle.Volume)
	if err != nil {
		return bucket, err
	}
	bucket = MABucket{
		timestamp: candle.Timestamp,
		low: lowVal,
		open: openVal,
		close: closeVal,
		high: highVal,
		volume: volume,
	}
	return bucket, nil
}

func (mab MABucket) Average() (avg decimal.Decimal){
	return mab.low.Add(mab.open).Add(mab.close).Add(mab.high).Div( decimal.NewFromInt(4))
}

// Some helper functions to sort candles by timestamp
type candlesByTS []bitvavo.Candle
func (c candlesByTS) Len() int           { return len(c) }
func (c candlesByTS) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c candlesByTS) Less(i, j int) bool { return c[i].Timestamp < c[j].Timestamp }

func (mah *MAHandler) initFromCandles() (err error) {
	candleOptions := bvvOptions{"limit": fmt.Sprintf("%d", mah.limit)}
	//candleOptions := bvvOptions{}
	candlesResponse, candlesErr := mah.market.handler.connection.Candles(mah.market.Name(), mah.interval, candleOptions)
	if candlesErr != nil {
		return candlesErr
	} else {
		// Sort the candles by Timestamp before processing
		sort.Sort(candlesByTS(candlesResponse))
		for _, candle := range candlesResponse {
			//mah.market.handler.PrettyPrint(candle)
			bucket, err := newMABucket(candle)
			if err != nil {
				return err
			}
			// Storing this last value for .GetOverrated() too
			mah.buckets = append(mah.buckets, bucket)
			mah.ema.AddValue(bucket.Average())
		}
	}
	return nil
}
