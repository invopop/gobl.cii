package cii

import (
	"fmt"

	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
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
	return newCDARFromPayment(pmt, ctx, nil, "", "")
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
	return goblPaymentFromCDAR(cdar, routing{})
}

// NewCDARFromPaymentWithSender is the same as NewCDARFromPayment but
// carries the supplied *org.Party as the CDAR
// ExchangedDocument/SenderTradeParty (MDT-21) — typically the
// dematerialisation platform's identity.
func NewCDARFromPaymentWithSender(pmt *bill.Payment, ctx Context, sender *org.Party) (*CDAR, error) {
	return newCDARFromPayment(pmt, ctx, sender, "", "")
}

// newCDARFromPayment converts a *bill.Payment into a CDAR. from / to
// carry the envelope's Head.From / Head.To routing URIs and, when they
// resolve to one of the payment's business parties, pick the issuer /
// recipient over the Payment.Type defaults.
func newCDARFromPayment(pmt *bill.Payment, ctx Context, sender *org.Party, from, to cbc.URI) (*CDAR, error) {
	if pmt == nil {
		return nil, fmt.Errorf("nil bill.Payment")
	}

	processCode := pmt.Ext.Get(flow6.ExtKeyStatus)
	if processCode == "" {
		return nil, fmt.Errorf("payment missing %s extension; run Calculate with the %s addon declared", flow6.ExtKeyStatus, flow6.V1)
	}

	guideline := cdarXMLGuideline(ctx)
	// 211 / 212 are treatment-phase, business-issued events.
	ackType := cdarAckTypeForCode(processCode)

	cdar := NewCDAR()
	cdar.ExchangedDocumentContext = &CDARExchangedContext{
		GuidelineParameter: &CDARDocumentContextParameter{ID: guideline},
	}
	if bp := cdarBusinessProcessID(ctx); bp != "" {
		cdar.ExchangedDocumentContext.BusinessProcessParameter = &CDARDocumentContextParameter{ID: bp}
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
	if p := partyWithEndpointURI(from, pmt.Supplier, pmt.Customer); p != nil {
		issuer = p
	}
	if p := partyWithEndpointURI(to, pmt.Supplier, pmt.Customer); p != nil {
		recipient = p
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

	if guideline == CDARGuidelinePPF {
		markPPFReferences(cdar)
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
		ProcessCondition:     cdarProcessConditions[processCode],
		StatusCode:           cdarRefStatusCodes[processCode],
	}
	if line.Document != nil {
		// IssuerAssignedID carries the referenced invoice's full number
		// (series + code) so the receiver can resolve it against its invoice
		// directory keyed on Series.Join(Code) — see cdar_status.go.
		ref.IssuerAssignedID = line.Document.Series.Join(line.Document.Code).String()
		if line.Document.IssueDate != nil {
			ref.FormattedIssueDateTime = &CDARFormattedIssueDateTime{
				DateTimeString: &CDARQDTDateTimeString{
					Value:  formatCDARDate(*line.Document.IssueDate),
					Format: cdarDateFormat,
				},
			}
		}
		// MDT-91: the referenced invoice's document type code, from the
		// canonical untdid-document-type extension. The flow6 addon migrates a
		// legacy Type key into this extension on normalize and requires it, so
		// a calculated document always carries it — no Type fallback needed.
		if dt := line.Document.Ext.Get(untdid.ExtKeyDocumentType); dt != "" {
			ref.TypeCode = dt.String()
		}
	}
	// MDT-129: the referenced invoice's issuer (its supplier).
	ref.IssuerTradeParty = cdarReferencedIssuer(line.Document, pmt.Supplier)

	// Amounts ride as document characteristics on a single status
	// entry: the payment's condition extension types the line amount
	// (MEN for receipts, MPA for advices) and any Due remainder emits
	// an additional RAP characteristic for partial payments.
	ds := &CDARDocumentStatus{SequenceNumeric: 1}
	condition := pmt.Ext.Get(flow6.ExtKeyCondition)
	if condition == "" {
		condition = flow6.ConditionAmountReceived
	}
	// PPF rejects an encaissement whose cashed amount (MDT-215) is not
	// split by VAT rate (MDT-224) — P1.18 / G7.45, motif REJ_ENCAISSEMENT.
	// When the payment line carries a tax breakdown, emit one
	// characteristic per distinct rate (gross amount + percentage);
	// otherwise fall back to a single amount-only characteristic.
	if split := newCDARVATSplitCharacteristics(condition, line, pmt.Currency); len(split) > 0 {
		ds.SpecifiedDocumentCharacteristics = split
	} else {
		ds.SpecifiedDocumentCharacteristics = append(ds.SpecifiedDocumentCharacteristics,
			newCDARAmountCharacteristic(condition, line.Amount, pmt.Currency))
	}
	if line.Due != nil && condition != flow6.ConditionAmountRemaining {
		ds.SpecifiedDocumentCharacteristics = append(ds.SpecifiedDocumentCharacteristics,
			newCDARAmountCharacteristic(flow6.ConditionAmountRemaining, *line.Due, pmt.Currency))
	}
	ref.SpecifiedDocumentStatuses = []*CDARDocumentStatus{ds}

	ack.ReferenceReferencedDocument = []*CDARReferencedDocument{ref}
	return ack
}

// newCDARVATSplitCharacteristics breaks a payment line's cashed amount
// down by VAT rate into one MDT-215 (ValueAmount, gross per rate) /
// MDT-224 (ValuePercent) characteristic per distinct rate, as PPF
// requires for an encaissement. Returns nil when the line carries no
// tax breakdown so the caller can keep the amount-only form.
func newCDARVATSplitCharacteristics(typeCode cbc.Code, line *bill.PaymentLine, cur currency.Code) []*CDARDocumentCharacteristic {
	if line == nil || line.Tax == nil {
		return nil
	}
	var chars []*CDARDocumentCharacteristic
	for _, cat := range line.Tax.Categories {
		if cat == nil {
			continue
		}
		for _, rt := range cat.Rates {
			if rt == nil || rt.Percent == nil {
				continue
			}
			gross := rt.Base.Add(rt.Amount)
			chars = append(chars, &CDARDocumentCharacteristic{
				TypeCode:     typeCode.String(),
				ValueAmount:  &CDARValueAmount{Value: gross.String(), CurrencyID: cur.String()},
				ValuePercent: rt.Percent.Amount().Rescale(2).String(),
			})
		}
	}
	return chars
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
func goblPaymentFromCDAR(cdar *CDAR, r routing) (*bill.Payment, error) {
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

	hydratePartyInboxes(pmt.Supplier, pmt.Customer, r)
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
	var vatRates []*tax.RateTotal
	var vatSum *num.Amount
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
				// The cashed amount (MDT-215). This assumes a single
				// cashed-amount characteristic; a 212 whose amount is split
				// across several VAT rates repeats this characteristic per
				// rate, in which case only the last rate's amount is kept here
				// (the per-rate VAT breakdown below is still captured in full).
				line.Amount = amount
				pmt.Ext = pmt.Ext.Set(flow6.ExtKeyCondition, cbc.Code(dc.TypeCode))
			}
			// MDT-224: a percentage on the characteristic means the cashed
			// amount is split by VAT rate (BR-FR-CDV-14, 212 Encaissée). The
			// amount is the gross (TTC) per rate; recover the base and VAT so
			// the parsed payment carries the tax breakdown the Flow 6 rules
			// require and the CDV round-trips.
			if dc.ValuePercent != "" {
				pct, err := num.PercentageFromString(dc.ValuePercent + "%")
				if err != nil {
					continue
				}
				vat := pct.From(amount)
				vatRates = append(vatRates, &tax.RateTotal{
					Base:    amount.Divide(pct.Factor()),
					Percent: &pct,
					Amount:  vat,
				})
				if vatSum == nil {
					sum := vat
					vatSum = &sum
				} else {
					sum := vatSum.Add(vat)
					vatSum = &sum
				}
			}
		}
	}
	if len(vatRates) > 0 && vatSum != nil {
		line.Tax = &tax.Total{
			Categories: []*tax.CategoryTotal{{
				Code:   tax.CategoryVAT,
				Rates:  vatRates,
				Amount: *vatSum,
			}},
			Sum: *vatSum,
		}
	}
	return line
}
