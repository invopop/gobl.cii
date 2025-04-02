package cii

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/validation"
)

// ExchangedContext defines the structure of the ExchangedDocumentContext of the CII standard
type ExchangedContext struct {
	BusinessContext  *ExchangedContextParameter `xml:"ram:BusinessProcessSpecifiedDocumentContextParameter,omitempty"`
	GuidelineContext *ExchangedContextParameter `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

// ExchangedContextParameter defines the structure of the ExchangedDocumentContextParameter of the CII standard
type ExchangedContextParameter struct {
	ID string `xml:"ram:ID"`
}

// Header a collection of data for a Cross Industry Invoice Header that is exchanged between two or more parties in written, printed or electronic form.
type Header struct {
	ID           string     `xml:"ram:ID"`
	TypeCode     string     `xml:"ram:TypeCode"`
	IssueDate    *IssueDate `xml:"ram:IssueDateTime"`
	IncludedNote []*Note    `xml:"ram:IncludedNote,omitempty"`
}

// IssueDate defines the structure of the IssueDateTime of the CII standard
type IssueDate struct {
	DateFormat *Date `xml:"udt:DateTimeString"`
}

// FormattedIssueDate defines the structure of the FormattedIssueDateTime of the CII standard
type FormattedIssueDate struct {
	DateFormat *Date `xml:"qdt:DateTimeString"`
}

// newHeader creates the ExchangedDocument part of a EN 16931 compliant invoice
func (out *Invoice) addHeader(inv *bill.Invoice) error {
	tc, err := getTypeCode(inv)
	if err != nil {
		return err
	}
	h := &Header{
		ID:       invoiceNumber(inv.Series, inv.Code),
		TypeCode: tc,
		IssueDate: &IssueDate{
			DateFormat: &Date{
				Value:  formatIssueDate(inv.IssueDate),
				Format: issueDateFormat,
			},
		},
	}
	if len(inv.Notes) > 0 {
		notes := make([]*Note, 0, len(inv.Notes))
		for _, n := range inv.Notes {
			notes = append(notes, &Note{
				Content:     n.Text,
				SubjectCode: string(n.Code),
			})
		}
		h.IncludedNote = notes
	}
	out.ExchangedDocument = h
	return nil
}

func getTypeCode(inv *bill.Invoice) (string, error) {
	if inv.Tax == nil || inv.Tax.Ext == nil || inv.Tax.Ext[untdid.ExtKeyDocumentType].String() == "" {
		return "", validation.Errors{
			"tax": validation.Errors{
				"ext": validation.Errors{
					untdid.ExtKeyDocumentType.String(): errors.New("required"),
				},
			},
		}
	}
	return inv.Tax.Ext.Get(untdid.ExtKeyDocumentType).String(), nil
}
