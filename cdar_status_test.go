package cii_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/addons/fr/ctc/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/phive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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


// findSEPartyAcross returns whichever of Supplier/Issuer/Recipient on st
// carries fr-ctc-role=SE — used in tests that don't run the flow6
// normaliser (which would otherwise auto-fill st.Supplier from the SE
// party in the other slots). Falls back to a SIREN-only party with no
// role (e.g. from ref.IssuerTradeParty).
func findSEPartyAcross(st *bill.Status) *org.Party {
	for _, p := range []*org.Party{st.Supplier, st.Issuer, st.Recipient} {
		if p == nil {
			continue
		}
		if p.Ext.Get(flow6.ExtKeyRole) == flow6.RoleSE {
			return p
		}
		if p.Ext.IsZero() {
			return p
		}
	}
	return nil
}

// defaultReasonForCode picks a CDAR ReasonCode known to be allowed by
// the BR-FR-CDV-CL-09 schematron rule for the given status code, or
// returns ok=false if the code does not require a reason.
func defaultReasonForCode(code string) (string, bool) {
	// Each picked code is in the BR-FR-CDV-CL-09 allowed list for the
	// status it pairs with (per Annexe A "Tableau des motifs de STATUTS").
	switch code {
	case "206":
		return "AUTRE", true
	case "207", "210":
		return "TX_TVA_ERR", true
	case "208":
		return "JUSTIF_ABS", true
	case "213":
		return "REJ_SEMAN", true
	}
	return "", false
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
			Inboxes: []*org.Inbox{
				{Scheme: "0225", Code: "100000009_PEP"},
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
			Inboxes: []*org.Inbox{
				{Scheme: "0225", Code: "200000008_PEP"},
			},
			Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleBY),
		},
		// Issuer = the buyer-end-party reporting the status (UC1 default
		// pattern: buyer-platform sends 23 acks). For PPF / 305 callers
		// would set this to a WK platform party instead.
		Issuer: &org.Party{
			Name: "ACHETEUR",
			Identities: []*org.Identity{
				{
					Code: "200000008",
					Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
				},
			},
			Inboxes: []*org.Inbox{
				{Scheme: "0225", Code: "200000008_PEP"},
			},
			Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleBY),
		},
		// Recipient = the seller-end-party (counterparty of the buyer
		// issuer). Must carry an inbox per BR-FR-CDV-08.
		Recipient: &org.Party{
			Name: "VENDEUR",
			Identities: []*org.Identity{
				{
					Code: "100000009",
					Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
				},
			},
			Inboxes: []*org.Inbox{
				{Scheme: "0225", Code: "100000009_PEP"},
			},
			Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleSE),
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

	// Codes that require at least one Reason. The schematron restricts
	// the allowed reason codes per status (BR-FR-CDV-CL-09), so we pick a
	// code that's valid for each.
	if reasonCode, ok := defaultReasonForCode(code); ok {
		// Leave Key empty — flow6's reason normaliser fills it from the
		// ReasonCode extension, matching whichever bucket the code rolls
		// up to (AUTRE → other, TX_TVA_ERR → legal, etc.).
		line.Reasons = []*bill.Reason{{
			Ext:         tax.MakeExtensions().Set(flow6.ExtKeyReasonCode, cbc.Code(reasonCode)),
			Description: "Motif " + reasonCode,
		}}
		line.Actions = []*bill.Action{{Key: bill.ActionKeyReissue, Description: "Créer une facture rectificative"}}
	}

	// Code 212 (paid) — attach an MEN amount as a complement so we exercise
	// the characteristic round-trip from the StatusLine level.
	if code == "212" {
		amt := currency.Amount{
			Currency: currency.EUR,
			Value:    num.MakeAmount(50000, 2),
		}
		obj, err := schema.NewObject(&flow6.Characteristic{
			TypeCode: flow6.TypeCodeAmountReceived,
			Amount:   &amt,
		})
		require.NoError(t, err)
		line.Complements = append(line.Complements, obj)
	}

	st.Lines = []*bill.StatusLine{line}
	return st
}

