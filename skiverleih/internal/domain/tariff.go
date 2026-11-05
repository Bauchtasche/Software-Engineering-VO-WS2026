package domain

import (
	"math"
	"time"
)

// Tariff: Stundenpreis mit Tagesdeckel, alle Beträge in Cent.
type Tariff struct {
	HourlyCents   int `json:"hourlyCents"`
	DailyCapCents int `json:"dailyCapCents"`
}

// BillableHours rundet auf angefangene Stunden auf, mindestens 1.
func BillableHours(d time.Duration) int {
	h := int(math.Ceil(d.Hours()))
	if h < 1 {
		return 1
	}
	return h
}

// Price berechnet: tage × tagespreis + min(reststunden × stundenpreis, tagespreis).
func (t Tariff) Price(d time.Duration) int {
	hours := BillableHours(d)
	days := hours / 24
	rest := hours % 24

	restCost := rest * t.HourlyCents
	if restCost > t.DailyCapCents {
		restCost = t.DailyCapCents
	}
	return days*t.DailyCapCents + restCost
}
