package cii

import (
	"testing"

	"github.com/invopop/gobl.fr.ctc/addon/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCDARXMLGuideline(t *testing.T) {
	t.Run("the output guideline wins", func(t *testing.T) {
		got := cdarXMLGuideline(Context{GuidelineID: "spec", OutputGuidelineID: "written"})
		assert.Equal(t, "written", got)
	})

	t.Run("the spec guideline is the fallback", func(t *testing.T) {
		assert.Equal(t, "spec", cdarXMLGuideline(Context{GuidelineID: "spec"}))
	})

	t.Run("an empty context falls back to Flow 6", func(t *testing.T) {
		got := cdarXMLGuideline(Context{})
		assert.NotEmpty(t, got)
		want := ContextCDARFlow6.OutputGuidelineID
		if want == "" {
			want = ContextCDARFlow6.GuidelineID
		}
		assert.Equal(t, want, got)
	})
}

func TestCDARBusinessProcessID(t *testing.T) {
	t.Run("the output id wins", func(t *testing.T) {
		got := cdarBusinessProcessID(Context{BusinessID: "routing", OutputBusinessID: "REGULATED"})
		assert.Equal(t, "REGULATED", got)
	})

	t.Run("the routing id is the fallback", func(t *testing.T) {
		assert.Equal(t, "routing", cdarBusinessProcessID(Context{BusinessID: "routing"}))
	})

	t.Run("neither set means no element is emitted", func(t *testing.T) {
		assert.Empty(t, cdarBusinessProcessID(Context{}))
	})
}

func TestFirstLineProcessCode(t *testing.T) {
	withStatus := func(code string) *bill.StatusLine {
		return &bill.StatusLine{
			Ext: tax.ExtensionsOf(cbc.CodeMap{flow6.ExtKeyStatus: cbc.Code(code)}),
		}
	}

	t.Run("the first line's status code", func(t *testing.T) {
		st := &bill.Status{Lines: []*bill.StatusLine{withStatus("205"), withStatus("209")}}
		assert.Equal(t, cbc.Code("205"), firstLineProcessCode(st))
	})

	t.Run("nil lines are skipped", func(t *testing.T) {
		st := &bill.Status{Lines: []*bill.StatusLine{nil, withStatus("209")}}
		assert.Equal(t, cbc.Code("209"), firstLineProcessCode(st))
	})

	t.Run("a first line without the extension stops the search", func(t *testing.T) {
		// Only the first non-nil line is consulted, so a later code is not
		// reached.
		st := &bill.Status{Lines: []*bill.StatusLine{{}, withStatus("209")}}
		assert.Equal(t, cbc.CodeEmpty, firstLineProcessCode(st))
	})

	t.Run("no lines at all", func(t *testing.T) {
		assert.Equal(t, cbc.CodeEmpty, firstLineProcessCode(&bill.Status{}))
	})

	t.Run("only nil lines", func(t *testing.T) {
		assert.Equal(t, cbc.CodeEmpty, firstLineProcessCode(&bill.Status{Lines: []*bill.StatusLine{nil}}))
	})
}

func TestPartySIREN(t *testing.T) {
	siren := func(code string) *org.Identity {
		return &org.Identity{
			Code: cbc.Code(code),
			Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: cbc.Code(schemeIDSIREN)}),
		}
	}

	t.Run("the first SIREN identity", func(t *testing.T) {
		p := &org.Party{Identities: []*org.Identity{
			{Code: "no extensions"},
			siren("732829320"),
		}}
		assert.Equal(t, "732829320", partySIREN(p))
	})

	t.Run("an identity under another scheme is not a SIREN", func(t *testing.T) {
		p := &org.Party{Identities: []*org.Identity{{
			Code: "73282932000074",
			Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0009"}),
		}}}
		assert.Empty(t, partySIREN(p))
	})

	t.Run("nil identities are skipped", func(t *testing.T) {
		p := &org.Party{Identities: []*org.Identity{nil, siren("732829320")}}
		assert.Equal(t, "732829320", partySIREN(p))
	})

	t.Run("a party without identities", func(t *testing.T) {
		assert.Empty(t, partySIREN(&org.Party{Name: testPartyName}))
	})

	t.Run("no party at all", func(t *testing.T) {
		assert.Empty(t, partySIREN(nil))
	})
}

func TestCDARString(t *testing.T) {
	c := &CDAR{}
	s, err := c.String()
	require.NoError(t, err)
	assert.Contains(t, s, "<?xml")
}
