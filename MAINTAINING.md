# 维护说明（Hei-XiaoHu fork）

五个仓库都只保留 `main` 分支，上游更新一律用 **merge** 合入 `main`（不 rebase、不强推），
这样 `go.mod` 里引用过的旧提交永远可下载。

| 仓库 | 上游 (`upstream`) | 我们在上游之上的补丁 |
|---|---|---|
| [Hei-XiaoHu/Xboard-Node](https://github.com/Hei-XiaoHu/Xboard-Node) | cedar2025/Xboard-Node `dev` | 内核版本升级与适配 |
| [Hei-XiaoHu/Xray-core](https://github.com/Hei-XiaoHu/Xray-core) | XTLS/Xray-core 发布 tag | 按用户限速 (`features/bandwidth`, `app/dispatcher/ratelimit.go`) |
| [Hei-XiaoHu/sing-box](https://github.com/Hei-XiaoHu/sing-box) | SagerNet/sing-box 发布 tag | UpdateUsers/UpdateRules、mieru、DynamicInboundManager、并发安全的换用户、`block_destinations`（防 SSRF） |
| [Hei-XiaoHu/sing-quic](https://github.com/Hei-XiaoHu/sing-quic) | SagerNet/sing-quic `main` | hysteria/hysteria2/tuic 用户表原子替换 |
| [Hei-XiaoHu/REALITY](https://github.com/Hei-XiaoHu/REALITY) | XTLS/REALITY `main` | 去掉“必须带 X25519MLKEM768”的强制检查（见下文） |

Xray-core / sing-box 还有一个只读远程 `cedar`，用来查看原作者是否更新了补丁。
**活跃的主上游是 XTLS / SagerNet**（以及节点本体的原作者 cedar2025/Xboard-Node 和面板 cedar2025/Xboard），
cedar2025 的内核 fork 很少更新，只作为 Xboard 专用补丁的参考。

GitHub 上 Xboard-Node / Xray-core / sing-box 的 fork 父仓库是 cedar2025 的仓库，页面上的
“N commits behind” 是和 cedar 比的。cedar 的提交审阅/移植完后，用
`git merge -s ours <cedar 分支>` 记录为已合并（不改任何文件），提示就会消失。

检查所有上游的更新：`.pi/skills/upstream-sync/scripts/status.sh`（或在 pi 里 `/skill:upstream-sync`）。

**tag / release 规则：** fork 里不保留任何上游的 tag 和 release。Xray-core / sing-box / sing-quic / REALITY
一个 tag 都没有，Xboard-Node 只有自己发的 `v*-hx.N` 和滚动的 `dev`。
因此 go.mod 里对这几个 fork 的伪版本号一律是 `v0.0.0-<时间>-<hash>`；
**不要** `git push --tags`、也不要把上游 tag 推到 origin（本地 `git fetch upstream --tags` 没问题）。
若伪版本号写成 `vX.Y.(Z+1)-0.…` 形式，Go 会要求 fork 里存在 `vX.Y.Z` tag，`GOPROXY=direct` 时直接报
`invalid pseudo-version: preceding tag not found`。

本地路径：`~/桌面/proxy/{Xboard-Node,Xray-core,sing-box,sing-quic,REALITY}`。

### 工作约定（人和 AI 都遵守）

- 只在 `main` 上工作；不 rebase、不 `push -f`、不 `git push --tags`。
- `git push` 走 SSH（`git@github-heixiaohu:`），无需额外配置。
- 本机 `gh` 的活跃账号不是 Hei-XiaoHu，调 `gh` 一律加前缀：
  `GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh ...`
- 本机 Go 比 go.mod 要求的旧，Go / make 命令前加 `GOTOOLCHAIN=auto`。
- 推送后看 CI：`GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh run watch -R Hei-XiaoHu/Xboard-Node`。

### REALITY 补丁

上游 REALITY `8cdf7bf`（Xray ≥ v26.9.8）会静默拒绝不带 X25519MLKEM768 的 Client Hello：
服务端把连接当普通访客转给 target 站点，客户端表现为超时（mihomo 日志 `REALITY Authentication: false`）。
默认配置的 mihomo / sing-box 都不发 X25519MLKEM768，所以全部连不上。
我们的补丁恢复旧行为（优先普通 X25519，否则用 X25519MLKEM768 里的 X25519），新旧客户端都能连。

升级 Xray 时，看 Xray 的 go.mod 用哪个 REALITY 版本，把它 merge 进 fork 的 `main`，
再更新 Xboard-Node 的 replace。等 mihomo/sing-box 默认发送 X25519MLKEM768 后，
删掉 Xboard-Node go.mod 里的 REALITY replace 即可回到上游。

验证方法：本机起一个 Xray REALITY 服务端，分别用 mihomo 默认配置、mihomo 加
`reality-opts.support-x25519mlkem768: true`、Xray 客户端去连，三者都必须能通。

注意：`replace` 只在主模块生效。sing-box 的 go.mod 里替换了 sing-quic，
但 Xboard-Node 的 go.mod 里**也必须**有同样的 sing-quic replace，否则会悄悄用回上游版本。
Go 版本跟随 Xray（目前要求 1.27）；本机 Go 较旧时命令前加 `GOTOOLCHAIN=auto`。

## 升级 Xray-core

```bash
cd ~/桌面/proxy/Xray-core
git fetch upstream --tags
git merge vX.Y.Z            # 上游新版本 tag，解决冲突
go build ./... && go test ./app/dispatcher/ ./common/buf/ ./common/mux/ ./features/bandwidth/ ./core/
git push origin main
```

## 升级 sing-box

```bash
cd ~/桌面/proxy/sing-box
git fetch upstream --tags
git merge vX.Y.Z            # 只合并，不要把上游 tag 推到 origin
TAGS="with_quic with_utls with_wireguard with_acme with_clash_api"
go build -tags "$TAGS" ./cmd/sing-box ./protocol/... ./route/... ./adapter/...
go test -tags "$TAGS" ./adapter/... ./route/... ./protocol/...
git push origin main
```

升级 sing-box 后，看它的 go.mod 用的是哪个 sing-quic 版本，把 sing-quic 合并到那个版本：

```bash
cd ~/桌面/proxy/sing-quic
git fetch upstream
git merge <sing-box go.mod 里 sing-quic 的提交 hash>
go build ./... && git push origin main
# 然后 sing-box 和 Xboard-Node 都更新 sing-quic 的 replace
```

升级 sing-box 时重点关注这些 xboard 改动是否还能编译/语义正确：
`adapter/inbound.go`、`adapter/inbound/manager.go`、`adapter/lifecycle.go`(Scope.Remove)、
`route/router.go`(UpdateRules)、`protocol/*/user.go`、`protocol/mieru/`、
`common/dialer/destination_guard.go` 与 `common/dialer/default.go` 里的 guard 调用。

## 更新 Xboard-Node

```bash
cd ~/桌面/proxy/Xboard-Node
git fetch upstream
git merge upstream/dev      # 合入原作者的更新；go.mod 的 replace 冲突保留我们的

# 指向新的内核提交（填 Xray-core / sing-box main 的提交 hash）
go mod edit -replace=github.com/xtls/xray-core=github.com/Hei-XiaoHu/Xray-core@<commit>
go mod edit -replace=github.com/sagernet/sing-box=github.com/Hei-XiaoHu/sing-box@<commit>
go mod edit -replace=github.com/sagernet/sing-quic=github.com/Hei-XiaoHu/sing-quic@<commit>
go mod edit -replace=github.com/xtls/reality=github.com/Hei-XiaoHu/REALITY@<commit>
go mod tidy                 # 结果应为 v0.0.0-<时间>-<hash>

make test
make test-runtime
make build-linux build-linux-arm64
git push origin main
```

`make test-runtime` 会为两个内核真正启动节点：检查用户无法通过节点访问本机
（IP、`localhost`、`0.0.0.0`、IPv4 映射地址、UDP 夹带都要被拦），并为每种协议真正启动 sing-box，走一遍 Reload / Stop，
并在不断有连接握手的同时反复 UpdateUsers（开 `-race`）；对 hysteria/hysteria2/tuic
还会用真实客户端在换用户期间持续连接，并验证删掉的用户连不上、新加的用户能连上。
内核升级后务必跑一次。

## 发布

服务器上的 `install.sh` 和 `xbctl upgrade` 都从本仓库 Release 里标为 **Latest** 的版本下载。

- 推送到 `main`：CI 测试、编译，并更新滚动预发布 `dev`（不会被安装/升级选中，可用 `--version dev` 指定）。
- 发正式版：打 `v*` tag，CI 发布并标为 Latest。版本号 = 上游 Xboard-Node 版本 + `-hx.N`，避免和原作者撞号：
  上游版本没变就在现有最大 N 上加 1，上游版本变了就从 `-hx.1` 开始
  （现有版本：`GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh release list -R Hei-XiaoHu/Xboard-Node`）。

```bash
git tag -a v1.13-hx.2 -m "说明" && git push origin v1.13-hx.2
```

发版后确认 CI 成功且该版本被标成 Latest。服务器升级：`xbctl upgrade`。CI 里的 Docker 任务只在 cedar2025 仓库运行，fork 里自动跳过。

## 注意事项

- **防 SSRF（防止用户借节点访问节点所在机器/内网）：** 拦截列表在
  `internal/kernel/ssrf.go`，两个内核共用。路由规则只能匹配 IP，真正兜底的是直连出站
  （sing-box 的 `block_destinations`、Xray freedom 的 `finalRules`），它们在 DNS 解析之后、
  并对每个 UDP 包检查。面板里的路由“放行”规则放行不了这些地址；确实需要访问内网时，
  在 `kernel.custom_outbound` 里自定义一个 tag 为 `direct` 的出站来覆盖默认直连（风险自负）。

