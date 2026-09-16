package cii

import (
	"fmt"
	"math"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblAddLines(in *Transaction, out *bill.Invoice, taxMap map[string]*taxCategoryInfo) error {
	items := in.Lines
	lines := make([]*bill.Line, 0, len(items))

	for _, it := range items {
		l, err := goblNewLine(it, taxMap)
		if err != nil {
			return err
		}
		lines = append(lines, l)
	}

	out.Lines = lines
	return nil
}

func goblNewLine(it *Line, taxMap map[string]*taxCategoryInfo) (*bill.Line, error) {
	price, err := goblLinePrice(it.Agreement.NetPrice)
	if err != nil {
		return nil, err
	}

	l := &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item: &org.Item{
			Name:  strings.TrimSpace(it.Product.Name),
			Price: &price,
		},
	}

	if len(it.TradeSettlement.ApplicableTradeTax) > 0 {
		l.Taxes = tax.Set{
			{
				Category: cbc.Code(it.TradeSettlement.ApplicableTradeTax[0].TypeCode),
			},
		}
	}

	if it.Quantity != nil && it.Quantity.Quantity != nil {
		l.Quantity, err = num.AmountFromString(it.Quantity.Quantity.Amount)
		if err != nil {
			return nil, err
		}
	}

	if it.Quantity != nil && it.Quantity.Quantity != nil && it.Quantity.Quantity.UnitCode != "" {
		u := cbc.Code(it.Quantity.Quantity.UnitCode)
		l.Item.Unit = goblUnitFromUNECE(u)
	}

	goblLineProduct(it.Product, l.Item)
	goblLineNotes(it.LineDoc, l)

	if err := goblLineTaxes(it.TradeSettlement.ApplicableTradeTax, l, taxMap); err != nil {
		return nil, err
	}

	if len(it.TradeSettlement.AllowanceCharge) > 0 {
		l, err = getLineCharges(it.TradeSettlement.AllowanceCharge, l)
		if err != nil {
			return nil, err
		}
	}

	if len(it.Product.Characteristics) > 0 {
		l.Item.Meta = make(cbc.Meta)
		for _, char := range it.Product.Characteristics {
			key := formatKey(char.Description)
			l.Item.Meta[key] = char.Value
		}
	}

	goblLineAgreement(it.Agreement, l)
	goblLineSettlement(it.TradeSettlement, l)

	per, err := goblLinePeriod(it.TradeSettlement.Period)
	if err != nil {
		return nil, err
	}
	l.Period = per

	return l, nil
}

// goblLinePrice extracts and normalizes the net price, dividing by base quantity if present.
func goblLinePrice(np *NetPrice) (num.Amount, error) {
	price, err := num.AmountFromString(np.Amount)
	if err != nil {
		return price, err
	}
	// BT-148: Price base quantity — normalize unit price
	if bq := np.BaseQuantity; bq != nil && bq.Amount != "" {
		baseQuantity, err := num.AmountFromString(bq.Amount)
		if err != nil {
			return price, err
		}
		if !baseQuantity.IsZero() {
			precision := calculateRequiredPrecision(price, baseQuantity)
			price = price.RescaleUp(precision).Divide(baseQuantity)
		}
	}
	return price, nil
}

// goblLineProduct populates item identities and metadata from the CII product.
func goblLineProduct(prod *Product, item *org.Item) {
	if prod.SellerAssignedID != nil {
		item.Ref = cbc.Code(*prod.SellerAssignedID)
	}

	if prod.BuyerAssignedID != nil {
		item.Identities = append(item.Identities, &org.Identity{
			Code: cbc.Code(*prod.BuyerAssignedID),
		})
	}

	if prod.GlobalID != nil {
		item.Identities = append(item.Identities, &org.Identity{
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(prod.GlobalID.SchemeID),
			}),
			Code: cbc.Code(prod.GlobalID.Value),
		})
	}

	if prod.Description != nil {
		item.Description = strings.TrimSpace(*prod.Description)
	}

	if prod.Origin != nil {
		item.Origin = l10n.ISOCountryCode(*prod.Origin)
	}

	// BT-158: Item classification
	// Use Label for the scheme ID to avoid conflicting with GlobalID detection,
	// which relies on Ext[iso.ExtKeySchemeID].
	if prod.Classification != nil && prod.Classification.Code != nil {
		id := &org.Identity{
			Code: cbc.Code(prod.Classification.Code.Value),
		}
		if prod.Classification.Code.ListID != "" {
			id.Label = prod.Classification.Code.ListID
		}
		item.Identities = append(item.Identities, id)
	}
}

