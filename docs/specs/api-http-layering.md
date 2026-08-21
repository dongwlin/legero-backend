# HTTP 表示层与缓存分层规范

## 1. 目的

明确 HTTP 缓存、ETag / 表示验证器（Representation Validator）、响应渲染（`httpresp`）、Handler、`httpcache`、Response Envelope、Option / Metadata 之间的职责边界。

本规范为**架构约束规范**：定义“谁可以做什么、谁不得做什么”，不描述具体实现代码。任何现有代码与本规范不一致时，以本规范为准，代码应在后续演进中向规范对齐，而不是反向修改规范去适配代码。

> [!important]
> 本规范与 `api-etag.md` 互补：`api-etag.md` 定义 ETag 的语义与条件请求行为，本规范定义 ETag 与缓存能力在分层架构中的归属与协作方式。

---

## 2. 术语

| 术语 | 含义 |
| --- | --- |
| HTTP Representation | 同一资源在某一时刻通过 HTTP 传输的完整表示，包含状态码、头部与 body 字节。对 JSON API 而言，通常指最终写入 wire 的 JSON bytes 及相关头部。 |
| Representation Validator | 用于判定两个 representation 是否等价的验证器，例如 `ETag`、`Last-Modified`。分为 Strong（字节级一致）与 Weak（语义等价）两类。 |
| Composable Infrastructure | 可正交组合的基础设施能力：以中间件、装饰器、拦截器等形式叠加到请求处理链路上，不侵入业务渲染逻辑。 |
| `httpresp` | 将业务结果转换为 HTTP Response 的渲染层。提供 API JSON Contract 与通用 JSON 渲染能力。 |
| `httpcache` | HTTP 缓存与表示验证器基础设施。提供 `Validator` 抽象、`Strong` / `Weak` 构造及 `WithValidator` 等 Option。 |
| Response Envelope | 包裹业务负载的统一外层结构。自 `v2` 起为 `httpresp.Response { Code string; Message string; Data any }`（见 `api-versioning.md` §5.1）；`v1` 的历史结构由其自身包封闭管理，不提升为全局模型。 |
| Response Meta / Metadata | 描述 HTTP representation 附加信息的只读数据，例如验证器来源。不承载行为。 |
| Option / Config | `httpresp.JSON` 的扩展点：`type Option func(*Config)`，`Config{Metadata}`，`Metadata{Validator}`。仅描述 HTTP representation metadata。 |
| Validator | `httpcache` 提供的验证器抽象：`type Validator interface { ETag() string }`。 |

---

## 3. 总体分层原则

### 3.1 分层视图

```
API JSON Contract
        |
httpresp.Response
        |
httpresp.JSON(..., opts ...Option)
        |
HTTP Infrastructure
(ETag / Cache / Headers)
```

- `API JSON Contract` 与 `httpresp.Response` 表达“要返回符合项目 API 协议的 JSON”。
- `httpresp.JSON` 负责通用渲染。
- `HTTP Infrastructure`（`httpcache` 等）通过 `Option` 注入 representation metadata。

### 3.2 HTTP 缓存是可组合基础设施

- HTTP 缓存能力（`Cache-Control`、`Vary`、`ETag` 生成、`If-None-Match` / `If-Modified-Since` 匹配、`304 Not Modified` 短路等）**必须**作为可组合基础设施存在。
- **不得**嵌入 response 渲染层（`httpresp`）内部。渲染层不得通过分支、开关或参数来承载缓存行为。
- 缓存能力应以可插拔方式作用于请求链路，例如独立的 middleware / interceptor / decorator，或通过 `Option` 声明式注入后由基础设施消费，对 Handler 与 `httpresp` 保持正交。
- 启用、禁用或替换缓存策略时，`httpresp` 与业务 Handler 的渲染代码**不得**需要以侵入式方式修改。

> 反模式：`httpresp.JSONWithETag()`、`renderJSON(..., withETag=true)` —— 将缓存决策写进渲染函数签名。

> 正模式：`httpresp.JSON(c, 200, resp, httpcache.WithValidator(httpcache.Weak(...)))` —— 渲染与验证器声明分离。

