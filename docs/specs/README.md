# Specifications

本目录包含 Legero 项目的所有规范文档。规范是规范性的——当代码与规范冲突时，修复代码或有意更新规范。

## 结构

```
docs/specs/

README.md                          ← 本文件

api/
├── architecture.md                ← HTTP 架构分层与职责边界
├── versioning.md                  ← API 生命周期与版本策略
└── conditional-request.md         ← 条件请求与缓存验证（ETag / If-None-Match）

operations/
└── health-check.md                ← 探活接口规范

domain/
└── order-date-semantics.md        ← 订单日期语义
```

## 依赖关系

规范之间存在以下依赖方向：

```
versioning
    │
    ▼
architecture
    │
    ▼
conditional-request
```

- **versioning** 定义 API 版本策略与响应信封，是其他规范的基础。
- **architecture** 依赖 versioning 的信封定义，定义 HTTP 分层架构中各层的职责边界。
- **conditional-request** 依赖 architecture 的分层约束，定义 ETag 生成、`If-None-Match` 重验证等条件请求行为。
- **health-check** 独立于 API 缓存体系，明确排除在 ETag 与条件请求之外。

## API 规范

| 规范 | 职责 |
| --- | --- |
| [`api/architecture.md`](api/architecture.md) | HTTP 缓存 / 验证器 / httpresp / Handler / Response Meta 的分层与可组合基础设施约束 |
| [`api/versioning.md`](api/versioning.md) | 路由版本约定、v2 统一 `Response{Code, Message, Data}` 信封、v1 自管理冻结 |
| [`api/conditional-request.md`](api/conditional-request.md) | ETag 生成规则（Version Weak / SHA-256 Strong）、`If-None-Match` 重验证、`HEAD`、`Cache-Control` / `Vary`、v1/v2 版本范围 |

## 运维规范

| 规范 | 职责 |
| --- | --- |
| [`operations/health-check.md`](operations/health-check.md) | `GET /healthz` 探活接口语义，排除于 ETag 与缓存体系之外 |

## 领域规范

| 规范 | 职责 |
| --- | --- |
| [`domain/order-date-semantics.md`](domain/order-date-semantics.md) | 订单日期语义与时间处理规则 |
