# Subscription Service 前端功能分析

## 数据库表结构

根据 `subscription_service.sql`，数据库包含以下5张表：

1. **plan** - 订阅套餐表
2. **plan_pricing** - 套餐区域定价表
3. **user_subscription** - 用户订阅表
4. **subscription_order** - 订阅订单表
5. **subscription_history** - 订阅历史记录表

## 后端 API 功能

根据 `subscription.proto`，后端提供以下 API：

### 套餐管理（✅ 前端已实现）
- ✅ `ListPlans` - 获取套餐列表
- ✅ `CreatePlan` - 创建套餐
- ✅ `UpdatePlan` - 更新套餐
- ✅ `DeletePlan` - 删除套餐

### 区域定价管理（✅ 前端已实现）
- ✅ `ListPlanPricings` - 获取区域定价列表
- ✅ `CreatePlanPricing` - 创建区域定价
- ✅ `UpdatePlanPricing` - 更新区域定价
- ✅ `DeletePlanPricing` - 删除区域定价

### 用户订阅管理（⚠️ 需要明确视角）

**重要说明：**
- `dev-share-web` 是**管理员后台**，不是用户视角
- 用户视角的功能（用户查看自己的订阅、取消订阅等）由开发者在自己的 app 产品中实现
- 管理员后台需要的是**管理用户订阅**的功能

**用户视角 API（供开发者使用，不在 dev-share-web 中实现）：**
- `GetOrEnsureMySubscription`（HTTP: `GET /subscription/v1/my/{userId}`）- 获取或初始化我的订阅
- `CancelSubscription` - 取消订阅
- `PauseSubscription` - 暂停订阅
- `ResumeSubscription` - 恢复订阅
- `SetAutoRenew` - 设置自动续费

**管理员视角需求（需要在 dev-share-web 中实现）：**
- ❌ 查看某个应用的订阅用户列表
- ❌ 查看某个用户的订阅状态（管理员查看）
- ❌ 管理员操作：暂停/恢复/取消用户的订阅（如果需要）
- ❌ 查看订阅统计（活跃订阅数、过期订阅数等）

### 订阅订单管理（❌ 需要分析是否需要）

**当前情况：**
- ✅ `CreateSubscriptionOrder` - 创建订阅订单（API 已实现，供开发者使用）
- ✅ `GetOrder` - 获取单个订单详情（后端已实现）
- ❌ `ListSubscriptionOrders` - 订单列表查询（后端未实现，前端也没有）

**与支付交易记录的关系：**
- `AppRevenue` 组件显示的是 `payment-service` 的所有支付交易记录
- 支付交易记录包括：订阅订单、充值订单、其他支付订单等
- 订阅订单会同步到 `payment-service`，但无法在支付交易记录中区分订阅订单

**订阅订单的独特性：**
- 关联 `plan_id`（套餐ID）- 可以知道用户购买的是哪个套餐
- 订阅相关的支付状态（pending/success/failed/closed/refunded/partially_refunded）
- 可以关联到用户订阅状态（`user_subscription` 表）

**是否需要订阅订单管理功能？**

**需要的情况：**
1. ✅ **业务分析需求** - 管理员需要分析：
   - 哪些套餐卖得好（按套餐统计订单数、收入）
   - 订阅订单的转化率（创建订单数 vs 支付成功数）
   - 按套餐筛选订单，查看特定套餐的销售情况

2. ✅ **订阅业务管理** - 管理员需要：
   - 查看订阅订单详情（关联的套餐信息）
   - 了解哪些用户购买了哪些套餐
   - 查看订阅订单的支付状态，处理失败的订单

3. ✅ **数据完整性** - 订阅订单表有独立的数据结构：
   - `plan_id` 关联套餐信息
   - `app_id` 便于按应用查询
   - 订阅订单状态与支付交易状态可能有差异

**不需要的情况：**
- 如果管理员只需要查看总收入，`AppRevenue` 已经足够
- 如果不需要按套餐分析业务，可以不需要

**建议：**
- ⚠️ **建议添加** - 订阅订单管理功能对订阅业务分析很有价值
- 可以按应用、按套餐、按用户、按状态筛选订阅订单
- 可以查看订阅订单详情，包括关联的套餐信息
- 有助于分析订阅业务的健康度

### 订阅历史记录（⚠️ 需要明确视角）

**用户视角 API（供开发者使用）：**
- `GetSubscriptionHistory` - 获取订阅历史记录（按 userId 查询）

**管理员视角需求（需要在 dev-share-web 中实现）：**
- ❌ 查看某个应用的订阅历史记录列表（按 appId 查询）
- ❌ 查看某个用户的订阅历史记录（管理员查看）
- ❌ 按操作类型筛选（created/renewed/upgraded/paused等）
- ❌ 按时间范围筛选

