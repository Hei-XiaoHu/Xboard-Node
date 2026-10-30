package config

import (
	"strings"
	"testing"
)

// ── xray_reality_dest_override / xray_reality_xver / xray_accept_proxy_protocol ──

func TestLoadRoot_KernelExtrasInstanceOverridesGlobal(t *testing.T) {
	path := writeTemp(t, `
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8443"
  xray_reality_xver: 1
  xray_accept_proxy_protocol: true
instances:
  - panel: { url: "https://panel.example.com", token: "a", node_id: 1 }
  - panel: { url: "https://panel.example.com", token: "b", node_id: 2 }
    kernel:
      xray_reality_dest_override: "127.0.0.1:9443"
      xray_reality_xver: 2
`)
	root, err := LoadRoot(path)
	if err != nil {
		t.Fatalf("LoadRoot: %v", err)
	}
	inherit := root.Instances[0].Kernel
	if inherit.XrayRealityDestOverride != "127.0.0.1:8443" || inherit.XrayRealityXver != 1 || !inherit.XrayAcceptProxyProtocol {
		t.Errorf("instance without kernel extras should inherit global, got %+v", inherit)
	}
	override := root.Instances[1].Kernel
	if override.XrayRealityDestOverride != "127.0.0.1:9443" || override.XrayRealityXver != 2 {
		t.Errorf("instance values should override global, got dest=%q xver=%d",
			override.XrayRealityDestOverride, override.XrayRealityXver)
	}
	if !override.XrayAcceptProxyProtocol {
		t.Errorf("unset xray_accept_proxy_protocol should inherit global true")
	}
}

func TestLoadRoot_KernelExtrasUnsetStayZero(t *testing.T) {
	path := writeTemp(t, `
kernel: { type: xray }
instances:
  - panel: { url: "https://panel.example.com", token: "a", node_id: 1 }
`)
	root, err := LoadRoot(path)
	if err != nil {
		t.Fatalf("LoadRoot: %v", err)
	}
	k := root.Instances[0].Kernel
	if k.XrayRealityDestOverride != "" || k.XrayRealityXver != 0 || k.XrayAcceptProxyProtocol {
		t.Errorf("unset kernel extras must stay zero (hx.2 behaviour), got %+v", k)
	}
}

func TestExpandNodes_KernelExtrasPerNode(t *testing.T) {
	path := writeTemp(t, `
panel: { url: "https://panel.example.com", token: "a" }
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8443"
nodes:
  - node_id: 1
  - node_id: 2
    kernel:
      xray_reality_dest_override: "127.0.0.1:9443"
      xray_reality_xver: 1
      xray_accept_proxy_protocol: true
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	nodes := cfg.ExpandNodes()
	if got := nodes[0].Kernel; got.XrayRealityDestOverride != "127.0.0.1:8443" || got.XrayRealityXver != 0 || got.XrayAcceptProxyProtocol {
		t.Errorf("node 1 should keep global values, got %+v", got)
	}
	if got := nodes[1].Kernel; got.XrayRealityDestOverride != "127.0.0.1:9443" || got.XrayRealityXver != 1 || !got.XrayAcceptProxyProtocol {
		t.Errorf("node 2 should use its overrides, got %+v", got)
	}
}

func TestLoad_KernelExtrasValidation(t *testing.T) {
	cases := map[string]string{
		"xver out of range": "xray_reality_xver: 3",
		"dest without port": `xray_reality_dest_override: "127.0.0.1"`,
		"node xver bad":     "",
	}
	for name, kernelLine := range cases {
		t.Run(name, func(t *testing.T) {
			yaml := "panel: { url: \"https://p.example.com\", token: \"a\", node_id: 1 }\nkernel:\n  type: xray\n  " + kernelLine + "\n"
			if name == "node xver bad" {
				yaml = "panel: { url: \"https://p.example.com\", token: \"a\" }\nkernel: { type: xray }\nnodes:\n  - node_id: 1\n    kernel: { xray_reality_xver: 9 }\n"
			}
			if _, err := Load(writeTemp(t, yaml)); err == nil || !strings.Contains(err.Error(), "xray_reality") {
				t.Fatalf("want xray_reality validation error, got %v", err)
			}
		})
	}
}