// goblLineNotes populates line notes from the CII line document.
func goblLineNotes(lineDoc *LineDoc, l *bill.Line) {
	if lineDoc == nil || len(lineDoc.Note) == 0 {
		return
	}
	l.Notes = make([]*org.Note, 0, len(lineDoc.Note))
	for _, note := range lineDoc.Note {
		n := &org.Note{}
		if note.Content != "" {
			n.Text = strings.TrimSpace(note.Content)
		}
		if note.SubjectCode != "" {
			n.Ext = tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: cbc.Code(note.SubjectCode)})
		}
		l.Notes = append(l.Notes, n)
	}
}

// goblLineAgreement populates line-level references from the CII agreement.
func goblLineAgreement(ag *LineAgreement, l *bill.Line) {
	// BT-128: Invoice line object identifier (TypeCode 130 indicates object identifier)
	if ag.AdditionalReference != nil && ag.AdditionalReference.TypeCode == "130" && ag.AdditionalReference.ID != "" {
		l.Identifier = &org.Identity{
			Code: cbc.Code(ag.AdditionalReference.ID),
		}
		if ag.AdditionalReference.RefCode != nil {
			l.Identifier.Ext = tax.ExtensionsOf(cbc.CodeMap{
				untdid.ExtKeyReference: cbc.Code(*ag.AdditionalReference.RefCode),
			})
		}
	}

	// BT-132: Purchase order line reference
	if ag.OrderReference != nil && ag.OrderReference.LineID != "" {
		l.Order = cbc.Code(ag.OrderReference.LineID)
	}
}

// goblLineSettlement populates line-level settlement fields from the CII trade settlement.
func goblLineSettlement(ts *TradeSettlement, l *bill.Line) {
	// BT-133: Line buyer accounting reference
	if ts.AccountingAccount != nil && ts.AccountingAccount.ID != "" {
		l.Cost = cbc.Code(ts.AccountingAccount.ID)
	}
}

// goblLinePeriod parses BT-134/BT-135 invoice line period dates.
func goblLinePeriod(p *Period) (*cal.Period, error) {
	if p == nil {
		return nil, nil
	}
	per := &cal.Period{}
	if p.Start != nil && p.Start.DateFormat != nil {
		start, err := parseDate(p.Start.DateFormat.Value)
		if err != nil {
			return nil, err
		}
		per.Start = start
	}
	if p.End != nil && p.End.DateFormat != nil {
		end, err := parseDate(p.End.DateFormat.Value)
		if err != nil {
			return nil, err
		}
		per.End = end
	}
	if per.Start.IsZero() && per.End.IsZero() {
		return nil, nil
	}
	return per, nil
}

// goblLineTaxes populates line tax information from the CII trade tax entries.
func goblLineTaxes(taxes []*Tax, l *bill.Line, taxMap map[string]*taxCategoryInfo) error {
	for i, tt := range taxes {
		// Ensure the Taxes slice has enough capacity
		for len(l.Taxes) <= i {
			l.Taxes = append(l.Taxes, &tax.Combo{
				Category: cbc.Code(tt.TypeCode),
			})
		}
		if tt.CategoryCode != "" {
			l.Taxes[i].Ext = tax.ExtensionsOf(cbc.CodeMap{
				untdid.ExtKeyTaxCategory: cbc.Code(tt.CategoryCode),
			})
			key := buildTaxCategoryKey(tt.TypeCode, tt.CategoryCode, tt.RateApplicablePercent)
			if info, ok := taxMap[key]; ok && info.exemptionReasonCode != "" {
				l.Taxes[i].Ext = l.Taxes[i].Ext.Set(cef.ExtKeyVATEX, cbc.Code(info.exemptionReasonCode))
			}
		}
		if tt.RateApplicablePercent != "" {
			if !strings.HasSuffix(tt.RateApplicablePercent, "%") {
				tt.RateApplicablePercent += "%"
			}
			p, err := num.PercentageFromString(tt.RateApplicablePercent)
			if err != nil {
				return err
			}
			// Skip setting percent if it's 0% and tax category is not "Z" (zero-rated)
			if p.IsZero() && tt.CategoryCode != "Z" {
				continue
			}
			l.Taxes[i].Percent = &p
		}
	}
	return nil
}

