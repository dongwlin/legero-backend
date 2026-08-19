# ETag 设计规范

## 1. 目的

定义 GET / HEAD 响应的 ETag 生成与 `If-None-Match` 重验证规则，使客户端能够对 HTTP representation 发起条件请求。

当表示未发生变化时，服务端返回：

```http
304 Not Modified
```

从而避免重复传输响应体喵。

> [!important]  
> ETag 只负责 **cache validation / revalidation**，不承担新鲜度控制职责。
> 
> 鉴权接口必须保持私有缓存，并要求客户端在复用缓存前进行重验证。

---

## 2. 适用范围

ETag 仅应用于：

- `GET`
- `HEAD`
- `200 OK`

以下情况不生成 ETag：

- `POST`
- `PUT`
- `PATCH`
- `DELETE`
- `204 No Content`
- `206 Partial Content`
- 其他非 `200` 状态
- `4xx`
- `5xx`

---

# 3. ETag 选型规则

GET / HEAD 成功返回 `200 OK` 时，根据接口 representation 的来源选择两种 ETag 方案之一。

```text
单资源
  ↓
resource version
  ↓
Weak ETag

列表 / 统计 / 聚合 / 报表
  ↓
最终 response bytes
  ↓
SHA-256
  ↓
Strong ETag
```

---

## 3.1 单资源接口：Version ETag

### 格式

推荐格式：

```http
ETag: W/"{resource}-{id}-{version}"
```

例如：

```http
ETag: W/"order-0198cabc-1234-5678-9abc-def012345678-42"
```

各字段含义：

| 字段         | 含义              |
| ---------- | ------------------- |
| `resource` | 资源类型，例如 `order` |
| `id`       | 资源 ID，例如 UUID     |
| `version`  | 单调递增的业务资源版本     |

### Version

`version` 必须满足：

- 每次可能影响资源语义的修改都递增。
- 同一个资源同一个 version 下 ETag 保持稳定。
- version 发生变化后 ETag 必须发生变化。

例如：

```text
Order
├── ID      = abc
└── Version = 42
```

对应：

```http
ETag: W/"order-abc-42"
```

资源更新后：

```text
Version = 43
```

对应：

```http
ETag: W/"order-abc-43"
```

---

### 修改响应结构

由于 ETag 格式不包含 representation revision，如果后端需要修改某个单资源接口的响应结构，**不允许**在现有路径上直接变更响应结构，否则旧客户端持有的缓存会与新的响应结构不一致。

正确做法是开一个新的路由版本，例如：

```text
GET /orders/:id      → 保持旧响应结构不变
GET /v2/order/:id    → 新响应结构
```

> [!note]  
> 响应结构变更必须通过新的路由版本进行，具体约定见 `api-versioning.md`。

---

### 为什么使用 Weak ETag

Version ETag 表示：

> 同一个资源 version 对应语义等价的 representation 。

但它无法保证最终 JSON bytes 完全一致。

因此必须使用：

```http
W/"..."
```

而不是 strong ETag。

例如：

```http
ETag: W/"order-abc-42"
```

---

### Version ETag 的优势

服务端可以直接从资源数据构造 ETag：

```text
查询资源
  ↓
读取 ID + Version
  ↓
构造 ETag
  ↓
检查 If-None-Match
  ├── 命中 → 304
  └── 未命中
        ↓
      DTO 转换
        ↓
      JSON Marshal
        ↓
      200
```

因此当缓存命中时，可以绕过：

- DTO 转换
- JSON 序列化
- Response Body 构造
- Response Body 传输

这也是 Version ETag 相比 response hash 的主要优势。

---

### 判定标准

适合 Version ETag 的接口必须满足：

> 请求通过固定 ID 寻址单个资源，并且响应语义主要由该资源自身决定。

例如：

```http
GET /orders/:id
GET /users/:id
GET /workspaces/:id
```

---

## 3.2 复杂聚合接口：Response Hash ETag

