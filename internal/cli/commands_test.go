package cli

import "testing"

func TestNormalizeHelp(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"open", "--help"}} {
		cmd, err := Normalize(args)
		if err != nil {
			t.Fatalf("Normalize(%v): %v", args, err)
		}
		if cmd.Name != "help" {
			t.Fatalf("Normalize(%v) = %+v", args, cmd)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}, {"open", "--version"}} {
		cmd, err := Normalize(args)
		if err != nil {
			t.Fatalf("Normalize(%v): %v", args, err)
		}
		if cmd.Name != "version" {
			t.Fatalf("Normalize(%v) = %+v", args, cmd)
		}
	}
}

func TestNormalizeListCommand(t *testing.T) {
	cmd, err := Normalize([]string{"list"})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if cmd.Name != "list" {
		t.Fatalf("Command = %+v", cmd)
	}
}

func TestNormalizeBareHTMLPath(t *testing.T) {
	cmd, err := Normalize([]string{"/tmp/doc.html"})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if cmd.Name != "open" || cmd.File != "/tmp/doc.html" {
		t.Fatalf("Command = %+v", cmd)
	}
}

func TestNormalizeBareHTMPath(t *testing.T) {
	cmd, err := Normalize([]string{"/tmp/doc.htm"})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if cmd.Name != "open" || cmd.File != "/tmp/doc.htm" {
		t.Fatalf("Command = %+v", cmd)
	}
}

func TestNormalizePollFlags(t *testing.T) {
	cmd, err := Normalize([]string{"poll", "--json", "--agent-reply", "Done", "--timeout-ms", "50", "/tmp/doc.html"})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if cmd.Name != "poll" || !cmd.JSON || cmd.AgentReply != "Done" || cmd.Timeout == 0 {
		t.Fatalf("Command = %+v", cmd)
	}
}

func TestNormalizeGlobalPortAndNoOpen(t *testing.T) {
	cmd, err := Normalize([]string{"--port", "49001", "--no-open", "open", "/tmp/doc.html"})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if cmd.Port != 49001 || !cmd.NoOpen {
		t.Fatalf("Command = %+v", cmd)
	}
}

func TestNormalizeExposeExternal(t *testing.T) {
	cmd, err := Normalize([]string{"--expose-external", "open", "/tmp/doc.html"})
	if err != nil {
		t.Fatalf("Normalize global expose: %v", err)
	}
	if !cmd.Expose {
		t.Fatalf("Command = %+v", cmd)
	}
	cmd, err = Normalize([]string{"server", "--expose-external"})
	if err != nil {
		t.Fatalf("Normalize server expose: %v", err)
	}
	if cmd.Name != "server" || !cmd.Expose {
		t.Fatalf("Command = %+v", cmd)
	}
	if bindHost(cmd) != "0.0.0.0" {
		t.Fatalf("bindHost = %q", bindHost(cmd))
	}
}
