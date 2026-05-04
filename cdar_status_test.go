package cii_test

import (
	"testing"
	"time"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/addons/fr/ctc/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allProcessCodes is the canonical list of CDAR ProcessConditionCodes
// supported by Flow 6 (200..213, excluding the unused values).
var allProcessCodes = []string{
	"200", "201", "202", "203", "204", "205", "206",
	"207", "208", "209", "210", "211", "212", "213",
}

func TestFlow6CodeTablesRoundTrip(t *testing.T) {
	t.Run("process codes", func(t *testing.T) {
		for _, code := range allProcessCodes {
			key, typ, ok := flow6.StatusKeyFor(code)
			require.True(t, ok, "process code %s should resolve to a key", code)
			out, ok := flow6.CDARProcessCodeFor(key, typ)
			require.True(t, ok)
			require.Equal(t, code, out, "round-trip mismatch for %s", code)
		}
	})

	t.Run("action codes", func(t *testing.T) {
		actions := []string{"NOA", "PIN", "NIN", "CNF", "CNP", "CNA", "OTH"}
		for _, code := range actions {
			key, ok := flow6.ActionKeyFor(code)
			require.True(t, ok)
			out, ok := flow6.CDARActionCodeFor(key)
			require.True(t, ok)
			require.Equal(t, code, out)
		}
	})

	t.Run("reason default codes", func(t *testing.T) {
		// At least all the default-for-key buckets should round-trip.
		buckets := []cbc.Key{
			bill.ReasonKeyOther, bill.ReasonKeyFinanceTerms,
			bill.ReasonKeyLegal, bill.ReasonKeyNotRecognized,
			bill.ReasonKeyUnknownReceiver, bill.ReasonKeyReferences,
			bill.ReasonKeyPrices, bill.ReasonKeyQuantity,
			bill.ReasonKeyItems, bill.ReasonKeyPaymentTerms,
			bill.ReasonKeyQuality, bill.ReasonKeyDelivery,
		}
		for _, key := range buckets {
			code, ok := flow6.CDARReasonCodeFor(key)
			require.True(t, ok, "no default code for bucket %s", key)
			outKey, ok := flow6.ReasonKeyFor(code)
			require.True(t, ok)
			require.Equal(t, key, outKey, "round-trip for bucket %s broke", key)
		}
	})
}

// buildSyntheticStatus builds a minimal but valid bill.Status for the
// given Flow 6 process code.
func buildSyntheticStatus(t *testing.T, code string) *bill.Status {
	t.Helper()
	key, typ, ok := flow6.StatusKeyFor(code)
	require.True(t, ok, "unknown process code %s", code)

	st := &bill.Status{
		Type:      typ,
		Code:      cbc.Code("STATUS-" + code),
		IssueDate: cal.MakeDate(2025, time.July, 1),
		IssueTime: cal.NewTime(15, 10, 0),
		Supplier: &org.Party{
			Name: "VENDEUR",
			Identities: []*org.Identity{
				{
					Code: "100000009",
					Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
				},
			},
			Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleSE),
		},
		Customer: &org.Party{
			Name: "ACHETEUR",
			Identities: []*org.Identity{
				{
					Code: "200000008",
					Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
				},
			},
			Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleBY),
		},
	}
	st.SetAddons(flow6.V1)

	docDate := cal.MakeDate(2025, time.July, 1)
	line := &bill.StatusLine{
		Key: key,
		Doc: &org.DocumentRef{
			Code:      cbc.Code("F202500003"),
			IssueDate: &docDate,
			Type:      "380",
		},
	}

	// Codes that require at least one Reason per BR-FR-CDV-15.
	switch code {
	case "206", "207", "208", "210", "213":
		line.Reasons = []*bill.Reason{{Key: bill.ReasonKeyLegal,
			Ext:         tax.MakeExtensions().Set(flow6.ExtKeyReasonCode, "TX_TVA_ERR"),
			Description: "Taux TVA erroné",
		}}
		line.Actions = []*bill.Action{{Key: bill.ActionKeyReissue, Description: "Créer une facture rectificative"}}
	}

	// Code 212 (paid) — attach an MEN amount as a complement so we exercise
	// the characteristic round-trip from the StatusLine level.
	if code == "212" {
		amt := num.MakeAmount(50000, 2)
		obj, err := schema.NewObject(&flow6.Characteristic{
			TypeCode: flow6.TypeCodeAmountReceived,
			Numeric:  &amt,
		})
		require.NoError(t, err)
		line.Complements = append(line.Complements, obj)
	}

	st.Lines = []*bill.StatusLine{line}
	return st
}

