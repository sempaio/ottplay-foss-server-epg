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

func TestSplitMatchBodyAcceptsTwentyThousandChannels(t *testing.T) {
	var body strings.Builder
	body.WriteString("{}\n\t\n\n\t\n")
	for i := 0; i < 20000; i++ {
		if i != 0 {
			body.WriteByte('\n')
		}
		body.WriteString("123456789-123456789-123456789-123456789")
	}
	r := &http.Request{Method: http.MethodPost, Body: io.NopCloser(strings.NewReader(body.String()))}
	parts := SplitMatchBody(r)
	if len(parts) != 3 {
		t.Fatalf("expected three blocks, got %d", len(parts))
	}
	if got := strings.Count(string(parts[2]), "\n") + 1; got != 20000 {
		t.Fatalf("expected 20000 channels, got %d", got)
	}
}
