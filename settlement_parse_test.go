package cii_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGPayment(t *testing.T) {
	e, err := parseInvoiceFrom(t, "invoice-test-04.xml")
	require.NoError(t, err)

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)

	payment := inv.Payment
	assert.NotNil(t, payment)

	assert.Equal(t, "Sample Payee", payment.Payee.Name)
	assert.Len(t, payment.Payee.Addresses, 1)
	assert.Equal(t, "Sample Street 3", payment.Payee.Addresses[0].Street)
	assert.Equal(t, "Sample City", payment.Payee.Addresses[0].Locality)
	assert.Equal(t, cbc.Code("56000"), payment.Payee.Addresses[0].Code)
	assert.Equal(t, l10n.ISOCountryCode("DE"), payment.Payee.Addresses[0].Country)

	assert.NotNil(t, payment.Terms)
	assert.Equal(t, "Partial Payment", payment.Terms.Detail)
	assert.Len(t, payment.Terms.DueDates, 1)
	assert.Equal(t, "2024-10-01", payment.Terms.DueDates[0].Date.String())
	expectedAmount, _ := num.AmountFromString("20.00")
	assert.Equal(t, expectedAmount, payment.Terms.DueDates[0].Amount)

	assert.NotNil(t, payment.Instructions)
	assert.Equal(t, pay.MeansKeyDebitTransfer, payment.Instructions.Key)
	assert.Equal(t, "Barzahlung", payment.Instructions.Detail)

	assert.NotNil(t, payment.Instructions.Card)
	assert.Equal(t, "3456", payment.Instructions.Card.Last4)
	assert.Equal(t, "Schidt", payment.Instructions.Card.Holder)

	assert.Len(t, payment.Instructions.CreditTransfer, 1)
	assert.Equal(t, "123456789012345678", payment.Instructions.CreditTransfer[0].IBAN)
}
