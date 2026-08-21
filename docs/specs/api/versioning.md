# API 版本规范

## 1. 目的

定义 API 响应结构变更（breaking change）时的路由版本约定，保证旧客户端持有的缓存与响应结构始终一致。

## 2. 版本从哪里开始

当前所有 API 路由直接挂在 `/api` 下，**没有** `/v1` 前缀。这是历史遗留：在路由刚建立时没有版本规划，等意识到需要版本化时，现有客户端已经接入 `/api/...`，无法再为这些路径补上 `/v1`，否则需要全体客户端同步迁移，代价不可接受。

因此：

- `/api` 下的现有路由即事实上的 v1，保持响应结构不变。
- 项目**无法直接增加 `/api/v1`**，后续新版本从 `/api/v2` 开始。

## 3. 什么算响应结构变更

以下变更视为 breaking change，**必须**通过新的路由版本发布：

- 响应字段增删或重命名。
- 字段类型变化（例如整数改为字符串）。
- 嵌套结构层级调整。
- 状态码或错误码语义变化。
- 默认值、分页行为等会影响旧客户端解析或展示的语义变化。

仅新增可选字段，且旧客户端可以安全忽略时，不需要新版本，但需要谨慎评估。

## 4. 路由版本约定

新版本路由格式：

```text
/api/v{n}/...
```

从 `/api/v2` 开始。例如：

```text
GET /api/orders/:id      → 保持旧响应结构不变
GET /api/v2/orders/:id   → 新响应结构
```

要求：

- 旧版本路径与响应结构永久保持，不修改、不删除。
- 新版本只能追加，不能替换旧版本。
- 每个版本对应独立的 handler 与 DTO 实现，不复用旧版本结构。

## 5. 响应信封（Response Envelope）

### 5.1 v2 统一信封

自 `v2`（` /api/v2/*`）起，所有 API 响应使用统一信封 `httpresp.Response`：

```go
package httpresp

type Response struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}

func Success(data any) Response {
    return Response{Code: "0", Message: "success", Data: data}
}
```

约定：

- `code` 为机器可读的业务状态码（例如 `success` / 具体错误码），与 HTTP 状态码正交。
- `message` 为面向人类的可读信息，可用于兜底展示。
- `data` 承载业务负载；无业务数据时取 `null` 或对应的空结构，由各接口 DTO 定义。
- 信封字段名与结构在 v2 内保持稳定，不得随意增删或重命名；变更按 §3 视为 breaking change，需开新版本。
- 使用方式：`resp := httpresp.Success(dto); httpresp.JSON(c, http.StatusOK, resp, ...)`，表达“返回符合项目 API contract 的 JSON”（见 `architecture.md` §4.1 / §6.1）。

成功与错误响应均包裹于同一信封内，客户端以 `code` 判定业务结果，而非仅依赖 HTTP 状态码。

### 5.2 v1 自管理

`v1`（` /api/*`）当前使用的响应结构由其自身包封闭管理：

- 响应结构、DTO、序列化逻辑收敛于 `v1` 自身包下（例如 `internal/handler/v1` 及子包），不提升为全局共享模型。
- 不在全局层为 v1 定义统一信封；v1 的历史形态保持冻结，不因 v2 的信封收敛而被动迁移或改造。
- 如需对 v1 做兼容性修复，仅在 v1 包内进行，且不得改变其对外可见的响应形态（保持 §4 的冻结语义）。

### 5.3 信封与版本的绑定

- 统一信封是 `v2` 的版本化契约，不追溯适用于 `v1`。`v1` 与 `v2` 的信封差异本身即为版本差异的一部分。
- 跨版本不得复用对方的信封或 DTO；每个版本拥有独立的 handler 与 DTO 实现（见 §4）。

## 6. 与 ETag 的关系

ETag 只描述当前版本下 representation 是否变化，不承担版本演进职责。

单资源接口的 ETag 格式：

```http
ETag: W/"{resource}-{id}-{version}"
```

`resource` 使用服务端固定标识，`id` 使用合法 UUID 字符串，因而该 weak
ETag 可以直接保持可读格式；不对受约束的字段额外做编码。

当响应结构发生 breaking change 时：

```text
不能：修改现有路径的响应结构，让旧 ETag 缓存语义失效
应该：开新路由版本 /api/v{n}/...，旧路径保持不变
```

### 6.1 ETag 的版本范围与分层

`conditional-request.md` 定义的 ETag 能力仅自 `v2` 起生效（`conditional-request.md` §2.1），且按 `architecture.md` 的分层执行：

- `v1`（` /api/*`）不再支持 ETag：不生成 `ETag`，不处理 `If-None-Match`，不叠加 `Cache-Control: private, no-cache` / `Vary: Authorization` 等缓存设施。
- `v2`（` /api/v2/*`）的验证器由 `httpcache` 提供（`httpcache.Validator` / `httpcache.Weak` / `httpcache.Strong`），通过 `httpresp.JSON(..., httpcache.WithValidator(...))` 声明式注入，`httpresp` 不感知 ETag（见 `architecture.md` §5 / §6 / §9）。

## 7. 兼容性保证

- 旧客户端继续使用 `/api/...`，不受新版本与新信封影响。
- 新客户端使用 `/api/v2/...`（或更新的版本）并按 v2 统一信封解析。
- 服务端同时服务多个版本，直到旧版本不再有客户端使用。
