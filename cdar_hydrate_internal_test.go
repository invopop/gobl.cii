package cii

import (
	"testing"

	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func sirenParty(siren string) *org.Party {
	return &org.Party{Identities: []*org.Identity{{
		Code: cbc.Code(siren),
		Ext:  tax.MakeExtensions().Set(iso.ExtKeySchemeID, schemeIDSIREN),
	}}}
}

func TestParticipantInbox(t *testing.T) {
	cases := []struct {
		in           string
		scheme, code string
		nil          bool
	}{
		{"0225:698680774", "0225", "698680774", false},
		{"iso6523-actorid-upis::0225:698680774_STATUTS", "0225", "698680774_STATUTS", false},
		{"", "", "", true},
		{"no-separator", "", "", true},
	}
	for _, c := range cases {
		ib := participantInbox(cbc.URI(c.in))
		if c.nil {
			assert.Nil(t, ib, "input %q", c.in)
			continue
		}
		if assert.NotNil(t, ib, "input %q", c.in) {
			assert.Equal(t, c.scheme, ib.Scheme.String())
			assert.Equal(t, c.code, ib.Code.String())
		}
	}
}

// TestHydratePartyInboxes pins the inbox-from-routing logic: SIREN-match,
// leftover fallback, and never overwriting an inbox the CDV body already had.
func TestHydratePartyInboxes(t *testing.T) {
	t.Run("fills the issuer's missing inbox, matched by SIREN", func(t *testing.T) {
		supplier := sirenParty("100000009")
		supplier.Inboxes = []*org.Inbox{{Scheme: "0225", Code: "100000009_STATUTS"}} // body had it
		customer := sirenParty("200000008")                                          // issuer, no inbox
		hydratePartyInboxes(supplier, customer, routing{
			from: "0225:200000008_STATUTS",
			to:   "0225:100000009_STATUTS",
		})
		if assert.Len(t, customer.Inboxes, 1) {
			assert.Equal(t, "200000008_STATUTS", customer.Inboxes[0].Code.String())
		}
		// Supplier's existing inbox is untouched.
		assert.Equal(t, "100000009_STATUTS", supplier.Inboxes[0].Code.String())
	})

	t.Run("does not overwrite parties that already have an inbox", func(t *testing.T) {
		supplier := sirenParty("100000009")
		supplier.Inboxes = []*org.Inbox{{Scheme: "0225", Code: "orig"}}
		customer := sirenParty("200000008")
		customer.Inboxes = []*org.Inbox{{Scheme: "0225", Code: "orig2"}}
		hydratePartyInboxes(supplier, customer, routing{from: "0225:999", to: "0225:888"})
		assert.Equal(t, "orig", supplier.Inboxes[0].Code.String())
		assert.Equal(t, "orig2", customer.Inboxes[0].Code.String())
	})

	t.Run("leftover participant fills a party whose SIREN did not match", func(t *testing.T) {
		// Routing codes carry no SIREN substring; the lone missing party still
		// gets the lone unconsumed participant.
		customer := sirenParty("200000008")
		hydratePartyInboxes(nil, customer, routing{from: "0225:OPERATOR-INBOX"})
		if assert.Len(t, customer.Inboxes, 1) {
			assert.Equal(t, "OPERATOR-INBOX", customer.Inboxes[0].Code.String())
		}
	})

	t.Run("no routing leaves parties unchanged", func(t *testing.T) {
		customer := sirenParty("200000008")
		hydratePartyInboxes(nil, customer, routing{})
		assert.Empty(t, customer.Inboxes)
	})
}
