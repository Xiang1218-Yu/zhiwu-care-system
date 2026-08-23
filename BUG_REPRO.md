# 上传照片生命周期问题复现说明

## Bug 是什么
养护记录携带照片提交后，数据库记录已经保存，但照片资源无法继续访问；提交失败时，上传目录中的临时文件也可能失去数据库引用，形成后续难以清理的孤儿文件。

## 如何触发
通过网页养护记录表单上传一张照片并成功保存，然后从时间线或成长相册读取这条记录。若让保存过程失败，再检查上传目录，会看到成功与失败两条路径的文件生命周期都不符合预期。

## 根因
`internal/handler/view.go` 的 `ViewHandler.AddCare` 使用无条件的延迟删除逻辑，请求成功返回时也会调用删除操作，导致数据库中的照片地址指向已经被删除的文件。`internal/service/plant_service.go` 的 `PlantService.AddCare` 又把 `/uploads/` 前缀从照片地址中裁掉，保存的裸文件名与模板和静态资源路由使用的完整地址不一致。`pkg/utils/file.go` 中的 `utils.DeleteUploadAfterUse` 进一步固化了“用完即删”的错误语义。

这属于资源生命周期失效：成功提交应保留文件，失败提交才应清理未被引用的临时文件，同时数据库中的地址必须与实际静态资源入口保持一致。Claude 已将清理动作限制到失败分支，保留完整 URL，并移除误导性的辅助函数。

## 运行指令
```bash
go test ./internal/verification -run '^TestSuccessfulCarePhotoRemainsAvailable$' -count=1 -v
```

## 错误信息
成功保存的养护记录仍然引用照片，但上传目录中找不到对应文件，时间线和成长相册因此无法加载图片。

## 错误堆栈
```text
=== RUN   TestSuccessfulCarePhotoRemainsAvailable
bug003_upload_lifecycle_test.go:66: photo referenced by a successful care record is unavailable: stat /tmp/upload/2b4f7430-64e5-4fec-bd99-8203b5d78d69.png: no such file or directory
--- FAIL: TestSuccessfulCarePhotoRemainsAvailable (0.01s)
FAIL
```

