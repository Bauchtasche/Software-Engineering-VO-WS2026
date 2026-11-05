# Ski-Verleih

Prototyp eines Ski-Verleihs: Go-Backend mit In-Memory-Speicher und HTML/JS-Frontend.
Entwickelt mit Spec-Driven Development: diese Datei ist die Spezifikation, `CLAUDE.md` enthält die technischen Regeln.

## Starten

```bash
go run .
# Browser: http://localhost:8080
go test ./...
```

Beim Start werden Beispieldaten (Ski und Personen) angelegt. Alle Daten gehen beim Beenden verloren (In-Memory).

## Anforderungen

### a) Grundsystem mit Dashboard (2 Punkte)
- **Ski verwalten:** anlegen (Marke, Modell, Typ, Länge in cm), auflisten, löschen.
  Ein Ski mit laufender oder geplanter Ausleihe kann nicht gelöscht werden.
- **Personen verwalten:** anlegen (Vorname, Nachname, Telefon, E-Mail), auflisten, löschen.
  Eine Person mit laufender oder geplanter Ausleihe kann nicht gelöscht werden.
- **Ausweis:** Jede Person hat einen Ausweis mit
  - Typ: `Personalausweis`, `Reisepass` oder `Führerschein` (Pflicht)
  - Ausweisnummer (Pflicht, 5–20 Zeichen, Buchstaben/Ziffern)
  - optional ein Upload als Bild (JPG/PNG) oder PDF, max. 5 MB. Der Upload kann angesehen werden.
- **Ausleihen:** Person + Ski + Zeitraum wählen, Rückgabe buchen, geplante Ausleihen stornieren.
- **Dashboard:** auf einen Blick
  - Ski gesamt / jetzt verfügbar / jetzt verliehen
  - registrierte Personen
  - überfällige Ausleihen (Rückgabe-Termin überschritten) – farblich hervorgehoben
  - kommende Reservierungen
  - Umsatz aus abgeschlossenen Ausleihen
  - Liste „Heute fällig“ und „Überfällig“ mit Direkt-Button „Rückgabe“
- **Benutzerfreundlichkeit:** Navigation über Tabs, Formulare mit Validierung, Fehlermeldungen des
  Servers werden angezeigt, Suchfeld in den Listen, Bestätigung vor dem Löschen.

### b) Flexible Zeiten und Stundenpreise (3 Punkte)
- **Verleihbeginn frei wählbar:** sofort oder in der Zukunft (= Reservierung).
  Beginn darf höchstens 15 Minuten in der Vergangenheit liegen.
- **Verleihdauer frei wählbar:** Ende frei wählbar, mindestens 1 Stunde, höchstens 30 Tage.
  Schnellauswahl im Formular: 2 h, 4 h, 1 Tag, 3 Tage, 1 Woche.
- **Keine Doppelbuchung:** Ein Ski kann für sich überschneidende Zeiträume nicht zweimal verliehen werden.
  Im Formular werden nur Ski angeboten, die im gewählten Zeitraum frei sind.
  Ein überfälliger Ski gilt bis zur Rückgabe als belegt (mindestens bis jetzt + 1 Stunde).
- **Stundenpreis mit Tagesdeckel:** Jeder Ski-Typ hat einen Stundenpreis und einen Tagespreis.
  - Angefangene Stunden zählen als volle Stunden, mindestens 1 Stunde.
  - Je volle 24 Stunden wird der Tagespreis verrechnet.
  - Für die restlichen Stunden: Stundenpreis × Stunden, aber höchstens der Tagespreis.
  - Formel: `preis = tage × tagespreis + min(reststunden × stundenpreis, tagespreis)`

| Typ       | pro Stunde | pro Tag (Deckel) |
|-----------|-----------:|-----------------:|
| Allround  |     4,00 € |          25,00 € |
| Carving   |     5,00 € |          30,00 € |
| Freeride  |     6,00 € |          38,00 € |
| Race      |     7,00 € |          45,00 € |
| Kinder    |     2,50 € |          15,00 € |

  Beispiel Carving: 3 h → 15 €; 8 h → 30 € (Deckel statt 40 €); 26 h → 30 € + 10 € = 40 €.
- **Preisvorschau:** Vor dem Buchen wird der voraussichtliche Preis live angezeigt.
- **Abrechnung bei Rückgabe:** Der Endpreis wird mit der tatsächlichen Dauer (Beginn bis Rückgabe) berechnet.
  Bei Rückgabe vor dem Beginn (Reservierung nie angetreten) → Stornierung statt Rückgabe.

### Status einer Ausleihe
| Status        | Bedingung                                  |
|---------------|--------------------------------------------|
| `reserviert`  | jetzt < Beginn                             |
| `aktiv`       | Beginn ≤ jetzt ≤ Ende, nicht zurückgegeben |
| `überfällig`  | jetzt > Ende, nicht zurückgegeben          |
| `zurückgegeben` | Rückgabe gebucht                         |

## REST-API

| Methode | Pfad                              | Beschreibung |
|---------|-----------------------------------|--------------|
| GET     | `/api/dashboard`                  | Kennzahlen + fällige/überfällige Ausleihen |
| GET     | `/api/tariffs`                    | Ski-Typen mit Tarif |
| GET     | `/api/skis`                       | alle Ski inkl. aktuellem Status |
| GET     | `/api/skis/available?start=&end=` | im Zeitraum freie Ski |
| POST    | `/api/skis`                       | Ski anlegen |
| DELETE  | `/api/skis/{id}`                  | Ski löschen |
| GET     | `/api/persons`                    | alle Personen |
| POST    | `/api/persons`                    | Person inkl. Ausweis (Typ + Nr.) anlegen |
| DELETE  | `/api/persons/{id}`               | Person löschen |
| POST    | `/api/persons/{id}/document`      | Ausweis-Datei hochladen (multipart, Feld `file`) |
| GET     | `/api/persons/{id}/document`      | Ausweis-Datei ansehen |
| GET     | `/api/rentals`                    | alle Ausleihen inkl. Status |
| GET     | `/api/quote?skiId=&start=&end=`   | Preisvorschau |
| POST    | `/api/rentals`                    | Ausleihe/Reservierung anlegen |
| POST    | `/api/rentals/{id}/return`        | Rückgabe buchen (Endpreis) |
| DELETE  | `/api/rentals/{id}`               | Reservierung stornieren (nur Status `reserviert`) |

Fehler kommen als `{"error": "..."}` mit Status 400 (Validierung), 404 (nicht gefunden) oder 409 (Konflikt).

## Projektstruktur

```
main.go                  Server, Routing, Seed-Daten
internal/domain/         Ski, SkiType, Tariff, Person, IDDocument, Rental
internal/store/          MemoryStore (thread-safe)
internal/handler/        JSON-API
web/                     index.html, style.css, app.js
```

## Nicht im Umfang
Login/Benutzerrollen, persistente Datenbank, Zahlungsabwicklung, Mehrsprachigkeit.
