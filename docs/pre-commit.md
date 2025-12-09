# Pre-commit Configuration

这个项目使用 [pre-commit](https://pre-commit.com/) 框架来自动化代码质量检查。

## 快速开始

### 1. 安装 pre-commit

```bash
# macOS
brew install pre-commit

# 或使用 pip
pip install pre-commit
# 或
pip3 install pre-commit
```

### 2. 安装 hooks

```bash
# 使用 Makefile（推荐）
make pre-commit-install

# 或直接使用 pre-commit
pre-commit install
```

### 3. 首次运行（可选）

```bash
# 在所有文件上运行检查
make pre-commit-run

# 或
pre-commit run --all-files
```

## 配置的检查项

### 通用检查

- ✅ 删除行尾空格
- ✅ 确保文件以换行符结尾
- ✅ YAML 文件语法检查
- ✅ 检查大文件（>1MB）
- ✅ 检查合并冲突标记
- ✅ 检查私钥泄露
- ✅ 检查敏感信息（secrets）

### Go 相关

- ✅ `go fmt` - 代码格式化
- ✅ `go vet` - 静态分析
- ✅ `go imports` - 导入排序
- ✅ `go mod tidy` - 依赖整理
- ✅ `golangci-lint` - 综合代码检查

### Web/Frontend

- ✅ Prettier - 代码格式化（TypeScript, JavaScript, JSON, CSS, Markdown）
- ✅ ESLint - JavaScript/TypeScript 代码检查

### 配置文件

- ✅ YAML 语法和格式检查
- ✅ Dockerfile 检查（hadolint）

### 文档

- ✅ Markdown 格式检查

### Shell 脚本

- ✅ ShellCheck - Shell 脚本检查

## 使用方法

### 自动运行

配置完成后，每次 `git commit` 时会自动运行检查：

```bash
git add .
git commit -m "your message"
# pre-commit 会自动运行所有检查
```

### 手动运行

```bash
# 检查所有文件
make pre-commit-run

# 只检查暂存的文件
pre-commit run

# 检查特定的 hook
pre-commit run go-fmt
pre-commit run eslint
```

### 跳过检查（不推荐）

```bash
# 跳过所有 pre-commit 检查
git commit --no-verify -m "message"

# 跳过特定的 hook
SKIP=eslint git commit -m "message"
```

## Makefile 命令

项目提供了便捷的 Makefile 目标：

```bash
make pre-commit-install    # 安装 pre-commit hooks
make pre-commit-run        # 在所有文件上运行检查
make pre-commit-update     # 更新 hooks 到最新版本
make pre-commit-uninstall  # 卸载 pre-commit hooks
```

## 更新 Hooks

定期更新 hooks 到最新版本：

```bash
make pre-commit-update
```

## 配置文件说明

- **`.pre-commit-config.yaml`** - Pre-commit 主配置文件
- **`.yamllint`** - YAML 检查配置
- **`.markdownlint.yaml`** - Markdown 检查配置
- **`.secrets.baseline`** - 敏感信息检查基线（自动生成）

## 自定义配置

### 修改检查规则

编辑 `.pre-commit-config.yaml` 文件：

```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
        # 添加自定义参数
        args: ['--markdown-linebreak-ext=md']
```

### 排除特定文件

在 `.pre-commit-config.yaml` 中使用 `exclude` 参数：

```yaml
- id: prettier
  exclude: ^(dist/|build/|node_modules/)
```

### 禁用特定 hook

注释掉不需要的 hook：

```yaml
# - id: markdownlint  # 临时禁用
```

## CI/CD 集成

在 CI 流程中运行 pre-commit：

```bash
# 在 CI 脚本中
pip install pre-commit
pre-commit run --all-files
```

## 故障排除

### Python 版本问题

如果遇到类似 "Could not find a version that satisfies the requirement setuptools" 的错误：

```bash
# 1. 清理缓存
pre-commit clean

# 2. 检查 Python 版本
python3 --version

# 3. 确保使用稳定版本（Python 3.9-3.12）
# 编辑 .pre-commit-config.yaml，设置：
# default_language_version:
#   python: python3

# 4. 重新安装
pre-commit install
```

### Hook 安装失败

```bash
# 清理并重新安装
pre-commit clean
pre-commit install --install-hooks
```

### golangci-lint 失败

```bash
# 确保 golangci-lint 已安装
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 或使用 brew
brew install golangci-lint
```

### ESLint 在 web 目录失败

```bash
cd web/console
npm install
```

## 性能优化

如果 pre-commit 运行较慢：

1. 只检查变更的文件（默认行为）
2. 使用 `--fast` 参数（部分 hooks 支持）
3. 在 `.pre-commit-config.yaml` 中移除不需要的 hooks

## 参考资源

- [Pre-commit 官方文档](https://pre-commit.com/)
- [Pre-commit Hooks 列表](https://pre-commit.com/hooks.html)
- [Go Pre-commit Hooks](https://github.com/dnephin/pre-commit-golang)
