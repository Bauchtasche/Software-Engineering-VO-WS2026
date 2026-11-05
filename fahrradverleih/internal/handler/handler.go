package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fahrradverleih/internal/domain"
	"fahrradverleih/internal/store"
)

// API bündelt alle HTTP-Handler und hält den Store.
type API struct {
	store *store.MemoryStore
}

func New(s *store.MemoryStore) *API {
	return &API{store: s}
}

// Register hängt alle Routen an den Mux (Go 1.22 Pattern mit Methode).
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard", a.dashboard)
	mux.HandleFunc("GET /api/tariffs", a.tariffs)

	mux.HandleFunc("GET /api/bikes", a.listBikes)
	mux.HandleFunc("GET /api/bikes/available", a.availableBikes)
	mux.HandleFunc("POST /api/bikes", a.createBike)
	mux.HandleFunc("DELETE /api/bikes/{id}", a.deleteBike)

	mux.HandleFunc("GET /api/persons", a.listPersons)
	mux.HandleFunc("POST /api/persons", a.createPerson)
	mux.HandleFunc("DELETE /api/persons/{id}", a.deletePerson)
	mux.HandleFunc("POST /api/persons/{id}/document", a.uploadDocument)
	mux.HandleFunc("GET /api/persons/{id}/document", a.downloadDocument)

	mux.HandleFunc("GET /api/rentals", a.listRentals)
	mux.HandleFunc("GET /api/quote", a.quote)
	mux.HandleFunc("POST /api/rentals", a.createRental)
	mux.HandleFunc("POST /api/rentals/{id}/return", a.returnRental)
	mux.HandleFunc("DELETE /api/rentals/{id}", a.cancelRental)
}

// ---------- Views (JSON-Antworten) ----------

type bikeView struct {
	domain.Bike
	Tariff domain.Tariff `json:"tariff"`
	Status string        `json:"status"`
}

type rentalView struct {
	domain.Rental
	Status     domain.RentalStatus `json:"status"`
	BikeLabel  string              `json:"bikeLabel"`
	PersonName string              `json:"personName"`
	Document   string              `json:"document"`
}

func (a *API) rentalViews() []rentalView {
	now := a.store.Now()
	rentals := a.store.Rentals()
	out := make([]rentalView, 0, len(rentals))
	for _, r := range rentals {
		v := rentalView{Rental: r, Status: r.Status(now), BikeLabel: "(gelöscht)", PersonName: "(gelöscht)"}
		if bk, err := a.store.Bike(r.BikeID); err == nil {
			v.BikeLabel = bk.Label()
		}
		if p, err := a.store.Person(r.PersonID); err == nil {
			v.PersonName = p.FullName()
			v.Document = fmt.Sprintf("%s %s", p.Document.Type, p.Document.Number)
		}
		out = append(out, v)
	}
	return out
}

// bikeStatus: "verliehen"/"überfällig", wenn gerade eine Ausleihe läuft, sonst "verfügbar".
func bikeStatus(bikeID int, rentals []rentalView) string {
	for _, r := range rentals {
		if r.BikeID != bikeID {
			continue
		}
		switch r.Status {
		case domain.StatusActive:
			return "verliehen"
		case domain.StatusOverdue:
			return "überfällig"
		}
	}
	return "verfügbar"
}

// ---------- Dashboard ----------

func (a *API) dashboard(w http.ResponseWriter, _ *http.Request) {
	now := a.store.Now()
	rentals := a.rentalViews()
	bikes := a.store.Bikes()

	var rented, overdue, reserved, revenue int
	dueToday := []rentalView{}
	overdueList := []rentalView{}
	upcoming := []rentalView{}
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	for _, r := range rentals {
		switch r.Status {
		case domain.StatusActive:
			rented++
			if r.End.Before(endOfDay) {
				dueToday = append(dueToday, r)
			}
		case domain.StatusOverdue:
			rented++
			overdue++
			overdueList = append(overdueList, r)
		case domain.StatusReserved:
			reserved++
			if len(upcoming) < 5 {
				upcoming = append(upcoming, r)
			}
		case domain.StatusReturned:
			if r.FinalCost != nil {
				revenue += *r.FinalCost
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"bikesTotal":     len(bikes),
		"bikesRented":    rented,
		"bikesAvailable": len(bikes) - rented,
		"persons":        len(a.store.Persons()),
		"overdue":        overdue,
		"reserved":       reserved,
		"revenueCents":   revenue,
		"dueToday":       dueToday,
		"overdueList":    overdueList,
		"upcoming":       upcoming,
	})
}

func (a *API) tariffs(w http.ResponseWriter, _ *http.Request) {
	type row struct {
		Type   domain.BikeType `json:"type"`
		Tariff domain.Tariff   `json:"tariff"`
	}
	out := []row{}
	for _, t := range domain.AllBikeTypes() {
		out = append(out, row{t, t.Tariff()})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"bikeTypes":     out,
		"documentTypes": domain.AllDocumentTypes(),
		"frameSizes":    domain.AllFrameSizes(),
	})
}

