package gtoc

import (
	"github.com/invopop/gobl/org"
)

// NewBuyer creates the BuyerTradeParty part of a EN 16931 compliant invoice
func NewBuyer(customer *org.Party) *Buyer {
	buyer := &Buyer{
		Name:                      customer.Name,
		Contact:                   newContact(customer),
		PostalTradeAddress:        NewPostalTradeAddress(customer.Addresses),
		URIUniversalCommunication: NewEmail(customer.Emails),
	}

	if customer.TaxID != nil {
		buyer.ID = customer.TaxID.String()
	}

	return buyer
}
