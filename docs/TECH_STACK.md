# 技术栈使用清单

---

## 一、项目概览

这是一个类似大众点评的本地生活服务平台（**hm-dianping**），包含商户展示、探店博客、秒杀优惠券、用户关注、地理搜索等功能。项目分为三个模块：

| 模块 | 说明 |
|---|---|
| **hm-dianping** | 主后端服务（Spring Boot） |
| **nginx-1.18.0** | 反向代理 + 前端静态资源托管 |
| **Multi-LevelCache** | 多级缓存中间件（Caffeine + Canal） |

---

## 二、后端技术栈 — hm-dianping 模块

### 语言 & 基础框架

| 技术 | 版本 | 用途 | 位置 |
|---|---|---|---|
| **Java 8** | 1.8 | 运行时语言 | `pom.xml:22` |
| **Spring Boot** | 2.3.12.RELEASE | 应用主框架 | `pom.xml:6` |
| **Spring Web (MVC)** | — | RESTful API 控制器 | `controller/*.java` |
| **MyBatis-Plus** | 3.4.3 | ORM / 数据库操作 | `pom.xml:44`, `MybatisConfig.java` |
| **Lombok** | — | 自动生成 getter/setter/构造器 | `pom.xml:36` |
| **Hutool** | 5.7.17 | 工具库（JSON、Bean 拷贝、UUID 等） | `pom.xml:50` |

### 数据库 & 缓存

| 技术 | 版本 | 用途 | 位置 |
|---|---|---|---|
| **MySQL** | 5.1.47 (驱动) | 关系型数据库（主存储） | `pom.xml:32`, `application.yaml:14` |
| **Redis** | — | 缓存、分布式锁、点赞/关注/签到/消息队列 | `pom.xml:15`, `RedisConfig.java` |
| **Redisson** | 3.13.6 | Redis 分布式锁框架（可重入锁） | `pom.xml:57`, `RedisConfig.java:14` |
| **Lettuce** | (Spring Boot 内置) | Redis 客户端连接池 | `application.yaml:22-27` |

#### Redis 使用场景详表

| 场景 | Key 前缀 | 数据结构 | 文件 |
|---|---|---|---|
| 登录用户 token | `login:token:` | Hash | `LoginInterceptor.java:27` |
| 验证码 | `login:code:` | String | `RedisConstants.java` |
| 缓存空值（防缓存穿透） | — | String (空串) | `ShopServiceImpl.java:161` |
| 商铺缓存 | `cache:shop:` | String (JSON) | `CacheClient.java`, `ShopServiceImpl.java` |
| 店铺类型缓存 | `cache:shop:type:` | String | `RedisConstants.java` |
| 分布式锁 | `lock:shop:` / `lock:` | String (SET NX) | `SimpleRedisLock.java:38` |
| 秒杀库存 | `seckill:stock:` | String | `seckill.lua:10` |
| 秒杀下单集合 | `seckill:order:` | Set | `seckill.lua:13` |
| 博客点赞 | `blog:liked:` | ZSet (时间戳排序) | `BlogServiceImpl.java:79` |
| 关注列表 | `follows:` | Set | `FollowServiceImpl.java:39` |
| 推送收件箱 | `feed:` | ZSet | `BlogServiceImpl.java:149` |
| GEO 店铺 | `shop:geo:` | Geo | `RedisConstants.java:17` |
| 用户签到 | `sign:` | Bitmap | `RedisConstants.java:18` |

### 消息队列

| 技术 | 版本 | 用途 | 位置 |
|---|---|---|---|
| **Apache Kafka** | (spring-kafka 内嵌) | 秒杀订单异步处理 | `pom.xml:63`, `KafkaConfig.java` |
| **Redis Stream** | — | 原有方案（已注释，改用 Kafka） | `VoucherOrderServiceImpl.java` 注释区 |

#### Kafka 配置

