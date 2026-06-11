package statusreport_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func captureRequestBody(req *http.Request, dst *[]byte) error {
	if req.Body == nil {
		return nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	*dst = body
	req.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

func decodeBody[T proto.Message](t *testing.T, body []byte, into T) T {
	t.Helper()
	if len(body) == 0 {
		t.Fatal("expected non-empty request body")
	}
	if err := protojson.Unmarshal(body, into); err != nil {
		t.Fatalf("failed to unmarshal request body: %v\nbody: %s", err, string(body))
	}
	return into
}
