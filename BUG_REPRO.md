# 成长分析日期边界问题复现说明

## Bug 是什么
成长分析选择结束日期后，报告、导出结果和趋势统计可能继续纳入结束日期之后的养护记录，导致展示范围和用户选择不一致。

## 如何触发
为同一植物准备一条位于所选结束日期当天的养护记录，再准备一条位于次日的记录，提交一个带结束日期的成长分析请求。含 Bug 的实现会把边界之后的记录一并返回，报告记录数因此超过预期。

## 根因
成长分析链路中，`internal/handler/insight_handler.go` 将用户输入的结束日期推进到次日零点，作为查询的排他上界；`internal/service/insight_service.go` 又基于同一日期范围计算趋势和平均值。原实现的 `internal/repository/insight_repo.go` 在 `InsightRepository.ListCareRecords` 中再次向排他上界增加天数，使 SQL 查询范围越过用户选择的结束日期，最终污染记录列表及其派生统计。

Claude 在 `internal/repository/insight_repo.go` 中恢复了排他上界的边界语义，并补充了仓储层回归测试，确保结束日期之后的记录不再进入分析结果。

## 运行指令
```bash
go test ./internal/verification -run '^TestInsightEndDateExcludesFollowingDay$' -count=1 -v
```

## 错误信息
修复前，选择的结束日期会返回 2 条记录，而场景只应保留结束日期当天的 1 条记录。

## 错误堆栈
```text
=== RUN   TestInsightEndDateExcludesFollowingDay
    bug008_insight_date_range_test.go:69: selected end date included 2 records, want 1
--- FAIL: TestInsightEndDateExcludesFollowingDay (0.00s)
FAIL
FAIL	plant-diary/internal/verification	0.867s
FAIL
```
