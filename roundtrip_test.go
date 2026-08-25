package cii_test

import (
	"path/filepath"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFacturXRoundTrip converts the France Factur-X example to CII and parses
// it back, asserting that fields which used to be lost in one of the two
// directions survive the round trip.
func TestFacturXRoundTrip(t *testing.T) {
	env := loadEnvelope(t, filepath.Join(dirFRFacturX, "invoice-standard.json"))
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	// The example has no contact people; add one to cover the role mapping.
	inv.Supplier.People = []*org.Person{
		{
			Name: &org.Name{Given: "Jean", Surname: "Dupont"},
			Role: "Accounting",
		},
	}
	require.NoError(t, env.Calculate())

	out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextPeppolFranceFacturXV1))
	require.NoError(t, err)
	xmlData, err := out.Bytes()
	require.NoError(t, err)

	env2, err := cii.Parse(xmlData)
	require.NoError(t, err)
	inv2, ok := env2.Extract().(*bill.Invoice)
	require.True(t, ok)

	t.Run("legal identity type", func(t *testing.T) {
		require.NotEmpty(t, inv2.Supplier.Identities)
		id := inv2.Supplier.Identities[0]
		assert.Equal(t, cbc.Code("SIREN"), id.Type)
		assert.Equal(t, cbc.Code("356000000"), id.Code)
	})

	t.Run("contact person role", func(t *testing.T) {
		require.NotEmpty(t, inv2.Supplier.People)
		person := inv2.Supplier.People[0]
		assert.Equal(t, "Jean Dupont", person.Name.Given)
		assert.Equal(t, "Accounting", person.Role)
	})

	t.Run("note keys", func(t *testing.T) {
		keys := make([]cbc.Key, 0, len(inv2.Notes))
		for _, n := range inv2.Notes {
			keys = append(keys, n.Key)
		}
		assert.Contains(t, keys, org.NoteKeyPayment)
		assert.Contains(t, keys, org.NoteKeyPaymentMethod)
		assert.Contains(t, keys, org.NoteKeyPaymentTerm)
	})

	t.Run("due date amount and percent", func(t *testing.T) {
		require.NotNil(t, inv2.Payment)
		require.NotNil(t, inv2.Payment.Terms)
		require.NotEmpty(t, inv2.Payment.Terms.DueDates)
		dd := inv2.Payment.Terms.DueDates[0]
		assert.Equal(t, "120.00", dd.Amount.String())
		require.NotNil(t, dd.Percent)
		assert.Equal(t, "10.0%", dd.Percent.String())
	})

	t.Run("credit transfer account name", func(t *testing.T) {
		require.NotNil(t, inv2.Payment.Instructions)
		require.NotEmpty(t, inv2.Payment.Instructions.CreditTransfer)
		assert.Equal(t, "Supplier SARL", inv2.Payment.Instructions.CreditTransfer[0].Name)
	})
}

// TestParseSingleDueDateAmount checks that a single payment due date without
// an explicit amount recovers the full payable amount (BT-9 semantics).
func TestParseSingleDueDateAmount(t *testing.T) {
	env, err := parseInvoiceFrom(t, "CII_example1.xml")
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotNil(t, inv.Payment)
	require.NotNil(t, inv.Payment.Terms)
	require.Len(t, inv.Payment.Terms.DueDates, 1)
	assert.Equal(t, "250.33", inv.Payment.Terms.DueDates[0].Amount.String())
}
