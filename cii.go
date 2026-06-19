// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.fr.ctc/addon/flow2"
	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/de/zugferd"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/choruspro"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
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

// Common Guideline and VESID values reused across multiple contexts.
const (
	guidelineIDEN16931V2017 = "urn:cen.eu:en16931:2017"
	vesIDEN16931CII         = "eu.cen.en16931:cii:1.3.16"
)

// Profile ID codes
const (
	ProfileIDPeppolBilling       = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
	ProfileIDPeppolFranceBilling = "urn:peppol:france:billing:regulated"
)

// CII Versions
const (
	VersionD16B string = "D16B"
	VersionD22B string = "D22B"
)

// Context is used to ensure that the generated CII document
// uses a specific set of Guidline and Business rules when generating
// the output.
type Context struct {
	GuidelineID string
	BusinessID  string
	// OutputGuidelineID optionally specifies a different GuidelineID
	// to use in the actual generated CII XML document. If empty, GuidelineID
	// is used. This allows the context to be identified by one ID externally while
	// generating different values in the XML output.
	OutputGuidelineID string
	// OutputBusinessID optionally specifies a different BusinessID to write
	// into the generated CII/CDAR XML's BusinessProcessParameter. If empty,
	// BusinessID is used. This lets BusinessID carry the external busdox
	// process id (SMP/SBD routing) while the XML keeps a distinct value
	// (e.g. CDAR MDT-2 "REGULATED" vs busdox process
	// urn:peppol:france:billing:regulated).
	OutputBusinessID  string
	Version           string
	Addons            []cbc.Key
	// VESID is the Validation Exchange Specification ID used for validation
	VESID string
}

// Is checks if two contexts are the same.
func (c *Context) Is(c2 Context) bool {
	return c.GuidelineID == c2.GuidelineID && c.BusinessID == c2.BusinessID
}

// ContextEN16931V2017 is used for EN 16931 documents, and is the default.
var ContextEN16931V2017 = Context{
	GuidelineID: guidelineIDEN16931V2017,
	Version:     VersionD16B,
	Addons:      []cbc.Key{en16931.V2017},
	VESID:       vesIDEN16931CII,
}

// ContextPeppolV3 for Peppol Billing V3.0 context.
var ContextPeppolV3 = Context{
	GuidelineID: guidelineIDEN16931V2017 + "#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	BusinessID:  ProfileIDPeppolBilling,
	Version:     VersionD16B,
	Addons:      []cbc.Key{en16931.V2017},
	VESID:       vesIDEN16931CII,
}

// ContextFacturXV1 is used for Factur-X V1 documents.
var ContextFacturXV1 = Context{
	GuidelineID: guidelineIDEN16931V2017,
	Version:     VersionD22B,
	Addons:      []cbc.Key{facturx.V1},
	VESID:       "fr.factur-x:en16931:1.0.8",
}

// ContextPeppolFranceFacturXV1 is used for Peppol France Factur-X documents.
var ContextPeppolFranceFacturXV1 = Context{
	GuidelineID:       guidelineIDEN16931V2017 + "#conformant#urn:peppol:france:billing:Factur-X:1.0",
	BusinessID:        ProfileIDPeppolFranceBilling,
	OutputGuidelineID: guidelineIDEN16931V2017 + "#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr",
	Version:           VersionD16B,
	Addons:            []cbc.Key{flow2.V1},
	VESID:             "fr.ctc:extended-cii:1.3.1",
}

// ContextPeppolFranceCIUSV1 is used for Peppol France CIUS documents.
var ContextPeppolFranceCIUSV1 = Context{
	GuidelineID:       guidelineIDEN16931V2017 + "#compliant#urn:peppol:france:billing:cius:1.0",
	BusinessID:        ProfileIDPeppolFranceBilling,
	OutputGuidelineID: guidelineIDEN16931V2017,
	Version:           VersionD22B,
	Addons:            []cbc.Key{flow2.V1},
	VESID:             "fr.ctc:cii:1.3.1",
}

// ContextZUGFeRDV2 is the context used for ZUGFeRD documents.
var ContextZUGFeRDV2 = Context{
	GuidelineID: guidelineIDEN16931V2017,
	Version:     VersionD16B,
	Addons:      []cbc.Key{zugferd.V2},
	VESID:       "de.zugferd:en16931:2.4",
}

