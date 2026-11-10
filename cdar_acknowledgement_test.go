package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cdarWithSequence(seq string) []byte {
	return []byte(`<rsm:CrossDomainAcknowledgementAndResponse` +
		` xmlns:rsm="` + cii.NamespaceCDARRSM + `"` +
		` xmlns:ram="` + cii.NamespaceRAM + `">` +
		`<rsm:AcknowledgementDocument><ram:ReferenceReferencedDocument>` +
		`<ram:SpecifiedDocumentStatus><ram:SequenceNumeric>` + seq + `</ram:SequenceNumeric></ram:SpecifiedDocumentStatus>` +
		`</ram:ReferenceReferencedDocument></rsm:AcknowledgementDocument>` +
		`</rsm:CrossDomainAcknowledgementAndResponse>`)
}

func TestCDARNumeric(t *testing.T) {
	sequence := func(t *testing.T, seq string) cii.CDARNumeric {
		t.Helper()
		doc, err := cii.Decode(cdarWithSequence(seq))
		require.NoError(t, err)
		cdar, ok := doc.(*cii.CDAR)
		require.True(t, ok)
		return cdar.AcknowledgementDocuments[0].ReferenceReferencedDocument[0].SpecifiedDocumentStatuses[0].SequenceNumeric
	}

	t.Run("integer", func(t *testing.T) {
		assert.Equal(t, cii.CDARNumeric(4), sequence(t, "4"))
	})

	t.Run("decimal with zero fraction", func(t *testing.T) {
		assert.Equal(t, cii.CDARNumeric(1), sequence(t, "1.000000000"))
	})

	t.Run("fractional value", func(t *testing.T) {
		_, err := cii.Decode(cdarWithSequence("1.5"))
		assert.ErrorContains(t, err, "not a whole number")
	})

	t.Run("not a number", func(t *testing.T) {
		_, err := cii.Decode(cdarWithSequence("abc"))
		assert.ErrorContains(t, err, "invalid numeric")
	})

	t.Run("encodes as integer", func(t *testing.T) {
		in := cii.NewCDAR()
		in.AcknowledgementDocuments = []*cii.CDARAcknowledgement{{
			ReferenceReferencedDocument: []*cii.CDARReferencedDocument{{
				SpecifiedDocumentStatuses: []*cii.CDARDocumentStatus{{SequenceNumeric: 2}},
			}},
		}}
		data, err := cii.Encode(in)
		require.NoError(t, err)
		assert.Contains(t, string(data), "<ram:SequenceNumeric>2</ram:SequenceNumeric>")
	})
}