func TestCDARStatusRoundTripPerCode(t *testing.T) {
	for _, code := range allProcessCodes {
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

			// Seller's SIREN is preserved somewhere in the parsed status —
			// either on Supplier (filled by the flow6 normaliser when the
			// envelope is calculated) or on whichever of Issuer/Recipient
			// carries the SE role.
			seller := findSEPartyAcross(out)
			require.NotNil(t, seller, "no SE-roled party in parsed status %s", code)
			found := ""
			for _, id := range seller.Identities {
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

// TestCDARSenderTradeParty checks that NewCDARFromStatusWithSender (and
// the equivalent Convert option) carries the supplied platform identity
// into the CDAR ExchangedDocument/SenderTradeParty slot, while leaving
// IssuerTradeParty and the recipients untouched.
func TestCDARSenderTradeParty(t *testing.T) {
	st := buildSyntheticStatus(t, "205")

	platform := &org.Party{
		Name: "PA-FR",
		Identities: []*org.Identity{
			{
				Code: "9998",
				Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0238"),
			},
		},
		Inboxes: []*org.Inbox{
			{Scheme: "0225", Code: "9998_PEP"},
		},
		Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleWK),
	}

	cdar, err := cii.NewCDARFromStatusWithSender(st, cii.ContextCDARFlow6, platform)
	require.NoError(t, err)
	require.NotNil(t, cdar.ExchangedDocument.SenderTradeParty)
	assert.Equal(t, "PA-FR", cdar.ExchangedDocument.SenderTradeParty.Name)
	require.Len(t, cdar.ExchangedDocument.SenderTradeParty.GlobalIDs, 1)
	assert.Equal(t, "9998", cdar.ExchangedDocument.SenderTradeParty.GlobalIDs[0].Value)
	assert.Equal(t, "WK", cdar.ExchangedDocument.SenderTradeParty.RoleCode)

	// IssuerTradeParty stays as the end-party (Customer, default for ack 23).
	require.NotNil(t, cdar.ExchangedDocument.IssuerTradeParty)
	assert.Equal(t, "ACHETEUR", cdar.ExchangedDocument.IssuerTradeParty.Name)

	// And without the option, sender goes back to bare WK.
	bare, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
	require.NoError(t, err)
	require.NotNil(t, bare.ExchangedDocument.SenderTradeParty)
	assert.Empty(t, bare.ExchangedDocument.SenderTradeParty.Name)
	assert.Empty(t, bare.ExchangedDocument.SenderTradeParty.GlobalIDs)
	assert.Equal(t, "WK", bare.ExchangedDocument.SenderTradeParty.RoleCode)
}

// TestCDARSchematron generates a CDAR for each process code and validates it
// against the live Phive instance using the fr.ctc:cdar VESID. Requires
// `-validate` and a Phive gRPC service reachable on localhost:9091.
func TestCDARSchematron(t *testing.T) {
	if !*validate {
		t.Skip("requires -validate flag and a running Phive gRPC service")
	}

	conn, err := grpc.NewClient("localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close() //nolint:errcheck
	pc := phive.NewValidationServiceClient(conn)

	for _, code := range allProcessCodes {
		t.Run("CDV-"+code, func(t *testing.T) {
			st := buildSyntheticStatus(t, code)
			cdar, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
			require.NoError(t, err)
			data, err := cdar.Bytes()
			require.NoError(t, err)

			resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
				Vesid:      cii.ContextCDARFlow6.VESID,
				XmlContent: data,
			})
			require.NoError(t, err)
			out, _ := json.MarshalIndent(resp.Results, "", "  ")
			require.True(t, resp.Success,
				"CDAR for code %s failed Phive validation: %s", code, string(out))
		})
	}
}