// ContextXRechnungV3 is used for XRechnung documents
var ContextXRechnungV3 = Context{
	GuidelineID: guidelineIDEN16931V2017 + "#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0",
	BusinessID:  ProfileIDPeppolBilling,
	Version:     VersionD16B,
	Addons:      []cbc.Key{xrechnung.V3},
	VESID:       "de.xrechnung:cii:3.0.2",
}

// ContextChorusProV1 is used for Chorus Pro V1 documents.
var ContextChorusProV1 = Context{
	GuidelineID: "A1", // Default framework type
	Version:     VersionD16B,
	Addons:      []cbc.Key{choruspro.V1},
	VESID:       "", // ChorusPro does not have a specific VESID
}

// ContextCDARFlow6 is used for French CTC Flow 6 CDARs addressed to an
// end-party: the GuidelineID is the "invoice" URN (BR-FR-CDV-02) with
// the REGULATED BusinessProcessParameter. The ack TypeCode (23 vs 305)
// is independent of the context — it follows the ProcessConditionCode's
// phase (see cdarAckTypeForCode).
var ContextCDARFlow6 = Context{
	// GuidelineID is the Peppol document-type customization used for the
	// busdox SMP lookup / SBD (the receiver registers this exact id);
	// OutputGuidelineID keeps the internal CDV guideline (BR-FR-CDV-02) in
	// the CDAR XML's GuidelineParameter. They differ: the network identifies
	// the doc by the Peppol id, the XML carries the cpro guideline.
	GuidelineID:       "urn:peppol:france:billing:cdv:1.0",
	OutputGuidelineID: CDARGuidelineInvoice,
	// BusinessID is the busdox SBD/SMP process id the receiver registers its
	// CDV service under (cenbii-procid-ubl scheme); OutputBusinessID keeps the
	// CDAR XML's BusinessProcessParameter at the CDV MDT-2 "REGULATED" value.
	BusinessID:        ProfileIDPeppolFranceBilling,
	OutputBusinessID:  "REGULATED",
	Version:           VersionD22B,
	Addons:            []cbc.Key{flow6.V1},
	VESID:             "fr.ctc:cdar:1.3.1",
}

// ContextCDARFlow6PPF is used for French CTC Flow 6 CDAR copies sent to
// the PPF: the GuidelineID is the einvoicingF2 URN per BR-FR-CDV-02, no
// BusinessProcessParameter is emitted, and the single recipient is the
// PPF party (9998 / 0238 / DFH).
var ContextCDARFlow6PPF = Context{
	GuidelineID: CDARGuidelinePPF,
	Addons:      []cbc.Key{flow6.V1},
	VESID:       "fr.ctc:cdar:1.3.1",
}

// contexts is used internally for reverse lookups during parsing.
// When adding new contexts, remember to add them here AND as exported variables above.
var contexts = []Context{
	ContextEN16931V2017, ContextPeppolV3, ContextFacturXV1,
	ContextPeppolFranceFacturXV1, ContextPeppolFranceCIUSV1,
	ContextZUGFeRDV2, ContextXRechnungV3, ContextChorusProV1,
	ContextCDARFlow6, ContextCDARFlow6PPF,
}

// FindContext looks up a context by GuidelineID and optionally BusinessID.
// Returns nil if no matching context is found.
//
// The lookup logic works as follows:
//  1. If the BusinessID is a French billing mode code, checks for a context whose
//     OutputGuidelineID matches (France CIUS documents use EN16931's
//     GuidelineID in the XML but can be identified by their billing mode BusinessID)
//  2. Tries to match on the full GuidelineID (for external identification)
//  3. If not found, tries to match on OutputGuidelineID (for parsing incoming documents)
func FindContext(guidelineID string, businessID string) *Context {
	// French billing mode check: France CIUS documents use the same
	// GuidelineID as EN16931 but can be identified by their BusinessID
	// containing a billing mode code (e.g., "B1", "S1", "M4").
	if isFrenchBillingMode(businessID) {
		for i := range contexts {
			ctx := &contexts[i]
			if ctx.OutputGuidelineID == guidelineID {
				return ctx
			}
		}
	}

	// First pass: try to match on full GuidelineID
	for i := range contexts {
		ctx := &contexts[i]
		if ctx.GuidelineID == guidelineID {
			if ctx.BusinessID != "" && businessID != "" && ctx.BusinessID != businessID {
				continue
			}
			return ctx
		}
	}

	// Second pass: try to match on OutputGuidelineID (for parsing incoming documents)
	for i := range contexts {
		ctx := &contexts[i]
		if ctx.OutputGuidelineID != "" && ctx.OutputGuidelineID == guidelineID {
			return ctx
		}
	}

	return nil
}

