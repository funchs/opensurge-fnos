# fnOS 面板密码登录

## 背景

局域网模式下 `GET /enter` 只校验 Host 白名单就签发 `opensurge_session`。
FN Connect 把 GUI 暴露到 `https://opensurge.<fnid>.fnos.net/` 之后，这等于公网免密进入控制面。

仓库里从未实现过 GUI 密码页。现有鉴权（Bearer `control-token`、HttpOnly session、Host 白名单、写操作 Origin、30 秒 bootstrap）全部保留。本 spec 只给 **fnOS 局域网模式** 的浏览器入口补一层密码。

## 目标

- 首次打开面板强制设置密码（只要密码，不要用户名）。
- 之后每次浏览器进入必须登录才能拿到 session。
- 忘记密码：删除 `{store}/gui-password`，下次打开回到设置页。
- 已登录用户可改密、退出。
- macOS 桌面（非局域网模式）行为不变。

## 非目标

- 多用户、飞牛账号 SSO、HTTP Basic、反代认证。
- 面板内「忘记密码」。
- 安装向导 / 环境变量预置密码。
- 改网关数据面、DHCP、TUN。
- 本任务不 bump 版本、不推 GHCR、不把无关的 0.1.2 工作区改动打进同一提交。

## 适用范围

仅当 `lanMode()` 为真（已配置 BaseURL / `allowedHosts`，即 fnOS Docker/fpk）时启用面板密码。

`lanMode()` 为假时：不读密码文件、不提供 setup/login 业务语义、`GET /enter` 仍 401、桌面 bootstrap 路径不变。

## 架构

三个边界清晰的单元：

1. **密码存储**（`Store`）  
   读写 `{store}/gui-password`。调用方只看到「是否已配置 / 创建 / 替换 / 校验」，不接触 bcrypt 细节以外的哈希格式。

2. **会话入口**（`controlapi` HTTP）  
   未登录：`/api/v1/session/status|setup|login`。已登录：`logout`、改密。  
   `GET /enter` 改为跳转，不再无密码发 cookie。  
   `auth()` 在局域网模式下：无密码文件则浏览器 session 无效；有密码文件才认 cookie。Bearer 与 `/bootstrap` 仍绕过面板密码（运维）。

3. **Web GUI**  
   `/setup`、`/login` 两页；侧栏快捷设置里改密和退出。先调 status 再决定渲染，401 不再自动 `/enter` 领 cookie。

数据流：浏览器 → Host 白名单 → status/setup/login → bcrypt 校验 → 现有 `issueWebSession` → 之后业务 API 仍走现有 `auth()`。

## 存储

路径：`{StoreDir}/gui-password`  
权限：`0600`（与 `control-token` 相同，经现有 `writeAtomic`）

```json
{
  "schema_version": 1,
  "algo": "bcrypt",
  "hash": "$2a$10$...",
  "updated_at": "2026-10-17T00:00:00Z"
}
```

- `algo` 必须是 `bcrypt`，其它值视为损坏，登录失败（500 `password_unusable`，不提示「密码错误」）。
- 哈希用 `SHA-256(utf8(password))` 再 `bcrypt.GenerateFromPassword`（`DefaultCost`）。先哈希是为了避开 bcrypt 的 72 字节上限，仍允许 8–128 个 Unicode code point。
- 明文密码长度 8–128 个 Unicode code point；不 trim。
- 每次 setup / login / `auth()` / 改密都读文件；删除文件无需重启即可变成「未设置」。
- 首次创建在 `Store` 锁内：若文件已存在则失败（`errGUIPasswordConfigured`），避免双窗口同时 setup。
- 密码、哈希不得写入 log、错误正文、overview、诊断。

`Store` 方法：

| 方法 | 行为 |
| --- | --- |
| `GUIPassword()` | 文件不存在 → `os.ErrNotExist`；存在则解析 JSON |
| `CreateGUIPassword(password)` | 校验长度、哈希、独占写入；已存在则 `errGUIPasswordConfigured` |
| `ReplaceGUIPassword(password)` | 校验长度、哈希、覆盖写入（改密） |
| `PasswordMatches(password)` | 读文件 + `bcrypt.CompareHashAndPassword`；文件不存在则 false + `os.ErrNotExist` |

## HTTP API

全部仍经 `securityHeaders`（Host 白名单）。setup / login 的 POST 必须 `originAllowed`；缺少 Origin 或 Origin 不在白名单 → 403 `origin_rejected`。

`GET /api/v1/session/status` 不要求 Origin、不要求 session。

