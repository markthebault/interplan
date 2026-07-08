package session

import "testing"

func TestKeyIsStableShortSHA(t *testing.T) {
	path := "/tmp/doc.html"
	got := Key(path)
	if len(got) != 16 {
		t.Fatalf("Key length = %d, want 16", len(got))
	}
	if got != Key(path) {
		t.Fatal("Key is not stable")
	}
	if got == Key("/tmp/other.html") {
		t.Fatal("Different paths should not produce the same short key in this fixture")
	}
}

func TestURLForHost(t *testing.T) {
	if got := URLForHost("abc123", "192.168.1.20", 49001); got != "http://192.168.1.20:49001/session/abc123" {
		t.Fatalf("URLForHost = %q", got)
	}
	if got := URLForHost("abc123", "", 49001); got != "http://127.0.0.1:49001/session/abc123" {
		t.Fatalf("URLForHost default = %q", got)
	}
}
