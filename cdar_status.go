package cii

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/addons/fr/ctc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
)

// Date format codes for CDAR (UN/EDIFACT date format qualifier 2379).
const (
	cdarDateTimeFormat = "204" // CCYYMMDDHHMMSS
	cdarDateFormat     = "102" // CCYYMMDD
	schemeIDSIREN      = "0002"
)

// CDAR GuidelineID URNs per BR-FR-CDV-02 (MDT-3). Used as the stable
// identifier that distinguishes one CDAR Context from another — the
// rest of the converter looks up the ack TypeCode from this URN.
const (
	// CDARGuidelineInvoice is the GuidelineID for end-party CDARs
	// (treatment phase). Pairs with ack TypeCode 23.
	CDARGuidelineInvoice = "urn.cpro.gouv.fr:1p0:CDV:invoice"
	// CDARGuidelinePPF is the GuidelineID for CDARs transmitted to the
	// PPF (transmission phase). Pairs with ack TypeCode 305.
	CDARGuidelinePPF = "urn.cpro.gouv.fr:1p0:CDV:einvoicingF2"
)

// cdarAckTypeByGuideline maps a CDAR GuidelineID to the wire TypeCode
// emitted on the AcknowledgementDocument. Lookups against an unknown
// guideline return "", which the generator treats as "fall back to the
// Status.Type derivation" for older callers.
var cdarAckTypeByGuideline = map[string]string{
	CDARGuidelineInvoice: "23",
	CDARGuidelinePPF:     "305",
}

// NewCDARFromStatus is the exported entry point for converting a
// *bill.Status into a CDAR XML document using the provided context.
// Most callers should use Convert with a *gobl.Envelope wrapping the
// status; this helper is useful when the caller wants to bypass
// envelope-level validation (for example, in tests).
//
// SenderTradeParty is emitted as a bare <ram:RoleCode>WK</ram:RoleCode>.
// To carry an identified platform party in that slot, use
// NewCDARFromStatusWithSender or pass WithSenderTradeParty to Convert.
func NewCDARFromStatus(st *bill.Status, ctx Context) (*CDAR, error) {
	return newCDAR(st, ctx, nil)
}

// NewCDARFromStatusWithSender is the same as NewCDARFromStatus but
// carries the supplied *org.Party as the CDAR
// ExchangedDocument/SenderTradeParty (MDT-21) — typically the
// dematerialisation platform's identity (Name + GlobalID + Inbox).
func NewCDARFromStatusWithSender(st *bill.Status, ctx Context, sender *org.Party) (*CDAR, error) {
	return newCDAR(st, ctx, sender)
}

// ParseCDARStatus parses a raw CDAR XML byte slice into a *bill.Status
// without going through an envelope.
func ParseCDARStatus(data []byte) (*bill.Status, error) {
	return parseStatus(data)
}

// newCDAR converts a *bill.Status into a CDAR XML document.
//
// The CDAR wire layout (ack TypeCode 23 vs 305 and the corresponding
// trade-party shape) is driven entirely by the caller-supplied Context —
// not by bill.Status.Type. The GOBL Type field is a lifecycle category
// (response / update / system) and is used only by flow6 to disambiguate
// the ProcessConditionCode for a given StatusLine.Key. The CDAR phase
// (treatment vs transmission) is a separate axis chosen at conversion
// time via ContextCDARFlow6 (treatment, 23) or ContextCDARFlow6PPF
// (transmission, 305).
func newCDAR(st *bill.Status, ctx Context, sender *org.Party) (*CDAR, error) {
	if st == nil {
		return nil, fmt.Errorf("nil bill.Status")
	}

	guideline := ctx.GuidelineID
	if guideline == "" {
		guideline = ContextCDARFlow6.GuidelineID
	}
	// Resolve the ack TypeCode from the Context's GuidelineID. This is the
	// single discriminator for "what shape of CDAR are we producing".
	ackType := cdarAckTypeByGuideline[guideline]
	if ackType == "" {
		ackType = "23"
	}

	cdar := NewCDAR()
	cdar.ExchangedDocumentContext = &CDARExchangedContext{
		GuidelineParameter: &CDARDocumentContextParameter{ID: guideline},
	}
	// BusinessProcessParameter is only present on the end-party
	// "invoice" guideline (REGULATED); PPF transmissions omit it.
	if ctx.BusinessID != "" {
		cdar.ExchangedDocumentContext.BusinessProcessParameter = &CDARDocumentContextParameter{ID: ctx.BusinessID}
	}

	cdar.ExchangedDocument = &CDARExchangedDocument{
		ID:            string(st.Code),
		IssueDateTime: makeIssueDateTime(st.IssueDate, st.IssueTime),
	}
	if st.Series != "" && cdar.ExchangedDocument.ID == "" {
		cdar.ExchangedDocument.ID = string(st.Series)
	}
	// Direct mapping — flow6 validations guarantee st.Issuer and
	// st.Recipient are present (MDG-16, MDG-23).
	//
	//   SenderTradeParty (MDT-21):  bare <RoleCode>WK</RoleCode> by
	//     default; overridden by the WithSenderTradeParty option.
	//   IssuerTradeParty (MDG-16):  st.Issuer.
	//   RecipientTradeParty (MDG-23): st.Recipient.
	if sender == nil {
		sender = bareWKParty()
	}
	cdar.ExchangedDocument.SenderTradeParty = newCDARTradeParty(sender)
	if st.Issuer != nil {
		cdar.ExchangedDocument.IssuerTradeParty = newCDARTradeParty(st.Issuer)
	}
	if st.Recipient != nil {
		cdar.ExchangedDocument.RecipientTradeParties = []*CDARTradeParty{
			newCDARTradeParty(st.Recipient),
		}
	}

	for _, line := range st.Lines {
		if line == nil {
			continue
		}
		ack, err := newCDARAcknowledgement(st, line, ackType)
		if err != nil {
			return nil, err
		}
		cdar.AcknowledgementDocuments = append(cdar.AcknowledgementDocuments, ack)
	}

	return cdar, nil
}

