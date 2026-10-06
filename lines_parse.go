package cii

import (
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

// lineSource ties a parsed line to the document line it was read from and,
// for a line with a breakdown, the sub-invoice lines folded into it.
type lineSource struct {
	doc      *Line
	children []*Line
}

// goblAddLines reads the document's lines. Sub-invoice lines (EXT-FR-FE-162/163)
// under a GROUP line become the GROUP line's breakdown when GOBL can express
// them that way. Otherwise, and for every group named in flat, the hierarchy
// is read flat, in document order: DETAIL lines become lines carrying the
// amounts, and GROUP and INFORMATION lines are kept at a zero price.
func goblAddLines(in *Transaction, out *bill.Invoice, taxMap map[string]*taxCategoryInfo, flat map[string]bool) ([]lineSource, error) {
	items := in.Lines
	children := make(map[string][]*Line)
	ids := make(map[string]bool)
	for _, it := range items {
		if id := goblLineID(it); id != "" {
			ids[id] = true
		}
	}
	for _, it := range items {
		if p := goblParentLineID(it); p != "" && ids[p] {
			children[p] = append(children[p], it)
		}
	}

	lines := make([]*bill.Line, 0, len(items))
	srcs := make([]lineSource, 0, len(items))
	seen := make(map[*Line]bool, len(items))
	var addTree func(it *Line) error
	addTree = func(it *Line) error {
		if seen[it] {
			return nil
		}
		seen[it] = true
		l, err := goblNewLine(it, taxMap)
		if err != nil {
			return err
		}
		lines = append(lines, l)
		srcs = append(srcs, lineSource{doc: it})
		for _, c := range children[goblLineID(it)] {
			if err := addTree(c); err != nil {
				return err
			}
		}
		return nil
	}

	for _, it := range items {
		if p := goblParentLineID(it); p != "" && ids[p] {
			// Read along with its parent.
			continue
		}
		id := goblLineID(it)
		kids := children[id]
		if len(kids) > 0 && !flat[id] && !seen[it] && !goblAnySeen(seen, kids) && goblCanFold(it, kids, taxMap) {
			l, err := goblNewGroupLine(it, kids, taxMap)
			if err != nil {
				return nil, err
			}
			seen[it] = true
			for _, k := range kids {
				seen[k] = true
			}
			lines = append(lines, l)
			srcs = append(srcs, lineSource{doc: it, children: kids})
			continue
		}
		if err := addTree(it); err != nil {
			return nil, err
		}
	}

	// Lines whose parents never lead back to a top-level line, such as a
	// cycle, are still read, flat.
	for _, it := range items {
		if err := addTree(it); err != nil {
			return nil, err
		}
	}

	out.Lines = lines
	return srcs, nil
}

func goblAnySeen(seen map[*Line]bool, lines []*Line) bool {
	for _, l := range lines {
		if seen[l] {
			return true
		}
	}
	return false
}

func goblLineID(it *Line) string {
	if it.LineDoc == nil {
		return ""
	}
	return strings.TrimSpace(it.LineDoc.ID)
}

func goblParentLineID(it *Line) string {
	if it.LineDoc == nil {
		return ""
	}
	return strings.TrimSpace(it.LineDoc.ParentLineID)
}

func goblLineStatus(it *Line) string {
	if it.LineDoc == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(it.LineDoc.LineStatusReasonCode))
}

// goblLineIsSummed reports whether a line's amount counts towards the
// document totals. Only DETAIL lines and lines without a sub-line type do.
func goblLineIsSummed(it *Line) bool {
	switch goblLineStatus(it) {
	case lineStatusGroup, lineStatusInformation:
		return false
	}
	return true
}

