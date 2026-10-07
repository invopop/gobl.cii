// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/schema"
)

var (
	// ErrUnknownDocumentType is returned when the document type
	// is not recognized during parsing.
	ErrUnknownDocumentType = fmt.Errorf("unknown document type")

	// ErrUnsupportedDocumentType is returned when the document type
	// is not supported for conversion.
	ErrUnsupportedDocumentType = fmt.Errorf("unsupported document type")
)

// CII namespaces
const (
	NamespaceRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	NamespaceRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	NamespaceQDT = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	NamespaceUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
)

// Namespace prefixes used when unmarshalling CII XML documents.
const (
	nsPrefixRSM = "rsm"
	nsPrefixRAM = "ram"
	nsPrefixQDT = "qdt"
	nsPrefixUDT = "udt"
)

// Document is a UN/CEFACT document: an *Invoice (Cross Industry Invoice) or a
// *CDAR (Cross Domain Acknowledgement and Response).
type Document interface {
	ciiDocument()
}

func (*Invoice) ciiDocument() {}
func (*CDAR) ciiDocument()    {}

// Decode reads raw XML data into the Document it contains, chosen by the
// root element's namespace.
func Decode(data []byte) (Document, error) {
	ns, err := extractRootNamespace(data)
	if err != nil {
		return nil, err
	}

	switch ns {
	case NamespaceRSM:
		return decodeInvoice(data)
	case NamespaceCDARRSM:
		return decodeCDAR(data)
	default:
		return nil, ErrUnknownDocumentType
	}
}

// Encode returns the XML of the document, including the XML header.
func Encode(doc Document) ([]byte, error) {
	switch d := doc.(type) {
	case *Invoice:
		return d.encode()
	case *CDAR:
		return d.encode()
	}
	return nil, ErrUnsupportedDocumentType
}

// Import converts the CII invoice into a GOBL envelope. The format is
// determined from the document's guideline and business process IDs, unless
// a WithFormat option is provided. Other documents, such as a CDAR, are
// imported by the packages that implement their formats.
func Import(doc Document, opts ...Option) (*gobl.Envelope, error) {
	in, ok := doc.(*Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}

	o := new(options)
	guidelineID, businessID := in.contextIDs()
	if f := FindFormat(guidelineID, businessID); f != nil {
		o.format = *f
	}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	inv, err := goblInvoice(in, &o.format)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	// A parsed document is one we received: its transport addresses are the
	// ones the Peppol layer routed it with (who sent it → who received it),
	// supplied via WithRouting as fully-qualified participant URIs. Set
	// Head.From / Head.To from those args verbatim, BEFORE calculation, so GOBL
	// respects them — normalizeRouting only fills empty routing fields, so it
	// won't overwrite them with the document-derived, OUTGOING-direction guess
	// (supplier → customer) that is wrong for a received document.
	env.Head.From = o.from
	env.Head.To = o.to
	if env.Document, err = schema.NewObject(inv); err != nil {
		return nil, err
	}
	if err := o.format.runImportFuncs(in, env); err != nil {
		return nil, err
	}
	if err := env.Calculate(); err != nil {
		return nil, err
	}
	return env, nil
}

// Export converts the GOBL envelope containing an invoice into a CII
// document.
//
// Add a WithFormat option to specify the desired CII format. If none is
// provided, EN 16931 will be used by default.
func Export(env *gobl.Envelope, opts ...Option) (Document, error) {
	o := &options{
		format: FormatEN16931,
	}
	for _, opt := range opts {
		opt(o)
	}

	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}

	// Check addons
	for _, ao := range o.format.Addons {
		if !ao.In(inv.GetAddons()...) {
			return nil, fmt.Errorf("gobl invoice missing addon %s", ao)
		}
	}

	// Removes included taxes as they are not supported in CII
	if err := inv.RemoveIncludedTaxes(); err != nil {
		return nil, fmt.Errorf("cannot convert invoice with included taxes: %w", err)
	}
	if err := inv.RoundToCurrency(); err != nil {
		return nil, fmt.Errorf("cannot round invoice to currency precision: %w", err)
	}

	out, err := newInvoice(inv, o.format)
	if err != nil {
		return nil, err
	}
	if err := o.format.runExportFuncs(env, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExportInvoice is a convenience function that exports a GOBL envelope
// containing an invoice into a CII Invoice.
func ExportInvoice(env *gobl.Envelope, opts ...Option) (*Invoice, error) {
	doc, err := Export(env, opts...)
	if err != nil {
		return nil, err
	}
	return doc.(*Invoice), nil
}

type options struct {
	format   Format
	from, to cbc.URI
}

// Option is used to define configuration options to use during the
// import and export processes.
type Option func(*options)

// WithFormat sets the format to use instead of the default on export, or the
// one detected on import.
func WithFormat(f Format) Option {
	return func(o *options) {
		o.format = f
	}
}

// WithRouting supplies the transport addresses a received document was routed
// with — the Peppol SBD From / To — as fully-qualified participant URIs (e.g.
// "iso6523-actorid-upis::0225:code"). They are recorded verbatim on the
// envelope's Head.From / Head.To to mark who sent and who received the document.
func WithRouting(from, to cbc.URI) Option {
	return func(o *options) {
		o.from = from
		o.to = to
	}
}

func extractRootNamespace(data []byte) (string, error) {
	dc := xml.NewDecoder(bytes.NewReader(data))
	for {
		tk, err := dc.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error parsing XML: %w", err)
		}
		switch t := tk.(type) {
		case xml.StartElement:
			return t.Name.Space, nil // Extract and return the namespace
		}
	}
	return "", ErrUnknownDocumentType
}
