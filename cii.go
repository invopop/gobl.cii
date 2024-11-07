// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"github.com/invopop/gobl"
	"github.com/invopop/gobl.cii/ctog"
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl.cii/gtoc"
)

// Converter is a struct that encapsulates both CtoG and GtoC converters
type Converter struct {
	ctog *ctog.Converter
	gtoc *gtoc.Converter
}

// NewConverter creates a new Converter instance
func NewConverter() *Converter {
	c := new(Converter)
	c.ctog = ctog.NewConverter()
	c.gtoc = gtoc.NewConverter()
	return c
}

// ToGOBL converts a CII document to a GOBL envelope
func (c *Converter) ToGOBL(ciiDoc []byte) (*gobl.Envelope, error) {
	return c.ctog.ConvertToGOBL(ciiDoc)
}

// ToCII converts a GOBL envelope to a CII document
func (c *Converter) ToCII(env *gobl.Envelope) (*document.Document, error) {
	ciiDoc, err := c.gtoc.ConvertToCII(env)
	if err != nil {
		return nil, err
	}
	return ciiDoc, nil
}
