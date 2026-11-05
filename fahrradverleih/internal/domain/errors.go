package domain

import "errors"

// ErrValidation markiert fachlich ungültige Eingaben (→ HTTP 400).
var ErrValidation = errors.New("ungültige Eingabe")
