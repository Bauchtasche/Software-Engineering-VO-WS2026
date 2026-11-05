package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"fahrradverleih/internal/domain"
)

var (
	ErrNotFound = errors.New("nicht gefunden")
	ErrConflict = errors.New("Konflikt")
)

// MemoryStore hält alle Daten im Speicher. Alle Methoden sind thread-safe.
type MemoryStore struct {
	mu      sync.RWMutex
	bikes   map[int]domain.Bike
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
		bikes:   map[int]domain.Bike{},
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

// ---------- Fahrräder ----------

func (s *MemoryStore) AddBike(bike domain.Bike) (domain.Bike, error) {
	bike.Normalize()
	if err := bike.Validate(); err != nil {
		return domain.Bike{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, other := range s.bikes {
		if other.FrameNo == bike.FrameNo {
			return domain.Bike{}, fmt.Errorf("%w: Rahmennummer %s ist schon vergeben", ErrConflict, bike.FrameNo)
		}
	}
	bike.ID = s.newID()
	s.bikes[bike.ID] = bike
	return bike, nil
}

func (s *MemoryStore) Bikes() []domain.Bike {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Bike, 0, len(s.bikes))
	for _, bk := range s.bikes {
		out = append(out, bk)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *MemoryStore) Bike(id int) (domain.Bike, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bk, ok := s.bikes[id]
	if !ok {
		return domain.Bike{}, fmt.Errorf("Fahrrad %d: %w", id, ErrNotFound)
	}
	return bk, nil
}

func (s *MemoryStore) DeleteBike(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bikes[id]; !ok {
		return fmt.Errorf("Fahrrad %d: %w", id, ErrNotFound)
	}
	for _, r := range s.rentals {
		if r.BikeID == id && r.BlocksBike() {
			return fmt.Errorf("%w: Fahrrad hat eine laufende oder geplante Ausleihe", ErrConflict)
		}
	}
	delete(s.bikes, id)
	return nil
}

// bikeAvailableLocked prüft, ob das Fahrrad im Zeitraum frei ist. Aufrufer muss Lock halten.
func (s *MemoryStore) bikeAvailableLocked(bikeID int, start, end time.Time) bool {
	now := s.now()
	for _, r := range s.rentals {
		if r.BikeID == bikeID && r.Overlaps(start, end, now) {
			return false
		}
	}
	return true
}

// AvailableBikes liefert alle Fahrräder, die im Zeitraum frei sind.
func (s *MemoryStore) AvailableBikes(start, end time.Time) []domain.Bike {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.Bike{}
	for _, bk := range s.bikes {
		if s.bikeAvailableLocked(bk.ID, start, end) {
			out = append(out, bk)
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
		if r.PersonID == id && r.BlocksBike() {
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

// CreateRental prüft Fahrrad, Person und Überschneidung atomar und legt die Ausleihe an.
func (s *MemoryStore) CreateRental(bikeID, personID int, start, end time.Time) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bike, ok := s.bikes[bikeID]
	if !ok {
		return domain.Rental{}, fmt.Errorf("Fahrrad %d: %w", bikeID, ErrNotFound)
	}
	if _, ok := s.persons[personID]; !ok {
		return domain.Rental{}, fmt.Errorf("Person %d: %w", personID, ErrNotFound)
	}
	r, err := domain.NewRental(bike, personID, start, end, s.now())
	if err != nil {
		return domain.Rental{}, err
	}
	if !s.bikeAvailableLocked(bikeID, start, end) {
		return domain.Rental{}, fmt.Errorf("%w: Fahrrad ist in diesem Zeitraum schon verliehen", ErrConflict)
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
	bike, ok := s.bikes[r.BikeID]
	if !ok {
		return domain.Rental{}, fmt.Errorf("Fahrrad %d: %w", r.BikeID, ErrNotFound)
	}
	if err := r.Return(s.now(), bike.Type.Tariff()); err != nil {
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
