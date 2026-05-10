# Door Dump

OpenWrt 远程抓包系统。通过 HTTP 流式传输将路由器上的抓包数据实时上传到远程服务器。

## 架构

```
┌──────────────┐     SSH 推送/触发      ┌──────────────┐    HTTP chunked     ┌──────────────┐
│   CLI 控制端  │ ──────────────────→   │  Agent 路由器  │ ─────────────────→  │   Server 接收端 │
│  (本地 Mac)   │                       │  (OpenWrt)    │                     │  (远程服务器)    │
└──────────────┘                       └──────────────┘                     └──────────────┘
```

- **Agent**：运行在 OpenWrt 路由器上，调用 tcpdump 抓包，通过 HTTP chunked 流式上传
- **Server**：运行在远程服务器上，接收 pcap 数据并存储，提供查询/下载 API
- **CLI**：运行在本地，通过 SSH 部署 Agent、触发抓包、管理任务

## 环境要求

- Go 1.21+
- UPX（可选，用于压缩二进制）
- OpenWrt 路由器（已安装 tcpdump）
- 远程服务器（x86_64 Linux）

## 编译

```bash
# 编译全部
make build

# 单独编译
make build-agent    # 交叉编译 Agent（mipsle，适用于 OpenWrt）
make build-server   # 编译 Server（当前平台）
make build-cli      # 编译 CLI（当前平台）

# 编译 Server 用于 Linux x86_64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o door-server ./cmd/server

# 压缩 Agent 二进制（推荐，6.3MB → 2.2MB）
upx --best door-agent
```

## 部署

### 1. 部署 Server（远程服务器）

```bash
# 上传到服务器
scp door-server user@your-server:/opt/door-dump/

# SSH 到服务器运行
ssh user@your-server
cd /opt/door-dump
./door-server -addr :8080 -data ./data/captures
```

Server 参数：
- `-addr`：监听地址，默认 `:8080`
- `-data`：pcap 存储目录，默认 `./data/captures`

### 2. 部署 Agent（OpenWrt 路由器）

```bash
# 方式1：使用 scp 旧协议
scp -O -o HostKeyAlgorithms=+ssh-rsa door-agent root@192.168.1.30:/tmp/

# 方式2：使用 SSH 管道传输（推荐）
cat door-agent | ssh -o HostKeyAlgorithms=+ssh-rsa root@192.168.1.30 "cat > /tmp/door-agent && chmod +x /tmp/door-agent"
```

注意：Agent 放在 `/tmp`（tmpfs），路由器重启后需要重新部署。

### 3. 使用 CLI（本地）

```bash
# 部署 Agent 到路由器
./doorctl deploy --host 192.168.1.30 --password your-password --binary ./door-agent

# 查看路由器网络接口
./doorctl ifaces --host 192.168.1.30 --password your-password

# 触发抓包
./doorctl capture --host 192.168.1.30 --password your-password \
  --iface br-lan --duration 60s --server http://your-server:8080

# 查看抓包记录
./doorctl list --server http://your-server:8080

# 下载 pcap 文件
./doorctl download <capture-id> --server http://your-server:8080 -o output.pcap
```

## Agent 参数

```
/tmp/door-agent [options]

参数：
  -i string      网络接口 (默认 "br-lan")
  -f string      BPF 过滤表达式 (如 "port 80" "host 192.168.1.1")
  -d string      抓包时长 (如 "30s", "5m", "1h") (默认 "60s")
  -s string      服务端 URL (必填，如 "http://your-server:8080")
  -device string 设备标识符 (可选)
  -list          列出可用网络接口并退出
```

列出网络接口：

```bash
/tmp/door-agent -list
```

示例：

```bash
# 抓取 br-lan 接口 60 秒
/tmp/door-agent -i br-lan -d 60s -s http://your-server:8080

# 抓取 80 端口流量
/tmp/door-agent -i br-lan -f "port 80" -d 5m -s http://your-server:8080

# 抓取指定主机流量
/tmp/door-agent -i br-lan -f "host 192.168.1.100" -d 30s -s http://your-server:8080

# 指定设备标识
/tmp/door-agent -i br-lan -d 60s -s http://your-server:8080 -device router-01
```

## Server API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/upload` | 流式接收 pcap |
| GET | `/api/v1/captures` | 列出所有抓包记录 |
| GET | `/api/v1/captures/get?id=<id>` | 下载指定 pcap 文件 |
| DELETE | `/api/v1/captures/delete?id=<id>` | 删除指定抓包记录 |
| GET | `/api/v1/health` | 健康检查 |

## 技术细节

- Agent 调用系统 tcpdump，不依赖 libpcap（避免 CGO）
- 使用 HTTP chunked transfer encoding 流式传输，边抓边传
- Agent 内存占用仅 64KB 缓冲区
- pcap 文件按日期自动归档存储
