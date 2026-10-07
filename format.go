package cii

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/schema"
)

// GuidelineIDEN16931 identifies plain EN 16931 documents (BT-24), which
// most specifications build on.
const GuidelineIDEN16931 = "urn:cen.eu:en16931:2017"

const vesIDEN16931CII = "eu.cen.en16931:cii:1.3.16"

// Profile ID codes
const (
	ProfileIDPeppolBilling = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)

// CII Versions
const (
	VersionD16B string = "D16B"
	VersionD22B string = "D22B"
)

// Format defines a CII format: the guideline and business process its
// documents carry, and the functions that adjust the base import and export
// for the specification it represents.
type Format struct {
	// Key identifies the format in the GOBL convert register, with one layer
	// for each specification it builds on, e.g. "cii+peppol".
	Key cbc.Key
	// Name of the format.
	Name i18n.String
	// Countries where the format applies. Empty means no restriction.
	Countries []l10n.Code
	// Schemas of the GOBL documents the format converts, in both directions.
	Schemas []schema.ID
	// GuidelineID identifies the specification (BT-24).
	GuidelineID string
	// BusinessID identifies the business process (BT-23).
	BusinessID string
	// OutputGuidelineID optionally specifies a different GuidelineID
	// to use in the actual generated CII XML document. If empty, GuidelineID
	// is used. This allows the format to be identified by one ID externally while
	// generating different values in the XML output.
	OutputGuidelineID string
	// Version of the CII syntax the format uses.
	Version string
	// Addons required by the format.
	Addons []cbc.Key
	// VESID is the Validation Exchange Specification ID used for validation
	VESID string
	// Match optionally identifies the format's documents before the
	// GuidelineID and BusinessID are compared.
	Match func(guidelineID, businessID string) bool
	// Fallback optionally claims documents that no format matched.
	Fallback func(guidelineID, businessID string) bool
	// ExportFuncs adjust the CII document exported from GOBL, in order, after
	// the base export.
	ExportFuncs []ExportFunc
	// ImportFuncs adjust the GOBL envelope imported from CII, in order, after
	// the base import and before the document is calculated.
	ImportFuncs []ImportFunc
}

// Is checks if two formats are the same.
func (f *Format) Is(f2 Format) bool {
	return f.GuidelineID == f2.GuidelineID && f.BusinessID == f2.BusinessID
}

// FormatEN16931 is used for EN 16931 documents, and is the default.
var FormatEN16931 = Format{
	Key:         "cii+en16931",
	Name:        i18n.NewString("CII EN 16931"),
	Schemas:     []schema.ID{invoiceSchema},
	GuidelineID: GuidelineIDEN16931,
	Version:     VersionD16B,
	Addons:      []cbc.Key{en16931.V2017},
	VESID:       vesIDEN16931CII,
}

// FormatPeppol for Peppol Billing V3.0.
var FormatPeppol = Format{
	Key:         "cii+peppol",
	Name:        i18n.NewString("CII Peppol BIS Billing 3"),
	Schemas:     []schema.ID{invoiceSchema},
	GuidelineID: GuidelineIDEN16931 + "#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	BusinessID:  ProfileIDPeppolBilling,
	Version:     VersionD16B,
	Addons:      []cbc.Key{en16931.V2017},
	VESID:       vesIDEN16931CII,
}

// formats holds every registered format for lookups during parsing.
var formats []Format

func init() {
	registerFormats(true, []Format{FormatEN16931, FormatPeppol})
}

// RegisterFormats makes the formats available to FindFormat, and registers
// them with the GOBL convert register. Packages that implement regional
// formats call it from their init function.
func RegisterFormats(fs ...Format) {
	registerFormats(false, fs)
}

func registerFormats(fallback bool, fs []Format) {
	formats = append(formats, fs...)
	convert.Register(&converter{formats: fs, fallback: fallback})
}

// FormatFor provides the registered format with the key, or nil.
func FormatFor(key cbc.Key) *Format {
	for i := range formats {
		if formats[i].Key == key {
			return &formats[i]
		}
	}
	return nil
}

// FindFormat looks up a registered format by GuidelineID and optionally
// BusinessID. Returns nil if no matching format is found.
//
// The lookup logic works as follows:
//  1. Formats whose Match function claims the document
//  2. Tries to match on the full GuidelineID (for external identification)
//  3. If not found, tries to match on OutputGuidelineID (for parsing incoming documents)
//  4. Formats whose Fallback function claims the document
func FindFormat(guidelineID string, businessID string) *Format {
	for i := range formats {
		f := &formats[i]
		if f.Match != nil && f.Match(guidelineID, businessID) {
			return f
		}
	}

	// First pass: try to match on full GuidelineID
	for i := range formats {
		f := &formats[i]
		if f.GuidelineID == guidelineID {
			if f.BusinessID != "" && businessID != "" && f.BusinessID != businessID {
				continue
			}
			return f
		}
	}

	// Second pass: try to match on OutputGuidelineID (for parsing incoming documents)
	for i := range formats {
		f := &formats[i]
		if f.OutputGuidelineID != "" && f.OutputGuidelineID == guidelineID {
			return f
		}
	}

	for i := range formats {
		f := &formats[i]
		if f.Fallback != nil && f.Fallback(guidelineID, businessID) {
			return f
		}
	}

	return nil
}
