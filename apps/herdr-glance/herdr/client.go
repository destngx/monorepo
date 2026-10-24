// Package herdr talks to a running herdr server over its socket API:
// newline-delimited JSON on a Unix socket, one request and one reply per
// connection. Every herdr call in the app goes through this file, so a change
// in herdr's API touches only this package.
package herdr

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

// Status is herdr's agent state.
type Status string

const (
	StatusIdle    Status = "idle"
	StatusWorking Status = "working"
	StatusBlocked Status = "blocked"
	StatusDone    Status = "done"
	StatusUnknown Status = "unknown"
)

// Agent is one agent pane, with its workspace's label.
type Agent struct {
	PaneID      string `json:"paneId"`
	WorkspaceID string `json:"workspaceId"`
	Workspace   string `json:"workspace"`
	TabID       string `json:"tabId"`
	// Kind is the agent program herdr detected: claude, codex, and so on.
	Kind string `json:"kind"`
	// Title is the terminal title without the agent's spinner glyph.
	Title   string `json:"title"`
	Status  Status `json:"status"`
	Focused bool   `json:"focused"`
	Dir     string `json:"dir"`
}

// ErrUnreachable means nothing answered on the socket: herdr is not running,
// or the path is wrong.
var ErrUnreachable = errors.New("herdr is not reachable")

// Error is an error reply from herdr, such as not_found or agent_blocked.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return "herdr: " + e.Code + ": " + e.Message }

// Client calls one herdr server. It is safe for concurrent use.
type Client struct {
	socket  string
	timeout time.Duration
	seq     atomic.Uint64
}

// New returns a client for the socket at path (herdr's default is
// ~/.config/herdr/herdr.sock).
func New(path string) *Client {
	return &Client{socket: path, timeout: 5 * time.Second}
}

// Agents lists every agent pane, joined with its workspace's label.
func (c *Client) Agents(ctx context.Context) ([]Agent, error) {
	var list struct {
		Agents []struct {
			Agent       string `json:"agent"`
			Status      Status `json:"agent_status"`
			PaneID      string `json:"pane_id"`
			WorkspaceID string `json:"workspace_id"`
			TabID       string `json:"tab_id"`
			Title       string `json:"terminal_title_stripped"`
			Focused     bool   `json:"focused"`
			Dir         string `json:"cwd"`
		} `json:"agents"`
	}
	if err := c.call(ctx, "agent.list", struct{}{}, &list); err != nil {
		return nil, err
	}
	var spaces struct {
		Workspaces []struct {
			ID    string `json:"workspace_id"`
			Label string `json:"label"`
		} `json:"workspaces"`
	}
	if err := c.call(ctx, "workspace.list", struct{}{}, &spaces); err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(spaces.Workspaces))
	for _, w := range spaces.Workspaces {
		labels[w.ID] = w.Label
	}
	agents := make([]Agent, 0, len(list.Agents))
	for _, a := range list.Agents {
		agents = append(agents, Agent{
			PaneID: a.PaneID, WorkspaceID: a.WorkspaceID, Workspace: labels[a.WorkspaceID], TabID: a.TabID,
			Kind: a.Agent, Title: a.Title, Status: a.Status, Focused: a.Focused, Dir: a.Dir,
		})
	}
	return agents, nil
}

// Screen returns an agent's recent output as plain text: about one terminal
// screen. Lines are unwrapped, so the phone can wrap them to its own width.
// It sends no line count: herdr refuses one while the agent works
// (agent_not_idle), because reading further back means scrolling the agent's
// alternate screen.
func (c *Client) Screen(ctx context.Context, pane string) (string, error) {
	params := struct {
		Target    string `json:"target"`
		Source    string `json:"source"`
		StripANSI bool   `json:"strip_ansi"`
	}{pane, "recent_unwrapped", true}
	var out struct {
		Read struct {
			Text string `json:"text"`
		} `json:"read"`
	}
	if err := c.call(ctx, "agent.read", params, &out); err != nil {
		return "", err
	}
	return out.Read.Text, nil
}

// Prompt submits text to an agent as if typed and sent. herdr refuses it with
// agent_blocked while the agent waits for an answer.
func (c *Client) Prompt(ctx context.Context, pane, text string) error {
	params := struct {
		Target string `json:"target"`
		Text   string `json:"text"`
	}{pane, text}
	return c.call(ctx, "agent.prompt", params, nil)
}

// SendKeys presses keys in an agent's terminal, such as "1" or "enter".
func (c *Client) SendKeys(ctx context.Context, pane string, keys []string) error {
	params := struct {
		Target string   `json:"target"`
		Keys   []string `json:"keys"`
	}{pane, keys}
	return c.call(ctx, "agent.send_keys", params, nil)
}

// call sends one request and decodes the reply's result into out (nil to drop it).
func (c *Client) call(ctx context.Context, method string, params, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.socket)
	if err != nil {
		return fmt.Errorf("%w at %s: %w", ErrUnreachable, c.socket, err)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("herdr %s: %w", method, err)
		}
	}

	id := "herdr-glance-" + strconv.FormatUint(c.seq.Add(1), 10)
	line, err := json.Marshal(struct {
		ID     string `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}{id, method, params})
	if err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("herdr %s: send: %w", method, err)
	}
	// A screen read can be large, so the reader grows past bufio's default size.
	raw, err := bufio.NewReaderSize(conn, 64<<10).ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("herdr %s: read reply: %w", method, err)
	}
	var reply struct {
		ID     string          `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *Error          `json:"error"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return fmt.Errorf("herdr %s: decode reply: %w", method, err)
	}
	if reply.Error != nil {
		return reply.Error
	}
	if reply.ID != id {
		return fmt.Errorf("herdr %s: reply id %q, want %q", method, reply.ID, id)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(reply.Result, out); err != nil {
		return fmt.Errorf("herdr %s: decode result: %w", method, err)
	}
	return nil
}
