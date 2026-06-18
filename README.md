# Paste - 现代化 macOS 剪贴板管理工具

Paste 是一款专为 macOS 设计的现代化剪贴板管理工具，采用 Go + Electron 架构构建，提供流畅、高效、安全的剪贴板历史管理体验。

## ✨ 主要特性

### 核心功能
- **持续剪贴板监听** - 自动记录所有复制的文本和图片内容
- **智能去重** - 基于内容哈希自动识别并合并重复记录
- **快速搜索** - 毫秒级全文搜索，支持 5000+ 条历史记录
- **收藏夹** - 收藏常用内容，方便快速访问
- **全局快捷键** - 默认 `⌘⇧V` 快速唤出，可自定义
- **菜单栏常驻** - 后台静默运行，不占用 Dock 栏
- **自动粘贴** - 双击条目自动粘贴到当前应用
- **开机启动** - 支持登录时自动启动

### 隐私与安全
- **完全本地运行** - 所有数据存储在本地，不上传任何用户数据
- **无遥测/分析** - 不包含任何行为跟踪或数据分析功能
- **敏感应用排除** - 自动忽略密码管理器、银行应用等敏感场景
- **内容黑名单** - 基于正则表达式的内容过滤规则
- **自定义忽略应用** - 用户可自定义需要排除的应用列表

### UI/UX 体验
- **毛玻璃效果** - macOS 原生风格的透明毛玻璃界面
- **深浅色模式** - 支持浅色/深色/跟随系统三种主题
- **流畅动画** - Framer Motion 驱动的丝滑动画
- **Retina 适配** - 完美支持高分辨率显示
- **键盘操作** - 完整的键盘快捷键支持

### 性能指标
- 🚀 启动时间 < 500ms
- ⚡ 搜索延迟 < 50ms
- 📦 支持 5000+ 条历史记录保持流畅
- 🔋 后台监听长期 CPU 占用 < 1%
- 💾 内存占用优化

## 🏗️ 技术架构

### 后端 (Go)
- **数据库**: SQLite (WAL 模式)
- **HTTP 服务**: Gin 框架
- **日志**: Zap (结构化日志)
- **剪贴板**: atotto/clipboard + macOS osascript
- **配置**: JSON 文件持久化

### 前端 (Electron + React)
- **框架**: React 18 + TypeScript
- **样式**: Tailwind CSS
- **状态管理**: Zustand
- **动画**: Framer Motion
- **HTTP 客户端**: Axios
- **UI**: 原生 macOS 毛玻璃 (vibrancy)

### 前后端通信
- 本地 HTTP API (127.0.0.1)
- Electron IPC 桥接

## 📁 项目结构

```
paste/
├── backend/                     # Go 后端
│   ├── cmd/
│   │   └── main.go              # 程序入口
│   ├── internal/
│   │   ├── api/                 # HTTP API 服务
│   │   ├── clipboard/           # 剪贴板监听
│   │   ├── config/              # 配置管理
│   │   ├── logger/              # 日志系统
│   │   ├── paste/               # 粘贴功能
│   │   ├── security/            # 安全与隐私
│   │   ├── storage/             # SQLite 存储
│   │   └── autostart/           # 开机启动
│   ├── pkg/
│   │   ├── models/              # 数据模型
│   │   └── utils/               # 工具函数
│   └── bin/                     # 编译输出
├── frontend/                    # Electron 前端
│   ├── src/
│   │   ├── main/                # Electron 主进程
│   │   │   ├── index.ts         # 主进程入口
│   │   │   └── preload.ts       # 预加载脚本
│   │   └── renderer/            # 渲染进程 (React)
│   │       ├── components/      # UI 组件
│   │       ├── stores/          # Zustand 状态管理
│   │       ├── api/             # API 客户端
│   │       ├── types/           # TypeScript 类型
│   │       ├── utils/           # 工具函数
│   │       ├── styles/          # 全局样式
│   │       ├── App.tsx          # 应用根组件
│   │       └── main.tsx         # 渲染入口
│   ├── build/                   # 构建资源
│   └── public/                  # 静态资源
├── Makefile                     # 构建脚本
├── go.mod
└── package.json
```

## 🚀 快速开始

### 环境要求
- macOS 11.0+
- Go 1.21+
- Node.js 18+
- npm 或 yarn

### 安装依赖

```bash
# 一键安装所有依赖
make install-deps

# 或分别安装
cd backend && go mod download
cd ../frontend && npm install
```

### 开发模式

```bash
# 启动后端
make dev-backend

# 启动前端（另开一个终端）
make dev-frontend
```

### 构建生产版本

```bash
# 构建后端 + 前端
make build

# 打包应用
make pack

# 构建发布版本 (.dmg / .zip)
make dist
```

### 运行测试

```bash
# 运行所有测试
make test

# 仅后端测试
make test-backend

# 仅前端测试
make test-frontend
```

## ⌨️ 快捷键

| 快捷键 | 功能 |
|--------|------|
| `⌘⇧V` | 唤出/隐藏 Paste 窗口 |
| `⌘F` | 聚焦搜索框 |
| `↑` / `↓` | 上下选择条目 |
| `Enter` | 粘贴选中条目 |
| `⌫` / `Delete` | 删除选中条目 |
| `Esc` | 隐藏窗口 |

## 📊 API 接口

后端 HTTP API 运行在 `http://127.0.0.1:48175/api/v1`

### 剪贴板记录
- `GET /items` - 获取剪贴板列表
- `GET /items/:id` - 获取单条记录
- `DELETE /items/:id` - 删除记录
- `PUT /items/:id/favorite` - 切换收藏
- `POST /items/:id/copy` - 复制到剪贴板
- `POST /items/:id/paste` - 自动粘贴

### 搜索
- `GET /search?q=关键词` - 搜索剪贴板内容

### 配置
- `GET /config` - 获取配置
- `PUT /config` - 更新配置

### 统计
- `GET /stats` - 获取数据统计
- `DELETE /items` - 清除历史记录

## 🔒 隐私承诺

Paste 严格保护您的隐私：

1. **100% 本地运行** - 所有剪贴板数据仅存储在您的 Mac 上
2. **无网络请求** - 除了本地 API 通信，不发起任何外部网络请求
3. **无遥测** - 不收集、不上传任何使用数据
4. **开源透明** - 代码完全开源可审计
5. **敏感保护** - 默认启用敏感应用和内容排除功能

## 🧪 测试覆盖

项目包含完善的测试覆盖：

- **后端单元测试** - 存储层、安全模块、配置管理
- **端到端测试** - API 接口测试

运行测试：

```bash
make test
```

## 📝 License

MIT License - 详见 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**Made with ❤️ for macOS users**
