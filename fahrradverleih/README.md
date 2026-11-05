# Fahrradverleih

Prototyp eines Fahrradverleihs (Aufgabe 1c): Go-Backend mit In-Memory-Speicher und HTML/JS-Frontend.
Eigenständiges Projekt, aufgebaut nach denselben Regeln wie der Ski-Verleih (a+b).
Entwickelt mit Spec-Driven Development: diese Datei ist die Spezifikation, `CLAUDE.md` enthält die technischen Regeln.

## Starten

```bash
go run .
# Browser: http://localhost:8080
go test ./...
```

Beim Start werden Beispieldaten (Fahrräder und Personen) angelegt. Alle Daten gehen beim Beenden verloren (In-Memory).

## Anforderungen

### Fahrräder
- Anlegen mit Marke, Modell, Typ, Rahmengröße und Rahmennummer; auflisten (mit Suche); löschen.
- **Typen:** City, Trekking, Mountainbike, E-Bike, Kinderrad
- **Rahmengröße:** XS, S, M, L, XL
- **Rahmennummer:** Pflicht, mind. 4 Zeichen, wird in Großbuchstaben gespeichert und muss eindeutig sein
  (identifiziert das Rad physisch, z.B. bei Diebstahl).
- Ein Fahrrad mit laufender oder geplanter Ausleihe kann nicht gelöscht werden.

### Personen und Ausweis
- Anlegen mit Vorname, Nachname, Telefon, E-Mail; auflisten (mit Suche); löschen.
- Ausweis: Typ `Personalausweis`, `Reisepass` oder `Führerschein` + Nummer (5–20 Buchstaben/Ziffern) – Pflicht.
- Derselbe Ausweis (Typ + Nummer) darf nur einmal registriert sein.
- Optionaler Upload als JPG, PNG oder PDF (max. 5 MB), auch nachträglich; Upload kann angesehen werden.
  Das Format wird am Dateiinhalt geprüft, nicht an der Dateiendung.
- Eine Person mit laufender oder geplanter Ausleihe kann nicht gelöscht werden.

### Ausleihe mit flexiblen Zeiten
- **Beginn frei wählbar:** sofort oder in der Zukunft (= Reservierung). Höchstens 15 Minuten in der Vergangenheit.
- **Dauer frei wählbar:** mindestens 1 Stunde, höchstens 30 Tage. Schnellauswahl: 2 h, 4 h, 1 Tag, 3 Tage, 1 Woche.
- **Keine Doppelbuchung:** Überschneidende Zeiträume für dasselbe Rad werden abgelehnt.
  Im Formular werden nur Räder angeboten, die im gewählten Zeitraum frei sind.
  Ein überfälliges Rad gilt bis zur Rückgabe als belegt (mindestens bis jetzt + 1 Stunde).
- **Rückgabe:** Endpreis wird mit der tatsächlichen Dauer berechnet.
- **Stornieren:** nur Reservierungen, die noch nicht begonnen haben.

### Preise: Stundenpreis mit Tagesdeckel
- Angefangene Stunden zählen voll, mindestens 1 Stunde.
- `preis = tage × tagespreis + min(reststunden × stundenpreis, tagespreis)`

| Typ          | pro Stunde | pro Tag (Deckel) |
|--------------|-----------:|-----------------:|
| City         |     3,00 € |          15,00 € |
| Trekking     |     4,00 € |          20,00 € |
| Mountainbike |     6,00 € |          32,00 € |
| E-Bike       |     9,00 € |          45,00 € |
| Kinderrad    |     2,00 € |          10,00 € |

Beispiel Mountainbike: 3 h → 18 €; 6 h → 32 € (Deckel statt 36 €); 26 h → 32 € + 12 € = 44 €.
Vor dem Buchen wird der Preis live angezeigt.

### Dashboard
- Räder gesamt / jetzt verfügbar / jetzt verliehen, Personen, Reservierungen, Umsatz (abgeschlossen)
- Überfällige Ausleihen rot hervorgehoben
- Listen „Überfällig“, „Heute fällig“ (mit Button „Rückgabe“) und „Nächste Reservierungen“
- Schnellzugriff: neue Ausleihe, Person anlegen, Fahrrad anlegen

### Status einer Ausleihe
| Status          | Bedingung                                  |
|-----------------|--------------------------------------------|
| `reserviert`    | jetzt < Beginn                             |
| `aktiv`         | Beginn ≤ jetzt ≤ Ende, nicht zurückgegeben |
| `überfällig`    | jetzt > Ende, nicht zurückgegeben          |
| `zurückgegeben` | Rückgabe gebucht                           |

## REST-API

| Methode | Pfad                               | Beschreibung |
|---------|------------------------------------|--------------|
| GET     | `/api/dashboard`                   | Kennzahlen + fällige/überfällige Ausleihen |
| GET     | `/api/tariffs`                     | Typen mit Tarif, Ausweistypen, Rahmengrößen |
| GET     | `/api/bikes`                       | alle Räder inkl. aktuellem Status |
| GET     | `/api/bikes/available?start=&end=` | im Zeitraum freie Räder |
| POST    | `/api/bikes`                       | Fahrrad anlegen |
| DELETE  | `/api/bikes/{id}`                  | Fahrrad löschen |
| GET     | `/api/persons`                     | alle Personen |
| POST    | `/api/persons`                     | Person inkl. Ausweis anlegen |
| DELETE  | `/api/persons/{id}`                | Person löschen |
| POST    | `/api/persons/{id}/document`       | Ausweis-Datei hochladen (multipart, Feld `file`) |
| GET     | `/api/persons/{id}/document`       | Ausweis-Datei ansehen |
| GET     | `/api/rentals`                     | alle Ausleihen inkl. Status |
| GET     | `/api/quote?bikeId=&start=&end=`   | Preisvorschau |
| POST    | `/api/rentals`                     | Ausleihe/Reservierung anlegen |
| POST    | `/api/rentals/{id}/return`         | Rückgabe buchen (Endpreis) |
| DELETE  | `/api/rentals/{id}`                | Reservierung stornieren |

Fehler kommen als `{"error": "..."}` mit Status 400 (Validierung), 404 (nicht gefunden) oder 409 (Konflikt).

## Projektstruktur

```
main.go                  Server, Routing, Seed-Daten
internal/domain/         Bike, BikeType, FrameSize, Tariff, Person, IDDocument, Rental
internal/store/          MemoryStore (thread-safe)
internal/handler/        JSON-API
web/                     index.html, style.css, app.js
```

## Nicht im Umfang
Login/Benutzerrollen, persistente Datenbank, Zahlungsabwicklung, Zubehör (Helm, Schloss),
gemeinsames Verleih-Framework für Ski und Fahrrad (wäre Aufgabe d).
