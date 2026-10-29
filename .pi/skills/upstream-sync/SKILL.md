---
name: upstream-sync
description: 检查 Xboard-Node 及其内核 fork（Xray-core、sing-box、sing-quic、REALITY）的活跃上游（XTLS、SagerNet、cedar2025/Xboard-Node 与 Xboard 面板）以及 cedar2025 内核补丁有没有更新，评估每项更新该直接 merge、手动参考移植还是跳过，经用户确认后合并、更新 go.mod、测试、推送，按需发版。用于“看看上游有没有更新”“同步上游”“升级内核/Xray/sing-box”这类请求。
---

# 上游同步

先读仓库根目录的 `MAINTAINING.md`：补丁清单、各仓库升级命令、工作约定（`GOTOOLCHAIN=auto`、
`gh` 的 `GH_TOKEN` 前缀、只用 merge）、发版规则都**以它为准**；本 skill 只规定流程。
流程或补丁清单变了，同步更新 `MAINTAINING.md`。

| 步骤 | 看 / 跑 |
|---|---|
| 1. 收集现状 | `scripts/status.sh`（在 Xboard-Node 根目录运行，会 fetch 所有远程；`--no-fetch` 跳过） |
| 2. 逐项评估 | `references/assess.md` |
| 3. 报告并确认 | 本文件“规则” |
| 4–7. 合并、验证、推送、发版、记录 | `references/apply.md` |

`status.sh` 报告：Xboard-Node 未合入的 `upstream/dev` 提交；面板节点 API 相关新提交；
Xray/sing-box 已合入的 tag 与上游最新正式版/预发布、开发分支上未发布的提交；cedar 补丁分支的新提交；
GitHub 是否显示落后 fork 父仓库；sing-quic/REALITY 是否满足内核 go.mod；replace 是否都指向各 fork 的 `origin/main`；
各仓库是否有未提交/未推送的改动。

## 规则

- 任一仓库有未提交改动：先停下问用户。
- 第 3 步用简短表格列出：仓库 / 更新内容 / 建议（merge·cherry-pick·移植·跳过）/ 理由 / 风险，
  安全修复单独标出，然后用 ask_user_question 让用户勾选要做的项。
- **未经确认不 merge、不 push、不发版。** 什么都不用做时直接说明，再执行“记录审阅位置”。
- 只用 merge 合入 `main`；不 rebase、不 `push -f`、不 `git push --tags`、不把上游 tag 推到 origin；
  go.mod 里 fork 的伪版本必须是 `v0.0.0-<时间>-<hash>`。
- 测试失败修不好就停下汇报，不推送。
- 最后汇报：每个仓库合入/移植/跳过了什么、新的 go.mod replace、测试结果、推送的提交、
  是否发版及版本号、遗留问题（例如等上游修复后可删的补丁）。
