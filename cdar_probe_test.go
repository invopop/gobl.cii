package cii_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/invopop/gobl"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// probeCase defines one minimal-status fixture to drive through Convert
// and then Phive. Treat warnings as errors — they will become errors in
// future schematron releases.
type probeCase struct {
	name    string
	context cii.Context
	build   func() *bill.Status
}

// minSupplier returns the minimum spec-conformant supplier party for the
// probe: SIREN identity (scheme 0002) and an electronic address inbox so
// BR-FR-CDV-08 is satisfied when the party lands in a recipient slot.
func minSupplier() *org.Party {
	return &org.Party{
		Name: "VENDEUR",
		Identities: []*org.Identity{{
			Code: "100000009",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
		}},
		Inboxes: []*org.Inbox{{Scheme: "0225", Code: "100000009_PEP"}},
	}
}

// minCustomer is the buyer counterpart.
func minCustomer() *org.Party {
	return &org.Party{
		Name: "ACHETEUR",
		Identities: []*org.Identity{{
			Code: "200000008",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
		}},
		Inboxes: []*org.Inbox{{Scheme: "0225", Code: "200000008_PEP"}},
	}
}

// minIssuerBY returns a buyer-end-party Issuer (used as MDG-16 on
// ack 23 buyer-side responses).
func minIssuerBY() *org.Party {
	return &org.Party{
		Name: "ACHETEUR",
		Identities: []*org.Identity{{
			Code: "200000008",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
		}},
		Inboxes: []*org.Inbox{{Scheme: "0225", Code: "200000008_PEP"}},
		Ext:     tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleBY),
	}
}

// minRecipientSE returns the seller-end-party recipient (MDG-23) with
// the required URIID per BR-FR-CDV-08.
func minRecipientSE() *org.Party {
	return &org.Party{
		Name: "VENDEUR",
		Identities: []*org.Identity{{
			Code: "100000009",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0002"),
		}},
		Inboxes: []*org.Inbox{{Scheme: "0225", Code: "100000009_PEP"}},
		Ext:     tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleSE),
	}
}

// ppfRecipient returns the canonical PPF identity used as MDG-23 on
// ack 305 messages. Lives in the test file because the French app
// owns the PPF identity in production code.
func ppfRecipient() *org.Party {
	return &org.Party{
		Name: "PPF",
		Identities: []*org.Identity{{
			Code: "9998",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0238"),
		}},
		Ext: tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleDFH),
	}
}

// minIssuerWK is the platform identity used as MDG-16 on PPF (305) CDVs.
func minIssuerWK() *org.Party {
	return &org.Party{
		Name: "PA-FR",
		Identities: []*org.Identity{{
			Code: "9998",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0238"),
		}},
		Inboxes: []*org.Inbox{{Scheme: "0225", Code: "9998_PEP"}},
		Ext:     tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleWK),
	}
}

func minPaidStatus() *bill.Status {
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "STATUS-212",
		IssueDate: cal.MakeDate(2026, time.May, 2),
		IssueTime: cal.NewTime(15, 10, 0),
		Supplier:  minSupplier(),
		Customer:  minCustomer(),
		Issuer:    minIssuerBY(),
		Recipient: minRecipientSE(),
	}
	st.SetAddons(flow6.V1)

	docDate := cal.MakeDate(2026, time.April, 15)
	amt := currency.Amount{Currency: currency.EUR, Value: num.MakeAmount(120000, 2)}
	cobj, err := schema.NewObject(&flow6.Characteristic{
		TypeCode: flow6.TypeCodeAmountReceived,
		Amount:   &amt,
	})
	if err != nil {
		panic(err)
	}
	st.Lines = []*bill.StatusLine{{
		Key: bill.StatusEventPaid,
		Doc: &org.DocumentRef{
			Code:      cbc.Code("F-2026-0042"),
			IssueDate: &docDate,
			Type:      "380",
		},
		Complements: []*schema.Object{cobj},
	}}
	return st
}

