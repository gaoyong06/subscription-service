# 字段重命名总结

## 修改内容

将 `subscription_order` 表的主键字段从 `subscription_order_id` 重命名为 `order_id`

## 修改原因

1. **遵循命名规范**
   - 主键命名规则：`表名_id`（适用于技术性主键）
   - 业务字段命名：根据业务语义，简洁明了（适用于业务性主键）

2. **语义清晰**
   - 在 `subscription_order` 表中，`order_id` 就是"订单号"
   - 简洁明了，符合业界常规，避免冗余

3. **命名规范细化**
   ```
   - 主键（技术性）：表名_id
     例如：user_subscription_id
   
   - 主键（业务性）：业务语义命名
     例如：order_id（订单号）、plan_id（套餐ID）
   
   - 外键：关联表名_id
     例如：user_id、plan_id
   ```

## 修改的文件

### 1. SQL 脚本
**文件**: `docs/sql/subscription_service.sql`

```sql
-- 修改前
CREATE TABLE `subscription_order` (
  `subscription_order_id` varchar(64) NOT NULL COMMENT '订单号',
  ...
  PRIMARY KEY (`subscription_order_id`),
  ...
);

-- 修改后
CREATE TABLE `subscription_order` (
  `order_id` varchar(64) NOT NULL COMMENT '订单号',
  ...
  PRIMARY KEY (`order_id`),
  ...
);
```

### 2. 数据模型
**文件**: `internal/data/subscription.go`

```go
// 修改前
type Order struct {
	ID            string    `gorm:"primaryKey;column:subscription_order_id"`
	...
}

// 修改后
type Order struct {
	ID            string    `gorm:"primaryKey;column:order_id"`
	...
}
```

### 3. 文档
**文件**: `README.md`

```markdown
# 修改前
| `subscription_order` | 订阅订单表 | subscription_order_id |

# 修改后
| `subscription_order` | 订阅订单表 | order_id |
```

```bash
# 修改前
mysql -u root -D subscription_service -e "SELECT * FROM subscription_order WHERE subscription_order_id='订单号';"

# 修改后
mysql -u root -D subscription_service -e "SELECT * FROM subscription_order WHERE order_id='订单号';"
```

## 验证结果

### 1. 代码检查
- ✅ 无 linter 错误
- ✅ 编译成功

### 2. 搜索验证
```bash
# 确认没有遗留的 subscription_order_id
grep -r "subscription_order_id" .
# 结果：无匹配项
```

### 3. 编译验证
```bash
go build -o ./bin/server ./cmd/server
# 结果：编译成功
```

## 数据库迁移

如果数据库已经创建，需要执行以下 SQL 进行迁移：

```sql
-- 重命名字段
ALTER TABLE `subscription_order` 
  CHANGE COLUMN `subscription_order_id` `order_id` varchar(64) NOT NULL COMMENT '订单号';

-- 验证修改
DESC subscription_order;
```

## 影响范围

- ✅ SQL 脚本
- ✅ Go 数据模型
- ✅ 文档
- ✅ 代码编译通过
- ⚠️ 如果数据库已存在，需要执行迁移 SQL

## 总结

本次修改将 `subscription_order_id` 重命名为 `order_id`，使命名更加简洁明了，符合业务语义。修改涉及 3 个文件，已全部完成并验证通过。

---

**修改时间**: 2025-11-26  
**修改状态**: ✅ 完成  
**编译状态**: ✅ 通过

