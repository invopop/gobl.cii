package gtoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgreement(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		env, err := LoadTestEnvelope("invoice-complete.json")
		require.NoError(t, err)

		converter := NewConverter()
		_, err = converter.ConvertToCII(env)
		require.NoError(t, err)

		doc := converter.GetDocument()
		assert.Len(t, doc.Transaction.Agreement.AllowanceCharge, 2)
		assert.Nil(t, err)
		assert.Equal(t, "XR-2024-2", doc.Transaction.Agreement.BuyerReference)

	})

	t.Run("should contain the agreement info from credit note", func(t *testing.T) {
		doc, err := NewDocumentFrom("credit-note.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "XR-2024-4", doc.Transaction.Agreement.BuyerReference)

	})
}
