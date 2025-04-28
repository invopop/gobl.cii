// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/de/zugferd"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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
	ProfileIDPeppolBilling = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
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

// ContextZUGFeRDV2 is the context used for ZUGFeRD documents which is identical to
// FacturX
var ContextZUGFeRDV2 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended",
	Addons:      []cbc.Key{zugferd.V2},
}

// ContextXRechnungV3 is used for XRechnung documents
var ContextXRechnungV3 = Context{
	GuidelineID: "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0",
	BusinessID:  ProfileIDPeppolBilling,
	Addons:      []cbc.Key{xrechnung.V3},
}

// ParseInvoice parses a raw XML CII Invoice and converts it into
// a GOBL envelope.
func ParseInvoice(data []byte) (*gobl.Envelope, error) {
	env := gobl.NewEnvelope()
	inv, err := parseInvoice(data)
	if err != nil {
		return nil, err
	}
	if err := env.Insert(inv); err != nil {
		return nil, err
	}
	return env, nil
}

// ConvertInvoice takes a gobl Invoice and converts it into a CII Invoice
// ready to be serialized into an XML data object.
func ConvertInvoice(env *gobl.Envelope, opts ...Option) (*Invoice, error) {
	o := &options{
		context: ContextEN16931V2017,
	}
	for _, opt := range opts {
		opt(o)
	}

	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("expected bill/invoice, got %T", env.Extract())
	}

	// Check addons
	for _, ao := range o.context.Addons {
		if !ao.In(inv.GetAddons()...) {
			return nil, fmt.Errorf("gobl invoice missing addon %s", ao)
		}
	}

	return newInvoice(inv, o.context)
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
