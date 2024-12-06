package gtoc

import (
	"strconv"

	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
)

// NewLines generates lines for XInvoice
func (c *Converter) NewLines(lines []*bill.Line) error {
	var Lines []*document.Line

	for _, l := range lines {
		Lines = append(Lines, newLine(l))
	}

	c.doc.Transaction.Lines = Lines
	return nil
}

func newLine(l *bill.Line) *document.Line {
	if l.Item == nil {
		return nil
	}
	it := l.Item

	lineItem := &document.Line{
		LineDoc: &document.LineDoc{
			ID: strconv.Itoa(l.Index),
		},
		Product: &document.Product{
			Name: it.Name,
		},
		Agreement: &document.LineAgreement{
			NetPrice: &document.NetPrice{
				Amount: it.Price.String(),
			},
		},
		Quantity: &document.LineDelivery{
			Quantity: &document.Quantity{
				Amount:   l.Quantity.String(),
				UnitCode: string(it.Unit.UNECE()),
			},
		},
		TradeSettlement: newTradeSettlement(l),
	}

	if it.Description != "" {
		lineItem.Product.Description = &it.Description
	}

	if len(l.Notes) > 0 {
		var notes []*document.Note
		for _, n := range l.Notes {
			notes = append(notes, &document.Note{
				SubjectCode: n.Key.String(),
				Content:     n.Text,
			})
		}
		lineItem.LineDoc.Note = notes
	}

	if len(it.Identities) > 0 {
		for _, id := range it.Identities {
			if id.Ext.Has(iso.ExtKeySchemeID) {
				lineItem.Product.GlobalID = &document.GlobalID{
					SchemeID: id.Ext[iso.ExtKeySchemeID].String(),
					Value:    id.Code.String(),
				}
			}
		}
	}

	return lineItem
}

func newTradeSettlement(l *bill.Line) *document.TradeSettlement {
	var taxes []*document.Tax
	for _, tax := range l.Taxes {
		t := makeTaxCategory(tax)
		if tax.Percent != nil {
			t.RateApplicablePercent = tax.Percent.StringWithoutSymbol()
		}

		taxes = append(taxes, t)
	}

	stlm := &document.TradeSettlement{
		ApplicableTradeTax: taxes,
		Sum: &document.Summation{
			Amount: l.Total.String(),
		},
	}

	if len(l.Charges) > 0 || len(l.Discounts) > 0 {
		stlm.AllowanceCharge = newLineAllowanceCharges(l)
	}

	return stlm
}
