// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

// ctog "github.com/invopop/gobl.cii/ctog"
// gtoc "github.com/invopop/gobl.cii/gtoc"

import (
	"github.com/invopop/gobl"
	ctog "github.com/invopop/gobl.cii/ctog"
	gtoc "github.com/invopop/gobl.cii/gtoc"
)

// Conversor is a struct that encapsulates both CtoG and GtoC conversors
type Conversor struct {
	CtoG *ctog.Converter
	GtoC *gtoc.Converter
}

// NewConversor creates a new Conversor instance
func NewConversor() *Conversor {
	c := new(Conversor)
	c.CtoG = ctog.NewConverter()
	c.GtoC = gtoc.NewConverter()
	return c
}

// ConvertToGOBL converts a CII document to a GOBL envelope
func (c *Conversor) ConvertToGOBL(ciiDoc []byte) (*gobl.Envelope, error) {
	return c.CtoG.ConvertToGOBL(ciiDoc)
}

// ConvertToCII converts a GOBL envelope to a CII document
func (c *Conversor) ConvertToCII(env *gobl.Envelope) (*gtoc.Document, error) {
	ciiDoc, err := c.GtoC.ConvertToCII(env)
	if err != nil {
		return nil, err
	}
	return ciiDoc, nil
}
