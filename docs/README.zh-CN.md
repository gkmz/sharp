# Sharp

```text
   _____ __  _____    ____  ____
  / ___// / / /   |  / __ \/ __ \
  \__ \/ /_/ / /| | / /_/ / /_/ /
 ___/ / __  / ___ |/ _, _/ ____/
/____/_/ /_/_/  |_/_/ |_/_/
        本地开发者工具箱
```

[![Go 版本](https://img.shields.io/badge/Go-1.25.1-00ADD8?style=for-the-badge&logo=go)](../go.mod)
[![许可证](https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge)](../LICENSE)
[![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea-FF75B7?style=for-the-badge)](https://github.com/charmbracelet/bubbletea)
[![CLI](https://img.shields.io/badge/CLI-Cobra-6F42C1?style=for-the-badge)](https://github.com/spf13/cobra)
[![状态](https://img.shields.io/badge/Status-Pre--1.0-orange?style=for-the-badge)](CHANGELOG.zh-CN.md)
[![测试](https://img.shields.io/badge/Tests-go%20test%20%2F%20go%20vet-success?style=for-the-badge)](#开发)

[英文文档](../README.md)

`sharp` 是一个本地优先、面向终端工作流的开发者工具箱。它同时提供受 lazygit 启发的键盘驱动 TUI，以及适合脚本调用的 CLI 命令，覆盖 JSON、编码、加密哈希、时间、文本、网络、格式转换、JWT 检查和随机生成等常见场景。

这个项目的目标是服务日常开发：粘贴数据、转换数据、查看输出、把输出继续应用到下一步，或者在 shell 脚本里调用同一套工具。

## 亮点

| 能力 | 说明 |
| --- | --- |
| ![TUI](https://img.shields.io/badge/TUI-lazygit--style-FF75B7) | 数字区域、边框面板、键盘优先导航、状态栏提示和可搜索帮助。 |
| ![CLI](https://img.shields.io/badge/CLI-scriptable-6F42C1) | 每个注册工具都可以通过 shell、文件或 stdin 调用。 |
| ![JSON](https://img.shields.io/badge/JSON-workbench-2EA44F) | 美化、压缩、校验、排序、转义、反转义、路径查询和输出链式应用集中在一个工作台。 |
| ![Tools](https://img.shields.io/badge/Tools-built--in-blue) | 覆盖 JSON、Encode、Crypto、Time、Text、Network、Convert、Inspector、Generator 工作流。 |
| ![Local](https://img.shields.io/badge/Local-first-lightgrey) | 核心转换都在本地运行，不依赖远程服务。 |

## 安装

从源码安装最新版本：

```bash
go install github.com/gkmz/sharp/cmd/sharp@latest
```

从本地仓库安装：

```bash
git clone https://github.com/gkmz/sharp.git
cd sharp
go install ./cmd/sharp
```

开发模式直接运行：

```bash
go run ./cmd/sharp
```

## TUI 使用

启动交互界面：

```bash
sharp
```

TUI 分为四个带数字快捷键的区域：

| 区域 | 用途 |
| --- | --- |
| `1 Search` | 搜索类目、子类目和具体工具。 |
| `2 Categories` | 选择类目或子类目。有子类目的父类只作为标题，不能选中。 |
| `3 Tools` | 选择当前子类目下的工具。JSON 类目使用单一 Workspace 页面。 |
| `4 Workspace` | 处理输入、参数/路径、动作和输出。 |

常用快捷键：

| 快捷键 | 动作 |
| --- | --- |
| `1`, `2`, `3`, `4` | 聚焦对应区域。 |
| `/` | 打开搜索。 |
| `j` / `k` | 在类目列表中上下移动。 |
| `h` / `l` | 在区域 3 中左右选择工具。 |
| `i` 或 `Enter` | 当前工具支持输入时，进入输入编辑。 |
| `o` | 编辑参数，或在 JSON 工作台中编辑 path。 |
| `r` | 运行当前工具或默认动作。 |
| `v` | 将剪贴板内容粘贴到输入。 |
| `x` / `X` | 清空输入 / 清空输出。 |
| `y` | 复制 trim 后的输出。 |
| `s` | 保存 trim 后的输出到 `sharp-output.txt`。 |
| `p` | 将输出回填到输入，便于链式处理。 |
| `ctrl+u` / `ctrl+d` | 输出区域半页滚动。 |
| `alt+u` / `alt+d` | 输入区域半页滚动。 |
| `?` | 打开可搜索的命令帮助。 |
| `q` | 普通模式下退出。 |

## CLI 使用

列出工具：

```bash
sharp list
```

输出版本信息：

```bash
sharp version
```

示例：

```bash
echo '{"data":{"id":1}}' | sharp json pretty
echo '{"data":{"id":1}}' | sharp json get --path data.id
sharp b64 encode hello
sharp b64 decode aGVsbG8=
sharp url encode "a=b&c=d"
sharp hash sha256 hello
sharp hmac sha256 --key secret hello
sharp time now
sharp time from 1714723200
sharp uuid v4
sharp password --length 32
sharp jwt decode "$JWT"
```

CLI 输入支持参数、文件路径或 stdin。

## 工具分类

| 类目 | 工具 |
| --- | --- |
| ![JSON](https://img.shields.io/badge/JSON-workbench-2EA44F) | 美化、压缩、校验、查询、排序、转义、反转义 |
| ![Encode](https://img.shields.io/badge/Encode-code-blue) | Base64、Base64 URL、Raw Base64 URL、URL、Hex、HTML、Unicode |
| ![Crypto](https://img.shields.io/badge/Crypto-hash-critical) | CRC32、MD5、SHA1、SHA224、SHA256、SHA384、SHA512、HMAC SHA256、HMAC SHA512 |
| ![Time](https://img.shields.io/badge/Time-timestamp-yellow) | 当前时间、时间戳转时间、时间转时间戳 |
| ![Text](https://img.shields.io/badge/Text-transform-informational) | 大小写、trim、命名风格转换、排序、去重、统计、正则测试、正则替换 |
| ![Network](https://img.shields.io/badge/Network-parse-9cf) | URL 解析、Query 解析、DNS 查询、CIDR 解析 |
| ![Convert](https://img.shields.io/badge/Convert-format-success) | JSON 转 YAML、YAML 转 JSON、CSV 转 JSON、HTTP Headers 转 JSON |
| ![Inspector](https://img.shields.io/badge/Inspector-JWT-lightgrey) | JWT 解码，不校验签名 |
| ![Generator](https://img.shields.io/badge/Generator-random-orange) | UUID v4、随机密码、随机 token |

## 开发

要求：

- Go 1.25.1 或与 `go.mod` 匹配的更新版本。

常用命令：

```bash
make test
make run
make build
make fmt
make tidy
```

构建时注入发布版本：

```bash
go build -ldflags "-X github.com/gkmz/sharp/internal/cli.Version=0.1.0" ./cmd/sharp
```

如果在沙箱环境中默认 Go 构建缓存不可写，可以使用：

```bash
GOCACHE=/private/tmp/sharp-gocache go test ./...
GOCACHE=/private/tmp/sharp-gocache go vet ./...
```

项目结构：

```text
cmd/sharp              CLI 入口
internal/cli           Cobra 命令树
internal/tui           Bubble Tea TUI
internal/tools         内置工具和默认 registry
pkg/tool               公共工具接口和 registry
docs                   产品和开发说明
```

## 文档

- [贡献指南](CONTRIBUTING.zh-CN.md)
- [行为规范](CODE_OF_CONDUCT.zh-CN.md)
- [安全策略](SECURITY.zh-CN.md)
- [变更日志](CHANGELOG.zh-CN.md)
- [产品需求](PRD.md)
- [开发计划](DEVELOPMENT_PLAN.md)

## 许可证

Apache License 2.0。见 [LICENSE](../LICENSE)。
