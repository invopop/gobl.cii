package cii

import (
	"bytes"
	"encoding/xml"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/xmlctx"
)

// KeyCII identifies CII invoices that do not declare a known specification.
// They can be imported, but not exported.
const KeyCII cbc.Key = "cii"

const mimeXML = "application/xml"

// converter implements convert.Converter for a set of registered formats.
// The base converter also imports CII invoices that declare no known
// specification.
type converter struct {
	formats  []Format
	fallback bool
}

func (c *converter) Formats() []*convert.Format {
	list := make([]*convert.Format, 0, len(c.formats)+1)
	if c.fallback {
		list = append(list, &convert.Format{
			Key:    KeyCII,
			Name:   i18n.NewString("CII"),
			MIME:   mimeXML,
			Syntax: "cii",
			Import: []schema.ID{invoiceSchema},
		})
	}
	for _, f := range c.formats {
		list = append(list, &convert.Format{
			Key:         f.Key,
			Name:        f.Name,
			MIME:        mimeXML,
			Syntax:      "cii",
			Countries:   f.Countries,
			Import:      f.Import,
			Export:      f.Export,
			Attachments: f.Attachments,
		})
	}
	return list
}

func (c *converter) Detect(in *convert.Input) cbc.Key {
	h := ReadHeader(in)
	if h.Err != nil || h.Namespace != NamespaceRSM {
		return cbc.KeyEmpty
	}
	// A requested format the document could be in wins over detection.
	for _, k := range in.Requested {
		if f := c.format(k); f != nil && f.claims(h.GuidelineID, h.BusinessID) {
			return k
		}
	}
	if f := FindFormat(h.GuidelineID, h.BusinessID); f != nil {
		if c.format(f.Key) != nil {
			return f.Key
		}
		return cbc.KeyEmpty
	}
	if c.fallback {
		return KeyCII
	}
	return cbc.KeyEmpty
}

// Import decodes the data and imports it in the detected format, and provides
// the files embedded in invoices as attachments.
func (c *converter) Import(key cbc.Key, data []byte, _ *convert.ImportOptions) (*convert.ImportResult, error) {
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	// The detected key may be a requested format detection alone wouldn't
	// pick, so import in that format rather than finding it again.
	var opts []Option
	if f := c.format(key); f != nil {
		opts = append(opts, WithFormat(*f))
	}
	env, err := Import(doc, opts...)
	if err != nil {
		return nil, err
	}
	res := &convert.ImportResult{Envelope: env}
	if inv, ok := doc.(*Invoice); ok {
		res.Attachments = convertAttachments(inv.ExtractBinaryAttachments())
	}
	return res, nil
}

// Accepts checks the document includes the addons of the format.
func (c *converter) Accepts(key cbc.Key, env *gobl.Envelope) error {
	f := c.format(key)
	if f == nil {
		return ErrUnsupportedDocumentType
	}
	if len(f.Addons) == 0 {
		return nil
	}
	doc := env.Extract()
	set := convert.Rules(f.Key, rules.For(doc, convert.RequireAddons("01", f.Addons...)))
	if fs := set.Validate(doc); fs != nil {
		return fs
	}
	return nil
}

// Export exports the envelope in the format, embedding the attachments.
func (c *converter) Export(key cbc.Key, env *gobl.Envelope, opts *convert.ExportOptions) ([]byte, error) {
	f := c.format(key)
	if f == nil {
		return nil, ErrUnsupportedDocumentType
	}
	doc, err := Export(env, WithFormat(*f), WithAttachments(binaryAttachments(opts.Attachments)...))
	if err != nil {
		return nil, err
	}
	return Encode(doc)
}

// binaryAttachments provides the register's attachments as CII binary
// attachments, identified by their name when they have no ID.
func binaryAttachments(atts []*convert.Attachment) []BinaryAttachment {
	out := make([]BinaryAttachment, len(atts))
	for i, a := range atts {
		id := a.ID
		if id == "" {
			id = a.Name
		}
		out[i] = BinaryAttachment{
			ID:          id,
			Description: a.Description,
			Data:        a.Data,
			MimeCode:    a.MIME,
			Filename:    a.Name,
		}
	}
	return out
}

// convertAttachments provides CII binary attachments as the register's
// attachments.
func convertAttachments(atts []BinaryAttachment) []*convert.Attachment {
	out := make([]*convert.Attachment, len(atts))
	for i, a := range atts {
		out[i] = &convert.Attachment{
			ID:          a.ID,
			Name:        a.Filename,
			Description: a.Description,
			MIME:        a.MimeCode,
			Data:        a.Data,
		}
	}
	return out
}

func (c *converter) format(key cbc.Key) *Format {
	for _, f := range c.formats {
		if f.Key == key {
			return &f
		}
	}
	return nil
}

type headerKey struct{}

// HeaderInfo holds the identifiers at the start of a CII or CDAR document
// that determine its format.
type HeaderInfo struct {
	// Namespace of the root element.
	Namespace string
	// GuidelineID declares the specification the document follows.
	GuidelineID string
	// BusinessID declares the business process.
	BusinessID string
	// Err is set if the identifiers could not be read.
	Err error
}

// ReadHeader provides the root namespace and the guideline and business
// process identifiers of the input, reading them only once for all the
// converters that ask.
func ReadHeader(in *convert.Input) *HeaderInfo {
	if v, ok := in.Get(headerKey{}); ok {
		return v.(*HeaderInfo)
	}
	h := readHeader(in.Data)
	in.Set(headerKey{}, h)
	return h
}

// headerContext is the exchanged document context shared by CII invoices
// and CDARs, matched by local name.
type headerContext struct {
	Business *struct {
		ID string `xml:"ID"`
	} `xml:"BusinessProcessSpecifiedDocumentContextParameter"`
	Guideline *struct {
		ID string `xml:"ID"`
	} `xml:"GuidelineSpecifiedDocumentContextParameter"`
}

// readHeader reads the root element and its exchanged document context,
// stopping at the first other element.
func readHeader(data []byte) *HeaderInfo {
	h := new(HeaderInfo)
	d := xmlctx.NewXMLDecoder(bytes.NewReader(data))
	root := false
	for {
		tk, err := d.Token()
		if err == io.EOF {
			if !root {
				h.Err = ErrUnknownDocumentType
			}
			return h
		}
		if err != nil {
			h.Err = err
			return h
		}
		se, ok := tk.(xml.StartElement)
		if !ok {
			continue
		}
		if !root {
			h.Namespace = se.Name.Space
			root = true
			continue
		}
		if se.Name.Local != "ExchangedDocumentContext" {
			return h
		}
		hc := new(headerContext)
		if err := d.DecodeElement(hc, &se); err != nil {
			h.Err = err
			return h
		}
		if hc.Guideline != nil {
			h.GuidelineID = hc.Guideline.ID
		}
		if hc.Business != nil {
			h.BusinessID = hc.Business.ID
		}
		return h
	}
}
