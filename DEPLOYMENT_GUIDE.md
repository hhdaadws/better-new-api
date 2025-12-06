# 按月套餐系统 - 快速部署指南

## ✅ 已完成的所有实施

### 后端完整实现 ✅

1. ✅ 数据库迁移脚本：`bin/migration_add_subscriptions.sql`
2. ✅ 模型层：`model/subscription.go`
3. ✅ 服务层：`service/subscription_quota.go`
4. ✅ 控制器层：`controller/subscription.go`
5. ✅ 兑换码系统集成：`model/redemption.go`
6. ✅ 配额消费逻辑：`service/quota.go`
7. ✅ 常量定义：`common/constants.go`
8. ✅ API 路由：`router/api-router.go`
9. ✅ 数据库初始化：`model/main.go`
10. ✅ 定时任务：`main.go`

---

## 🚀 部署步骤（仅需 5 步）

### 1. 备份数据库
```bash
mysqldump -u your_user -p your_database > backup_$(date +%Y%m%d).sql
```

### 2. 停止服务
```bash
systemctl stop new-api
# 或者
pkill new-api
```

### 3. 运行数据库迁移
```bash
cd better-new-api

# 登录数据库
mysql -u your_user -p your_database

# 在 MySQL 提示符中执行
source bin/migration_add_subscriptions.sql

# 或直接执行
mysql -u your_user -p your_database < bin/migration_add_subscriptions.sql
```

### 4. 编译项目
```bash
cd better-new-api

# 编译后端
go build -o new-api .

# 如果需要编译前端（如果你修改了前端）
cd web
npm run build
cd ..
go build -o new-api .
```

### 5. 启动服务
```bash
# 使用 systemd
systemctl start new-api

# 或直接运行
./new-api

# 查看日志确认启动成功
tail -f logs/new-api.log
```

---

## 📋 功能验证

### 测试 API 可用性

```bash
# 1. 检查订阅套餐接口（需要管理员权限）
curl -X GET "http://localhost:3000/api/subscription/" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"

# 2. 创建测试套餐
curl -X POST "http://localhost:3000/api/subscription/" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试月卡",
    "description": "测试订阅套餐",
    "daily_quota_limit": 100000,
    "weekly_quota_limit": 500000,
    "monthly_quota_limit": 2000000,
    "allowed_groups": "[\"default\"]",
    "duration_days": 30,
    "status": 1
  }'

# 3. 生成兑换码
curl -X POST "http://localhost:3000/api/subscription/redemption" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试兑换码",
    "subscription_id": 1,
    "count": 1,
    "expired_time": 0
  }'
```

---

## 🎯 使用流程示例

### 管理员操作

1. **登录管理后台** → 访问 `/subscription` 页面（前端页面待实现）
2. **创建套餐**：
   - 名称：基础月卡
   - 每日限额：100,000 tokens
   - 每周限额：500,000 tokens
   - 每月限额：2,000,000 tokens
   - 允许分组：["default", "premium"]
   - 有效期：30 天

3. **生成兑换码**：
   - 点击套餐的"生成兑换码"按钮
   - 设置数量：10 个
   - 设置过期时间：30 天后
   - 复制生成的兑换码

4. **分发兑换码** → 通过邮件/群组分发给用户

### 用户操作

1. **兑换订阅**：
   - 访问充值页面
   - 输入兑换码：`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
   - 点击兑换
   - 系统提示：成功兑换订阅套餐"基础月卡"

2. **查看订阅**（前端页面待实现）：
   - 访问 `/my-subscription`
   - 查看：
     - 套餐名称：基础月卡
     - 到期时间：2025-01-06
     - 今日已用：25,000 / 100,000
     - 本周已用：150,000 / 500,000
     - 本月已用：600,000 / 2,000,000

3. **使用 API**：
   - 正常调用 API
   - 系统自动优先使用订阅额度
   - 订阅额度不足时自动降级到普通充值额度

---

## 🔍 数据库验证

### 检查表是否创建成功

```sql
-- 查看订阅套餐表
SHOW TABLES LIKE 'subscriptions';
DESC subscriptions;
SELECT * FROM subscriptions;

-- 查看用户订阅表
SHOW TABLES LIKE 'user_subscriptions';
DESC user_subscriptions;
SELECT * FROM user_subscriptions;

-- 查看订阅日志表
SHOW TABLES LIKE 'subscription_logs';
DESC subscription_logs;
SELECT * FROM subscription_logs;

-- 检查兑换码表是否添加了新字段
DESC redemptions;
SELECT id, name, type, subscription_id FROM redemptions WHERE type = 2;
```

### 查看订阅使用情况

```sql
-- 查看所有激活的订阅
SELECT
    us.id,
    u.username,
    s.name AS subscription_name,
    us.daily_quota_used,
    us.weekly_quota_used,
    us.monthly_quota_used,
    FROM_UNIXTIME(us.expire_time) AS expire_time
FROM user_subscriptions us
JOIN users u ON us.user_id = u.id
JOIN subscriptions s ON us.subscription_id = s.id
WHERE us.status = 1;

-- 查看订阅消费日志（最近 100 条）
SELECT
    sl.id,
    u.username,
    sl.quota_used,
    sl.model_name,
    FROM_UNIXTIME(sl.created_time) AS created_time
FROM subscription_logs sl
JOIN users u ON sl.user_id = u.id
ORDER BY sl.id DESC
LIMIT 100;
```

---

## 📊 Redis 缓存验证

```bash
# 连接 Redis
redis-cli