## 前端当前实现情况

### ✅ 已实现的功能

1. **套餐管理页面**
   - 路径：`/dashboard/apps/[appId]?tab=subscriptions`
   - 组件：`SubscriptionPlans`
   - 功能：
     - 查看套餐列表
     - 创建套餐：`/dashboard/apps/[appId]/subscriptions/create`
     - 编辑套餐：`/dashboard/apps/[appId]/subscriptions/[planId]/edit`
     - 删除套餐

2. **区域定价管理**
   - 在编辑套餐页面中：`PlanRegionalPricing` 组件
   - 功能：添加、编辑、删除区域定价

### ❌ 缺失的功能

1. **用户订阅管理页面（管理员视角）**
   - 需要添加：查看和管理应用的订阅用户
   - 建议路径：`/dashboard/apps/[appId]/subscriptions/users`
   - 功能需求：
     - 查看应用的订阅用户列表（按 appId 查询）
     - 查看每个用户的订阅状态（active/expired/paused/cancelled）
     - 查看订阅的套餐信息
     - 查看订阅开始/结束时间
     - 查看是否自动续费
     - 管理员操作：暂停/恢复/取消用户的订阅（如果需要）
     - 订阅统计：活跃订阅数、过期订阅数、暂停订阅数等

2. **订阅历史记录页面（管理员视角）**
   - 需要添加：查看应用的订阅历史记录
   - 建议路径：`/dashboard/apps/[appId]/subscriptions/history`
   - 功能需求：
     - 查看应用的订阅历史记录列表（按 appId 查询）
     - 查看某个用户的订阅历史（管理员查看）
     - 查看历史操作（created/renewed/upgraded/paused/resumed/cancelled/expired等）
     - 按操作类型筛选
     - 按时间范围筛选
     - 分页显示

3. **订阅订单管理页面（管理员视角）**
   - 需要添加：订阅订单列表和详情页面
   - 建议路径：`/dashboard/subscriptions/orders` 或 `/dashboard/apps/[appId]/subscriptions/orders`
   - 功能需求：
     - 查看订单列表
     - 查看订单详情
     - 查看订单状态（pending/success/failed/closed/refunded等）
     - 查看支付信息

## 需要添加的入口链接

### 1. 在应用详情页添加订阅相关入口

当前应用详情页（`/dashboard/apps/[appId]`）已有 `subscriptions` tab，但只显示套餐管理。

**建议修改：**
- 在 `subscriptions` tab 中添加子菜单或标签页：
  - 套餐管理（当前已有）
  - 用户订阅（新增）
  - 订阅订单（新增）
  - 订阅历史（新增）

### 2. 在主导航中添加订阅入口（可选）

如果需要全局查看所有应用的订阅情况，可以在主导航中添加：
- `/dashboard/subscriptions` - 所有应用的订阅概览（管理员视角）
- `/dashboard/subscriptions/history` - 所有应用的订阅历史（管理员视角）

**注意：** 这不是用户视角的功能，而是管理员查看所有应用的订阅情况

### 3. 在用户管理页面添加订阅入口

在应用用户列表页面（`/dashboard/apps/[appId]?tab=users`）中，可以为每个用户添加：
- 查看该用户的订阅状态（管理员查看）
- 查看该用户的订阅历史（管理员查看）
- 查看该用户的订阅订单（管理员查看）

## 需要添加的 API 客户端函数

在 `dev-share-web/lib/api/subscription.ts` 中需要添加：

```typescript
// 管理员视角：查看用户的订阅状态
export async function getUserSubscription(userId: string): Promise<ApiResponse<GetOrEnsureMySubscriptionReply>>

// 管理员视角：取消用户的订阅（如果需要）
export async function cancelUserSubscription(userId: string, reason?: string): Promise<ApiResponse<void>>

// 管理员视角：暂停用户的订阅（如果需要）
export async function pauseUserSubscription(userId: string, reason?: string): Promise<ApiResponse<void>>

// 管理员视角：恢复用户的订阅（如果需要）
export async function resumeUserSubscription(userId: string): Promise<ApiResponse<void>>

// 管理员视角：设置用户的自动续费（如果需要）
export async function setUserAutoRenew(userId: string, autoRenew: boolean): Promise<ApiResponse<void>>

// 管理员视角：获取应用的订阅用户列表（需要后端提供按 appId 查询的 API）
// export async function listAppSubscriptions(appId: string, page?: number, pageSize?: number): Promise<ApiResponse<ListAppSubscriptionsReply>>

// 管理员视角：获取应用的订阅历史（需要后端提供按 appId 查询的 API）
// export async function getAppSubscriptionHistory(appId: string, userId?: string, page?: number, pageSize?: number): Promise<ApiResponse<GetSubscriptionHistoryReply>>

// 管理员视角：获取订阅订单列表（需要后端提供按 appId 查询的 API）
export async function listSubscriptionOrders(appId?: string, userId?: string, planId?: string, status?: string, page?: number, pageSize?: number): Promise<ApiResponse<ListSubscriptionOrdersReply>>

// 管理员视角：获取订阅订单详情
export async function getSubscriptionOrder(orderId: string): Promise<ApiResponse<SubscriptionOrder>>
```

