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
	// Detect context from guideline/business IDs
	var ctx *Context
	if in.ExchangedContext != nil && in.ExchangedContext.GuidelineContext != nil {
		guidelineID := in.ExchangedContext.GuidelineContext.ID
		var businessID string
		if in.ExchangedContext.BusinessContext != nil {
			businessID = in.ExchangedContext.BusinessContext.ID
		}
		ctx = FindContext(guidelineID, businessID)
	}

	out := &bill.Invoice{
		Code:     cbc.Code(in.ExchangedDocument.ID),
		Type:     typeCodeParse(in.ExchangedDocument.TypeCode),
		Currency: currency.Code(in.Transaction.Settlement.Currency),
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
		if ctx.GuidelineID == ContextPeppolFranceFacturXV1.GuidelineID || ctx.GuidelineID == ContextPeppolFranceCIUSV1.GuidelineID {
			if in.ExchangedContext.BusinessContext != nil {
				out.Tax.Ext.Set(ctc.ExtKeyBillingMode, cbc.Code(in.ExchangedContext.BusinessContext.ID))
			}
		}
	}

	issueDate, err := parseDate(in.ExchangedDocument.IssueDate.DateFormat.Value)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	// Build tax category map from header-level tax summary for exemption code lookups
	ahts := in.Transaction.Settlement
	taxMap := buildTaxCategoryMap(ahts.Tax)

	if err = goblAddLines(in.Transaction, out, taxMap); err != nil {
		return nil, err
	}

	// Payment comprised of terms, means and payee. Check there is relevant info in at least one of them to create a payment
	if out.Payment, err = goblNewPaymentDetails(ahts); err != nil {
		return nil, err
	}

	if len(in.ExchangedDocument.IncludedNote) > 0 {
		out.Notes = make([]*org.Note, 0, len(in.ExchangedDocument.IncludedNote))
		for _, note := range in.ExchangedDocument.IncludedNote {
			n := &org.Note{
				Text: note.Content,
			}
			if note.SubjectCode != "" {
				n.Code = cbc.Code(note.SubjectCode)
			}
			out.Notes = append(out.Notes, n)
		}
	}

	if out.Ordering, err = goblNewOrdering(in); err != nil {
		return nil, err
	}
	if out.Delivery, err = goblNewDeliveryDetails(in.Transaction.Delivery); err != nil {
		return nil, err
	}

	if len(ahts.ReferencedDocument) > 0 {
		out.Preceding = make([]*org.DocumentRef, 0, len(ahts.ReferencedDocument))
		for _, ref := range ahts.ReferencedDocument {
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
			out.Preceding = append(out.Preceding, docRef)
		}
	}

	if in.Transaction.Agreement.TaxRepresentative != nil {
		// Move the original seller to the ordering.seller party
		if out.Ordering == nil {
			out.Ordering = new(bill.Ordering)
		}
		out.Ordering.Seller = out.Supplier

		// Overwrite the seller field with the tax representative
		out.Supplier = goblNewParty(in.Transaction.Agreement.TaxRepresentative)
	}

	if len(ahts.AllowanceCharges) > 0 {
		if err = goblAddChargesAndDiscounts(ahts, out, taxMap); err != nil {
			return nil, err
		}
	}

	return out, nil
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
