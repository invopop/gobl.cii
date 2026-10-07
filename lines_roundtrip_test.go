package cii_test

import (
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// richLinesEnvelope loads a complete invoice whose first line carries every
// reference the base export writes, and a breakdown with allowances.
func richLinesEnvelope(t *testing.T) *gobl.Envelope {
	t.Helper()
	env := loadEnvelope(t, "en16931/invoice-complete.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	l := inv.Lines[0]
	l.Discounts, l.Charges = nil, nil
	l.Seller = &org.Party{Name: "Line Seller"}
	l.Identifier = &org.Identity{
		Code: "OBJ-L1",
		Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyReference: "AAA"}),
	}
	l.Item.Identities = []*org.Identity{
		{Code: "CLS-1", Label: "STI"},
		{Code: "BUY-1"},
	}
	price := num.MakeAmount(1000, 2)
	base := num.MakeAmount(2000, 2)
	l.Breakdown = []*bill.SubLine{
		{
			Quantity: num.MakeAmount(2, 0),
			Item:     &org.Item{Name: "Part", Price: &price},
			Charges: []*bill.LineCharge{
				{Amount: num.MakeAmount(100, 2), Base: &base, Reason: "Handling"},
			},
			Discounts: []*bill.LineDiscount{
				{Amount: num.MakeAmount(200, 2), Base: &base, Reason: "Promotion"},
			},
			Identifier: &org.Identity{Code: "OBJ-S1"},
			Order:      "PO-S1",
			Cost:       "ACC-S1",
		},
	}
	require.NoError(t, env.Calculate())
	return env
}

func TestLinesRoundTrip(t *testing.T) {
	env := richLinesEnvelope(t)
	inv := env.Extract().(*bill.Invoice)

	doc, err := cii.ExportInvoice(env, cii.WithFormat(formatSubLines))
	require.NoError(t, err)
	lines := doc.Transaction.Lines
	require.Len(t, lines, len(inv.Lines)+1)

	group := lines[0]
	assert.Equal(t, "Line Seller", group.Agreement.ItemSellerParty.Name)
	assert.Equal(t, "OBJ-L1", group.Agreement.AdditionalReference.ID)
	assert.Equal(t, "AAA", *group.Agreement.AdditionalReference.RefCode)
	assert.Equal(t, "CLS-1", group.Product.Classification.Code.Value)
	assert.Equal(t, "BUY-1", *group.Product.BuyerAssignedID)

	sub := lines[1]
	assert.Equal(t, "OBJ-S1", sub.Agreement.AdditionalReference.ID)
	assert.Equal(t, "PO-S1", sub.Agreement.OrderReference.LineID)
	assert.Equal(t, "ACC-S1", sub.TradeSettlement.AccountingAccount.ID)
	require.Len(t, sub.TradeSettlement.AllowanceCharge, 2)

	data, err := cii.Encode(doc)
	require.NoError(t, err)
	out, err := parseCII(data)
	require.NoError(t, err)
	got := out.Extract().(*bill.Invoice)

	require.Len(t, got.Lines, len(inv.Lines))
	l := got.Lines[0]
	assert.Equal(t, "OBJ-L1", l.Identifier.Code.String())
	assert.Equal(t, "Line Seller", l.Seller.Name)
	require.Len(t, l.Breakdown, 1)
	sl := l.Breakdown[0]
	assert.Equal(t, "Part", sl.Item.Name)
	assert.Len(t, sl.Charges, 1)
	assert.Len(t, sl.Discounts, 1)
	assert.Equal(t, inv.Totals.Payable.String(), got.Totals.Payable.String())
}

func TestExpandGroupLinesMismatch(t *testing.T) {
	env := richLinesEnvelope(t)
	doc, err := cii.ExportInvoice(env)
	require.NoError(t, err)
	doc.Transaction.Lines = nil
	cii.ExpandGroupLines(env.Extract().(*bill.Invoice), doc)
	assert.Empty(t, doc.Transaction.Lines, "left unchanged")
}

func TestLineWithoutItem(t *testing.T) {
	env := loadEnvelope(t, "en16931/invoice-complete.json")
	env.Extract().(*bill.Invoice).Lines[0].Item = nil
	doc, err := cii.ExportInvoice(env)
	require.NoError(t, err)
	assert.Nil(t, doc.Transaction.Lines[0])
}
