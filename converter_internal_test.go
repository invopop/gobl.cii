package cii

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConverterUnknownKey(t *testing.T) {
	c := &converter{formats: []Format{FormatEN16931}}
	assert.True(t, c.Accepts(FormatEN16931.Key, nil))
	_, err := c.Export("cii+unknown", nil)
	assert.ErrorIs(t, err, ErrUnsupportedDocumentType)
}