// goblCanFold reports whether a parent and its sub-invoice lines fit in a
// single GOBL line with a breakdown. Sub-lines have no taxes of their own, so
// every DETAIL line must share one tax category and rate. They count per unit
// of the parent, so each quantity, allowance and charge must divide evenly by
// the parent's quantity, and their declared amounts must add up to the
// parent's. Nested groups are read flat, as are a GROUP line's own allowances
// and charges, which no total counts. A parent that counts itself can only
// carry INFORMATION lines, which then describe it.
func goblCanFold(parent *Line, kids []*Line, taxMap map[string]*taxCategoryInfo) bool {
	group := goblLineStatus(parent) == lineStatusGroup
	if group && parent.TradeSettlement != nil && len(parent.TradeSettlement.AllowanceCharge) > 0 {
		return false
	}
	qty, ok := goblLineQuantity(parent)
	if !ok {
		return false
	}

	taxKey := ""
	detail := 0
	sum := num.AmountZero
	for _, k := range kids {
		if k.TradeSettlement == nil {
			return false
		}
		switch goblLineStatus(k) {
		case lineStatusGroup:
			return false
		case lineStatusInformation:
			continue
		}
		if !group {
			return false
		}
		detail++
		key := goblLineTaxKey(k, taxMap)
		if key == "" || (taxKey != "" && key != taxKey) {
			return false
		}
		taxKey = key

		q, ok := goblLineQuantity(k)
		if !ok || !goblDividesBy(q, qty) {
			return false
		}
		for _, ac := range k.TradeSettlement.AllowanceCharge {
			for _, v := range []string{ac.Amount, ac.Base} {
				if a, ok := goblDeclaredAmount(v); ok && !goblDividesBy(a, qty) {
					return false
				}
			}
		}
		amount, ok := goblDeclaredLineAmount(k)
		if !ok {
			return false
		}
		sum = sum.MatchPrecision(amount).Add(amount)
	}
	if group && detail == 0 {
		return false
	}
	if declared, ok := goblDeclaredLineAmount(parent); ok && group && !declared.Equals(sum) {
		return false
	}
	return true
}

// goblDividesBy reports whether an amount divides by a quantity without a
// remainder at its own precision.
func goblDividesBy(a, qty num.Amount) bool {
	return a.Divide(qty).Multiply(qty).Equals(a)
}

// goblLineQuantity reads a line's billed quantity, 1 when it states none.
func goblLineQuantity(it *Line) (num.Amount, bool) {
	if it.Quantity == nil || it.Quantity.Quantity == nil || strings.TrimSpace(it.Quantity.Quantity.Amount) == "" {
		return num.MakeAmount(1, 0), true
	}
	q, err := num.AmountFromString(strings.TrimSpace(it.Quantity.Quantity.Amount))
	if err != nil || q.IsZero() {
		return q, false
	}
	return q, true
}

// goblLineTaxKey identifies the tax a line applies: its category, rate and
// exemption.
func goblLineTaxKey(it *Line, taxMap map[string]*taxCategoryInfo) string {
	taxes := goblLineTradeTaxes(it.TradeSettlement.ApplicableTradeTax)
	if len(taxes) != 1 {
		return ""
	}
	t := taxes[0]
	key := buildTaxCategoryKey(t.TypeCode, t.CategoryCode, t.RateApplicablePercent)
	exemption := ""
	if info, ok := taxMap[key]; ok {
		exemption = info.exemptionReasonCode
	}
	return key + ":" + normalizeTaxPercent(t.RateApplicablePercent) + ":" + exemption
}

// goblNewGroupLine reads a parent line and its sub-invoice lines as one line
// with a breakdown, the reverse of newGroupLines. A GROUP line's price and
// amount follow from its DETAIL lines, and it takes their tax.
func goblNewGroupLine(parent *Line, kids []*Line, taxMap map[string]*taxCategoryInfo) (*bill.Line, error) {
	l, err := goblNewLine(parent, taxMap)
	if err != nil {
		return nil, err
	}
	qty := l.Quantity
	group := goblLineStatus(parent) == lineStatusGroup
	taxed := false

	for _, k := range kids {
		cl, err := goblNewLine(k, taxMap)
		if err != nil {
			return nil, err
		}
		sl := &bill.SubLine{
			Quantity:   cl.Quantity.Divide(qty),
			Identifier: cl.Identifier,
			Period:     cl.Period,
			Order:      cl.Order,
			Cost:       cl.Cost,
			Item:       cl.Item,
			Discounts:  cl.Discounts,
			Charges:    cl.Charges,
			Notes:      cl.Notes,
		}
		for _, d := range sl.Discounts {
			d.Amount = d.Amount.Divide(qty)
			if d.Base != nil {
				b := d.Base.Divide(qty)
				d.Base = &b
			}
		}
		for _, c := range sl.Charges {
			c.Amount = c.Amount.Divide(qty)
			if c.Base != nil {
				b := c.Base.Divide(qty)
				c.Base = &b
			}
		}
		if !goblLineIsSummed(k) {
			sl.Item.Price = nil
		} else if group && !taxed {
			l.Taxes = cl.Taxes
			taxed = true
		}
		l.Breakdown = append(l.Breakdown, sl)
	}
	return l, nil
}

