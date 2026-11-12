# MarkGo

[![CI](https://github.com/xddcode/markgo/actions/workflows/ci.yml/badge.svg)](https://github.com/xddcode/markgo/actions/workflows/ci.yml)
[![许可证](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.27.4%2B-00ADD8.svg)](go.mod)

**[English](README.md) | 简体中文**

MarkGo 是一款基于 Go 和 [MyGo](https://github.com/egoist/mygo) 的免费开源、本地优先 AI Markdown 编辑器。原生界面不依赖 Electron、内嵌浏览器或 JavaScript 层。

> MarkGo 当前处于工程基础建设阶段。源码已提供原生工作台、可编辑的内存示例文档、Markdown 实时预览和 UI 行为测试，但尚不支持真实文件打开与保存，也尚未连接 AI 模型。

## 项目原则

- **纯 Markdown**：文档始终是可迁移的普通文件，不绑定私有数据库。
- **本地优先**：无需账号或 MarkGo 服务器即可完成核心写作工作。
- **AI 必须可控**：模型读取的上下文和产生的修改应当可见、可审阅、可撤销。
- **原生且专注**：低干扰写作界面、流畅交互，不强制携带 Web 运行时。
- **完全开源**：无订阅、功能锁、广告和遥测。

## 当前状态

| 能力 | 状态 |
| --- | --- |
| 原生三栏工作台 | 可用 |
| 内存 Markdown 示例文档编辑 | 可用 |
| 文档切换与筛选 | 可用 |
| 源码标记实时预览、可点击任务框 | 可用 |
| 选区上下文与 AI 面板框架 | 可用 |
| 无窗口 UI 行为测试 | 可用 |
| 文件打开、保存与恢复 | 规划中 |
| 完整 CommonMark/GFM 编辑与渲染 | 规划中 |
| 本地模型接入与可审阅 AI Diff | 规划中 |
| 正式发布与安装包 | 规划中 |

项目方向和非目标见 [ROADMAP.md](ROADMAP.md)。

## 从源码运行

环境要求：Go 1.27.4 或更高版本、Git，以及 MyGo 支持的桌面平台。建议安装 Make 以使用统一的质量检查命令。

```sh
git clone https://github.com/xddcode/markgo.git
cd markgo
go run ./cmd/markgo
```

开发与验证：

```sh
go tool mygo dev .
make check
make race
make security
go tool mygo build
```

详见[开发指南](docs/development/setup.md)和[测试策略](docs/testing/strategy.md)。

## 文档

- [文档索引](docs/README.md)
- [架构概览](docs/architecture/overview.md)
- [设计规范](docs/design/design-system.md)
- [贡献指南](CONTRIBUTING.md)
- [安全政策](SECURITY.md)
- [更新日志](CHANGELOG.md)

## 许可证

MarkGo 使用 [Apache License 2.0](LICENSE)。署名与第三方声明请参阅 [NOTICE](NOTICE) 和 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。MarkGo 名称和 Logo 不以 Apache-2.0 许可授权。
