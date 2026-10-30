package xray

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/kernel"
	"github.com/cedar2025/xboard-node/internal/panel"
	"github.com/xtls/xray-core/infra/conf/serial"
)

func firstInbound(t *testing.T, kcfg config.KernelConfig, nc *panel.NodeConfig) M {
	t.Helper()
	cfg := buildConfig(kcfg, testNodeSpec(nc), testUsers, kernel.TLSCert{})
	inbounds, ok := cfg["inbounds"].([]M)
	if !ok || len(inbounds) == 0 {
		t.Fatalf("no inbound generated")
	}
	return inbounds[0]
}

func realityNode() *panel.NodeConfig {
	return &panel.NodeConfig{
		Protocol: "vless", ServerPort: 443, TLS: 2, Flow: "xtls-rprx-vision",
		TLSSettings: map[string]interface{}{"private_key": "k", "short_id": "ab", "server_name": "decoy.example.com"},
	}
}

// Instance-level kernel values must win over global ones, and an instance
// without its own values must inherit the global ones; the resulting kernel
// config must reach Reality's runtime settings.
func TestBuildConfig_RealityOverrideInstanceVsGlobal(t *testing.T) {
	cases := []struct {
		name     string
		instance string // instance-level kernel YAML
		wantDest string
		wantXver int
	}{
		{"instance unset inherits global", "{}", "127.0.0.1:8443", 1},
		{"instance overrides global", `{xray_reality_dest_override: "127.0.0.1:9443", xray_reality_xver: 2}`, "127.0.0.1:9443", 2},
		{"instance overrides dest only", `{xray_reality_dest_override: "127.0.0.1:9443"}`, "127.0.0.1:9443", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			yml := `
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8443"
  xray_reality_xver: 1
instances:
  - panel: { url: "https://panel.example.com", token: "a", node_id: 1 }
    kernel: ` + tc.instance + "\n"
			if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
				t.Fatal(err)
			}
			root, err := config.LoadRoot(path)
			if err != nil {
				t.Fatalf("LoadRoot: %v", err)
			}
			insts, err := root.NormalizeInstances()
			if err != nil {
				t.Fatal(err)
			}
			reality := firstInbound(t, insts[0].ExpandNodes()[0].Kernel, realityNode())["streamSettings"].(M)["realitySettings"].(M)
			if reality["dest"] != tc.wantDest {
				t.Errorf("dest = %v, want %v", reality["dest"], tc.wantDest)
			}
			if reality["xver"] != tc.wantXver {
				t.Errorf("xver = %v, want %v", reality["xver"], tc.wantXver)
			}
			if sn := reality["serverNames"].([]string); len(sn) != 1 || sn[0] != "decoy.example.com" {
				t.Errorf("serverNames = %v, want panel server_name", sn)
			}
		})
	}
}

func TestBuildConfig_RealityXverOnlyKeepsPanelDest(t *testing.T) {
	reality := firstInbound(t, config.KernelConfig{Type: "xray", XrayRealityXver: 2}, realityNode())["streamSettings"].(M)["realitySettings"].(M)
	if reality["dest"] != "decoy.example.com:443" {
		t.Errorf("dest = %v, want panel default", reality["dest"])
	}
	if reality["xver"] != 2 {
		t.Errorf("xver = %v, want 2", reality["xver"])
	}
}

func TestBuildConfig_RealityOverrideIgnoredForNonReality(t *testing.T) {
	kcfg := config.KernelConfig{Type: "xray", XrayRealityDestOverride: "127.0.0.1:8443", XrayRealityXver: 1}
	nc := &panel.NodeConfig{Protocol: "vless", ServerPort: 443, Network: "tcp"}
	with, _ := json.Marshal(firstInbound(t, kcfg, nc))
	without, _ := json.Marshal(firstInbound(t, config.KernelConfig{Type: "xray"}, nc))
	if string(with) != string(without) {
		t.Errorf("non-reality inbound changed by reality override:\n%s\n%s", with, without)
	}
}

// ── xray_accept_proxy_protocol ──

func sockopt(t *testing.T, inbound M) M {
	t.Helper()
	ss, _ := inbound["streamSettings"].(M)
	so, _ := ss["sockopt"].(M)
	return so
}

func TestBuildConfig_AcceptProxyProtocolOffUnchanged(t *testing.T) {
	for _, proto := range []string{"vless", "trojan", "shadowsocks", "vmess", "socks"} {
		nc := &panel.NodeConfig{Protocol: proto, ServerPort: 443, Network: "tcp", Cipher: "aes-128-gcm"}
		data, _ := json.Marshal(firstInbound(t, config.KernelConfig{Type: "xray"}, nc))
		if strings.Contains(string(data), "acceptProxyProtocol") {
			t.Errorf("%s: acceptProxyProtocol present while disabled: %s", proto, data)
		}
	}
}