复杂 representation 使用最终响应 body 的 SHA-256 作为 ETag 。

### 格式

```http
ETag: "<hex(sha256(body))>"
```

例如：

```http
ETag: "be72ac42f515dbf91c6c7eae9d09845..."
```

不添加：

```text
W/
```

因此这是 **Strong ETag** 。

---

### 生成流程

```text
查询 / 聚合数据
      ↓
构造 DTO
      ↓
JSON Marshal
      ↓
得到最终 response body bytes
      ↓
SHA-256
      ↓
生成 ETag
      ↓
检查 If-None-Match
      ├── 命中 → 304
      └── 未命中 → 写入同一份 body
```

必须对：

> **最终实际准备写入 HTTP response 的 bytes**

进行 SHA-256 。

例如：

```go
body, err := json.Marshal(response)
if err != nil {
    // handle error
}

sum := sha256.Sum256(body)
etag := `"` + hex.EncodeToString(sum[:]) + `"`
```

随后发送的也必须是同一份：

```go
body
```

从而保证：

```text
ETag hash source == actual HTTP representation
```

---

### 为什么 Hash 整个 Response

假设统一响应结构：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

即使主要业务内容位于：

```text
data
```

仍然推荐对**整个最终 response body**进行 hash 。

原因是 strong ETag 描述的是：

> HTTP representation 是否字节级一致。

而不是：

> 业务 `data` 是否一致。

因此即使：

```text
data 不变
```

但：

```text
message
```

或其他 envelope 字段发生变化，ETag 也应该发生变化。

---

### 适用场景

Response Hash ETag 适合：

- 列表
- 分页
- 过滤
- 搜索
- 统计
- 聚合
- 报表
- 多资源组合

例如：

```http
GET /orders
GET /orders?status=uncompleted
GET /orders?limit=50
GET /stats/daily
GET /stats/report
```

---

### 判定标准

如果 representation：

> 依赖一组底层资源、查询参数、统计结果或聚合计算，而不是由一个具有可靠 version 的实体唯一决定。

则使用 Response Hash ETag 。

---

## 3.3 Version 与 Hash 的关系

可以简单记为：

```text
resource version
    ↓
Weak ETag

response bytes
    ↓
SHA-256
    ↓
Strong ETag
```

但真正决定 weak / strong 的并不是：

```text
单资源 / 聚合资源
```

而是：

```text
validator 是否能保证 representation byte-level equality
```

因此理论上：

```text
单资源 + response hash
```

也可以产生 Strong ETag 。

反过来：

```text
聚合接口 + max(updated_at)
```

通常只能作为 Weak ETag 。

---

# 4. 格式要求

## Version ETag

格式：

```http
W/"{resource}-{id}-{version}"
```

要求：

- 必须带 `W/` 。
- ETag 必须使用双引号包围。
- 内容保持稳定。
- 推荐只使用字母、数字与连字符。

例如：

```http
W/"order-0198cabc1234-42"
```

---

## Response Hash ETag

格式：

```http
"<hex(sha256(body))>"
```

例如：

```http
"526573706f6e736548617368..."
```

要求：

- 不带 `W/` 。
- SHA-256 输入必须是最终 HTTP body bytes 。
- 相同 response bytes 必须生成相同 ETag 。
- response bytes 任意变化都必须导致 ETag 变化。

---

# 5. If-None-Match 重验证

客户端可以发送：

```http
If-None-Match: W/"order-abc-42"
```

或者：

```http
If-None-Match: "abcdef..."
```

服务端计算当前 ETag 后进行比较。

---

## 5.1 命中

如果匹配：

```http
HTTP/1.1 304 Not Modified
ETag: ...
```

要求：

- 返回 `304 Not Modified`。
- 保留 `ETag`。
- 保留必要的缓存相关 response headers。
- 不写响应体。
- 项目统一删除 `Content-Length`。
- 项目统一删除 `Content-Type`。

---

## 5.2 Weak Comparison

