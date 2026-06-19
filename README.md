# LanShare

局域网文件共享工具，支持 Windows 电脑与 Android 手机之间的快速文件传输、剪贴板共享和即时聊天。

## 功能特性

- **自动发现设备** - UDP 广播自动扫描局域网设备，无需手动输入 IP
- **文件传输** - TCP 直连传输，支持断点续传，速度可达 100MB/s+
- **加密传输** - AES-256-GCM 端到端加密，保障数据安全
- **共享文件夹** - 类 SMB 文件夹共享，支持只读/读写权限
- **剪贴板同步** - 跨设备复制粘贴文本和图片
- **文字聊天** - 设备间即时消息
- **扫码连接** - 手机扫描二维码即可打开 Web 界面
- **Web 界面** - 手机无需安装 APP，浏览器直接访问
- **暗色主题** - 现代化 UI 设计

## 截图

<div align="center">
  <img src="docs/screenshot.png" alt="LanShare Screenshot" width="800">
</div>

## 快速开始

### 环境要求

- Go 1.21+
- 支持 Windows / macOS / Linux

### 编译运行

```bash
# 克隆项目
git clone https://github.com/Breedom/lan_share.git
cd lan_share

# 安装依赖
go mod tidy

# 编译运行
go run cmd/lanshare/main.go
```

### 手机访问

1. 确保手机和电脑在同一局域网
2. 点击页面右上角二维码图标
3. 手机扫描二维码打开网页

或直接在手机浏览器访问：`http://电脑IP:8080`

### 防火墙设置

首次运行需要添加防火墙规则，否则手机无法访问。

**方式一：使用构建脚本（推荐）**

```bash
# Windows 下双击运行
build.bat
```

生成的 `lanshare.exe` 自带管理员权限，首次运行会自动添加防火墙规则。

**方式二：手动添加规则**

以管理员身份运行 PowerShell：

```powershell
netsh advfirewall firewall add rule name="LanShare" dir=in action=allow protocol=TCP localport=8080
```

**方式三：Windows 设置**

1. 打开 Windows 安全中心 → 防火墙
2. 点击「允许应用通过防火墙」
3. 添加 `lanshare.exe` 或手动开放 8080 端口

## 项目结构

```
lan_share/
├── cmd/lanshare/          # 程序入口
│   └── main.go
├── internal/
│   ├── core/              # 核心模块
│   │   ├── config.go      # 配置管理
│   │   ├── crypto.go      # AES 加密
│   │   ├── discovery.go   # 设备发现
│   │   ├── transfer.go    # 文件传输
│   │   ├── clipboard.go   # 剪贴板/聊天
│   │   └── firewall.go    # 防火墙管理
│   ├── server/            # HTTP 服务
│   │   └── http.go
│   └── models/            # 数据模型
├── web/                   # 前端资源
│   ├── index.html
│   └── static/
│       ├── app.js
│       └── style.css
├── configs/
│   └── default.json       # 默认配置
├── lanshare.manifest      # Windows 管理员权限清单
├── build.bat              # 构建脚本
├── go.mod
└── go.sum
```

## API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/peers` | GET | 获取在线设备列表 |
| `/api/device` | GET | 获取本机信息 |
| `/api/settings` | GET/POST | 读取/保存设置 |
| `/api/shares` | GET | 获取共享文件夹列表 |
| `/api/files/{share}` | GET | 浏览共享文件 |
| `/api/download/{share}/{file}` | GET | 下载文件 |
| `/api/upload/{share}` | POST | 上传文件 |
| `/api/chat/history` | GET | 获取聊天记录 |
| `/api/qrcode` | GET | 生成二维码 |
| `/ws` | WebSocket | 实时通信 |

## 配置说明

配置文件位于 `%APPDATA%/lanshare/config.json`（Windows）：

```json
{
  "server": {
    "http_port": 8080,
    "tcp_port": 53317,
    "udp_port": 53317
  },
  "security": {
    "encryption_enabled": true
  },
  "shares": [
    {
      "name": "Downloads",
      "path": "C:\\Users\\Downloads",
      "readonly": false
    }
  ],
  "general": {
    "device_name": "我的电脑"
  }
}
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go + net/http |
| 前端 | Tailwind CSS + Remix Icon |
| 通信 | WebSocket + TCP |
| 加密 | AES-256-GCM |
| 二维码 | go-qrcode |

## 开发计划

- [x] 设备自动发现
- [x] 文件传输 (TCP)
- [x] AES 加密
- [x] Web UI
- [x] 二维码扫码连接
- [ ] 剪贴板同步 (系统级)
- [ ] 屏幕共享
- [ ] Android APK
- [ ] 文件预览
- [ ] 传输历史

## License

MIT License
