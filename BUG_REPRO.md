# 周期更新问题复现说明

## Bug 是什么
同一盆植物的同一种养护周期先后更新时，系统没有稳定地维护当前周期。更新后的数据可能留下旧记录，首页待办因此出现重复或过期提醒，周期历史和当前配置也会失去一致性。

## 如何触发
对同一植物的浇水周期连续提交两次不同的间隔，随后读取周期记录和待办数据。并发提交或重复保存时，旧实现可能同时走过“读取后再删除并插入”的路径。

## 根因
`internal/model/care_cycle.go` 的周期模型没有为 `plant_id` 与 `type` 建立唯一约束，`internal/repository/care_repo.go` 的 `CareRepository.UpsertCycle` 又采用非原子的读取、删除、插入流程。多个写入同时未读到已有记录时，会各自插入一行，导致同一业务周期产生多个数据库实体。待办查询按记录逐行读取，所以过期周期仍会继续展示。

Claude 的修复在模型层补充周期唯一性，在仓储层处理并发插入冲突并更新已有记录，同时在 `cmd/server/main.go` 的启动迁移流程中兼容历史重复数据，先完成去重再建立约束。

## 运行指令
```bash
go test ./internal/verification -run '^TestCycleUpdateKeepsCurrentCycleIdentity$' -count=1 -v
```

## 错误信息
修复前连续更新同一周期后，第二次响应生成了不同的周期 ID，数据库中也可能保留多条同类型周期。

## 错误堆栈
```text
=== RUN   TestCycleUpdateKeepsCurrentCycleIdentity
care_repo.go:31 record not found
bug001_cycle_update_test.go:46: updating one cycle changed its identity: first="57a7224a-77f0-424d-a6a3-b4f193b40f04" second="12bcd1e3-0ff5-4fc5-8500-9a4c66fe24a7"
--- FAIL: TestCycleUpdateKeepsCurrentCycleIdentity (0.00s)
FAIL
```

