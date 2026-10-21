package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"sort"
	"sync"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Engine is the engine settings service (HW-OPS-9). The settings apply to
// every tenant on the Mac, so only engine admins read or change them, and
// only admins make other accounts admins.
type Engine struct {
	Deps
	Log *slog.Logger
	// DefaultMacOCRBin is the mac-ocr build next to the engine, used while
	// the setting is empty.
	DefaultMacOCRBin string
	// LocalAddrs lists this Mac's addresses; a new host must be one of them.
	LocalAddrs func() ([]netip.Addr, error)
	// Apply puts saved settings to work. Host, port and a larger batch
	// limit wait for the next start.
	Apply func(domain.Settings)
	// PingAI sends a small request to the AI provider of s.
	PingAI func(ctx context.Context, s domain.Settings) error

	// saveMu runs one Update at a time, so the stored and the live settings
	// never disagree.
	saveMu  sync.Mutex
	mu      sync.RWMutex
	current domain.Settings
	started domain.Settings
	stamp   domain.SettingsStamp
	// rejected are stored keys that failed the checks at Load.
	rejected []string
}

// Load reads the stored settings at start. A stored value that fails the
// checks keeps its default and is logged and shown on the page, so the
// engine still starts and an admin can fix it.
//
// legacy are values from the retired environment variables. When nothing
// is stored yet they are saved once, so an upgrade keeps the engine as it
// was; Load reports whether it imported them.
func (e *Engine) Load(ctx context.Context, legacy map[string]string) (domain.Settings, bool, error) {
	values, stamp, err := e.Store.Settings(ctx)
	if err != nil {
		return domain.Settings{}, false, err
	}
	imported := false
	if len(values) == 0 && len(legacy) > 0 {
		s, rejected := domain.SettingsFromValues(legacy)
		if len(rejected) > 0 {
			e.Log.Warn("retired environment variables not imported: their values fail the checks", "keys", rejected)
		}
		err := e.Store.InTx(ctx, func(r port.Repos) error {
			if err := r.SaveSettings(ctx, s.Values(), e.Clock(), ""); err != nil {
				return err
			}
			values, stamp, err = r.Settings(ctx)
			return err
		})
		if err != nil {
			return domain.Settings{}, false, err
		}
		imported = true
	}
	s, rejected := domain.SettingsFromValues(values)
	if len(rejected) > 0 {
		e.Log.Error("stored engine settings fail the checks; they keep their defaults until an admin saves", "keys", rejected)
	}
	e.mu.Lock()
	e.current, e.started, e.stamp, e.rejected = s, s, stamp, rejected
	e.mu.Unlock()
	return s, imported, nil
}

// Current returns the settings in force.
func (e *Engine) Current() domain.Settings {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current
}

// Listening is the address the engine listens on now.
func (e *Engine) Listening() (string, int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.started.Host, e.started.Port
}

// StartedOn records the address the engine listens on, when it had to fall
// back from the stored one.
func (e *Engine) StartedOn(host string, port int) {
	e.mu.Lock()
	e.started.Host, e.started.Port = host, port
	e.mu.Unlock()
}

// SettingsView is what the Engine settings page shows. The API key is never
// in it, only whether one is saved.
type SettingsView struct {
	Settings    domain.Settings `json:"settings"`
	AIAPIKeySet bool            `json:"aiApiKeySet"`
	AIModel     string          `json:"aiModel"`
	AIEffort    string          `json:"aiEffort"`
	// DefaultMacOCRBin is used while Settings.MacOCRBin is empty.
	DefaultMacOCRBin string `json:"defaultMacOcrBin"`
	// MacOCRFound says the binary in use exists and can run.
	MacOCRFound bool `json:"macOcrFound"`
	// Listening is the address the engine listens on now.
	Listening string `json:"listening"`
	// RestartFields are saved fields that wait for the next start.
	RestartFields []string `json:"restartFields"`
	// Rejected are fields whose stored value failed the checks at start and
	// use the default until the next save.
	Rejected []string             `json:"rejected"`
	Stamp    domain.SettingsStamp `json:"stamp"`
}

// SettingsInput changes every setting at once.
type SettingsInput struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	MaxFileMB        int    `json:"maxFileMb"`
	MaxBatchMB       int    `json:"maxBatchMb"`
	MacOCRBin        string `json:"macOcrBin"`
	Workers          int    `json:"workers"`
	AIBaseURL        string `json:"aiBaseUrl"`
	AITimeoutSeconds int    `json:"aiTimeoutSeconds"`
	// AIAPIKey nil keeps the saved key; "" removes it.
	AIAPIKey *string `json:"aiApiKey"`
}

// AICheck is the outcome of a test request to the AI provider.
type AICheck struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (e *Engine) requireAdmin(ctx context.Context, userID string) error {
	u, _, err := e.Store.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.Admin {
		return domain.Forbidden("only an engine admin can see or change engine settings")
	}
	return nil
}

// View returns the settings page.
func (e *Engine) View(ctx context.Context, userID string) (SettingsView, error) {
	if err := e.requireAdmin(ctx, userID); err != nil {
		return SettingsView{}, err
	}
	return e.view(), nil
}