## 需要添加的 TypeScript 类型定义

在 `dev-share-web/lib/api/types.ts` 中需要添加：

```typescript
// 订阅状态
export interface SubscriptionStatus {
  isActive: boolean
  planId: string
  startTime: number // timestamp
  endTime: number // timestamp
  status: 'active' | 'expired' | 'paused' | 'cancelled'
  autoRenew: boolean
}

// 订阅历史记录项
export interface SubscriptionHistoryItem {
  id: number
  planId: string
  planName: string
  startTime: number // timestamp
  endTime: number // timestamp
  status: string
  action: 'created' | 'renewed' | 'upgraded' | 'paused' | 'resumed' | 'cancelled' | 'expired' | 'enabled_auto_renew' | 'disabled_auto_renew'
  createdAt: number // timestamp
}

// 订阅历史响应
export interface GetSubscriptionHistoryReply {
  items: SubscriptionHistoryItem[]
  total: number
  page: number
  pageSize: number
}
```

## 后端 API 缺失分析

### 需要后端添加的 API（管理员视角）

1. **按应用查询订阅用户列表**
   - `ListAppSubscriptions(appId, page, pageSize)` - 获取应用的订阅用户列表
   - 返回：用户列表、订阅状态、套餐信息等

2. **按应用查询订阅历史**
   - `GetAppSubscriptionHistory(appId, userId?, page, pageSize)` - 获取应用的订阅历史
   - 支持按用户筛选（可选）

3. **订阅订单列表查询**
   - `ListSubscriptionOrders(appId?, userId?, planId?, status?, page, pageSize)` - 获取订阅订单列表
   - 支持多维度筛选：按应用、按用户、按套餐、按状态

4. **订阅统计信息**
   - `GetSubscriptionStats(appId)` - 获取订阅统计
   - 返回：活跃订阅数、过期订阅数、暂停订阅数、总收入等

## 优先级建议

### 高优先级（核心管理功能）
1. ✅ **用户订阅管理页面** - 管理员查看应用的订阅用户列表和状态
2. ✅ **订阅历史记录页面** - 管理员查看应用的订阅历史记录

### 中优先级（业务分析功能）
3. ⚠️ **订阅订单管理页面** - 管理员查看订阅订单，分析订阅业务
   - **价值**：按套餐分析销售情况、查看订单转化率、处理失败订单
   - **建议**：如果需要进行订阅业务分析，建议添加

### 低优先级（可选功能）
4. ⚠️ **在用户管理页面添加订阅入口** - 管理员查看特定用户的订阅详情
5. ⚠️ **订阅统计 Dashboard** - 订阅业务概览和统计图表

## 注意事项

1. **视角区分**：
   - `dev-share-web` 是**管理员后台**，用于管理应用的订阅业务
   - 用户视角的功能（用户查看自己的订阅、取消订阅等）由开发者在自己的 app 产品中实现
   - 管理员后台需要的是**管理用户订阅**的功能

2. **用户ID类型**：所有 API 都使用 `string` 类型的 `userId`（UUID），前端需要确保传递正确的类型

3. **权限控制**：
   - 管理员可以查看和管理所有应用的订阅
   - 需要确保权限控制正确，防止越权访问

4. **后端 API 缺失**：
   - 当前后端缺少按 `appId` 查询订阅用户列表的 API
   - 当前后端缺少按 `appId` 查询订阅历史的 API
   - 当前后端缺少订阅订单列表查询的 API
   - **需要先在后端添加这些 API，前端才能实现对应功能**

5. **状态同步**：订阅状态变更后需要及时刷新相关数据

6. **死链接检查**：确保所有新增的页面都有对应的入口链接，没有死链接

7. **订阅订单 vs 支付交易**：
   - `AppRevenue` 显示的是 `payment-service` 的所有支付交易（包括订阅订单、充值订单等）
   - 订阅订单管理可以专门查看订阅相关的订单，按套餐分析业务
   - 两者可以互补：支付交易看总收入，订阅订单看订阅业务分析
