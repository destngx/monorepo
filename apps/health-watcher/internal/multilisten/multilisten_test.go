package multilisten

import (
	"errors"
	"io"
	"net"
	"testing"
)

func TestGivenTwoListenersWhenJoinedThenConnectionsToEitherAreAcceptedUntilClosed(t *testing.T) {
	a, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := New(a, b)
	if l.Addr() != a.Addr() || len(l.Addrs()) != 2 {
		t.Fatalf("addrs = %v", l.Addrs())
	}
	for _, ln := range []net.Listener{a, b} {
		go func() {
			conn, err := net.Dial("tcp", ln.Addr().String())
			if err == nil {
				_, _ = conn.Write([]byte("hi"))
				_ = conn.Close()
			}
		}()
		conn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(conn)
		_ = conn.Close()
		if string(got) != "hi" {
			t.Fatalf("read %q", got)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Accept(); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("accept after close = %v", err)
	}
	if _, err := net.Dial("tcp", b.Addr().String()); err == nil {
		t.Fatal("second listener still open")
	}
}