`If-None-Match` 必须始终使用 **weak comparison**。

因此：

```http
If-None-Match: "abc"
```

可以匹配：

```http
ETag: W/"abc"
```

反过来也成立。

比较时相当于忽略：

```text
W/
```

---

## 5.3 多个 Entity Tag

例如：

```http
If-None-Match: "abc", W/"def", "ghi"
```

必须按照 entity-tag 语法逐个解析。

不得简单：

```go
strings.Split(value, ",")
```

因为 opaque-tag 内部合法字符中可能出现逗号。

---

## 5.4 Wildcard

合法：

```http
If-None-Match: *
```

表示当前 representation 存在时匹配。

`*` 必须作为整个字段值出现。

例如：

```http
If-None-Match: "abc", *
```

不得视为合法 tag list 中的 wildcard 。

---

## 5.5 非法语法

如果：

```http
If-None-Match
```

不符合合法语法，则忽略该条件头并正常处理请求。

例如最终返回：

```http
200 OK
```

而不是因为解析失败返回：

```http
400 Bad Request
```

---

## 5.6 多个同名 Header

如果请求中存在多个：

```http
If-None-Match
```

header field，需要按 HTTP field value 规则组合后统一解析。

---

# 6. If-Match

Version ETag 是 Weak ETag：

```http
W/"order-abc-42"
```

因此不得用于要求 strong comparison 的：

```http
If-Match
```

喵。

即：

```text
Version Weak ETag
    ↓
If-None-Match ✅
If-Match      ❌
```

Response Hash Strong ETag 理论上可以参与：

```http
If-Match
```

严格比较喵。

不过是否实际开放 `If-Match` 应由具体 API 设计决定。

---

# 7. HEAD

`HEAD` 与对应的 `GET` 使用相同 representation validator，因此必须返回相同 ETag 。

例如：

```http
GET /orders/abc
ETag: W/"order-abc-42"
```

对应：

```http
HEAD /orders/abc
ETag: W/"order-abc-42"
```

HEAD 不写 response body 。

---

## Content-Type

如果对应 GET 会返回：

```http
Content-Type: application/json; charset=utf-8
```

HEAD 可以返回相同 Content-Type。

---

## Content-Length

如果可以在不生成 response body 的情况下确定长度，则可以设置：

```http
Content-Length: ...
```

其值必须与对应 GET 实际发送的 body 长度一致喵。

如果精确长度只有执行：

```text
DTO → JSON Marshal
```

后才能确定，则允许省略 `Content-Length`，避免仅为 HEAD 构造并丢弃 response body。

> [!important]  
> 不得为了避免 marshal 而填写估算值或错误的 Content-Length。

---

# 8. 缓存策略

鉴权 GET / HEAD 接口统一：

```http
Cache-Control: private, no-cache
```

含义：

- `private`：响应只能由私有缓存保存，例如浏览器缓存。
- `no-cache`：缓存可以保存，但再次使用之前必须向 origin server 重验证。

因此：

```text
private, no-cache
```

并不是：

```text
完全禁止缓存
```

如果希望完全禁止保存，则应该使用：

```http
Cache-Control: no-store
```

但本 ETag 方案的目的本身就是允许客户端保存 representation 并重验证，因此不使用 `no-store` 。

---

## Vary

鉴权 representation 应增加：

```http
Vary: Authorization
```

追加时：

- 不覆盖已有 `Vary`。
    
- 大小写不敏感去重。
    

例如已有：

```http
Vary: Accept-Encoding
```

最终：

```http
Vary: Accept-Encoding, Authorization
```

---

# 9. CORS

浏览器客户端需要能够：

1. 发送 `If-None-Match`。
    
2. 读取 `ETag`。
    
3. 发起 `HEAD`。
    

因此 CORS 至少需要：

```text
AllowHeaders:
  If-None-Match

ExposeHeaders:
  ETag

AllowMethods:
  GET
  HEAD
```

