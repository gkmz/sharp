# Sharp 版本发布指南

[English](RELEASING.md)

当前已准备 v0.1.0 发布流程；配置和版本说明草稿不代表版本已经发布。正式发布由维护者操作。

## 版本规则

使用带 v 前缀的 SemVer 标签，从 v0.1.0 开始。1.0 前，补丁版本用于修复且保持现有使用方式兼容；次版本用于新能力或已说明的不兼容调整。进入 v1.0.0 后，公共 CLI/配置的不兼容修改需要升级主版本。预发布示例为 v0.2.0-rc.1。已发布的标签不可移动或复用，修复通过新版本交付。

## 本地验证

使用 go.mod 声明的 Go 版本或更新版本。CI 使用 go.mod，覆盖 Linux/macOS。从 GoReleaser 官方 Release 安装 v2.19.0，或通过 Homebrew 安装并确认版本；只有开发发布流程需要它，Sharp 用户无需安装。安装包检查另需 Python 3.9+。

```sh
make verify
go test -race ./...
goreleaser check
goreleaser release --snapshot --clean
python3 scripts/verify-release.py dist
```

snapshot 是仅供本地验证、不会上传的开发包；允许在未打标签或有改动的目录构建，不可当作正式版本分发。四个目标为 macOS/Linux × amd64/arm64；tar.gz 包含可执行文件、LICENSE 和中英文 README，并生成 checksums.txt。Python 脚本核对所有校验和和包内容，执行本机架构的 version 和 URL 命令。交叉编译成功不能代替目标平台运行验证；原生 CI 仅验证托管 runner 所在架构，其余架构在正式发布前应安排本机检查。

PR CI 同样检查 snapshot 安装包。标签发布流程会在两个系统重新执行 Go 检查和 race 测试，全部通过后才打包。

## 准备发布提交

1. 完成并提交待处理代码，通过 PR 和 CI 合入 main；打标签前工作区必须干净。
2. 审核 docs/releases/v0.1.0.md。后续版本新增 docs/releases/<标签>.md，包含中英文亮点、安装、限制和迁移说明。准备正式发布时移除“草稿/尚未发布”的状态句。
3. 将两个 CHANGELOG 中本次交付的 Unreleased 条目移到 `## [v0.1.0] - YYYY-MM-DD`，日期使用实际准备发布的日期；保留空 Unreleased 供后续开发使用。
4. 将发布准备修改通过 PR 合入 main，并等所有 CI（含安装包检查）通过。
5. 确认 GitHub Actions 已开启，仓库/组织策略允许打包任务使用 contents:write 创建 Release。只有打包任务申请写权限。两份 workflow 固定同一 GoReleaser 版本，升级时同步修改并本地验证。

## 打标签与生成草稿

下列命令会将标签推送到远端并触发工作流。仅在确定开始发布、审核 main 对应提交后执行：

```sh
git switch main
git pull --ff-only origin main
test -z "$(git status --porcelain)"
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

`Release draft` 检查标签指向的提交属于 origin/main、版本说明存在、两个 CHANGELOG 均含该标签的带日期标题；通过后注入版本号、提交号和 UTC 构建日期，生成 GitHub Release **草稿**。RC 标签自动标记为预发布；不会更新 Homebrew tap 等外部仓库。

标签推送后已公开，Go 模块工具也可能获取，即使 GitHub Release 仍是草稿。因此打标签本身就是发布步骤。工作流失败时检查原因，适用时重跑失败任务；不要强制移动已公开标签。需要修改代码时发布新版本。

## 审核并正式发布

1. 在 https://github.com/gkmz/sharp/releases 打开草稿，核对目标提交、版本和说明。
2. 将四个压缩包和 checksums.txt 下载到空目录，核对校验和和包内容：

```sh
# macOS
shasum -a 256 -c checksums.txt
# Linux
sha256sum -c checksums.txt
```

3. 解压本机安装包，运行 `./sharp version`、典型 CLI 命令和 TUI，验证 j/k、s 搜索、h/l、表单、最近记录、剪贴板。版本与提交号必须匹配标签，并补齐其余目标架构的本机验证。
4. 审核后点击 **Publish release**。用独立安装验证下载和固定版本 go install，保持发布说明与变更日志一致。

SHA256 用于检查文件是否损坏；校验和及二进制尚未签名，macOS 尚未做 Developer ID 签名和公证。后续可补充签名、来源证明和 Homebrew 分发。

## 代码之外的仓库设置

维护者应为 main 配置分支保护，要求 PR 及 `Verify (ubuntu-latest)`、`Verify (macos-latest)`、`Verify release packages` 检查通过；使用标签规则限制 v* 的创建/删除，仅允许维护者发版。可在 Settings → Security 启用私密漏洞报告，再将实际可用入口写入安全政策。这些属于 GitHub 仓库设置，代码中的 workflow 不会自动开启。
