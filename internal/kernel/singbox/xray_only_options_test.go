package singbox

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/kernel"
	"github.com/cedar2025/xboard-node/internal/panel"
)

// sing-box ignores the xray-only options: config is unchanged, only warnings.
func TestXrayOnlyOptionsIgnoredBySingbox(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol: "vless", ServerPort: 443, TLS: 2,
		TLSSettings: map[string]interface{}{"private_key": "k", "short_id": "ab", "server_name": "decoy.example.com"},
	}
	plain := config.KernelConfig{Type: "singbox", LogLevel: "warn"}
	extras := plain
	extras.XrayRealityDestOverride = "127.0.0.1:8443"
	extras.XrayRealityXver = 1
	extras.XrayAcceptProxyProtocol = true

	a, _ := json.Marshal(buildConfig(plain, testNodeSpec(nc), testUsers, kernel.TLSCert{}))
	b, _ := json.Marshal(buildConfig(extras, testNodeSpec(nc), testUsers, kernel.TLSCert{}))
	if string(a) != string(b) {
		t.Errorf("sing-box config changed by xray-only options:\n%s\n%s", a, b)
	}

	if w := unsupportedOptionWarnings(plain, testNodeSpec(nc)); len(w) != 0 {
		t.Errorf("no warnings expected without options, got %v", w)
	}
	w := unsupportedOptionWarnings(extras, testNodeSpec(nc))
	if len(w) != 2 || !strings.Contains(w[0], "xray_reality") || !strings.Contains(w[1], "proxy_protocol") {
		t.Errorf("unexpected warnings: %v", w)
	}

	nc.AcceptProxyProtocol = true // panel-side flag alone also warns
	if w := unsupportedOptionWarnings(plain, testNodeSpec(nc)); len(w) != 1 {
		t.Errorf("panel accept_proxy_protocol should warn once, got %v", w)
	}
}
