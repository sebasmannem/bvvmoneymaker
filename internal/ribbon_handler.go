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

type RibbonHandler struct {
	market    *BvvMarket
	timeframe string
	buckets   MABuckets
	emas      []*moving_average.EMA
	limit     uint
}

func NewRibbonHandler(market *BvvMarket, config bvvRibbonConfig) (rh *RibbonHandler, err error) {
	if ! config.Enabled() {
		return rh, fmt.Errorf("ribbon config is disabled")
	}
	config.Initialize()
	rh = &RibbonHandler{
		market: market,
		timeframe: config.Timeframe,
	}

	var maxWindow uint
	for _, window := range config.Windows {
		rh.emas = append(rh.emas, moving_average.NewEMA(window))
		if maxWindow < window {
			maxWindow = window
		}
	}
	rh.limit = maxWindow + config.PreWarm

	err = rh.initFromCandles()
	if err != nil {
		return nil, err
	}
	return rh, nil
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

func (rh *RibbonHandler) initFromCandles() (err error) {
	candleOptions := bvvOptions{"limit": fmt.Sprintf("%d", rh.limit)}
	//candleOptions := bvvOptions{}
	candlesResponse, candlesErr := rh.market.handler.connection.Candles(rh.market.Name(), rh.timeframe, candleOptions)
	if candlesErr != nil {
		return candlesErr
	} else {
		// Sort the candles by Timestamp before processing
		sort.Sort(candlesByTS(candlesResponse))
		for _, candle := range candlesResponse {
			//rh.market.handler.PrettyPrint(candle)
			bucket, err := newMABucket(candle)
			if err != nil {
				return err
			}
			// Storing this last value for .GetOverrated() too
			rh.buckets = append(rh.buckets, bucket)
			avg := bucket.Average()
			for _, ema := range rh.emas{
				ema.AddValue(avg)
			}
		}
	}
	return nil
}
