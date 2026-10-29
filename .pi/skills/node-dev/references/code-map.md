# Xboard-Node 代码地图

| 路径 | 内容 |
|---|---|
| `cmd/xboard-node` / `cmd/xbctl` | 节点主程序 / 管理命令（安装、升级、状态、绑定） |
| `install.sh` | 服务器一键安装脚本（从本 fork 的 Release 下载） |
| `internal/service` | 主循环：拉面板配置/用户、调内核、上报 |
| `internal/panel`, `internal/controlplane` | 与 Xboard 面板的 HTTP/WS 通信、本地/面板控制面 |
| `internal/config`, `internal/model` | 本地配置、面板数据模型与校验、自定义出站/路由 |
| `internal/kernel` | 内核抽象；`singbox/`、`xray/` 两个实现；`ssrf.go` 防访问本机/内网；`kerneltest/` 运行时测试 |
| `internal/limiter`, `internal/tracker` | 限速/设备数限制、流量与在线 IP 统计 |
| `internal/cert` | 证书（ACME、DNS provider） |
| `internal/monitor`, `internal/machine`, `internal/nlog` | 系统监控、机器信息、日志 |
| `config.yml.example`, `docs-*.md`, `README.md` | 配置示例与用户文档 |

内核 fork（Xray-core、sing-box、sing-quic、REALITY）在 Xboard-Node 的上一级目录，
Xboard-Node 通过 go.mod 的 `replace` 引用各 fork 的 `main`。各 fork 上我们的补丁见 `MAINTAINING.md`。
