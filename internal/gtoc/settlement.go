package gtoc

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// prepareSettlement creates the ApplicableHeaderTradeSettlement part of a EN 16931 compliant invoice
func (c *Converter) prepareSettlement(inv *bill.Invoice) error {
	c.doc.Transaction.Settlement = &document.Settlement{
		Currency: string(inv.Currency),
	}
	stlm := c.doc.Transaction.Settlement
	if inv.Payment != nil && inv.Payment.Terms != nil {
		stlm.PaymentTerms = []*document.Terms{
			{
				Description: inv.Payment.Terms.Detail,
			},
		}
	}

	if inv.Totals != nil {
		stlm.Tax = newTaxes(inv.Totals.Taxes)
		stlm.Summary = newSummary(inv.Totals, string(inv.Currency))
	}

	if len(inv.Preceding) > 0 {
		pre := inv.Preceding[0]
		stlm.ReferencedDocument = []*document.ReferencedDocument{
			{
				IssuerAssignedID: invoiceNumber(pre.Series, pre.Code),
				IssueDate: &document.FormattedIssueDate{
					DateFormat: &document.Date{
						Value:  formatIssueDate(*pre.IssueDate),
						Format: issueDateFormat,
					},
				},
			},
		}
	}
	if inv.Payment != nil && inv.Payment.Payee != nil {
		stlm.Payee = newPayee(inv.Payment.Payee)
	}

	if inv.Delivery != nil && inv.Delivery.Period != nil {
		stlm.Period = &document.Period{
			Start: &document.IssueDate{
				DateFormat: &document.Date{
					Value:  formatIssueDate(inv.Delivery.Period.Start),
					Format: issueDateFormat,
				},
			},
			End: &document.IssueDate{
				DateFormat: &document.Date{
					Value:  formatIssueDate(inv.Delivery.Period.End),
					Format: issueDateFormat,
				},
			},
		}
	}

	if inv.Payment != nil && inv.Payment.Instructions != nil {
		instr := inv.Payment.Instructions
		means := make([]*document.PaymentMeans, 0)

		if instr.CreditTransfer != nil {
			credit := &document.PaymentMeans{
				TypeCode:    instr.Ext[untdid.ExtKeyPaymentMeans].String(),
				Information: instr.Detail,
				Creditor: &document.Creditor{
					IBAN:   instr.CreditTransfer[0].IBAN,
					Name:   instr.CreditTransfer[0].Name,
					Number: instr.CreditTransfer[0].Number,
				},
			}
			if instr.CreditTransfer[0].BIC != "" {
				credit.CreditorInstitution = &document.CreditorInstitution{
					BIC: instr.CreditTransfer[0].BIC,
				}
			}

			means = append(means, credit)
		}

		if instr.DirectDebit != nil {
			direct := &document.PaymentMeans{
				TypeCode:    instr.Ext[untdid.ExtKeyPaymentMeans].String(),
				Information: instr.Detail,
				Debtor: &document.DebtorAccount{
					IBAN: instr.DirectDebit.Account,
				},
			}

			means = append(means, direct)

			if stlm.PaymentTerms == nil {
				stlm.PaymentTerms = []*document.Terms{
					{
						Mandate: instr.DirectDebit.Ref,
					},
				}
			} else {
				stlm.PaymentTerms[0].Mandate = instr.DirectDebit.Ref
			}

			stlm.CreditorRefID = instr.DirectDebit.Creditor
		}

		if instr.Card != nil {
			card := &document.PaymentMeans{
				TypeCode:    instr.Ext[untdid.ExtKeyPaymentMeans].String(),
				Information: instr.Detail,
				Card: &document.Card{
					ID:   instr.Card.Last4,
					Name: instr.Card.Holder,
				},
			}
			means = append(means, card)
		}

		stlm.PaymentMeans = means
	}

	if len(inv.Charges) > 0 || len(inv.Discounts) > 0 {
		stlm.AllowanceCharges = newAllowanceCharges(inv)
	}

	return nil
}

func newSummary(totals *bill.Totals, currency string) *document.Summary {
	s := &document.Summary{
		TotalAmount:         totals.Total.String(),
		TaxBasisTotalAmount: totals.Total.String(),
		GrandTotalAmount:    totals.TotalWithTax.String(),
		DuePayableAmount:    totals.Payable.String(),
		TaxTotalAmount: &document.TaxTotalAmount{
			Amount:   totals.Tax.String(),
			Currency: currency,
		},
	}

	if totals.Charge != nil {
		s.Charges = totals.Charge.String()
	}

	if totals.Discount != nil {
		s.Discounts = totals.Discount.String()
	}

	return s
}

func newTaxes(total *tax.Total) []*document.Tax {
	var Taxes []*document.Tax

	if total == nil {
		return nil
	}

	for _, category := range total.Categories {
		for _, rate := range category.Rates {
			tax := newTax(rate, category)

			Taxes = append(Taxes, tax)
		}
	}

	return Taxes
}

func newTax(rate *tax.RateTotal, category *tax.CategoryTotal) *document.Tax {
	if rate.Percent == nil {
		return nil
	}

	tax := &document.Tax{
		CalculatedAmount:      rate.Amount.String(),
		TypeCode:              category.Code.String(),
		BasisAmount:           rate.Base.String(),
		CategoryCode:          rate.Ext[untdid.ExtKeyTaxCategory].String(),
		RateApplicablePercent: rate.Percent.StringWithoutSymbol(),
	}

	return tax
}

func newPayee(party *org.Party) *document.Party {
	payee := &document.Party{
		Name:                      party.Name,
		Contact:                   newContact(party),
		PostalTradeAddress:        newPostalTradeAddress(party.Addresses),
		URIUniversalCommunication: newEmail(party.Emails),
	}

	if party.TaxID != nil {
		payee.ID = &document.PartyID{
			Value: party.TaxID.String(),
		}
	}

	return payee
}
