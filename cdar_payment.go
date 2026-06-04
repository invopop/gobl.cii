package cii

import (
	"fmt"

	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
)

// NewCDARFromPayment converts a *bill.Payment into a CDAR XML document
// using the provided context. Flow 6 payment lifecycle messages map a
// payment advice to ProcessConditionCode 211 (Paiement transmis) and a
// payment receipt to 212 (Encaissée); the code is read from the
// fr-ctc-flow6-status extension that the flow6 normalizer derives from
// Payment.Type — run Calculate on the enclosing envelope first.
//
// Amounts are carried as SpecifiedDocumentCharacteristics: the payment's
// fr-ctc-flow6-condition extension (MEN / MPA / RAP) types the main
// amount characteristic for each line, and a line with a Due remainder
// also emits a RAP (reste à payer) characteristic — one 212 per partial
// cash-in event.
func NewCDARFromPayment(pmt *bill.Payment, ctx Context) (*CDAR, error) {
	return newCDARFromPayment(pmt, ctx, nil)
}

// ParseCDARPayment parses a raw CDAR XML byte slice carrying a payment
// lifecycle code (211 / 212) into a *bill.Payment without going through
// an envelope. Like ParseCDARStatus, the returned document carries the
// CDAR codes on the flow6 extensions; wrap it in an envelope or run
// Calculate to complete the derived fields.
func ParseCDARPayment(data []byte) (*bill.Payment, error) {
	cdar, err := UnmarshalCDAR(data)
	if err != nil {
		return nil, err
	}
	return goblPaymentFromCDAR(cdar)
}

// NewCDARFromPaymentWithSender is the same as NewCDARFromPayment but
// carries the supplied *org.Party as the CDAR
// ExchangedDocument/SenderTradeParty (MDT-21) — typically the
// dematerialisation platform's identity.
func NewCDARFromPaymentWithSender(pmt *bill.Payment, ctx Context, sender *org.Party) (*CDAR, error) {
	return newCDARFromPayment(pmt, ctx, sender)
}

func newCDARFromPayment(pmt *bill.Payment, ctx Context, sender *org.Party) (*CDAR, error) {
	if pmt == nil {
		return nil, fmt.Errorf("nil bill.Payment")
	}

	processCode := pmt.Ext.Get(flow6.ExtKeyStatus)
	if processCode == "" {
		return nil, fmt.Errorf("payment missing %s extension; run Calculate with the %s addon declared", flow6.ExtKeyStatus, flow6.V1)
	}

	guideline := ctx.GuidelineID
	if guideline == "" {
		guideline = ContextCDARFlow6.GuidelineID
	}
	// 211 / 212 are treatment-phase, business-issued events.
	ackType := cdarAckTypeForCode(processCode)

	cdar := NewCDAR()
	cdar.ExchangedDocumentContext = &CDARExchangedContext{
		GuidelineParameter: &CDARDocumentContextParameter{ID: guideline},
	}
	if ctx.BusinessID != "" {
		cdar.ExchangedDocumentContext.BusinessProcessParameter = &CDARDocumentContextParameter{ID: ctx.BusinessID}
	}

	cdar.ExchangedDocument = &CDARExchangedDocument{
		ID:            string(pmt.Code),
		IssueDateTime: makeIssueDateTime(pmt.IssueDate, pmt.IssueTime),
	}
	if pmt.Series != "" && cdar.ExchangedDocument.ID == "" {
		cdar.ExchangedDocument.ID = string(pmt.Series)
	}

	// Party slots from payment semantics: a receipt (212) is declared
	// by the payee — the Supplier (SE) — towards the Customer; an
	// advice (211) by the payer — the Customer (BY) — towards the
	// Supplier. einvoicingF2 copies go to the single PPF recipient.
	if sender == nil {
		sender = bareWKParty()
	}
	cdar.ExchangedDocument.SenderTradeParty = newCDARTradeParty(sender)

	issuer, recipient := pmt.Supplier, pmt.Customer
	if pmt.Type == bill.PaymentTypeAdvice {
		issuer, recipient = pmt.Customer, pmt.Supplier
	}
	if issuer != nil {
		cdar.ExchangedDocument.IssuerTradeParty = newCDARTradeParty(issuer)
	}
	if guideline == CDARGuidelinePPF {
		cdar.ExchangedDocument.RecipientTradeParties = []*CDARTradeParty{
			ppfTradeParty(),
		}
	} else if recipient != nil {
		cdar.ExchangedDocument.RecipientTradeParties = []*CDARTradeParty{
			newCDARTradeParty(recipient),
		}
	}

	for _, line := range pmt.Lines {
		if line == nil {
			continue
		}
		ack := newCDARPaymentAcknowledgement(pmt, line, ackType, processCode.String())
		cdar.AcknowledgementDocuments = append(cdar.AcknowledgementDocuments, ack)
	}

	return cdar, nil
}

