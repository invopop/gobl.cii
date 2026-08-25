package cii

import (
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// noteKeySubjectCodeMap mirrors the UNTDID 4451 subject codes that gobl's
// en16931 addon assigns to GOBL note keys, so that generated notes carry a
// subject code even without the extension, and parsed notes recover their key.
var noteKeySubjectCodeMap = map[cbc.Key]cbc.Code{
	org.NoteKeyGoods:          "AAA",
	org.NoteKeyPayment:        "PMT",
	org.NoteKeyPaymentMethod:  "PMD",
	org.NoteKeyPaymentTerm:    "AAB",
	org.NoteKeyGeneral:        "AAI",
	org.NoteKeyLegal:          "ABL",
	org.NoteKeyDangerousGoods: "AAC",
	org.NoteKeyAck:            "AAE",
	org.NoteKeyRate:           "AAF",
	org.NoteKeyReason:         "ACD",
	org.NoteKeyDispute:        "ACE",
	org.NoteKeyCustomer:       "CUR",
	org.NoteKeyGlossary:       "ACZ",
	org.NoteKeyCustoms:        "CUS",
	org.NoteKeyHandling:       "HAN",
	org.NoteKeyPackaging:      "PKG",
	org.NoteKeyLoading:        "LOI",
	org.NoteKeyPrice:          "AAK",
	org.NoteKeyPriority:       "PRI",
	org.NoteKeyRegulatory:     "REG",
	org.NoteKeySafety:         "SAF",
	org.NoteKeyShipLine:       "SLR",
	org.NoteKeySupplier:       "SUR",
	org.NoteKeyTransport:      "TRA",
	org.NoteKeyDelivery:       "DEL",
	org.NoteKeyQuarantine:     "QIN",
	org.NoteKeyTax:            "TXD",
	org.NoteKeyOther:          "ZZZ",
}

// noteKeyForSubjectCode provides the GOBL note key for an UNTDID 4451 text
// subject code, or an empty key when none matches.
func noteKeyForSubjectCode(code cbc.Code) cbc.Key {
	for k, c := range noteKeySubjectCodeMap {
		if c == code {
			return k
		}
	}
	return cbc.KeyEmpty
}

// newNote converts a GOBL note into a CII note, taking the subject code from
// the UNTDID text subject extension or, failing that, from the note key.
func newNote(n *org.Note) *Note {
	note := &Note{Content: n.Text}
	if code := n.Ext.Get(untdid.ExtKeyTextSubject); code != "" {
		note.SubjectCode = code.String()
	} else if code, ok := noteKeySubjectCodeMap[n.Key]; ok {
		note.SubjectCode = code.String()
	}
	return note
}

// goblParseNote converts a CII note into a GOBL note, recovering the note key
// and the UNTDID text subject extension from the subject code.
func goblParseNote(note *Note) *org.Note {
	n := &org.Note{Text: note.Content}
	if note.SubjectCode != "" {
		code := cbc.Code(note.SubjectCode)
		n.Key = noteKeyForSubjectCode(code)
		n.Ext = tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyTextSubject: code,
		})
	}
	return n
}
