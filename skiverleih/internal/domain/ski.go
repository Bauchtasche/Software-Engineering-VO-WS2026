package domain

import (
	"fmt"
	"strings"
)

// SkiType ist die Kategorie eines Skis; sie bestimmt den Tarif.
type SkiType string

const (
	Allround SkiType = "Allround"
	Carving  SkiType = "Carving"
	Freeride SkiType = "Freeride"
	Race     SkiType = "Race"
	Kinder   SkiType = "Kinder"
)

// AllSkiTypes liefert alle Typen in fester Reihenfolge (für Dropdowns).
func AllSkiTypes() []SkiType {
	return []SkiType{Allround, Carving, Freeride, Race, Kinder}
}

// Valid prüft, ob der Typ bekannt ist.
func (t SkiType) Valid() bool {
	for _, s := range AllSkiTypes() {
		if s == t {
			return true
		}
	}
	return false
}

// Tariff liefert Stunden- und Tagespreis des Typs (siehe README, Abschnitt b).
func (t SkiType) Tariff() Tariff {
	switch t {
	case Allround:
		return Tariff{HourlyCents: 400, DailyCapCents: 2500}
	case Carving:
		return Tariff{HourlyCents: 500, DailyCapCents: 3000}
	case Freeride:
		return Tariff{HourlyCents: 600, DailyCapCents: 3800}
	case Race:
		return Tariff{HourlyCents: 700, DailyCapCents: 4500}
	case Kinder:
		return Tariff{HourlyCents: 250, DailyCapCents: 1500}
	}
	return Tariff{}
}

// Ski ist ein einzelnes verleihbares Paar Ski.
type Ski struct {
	ID       int     `json:"id"`
	Brand    string  `json:"brand"`
	Model    string  `json:"model"`
	Type     SkiType `json:"type"`
	LengthCm int     `json:"lengthCm"`
}

// Normalize entfernt überflüssige Leerzeichen.
func (s *Ski) Normalize() {
	s.Brand = strings.TrimSpace(s.Brand)
	s.Model = strings.TrimSpace(s.Model)
}

// Validate prüft die Pflichtfelder.
func (s Ski) Validate() error {
	if s.Brand == "" || s.Model == "" {
		return fmt.Errorf("%w: Marke und Modell sind Pflicht", ErrValidation)
	}
	if !s.Type.Valid() {
		return fmt.Errorf("%w: unbekannter Ski-Typ %q", ErrValidation, s.Type)
	}
	if s.LengthCm < 70 || s.LengthCm > 220 {
		return fmt.Errorf("%w: Länge muss zwischen 70 und 220 cm liegen", ErrValidation)
	}
	return nil
}

// Label ist eine lesbare Kurzbezeichnung.
func (s Ski) Label() string {
	return fmt.Sprintf("%s %s (%s, %d cm)", s.Brand, s.Model, s.Type, s.LengthCm)
}
