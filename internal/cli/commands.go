package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/markthebault/interplan/internal/platform"
	"github.com/markthebault/interplan/internal/protocol"
	"github.com/markthebault/interplan/internal/server"
	"github.com/markthebault/interplan/internal/session"
)

const defaultPort = 37917

type Command struct {
	Name       string
	File       string
	JSON       bool
	Reopen     bool
	NoOpen     bool
	Port       int
	AgentReply string
	Timeout    time.Duration
}

func Normalize(args []string) (Command, error) {
	cmd := Command{Name: "list", Port: portFromEnv()}
	var err error
	args, err = parseGlobalFlags(args, &cmd)
	if err != nil {
		return cmd, err
	}
	if len(args) == 0 {
		return cmd, nil
	}
	known := map[string]bool{"open": true, "poll": true, "end": true, "server": true, "stop": true}
	if !known[args[0]] && isHTMLFile(args[0]) {
		args = append([]string{"open"}, args...)
	}
	cmd.Name = args[0]
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&cmd.JSON, "json", cmd.JSON, "print JSON")
	fs.BoolVar(&cmd.Reopen, "reopen", false, "reopen user-ended sessions")
	fs.BoolVar(&cmd.NoOpen, "no-open", cmd.NoOpen, "disable browser opening")
	fs.IntVar(&cmd.Port, "port", cmd.Port, "server port")
	fs.StringVar(&cmd.AgentReply, "agent-reply", "", "append agent reply")
	timeout := fs.Int("timeout-ms", 0, "poll timeout in milliseconds")
	if err := fs.Parse(args[1:]); err != nil {
		return cmd, err
	}
	if *timeout > 0 {
		cmd.Timeout = time.Duration(*timeout) * time.Millisecond
	}
	if fs.NArg() > 0 {
		cmd.File = fs.Arg(0)
	}
	if _, ok := known[cmd.Name]; !ok {
		return cmd, fmt.Errorf("unknown command %q", cmd.Name)
	}
	if requiresFile(cmd.Name) && cmd.File == "" {
		return cmd, fmt.Errorf("%s requires an html file", cmd.Name)
	}
	if requiresFile(cmd.Name) && !isHTMLFile(cmd.File) {
		return cmd, fmt.Errorf("%s requires a .html or .htm file", cmd.Name)
	}
	return cmd, nil
}

func Run(args []string, stdout, stderr io.Writer) error {
	cmd, err := Normalize(args)
	if err != nil {
		return err
	}
	store, err := defaultStore()
	if err != nil {
		return err
	}
	switch cmd.Name {
	case "list":
		return runList(stdout, store, cmd.JSON)
	case "open":
		return runOpen(stdout, stderr, cmd)
	case "poll":
		return runPoll(stdout, stderr, cmd)
	case "end":
		return runEnd(stdout, stderr, cmd)
	case "server":
		return server.Serve("127.0.0.1:"+strconv.Itoa(cmd.Port), store)
	case "stop":
		return fmt.Errorf("stop is not implemented yet")
	default:
		return fmt.Errorf("unhandled command %q", cmd.Name)
	}
}

func runList(stdout io.Writer, store *session.Store, asJSON bool) error {
	state, err := store.Load()
	if err != nil {
		return err
	}
	out := protocol.SessionListResponse{NextStep: "Run `interplan <artifact.html>` to open a review session."}
	for _, s := range state.Sessions {
		out.Sessions = append(out.Sessions, protocol.SessionInfo{File: s.File, URL: s.URL, Status: s.Status})
	}
	return writeOutput(stdout, out, asJSON)
}

func runOpen(stdout, stderr io.Writer, cmd Command) error {
	file, err := session.CanonicalPath(cmd.File)
	if err != nil {
		return err
	}
	if err := ensureServer(cmd, stderr); err != nil {
		return err
	}
	client := newAPIClient(cmd.Port)
	out, status, err := client.open(file, cmd.Reopen)
	if status == 409 {
		_ = writeOutput(stdout, out, cmd.JSON)
		return nil
	}
	if err != nil {
		return err
	}
	if !cmd.NoOpen && os.Getenv("INTERPLAN_NO_OPEN") != "1" {
		if err := platform.OpenBrowser(out.Session.URL); err != nil {
			fmt.Fprintf(stderr, "interplan: could not open browser: %v\n", err)
		}
	}
	return writeOutput(stdout, out, cmd.JSON)
}

func runPoll(stdout, stderr io.Writer, cmd Command) error {
	file, err := session.CanonicalPath(cmd.File)
	if err != nil {
		return err
	}
	if err := ensureServer(cmd, stderr); err != nil {
		return err
	}
	client := newAPIClient(cmd.Port)
	if cmd.AgentReply != "" {
		if err := client.agentReply(file, cmd.AgentReply); err != nil {
			return err
		}
	}
	poll, err := client.poll(file, cmd.Timeout)
	if err != nil {
		return err
	}
	return writeOutput(stdout, poll, cmd.JSON)
}

func runEnd(stdout, stderr io.Writer, cmd Command) error {
	file, err := session.CanonicalPath(cmd.File)
	if err != nil {
		return err
	}
	if err := ensureServer(cmd, stderr); err != nil {
		return err
	}
	out, err := newAPIClient(cmd.Port).end(file)
	if err != nil {
		return err
	}
	return writeOutput(stdout, out, cmd.JSON)
}

func defaultStore() (*session.Store, error) {
	path, err := platform.StateFile()
	if err != nil {
		return nil, err
	}
	return session.NewStore(path), nil
}

func portFromEnv() int {
	if raw := os.Getenv("INTERPLAN_PORT"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			return p
		}
	}
	return defaultPort
}

func requiresFile(name string) bool {
	return name == "open" || name == "poll" || name == "end"
}

func isHTMLFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}

func parseGlobalFlags(args []string, cmd *Command) ([]string, error) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			cmd.JSON = true
		case arg == "--no-open":
			cmd.NoOpen = true
		case arg == "--port":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("--port requires a value")
			}
			p, err := strconv.Atoi(args[i])
			if err != nil || p <= 0 {
				return nil, fmt.Errorf("--port requires a positive integer")
			}
			cmd.Port = p
		case strings.HasPrefix(arg, "--port="):
			p, err := strconv.Atoi(strings.TrimPrefix(arg, "--port="))
			if err != nil || p <= 0 {
				return nil, fmt.Errorf("--port requires a positive integer")
			}
			cmd.Port = p
		default:
			out = append(out, arg)
		}
	}
	return out, nil
}

func ensureServer(cmd Command, stderr io.Writer) error {
	client := newAPIClient(cmd.Port)
	if client.health() {
		return nil
	}
	exe, err := platform.CurrentExecutable()
	if err != nil {
		return err
	}
	if err := platform.StartDetached(exe, "server", "--port", strconv.Itoa(cmd.Port)); err != nil {
		return err
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if client.health() {
			return nil
		}
	}
	return fmt.Errorf("server did not become healthy on port %d", cmd.Port)
}
