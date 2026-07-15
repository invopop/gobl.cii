package cii_test

import (
	"testing"
	"time"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mdt129Issuer converts a 212 receipt referencing an invoice of the given type
// (with an optional explicit issuer identity on the doc ref) and returns the
// referenced-invoice issuer SIREN the CDV carries in MDT-129.
func mdt129Issuer(t *testing.T, docType cbc.Code, docIDs []*org.Identity) string {
	t.Helper()
	docDate := cal.MakeDate(2025, time.July, 1)
	pct := num.MakePercentage(20, 2)
	pmt := &bill.Payment{
		Type:      bill.PaymentTypeReceipt,
		Code:      "PAY-212",
		IssueDate: cal.MakeDate(2025, time.July, 2),
		Currency:  currency.EUR,
		Ext: tax.ExtensionsOf(cbc.CodeMap{
			flow6.ExtKeyStatus:    "212",
			flow6.ExtKeyCondition: flow6.ConditionAmountReceived,
		}),
		Supplier: testSIRENParty("VENDEUR", "100000009"),
		Customer: testSIRENParty("ACHETEUR", "200000008"),
		Lines: []*bill.PaymentLine{{
			Document: &org.DocumentRef{
				Code:       "F202500003",
				IssueDate:  &docDate,
				Ext:        tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyDocumentType: docType}),
				Identities: docIDs,
			},
			Amount: num.MakeAmount(50000, 2),
			Tax: &tax.Total{Categories: []*tax.CategoryTotal{{
				Code:   tax.CategoryVAT,
				Rates:  []*tax.RateTotal{{Base: num.MakeAmount(41667, 2), Percent: &pct, Amount: num.MakeAmount(8333, 2)}},
				Amount: num.MakeAmount(8333, 2),
			}}, Sum: num.MakeAmount(8333, 2)},
		}},
	}

	cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
	require.NoError(t, err)
	require.NotEmpty(t, cdar.AcknowledgementDocuments)
	refs := cdar.AcknowledgementDocuments[0].ReferenceReferencedDocument
	require.NotEmpty(t, refs)
	require.NotNil(t, refs[0].IssuerTradeParty, "MDT-129 issuer party must be present")
	require.NotEmpty(t, refs[0].IssuerTradeParty.GlobalIDs)
	return refs[0].IssuerTradeParty.GlobalIDs[0].Value
}

// TestCDARReferencedIssuerMDT129 covers the referenced-invoice issuer (MDT-129):
// the invoice supplier (seller), including for a self-billed reference — PPF
// names the supplier as MDT-129 even for a self-billed extract (confirmed on
// QUAL 2026-07-15) — with an explicit doc-ref identity overriding the default.
func TestCDARReferencedIssuerMDT129(t *testing.T) {
	t.Run("normal invoice (380): issuer is the supplier", func(t *testing.T) {
		assert.Equal(t, "100000009", mdt129Issuer(t, "380", nil))
	})

	t.Run("self-billed invoice (389): issuer is still the supplier", func(t *testing.T) {
		assert.Equal(t, "100000009", mdt129Issuer(t, "389", nil))
	})

	t.Run("self-billed credit note (261): issuer is still the supplier", func(t *testing.T) {
		assert.Equal(t, "100000009", mdt129Issuer(t, "261", nil))
	})

	t.Run("explicit doc-ref identity wins over the supplier default", func(t *testing.T) {
		// A parsed CDAR round-trips the true issuer onto the doc ref's
		// identities; that identity is authoritative.
		explicit := []*org.Identity{{
			Code: "123456782",
			Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0002"}),
		}}
		assert.Equal(t, "123456782", mdt129Issuer(t, "389", explicit))
	})
}

// TestCDARPaymentVATRoundTrip confirms a 212 receipt's VAT breakdown (MDT-224)
// survives the GOBL → CDAR → GOBL round-trip: the parse recovers the base, rate
// and VAT amount from the gross-and-percentage characteristic.
func TestCDARPaymentVATRoundTrip(t *testing.T) {
	pmt := buildSyntheticPayment(t, num.Amount{})

	cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
	require.NoError(t, err)
	data, err := cdar.Bytes()
	require.NoError(t, err)

	out, err := cii.ParseCDARPayment(data)
	require.NoError(t, err)
	require.Len(t, out.Lines, 1)
	require.NotNil(t, out.Lines[0].Tax, "212 VAT breakdown (MDT-224) must round-trip")
	require.NotEmpty(t, out.Lines[0].Tax.Categories)

	cat := out.Lines[0].Tax.Categories[0]
	assert.Equal(t, tax.CategoryVAT, cat.Code)
	require.NotEmpty(t, cat.Rates)
	require.NotNil(t, cat.Rates[0].Percent)
	assert.Equal(t, "416.67", cat.Rates[0].Base.String())
	assert.Equal(t, "83.33", cat.Rates[0].Amount.String())
}
