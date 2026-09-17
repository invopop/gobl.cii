package cii_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/invopop/phorm"
	"github.com/stretchr/testify/require"
)

// Schematron validation runs against phorm (https://github.com/phax/phorm), the
// standalone validation service that replaced the archived invopop/phive gRPC
// wrapper. Run one locally with:
//
//	docker run -d --name phorm -p 8080:8080 phelger/phorm
//
// then run the suite with -validate. Point PHORM_URL / PHORM_TOKEN elsewhere to
// use a shared instance; the defaults match a stock local container.
const (
	defaultPhormURL = "http://localhost:8080"

	phormURLEnv   = "PHORM_URL"
	phormTokenEnv = "PHORM_TOKEN"
)

// phormClient returns a client for the validation service, skipping the test
// when -validate was not requested. An empty token makes phorm fall back to its
// stock development token.
func phormClient(t *testing.T) *phorm.Client {
	t.Helper()

	if !*validate {
		t.Skip("schematron validation not requested (use -validate)")
	}

	url := os.Getenv(phormURLEnv)
	if url == "" {
		url = defaultPhormURL
	}
	return phorm.New(url, os.Getenv(phormTokenEnv))
}

// levelWarn is the severity phorm reports for a non-fatal finding.
const levelWarn = "WARN"

// finding is a single schematron error or warning.
type finding struct {
	Level string
	Rule  string
	Text  string
}

func (f finding) String() string {
	// Most rule sets already open the message with the rule id.
	if f.Rule != "" && !strings.Contains(f.Text, f.Rule) {
		return f.Level + ": " + f.Text + " [" + f.Rule + "]"
	}
	return f.Level + ": " + f.Text
}

// phormValidate validates the document and returns every finding, errors and
// warnings alike. A document that simply breaks a rule is reported through the
// response; an error means the validation never ran at all, which is fatal to
// the test since nothing was checked.
func phormValidate(t *testing.T, pc *phorm.Client, vesid string, data []byte) []finding {
	t.Helper()

	resp, err := pc.ValidateXml(context.Background(), &phorm.ValidateXmlRequest{
		Vesid:      vesid,
		XmlContent: data,
	})
	require.NoError(t, err, "validation did not run for %s", vesid)

	var out []finding
	for _, r := range resp.Results {
		for _, e := range r.Errors {
			out = append(out, finding{Level: e.Level, Rule: e.ErrorID, Text: e.Message})
		}
		for _, w := range r.Warnings {
			out = append(out, finding{Level: levelWarn, Rule: w.ErrorID, Text: w.Message})
		}
	}
	return out
}

// validateXML pushes the document through phorm and fails the test with every
// error the rule set raises.
func validateXML(t *testing.T, pc *phorm.Client, vesid string, data []byte) {
	t.Helper()

	var errs []string
	for _, f := range phormValidate(t, pc, vesid, data) {
		if f.Level != levelWarn {
			errs = append(errs, f.String())
		}
	}
	if len(errs) > 0 {
		t.Errorf("%s: %d schematron error(s):\n%s", vesid, len(errs), strings.Join(errs, "\n"))
	}
}
