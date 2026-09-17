package cii

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func issueDate(v string) *IssueDate {
	return &IssueDate{DateFormat: &Date{Value: v, Format: issueDateFormat}}
}

func formattedDate(v string) *FormattedIssueDate {
	return &FormattedIssueDate{DateFormat: &Date{Value: v, Format: issueDateFormat}}
}

func TestGoblNewPaymentDetails(t *testing.T) {
	t.Run("a settlement with nothing to say carries no payment details", func(t *testing.T) {
		pymt, err := goblNewPaymentDetails(&Settlement{Summary: &Summary{}})
		require.NoError(t, err)
		assert.Nil(t, pymt)
	})

	t.Run("the payee and its address", func(t *testing.T) {
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{},
			Payee: &Party{
				Name:               "Factoring Co",
				PostalTradeAddress: &PostalTradeAddress{City: "Berlin", CountryID: "DE"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, pymt)
		require.NotNil(t, pymt.Payee)
		assert.Equal(t, "Factoring Co", pymt.Payee.Name)
		require.Len(t, pymt.Payee.Addresses, 1)
		assert.Equal(t, "Berlin", pymt.Payee.Addresses[0].Locality)
	})

	t.Run("a payee without an address", func(t *testing.T) {
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{},
			Payee:   &Party{Name: "Factoring Co"},
		})
		require.NoError(t, err)
		require.NotNil(t, pymt.Payee)
		assert.Empty(t, pymt.Payee.Addresses)
	})

	t.Run("advance payments", func(t *testing.T) {
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{},
			Advance: []*Advance{
				{Amount: "100.00", Date: formattedDate("20240115")},
				{Amount: testAmountHalf},
			},
		})
		require.NoError(t, err)
		require.Len(t, pymt.Advances, 2)
		assert.Equal(t, "100.00", pymt.Advances[0].Amount.String())
		require.NotNil(t, pymt.Advances[0].Date)
		assert.Equal(t, "2024-01-15", pymt.Advances[0].Date.String())
		assert.Equal(t, testAmountHalf, pymt.Advances[1].Amount.String())
		assert.Nil(t, pymt.Advances[1].Date)
	})

	t.Run("an advance amount that is not a number is an error", func(t *testing.T) {
		_, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{},
			Advance: []*Advance{{Amount: testNotANumber}},
		})
		assert.Error(t, err)
	})

	t.Run("an advance date that is not a date is an error", func(t *testing.T) {
		_, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{},
			Advance: []*Advance{{Amount: "10.00", Date: formattedDate("15/01/2024")}},
		})
		assert.Error(t, err)
	})

	t.Run("a prepaid total stands in for the advances it summarises", func(t *testing.T) {
		// Without the individual payments the totals would not recalculate, so
		// the prepaid total becomes a single synthetic advance.
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{TotalPrepaidAmount: "196.02"},
		})
		require.NoError(t, err)
		require.Len(t, pymt.Advances, 1)
		assert.Equal(t, "196.02", pymt.Advances[0].Amount.String())
	})

	t.Run("a prepaid total that is not a number is an error", func(t *testing.T) {
		_, err := goblNewPaymentDetails(&Settlement{Summary: &Summary{TotalPrepaidAmount: testNotANumber}})
		assert.Error(t, err)
	})

	t.Run("itemised advances win over the prepaid total", func(t *testing.T) {
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary: &Summary{TotalPrepaidAmount: "150.00"},
			Advance: []*Advance{{Amount: "100.00"}, {Amount: testAmountHalf}},
		})
		require.NoError(t, err)
		assert.Len(t, pymt.Advances, 2, "the summary must not be added on top")
	})

	t.Run("payment means type 1 is not an instruction", func(t *testing.T) {
		// "1" is the CII placeholder for an undefined means, which carries no
		// instruction to speak of.
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary:      &Summary{},
			PaymentMeans: []*PaymentMeans{{TypeCode: "1"}},
		})
		require.NoError(t, err)
		assert.Nil(t, pymt)
	})

	t.Run("a real payment means becomes an instruction", func(t *testing.T) {
		pymt, err := goblNewPaymentDetails(&Settlement{
			Summary:      &Summary{},
			PaymentMeans: []*PaymentMeans{{TypeCode: "30"}},
		})
		require.NoError(t, err)
		require.NotNil(t, pymt)
		require.NotNil(t, pymt.Instructions)
	})
}

func TestGoblNewTerms(t *testing.T) {
	t.Run("a description becomes the notes", func(t *testing.T) {
		terms, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{Description: "Net 30 days"}},
		})
		require.NoError(t, err)
		require.NotNil(t, terms)
		assert.Equal(t, "Net 30 days", terms.Notes)
	})

	t.Run("a lone due date is taken as the whole amount", func(t *testing.T) {
		terms, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{DueDate: issueDate("20240215")}},
		})
		require.NoError(t, err)
		require.Len(t, terms.DueDates, 1)
		assert.Equal(t, "2024-02-15", terms.DueDates[0].Date.String())
		require.NotNil(t, terms.DueDates[0].Percent, "a single due date covers everything")
		assert.Equal(t, "100%", terms.DueDates[0].Percent.String())
	})

	t.Run("a due date with a partial amount keeps it", func(t *testing.T) {
		terms, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{DueDate: issueDate("20240215"), Amount: testAmountHalf}},
		})
		require.NoError(t, err)
		require.Len(t, terms.DueDates, 1)
		require.NotNil(t, terms.DueDates[0].Amount)
		assert.Equal(t, testAmountHalf, terms.DueDates[0].Amount.String())
		assert.Nil(t, terms.DueDates[0].Percent, "an amount and a percent are alternatives")
	})

	t.Run("a due date with a percentage keeps it", func(t *testing.T) {
		terms, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{DueDate: issueDate("20240215"), Percent: "50%"}},
		})
		require.NoError(t, err)
		require.Len(t, terms.DueDates, 1)
		require.NotNil(t, terms.DueDates[0].Percent)
		assert.Equal(t, "50%", terms.DueDates[0].Percent.String())
	})

	t.Run("several due dates are left as they are", func(t *testing.T) {
		// The 100% default only applies to a single due date.
		terms, err := goblNewTerms(&Settlement{
			Summary: &Summary{},
			PaymentTerms: []*Terms{
				{DueDate: issueDate("20240215"), Amount: testAmountHalf},
				{DueDate: issueDate("20240315"), Amount: testAmountHalf},
			},
		})
		require.NoError(t, err)
		require.Len(t, terms.DueDates, 2)
		assert.Nil(t, terms.DueDates[0].Percent)
	})

	t.Run("a due date that is not a date is an error", func(t *testing.T) {
		_, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{DueDate: issueDate("15/02/2024")}},
		})
		assert.Error(t, err)
	})

	t.Run("a partial amount that is not a number is an error", func(t *testing.T) {
		_, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{DueDate: issueDate("20240215"), Amount: testNotANumber}},
		})
		assert.Error(t, err)
	})

	t.Run("a percentage that is not a number is an error", func(t *testing.T) {
		_, err := goblNewTerms(&Settlement{
			Summary:      &Summary{},
			PaymentTerms: []*Terms{{DueDate: issueDate("20240215"), Percent: testNotAPercent}},
		})
		assert.Error(t, err)
	})

	t.Run("terms with nothing in them are no terms", func(t *testing.T) {
		terms, err := goblNewTerms(&Settlement{Summary: &Summary{}, PaymentTerms: []*Terms{{}}})
		require.NoError(t, err)
		assert.Nil(t, terms)
	})
}
