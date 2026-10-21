package domain

import (
	"net/netip"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// The model and reasoning effort are fixed, not settings: readings were
// measured with this pair (D6), and the gateway account serves few models.
const (
	AIModel  = "gpt-6-luna"
	AIEffort = "low"
)

// Settings are the engine's settings. They live in the database and an
// engine admin edits them on the Engine settings page (HW-OPS-9). Only the
// data folder stays outside, because the database lives in it.
type Settings struct {
	// Host and Port are the listen address. They apply on the next start.
	Host string `json:"host"`
	Port int    `json:"port"`
	// MaxFileMB and MaxBatchMB are the upload limits (HW-IMP-2).
	MaxFileMB  int `json:"maxFileMb"`
	MaxBatchMB int `json:"maxBatchMb"`
	// MacOCRBin is the mac-ocr binary; "" means the build next to the engine.
	MacOCRBin string `json:"macOcrBin"`
	// Workers is how many background jobs run at once.
	Workers int `json:"workers"`
	// AIBaseURL is the Responses API base; "" turns AI reading off.
	AIBaseURL string `json:"aiBaseUrl"`
	// AIAPIKey is never sent back to a browser.
	AIAPIKey         string `json:"-"`
	AITimeoutSeconds int    `json:"aiTimeoutSeconds"`
}

// Setting ranges.
const (
	MinPort, MaxPort                 = 1024, 65535
	MaxFileMBLimit                   = 1024
	MaxBatchMBLimit                  = 4096
	MinWorkers, MaxWorkers           = 1, 8
	MinAITimeout, MaxAITimeout       = 5, 1800
	maxAPIKeyLength                  = 4096
	defaultHost, defaultPort         = "127.0.0.1", 9191
	defaultWorkers, defaultAITimeout = 2, 180
)

// DefaultSettings are the settings of a new engine.
func DefaultSettings() Settings {
	return Settings{Host: defaultHost, Port: defaultPort, MaxFileMB: DefaultMaxFileMB, MaxBatchMB: DefaultMaxBatchMB,
		Workers: defaultWorkers, AITimeoutSeconds: defaultAITimeout}
}

// AIEnabled reports whether AI reading is configured.
func (s Settings) AIEnabled() bool { return s.AIBaseURL != "" }

// Addr is the listen address.
func (s Settings) Addr() string {
	return netip.AddrPortFrom(netip.MustParseAddr(s.Host), uint16(s.Port)).String()
}

// Validate cleans s and checks every field. Whether the host is an address
// of this Mac and the binary exists are checked by the service.
func (s Settings) Validate() (Settings, error) {
	s.Host = strings.TrimSpace(s.Host)
	addr, err := netip.ParseAddr(s.Host)
	if err != nil || addr.Zone() != "" {
		return s, Invalid("host must be an IP address such as 127.0.0.1")
	}
	if addr.IsUnspecified() {
		return s, Invalid("host must be one address, not %s: listening on every network would expose the engine", s.Host)
	}
	s.Host = addr.String()
	if s.Port < MinPort || s.Port > MaxPort {
		return s, Invalid("port must be from %d to %d", MinPort, MaxPort)
	}
	if s.MaxFileMB < 1 || s.MaxFileMB > MaxFileMBLimit {
		return s, Invalid("largest file must be from 1 to %d MB", MaxFileMBLimit)
	}
	if s.MaxBatchMB < 1 || s.MaxBatchMB > MaxBatchMBLimit {
		return s, Invalid("largest upload must be from 1 to %d MB", MaxBatchMBLimit)
	}
	if s.MaxBatchMB < s.MaxFileMB {
		return s, Invalid("largest upload (%d MB) is smaller than largest file (%d MB)", s.MaxBatchMB, s.MaxFileMB)
	}
	s.MacOCRBin = strings.TrimSpace(s.MacOCRBin)
	if s.MacOCRBin != "" {
		if !filepath.IsAbs(s.MacOCRBin) {
			return s, Invalid("mac-ocr path must be a full path starting with /")
		}
		s.MacOCRBin = filepath.Clean(s.MacOCRBin)
		// Only a program named mac-ocr: the setting must not run any program on the Mac.
		if filepath.Base(s.MacOCRBin) != "mac-ocr" {
			return s, Invalid("mac-ocr path must end in /mac-ocr")
		}
	}
	if s.Workers < MinWorkers || s.Workers > MaxWorkers {
		return s, Invalid("background jobs must be from %d to %d", MinWorkers, MaxWorkers)
	}
	if s.AIBaseURL, err = cleanBaseURL(s.AIBaseURL); err != nil {
		return s, err
	}
	s.AIAPIKey = strings.TrimSpace(s.AIAPIKey)
	if len(s.AIAPIKey) > maxAPIKeyLength || strings.IndexFunc(s.AIAPIKey, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return s, Invalid("API key must be one line without spaces, at most %d characters", maxAPIKeyLength)
	}
	if s.AITimeoutSeconds < MinAITimeout || s.AITimeoutSeconds > MaxAITimeout {
		return s, Invalid("AI time limit must be from %d to %d seconds", MinAITimeout, MaxAITimeout)
	}
	return s, nil
}

func cleanBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return raw, Invalid("AI address must be an http or https URL, such as http://localhost:8080/v1")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return raw, Invalid("AI address must not carry a login, query or fragment; put a key in the API key field")
	}
	return raw, nil
}

