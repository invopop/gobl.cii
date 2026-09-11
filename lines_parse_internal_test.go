package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoblLineAgreement(t *testing.T) {
	ref := func(s string) *string { return &s }

	t.Run("BT-128 object identifier with its scheme", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{
			AdditionalReference: &LineDocReference{ID: "OBJ-1", TypeCode: "130", RefCode: ref("AWV")},
		}, l)

		require.NotNil(t, l.Identifier)
		assert.Equal(t, "OBJ-1", l.Identifier.Code.String())
		assert.Equal(t, "AWV", l.Identifier.Ext.Get(untdid.ExtKeyReference).String())
	})

	t.Run("an object identifier without a scheme", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{
			AdditionalReference: &LineDocReference{ID: "OBJ-1", TypeCode: "130"},
		}, l)

		require.NotNil(t, l.Identifier)
		assert.True(t, l.Identifier.Ext.IsZero())
	})

	t.Run("another document type is not an object identifier", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{
			AdditionalReference: &LineDocReference{ID: testDocID, TypeCode: "916"},
		}, l)
		assert.Nil(t, l.Identifier)
	})

	t.Run("a reference without an id is ignored", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{
			AdditionalReference: &LineDocReference{TypeCode: "130"},
		}, l)
		assert.Nil(t, l.Identifier)
	})

	t.Run("BT-132 purchase order line reference", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{OrderReference: &LineOrderReference{LineID: "PO-LINE-5"}}, l)
		assert.Equal(t, "PO-LINE-5", l.Order.String())
	})

	t.Run("an order reference without a line id is ignored", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{OrderReference: &LineOrderReference{}}, l)
		assert.Empty(t, l.Order.String())
	})

	t.Run("an agreement with neither", func(t *testing.T) {
		l := &bill.Line{}
		goblLineAgreement(&LineAgreement{}, l)
		assert.Nil(t, l.Identifier)
		assert.Empty(t, l.Order.String())
	})
}

func TestGoblAddTaxDates(t *testing.T) {
	t.Run("no taxes means no dates", func(t *testing.T) {
		out := &bill.Invoice{}
		require.NoError(t, goblAddTaxDates(nil, out))
		assert.Nil(t, out.ValueDate)
	})

	t.Run("BT-7 value date", func(t *testing.T) {
		out := &bill.Invoice{Tax: &bill.Tax{}}
		require.NoError(t, goblAddTaxDates([]*Tax{{
			TaxPointDate: issueDate("20240115"),
		}}, out))

		require.NotNil(t, out.ValueDate)
		assert.Equal(t, "2024-01-15", out.ValueDate.String())
	})

	t.Run("a value date that is not a date is an error", func(t *testing.T) {
		out := &bill.Invoice{Tax: &bill.Tax{}}
		err := goblAddTaxDates([]*Tax{{
			TaxPointDate: issueDate("15/01/2024"),
		}}, out)
		assert.Error(t, err)
	})

	t.Run("BT-8 value date codes", func(t *testing.T) {
		for code, want := range map[string]interface{ String() string }{
			"5":  tax.PointIssue,
			"29": tax.PointDelivery,
			"72": tax.PointPayment,
		} {
			out := &bill.Invoice{Tax: &bill.Tax{}}
			require.NoError(t, goblAddTaxDates([]*Tax{{DueDateTypeCode: code}}, out))
			assert.Equal(t, want.String(), out.Tax.Point.String(), "code %s", code)
		}
	})

	t.Run("an unmapped code leaves the point unset", func(t *testing.T) {
		out := &bill.Invoice{Tax: &bill.Tax{}}
		require.NoError(t, goblAddTaxDates([]*Tax{{DueDateTypeCode: "99"}}, out))
		assert.Empty(t, out.Tax.Point.String())
	})

	t.Run("only the first tax is consulted", func(t *testing.T) {
		out := &bill.Invoice{Tax: &bill.Tax{}}
		require.NoError(t, goblAddTaxDates([]*Tax{
			{DueDateTypeCode: "5"},
			{DueDateTypeCode: "72"},
		}, out))
		assert.Equal(t, tax.PointIssue.String(), out.Tax.Point.String())
	})
}

func TestGoblLinePeriod(t *testing.T) {
	t.Run("both ends", func(t *testing.T) {
		p, err := goblLinePeriod(&Period{Start: issueDate("20240101"), End: issueDate("20240131")})
		require.NoError(t, err)
		require.NotNil(t, p)
		require.NotNil(t, p.Start)
		require.NotNil(t, p.End)
		assert.Equal(t, "2024-01-01", p.Start.String())
		assert.Equal(t, "2024-01-31", p.End.String())
	})

	// GOBL v0.505 made both ends optional, so a half-open period is valid.
	t.Run("start only", func(t *testing.T) {
		p, err := goblLinePeriod(&Period{Start: issueDate("20240101")})
		require.NoError(t, err)
		require.NotNil(t, p)
		require.NotNil(t, p.Start)
		assert.Nil(t, p.End)
	})

	t.Run("end only", func(t *testing.T) {
		p, err := goblLinePeriod(&Period{End: issueDate("20240131")})
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Nil(t, p.Start)
		require.NotNil(t, p.End)
	})

	t.Run("an empty period is no period", func(t *testing.T) {
		p, err := goblLinePeriod(&Period{})
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("a period without date formats is no period", func(t *testing.T) {
		p, err := goblLinePeriod(&Period{Start: &IssueDate{}, End: &IssueDate{}})
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("no period at all", func(t *testing.T) {
		p, err := goblLinePeriod(nil)
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("a malformed start date is an error", func(t *testing.T) {
		_, err := goblLinePeriod(&Period{Start: issueDate("01/01/2024")})
		assert.Error(t, err)
	})

	t.Run("a malformed end date is an error", func(t *testing.T) {
		_, err := goblLinePeriod(&Period{Start: issueDate("20240101"), End: issueDate("31/01/2024")})
		assert.Error(t, err)
	})
}

func TestGoblLineSettlement(t *testing.T) {
	t.Run("BT-133 line buyer accounting reference", func(t *testing.T) {
		l := &bill.Line{}
		goblLineSettlement(&TradeSettlement{AccountingAccount: &AccountingAccount{ID: testCostRef}}, l)
		assert.Equal(t, testCostRef, l.Cost.String())
	})

	t.Run("an account without an id is ignored", func(t *testing.T) {
		l := &bill.Line{}
		goblLineSettlement(&TradeSettlement{AccountingAccount: &AccountingAccount{}}, l)
		assert.Empty(t, l.Cost.String())
	})

	t.Run("no accounting account", func(t *testing.T) {
		l := &bill.Line{}
		goblLineSettlement(&TradeSettlement{}, l)
		assert.Empty(t, l.Cost.String())
	})
}