# 查看用户订阅缓存
keys user_subscription:*
get user_subscription:1

# 查看订阅额度缓存
keys subscription_quota:*
hgetall subscription_quota:1

# 清除所有订阅缓存（测试用）
del user_subscription:*
del subscription_quota:*
```

---

## ⚙️ 系统配置

### 环境变量（可选）

在 `.env` 文件中可以添加（当前使用默认值）：

```bash
# Redis 配置（如果启用了 Redis）
REDIS_CONN_STRING=redis://localhost:6379

# 批量更新配置
BATCH_UPDATE_ENABLED=true
BATCH_UPDATE_INTERVAL=5
```

---

## 🐛 故障排查

### 问题 1：API 返回 404

**原因**：路由未正确注册

**解决**：
```bash
# 检查编译是否成功
go build -o new-api .

# 查看日志
tail -f logs/new-api.log | grep subscription
```

### 问题 2：数据库表不存在

**原因**：迁移脚本未执行或执行失败

**解决**：
```sql
-- 检查表
SHOW TABLES LIKE '%subscription%';

-- 如果不存在，重新执行迁移
source bin/migration_add_subscriptions.sql
```

### 问题 3：兑换订阅码失败

**原因**：套餐被禁用或兑换码已使用

**解决**：
```sql
-- 检查套餐状态
SELECT id, name, status FROM subscriptions WHERE id = 1;

-- 检查兑换码状态
SELECT id, name, status, type, subscription_id FROM redemptions WHERE `key` = 'your-key';

-- 如果需要重置兑换码
UPDATE redemptions SET status = 1 WHERE id = 1;
```

### 问题 4：订阅额度未生效

**原因**：
1. 订阅已过期
2. 订阅不支持当前分组
3. Redis 缓存问题

**解决**：
```sql
-- 检查订阅状态
SELECT * FROM user_subscriptions WHERE user_id = 1;

-- 检查套餐分组
SELECT id, name, allowed_groups FROM subscriptions WHERE id = 1;

-- 清除 Redis 缓存
redis-cli del user_subscription:1
```

### 问题 5：额度未重置

**原因**：定时任务未运行或服务器时区错误

**解决**：
```bash
# 检查服务器时区
date
timedatectl

# 检查日志是否有定时任务运行记录
tail -f logs/new-api.log | grep subscription

# 手动触发一次过期检查（在数据库中）
UPDATE user_subscriptions SET status = 2 WHERE expire_time <= UNIX_TIMESTAMP();
```

---

## 📈 性能监控

### 监控 Redis 性能

```bash
# 查看 Redis 命中率
redis-cli info stats | grep keyspace

# 查看订阅相关的键数量
redis-cli keys "user_subscription:*" | wc -l
redis-cli keys "subscription_quota:*" | wc -l
```

### 监控数据库性能

```sql
-- 查看慢查询
SHOW FULL PROCESSLIST;

-- 查看订阅相关表的大小
SELECT
    table_name,
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS size_mb
FROM information_schema.TABLES
WHERE table_name IN ('subscriptions', 'user_subscriptions', 'subscription_logs')
    AND table_schema = DATABASE();

-- 查看订阅日志增长情况
SELECT
    DATE(FROM_UNIXTIME(created_time)) AS date,
    COUNT(*) AS log_count,
    SUM(quota_used) AS total_quota
FROM subscription_logs
GROUP BY DATE(FROM_UNIXTIME(created_time))
ORDER BY date DESC
LIMIT 30;
```

---

## 🎉 部署完成检查清单

- [ ] 数据库迁移成功（三张表创建成功）
- [ ] 后端编译成功（无编译错误）
- [ ] 服务启动成功（日志无报错）
- [ ] API 可以访问（测试接口返回正常）
- [ ] 可以创建订阅套餐
- [ ] 可以生成订阅兑换码
- [ ] 可以成功兑换订阅
- [ ] 订阅额度正常消费
- [ ] Redis 缓存工作正常（如果启用）
- [ ] 定时任务正常运行

---

## 📞 技术支持

### 日志位置
- 应用日志：`logs/new-api.log`
- 系统日志：`journalctl -u new-api -f`

### 查看实时日志
```bash
# 查看所有日志
tail -f logs/new-api.log

# 只查看订阅相关日志
tail -f logs/new-api.log | grep -i subscription

# 查看错误日志
tail -f logs/new-api.log | grep -i error
```

### 常用命令

```bash
# 重启服务
systemctl restart new-api

# 查看服务状态
systemctl status new-api

# 查看进程
ps aux | grep new-api

# 查看端口占用
netstat -tlnp | grep 3000
```

---

## 🎯 下一步：前端实现

后端已完全实现，接下来可以实现前端界面：

1. **订阅管理页面**（管理员）
   - 路径：`web/src/pages/Subscription/index.jsx`
   - 功能：创建、编辑、删除套餐，生成兑换码

2. **我的订阅页面**（用户）
   - 路径：`web/src/pages/MySubscription/index.jsx`
   - 功能：查看订阅状态、额度使用、到期时间

3. **兑换码页面增强**
   - 修改：`web/src/components/table/redemptions/`
   - 功能：显示订阅类型兑换码

详细的前端实现方案请参考 `SUBSCRIPTION_IMPLEMENTATION.md`

---

**部署成功！🎊**