func newCDARAcknowledgement(st *bill.Status, line *bill.StatusLine, ackType string) (*CDARAcknowledgement, error) {
	processCode, ok := ctc.CDARProcessCodeFor(line.Key, st.Type)
	if !ok {
		return nil, fmt.Errorf("no CDAR process code for status line key %q with type %q", line.Key, st.Type)
	}

	ack := &CDARAcknowledgement{
		MultipleReferencesIndicator: &CDARIndicator{Value: false},
		TypeCode:                    ackType,
		IssueDateTime:               makeIssueDateTime(st.IssueDate, st.IssueTime),
	}

	ref := &CDARReferencedDocument{
		ProcessConditionCode: processCode,
	}
	if line.Doc != nil {
		ref.IssuerAssignedID = string(line.Doc.Code)
		if line.Doc.IssueDate != nil {
			ref.FormattedIssueDateTime = &CDARFormattedIssueDateTime{
				DateTimeString: &CDARQDTDateTimeString{
					Value:  formatCDARDate(*line.Doc.IssueDate),
					Format: cdarDateFormat,
				},
			}
		}
		if line.Doc.Type != "" {
			ref.TypeCode = string(line.Doc.Type)
		}
	}
	if st.Supplier != nil {
		if siren := partyIdentityCode(st.Supplier, schemeIDSIREN); siren != "" {
			ref.IssuerTradeParty = &CDARTradeParty{
				GlobalIDs: []*CDARGlobalID{{SchemeID: schemeIDSIREN, Value: siren}},
			}
		}
	}

	// Build the SpecifiedDocumentStatus list. Pair each Reason with each
	// Action when both are present, or emit one per Reason / Action alone.
	var statuses []*CDARDocumentStatus
	seq := 0
	if len(line.Reasons) == 0 && len(line.Actions) == 0 {
		// no extra status info
	} else if len(line.Reasons) > 0 && len(line.Actions) > 0 {
		for _, reason := range line.Reasons {
			for _, action := range line.Actions {
				seq++
				ds, err := newCDARDocumentStatus(reason, action, seq)
				if err != nil {
					return nil, err
				}
				statuses = append(statuses, ds)
			}
		}
	} else if len(line.Reasons) > 0 {
		for _, reason := range line.Reasons {
			seq++
			ds, err := newCDARDocumentStatus(reason, nil, seq)
			if err != nil {
				return nil, err
			}
			statuses = append(statuses, ds)
		}
	} else {
		for _, action := range line.Actions {
			seq++
			ds, err := newCDARDocumentStatus(nil, action, seq)
			if err != nil {
				return nil, err
			}
			statuses = append(statuses, ds)
		}
	}

	// Append characteristics derived from StatusLine.Complements (e.g. the
	// MEN amount on a paid line) onto the first / new status if no reason.
	cChars := characteristicsFromComplements(line.Complements, "")
	if len(cChars) > 0 {
		if len(statuses) == 0 {
			seq++
			statuses = append(statuses, &CDARDocumentStatus{
				SequenceNumeric:                  seq,
				SpecifiedDocumentCharacteristics: cChars,
			})
		} else {
			statuses[0].SpecifiedDocumentCharacteristics = append(statuses[0].SpecifiedDocumentCharacteristics, cChars...)
		}
	}

	ref.SpecifiedDocumentStatuses = statuses
	ack.ReferenceReferencedDocument = []*CDARReferencedDocument{ref}
	return ack, nil
}