- **Topic**: `kafka-orders`（3 分区，3 副本） — `KafkaConfig.java:12`
- **Consumer Group**: `my-kafka-group`，并发 `concurrency = 3` — `VoucherOrderServiceImpl.java:142`
- **Brokers**: `localhost:9094,9095,9096` — `application.yaml:3`

### 中间件 & 基础设施

| 技术 | 用途 | 位置 |
|---|---|---|
| **Nginx 1.18.0** | 反向代理（`/api` → `localhost:8081`）+ 前端静态资源托管 | `nginx-1.18.0/conf/nginx.conf` |
| **Upstream 负载均衡** | 后端双节点 `8081` / `8082` | `nginx.conf:38-40` |
| **阿里云 OSS** | 图片/文件上传存储 | `AliOssUtil.java`, `CommonConfig.java` |
| **Spring AOP (AspectJ)** | AOP 代理暴露（事务/秒杀） | `HmDianPingApplication.java:14` |
| **Spring Actuator** | 健康检查/监控 | `pom.xml:10` |

### 安全 & 拦截

| 技术 | 用途 | 位置 |
|---|---|---|
| **Interceptor (HandlerInterceptor)** | 登录鉴权（Token 校验） | `LoginInterceptor.java`, `MvcConfig.java` |
| **ThreadLocal (UserHolder)** | 请求级用户上下文 | `UserHolder.java` |
| **全局异常处理** | `@RestControllerAdvice` 统一异常捕获 | `WebExceptionAdvice.java` |

### Lua 脚本（保证原子性）

| 脚本 | 用途 | 位置 |
|---|---|---|
| `seckill.lua` | 秒杀：库存检查 + 一人一单 + 扣库存 + 订单入队 | `resources/seckill.lua` |
| `unlock.lua` | 分布式锁：安全释放（校验线程标识后 DEL） | `resources/unlock.lua` |

### 分布式锁演进（秒杀模块）

```
1. SimpleRedisLock (SET NX + lua unlock)      → 自定义实现
2. Redisson (RLock 可重入锁)                   → VoucherOrderServiceImpl.java
```

### 缓存策略（商铺查询）

```
1. 缓存穿透 → 缓存空值（短 TTL）
2. 缓存击穿 → 互斥锁（queryWithMutex）
3. 缓存击穿 → 逻辑过期（queryWithLogicExpire）+ 独立线程池重建
4. 统一封装  → CacheClient 工具类（Factory Method 模式）
```

---

## 三、Multi-LevelCache 模块

| 技术 | 用途 | 位置 |
|---|---|---|
| **Spring Boot 4.0.6** | 应用容器 | `pom.xml:6` |
| **Java 17** | 运行语言 | `pom.xml:20` |
| **Caffeine 3.2.4** | 本地（一级）缓存 | `pom.xml:35` |
| **Canal 1.2.1-RELEASE** | MySQL binlog 监听 → 缓存失效/更新 | `pom.xml:40`, `Canal.java` |
| **Lombok** | 简化代码 | `pom.xml:28` |

> 该模块定位为**多级缓存中间件**：Caffeine 做 JVM 内缓存，Canal 监听 MySQL binlog 变更以驱逐/刷新缓存。目前 `Canal.java` 的三个 `EntryHandler` 方法（insert/update/delete）为骨架实现，尚未填充业务逻辑。

---

## 四、前端技术栈

| 技术 | 用途 | 位置 |
|---|---|---|
| **Vue.js 2.x** | 前端 SPA 框架 | `js/vue.js` |
| **Axios** | HTTP 请求库 | `js/axios.min.js` |
| **HTML5** | 页面结构 | `*.html`（9 个页面） |
| **CSS3** | 样式 | `css/` |

### 前端页面清单

| 页面 | 用途 |
|---|---|
| `login.html` / `login2.html` | 登录 |
| `index.html` | 首页 |
| `shop-detail.html` | 商铺详情 |
| `shop-list.html` | 商铺列表 |
| `blog-detail.html` | 博客详情 |
| `blog-edit.html` | 编辑博客 |
| `info.html` / `info-edit.html` | 用户信息 |
| `other-info.html` | 他人主页 |

