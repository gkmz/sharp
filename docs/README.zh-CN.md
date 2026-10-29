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

TUI 分为三个带数字快捷键的区域：

| 区域 | 用途 |
| --- | --- |
| `1 Categories` | 浏览类目和子类目，直接输入文字即可筛选工具。 |
| `2 Actions` | 选择当前工具组中的动作，例如 Encode/Decode 或 Pretty/Minify。 |
| `3 Workspace` | 编辑输入/参数、预览或执行当前动作、查看输出。 |

搜索框不再常驻。在 Categories 直接输入 `json`、`minify` 或工具名即可搜索，用上下方向键选择结果，Enter 打开，Esc 清空筛选。只有一个工具时直接进入工作区；打开结果后清空筛选，恢复该组全部动作。在其他区域的非编辑状态下，可按 `/` 开始搜索。Categories 中的 `j`、`k`、`q` 等字母均作为搜索文字。

所有工具组采用同一套工作台规则。组内动作共享输入、输出和输入撤销记录：Base64 编码/解码、文本转换、哈希算法、正则匹配/替换切换时都会保留当前文档。每个动作在内存中分别保留参数草稿；不同类目和子类目之间的文档相互独立。输出标题注明生成结果的动作。

Normal 模式处理命令与导航，只有 `i` 或 `o` 才进入编辑。Tab/Shift+Tab 循环切换 Categories → Actions → Workspace。在任意区域按 `[`/`]` 即可切换动作并保持所在区域，也可在 Actions 用左右方向键或 `h/l` 切换。已知的本地转换在 `v` 粘贴、切换动作后立即预览；编辑停顿约 200ms 后自动更新，Esc 结束编辑时立即补齐预览。DNS 查询、当前时间和随机生成只在 `r` 或 Enter 时运行。只有当前动作需要参数时才显示 Options：`o` 编辑，Enter 执行，Esc 结束编辑。无输入动作隐藏 Input，同时保留组内其他动作的文档。

JSON 使用相同规则，提供 Pretty、Minify、Sort、Query、Escape、Unescape、Check。仅 Query 显示 Path，在路径中按 Enter 查询共用输入；Check 在状态行显示校验结果。所有工具组的 `p`/`a` 都将成功结果应用回 Input 并清空 Output，错误信息不能回填输入。输入或参数尚未完整时，旧结果保留并标记为 `stale`，更新完成前无法复制、保存或应用。

Input 和 Output 无需单独选中。`i` 直接编辑 Input，`o` 编辑 Options/Path。在 Input 中 Enter 换行；在参数中 Enter 执行并返回 Normal。编辑时字母只作为文本，Tab 不切换区域，需要先按 Esc；Normal 模式下 Workspace 的上下方向键直接滚动 Output。工具参数使用 `key=value`，带空格的值需要加引号，例如 `replacement="hello world"`。HMAC key 等敏感参数会遮蔽显示。按 `?` 查看命令和参数说明。操作反馈显示在底部快捷键提示的上方。

非编辑状态下的常用快捷键：

| 快捷键 | 动作 |
| --- | --- |
| Tab / Shift+Tab | 在 Normal 模式下向前 / 向后循环切换区域。 |
| `[` / `]` | 切换上一 / 下一动作，保持所在区域。 |
| `1`, `2`, `3` | 聚焦 Categories、Actions、Workspace。 |
| Categories 中输入文字 / `/` | 开始工具搜索，筛选框临时显示在类目内。 |
| 上 / 下方向键 | 选择类目或搜索结果；在 Workspace 中滚动 Output。 |
| 左 / 右方向键或 `h` / `l` | 在区域 2 选择动作。 |
| Enter | 在类目中打开所选项；在区域 2、3 执行当前工具/动作。 |
| `i` | 直接编辑 Input。 |
| `o` | 编辑参数；JSON 选择 Query 时编辑 Path。 |
| `r` | 执行所选工具或动作。 |
| `v` | 粘贴到 Input 并自动预览，保持 Normal 模式。 |
| `x` / `X` | 清空输入 / 清空输出。 |
| `y` | 复制 trim 后的输出。 |
| `s` | 保存 trim 后的输出到 `sharp-output.txt`。 |
| `p` / `a` | 将输出应用回输入，便于链式处理。 |
| `z` | 撤销输入修改；编辑 Input 时用 Ctrl+Z/Ctrl+Y 撤销/重做。 |
| `ctrl+u` / `ctrl+d` | 输出区域半页滚动。 |
| `alt+u` / `alt+d` | 输入区域半页滚动。 |
| Esc | 结束编辑 → Workspace → Actions → Categories；搜索时清空筛选。 |
| `?` | 打开可搜索的命令帮助。 |
| `ctrl+c` / `ctrl+q` | 在任意区域退出；Categories、搜索和编辑以外也可按 `q` 退出。 |

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

