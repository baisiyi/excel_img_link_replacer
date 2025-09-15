# Excel 图片替换工具

一个基于 Go + Fyne 开发的跨平台桌面应用程序，用于将 Excel 文件中的图片链接批量替换为实际的图片。

## ✨ 主要功能

- **文件选择**：支持拖拽或点击选择 Excel 文件（.xlsx/.xlsm）
- **智能表头识别**：自动读取 Excel 首行表头，支持多选需要处理的列
- **批量图片下载**：并发下载选中列中的图片链接（支持 JPEG、PNG、WebP）
- **图片处理优化**：自动调整图片尺寸适配 Excel 单元格，保持宽高比
- **可点击输出路径**：处理完成后可直接点击文件路径打开文件所在目录
- **跨平台支持**：支持 Windows、macOS、Linux 系统
- **实时进度显示**：处理过程中显示进度条和状态信息

## 🏗️ 项目结构

```
excel_img_link_replacer/
├── cmd/
│   └── desktop/
│       └── main.go                    # 程序入口点
├── internal/
│   └── app/
│       ├── ui/                        # 用户界面层
│       │   ├── ui.go                  # 主界面构建和交互逻辑
│       │   └── load_headers.go        # 表头加载处理
│       ├── usecase/                   # 业务逻辑层
│       │   ├── excel_process.go       # Excel 处理核心逻辑
│       │   └── headers.go             # 表头解析和验证
│       └── tools/                     # 工具层
│           ├── net.go                 # 网络下载工具（并发下载、格式支持）
│           ├── pic.go                 # 图片处理工具（尺寸调整、格式转换）
│           └── file.go                # 文件系统工具（跨平台文件操作）
├── pic_export.go                      # 图片导出封装接口
├── go.mod                             # Go 模块依赖
├── go.sum                             # 依赖版本锁定
└── README.md                          # 项目说明文档
```

## 🚀 快速开始

### 环境要求

- Go 1.24+ 
- Fyne 工具链（用于打包）

### 安装依赖

```bash
# 克隆项目
git clone <repository-url>
cd excel_img_link_replacer

# 下载依赖
go mod download
```

### 开发运行

```bash
# 直接运行（开发模式）
go run ./cmd/desktop

# 或者先编译再运行
go build -o excel_tool ./cmd/desktop
./excel_tool
```

## 📦 跨平台打包

### 安装 Fyne 工具

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

### macOS 打包

```bash
# 生成 .app 文件（未签名）
fyne package -os darwin -icon icon.png -name "Excel 图片替换工具" -release -appID com.example.pictool -src ./cmd/desktop

# 在 ARM Mac 上生成 Intel 版本
GOOS=darwin GOARCH=amd64 fyne package -os darwin -icon icon.png -name "Excel 图片替换工具" -release -appID com.example.pictool -src ./cmd/desktop

# 在 Intel Mac 上生成 ARM 版本  
GOOS=darwin GOARCH=arm64 fyne package -os darwin -icon icon.png -name "Excel 图片替换工具" -release -appID com.example.pictool -src ./cmd/desktop

# 生成 .dmg 文件（需要额外工具）
# 建议使用第三方打包器如 create-dmg
```

### Windows 打包

```bash
# 生成 .exe 文件
fyne package -os windows -icon icon.png -name "Excel 图片替换工具" -release -appID com.example.pictool -src ./cmd/desktop

# 可进一步使用 NSIS 或 Inno Setup 创建安装包
```

### Linux 打包

```bash
# 生成 Linux 可执行文件
fyne package -os linux -icon icon.png -name "Excel 图片替换工具" -release -appID com.example.pictool -src ./cmd/desktop
```

### 交叉编译

```bash
# 设置目标平台
export GOOS=windows
export GOARCH=amd64
go build -o excel_tool.exe ./cmd/desktop

# 或使用 fyne 工具
fyne package -os windows -arch amd64 -icon icon.png -name "Excel 图片替换工具" -src ./cmd/desktop
```

## 🛠️ 开发指南

### 项目架构

项目采用分层架构设计：