func goblNewLine(it *Line, taxMap map[string]*taxCategoryInfo) (*bill.Line, error) {
	var err error
	summed := goblLineIsSummed(it)
	l := &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item: &org.Item{
			Name: cleanString(strings.TrimSpace(it.Product.Name)),
		},
	}

	// GROUP and INFORMATION lines are kept for their details, but at a zero
	// price: their DETAIL lines already carry the amounts, and GOBL requires
	// every invoice line to have a price.
	price := num.AmountZero
	if it.Agreement != nil && it.Agreement.NetPrice != nil && summed {
		price, err = goblLinePrice(it.Agreement.NetPrice)
		if err != nil {
			return nil, err
		}
	}
	l.Item.Price = &price

	ts := it.TradeSettlement
	if ts == nil {
		ts = new(TradeSettlement)
	}
	taxes := goblLineTradeTaxes(ts.ApplicableTradeTax)
	if len(taxes) > 0 {
		l.Taxes = tax.Set{
			{
				Category: cbc.Code(taxes[0].TypeCode),
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
		// BT-130: the code becomes a GOBL unit, or stays in the extension
		// when GOBL has no unit for it.
		u := cbc.Code(it.Quantity.Quantity.UnitCode)
		l.Item.Unit, l.Item.Ext = goblUnit(l.Item.Ext, u)
	}

	goblLineProduct(it.Product, l.Item)
	goblLineNotes(it.LineDoc, l)

	if err := goblLineTaxes(taxes, l, taxMap); err != nil {
		return nil, err
	}

	// What a line no total counts allows or charges is not part of the
	// totals either.
	if len(ts.AllowanceCharge) > 0 && summed {
		l, err = getLineCharges(ts.AllowanceCharge, l)
		if err != nil {
			return nil, err
		}
	}

	// BG-32: item attributes
	for _, char := range it.Product.Characteristics {
		attr, err := goblItemAttribute(char)
		if err != nil {
			return nil, err
		}
		if attr != nil {
			l.Item.Attributes = append(l.Item.Attributes, attr)
		}
	}

	if it.Agreement != nil {
		goblLineAgreement(it.Agreement, l)
	}
	goblLineSettlement(ts, l)

	per, err := goblLinePeriod(ts.Period)
	if err != nil {
		return nil, err
	}
	l.Period = per

	return l, nil
}

// goblLineTradeTaxes drops the trade tax entries that name no tax, such as
// the one a French GROUP line may carry only for its tax due date
// (EXT-FR-FE-180): there is no category to apply.
func goblLineTradeTaxes(taxes []*Tax) []*Tax {
	out := make([]*Tax, 0, len(taxes))
	for _, t := range taxes {
		if t == nil || (t.TypeCode == "" && t.CategoryCode == "" && t.RateApplicablePercent == "") {
			continue
		}
		out = append(out, t)
	}
	return out
}

// goblItemAttribute converts a CII ApplicableProductCharacteristic (BG-32)
// into a GOBL attribute, preferring the measure over the plain value when both
// are present, as the measure also carries the unit.
func goblItemAttribute(char *Characteristic) (*org.Attribute, error) {
	description := cleanString(strings.TrimSpace(char.Description))
	if description == "" {
		return nil, nil
	}
	attr := &org.Attribute{Label: description}
	switch {
	case char.ValueMeasure != nil && char.ValueMeasure.Amount != "":
		amount, err := num.AmountFromString(char.ValueMeasure.Amount)
		if err != nil {
			return nil, err
		}
		attr.Amount = &amount
		if char.ValueMeasure.UnitCode != "" {
			attr.Unit, attr.Ext = goblUnit(attr.Ext, cbc.Code(char.ValueMeasure.UnitCode))
		}
	case char.Value != "":
		attr.Text = cleanString(strings.TrimSpace(char.Value))
	default:
		return nil, nil
	}
	return attr, nil
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
		item.Description = cleanString(strings.TrimSpace(*prod.Description))
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
			n.Text = cleanString(strings.TrimSpace(note.Content))
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

	if ag.ItemSellerParty != nil {
		l.Seller = goblNewParty(ag.ItemSellerParty)
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
		per.Start = &start
	}
	if p.End != nil && p.End.DateFormat != nil {
		end, err := parseDate(p.End.DateFormat.Value)
		if err != nil {
			return nil, err
		}
		per.End = &end
	}
	if per.Start == nil && per.End == nil {
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
