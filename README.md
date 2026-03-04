# gcli

`gcli` 是一个基于 `google-api-go-client` 的 Gmail 只读 CLI，适合在云服务器上运行、在本地浏览器完成授权。

## 功能概览

- `auth login`：OAuth Authorization Code + PKCE，获取 `refresh_token`
- `mail list`：分页列出邮件（默认轻量模式，低配额消耗）
- `mail search`：按 Gmail 原生 `q` 语法搜索（默认轻量模式）
- `mail get`：读取单封邮件（`metadata|full|minimal|raw`）
- 默认 JSON 输出，适合自动化脚本

## 1 分钟快速开始

```bash
cd /root/go/src/gcli
make build
./bin/gcli version
```

---

## 保姆级教程（中文，云服务器场景）

下面按“从 0 到可用”一步一步来。

### 第 0 步：你需要准备什么

- 一个 Google 账号（个人 Gmail 或 Workspace 账号）
- 一台云服务器（已能运行本项目）
- 一台本地电脑（有浏览器，可 SSH 到云服务器）

### 第 1 步：创建 Google Cloud 项目

1. 打开 `https://console.cloud.google.com/`
2. 顶部选择项目 -> `新建项目`
3. 项目名随意（例如 `openclaw`）
4. 创建完成后，确认当前选中的就是这个新项目

### 第 2 步：启用 Gmail API

1. 左上角菜单 `≡` -> `API 和服务` -> `已启用的 API 和服务`
2. 点 `启用 API 和服务`
3. 搜索 `Gmail API`
4. 进入后点击 `启用`

### 第 3 步：配置 OAuth 同意屏幕

1. 左侧 `API 和服务` -> `OAuth 同意屏幕`
2. 用户类型建议：
- 个人 Gmail：选 `外部`
- 企业 Workspace：按你组织策略选 `内部/外部`
3. 按页面要求填写应用名称、支持邮箱、开发者邮箱
4. Scopes 添加：
- `https://www.googleapis.com/auth/gmail.readonly`
5. 如果应用状态是“测试中”：
- 到 `测试用户` 区域
- 添加你要登录的邮箱（例如 `zreallyli@gmail.com`）

### 第 4 步：创建 OAuth 客户端（重点）

`gcli auth login` 的回调模式推荐使用 **Web 应用** 客户端。

1. 左侧 `API 和服务` -> `凭据`
2. 点击 `创建凭据` -> `OAuth 客户端 ID`
3. 应用类型选择：`Web 应用`
4. 在“已获授权的重定向 URI”里新增：
- `http://127.0.0.1:8787/callback`
5. 创建后复制：
- `Client ID`
- `Client Secret`

### 关于 “OAuth Desktop”

你提到想建 Desktop。可以建，但本项目当前的云端回调流程以 `Web 应用 + 固定 redirect URI` 最稳。  
如果你已经建了 Desktop 且出现 `invalid_client` / `redirect_uri_mismatch` / `access_denied`，建议直接改成上面 Web 应用方案。

### 第 5 步：本地建立 SSH 隧道（非常关键）

在你本地电脑执行（不是云服务器）：

```bash
ssh -N -L 8787:127.0.0.1:8787 root@xxx
```

说明：
- `-N`：只建隧道，不执行远端命令
- `-L 8787:127.0.0.1:8787`：把本地 8787 转发到云服务器 127.0.0.1:8787
- 这个窗口要保持打开，直到授权完成

### 第 6 步：在云服务器执行登录命令

在云服务器执行：

```bash
cd /root/go/src/gcli
./bin/gcli auth login \
  --client-id "你的_client_id" \
  --client-secret "你的_client_secret" \
  --redirect-uri "http://127.0.0.1:8787/callback" \
  --auth-timeout 10m \
  --print-env
```

然后：
1. 复制终端输出的授权 URL
2. 在本地浏览器打开
3. 登录 Google 并同意权限
4. 成功后，CLI 会输出 JSON，里面有 `refresh_token`

### 第 7 步：设置环境变量（推荐用 env 文件）

在云服务器写入 `/tmp/gcli.env`：

```bash
cat >/tmp/gcli.env <<'EOF_ENV'
GCLI_GMAIL_CLIENT_ID=你的_client_id
GCLI_GMAIL_CLIENT_SECRET=你的_client_secret
GCLI_GMAIL_REFRESH_TOKEN=你的_refresh_token
EOF_ENV
```

加载变量：

```bash
set -a
source /tmp/gcli.env
set +a
```

### 第 8 步：验证 Gmail 访问

```bash
cd /root/go/src/gcli

# 列表
./bin/gcli mail list --label INBOX --limit 5

# 如需 from/subject/date（会触发额外 API 调用）
./bin/gcli mail list --label INBOX --limit 5 --hydrate

# 搜索
./bin/gcli mail search --q "newer_than:7d" --limit 5

# 读取单封元信息
./bin/gcli mail get --id "邮件ID" --format metadata

# 读取正文（返回 body_text / body_html）
./bin/gcli mail get --id "邮件ID" --format full

# 读取原始 MIME
./bin/gcli mail get --id "邮件ID" --format raw
```

### 第 9 步：常见报错与处理

1. `403 access_denied` + “应用正在测试中”
- 去 OAuth 同意屏幕把当前邮箱加入“测试用户”
- 等 1-5 分钟再试

2. `redirect_uri_mismatch`
- 检查 OAuth 客户端里是否配置了：
  `http://127.0.0.1:8787/callback`
- 检查命令里的 `--redirect-uri` 是否完全一致

3. `AUTH_NO_REFRESH_TOKEN`
- 说明本次没拿到 refresh token
- 重新授权时保持 `prompt=consent`（本 CLI 默认已设置）
- 必要时在 Google 账号“第三方访问”里移除旧授权后重试

4. `Could not resolve host: oauth2.googleapis.com`
- 服务器 DNS/网络受限，不是代码逻辑问题
- 先排查服务器出网策略和 DNS 配置

5. 泄露了 `client_secret`
- 立即在 Google Cloud 删除旧 OAuth 客户端并重建

---

## 运行时环境变量

必填：

- `GCLI_GMAIL_CLIENT_ID`
- `GCLI_GMAIL_CLIENT_SECRET`
- `GCLI_GMAIL_REFRESH_TOKEN`

可选：

- `GCLI_GMAIL_TOKEN_URL`（默认 `https://oauth2.googleapis.com/token`）
- `GCLI_GMAIL_AUTH_URL`（默认 `https://accounts.google.com/o/oauth2/v2/auth`）
- `GCLI_GMAIL_API_ENDPOINT`（仅 mock/testing）

## 输出契约

成功：

```json
{"version":"v1","data":{},"error":null}
```

失败：

```json
{"version":"v1","data":null,"error":{"code":"...","message":"...","retryable":false}}
```

失败时可能包含可选字段：

```json
{"version":"v1","data":null,"error":{"code":"...","message":"...","retryable":false,"details":{"operation":"users.messages.get","http_status":"403","google_reason":"insufficientPermissions"}}}
```

## 质量门禁

```bash
make fmt
make vet
make lint
make test
make release-check
```

## 发布产物

Tag push 会触发 release，产物包含：

- `gcli-linux-amd64`
- `gcli-linux-arm64`
- `gcli-darwin-amd64`
- `gcli-darwin-arm64`
- `SHA256SUMS`
