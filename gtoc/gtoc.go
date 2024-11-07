// Package gtoc contains the logic to convert a GOBL envelope into a CII document
package gtoc

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
)

// Converter is the struct that contains the logic to convert a GOBL envelope into a CII document
type Converter struct {
	doc *document.Document
}

// NewConverter Builder function
func NewConverter() *Converter {
	c := new(Converter)
	c.doc = new(document.Document)
	return c
}

// GetDocument returns the CII document
func (c *Converter) GetDocument() *document.Document {
	return c.doc
}

// ConvertToCII converts a GOBL envelope into a CIIdocument
func (c *Converter) ConvertToCII(env *gobl.Envelope) (*document.Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}
	err := c.newDocument(inv)
	if err != nil {
		return nil, err
	}
	return c.doc, nil
}

func (c *Converter) newDocument(inv *bill.Invoice) error {

	c.doc = &document.Document{
		RSMNamespace: document.RSM,
		RAMNamespace: document.RAM,
		QDTNamespace: document.QDT,
		UDTNamespace: document.UDT,
		ExchangedContext: &document.ExchangedContext{
			BusinessContext:  &document.ExchangedContextParameter{ID: document.BusinessProcess},
			GuidelineContext: &document.ExchangedContextParameter{ID: document.GuidelineContext},
		},
	}

	err := c.NewHeader(inv)
	if err != nil {
		return err
	}

	err = c.NewTransaction(inv)
	if err != nil {
		return err
	}

	return nil
}
