package cii

import (
	"errors"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/validation"
)

// Settlement defines the structure of ApplicableHeaderTradeSettlement of the CII standard
type Settlement struct {
	CreditorRefID      string                `xml:"ram:CreditorReferenceID,omitempty"`
	PaymentReference   string                `xml:"ram:PaymentReference,omitempty"`
	Currency           string                `xml:"ram:InvoiceCurrencyCode"`
	Payee              *Party                `xml:"ram:PayeeTradeParty,omitempty"`
	PaymentMeans       []*PaymentMeans       `xml:"ram:SpecifiedTradeSettlementPaymentMeans"`
	Tax                []*Tax                `xml:"ram:ApplicableTradeTax"`
	Period             *Period               `xml:"ram:BillingSpecifiedPeriod,omitempty"`
	AllowanceCharges   []*AllowanceCharge    `xml:"ram:SpecifiedTradeAllowanceCharge,omitempty"`
	PaymentTerms       []*Terms              `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	Summary            *Summary              `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
	ReferencedDocument []*ReferencedDocument `xml:"ram:InvoiceReferencedDocument,omitempty"`
	Advance            []*Advance            `xml:"ram:SpecifiedAdvancePayment,omitempty"`
}

// Terms defines the structure of SpecifiedTradePaymentTerms of the CII standard
type Terms struct {
	Description string     `xml:"ram:Description,omitempty"`
	DueDate     *IssueDate `xml:"ram:DueDateDateTime,omitempty"`
	Mandate     string     `xml:"ram:DirectDebitMandateID,omitempty"`
	Amount      string     `xml:"ram:PartialPaymentAmount,omitempty"`
	Percent     string     `xml:"ram:PartialPaymentPercent,omitempty"`
}

// PaymentMeans defines the structure of SpecifiedTradeSettlementPaymentMeans of the CII standard
type PaymentMeans struct {
	TypeCode            string               `xml:"ram:TypeCode"`
	Information         string               `xml:"ram:Information,omitempty"`
	Card                *Card                `xml:"ram:ApplicableTradeSettlementFinancialCard,omitempty"`
	Debtor              *DebtorAccount       `xml:"ram:PayerPartyDebtorFinancialAccount,omitempty"`
	Creditor            *Creditor            `xml:"ram:PayeePartyCreditorFinancialAccount,omitempty"`
	CreditorInstitution *CreditorInstitution `xml:"ram:PayeeSpecifiedCreditorFinancialInstitution,omitempty"`
}

// DebtorAccount defines the structure of PayerPartyDebtorFinancialAccount of the CII standard
type DebtorAccount struct {
	IBAN string `xml:"ram:IBANID,omitempty"`
}

// Creditor defines the structure of PayeePartyCreditorFinancialAccount of the CII standard
type Creditor struct {
	IBAN   string `xml:"ram:IBANID,omitempty"`
	Name   string `xml:"ram:AccountName,omitempty"`
	Number string `xml:"ram:ProprietaryID,omitempty"`
}

// CreditorInstitution defines the structure of PayeeSpecifiedCreditorFinancialInstitution of the CII standard
type CreditorInstitution struct {
	BIC string `xml:"ram:BICID,omitempty"`
}

// Card defines the structure of ApplicableTradeSettlementFinancialCard of the CII standard
type Card struct {
	ID   string `xml:"ram:ID,omitempty"`
	Name string `xml:"ram:CardholderName,omitempty"`
}

// Advance defines the structure of SpecifiedAdvancePayment of the CII standard
type Advance struct {
	Amount string              `xml:"ram:PaidAmount"`
	Date   *FormattedIssueDate `xml:"ram:FormattedReceivedDateTime,omitempty"`
}

// ReferencedDocument defines the structure of InvoiceReferencedDocument of the CII standard
type ReferencedDocument struct {
	IssuerAssignedID string              `xml:"ram:IssuerAssignedID,omitempty"`
	IssueDate        *FormattedIssueDate `xml:"ram:FormattedIssueDateTime,omitempty"`
}

// Period defines the structure of the ExpectedDeliveryPeriod of the CII standard
type Period struct {
	Description *string    `xml:"ram:Description,omitempty"`
	Start       *IssueDate `xml:"ram:StartDateTime"`
	End         *IssueDate `xml:"ram:EndDateTime"`
}

