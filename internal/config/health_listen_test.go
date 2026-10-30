package config

import (
	"strings"
	"testing"
)

// ── health_listen ──

func TestHealthAddr(t *testing.T) {
	cases := []struct {
		name     string
		port     int
		listen   string
		wantAddr string
		wantPort int
	}{
		{"only port", 65530, "", ":65530", 65530},
		{"only listen", 0, "127.0.0.1:65531", "127.0.0.1:65531", 65531},
		{"both: listen wins", 65530, "127.0.0.1:65531", "127.0.0.1:65531", 65531},
		{"neither: disabled", 0, "", "", 0},
		{"ipv6 listen", 0, "[::1]:65530", "[::1]:65530", 65530},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{HealthPort: tc.port, HealthListen: tc.listen}
			if got := c.HealthAddr(); got != tc.wantAddr {
				t.Errorf("HealthAddr = %q, want %q", got, tc.wantAddr)
			}
			if got := c.HealthEffectivePort(); got != tc.wantPort {
				t.Errorf("HealthEffectivePort = %d, want %d", got, tc.wantPort)
			}
		})
	}
}

func TestHealthProbeURL(t *testing.T) {
	cases := map[string]string{
		":65530":          "http://127.0.0.1:65530/healthz",
		"0.0.0.0:65530":   "http://127.0.0.1:65530/healthz",
		"[::]:65530":      "http://127.0.0.1:65530/healthz",
		"127.0.0.1:65530": "http://127.0.0.1:65530/healthz",
		"[::1]:65530":     "http://[::1]:65530/healthz",
		"10.0.0.5:8080":   "http://10.0.0.5:8080/healthz",
		"":                "",
		"garbage":         "",
	}
	for addr, want := range cases {
		if got := HealthProbeURL(addr); got != want {
			t.Errorf("HealthProbeURL(%q) = %q, want %q", addr, got, want)
		}
	}
}

func TestLoadRoot_HealthListen(t *testing.T) {
	cases := []struct {
		name     string
		fields   string
		wantAddr string
	}{
		{"only port", "health_port: 65530", ":65530"},
		{"only listen", `health_listen: "127.0.0.1:65530"`, "127.0.0.1:65530"},
		{"both", "health_port: 65530\n    health_listen: \"[::1]:65531\"", "[::1]:65531"},
		{"neither", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, `
instances:
  - panel: { url: "https://panel.example.com", token: "a", node_id: 1 }
    `+tc.fields+"\n")
			root, err := LoadRoot(path)
			if err != nil {
				t.Fatalf("LoadRoot: %v", err)
			}
			insts, err := root.NormalizeInstances()
			if err != nil {
				t.Fatal(err)
			}
			if got := insts[0].HealthAddr(); got != tc.wantAddr {
				t.Errorf("HealthAddr = %q, want %q", got, tc.wantAddr)
			}
		})
	}
}

func TestLoad_HealthListenLegacyTopLevel(t *testing.T) {
	cfg, err := Load(writeTemp(t, `
panel: { url: "https://panel.example.com", token: "a", node_id: 1 }
health_port: 65530
health_listen: "127.0.0.1:65530"
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.HealthAddr(); got != "127.0.0.1:65530" {
		t.Errorf("HealthAddr = %q", got)
	}
}

func TestLoad_HealthListenValidation(t *testing.T) {
	for _, bad := range []string{"65530", "127.0.0.1", "127.0.0.1:0", "127.0.0.1:70000", "127.0.0.1:http"} {
		_, err := Load(writeTemp(t, "panel: { url: \"https://p.example.com\", token: \"a\", node_id: 1 }\nhealth_listen: \""+bad+"\"\n"))
		if err == nil || !strings.Contains(err.Error(), "health_listen") {
			t.Errorf("health_listen %q: want validation error, got %v", bad, err)
		}
	}
}

func TestValidateStartupLayout_HealthAddrConflicts(t *testing.T) {
	nodeID := 0
	mk := func(id string, port int, listen string) *Config {
		nodeID++
		return &Config{InstanceID: id, HealthPort: port, HealthListen: listen,
			Panel:  PanelConfig{URL: "https://p.example.com", NodeID: nodeID},
			Kernel: KernelConfig{ConfigDir: "/tmp/" + id}}
	}
	if err := ValidateStartupLayout([]*Config{mk("a", 65530, ""), mk("b", 65530, "")}); err == nil || !strings.Contains(err.Error(), "health_port 65530") {
		t.Errorf("duplicate health_port: got %v", err)
	}
	if err := ValidateStartupLayout([]*Config{mk("a", 0, "127.0.0.1:1"), mk("b", 0, "127.0.0.1:1")}); err == nil || !strings.Contains(err.Error(), "health_listen") {
		t.Errorf("duplicate health_listen: got %v", err)
	}
	if err := ValidateStartupLayout([]*Config{mk("a", 0, "127.0.0.1:1"), mk("b", 0, "127.0.0.1:2")}); err != nil {
		t.Errorf("distinct health_listen should pass: %v", err)
	}
}
