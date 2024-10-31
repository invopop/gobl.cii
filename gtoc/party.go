package gtoc

import (
	"fmt"

	"github.com/invopop/gobl/org"
)

// NewParty creates the SellerTradeParty part of a EN 16931 compliant invoice
func NewParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}
	p := &Party{
		Name:                      party.Name,
		Contact:                   newContact(party),
		PostalTradeAddress:        NewPostalTradeAddress(party.Addresses),
		URIUniversalCommunication: NewEmail(party.Emails),
	}

	if party.TaxID != nil {
		// Assumes VAT ID being used instead of possible tax number
		p.SpecifiedTaxRegistration = &SpecifiedTaxRegistration{
			ID:       party.TaxID.String(),
			SchemeID: "VA",
		}
	}

	return p
}

func newContact(supplier *org.Party) *Contact {
	if len(supplier.People) == 0 {
		return nil
	}

	contact := new(Contact)
	if len(supplier.People) > 0 {
		contact.PersonName = contactName(supplier.People[0].Name)
		if len(supplier.People[0].Emails) > 0 {
			contact.Email = &Email{
				URIID: supplier.People[0].Emails[0].Address,
			}
		}
	}
	if len(supplier.Telephones) > 0 {
		contact.Phone = &PhoneNumber{
			CompleteNumber: supplier.Telephones[0].Number,
		}
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
