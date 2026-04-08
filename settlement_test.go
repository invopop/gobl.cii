package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettlement(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "EUR", doc.Transaction.Settlement.Currency)
		assert.Equal(t, "lorem ipsum", doc.Transaction.Settlement.PaymentTerms[0].Description)
		assert.Equal(t, "20240227", doc.Transaction.Settlement.PaymentTerms[0].DueDate.DateFormat.Value)
		assert.Equal(t, "1800.00", doc.Transaction.Settlement.Summary.LineTotalAmount)
		assert.Equal(t, "1800.00", doc.Transaction.Settlement.Summary.TaxBasisTotalAmount)
		assert.Equal(t, "2142.00", doc.Transaction.Settlement.Summary.GrandTotalAmount)
		assert.Equal(t, "2142.01", doc.Transaction.Settlement.Summary.DuePayableAmount)
		assert.Equal(t, "0.01", doc.Transaction.Settlement.Summary.RoundingAmount)
		assert.Equal(t, "342.00", doc.Transaction.Settlement.Summary.TaxTotalAmount.Amount)
		assert.Equal(t, "EUR", doc.Transaction.Settlement.Summary.TaxTotalAmount.Currency)
	})

	t.Run("correction-invoice.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/correction-invoice.json")
		require.NoError(t, err)

		assert.Equal(t, "SAMPLE-001", doc.Transaction.Settlement.ReferencedDocument[0].IssuerAssignedID)
		assert.Equal(t, "20240213", doc.Transaction.Settlement.ReferencedDocument[0].IssueDate.DateFormat.Value)
		assert.Equal(t, "102", doc.Transaction.Settlement.ReferencedDocument[0].IssueDate.DateFormat.Format)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-complete.json")
		require.NoError(t, err)

		assert.Equal(t, "30", doc.Transaction.Settlement.PaymentMeans[0].TypeCode)

		assert.Equal(t, "NO9386011117947", doc.Transaction.Settlement.PaymentMeans[0].Creditor.IBAN)

		assert.Equal(t, "1234567890", doc.Transaction.Settlement.PaymentTerms[0].Mandate)
		assert.Equal(t, "DE89370400440532013000", doc.Transaction.Settlement.PaymentMeans[0].Debtor.IBAN)

		assert.Equal(t, "1234", doc.Transaction.Settlement.PaymentMeans[0].Card.ID)
		assert.Equal(t, "John Doe", doc.Transaction.Settlement.PaymentMeans[0].Card.Name)
	})

	t.Run("empty direct debit ref and card fields omitted", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.DirectDebit.Ref = ""
		inv.Payment.Instructions.Card = nil

		doc, err := cii.ConvertInvoice(env)
		require.NoError(t, err)

		for _, term := range doc.Transaction.Settlement.PaymentTerms {
			assert.Empty(t, term.Mandate)
		}
		assert.Nil(t, doc.Transaction.Settlement.PaymentMeans[0].Card)
	})

	t.Run("extension errors", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		inv.Payment.Instructions.Ext = nil

		_, err := cii.ConvertInvoice(env)
		assert.ErrorContains(t, err, "instructions: (ext: (untdid-payment-means: required.).).")
	})

	t.Run("exemption reason from tax notes", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "xrechnung/invoice-de-es-b2b.json")
		require.NoError(t, err)

		tax := doc.Transaction.Settlement.Tax[0]
		assert.Equal(t, "Reverse Charge / Umkehr der Steuerschuld.", tax.ExemptionReason)
		assert.Equal(t, "VATEX-EU-AE", tax.ExemptionReasonCode)
	})

	t.Run("standard_no_exemption_reason", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		for _, tax := range doc.Transaction.Settlement.Tax {
			assert.Empty(t, tax.ExemptionReason)
			assert.Empty(t, tax.ExemptionReasonCode)
		}
	})
}

func TestTaxPointConversion(t *testing.T) {
	tests := []struct {
		name string
		key  cbc.Key
		code string
	}{
		{"issue", tax.PointIssue, "5"},
		{"delivery", tax.PointDelivery, "29"},
		{"payment", tax.PointPayment, "72"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := loadEnvelope(t, "en16931/invoice-complete.json")
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			inv.Tax.Point = tt.key
			doc, err := cii.ConvertInvoice(env)
			require.NoError(t, err)

			// All header-level tax entries should have the code
			for _, tax := range doc.Transaction.Settlement.Tax {
				assert.Equal(t, tt.code, tax.DueDateTypeCode)
			}
		})
	}

	t.Run("unknown key ignored", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Tax.Point = "unknown"
		doc, err := cii.ConvertInvoice(env)
		require.NoError(t, err)

		for _, tax := range doc.Transaction.Settlement.Tax {
			assert.Empty(t, tax.DueDateTypeCode)
		}
	})

	t.Run("empty point ignored", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Tax.Point = ""
		doc, err := cii.ConvertInvoice(env)
		require.NoError(t, err)

		for _, tax := range doc.Transaction.Settlement.Tax {
			assert.Empty(t, tax.DueDateTypeCode)
		}
	})
}

func TestTaxPointRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		key  cbc.Key
		code string
	}{
		{"issue", tax.PointIssue, "5"},
		{"delivery", tax.PointDelivery, "29"},
		{"payment", tax.PointPayment, "72"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := loadEnvelope(t, "en16931/invoice-complete.json")
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			inv.Tax.Point = tt.key
			doc, err := cii.ConvertInvoice(env)
			require.NoError(t, err)

			// Marshal to XML and parse back
			data, err := doc.Bytes()
			require.NoError(t, err)

			parsed, err := cii.Parse(data)
			require.NoError(t, err)
			parsedInv, ok := parsed.Extract().(*bill.Invoice)
			require.True(t, ok)
			assert.Equal(t, tt.key, parsedInv.Tax.Point)
		})
	}
}

func TestParseTaxNotes(t *testing.T) {
	t.Run("outside_scope", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_example7.xml")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Tax)
		require.NotEmpty(t, inv.Tax.Notes)

		var found bool
		for _, note := range inv.Tax.Notes {
			if note.Ext.Get(untdid.ExtKeyTaxCategory) == "O" {
				assert.Equal(t, cbc.Code("VAT"), note.Category)
				assert.Equal(t, "Tax", note.Text)
				found = true
				break
			}
		}
		assert.True(t, found, "should find outside-scope tax note")
	})

	t.Run("exempt", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_example2.xml")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Tax)
		require.NotEmpty(t, inv.Tax.Notes)

		// Find the exempt note
		var found bool
		for _, note := range inv.Tax.Notes {
			if note.Ext.Get(untdid.ExtKeyTaxCategory) == "E" {
				assert.Equal(t, cbc.Code("VAT"), note.Category)
				assert.Equal(t, "Exempt New Means of Transport", note.Text)
				found = true
				break
			}
		}
		assert.True(t, found, "should find exempt tax note")
	})

	t.Run("standard_no_tax_notes", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_example1.xml")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Tax != nil {
			assert.Empty(t, inv.Tax.Notes)
		}
	})
}