### `GET /api/v1/session/status`

```json
{
  "schema_version": 1,
  "gui_password_required": true,
  "password_configured": false,
  "authenticated": false
}
```

- `gui_password_required`：`lanMode()`。
- `password_configured`：密码文件存在（`os.Stat` 成功）。损坏 JSON / 非 bcrypt `algo` 仍算已配置，避免把用户送去 `/setup` 再 409；登录返回 500 `password_unusable`。重置方式仍是删除该文件。
- `authenticated`：当前请求带有未过期 `opensurge_session`。若 `gui_password_required && !password_configured`，即使 cookie 仍在内存表里也返回 `authenticated: false`（文件已删）。

### `POST /api/v1/session/setup`

Body：`{ "password": "..." }`

仅 `lanMode()`。否则 401 `authentication_required`。

| 条件 | 结果 |
| --- | --- |
| 已有密码文件 | 409 `password_already_configured` |
| 长度非法 | 400 `invalid_request` |
| 成功 | 写文件、签发 session、204 |

### `POST /api/v1/session/login`

Body：`{ "password": "..." }`

仅 `lanMode()`。

| 条件 | 结果 |
| --- | --- |
| 未设密码 | 409 `password_not_configured` |
| 密码错误 | 401 `invalid_password`，计入限速 |
| 文件损坏 / 无法校验 | 500 `password_unusable`，不计入限速 |
| 限速触发 | 429 `too_many_attempts` |
| 成功 | 签发 session、204，清该 IP 失败计数 |

### `POST /api/v1/session/logout`

不强制已登录。删除内存 session（若有）、`Set-Cookie` 过期、204。

### `PUT /api/v1/session/password`

经 `auth()`。Body：`{ "current": "...", "next": "..." }`

仅 `lanMode()` 且已配置密码。

| 条件 | 结果 |
| --- | --- |
| 非 lanMode | 401 `authentication_required` |
| 未设密码 | 409 `password_not_configured` |
| current 不对 | 401 `invalid_password`（前端不得因此派发 `authenticationRequiredEvent`） |
| 文件损坏 / 无法校验 | 500 `password_unusable` |
| next 长度非法 | 400 `invalid_request` |
| 成功 | 替换哈希、清空**全部**内存 session、给当前响应重发新 session、204 |

### `GET /enter`

- 非 `lanMode()`：保持 401。
- `lanMode()` 且已有有效 session 且密码已配置：保持现有行为，刷新 cookie 并 302 `/{path}`（默认 dashboard）。
- `lanMode()` 且未设密码：302 `/setup`，**不**发 cookie。
- `lanMode()` 且已设密码但无有效 session：302 `/login`，**不**发 cookie。

查询参数 `path` 仍经 `allowedWebPath`；未登录跳转时忽略 `path`，登录成功后前端进总览即可（setup/login POST 不接收 next path，YAGNI）。

## `auth()` 变更

现有 Bearer 与 session 逻辑保留，局域网模式增加：

1. 若 `gui_password_required` 且密码文件不存在：忽略浏览器 session（视为未登录）。Bearer 仍可通过。
2. 否则与现在相同：Bearer 或有效 session。

`POST /api/v1/session/bootstrap` 与 `GET /bootstrap` 不检查面板密码。

## 限速

- 只作用于 `POST /api/v1/session/login` 的 `invalid_password`。
- 键：`RemoteAddr` 的主机部分。Host 为 `opensurge.<fnid>.fnos.net` 且存在 `X-Forwarded-For` 时，取该头最左边的 IP（FN Connect 反代）。
- 窗口：15 分钟内 5 次失败 → 429。成功登录清零该键。
- 内存 map，进程重启即清空。不持久化。

## 前端

`webui` 已将未知路径回落到 `index.html`，`/setup`、`/login` 无需新静态文件。

### 启动分流

`App` 在拉 `overview` 之前先 `GET /api/v1/session/status`（桌面版可跳过，`isDesktop()` 走原路径）。

| status | 渲染 |
| --- | --- |
| `gui_password_required && !password_configured` | `SetupPage`，不调 overview |
| `gui_password_required && !authenticated` | `LoginPage` |
| 其它 | 现有控制台 |

已登录访问 `/login` 或 `/setup`：`history.replaceState` 到 `/dashboard`。

**删除** `redirectToEnterOnce()` 对局域网 GUI 的「401 → `/enter` 自动领 cookie」。fnOS 下 401 改为 `location.replace('/login')`（若未配置密码则 `/setup`）。桌面版 401 仍显示现有「安全连接已过期」页。

