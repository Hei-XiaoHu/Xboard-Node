---
name: upstream-sync
description: 检查 Xboard-Node 及其内核 fork（Xray-core、sing-box、sing-quic、REALITY）的活跃上游（XTLS、SagerNet、cedar2025/Xboard-Node 与 Xboard 面板）以及 cedar2025 内核补丁有没有更新，评估每项更新该直接 merge、手动参考移植还是跳过，经用户确认后合并、更新 go.mod、测试、推送，按需发版。用于“看看上游有没有更新”“同步上游”“升级内核/Xray/sing-box”这类请求。
---

# 上游同步（Hei-XiaoHu fork 全家桶）

## 背景（先读）

- 先读本仓库的 `MAINTAINING.md`，里面有各仓库的补丁清单、升级命令和注意事项，**以它为准**；
  本 skill 只规定流程。流程或补丁清单有变化时，同步更新 `MAINTAINING.md`。
- 五个仓库都在 Xboard-Node 的上一级目录（`~/桌面/proxy/`）。
- **以活跃的原始上游为主**，cedar2025 的内核 fork 很少更新，只作为“Xboard 专用补丁”的参考：

| 仓库 | 主上游（活跃，`upstream` 远程） | 跟随什么 | 参考（`cedar` 远程） |
|---|---|---|---|
| Xboard-Node | cedar2025/Xboard-Node（节点本体原作者） | `upstream/dev` | — |
| Xray-core | XTLS/Xray-core | 正式版 tag；`upstream/main` 上未发布的安全修复可单独 cherry-pick | cedar2025/Xray-core `main` |
| sing-box | SagerNet/sing-box | 正式版 tag；`upstream/stable` 上未发布的修复同上；`upstream/testing` 是下个大版本，只看不合 | cedar2025/sing-box `testing` |
| sing-quic | SagerNet/sing-quic | sing-box 的 go.mod 里 require 的那个提交 | — |
| REALITY | XTLS/REALITY | Xray-core 的 go.mod 里用的那个提交 | — |
| （面板） | cedar2025/Xboard `master` | 不在本地；只看节点 API 相关改动，节点要不要跟着改 | — |

- GitHub 上 Xboard-Node / Xray-core / sing-box 的 fork 父仓库是 cedar2025 的仓库，
  页面上的 “N commits behind” 是和 cedar 比的，不代表落后 XTLS/SagerNet；脚本会单独算这两种差距。
  cedar 的提交审阅完、内容已吸收（或决定不要）后，用
  `git merge -s ours <cedar 分支> -m "Record cedar2025/... as merged (no content change)"`
  记录为已合并（不改任何文件，先确认 `HEAD^{tree}` 前后不变），GitHub 就不再显示落后。
- 只用 **merge** 合入 `main`，禁止 rebase、`push -f`、`git push --tags`、把上游 tag 推到 origin。
  fork 里不保留上游 tag/release；go.mod 里 fork 的伪版本必须是 `v0.0.0-<时间>-<hash>`。
- push 走 SSH（`git@github-heixiaohu:`），无需额外配置。`gh` 当前活跃账号不是 Hei-XiaoHu，
  调 `gh` 时一律加前缀：`GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh ...`。
- 本机 Go 比 go.mod 要求的旧，Go 命令前加 `GOTOOLCHAIN=auto`。

## 流程

### 1. 收集现状

```bash
.pi/skills/upstream-sync/scripts/status.sh        # 在 Xboard-Node 根目录运行，会 fetch 所有远程
```

它会报告：Xboard-Node 未合入的 upstream/dev 提交；Xboard 面板里节点 API 相关的新提交；
Xray/sing-box 已合入的 tag 与上游最新正式版/预发布、上游开发分支上未发布的提交；
cedar 补丁分支自上次审阅以来的新提交；GitHub 相对 fork 父仓库是否显示落后；sing-quic/REALITY 是否满足内核 go.mod 的要求；
Xboard-Node 的 replace 是否都指向各 fork 的 `origin/main`；各仓库是否有未提交/未推送的改动。
有未提交改动时先停下问用户。

### 2. 逐项评估

对每一项新东西读实际改动再下结论，不要只看标题：

```bash
git -C <repo> log --no-merges --stat <旧>..<新>
git -C <repo> diff <旧>..<新> -- <关心的路径>
git -C <repo> merge-tree --write-tree --name-only main <要合入的 ref>   # 预演冲突，不动工作区
```

分三类：

- **直接 merge**：Xray/sing-box 新的正式版 tag；sing-quic / REALITY 跟到内核需要的提交；
  Xboard-Node `upstream/dev` 的普通提交。
- **cherry-pick**：XTLS/SagerNet 已合进开发分支但还没发版的**安全修复或影响节点的严重 bug 修复**
  （普通改进等正式版）。提交信息注明 `cherry-pick upstream <hash>`，以后合正式版时 git 会自动处理。
