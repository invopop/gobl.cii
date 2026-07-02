package cii

import (
	"errors"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
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
	// Amount and Percent are parse-only: present in some CII documents but not emitted
	// as PartialPaymentAmount is not in the EN16931/Factur-X/ZUGFeRD schema.
	Amount  string `xml:"ram:PartialPaymentAmount,omitempty"`
	Percent string `xml:"ram:PartialPaymentPercent,omitempty"`
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

// taxPointCIICodeMap maps GOBL tax point keys to UNTDID 2475 codes for CII.
var taxPointCIICodeMap = map[cbc.Key]string{
	tax.PointIssue:    "5",
	tax.PointDelivery: "29",
	tax.PointPayment:  "72",
}

// taxPointCIIKeyMap is the reverse mapping from UNTDID 2475 codes to GOBL tax point keys.
var taxPointCIIKeyMap = map[string]cbc.Key{
	"5":  tax.PointIssue,
	"29": tax.PointDelivery,
	"72": tax.PointPayment,
}

// prepareSettlement creates the ApplicableHeaderTradeSettlement part of a EN 16931 compliant invoice
func newSettlement(inv *bill.Invoice, ctx Context) (*Settlement, error) {
	stlm := &Settlement{
		Currency: string(inv.Currency),
	}

	if inv.Payment != nil && inv.Payment.Terms != nil {
		stlm.PaymentTerms = newPaymentTerms(inv.Payment.Terms)
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
		// BT-8: VAT point date code (UNTDID 2475)
		if inv.Tax != nil && inv.Tax.Point != cbc.KeyEmpty {
			if code, ok := taxPointCIICodeMap[inv.Tax.Point]; ok {
				for _, t := range stlm.Tax {
					t.DueDateTypeCode = code
				}
			}
		}
		stlm.Summary = newSummary(inv.Totals, string(inv.Currency))
	}

	if len(inv.Preceding) > 0 {
		pre := inv.Preceding[0]
		rd := &ReferencedDocument{
			IssuerAssignedID: invoiceNumber(pre.Series, pre.Code),
		}
		// IssueDate (BT-26) is optional; only emit FormattedIssueDateTime when the
		// preceding reference actually has a date, otherwise it renders as an empty
		// element and fails the CII XSD.
		if d := documentDate(pre.IssueDate); d != nil {
			rd.IssueDate = &FormattedIssueDate{DateFormat: d}
		}
		stlm.ReferencedDocument = []*ReferencedDocument{rd}
	}
	if inv.Payment != nil && inv.Payment.Payee != nil {
		stlm.Payee = newPayee(inv.Payment.Payee, ctx)
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

func newPaymentTerms(terms *pay.Terms) []*Terms {
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
		card := &Card{ID: instr.Card.Last4}
		if instr.Card.Holder != "" {
			card.Name = instr.Card.Holder
		}
		pm.Card = card
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
	// BR-O-05: category O (not subject to VAT) must not have a rate
	if !cat.In("O") {
		t.RateApplicablePercent = "0"
		if rate.Percent != nil {
			t.RateApplicablePercent = rate.Percent.StringWithoutSymbol()
		}
	}
	// Set exemption reason code from extensions if provided
	if rate.Ext.Has(cef.ExtKeyVATEX) {
		t.ExemptionReasonCode = rate.Ext.Get(cef.ExtKeyVATEX).String()
	}
	// BT-120: Set exemption reason from tax notes
	if inv.Tax != nil {
		if note := findTaxNote(inv.Tax.Notes, category.Code, rate); note != nil {
			t.ExemptionReason = note.Text
		}
	}
	return t
}

func newPayee(party *org.Party, ctx Context) *Party {
	// Reflects rules from CII-SR-352 to 364 and CII-SR-364
	// These rules are warnings but have been added as they produce cleaner invoices
	p := newParty(party, ctx)
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
	if instr == nil || instr.Ext.Len() == 0 || instr.Ext.Get(untdid.ExtKeyPaymentMeans).String() == "" {
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

// goblAddTaxNotes extracts tax notes from header-level ApplicableTradeTax entries
// and adds them to the invoice's Tax.Notes.
func goblAddTaxNotes(taxes []*Tax, inv *bill.Invoice) {
	for _, t := range taxes {
		if t.ExemptionReason == "" || t.CategoryCode == "" || t.TypeCode == "" {
			continue
		}
		note := &tax.Note{
			Category: cbc.Code(t.TypeCode),
			Text:     t.ExemptionReason,
			Ext:      tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: cbc.Code(t.CategoryCode)}),
		}
		inv.Tax = inv.Tax.MergeNotes(note)
	}
}

// findTaxNote finds a tax note that matches the given category code and rate total
// by comparing category and the UNTDID tax category extension.
func findTaxNote(notes []*tax.Note, catCode cbc.Code, rate *tax.RateTotal) *tax.Note {
	for _, n := range notes {
		if n.Category != catCode {
			continue
		}
		if nc := n.Ext.Get(untdid.ExtKeyTaxCategory); nc != "" && nc == rate.Ext.Get(untdid.ExtKeyTaxCategory) {
			return n
		}
	}
	return nil
}
