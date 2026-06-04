package cii

import (
	"fmt"
	"strconv"
	"time"

	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// ParseCDARStatus parses a raw CDAR XML byte slice into a *bill.Status
// without going through an envelope.
//
// The returned status carries the CDAR codes on the flow6 extensions
// (fr-ctc-flow6-status / -reason / -action) with the addon declared;
// the GOBL-level fields they derive (Status.Type, line and reason keys)
// are filled by the flow6 normalizer when the document is calculated —
// wrap it in an envelope (gobl.Envelop / Envelope.Insert) or run
// Calculate to complete it.
func ParseCDARStatus(data []byte) (*bill.Status, error) {
	cdar, err := UnmarshalCDAR(data)
	if err != nil {
		return nil, err
	}
	return goblStatusFromCDAR(cdar)
}

// parseCDAR converts a raw CDAR XML byte slice into the GOBL document
// matching its ProcessConditionCode: payment lifecycle codes (211 /
// 212) produce a *bill.Payment, everything else a *bill.Status.
func parseCDAR(data []byte) (any, error) {
	cdar, err := UnmarshalCDAR(data)
	if err != nil {
		return nil, err
	}
	if code := cdarProcessCode(cdar); code == "211" || code == "212" {
		return goblPaymentFromCDAR(cdar)
	}
	return goblStatusFromCDAR(cdar)
}

// cdarProcessCode returns the ProcessConditionCode of the first
// referenced document in the CDAR, or "".
func cdarProcessCode(cdar *CDAR) string {
	for _, ack := range cdar.AcknowledgementDocuments {
		if ack == nil {
			continue
		}
		for _, ref := range ack.ReferenceReferencedDocument {
			if ref != nil && ref.ProcessConditionCode != "" {
				return ref.ProcessConditionCode
			}
		}
	}
	return ""
}

func goblStatusFromCDAR(cdar *CDAR) (*bill.Status, error) {
	if cdar == nil || cdar.ExchangedDocument == nil {
		return nil, fmt.Errorf("invalid CDAR document")
	}
	st := &bill.Status{}
	st.SetAddons(flow6.V1)

	if cdar.ExchangedDocument.ID != "" {
		st.Code = cbc.Code(cdar.ExchangedDocument.ID)
	}
	if cdar.ExchangedDocument.IssueDateTime != nil && cdar.ExchangedDocument.IssueDateTime.DateTimeString != nil {
		d, t, err := parseCDARDateTime(cdar.ExchangedDocument.IssueDateTime.DateTimeString.Value)
		if err != nil {
			return nil, err
		}
		st.IssueDate = d
		if t != nil {
			st.IssueTime = t
		}
	}

	// bill.Status only has the two business-party slots. Map the CDAR
	// trade parties onto them by their RoleCode: SE → Supplier, BY →
	// Customer. Platform-level parties (WK sender, DFH PPF) have no
	// GOBL slot and are dropped — they are transport detail that the
	// generator re-derives.
	assignStatusParty := func(tp *CDARTradeParty) {
		p := goblPartyFromCDAR(tp)
		if p == nil {
			return
		}
		switch p.Ext.Get(flow6.ExtKeyRole) {
		case flow6.RoleSeller:
			if st.Supplier == nil {
				st.Supplier = p
			}
		case flow6.RoleBuyer:
			if st.Customer == nil {
				st.Customer = p
			}
		}
	}
	assignStatusParty(cdar.ExchangedDocument.IssuerTradeParty)
	for _, rp := range cdar.ExchangedDocument.RecipientTradeParties {
		assignStatusParty(rp)
	}

	// Fall back to the canonical MDT-129 ref.IssuerTradeParty slot for
	// the Supplier — that's where the wire spec puts the seller's SIREN,
	// and it's always present (BR-FR-CDV-13). The fallback Supplier
	// carries just the SIREN; richer seller data only exists when the
	// SE party also appears at the ExchangedDocument level.
	if st.Supplier == nil {
		for _, ack := range cdar.AcknowledgementDocuments {
			if ack == nil {
				continue
			}
			for _, ref := range ack.ReferenceReferencedDocument {
				if ref != nil && ref.IssuerTradeParty != nil {
					if p := goblPartyFromCDAR(ref.IssuerTradeParty); p != nil {
						st.Supplier = p
						break
					}
				}
			}
			if st.Supplier != nil {
				break
			}
		}
	}

	// Build StatusLines from each AcknowledgementDocument. The CDAR
	// ProcessConditionCode is pinned on the fr-ctc-flow6-status ext;
	// flow6's reverse mapping derives line.Key and Status.Type from it
	// at normalize-time.
	for _, ack := range cdar.AcknowledgementDocuments {
		if ack == nil {
			continue
		}
		for _, ref := range ack.ReferenceReferencedDocument {
			if ref == nil {
				continue
			}
			st.Lines = append(st.Lines, goblStatusLineFromCDAR(ref))
		}
	}

	return st, nil
}

func goblStatusLineFromCDAR(ref *CDARReferencedDocument) *bill.StatusLine {
	line := &bill.StatusLine{}
	if ref.ProcessConditionCode != "" {
		line.Ext = line.Ext.Set(flow6.ExtKeyStatus, cbc.Code(ref.ProcessConditionCode))
	}
	if dr := goblDocRefFromCDAR(ref); dr != nil {
		line.Doc = dr
	}

	for _, ds := range ref.SpecifiedDocumentStatuses {
		if ds == nil {
			continue
		}
		if ds.ReasonCode != "" {
			// Reason.Key is recovered from the ext by flow6's
			// prepareReasonKey at normalize-time.
			r := &bill.Reason{
				Ext: tax.MakeExtensions().Set(flow6.ExtKeyReason, cbc.Code(ds.ReasonCode)),
			}
			if len(ds.Reason) > 0 {
				r.Description = ds.Reason[0]
			}
			line.Reasons = append(line.Reasons, r)
		}
		if ds.RequestedActionCode != "" {
			// Action.Key is recovered from the ext by flow6's
			// prepareActionKey at normalize-time.
			a := &bill.Action{
				Ext: tax.MakeExtensions().Set(flow6.ExtKeyAction, cbc.Code(ds.RequestedActionCode)),
			}
			if ds.RequestedAction != "" {
				a.Description = ds.RequestedAction
			}
			line.Actions = append(line.Actions, a)
		}
		// SpecifiedDocumentCharacteristics (field-level corrections,
		// DIV/DVA/MAJ…) are not mapped: deep characteristic support is
		// deferred — the reason codes carry the mandatory payload.
	}
	return line
}

// goblDocRefFromCDAR maps the referenced-document identity (invoice
// number, type code, issue date) onto an org.DocumentRef.
func goblDocRefFromCDAR(ref *CDARReferencedDocument) *org.DocumentRef {
	if ref.IssuerAssignedID == "" {
		return nil
	}
	dr := &org.DocumentRef{Code: cbc.Code(ref.IssuerAssignedID)}
	if ref.TypeCode != "" {
		dr.Type = cbc.Key(ref.TypeCode)
	}
	if ref.FormattedIssueDateTime != nil && ref.FormattedIssueDateTime.DateTimeString != nil {
		d, _, err := parseCDARDateTime(ref.FormattedIssueDateTime.DateTimeString.Value)
		if err == nil {
			dd := d
			dr.IssueDate = &dd
		}
	}
	return dr
}

func goblPartyFromCDAR(tp *CDARTradeParty) *org.Party {
	if tp == nil {
		return nil
	}
	p := &org.Party{Name: tp.Name}
	if tp.RoleCode != "" {
		p.Ext = tax.MakeExtensions().Set(flow6.ExtKeyRole, cbc.Code(tp.RoleCode))
	}
	for _, gid := range tp.GlobalIDs {
		if gid == nil || gid.Value == "" {
			continue
		}
		p.Identities = append(p.Identities, &org.Identity{
			Code: cbc.Code(gid.Value),
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, cbc.Code(gid.SchemeID)),
		})
	}
	if tp.URIUniversalCommunication != nil && tp.URIUniversalCommunication.URIID != nil {
		ib := &org.Inbox{
			Scheme: cbc.Code(tp.URIUniversalCommunication.URIID.SchemeID),
			Code:   cbc.Code(tp.URIUniversalCommunication.URIID.Value),
		}
		p.Inboxes = []*org.Inbox{ib}
	}
	if p.Name == "" && len(p.Identities) == 0 && p.Ext.IsZero() && len(p.Inboxes) == 0 {
		return nil
	}
	return p
}

// parseCDARDateTime parses a CDAR CCYYMMDD or CCYYMMDDHHMMSS string into
// (date, optional time). The format attribute hints at the structure but
// the function is tolerant of either length.
func parseCDARDateTime(s string) (cal.Date, *cal.Time, error) {
	if len(s) < 8 {
		return cal.Date{}, nil, fmt.Errorf("invalid CDAR date %q", s)
	}
	y, err := strconv.Atoi(s[0:4])
	if err != nil {
		return cal.Date{}, nil, err
	}
	m, err := strconv.Atoi(s[4:6])
	if err != nil {
		return cal.Date{}, nil, err
	}
	d, err := strconv.Atoi(s[6:8])
	if err != nil {
		return cal.Date{}, nil, err
	}
	date := cal.MakeDate(y, time.Month(m), d)
	if len(s) >= 14 {
		hh, _ := strconv.Atoi(s[8:10])
		mm, _ := strconv.Atoi(s[10:12])
		ss, _ := strconv.Atoi(s[12:14])
		t := cal.MakeTime(hh, mm, ss)
		return date, &t, nil
	}
	return date, nil, nil
}
