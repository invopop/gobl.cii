package gtoc

import (
	"errors"

	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/validation"
)

// prepareSettlement creates the ApplicableHeaderTradeSettlement part of a EN 16931 compliant invoice
func (c *Converter) prepareSettlement(inv *bill.Invoice) error {
	c.doc.Transaction.Settlement = &document.Settlement{
		Currency: string(inv.Currency),
	}
	stlm := c.doc.Transaction.Settlement
	if inv.Payment != nil && inv.Payment.Terms != nil {
		description := inv.Payment.Terms.Detail
		if len(inv.Payment.Terms.DueDates) == 0 {
			stlm.PaymentTerms = []*document.Terms{
				{
					Description: description,
				},
			}
		} else {
			for _, dueDate := range inv.Payment.Terms.DueDates {
				term := &document.Terms{
					Description: description,
				}

				if !dueDate.Amount.Equals(inv.Totals.Payable) {
					term.PartialPayment = dueDate.Amount.Rescale(2).String()
				}

				if dueDate.Date != nil {
					term.DueDate = &document.IssueDate{
						DateFormat: documentDate(dueDate.Date),
					}
				}

				stlm.PaymentTerms = append(stlm.PaymentTerms, term)
			}
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
					DateFormat: documentDate(pre.IssueDate),
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
				DateFormat: documentDate(&inv.Delivery.Period.Start),
			},
			End: &document.IssueDate{
				DateFormat: documentDate(&inv.Delivery.Period.End),
			},
		}
	}

	if inv.Payment != nil && inv.Payment.Instructions != nil {
		instr := inv.Payment.Instructions
		means := make([]*document.PaymentMeans, 0)
		pmc, err := getPaymentMeansCode(instr)
		if err != nil {
			return err
		}

		if instr.CreditTransfer != nil {
			credit := &document.PaymentMeans{
				TypeCode:    pmc,
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
				TypeCode:    pmc,
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
				for _, term := range stlm.PaymentTerms {
					term.Mandate = instr.DirectDebit.Ref
				}
			}

			stlm.CreditorRefID = instr.DirectDebit.Creditor
		}

		if instr.Card != nil {
			card := &document.PaymentMeans{
				TypeCode:    pmc,
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
		LineTotalAmount:     totals.Sum.Rescale(2).String(),
		TaxBasisTotalAmount: totals.Total.Rescale(2).String(),
		GrandTotalAmount:    totals.TotalWithTax.Rescale(2).String(),
		DuePayableAmount:    totals.Payable.Rescale(2).String(),
		TaxTotalAmount: &document.TaxTotalAmount{
			Amount:   totals.Tax.Rescale(2).String(),
			Currency: currency,
		},
	}

	if totals.Charge != nil {
		s.Charges = totals.Charge.Rescale(2).String()
	}

	if totals.Discount != nil {
		s.Discounts = totals.Discount.Rescale(2).String()
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
		CalculatedAmount:      rate.Amount.Rescale(2).String(),
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

func getPaymentMeansCode(instr *pay.Instructions) (string, error) {
	if instr == nil || instr.Ext == nil || instr.Ext[untdid.ExtKeyPaymentMeans].String() == "" {
		return "", validation.Errors{
			"instructions": validation.Errors{
				"ext": validation.Errors{
					untdid.ExtKeyPaymentMeans.String(): errors.New("required"),
				},
			},
		}
	}
	return instr.Ext.Get(untdid.ExtKeyPaymentMeans).String(), nil
}
