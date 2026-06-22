package cii

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
)

func TestGoblFaultFromCDAR(t *testing.T) {
	t.Run("percent characteristic with name, ID and location", func(t *testing.T) {
		f := goblFaultFromCDAR(&CDARDocumentCharacteristic{
			ID:           "BT-152",
			TypeCode:     "DIV",
			Name:         "Taux TVA",
			Location:     "/rsm:CrossIndustryInvoice/ram:RateApplicablePercent",
			ValuePercent: "10.00",
		})
		assert.Equal(t, cbc.Code("DIV"), f.Code)
		assert.Equal(t, "Taux TVA (BT-152): 10.00%", f.Message)
		assert.Equal(t, []string{"/rsm:CrossIndustryInvoice/ram:RateApplicablePercent"}, f.Paths)
	})

	t.Run("amount characteristic", func(t *testing.T) {
		f := goblFaultFromCDAR(&CDARDocumentCharacteristic{
			TypeCode:    "MNA",
			ValueAmount: &CDARValueAmount{Value: "120.00", CurrencyID: "EUR"},
		})
		assert.Equal(t, cbc.Code("MNA"), f.Code)
		assert.Equal(t, "120.00 EUR", f.Message)
		assert.Empty(t, f.Paths)
	})

	t.Run("date characteristic", func(t *testing.T) {
		f := goblFaultFromCDAR(&CDARDocumentCharacteristic{
			TypeCode: "MAJ",
			Name:     "Date de livraison",
			ValueDateTime: &CDARIssueDateTime{
				DateTimeString: &CDARDateTimeString{Value: "20250730", Format: "102"},
			},
		})
		assert.Equal(t, "Date de livraison: 2025-07-30", f.Message)
	})

	t.Run("no TypeCode falls back to the business-term ID", func(t *testing.T) {
		f := goblFaultFromCDAR(&CDARDocumentCharacteristic{
			ID:   "BT-152",
			Name: "Taux TVA",
		})
		assert.Equal(t, cbc.Code("BT-152"), f.Code)
		assert.Equal(t, "Taux TVA", f.Message)
	})

	t.Run("empty characteristic is dropped", func(t *testing.T) {
		assert.Nil(t, goblFaultFromCDAR(&CDARDocumentCharacteristic{Name: "orphan"}))
		assert.Nil(t, goblFaultFromCDAR(nil))
	})
}