### HTTP 请求流程

```
前端 Axios → /api → Nginx 反向代理 → http://localhost:8081 → Spring Boot 后端
```

---

## 五、数据库表结构（MySQL: hmdp）

| 表 | 说明 |
|---|---|
| `tb_user` | 用户 |
| `tb_user_info` | 用户详细信息 |
| `tb_shop` | 商户 |
| `tb_shop_type` | 商户分类 |
| `tb_blog` | 探店博客 |
| `tb_blog_comments` | 博客评论 |
| `tb_follow` | 用户关注 |
| `tb_voucher` | 优惠券 |
| `tb_seckill_voucher` | 秒杀优惠券（含开始/结束时间+库存） |
| `tb_voucher_order` | 优惠券订单 |

---

## 六、整体架构图

```
┌──────────────────────────────────────────────────┐
│                  用户浏览器                        │
│         Vue.js SPA (HTML + Axios)                │
└─────────────────┬────────────────────────────────┘
                  │ HTTP (port 8080)
                  ▼
┌──────────────────────────────────────────────────┐
│             Nginx 1.18.0 (反向代理)               │
│    / → html/hmdp (静态资源)                       │
│    /api → proxy_pass → localhost:8081            │
│    upstream: 8081 / 8082 (负载均衡)               │
└─────────────────┬────────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────────┐
│          Spring Boot 2.3.12 (hm-dianping)        │
│                                                  │
│  ┌──────────┐ ┌──────────┐ ┌─────────────────┐  │
│  │ Controller│ │ Service  │ │ Interceptor     │  │
│  │  10 个    │ │  10 个   │ │ LoginInterceptor│  │
│  └────┬─────┘ └────┬─────┘ └─────────────────┘  │
│       │            │                             │
│  ┌────▼────────────▼─────┐ ┌─────────────────┐  │
│  │ MyBatis-Plus (Mapper) │ │   Redis / Redisson│  │
│  │  10 Mapper + XML      │ │  Lua / Stream     │  │
│  └────┬──────────────────┘ └────────┬──────────┘  │
│       │                             │             │
│  ┌────▼─────┐              ┌────────▼─────────┐  │
│  │  MySQL   │              │     Kafka         │  │
│  │ (hmdp)   │              │  topic: orders    │  │
│  └──────────┘              └──────────────────┘  │
│                                                  │
│  ┌──────────────────────────────┐                │
│  │  阿里云 OSS (图片存储)        │                │
│  └──────────────────────────────┘                │
└──────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────┐
│     Multi-LevelCache (Spring Boot 4.0.6)         │
│   ┌────────────┐   ┌─────────────────────────┐   │
│   │ Caffeine   │   │ Canal (MySQL binlog)    │   │
│   │ (本地缓存)  │   │ → 缓存失效/更新          │   │
│   └────────────┘   └─────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

---

## 七、关键业务时序 — 秒杀流程

```
用户点击秒杀
    │
    ▼
VoucherOrderController.orderSeckillVoucher()
    │
    ▼
VoucherOrderServiceImpl.orderSeckillVoucher()
    │
    ├─ 执行 seckill.lua (Redis 原子操作)
    │    ├─ 检查库存 (GET stock > 0)
    │    ├─ 检查一人一单 (SISMEMBER order set)
    │    ├─ 扣库存 (INCRBY stock -1)
    │    ├─ 记录用户 (SADD order set)
    │    └─ XADD 订单到 stream.orders
    │
    ├─ result = 1 → "库存不足"
    ├─ result = 2 → "已抢过"
    │
    └─ result = 0 → Kafka 发送订单消息
         │
         ▼
    @KafkaListener 消费 (3 个并发线程)
         │
         ▼
    createVoucherOrder() [@Transactional]
         ├─ 一人一单校验
         ├─ 乐观锁扣库存 (stock > 0)
         └─ 保存订单到 MySQL
```
