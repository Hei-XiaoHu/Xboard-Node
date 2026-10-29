---
name: node-dev
description: 按用户需求给 Xboard-Node（及必要时 Xray-core / sing-box / sing-quic / REALITY fork）新增功能或修复 bug：澄清需求、定位改动所在仓库、实现、补测试、跑全套验证、提交推送并按需发版。用于“加个功能”“修个 bug”“节点出现 xxx 问题”这类请求。
---

# 新增功能 / 修复问题

需求来自用户消息；没说清楚就先问。先读仓库根目录的 `MAINTAINING.md`：fork 关系、补丁清单、
工作约定（`GOTOOLCHAIN=auto`、`gh` 的 `GH_TOKEN` 前缀、只用 merge）、测试与发布流程、防 SSRF 设计都以它为准。

| 需要 | 看 |
|---|---|
| 改动该落在哪个目录 | `references/code-map.md` |
| 内核 fork 的 build/test、发版命令 | `MAINTAINING.md` |

## 流程

1. **弄清需求**：复述理解的需求、影响范围和验收标准。有歧义、有多种做法，或会改变现有行为
   （配置格式、面板协议、默认值、安全边界）时，用 ask_user_question 一次问清。
   修 bug 先复现或找到确凿的代码路径，定位根因，不只压症状。
   可用 termius MCP 登录服务器**只读**排查（日志、`xbctl status`、配置）。
2. **决定改哪个仓库**：优先只改 Xboard-Node。只有它做不到时才改内核 fork：改动小、集中在新文件，
   提交信息以 `xboard:` 开头，并加进 `MAINTAINING.md` 的补丁表和“升级时重点关注”。
   涉及面板字段/接口时说明面板侧需要的配合，不臆造面板没有的字段。
3. **实现**：遵循周围代码的风格和抽象。sing-box、xray 两个内核都要实现，做不到就显式报错/打日志。
   用户能控制目标地址/出站时，确认没绕过 `internal/kernel/ssrf.go`。热更新、并发代码考虑竞态。
   配置项变更同步 `config.yml.example`、README 或 `docs-*.md`。
4. **测试**：补单元测试（修 bug 先写能复现的测试），然后：

   ```bash
   GOTOOLCHAIN=auto go vet ./...
   GOTOOLCHAIN=auto make test
   GOTOOLCHAIN=auto make test-runtime   # 动到内核、协议、用户热更新、SSRF、出站/路由时必须跑
   GOTOOLCHAIN=auto make build-linux build-linux-arm64
   ```

   改了内核 fork：先在 fork 里验证并 `git push origin main`，再在 Xboard-Node
   `go mod edit -replace=...@<新 hash>` + `go mod tidy`，然后跑上面全套。
5. **提交推送**：每个独立改动一个提交，前缀 `feat:` / `fix:` / `security:` / `docs:` / `deps:`，正文写为什么。
   推送前给用户看 `git diff --stat` 和要点（用户已说“直接推”除外）。`git push origin main` 后看 CI；
   推 main 只更新滚动预发布 `dev`，服务器可 `xbctl upgrade --version dev` 先试。
6. **发版**：问过用户再按 `MAINTAINING.md` 的“发布”一节打 `-hx.N` tag。

## 规则

- 改服务器上的任何东西前先问用户。
- 测试全部通过才提交；修不好就停下汇报，不推送。
- 最后汇报：做了什么、改了哪些仓库/文件、测试结果（test-runtime 是否跑过）、推送的提交、
  是否发版、需要用户在服务器或面板上配合的事、已知限制。
