要求 [Go 1.27.1+](https://studygolang.com/dl) 版本。

> **提示**：Go下载第三方包时可能会遇到依赖下载超时问题，建议设置国内代理：
> ```bash
> go env -w GOPROXY=https://goproxy.cn,direct
> ```

<br>

---

<br>

### Windows 环境

#### 前置准备

确保已安装 Go 并配置 `GOBIN` 环境变量：

```bash
# 检查 GOBIN 是否已设置（输出为空表示未设置）
go env GOBIN

# 若未设置，请配置（根据实际修改示例路径）
go env -w GOBIN=D:\go\bin
# 然后将 GOBIN 路径添加到系统 PATH 环境变量
```

<br>

#### 快速安装 sponge

我们提供了包含所有依赖的集成安装包：
- 百度云盘：[sponge-install.zip](https://pan.baidu.com/s/1adMIlUyQlH6vRK2UIN7MRg?pwd=3fja)
- 蓝奏云：[sponge 安装文件](https://wwm.lanzoue.com/b049fldpi) 密码:5rq9，*需下载全部4个文件，安装前请阅读`安装说明.txt`*

<br>

**安装步骤**：

1. 解压后运行 `install.bat`
    - 安装 Git 时保持默认选项即可（已安装可跳过）
2. 打开 Git Bash 终端，鼠标右键 → 【Open Git Bash here】
   ```bash
   sponge init          # 初始化并安装依赖
   sponge plugins       # 查看已安装的插件
   sponge -v            # 查看版本
   ```

   **注意**：请始终使用 Git Bash，不要使用 Windows 默认的 cmd 终端，否则可能出现找不到命令的错误。

<br>

> 上面是集成安装包的安装方式，另支持原生安装 sponge 方式，详见：  
> 👉 [【安装 Sponge】→【Windows 环境】](https://go-sponge.com/zh/getting-started/install.html#安装-sponge)

<br>

---

<br>

### Linux/MacOS 环境

#### 环境配置

配置环境变量（已配置可跳过）：

```bash
vim ~/.bashrc
```

添加以下内容（根据实际情况修改路径）：
```bash
export GOROOT="/opt/go"       # Go 安装目录
export GOPATH=$HOME/go        # Go 模块下载目录
export GOBIN=$GOPATH/bin      # 可执行文件存放目录
export PATH=$PATH:$GOBIN:$GOROOT/bin
```

生效配置：
```bash
source ~/.bashrc
go env GOBIN  # 验证是否配置成功，如果输出不为空，说明设置成功
```

<br>

#### 安装步骤

1. 安装 protoc：
    - 下载地址：[protoc v31.1](https://github.com/protocolbuffers/protobuf/releases/tag/v31.1)
    - 将 `protoc` 可执行文件放入 `GOBIN` 目录

2. 安装 sponge：
   ```bash
   go install github.com/go-dev-frame/sponge/cmd/sponge@latest
   sponge init          # 初始化并安装依赖
   sponge plugins       # 查看已安装的插件
   sponge -v            # 查看版本
   ```

<br>

---

<br>

### Docker 环境

> ⚠ 注意：Docker 版仅支持 UI 代码生成功能，如需在生成的服务代码基础上进行开发，仍需在本地安装 sponge 完整环境。

#### 快速启动

**方案一：Docker 命令**
```bash
docker run -d --name sponge -p 24631:24631 zhufuyi/sponge:latest -a http://<宿主机IP>:24631
```

<br>

**方案二：docker-compose**
```yaml
version: "3.7"
services:
  sponge:
    image: zhufuyi/sponge:latest
    container_name: sponge
    restart: always
    command: ["-a","http://<宿主机IP>:24631"]
    ports:
      - "24631:24631"
```
启动命令：
```bash
docker-compose up -d
```

访问地址：`http://<宿主机IP>:24631`