func minDepositedStatus() *bill.Status {
	st := &bill.Status{
		Type:      bill.StatusTypeUpdate,
		Code:      "STATUS-200",
		IssueDate: cal.MakeDate(2026, time.May, 2),
		IssueTime: cal.NewTime(15, 10, 0),
		Supplier:  minSupplier(),
		Customer:  minCustomer(),
		Issuer:    minIssuerWK(),
		Recipient: ppfRecipient(),
	}
	st.SetAddons(flow6.V1)
	docDate := cal.MakeDate(2026, time.April, 15)
	st.Lines = []*bill.StatusLine{{
		Key: bill.StatusEventIssued,
		Doc: &org.DocumentRef{
			Code:      cbc.Code("F-2026-0042"),
			IssueDate: &docDate,
			Type:      "380",
		},
	}}
	return st
}

func minDisputedStatus() *bill.Status {
	st := minPaidStatus()
	st.Code = "STATUS-207"
	st.Lines[0].Key = flow6.StatusEventDisputed
	st.Lines[0].Complements = nil
	st.Lines[0].Reasons = []*bill.Reason{{
		Key:         bill.ReasonKeyLegal,
		Ext:         tax.MakeExtensions().Set(flow6.ExtKeyReasonCode, cbc.Code("TX_TVA_ERR")),
		Description: "Taux de TVA erroné",
	}}
	st.Lines[0].Actions = []*bill.Action{{
		Key:         bill.ActionKeyReissue,
		Description: "Créer une facture rectificative",
	}}
	return st
}

// TestProbeAllProcessCodes pushes every Flow 6 process code through both
// CDAR contexts (treatment 23 and PPF 305) and reports any phive errors
// or warnings. Used to surface schematron constraints that flow6 should
// enforce upstream.
func TestProbeAllProcessCodes(t *testing.T) {
	if !*validate {
		t.Skip("requires -validate")
	}
	conn, err := grpc.NewClient("localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close() //nolint:errcheck
	pc := phive.NewValidationServiceClient(conn)

	type combo struct {
		ctxName string
		ctx     cii.Context
	}
	contexts := []combo{
		{"23", cii.ContextCDARFlow6},
		{"305", cii.ContextCDARFlow6PPF},
	}

	for _, code := range allProcessCodes {
		for _, c := range contexts {
			t.Run("CDV-"+code+"/"+c.ctxName, func(t *testing.T) {
				st := buildSyntheticStatus(t, code)
				if c.ctx.GuidelineID == cii.CDARGuidelinePPF {
					// 305: Issuer must be WK, Recipient must be the PPF.
					st.Issuer = minIssuerWK()
					st.Recipient = ppfRecipient()
				}
				env, err := gobl.Envelop(st)
				if err != nil {
					t.Fatalf("Envelop: %v", err)
				}
				if err := env.Calculate(); err != nil {
					t.Fatalf("Calculate: %v", err)
				}
				if err := env.Validate(); err != nil {
					t.Fatalf("GOBL Validate: %v", err)
				}
				out, err := cii.Convert(env, cii.WithContext(c.ctx))
				if err != nil {
					t.Fatalf("Convert: %v", err)
				}
				cdar := out.(*cii.CDAR)
				data, _ := cdar.Bytes()
				resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
					Vesid:      c.ctx.VESID,
					XmlContent: data,
				})
				if err != nil {
					t.Fatalf("phive: %v", err)
				}
				var problems []string
				for _, r := range resp.Results {
					for _, e := range r.Errors {
						problems = append(problems, "ERROR: "+e.Message)
					}
					for _, w := range r.Warnings {
						problems = append(problems, "WARN:  "+w.Message)
					}
				}
				if len(problems) > 0 {
					t.Errorf("[%s/%s] %d problem(s):\n%s",
						code, c.ctxName, len(problems), strings.Join(problems, "\n\n"))
				}
			})
		}
	}
}

