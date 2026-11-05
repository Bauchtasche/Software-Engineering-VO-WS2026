package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"skiverleih/internal/domain"
)

func setup(t *testing.T) (*MemoryStore, *time.Time, domain.Ski, domain.Person) {
	t.Helper()
	now := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	s := NewMemoryStore(func() time.Time { return now })
	sk, err := s.AddSki(domain.Ski{Brand: "Head", Model: "V8", Type: domain.Carving, LengthCm: 165})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.AddPerson(domain.Person{FirstName: "Anna", LastName: "Huber",
		Document: domain.IDDocument{Type: domain.Personalausweis, Number: "P1234567"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, &now, sk, p
}

func TestNoDoubleBooking(t *testing.T) {
	s, now, sk, p := setup(t)
	if _, err := s.CreateRental(sk.ID, p.ID, *now, now.Add(4*time.Hour)); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateRental(sk.ID, p.ID, now.Add(2*time.Hour), now.Add(5*time.Hour))
	if !errors.Is(err, ErrConflict) {
		t.Errorf("Überschneidung muss Konflikt sein, got %v", err)
	}
	if _, err := s.CreateRental(sk.ID, p.ID, now.Add(4*time.Hour), now.Add(6*time.Hour)); err != nil {
		t.Errorf("Anschluss-Buchung muss gehen: %v", err)
	}
	if got := len(s.AvailableSkis(now.Add(time.Hour), now.Add(2*time.Hour))); got != 0 {
		t.Errorf("Ski darf nicht verfügbar sein, got %d", got)
	}
}

func TestReturnFreesSki(t *testing.T) {
	s, now, sk, p := setup(t)
	r, _ := s.CreateRental(sk.ID, p.ID, *now, now.Add(8*time.Hour))
	*now = now.Add(2 * time.Hour)
	ret, err := s.ReturnRental(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *ret.FinalCost != 1000 { // 2 h × 5 €
		t.Errorf("Endpreis: got %d, want 1000", *ret.FinalCost)
	}
	if len(s.AvailableSkis(*now, now.Add(time.Hour))) != 1 {
		t.Error("Ski muss nach Rückgabe wieder verfügbar sein")
	}
}

func TestDeleteBlockedByRental(t *testing.T) {
	s, now, sk, p := setup(t)
	s.CreateRental(sk.ID, p.ID, *now, now.Add(time.Hour))
	if err := s.DeleteSki(sk.ID); !errors.Is(err, ErrConflict) {
		t.Error("Ski mit Ausleihe darf nicht gelöscht werden")
	}
	if err := s.DeletePerson(p.ID); !errors.Is(err, ErrConflict) {
		t.Error("Person mit Ausleihe darf nicht gelöscht werden")
	}
}

func TestCancelOnlyReservations(t *testing.T) {
	s, now, sk, p := setup(t)
	active, _ := s.CreateRental(sk.ID, p.ID, *now, now.Add(time.Hour))
	reserved, _ := s.CreateRental(sk.ID, p.ID, now.Add(24*time.Hour), now.Add(26*time.Hour))
	if err := s.CancelRental(active.ID); !errors.Is(err, ErrConflict) {
		t.Error("aktive Ausleihe darf nicht storniert werden")
	}
	if err := s.CancelRental(reserved.ID); err != nil {
		t.Errorf("Reservierung muss stornierbar sein: %v", err)
	}
}

func TestDuplicateDocument(t *testing.T) {
	s, _, _, _ := setup(t)
	_, err := s.AddPerson(domain.Person{FirstName: "B", LastName: "C",
		Document: domain.IDDocument{Type: domain.Personalausweis, Number: "p1234567"}})
	if !errors.Is(err, ErrConflict) {
		t.Error("gleicher Ausweis darf nicht zweimal registriert werden")
	}
}

func TestConcurrentBookingOnlyOneWins(t *testing.T) {
	s, now, sk, p := setup(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.CreateRental(sk.ID, p.ID, *now, now.Add(time.Hour)); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Errorf("genau eine Buchung darf gewinnen, got %d", ok)
	}
}
