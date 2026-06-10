package cii

import (
	"fmt"

	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// Date format codes for CDAR (UN/EDIFACT date format qualifier 2379).
const (
	cdarDateTimeFormat = "204" // CCYYMMDDHHMMSS
	cdarDateFormat     = "102" // CCYYMMDD
	schemeIDSIREN      = "0002"
	schemeIDPDP        = "0238" // Matricule PDP / PPF
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

// cdarAckTypeForCode maps a CDAR ProcessConditionCode to the ack
// TypeCode (MDT-77): transmission-phase codes (200/201/202/203/213) are
// platform-issued and ride TypeCode 305; treatment-phase codes
// (204-210 and the 211/212 payment events) are business-issued and
// ride TypeCode 23. The guideline (invoice vs einvoicingF2) is an
// independent axis that only encodes the destination — the official
// UC corpus carries 305 Déposée copies under the end-party "invoice"
// guideline and 23 Encaissée copies under the PPF guideline.
func cdarAckTypeForCode(code cbc.Code) string {
	switch code {
	case "200", "201", "202", "203", "213":
		return "305"
	default:
		return "23"
	}
}

// buyerIssuedProcessCodes lists the CDAR ProcessConditionCodes for
// lifecycle events declared by the invoice recipient: the Customer is
// the CDV issuer and the Supplier its business recipient. 209
// (Complétée) is seller-issued; the platform codes (305 phase) carry
// the sending platform (WK) in the issuer slot.
var buyerIssuedProcessCodes = map[cbc.Code]bool{
	"204": true, "205": true, "206": true, "207": true,
	"208": true, "210": true,
}

// NewCDARFromStatus is the exported entry point for converting a
// *bill.Status into a CDAR XML document using the provided context.
// Most callers should use Convert with a *gobl.Envelope wrapping the
// status; this helper is useful when the caller wants to bypass
// envelope-level validation (for example, in tests).
//
// The status must carry the flow6 extensions (fr-ctc-flow6-status on
// each line, fr-ctc-flow6-role on the parties) — run Calculate on the
// enclosing envelope with the fr-ctc-flow6-v1 addon declared so the
// normalizer derives them.
//
// SenderTradeParty is emitted as a bare <ram:RoleCode>WK</ram:RoleCode>.
// To carry an identified platform party in that slot, use
// NewCDARFromStatusWithSender or pass WithSenderTradeParty to Convert.
func NewCDARFromStatus(st *bill.Status, ctx Context) (*CDAR, error) {
	return newCDAR(st, ctx, nil, "", "")
}

// NewCDARFromStatusWithSender is the same as NewCDARFromStatus but
// carries the supplied *org.Party as the CDAR
// ExchangedDocument/SenderTradeParty (MDT-21) — typically the
// dematerialisation platform's identity (Name + GlobalID + Inbox).
func NewCDARFromStatusWithSender(st *bill.Status, ctx Context, sender *org.Party) (*CDAR, error) {
	return newCDAR(st, ctx, sender, "", "")
}

// partyWithEndpointURI returns the first party that carries the given
// URI among its endpoints, or nil. Used to resolve the envelope's
// Head.From / Head.To routing addresses back to the document's
// business-party slots.
func partyWithEndpointURI(uri cbc.URI, parties ...*org.Party) *org.Party {
	if uri == "" {
		return nil
	}
	for _, p := range parties {
		if p == nil {
			continue
		}
		for _, e := range p.Endpoints {
			if e != nil && e.URI == uri {
				return p
			}
		}
	}
	return nil
}

// newCDAR converts a *bill.Status into a CDAR XML document.
//
// Two independent axes drive the wire layout:
//
//   - The ack TypeCode (MDT-77) follows the ProcessConditionCode's
//     phase — see cdarAckTypeForCode — together with the issuer slot
//     (business party for 23, platform for 305).
//   - The caller-supplied Context picks the destination: ContextCDARFlow6
//     (end-party copy, "invoice" guideline + REGULATED) or
//     ContextCDARFlow6PPF ("einvoicingF2" guideline, single PPF
//     recipient).
//
// from / to carry the envelope's Head.From / Head.To routing URIs: on
// business-issued (23-phase) codes they pick the issuer / recipient
// party when they resolve to one of the status's business parties,
// overriding the per-code defaults. Platform-issued (305-phase) codes
// keep the sending platform in the issuer slot regardless.
func newCDAR(st *bill.Status, ctx Context, sender *org.Party, from, to cbc.URI) (*CDAR, error) {
	if st == nil {
		return nil, fmt.Errorf("nil bill.Status")
	}

	guideline := ctx.GuidelineID
	if guideline == "" {
		guideline = ContextCDARFlow6.GuidelineID
	}
	code := firstLineProcessCode(st)
	ackType := cdarAckTypeForCode(code)

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

	// Party slots are derived from the status semantics — bill.Status
	// only carries the business parties (Supplier / Customer); the
	// platform (WK) and PPF (DFH) parties are transport-level and
	// injected here:
	//
	//   SenderTradeParty (MDT-21): bare <RoleCode>WK</RoleCode> by
	//     default; overridden by the WithSenderTradeParty option.
	//   IssuerTradeParty (MDG-16): the business party declaring the
	//     event on 23-phase codes (Customer for 204-210, Supplier for
	//     209); the sending platform on 305-phase codes.
	//   RecipientTradeParty (MDG-23): the single PPF party on the
	//     einvoicingF2 guideline; otherwise the issuer's counterparty.
	if sender == nil {
		sender = bareWKParty()
	}
	cdar.ExchangedDocument.SenderTradeParty = newCDARTradeParty(sender)

	issuer, recipient := sender, st.Supplier
	switch {
	case buyerIssuedProcessCodes[code]:
		issuer, recipient = st.Customer, st.Supplier
	case code == "209":
		issuer, recipient = st.Supplier, st.Customer
	}
	if ackType == "23" {
		if p := partyWithEndpointURI(from, st.Supplier, st.Customer); p != nil {
			issuer = p
		}
		if p := partyWithEndpointURI(to, st.Supplier, st.Customer); p != nil {
			recipient = p
		}
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

// firstLineProcessCode returns the flow6 ProcessConditionCode of the
// status's first line (flow6 validation enforces exactly one line).
func firstLineProcessCode(st *bill.Status) cbc.Code {
	for _, line := range st.Lines {
		if line == nil {
			continue
		}
		return line.Ext.Get(flow6.ExtKeyStatus)
	}
	return ""
}

func newCDARAcknowledgement(st *bill.Status, line *bill.StatusLine, ackType string) (*CDARAcknowledgement, error) {
	processCode := line.Ext.Get(flow6.ExtKeyStatus)
	if processCode == "" {
		return nil, fmt.Errorf("status line %q missing %s extension; run Calculate with the %s addon declared", line.Key, flow6.ExtKeyStatus, flow6.V1)
	}

	ack := &CDARAcknowledgement{
		MultipleReferencesIndicator: &CDARIndicator{Value: false},
		TypeCode:                    ackType,
		IssueDateTime:               makeIssueDateTime(st.IssueDate, st.IssueTime),
	}

	ref := &CDARReferencedDocument{
		ProcessConditionCode: processCode.String(),
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
	switch {
	case len(line.Reasons) > 0 && len(line.Actions) > 0:
		for _, reason := range line.Reasons {
			for _, action := range line.Actions {
				seq++
				statuses = append(statuses, newCDARDocumentStatus(reason, action, seq))
			}
		}
	case len(line.Reasons) > 0:
		for _, reason := range line.Reasons {
			seq++
			statuses = append(statuses, newCDARDocumentStatus(reason, nil, seq))
		}
	case len(line.Actions) > 0:
		for _, action := range line.Actions {
			seq++
			statuses = append(statuses, newCDARDocumentStatus(nil, action, seq))
		}
	}

	ref.SpecifiedDocumentStatuses = statuses
	ack.ReferenceReferencedDocument = []*CDARReferencedDocument{ref}
	return ack, nil
}

// statusCharacteristicTypeCodes is the MDT-207 vocabulary admissible on
// a status-side SpecifiedDocumentCharacteristic. A fault whose code is
// in this set emits it as the characteristic's TypeCode; other fault
// codes (business rules, BT identifiers…) ride in the Name only, so
// the generated CDV stays within the controlled list.
var statusCharacteristicTypeCodes = map[cbc.Code]bool{
	flow6.ConditionBankDetailsUpdate:  true, // CBB
	flow6.ConditionInvalidData:        true, // DIV
	flow6.ConditionExpectedData:       true, // DVA
	flow6.ConditionReplacementData:    true, // MAJ
	flow6.ConditionAmountApprovedHT:   true, // MAP
	flow6.ConditionAmountApprovedTTC:  true, // MAPTTC
	flow6.ConditionAmountRejectedHT:   true, // MNA
	flow6.ConditionAmountRejectedTTC:  true, // MNATTC
	flow6.ConditionDiscount:           true, // ESC
	flow6.ConditionRebate:             true, // RAB
	flow6.ConditionReduction:          true, // REM
}

// newCDARDocumentStatus maps a (Reason, Action) pair onto a CDAR
// SpecifiedDocumentStatus. The CDAR codes are read straight from the
// flow6 extensions — normalizeReason / normalizeAction guarantee they
// are populated whenever the Key is set, so no fallback tables are
// needed here. Reason faults emit as SpecifiedDocumentCharacteristics
// (field-level corrections): the fault code becomes the TypeCode when
// it belongs to the MDT-207 vocabulary, the message the data Name and
// the first path the XML Location.
func newCDARDocumentStatus(reason *bill.Reason, action *bill.Action, seq int) *CDARDocumentStatus {
	ds := &CDARDocumentStatus{SequenceNumeric: seq}
	if reason != nil {
		ds.ReasonCode = reason.Ext.Get(flow6.ExtKeyReason).String()
		if reason.Description != "" {
			ds.Reason = []string{reason.Description}
		}
		for _, f := range reason.Faults {
			if f == nil {
				continue
			}
			dc := &CDARDocumentCharacteristic{Name: f.Message}
			if statusCharacteristicTypeCodes[f.Code] {
				dc.TypeCode = f.Code.String()
			} else if dc.Name == "" {
				dc.Name = f.Code.String()
			}
			if len(f.Paths) > 0 {
				dc.Location = f.Paths[0]
			}
			ds.SpecifiedDocumentCharacteristics = append(ds.SpecifiedDocumentCharacteristics, dc)
		}
	}
	if action != nil {
		ds.RequestedActionCode = action.Ext.Get(flow6.ExtKeyAction).String()
		if action.Description != "" {
			ds.RequestedAction = action.Description
		}
	}
	return ds
}

// bareWKParty returns a minimal *org.Party tagged with the WK platform role
// — used as the SenderTradeParty (always) and as the IssuerTradeParty
// fallback for platform-issued codes when no platform identity is supplied.
// Matches the UC1 corpus shape of <ram:RoleCode>WK</ram:RoleCode> with no
// body.
func bareWKParty() *org.Party {
	return &org.Party{
		Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RolePlatform),
	}
}

// ppfTradeParty returns the constant CDAR trade party for the Portail
// Public de Facturation — GlobalID 9998 under the 0238 (matricule PDP)
// scheme with the DFH role, per BR-FR-CDV-02 — the single recipient of
// einvoicingF2 transmissions.
func ppfTradeParty() *CDARTradeParty {
	return &CDARTradeParty{
		GlobalIDs: []*CDARGlobalID{{SchemeID: schemeIDPDP, Value: "9998"}},
		RoleCode:  flow6.RolePPF.String(),
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