### 3.3 职责单向流动

```text
Handler  --声明-->  Option(Metadata{Validator})  --由 httpcache 提供
   |
   +---->  httpresp.Response / httpresp.JSON  --渲染-->  HTTP Representation
                                                        |
                                                        v
                                           httpcache Infrastructure --生成/比较/写入--> ETag / 304
```

任何一层不得越权执行相邻层或跨层职责。

---

## 4. `httpresp` 的职责与边界

### 4.1 职责

`httpresp` 包的唯一职责是承载 API JSON Contract 与通用 JSON 渲染，并保持轻量：

- 定义统一信封 `Response` 并提供 `Success` 构造：

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

- 提供通用渲染入口：

  ```go
  func JSON(c *gin.Context, status int, body any, opts ...Option)
  ```

  其中 `body` 为业务 JSON（通常是已包裹的 `Response`），`opts` 仅承载 HTTP representation metadata。
- 将业务对象 / DTO 映射为 HTTP 状态码与 body，序列化 body 并设置 `Content-Type` 等表示头部，映射应用错误到 HTTP 状态码（仅传输层映射，不做业务判断）。

使用示例：

```go
resp := httpresp.Success(dto)
httpresp.JSON(c, http.StatusOK, resp)
```

表达“我要返回一个符合项目 API contract 的 JSON”。

### 4.1.1 与 Response Envelope 的关系

- 自 `v2` 起的统一信封 `Response{Code, Message, Data}` 即 `httpresp.Response`（见 `api-versioning.md` §5.1）。是否以及如何包裹由版本决定：
  - `v2` 及后续版本：统一使用 `httpresp.Response` / `httpresp.Success` 包裹业务负载，`httpresp.JSON` 负责序列化，不持有缓存逻辑。
  - `v1`：响应结构与渲染逻辑由 `v1` 自身包封闭管理，不依赖全局 `httpresp` 的信封抽象；`v1` 保持冻结，不因 `v2` 的信封收敛而被动迁移。

### 4.2 保持轻量的要求

- **必须**保持轻量：不引入缓存、验证器、条件请求等 HTTP 行为逻辑。
- **不得**依赖请求头（如 `If-None-Match`）做分支。
- **不得**持有缓存配置或 ETag 策略。
- 新增渲染能力时，应评估是否属于“结果到表示的纯转换”；若涉及 HTTP 行为，则应归入基础设施层。

### 4.3 `httpresp.Option / Config / Metadata` 的形态约束

- `Option` 仅描述 HTTP response extension：

  ```go
  type Option func(*Config)

  type Config struct {
      Metadata Metadata
  }

  type Metadata struct {
      Validator Validator
  }
  ```

- `Metadata` 内**不得**直接放置 `CacheControl string` / `ETag string` 等 header 明文，否则退化为 header 层。
- `Metadata` 字段应为事实陈述（如 `Validator`），非指令；`httpresp` 不解释业务语义，仅透传给基础设施消费。
- `Validator` 的具体类型由 `httpcache` 定义（见 §5），`httpresp` 不定义验证器。

### 4.4 禁止事项

- `httpresp` **不感知** ETag 的存在。
- **不负责** Strong / Weak ETag 的类型选择。
- **不负责** ETag 的生成（hash / version 拼接等）。
- **不负责** ETag 的比较（weak comparison、`*`、多值解析等）。
- **不负责** ETag 的写入（`ETag` 响应头）与 `304` 的短路返回。

> 判定：若 `httpresp` 需要 `import` 与 ETag 生成或比较相关的依赖，或函数签名中出现 `etag`、`If-None-Match`、`validator` 等缓存概念的实现，即视为越界。

---

## 5. ETag 与 Representation Validator 的归属