func newCDARPaymentAcknowledgement(pmt *bill.Payment, line *bill.PaymentLine, ackType, processCode string) *CDARAcknowledgement {
	ack := &CDARAcknowledgement{
		MultipleReferencesIndicator: &CDARIndicator{Value: false},
		TypeCode:                    ackType,
		IssueDateTime:               makeIssueDateTime(pmt.IssueDate, pmt.IssueTime),
	}

	ref := &CDARReferencedDocument{
		ProcessConditionCode: processCode,
	}
	if line.Document != nil {
		ref.IssuerAssignedID = string(line.Document.Code)
		if line.Document.IssueDate != nil {
			ref.FormattedIssueDateTime = &CDARFormattedIssueDateTime{
				DateTimeString: &CDARQDTDateTimeString{
					Value:  formatCDARDate(*line.Document.IssueDate),
					Format: cdarDateFormat,
				},
			}
		}
		if line.Document.Type != "" {
			ref.TypeCode = string(line.Document.Type)
		}
	}
	if pmt.Supplier != nil {
		if siren := partyIdentityCode(pmt.Supplier, schemeIDSIREN); siren != "" {
			ref.IssuerTradeParty = &CDARTradeParty{
				GlobalIDs: []*CDARGlobalID{{SchemeID: schemeIDSIREN, Value: siren}},
			}
		}
	}

	// Amounts ride as document characteristics on a single status
	// entry: the payment's condition extension types the line amount
	// (MEN for receipts, MPA for advices) and any Due remainder emits
	// an additional RAP characteristic for partial payments.
	ds := &CDARDocumentStatus{SequenceNumeric: 1}
	condition := pmt.Ext.Get(flow6.ExtKeyCondition)
	if condition == "" {
		condition = flow6.ConditionAmountReceived
	}
	ds.SpecifiedDocumentCharacteristics = append(ds.SpecifiedDocumentCharacteristics,
		newCDARAmountCharacteristic(condition, line.Amount, pmt.Currency))
	if line.Due != nil && condition != flow6.ConditionAmountRemaining {
		ds.SpecifiedDocumentCharacteristics = append(ds.SpecifiedDocumentCharacteristics,
			newCDARAmountCharacteristic(flow6.ConditionAmountRemaining, *line.Due, pmt.Currency))
	}
	ref.SpecifiedDocumentStatuses = []*CDARDocumentStatus{ds}

	ack.ReferenceReferencedDocument = []*CDARReferencedDocument{ref}
	return ack
}

func newCDARAmountCharacteristic(typeCode cbc.Code, amount num.Amount, cur currency.Code) *CDARDocumentCharacteristic {
	return &CDARDocumentCharacteristic{
		TypeCode: typeCode.String(),
		ValueAmount: &CDARValueAmount{
			Value:      amount.String(),
			CurrencyID: cur.String(),
		},
	}
}

