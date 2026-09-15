package cii

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
)

// replacementCharRef matches the XML character references that decode to
// U+FFFD. They are plain ASCII in the document, so they survive a byte-level
// clean and only become the replacement character once the XML is decoded.
var replacementCharRef = regexp.MustCompile(`&#(?:[xX]0*[fF][fF][fF][dD]|0*65533);`)

// cleanString drops what a sender's broken encoding leaves behind: bytes that
// are not valid UTF-8, which the XML decoder rejects, and U+FFFD, which gobl's
// canonical JSON rejects, written literally or as a character reference.
// Neither is recoverable. Applied to the whole document before decoding, and
// idempotent.
//
// The U+FFFD half is a stopgap for invopop/gobl#975.
func cleanString(s string) string {
	s = replacementCharRef.ReplaceAllString(s, "")
	if utf8.ValidString(s) && !strings.ContainsRune(s, utf8.RuneError) {
		return s
	}
	s = strings.ToValidUTF8(s, "")
	return strings.ReplaceAll(s, string(utf8.RuneError), "")
}

// issueDateFormat is the issue date format in the form YYYYMMDD
const issueDateFormat = "102"

// Bytes returns the XML representation of the document in bytes
func (out *Invoice) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

func documentDate(date *cal.Date) *Date {
	if date == nil {
		return nil
	}
	return &Date{
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

// parseDate converts a date string to a cal.Date
func parseDate(date string) (cal.Date, error) {
	t, err := time.Parse("20060102", date)
	if err != nil {
		return cal.Date{}, err
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}

func invoiceNumber(s cbc.Code, c cbc.Code) string {
	if s == "" {
		return c.String()
	}
	return fmt.Sprintf("%s-%s", s, c)
}
