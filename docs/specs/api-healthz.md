# API Liveness 规范：`GET /healthz`

## 1. 目的

明确基础设施探活接口 `GET /healthz` 的 HTTP 语义边界，避免将其纳入业务接口的缓存与条件请求体系。

## 2. 范围

本规范仅适用于：

- `GET /healthz`

该接口为无鉴权 liveness 探针，用于负载均衡、容器编排与监控系统判断进程是否存活。

## 3. 方法约束

- 仅提供 `GET /healthz`。
- 不提供 `HEAD /healthz`。
- 对 `HEAD /healthz` 的请求按未注册路由处理，返回 `404`。

原因：探活接口不需要与业务 `GET` 共享 `HEAD` 语义，也无需为其维护一致的 representation validator。

## 4. 缓存与条件请求排除

`GET /healthz` 不参与任何缓存体系：

- 不生成 `ETag`。
- 不处理 `If-None-Match`，即使客户端发送该头也始终返回 `200` 并携带完整响应体，不返回 `304`。
- 不设置 `Cache-Control`。
- 不设置 `Vary`。

## 5. 与 ETag 规范的关系

`docs/specs/etag.md` 定义的 ETag 生成与 `If-None-Match` 重验证规则不适用于 `GET /healthz`。业务接口的 `GET / HEAD` 缓存策略（`Cache-Control: private, no-cache`、`Vary: Authorization`、Weak/Strong ETag）均与该探活接口无关。

## 6. 测试要求

- `GET /healthz` 返回 `200`，响应体非空。
- `GET /healthz` 响应头中不存在 `ETag`、`Cache-Control`、`Vary`。
- 携带任意 `If-None-Match` 请求 `GET /healthz` 仍返回 `200`，不产生 `ETag` 或 `304`。
- `HEAD /healthz` 返回 `404`，且不携带 `ETag`。

## 7. CORS 说明

`GET /healthz` 仍受全局 `CORS` 中间件影响，但不得因缓存排除而额外暴露或要求 `ETag` / `If-None-Match` 相关头。
