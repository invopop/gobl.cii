package cii

import (
	"fmt"
	"strconv"
	"time"

	"github.com/invopop/gobl/addons/fr/ctc/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/xmlctx"
)

// parseStatus converts a raw CDAR XML byte slice into a *bill.Status.
func parseStatus(data []byte) (*bill.Status, error) {
	cdar := new(CDAR)
	if err := xmlctx.Unmarshal(data, cdar, xmlctx.WithNamespaces(
		map[string]string{
			"rsm": NamespaceCDARRSM,
			"ram": NamespaceRAM,
			"qdt": NamespaceQDT,
			"udt": NamespaceUDT,
		},
	)); err != nil {
		return nil, fmt.Errorf("error unmarshaling CDAR: %w", err)
	}
	return goblStatusFromCDAR(cdar)
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

	// Determine type from first acknowledgement's process code.
	if len(cdar.AcknowledgementDocuments) > 0 {
		ack := cdar.AcknowledgementDocuments[0]
		for _, ref := range ack.ReferenceReferencedDocument {
			if ref == nil {
				continue
			}
			if _, typ, ok := flow6.StatusKeyFor(ref.ProcessConditionCode); ok {
				st.Type = typ
				break
			}
		}
	}

	// Map parties. The CDAR places the issuer in IssuerTradeParty and the
	// recipients (including the counterparty and any platforms) in
	// RecipientTradeParty. We map the SE-roled party to Supplier and BY-roled
	// party to Customer; any leftover party becomes Recipient.
	// Map by CDAR slot rather than role-bucketing. The slots are stable
	// across status types; the role codes overlap (e.g. an "SE" Sender on
	// a 23-typed response is the platform claiming the seller's role, not
	// the supplier itself).
	//   ExchangedDocument.IssuerTradeParty           → bill.Status.Issuer
	//   ReferenceReferencedDocument.IssuerTradeParty → bill.Status.Supplier
	//   ExchangedDocument.RecipientTradeParty (BY)   → bill.Status.Customer
	//   any remaining recipient                      → bill.Status.Recipient
	if p := goblPartyFromCDAR(cdar.ExchangedDocument.IssuerTradeParty); p != nil {
		st.Issuer = p
	}
	for _, rp := range cdar.ExchangedDocument.RecipientTradeParties {
		p := goblPartyFromCDAR(rp)
		if p == nil {
			continue
		}
		role := p.Ext.Get(flow6.ExtKeyRole)
		switch {
		case role == flow6.RoleBY && st.Customer == nil:
			st.Customer = p
		case role == flow6.RoleSE && st.Supplier == nil:
			st.Supplier = p
		case st.Recipient == nil:
			st.Recipient = p
		}
	}

	// Pull the supplier from the referenced-document's issuer slot —
	// regardless of any earlier assignment, that slot is the canonical
	// place for the seller's identity in CDAR.
	if len(cdar.AcknowledgementDocuments) > 0 {
		ack := cdar.AcknowledgementDocuments[0]
		for _, ref := range ack.ReferenceReferencedDocument {
			if ref != nil && ref.IssuerTradeParty != nil {
				if p := goblPartyFromCDAR(ref.IssuerTradeParty); p != nil {
					st.Supplier = p
					break
				}
			}
		}
	}

	// Build StatusLines from each AcknowledgementDocument.
	for _, ack := range cdar.AcknowledgementDocuments {
		if ack == nil {
			continue
		}
		for _, ref := range ack.ReferenceReferencedDocument {
			if ref == nil {
				continue
			}
			line, err := goblStatusLineFromCDAR(ref)
			if err != nil {
				return nil, err
			}
			st.Lines = append(st.Lines, line)
		}
	}

	return st, nil
}

func goblStatusLineFromCDAR(ref *CDARReferencedDocument) (*bill.StatusLine, error) {
	line := &bill.StatusLine{}
	if ref.ProcessConditionCode != "" {
		if key, _, ok := flow6.StatusKeyFor(ref.ProcessConditionCode); ok {
			line.Key = key
		}
	}
	if ref.IssuerAssignedID != "" {
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
		line.Doc = dr
	}

	for _, ds := range ref.SpecifiedDocumentStatuses {
		if ds == nil {
			continue
		}
		if ds.ReasonCode != "" {
			r := &bill.Reason{}
			if key, ok := flow6.ReasonKeyFor(ds.ReasonCode); ok {
				r.Key = key
			}
			r.Ext = tax.MakeExtensions().Set(flow6.ExtKeyReasonCode, cbc.Code(ds.ReasonCode))
			if len(ds.Reason) > 0 {
				r.Description = ds.Reason[0]
			}
			line.Reasons = append(line.Reasons, r)
		}
		if ds.RequestedActionCode != "" {
			a := &bill.Action{}
			if key, ok := flow6.ActionKeyFor(ds.RequestedActionCode); ok {
				a.Key = key
			}
			if ds.RequestedAction != "" {
				a.Description = ds.RequestedAction
			}
			line.Actions = append(line.Actions, a)
		}
		// Map characteristics into Complements.
		for _, dc := range ds.SpecifiedDocumentCharacteristics {
			if dc == nil {
				continue
			}
			c := &flow6.Characteristic{
				ID:       dc.ID,
				TypeCode: cbc.Code(dc.TypeCode),
				Name:     dc.Name,
				Location: dc.Location,
			}
			if ds.ReasonCode != "" {
				c.ReasonCode = cbc.Code(ds.ReasonCode)
			}
			if dc.ValueChangedIndicator != nil && dc.ValueChangedIndicator.Value != "" {
				if v, err := strconv.ParseBool(dc.ValueChangedIndicator.Value); err == nil {
					c.Changed = &v
				}
			}
			if dc.ValuePercent != "" {
				if p, err := num.PercentageFromString(dc.ValuePercent + "%"); err == nil {
					c.Percent = &p
				}
			}
			obj, err := schema.NewObject(c)
			if err != nil {
				return nil, fmt.Errorf("wrap characteristic: %w", err)
			}
			line.Complements = append(line.Complements, obj)
		}
	}
	return line, nil
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

