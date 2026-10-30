# xboard-node

Node backend for [Xboard](https://github.com/cedar2025/Xboard). Supports `sing-box` / `xray-core` dual kernels.

> **Disclaimer**: This project is for educational and learning purposes only.

## Features

- Protocols: V2Ray family, Trojan, Shadowsocks, Hysteria2, TUIC, AnyTLS
- Sync: WebSocket push + REST polling dual channel
- User controls: speed limit, device limit, alive-IP tracking, hot update
- Deploy modes: node mode, machine mode, standalone mode
- Multi-instance: single process binding multiple panels / nodes

## Install

### Docker

```bash
docker run -d --restart=always --network=host \
  -e apiHost=https://panel.com -e apiKey=TOKEN -e nodeID=1 \
  ghcr.io/cedar2025/xboard-node:latest
```

### Docker Compose

```bash
git clone -b compose --depth 1 https://github.com/cedar2025/xboard-node.git
cd xboard-node
vim config/config.yml   # set panel.url / token / node_id
docker compose up -d
```

### Installer (Linux systemd)

```bash
# Node mode
curl -fsSL https://raw.githubusercontent.com/Hei-XiaoHu/Xboard-Node/main/install.sh | \
  sudo bash -s -- --mode node --panel https://panel.example.com --token TOKEN --node-id 1

# Machine mode
curl -fsSL https://raw.githubusercontent.com/Hei-XiaoHu/Xboard-Node/main/install.sh | \
  sudo bash -s -- --mode machine --panel https://panel.example.com --token TOKEN --machine-id 1

## xbctl

Run `xbctl` after installation for help. Common commands:

```bash
xbctl list                          # list all instances
xbctl status                        # running status
xbctl bind add-node --panel URL --token TOKEN --node-id 1
xbctl bind add-machine --panel URL --token TOKEN --machine-id 1
xbctl bind remove-node --panel URL --node-id 1
xbctl service restart
```

## Configuration

Legacy single-panel config is fully compatible. Appending bindings auto-migrates to `instances` format. See `config.yml.example`.

Fork-only options (all off by default, see [CHANGELOG.md](CHANGELOG.md)):

| Option | Where | Default | Effect |
|---|---|---|---|
| `kernel.xray_reality_dest_override` | top-level / instance / `nodes[].kernel` | `""` | Runtime-only Reality `dest` (e.g. `127.0.0.1:8443`); panel `server_name` stays in `serverNames`, subscriptions unchanged. xray only. |
| `kernel.xray_reality_xver` | same | `0` | `1`/`2`: PROXY protocol header on Reality fallback to `dest`. xray only. |
| `kernel.xray_accept_proxy_protocol` | same | `false` | Inbound requires a PROXY protocol header (real client IP behind a relay); ORed with the panel's `accept_proxy_protocol`. Direct connections fail when on. xray only. |
| `health_listen` | top-level / instance (like `health_port`) | `""` | Health endpoint address, e.g. `127.0.0.1:65530`; overrides `health_port`. |

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

## Extensions

- Custom routes: [docs-custom-routes.md](docs-custom-routes.md)
- Custom outbounds: [docs-custom-outbounds.md](docs-custom-outbounds.md)
- DNS providers (ACME DNS-01): [docs-dns-providers.md](docs-dns-providers.md)

## License

MPL-2.0.
