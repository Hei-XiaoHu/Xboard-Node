package main

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
)

func writeCfg(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	runErr := fn()
	os.Stdout = old
	w.Close()
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatalf("command error: %v", runErr)
	}
	return strings.TrimSpace(string(out))
}

const instHead = "instances:\n  - panel: { url: \"https://p.example.com\", token_env: X, node_id: 1 }\n    kernel: { type: xray, config_dir: /tmp/x }\n"

func TestConfigHealthPortAndURL(t *testing.T) {
	cases := []struct {
		name     string
		fields   string
		wantPort string
		wantURL  string
	}{
		{"only port", "    health_port: 65530\n", "65530", "http://127.0.0.1:65530/healthz"},
		{"only listen", "    health_listen: \"127.0.0.1:65531\"\n", "65531", "http://127.0.0.1:65531/healthz"},
		{"both: listen wins", "    health_port: 65530\n    health_listen: \"[::1]:65532\"\n", "65532", "http://[::1]:65532/healthz"},
		{"neither", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeCfg(t, instHead+tc.fields)
			args := []string{"--config", path}
			if got := captureStdout(t, func() error { return runConfigHealthPort(args) }); got != tc.wantPort {
				t.Errorf("health-port = %q, want %q", got, tc.wantPort)
			}
			if got := captureStdout(t, func() error { return runConfigHealthURL(args) }); got != tc.wantURL {
				t.Errorf("health-url = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

func TestConfigHealthPortLegacyTopLevel(t *testing.T) {
	path := writeCfg(t, "panel: { url: \"https://p.example.com\", node_id: 1 }\nhealth_listen: \"127.0.0.1:65533\"\n")
	if got := captureStdout(t, func() error { return runConfigHealthURL([]string{"--config", path}) }); got != "http://127.0.0.1:65533/healthz" {
		t.Errorf("health-url = %q", got)
	}
}

func TestResolveHealthAddr(t *testing.T) {
	rootWith := func(port int, listen string) *config.RootConfig {
		return &config.RootConfig{Instances: []config.Config{{
			Panel: config.PanelConfig{URL: "https://p.example.com", NodeID: 1}, HealthPort: port, HealthListen: listen,
		}}}
	}
	metaPort := &installMeta{Instances: []instanceSummary{{HealthPort: 65530}}}
	cases := []struct {
		name string
		meta *installMeta
		root *config.RootConfig
		want string
	}{
		{"meta port only (hx.2 behaviour)", metaPort, nil, ":65530"},
		{"config port only", nil, rootWith(65530, ""), ":65530"},
		{"config listen beats stale meta port", metaPort, rootWith(65530, "[::1]:65530"), "[::1]:65530"},
		{"meta listen", &installMeta{Instances: []instanceSummary{{HealthPort: 65530, HealthListen: "127.0.0.1:65530"}}}, rootWith(0, ""), "127.0.0.1:65530"},
		{"nothing", nil, rootWith(0, ""), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveHealthAddr(tc.meta, tc.root); got != tc.want {
				t.Errorf("resolveHealthAddr = %q, want %q", got, tc.want)
			}
		})
	}
}

// `xbctl health` with health_listen: 127.0.0.1:<port> reaches a listener that
// is bound to loopback only and reports ok.
func TestHealthOKWithLoopbackListen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	root := &config.RootConfig{Instances: []config.Config{{
		Panel: config.PanelConfig{URL: "https://p.example.com", NodeID: 1}, HealthListen: ln.Addr().String(),
	}}}
	stale := &installMeta{Instances: []instanceSummary{{HealthPort: 1}}} // would be "down"
	url := config.HealthProbeURL(resolveHealthAddr(stale, root))
	if got := probeHealth(url); got != "ok" {
		t.Fatalf("probeHealth(%s) = %q, want ok", url, got)
	}
}

func TestWriteRootConfigKeepsHX3Fields(t *testing.T) {
	in := writeCfg(t, `
kernel:
  type: xray
  xray_reality_xver: 1
instances:
  - panel: { url: "https://p.example.com", token_env: X, node_id: 1 }
    kernel:
      type: xray
      config_dir: /tmp/x
      xray_reality_dest_override: "127.0.0.1:8443"
      xray_reality_xver: 2
      xray_accept_proxy_protocol: true
    health_port: 65530
    health_listen: "127.0.0.1:65530"
`)
	root, err := loadWritableRootConfig(in)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.yml")
	if err := writeRootConfig(out, root); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	s := string(data)
	for _, want := range []string{
		"xray_reality_dest_override: 127.0.0.1:8443",
		"xray_reality_xver: 2",
		"xray_reality_xver: 1",
		"xray_accept_proxy_protocol: true",
		"health_port: 65530",
		"health_listen: 127.0.0.1:65530",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("rewritten config lost %q:\n%s", want, s)
		}
	}

	// Unset fields must not appear (hx.2 output unchanged).
	plain := writeCfg(t, instHead+"    health_port: 65530\n")
	root, err = loadWritableRootConfig(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRootConfig(out, root); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(out)
	for _, bad := range []string{"xray_", "health_listen"} {
		if strings.Contains(string(data), bad) {
			t.Errorf("unexpected %q in rewritten config:\n%s", bad, data)
		}
	}
}