// isFrenchBillingMode checks if the given businessID matches a known French
// billing mode code pattern (e.g., "S1", "B1", "M4"). These codes consist of
// a letter (B for goods, S for services, M for mixed) followed by a digit.
func isFrenchBillingMode(businessID string) bool {
	if len(businessID) != 2 {
		return false
	}
	switch businessID[0] {
	case 'B', 'S', 'M':
		return businessID[1] >= '0' && businessID[1] <= '9'
	}
	return false
}

// Parse parses a raw XML CII invoice document and converts it into
// a GOBL envelope. If the type is unsupported, an
// ErrUnknownDocumentType is provided. CDAR and other non-invoice CII
// documents are not supported by Parse and should be handled using Unmarshal.
func Parse(data []byte) (*gobl.Envelope, error) {
	ns, err := extractRootNamespace(data)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	switch ns {
	case NamespaceRSM:
		res, err := parseInvoice(data)
		if err != nil {
			return nil, err
		}
		if err := env.Insert(res); err != nil {
			return nil, err
		}
		return env, nil
	case NamespaceCDARRSM:
		res, err := parseCDAR(data)
		if err != nil {
			return nil, err
		}
		if err := env.Insert(res); err != nil {
			return nil, err
		}
		return env, nil
	default:
		return nil, ErrUnknownDocumentType
	}
}

// Unmarshal detects the document type and unmarshals XML into the appropriate
// Go struct. Returns either *Invoice (for CII) or *CDAR (for acknowledgements).
// This is pure unmarshaling without GOBL conversion.
func Unmarshal(data []byte) (any, error) {
	ns, err := extractRootNamespace(data)
	if err != nil {
		return nil, err
	}

	switch ns {
	case NamespaceRSM:
		// CII Invoice - unmarshal to Invoice struct
		return UnmarshalInvoice(data)
	case NamespaceCDARRSM:
		// CDAR acknowledgement - unmarshal to CDAR struct
		return UnmarshalCDAR(data)
	default:
		return nil, ErrUnknownDocumentType
	}
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

	// Head.From / Head.To routing URIs steer the CDAR issuer/recipient
	// slots for lifecycle documents.
	var from, to cbc.URI
	if env.Head != nil {
		from, to = env.Head.From, env.Head.To
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
	case *bill.Status:
		ctx := o.context
		// If the caller didn't override the default invoice context, fall back
		// to the CDAR Flow 6 context for status documents.
		if ctx.GuidelineID == ContextEN16931V2017.GuidelineID {
			ctx = ContextCDARFlow6
		}
		return newCDAR(doc, ctx, o.sender, from, to)
	case *bill.Payment:
		ctx := o.context
		// Payments (211 / 212 lifecycle messages) share the CDAR Flow 6
		// contexts with statuses.
		if ctx.GuidelineID == ContextEN16931V2017.GuidelineID {
			ctx = ContextCDARFlow6
		}
		return newCDARFromPayment(doc, ctx, o.sender, from, to)
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

type options struct {
	context Context
	sender  *org.Party
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

// WithSenderTradeParty pins the *org.Party emitted as the CDAR
// ExchangedDocument/SenderTradeParty (MDT-21). Use this to carry the
// dematerialisation platform's identity (Name + GlobalID + Inbox +
// RoleCode) on the wire when it isn't anonymous. When unset, the writer
// emits a bare <ram:RoleCode>WK</ram:RoleCode> — matching the anonymous-
// platform pattern used throughout the official UC1 corpus.
//
// The option is a no-op for non-CDAR conversions.
func WithSenderTradeParty(p *org.Party) Option {
	return func(o *options) {
		o.sender = p
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
