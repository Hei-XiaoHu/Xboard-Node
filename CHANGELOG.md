# Changelog (Hei-XiaoHu fork)

## v1.13-hx.4

Security: MPTCP is now off on all listeners (both kernels and the health endpoint).
Go ≥ 1.24 opens MPTCP listeners by default, which is the demonstrated remote-root path of
Linux kernel CVE-2026-98432. The binary is built with `godebug multipathtcp=0`; an inbound that
explicitly asks for MPTCP (Xray `sockopt.tcpMptcp`, sing-box `tcp_multi_path`) still gets it.
This only removes the MPTCP attack path — **upgrade the server kernel** as well.

- Xray-core v26.10.31 (includes "DefaultListener: Disable MPTCP by default", DNS outbound buffer
  bound fix, QUIC sniffer hardening).

## v1.13-hx.3

All new options are off by default; without them the node behaves exactly like v1.13-hx.2.

### New: Reality runtime dest override + xver (xray only)

Kernel options, top-level or per instance (instance wins, unset inherits top-level, same as
`custom_config`; also accepted in `nodes[].kernel`):

| Option | Default | Effect |
|---|---|---|
| `xray_reality_dest_override` | `""` (use panel dest) | Replaces Reality `dest` in the runtime Xray config only, e.g. `127.0.0.1:8443`. The panel's `server_name` is still written to `serverNames`, so subscriptions do not change. |
| `xray_reality_xver` | `0` (off) | `1` or `2`: Reality sends a PROXY protocol v1/v2 header when it falls back to `dest`, so the local web server sees the visitor's real IP. |

Only applied to xray inbounds with `security: reality`. sing-box ignores them and logs a warning.

### New: `health_listen`

`health_listen: "127.0.0.1:65530"` (host:port, `[::1]:65530` works) binds the health endpoint to a
specific address. It takes precedence over `health_port`; unset keeps the old `:<health_port>`
(all interfaces). Same scope as `health_port`. `xbctl health`, `xbctl config health-port` and
`install.sh` follow it; `xbctl config health-url` prints the local probe URL.

### New: `xray_accept_proxy_protocol` (xray only)

`kernel.xray_accept_proxy_protocol: true` (top-level / instance / `nodes[].kernel`) adds
`streamSettings.sockopt.acceptProxyProtocol: true` to the inbound (merged with existing sockopt),
so a node behind a relay (e.g. realm with `send_proxy`) sees users' real IPs. It is ORed with the
panel's `accept_proxy_protocol`. UDP inbounds (hysteria, mKCP) are skipped with a warning; sing-box
ignores it with a warning.

**Connections without a PROXY header fail once this is on** — only enable it on nodes that are
reached exclusively through the relay.

### Example

```yaml
instances:
  - panel: { url: https://panel.example.com, token: TOKEN, node_id: 32 }
    kernel:
      type: xray
      xray_reality_dest_override: "127.0.0.1:8443"
      xray_reality_xver: 1
      # xray_accept_proxy_protocol: true   # only for landing nodes behind a relay
    health_listen: "127.0.0.1:65530"
```
