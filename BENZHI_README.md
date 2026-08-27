# zhiwu-care-system

一句话简介：面向家庭植物爱好者的植物养护日记系统，用于记录植物档案、浇水施肥等养护操作、成长照片和提醒分析。

项目用途：zhiwu-care-system 是一个可在容器中构建、运行和测试的 Go 服务或工具。项目源代码、依赖描述和评测专用 Docker 文件共同构成自包含任务；不依赖本机预编译二进制。

## 标准构建、运行和测试命令

```bash
go build ./...
go run ./cmd/server
go test ./...
```
## 评测容器

评测专用 Dockerfile 为 `benzhi.Dockerfile`，构建脚本为 `build_benzhi_docker.sh`。

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh my-go-task linux/arm64
./build_benzhi_docker.sh my-go-task linux/amd64
docker run -it my-go-task:latest
```
