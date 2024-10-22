package gtoc

import (
	"github.com/invopop/gobl/bill"
)

// NewTransaction creates the transaction part of a EN 16931 compliant invoice
func NewTransaction(inv *bill.Invoice) (*Transaction, error) {
	agreement, err := NewAgreement(inv)
	if err != nil {
		return nil, err
	}

	transaction := &Transaction{
		Lines:      NewLines(inv.Lines),
		Agreement:  agreement,
		Delivery:   &Delivery{},
		Settlement: NewSettlement(inv),
	}

	return transaction, nil
}
