package cii

import (
	"strings"

	"github.com/invopop/gobl/addons/fr/ctc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/xmlctx"
)

func parseInvoice(data []byte) (*bill.Invoice, error) {

	in := new(Invoice)
	if err := xmlctx.Unmarshal(data, in, xmlctx.WithNamespaces(
		map[string]string{
			"rsm": NamespaceRSM,
			"ram": NamespaceRAM,
			"qdt": NamespaceQDT,
			"udt": NamespaceUDT,
		},
	)); err != nil {
		return nil, err
	}
	return goblInvoice(in)
}

func goblInvoice(in *Invoice) (*bill.Invoice, error) {
	ctx := goblDetectContext(in)
	ahts := in.Transaction.Settlement

	out := &bill.Invoice{
		Code:     cbc.Code(in.ExchangedDocument.ID),
		Type:     typeCodeParse(in.ExchangedDocument.TypeCode),
		Currency: currency.Code(ahts.Currency),
		Supplier: goblNewParty(in.Transaction.Agreement.Seller),
		Customer: goblNewParty(in.Transaction.Agreement.Buyer),
		Tax: &bill.Tax{
			Rounding: tax.RoundingRuleCurrency,
			Ext: tax.Extensions{
				untdid.ExtKeyDocumentType: cbc.Code(in.ExchangedDocument.TypeCode),
			},
		},
	}

	if ctx != nil {
		out.Addons = tax.Addons{List: ctx.Addons}
		if ctx.Is(ContextPeppolFranceCIUSV1) || ctx.Is(ContextPeppolFranceFacturXV1) {
			if in.ExchangedContext.BusinessContext != nil {
				out.Tax.Ext = out.Tax.Ext.Set(ctc.ExtKeyBillingMode, cbc.Code(in.ExchangedContext.BusinessContext.ID))
			}
		}
	}

	issueDate, err := parseDate(in.ExchangedDocument.IssueDate.DateFormat.Value)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	taxMap := buildTaxCategoryMap(ahts.Tax)

	if err := goblAddTaxDates(ahts.Tax, out); err != nil {
		return nil, err
	}

	if err = goblAddLines(in.Transaction, out, taxMap); err != nil {
		return nil, err
	}

	if out.Payment, err = goblNewPaymentDetails(ahts); err != nil {
		return nil, err
	}

	out.Notes = goblParseNotes(in.ExchangedDocument.IncludedNote)

	if out.Ordering, err = goblNewOrdering(in); err != nil {
		return nil, err
	}
	if out.Delivery, err = goblNewDeliveryDetails(in.Transaction.Delivery); err != nil {
		return nil, err
	}

	if out.Preceding, err = goblParsePreceding(ahts.ReferencedDocument); err != nil {
		return nil, err
	}

	goblApplyTaxRepresentative(in, out)

	if len(ahts.AllowanceCharges) > 0 {
		if err = goblAddChargesAndDiscounts(ahts, out, taxMap); err != nil {
			return nil, err
		}
	}

	if atts := goblAttachments(in.Transaction.Agreement.AdditionalDocument); len(atts) > 0 {
		out.Attachments = atts
	}

	goblAddTaxNotes(ahts.Tax, out)

	return out, nil
}

// goblDetectContext determines the conversion context from guideline and business IDs.
func goblDetectContext(in *Invoice) *Context {
	if in.ExchangedContext == nil || in.ExchangedContext.GuidelineContext == nil {
		return nil
	}
	guidelineID := in.ExchangedContext.GuidelineContext.ID
	var businessID string
	if in.ExchangedContext.BusinessContext != nil {
		businessID = in.ExchangedContext.BusinessContext.ID
	}
	return FindContext(guidelineID, businessID)
}

