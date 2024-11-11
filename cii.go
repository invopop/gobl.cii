// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"github.com/invopop/gobl"
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl.cii/internal/ctog"
	"github.com/invopop/gobl.cii/internal/gtoc"
)

// ToGOBL converts a CII document to a GOBL envelope
func ToGOBL(ciiDoc []byte) (*gobl.Envelope, error) {
	return ctog.Convert(ciiDoc)
}

// ToCII converts a GOBL envelope to a CII document
func ToCII(env *gobl.Envelope) (*document.Document, error) {
	ciiDoc, err := gtoc.Convert(env)
	if err != nil {
		return nil, err
	}
	return ciiDoc, nil
}
