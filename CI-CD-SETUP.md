# CI/CD 自动打包配置

本项目同时支持 GitLab CI/CD 和 GitHub Actions 自动构建多平台可执行文件，支持用户直接下载使用。

## 🚀 快速开始

### GitLab CI/CD

1. **推送代码触发构建**：
   ```bash
   git add .
   git commit -m "Add CI/CD configuration"
   git push origin main
   ```

2. **查看构建结果**：
   - 进入 GitLab 项目的 CI/CD > Pipelines
   - 下载构建产物

### GitHub Actions

1. **推送代码触发构建**：
   ```bash
   git add .
   git commit -m "Add CI/CD configuration"
   git push github main
   ```

2. **查看构建结果**：
   - 进入 GitHub 项目的 Actions 页面
   - 下载构建产物

## 📦 构建产物

每次构建会自动生成以下文件：

- `excel-img-link-replacer-linux-amd64` - Linux 可执行文件
- `excel-img-link-replacer-windows-amd64.exe` - Windows 可执行文件  
- `excel-img-link-replacer-darwin-amd64` - macOS Intel 可执行文件
- `excel-img-link-replacer-darwin-arm64` - macOS Apple Silicon 可执行文件

## 🏷️ 创建 Release

使用 Git 标签创建正式发布版本：

### GitLab
```bash
# 创建标签
git tag v1.0.0
git push origin v1.0.0
```

### GitHub
```bash
# 创建标签
git tag v1.0.0
git push github v1.0.0
```

标签推送后会自动创建 Release 并上传所有平台的构建产物。

## 📋 使用说明

1. **下载对应平台的文件**：
   - Windows 用户：下载 `*-windows-amd64.exe`
   - macOS Intel：下载 `*-darwin-amd64`
   - macOS Apple Silicon：下载 `*-darwin-arm64`
   - Linux 用户：下载 `*-linux-amd64`

2. **运行应用程序**：
   - Windows：双击 `.exe` 文件
   - macOS：在终端中运行 `chmod +x 文件名 && ./文件名`
   - Linux：在终端中运行 `chmod +x 文件名 && ./文件名`

## 🔧 故障排除

如果构建失败，请检查：
1. 代码是否有语法错误
2. 所有依赖是否正确安装
3. GitLab CI/CD 运行器是否正常

---

**注意**：构建产物会在 GitLab 中保存 1 周，请及时下载。Release 版本会永久保存。