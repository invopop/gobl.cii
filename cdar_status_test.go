package cii_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/phive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// statusProcessCodes lists the CDAR ProcessConditionCodes carried by
// bill.Status documents for which flow6 derives a (Status.Type,
// StatusLine.Key) pair. 203 (Mise à disposition) rides (response,
// other) and 209 (Complétée) rides (update, other) — both with the ext
// as source of truth; 211 / 212 are bill.Payment lifecycle codes — see
// the payment tests.
var statusProcessCodes = []string{
	"200", "201", "202", "203", "204", "205",
	"206", "207", "208", "209", "210", "213",
}

// findSEParty returns the parsed status party carrying fr-ctc-flow6-role=SE
// — the Supplier slot under the role-based parse mapping — falling back
// to a role-less SIREN-only party (e.g. from ref.IssuerTradeParty).
func findSEParty(st *bill.Status) *org.Party {
	for _, p := range []*org.Party{st.Supplier, st.Customer} {
		if p == nil {
			continue
		}
		if p.Ext.Get(flow6.ExtKeyRole) == flow6.RoleSeller {
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

func testSIRENParty(name, siren string) *org.Party {
	return &org.Party{
		Name: name,
		Identities: []*org.Identity{
			{
				Code: cbc.Code(siren),
				Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
			},
		},
		Inboxes: []*org.Inbox{
			{Scheme: "0225", Code: cbc.Code(siren + "_PEP")},
		},
	}
}

// buildSyntheticStatus builds a minimal but valid bill.Status for the
// given Flow 6 process code, normalized through the flow6 addon so the
// converter's extension reads are populated.
func buildSyntheticStatus(t *testing.T, code string) *bill.Status {
	t.Helper()

	st := &bill.Status{
		Code:      cbc.Code("STATUS-" + code),
		IssueDate: cal.MakeDate(2025, time.July, 1),
		IssueTime: cal.NewTime(15, 10, 0),
		Supplier:  testSIRENParty("VENDEUR", "100000009"),
		Customer:  testSIRENParty("ACHETEUR", "200000008"),
	}
	st.SetAddons(flow6.V1)

	docDate := cal.MakeDate(2025, time.July, 1)
	line := &bill.StatusLine{
		// Pin the CDAR ProcessConditionCode directly; flow6's reverse
		// mapping derives Status.Type and StatusLine.Key from it.
		Ext: tax.MakeExtensions().Set(flow6.ExtKeyStatus, cbc.Code(code)),
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
			Ext:         tax.MakeExtensions().Set(flow6.ExtKeyReason, cbc.Code(reasonCode)),
			Description: "Motif " + reasonCode,
			// A field-level correction pair, as a buyer's PA would
			// attach to a dispute/rejection: emitted as
			// SpecifiedDocumentCharacteristics on the CDV.
			Faults: []*bill.Fault{
				{
					Code:    "DIV",
					Message: "Taux TVA (BT-152): 10.00%",
					Paths:   []string{"/rsm:CrossIndustryInvoice/ram:ApplicableTradeTax/ram:RateApplicablePercent"},
				},
				{
					Code:    "DVA",
					Message: "Taux TVA (BT-152): 20.00%",
					Paths:   []string{"/rsm:CrossIndustryInvoice/ram:ApplicableTradeTax/ram:RateApplicablePercent"},
				},
			},
		}}
		line.Actions = []*bill.Action{{Key: bill.ActionKeyReissue, Description: "Créer une facture rectificative"}}
	}

	st.Lines = []*bill.StatusLine{line}
	return calculateStatus(t, st)
}

// calculateStatus runs the status through an envelope Calculate so the
// flow6 normalizer fills the derived fields (Type, line keys, roles,
// reason / action extensions).
func calculateStatus(t *testing.T, st *bill.Status) *bill.Status {
	t.Helper()
	env, err := gobl.Envelop(st)
	require.NoError(t, err, "envelop status")
	out, ok := env.Extract().(*bill.Status)
	require.True(t, ok, "extract status")
	return out
}

func TestCDARStatusRoundTripPerCode(t *testing.T) {
	for _, code := range statusProcessCodes {
		t.Run("CDV-"+code, func(t *testing.T) {
			st := buildSyntheticStatus(t, code)
			require.NotEmpty(t, st.Type, "flow6 should derive a status type for %s", code)
			require.NotEmpty(t, st.Lines[0].Key, "flow6 should derive a line key for %s", code)

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

			// Now turn that parsed CDAR back into a *bill.Status and
			// normalize it so the flow6 reverse mapping recovers the
			// GOBL-level fields from the extensions.
			out, err := cii.ParseCDARStatus(data)
			require.NoError(t, err)
			require.NotNil(t, out)
			require.Len(t, out.Lines, 1, "should have one line for %s", code)
			require.Equal(t, cbc.Code(code), out.Lines[0].Ext.Get(flow6.ExtKeyStatus))

			out = calculateStatus(t, out)
			require.Equal(t, st.Type, out.Type, "type mismatch for %s", code)
			require.Equal(t, st.Lines[0].Key, out.Lines[0].Key, "line key mismatch for %s", code)

			// Seller's SIREN is preserved somewhere in the parsed status —
			// either on Supplier (SE-roled trade party) or recovered from
			// the MDT-129 ref.IssuerTradeParty SIREN-only slot.
			seller := findSEParty(out)
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

func TestCDARStatusReasonRoundTrip(t *testing.T) {
	st := buildSyntheticStatus(t, "210")
	require.Len(t, st.Lines, 1)
	require.Len(t, st.Lines[0].Reasons, 1)

	cdar, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
	require.NoError(t, err)
	data, err := cdar.Bytes()
	require.NoError(t, err)

	out, err := cii.ParseCDARStatus(data)
	require.NoError(t, err)
	out = calculateStatus(t, out)

	require.Len(t, out.Lines, 1)
	require.Len(t, out.Lines[0].Reasons, 1)
	reason := out.Lines[0].Reasons[0]
	assert.Equal(t, cbc.Code("TX_TVA_ERR"), reason.Ext.Get(flow6.ExtKeyReason))
	assert.Equal(t, "Motif TX_TVA_ERR", reason.Description)
	assert.NotEmpty(t, reason.Key, "flow6 should recover the reason bucket from the code")

	require.Len(t, out.Lines[0].Actions, 1)
	action := out.Lines[0].Actions[0]
	assert.Equal(t, cbc.Code("NIN"), action.Ext.Get(flow6.ExtKeyAction))
	assert.Equal(t, bill.ActionKeyReissue, action.Key)
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
		Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RolePlatform),
	}

	cdar, err := cii.NewCDARFromStatusWithSender(st, cii.ContextCDARFlow6, platform)
	require.NoError(t, err)
	require.NotNil(t, cdar.ExchangedDocument.SenderTradeParty)
	assert.Equal(t, "PA-FR", cdar.ExchangedDocument.SenderTradeParty.Name)
	require.Len(t, cdar.ExchangedDocument.SenderTradeParty.GlobalIDs, 1)
	assert.Equal(t, "9998", cdar.ExchangedDocument.SenderTradeParty.GlobalIDs[0].Value)
	assert.Equal(t, "WK", cdar.ExchangedDocument.SenderTradeParty.RoleCode)

	// IssuerTradeParty is the Customer — 205 is a buyer-issued code.
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

// TestCDARStatusFromToRouting covers the envelope Head.From / Head.To
// routing URIs steering the CDAR issuer / recipient slots on
// business-issued (23-phase) codes.
func TestCDARStatusFromToRouting(t *testing.T) {
	const (
		supplierURI = cbc.URI("iso6523-actorid-upis::0225:100000009")
		customerURI = cbc.URI("iso6523-actorid-upis::0225:200000008")
	)

	// envelopWithEndpoints re-envelops a calculated status after giving
	// both parties an endpoint, so Envelope.Calculate auto-fills
	// Head.From / Head.To from the document's type semantics.
	envelopWithEndpoints := func(t *testing.T, code string) *gobl.Envelope {
		t.Helper()
		st := buildSyntheticStatus(t, code)
		st.Supplier.Endpoints = []*org.Endpoint{{URI: supplierURI}}
		st.Customer.Endpoints = []*org.Endpoint{{URI: customerURI}}
		env, err := gobl.Envelop(st)
		require.NoError(t, err)
		return env
	}

	convert := func(t *testing.T, env *gobl.Envelope) *cii.CDAR {
		t.Helper()
		out, err := cii.Convert(env, cii.WithContext(cii.ContextCDARFlow6))
		require.NoError(t, err)
		cdar, ok := out.(*cii.CDAR)
		require.True(t, ok)
		return cdar
	}

	t.Run("205: auto From/To matches the buyer-issued default", func(t *testing.T) {
		env := envelopWithEndpoints(t, "205")
		// response → Customer issues, Supplier receives.
		assert.Equal(t, customerURI, env.Head.From)
		assert.Equal(t, supplierURI, env.Head.To)
		cdar := convert(t, env)
		assert.Equal(t, "ACHETEUR", cdar.ExchangedDocument.IssuerTradeParty.Name)
		require.Len(t, cdar.ExchangedDocument.RecipientTradeParties, 1)
		assert.Equal(t, "VENDEUR", cdar.ExchangedDocument.RecipientTradeParties[0].Name)
	})

	t.Run("205: operator-set From/To overrides the default", func(t *testing.T) {
		env := envelopWithEndpoints(t, "205")
		env.Head.From, env.Head.To = supplierURI, customerURI
		cdar := convert(t, env)
		assert.Equal(t, "VENDEUR", cdar.ExchangedDocument.IssuerTradeParty.Name)
		require.Len(t, cdar.ExchangedDocument.RecipientTradeParties, 1)
		assert.Equal(t, "ACHETEUR", cdar.ExchangedDocument.RecipientTradeParties[0].Name)
	})

	t.Run("209: auto From/To follows the supplier-side update type", func(t *testing.T) {
		// 209 (Complétée) is supplier-issued and rides the `update`
		// status type, so the envelope auto-derives From at the
		// supplier — no explicit pinning needed.
		env := envelopWithEndpoints(t, "209")
		assert.Equal(t, supplierURI, env.Head.From)
		assert.Equal(t, customerURI, env.Head.To)
		cdar := convert(t, env)
		assert.Equal(t, "VENDEUR", cdar.ExchangedDocument.IssuerTradeParty.Name)
		require.Len(t, cdar.ExchangedDocument.RecipientTradeParties, 1)
		assert.Equal(t, "ACHETEUR", cdar.ExchangedDocument.RecipientTradeParties[0].Name)
	})

	t.Run("209: no From/To falls back to the supplier-issued heuristic", func(t *testing.T) {
		st := buildSyntheticStatus(t, "209")
		// Strip the routing addresses so neither flow6's inbox→endpoint
		// derivation nor the envelope's auto From/To can kick in.
		for _, p := range []*org.Party{st.Supplier, st.Customer} {
			p.Inboxes = nil
			p.Endpoints = nil
		}
		env, err := gobl.Envelop(st)
		require.NoError(t, err)
		assert.Empty(t, env.Head.From, "no endpoints, no auto From")
		cdar := convert(t, env)
		assert.Equal(t, "VENDEUR", cdar.ExchangedDocument.IssuerTradeParty.Name)
	})

	t.Run("201: platform-issued code ignores From/To", func(t *testing.T) {
		env := envelopWithEndpoints(t, "201")
		env.Head.From, env.Head.To = customerURI, supplierURI
		cdar := convert(t, env)
		// 305-phase issuer stays the sending platform (bare WK).
		assert.Equal(t, "WK", cdar.ExchangedDocument.SenderTradeParty.RoleCode)
		if itp := cdar.ExchangedDocument.IssuerTradeParty; itp != nil {
			assert.NotEqual(t, "ACHETEUR", itp.Name)
		}
	})
}

// buildSyntheticPayment builds a minimal bill.Payment receipt (CDAR 212,
// Encaissée) referencing an invoice, normalized through the flow6 addon.
// due, when non-zero, models a partial cash-in with a remainder.
func buildSyntheticPayment(t *testing.T, due num.Amount) *bill.Payment {
	t.Helper()

	docDate := cal.MakeDate(2025, time.July, 1)
	line := &bill.PaymentLine{
		Document: &org.DocumentRef{
			Code:      cbc.Code("F202500003"),
			IssueDate: &docDate,
			Type:      "380",
		},
		Amount: num.MakeAmount(50000, 2),
	}
	if !due.IsZero() {
		line.Due = &due
	}

	pmt := &bill.Payment{
		Type:      bill.PaymentTypeReceipt,
		Code:      cbc.Code("PAY-212"),
		IssueDate: cal.MakeDate(2025, time.July, 2),
		Currency:  currency.EUR,
		Supplier:  testSIRENParty("VENDEUR", "100000009"),
		Customer:  testSIRENParty("ACHETEUR", "200000008"),
		Lines:     []*bill.PaymentLine{line},
		Methods:   []*pay.Record{{Key: pay.MeansKeyCreditTransfer, Amount: num.MakeAmount(50000, 2)}},
	}
	pmt.SetAddons(flow6.V1)

	env, err := gobl.Envelop(pmt)
	require.NoError(t, err, "envelop payment")
	out, ok := env.Extract().(*bill.Payment)
	require.True(t, ok, "extract payment")
	return out
}

func TestCDARPaymentRoundTrip(t *testing.T) {
	pmt := buildSyntheticPayment(t, num.Amount{})
	require.Equal(t, cbc.Code("212"), pmt.Ext.Get(flow6.ExtKeyStatus),
		"flow6 should derive 212 for a receipt")

	cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
	require.NoError(t, err)
	require.NotNil(t, cdar)

	// The receipt is declared by the supplier towards the customer.
	require.NotNil(t, cdar.ExchangedDocument.IssuerTradeParty)
	assert.Equal(t, "VENDEUR", cdar.ExchangedDocument.IssuerTradeParty.Name)
	require.Len(t, cdar.ExchangedDocument.RecipientTradeParties, 1)
	assert.Equal(t, "ACHETEUR", cdar.ExchangedDocument.RecipientTradeParties[0].Name)

	data, err := cdar.Bytes()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	out, err := cii.ParseCDARPayment(data)
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, bill.PaymentTypeReceipt, out.Type)
	assert.Equal(t, cbc.Code("212"), out.Ext.Get(flow6.ExtKeyStatus))
	assert.Equal(t, currency.EUR, out.Currency)
	require.Len(t, out.Lines, 1)
	assert.Equal(t, "500.00", out.Lines[0].Amount.String())
	require.NotNil(t, out.Lines[0].Document)
	assert.Equal(t, cbc.Code("F202500003"), out.Lines[0].Document.Code)

	// Parties round-trip by role.
	require.NotNil(t, out.Supplier)
	assert.Equal(t, "VENDEUR", out.Supplier.Name)
	require.NotNil(t, out.Customer)
	assert.Equal(t, "ACHETEUR", out.Customer.Name)
}

func TestCDARPaymentPartialRoundTrip(t *testing.T) {
	pmt := buildSyntheticPayment(t, num.MakeAmount(25000, 2))

	cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
	require.NoError(t, err)
	data, err := cdar.Bytes()
	require.NoError(t, err)

	out, err := cii.ParseCDARPayment(data)
	require.NoError(t, err)
	require.Len(t, out.Lines, 1)
	assert.Equal(t, "500.00", out.Lines[0].Amount.String(), "MEN amount")
	require.NotNil(t, out.Lines[0].Due, "RAP remainder should round-trip")
	assert.Equal(t, "250.00", out.Lines[0].Due.String())
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

	for _, code := range statusProcessCodes {
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

	t.Run("CDV-212-payment", func(t *testing.T) {
		pmt := buildSyntheticPayment(t, num.MakeAmount(25000, 2))
		cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
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
			"CDAR for payment 212 failed Phive validation: %s", string(out))
	})
}
