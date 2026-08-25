package cii

import (
	"testing"

	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupplierWithReferencedSIREN(t *testing.T) {
	withIssuer := func(gids ...*CDARGlobalID) *CDAR {
		return &CDAR{AcknowledgementDocuments: []*CDARAcknowledgement{{
			ReferenceReferencedDocument: []*CDARReferencedDocument{{
				IssuerTradeParty: &CDARTradeParty{GlobalIDs: gids},
			}},
		}}}
	}
	sirenGID := &CDARGlobalID{SchemeID: schemeIDSIREN, Value: "100000009"}

	t.Run("no SE party: the MDT-129 issuer becomes the supplier", func(t *testing.T) {
		s := supplierWithReferencedSIREN(nil, withIssuer(sirenGID))
		require.NotNil(t, s)
		assert.Equal(t, "100000009", partySIREN(s))
	})

	t.Run("SE party without SIREN: MDT-129 SIREN is added", func(t *testing.T) {
		se := &org.Party{Name: "VENDEUR"}
		s := supplierWithReferencedSIREN(se, withIssuer(sirenGID))
		assert.Same(t, se, s)
		assert.Equal(t, "VENDEUR", s.Name)
		assert.Equal(t, "100000009", partySIREN(s))
	})

	t.Run("SE party with SIREN: left untouched", func(t *testing.T) {
		se := &org.Party{Identities: []*org.Identity{sirenIdentity("300000007")}}
		s := supplierWithReferencedSIREN(se, withIssuer(sirenGID))
		require.Len(t, s.Identities, 1)
		assert.Equal(t, "300000007", partySIREN(s))
	})

	t.Run("MDT-129 without 0002: nothing is invented", func(t *testing.T) {
		se := &org.Party{Name: "VENDEUR"}
		s := supplierWithReferencedSIREN(se, withIssuer(&CDARGlobalID{SchemeID: "0225", Value: "100000009_00012"}))
		assert.Empty(t, s.Identities)
	})

	t.Run("no MDT-129: supplier unchanged", func(t *testing.T) {
		assert.Nil(t, supplierWithReferencedSIREN(nil, &CDAR{}))
	})
}