`api.ts`：`POST /api/v1/session/login`、`setup`、`PUT /api/v1/session/password` 的 401 **不得** 派发 `authenticationRequiredEvent`，否则输错密码会跳到过期页。setup/login 整页替换（无侧栏），并停掉 overview 轮询与 SSE。

过期文案（仅桌面/非密码流）：局域网提示改为打开 `/login`，链接同步改掉。fnOS 密码流不应再落到该页。

### 设置页 `/setup`

- fnOS 图标 +「设置面板密码」。
- 密码、确认；客户端检查一致且 8–128。
- `POST /api/v1/session/setup`；成功后进总览。
- 409：改走登录页。

### 登录页 `/login`

- 同一视觉：单栏居中，现有深色控制台风格。
- 一个密码框。`POST /api/v1/session/login`。
- 错误、429 页内提示。无「忘记密码」。

### 侧栏

快捷设置（`edition === 'fnos'` 或 `gui_password_required`）增加：

- 「修改面板密码」：当前 / 新 / 确认 → `PUT /api/v1/session/password`。
- 「退出登录」→ `POST /api/v1/session/logout` 后进入 `/login`。

macOS 桌面不显示这两项。

## 错误码

| HTTP | code | 何时 |
| --- | --- | --- |
| 401 | `authentication_required` | 未登录调业务 API；非 lanMode 调 setup/login |
| 401 | `invalid_password` | 登录或改密 current 错误 |
| 409 | `password_not_configured` | 未设却登录 |
| 409 | `password_already_configured` | 已设却 setup |
| 400 | `invalid_request` | 长度非法、JSON 非法 |
| 429 | `too_many_attempts` | 登录限速 |
| 403 | `invalid_host` / `origin_rejected` | 现有 Host / Origin |

错误 `message` 给操作者看的短句，不含密码。

## 测试

Go（`internal/controlapi`，表驱动）：

- setup 成功写 0600 文件并发 session；第二次 setup 409。
- 双请求并发 setup 只有一次成功。
- login 成功发 cookie；错误 401；未配置 409。
- 5 次错误后第 6 次 429；成功后计数清零。
- `/enter` 未设密码 → 302 `/setup` 无 cookie；已设无 session → 302 `/login` 无 cookie；有效 session → 302 dashboard 且有 cookie。
- 非 lanMode：`/enter` 仍 401；setup/login 401。
- 删密码文件后，旧 session 调 overview → 401。
- 改密：旧 session 401，当前响应带新 cookie。
- logout 清 cookie。
- Bearer 与 bootstrap 在已设密码后仍可进 API。
- Host 不在白名单仍 403。
- 密码不出现在错误 body。

前端 vitest：

- status 分流 setup / login / app。
- 401 不再 `location.replace('/enter')` 领 cookie（fnOS）。
- setup/login 表单校验；login 401 不打开「安全连接已过期」。
- fnOS 侧栏有改密/退出；mac 没有。

门槛：`make test`（`go test ./...`）+ `pnpm exec vitest run` 相关文件。不跑 `make lab-test` / `lab-test-tun`。

## 文档

实现时同步：

- `docs/fnos-port/FPK-USER-GUIDE.md` §5：`/enter` 改为跳到设置/登录；补充首次设密、删文件重置。
- `docs/fnos-port/DEPLOY.md` 安全说明。
- `docs/fnos-port/DESIGN.md` 中「/enter 校验 Host 后签发 session」改为「校验 Host 后进入设密或登录」。
- `docker/entrypoint.sh` 启动提示。
- `docs/agent-wiki/sources/control-api.md` 与 `wiki/concepts/gui-control-plane.md`：局域网浏览器入口需要面板密码。

## 实现落点

| 单元 | 文件 |
| --- | --- |
| 存储 | `internal/controlapi/store.go`（或新 `gui_password.go`） |
| HTTP / auth / enter | `internal/controlapi/server.go`；测试 `gui_password_test.go`、改 `baseurl_test.go` 的 `/enter` 用例 |
| 前端 API | `web/src/api.ts`、`web/src/types.ts` |
| 页面 | `web/src/pages/SetupPage.tsx`、`LoginPage.tsx`；`App.tsx` 分流与侧栏 |
| 文案 | `web/src/i18n.ts`、`i18n.en.ts` |
| 样式 | `web/src/styles.css` 少量登录卡 |

不把 `server.go` 无限制加长：密码 HTTP handler 放独立文件 `session_password.go`。
