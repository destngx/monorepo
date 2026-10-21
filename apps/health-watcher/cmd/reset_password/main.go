// Command reset_password sets a new password for an account and signs it
// out everywhere (HW-AUTH-1). It runs on the Mac that holds the data:
//
//	pnpm exec nx run health-watcher:reset-password -- --email someone@example.com
//
// The new password is read from the terminal twice, never from arguments.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"apps/health-watcher/internal/app"
	"apps/health-watcher/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "reset-password:", err)
		os.Exit(1)
	}
}

func run() error {
	email := flag.String("email", "", "the account's email")
	flag.Parse()
	if strings.TrimSpace(*email) == "" {
		return errors.New("--email is required")
	}
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	first, err := readPassword("New password: ")
	if err != nil {
		return err
	}
	second, err := readPassword("Again: ")
	if err != nil {
		return err
	}
	if first != second {
		return errors.New("the two passwords differ")
	}
	ctx := context.Background()
	engine, err := app.Open(ctx, cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		return err
	}
	defer engine.Close()
	if err := engine.Services.Accounts.ResetPassword(ctx, *email, first); err != nil {
		return err
	}
	fmt.Println("Password changed. Every session of this account is signed out.")
	return nil
}

// readPassword reads a line from stdin, with echo turned off when stdin is
// a terminal (stty fails on a pipe, and the line is read as it is).
func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	echoOff := stty("-echo") == nil
	line, err := stdin.ReadString('\n')
	if echoOff {
		_ = stty("echo")
		fmt.Fprintln(os.Stderr)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

var stdin = bufio.NewReader(os.Stdin)

func stty(arg string) error {
	cmd := exec.Command("stty", arg)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
