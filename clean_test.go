package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

func TestCleanDocument(t *testing.T) {
	t.Run("reaches every string, however nested", func(t *testing.T) {
		inv := &bill.Invoice{
			Code: "INV-�1",
			Notes: []*org.Note{
				{Text: "Premi�re v�rification"},
			},
			Supplier: &org.Party{
				Name: "Caf� SA",
				Addresses: []*org.Address{
					{Street: "Rue de l'�glise", Locality: "Ch�teau"},
				},
			},
		}
		cleanDocument(inv)

		assert.Equal(t, "INV-1", inv.Code.String())
		assert.Equal(t, "Premire vrification", inv.Notes[0].Text)
		assert.Equal(t, "Caf SA", inv.Supplier.Name)
		assert.Equal(t, "Rue de l'glise", inv.Supplier.Addresses[0].Street)
		assert.Equal(t, "Chteau", inv.Supplier.Addresses[0].Locality)
	})

	t.Run("leaves clean text alone", func(t *testing.T) {
		inv := &bill.Invoice{
			Notes:    []*org.Note{{Text: "Première vérification — 1000 m²"}},
			Supplier: &org.Party{Name: "Café SA"},
		}
		cleanDocument(inv)
		assert.Equal(t, "Première vérification — 1000 m²", inv.Notes[0].Text)
		assert.Equal(t, "Café SA", inv.Supplier.Name)
	})

	t.Run("survives nil fields", func(t *testing.T) {
		inv := &bill.Invoice{}
		assert.NotPanics(t, func() { cleanDocument(inv) })
	})
}
