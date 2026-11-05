package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"skiverleih/internal/domain"
	"skiverleih/internal/handler"
	"skiverleih/internal/store"
)

//go:embed web
var webFiles embed.FS

func main() {
	s := store.NewMemoryStore(time.Now)
	seed(s)

	mux := http.NewServeMux()
	handler.New(s).Register(mux)

	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(static)))

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("Ski-Verleih läuft auf http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// seed legt Beispieldaten an, damit das Dashboard nicht leer ist.
func seed(s *store.MemoryStore) {
	skis := []domain.Ski{
		{Brand: "Atomic", Model: "Redster S9", Type: domain.Race, LengthCm: 165},
		{Brand: "Atomic", Model: "Bent 90", Type: domain.Freeride, LengthCm: 180},
		{Brand: "Fischer", Model: "RC4 The Curv", Type: domain.Carving, LengthCm: 160},
		{Brand: "Fischer", Model: "Ranger 96", Type: domain.Freeride, LengthCm: 178},
		{Brand: "Head", Model: "Shape e-V8", Type: domain.Allround, LengthCm: 163},
		{Brand: "Head", Model: "Supershape e-Magnum", Type: domain.Carving, LengthCm: 170},
		{Brand: "Rossignol", Model: "Experience 78", Type: domain.Allround, LengthCm: 168},
		{Brand: "Rossignol", Model: "Hero Junior", Type: domain.Kinder, LengthCm: 120},
		{Brand: "Völkl", Model: "Deacon 76", Type: domain.Carving, LengthCm: 166},
		{Brand: "Völkl", Model: "Racetiger Junior", Type: domain.Kinder, LengthCm: 110},
		{Brand: "Blizzard", Model: "Rustler 9", Type: domain.Freeride, LengthCm: 180},
		{Brand: "Salomon", Model: "S/Max 8", Type: domain.Allround, LengthCm: 160},
	}
	var skiIDs []int
	for _, sk := range skis {
		created, err := s.AddSki(sk)
		if err != nil {
			log.Fatalf("seed ski: %v", err)
		}
		skiIDs = append(skiIDs, created.ID)
	}

	persons := []domain.Person{
		{FirstName: "Anna", LastName: "Huber", Phone: "+43 660 1234567", Email: "anna.huber@example.at",
			Document: domain.IDDocument{Type: domain.Personalausweis, Number: "P1234567"}},
		{FirstName: "Lukas", LastName: "Gruber", Phone: "+43 664 7654321", Email: "lukas.gruber@example.at",
			Document: domain.IDDocument{Type: domain.Reisepass, Number: "U9876543"}},
		{FirstName: "Sophie", LastName: "Mayr", Phone: "+49 151 2223334", Email: "sophie.mayr@example.de",
			Document: domain.IDDocument{Type: domain.Fuehrerschein, Number: "B072RRE2I55"}},
		{FirstName: "Jonas", LastName: "Wimmer", Phone: "+43 676 5556667",
			Document: domain.IDDocument{Type: domain.Personalausweis, Number: "P7654321"}},
	}
	var personIDs []int
	for _, p := range persons {
		created, err := s.AddPerson(p)
		if err != nil {
			log.Fatalf("seed person: %v", err)
		}
		personIDs = append(personIDs, created.ID)
	}

	now := time.Now().Truncate(time.Minute)
	// laufend, heute fällig
	mustRent(s, skiIDs[2], personIDs[0], now, now.Add(4*time.Hour))
	// laufend, mehrtägig
	mustRent(s, skiIDs[1], personIDs[1], now, now.Add(3*24*time.Hour))
	// Reservierung morgen
	mustRent(s, skiIDs[7], personIDs[2], now.Add(20*time.Hour), now.Add(26*time.Hour))
}

func mustRent(s *store.MemoryStore, skiID, personID int, start, end time.Time) {
	if _, err := s.CreateRental(skiID, personID, start, end); err != nil {
		log.Fatalf("seed rental: %v", err)
	}
}