// Summary defines the structure of SpecifiedTradeSettlementHeaderMonetarySummation of the CII standard
type Summary struct {
	LineTotalAmount     string          `xml:"ram:LineTotalAmount"`
	Charges             string          `xml:"ram:ChargeTotalAmount,omitempty"`
	Discounts           string          `xml:"ram:AllowanceTotalAmount,omitempty"`
	TaxBasisTotalAmount string          `xml:"ram:TaxBasisTotalAmount"`
	TaxTotalAmount      *TaxTotalAmount `xml:"ram:TaxTotalAmount"`
	RoundingAmount      string          `xml:"ram:RoundingAmount,omitempty"`
	GrandTotalAmount    string          `xml:"ram:GrandTotalAmount"`
	TotalPrepaidAmount  string          `xml:"ram:TotalPrepaidAmount,omitempty"`
	DuePayableAmount    string          `xml:"ram:DuePayableAmount"`
}

// TaxTotalAmount defines the structure of the TaxTotalAmount of the CII standard
type TaxTotalAmount struct {
	Amount   string `xml:",chardata"`
	Currency string `xml:"currencyID,attr"`
}

// prepareSettlement creates the ApplicableHeaderTradeSettlement part of a EN 16931 compliant invoice
func newSettlement(inv *bill.Invoice) (*Settlement, error) {
	stlm := &Settlement{
		Currency: string(inv.Currency),
	}

	if inv.Payment != nil && inv.Payment.Terms != nil {
		stlm.PaymentTerms = newPaymentTerms(inv.Payment.Terms, inv.Totals)
	}

	if inv.Totals != nil {
		stlm.Tax = newTaxes(inv, inv.Totals.Taxes)
		// BT-7: VAT point date
		if inv.ValueDate != nil {
			for _, t := range stlm.Tax {
				t.TaxPointDate = &IssueDate{
					DateFormat: documentDate(inv.ValueDate),
				}
			}
		}
		stlm.Summary = newSummary(inv.Totals, string(inv.Currency))
	}

	if len(inv.Preceding) > 0 {
		pre := inv.Preceding[0]
		stlm.ReferencedDocument = []*ReferencedDocument{
			{
				IssuerAssignedID: invoiceNumber(pre.Series, pre.Code),
				IssueDate: &FormattedIssueDate{
					DateFormat: documentDate(pre.IssueDate),
				},
			},
		}
	}
	if inv.Payment != nil && inv.Payment.Payee != nil {
		stlm.Payee = newPayee(inv.Payment.Payee)
	}

	if inv.Delivery != nil && inv.Delivery.Period != nil {
		stlm.Period = &Period{
			Start: &IssueDate{
				DateFormat: documentDate(&inv.Delivery.Period.Start),
			},
			End: &IssueDate{
				DateFormat: documentDate(&inv.Delivery.Period.End),
			},
		}
	}

	if inv.Payment != nil && inv.Payment.Instructions != nil {
		if err := addPaymentInstructions(stlm, inv.Payment.Instructions); err != nil {
			return nil, err
		}
	}

	if len(inv.Charges) > 0 || len(inv.Discounts) > 0 {
		stlm.AllowanceCharges = newAllowanceCharges(inv)
	}

	return stlm, nil
}

func newPaymentTerms(terms *pay.Terms, totals *bill.Totals) []*Terms {
	description := terms.Notes
	if len(terms.DueDates) == 0 {
		if description == "" {
			return nil
		}
		return []*Terms{{Description: description}}
	}
	result := make([]*Terms, 0, len(terms.DueDates))
	for _, dueDate := range terms.DueDates {
		term := &Terms{}
		if description != "" {
			term.Description = description
		}
		if dueDate.Notes != "" {
			if term.Description != "" {
				term.Description = strings.Join([]string{term.Description, dueDate.Notes}, ". ")
			} else {
				term.Description = dueDate.Notes
			}
		}
		if totals != nil && !dueDate.Amount.Equals(totals.Payable) {
			term.Amount = dueDate.Amount.Rescale(2).String()
		}
		if dueDate.Date != nil {
			term.DueDate = &IssueDate{DateFormat: documentDate(dueDate.Date)}
		}
		result = append(result, term)
	}
	return result
}

