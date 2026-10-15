# 请求性能关联计时

服务端支持现场记录器按请求启用 `Server-Timing`。普通请求不创建请求计时器，也不返回性能头；没有新增默认开放的 pprof 路由或后台采样任务。

客户端在一次记录开始时生成随机会话标识，为每次 HTTP 尝试递增计数，并发送：

```text
X-Performance-Request-ID: <随机会话标识>:<请求计数>
```

会话标识只接受标准 UUID 或 `local-` 加 8–32 个小写 ASCII 字母／数字；计数为 1–999999999，不接受前导零。这与客户端的 UUID／base36 随机后备格式一致。不要使用用户、手机号、订单或门店标识。重复头、格式不合法的头和协议升级请求不启用计时，不回显传入值。计时头只是受控关联标识，不用于身份验证。

合法请求示例响应：

```text
X-Performance-Request-ID: 6d6867ed-c1c2-48b2-bc29-d1c0d3d40466:1
Server-Timing: app;dur=27.410, db;dur=25.390;desc="queries:1"
```

- `app`：进入性能中间件至首次写出响应头的单调时钟耗时，单位毫秒。包含此边界之前的身份校验、服务、仓储和响应序列化工作；不包含客户端输入排队、网络往返、响应体传送或首帧显示。流式响应首次 flush 后的工作也不在此值内。
- `db`：同一请求 context 中已完成的 Bun 查询 hook 时间之和，单位毫秒，包含失败查询。依据 Bun 查询 API 的执行边界，该值可包含连接池等待、驱动通信及部分结果扫描；它不是 PostgreSQL 引擎独占耗时，也不等同于慢查询日志。并发查询时间相加可能超过 `app`。
- `queries:N`：上面的已完成查询数量，仅整数计数。响应头发送后才结束的异步工作不会修改已发出的值。事务管理操作若经 Bun hook 同样计数。

CORS 允许请求中的 `X-Performance-Request-ID`，并暴露同名响应头及 `Server-Timing`，供前端读取并与本地记录对应。没有设置 `Timing-Allow-Origin`：此实现通过 fetch 响应头读取，不承诺跨源 Resource Timing 自动暴露。旧服务端缺少这些头时，前端应记录能力不可用，而不是填入零毫秒。

计时器只保留 duration 和 count，不保留 SQL、参数、token、session ticket、业务正文或凭据。它的存储量与查询次数无关。固定格式响应头不改变任何 v1/v2 响应体、状态码或业务行为。

## 验证

```sh
go test ./internal/infra/performance ./internal/handler/middleware -count=1 -v
go test ./internal/infra/performance -run '^$' -bench 'Benchmark(QueryHookDisabled|BunQueryInstrumentation)' -benchmem -count=5
go test ./... -count=1
go vet ./...
go build ./...
```

中间件集成用例通过 testcontainers 启动隔离 `postgres:18`，执行合成 `SELECT pg_sleep(0.025)`，经真实本地 HTTP 核对关联标识、实际查询计时、查询数、跨源响应头及不变的 JSON 响应；不会读取或修改开发／门店数据库。Docker 不可用时该独立集成用例会明确跳过，不能作为数据库实测通过。

无记录状态仍注册了一个 Bun hook，因此 Bun 会创建查询事件并读时钟；hook 本身只检查 context。基准同时报告无 hook、已注册但无记录、启用记录三种状态，包含事件创建开销。这里使用无 I/O 驱动隔离计时代价，不能据此推导生产请求加速、数据库耗时或门店响应预算。

2026-10-15 的探索性五轮原值保存在 [performance-timing-overhead.txt](performance-timing-overhead.txt)，并保留同内容备份 [performance-timing-overhead-contended.txt](performance-timing-overhead-contended.txt)。该次采集与编译／测试并行，时间波动明显，**不能作为稳定的时间基线**。本次得到的分配计数为：无 hook 32 B／1 次分配，注册但未记录及启用记录均为 192 B／2 次分配；该驱动下注册 hook 增加 160 B／1 次分配。仅运行本 hook 的关闭路径为 0 B／0 次分配。正式比较应在环境空闲、同配置下串行重采五轮并保留这次探索记录。

需要进一步定位 Go 热点时，可以在隔离测试进程中使用 `go test -cpuprofile <本地文件>`，按受控用例分析。生产 HTTP 服务未开放 `/debug/pprof/`；不应为了本地采集把诊断端点直接暴露到门店网络。
