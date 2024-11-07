package gtoc

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
)

// NewTransaction creates the transaction part of a EN 16931 compliant invoice
func (c *Converter) NewTransaction(inv *bill.Invoice) error {

	c.doc.Transaction = &document.Transaction{}

	err := c.NewLines(inv.Lines)
	if err != nil {
		return err
	}

	err = c.prepareAgreement(inv)
	if err != nil {
		return err
	}

	err = c.prepareDelivery(inv)
	if err != nil {
		return err
	}

	err = c.prepareSettlement(inv)
	if err != nil {
		return err
	}

	return nil
}
