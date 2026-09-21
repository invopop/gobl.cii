package cii_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/phorm"

	cii "github.com/invopop/gobl.cii"
)

// franceInvoiceProbe pins one Flow 2 invoice fixture to the CII context it
// must validate against. Treat warnings as errors — the French CTC
// schematrons emit warnings that future releases promote to errors, so a
// clean run must have zero of both (mirrors cdar_probe_test.go).
type franceInvoiceProbe struct {
	name    string
	dir     string
	file    string
	context cii.Context
}

// franceInvoiceProbes drives TestProbeFranceInvoices over the same fixtures
// that feed TestConvertToInvoice, but through the warnings-as-errors gate.
var franceInvoiceProbes = []franceInvoiceProbe{
	{"CIUS/380", dirFRCIUS, fixtureInvoiceStandard, cii.ContextPeppolFranceCIUSV1},
	{"CIUS/381", dirFRCIUS, fixtureCreditNote, cii.ContextPeppolFranceCIUSV1},
	{"FacturX/380", dirFRFacturX, fixtureInvoiceStandard, cii.ContextPeppolFranceFacturXV1},
	{"FacturX/381", dirFRFacturX, fixtureCreditNote, cii.ContextPeppolFranceFacturXV1},
	{"Extended/380", dirFRExtended, fixtureInvoiceStandard, cii.ContextPeppolFranceExtendedV1},
	{"Extended/381", dirFRExtended, fixtureCreditNote, cii.ContextPeppolFranceExtendedV1},
}

// TestProbeFranceInvoices converts each French CTC invoice fixture and
// pushes the generated CII XML through phive, failing on any error OR
// warning against the dedicated French schematron pinned on the context.
func TestProbeFranceInvoices(t *testing.T) {
	pc := phormClient(t)

	for _, p := range franceInvoiceProbes {
		t.Run(p.name, func(t *testing.T) {
			env := loadEnvelope(t, filepath.Join(p.dir, p.file))
			out, err := cii.ConvertInvoice(env, cii.WithContext(p.context))
			if err != nil {
				t.Fatalf("ConvertInvoice: %v", err)
			}
			data, err := out.Bytes()
			if err != nil {
				t.Fatalf("Bytes: %v", err)
			}
			resp, err := pc.ValidateXml(context.Background(), &phorm.ValidateXmlRequest{
				Vesid:      p.context.VESID,
				XmlContent: data,
			})
			if err != nil {
				t.Fatalf("phorm: %v", err)
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
				t.Errorf("[%s] %s: %d problem(s) (warnings are treated as errors):\n%s",
					p.name, p.context.VESID, len(problems), strings.Join(problems, "\n\n"))
			}
		})
	}
}
