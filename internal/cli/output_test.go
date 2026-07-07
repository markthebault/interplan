package cli

import (
	"bytes"
	"testing"

	"github.com/markthebault/interplan/internal/protocol"
	toon "github.com/toon-format/toon-go"
)

func TestWriteOutputDefaultIsDecodableTOON(t *testing.T) {
	var buf bytes.Buffer
	err := writeOutput(&buf, protocol.PollResponse{
		Session: protocol.PollSessionInfo{
			File:         "/tmp/doc.html",
			Status:       "feedback",
			SessionEnded: true,
			EndedBy:      "user",
		},
		DOMSnapshot: "h1 Draft",
		Prompts:     []protocol.Prompt{{Tag: "message", Prompt: "Change the title."}},
		NextStep:    "Apply final feedback, stop polling, do not reopen.",
	}, false)
	if err != nil {
		t.Fatalf("writeOutput: %v", err)
	}
	if _, err := toon.Decode(buf.Bytes(), toon.WithStrictMode(true), toon.WithDecoderIndent(2)); err != nil {
		t.Fatalf("default output is not strict TOON: %v\n%s", err, buf.String())
	}
}
