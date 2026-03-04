---
name: gcli
description: Operate and troubleshoot the gcli Gmail CLI in cloud-server scenarios. Use this skill whenever users mention Gmail automation, OAuth login on servers, SSH tunnel auth, mail list/search/get, refresh token setup, or ask for step-by-step operational help with this repository.
---

# gcli skill

面向本仓库 `gcli` 的 Agent 操作手册。目标是让 Agent 在“云服务器运行 CLI、本地浏览器授权”的场景下，稳定完成鉴权、读信、排障、验证与交付。

## 适用场景

当用户出现以下任一需求时，优先使用本技能：

- 让你帮忙“登录 Gmail / 做 OAuth 鉴权 / 拿 refresh token”
- 让你在服务器上跑 `gcli mail list/search/get`
- 让你排查 `403 access_denied` / `redirect_uri_mismatch` / `AUTH_*` 错误
- 让你写保姆级教程、SOP、值班操作手册
- 让你做 release 前验证（`make fmt vet lint test release-check build`）

## 核心原则

1. 先确认上下文，再执行命令。
- 确认目录为仓库根：`/root/go/src/gcli`
- 确认二进制存在：`./bin/gcli`
- 确认 env 文件位置（默认 `/tmp/gcli.env`）

2. 生产安全优先。
- 不在日志中回显 `client_secret` / `refresh_token` 全文
- 若凭据泄露，立即建议用户在 Google Cloud 重建 OAuth 客户端
- 仅申请最小权限：`gmail.readonly`

3. 优先可观测输出。
- 默认 JSON 输出，错误读取 `error.code` 和可选 `error.details`
- 出错先判定网络、凭据、OAuth 配置、scope，再判断代码问题

## 标准执行流程

### A. 鉴权准备

1. 确认 Google Cloud 基础配置
- 已启用 Gmail API
- OAuth 同意屏幕可用
- 测试中应用已把登录邮箱加到“测试用户”
- OAuth 客户端已配置 redirect URI：`http://127.0.0.1:8787/callback`

2. 建议使用云服务器授权模式
- 本地建隧道：
```bash
ssh -N -L 8787:127.0.0.1:8787 <user>@<server>
```
- 服务器执行：
```bash
./bin/gcli auth login \
  --client-id "..." \
  --client-secret "..." \
  --redirect-uri "http://127.0.0.1:8787/callback" \
  --auth-timeout 10m \
  --print-env
```

3. 写入运行环境
```bash
cat >/tmp/gcli.env <<'EOF_ENV'
GCLI_GMAIL_CLIENT_ID=...
GCLI_GMAIL_CLIENT_SECRET=...
GCLI_GMAIL_REFRESH_TOKEN=...
EOF_ENV

set -a
source /tmp/gcli.env
set +a
```

### B. 读取邮件

1. 轻量列表（低配额）
```bash
./bin/gcli mail list --label INBOX --limit 20
```
说明：默认不逐条 hydrate，`from/subject` 可能为空。

2. 富信息列表（会额外调用 API）
```bash
./bin/gcli mail list --label INBOX --limit 20 --hydrate
```

3. 搜索
```bash
./bin/gcli mail search --q "newer_than:7d" --limit 20
./bin/gcli mail search --q "from:alerts@example.com" --limit 20 --hydrate
```

4. 单封邮件
```bash
./bin/gcli mail get --id "<message_id>" --format metadata
./bin/gcli mail get --id "<message_id>" --format full
./bin/gcli mail get --id "<message_id>" --format raw
```

### C. 质量门禁（开发/交付）

```bash
make fmt vet lint test release-check build
```

## 错误排查手册

1. `403 access_denied` 且提示“应用正在测试中”
- 在 OAuth 同意屏幕的“测试用户”添加当前邮箱
- 等 1-5 分钟后重试

2. `redirect_uri_mismatch`
- 检查 Google Cloud OAuth 客户端是否配置：
  `http://127.0.0.1:8787/callback`
- 检查命令 `--redirect-uri` 是否完全一致

3. `AUTH_NO_REFRESH_TOKEN`
- 说明 token 响应没有 refresh token
- 重新授权并确保触发 consent

4. `AUTH_SCOPE_INSUFFICIENT`
- 检查 scope 是否至少为：
  `https://www.googleapis.com/auth/gmail.readonly`

5. `INTERNAL` 且 `details.operation=users.messages.*`
- 先看 `details.http_status` 与 `details.google_reason`
- 再区分是网络问题、配额问题、权限问题、输入参数问题

6. `Could not resolve host: oauth2.googleapis.com`
- 服务器 DNS/出网策略问题，不是业务代码问题

## 输出契约速记

成功：
```json
{"version":"v1","data":{},"error":null}
```

失败（含可选 details）：
```json
{"version":"v1","data":null,"error":{"code":"...","message":"...","retryable":false,"details":{"operation":"...","http_status":"...","google_reason":"..."}}}
```

## Agent 执行建议

1. 用户让你“帮我测一下”，优先执行：
- `source /tmp/gcli.env`
- `./bin/gcli mail list --label INBOX --limit 1`

2. 用户要完整字段，明确提醒：
- `list/search` 需加 `--hydrate`
- `get --format full/raw` 返回体积可能很大

3. 用户要上线发布，最小清单：
- 门禁全绿
- `CHANGELOG.md` 同步
- `scripts/install.sh`/`Makefile`/`CHANGELOG.md` 版本一致