func TestBuildConfig_AcceptProxyProtocolOn(t *testing.T) {
	kcfg := config.KernelConfig{Type: "xray", XrayAcceptProxyProtocol: true}
	for _, proto := range []string{"vless", "trojan", "shadowsocks", "vmess", "socks", "http"} {
		nc := &panel.NodeConfig{Protocol: proto, ServerPort: 443, Network: "tcp", Cipher: "aes-128-gcm"}
		in := firstInbound(t, kcfg, nc)
		if sockopt(t, in)["acceptProxyProtocol"] != true {
			data, _ := json.Marshal(in)
			t.Errorf("%s: acceptProxyProtocol not set: %s", proto, data)
		}
	}
	// Reality inbound gets it too, without disturbing realitySettings.
	in := firstInbound(t, kcfg, realityNode())
	if sockopt(t, in)["acceptProxyProtocol"] != true {
		t.Errorf("reality: acceptProxyProtocol not set")
	}
	if in["streamSettings"].(M)["security"] != "reality" {
		t.Errorf("reality security lost")
	}
}

func TestBuildConfig_AcceptProxyProtocolMergesSockopt(t *testing.T) {
	kcfg := config.KernelConfig{Type: "xray", XrayAcceptProxyProtocol: true}

	// socks keeps the base sockopt (reusePort) from buildInbound.
	so := sockopt(t, firstInbound(t, kcfg, &panel.NodeConfig{Protocol: "socks", ServerPort: 1080}))
	if so["reusePort"] != true || so["acceptProxyProtocol"] != true {
		t.Errorf("socks sockopt not merged: %v", so)
	}

	// Panel already enabled it (sockopt = reusePort + acceptProxyProtocol):
	// the local flag must keep both, i.e. same result as panel-only.
	nc := &panel.NodeConfig{Protocol: "vless", ServerPort: 443, Network: "tcp", AcceptProxyProtocol: true}
	panelOnly, _ := json.Marshal(firstInbound(t, config.KernelConfig{Type: "xray"}, nc))
	both, _ := json.Marshal(firstInbound(t, kcfg, nc))
	if string(panelOnly) != string(both) {
		t.Errorf("panel+local differs from panel-only:\n%s\n%s", panelOnly, both)
	}
	so = sockopt(t, firstInbound(t, kcfg, nc))
	if so["reusePort"] != true || so["acceptProxyProtocol"] != true {
		t.Errorf("vless sockopt not merged: %v", so)
	}

	// Other sockopt fields already present are preserved.
	in := M{"streamSettings": M{"network": "tcp", "sockopt": M{"tcpFastOpen": true, "mark": 255}}}
	applyAcceptProxyProtocol(in, testNodeSpec(&panel.NodeConfig{Protocol: "vless"}), kcfg)
	so = sockopt(t, in)
	if so["tcpFastOpen"] != true || so["mark"] != 255 || so["acceptProxyProtocol"] != true {
		t.Errorf("existing sockopt fields lost: %v", so)
	}

	// No streamSettings at all: created.
	in = M{}
	applyAcceptProxyProtocol(in, testNodeSpec(&panel.NodeConfig{Protocol: "vless"}), kcfg)
	if sockopt(t, in)["acceptProxyProtocol"] != true {
		t.Errorf("streamSettings not created: %v", in)
	}
}

func TestBuildConfig_AcceptProxyProtocolSkipsUDP(t *testing.T) {
	kcfg := config.KernelConfig{Type: "xray", XrayAcceptProxyProtocol: true}
	nc := &panel.NodeConfig{Protocol: "hysteria", ServerPort: 443, Version: 2}
	with, _ := json.Marshal(firstInbound(t, kcfg, nc))
	without, _ := json.Marshal(firstInbound(t, config.KernelConfig{Type: "xray"}, nc))
	if string(with) != string(without) {
		t.Errorf("hysteria inbound changed:\n%s\n%s", with, without)
	}

	in := M{"streamSettings": M{"network": "kcp"}}
	applyAcceptProxyProtocol(in, testNodeSpec(&panel.NodeConfig{Protocol: "vmess"}), kcfg)
	if _, has := in["streamSettings"].(M)["sockopt"]; has {
		t.Errorf("kcp inbound got sockopt: %v", in)
	}
}

// The generated JSON (local Reality dest + xver + acceptProxyProtocol) must be
// accepted by xray-core's own config loader, exactly as Start() does.
func TestBuildConfig_HX3OptionsLoadInXrayCore(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	nc := realityNode()
	nc.TLSSettings["private_key"] = base64.RawURLEncoding.EncodeToString(priv.Bytes())
	kcfg := config.KernelConfig{
		Type: "xray", LogLevel: "warn",
		XrayRealityDestOverride: "127.0.0.1:8443", XrayRealityXver: 1,
		XrayAcceptProxyProtocol: true,
	}
	data, err := marshalConfig(kcfg, testNodeSpec(nc), testUsers, kernel.TLSCert{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serial.LoadJSONConfig(bytes.NewReader(data)); err != nil {
		t.Fatalf("xray-core rejected generated config: %v\n%s", err, data)
	}
}
