package cii

import (
	"testing"

	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/fr"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/invopop/xmlctx"
)

// supplierName is shared by the party fixtures below.
const supplierName = "Fournisseur SARL"

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

// TestNewPartyMultipleIdentifiers pins BT-29/BT-46 as 0..n: a French CTC seller
// carries a SIRET (0009), a routing code (0224) and, when it belongs to a single
// taxable entity, the group SIREN (0231). All of them must reach the wire.
func TestNewPartyMultipleIdentifiers(t *testing.T) {
	t.Run("every qualified identity becomes a GlobalID", func(t *testing.T) {
		p := newParty(&org.Party{
			Name: supplierName,
			Identities: []*org.Identity{
				{
					Scope: org.IdentityScopeLegal,
					Code:  "341200068",
					Ext:   tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0002"}),
				},
				{
					Code: "34120006871491",
					Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0009"}),
				},
				{
					Code: "ROUTAGE_B2G",
					Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0224"}),
				},
				{
					Code: "356000000",
					Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0231"}),
				},
			},
		}, ContextEN16931V2017)

		// The legal identity is BT-30, not BT-29.
		require.NotNil(t, p.LegalOrganization)
		assert.Equal(t, "341200068", p.LegalOrganization.ID.Value)
		assert.Equal(t, "0002", p.LegalOrganization.ID.SchemeID)

		require.Len(t, p.GlobalID, 3)
		assert.Equal(t, "0009", p.GlobalID[0].SchemeID)
		assert.Equal(t, "34120006871491", p.GlobalID[0].Value)
		assert.Equal(t, "0224", p.GlobalID[1].SchemeID)
		assert.Equal(t, "ROUTAGE_B2G", p.GlobalID[1].Value)
		assert.Equal(t, "0231", p.GlobalID[2].SchemeID)
		assert.Equal(t, "356000000", p.GlobalID[2].Value)
	})

	t.Run("every unqualified identity becomes an ID", func(t *testing.T) {
		p := newParty(&org.Party{
			Name: supplierName,
			Identities: []*org.Identity{
				{Code: "REF-ONE"},
				{Code: "REF-TWO"},
			},
		}, ContextEN16931V2017)

		require.Len(t, p.ID, 2)
		assert.Equal(t, "REF-ONE", p.ID[0].Value)
		assert.Equal(t, "REF-TWO", p.ID[1].Value)
		assert.Empty(t, p.GlobalID)
	})
}

// TestParsePartyMultipleIdentifiers guards the wire side of BT-29/BT-46 being
// 0..n: a single-valued field would silently keep only the last ram:GlobalID.
func TestParsePartyMultipleIdentifiers(t *testing.T) {
	const in = `<ram:SellerTradeParty xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
		<ram:ID>REF-ONE</ram:ID>
		<ram:ID>REF-TWO</ram:ID>
		<ram:GlobalID schemeID="0009">34120006871491</ram:GlobalID>
		<ram:GlobalID schemeID="0224">ROUTAGE_B2G</ram:GlobalID>
		<ram:GlobalID schemeID="0231">356000000</ram:GlobalID>
		<ram:Name>Fournisseur SARL</ram:Name>
	</ram:SellerTradeParty>`

	party := new(Party)
	require.NoError(t, xmlctx.Unmarshal([]byte(in), party, xmlctx.WithNamespaces(
		map[string]string{nsPrefixRAM: NamespaceRAM},
	)))

	require.Len(t, party.ID, 2)
	require.Len(t, party.GlobalID, 3)

	p := goblNewParty(party)
	require.Len(t, p.Identities, 5)

	got := make([]string, 0, len(p.Identities))
	for _, id := range p.Identities {
		got = append(got, id.Ext.Get(iso.ExtKeySchemeID).String()+":"+id.Code.String())
	}
	assert.Equal(t, []string{
		":REF-ONE",
		":REF-TWO",
		"0009:34120006871491",
		"0224:ROUTAGE_B2G",
		"0231:356000000",
	}, got)
}

