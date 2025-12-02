// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/de/zugferd"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/choruspro"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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

// Profile ID codes
const (
	ProfileIDPeppolBilling       = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
	ProfileIDPeppolFranceBilling = "urn:peppol:france:billing:regulated"
)

// Context is used to ensure that the generated CII document
// uses a specific set of Guidline and Business rules when generating
// the output.
type Context struct {
	GuidelineID string
	BusinessID  string
	Addons      []cbc.Key
}

// ContextEN16931V2017 is used for EN 16931 documents, and is the default.
var ContextEN16931V2017 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017",
	Addons:      []cbc.Key{en16931.V2017},
}

// ContextPeppolV3 for Peppol Billing V3.0 context.
var ContextPeppolV3 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	BusinessID:  ProfileIDPeppolBilling,
	Addons:      []cbc.Key{en16931.V2017},
}

// ContextFacturXV1 is used for Factur-X V1 documents.
var ContextFacturXV1 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended",
	Addons:      []cbc.Key{facturx.V1},
}

// ContextPeppolFranceFacturXV1 is used for Peppol France Factur-X documents.
var ContextPeppolFranceFacturX = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:Factur-X:1.0",
	BusinessID:  ProfileIDPeppolFranceBilling,
	Addons:      []cbc.Key{facturx.V1},
}

// ContextPeppolFranceCIUS is used for Peppol France CIUS documents.
var ContextPeppolFranceCIUS = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0",
	BusinessID:  ProfileIDPeppolFranceBilling,
	Addons:      []cbc.Key{facturx.V1},
}

// ContextPeppolFranceExtended is used for Peppol France CIUS documents.
var ContextPeppolFranceExtended = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0",
	BusinessID:  ProfileIDPeppolFranceBilling,
	Addons:      []cbc.Key{facturx.V1},
}

// ContextZUGFeRDV2 is the context used for ZUGFeRD documents.
var ContextZUGFeRDV2 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#conformant#urn:zugferd.de:2p0:extended",
	Addons:      []cbc.Key{zugferd.V2},
}

// ContextXRechnungV3 is used for XRechnung documents
var ContextXRechnungV3 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0",
	BusinessID:  ProfileIDPeppolBilling,
	Addons:      []cbc.Key{xrechnung.V3},
}

// ContextChorusProV1 is used for Chorus Pro V1 documents.
var ContextChorusProV1 = Context{
	GuidelineID: "A1", // Default framework type
	Addons:      []cbc.Key{choruspro.V1},
}

// Parse parses a raw XML CII document and converts it into
// a GOBL envelope. If the type is unsupported, an
// ErrUnknownDocumentType is provided.
func Parse(data []byte) (*gobl.Envelope, error) {
	ns, err := extractRootNamespace(data)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	var res any
	switch ns {
	case NamespaceRSM:
		res, err = parseInvoice(data)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnknownDocumentType
	}

	if err := env.Insert(res); err != nil {
		return nil, err
	}
	return env, nil
}

// Convert takes a gobl envelope and converts it into a CII document
// ready to be serialized into an XML data object.
func Convert(env *gobl.Envelope, opts ...Option) (any, error) {
	o := &options{
		context: ContextEN16931V2017,
	}
	for _, opt := range opts {
		opt(o)
	}

	switch doc := env.Extract().(type) {
	case *bill.Invoice:
		// Check addons
		for _, ao := range o.context.Addons {
			if !ao.In(doc.GetAddons()...) {
				return nil, fmt.Errorf("gobl invoice missing addon %s", ao)
			}
		}

		// Removes included taxes as they are not supported in CII
		if err := doc.RemoveIncludedTaxes(); err != nil {
			return nil, fmt.Errorf("cannot convert invoice with included taxes: %w", err)
		}

		return newInvoice(doc, o.context)
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

// ConvertInvoice is a convenience function that converts a GOBL envelope
// containing an invoice into a CII Invoice.
func ConvertInvoice(env *gobl.Envelope, opts ...Option) (*Invoice, error) {
	doc, err := Convert(env, opts...)
	if err != nil {
		return nil, err
	}
	inv, ok := doc.(*Invoice)
	if !ok {
		return nil, fmt.Errorf("expected invoice, got %T", doc)
	}
	return inv, nil
}

type options struct {
	context Context
}

// Option is used to define configuration options to use during the
// conversion processes.
type Option func(*options)

// WithContext sets the context for the output CII document if not using the default.
func WithContext(context Context) Option {
	return func(o *options) {
		o.context = context
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
