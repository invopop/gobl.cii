package cii

// Literals shared across the package's tests. They live here so that repeating
// a currency or a sample identifier across files does not trip goconst.
const (
	testCurrencyEUR = "EUR"
	testCategoryVAT = "VAT"

	testDocID     = "DOC-1"
	testCostRef   = "COST-1"
	testBuyerRef  = "PO4711"
	testPartyName = "Acme"
	testGivenName = "Jane"
	testSurname   = "Sample"

	testAmountSmall = "2.50"
	testAmountHalf  = "50.00"

	// Values that must fail to parse as a number and a percentage.
	testNotANumber  = "n/a"
	testNotAPercent = "lots"
)
