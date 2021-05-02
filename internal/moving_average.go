package internal

import (
	"fmt"
	"github.com/bitvavo/go-bitvavo-api"
	"github.com/shopspring/decimal"
	"math"
)

const (
	// This is the interval one candle takes
	bvvMAInterval string = "1d"
	// How much candles, this is a bit more then 4 years for day candles
	bvvMALimit int64 = 1500
)

type MABuckets []MABucket

type MABucketValue struct {
	value decimal.Decimal
	exp   float64
}

type MABucket struct {
	timestamp int
	close     MABucketValue
	high      MABucketValue
	low       MABucketValue
	open      MABucketValue
	volume    decimal.Decimal
}

type MovingAverage struct {
	market   *BvvMarket
	offset   float64
	interval string
	limit    int64
	start    int64
	end      int64
	buckets  MABuckets
}

func NewMovingAverage(market *BvvMarket, interval string, limit int64, start int64, end int64) (ma MovingAverage,
	err error) {
	ma = MovingAverage{
		market: market,
		interval: interval,
		limit: limit,
		start: start,
		end: end,
	}
	err = ma.initFromCandles()
	if err != nil {
		return MovingAverage{}, err
	}
	return ma, nil
}

func newMABucketValue(value string) (bv MABucketValue, err error) {
	dValue, err := decimal.NewFromString(value)
	if err != nil {
		return MABucketValue{}, err
	}
	fValue, _ := dValue.Float64()
	return MABucketValue{dValue, math.Exp(fValue)}, nil
}

func newMABucket(candle bitvavo.Candle) (bucket MABucket, err error) {
	lowVal, err := newMABucketValue(candle.Low)
	if err != nil {
		return bucket, err
	}
	openVal, err := newMABucketValue(candle.Open)
	if err != nil {
		return bucket, err
	}
	closeVal, err := newMABucketValue(candle.Close)
	if err != nil {
		return bucket, err
	}
	highVal, err := newMABucketValue(candle.High)
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

func (ma MovingAverage) initFromCandles() (err error) {
	candlesResponse, candlesErr := ma.market.handler.connection.Candles(ma.market.Name(), ma.interval, bvvOptions{})
	if candlesErr != nil {
		fmt.Println(candlesErr)
	} else {
		for _, candle := range candlesResponse {
			ma.market.handler.PrettyPrint(candle)
			bucket, err := newMABucket(candle)
			if err != nil {
				return err
			}
			ma.buckets = append(ma.buckets, bucket)
		}
	}
	return nil
}