package gtoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeader(t *testing.T) {
	t.Run("should contain the header info from standard invoice", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "SAMPLE-001", doc.ExchangedDocument.ID)
		assert.Equal(t, "380", doc.ExchangedDocument.TypeCode)
		assert.Equal(t, "20240213", doc.ExchangedDocument.IssueDate.Date.Date)
		assert.Equal(t, IssueDateFormat, doc.ExchangedDocument.IssueDate.Date.Format)
	})

	t.Run("should contain the header info from credit note", func(t *testing.T) {
		doc, err := newDocumentFrom("credit-note.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "CN-002", doc.ExchangedDocument.ID)
		assert.Equal(t, "381", doc.ExchangedDocument.TypeCode)
		assert.Equal(t, "20240214", doc.ExchangedDocument.IssueDate.Date.Date)
		assert.Equal(t, IssueDateFormat, doc.ExchangedDocument.IssueDate.Date.Format)
	})

	t.Run("should return self billed type code for self billed invoice", func(t *testing.T) {
		doc, err := newDocumentFrom("self-billed-invoice.json")
		require.NoError(t, err)

		assert.Equal(t, "389", doc.ExchangedDocument.TypeCode)
	})
}