- ETag 属于 **HTTP Representation Validator** 能力，**必须**独立于 `httpresp` 包，由 `httpcache` 提供。
- Representation Validator 是 HTTP 层的概念，作用对象是**已确定的 representation**（或其等价描述），而非渲染过程的中间产物。
- `httpcache` 定义验证器抽象：

  ```go
  package httpcache

  type Validator interface {
      ETag() string
  }

  func Strong() Validator
  func Weak(resource string, id string, version int64) Validator
  ```

  - `Strong()` 的 ETag 由最终 JSON bytes 的 SHA-256 推导，不由 handler 计算。
  - `Weak(resource, id, version)` 生成 `W/"{resource}-{id}-{version}"`，`resource` 为服务端固定 token，`id` 为合法 UUID，`version` 为单调递增业务版本。
- 独立的验证器基础设施负责：根据输入（representation bytes 或声明的 `Validator`）生成 ETag；解析与比较 `If-None-Match` / `If-Match`；决定是否写入 `ETag`、是否返回 `304`、是否移除 `Content-Length` / `Content-Type` 等行为。
- `httpresp` 与 `httpcache` 之间**不得**直接耦合；两者通过 `Option` / `Metadata` 与外层编排（如 middleware 读取最终 body 或 `Config` 后调用验证器）协作。

---

## 6. Handler 的职责与边界

### 6.1 可以做的

- 通过 `httpcache` 提供的 `Option` 声明 representation validator 的来源（例如：声明本接口应为 Strong 还是 Weak，以及 Weak 时所依赖的 `resource/version`）。该声明是**描述性**的，用于告知基础设施“本 representation 的等价性应如何判定”，而非指令性操作。

  ```go
  httpresp.JSON(c, http.StatusOK, resp,
      httpcache.WithValidator(httpcache.Weak("order", order.ID.String(), order.Version)),
  )
  ```

  或等价的：

  ```go
  httpresp.JSON(c, http.StatusOK, resp,
      httpresp.Option(httpcache.Metadata{Validator: httpcache.Weak(...)}),
  )
  ```

  （依赖方向保持 `handler -> httpcache`，见 §9）

- 将业务负载包裹为 `httpresp.Response` / `httpresp.Success` 后交由 `httpresp.JSON` 渲染。

### 6.2 不得做的

- Handler **不得**负责 ETag 的生成（不得计算 hash、拼接 `W/"..."`）。
- **不得**负责 ETag 的比较（不得解析 `If-None-Match`）。
- **不得**负责 ETag 的写入（不得直接 `c.Header("ETag", ...)` 或 `c.AbortWithStatus(304)`）。

> 推论：Handler 不应直接 `import` ETag 生成或比较的实现细节；若需要验证器，只通过 `httpcache` 提供的 `Validator` / `Option` 来表达意图。

### 6.3 示例边界

Strong（订单列表）：

```go
func (h *OrderHandler) List(c *gin.Context) {
    result, err := h.orderSvc.List(...)
    if err != nil { httpresp.AbortError(c, err); return }
    resp := httpresp.Success(dto.ListOrdersResponse{Items: toOrderDTOs(result.Items, h.location)})
    httpresp.JSON(c, http.StatusOK, resp, httpcache.WithValidator(httpcache.Strong()))
}
// 流程：Response -> Marshal -> JSON bytes -> SHA-256 -> ETag
```

Weak（订单详情）：

```go
func (h *OrderHandler) Get(c *gin.Context) {
    order, err := h.orderSvc.Get(...)
    if err != nil { httpresp.AbortError(c, err); return }
    resp := httpresp.Success(dto.OrderResponse{Item: order.ToDTO(h.location)})
    httpresp.JSON(c, http.StatusOK, resp, httpcache.WithValidator(httpcache.Weak("order", order.ID.String(), order.Version)))
}
// 流程：order version -> W/"order-id-version"
```

反例：

```text
错误：Handler 内计算 SHA-256 / 拼接 W/"..." 并写入 Header，或根据 If-None-Match 自行返回 304
正确：Handler 仅通过 httpcache.Weak / Strong 声明验证器来源，生成/比较/写入由 httpcache 基础设施完成
```

---

## 7. Response Metadata / Option 的定义与约束

### 7.1 定义

- `Option` / `Metadata` 用于**描述** HTTP representation 的附加信息，例如验证器来源。它们是**数据**，不是**行为**。
- `Metadata` 的典型字段为 `Validator`；未来可扩展 `Content-Disposition`、`Content-Type override` 等纯表示描述。

