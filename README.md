# Excel 图片替换工具

一个基于 Go + Fyne 的桌面应用，用于把 Excel 指定工作表、指定列中的图片链接批量下载，并写入链接列右侧新增的图片列。当前正式验证目标为 Windows 和 macOS。

## 主要功能

- 选择或拖拽 `.xlsx` / `.xlsm` 文件
- 读取工作簿中的工作表，并由用户选择要处理的工作表
- 读取首行表头，支持勾选一个或多个图片链接列
- 并发下载 JPEG、PNG、WebP 图片，并统一写入 Excel
- 单个图片失败不阻断整体处理，完成后显示成功和失败数量
- 输出文件自动避开覆盖，例如 `book_output.xlsx`、`book_output_1.xlsx`
- 处理完成后可点击输出路径打开文件所在目录

## 项目结构

```text
excel_img_link_replacer/
├── cmd/desktop/main.go               # 程序入口
├── internal/app/ui/                  # Fyne 界面和交互
├── internal/app/usecase/             # Excel 处理流程、工作表和表头读取
├── internal/app/tools/               # 网络下载、图片处理、文件打开工具
├── .github/workflows/build.yml       # 测试、vet、Windows/macOS 打包
├── go.mod
├── go.sum
├── icon.png
└── README.md
```

## 环境要求

- Go 1.24.7
- Fyne CLI：`go install fyne.io/tools/cmd/fyne@latest`
- macOS：Xcode Command Line Tools
- Windows：Visual Studio Build Tools

## 开发运行

```bash
go mod download
go run ./cmd/desktop
```

也可以先编译：

```bash
go build -o excel_tool ./cmd/desktop
./excel_tool
```

## 本地验证

```bash
go test ./...
go vet ./...
go build ./cmd/desktop
```

测试覆盖了 URL 校验、下载失败汇总、图片处理、唯一输出文件名、多工作表 Excel fixture 和 UI 摘要逻辑。

## 打包发布

Fyne 桌面应用依赖 CGO 和平台图形库，不建议在一台机器上直接 `GOOS=... go build` 交叉编译正式产物。项目使用 GitHub Actions 在原生 runner 上打包：

- Windows amd64：`windows-latest`
- macOS Apple Silicon / arm64：`macos-14`

当前不提供 macOS Intel 二进制包。Intel Mac 用户如有需要，可以在 Intel Mac 本机自行编译：

```bash
make package-macos VERSION=x.y.z
```

或直接构建可执行文件：

```bash
go build ./cmd/desktop
```

CI 会先执行：

```bash
go test ./...
go vet ./...
```

通过后再上传 Windows 和 macOS arm64 打包产物。推送 `v*` tag 时会创建 GitHub Release。

### Windows 未签名安装说明

由于当前没有企业代码签名证书，也没有 Azure Trusted Signing 资源，Windows 包会以未签名测试包发布：

```text
Excel图片链接替换工具-windows-amd64-unsigned.zip
```

普通用户安装方式：

1. 从企业内部可信下载渠道下载 zip
2. 解压 zip
3. 双击 `Excel图片链接替换工具.exe`
4. 如出现 Windows Defender SmartScreen 提示，点击“更多信息”
5. 点击“仍要运行”

注意：未签名包可能被浏览器、Windows Defender SmartScreen 或安全软件提示风险。该包仅建议用于企业内部可信渠道的小范围分发。

### 为什么不使用自签名证书

自签名证书可以免费生成，也可以用 `signtool` 给 exe 加签名，但普通用户电脑默认不信任这个证书，仍然会出现安全提示。若要让自签名证书被信任，需要把根证书安装到每台用户电脑的 Trusted Root，这通常需要管理员权限或 IT 管控。因此在没有企业资源、用户也没有技术背景的情况下，自签名不是更简单的方案。

## 使用说明

1. 启动程序
2. 拖拽 Excel 文件，或点击“选择 Excel 文件”
3. 选择要处理的工作表
4. 勾选包含图片链接的表头列
5. 点击“开始处理并生成”
6. 查看成功/失败统计，点击输出路径打开文件所在目录

## 处理规则

- 只处理选中工作表，不自动处理其他工作表
- 只接受 `http` 和 `https` 图片链接
- 空单元格会跳过
- 下载失败、非法 URL、图片写入失败会进入失败摘要
- 成功写入图片后会保留原链接，并在链接列右侧新增 `<表头>_图片` 列保存图片
- 失败单元格保留原内容，便于人工复查

## 常见问题

- 文件无法保存：确认原 Excel 未被其他程序锁定，并且目标目录可写
- 图片下载失败：检查网络、URL 有效性、服务器状态和图片格式
- macOS 无法打开未签名应用：当前产物未做代码签名，需要按发布策略补充签名和公证
