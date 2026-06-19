# LAN Share

局域网文件共享工具 —— 零配置、基于 Web，纯 Python 实现。

## 特性

- **零配置**：无需安装依赖，一条命令启动
- **Web 界面**：浏览器访问，拖拽上传 / 一键下载 / 复制链接
- **LAN 发现**：UDP 广播自动发现局域网中其他运行本工具的设备
- **文件管理**：上传、下载、删除文件，实时刷新
- **暗色 UI**：现代化深色主题，移动端自适应
- **纯标准库**：仅使用 Python 内置模块，无额外依赖

## 快速开始

```powershell
# 启动（默认分享 shared/shared_data/ 目录，端口 8000）
python -m shared

# 指定目录和端口
python -m shared D:\Share --port 9000

# 或用启动脚本
.\lan_share.ps1 -Directory D:\Share -Port 8000
```

启动后在浏览器打开控制台显示的地址即可使用。

## 使用说明

打开 Web 页面后：

- **上传**：拖拽文件到虚线区域，或点击 "click to browse"
- **下载**：点击文件行右侧的 ⬇ 按钮
- **复制链接**：点击 🔗 按钮复制文件直链
- **删除**：点击 🗑 按钮删除文件（带确认弹窗）
- **发现设备**：页面顶部的 peer 栏显示局域网中其他运行 LAN Share 的设备，点击可直接跳转

## 项目结构

```
shared/
  __init__.py       # 包定义
  __main__.py       # 命令行入口
  server.py         # HTTP 文件服务器（上传/下载/删除/文件列表）
  discovery.py      # UDP 广播自动发现
  shared_data/      # 分享文件存放目录（自动创建，已 gitignore）
  static/
    index.html      # Web UI（嵌入在 server.py 中）
    style.css       # 样式
    app.js          # 前端逻辑
lan_share.ps1       # PowerShell 启动脚本
```

## 许可证

MIT
