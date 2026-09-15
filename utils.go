package cii

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
)

// cleanString strips the traces of a sender mishandling its own character
// encoding, neither of which is recoverable here — the original characters are
// gone before the document reaches us:
//
//   - byte sequences that are not valid UTF-8, which the XML decoder rejects
//     outright with "invalid UTF-8";
//   - the Unicode replacement character (U+FFFD), which a sender emits when its
//     own conversion has already given up. It is valid UTF-8, so it reaches
//     gobl, where canonical JSON refuses it and the document fails to digest.
//
// The parse entry points apply this to the whole document before decoding, so a
// field nobody thought to wrap cannot reintroduce the problem. It stays safe to
// call on individual values too, and is idempotent.
//
// The U+FFFD half is a stopgap: gobl/c14n rejects a valid code point, fixed
// upstream in invopop/gobl#975. Drop it once the gobl dependency carries that.
func cleanString(s string) string {
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