// ---------- Fahrräder ----------

func (a *API) listBikes(w http.ResponseWriter, _ *http.Request) {
	rentals := a.rentalViews()
	out := []bikeView{}
	for _, bk := range a.store.Bikes() {
		out = append(out, bikeView{Bike: bk, Tariff: bk.Type.Tariff(), Status: bikeStatus(bk.ID, rentals)})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) availableBikes(w http.ResponseWriter, r *http.Request) {
	start, end, err := parsePeriod(r)
	if err != nil {
		writeError(w, err)
		return
	}
	out := []bikeView{}
	for _, bk := range a.store.AvailableBikes(start, end) {
		out = append(out, bikeView{Bike: bk, Tariff: bk.Type.Tariff(), Status: "verfügbar"})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createBike(w http.ResponseWriter, r *http.Request) {
	var in domain.Bike
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	bk, err := a.store.AddBike(in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bk)
}

func (a *API) deleteBike(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		err = a.store.DeleteBike(id)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Personen ----------

func (a *API) listPersons(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.store.Persons())
}

func (a *API) createPerson(w http.ResponseWriter, r *http.Request) {
	var in domain.Person
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	p, err := a.store.AddPerson(in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *API) deletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		err = a.store.DeletePerson(id)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) uploadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxDocumentFileSize+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, fmt.Errorf("%w: keine Datei im Feld 'file' (max. 5 MB)", domain.ErrValidation))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, fmt.Errorf("%w: Datei konnte nicht gelesen werden", domain.ErrValidation))
		return
	}
	// Inhalt prüfen statt dem Browser-Header zu vertrauen.
	contentType := http.DetectContentType(data)
	if err := a.store.AttachDocument(id, header.Filename, contentType, data); err != nil {
		writeError(w, err)
		return
	}
	p, _ := a.store.Person(id)
	writeJSON(w, http.StatusOK, p)
}

func (a *API) downloadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := a.store.Person(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if !p.Document.HasFile {
		writeError(w, fmt.Errorf("Ausweis-Datei: %w", store.ErrNotFound))
		return
	}
	w.Header().Set("Content-Type", p.Document.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", p.Document.FileName))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(p.Document.File)
}

// ---------- Ausleihen ----------

func (a *API) listRentals(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.rentalViews())
}

func (a *API) quote(w http.ResponseWriter, r *http.Request) {
	start, end, err := parsePeriod(r)
	if err != nil {
		writeError(w, err)
		return
	}
	bikeID, err := strconv.Atoi(r.URL.Query().Get("bikeId"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: bikeId fehlt", domain.ErrValidation))
		return
	}
	bk, err := a.store.Bike(bikeID)
	if err != nil {
		writeError(w, err)
		return
	}
	d := end.Sub(start)
	writeJSON(w, http.StatusOK, map[string]any{
		"hours":      domain.BillableHours(d),
		"priceCents": bk.Type.Tariff().Price(d),
		"tariff":     bk.Type.Tariff(),
	})
}

func (a *API) createRental(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BikeID   int       `json:"bikeId"`
		PersonID int       `json:"personId"`
		Start    time.Time `json:"start"`
		End      time.Time `json:"end"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	rental, err := a.store.CreateRental(in.BikeID, in.PersonID, in.Start, in.End)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rental)
}

func (a *API) returnRental(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rental, err := a.store.ReturnRental(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rental)
}

func (a *API) cancelRental(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		err = a.store.CancelRental(id)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Hilfsfunktionen ----------

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: JSON ungültig (%v)", domain.ErrValidation, err)
	}
	return nil
}

func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return 0, fmt.Errorf("%w: ungültige ID", domain.ErrValidation)
	}
	return id, nil
}

func parsePeriod(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()
	start, err1 := time.Parse(time.RFC3339, q.Get("start"))
	end, err2 := time.Parse(time.RFC3339, q.Get("end"))
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: start und end im Format RFC 3339 angeben", domain.ErrValidation)
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: Ende muss nach dem Beginn liegen", domain.ErrValidation)
	}
	return start, end, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	}
	msg := err.Error()
	// Präfixe der Sentinel-Fehler für die Anzeige entfernen.
	for _, p := range []string{domain.ErrValidation.Error() + ": ", store.ErrConflict.Error() + ": "} {
		msg = strings.TrimPrefix(msg, p)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
