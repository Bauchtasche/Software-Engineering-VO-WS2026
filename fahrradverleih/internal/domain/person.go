package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// DocumentType ist die Art des Ausweises.
type DocumentType string

const (
	Personalausweis DocumentType = "Personalausweis"
	Reisepass       DocumentType = "Reisepass"
	Fuehrerschein   DocumentType = "Führerschein"
)

// AllDocumentTypes liefert alle erlaubten Ausweistypen.
func AllDocumentTypes() []DocumentType {
	return []DocumentType{Personalausweis, Reisepass, Fuehrerschein}
}

// Valid prüft, ob der Ausweistyp erlaubt ist.
func (t DocumentType) Valid() bool {
	for _, d := range AllDocumentTypes() {
		if d == t {
			return true
		}
	}
	return false
}

var docNumberPattern = regexp.MustCompile(`^[A-Z0-9]{5,20}$`)

// MaxDocumentFileSize: maximal 5 MB pro Upload.
const MaxDocumentFileSize = 5 << 20

// AllowedDocumentContentTypes: erlaubte Dateiformate für den Ausweis-Upload.
func AllowedDocumentContentTypes() []string {
	return []string{"image/jpeg", "image/png", "application/pdf"}
}

// IDDocument ist der Ausweis einer Person (Typ + Nummer, optional Datei).
type IDDocument struct {
	Type        DocumentType `json:"type"`
	Number      string       `json:"number"`
	FileName    string       `json:"fileName,omitempty"`
	ContentType string       `json:"contentType,omitempty"`
	HasFile     bool         `json:"hasFile"`
	File        []byte       `json:"-"`
}

// Validate prüft Typ und Nummer.
func (d IDDocument) Validate() error {
	if !d.Type.Valid() {
		return fmt.Errorf("%w: Ausweistyp muss Personalausweis, Reisepass oder Führerschein sein", ErrValidation)
	}
	if !docNumberPattern.MatchString(d.Number) {
		return fmt.Errorf("%w: Ausweisnummer muss 5–20 Buchstaben/Ziffern haben", ErrValidation)
	}
	return nil
}

// AttachFile hängt einen Upload an, nachdem Format und Größe geprüft wurden.
func (d *IDDocument) AttachFile(name, contentType string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: Datei ist leer", ErrValidation)
	}
	if len(data) > MaxDocumentFileSize {
		return fmt.Errorf("%w: Datei ist größer als 5 MB", ErrValidation)
	}
	allowed := false
	for _, ct := range AllowedDocumentContentTypes() {
		if ct == contentType {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("%w: nur JPG, PNG oder PDF erlaubt", ErrValidation)
	}
	d.FileName = name
	d.ContentType = contentType
	d.File = data
	d.HasFile = true
	return nil
}

// Person ist eine Kundin bzw. ein Kunde.
type Person struct {
	ID        int        `json:"id"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Phone     string     `json:"phone"`
	Email     string     `json:"email"`
	Document  IDDocument `json:"document"`
}

// Normalize vereinheitlicht die Eingaben (Ausweisnummer in Großbuchstaben, ohne Leerzeichen).
func (p *Person) Normalize() {
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	p.Phone = strings.TrimSpace(p.Phone)
	p.Email = strings.TrimSpace(p.Email)
	p.Document.Number = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(p.Document.Number), " ", ""))
}

// Validate prüft Pflichtfelder und Ausweis.
func (p Person) Validate() error {
	if p.FirstName == "" || p.LastName == "" {
		return fmt.Errorf("%w: Vor- und Nachname sind Pflicht", ErrValidation)
	}
	if p.Email != "" && !strings.Contains(p.Email, "@") {
		return fmt.Errorf("%w: E-Mail-Adresse ist ungültig", ErrValidation)
	}
	return p.Document.Validate()
}

// FullName liefert "Vorname Nachname".
func (p Person) FullName() string {
	return p.FirstName + " " + p.LastName
}