// goblAddTaxDates extracts BT-7 (VAT point date) and BT-8 (VAT point date code) from
// the first header-level ApplicableTradeTax entry.
func goblAddTaxDates(taxes []*Tax, out *bill.Invoice) error {
	if len(taxes) == 0 {
		return nil
	}
	first := taxes[0]

	// BT-7: VAT point date
	if first.TaxPointDate != nil && first.TaxPointDate.DateFormat != nil {
		vd, err := parseDate(first.TaxPointDate.DateFormat.Value)
		if err != nil {
			return err
		}
		out.ValueDate = &vd
	}

	// BT-8: VAT point date code (UNTDID 2475)
	if first.DueDateTypeCode != "" {
		if key, ok := taxPointCIIKeyMap[first.DueDateTypeCode]; ok {
			out.Tax.Point = key
		}
	}

	return nil
}

// goblParseNotes converts CII IncludedNote entries to GOBL notes.
func goblParseNotes(notes []*Note) []*org.Note {
	if len(notes) == 0 {
		return nil
	}
	out := make([]*org.Note, 0, len(notes))
	for _, note := range notes {
		n := &org.Note{Text: note.Content}
		if note.SubjectCode != "" {
			n.Ext = tax.Extensions{untdid.ExtKeyTextSubject: cbc.Code(note.SubjectCode)}
		}
		out = append(out, n)
	}
	return out
}

// goblParsePreceding converts CII InvoiceReferencedDocument entries to GOBL document references.
func goblParsePreceding(refs []*ReferencedDocument) ([]*org.DocumentRef, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	out := make([]*org.DocumentRef, 0, len(refs))
	for _, ref := range refs {
		docRef := &org.DocumentRef{
			Code: cbc.Code(ref.IssuerAssignedID),
		}
		if ref.IssueDate != nil && ref.IssueDate.DateFormat != nil {
			refDate, err := parseDate(ref.IssueDate.DateFormat.Value)
			if err != nil {
				return nil, err
			}
			docRef.IssueDate = &refDate
		}
		out = append(out, docRef)
	}
	return out, nil
}

// goblApplyTaxRepresentative handles the tax representative party, moving the
// original seller to ordering.seller and replacing the supplier.
func goblApplyTaxRepresentative(in *Invoice, out *bill.Invoice) {
	if in.Transaction.Agreement.TaxRepresentative == nil {
		return
	}
	if out.Ordering == nil {
		out.Ordering = new(bill.Ordering)
	}
	out.Ordering.Seller = out.Supplier
	out.Supplier = goblNewParty(in.Transaction.Agreement.TaxRepresentative)
}

// taxCategoryInfo holds tax category information from header-level tax summary.
type taxCategoryInfo struct {
	exemptionReasonCode string
}

// buildTaxCategoryMap builds a map from the header-level ApplicableTradeTax entries,
// keyed by TypeCode:CategoryCode[:percent] for looking up exemption reason codes.
func buildTaxCategoryMap(taxes []*Tax) map[string]*taxCategoryInfo {
	categoryMap := make(map[string]*taxCategoryInfo)
	for _, t := range taxes {
		if t.CategoryCode == "" {
			continue
		}
		key := buildTaxCategoryKey(t.TypeCode, t.CategoryCode, t.RateApplicablePercent)
		info := &taxCategoryInfo{}
		if t.ExemptionReasonCode != "" {
			info.exemptionReasonCode = t.ExemptionReasonCode
		}
		categoryMap[key] = info
	}
	return categoryMap
}

// buildTaxCategoryKey constructs a unique key for a tax category.
// For standard-rate "S" entries, the normalized percent is included since
// an invoice can have multiple S entries at different rates.
// For other categories (E, AE, Z, O, etc.) percent is omitted.
func buildTaxCategoryKey(typeCode, categoryCode, percent string) string {
	if categoryCode == "S" {
		return typeCode + ":" + categoryCode + ":" + normalizeTaxPercent(percent)
	}
	return typeCode + ":" + categoryCode
}

// normalizeTaxPercent converts a percent string to a canonical form
// by parsing and re-rendering, so "20", "20.0", "20.00" all produce "20".
func normalizeTaxPercent(percent string) string {
	s := strings.TrimSuffix(strings.TrimSpace(percent), "%")
	if s == "" {
		return "0"
	}
	a, err := num.AmountFromString(s)
	if err != nil {
		return s
	}
	return a.String()
}
