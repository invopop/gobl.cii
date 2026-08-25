package cii

import (
	"testing"

	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/fr"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPartyTaxRegistrations pins the scheme IDs on SpecifiedTaxRegistration,
// which BR-E-02, BR-Z-02 and BR-AE-02 test for.
func TestNewPartyTaxRegistrations(t *testing.T) {
	t.Run("tax-scope identity becomes an FC registration", func(t *testing.T) {
		p := newParty(&org.Party{
			Name: "Franchise Test SARL",
			Identities: []*org.Identity{
				{
					Scope: org.IdentityScopeTax,
					Type:  fr.IdentityTypeSIREN,
					Code:  "483671517",
				},
			},
		}, ContextEN16931V2017)

		require.Len(t, p.SpecifiedTaxRegistration, 1)
		assert.Equal(t, SchemeIDTaxRegistration, p.SpecifiedTaxRegistration[0].ID.SchemeID)
		assert.Equal(t, "483671517", p.SpecifiedTaxRegistration[0].ID.Value)

		// Must not also fall through to BT-29 or the GlobalID
		assert.Nil(t, p.ID)
		assert.Nil(t, p.GlobalID)
	})

	t.Run("VAT id and tax-scope identity coexist", func(t *testing.T) {
		p := newParty(&org.Party{
			Name: "Supplier SARL",
			TaxID: &tax.Identity{
				Country: "FR",
				Code:    "39356000000",
			},
			Identities: []*org.Identity{
				{
					Scope: org.IdentityScopeTax,
					Type:  fr.IdentityTypeSIRET,
					Code:  "35600000000048",
				},
			},
		}, ContextEN16931V2017)

		require.Len(t, p.SpecifiedTaxRegistration, 2)
		assert.Equal(t, SchemeIDVAT, p.SpecifiedTaxRegistration[0].ID.SchemeID)
		assert.Equal(t, "FR39356000000", p.SpecifiedTaxRegistration[0].ID.Value)
		assert.Equal(t, SchemeIDTaxRegistration, p.SpecifiedTaxRegistration[1].ID.SchemeID)
		assert.Equal(t, "35600000000048", p.SpecifiedTaxRegistration[1].ID.Value)
	})

	t.Run("legal-scope identity is not promoted to a tax registration", func(t *testing.T) {
		p := newParty(&org.Party{
			Name: "Franchise Test SARL",
			Identities: []*org.Identity{
				{
					Scope: org.IdentityScopeLegal,
					Type:  fr.IdentityTypeSIREN,
					Code:  "483671517",
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						iso.ExtKeySchemeID: "0002",
					}),
				},
			},
		}, ContextEN16931V2017)

		require.NotNil(t, p.LegalOrganization)
		assert.Equal(t, "483671517", p.LegalOrganization.ID.Value)
		assert.Equal(t, "0002", p.LegalOrganization.ID.SchemeID)
		assert.Empty(t, p.SpecifiedTaxRegistration)
	})
}
