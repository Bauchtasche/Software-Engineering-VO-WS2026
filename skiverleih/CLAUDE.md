# Ski-Verleih – Prototype Project

Nur Prototypen. Priorität: schnell verständlich, leicht änderbar.
Die fachliche Spezifikation steht in `README.md` – sie ist die Quelle der Wahrheit (Spec-Driven Development).
Vor jeder Änderung am Code zuerst die Spec in `README.md` anpassen, dann implementieren, dann testen.

## Stack
- Backend: Go ('objektorientiert': Structs + Methoden, Interfaces wo sinnvoll)
- DB: In-Memory (kein Postgres/SQLite, kein ORM)
- Frontend: reines HTML/JS (`web/`), wird vom Go-Server ausgeliefert
- Versionierung: lokales Git, bei Bedarf GitHub

## Commands
- Backend starten: `go run .` → http://localhost:8080
- Tests: `go test ./...`
- Format: `go fmt ./...`
- Lint (falls eingerichtet): `golangci-lint run`

## Architektur (Go)
- `main.go` → Einstieg, HTTP-Server, Routing, Seed-Daten
- `internal/domain/` → Kern-Structs + Methoden (Ski, Person, Ausweis, Ausleihe, Tarif)
- `internal/store/` → In-Memory-Store (Maps + `sync.RWMutex`)
- `internal/handler/` → HTTP-Handler (JSON-API unter `/api/...`)
- `web/` → `index.html`, `style.css`, `app.js`
- Keine globalen Variablen außer dem Store-Singleton wenn nötig
- Fehler mit `fmt.Errorf("...: %w", err)` wrappen, nicht verschlucken

## Konventionen
- Methoden an Structs hängen statt freier Funktionen, wo es Sinn ergibt
  (z.B. `SkiType.Tariff()`, `Tariff.Price()`, `Rental.Status()`, `Rental.Overlaps()`)
- Geldbeträge immer als `int` in Cent, nie als float
- Zeiten im Backend als `time.Time` (RFC 3339 in JSON), Frontend rechnet in lokale Zeit um
- Interfaces nur bei echtem Bedarf
- Kein Dependency Injection Framework
- Keine neuen Dependencies ohne Nachfrage (aktuell: nur Standardbibliothek)
- In-Memory-Store muss thread-safe sein
- Keine Secrets im Code

## Frontend
- Möglichst minimales JS, klare Trennung von Backend-API
- API-Calls immer gegen die Go-Endpoints, nie direkt gegen In-Memory-Strukturen
- Benutzerfreundlich: Fehlermeldungen vom Backend sichtbar anzeigen, Preis live vor dem Buchen zeigen

## Regeln
- Keine generierten Dateien von Hand editieren
- Commit-Messages: kurz und imperativ
- Bei Unklarheit fragen statt raten
- Prototyp-Fokus: lieber einfach und klar als elegant und komplex
