package domain

import (
	"fmt"
	"time"
)

// RentalStatus wird immer aus den Zeiten abgeleitet, nie gespeichert.
type RentalStatus string

const (
	StatusReserved RentalStatus = "reserviert"
	StatusActive   RentalStatus = "aktiv"
	StatusOverdue  RentalStatus = "überfällig"
	StatusReturned RentalStatus = "zurückgegeben"
)

const (
	MinRentalDuration = time.Hour
	MaxRentalDuration = 30 * 24 * time.Hour
	StartGracePeriod  = 15 * time.Minute
)

// Rental ist eine Ausleihe bzw. Reservierung eines Fahrrads durch eine Person.
type Rental struct {
	ID          int        `json:"id"`
	BikeID      int        `json:"bikeId"`
	PersonID    int        `json:"personId"`
	Start       time.Time  `json:"start"`
	End         time.Time  `json:"end"`
	PlannedCost int        `json:"plannedCents"`
	ReturnedAt  *time.Time `json:"returnedAt,omitempty"`
	FinalCost   *int       `json:"finalCents,omitempty"`
}

// NewRental prüft den Zeitraum und berechnet den geplanten Preis.
func NewRental(bike Bike, personID int, start, end, now time.Time) (Rental, error) {
	if start.Before(now.Add(-StartGracePeriod)) {
		return Rental{}, fmt.Errorf("%w: Beginn darf nicht in der Vergangenheit liegen", ErrValidation)
	}
	d := end.Sub(start)
	if d < MinRentalDuration {
		return Rental{}, fmt.Errorf("%w: Mindestdauer ist 1 Stunde", ErrValidation)
	}
	if d > MaxRentalDuration {
		return Rental{}, fmt.Errorf("%w: Höchstdauer ist 30 Tage", ErrValidation)
	}
	return Rental{
		BikeID:      bike.ID,
		PersonID:    personID,
		Start:       start,
		End:         end,
		PlannedCost: bike.Type.Tariff().Price(d),
	}, nil
}

// Status leitet den aktuellen Zustand ab (siehe README-Tabelle).
func (r Rental) Status(now time.Time) RentalStatus {
	switch {
	case r.ReturnedAt != nil:
		return StatusReturned
	case now.Before(r.Start):
		return StatusReserved
	case now.After(r.End):
		return StatusOverdue
	default:
		return StatusActive
	}
}

// BlocksBike sagt, ob die Ausleihe das Fahrrad (noch) belegt.
func (r Rental) BlocksBike() bool {
	return r.ReturnedAt == nil
}

// Overlaps prüft, ob der Zeitraum [start, end) mit dieser Ausleihe kollidiert.
// Zurückgegebene Ausleihen blockieren nicht mehr. Überfällige blockieren,
// bis sie zurückgegeben sind.
func (r Rental) Overlaps(start, end, now time.Time) bool {
	if !r.BlocksBike() {
		return false
	}
	occupiedUntil := r.End
	if r.Status(now) == StatusOverdue {
		occupiedUntil = maxTime(now, r.End).Add(MinRentalDuration)
	}
	return start.Before(occupiedUntil) && r.Start.Before(end)
}

// Return bucht die Rückgabe und berechnet den Endpreis mit der tatsächlichen Dauer.
func (r *Rental) Return(at time.Time, t Tariff) error {
	if r.ReturnedAt != nil {
		return fmt.Errorf("%w: Ausleihe wurde bereits zurückgegeben", ErrValidation)
	}
	if at.Before(r.Start) {
		return fmt.Errorf("%w: Ausleihe hat noch nicht begonnen – bitte stornieren", ErrValidation)
	}
	cost := t.Price(at.Sub(r.Start))
	r.ReturnedAt = &at
	r.FinalCost = &cost
	return nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