### 7.2 核心约束

- `Option` / `Metadata` **只能描述**，**不得包含** HTTP 行为。
- **不得**执行 HTTP 操作，例如：写入 Header / Status；读取请求头；触发 `304` 短路；序列化或修改 body。
- 必须是无副作用的值对象，可在测试中直接断言。

### 7.3 设计要求

- 字段应为**事实陈述**（如 `Validator: httpcache.Weak(...)`），而非指令（如 `writeETag=true`）。
- 基础设施层消费 `Config/Metadata` 来决定如何生成与校验 validator；`httpresp` 仅透传，不消费业务语义。

> 反模式：`Meta.WriteHeaders(c)`、`Meta.Apply(c)`、`Meta.HandleIfNoneMatch(c)`。

> 正模式：`httpcache` 的 middleware / 渲染后置逻辑读取 `Config.Metadata.Validator`，自行决定 ETag 策略；`httpresp` 仅渲染 `body`。

### 7.4 Option 允许清单（防垃圾桶约束）

`httpresp.JSON` 的 `Option` 只能描述 **HTTP representation metadata**，不允许承载业务逻辑或缓存策略执行。未来若放开为 `WithHeader` / `WithCookie` / `WithCache` / `WithETag` 等行为式入口，`Option` 将退化为垃圾桶，必须禁止。

| 类别 | 示例 | 约束 |
| --- | --- | --- |
| 允许 | `Validator`、`Content-Disposition`、`Content-Type override` | 纯表示描述，可直接作为 metadata |
| 谨慎 | `Cache-Control`、`Vary` | 是否应由基础设施默认策略统一处理而非逐接口声明，需评估；避免逐接口散落策略 |
| 禁止 | `Return304`、`CheckIfNoneMatch`、`GenerateETag`、`WithHeader`、`WithCookie`、`WithCache`、`WithETag` 等 | 行为式或策略式 Option，不得出现；出现此类命名即视为越界 |

---

## 8. Validator 的面向对象

- Validator **面向 HTTP Representation**，而不是直接面向 Domain Object。
- Domain Object（`Order`、`User` 等）是业务事实的载体；Representation 是这些事实经过 DTO 转换、序列化后并包裹 `Response` 信封的 HTTP 形态。
- 因此：
  - Strong Validator 的输入**必须**是最终 representation bytes（含 `Response` 信封后的 JSON bytes）或其确定性派生，而非 `domain.Order` 本身。
  - Weak Validator 虽可基于 `version` 等 domain 事实推导，但其语义仍是“同一 version 下的 representation 语义等价”，而非“domain 对象相等”。Validator 的正确性由 representation 是否等价来判定。
- **不得**将 domain 对象直接作为 validator 的比较对象（例如 `if oldOrder == newOrder`），也不得在 domain 层生成面向 HTTP 的 `ETag` 字符串。

> 推论：`version` 是推导 Weak ETag 的**依据**，但 Weak ETag 本身仍是 representation 层的概念；更换 representation 结构（如 DTO 字段增删）即使 `version` 未变，也可能需要通过新路由版本来保证 validator 的正确性（见 `api-versioning.md`）。

---

## 9. 依赖方向与禁止事项

### 9.1 允许的依赖

```text
handler  -->  httpresp       (渲染, Response/Success/JSON)
handler  -->  httpcache      (声明, Validator/WithValidator)

httpresp --X-->  httpcache

httpcache infrastructure  -->  Config/Metadata/Validator  (消费)
httpcache infrastructure  -->  HTTP representation bytes   (消费)

domain    --X-->  ETag / HTTP
```

### 9.2 禁止清单