## 工具扩展待办

以下清单用于持续追踪工具扩展。当前已实现的能力见上方“工具分类”；未勾选的项目均为规划，尚不可用。优先级表示建议的实施顺序，不代表版本交付承诺。

sharp 聚焦本地运行、可组合的数据与文本工作流。每项工具的共用实现需同时支持 CLI 和 TUI，通过相关测试，并同步更新中英文文档，才算完成。交付后逐项勾选；具体命令名在实现时确定。

### P0 — 日常高频工作流

- [ ] 文本 Diff：比较两份输入，展示新增、删除和上下文。
- [ ] 数字进制转换：二进制、八进制、十进制、十六进制互转。
- [ ] JSON 转 CSV：补齐已有 CSV 转 JSON 的反向转换。
- [ ] 字符串检查器：在已有文本统计上增加 Unicode 码点、不可见字符和字节详情。
- [ ] 反斜杠转义/反转义：明确支持的转义规则，与已有 JSON、Go 风格 Unicode 工具区分。
- [ ] 随机字符串生成：支持自定义长度和字符集，补充已有密码与 token 生成。
- [ ] Cron 解析：解释表达式，并按指定时区展示后续执行时间。
- [ ] 颜色转换：HEX、RGB、HSL、HSV、CMYK 互转。
- [ ] X.509 证书解码：在本地检查 PEM/DER 证书字段、有效期、SAN 和指纹。
- [ ] cURL 解析：提取方法、URL、Headers 和 Body，不执行命令。
- [ ] Query String 构建：从结构化参数生成查询字符串，补齐已有解析能力。
- [ ] JWT 签名验证：在已有解码工具上增加明确的算法和密钥选项。

### P1 — 格式化与结构化数据

- [ ] HTML 美化/压缩。
- [ ] CSS 美化/压缩。
- [ ] JavaScript/TypeScript 美化/压缩。
- [ ] XML 美化/压缩。
- [ ] SQL 格式化，支持明确选择方言。
- [ ] JSON 转代码：先支持 Go struct 和 TypeScript interface，后续考虑 Java、Python。
- [ ] TOML ↔ JSON 互转。
- [ ] ENV ↔ JSON 互转。
- [ ] INI ↔ JSON 互转。
- [ ] Markdown 终端预览。
- [ ] HTTP 请求构造与发送：支持超时设置和易读的响应输出。
- [ ] TLS 证书检查：获取并检查远端服务的证书。
- [ ] IP 检查：本地地址分类与子网信息。
- [ ] 端口连通性检查。
- [ ] MIME 类型查询。
- [ ] Data URI 解析。
- [ ] cURL 转代码：基于已解析命令生成请求代码片段。
- [ ] UUID/ULID 工具：在 UUID v4 生成基础上增加 ULID 生成及 UUID/ULID 信息检查。

### P2 — 专项与可视化工具

- [ ] 二维码生成。
- [ ] 从图片文件识别二维码。
- [ ] Base64 图片编码/解码，支持文件和 Data URI。
- [ ] HTML 浏览器预览。
- [ ] HTML 转 JSX。
- [ ] SVG 转 CSS。
- [ ] Lorem Ipsum 占位文本生成。
- [ ] PHP ↔ JSON 互转。
- [ ] PHP 序列化/反序列化。
- [ ] ERB 美化/压缩。
- [ ] LESS 美化/压缩。
- [ ] SCSS 美化/压缩。

### 配套能力

根据上述工具的实际需要逐步补充：

- [ ] 双输入 CLI/TUI 工作区，用于文本 Diff 等比较工具。
- [ ] 明确区分文本、文件、二进制输入，并支持保留原始字节的文件输出。
- [ ] 根据 MIME 类型处理结构化文本、图片和预览输出。

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