- **UI 层** (`internal/app/ui/`)：负责用户界面和交互逻辑
- **用例层** (`internal/app/usecase/`)：包含核心业务逻辑
- **工具层** (`internal/app/tools/`)：提供各种工具函数和跨平台支持

### 开发环境设置

1. **安装 Go 开发环境**
   ```bash
   # 安装 Go 1.24+
   # 配置 GOPATH 和 GOROOT
   ```

2. **安装 Fyne 开发工具**
   ```bash
   go install fyne.io/fyne/v2/cmd/fyne@latest
   ```

3. **安装系统依赖**
   - **macOS**: 需要 Xcode Command Line Tools
   - **Windows**: 需要 Visual Studio Build Tools
   - **Linux**: 需要 gcc 和开发库

### 代码结构说明

#### UI 层 (`internal/app/ui/`)
- `ui.go`: 主界面构建，包含拖拽区域、文件选择、表头多选、进度显示
- `load_headers.go`: 异步加载 Excel 表头，避免界面阻塞

#### 用例层 (`internal/app/usecase/`)
- `excel_process.go`: 核心处理逻辑，包括 URL 收集、批量下载、图片写入
- `headers.go`: 表头解析和验证

#### 工具层 (`internal/app/tools/`)
- `net.go`: 网络下载工具，支持并发下载、格式检测、错误处理
- `pic.go`: 图片处理工具，包括尺寸调整、格式转换、Excel 集成
- `file.go`: 文件系统工具，提供跨平台文件操作

### 开发工作流

1. **功能开发**
   ```bash
   # 启动开发服务器
   go run ./cmd/desktop
   
   # 代码修改后自动重新编译运行
   ```

2. **代码测试**
   ```bash
   # 运行测试
   go test ./...
   
   # 检查代码质量
   go vet ./...
   ```

3. **构建验证**
   ```bash
   # 编译检查
   go build ./cmd/desktop
   
   # 跨平台编译测试
   GOOS=windows GOARCH=amd64 go build ./cmd/desktop
   ```

### 添加新功能

1. **UI 功能**：在 `internal/app/ui/` 中添加新的界面组件
2. **业务逻辑**：在 `internal/app/usecase/` 中实现核心功能
3. **工具函数**：在 `internal/app/tools/` 中添加可复用的工具

## 📋 使用说明

1. **启动程序**：运行可执行文件或使用 `go run ./cmd/desktop`
2. **选择文件**：拖拽 Excel 文件到程序窗口或点击选择按钮
3. **选择列**：在右侧面板勾选需要处理的表头列（默认勾选"商品图片链接"）
4. **开始处理**：点击"开始处理并生成"按钮
5. **查看结果**：处理完成后点击输出文件路径打开文件所在目录

## 🔧 技术特性

### 网络下载优化
- 并发下载（默认 8 个协程）
- 连接池复用
- 支持重定向（最多 10 次）
- 超时控制（30 秒）

### 图片处理
- 支持 JPEG、PNG、WebP 格式
- 自动格式检测（魔数验证）
- 统一转换为 PNG 格式
- 智能尺寸调整（保持宽高比）
- 高质量缩放算法

### 跨平台支持
- Windows: 使用 `explorer` 命令
- macOS: 使用 `open` 命令
- Linux: 使用 `xdg-open` 命令

## 🐛 常见问题

### 编译问题
- **CGO 错误**：确保安装了系统开发工具
- **依赖问题**：运行 `go mod tidy` 更新依赖

### 运行时问题
- **文件无法打开**：检查文件权限和路径
- **图片下载失败**：检查网络连接和 URL 有效性
- **Excel 写入失败**：确保文件未被其他程序占用

### 打包问题
- **图标不显示**：确保图标文件存在且格式正确
- **签名问题**：macOS 需要开发者证书进行签名
- **依赖缺失**：使用 `fyne package` 自动处理依赖

## 📄 许可证

本项目采用 MIT 许可证，详见 LICENSE 文件。

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📞 支持

如有问题或建议，请提交 Issue 或联系维护者。