func newCDARDocumentStatus(reason *bill.Reason, action *bill.Action, seq int) (*CDARDocumentStatus, error) {
	ds := &CDARDocumentStatus{SequenceNumeric: seq}
	if reason != nil {
		// ReasonCode: ext value if set, else default-for-key.
		code := reason.Ext.Get(ctc.ExtKeyReasonCode).String()
		if code == "" {
			if def, ok := ctc.CDARReasonCodeFor(reason.Key); ok {
				code = def
			}
		}
		ds.ReasonCode = code
		if reason.Description != "" {
			ds.Reason = []string{reason.Description}
		}
		// Append per-reason characteristics
		// (caller may attach Complements at the StatusLine level; here we
		// only emit reason-level if any characteristics are tagged with
		// the reason's ext code.)
	}
	if action != nil {
		if code, ok := ctc.CDARActionCodeFor(action.Key); ok {
			ds.RequestedActionCode = code
		}
		if action.Description != "" {
			ds.RequestedAction = action.Description
		}
	}
	return ds, nil
}

// characteristicsFromComplements pulls ctc.Characteristic instances out of
// a Complements slice and builds CDAR characteristics. If reasonCode is
// non-empty, only characteristics matching that ReasonCode (or with an
// empty ReasonCode) are returned; if empty, all are returned.
func characteristicsFromComplements(comps []*schema.Object, reasonCode cbc.Code) []*CDARDocumentCharacteristic {
	var out []*CDARDocumentCharacteristic
	for _, obj := range comps {
		if obj == nil {
			continue
		}
		c, ok := obj.Instance().(*ctc.Characteristic)
		if !ok || c == nil {
			continue
		}
		if reasonCode != "" && c.ReasonCode != "" && c.ReasonCode != reasonCode {
			continue
		}
		dc := &CDARDocumentCharacteristic{
			ID:       c.ID,
			TypeCode: string(c.TypeCode),
			Name:     c.Name,
			Location: c.Location,
		}
		if c.Changed != nil {
			dc.ValueChangedIndicator = &CDARIndicatorString{
				Value: strconv.FormatBool(*c.Changed),
			}
		}
		if c.Percent != nil {
			dc.ValuePercent = c.Percent.StringWithoutSymbol()
		}
		if c.Amount != nil {
			dc.ValueAmount = &CDARValueAmount{
				Value:      c.Amount.Value.String(),
				CurrencyID: c.Amount.Currency.String(),
			}
		}
		out = append(out, dc)
	}
	return out
}

// bareWKParty returns a minimal *org.Party tagged with the WK platform role
// — used as the SenderTradeParty (always) and as the IssuerTradeParty
// fallback for ack 305 when no platform identity is supplied. Matches the
// UC1 corpus shape of <ram:RoleCode>WK</ram:RoleCode> with no body.
func bareWKParty() *org.Party {
	return &org.Party{
		Ext: tax.MakeExtensions().Set(ctc.ExtKeyRole, ctc.RoleWK),
	}
}

func newCDARTradeParty(p *org.Party) *CDARTradeParty {
	if p == nil {
		return nil
	}
	tp := &CDARTradeParty{
		Name: p.Name,
	}
	if !p.Ext.IsZero() {
		tp.RoleCode = p.Ext.Get(ctc.ExtKeyRole).String()
	}
	for _, id := range p.Identities {
		if id == nil || id.Ext.IsZero() {
			continue
		}
		scheme := id.Ext.Get(iso.ExtKeySchemeID).String()
		if scheme == "" {
			continue
		}
		tp.GlobalIDs = append(tp.GlobalIDs, &CDARGlobalID{
			SchemeID: scheme,
			Value:    id.Code.String(),
		})
	}
	if len(p.Inboxes) > 0 {
		ib := p.Inboxes[0]
		if ib.Code != "" {
			tp.URIUniversalCommunication = &CDARUniversalCommunication{
				URIID: &CDARURIID{
					SchemeID: ib.Scheme.String(),
					Value:    ib.Code.String(),
				},
			}
		}
	}
	return tp
}

func partyIdentityCode(p *org.Party, scheme string) string {
	if p == nil {
		return ""
	}
	for _, id := range p.Identities {
		if id == nil || id.Ext.IsZero() {
			continue
		}
		if id.Ext.Get(iso.ExtKeySchemeID).String() == scheme {
			return id.Code.String()
		}
	}
	return ""
}

func makeIssueDateTime(d cal.Date, t *cal.Time) *CDARIssueDateTime {
	value := formatCDARDate(d)
	if t != nil && !t.IsZero() {
		value = fmt.Sprintf("%s%02d%02d%02d", value, t.Hour, t.Minute, t.Second)
		return &CDARIssueDateTime{DateTimeString: &CDARDateTimeString{Value: value, Format: cdarDateTimeFormat}}
	}
	// No time supplied — emit at midnight using full datetime format.
	value = fmt.Sprintf("%s000000", value)
	return &CDARIssueDateTime{DateTimeString: &CDARDateTimeString{Value: value, Format: cdarDateTimeFormat}}
}

func formatCDARDate(d cal.Date) string {
	return fmt.Sprintf("%04d%02d%02d", d.Year, d.Month, d.Day)
}
