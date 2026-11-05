package domain

import (
	"errors"
	"testing"
	"time"
)

func TestTariffPrice(t *testing.T) {
	carving := Carving.Tariff() // 5 €/h, 30 €/Tag
	cases := []struct {
		name string
		d    time.Duration
		want int
	}{
		{"unter 1 h zählt als 1 h", 20 * time.Minute, 500},
		{"3 h", 3 * time.Hour, 1500},
		{"angefangene Stunde zählt voll", 3*time.Hour + time.Minute, 2000},
		{"8 h → Tagesdeckel", 8 * time.Hour, 3000},
		{"genau 24 h", 24 * time.Hour, 3000},
		{"26 h → 1 Tag + 2 h", 26 * time.Hour, 4000},
		{"3 Tage", 72 * time.Hour, 9000},
	}
	for _, c := range cases {
		if got := carving.Price(c.d); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestNewRentalValidation(t *testing.T) {
	now := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	ski := Ski{ID: 1, Brand: "X", Model: "Y", Type: Allround, LengthCm: 160}

	if _, err := NewRental(ski, 1, now.Add(-time.Hour), now.Add(time.Hour), now); !errors.Is(err, ErrValidation) {
		t.Error("Beginn in der Vergangenheit muss abgelehnt werden")
	}
	if _, err := NewRental(ski, 1, now, now.Add(30*time.Minute), now); !errors.Is(err, ErrValidation) {
		t.Error("Dauer < 1 h muss abgelehnt werden")
	}
	if _, err := NewRental(ski, 1, now, now.Add(31*24*time.Hour), now); !errors.Is(err, ErrValidation) {
		t.Error("Dauer > 30 Tage muss abgelehnt werden")
	}
	r, err := NewRental(ski, 1, now.Add(48*time.Hour), now.Add(50*time.Hour), now)
	if err != nil {
		t.Fatalf("Reservierung in der Zukunft muss gehen: %v", err)
	}
	if r.Status(now) != StatusReserved {
		t.Errorf("Status: got %s, want reserviert", r.Status(now))
	}
	if r.PlannedCost != 800 {
		t.Errorf("geplanter Preis: got %d, want 800", r.PlannedCost)
	}
}

func TestRentalStatusAndReturn(t *testing.T) {
	start := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	r := Rental{Start: start, End: start.Add(4 * time.Hour)}

	if s := r.Status(start.Add(time.Hour)); s != StatusActive {
		t.Errorf("got %s, want aktiv", s)
	}
	if s := r.Status(start.Add(5 * time.Hour)); s != StatusOverdue {
		t.Errorf("got %s, want überfällig", s)
	}
	// Rückgabe nach 2,5 h → 3 h abgerechnet (Allround 4 €/h)
	if err := r.Return(start.Add(150*time.Minute), Allround.Tariff()); err != nil {
		t.Fatal(err)
	}
	if *r.FinalCost != 1200 {
		t.Errorf("Endpreis: got %d, want 1200", *r.FinalCost)
	}
	if r.Status(start) != StatusReturned {
		t.Error("Status muss zurückgegeben sein")
	}
	if err := r.Return(start.Add(5*time.Hour), Allround.Tariff()); err == nil {
		t.Error("doppelte Rückgabe muss scheitern")
	}
}

func TestOverlaps(t *testing.T) {
	now := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	r := Rental{Start: now.Add(2 * time.Hour), End: now.Add(6 * time.Hour)}

	if !r.Overlaps(now.Add(5*time.Hour), now.Add(8*time.Hour), now) {
		t.Error("teilweise Überschneidung nicht erkannt")
	}
	if r.Overlaps(now.Add(6*time.Hour), now.Add(8*time.Hour), now) {
		t.Error("direkt anschließend darf keine Überschneidung sein")
	}
	if r.Overlaps(now, now.Add(2*time.Hour), now) {
		t.Error("direkt davor darf keine Überschneidung sein")
	}
}

func TestPersonValidation(t *testing.T) {
	p := Person{FirstName: "Anna", LastName: "Huber",
		Document: IDDocument{Type: Reisepass, Number: " u 123 4567 "}}
	p.Normalize()
	if p.Document.Number != "U1234567" {
		t.Errorf("Normalize: got %q", p.Document.Number)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("gültige Person abgelehnt: %v", err)
	}
	p.Document.Type = "Studentenausweis"
	if err := p.Validate(); !errors.Is(err, ErrValidation) {
		t.Error("ungültiger Ausweistyp muss abgelehnt werden")
	}
}

func TestAttachFile(t *testing.T) {
	var d IDDocument
	if err := d.AttachFile("x.exe", "application/octet-stream", []byte{1}); err == nil {
		t.Error("falsches Format muss abgelehnt werden")
	}
	if err := d.AttachFile("ausweis.png", "image/png", []byte{1, 2, 3}); err != nil || !d.HasFile {
		t.Errorf("PNG muss akzeptiert werden: %v", err)
	}
}
