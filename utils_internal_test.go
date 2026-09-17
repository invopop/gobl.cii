package cii

import (
	"testing"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatIssueDate(t *testing.T) {
	t.Run("a date is written in the CII format", func(t *testing.T) {
		assert.Equal(t, "20240115", formatIssueDate(cal.MakeDate(2024, 1, 15)))
	})

	t.Run("a zero date is written as nothing", func(t *testing.T) {
		assert.Equal(t, "", formatIssueDate(cal.Date{}))
	})
}

func TestDocumentDate(t *testing.T) {
	t.Run("a date carries the CII format code", func(t *testing.T) {
		d := documentDate(cal.NewDate(2024, 2, 29))
		require.NotNil(t, d)
		assert.Equal(t, "20240229", d.Value)
		assert.Equal(t, issueDateFormat, d.Format)
	})

	// cal.Period start and end are optional since GOBL v0.505, so a period may
	// carry only one of the two.
	t.Run("an absent date produces no element", func(t *testing.T) {
		assert.Nil(t, documentDate(nil))
	})
}

func TestNormalizeTaxPercent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"a percentage sign is dropped", "21%", "21"},
		{"surrounding space is trimmed", " 21 ", "21"},
		{"both at once", " 8.5% ", "8.5"},
		{"an empty percentage reads as zero", "", "0"},
		{"only a percentage sign reads as zero", "%", "0"},
		{"a value that is not a number is passed through", "N/A", "N/A"},
		{"zero", "0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeTaxPercent(tt.in))
		})
	}
}

func TestContactName(t *testing.T) {
	tests := []struct {
		name string
		in   *org.Name
		want string
	}{
		{"given and surname", &org.Name{Given: testGivenName, Surname: testSurname}, "Jane Sample"},
		{"surname only", &org.Name{Surname: testSurname}, testSurname},
		{"given only", &org.Name{Given: testGivenName}, testGivenName},
		{"an empty name", &org.Name{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, contactName(tt.in))
		})
	}
}

func TestGoblPaymentMeansCode(t *testing.T) {
	tests := []struct {
		code string
		want cbc.Key
	}{
		{"10", pay.MeansKeyCash},
		{"20", pay.MeansKeyCheque},
		{"30", pay.MeansKeyCreditTransfer},
		{"48", pay.MeansKeyCard},
		{"49", pay.MeansKeyDirectDebit},
		{"59", pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA)},
		{"", pay.MeansKeyAny},
		{"97", pay.MeansKeyAny},
		{"not-a-code", pay.MeansKeyAny},
	}

	for _, tt := range tests {
		t.Run("code "+tt.code, func(t *testing.T) {
			assert.Equal(t, tt.want, goblPaymentMeansCode(tt.code))
		})
	}
}

func TestExtractRootNamespace(t *testing.T) {
	t.Run("the namespace of the root element", func(t *testing.T) {
		ns, err := extractRootNamespace([]byte(
			`<?xml version="1.0"?><rsm:CrossIndustryInvoice xmlns:rsm="urn:example:cii"/>`))
		require.NoError(t, err)
		assert.Equal(t, "urn:example:cii", ns)
	})

	t.Run("a document without a namespace", func(t *testing.T) {
		ns, err := extractRootNamespace([]byte(`<Invoice/>`))
		require.NoError(t, err)
		assert.Empty(t, ns)
	})

	t.Run("a document with no elements at all", func(t *testing.T) {
		_, err := extractRootNamespace([]byte(`<?xml version="1.0"?>`))
		assert.ErrorIs(t, err, ErrUnknownDocumentType)
	})

	t.Run("nothing at all", func(t *testing.T) {
		_, err := extractRootNamespace(nil)
		assert.ErrorIs(t, err, ErrUnknownDocumentType)
	})

	t.Run("malformed XML before the root element is an error", func(t *testing.T) {
		// An unterminated comment: the decoder fails before it reaches any
		// element, so there is no namespace to report.
		_, err := extractRootNamespace([]byte(`<!-- unterminated`))
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrUnknownDocumentType)
	})

	t.Run("a root element is reported even if the rest is malformed", func(t *testing.T) {
		// The namespace is known as soon as the root opens, so the truncated
		// remainder never has to be read.
		ns, err := extractRootNamespace([]byte(`<rsm:X xmlns:rsm="urn:x"><`))
		require.NoError(t, err)
		assert.Equal(t, "urn:x", ns)
	})
}

func TestWithSenderTradeParty(t *testing.T) {
	party := &org.Party{Name: "Platform"}
	o := &options{}
	WithSenderTradeParty(party)(o)
	assert.Same(t, party, o.sender)
}

func TestUnmarshal(t *testing.T) {
	t.Run("an unknown namespace is not a document we handle", func(t *testing.T) {
		_, err := Unmarshal([]byte(`<x:Root xmlns:x="urn:not:ours"/>`))
		assert.ErrorIs(t, err, ErrUnknownDocumentType)
	})

	t.Run("a document with no root element", func(t *testing.T) {
		_, err := Unmarshal([]byte(`<?xml version="1.0"?>`))
		assert.ErrorIs(t, err, ErrUnknownDocumentType)
	})

	t.Run("malformed XML before the root element is an error", func(t *testing.T) {
		_, err := Unmarshal([]byte(`<!-- unterminated`))
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrUnknownDocumentType)
	})

	t.Run("a CII invoice namespace yields an Invoice", func(t *testing.T) {
		doc, err := Unmarshal([]byte(
			`<rsm:CrossIndustryInvoice xmlns:rsm="` + NamespaceRSM + `"/>`))
		require.NoError(t, err)
		assert.IsType(t, &Invoice{}, doc)
	})

	t.Run("a CDAR namespace yields a CDAR", func(t *testing.T) {
		doc, err := Unmarshal([]byte(
			`<rsm:CrossIndustryDocumentAcknowledgement xmlns:rsm="` + NamespaceCDARRSM + `"/>`))
		require.NoError(t, err)
		assert.IsType(t, &CDAR{}, doc)
	})
}

func TestNewDeliveryParty(t *testing.T) {
	t.Run("the name and address come through", func(t *testing.T) {
		p := newDeliveryParty(&org.Party{
			Name:      "Warehouse",
			Addresses: []*org.Address{{Locality: "Berlin", Country: "DE"}},
		})
		require.NotNil(t, p)
		assert.Equal(t, "Warehouse", p.Name)
		require.NotNil(t, p.PostalTradeAddress)
		assert.Equal(t, "Berlin", p.PostalTradeAddress.City)
	})

	t.Run("a party without an address", func(t *testing.T) {
		p := newDeliveryParty(&org.Party{Name: "Warehouse"})
		require.NotNil(t, p)
		assert.Nil(t, p.PostalTradeAddress)
	})

	t.Run("no party at all", func(t *testing.T) {
		assert.Nil(t, newDeliveryParty(nil))
	})
}

func TestUnmarshalInvoiceRejectsMalformedXML(t *testing.T) {
	_, err := UnmarshalInvoice([]byte(`<rsm:CrossIndustryInvoice`))
	assert.Error(t, err)
}

func TestUnmarshalCDARRejectsMalformedXML(t *testing.T) {
	_, err := UnmarshalCDAR([]byte(`<rsm:CrossIndustryDocumentAcknowledgement`))
	assert.Error(t, err)
}