- **节点跟进**：Xboard 面板节点 API（下发配置字段、用户列表、上报格式、WebSocket 事件）有变化时，
  评估 Xboard-Node 是否需要适配；需要的话按 node-dev skill 的流程实现。
- **手动参考移植**：cedar2025 补丁分支（Xray `cedar/main`、sing-box `cedar/testing`）的更新——
  我们的补丁是在新版本上重新实现的，不能直接 merge cedar 分支，只能看 diff 把有价值的修复移植过来；
  Xboard-Node 上游改动了我们也改过的地方时（go.mod 的 replace、`install.sh`/xbctl 的下载地址、
  `.github/workflows/ci.yml` 的发布流程、`internal/kernel/ssrf.go` 及用户热更新相关代码），
  merge 后必须保留我们的行为，或者干脆手动挑着移植。
- **跳过**：预发布（alpha/beta/rc，除非用户点名要）；只和 Docker/ghcr、renovate、其他平台
  （Android/macOS/Windows 客户端）相关的改动；上游已撤回的提交。

特别检查：
- 上游的改动会不会与 `MAINTAINING.md` 里列出的我们的补丁冲突或使其失效
  （例如 REALITY 又加回 X25519MLKEM768 的强制检查、sing-box 改了 `adapter/inbound` 接口、
  Xray 改了 dispatcher）。
- 上游有没有修掉我们自己也补过的问题——有的话可以删掉我们的补丁，回到上游实现。
- 安全修复优先级最高，要在报告里单独标出。

### 3. 给用户报告并确认

用一个简短表格列出：仓库 / 更新内容 / 建议（merge·移植·跳过）/ 理由 / 风险。
然后用 ask_user_question 让用户确认要做哪些。**未经确认不 merge、不 push、不发版。**
如果什么都不需要做，直接说明，然后执行第 7 步记录审阅位置。

### 4. 按依赖顺序执行

顺序：REALITY → Xray-core；sing-quic → sing-box；最后 Xboard-Node。具体命令见 `MAINTAINING.md`。
- 合并冲突时，理解两边意图后手动解决，保留我们的补丁语义；解决不了或拿不准就停下问。
- 每个内核 fork 合并后，按 `MAINTAINING.md` 里对应的 build/test 命令验证，通过才 `git push origin main`。
- 移植 cedar 补丁时，提交信息注明来源，例如 `port cedar2025/sing-box@<hash>: <标题>`；
  移植/审阅完后按上文用 `merge -s ours` 记录 cedar 分支，消除 GitHub 的落后提示。

### 5. 更新 Xboard-Node 并验证

```bash
go mod edit -replace=<模块>=github.com/Hei-XiaoHu/<fork>@<fork main 的提交 hash>   # 每个改动过的 fork
GOTOOLCHAIN=auto go mod tidy        # 确认结果是 v0.0.0-<时间>-<hash>
GOTOOLCHAIN=auto make test
GOTOOLCHAIN=auto make test-runtime  # 内核有变化时必须跑
GOTOOLCHAIN=auto make build-linux build-linux-arm64
.pi/skills/upstream-sync/scripts/status.sh --no-fetch   # 最后一节应全部 ✓
```

记得 sing-quic 的 replace 在 sing-box 和 Xboard-Node 两边都要更新。
测试失败就修，修不好就停下汇报，不要推送。

### 6. 提交、推送、发版

- 提交信息写清楚合入了什么版本，例如 `deps: bump xray-core to v26.11.1, sing-box to v1.14.5`。
- `git push origin main` 后看 CI：`GH_TOKEN=$(gh auth token --user Hei-XiaoHu) gh run watch -R Hei-XiaoHu/Xboard-Node`。
  推 main 只会更新滚动预发布 `dev`。
- 问用户是否发正式版。版本号 = 上游 Xboard-Node 版本 + `-hx.N`：上游版本没变就 N+1，
  上游版本变了就从 `-hx.1` 开始（现有版本看 `gh release list -R Hei-XiaoHu/Xboard-Node`）。
  `git tag -a vX.Y-hx.N -m "<更新摘要>" && git push origin vX.Y-hx.N`，再确认 CI 把它标成 Latest。

### 7. 记录审阅位置

```bash
.pi/skills/upstream-sync/scripts/status.sh --no-fetch --mark
```

会把当前各上游/cedar 分支的位置写入 `.pi/skills/upstream-sync/state.env`，下次只显示新增的提交。
被“跳过”的更新也算已审阅。把 `state.env` 随 Xboard-Node 一起提交推送。

## 最后的汇报

列出：每个仓库合入/移植/跳过了什么、新的 go.mod replace、测试结果、推送的提交、是否发版及版本号、
遗留问题（例如等上游修复后可以删掉的补丁）。
