# 分叉与上游差异（divergence）

本仓库从分叉 `Eviden0/yaklang` 的 `fix/overlay-query-and-sarif` 分支抽取，该分支在 merge-base `238faf8`（2026-11-01）之上有 3 个本地提交：

| 提交 | 主题 | 主要范围 | 动机 | 上游状态 |
|------|------|----------|------|----------|
| `a73dee1` | overlay 查询修复 + SARIF 输出修复 | `common/yak/cmd/yakcmds/ssacli.go`、`common/yak/ssaapi/sf_native_call_include.go`、`common/yak/ssaapi/sfreport/output.go` | overlay（base + diff）程序重建时查询走 `ssaapi.NewProgramFromDB`，修复结果缺失；SARIF 输出字段规范化 | 未见上游合入 |
| `e9024db` | ssa-scan CLI + value-aware 风险管线 | 34 文件 / 约 +4k 行：`cmd/ssa-scan/main.go`、`common/yak/cmd/ssacmds/ssacli.go`、`programs_overlay_files.go`、`sfvm/values_merge_by_id.go`、`sfbuildin/embedded_library.go`、`onRiskValue` 贯通 | 提供独立扫描入口与基于 Value 的风险生成；覆盖同一次扫描后一个模式覆盖前一个模式的键 | 未见上游合入 |
| `b7206a7` | `ssa_only` 瘦构建 | 44 文件：`cmd/yak-ssa/main.go`、`runner_ssa_only.go`、`*_slim.go`/`*_grpc.go` 成对 shim | 只编 SSA 所需闭包；`ssa_only` 下 ypb 仅需消息类型，gRPC 服务文件被 `!ssa_only` 排除，无需重新生成 proto | 未见上游合入 |

## 保留的设计

- **模块路径**：保持 `github.com/yaklang/yaklang`，使上游 SSA 补丁的 import 与生成代码可原样落地；消费者通过 `replace` 指向本仓库。
- **overlay 修复**：瘦构建侧 `common/yak/cmd/ssacmds/ssacli.go` 已与 `yakcmds/ssacli.go` 一样使用 `ssaapi.NewProgramFromDB`，两端行为一致；需补瘦侧 overlay 查询回归测试。
- **cgo/PCRE2**：保留 `go-pcre2-lite`（自带 PCRE2 C 源码、静态编入）以保证 `lookaround`/`backref` 规则的扫描语义与性能；构建需要 MinGW-w64 gcc（见 `BUILD.md`）。

## 跟进上游策略

- 只对触及 SSA 闭包路径的提交做**路径过滤 cherry-pick**（见 `scripts/sync/`）。
- 冲突停在首个失败提交，记录到 `scripts/sync/conflicts/<sha>.md`，人工处置。
- 闭包外共享代码（ypb/consts/schema 等）若被上游 SSA 提交同时改动，单独评估，不自动扩大闭包。
