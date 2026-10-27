---
name: node-dev
description: 按用户需求给 Xboard-Node（及必要时 Xray-core / sing-box / sing-quic / REALITY fork）新增功能或修复 bug：澄清需求、定位改动所在仓库、实现、补测试、跑全套验证、提交推送并按需发版。用于“加个功能”“修个 bug”“节点出现 xxx 问题”这类请求。
---

# 新增功能 / 修复问题

用户的需求跟在 `/skill:node-dev` 后面；没有给需求就先问。

## 背景（先读）

- 先读本仓库 `MAINTAINING.md`（fork 关系、我们的补丁清单、测试与发布流程、防 SSRF 设计）。
- 五个仓库在 `~/桌面/proxy/`：Xboard-Node（主程序）、Xray-core、sing-box、sing-quic、REALITY（内核 fork）。
  Xboard-Node 通过 go.mod 的 `replace` 引用各 fork 的 `main`，伪版本统一为 `v0.0.0-<时间>-<hash>`。
- 只在 `main` 上工作；不 rebase、不 `push -f`、不 `git push --tags`。
- `gh` 当前活跃账号不是 Hei-XiaoHu，调用时加 `GH_TOKEN=$(gh auth token --user Hei-XiaoHu)`；git push 走 SSH 即可。
- Go 命令前加 `GOTOOLCHAIN=auto`（本机 Go 较旧）。

Xboard-Node 代码地图：

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
| `docs-*.md`, `README.md` | 用户文档 |

## 流程

### 1. 弄清需求

- 复述你理解的需求、影响范围和验收标准；有歧义、有多种做法、或会改变现有行为（配置格式、
  面板协议、默认值、安全边界）时，用 ask_user_question 一次问清楚。需求明确就直接做。
- 修 bug：先复现或找到确凿的代码路径证据，定位根因，不要只压症状。用户给了日志/报错就从那里倒查。
  需要看线上节点时，可以用 termius MCP 登录服务器只读排查（看日志、`xbctl status`、配置），
  **改服务器上的任何东西前先问用户**。

### 2. 决定改哪个仓库

- 优先只改 Xboard-Node。改内核 fork 会增加以后合并上游的成本，只有在 Xboard-Node 层做不到时才改。
- 必须改内核时：改动尽量小、集中在新文件里，提交信息以 `xboard:` 开头；
  并把这个补丁加进 `MAINTAINING.md` 的补丁表和“升级时重点关注”列表。
- 涉及面板（Xboard PHP 端）字段或接口的，说明面板侧需要什么配合；不要臆造面板没有的字段。

### 3. 实现

- 遵循周围代码的风格和已有抽象；两个内核（sing-box、xray）都支持的功能要两边都实现，
  做不到的要显式报错/日志说明，不能静默忽略。
- 涉及用户能控制的目标地址/出站时，确认不会绕过 `internal/kernel/ssrf.go` 的拦截。
- 涉及用户热更新、并发的代码要考虑竞态（测试会开 `-race`）。
- 配置项新增/变更时同步更新 `config.yml.example`、README 或对应 `docs-*.md`。

### 4. 测试

- 为改动补单元测试（修 bug 时先写能复现问题的测试）。
- 运行：

```bash
GOTOOLCHAIN=auto go vet ./...
GOTOOLCHAIN=auto make test
GOTOOLCHAIN=auto make test-runtime     # 动到内核、协议、用户热更新、SSRF、出站/路由时必须跑
GOTOOLCHAIN=auto make build-linux build-linux-arm64
```

- 改了内核 fork：先在 fork 里按 `MAINTAINING.md` 的 build/test 命令验证并 `git push origin main`，
  再在 Xboard-Node 里 `go mod edit -replace=...@<新提交 hash>` + `go mod tidy`，然后跑上面全套。
- 全部通过才能进入下一步；失败就修，修不好就停下汇报。

### 5. 提交与推送

- 每个独立的改动一个提交，信息用 `feat:` / `fix:` / `security:` / `docs:` / `deps:` 前缀，正文写清楚为什么。
- 推送前给用户看一眼改动摘要（`git diff --stat` + 关键点）；用户此前已明确说“直接推”则不必再问。
- `git push origin main`，然后看 CI：
  `GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh run watch -R Hei-XiaoHu/Xboard-Node`。
  推 main 只会更新滚动预发布 `dev`，服务器可用 `xbctl upgrade --version dev`（或 install.sh `--version dev`）先试。

### 6. 发版（问过用户再做）

版本号 = 上游 Xboard-Node 版本 + `-hx.N`，N 在当前最大值上加 1
（`GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh release list -R Hei-XiaoHu/Xboard-Node`）：

```bash
git tag -a vX.Y-hx.N -m "<本次改动摘要>" && git push origin vX.Y-hx.N
```

确认 CI 成功且该版本被标成 Latest；服务器上 `xbctl upgrade` 即可升级。

## 最后的汇报

说明：做了什么、改了哪些仓库/文件、测试结果（含 test-runtime 是否跑过）、推送的提交、是否发版、
需要用户在服务器或面板上配合的事、已知限制。
