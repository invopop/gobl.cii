package cii_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityCodes flattens identities to "scheme:code" for one-line comparisons.
func identityCodes(ids []*org.Identity) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Ext.Get(iso.ExtKeySchemeID).String()+":"+id.Code.String())
	}
	return out
}

func parseStatusFixture(t *testing.T, name string) *bill.Status {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(getParsePath(), name))
	require.NoError(t, err)
	st, err := cii.ParseCDARStatus(data)
	require.NoError(t, err)
	return st
}

// The seller's SIREN is mandatory in MDT-129 (BR-FR-CDV-13) while the
// identifiers on the ExchangedDocument parties are optional, so the parsed
// supplier must carry the MDT-129 SIREN even when the SE party came without
// one. Anything else the platform put on the parties is kept as-is and left
// to validation.
func TestParseCDARStatusReferencedSIREN(t *testing.T) {
	t.Run("202: SE party has no 0002, MDT-129 does", func(t *testing.T) {
		st := parseStatusFixture(t, "cdv-202-recipients-0225-globalid.xml")

		require.NotNil(t, st.Supplier)
		assert.Equal(t, "VENDEUR", st.Supplier.Name)
		assert.Equal(t, []string{"0225:100000009", "0002:100000009"}, identityCodes(st.Supplier.Identities))

		require.NotNil(t, st.Customer)
		assert.Equal(t, []string{"0225:200000008"}, identityCodes(st.Customer.Identities))

		require.Len(t, st.Lines, 1)
		require.NotNil(t, st.Lines[0].Doc)
		assert.Equal(t, []string{"0002:100000009"}, identityCodes(st.Lines[0].Doc.Identities))

		// 0225 is not a party identity scheme (G1.73): the platform's
		// deviation surfaces as a validation fault, not as a lost SIREN.
		env, err := gobl.Envelop(st)
		require.NoError(t, err)
		err = env.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GOBL-FR-CTC-FLOW6-ORG-PARTY-05")
		assert.NotContains(t, err.Error(), "GOBL-FR-CTC-FLOW6-BILL-STATUS-06")
	})

	t.Run("204: MDT-129 itself has no 0002, no SIREN is invented", func(t *testing.T) {
		st := parseStatusFixture(t, "cdv-204-issuer-0225-globalid.xml")

		require.NotNil(t, st.Supplier)
		assert.Equal(t, []string{"0225:100000009_00012"}, identityCodes(st.Supplier.Identities))
		require.Len(t, st.Lines, 1)
		require.NotNil(t, st.Lines[0].Doc)
		assert.Equal(t, []string{"0225:100000009_00012"}, identityCodes(st.Lines[0].Doc.Identities))

		env, err := gobl.Envelop(st)
		require.NoError(t, err)
		err = env.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GOBL-FR-CTC-FLOW6-BILL-STATUS-06")
	})
}
