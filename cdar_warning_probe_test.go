package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/require"
)

// TestCDARWarningProbe deliberately violates the BR-FR-04/MDT-91 rule (invalid
// referenced-document type code) and asserts the validator surfaces it, proving
// the zero-finding assertion in TestCDARSchematron actually exercises the
// schematron rather than passing because nothing was checked.
//
// The invalid code has to go in the untdid-document-type extension: MDT-91 is
// read from there, and the converter deliberately keeps no fallback to the
// legacy Doc.Type key. Setting Doc.Type instead leaves the document valid, so
// the probe proves nothing — which is what it did until this was corrected.
func TestCDARWarningProbe(t *testing.T) {
	pc := phormClient(t)

	st := buildSyntheticStatus(t, "205")
	st.Lines[0].Doc.Ext = st.Lines[0].Doc.Ext.
		Set(untdid.ExtKeyDocumentType, cbc.Code("999")) // not a UNTDID 1001 invoice type

	cdar, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
	require.NoError(t, err)
	data, err := cdar.Bytes()
	require.NoError(t, err)

	findings := phormValidate(t, pc, cii.ContextCDARFlow6.VESID, data)
	require.NotEmpty(t, findings,
		"expected the invalid type code to surface — the schematron channel may be silently broken")

	for _, f := range findings {
		t.Log(f)
	}
}
