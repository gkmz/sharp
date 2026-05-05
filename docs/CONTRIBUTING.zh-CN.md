# 为 sharp 贡献代码

感谢你考虑参与贡献。`sharp` 是一个终端优先的开发者工具箱，因此所有改动都应保持快速、可预测、键盘友好。

## 如何贡献

1. 对于较大的功能或行为变更，请先创建 issue 讨论。
2. Fork 仓库并创建主题分支。
3. 保持改动聚焦，避免在功能或 bug fix PR 中混入无关重构。
4. 为行为变化添加或更新测试。
5. 提交 PR 前运行格式化和验证命令。

## 开发环境

```bash
git clone https://github.com/gkmz/sharp.git
cd sharp
go mod download
```

常用命令：

```bash
make fmt
make test
go vet ./...
```

如果当前环境不能写入默认 Go 构建缓存：

```bash
GOCACHE=/private/tmp/sharp-gocache go test ./...
GOCACHE=/private/tmp/sharp-gocache go vet ./...
```

## 代码规范

- 工具应保持小而清晰，具备稳定 ID、名称、描述和可预测行为。
- 新工具放在 `internal/tools/<category>`，并通过 `internal/tools/registry.go` 注册。
- 新增核心行为时，在 `internal/tools/tools_test.go` 增加 registry 级测试。
- TUI 交互应保持键盘优先，并遵循现有焦点、帮助、状态栏模式。
- 导出的 Go 符号必须有文档注释。
- 标准库或现有解析器可用时，优先使用结构化解析，避免临时字符串处理。

## 添加新工具

1. 选择稳定 ID，通常为 `<group>.<action>`，例如 `json.pretty`。
2. 使用 `tool.SimpleTool` 或实现 `tool.Tool`。
3. 分配正确的 `tool.Category`。
4. 用 `tool.Option` 定义运行参数。
5. 添加正常输入和无效输入测试。
6. 如果是用户可见工具，同步更新 README 工具列表。

## PR 检查清单

- [ ] 代码已用 `gofmt` 格式化。
- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] 新行为有测试覆盖。
- [ ] 新增公共 API 有 Go doc 注释。
- [ ] 用户可见行为变化已更新 README 或相关文档。
