# Go 项目架构教学

本项目已从 Spring Boot 单模块改为标准 Go 单体服务结构，采用 `cmd/server + internal/*` 的常见架构模式。

## 目录职责

```text
cmd/server/main.go          服务启动入口
config/config.yaml          本地默认配置
internal/app                组装 DB、Redis、Kafka、Router
internal/config             Viper 配置加载
internal/constants          Redis Key、分页、TTL 等常量
internal/ctx                Gin Context 中的当前登录用户
internal/dto                API 请求/响应 DTO
internal/handler            Gin Handler，负责 HTTP 入参和响应
internal/middleware         认证、恢复、通用中间件
internal/model              GORM 数据模型
internal/router             Gin 路由注册
internal/script             go:embed 嵌入 Lua 脚本
internal/service            核心业务逻辑
migrations                  golang-migrate 数据库迁移脚本
docs                        教学和迁移文档
```

## 为什么使用这个结构

`cmd/server` 只做启动，不承载业务逻辑，方便后续增加 `cmd/worker`、`cmd/migrate` 等独立进程。

`internal` 是 Go 官方推荐的私有代码目录，外部模块不能直接 import，适合放服务内部实现。

`handler -> service -> model` 保持清晰分层。Handler 只处理 HTTP，Service 负责事务、缓存、Kafka、Redis Lua，Model 只描述数据库表结构。

`app.New()` 是依赖组装点，集中创建 GORM、go-redis、kafka-go、Service、Handler、Router，避免在业务代码里到处初始化连接。

## 请求流程

```text
浏览器/Nginx
  -> Gin Router
  -> Middleware(Auth/Recovery/CORS)
  -> Handler
  -> Service
  -> GORM / Redis / Kafka
  -> Result JSON
```

## 响应兼容策略

Go 版继续使用旧 Java 项目的响应格式：

```json
{
  "success": true,
  "data": {}
}
```

失败响应：

```json
{
  "success": false,
  "errorMsg": "错误信息"
}
```

这样前端可以尽量少改代码。

## 路由兼容策略

旧项目存在 `/shop/{id}`、`/shop/of/type` 这种同层级静态路径和通配路径并存的写法。Gin 的路由树不允许直接同时注册冲突路径，所以 Go 版把这类通配接口放到 `NoRoute` 里兜底分发。

当前兼容的兜底接口包括：

```text
GET /shop/{id}
GET /blog/{id}
GET /user/{id}
PUT /follow/{id}/{isFollow}
```

## 配置策略

使用 Viper 读取 `config/config.yaml`，并支持环境变量覆盖。例如：

```powershell
$env:HMDP_MYSQL_DSN="root:123456@tcp(127.0.0.1:3306)/hmdp?charset=utf8mb4&parseTime=True&loc=Local"
$env:HMDP_REDIS_ADDR="127.0.0.1:6379"
```

## 日志策略

用户明确不接受 zap，所以项目使用 Go 标准库 `log` 和 Gin 默认访问日志。后续如果需要结构化日志，可以再替换为 `slog`。
