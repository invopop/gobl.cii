package cii

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseCDARDateTime pins that a malformed time portion is reported as an
// error rather than silently parsed as 00:00:00 (Copilot review on #57).
func TestParseCDARDateTime(t *testing.T) {
	t.Run("date only", func(t *testing.T) {
		d, tm, err := parseCDARDateTime("20260619")
		require.NoError(t, err)
		assert.Equal(t, "2026-06-19", d.String())
		assert.Nil(t, tm)
	})

	t.Run("date and time", func(t *testing.T) {
		d, tm, err := parseCDARDateTime("20260619151000")
		require.NoError(t, err)
		assert.Equal(t, "2026-06-19", d.String())
		require.NotNil(t, tm)
		assert.Equal(t, "15:10:00", tm.String())
	})

	t.Run("too short", func(t *testing.T) {
		_, _, err := parseCDARDateTime("2026")
		assert.Error(t, err)
	})

	t.Run("malformed date errors", func(t *testing.T) {
		_, _, err := parseCDARDateTime("20XX0619")
		assert.Error(t, err)
	})

	t.Run("malformed time errors (not silently 00:00:00)", func(t *testing.T) {
		_, _, err := parseCDARDateTime("20260619XX1000")
		assert.Error(t, err, "a non-numeric time portion must error, not fall back to midnight")
	})
}
