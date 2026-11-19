# ssa-engine 构建说明

这是从 `yaklang` 抽取的 **独立 SSA 静态分析引擎**，只保留 `cmd/yak-ssa`（`ssa_only` 瘦构建）的依赖闭包。
目录结构**逐路径镜像上游** `github.com/yaklang/yaklang`，模块路径保持不变，便于对上游 SSA 提交做路径过滤的 cherry-pick。

## 关键事实

- 入口：`./cmd/yak-ssa`，构建标签 `ssa_only`。
- 闭包：由 `go list -deps -tags ssa_only ./cmd/yak-ssa` 静态计算（当前 179 个内部包 / 约 1662 个非测试 `.go` 文件）。
- `go-pcre2-lite` 自带完整 PCRE2 C 源码并以 `-DPCRE2_STATIC` 静态编入，因此 **需要 C 编译器 + cgo**，但无需外部 libpcre2。

## 前置：C 工具链（cgo）

本机已安装 MinGW-w64（winlibs，x86_64 UCRT）到 `E:\env\mingw64`。
构建前把它的 `bin` 放进 `PATH` 并打开 cgo：

```bash
export PATH="/e/env/mingw64/bin:$PATH"   # Git Bash 视角
export CGO_ENABLED=1
export CC=gcc
```

验证：

```bash
gcc --version
go env CGO_ENABLED   # 应为 1
```

> 说明：`go-pcre2-lite` 的内部包在 `CGO_ENABLED=0` 时会因缺少 `internal/pcre2lite` 的 C 符号而编译失败（`undefined: lib.Regexp`），这是保留 cgo 后端的预期前提。

## 从上游/分叉重新抽取闭包

源码来自分叉仓库（示例：`E:/Code/golang/yaklang-fix-overlay-query-and-sarif`）：

```bash
# 1) 重新计算闭包清单
bash scripts/closure/gen-closure.sh "$YAK_SOURCE"

# 2) 把闭包包目录按原路径复制进本仓库
bash scripts/closure/extract.sh "$YAK_SOURCE" "$(pwd)"

# 3) 检测闭包漂移（应输出 closure OK）
bash scripts/closure/check-closure.sh "$YAK_SOURCE"
```

`gen-closure.sh` 会写入：

- `closure/closure.packages.txt`：内部 import path（仓库相对）。
- `closure/closure.dirs.txt`：内部包目录（仓库相对）。
- `closure/closure.build.txt`：每个包的 import path 与非测试 `.go` 文件数（含包数/文件数汇总）。

闭包发生变化（上游新增/删除内部依赖）必须单独提 PR，便于审查。

## 构建

```bash
export PATH="/e/env/mingw64/bin:$PATH"
export CGO_ENABLED=1 CC=gcc
go build -tags ssa_only -trimpath -ldflags="-s -w" -o yak-ssa ./cmd/yak-ssa
```

## 运行（隔离 & 确定性）

```bash
export YAKIT_HOME="$(mktemp -d)"
./yak-ssa code-scan -t <target-dir> --format sarif -o out.sarif
```

常用隔离/确定性环境变量：

- `YAKIT_HOME`：隔离数据目录。
- `YAK_DEFAULT_PROJECT_DATABASE_NAME` / `YAK_DEFAULT_PROFILE_DATABASE_NAME`：固定数据库名。
- `SSA_DATABASE_RAW` / `SSA_DB_SKIP_MIGRATE`：数据库行为。
- `YAK_SKIP_BUILTIN_SYNC=1`：跳过内置规则同步，保证规则集合固定。

## 上游同步

见 `scripts/sync/`（路径过滤 cherry-pick）与 `docs/fork/divergence.md`（本地改动动机）。
