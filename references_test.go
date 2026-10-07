package cii_test

import (
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// referencesEnvelope loads a complete invoice with every ordering, delivery
// and preceding document reference the base export writes.
func referencesEnvelope(t *testing.T) *gobl.Envelope {
	t.Helper()
	env := loadEnvelope(t, "en16931/invoice-complete.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	inv.Ordering.Sales = []*org.DocumentRef{{Code: "SO-1"}}
	inv.Ordering.Projects = []*org.DocumentRef{{Code: "PRJ-1", Description: "Bridge"}}
	inv.Ordering.Tender = []*org.DocumentRef{{Code: "LOT-1"}}
	inv.Ordering.Identities = []*org.Identity{{Code: "OBJ-1"}}
	inv.Delivery.Receiver = nil
	inv.Delivery.Identities = []*org.Identity{{Code: "LOC-1"}}
	inv.Preceding = []*org.DocumentRef{{Code: "INV-0", IssueDate: cal.NewDate(2013, 1, 1)}}
	require.NoError(t, env.Calculate())
	return env
}

// roundTrip exports the envelope, then decodes and imports the result,
// letting edit change the decoded document first.
func roundTrip(t *testing.T, env *gobl.Envelope, edit func(*cii.Invoice)) (*bill.Invoice, error) {
	t.Helper()
	doc, err := cii.ExportInvoice(env)
	require.NoError(t, err)
	data, err := cii.Encode(doc)
	require.NoError(t, err)
	in, err := cii.Decode(data)
	require.NoError(t, err)
	if edit != nil {
		edit(in.(*cii.Invoice))
	}
	out, err := cii.Import(in)
	if err != nil {
		return nil, err
	}
	return out.Extract().(*bill.Invoice), nil
}

func TestReferencesRoundTrip(t *testing.T) {
	t.Run("all references", func(t *testing.T) {
		out, err := roundTrip(t, referencesEnvelope(t), nil)
		require.NoError(t, err)

		assert.Equal(t, "SO-1", out.Ordering.Sales[0].Code.String())
		assert.Equal(t, "PRJ-1", out.Ordering.Projects[0].Code.String())
		assert.Equal(t, "Bridge", out.Ordering.Projects[0].Description)
		assert.Equal(t, "LOT-1", out.Ordering.Tender[0].Code.String())
		assert.Equal(t, "OBJ-1", out.Ordering.Identities[0].Code.String())
		assert.Equal(t, "LOC-1", out.Delivery.Identities[0].Code.String())
		assert.Empty(t, out.Delivery.Identities[0].Label)
		require.Len(t, out.Preceding, 1)
		assert.Equal(t, "INV-0", out.Preceding[0].Code.String())
		assert.Equal(t, "2013-01-01", out.Preceding[0].IssueDate.String())
	})

	t.Run("delivery location with scheme", func(t *testing.T) {
		env := referencesEnvelope(t)
		env.Extract().(*bill.Invoice).Delivery.Identities[0].Label = "0088"
		out, err := roundTrip(t, env, nil)
		require.NoError(t, err)
		assert.Equal(t, "LOC-1", out.Delivery.Identities[0].Code.String())
		assert.Equal(t, "0088", out.Delivery.Identities[0].Label)
	})

	t.Run("tender issue date", func(t *testing.T) {
		out, err := roundTrip(t, referencesEnvelope(t), func(in *cii.Invoice) {
			for _, d := range in.Transaction.Agreement.AdditionalDocument {
				if d.TypeCode == cii.AdditionalDocumentTypeTender {
					d.IssueDate = &cii.FormattedIssueDate{DateFormat: &cii.Date{Value: "20130201", Format: "102"}}
				}
			}
		})
		require.NoError(t, err)
		assert.Equal(t, "2013-02-01", out.Ordering.Tender[0].IssueDate.String())
	})

	t.Run("delivery party without name or address", func(t *testing.T) {
		out, err := roundTrip(t, referencesEnvelope(t), func(in *cii.Invoice) {
			in.Transaction.Delivery.Receiver.Name = ""
		})
		require.NoError(t, err)
		assert.Nil(t, out.Delivery.Receiver)
	})
}

func TestImportInvalidDates(t *testing.T) {
	bad := &cii.Date{Value: "bad", Format: "102"}
	tests := []struct {
		name string
		edit func(*cii.Invoice)
	}{
		{"preceding", func(in *cii.Invoice) {
			in.Transaction.Settlement.ReferencedDocument[0].IssueDate.DateFormat = bad
		}},
		{"tender", func(in *cii.Invoice) {
			for _, d := range in.Transaction.Agreement.AdditionalDocument {
				if d.TypeCode == cii.AdditionalDocumentTypeTender {
					d.IssueDate = &cii.FormattedIssueDate{DateFormat: bad}
				}
			}
		}},
		{"delivery date", func(in *cii.Invoice) {
			in.Transaction.Delivery.Event.OccurrenceDate.DateFormat = bad
		}},
		{"tax point", func(in *cii.Invoice) {
			in.Transaction.Settlement.Tax[0].TaxPointDate = &cii.IssueDate{DateFormat: bad}
		}},
		{"payment terms", func(in *cii.Invoice) {
			in.Transaction.Settlement.PaymentTerms[0].DueDate = &cii.IssueDate{DateFormat: bad}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := roundTrip(t, referencesEnvelope(t), tt.edit)
			assert.Error(t, err)
		})
	}
}
