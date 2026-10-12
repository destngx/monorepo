package cache

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// fakeUpstash records each command and answers with the reply for its verb.
type fakeUpstash struct {
	t        *testing.T
	commands [][]string
	reply    func(command []string) string
}

func (f *fakeUpstash) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/" {
		f.t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}
	if r.Header.Get("Authorization") != "Bearer test-token" {
		f.t.Fatalf("missing bearer token")
	}
	var command []string
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		f.t.Fatalf("expected a JSON array body: %v", err)
	}
	f.commands = append(f.commands, command)
	_, _ = w.Write([]byte(f.reply(command)))
}

func newTestClient(t *testing.T, reply func([]string) string) (*Client, *fakeUpstash) {
	fake := &fakeUpstash{t: t, reply: reply}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	client, err := NewClient(domain.CacheConfig{RESTURL: server.URL, RESTToken: "test-token", KeyPrefix: "wm:"}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("failed to build cache client: %v", err)
	}
	return client, fake
}

func TestSetSendsValueInBodyWithPrefixAndTTL(t *testing.T) {
	client, fake := newTestClient(t, func([]string) string { return `{"result":"OK"}` })
	large := strings.Repeat("x", 10_000)
	if err := client.Set(context.Background(), "accounts:all", large, 300); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := []string{"SET", "wm:accounts:all", large, "EX", "300"}
	if !reflect.DeepEqual(fake.commands[0], want) {
		t.Fatalf("got %v", fake.commands[0][:2])
	}
}

func TestGetReturnsValueOrMiss(t *testing.T) {
	client, _ := newTestClient(t, func(command []string) string {
		if command[1] == "wm:hit" {
			return `{"result":"42"}`
		}
		return `{"result":null}`
	})
	value, found, err := client.Get(context.Background(), "hit")
	if err != nil || !found || value != "42" {
		t.Fatalf("hit: %q %v %v", value, found, err)
	}
	if _, found, err := client.Get(context.Background(), "miss"); err != nil || found {
		t.Fatalf("miss: %v %v", found, err)
	}
}

func TestScanFollowsCursorAndStripsPrefix(t *testing.T) {
	client, fake := newTestClient(t, func(command []string) string {
		if command[1] == "0" {
			return `{"result":["17",["wm:budget:2026","wm:budget:2025"]]}`
		}
		return `{"result":["0",["wm:budget:2024"]]}`
	})
	keys, err := client.Scan(context.Background(), "budget:*")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !reflect.DeepEqual(keys, []string{"budget:2026", "budget:2025", "budget:2024"}) {
		t.Fatalf("keys %v", keys)
	}
	if fake.commands[0][3] != "wm:budget:*" || fake.commands[1][1] != "17" {
		t.Fatalf("unexpected scan commands %v", fake.commands)
	}
}

func TestDeleteSendsOneCommandForAllKeys(t *testing.T) {
	client, fake := newTestClient(t, func([]string) string { return `{"result":2}` })
	deleted, err := client.Delete(context.Background(), "a", "b")
	if err != nil || deleted != 2 {
		t.Fatalf("delete: %d %v", deleted, err)
	}
	if !reflect.DeepEqual(fake.commands[0], []string{"DEL", "wm:a", "wm:b"}) {
		t.Fatalf("command %v", fake.commands[0])
	}
	if n, err := client.Delete(context.Background()); n != 0 || err != nil || len(fake.commands) != 1 {
		t.Fatal("expected no request for an empty delete")
	}
}

func TestCommandErrorsSurface(t *testing.T) {
	client, _ := newTestClient(t, func([]string) string { return `{"error":"WRONGPASS invalid password"}` })
	if err := client.Ping(context.Background()); err == nil || !strings.Contains(err.Error(), "WRONGPASS") {
		t.Fatalf("expected upstash error, got %v", err)
	}
}
