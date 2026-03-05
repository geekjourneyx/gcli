# gcli

[![CI](https://github.com/geekjourneyx/gcli/actions/workflows/ci.yml/badge.svg)](https://github.com/geekjourneyx/gcli/actions/workflows/ci.yml)
[![Release](https://github.com/geekjourneyx/gcli/actions/workflows/release.yml/badge.svg)](https://github.com/geekjourneyx/gcli/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go)](https://go.dev/)
[![Gmail Readonly](https://img.shields.io/badge/gmail-readonly-34A853?logo=gmail&logoColor=white)](https://developers.google.com/gmail/api)

`gcli` 是一个 **Gmail 只读 CLI**，专门解决“云服务器执行 + 本地浏览器授权”的落地问题。  
定位很明确：**不做发信、不做后台管理，只把查邮件做成安全、稳定、可自动化的能力。**

## 为什么做这个项目

实际生产场景里，查邮件常见卡点不是 API 本身，而是：

- 服务器无浏览器，OAuth 很难走通。
- `redirect_uri_mismatch`、`access_denied` 反复踩坑。
- 脚本和 Agent 需要稳定 JSON，不希望靠文本解析。
- 只想查邮件，不想引入大而全的管理工具链。

`gcli` 的答案：

- `auth login` + SSH 隧道打通云端授权闭环。
- Gmail 原生 `q` 查询语法，覆盖绝大多数检索场景。
- 默认 JSON 输出 + 结构化错误码，自动化友好。
- 默认最小权限 `gmail.readonly`，风险可控。

## 与官方 `googleworkspace/cli` 的关系

官方仓库：`https://github.com/googleworkspace/cli`

`gcli` 不是替代它，而是聚焦一个更窄更深的子场景：**Gmail 只读检索自动化**。

| 维度 | googleworkspace/cli | gcli |
|---|---|---|
| 目标范围 | Workspace 多产品/多管理能力 | Gmail 只读检索与读取 |
| 云服务器授权落地 | 需要自行适配 | 内置云服务器实践（本地浏览器 + SSH 隧道） |
| 输出契约 | 因命令而异 | 默认 JSON，结构化错误模型 |
| 权限策略 | 覆盖面广 | 默认 `gmail.readonly` |
| 学习成本 | 功能全面但面广 | 聚焦查邮件，路径更短 |

## 核心能力

- `gcli auth login`
  - OAuth Authorization Code + PKCE
  - 支持 `--auth-timeout`
  - 可输出可写入环境变量的数据
- `gcli mail list`
  - 收件箱/标签分页读取
  - `--hydrate` 控制 rich headers 拉取
- `gcli mail search`
  - 支持 Gmail 原生 `q`
  - 支持位置参数或 `--q`
  - 支持 `--max`/`--page` 别名
- `gcli mail get`
  - `metadata|full|minimal|raw`
  - 可读正文或 raw MIME

## 下载与安装（Release）

推荐安装：

```bash
curl -fsSL https://raw.githubusercontent.com/geekjourneyx/gcli/main/scripts/install.sh | bash
```

验证：

```bash
gcli version
```

手动下载：

- Release 页面：`https://github.com/geekjourneyx/gcli/releases`
- 产物：
  - `gcli-linux-amd64`
  - `gcli-linux-arm64`
  - `gcli-darwin-amd64`
  - `gcli-darwin-arm64`
  - `SHA256SUMS`

## 快速开始

```bash
# 1) 首次登录
gcli auth login \
  --client-id "<client_id>" \
  --client-secret "<client_secret>" \
  --redirect-uri "http://127.0.0.1:8787/callback" \
  --auth-timeout 10m \
  --print-env

# 2) 搜索未读
gcli mail search "in:inbox is:unread" --max 20

# 3) 读取正文
gcli mail get --id "<message_id>" --format full
```

## 保姆级教程（云服务器场景）

这是 `gcli` 的核心优势：把最容易失败的授权路径讲清楚并可复现。

### 第 0 步：准备

- Google 账号（Gmail 或 Workspace）
- 云服务器（运行 `gcli`）
- 本地电脑（有浏览器，可 SSH 到云服务器）

### 第 1 步：创建 Google Cloud 项目并启用 Gmail API

1. 打开 `https://console.cloud.google.com/` 新建项目。
2. 在 `API 和服务` 中启用 `Gmail API`。

### 第 2 步：配置 OAuth 同意屏幕

1. 进入 `API 和服务 -> OAuth 同意屏幕`。
2. 添加 scope：`https://www.googleapis.com/auth/gmail.readonly`。
3. 测试状态下，把登录邮箱加入“测试用户”。

### 第 3 步：创建 OAuth 客户端（推荐 Web 应用）

1. 进入 `API 和服务 -> 凭据 -> 创建凭据 -> OAuth 客户端 ID`。
2. 类型选 `Web 应用`。
3. 添加重定向 URI：`http://127.0.0.1:8787/callback`。
4. 记录 `Client ID` 和 `Client Secret`。

### 第 4 步：本地建立 SSH 隧道

在本地电脑执行：

```bash
ssh -N -L 8787:127.0.0.1:8787 root@<your-server>
```

保持该终端窗口打开，直到授权结束。

### 第 5 步：云服务器执行登录

```bash
gcli auth login \
  --client-id "<client_id>" \
  --client-secret "<client_secret>" \
  --redirect-uri "http://127.0.0.1:8787/callback" \
  --auth-timeout 10m \
  --print-env
```

然后：

1. 复制授权 URL 到本地浏览器打开。
2. 登录并同意权限。
3. 成功后获取 `refresh_token`。

### 第 6 步：持久化环境变量（推荐）

`gcli` 启动时自动尝试读取 `~/.config/gcli/env`（系统环境变量优先）。

```bash
mkdir -p ~/.config/gcli
cat > ~/.config/gcli/env <<'EOF_ENV'
GCLI_GMAIL_CLIENT_ID='<client_id>'
GCLI_GMAIL_CLIENT_SECRET='<client_secret>'
GCLI_GMAIL_REFRESH_TOKEN='<refresh_token>'
EOF_ENV
chmod 600 ~/.config/gcli/env
```

验证：

```bash
gcli mail list --label INBOX --limit 5
```

## 搜索语法速查（Gmail q）

- `in:inbox` / `in:sent` / `in:drafts` / `in:trash` / `in:spam`
- `is:unread` / `is:starred` / `is:important`
- `from:sender@example.com` / `to:recipient@example.com`
- `subject:keyword`
- `has:attachment` / `filename:pdf`
- `after:2024/01/01` / `before:2024/12/31`
- `label:Work` / `label:UNREAD`

示例：

```bash
# 位置参数
gcli mail search "from:boss@example.com is:unread" --max 50

# --q 参数
gcli mail search --q "has:attachment filename:pdf" --limit 20

# 翻页
gcli mail search "subject:weekly report" --max 20 --page "<next_page_token>"

# 需要稳定 from/subject/date 时
gcli mail search "in:inbox" --max 20 --hydrate
```

## 常见错误与处理

- `AUTH_MISSING_CREDENTIALS`
  - 缺少 `GCLI_GMAIL_CLIENT_ID/SECRET/REFRESH_TOKEN`
- `AUTH_SCOPE_INSUFFICIENT`
  - scope 不是 `gmail.readonly`
- `AUTH_*`
  - 重新执行 `gcli auth login ...` 刷新授权
- `redirect_uri_mismatch`
  - Google Cloud 与命令行 `--redirect-uri` 不一致
- `MAIL_NOT_FOUND`
  - `message_id` 无效或邮件已删除

## 输出契约（自动化友好）

成功：

```json
{"version":"v1","data":{},"error":null}
```

失败：

```json
{"version":"v1","data":null,"error":{"code":"...","message":"...","retryable":false}}
```

失败时可包含 `details` 字段（如 `operation/http_status/google_reason`）用于程序处理。

## 安全与可靠性

- 默认最小权限：`gmail.readonly`
- 不输出完整密钥与令牌
- 凭据泄露时应立即轮换 OAuth 客户端
- 生产建议使用专用运行账号，不长期使用 root

## Agent 友好技能

- 技能文件：`skills/gcli/SKILL.md`
- 安装技能：

```bash
npx skills add https://github.com/geekjourneyx/gcli --skill gcli
```

## 开发与质量门禁

```bash
make fmt
make vet
make lint
make test
make release-check
make build
```
