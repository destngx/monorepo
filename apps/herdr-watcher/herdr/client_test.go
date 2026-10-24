package herdr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

// request is one line a fake herdr server received.
type request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// fakeServer answers each request line with the reply its handler returns,
// the way herdr's socket does: one JSON object per line.
func fakeServer(t *testing.T, handle func(request) string) (string, func() []request) {
	t.Helper()
	// macOS caps a socket path near 104 bytes, so the folder stays short.
	dir, err := os.MkdirTemp("", "hm")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "s")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	var (
		mu   sync.Mutex
		seen []request
	)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			line, err := bufio.NewReader(conn).ReadBytes('\n')
			if err == nil {
				var req request
				if json.Unmarshal(line, &req) == nil {
					mu.Lock()
					seen = append(seen, req)
					mu.Unlock()
					// herdr replies on one line; fixtures are indented for reading.
					var out bytes.Buffer
					if json.Compact(&out, []byte(handle(req))) == nil {
						_, _ = conn.Write(append(out.Bytes(), '\n'))
					}
				}
			}
			_ = conn.Close()
		}
	}()
	return path, func() []request {
		mu.Lock()
		defer mu.Unlock()
		return append([]request(nil), seen...)
	}
}

const agentList = `{"id":"%s","result":{"type":"agent_list","agents":[
 {"agent":"claude","agent_status":"blocked","pane_id":"w1:p1","workspace_id":"w1","tab_id":"w1:t1","terminal_title":"✳ Fix the login form","terminal_title_stripped":"Fix the login form","focused":true,"cwd":"/home/someone/projects/example"},
 {"agent":"codex","agent_status":"working","pane_id":"w2:p4","workspace_id":"w2","tab_id":"w2:t1","terminal_title_stripped":"","cwd":"/home/someone/projects/other"}
]}}`

const workspaceList = `{"id":"%s","result":{"type":"workspace_list","workspaces":[
 {"workspace_id":"w1","label":"example"},{"workspace_id":"w2","label":"other"}
]}}`

func reply(format, id string) string {
	return fmt.Sprintf(format, id)
}

func TestAgentsJoinsWorkspaceLabels(t *testing.T) {
	path, _ := fakeServer(t, func(r request) string {
		switch r.Method {
		case "agent.list":
			return reply(agentList, r.ID)
		case "workspace.list":
			return reply(workspaceList, r.ID)
		}
		return `{"id":"` + r.ID + `","error":{"code":"unknown_method","message":"no"}}`
	})

	agents, err := New(path).Agents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Agent{
		{PaneID: "w1:p1", WorkspaceID: "w1", Workspace: "example", TabID: "w1:t1", Kind: "claude",
			Title: "Fix the login form", Status: StatusBlocked, Focused: true, Dir: "/home/someone/projects/example"},
		{PaneID: "w2:p4", WorkspaceID: "w2", Workspace: "other", TabID: "w2:t1", Kind: "codex",
			Title: "", Status: StatusWorking, Dir: "/home/someone/projects/other"},
	}
	if !reflect.DeepEqual(agents, want) {
		t.Fatalf("agents:\n got %+v\nwant %+v", agents, want)
	}
}

func TestScreenReadsUnwrappedPlainText(t *testing.T) {
	path, seen := fakeServer(t, func(r request) string {
		return `{"id":"` + r.ID + `","result":{"type":"pane_read","read":{"pane_id":"w1:p1","text":"line one\nline two"}}}`
	})

	text, err := New(path).Screen(context.Background(), "w1:p1")
	if err != nil {
		t.Fatal(err)
	}
	if text != "line one\nline two" {
		t.Fatalf("text = %q", text)
	}
	var params map[string]any
	if err := json.Unmarshal(seen()[0].Params, &params); err != nil {
		t.Fatal(err)
	}
	// No line count: herdr refuses one while the agent works (agent_not_idle),
	// because it would have to scroll the agent's alternate screen to read it.
	want := map[string]any{"target": "w1:p1", "source": "recent_unwrapped", "strip_ansi": true}
	if seen()[0].Method != "agent.read" || !reflect.DeepEqual(params, want) {
		t.Fatalf("sent %s %v", seen()[0].Method, params)
	}
}

func TestPromptAndKeysSendTheirParams(t *testing.T) {
	path, seen := fakeServer(t, func(r request) string {
		return `{"id":"` + r.ID + `","result":{"type":"ok"}}`
	})
	c := New(path)
	if err := c.Prompt(context.Background(), "w1:p1", "run the tests"); err != nil {
		t.Fatal(err)
	}
	if err := c.SendKeys(context.Background(), "w1:p1", []string{"1", "enter"}); err != nil {
		t.Fatal(err)
	}
	got := []string{seen()[0].Method, string(seen()[0].Params), seen()[1].Method, string(seen()[1].Params)}
	want := []string{
		"agent.prompt", `{"target":"w1:p1","text":"run the tests"}`,
		"agent.send_keys", `{"target":"w1:p1","keys":["1","enter"]}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sent %q", got)
	}
}

func TestHerdrErrorKeepsItsCode(t *testing.T) {
	path, _ := fakeServer(t, func(r request) string {
		return `{"id":"` + r.ID + `","error":{"code":"agent_blocked","message":"agent is blocked"}}`
	})

	err := New(path).Prompt(context.Background(), "w1:p1", "hello")
	var herr *Error
	if !errors.As(err, &herr) || herr.Code != "agent_blocked" || herr.Message != "agent is blocked" {
		t.Fatalf("err = %v", err)
	}
}

func TestMissingSocketIsUnreachable(t *testing.T) {
	_, err := New(filepath.Join(t.TempDir(), "none")).Agents(context.Background())
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v", err)
	}
}

func TestSilentServerTimesOut(t *testing.T) {
	dir, err := os.MkdirTemp("", "hm")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "s")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			time.Sleep(time.Second)
			_ = conn.Close()
		}
	}()

	c := New(path)
	c.timeout = 50 * time.Millisecond
	start := time.Now()
	if _, err := c.Agents(context.Background()); err == nil {
		t.Fatal("want a timeout error")
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("waited %v", time.Since(start))
	}
}
