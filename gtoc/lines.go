package gtoc

import (
	"strconv"

	"github.com/invopop/gobl/bill"
)

// NewLines generates lines for XInvoice
func (c *Converter) NewLines(lines []*bill.Line) error {
	var Lines []*Line

	for _, l := range lines {
		Lines = append(Lines, newLine(l))
	}

	c.doc.Transaction.Lines = Lines
	return nil
}

func newLine(l *bill.Line) *Line {
	if l.Item == nil {
		return nil
	}
	it := l.Item

	lineItem := &Line{
		ID:       strconv.Itoa(l.Index),
		Name:     it.Name,
		NetPrice: it.Price.String(),
		TradeDelivery: &Quantity{
			Amount:   l.Quantity.String(),
			UnitCode: string(it.Unit.UNECE()),
		},
		TradeSettlement: newTradeSettlement(l),
	}

	if len(l.Notes) > 0 {
		var notes []Note
		for _, n := range l.Notes {
			notes = append(notes, Note{
				SubjectCode: n.Key.String(),
				Content:     n.Text,
			})
		}
		lineItem.Note = notes
	}

	return lineItem
}

func newTradeSettlement(l *bill.Line) *TradeSettlement {
	var taxes []*Tax
	for _, tax := range l.Taxes {
		t := makeTaxCategory(tax)
		if tax.Percent != nil {
			t.RateApplicablePercent = tax.Percent.StringWithoutSymbol()
		}

		taxes = append(taxes, t)
	}

	stlm := &TradeSettlement{
		ApplicableTradeTax: taxes,
		Sum:                l.Total.String(),
	}

	if len(l.Charges) > 0 || len(l.Discounts) > 0 {
		stlm.AllowanceCharge = newLineAllowanceCharges(l)
	}

	return stlm
}
