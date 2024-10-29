// Package gtoc contains the logic to convert a GOBL envelope into a CII document
package gtoc

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

// Converter is the struct that contains the logic to convert a GOBL envelope into a CII document
type Converter struct {
	doc *Document
}

// NewConversor Builder function
func NewConversor() *Converter {
	c := new(Converter)
	c.doc = new(Document)
	return c
}

// GetDocument returns the CII document
func (c *Converter) GetDocument() *Document {
	return c.doc
}

// ConvertToCII converts a GOBL envelope into a CIIdocument
func (c *Converter) ConvertToCII(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	transaction, err := NewTransaction(inv)
	if err != nil {
		return nil, err
	}

	ciiDoc := Document{
		RSMNamespace:           RSM,
		RAMNamespace:           RAM,
		QDTNamespace:           QDT,
		UDTNamespace:           UDT,
		BusinessProcessContext: BusinessProcess,
		GuidelineContext:       GuidelineContext,
		ExchangedDocument:      NewHeader(inv),
		Transaction:            transaction,
	}

	c.doc = &ciiDoc
	return c.doc, nil
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}
