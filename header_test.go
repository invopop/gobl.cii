package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeader(t *testing.T) {
	t.Run("should contain the header info from standard invoice", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "SAMPLE-001", doc.ExchangedDocument.ID)
		assert.Equal(t, "380", doc.ExchangedDocument.TypeCode)
		assert.Equal(t, "20240213", doc.ExchangedDocument.IssueDate.DateFormat.Value)
		assert.Equal(t, "102", doc.ExchangedDocument.IssueDate.DateFormat.Format)
	})
	t.Run("document type extension", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-de-de.json")
		inv := env.Extract().(*bill.Invoice)

		out, err := cii.ConvertInvoice(env)
		assert.NoError(t, err)
		assert.Equal(t, "380", out.ExchangedDocument.TypeCode)

		inv.Tax = nil
		_, err = cii.ConvertInvoice(env)
		assert.ErrorContains(t, err, "tax: (ext: (untdid-document-type: required.).).")

		inv.Tax = &bill.Tax{
			Ext: tax.Extensions{},
		}
		_, err = cii.ConvertInvoice(env)
		assert.ErrorContains(t, err, "ext: (untdid-document-type: required.).")
	})

	t.Run("should contain the header info from credit note", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/credit-note.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "CN-002", doc.ExchangedDocument.ID)
		assert.Equal(t, "381", doc.ExchangedDocument.TypeCode)
		assert.Equal(t, "20240214", doc.ExchangedDocument.IssueDate.DateFormat.Value)
		assert.Equal(t, "102", doc.ExchangedDocument.IssueDate.DateFormat.Format)
	})

	t.Run("should return self billed type code for self billed invoice", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/self-billed-invoice.json")
		require.NoError(t, err)

		assert.Equal(t, "389", doc.ExchangedDocument.TypeCode)
	})

}
