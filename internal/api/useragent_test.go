package api

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestCLITransport(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer srv.Close()

	t.Cleanup(func() { SetInvocation("dev", "") })
	SetInvocation("v9.9.9", "monitors apply")

	req, err := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-openstatus-key", "secret")
	client := &http.Client{Transport: NewTransport(nil)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if want := "openstatus-cli/v9.9.9 (" + runtime.GOOS + "; " + runtime.GOARCH + ")"; got.Get("User-Agent") != want {
		t.Errorf("User-Agent = %q, want %q", got.Get("User-Agent"), want)
	}
	if got.Get(HeaderCLICommand) != "monitors apply" {
		t.Errorf("%s = %q, want %q", HeaderCLICommand, got.Get(HeaderCLICommand), "monitors apply")
	}
	if id := got.Get(HeaderCLIInvocation); len(id) != 32 || id != invocationID {
		t.Errorf("%s = %q, want the process invocation ID", HeaderCLIInvocation, id)
	}
	if got.Get("x-openstatus-key") != "secret" {
		t.Error("existing headers must be preserved")
	}
	if req.Header.Get("User-Agent") != "" {
		t.Error("transport must not mutate the caller's request")
	}
}

func TestCLITransportWithoutCommand(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer srv.Close()

	t.Cleanup(func() { SetInvocation("dev", "") })
	SetInvocation("v9.9.9", "")

	resp, err := (&http.Client{Transport: NewTransport(nil)}).Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if _, ok := got[http.CanonicalHeaderKey(HeaderCLICommand)]; ok {
		t.Errorf("%s should be omitted when no command is set", HeaderCLICommand)
	}
}
