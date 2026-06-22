package cii_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
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
	build   func() any
}

// minPlatformWK is the platform identity carried as the CDAR
// SenderTradeParty (and issuer fallback) on PPF (305) CDVs.
func minPlatformWK() *org.Party {
	return &org.Party{
		Name: "PA-FR",
		Identities: []*org.Identity{{
			Code: "9998",
			Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, "0238"),
		}},
		Inboxes: []*org.Inbox{{Scheme: "0225", Code: "9998_PEP"}},
		Ext:     tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RolePlatform),
	}
}

// minStatus builds a minimal spec-conformant status for the given
// process code without normalization side effects beyond Calculate.
func minStatus(t *testing.T, code string) *bill.Status {
	t.Helper()
	return buildSyntheticStatus(t, code)
}

func minDepositedStatus(t *testing.T) *bill.Status {
	t.Helper()
	st := &bill.Status{
		Code:      "STATUS-200",
		IssueDate: cal.MakeDate(2026, time.May, 2),
		IssueTime: cal.NewTime(15, 10, 0),
		Supplier:  testSIRENParty("VENDEUR", "100000009"),
		Customer:  testSIRENParty("ACHETEUR", "200000008"),
	}
	st.SetAddons(flow6.V1)
	docDate := cal.MakeDate(2026, time.April, 15)
	st.Lines = []*bill.StatusLine{{
		Ext: tax.MakeExtensions().Set(flow6.ExtKeyStatus, "200"),
		Doc: &org.DocumentRef{
			Code:      cbc.Code("F-2026-0042"),
			IssueDate: &docDate,
			Type:      "380",
		},
	}}
	return calculateStatus(t, st)
}

// validateAndProbe envelopes the document, runs GOBL Calculate +
// Validate, converts it with the given context and pushes the XML
// through phive, returning any errors AND warnings as problems.
func validateAndProbe(t *testing.T, pc phive.ValidationServiceClient, doc any, ctx cii.Context, opts ...cii.Option) []string {
	t.Helper()
	env, err := gobl.Envelop(doc)
	if err != nil {
		t.Fatalf("Envelop: %v", err)
	}
	if err := env.Calculate(); err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if err := env.Validate(); err != nil {
		t.Fatalf("GOBL Validate: %v", err)
	}
	opts = append([]cii.Option{cii.WithContext(ctx)}, opts...)
	out, err := cii.Convert(env, opts...)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	cdar, ok := out.(*cii.CDAR)
	if !ok {
		t.Fatalf("Convert returned %T, want *cii.CDAR", out)
	}
	data, err := cdar.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
		Vesid:      ctx.VESID,
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
	return problems
}

func phiveClient(t *testing.T) phive.ValidationServiceClient {
	t.Helper()
	conn, err := grpc.NewClient("localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return phive.NewValidationServiceClient(conn)
}

// TestProbeAllProcessCodes pushes every supported Flow 6 process code
// through both CDAR contexts (treatment 23 and PPF 305) and reports any
// phive errors or warnings — warnings are treated as errors. Used to
// surface schematron constraints that flow6 should enforce upstream.
// Status codes ride bill.Status; 211 / 212 ride bill.Payment.
func TestProbeAllProcessCodes(t *testing.T) {
	if !*validate {
		t.Skip("requires -validate")
	}
	pc := phiveClient(t)

	type combo struct {
		ctxName string
		ctx     cii.Context
	}
	contexts := []combo{
		{"23", cii.ContextCDARFlow6},
		{"305", cii.ContextCDARFlow6PPF},
	}

	for _, code := range statusProcessCodes {
		for _, c := range contexts {
			t.Run("CDV-"+code+"/"+c.ctxName, func(t *testing.T) {
				st := minStatus(t, code)
				var opts []cii.Option
				if c.ctx.GuidelineID == cii.CDARGuidelinePPF {
					// 305: identified WK platform as sender; the converter
					// injects the PPF recipient itself.
					opts = append(opts, cii.WithSenderTradeParty(minPlatformWK()))
				}
				if problems := validateAndProbe(t, pc, st, c.ctx, opts...); len(problems) > 0 {
					t.Errorf("[%s/%s] %d problem(s) (warnings are treated as errors):\n%s",
						code, c.ctxName, len(problems), strings.Join(problems, "\n\n"))
				}
			})
		}
	}

	for _, code := range []string{"211", "212"} {
		for _, c := range contexts {
			t.Run("CDV-"+code+"/"+c.ctxName, func(t *testing.T) {
				pmt := buildSyntheticPayment(t, num.Amount{})
				if code == "211" {
					pmt.Type = bill.PaymentTypeAdvice
					pmt.Ext = pmt.Ext.Delete(flow6.ExtKeyStatus).Delete(flow6.ExtKeyCondition)
				}
				var opts []cii.Option
				if c.ctx.GuidelineID == cii.CDARGuidelinePPF {
					opts = append(opts, cii.WithSenderTradeParty(minPlatformWK()))
				}
				if problems := validateAndProbe(t, pc, pmt, c.ctx, opts...); len(problems) > 0 {
					t.Errorf("[%s/%s] %d problem(s) (warnings are treated as errors):\n%s",
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
	pc := phiveClient(t)

	cases := []probeCase{
		{"205-Approved", cii.ContextCDARFlow6, func() any {
			return minStatus(t, "205")
		}},
		{"207-Disputed", cii.ContextCDARFlow6, func() any {
			return minStatus(t, "207")
		}},
		{"210-Refused", cii.ContextCDARFlow6, func() any {
			return minStatus(t, "210")
		}},
		{"212-Paid", cii.ContextCDARFlow6, func() any {
			return buildSyntheticPayment(t, num.Amount{})
		}},
		{"212-PaidPartial", cii.ContextCDARFlow6, func() any {
			return buildSyntheticPayment(t, num.MakeAmount(25000, 2))
		}},
		{"200-Deposited", cii.ContextCDARFlow6PPF, func() any {
			return minDepositedStatus(t)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var opts []cii.Option
			if tc.context.GuidelineID == cii.CDARGuidelinePPF {
				opts = append(opts, cii.WithSenderTradeParty(minPlatformWK()))
			}
			if problems := validateAndProbe(t, pc, tc.build(), tc.context, opts...); len(problems) > 0 {
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
			"missing-supplier",
			func(st *bill.Status) { st.Supplier = nil },
			"supplier",
		},
		{
			"missing-customer",
			func(st *bill.Status) { st.Customer = nil },
			"customer",
		},
		{
			"customer-without-inbox",
			func(st *bill.Status) { st.Customer.Inboxes = nil },
			"inbox",
		},
		{
			"reason-not-allowed-for-status",
			func(st *bill.Status) {
				// 207 disputed accepts AUTRE/TX_TVA_ERR/…; REJ_SEMAN is 213-only.
				st.Lines[0].Reasons = []*bill.Reason{{
					Ext: tax.MakeExtensions().Set(flow6.ExtKeyReason, "REJ_SEMAN"),
				}}
			},
			"fr-ctc-flow6-reason for status code 207",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := buildSyntheticStatus(t, "207")
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
	st := buildSyntheticStatus(t, "210")
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
