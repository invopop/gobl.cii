package cii

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ordering builds the smallest invoice goblNewOrdering reads from.
func ordering(t *testing.T, f func(ag *Agreement, stlm *Settlement)) *Invoice {
	t.Helper()
	ag, stlm := &Agreement{}, &Settlement{}
	if f != nil {
		f(ag, stlm)
	}
	// Delivery is dereferenced unguarded, so it has to be present.
	return &Invoice{Transaction: &Transaction{
		Agreement:  ag,
		Settlement: stlm,
		Delivery:   &Delivery{},
	}}
}

func TestGoblNewOrdering(t *testing.T) {
	t.Run("BT-10 buyer reference", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, _ *Settlement) {
			ag.BuyerReference = testBuyerRef
		}))
		require.NoError(t, err)
		require.NotNil(t, ord)
		assert.Equal(t, testBuyerRef, ord.Code.String())
	})

	t.Run("BT-19 buyer accounting reference", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, stlm *Settlement) {
			ag.BuyerReference = testBuyerRef
			stlm.AccountingAccount = &AccountingAccount{ID: testCostRef}
		}))
		require.NoError(t, err)
		require.NotNil(t, ord)
		assert.Equal(t, testCostRef, ord.Cost.String())
	})

	t.Run("an accounting account without an id is ignored", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, stlm *Settlement) {
			ag.BuyerReference = testBuyerRef
			stlm.AccountingAccount = &AccountingAccount{}
		}))
		require.NoError(t, err)
		require.NotNil(t, ord)
		assert.Empty(t, ord.Cost.String())
	})

	t.Run("an accounting reference on its own is dropped", func(t *testing.T) {
		// goblOrderingHasData does not count Cost, so BT-19 is lost when it is
		// the only ordering detail the document carries. Pinned as the current
		// behaviour rather than endorsed.
		ord, err := goblNewOrdering(ordering(t, func(_ *Agreement, stlm *Settlement) {
			stlm.AccountingAccount = &AccountingAccount{ID: testCostRef}
		}))
		require.NoError(t, err)
		assert.Nil(t, ord)
	})

	t.Run("BT-14 sales order reference", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, _ *Settlement) {
			ag.Sales = &IssuerID{ID: "SO-1"}
		}))
		require.NoError(t, err)
		require.Len(t, ord.Sales, 1)
		assert.Equal(t, "SO-1", ord.Sales[0].Code.String())
	})

	t.Run("BT-13 purchase order reference", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, _ *Settlement) {
			ag.Purchase = &IssuerID{ID: "PO-1"}
		}))
		require.NoError(t, err)
		require.Len(t, ord.Purchases, 1)
		assert.Equal(t, "PO-1", ord.Purchases[0].Code.String())
	})

	t.Run("BT-11 project reference keeps its name", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, _ *Settlement) {
			ag.Project = &Project{ID: "PRJ-1", Name: "Rollout"}
		}))
		require.NoError(t, err)
		require.Len(t, ord.Projects, 1)
		assert.Equal(t, "PRJ-1", ord.Projects[0].Code.String())
		assert.Equal(t, "Rollout", ord.Projects[0].Description)
	})

	t.Run("BT-12 contract reference", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(ag *Agreement, _ *Settlement) {
			ag.Contract = &IssuerID{ID: "CON-1"}
		}))
		require.NoError(t, err)
		require.Len(t, ord.Contracts, 1)
		assert.Equal(t, "CON-1", ord.Contracts[0].Code.String())
	})

	t.Run("BG-14 billing period", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, func(_ *Agreement, stlm *Settlement) {
			stlm.Period = &Period{Start: issueDate("20240101"), End: issueDate("20240131")}
		}))
		require.NoError(t, err)
		require.NotNil(t, ord)
		require.NotNil(t, ord.Period)
		require.NotNil(t, ord.Period.Start)
		assert.Equal(t, "2024-01-01", ord.Period.Start.String())
		require.NotNil(t, ord.Period.End)
		assert.Equal(t, "2024-01-31", ord.Period.End.String())
	})

	t.Run("a period label", func(t *testing.T) {
		desc := "January"
		ord, err := goblNewOrdering(ordering(t, func(_ *Agreement, stlm *Settlement) {
			stlm.Period = &Period{Start: issueDate("20240101"), Description: &desc}
		}))
		require.NoError(t, err)
		require.NotNil(t, ord)
		require.NotNil(t, ord.Period)
		assert.Equal(t, "January", ord.Period.Label)
	})

	t.Run("a malformed period start is an error", func(t *testing.T) {
		_, err := goblNewOrdering(ordering(t, func(_ *Agreement, stlm *Settlement) {
			stlm.Period = &Period{Start: issueDate("01/01/2024")}
		}))
		assert.Error(t, err)
	})

	t.Run("a malformed period end is an error", func(t *testing.T) {
		_, err := goblNewOrdering(ordering(t, func(_ *Agreement, stlm *Settlement) {
			stlm.Period = &Period{End: issueDate("31/01/2024")}
		}))
		assert.Error(t, err)
	})

	t.Run("an invoice with no ordering details carries none", func(t *testing.T) {
		ord, err := goblNewOrdering(ordering(t, nil))
		require.NoError(t, err)
		assert.Nil(t, ord)
	})
}
