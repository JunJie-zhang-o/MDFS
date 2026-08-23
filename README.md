# MDFS

MDFS 是一个使用 Go、Vue 3 和 Vite 实现的轻量文件服务器。它提供 CHFS 3.1 风格的网页文件管理、R/W/D 路径权限、多共享目录、WebDAV、TLS、IP 访问控制和操作日志，并可将前端嵌入单个可执行文件。

## 功能

- 目录浏览、本页/全局搜索、排序和哈希路径导航
- 文件预览、Range 下载、二维码、目录 ZIP 下载
- 文件/目录上传、拖拽与上传进度
- 新建目录/文本、在线文本编辑、重命名和删除
- 简体中文、繁体中文和英文界面
- 匿名及多用户 R/W/D 权限，支持 `*` 和 `**` 路径规则
- 单目录或多目录虚拟根
- 与网页权限一致的 `/webdav` 服务
- 可选 TLS、IP allow/deny、可信代理和 JSONL 操作日志

## 开发运行

要求 Go 1.25+、Node.js 22+、npm 10+。

```bash
cd frontend && npm install
make dev-api
make dev-web
```

访问 `http://localhost:5173`。Vite 将 `/api` 和 `/webdav` 代理到 `http://127.0.0.1:8080`。

`make dev-api` 默认读取 `configs/mdfs.dev.toml`，共享项目的 `shared/` 测试目录。开发配置允许匿名 RWD，另提供 `admin/admin` 登录账号；请勿将该配置用于生产。

## 配置

程序只接受 `--config`：

```bash
cd backend
go run ./cmd/mdfs --config ../configs/mdfs.dev.toml
```

核心配置示例：

```toml
[server]
listen = "0.0.0.0:8080"
public_url = ""
session_timeout = "60m"

[[shares]]
name = "public"
path = "/srv/public"

[anonymous]
permissions = "R"

[[users]]
name = "admin"
password = "replace-this-password"
permissions = "RWD"

[[users.rules]]
pattern = "/private/**"
permissions = ""

[webdav]
enabled = true
prefix = "/webdav"
```

`R` 表示浏览/搜索/下载，`W` 表示上传/新建/编辑/重命名，`D` 表示删除。最具体的路径规则覆盖用户默认权限。`*` 匹配单个目录段，`**` 匹配任意深度。

配置中的密码为明文。生产环境必须限制配置文件权限：

```bash
chmod 600 /etc/mdfs/mdfs.toml
```

一个旧式 `[share] path = "..."` 仍可作为单共享目录配置；新增部署建议使用 `[[shares]]`。多个 share 时，网页和 WebDAV 根目录为只读虚拟根。

### TLS

同时配置证书和私钥后，服务直接启用 HTTPS：

```toml
[server.tls]
cert_file = "/etc/mdfs/tls/cert.pem"
key_file = "/etc/mdfs/tls/key.pem"
```

### IP 控制与日志

```toml
[access]
allow = ["192.168.0.0/16", "10.0.0.5-10.0.0.20"]
deny = ["192.168.1.100"]
trusted_proxies = ["127.0.0.1"]

[logging]
directory = "/var/log/mdfs"
```

deny 优先于 allow。只有可信代理地址才会触发 `X-Forwarded-For` 解析。日志目录非空时按日期生成 JSONL 文件，且不会记录密码或文件内容。

## REST API

主要接口位于 `/api/v1`：

- `GET /meta`、`GET|POST|DELETE /session`
- `GET /files`、`GET /search`、`GET /content`、`GET /archive`
- `POST /uploads`、`POST /directories`
- `POST|PUT /text-files`
- `PATCH|DELETE /files`

错误响应格式为 `{"error":{"code":"...","message":"..."}}`。

## 测试与构建

```bash
make test
make prepare-web
make build
```

`make test` 运行 Go 测试、Vitest、Vue 类型检查和前端生产构建。`make prepare-web` 将 Vite 构建复制到 Go 嵌入目录。`make build` 使用 GoReleaser 生成 Linux amd64/arm64 snapshot 包、配置和 systemd unit。

发布包中的 systemd 默认使用：

- 程序 `/opt/mdfs/bin/mdfs`
- 配置 `/etc/mdfs/mdfs.toml`
- 用户/组 `mdfs`
- 共享目录 `/srv/mdfs`

生产部署前请创建低权限运行用户，并确保它只拥有共享目录所需的文件系统权限。
