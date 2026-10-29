# 逐项评估上游更新

## 跟随谁

以活跃的原始上游为主；cedar2025 的内核 fork 很少更新，只作为 Xboard 专用补丁的参考。

| 仓库 | 主上游（`upstream` 远程） | 跟随什么 | 参考（`cedar` 远程） |
|---|---|---|---|
| Xboard-Node | cedar2025/Xboard-Node | `upstream/dev` | — |
| Xray-core | XTLS/Xray-core | 正式版 tag；`upstream/main` 上未发布的安全修复可单独 cherry-pick | cedar2025/Xray-core `main` |
| sing-box | SagerNet/sing-box | 正式版 tag；`upstream/stable` 上未发布的修复同上；`upstream/testing` 是下个大版本，只看不合 | cedar2025/sing-box `testing` |
| sing-quic | SagerNet/sing-quic | sing-box 的 go.mod 里 require 的那个提交 | — |
| REALITY | XTLS/REALITY | Xray-core 的 go.mod 里用的那个提交 | — |
| （面板） | cedar2025/Xboard `master` | 不在本地；只看节点 API 相关改动 | — |

## 读实际改动，不只看标题

```bash
git -C <repo> log --no-merges --stat <旧>..<新>
git -C <repo> diff <旧>..<新> -- <关心的路径>
git -C <repo> merge-tree --write-tree --name-only main <要合入的 ref>   # 预演冲突，不动工作区
```

## 分类

| 类别 | 适用 |
|---|---|
| **直接 merge** | Xray/sing-box 新正式版 tag；sing-quic / REALITY 跟到内核需要的提交；Xboard-Node `upstream/dev` 的普通提交 |
| **cherry-pick** | 已进上游开发分支但未发版的**安全修复或影响节点的严重 bug 修复**（普通改进等正式版）。提交信息注明 `cherry-pick upstream <hash>` |
| **节点跟进** | 面板节点 API（下发配置字段、用户列表、上报格式、WebSocket 事件）有变化：评估 Xboard-Node 是否要适配，需要就按 node-dev skill 实现 |
| **手动参考移植** | cedar 补丁分支（Xray `cedar/main`、sing-box `cedar/testing`）的更新：我们的补丁是在新版本上重新实现的，不能直接 merge，只能看 diff 移植有价值的修复。Xboard-Node 上游改了我们也改过的地方（go.mod replace、`install.sh`/xbctl 下载地址、`.github/workflows/ci.yml` 发布流程、`internal/kernel/ssrf.go`、用户热更新代码）时，merge 后保留我们的行为，或手动挑着移植 |
| **跳过** | 预发布（alpha/beta/rc，除非用户点名）；只涉及 Docker/ghcr、renovate、其他平台客户端的改动；上游已撤回的提交 |

## 特别检查

- 上游改动是否与 `MAINTAINING.md` 列出的我们的补丁冲突或使其失效
  （例如 REALITY 又加回 X25519MLKEM768 强制检查、sing-box 改了 `adapter/inbound` 接口、Xray 改了 dispatcher）。
- 上游是否修掉了我们也补过的问题——是的话可以删补丁，回到上游实现。
- 安全修复优先级最高，在报告里单独标出。

## GitHub 上的 “commits behind”

Xboard-Node / Xray-core / sing-box 的 fork 父仓库是 cedar2025，页面上的落后数是和 cedar 比的，
不代表落后 XTLS/SagerNet（`status.sh` 会分开算）。cedar 的提交审阅完、内容已吸收或决定不要后：

```bash
git merge -s ours <cedar 分支> -m "Record cedar2025/<repo> as merged (no content change)"
```

合并前后先确认 `git rev-parse HEAD^{tree}` 不变。
