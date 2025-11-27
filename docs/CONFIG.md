# 配置加载说明

## 配置文件位置

默认配置文件位于 `configs/config.yaml`

## 启动方式

### 使用默认配置
```bash
./bin/server
```

### 指定配置文件
```bash
./bin/server -conf /path/to/config.yaml
```

## 配置项说明

### Server 配置
- `server.http.addr`: HTTP 服务监听地址
- `server.grpc.addr`: gRPC 服务监听地址
- `server.http.timeout`: HTTP 超时时间
- `server.grpc.timeout`: gRPC 超时时间

### Data 配置
- `data.database.driver`: 数据库驱动 (mysql)
- `data.database.source`: 数据库连接字符串
  - MySQL: `user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local`

### Client 配置
- `client.payment.addr`: Payment Service 地址

### Log 配置
- `log.level`: 日志级别 (debug/info/warn/error)
- `log.format`: 日志格式 (json/text)
- `log.output`: 日志输出 (stdout/file/both)
- `log.file_path`: 日志文件路径
- `log.max_size`: 日志文件最大大小（MB）
- `log.max_age`: 日志文件保留天数
- `log.max_backups`: 日志文件最大备份数
- `log.compress`: 是否压缩旧日志文件

## 完整配置示例

```yaml
server:
  http:
    addr: 0.0.0.0:8102
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9102
    timeout: 1s

data:
  database:
    driver: mysql
    source: root:@tcp(localhost:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local

client:
  payment:
    addr: localhost:9101

log:
  level: info
  format: json
  output: both
  file_path: logs/subscription-service.log
  max_size: 100
  max_age: 30
  max_backups: 10
  compress: true
```

## 环境切换

### 开发环境
使用默认配置即可：
```yaml
server:
  http:
    addr: 0.0.0.0:8102
  grpc:
    addr: 0.0.0.0:9102

data:
  database:
    driver: mysql
    source: root:@tcp(localhost:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local

log:
  level: debug
  format: text
  output: stdout
```

### 生产环境
修改 `configs/config.yaml`:
```yaml
server:
  http:
    addr: 0.0.0.0:8102
    timeout: 5s
  grpc:
    addr: 0.0.0.0:9102
    timeout: 5s

data:
  database:
    driver: mysql
    source: user:password@tcp(db-host:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local

client:
  payment:
    addr: payment-service:9101

log:
  level: info
  format: json
  output: both
  file_path: /var/log/subscription-service/app.log
  max_size: 100
  max_age: 30
  max_backups: 10
  compress: true
```

## 使用环境变量

可以通过环境变量覆盖配置文件中的值：

```bash
# 设置数据库连接
export DB_SOURCE="user:password@tcp(localhost:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local"

# 设置日志级别
export LOG_LEVEL="debug"

# 设置 Payment Service 地址
export PAYMENT_ADDR="payment-service:9101"

# 启动服务
./bin/server -conf configs/config.yaml
```

在代码中读取环境变量：
```go
import "os"

// 优先使用环境变量
if dbSource := os.Getenv("DB_SOURCE"); dbSource != "" {
    c.Data.Database.Source = dbSource
}

if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
    c.Log.Level = logLevel
}

if paymentAddr := os.Getenv("PAYMENT_ADDR"); paymentAddr != "" {
    c.Client.Payment.Addr = paymentAddr
}
```

## Docker 配置

### 使用配置文件挂载

```bash
docker run -d \
  -p 8102:8102 \
  -p 9102:9102 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/logs:/app/logs \
  subscription-service:latest
```

### 使用环境变量

```bash
docker run -d \
  -p 8102:8102 \
  -p 9102:9102 \
  -e DB_SOURCE="user:password@tcp(db-host:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local" \
  -e PAYMENT_ADDR="payment-service:9101" \
  -e LOG_LEVEL="info" \
  subscription-service:latest
```

### Docker Compose 配置

```yaml
version: '3.8'

services:
  subscription-service:
    image: subscription-service:latest
    ports:
      - "8102:8102"
      - "9102:9102"
    volumes:
      - ./configs:/app/configs
      - ./logs:/app/logs
    environment:
      - DB_SOURCE=root:@tcp(mysql:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local
      - PAYMENT_ADDR=payment-service:9101
      - LOG_LEVEL=info
    depends_on:
      - mysql
      - payment-service
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: subscription_service
    volumes:
      - mysql_data:/var/lib/mysql
      - ./docs/sql:/docker-entrypoint-initdb.d
    ports:
      - "3306:3306"

volumes:
  mysql_data:
```

## 注意事项

1. **数据库密码**: 生产环境不要在配置文件中明文存储密码，使用环境变量或密钥管理服务
2. **Payment Service 地址**: 确保 Payment Service 已启动并可访问
3. **日志文件权限**: 确保日志目录有写权限
4. **端口冲突**: 确保 8102 和 9102 端口未被占用
5. **超时配置**: 根据实际业务需求调整超时时间

## 配置验证

启动服务后，可以通过以下方式验证配置：

```bash
# 检查服务是否启动
curl http://localhost:8102/health

# 检查日志输出
tail -f logs/subscription-service.log

# 检查数据库连接
mysql -u root -D subscription_service -e "SELECT COUNT(*) FROM plan;"
```

## 故障排查

### 配置文件未找到
```
Error: failed to load config: open configs/config.yaml: no such file or directory
```
解决方案：
- 检查配置文件路径是否正确
- 使用 `-conf` 参数指定配置文件路径

### 数据库连接失败
```
Error: failed to connect database: dial tcp 127.0.0.1:3306: connect: connection refused
```
解决方案：
- 检查 MySQL 是否启动
- 检查数据库连接字符串是否正确
- 检查网络连接

### 端口被占用
```
Error: listen tcp :8102: bind: address already in use
```
解决方案：
- 修改配置文件中的端口号
- 或停止占用端口的进程：`lsof -ti:8102 | xargs kill -9`

### Payment Service 连接失败
```
Error: failed to connect payment service: context deadline exceeded
```
解决方案：
- 检查 Payment Service 是否启动
- 检查 `client.payment.addr` 配置是否正确
- 检查网络连接

