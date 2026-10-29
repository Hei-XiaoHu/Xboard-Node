# 执行合并、验证、发版、记录

每个仓库的具体 merge / build / test 命令见 `MAINTAINING.md`，这里只规定顺序和检查点。

## 1. 按依赖顺序合并内核 fork

REALITY → Xray-core；sing-quic → sing-box；最后 Xboard-Node。

- 冲突时理解两边意图后手动解决，保留我们的补丁语义；拿不准就停下问。
- 每个 fork 合并后按 `MAINTAINING.md` 对应的 build/test 验证，通过才 `git push origin main`。
- 移植 cedar 补丁的提交信息注明来源，例如 `port cedar2025/sing-box@<hash>: <标题>`；
  移植/审阅完用 `merge -s ours` 记录 cedar 分支（见 `assess.md`）。

## 2. 更新 Xboard-Node 并验证

```bash
go mod edit -replace=<模块>=github.com/Hei-XiaoHu/<fork>@<fork main 的提交 hash>   # 每个改动过的 fork
GOTOOLCHAIN=auto go mod tidy        # 结果必须是 v0.0.0-<时间>-<hash>
GOTOOLCHAIN=auto make test
GOTOOLCHAIN=auto make test-runtime  # 内核有变化时必须跑
GOTOOLCHAIN=auto make build-linux build-linux-arm64
.pi/skills/upstream-sync/scripts/status.sh --no-fetch   # 最后一节应全部 ✓
```

sing-quic 的 replace 在 sing-box 和 Xboard-Node 两边都要更新。测试失败就修，修不好就停下汇报，不推送。

## 3. 提交、推送、发版

- 提交信息写清合入的版本，例如 `deps: bump xray-core to v26.11.1, sing-box to v1.14.5`。
- `git push origin main` 后看 CI（命令见 `MAINTAINING.md`“工作约定”）。推 main 只更新滚动预发布 `dev`。
- 问用户是否发正式版；版本号规则和命令见 `MAINTAINING.md`“发布”。

## 4. 记录审阅位置

```bash
.pi/skills/upstream-sync/scripts/status.sh --no-fetch --mark
```

把当前各上游/cedar 分支的位置写进 `state.env`，下次只显示新增提交。被跳过的更新也算已审阅。
把 `state.env` 随 Xboard-Node 一起提交推送。