// TestParsePartyTaxRegistrations covers the wire side of BT-31/BT-32: what
// reaches GOBL from a SpecifiedTaxRegistration, whatever scheme it carries.
func TestParsePartyTaxRegistrations(t *testing.T) {
	parse := func(t *testing.T, in string) *org.Party {
		t.Helper()
		party := new(Party)
		require.NoError(t, xmlctx.Unmarshal([]byte(in), party, xmlctx.WithNamespaces(
			map[string]string{nsPrefixRAM: NamespaceRAM},
		)))
		return goblNewParty(party)
	}

	t.Run("FC alongside VA", func(t *testing.T) {
		p := parse(t, `<ram:SellerTradeParty xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
			<ram:Name>Fournisseur SARL</ram:Name>
			<ram:PostalTradeAddress><ram:CountryID>FR</ram:CountryID></ram:PostalTradeAddress>
			<ram:SpecifiedTaxRegistration><ram:ID schemeID="FC">828701557</ram:ID></ram:SpecifiedTaxRegistration>
			<ram:SpecifiedTaxRegistration><ram:ID schemeID="VA">FR18828701557</ram:ID></ram:SpecifiedTaxRegistration>
		</ram:SellerTradeParty>`)

		require.NotNil(t, p.TaxID)
		assert.Equal(t, cbc.Code("18828701557"), p.TaxID.Code)
		require.Len(t, p.Identities, 1)
		assert.Equal(t, org.IdentityScopeTax, p.Identities[0].Scope)
		assert.Equal(t, cbc.Code("828701557"), p.Identities[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("FR"), p.Identities[0].Country)
		// The scheme code is not a national identifier type.
		assert.Empty(t, p.Identities[0].Type)
	})

	t.Run("UBL spelling of the VAT scheme", func(t *testing.T) {
		p := parse(t, `<ram:SellerTradeParty xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
			<ram:Name>Leverantör AB</ram:Name>
			<ram:PostalTradeAddress><ram:CountryID>SE</ram:CountryID></ram:PostalTradeAddress>
			<ram:SpecifiedTaxRegistration><ram:ID schemeID="VAT">SE123456789001</ram:ID></ram:SpecifiedTaxRegistration>
		</ram:SellerTradeParty>`)

		require.NotNil(t, p.TaxID)
		assert.Equal(t, l10n.TaxCountryCode("SE"), p.TaxID.Country)
		// The fixture code does not pass tax.ParseIdentity, so it takes
		// the fallback path; normalization strips the prefix later.
		assert.Equal(t, cbc.Code("SE123456789001"), p.TaxID.Code)
		assert.Empty(t, p.Identities)
	})

	t.Run("unrecognized scheme is kept as a tax registration", func(t *testing.T) {
		p := parse(t, `<ram:SellerTradeParty xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
			<ram:Name>Fournisseur SARL</ram:Name>
			<ram:PostalTradeAddress><ram:CountryID>FR</ram:CountryID></ram:PostalTradeAddress>
			<ram:SpecifiedTaxRegistration><ram:ID>828701557</ram:ID></ram:SpecifiedTaxRegistration>
		</ram:SellerTradeParty>`)

		assert.Nil(t, p.TaxID)
		require.Len(t, p.Identities, 1)
		assert.Equal(t, org.IdentityScopeTax, p.Identities[0].Scope)
		assert.Equal(t, cbc.Code("828701557"), p.Identities[0].Code)
	})

	t.Run("party without an address", func(t *testing.T) {
		p := parse(t, `<ram:SellerTradeParty xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
			<ram:Name>Fournisseur SARL</ram:Name>
			<ram:SpecifiedTaxRegistration><ram:ID schemeID="FC">828701557</ram:ID></ram:SpecifiedTaxRegistration>
			<ram:SpecifiedTaxRegistration><ram:ID schemeID="VA">FR18828701557</ram:ID></ram:SpecifiedTaxRegistration>
		</ram:SellerTradeParty>`)

		require.NotNil(t, p.TaxID)
		assert.Equal(t, cbc.Code("18828701557"), p.TaxID.Code)
		require.Len(t, p.Identities, 1)
		assert.Empty(t, p.Identities[0].Country)
	})
}

// TestNewPartyMultipleLegalIdentities pins BT-30 as 0..1: the extra legal
// identities must reach the wire as BT-29 identifiers instead of overwriting
// the SpecifiedLegalOrganization.
func TestNewPartyMultipleLegalIdentities(t *testing.T) {
	p := newParty(&org.Party{
		Name: supplierName,
		Identities: []*org.Identity{
			{
				Scope: org.IdentityScopeLegal,
				Type:  fr.IdentityTypeSIREN,
				Code:  "356000000",
				Ext:   tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0002"}),
			},
			{
				Scope: org.IdentityScopeLegal,
				Type:  fr.IdentityTypeSIRET,
				Code:  "35600000000048",
				Ext:   tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0009"}),
			},
		},
	}, ContextEN16931V2017)

	require.NotNil(t, p.LegalOrganization)
	assert.Equal(t, "356000000", p.LegalOrganization.ID.Value)
	assert.Equal(t, "0002", p.LegalOrganization.ID.SchemeID)

	require.Len(t, p.GlobalID, 1)
	assert.Equal(t, "35600000000048", p.GlobalID[0].Value)
	assert.Equal(t, "0009", p.GlobalID[0].SchemeID)
}
