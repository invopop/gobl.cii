package cii

import (
	"bytes"
	"unicode/utf8"
)

// cleanXML sanitises an incoming XML document so that broken text encoding
// upstream cannot fail the whole conversion.
//
// Two things are repaired, both symptoms of a sender mishandling its own
// character encoding, and neither recoverable here:
//
//   - byte sequences that are not valid UTF-8, which the XML decoder rejects
//     outright with "invalid UTF-8";
//   - U+FFFD, the replacement character, which a sender emits when its own
//     conversion has already given up on a character. It is valid UTF-8, so it
//     reaches gobl intact, where canonical JSON rejects it and the document
//     fails to digest.
//
// Both are dropped, matching what gobl.ubl's cleanString already does to the
// individual fields it covers. The original characters are lost before the
// document reaches us either way, so this only decides whether a document with
// damaged free text converts or is discarded whole.
//
// Cleaning the document rather than each field means a field nobody thought to
// wrap cannot reintroduce the problem — which is how this surfaced, on a note
// that no per-field cleaning touched.
//
// The U+FFFD half is a stopgap: gobl/c14n rejects a valid code point, fixed
// upstream in invopop/gobl#975. Drop it once the gobl dependency carries that.
func cleanXML(data []byte) []byte {
	replacementChar := []byte(string(utf8.RuneError))
	if utf8.Valid(data) && !bytes.Contains(data, replacementChar) {
		return data
	}
	out := bytes.ToValidUTF8(data, nil)
	return bytes.ReplaceAll(out, replacementChar, nil)
}
