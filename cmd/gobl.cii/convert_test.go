package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/flimzy/testy"
)

const (
	cmdConvert = "convert"
	testXML    = "../../test/data/parse/CII_example1.xml"
	testJSON   = "../../test/data/convert/peppol/invoice-complete.json"
)

func TestConvertCommand(t *testing.T) {
	t.Run("CII XML input converts to a GOBL envelope", func(t *testing.T) {
		out := runConvert(t, testXML)
		env := new(gobl.Envelope)
		require.NoError(t, json.Unmarshal([]byte(out), env))
		_, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok, "envelope should wrap a bill.Invoice")
	})

	t.Run("GOBL JSON input converts to a CII document", func(t *testing.T) {
		out := runConvert(t, testJSON)
		assert.Contains(t, out, "<rsm:CrossIndustryInvoice")
	})

	t.Run("with format", func(t *testing.T) {
		out := runConvert(t, testJSON, "--format", "cii+peppol")
		assert.Contains(t, out, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
	})
}

// runConvert runs `gobl.cii convert <infile>`, capturing stdout.
func runConvert(t *testing.T, infile string, flags ...string) string {
	t.Helper()
	cmd := root().cmd()
	cmd.SetArgs(append([]string{cmdConvert, infile}, flags...))
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())
	return out.String()
}

func TestConvertCommandErrors(t *testing.T) {
	tests := map[string]struct {
		args  []string
		stdin string
		err   string
	}{
		"no arguments":    {args: []string{cmdConvert}, err: "expected one or two arguments"},
		"too many":        {args: []string{cmdConvert, "a", "b", "c"}, err: "expected one or two arguments"},
		"unknown format":  {args: []string{cmdConvert, "-", "--format", "cii+unknown"}, err: "unsupported format"},
		"missing file":    {args: []string{cmdConvert, "does-not-exist.xml"}, err: "no such file"},
		"not an envelope": {args: []string{cmdConvert, "-"}, stdin: `{"head":1}`, err: "parsing input as GOBL Envelope"},
		"no document":     {args: []string{cmdConvert, "-"}, stdin: `{}`, err: "building cii+en16931 document"},
		"not cii":         {args: []string{cmdConvert, "-"}, stdin: `<Foo/>`, err: "converting CII to GOBL"},
		"CDAR": {
			args:  []string{cmdConvert, "-"},
			stdin: `<CrossDomainAcknowledgementAndResponse xmlns="urn:un:unece:uncefact:data:standard:CrossDomainAcknowledgementAndResponse:100"/>`,
			err:   "unsupported document type",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := root().cmd()
			cmd.SetArgs(tt.args)
			cmd.SetIn(strings.NewReader(tt.stdin))
			cmd.SetOut(new(bytes.Buffer))
			err := cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestConvertCommandOutputFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.json")
	cmd := root().cmd()
	cmd.SetArgs([]string{cmdConvert, testXML, out})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://gobl.org/draft-0/envelope")
}

func TestConvertCommandOutputError(t *testing.T) {
	out := filepath.Join(t.TempDir(), "missing", "out.json")
	cmd := root().cmd()
	cmd.SetArgs([]string{cmdConvert, testXML, out})
	assert.Error(t, cmd.Execute())
}

func TestRun(t *testing.T) {
	args := os.Args
	t.Cleanup(func() { os.Args = args })
	os.Args = []string{name, "version"}
	var err error
	testy.RedirIO(nil, func() {
		err = run()
	})
	assert.NoError(t, err)
}

func TestRunEnvError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".env"), 0o755))
	t.Chdir(dir)
	assert.ErrorContains(t, run(), "failed to load .env file")
}