func addPaymentInstructions(stlm *Settlement, instr *pay.Instructions) error {
	if instr.Ref != "" {
		stlm.PaymentReference = instr.Ref.String()
	}
	pmc, err := getPaymentMeansCode(instr)
	if err != nil {
		return err
	}

	pm := &PaymentMeans{
		TypeCode:    pmc,
		Information: instr.Detail,
	}

	if len(instr.CreditTransfer) > 0 {
		ct := instr.CreditTransfer[0]
		var c *Creditor
		if ct.IBAN != "" {
			c = &Creditor{}
			c.IBAN = ct.IBAN
		}

		if ct.Number != "" {
			if c == nil {
				c = &Creditor{}
			}
			c.Number = ct.Number
		}

		if c != nil {
			pm.Creditor = c
		}

		if ct.BIC != "" {
			pm.CreditorInstitution = &CreditorInstitution{
				BIC: instr.CreditTransfer[0].BIC,
			}
		}
	}

	if instr.DirectDebit != nil {
		if instr.DirectDebit.Account != "" {
			pm.Debtor = &DebtorAccount{IBAN: instr.DirectDebit.Account}
		}
		if instr.DirectDebit.Ref != "" {
			if stlm.PaymentTerms == nil {
				stlm.PaymentTerms = []*Terms{{Mandate: instr.DirectDebit.Ref}}
			} else {
				for _, term := range stlm.PaymentTerms {
					term.Mandate = instr.DirectDebit.Ref
				}
			}
		}
		stlm.CreditorRefID = instr.DirectDebit.Creditor
	}

	if instr.Card != nil && instr.Card.Last4 != "" {
		pm.Card = &Card{
			ID:   instr.Card.Last4,
			Name: instr.Card.Holder,
		}
	}

	stlm.PaymentMeans = []*PaymentMeans{pm}
	return nil
}

func newSummary(totals *bill.Totals, currency string) *Summary {
	s := &Summary{
		LineTotalAmount:     totals.Sum.String(),
		TaxBasisTotalAmount: totals.Total.String(),
		GrandTotalAmount:    totals.TotalWithTax.String(),
		DuePayableAmount:    totals.Payable.String(),
		TaxTotalAmount: &TaxTotalAmount{
			Amount:   totals.Tax.String(),
			Currency: currency,
		},
	}
	if totals.Due != nil {
		s.DuePayableAmount = totals.Due.String()
	}
	if totals.Charge != nil {
		s.Charges = totals.Charge.String()
	}
	if totals.Discount != nil {
		s.Discounts = totals.Discount.String()
	}
	if totals.Advances != nil {
		s.TotalPrepaidAmount = totals.Advances.String()
	}

	if totals.Rounding != nil {
		s.RoundingAmount = totals.Rounding.String()
	}

	return s
}

func newTaxes(inv *bill.Invoice, total *tax.Total) []*Tax {
	if total == nil {
		return nil
	}
	var taxes []*Tax
	for _, category := range total.Categories {
		for _, rate := range category.Rates {
			tax := newTax(inv, rate, category)
			taxes = append(taxes, tax)
		}
	}
	return taxes
}

func newTax(inv *bill.Invoice, rate *tax.RateTotal, category *tax.CategoryTotal) *Tax {
	cat := rate.Ext.Get(untdid.ExtKeyTaxCategory)
	t := &Tax{
		CalculatedAmount: rate.Amount.Rescale(2).String(),
		TypeCode:         category.Code.String(),
		BasisAmount:      rate.Base.String(),
		CategoryCode:     cat.String(),
	}

	// BR-E-05, BR-AG-05, BR-AF-05, BR-AE-05, BR-Z-05
	t.RateApplicablePercent = "0"

	if rate.Percent != nil {
		t.RateApplicablePercent = rate.Percent.StringWithoutSymbol()
	}
	// Set exemption reason code from extensions if provided
	if rate.Ext.Has(cef.ExtKeyVATEX) {
		t.ExemptionReasonCode = rate.Ext.Get(cef.ExtKeyVATEX).String()
	}
	// BT-120: Set exemption reason from legal note for exempt tax categories
	if inv.Notes != nil && cat.In("E", "AE", "K", "G", "O") {
		for _, n := range inv.Notes {
			if n.Key == org.NoteKeyLegal {
				t.ExemptionReason = n.Text
				break
			}
		}
	}
	return t
}

func newPayee(party *org.Party) *Party {
	// Reflects rules from CII-SR-352 to 364 and CII-SR-364
	// These rules are warnings but have been added as they produce cleaner invoices
	p := newParty(party)
	payee := &Party{
		Name: p.Name,
		ID:   p.ID,
	}

	if payee.ID != nil {
		payee.GlobalID = p.GlobalID
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
