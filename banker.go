package main

import (
	"math/rand"
	"time"
)

type Chef struct {
	r *rand.Rand
}

func NewChef() *Chef {
	return &Chef{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// OfferSwap randomly decides whether chef offers a swap (small chance)
func (b *Chef) OfferSwap() bool {
	// 20% chance to offer swap (tunable)
	return b.r.Float64() < 0.20
}

// CalculateOffer computes a chef offer from remaining values.
// This uses average * factor to mimic a chef algorithm.
func (b *Chef) CalculateOffer(values []int) int {
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

// GetRandomChefImage returns a random chef image number (27-50)
func (c *Chef) GetRandomChefImage() int {
	return 27 + c.r.Intn(24) // Random between 27 and 50
}
