package cii

import (
	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/schema"
)

// ExportFunc adjusts the CII document exported from the GOBL envelope for a
// format.
type ExportFunc func(f *Format, env *gobl.Envelope, doc Document) error

// ImportFunc adjusts the GOBL envelope imported from the CII document for a
// format, before the envelope is calculated.
type ImportFunc func(f *Format, doc Document, env *gobl.Envelope) error

var invoiceSchema = schema.Lookup(bill.Invoice{})

func (f *Format) runExportFuncs(env *gobl.Envelope, doc Document) error {
	for _, fn := range f.ExportFuncs {
		if err := fn(f, env, doc); err != nil {
			return err
		}
	}
	return nil
}

func (f *Format) runImportFuncs(doc Document, env *gobl.Envelope) error {
	for _, fn := range f.ImportFuncs {
		if err := fn(f, doc, env); err != nil {
			return err
		}
	}
	return nil
}

// CleanString removes the replacement characters left by broken text
// encodings from incoming text.
func CleanString(s string) string {
	return cleanString(s)
}
