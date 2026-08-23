# 养护提交事务一致性问题复现说明

## Bug 是什么
养护接口同时写入养护记录和养护周期时，其中一个写入失败，接口仍可能返回成功，并留下只写入一部分的数据。后续历史记录和待办周期会继续使用这份不完整状态。

## 如何触发
提交一条带周期更新的养护记录，并让周期写入在数据库层失败。旧实现会先留下养护记录，再报告“保存成功”，而周期写入并未完成。

## 根因
`internal/repository/care_repo.go` 的 `CareRepository.Transaction` 没有把 GORM 提供的事务连接 `tx` 包装给回调，而是继续使用原始数据库连接，导致回调内的写操作绕开事务。`internal/service/plant_service.go` 的 `PlantService.AddCare` 还直接使用服务持有的仓储写入记录，没有使用事务回调传入的仓储。即使数据库事务本身返回错误，`internal/handler/api_handler.go` 的 `APIHandler.AddCare` 仍然丢弃服务错误并返回成功状态。

这是状态一致性失效：记录写入、周期写入和 HTTP 响应没有共享同一失败边界，任何一步失败都可能留下孤儿养护记录并继续污染后续提醒。Claude 已让事务仓储使用 `tx`、让服务层所有写入走事务仓储，并让 handler 根据错误返回失败响应。

## 运行指令
```bash
go test ./internal/verification -run '^TestCareAPIIsAtomicWhenCycleWriteFails$' -count=1 -v
```

## 错误信息
周期写入失败后，养护接口仍返回“养护记录已保存”，同时数据库中已经留下不应存在的养护记录。

## 错误堆栈
```text
=== RUN   TestCareAPIIsAtomicWhenCycleWriteFails
care_repo.go:39 cycle interval too long
bug005_transaction_atomicity_test.go:55: care API reported success after the cycle write failed: {"date":"2026-08-23","message":"养护记录已保存"}
--- FAIL: TestCareAPIIsAtomicWhenCycleWriteFails (0.00s)
FAIL
```

