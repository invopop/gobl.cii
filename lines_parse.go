package cii

import (
	"math"
	"strings"

	"github.com/invopop/gobl/bill"
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
		Taxes: tax.Set{
			{
				Category: cbc.Code(it.TradeSettlement.ApplicableTradeTax[0].TypeCode),
			},
		},
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

	// BT-128: Invoice line object identifier (TypeCode 130 indicates object identifier)
	if it.Agreement.AdditionalReference != nil && it.Agreement.AdditionalReference.TypeCode == "130" && it.Agreement.AdditionalReference.ID != "" {
		l.Identifier = &org.Identity{
			Code: cbc.Code(it.Agreement.AdditionalReference.ID),
		}
		if it.Agreement.AdditionalReference.RefCode != nil {
			l.Identifier.Ext = tax.Extensions{
				untdid.ExtKeyReference: cbc.Code(*it.Agreement.AdditionalReference.RefCode),
			}
		}
	}

	// BT-132: Purchase order line reference
	if it.Agreement.OrderReference != nil && it.Agreement.OrderReference.LineID != "" {
		l.Order = cbc.Code(it.Agreement.OrderReference.LineID)
	}

	// BT-133: Line buyer accounting reference
	if it.TradeSettlement.AccountingAccount != nil && it.TradeSettlement.AccountingAccount.ID != "" {
		l.Cost = cbc.Code(it.TradeSettlement.AccountingAccount.ID)
	}

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
			Ext: tax.Extensions{
				iso.ExtKeySchemeID: cbc.Code(prod.GlobalID.SchemeID),
			},
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
			n.Key = cbc.Key(note.SubjectCode)
		}
		l.Notes = append(l.Notes, n)
	}
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
			l.Taxes[i].Ext = tax.Extensions{
				untdid.ExtKeyTaxCategory: cbc.Code(tt.CategoryCode),
			}
			key := buildTaxCategoryKey(tt.TypeCode, tt.CategoryCode, tt.RateApplicablePercent)
			if info, ok := taxMap[key]; ok && info.exemptionReasonCode != "" {
				l.Taxes[i].Ext[cef.ExtKeyVATEX] = cbc.Code(info.exemptionReasonCode)
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
