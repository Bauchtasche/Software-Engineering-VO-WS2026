package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"skiverleih/internal/domain"
)

var (
	ErrNotFound = errors.New("nicht gefunden")
	ErrConflict = errors.New("Konflikt")
)

// MemoryStore hält alle Daten im Speicher. Alle Methoden sind thread-safe.
type MemoryStore struct {
	mu      sync.RWMutex
	skis    map[int]domain.Ski
	persons map[int]domain.Person
	rentals map[int]domain.Rental
	nextID  int
	now     func() time.Time
}

// NewMemoryStore legt einen leeren Store an. now ist austauschbar (für Tests).
func NewMemoryStore(now func() time.Time) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	return &MemoryStore{
		skis:    map[int]domain.Ski{},
		persons: map[int]domain.Person{},
		rentals: map[int]domain.Rental{},
		nextID:  1,
		now:     now,
	}
}

// Now liefert die Uhrzeit des Stores.
func (s *MemoryStore) Now() time.Time { return s.now() }

func (s *MemoryStore) newID() int {
	id := s.nextID
	s.nextID++
	return id
}

// ---------- Ski ----------

func (s *MemoryStore) AddSki(ski domain.Ski) (domain.Ski, error) {
	ski.Normalize()
	if err := ski.Validate(); err != nil {
		return domain.Ski{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ski.ID = s.newID()
	s.skis[ski.ID] = ski
	return ski, nil
}

func (s *MemoryStore) Skis() []domain.Ski {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Ski, 0, len(s.skis))
	for _, sk := range s.skis {
		out = append(out, sk)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *MemoryStore) Ski(id int) (domain.Ski, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sk, ok := s.skis[id]
	if !ok {
		return domain.Ski{}, fmt.Errorf("Ski %d: %w", id, ErrNotFound)
	}
	return sk, nil
}

func (s *MemoryStore) DeleteSki(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.skis[id]; !ok {
		return fmt.Errorf("Ski %d: %w", id, ErrNotFound)
	}
	for _, r := range s.rentals {
		if r.SkiID == id && r.BlocksSki() {
			return fmt.Errorf("%w: Ski hat eine laufende oder geplante Ausleihe", ErrConflict)
		}
	}
	delete(s.skis, id)
	return nil
}

// SkiAvailable prüft, ob der Ski im Zeitraum frei ist. Aufrufer muss Lock halten.
func (s *MemoryStore) skiAvailableLocked(skiID int, start, end time.Time) bool {
	now := s.now()
	for _, r := range s.rentals {
		if r.SkiID == skiID && r.Overlaps(start, end, now) {
			return false
		}
	}
	return true
}

// AvailableSkis liefert alle Ski, die im Zeitraum frei sind.
func (s *MemoryStore) AvailableSkis(start, end time.Time) []domain.Ski {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.Ski{}
	for _, sk := range s.skis {
		if s.skiAvailableLocked(sk.ID, start, end) {
			out = append(out, sk)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ---------- Personen ----------

func (s *MemoryStore) AddPerson(p domain.Person) (domain.Person, error) {
	p.Normalize()
	p.Document.File, p.Document.HasFile, p.Document.FileName, p.Document.ContentType = nil, false, "", ""
	if err := p.Validate(); err != nil {
		return domain.Person{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, other := range s.persons {
		if other.Document.Type == p.Document.Type && other.Document.Number == p.Document.Number {
			return domain.Person{}, fmt.Errorf("%w: dieser Ausweis ist schon bei %s registriert", ErrConflict, other.FullName())
		}
	}
	p.ID = s.newID()
	s.persons[p.ID] = p
	return p, nil
}

func (s *MemoryStore) Persons() []domain.Person {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Person, 0, len(s.persons))
	for _, p := range s.persons {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *MemoryStore) Person(id int) (domain.Person, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.persons[id]
	if !ok {
		return domain.Person{}, fmt.Errorf("Person %d: %w", id, ErrNotFound)
	}
	return p, nil
}

func (s *MemoryStore) DeletePerson(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.persons[id]; !ok {
		return fmt.Errorf("Person %d: %w", id, ErrNotFound)
	}
	for _, r := range s.rentals {
		if r.PersonID == id && r.BlocksSki() {
			return fmt.Errorf("%w: Person hat eine laufende oder geplante Ausleihe", ErrConflict)
		}
	}
	delete(s.persons, id)
	return nil
}

// AttachDocument speichert den Ausweis-Upload einer Person.
func (s *MemoryStore) AttachDocument(personID int, name, contentType string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.persons[personID]
	if !ok {
		return fmt.Errorf("Person %d: %w", personID, ErrNotFound)
	}
	if err := p.Document.AttachFile(name, contentType, data); err != nil {
		return err
	}
	s.persons[personID] = p
	return nil
}

// ---------- Ausleihen ----------

// CreateRental prüft Ski, Person und Überschneidung atomar und legt die Ausleihe an.
func (s *MemoryStore) CreateRental(skiID, personID int, start, end time.Time) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ski, ok := s.skis[skiID]
	if !ok {
		return domain.Rental{}, fmt.Errorf("Ski %d: %w", skiID, ErrNotFound)
	}
	if _, ok := s.persons[personID]; !ok {
		return domain.Rental{}, fmt.Errorf("Person %d: %w", personID, ErrNotFound)
	}
	r, err := domain.NewRental(ski, personID, start, end, s.now())
	if err != nil {
		return domain.Rental{}, err
	}
	if !s.skiAvailableLocked(skiID, start, end) {
		return domain.Rental{}, fmt.Errorf("%w: Ski ist in diesem Zeitraum schon verliehen", ErrConflict)
	}
	r.ID = s.newID()
	s.rentals[r.ID] = r
	return r, nil
}

func (s *MemoryStore) Rentals() []domain.Rental {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Rental, 0, len(s.rentals))
	for _, r := range s.rentals {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// ReturnRental bucht die Rückgabe jetzt und berechnet den Endpreis.
func (s *MemoryStore) ReturnRental(id int) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rentals[id]
	if !ok {
		return domain.Rental{}, fmt.Errorf("Ausleihe %d: %w", id, ErrNotFound)
	}
	ski, ok := s.skis[r.SkiID]
	if !ok {
		return domain.Rental{}, fmt.Errorf("Ski %d: %w", r.SkiID, ErrNotFound)
	}
	if err := r.Return(s.now(), ski.Type.Tariff()); err != nil {
		return domain.Rental{}, err
	}
	s.rentals[id] = r
	return r, nil
}

// CancelRental storniert eine Reservierung, die noch nicht begonnen hat.
func (s *MemoryStore) CancelRental(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rentals[id]
	if !ok {
		return fmt.Errorf("Ausleihe %d: %w", id, ErrNotFound)
	}
	if r.Status(s.now()) != domain.StatusReserved {
		return fmt.Errorf("%w: nur Reservierungen können storniert werden", ErrConflict)
	}
	delete(s.rentals, id)
	return nil
}