// getLineCharges parses inline charges and discounts from the CII document
func getLineCharges(alwcs []*AllowanceCharge, l *bill.Line) (*bill.Line, error) {
	for _, ac := range alwcs {
		if ac.ChargeIndicator.Value {
			c, err := goblNewLineCharge(ac)
			if err != nil {
				return nil, err
			}
			if l.Charges == nil {
				l.Charges = make([]*bill.LineCharge, 0)
			}
			l.Charges = append(l.Charges, c)
		} else {
			d, err := goblNewLineDiscount(ac)
			if err != nil {
				return nil, err
			}
			if l.Discounts == nil {
				l.Discounts = make([]*bill.LineDiscount, 0)
			}
			l.Discounts = append(l.Discounts, d)
		}
	}
	return l, nil
}

// calculateRequiredPrecision determines the decimal precision needed when
// dividing a price by a base quantity to avoid rounding errors.
func calculateRequiredPrecision(price, baseQuantity num.Amount) uint32 {
	priceExp := price.Exp()

	baseQtyNormalized := baseQuantity.Rescale(0)
	baseQtyFloat := math.Abs(float64(baseQtyNormalized.Value()))

	additionalDecimals := uint32(0)
	if baseQtyFloat > 1 {
		additionalDecimals = uint32(math.Ceil(math.Log10(baseQtyFloat)))
	}

	return priceExp + additionalDecimals
}

// NoteSrcReconciliation marks notes generated during conversion rather than
// sent by the issuer. Shared with gobl.ubl; stripped again on re-export.
const NoteSrcReconciliation cbc.Key = "reconciliation"

// lineTotalTolerance matches the slack PEPPOL-EN16931-R120 allows on BT-131.
var lineTotalTolerance = num.MakeAmount(2, 2)

// goblReconcileLines rebuilds any line whose price and quantity disagree with
// the total the issuer stated. BR-CO-10 ties the document totals to BT-131, but
// nothing in EN 16931 checks BT-131 against the line's own components, so the
// stated total is the figure to trust.
func goblReconcileLines(in *Transaction, out *bill.Invoice) error {
	if err := out.Calculate(); err != nil {
		return err
	}
	var rebuilt bool
	for i, it := range in.Lines {
		if i >= len(out.Lines) {
			break
		}
		ok, err := goblTrustStatedLineTotal(it, out.Lines[i])
		if err != nil {
			return fmt.Errorf("line %d: %w", i, err)
		}
		rebuilt = rebuilt || ok
	}
	if !rebuilt {
		return nil
	}
	return out.Calculate()
}

func goblTrustStatedLineTotal(it *Line, l *bill.Line) (bool, error) {
	if it.TradeSettlement == nil || it.TradeSettlement.Sum == nil || l.Total == nil || l.Item == nil {
		return false, nil
	}
	stated, err := num.AmountFromString(it.TradeSettlement.Sum.Amount)
	if err != nil {
		return false, fmt.Errorf("parsing BT-131: %w", err)
	}
	if withinTolerance(*l.Total, stated) || l.Quantity.IsZero() {
		return false, nil
	}

	// Deriving the price from BT-131 means the line's own allowances and charges
	// are already in it, so record them before dropping them.
	note := fmt.Sprintf("Line net amount as received: %s. Price and quantity as sent gave %s%s.",
		stated.String(), l.Total.String(), goblAbsorbedAmounts(l))
	price := stated.RescaleUp(stated.Exp() + 4).Divide(l.Quantity)
	l.Item.Price = &price
	l.Discounts = nil
	l.Charges = nil
	l.Notes = append(l.Notes, &org.Note{
		Key:  org.NoteKeyGeneral,
		Src:  NoteSrcReconciliation,
		Text: note,
	})
	return true, nil
}

func goblAbsorbedAmounts(l *bill.Line) string {
	var alw, chg num.Amount
	for _, d := range l.Discounts {
		alw = alw.MatchPrecision(d.Amount).Add(d.Amount)
	}
	for _, c := range l.Charges {
		chg = chg.MatchPrecision(c.Amount).Add(c.Amount)
	}
	switch {
	case !alw.IsZero() && !chg.IsZero():
		return fmt.Sprintf(", absorbing an allowance of %s and a charge of %s", alw, chg)
	case !alw.IsZero():
		return fmt.Sprintf(", absorbing an allowance of %s", alw)
	case !chg.IsZero():
		return fmt.Sprintf(", absorbing a charge of %s", chg)
	}
	return ""
}

func withinTolerance(a, b num.Amount) bool {
	a = a.MatchPrecision(b)
	b = b.MatchPrecision(a)
	return a.Subtract(b).Abs().Compare(lineTotalTolerance.MatchPrecision(a)) <= 0
}
