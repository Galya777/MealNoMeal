package main

import (
	"math/rand"
	"time"
)

type Banker struct {
	r *rand.Rand
}

func NewBanker() *Banker {
	return &Banker{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// OfferSwap randomly decides whether banker offers a swap (small chance)
func (b *Banker) OfferSwap() bool {
	// 20% chance to offer swap (tunable)
	return b.r.Float64() < 0.20
}

// CalculateOffer computes a banker offer from remaining values.
// This uses average * factor to mimic a banker algorithm.
func (b *Banker) CalculateOffer(values []int) int {
	sum := 0
	count := 0
	for _, v := range values {
		if v > 0 {
			sum += v
			count++
		}
	}
	if count == 0 {
		return 0
	}
	avg := float64(sum) / float64(count)
	// factor between 0.6 and 0.95 (randomized)
	factor := 0.6 + b.r.Float64()*0.35
	return int(avg * factor)
}