// RestartFields lists the fields of s that wait for the next start, given
// the settings the engine started with. A smaller upload limit applies at
// once; a larger one needs the server's body limit raised at start.
func (s Settings) RestartFields(started Settings) []string {
	var out []string
	if s.Host != started.Host {
		out = append(out, "host")
	}
	if s.Port != started.Port {
		out = append(out, "port")
	}
	if s.MaxBatchMB > started.MaxBatchMB {
		out = append(out, "maxBatchMb")
	}
	return out
}

// Stored setting keys, in the order they are read.
const (
	settingHost       = "host"
	settingPort       = "port"
	settingMaxFileMB  = "max_file_mb"
	settingMaxBatchMB = "max_batch_mb"
	settingMacOCRBin  = "mac_ocr_bin"
	settingWorkers    = "workers"
	settingAIBaseURL  = "ai_base_url"
	settingAIAPIKey   = "ai_api_key"
	settingAITimeout  = "ai_timeout_seconds"
)

var settingKeys = []string{settingHost, settingPort, settingMaxFileMB, settingMaxBatchMB, settingMacOCRBin, settingWorkers,
	settingAIBaseURL, settingAIAPIKey, settingAITimeout}

// SettingField is the JSON field name of a stored key, for telling the
// page which stored values were rejected.
func SettingField(key string) string {
	return map[string]string{settingHost: "host", settingPort: "port", settingMaxFileMB: "maxFileMb", settingMaxBatchMB: "maxBatchMb",
		settingMacOCRBin: "macOcrBin", settingWorkers: "workers", settingAIBaseURL: "aiBaseUrl", settingAIAPIKey: "aiApiKey",
		settingAITimeout: "aiTimeoutSeconds"}[key]
}

// Values are the stored form of s, one entry per key.
func (s Settings) Values() map[string]string {
	return map[string]string{
		settingHost: s.Host, settingPort: strconv.Itoa(s.Port), settingMaxFileMB: strconv.Itoa(s.MaxFileMB),
		settingMaxBatchMB: strconv.Itoa(s.MaxBatchMB), settingMacOCRBin: s.MacOCRBin, settingWorkers: strconv.Itoa(s.Workers),
		settingAIBaseURL: s.AIBaseURL, settingAIAPIKey: s.AIAPIKey, settingAITimeout: strconv.Itoa(s.AITimeoutSeconds),
	}
}

// set puts one stored value into s; false when a number does not parse.
func (s *Settings) set(key, value string) bool {
	str := map[string]*string{settingHost: &s.Host, settingMacOCRBin: &s.MacOCRBin, settingAIBaseURL: &s.AIBaseURL, settingAIAPIKey: &s.AIAPIKey}
	num := map[string]*int{settingPort: &s.Port, settingMaxFileMB: &s.MaxFileMB, settingMaxBatchMB: &s.MaxBatchMB,
		settingWorkers: &s.Workers, settingAITimeout: &s.AITimeoutSeconds}
	if p, ok := str[key]; ok {
		*p = value
		return true
	}
	if p, ok := num[key]; ok {
		n, err := strconv.Atoi(strings.TrimSpace(value))
		*p = n
		return err == nil
	}
	return true
}

// SettingsFromValues reads stored values over the defaults; unknown keys
// are ignored. A value that fails the checks, for example after a hand edit
// of the database, keeps its default and its key is returned, so one bad
// value never resets the others (the API key above all).
func SettingsFromValues(values map[string]string) (Settings, []string) {
	all := DefaultSettings()
	parsed := true
	for _, k := range settingKeys {
		if v, ok := values[k]; ok {
			parsed = all.set(k, v) && parsed
		}
	}
	if clean, err := all.Validate(); parsed && err == nil {
		return clean, nil
	}
	// Add one value at a time, twice over, so a value checked against
	// another (largest upload against largest file) gets a second try.
	s := DefaultSettings()
	pending := []string{}
	for _, k := range settingKeys {
		if _, ok := values[k]; ok {
			pending = append(pending, k)
		}
	}
	for range 2 {
		var rejected []string
		for _, k := range pending {
			next := s
			if !next.set(k, values[k]) {
				rejected = append(rejected, k)
				continue
			}
			clean, err := next.Validate()
			if err != nil {
				rejected = append(rejected, k)
				continue
			}
			s = clean
		}
		pending = rejected
	}
	return s, pending
}

// SettingsStamp says when the settings last changed and who changed them.
type SettingsStamp struct {
	UpdatedAt     *time.Time `json:"updatedAt"`
	UpdatedBy     string     `json:"updatedBy"`
	UpdatedByName string     `json:"updatedByName"`
}
