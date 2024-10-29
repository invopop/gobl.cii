package gtoc

import (
	"fmt"

	"github.com/invopop/gobl/org"
)

// NewSeller creates the SellerTradeParty part of a EN 16931 compliant invoice
func NewSeller(supplier *org.Party) *Seller {
	if supplier == nil {
		return nil
	}
	seller := &Seller{
		Name:                      supplier.Name,
		Contact:                   newContact(supplier),
		PostalTradeAddress:        NewPostalTradeAddress(supplier.Addresses),
		URIUniversalCommunication: NewEmail(supplier.Emails),
	}

	if supplier.TaxID != nil {
		seller.SpecifiedTaxRegistration = &SpecifiedTaxRegistration{
			ID:       supplier.TaxID.String(),
			SchemeID: "VA",
		}
	}

	return seller
}

func newContact(supplier *org.Party) *Contact {
	if len(supplier.People) == 0 && len(supplier.Telephones) == 0 && len(supplier.Emails) == 0 {
		return nil
	}

	contact := new(Contact)
	if len(supplier.People) > 0 {
		contact.PersonName = contactName(supplier.People[0].Name)
	}
	if len(supplier.Telephones) > 0 {
		contact.Phone = supplier.Telephones[0].Number
	}
	if len(supplier.Emails) > 0 {
		contact.Email = supplier.Emails[0].Address
	}

	return contact
}

func contactName(personName *org.Name) string {
	given := personName.Given
	surname := personName.Surname

	if given == "" && surname == "" {
		return ""
	}

	if given == "" {
		return surname
	}

	if surname == "" {
		return given
	}

	return fmt.Sprintf("%s %s", given, surname)
}
