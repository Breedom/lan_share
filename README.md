# LAN Share

局域网文件共享工具 —— 零配置、基于 Web，纯 Python 实现。

## 特性

- **零配置**：无需安装依赖，一条命令启动
- **Web 界面**：浏览器访问，拖拽上传 / 一键下载 / 复制链接
- **LAN 发现**：UDP 广播自动发现局域网中其他运行本工具的设备
- **文件管理**：上传、下载、删除文件，支持子目录浏览，实时刷新
- **GUI 管理面板**：图形化界面，日志 / 在线设备 / 文件列表一目了然
- **可打包为 exe**：无需 Python 环境，双击即可运行
- **暗色 UI**：现代化深色主题，移动端自适应
- **纯标准库**：仅使用 Python 内置模块，无额外依赖

## 快速开始

```powershell
# 启动（默认分享 shared_data/ 目录，端口 8000）
python __main__.py

# 指定目录和端口
python __main__.py D:\Share --port 9000

# 或用启动脚本
.\lan_share.ps1 -Directory D:\Share -Port 8000
```

启动后在浏览器打开控制台显示的地址即可使用。

## GUI 管理面板

```powershell
# 启动图形化管理面板
python __main__.py --gui
python gui.py          # 等价
```

- 目录选择与端口配置
- 一键启停服务器
- 实时日志输出
- 在线设备自动发现
- 文件列表预览

## 打包为 exe

```powershell
pip install pyinstaller    # 首次需要安装
python build_exe.py        # 输出 dist/LanShare.exe（~11 MB）
```

打包后的 exe 为单文件，双击自动打开 GUI 管理面板，无需安装 Python。

## 使用说明

打开 Web 页面后：

- **上传**：拖拽文件到虚线区域，或点击 "click to browse"
- **下载**：点击文件行右侧的 ⬇ 按钮
- **复制链接**：点击 🔗 按钮复制文件直链
- **删除**：点击 🗑 按钮删除文件（带确认弹窗）
- **子目录浏览**：点击文件夹名称进入子目录
- **发现设备**：页面顶部的 peer 栏显示局域网中其他运行 LAN Share 的设备，点击可直接跳转

## 项目结构

```
__init__.py       # 包定义
__main__.py       # 命令行入口（--gui 启动图形面板）
gui.py            # tkinter GUI 管理面板
server.py         # HTTP 文件服务器（上传/下载/删除/文件列表）
discovery.py      # UDP 广播自动发现
shared_data/      # 分享文件存放目录（自动创建，已 gitignore）
static/
  app.js          # 前端逻辑
  style.css       # 样式
lan_share.ps1     # PowerShell 启动脚本
build_exe.py      # exe 打包脚本
build_exe.spec    # PyInstaller 打包配置
```

## 许可证

MIT
