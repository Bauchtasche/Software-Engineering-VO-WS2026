package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"fahrradverleih/internal/domain"
	"fahrradverleih/internal/handler"
	"fahrradverleih/internal/store"
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
	log.Printf("Fahrradverleih läuft auf http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// seed legt Beispieldaten (Fahrräder, Personen) an, damit das Dashboard nicht leer ist.
func seed(s *store.MemoryStore) {
	bikes := []domain.Bike{
		{Brand: "KTM", Model: "Life Joy", Type: domain.City, FrameSize: "M", FrameNo: "KTM-C-1001"},
		{Brand: "KTM", Model: "Macina Cross", Type: domain.EBike, FrameSize: "L", FrameNo: "KTM-E-2001"},
		{Brand: "Cube", Model: "Nature EXC", Type: domain.Trekking, FrameSize: "M", FrameNo: "CUB-T-3001"},
		{Brand: "Cube", Model: "Stereo 140", Type: domain.Mountainbike, FrameSize: "L", FrameNo: "CUB-M-3002"},
		{Brand: "Cube", Model: "Acid 200", Type: domain.Kinderrad, FrameSize: "XS", FrameNo: "CUB-K-3003"},
		{Brand: "Gazelle", Model: "Orange C7", Type: domain.City, FrameSize: "S", FrameNo: "GAZ-C-4001"},
		{Brand: "Gazelle", Model: "Ultimate C380", Type: domain.EBike, FrameSize: "M", FrameNo: "GAZ-E-4002"},
		{Brand: "Trek", Model: "Marlin 7", Type: domain.Mountainbike, FrameSize: "M", FrameNo: "TRK-M-5001"},
		{Brand: "Trek", Model: "FX 3", Type: domain.Trekking, FrameSize: "XL", FrameNo: "TRK-T-5002"},
		{Brand: "Woom", Model: "Original 4", Type: domain.Kinderrad, FrameSize: "XS", FrameNo: "WOO-K-6001"},
		{Brand: "Riese & Müller", Model: "Charger4", Type: domain.EBike, FrameSize: "L", FrameNo: "RUM-E-7001"},
		{Brand: "Specialized", Model: "Rockhopper", Type: domain.Mountainbike, FrameSize: "S", FrameNo: "SPZ-M-8001"},
	}
	var bikeIDs []int
	for _, bk := range bikes {
		created, err := s.AddBike(bk)
		if err != nil {
			log.Fatalf("seed fahrrad: %v", err)
		}
		bikeIDs = append(bikeIDs, created.ID)
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
	mustRent(s, bikeIDs[2], personIDs[0], now, now.Add(4*time.Hour))
	// laufend, mehrtägig
	mustRent(s, bikeIDs[1], personIDs[1], now, now.Add(3*24*time.Hour))
	// Reservierung morgen
	mustRent(s, bikeIDs[4], personIDs[2], now.Add(20*time.Hour), now.Add(26*time.Hour))
}

func mustRent(s *store.MemoryStore, bikeID, personID int, start, end time.Time) {
	if _, err := s.CreateRental(bikeID, personID, start, end); err != nil {
		log.Fatalf("seed rental: %v", err)
	}
}