以及项目原本允许的其他 HTTP methods。

---

# 10. Compression 注意事项

Strong ETag 描述的是 representation data 的字节级一致性。

如果未来加入：

```text
gzip
br
```

等 content coding，需要注意：

```text
原始 JSON bytes
      ↓
    gzip
      ↓
客户端收到 compressed bytes
```

如果 ETag 在压缩前计算：

```text
SHA256(JSON)
```

但：

```text
gzip representation
identity representation
```

共同使用完全相同的 Strong ETag，就可能不再满足严格的 strong validator 语义。

因此未来增加 compression middleware 时，需要保证：

> 不同 representation data 不得错误共享同一个 Strong ETag 。

可以选择：

- 根据 content coding 生成不同 Strong ETag 。
- 或重新评估是否应该使用 Weak ETag。

---

# 11. 测试要求

## 通用测试

Version 与 Hash 两种方案均至少覆盖：

- 相同输入产生稳定 ETag。
- representation 变化后 ETag 变化。
- `If-None-Match` 命中返回 `304`。
- `304` 没有 response body。
- `304` 保留 ETag。
- `If-None-Match` 未命中返回 `200`。
- 非法 `If-None-Match` 返回正常 `200`。
- wildcard `*` 行为正确。
- 多 entity-tag list 行为正确。
- weak comparison 正确。
- `HEAD` 返回与 GET 相同 ETag。
- `HEAD` 不写 response body。
- 非 GET / HEAD 不生成 ETag。
- 非 `200 OK` 不生成 ETag。
- 错误响应不生成 ETag。

---

## Version ETag 测试

必须覆盖：

```http
W/"{resource}-{id}-{version}"
```

以及：

- 带 `W/` 前缀。
- 相同 ID + version 产生相同值。
- 不同 ID 产生不同值。
- version 增长后 ETag 变化。
- 命中 `If-None-Match` 后能够在 JSON render 前直接返回 `304`。
- Weak ETag 不参与 `If-Match` strong comparison。

---

## Response Hash ETag 测试

必须覆盖：

```text
SHA256(final response body bytes)
```

以及：

- 不带 `W/` 前缀。
- 同一 response body 产生相同 ETag。
- response body 任意 byte 变化后 ETag 变化。
- hash 所使用的 body 与实际写入 response 的 body 完全相同。
- 查询参数变化导致 response bytes 变化时 ETag 必须变化。
- 查询参数变化但最终 response bytes 完全相同时，允许 ETag 保持一致。

---

# 12. 接口选择示例

|Endpoint|类型|ETag|
|---|---|---|
|`GET /orders/:id`|单资源|Version Weak ETag|
|`GET /users/:id`|单资源|Version Weak ETag|
|`GET /workspaces/:id`|单资源|Version Weak ETag|
|`GET /orders`|列表|SHA-256 Strong ETag|
|`GET /orders?status=...`|查询列表|SHA-256 Strong ETag|
|`GET /stats/daily`|统计|SHA-256 Strong ETag|
|`GET /stats/report`|聚合报表|SHA-256 Strong ETag|
|`GET /bootstrap`|动态组合数据|按实际 representation 特性决定|

---

# 13. 实现原则总结

> [!summary]  
> **Resource Version 描述资源有没有发生语义变化，Response Hash 描述最终 HTTP representation 有没有发生字节变化。**

最终规则：

```text
单资源
→ ID + Version
→ Weak ETag
→ W/"order-{id}-{version}"

复杂 representation
→ Marshal final response
→ SHA-256(body)
→ Strong ETag
→ "<sha256>"
```

条件请求：

```text
If-None-Match
→ Weak Comparison
→ Match
   ├── Yes → 304
   └── No  → 200
```

鉴权缓存：

```http
Cache-Control: private, no-cache
Vary: Authorization
```

整体目标是：

```text
正确性
  +
稳定的 validator
  +
减少 response body 传输
  +
Version ETag 命中时减少 JSON render
```
