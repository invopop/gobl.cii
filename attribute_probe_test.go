package cii_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/phorm"
	"github.com/stretchr/testify/require"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
)

// attributeProbe pins one invoice fixture to the CII context its BG-32 item
// attributes must validate against.
type attributeProbe struct {
	name    string
	file    string
	context cii.Context
}

// attributeProbes drives TestProbeItemAttributes over one fixture per context
// with a VESID. Warnings count as errors: CII-SR-070 rejects ram:ValueMeasure
// as a warning where the Factur-X and ZUGFeRD schemas reject it outright, so a
// clean run must have zero of both (mirrors invoice_probe_test.go).
var attributeProbes = []attributeProbe{
	{ctxEN16931, "en16931/invoice-minimal.json", cii.ContextEN16931V2017},
	{ctxPeppol, "peppol/invoice-minimal.json", cii.ContextPeppolV3},
	{ctxFacturX, "facturx/invoice-minimal.json", cii.ContextFacturXV1},
	{ctxXRechnung, "xrechnung/invoice-de-es-b2b.json", cii.ContextXRechnungV3},
	{ctxZUGFeRD, "zugferd/standard-invoice.json", cii.ContextZUGFeRDV2},
}

// TestProbeItemAttributes converts an invoice carrying both a text and a
// measured item attribute, then pushes the generated CII XML through phorm.
func TestProbeItemAttributes(t *testing.T) {
	pc := phormClient(t)

	for _, p := range attributeProbes {
		t.Run(p.name, func(t *testing.T) {
			env := loadEnvelope(t, filepath.FromSlash(p.file))
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			weight := num.MakeAmount(25, 1) // 2.5
			inv.Lines[0].Item.Attributes = []*org.Attribute{
				{Label: attrLabelColor, Text: attrValueBlack},
				{Label: attrLabelWeight, Amount: &weight, Unit: org.UnitKilogram},
			}
			require.NoError(t, env.Calculate())

			out, err := cii.ConvertInvoice(env, cii.WithContext(p.context))
			require.NoError(t, err)
			data, err := out.Bytes()
			require.NoError(t, err)

			resp, err := pc.ValidateXml(context.Background(), &phorm.ValidateXmlRequest{
				Vesid:      p.context.VESID,
				XmlContent: data,
			})
			require.NoError(t, err)

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
