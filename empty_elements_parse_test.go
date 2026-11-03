package cii_test

import (
	"os"
	"strings"
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmptyElements(t *testing.T) {
	data, err := os.ReadFile(dataPath(pathParse, "CII_example1.xml"))
	require.NoError(t, err)
	doc := string(data)
	inject := func(anchor, extra string) {
		t.Helper()
		require.Equal(t, 1, strings.Count(doc, anchor), anchor)
		doc = strings.Replace(doc, anchor, anchor+extra, 1)
	}
	inject("<ram:TypeCode>380</ram:TypeCode>",
		`<ram:IncludedNote><ram:Content></ram:Content><ram:SubjectCode/></ram:IncludedNote>`)
	inject("<ram:LineID>1</ram:LineID>",
		`<ram:IncludedNote><ram:Content/><ram:SubjectCode/></ram:IncludedNote>`)
	inject("<ram:SellerAssignedID>166022</ram:SellerAssignedID>",
		`<ram:GlobalID schemeID="0160"></ram:GlobalID><ram:BuyerAssignedID/>`)
	inject("<ram:SellerTradeParty>",
		`<ram:GlobalID schemeID="0088"/>`+
			`<ram:DefinedTradeContact><ram:TelephoneUniversalCommunication><ram:CompleteNumber/></ram:TelephoneUniversalCommunication>`+
			`<ram:EmailURIUniversalCommunication><ram:URIID/></ram:EmailURIUniversalCommunication></ram:DefinedTradeContact>`+
			`<ram:URIUniversalCommunication><ram:URIID schemeID="0088"/></ram:URIUniversalCommunication>`)
	inject("<ram:ApplicableHeaderTradeAgreement>",
		`<ram:SellerOrderReferencedDocument><ram:IssuerAssignedID/></ram:SellerOrderReferencedDocument>`+
			`<ram:BuyerOrderReferencedDocument><ram:IssuerAssignedID/></ram:BuyerOrderReferencedDocument>`+
			`<ram:ContractReferencedDocument><ram:IssuerAssignedID/></ram:ContractReferencedDocument>`)
	inject("<ram:ApplicableHeaderTradeSettlement>",
		`<ram:InvoiceReferencedDocument><ram:IssuerAssignedID/></ram:InvoiceReferencedDocument>`)

	env, err := parseCII([]byte(doc))
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, inv.Notes, 1)
	assert.NotEmpty(t, inv.Notes[0].Text)
	assert.Empty(t, inv.Lines[0].Notes)
	assert.Empty(t, inv.Lines[0].Item.Identities)

	require.Len(t, inv.Supplier.Identities, 1)
	assert.Equal(t, "57151520", inv.Supplier.Identities[0].Code.String())
	assert.Empty(t, inv.Supplier.Telephones)
	assert.Empty(t, inv.Supplier.Emails)
	assert.Empty(t, inv.Supplier.Inboxes)

	assert.Nil(t, inv.Ordering)
	assert.Empty(t, inv.Preceding)
}
