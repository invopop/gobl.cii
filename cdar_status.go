package cii

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/addons/fr/ctc/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/schema"
)

// Date format codes for CDAR (UN/EDIFACT date format qualifier 2379).
const (
	cdarDateTimeFormat = "204" // CCYYMMDDHHMMSS
	cdarDateFormat     = "102" // CCYYMMDD
	schemeIDSIREN      = "0002"
)

// Acknowledgement TypeCode mapping. CDAR uses 305 for "update" (issuer-side
// status messages such as "deposed" or "payment-forwarded") and 23 for
// "response" messages issued by the recipient or platform.
const (
	cdarTypeCodeUpdate   = "305"
	cdarTypeCodeResponse = "23"
)

// NewCDARFromStatus is the exported entry point for converting a
// *bill.Status into a CDAR XML document using the provided context.
// Most callers should use Convert with a *gobl.Envelope wrapping the
// status; this helper is useful when the caller wants to bypass
// envelope-level validation (for example, in tests).
func NewCDARFromStatus(st *bill.Status, ctx Context) (*CDAR, error) {
	return newCDAR(st, ctx)
}

// ParseCDARStatus parses a raw CDAR XML byte slice into a *bill.Status
// without going through an envelope.
func ParseCDARStatus(data []byte) (*bill.Status, error) {
	return parseStatus(data)
}

// newCDAR converts a *bill.Status into a CDAR XML document.
func newCDAR(st *bill.Status, ctx Context) (*CDAR, error) {
	if st == nil {
		return nil, fmt.Errorf("nil bill.Status")
	}

	guideline := ctx.GuidelineID
	if guideline == "" {
		guideline = ContextCDARFlow6.GuidelineID
	}
	business := ctx.BusinessID
	if business == "" {
		business = "REGULATED"
	}

	cdar := NewCDAR()
	cdar.ExchangedDocumentContext = &CDARExchangedContext{
		BusinessProcessParameter: &CDARDocumentContextParameter{ID: business},
		GuidelineParameter:       &CDARDocumentContextParameter{ID: guideline},
	}

	cdar.ExchangedDocument = &CDARExchangedDocument{
		ID:            string(st.Code),
		IssueDateTime: makeIssueDateTime(st.IssueDate, st.IssueTime),
	}
	if st.Series != "" && cdar.ExchangedDocument.ID == "" {
		cdar.ExchangedDocument.ID = string(st.Series)
	}
	// In a Flow 6 B2B status, both SenderTradeParty (MDT-21) and
	// IssuerTradeParty (MDT-40) of the ExchangedDocument represent the
	// platform (PA) producing the CDAR. The caller surfaces that party as
	// bill.Status.Issuer, with Ext[fr-ctc-role] = "WK". For "response"
	// status types where no explicit Issuer is set, we fall back to the
	// Customer (the buyer-side platform / recipient), and for "update"
	// types we fall back to the Supplier — those fallbacks lose the WK
	// role, which is why the platform should always populate Issuer.
	sender := st.Issuer
	if sender == nil {
		switch st.Type {
		case bill.StatusTypeUpdate:
			sender = st.Supplier
		case bill.StatusTypeResponse:
			sender = st.Customer
		}
	}
	if sender != nil {
		tp := newCDARTradeParty(sender)
		cdar.ExchangedDocument.SenderTradeParty = tp
		cdar.ExchangedDocument.IssuerTradeParty = tp
	}

	var recipients []*CDARTradeParty
	if st.Recipient != nil {
		recipients = append(recipients, newCDARTradeParty(st.Recipient))
	}
	if st.Type == bill.StatusTypeUpdate && st.Customer != nil {
		recipients = append(recipients, newCDARTradeParty(st.Customer))
	} else if st.Type == bill.StatusTypeResponse && st.Supplier != nil {
		recipients = append(recipients, newCDARTradeParty(st.Supplier))
	}
	cdar.ExchangedDocument.RecipientTradeParties = recipients

	// Acknowledgement TypeCode comes from the Context — the caller picks
	// ContextCDARFlow6Response (23) or ContextCDARFlow6Update (305). When
	// the context did not specify, fall back to deriving it from the
	// status Type so older callers keep working.
	ackType := ctx.CDARAckTypeCode
	if ackType == "" {
		if st.Type == bill.StatusTypeUpdate {
			ackType = cdarTypeCodeUpdate
		} else {
			ackType = cdarTypeCodeResponse
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
	processCode, ok := flow6.CDARProcessCodeFor(line.Key, st.Type)
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
		code := reason.Ext.Get(flow6.ExtKeyReasonCode).String()
		if code == "" {
			if def, ok := flow6.CDARReasonCodeFor(reason.Key); ok {
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
		if code, ok := flow6.CDARActionCodeFor(action.Key); ok {
			ds.RequestedActionCode = code
		}
		if action.Description != "" {
			ds.RequestedAction = action.Description
		}
	}
	return ds, nil
}

// characteristicsFromComplements pulls flow6.Characteristic instances out of
// a Complements slice and builds CDAR characteristics. If reasonCode is
// non-empty, only characteristics matching that ReasonCode (or with an
// empty ReasonCode) are returned; if empty, all are returned.
func characteristicsFromComplements(comps []*schema.Object, reasonCode cbc.Code) []*CDARDocumentCharacteristic {
	var out []*CDARDocumentCharacteristic
	for _, obj := range comps {
		if obj == nil {
			continue
		}
		c, ok := obj.Instance().(*flow6.Characteristic)
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

func newCDARTradeParty(p *org.Party) *CDARTradeParty {
	if p == nil {
		return nil
	}
	tp := &CDARTradeParty{
		Name: p.Name,
	}
	if !p.Ext.IsZero() {
		tp.RoleCode = p.Ext.Get(flow6.ExtKeyRole).String()
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