| 禁止项 | 说明 |
| --- | --- |
| `httpresp` import / 调用 `httpcache` 或 ETag 生成、比较、写入逻辑 | 违反 §4.4 / §5 |
| `httpresp` 定义 `WithValidator` / `WithETag` 等缓存相关 Option | 违反 §4.3 / §7.4，Option 归属应为 `httpcache` |
| `httpresp` 出现 Strong/Weak 选择分支 | 违反 §4.4 |
| Handler 直接计算或写入 ETag / `304` | 违反 §6.2 |
| Response Metadata / Config 包含 `WriteHeader` / `Abort` 等方法 | 违反 §7.2 |
| Validator 直接对 `domain.Order` 等做相等比较 | 违反 §8 |
| 将缓存策略写进 `httpresp` 函数参数（如 `withETag bool`） | 违反 §3.2 |
| `Option` 出现 `Return304` / `CheckIfNoneMatch` / `GenerateETag` / `WithHeader` / `WithCookie` 等行为式命名 | 违反 §7.4 |

---

## 10. 版本化与演进

### 10.1 v1 自管理与冻结

- `v1`（` /api/*`）的响应形态与渲染逻辑由其自身包封闭管理，视为已冻结的历史版本（见 `api-versioning.md` §5.2）。
- `v1` 不再接受新的基础设施能力叠加：自本规范起，`v1` 不再支持 ETag / `If-None-Match` / `Cache-Control: private, no-cache` / `Vary: Authorization`（见 `api-etag.md` §2.1）。
- 对 `v1` 的任何改动仅限于包内兼容性修复，不得改变对外可见响应结构，不得引入全局信封或缓存设施的耦合。

### 10.2 v2 统一信封与新的基础设施边界

- 自 `v2`（` /api/v2/*`）起，所有响应统一包裹为 `httpresp.Response{Code, Message, Data}` / `httpresp.Success`（见 `api-versioning.md` §5.1），该信封是版本化契约的一部分。
- `v2` 及后续版本的 ETag / 缓存能力按本规范 §3 / §4 / §5 / §6 / §7 / §8 的分层执行：`httpresp` 保持轻量且不感知 ETag，仅通过 `Option(Config{Metadata{Validator}})` 透传声明；ETag 由 `httpcache` 基础设施基于最终 representation（含信封后的 bytes）或声明的 `Validator` 生成与校验，Handler 仅声明验证器来源。
- Strong ETag 的输入为 v2 信封包裹后的最终 HTTP body bytes，见 `api-etag.md` §3.2。

### 10.3 演进与兼容

- 本规范为目标架构约束，不要求一次性重构现有代码。现有实现若与本规范冲突，应视为**待对齐的技术债**，在后续迭代中逐步迁移。
- 新增接口与新增缓存能力**必须**遵循本规范；不得以“与旧代码保持一致”为由延续越界设计。
- `api-etag.md` 中关于 ETag 格式、weak comparison、`304` 语义、`HEAD`、`Cache-Control` 等行为定义继续有效；本规范不改变其语义，仅约束这些行为由哪一层负责。
- 任何对本规范的偏离必须在设计文档中显式记录原因与回迁计划，不得默许。

---

## 11. 审查清单

实现或评审时，按以下清单逐项检查：

- [ ] 缓存/ETag 能力是否以可组合基础设施（`httpcache`）形式存在，未嵌入 `httpresp`？
- [ ] `httpresp` 是否仅做“结果到表示”的轻量转换与 `Response` 信封承载，未感知 ETag？
- [ ] `httpresp.JSON` 是否为 `func JSON(c, status, body, opts ...Option)` 形态且 `opts` 仅描述 representation metadata？
- [ ] `Option` / `Validator` 是否归属 `httpcache`，`httpresp` 未 `import httpcache`？
- [ ] ETag 生成/比较/写入是否在独立于 `httpresp` 的 `httpcache` 模块中？
- [ ] Handler 是否仅通过 `httpcache.Weak` / `Strong` 声明 validator 来源，未直接生成/比较/写入 ETag？
- [ ] `Config` / `Metadata` 是否仅为描述性数据，未包含 HTTP 行为？`Option` 是否未出现行为式命名？
- [ ] Validator 是否面向 representation（含信封后的 bytes），而非直接面向 domain object？
- [ ] v1 响应是否由 v1 包自管理且未叠加 ETag / 缓存设施？
- [ ] v2 响应是否统一包裹为 `httpresp.Response{Code, Message, Data}` 且 Strong ETag 基于信封后的最终 bytes？
