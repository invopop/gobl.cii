package gtoc

import (
	"strconv"

	"github.com/invopop/gobl/bill"
)

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

	return lineItem
}

func newTradeSettlement(line *bill.Line) *TradeSettlement {
	var applicableTradeTax []*ApplicableTradeTax
	for _, tax := range line.Taxes {
		tradeTax := &ApplicableTradeTax{
			TaxType: tax.Category.String(),
			TaxCode: FindTaxCode(tax.Rate),
		}

		if tax.Percent != nil {
			tradeTax.TaxRatePercent = tax.Percent.StringWithoutSymbol()
		}

		applicableTradeTax = append(applicableTradeTax, tradeTax)
	}

	settlement := &TradeSettlement{
		ApplicableTradeTax: applicableTradeTax,
		Sum:                line.Total.String(),
	}

	return settlement
}

// NewLines generates lines for XInvoice
func NewLines(lines []*bill.Line) []*Line {
	var Lines []*Line

	for _, line := range lines {
		Lines = append(Lines, newLine(line))
	}

	return Lines
}
