---
name: gcli
description: Operate and troubleshoot the gcli Gmail CLI on cloud servers. Use this skill whenever users ask for Gmail OAuth login, SSH tunnel authorization, refresh token setup, mail list/search/get execution, full email content extraction, or debugging 403/redirect_uri/token/network errors in this repository.
---

# gcli - Gmail CLI Agent 操作手册

用于指导 Agent 在本仓库中稳定操作 `gcli`：鉴权、读取邮件、排障、交付检查。

## 触发条件（意图识别）

当用户出现以下意图时，立即使用本技能：

- 要求“登录 Gmail / OAuth 鉴权 / 获取 refresh token”
- 提到“云服务器 + 本地浏览器授权 / SSH 隧道”
- 要求执行 `gcli mail list` / `search` / `get`
- 要求“读取某封邮件完整正文或 raw MIME”
- 报错包含 `403 access_denied`、`redirect_uri_mismatch`、`AUTH_*`、DNS 失败
- 要求编写或核对该 CLI 的使用手册/SOP

## 执行前检查

按顺序执行以下检查：

1. 确认仓库目录是 `/root/go/src/gcli`
2. 确认二进制可用：`./bin/gcli version`
3. 确认环境变量文件（默认 `/tmp/gcli.env`）
4. 不输出完整 `client_secret` 与 `refresh_token`

## 标准流程

### A. 鉴权流程（云服务器）

1. 检查 Google Cloud 配置
- 已启用 Gmail API
- OAuth 同意屏幕已配置
- 测试状态下已添加测试用户
- OAuth 客户端已配置 redirect URI：`http://127.0.0.1:8787/callback`

2. 在本地电脑建立隧道
```bash
ssh -N -L 8787:127.0.0.1:8787 <user>@<server>
```

3. 在云服务器执行登录
```bash
cd /root/go/src/gcli
./bin/gcli auth login \
  --client-id "..." \
  --client-secret "..." \
  --redirect-uri "http://127.0.0.1:8787/callback" \
  --auth-timeout 10m \
  --print-env
```

4. 写入运行环境并加载
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

### B. 读信流程

1. 低配额列表（默认）
```bash
./bin/gcli mail list --label INBOX --limit 20
```
说明：默认去 N+1，`from/subject` 可能为空。

2. 富字段列表（需要额外 API 调用）
```bash
./bin/gcli mail list --label INBOX --limit 20 --hydrate
```

3. 搜索
```bash
./bin/gcli mail search --q "newer_than:7d" --limit 20
./bin/gcli mail search --q "from:alerts@example.com" --limit 20 --hydrate
# 位置参数写法（等价于 --q）
./bin/gcli mail search "in:inbox is:unread from:boss@company.com" --max 50
# 分页别名（等价于 --page-token）
./bin/gcli mail search "has:attachment filename:pdf" --max 20 --page "<next_page_token>"
```
说明：
- `--max` 是 `--limit` 别名，`--page` 是 `--page-token` 别名。
- 返回字段包含：`id`、`thread_id`、`date`、`from`、`subject`、`label_ids`。
- `from/subject/date` 的完整精度依赖 `--hydrate`；未开启时可能为空或回退为 `internal_date`。

常见 Gmail `q` 语法：
- `in:inbox` / `in:sent` / `in:drafts` / `in:trash` / `in:spam`
- `is:unread` / `is:starred` / `is:important`
- `from:sender@example.com` / `to:recipient@example.com`
- `subject:keyword`
- `has:attachment` / `filename:pdf`
- `after:2024/01/01` / `before:2024/12/31`
- `label:Work` / `label:UNREAD`

4. 单封邮件
```bash
./bin/gcli mail get --id "<message_id>" --format metadata
./bin/gcli mail get --id "<message_id>" --format full
./bin/gcli mail get --id "<message_id>" --format raw
```
说明：`full` 返回 `body_text/body_html`；`raw` 返回 `raw_mime`，体积可能很大。

### C. 交付检查流程

在开发/提交前执行：

```bash
make fmt vet lint test release-check build
```

## 故障排查（先看错误码）

1. `403 access_denied`（应用测试中）
- 在 OAuth 同意屏幕添加测试用户
- 等 1-5 分钟后重试

2. `redirect_uri_mismatch`
- 确认 OAuth 客户端与命令行 `--redirect-uri` 完全一致

3. `AUTH_NO_REFRESH_TOKEN`
- 重新走授权同意流程，确保拿到 `refresh_token`

4. `AUTH_SCOPE_INSUFFICIENT`
- 确认 scope 至少包含：`https://www.googleapis.com/auth/gmail.readonly`

5. `INTERNAL` 且 `details.operation=users.messages.*`
- 优先看 `details.http_status` 与 `details.google_reason`

6. `Could not resolve host: oauth2.googleapis.com`
- 服务器 DNS/出网问题，不是业务代码问题

## 输出契约

成功：
```json
{"version":"v1","data":{},"error":null}
```

失败：
```json
{"version":"v1","data":null,"error":{"code":"...","message":"...","retryable":false,"details":{"operation":"...","http_status":"...","google_reason":"..."}}}
```

## 安全规则

- 不输出完整密钥与令牌
- 凭据泄露时，建议立刻轮换 OAuth 客户端
- 默认最小权限：`gmail.readonly`

## 示例触发语句

- “请使用 gcli 技能，帮我在云服务器完成 Gmail 鉴权并验证 `mail list --hydrate`。”
- “请按 gcli 操作手册排查 `redirect_uri_mismatch`。”
