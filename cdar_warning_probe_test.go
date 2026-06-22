package cii_test

import (
	"context"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/phive"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestCDARWarningProbe deliberately violates the warning-flagged
// BR-FR-04/MDT-91 rule (invalid referenced-document type code) and
// asserts Phive surfaces it, proving the zero-warning assertion in
// TestCDARSchematron actually exercises the warning channel.
func TestCDARWarningProbe(t *testing.T) {
	if !*validate {
		t.Skip("requires -validate flag and a running Phive gRPC service")
	}

	conn, err := grpc.NewClient("localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close() //nolint:errcheck
	pc := phive.NewValidationServiceClient(conn)

	st := buildSyntheticStatus(t, "205")
	st.Lines[0].Doc.Type = "999" // not a UNTDID 1001 invoice type

	cdar, err := cii.NewCDARFromStatus(st, cii.ContextCDARFlow6)
	require.NoError(t, err)
	data, err := cdar.Bytes()
	require.NoError(t, err)

	resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
		Vesid:      cii.ContextCDARFlow6.VESID,
		XmlContent: data,
	})
	require.NoError(t, err)

	var total int
	for _, layer := range resp.Results {
		total += len(layer.Warnings) + len(layer.Errors)
	}
	require.NotZero(t, total,
		"expected the invalid type code to surface as a warning or error — the warning channel may be silently broken")
	t.Logf("probe surfaced %d finding(s); success=%v", total, resp.Success)
	for _, layer := range resp.Results {
		for _, w := range layer.Warnings {
			t.Logf("WARNING [%s] %s", w.TestId, w.Message)
		}
		for _, e := range layer.Errors {
			t.Logf("ERROR [%s] %s", e.TestId, e.Message)
		}
	}
}
