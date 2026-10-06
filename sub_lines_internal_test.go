package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

// TestWritesSubLinesOnlyExtended covers every context: sub-invoice lines only
// exist in the extended profiles, so no other context may write them.
func TestWritesSubLinesOnlyExtended(t *testing.T) {
	extended := []Context{
		ContextFacturXExtendedV1,
		ContextZUGFeRDExtendedV2,
		ContextPeppolFranceFacturXV1,
		ContextPeppolFranceExtendedV1,
	}
	l := &bill.Line{
		Breakdown: []*bill.SubLine{{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Part"}}},
	}
	for _, ctx := range contexts {
		want := false
		for _, e := range extended {
			if ctx.Is(e) {
				want = true
			}
		}
		assert.Equal(t, want, writesSubLines(ctx, l), ctx.GuidelineID)
	}
}