// goblPaymentFromCDAR converts a parsed CDAR carrying a payment
// lifecycle code (211 / 212) into a *bill.Payment.
//
// CDARs carry no payment-method detail, so a single bare pay.Record
// with the line amount is synthesized to satisfy the core "at least
// one payment method" rule; the receiving system can enrich it.
func goblPaymentFromCDAR(cdar *CDAR) (*bill.Payment, error) {
	if cdar == nil || cdar.ExchangedDocument == nil {
		return nil, fmt.Errorf("invalid CDAR document")
	}
	pmt := &bill.Payment{}
	pmt.SetAddons(flow6.V1)

	code := cbc.Code(cdarProcessCode(cdar))
	switch code {
	case "211":
		pmt.Type = bill.PaymentTypeAdvice
	case "212":
		pmt.Type = bill.PaymentTypeReceipt
	default:
		return nil, fmt.Errorf("CDAR process code %q is not a payment lifecycle code", code)
	}
	pmt.Ext = pmt.Ext.Set(flow6.ExtKeyStatus, code)

	if cdar.ExchangedDocument.ID != "" {
		pmt.Code = cbc.Code(cdar.ExchangedDocument.ID)
	}
	if cdar.ExchangedDocument.IssueDateTime != nil && cdar.ExchangedDocument.IssueDateTime.DateTimeString != nil {
		d, t, err := parseCDARDateTime(cdar.ExchangedDocument.IssueDateTime.DateTimeString.Value)
		if err != nil {
			return nil, err
		}
		pmt.IssueDate = d
		if t != nil {
			pmt.IssueTime = t
		}
	}

	// Same role-based mapping as status parse: SE → Supplier, BY →
	// Customer; platform parties are dropped.
	assignPaymentParty := func(tp *CDARTradeParty) {
		p := goblPartyFromCDAR(tp)
		if p == nil {
			return
		}
		switch p.Ext.Get(flow6.ExtKeyRole) {
		case flow6.RoleSeller:
			if pmt.Supplier == nil {
				pmt.Supplier = p
			}
		case flow6.RoleBuyer:
			if pmt.Customer == nil {
				pmt.Customer = p
			}
		}
	}
	assignPaymentParty(cdar.ExchangedDocument.IssuerTradeParty)
	for _, rp := range cdar.ExchangedDocument.RecipientTradeParties {
		assignPaymentParty(rp)
	}

	var total num.Amount
	for _, ackDoc := range cdar.AcknowledgementDocuments {
		if ackDoc == nil {
			continue
		}
		for _, ref := range ackDoc.ReferenceReferencedDocument {
			if ref == nil {
				continue
			}
			// Supplier fallback from the MDT-129 seller-SIREN slot.
			if pmt.Supplier == nil && ref.IssuerTradeParty != nil {
				pmt.Supplier = goblPartyFromCDAR(ref.IssuerTradeParty)
			}
			line := goblPaymentLineFromCDAR(pmt, ref)
			total = total.Add(line.Amount)
			pmt.Lines = append(pmt.Lines, line)
		}
	}

	if len(pmt.Methods) == 0 {
		pmt.Methods = []*pay.Record{{Key: pay.MeansKeyOther, Amount: total}}
	}

	return pmt, nil
}

// goblPaymentLineFromCDAR builds a payment line from a referenced
// document, reading the MEN / MPA amount into Amount and any RAP
// remainder into Due, and back-filling the payment's currency and
// condition extension from the characteristics encountered.
func goblPaymentLineFromCDAR(pmt *bill.Payment, ref *CDARReferencedDocument) *bill.PaymentLine {
	line := &bill.PaymentLine{}
	if dr := goblDocRefFromCDAR(ref); dr != nil {
		line.Document = dr
	}
	for _, ds := range ref.SpecifiedDocumentStatuses {
		if ds == nil {
			continue
		}
		for _, dc := range ds.SpecifiedDocumentCharacteristics {
			if dc == nil || dc.ValueAmount == nil || dc.ValueAmount.Value == "" {
				continue
			}
			amount, err := num.AmountFromString(dc.ValueAmount.Value)
			if err != nil {
				continue
			}
			if pmt.Currency == currency.CodeEmpty && dc.ValueAmount.CurrencyID != "" {
				pmt.Currency = currency.Code(dc.ValueAmount.CurrencyID)
			}
			switch cbc.Code(dc.TypeCode) {
			case flow6.ConditionAmountRemaining:
				due := amount
				line.Due = &due
			case flow6.ConditionAmountReceived, flow6.ConditionAmountPaid:
				line.Amount = amount
				pmt.Ext = pmt.Ext.Set(flow6.ExtKeyCondition, cbc.Code(dc.TypeCode))
			}
		}
	}
	return line
}