func TestProbeMinimumStatuses(t *testing.T) {
	if !*validate {
		t.Skip("requires -validate and a running Phive gRPC service")
	}
	conn, err := grpc.NewClient("localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close() //nolint:errcheck
	pc := phive.NewValidationServiceClient(conn)

	cases := []probeCase{
		{"205-Approved", cii.ContextCDARFlow6, func() *bill.Status {
			st := minPaidStatus()
			st.Code = "STATUS-205"
			st.Lines[0].Key = bill.StatusEventAccepted
			st.Lines[0].Complements = nil
			return st
		}},
		{"207-Disputed", cii.ContextCDARFlow6, minDisputedStatus},
		{"212-Paid", cii.ContextCDARFlow6, minPaidStatus},
		{"200-Deposited", cii.ContextCDARFlow6PPF, minDepositedStatus},
		{"211-PaidUpdate", cii.ContextCDARFlow6PPF, func() *bill.Status {
			st := minDepositedStatus()
			st.Code = "STATUS-211"
			st.Type = bill.StatusTypeUpdate
			st.Lines[0].Key = bill.StatusEventPaid
			return st
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := tc.build()
			env, err := gobl.Envelop(st)
			if err != nil {
				t.Fatalf("Envelop: %v", err)
			}
			if err := env.Calculate(); err != nil {
				t.Fatalf("Calculate: %v", err)
			}
			if err := env.Validate(); err != nil {
				t.Fatalf("GOBL Validate: %v", err)
			}

			out, err := cii.Convert(env, cii.WithContext(tc.context))
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			cdar := out.(*cii.CDAR)
			data, err := cdar.Bytes()
			if err != nil {
				t.Fatalf("Bytes: %v", err)
			}

			resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
				Vesid:      tc.context.VESID,
				XmlContent: data,
			})
			if err != nil {
				t.Fatalf("phive: %v", err)
			}

			var problems []string
			for _, r := range resp.Results {
				for _, e := range r.Errors {
					problems = append(problems, "ERROR: "+e.Message)
				}
				for _, w := range r.Warnings {
					problems = append(problems, "WARN:  "+w.Message)
				}
			}
			if len(problems) > 0 {
				t.Errorf("[%s] %d problem(s) (warnings are treated as errors):\n%s",
					tc.name, len(problems), strings.Join(problems, "\n\n"))
			}
		})
	}
}

// TestProbeInvalidStatusesRejectedByGOBL: every input that would trigger
// a phive warning/error should be rejected at the GOBL-Validate level
// upstream — these tests pin the contract.
func TestProbeInvalidStatusesRejectedByGOBL(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*bill.Status)
		wantErr string
	}{
		{
			"missing-issuer",
			func(st *bill.Status) { st.Issuer = nil },
			"issuer is required",
		},
		{
			"missing-recipient",
			func(st *bill.Status) { st.Recipient = nil },
			"recipient is required",
		},
		{
			"recipient-without-inbox",
			func(st *bill.Status) { st.Recipient.Inboxes = nil },
			"electronic address",
		},
		// Note: clearing Issuer.Ext on a buyer-issued / seller-issued
		// status no longer fails — the flow6 normaliser fills the role
		// from the (line.Key, st.Type) → side mapping. To exercise the
		// "role required" branch the test would need a platform-issued
		// code (200/201/202/203/213) where no derivation applies.
		{
			"reason-not-allowed-for-status",
			func(st *bill.Status) {
				// 207 disputed accepts AUTRE/TX_TVA_ERR; REJ_SEMAN is 213-only.
				st.Lines[0].Key = flow6.StatusEventDisputed
				st.Type = bill.StatusTypeResponse
				st.Lines[0].Reasons = []*bill.Reason{{
					Ext: tax.MakeExtensions().Set(flow6.ExtKeyReasonCode, "REJ_SEMAN"),
				}}
				st.Lines[0].Complements = nil
			},
			"each Reason's fr-ctc-reason-code must be allowed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := minPaidStatus()
			tc.mutate(st)
			env, err := gobl.Envelop(st)
			if err != nil {
				t.Fatalf("Envelop: %v", err)
			}
			_ = env.Calculate()
			err = env.Validate()
			if err == nil {
				t.Fatalf("expected validation error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// noisily emit the generated XML for a given probe case — handy when
// chasing a specific schematron complaint.
func TestProbeDumpXML(t *testing.T) {
	if testing.Short() {
		t.Skip("set -v")
	}
	st := minPaidStatus()
	env, err := gobl.Envelop(st)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Calculate(); err != nil {
		t.Fatal(err)
	}
	if err := env.Validate(); err != nil {
		t.Fatalf("GOBL Validate: %v", err)
	}
	out, err := cii.Convert(env, cii.WithContext(cii.ContextCDARFlow6))
	if err != nil {
		t.Fatal(err)
	}
	cdar := out.(*cii.CDAR)
	data, _ := cdar.Bytes()
	t.Logf("\n%s", string(data))

	stJSON, _ := json.MarshalIndent(st, "", "  ")
	t.Logf("\nstatus:\n%s", string(stJSON))
}
