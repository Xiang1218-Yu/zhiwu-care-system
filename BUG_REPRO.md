# 统计错误传播问题复现说明

## Bug 是什么
统计看板读取近期养护次数时，存储查询失败没有转化为接口错误，返回结果仍然像正常统计一样包含零值。调用方无法区分“确实没有养护记录”和“查询过程已经失败”。

## 如何触发
读取统计接口的近期养护数据。在默认 SQLite 表结构下，近期养护计数使用了错误的表名，查询返回存储层错误；服务层和接口层随后继续组装统计结果并返回成功响应。

## 根因
根因涉及三处生产 Go 代码。`internal/repository/care_repo.go` 的 `CareRepository.CountLogsAfter` 使用了单数表名 `care_log`，而实际模型表名和连接条件使用复数 `care_logs`，因此查询直接失败。`internal/service/stats_service.go` 的 `StatsService.Get` 用空白接收变量丢弃了近期养护计数的错误，后续返回值又可能被其他查询的结果覆盖。`internal/handler/api_handler.go` 的 `APIHandler.Stats` 再次丢弃服务错误并固定返回 HTTP 200，最终把存储失败伪装成 `RecentCareLogs=0`。

这属于跨层错误传播失效：数据库错误没有沿仓储、服务、HTTP handler 链路继续传递，统计结果和调用方的错误语义因此被污染。Claude 未修改生产代码，按题面完成了调用链排查和影响说明。

## 运行指令
```bash
go test ./internal/verification -run '^TestStatsAPIPropagatesRecentLogQueryError$' -count=1 -v
```

## 错误信息
统计接口实际返回 HTTP 200 和 `RecentCareLogs=0`，而验证场景预期存储失败应暴露为 HTTP 500。

## 错误堆栈
```text
=== RUN   TestStatsAPIPropagatesRecentLogQueryError
care_repo.go:66 no such table: care_log
bug002_stats_error_test.go:47: expected stats API to expose the storage error with HTTP 500, got 200: {"TotalPlants":1,"Healthy":1,"Yellowing":0,"Pests":0,"Gone":0,"Dead":0,"RecentCareLogs":0,"MonthlyNewPlants":1}
--- FAIL: TestStatsAPIPropagatesRecentLogQueryError (0.00s)
FAIL
```

