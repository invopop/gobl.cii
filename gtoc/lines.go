package gtoc

import (
	"strconv"

	"github.com/invopop/gobl/bill"
)

// NewLines generates lines for XInvoice
func (c *Converter) NewLines(lines []*bill.Line) error {
	var Lines []*Line

	for _, line := range lines {
		Lines = append(Lines, newLine(line))
	}

	c.doc.Transaction.Lines = Lines
	return nil
}

func newLine(line *bill.Line) *Line {
	if line.Item == nil {
		return nil
	}
	item := line.Item

	lineItem := &Line{
		ID:       strconv.Itoa(line.Index),
		Name:     item.Name,
		NetPrice: item.Price.String(),
		TradeDelivery: &Quantity{
			Amount:   line.Quantity.String(),
			UnitCode: string(item.Unit.UNECE()),
		},
		TradeSettlement: newTradeSettlement(line),
	}

	if len(line.Notes) > 0 {
		var notes []Note
		for _, note := range line.Notes {
			notes = append(notes, Note{
				SubjectCode: note.Key.String(),
				Content:     note.Text,
			})
		}
		lineItem.Note = notes
	}

	return lineItem
}

func newTradeSettlement(line *bill.Line) *TradeSettlement {
	var applicableTradeTax []*Tax
	for _, tax := range line.Taxes {
		tradeTax := makeTaxCategory(tax)
		if tax.Percent != nil {
			tradeTax.RateApplicablePercent = tax.Percent.StringWithoutSymbol()
		}

		applicableTradeTax = append(applicableTradeTax, tradeTax)
	}

	settlement := &TradeSettlement{
		ApplicableTradeTax: applicableTradeTax,
		Sum:                line.Total.String(),
	}

	if len(line.Charges) > 0 || len(line.Discounts) > 0 {
		settlement.AllowanceCharge = newLineAllowanceCharges(line)
	}

	return settlement
}
