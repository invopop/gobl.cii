package gtoc

import (
	"errors"
	"fmt"

	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/validation"
)

// issueDateFormat is the issue date format in the form YYYYMMDD
const issueDateFormat = "102"

// newHeader creates the ExchangedDocument part of a EN 16931 compliant invoice
func (c *Converter) newHeader(inv *bill.Invoice) error {
	tc, err := getTypeCode(inv)
	if err != nil {
		return err
	}
	h := &document.Header{
		ID:       invoiceNumber(inv.Series, inv.Code),
		TypeCode: tc,
		IssueDate: &document.IssueDate{
			DateFormat: &document.Date{
				Value:  formatIssueDate(inv.IssueDate),
				Format: issueDateFormat,
			},
		},
	}
	if len(inv.Notes) > 0 {
		notes := make([]*document.Note, 0, len(inv.Notes))
		for _, n := range inv.Notes {
			notes = append(notes, &document.Note{
				Content:     n.Text,
				SubjectCode: string(n.Code),
			})
		}
		h.IncludedNote = notes
	}
	c.doc.ExchangedDocument = h
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

func documentDate(date *cal.Date) *document.Date {
	if date == nil {
		return nil
	}
	return &document.Date{
		Value:  formatIssueDate(*date),
		Format: issueDateFormat,
	}
}

func formatIssueDate(d cal.Date) string {
	if d.IsZero() {
		return ""
	}
	t := d.Time()
	return t.Format("20060102")
}

func invoiceNumber(s cbc.Code, c cbc.Code) string {
	if s == "" {
		return c.String()
	}
	return fmt.Sprintf("%s-%s", s, c)
}
