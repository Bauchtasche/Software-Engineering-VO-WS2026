package domain

import (
	"fmt"
	"strings"
)

// BikeType ist die Kategorie eines Fahrrads; sie bestimmt den Tarif.
type BikeType string

const (
	City         BikeType = "City"
	Trekking     BikeType = "Trekking"
	Mountainbike BikeType = "Mountainbike"
	EBike        BikeType = "E-Bike"
	Kinderrad    BikeType = "Kinderrad"
)

// AllBikeTypes liefert alle Typen in fester Reihenfolge (für Dropdowns).
func AllBikeTypes() []BikeType {
	return []BikeType{City, Trekking, Mountainbike, EBike, Kinderrad}
}

// Valid prüft, ob der Typ bekannt ist.
func (t BikeType) Valid() bool {
	for _, b := range AllBikeTypes() {
		if b == t {
			return true
		}
	}
	return false
}

// Tariff liefert Stunden- und Tagespreis des Typs (siehe README).
func (t BikeType) Tariff() Tariff {
	switch t {
	case City:
		return Tariff{HourlyCents: 300, DailyCapCents: 1500}
	case Trekking:
		return Tariff{HourlyCents: 400, DailyCapCents: 2000}
	case Mountainbike:
		return Tariff{HourlyCents: 600, DailyCapCents: 3200}
	case EBike:
		return Tariff{HourlyCents: 900, DailyCapCents: 4500}
	case Kinderrad:
		return Tariff{HourlyCents: 200, DailyCapCents: 1000}
	}
	return Tariff{}
}

// FrameSize ist die Rahmengröße.
type FrameSize string

// AllFrameSizes liefert alle erlaubten Rahmengrößen.
func AllFrameSizes() []FrameSize {
	return []FrameSize{"XS", "S", "M", "L", "XL"}
}

// Valid prüft die Rahmengröße.
func (f FrameSize) Valid() bool {
	for _, s := range AllFrameSizes() {
		if s == f {
			return true
		}
	}
	return false
}

// Bike ist ein einzelnes verleihbares Fahrrad.
type Bike struct {
	ID        int       `json:"id"`
	Brand     string    `json:"brand"`
	Model     string    `json:"model"`
	Type      BikeType  `json:"type"`
	FrameSize FrameSize `json:"frameSize"`
	FrameNo   string    `json:"frameNo"` // Rahmennummer, eindeutig
}

// Normalize entfernt überflüssige Leerzeichen.
func (b *Bike) Normalize() {
	b.Brand = strings.TrimSpace(b.Brand)
	b.Model = strings.TrimSpace(b.Model)
	b.FrameSize = FrameSize(strings.ToUpper(strings.TrimSpace(string(b.FrameSize))))
	b.FrameNo = strings.ToUpper(strings.TrimSpace(b.FrameNo))
}

// Validate prüft die Pflichtfelder.
func (b Bike) Validate() error {
	if b.Brand == "" || b.Model == "" {
		return fmt.Errorf("%w: Marke und Modell sind Pflicht", ErrValidation)
	}
	if !b.Type.Valid() {
		return fmt.Errorf("%w: unbekannter Fahrrad-Typ %q", ErrValidation, b.Type)
	}
	if !b.FrameSize.Valid() {
		return fmt.Errorf("%w: Rahmengröße muss XS, S, M, L oder XL sein", ErrValidation)
	}
	if len(b.FrameNo) < 4 {
		return fmt.Errorf("%w: Rahmennummer ist Pflicht (mind. 4 Zeichen)", ErrValidation)
	}
	return nil
}

// Label ist eine lesbare Kurzbezeichnung.
func (b Bike) Label() string {
	return fmt.Sprintf("%s %s (%s, %s)", b.Brand, b.Model, b.Type, b.FrameSize)
}
