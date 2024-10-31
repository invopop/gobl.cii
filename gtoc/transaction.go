package gtoc

import (
	"github.com/invopop/gobl/bill"
)

// NewTransaction creates the transaction part of a EN 16931 compliant invoice
func (c *Converter) NewTransaction(inv *bill.Invoice) error {

	c.doc.Transaction = &Transaction{}

	err := c.NewLines(inv.Lines)
	if err != nil {
		return err
	}

	err = c.NewAgreement(inv)
	if err != nil {
		return err
	}

	err = c.NewDelivery(inv)
	if err != nil {
		return err
	}

	err = c.NewSettlement(inv)
	if err != nil {
		return err
	}

	return nil
}