func TestCDARStatusRoundTripPerCode(t *testing.T) {
	for _, code := range allProcessCodes {
		code := code
		t.Run("CDV-"+code, func(t *testing.T) {
			st := buildSyntheticStatus(t, code)

			// Generate CDAR
			cdar, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
			require.NoError(t, err)
			require.NotNil(t, cdar)

			// Marshal & unmarshal
			data, err := cdar.Bytes()
			require.NoError(t, err)
			require.NotEmpty(t, data)

			parsed, err := cii.UnmarshalCDAR(data)
			require.NoError(t, err)
			require.NotNil(t, parsed)

			// Now turn that parsed CDAR back into a *bill.Status via the
			// public Parse path (uses parseStatus internally).
			out, err := cii.ParseCDARStatus(data)
			require.NoError(t, err)
			require.NotNil(t, out)

			// Type must round-trip.
			expectedKey, expectedType, _ := flow6.StatusKeyFor(code)
			require.Equal(t, expectedType, out.Type, "type mismatch for %s", code)
			require.Len(t, out.Lines, 1, "should have one line for %s", code)
			require.Equal(t, expectedKey, out.Lines[0].Key)

			// Supplier SIREN preserved.
			require.NotNil(t, out.Supplier)
			found := ""
			for _, id := range out.Supplier.Identities {
				if id.Ext.Get(iso.ExtKeySchemeID).String() == "0002" {
					found = id.Code.String()
				}
			}
			require.Equal(t, "100000009", found, "supplier SIREN not preserved for %s", code)
		})
	}
}

func TestCDARStatusRejectionWithCharacteristic(t *testing.T) {
	// Build a 207 (En litige) status with a characteristic on the reason.
	st := buildSyntheticStatus(t, "207")
	require.Len(t, st.Lines, 1)
	line := st.Lines[0]
	require.Len(t, line.Reasons, 1)

	changed := true
	pct, err := num.PercentageFromString("10.00%")
	require.NoError(t, err)

	obj, err := schema.NewObject(&flow6.Characteristic{
		ID:         "BT-152",
		TypeCode:   flow6.TypeCodeInvalidData,
		Name:       "Taux TVA",
		Changed:    &changed,
		Percent:    &pct,
		ReasonCode: "TX_TVA_ERR",
	})
	require.NoError(t, err)
	line.Complements = append(line.Complements, obj)

	cdar, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
	require.NoError(t, err)

	data, err := cdar.Bytes()
	require.NoError(t, err)

	parsed, err := cii.ParseCDARStatus(data)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Len(t, parsed.Lines, 1)

	// The characteristic should be present on the parsed status line.
	require.NotEmpty(t, parsed.Lines[0].Complements,
		"characteristic complement should be preserved on round-trip")

	var found *flow6.Characteristic
	for _, c := range parsed.Lines[0].Complements {
		if inst, ok := c.Instance().(*flow6.Characteristic); ok {
			if inst.TypeCode == flow6.TypeCodeInvalidData {
				found = inst
				break
			}
		}
	}
	require.NotNil(t, found, "DIV characteristic should be present")
	assert.Equal(t, "BT-152", found.ID)
	assert.Equal(t, "Taux TVA", found.Name)
	require.NotNil(t, found.Percent)
	assert.Equal(t, "10.00", found.Percent.StringWithoutSymbol())
	require.NotNil(t, found.Changed)
	assert.True(t, *found.Changed)
	assert.Equal(t, cbc.Code("TX_TVA_ERR"), found.ReasonCode)
}
