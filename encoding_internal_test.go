package cii

import (
	"bytes"
	"os"
	"testing"

	"github.com/invopop/gobl/bill"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A sender whose own encoding broke upstream ships U+FFFD in place of the
// character it lost. GOBL's canonical JSON refuses any document holding one, so
// a single field would otherwise cost us the whole invoice.
func TestCleanDocumentStripsReplacementCharacters(t *testing.T) {
	t.Run("reaches a field no cleanString call covers", func(t *testing.T) {
		// BT-13, the buyer's order reference, is mapped straight to a code.
		in := &Invoice{Transaction: &Transaction{Agreement: &Agreement{
			Purchase: &IssuerID{ID: "Offre n� 0797247"},
		}}}
		cleanDocument(in)
		assert.Equal(t, "Offre n 0797247", in.Transaction.Agreement.Purchase.ID)
	})

	t.Run("reaches text nested in slices", func(t *testing.T) {
		in := &Invoice{Transaction: &Transaction{Lines: []*Line{
			{Product: &Product{Name: "Caf�"}},
		}}}
		cleanDocument(in)
		assert.Equal(t, "Caf", in.Transaction.Lines[0].Product.Name)
	})

	t.Run("leaves a document without one untouched", func(t *testing.T) {
		in := &Invoice{Transaction: &Transaction{Agreement: &Agreement{
			Purchase: &IssuerID{ID: "Offre n° 0797247"},
		}}}
		cleanDocument(in)
		assert.Equal(t, "Offre n° 0797247", in.Transaction.Agreement.Purchase.ID)
	})

	t.Run("survives nil branches", func(t *testing.T) {
		require.NotPanics(t, func() { cleanDocument(&Invoice{}) })
		require.NotPanics(t, func() { cleanDocument((*Invoice)(nil)) })
	})
}

// The failure was not a mangled field but a lost document: GOBL's canonical
// JSON refuses a replacement character, so building the envelope errored out
// and nothing was converted at all.
func TestParseAcceptsReplacementCharacters(t *testing.T) {
	data, err := os.ReadFile("test/data/parse/invoice-test-01.xml")
	require.NoError(t, err)

	broken := bytes.Replace(data,
		[]byte("<ram:BuyerReference>"),
		[]byte("<ram:BuyerReference>Offre n� "), 1)
	require.NotEqual(t, data, broken, "fixture should carry a buyer reference")

	env, err := Parse(broken)
	require.NoError(t, err, "a replacement character must not cost us the document")
	require.NotNil(t, env)

	// Calculate exercises the canonical JSON that rejected the document.
	require.NoError(t, env.Calculate())
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.NotContains(t, inv.Ordering.Code.String(), "�")
}