func (e *Engine) view() SettingsView {
	e.mu.RLock()
	s, started, stamp := e.current, e.started, e.stamp
	rejected := make([]string, 0, len(e.rejected))
	for _, k := range e.rejected {
		rejected = append(rejected, domain.SettingField(k))
	}
	e.mu.RUnlock()
	bin := s.MacOCRBin
	if bin == "" {
		bin = e.DefaultMacOCRBin
	}
	return SettingsView{Settings: s, AIAPIKeySet: s.AIAPIKey != "", AIModel: domain.AIModel, AIEffort: domain.AIEffort,
		DefaultMacOCRBin: e.DefaultMacOCRBin, MacOCRFound: executable(bin) == nil, Listening: started.Addr(),
		RestartFields: append([]string{}, s.RestartFields(started)...), Rejected: rejected, Stamp: stamp}
}

// Update checks and saves every setting, then applies what can change while
// the engine runs. Only the names of changed settings are logged.
func (e *Engine) Update(ctx context.Context, userID string, in SettingsInput) (SettingsView, error) {
	if err := e.requireAdmin(ctx, userID); err != nil {
		return SettingsView{}, err
	}
	e.saveMu.Lock()
	defer e.saveMu.Unlock()
	old := e.Current()
	next := domain.Settings{Host: in.Host, Port: in.Port, MaxFileMB: in.MaxFileMB, MaxBatchMB: in.MaxBatchMB, MacOCRBin: in.MacOCRBin,
		Workers: in.Workers, AIBaseURL: in.AIBaseURL, AIAPIKey: old.AIAPIKey, AITimeoutSeconds: in.AITimeoutSeconds}
	if in.AIAPIKey != nil {
		next.AIAPIKey = *in.AIAPIKey
	}
	next, err := next.Validate()
	if err != nil {
		return SettingsView{}, err
	}
	if next.Host != old.Host {
		if err := e.checkHost(next.Host); err != nil {
			return SettingsView{}, err
		}
	}
	if next.MacOCRBin != "" && next.MacOCRBin != old.MacOCRBin {
		if err := executable(next.MacOCRBin); err != nil {
			return SettingsView{}, domain.Invalid("mac-ocr path: %v", err)
		}
	}
	changed := changedKeys(old.Values(), next.Values())
	e.mu.RLock()
	rejected := len(e.rejected) > 0
	e.mu.RUnlock()
	// A save also rewrites stored values that failed the checks at start.
	if len(changed) == 0 && !rejected {
		return e.view(), nil
	}
	var stamp domain.SettingsStamp
	err = e.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.SaveSettings(ctx, next.Values(), e.Clock(), userID); err != nil {
			return err
		}
		_, stamp, err = r.Settings(ctx)
		return err
	})
	if err != nil {
		return SettingsView{}, err
	}
	e.mu.Lock()
	e.current, e.stamp, e.rejected = next, stamp, nil
	e.mu.Unlock()
	if e.Apply != nil {
		e.Apply(next)
	}
	e.Log.Info("engine settings changed", "user_id", userID, "keys", changed)
	return e.view(), nil
}

func (e *Engine) checkHost(host string) error {
	addrs, err := e.LocalAddrs()
	if err != nil {
		return fmt.Errorf("list this Mac's addresses: %w", err)
	}
	want := netip.MustParseAddr(host)
	for _, a := range addrs {
		if a.Unmap() == want.Unmap() {
			return nil
		}
	}
	return domain.Invalid("%s is not an address of this Mac; use 127.0.0.1 or the Mac's Tailscale address", host)
}

// executable checks that path is a regular file the engine may run.
func executable(path string) error {
	if path == "" {
		return errors.New("no path")
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s does not exist", path)
		}
		return fmt.Errorf("check %s: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%s is not a program", path)
	}
	return nil
}

func changedKeys(old, next map[string]string) []string {
	var out []string
	for k, v := range next {
		if old[k] != v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// CheckAI sends a small request with the saved AI settings.
func (e *Engine) CheckAI(ctx context.Context, userID string) (AICheck, error) {
	if err := e.requireAdmin(ctx, userID); err != nil {
		return AICheck{}, err
	}
	s := e.Current()
	if !s.AIEnabled() {
		return AICheck{}, domain.Conflict("save an AI address first")
	}
	if err := e.PingAI(ctx, s); err != nil {
		return AICheck{Message: err.Error()}, nil
	}
	return AICheck{OK: true, Message: fmt.Sprintf("%s answered at %s effort", domain.AIModel, domain.AIEffort)}, nil
}

// Users lists every account, for granting admin.
func (e *Engine) Users(ctx context.Context, userID string) ([]domain.User, error) {
	if err := e.requireAdmin(ctx, userID); err != nil {
		return nil, err
	}
	return e.Store.ListUsers(ctx)
}

// SetAdmin grants or removes engine admin. The last admin stays.
func (e *Engine) SetAdmin(ctx context.Context, userID, targetID string, admin bool) (domain.User, error) {
	if err := e.requireAdmin(ctx, userID); err != nil {
		return domain.User{}, err
	}
	var out domain.User
	err := e.Store.InTx(ctx, func(r port.Repos) error {
		u, _, err := r.UserByID(ctx, targetID)
		if err != nil {
			return err
		}
		if u.Admin && !admin {
			n, err := r.CountAdmins(ctx)
			if err != nil {
				return err
			}
			if n <= 1 {
				return domain.Conflict("keep at least one engine admin")
			}
		}
		if err := r.SetUserAdmin(ctx, targetID, admin); err != nil {
			return err
		}
		u.Admin = admin
		out = u
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}
	e.Log.Info("engine admin changed", "user_id", userID, "target_id", targetID, "admin", admin)
	return out, nil
}
