package request_parser

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSplitMatchBodyAcceptsThreeBlocks(t *testing.T) {
	r := &http.Request{Method: http.MethodPost, Body: io.NopCloser(strings.NewReader("{}\n\t\nprovider\n\t\nchannel"))}
	parts := SplitMatchBody(r)
	if len(parts) != 3 || string(parts[2]) != "channel" {
		t.Fatalf("unexpected request blocks: %q", parts)
	}
}

func TestSplitMatchBodyRejectsOversize(t *testing.T) {
	r := &http.Request{Method: http.MethodPost, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", MaxMatchBodyBytes+1)))}
	if parts := SplitMatchBody(r); parts != nil {
		t.Fatal("oversized match request was accepted")
	}
}

func TestSplitMatchBodyRejectsOtherMethods(t *testing.T) {
	r := &http.Request{Method: http.MethodGet, Body: io.NopCloser(strings.NewReader("{}\n\t\n\n\t\nchannel"))}
	if parts := SplitMatchBody(r); parts != nil {
		t.Fatal("GET match request was accepted")
	}
}
