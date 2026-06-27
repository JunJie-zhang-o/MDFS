# MDFS

MDFS 是一个轻量、只读的 HTTP 文件服务器。它使用 Go 提供安全的目录浏览与文件下载 API，并将 Vue 3 页面嵌入单个可执行文件。

## 功能

- 浏览共享根目录及其子目录
- 显示文件大小和修改时间
- 直接下载文件
- 阻止目录越界和符号链接逃逸
- 使用 TOML 配置监听地址和共享目录
- 构建包含 Vue 页面的单个 Linux 可执行文件

## 环境要求

- Go 1.25+
- Node.js 22+ 和 npm 10+
- GoReleaser 2+
- Bash 和 Make（执行发布构建时）
- 已有至少一次提交和 remote 的 Git 仓库（GoReleaser 读取版本信息所需）

## 快速开始

先创建或修改一个配置文件，确保 `share.path` 指向已存在的目录：

```toml
[server]
listen = "0.0.0.0:8080"

[share]
path = "/srv/mdfs"
```

分别启动后端和前端开发服务器：

```bash
cd frontend && npm install
make dev-api
make dev-web
```

浏览器访问 `http://localhost:5173`。Vite 会将 `/api` 请求代理到 `http://127.0.0.1:8080`。

## 构建发布包

```bash
make build
```

默认生成：

```text
dist/
├── mdfs_0.0.0-snapshot_linux_amd64.tar.gz
├── mdfs_0.0.0-snapshot_linux_arm64.tar.gz
└── checksums.txt
```

每个压缩包都包含：

```text
mdfs_0.0.0-snapshot_linux_<架构>/
├── mdfs
├── README.md
├── config/mdfs.toml
└── systemd/mdfs.service
```

`make build` 生成本地 snapshot，不会发布。GoReleaser 使用 Git 信息生成构建元数据。

## 运行

```bash
tar -xzf dist/mdfs_0.0.0-snapshot_linux_amd64.tar.gz
cd dist/mdfs_0.0.0-snapshot_linux_amd64
./mdfs --config ./config/mdfs.toml
```

程序只接受 `--config` 参数。配置文件不存在、格式错误或共享目录不可用时会直接退出。

## systemd

发布包中的 unit 文件使用以下默认路径：

- 程序：`/opt/mdfs/bin/mdfs`
- 配置：`/etc/mdfs/mdfs.toml`
- 运行用户和组：`mdfs`
- 只读共享目录：`/srv/mdfs`

安装前需要自行创建用户、复制文件并确保 `mdfs` 用户能够读取共享目录。

## 开发命令

| 命令 | 用途 |
|---|---|
| `make dev-api` | 启动 Go API |
| `make dev-web` | 启动 Vite 开发服务器 |
| `make test` | 运行 Go 测试和 Vue 类型检查 |
| `make prepare-web` | 构建并复制 Vue 静态资源 |
| `make build` | 使用 GoReleaser 生成本地 snapshot 发布包 |
| `make clean` | 删除构建产物 |

## 项目结构

```text
backend/   Go 服务、API、文件访问和嵌入式前端
frontend/  Vue 3 + TypeScript + Vite 页面
configs/   TOML 示例配置
deploy/    systemd 部署模板
scripts/   前端静态资源准备脚本
dist/      GoReleaser 本地发布产物
```

当前版本不包含上传、用户权限、WebDAV、HTTPS或自动安装脚本。
