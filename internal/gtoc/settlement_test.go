package gtoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettlement(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "EUR", doc.Transaction.Settlement.Currency)
		assert.Equal(t, "lorem ipsum", doc.Transaction.Settlement.PaymentTerms[0].Description)
		assert.Equal(t, "20240227", doc.Transaction.Settlement.PaymentTerms[0].DueDate.DateFormat.Value)
		assert.Equal(t, "2000.00", doc.Transaction.Settlement.PaymentTerms[0].PartialPayment)
		assert.Equal(t, "1800.00", doc.Transaction.Settlement.Summary.TotalAmount)
		assert.Equal(t, "1800.00", doc.Transaction.Settlement.Summary.TaxBasisTotalAmount)
		assert.Equal(t, "2142.00", doc.Transaction.Settlement.Summary.GrandTotalAmount)
		assert.Equal(t, "2142.00", doc.Transaction.Settlement.Summary.DuePayableAmount)
		assert.Equal(t, "342.00", doc.Transaction.Settlement.Summary.TaxTotalAmount.Amount)
		assert.Equal(t, "EUR", doc.Transaction.Settlement.Summary.TaxTotalAmount.Currency)
	})

	t.Run("correction-invoice.json", func(t *testing.T) {
		doc, err := newDocumentFrom("correction-invoice.json")
		require.NoError(t, err)

		assert.Equal(t, "SAMPLE-001", doc.Transaction.Settlement.ReferencedDocument[0].IssuerAssignedID)
		assert.Equal(t, "20240213", doc.Transaction.Settlement.ReferencedDocument[0].IssueDate.DateFormat.Value)
		assert.Equal(t, "102", doc.Transaction.Settlement.ReferencedDocument[0].IssueDate.DateFormat.Format)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-complete.json")
		require.NoError(t, err)

		assert.Equal(t, "30", doc.Transaction.Settlement.PaymentMeans[0].TypeCode)

		assert.Equal(t, "NO9386011117947", doc.Transaction.Settlement.PaymentMeans[0].Creditor.IBAN)

		assert.Equal(t, "1234567890", doc.Transaction.Settlement.PaymentTerms[0].Mandate)
		assert.Equal(t, "DE89370400440532013000", doc.Transaction.Settlement.PaymentMeans[1].Debtor.IBAN)

		assert.Equal(t, "1234", doc.Transaction.Settlement.PaymentMeans[2].Card.ID)
		assert.Equal(t, "John Doe", doc.Transaction.Settlement.PaymentMeans[2].Card.Name)
	})
}
