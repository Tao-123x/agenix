# V0 Release Checklist

## 范围

这个 checklist 适用于 Agenix V0 reference runtime。它是仓库实现的本地 acceptance
gate，不声称覆盖强 sandbox、远程执行器、托管 registry 或 provider-backed 远程 adapter。

## 必需 Gate

在仓库根目录运行：

```bash
go run ./cmd/agenix acceptance
```

该命令预期会在三个 canonical skill 上运行 canonical V0 acceptance sweep，并且通过。

## 本地完整验证

在 cut 或 review V0 release 前运行：

```bash
go run ./cmd/agenix acceptance
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

GitHub Actions 会通过 `.github/workflows/v0-release-gate.yml` 运行同一组 gate。

## Acceptance 覆盖范围

V0 acceptance 命令覆盖：

- manifest 校验
- 可移植 capsule 的 build 与 inspect
- artifact 执行
- trace 校验
- verifier 重新运行
- trace replay
- 本地 registry publish 与 pull
- 直接使用 registry reference 执行
- analysis skill 的 builtin 只读 `heuristic-analyze` adapter 路径

## 有意排除

V0 有意排除：

- 强 sandbox 保证
- 远程执行器语义
- 默认 acceptance sweep 中的 provider-backed 远程 adapter 覆盖
- registry trust policy
- artifact 签名
- OCI 分发语义
- 托管或共享 registry 行为
- 超出 canonical 本地 sweep 的 provider/runtime 兼容性矩阵

可选的 `openai-analyze` smoke 路径仍然不属于默认 V0 release gate。
