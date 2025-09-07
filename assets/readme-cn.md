## [English](../README.md) | 简体中文

<br>

**Sponge** 是一个强大且易用的 Go 开发框架，其核心理念是 **定义即代码** (Definition is Code)，通过解析 `SQL`、`Protobuf`、`JSON` 文件生成模块化的 Go 代码，这些模块代码可灵活组合成多种类型的完整后端服务。

Sponge 提供一站式项目开发解决方案，涵盖代码生成、开发、测试、API 文档和部署等，帮助开发者以"低代码"方式轻松构建稳定可靠的高性能后端服务（包括 RESTful API、gRPC、HTTP+gRPC、gRPC Gateway 等）。

<br>

### 为什么选择 Sponge？

- **开发效率极高**： 自动生成 CRUD API、项目脚手架和胶水代码(非业务代码)，彻底解决传统开发过程中的大量重复劳动问题。
- **开箱即用**：覆盖开发全生命周期（生成→开发→测试→部署→监控），避免碎片化工具链。  
- **标准化最佳实践**：基于 Go 社区成熟方案（Gin/gRPC/Protobuf 等），避免技术选型纠结。  
- **极简学习曲线**：通过代码生成和清晰示例，快速上手，专注业务逻辑。  
- **适合团队协作**：统一项目结构，提升代码可维护性。
- **AI 协作开发**：Sponge 内置 AI 助手感知目录与文件结构，根据上下文智能生成业务逻辑代码，显著降低手工编码，提升开发效率与代码一致性。

<br>

### 关键特性

<details>
<summary> <b>一键生成完整后端服务代码。</b> </summary>

> 对于仅需 `CRUD API` 的 `Web`、`gRPC`或`HTTP+gRPC`服务，无需编写任何 `Go` 代码。只需连接数据库(如 `MySQL`、`MongoDB`、`PostgreSQL`、`SQLite`)，即可一键生成完整后端服务代码，并轻松部署到 Linux 服务器、Docker 或 Kubernetes 上。
</details>

<details>
<summary> <b>高效开发通用服务，从定义到实现一步到位。</b> </summary>

> 构建通用的 `Web`、`gRPC`、`HTTP+gRPC` 或 `gRPC Gateway` 服务，只需专注以下三步：
> - 定义数据库表 (SQL DDL)；
> - 在 Protobuf 文件中描述 API (Protobuf IDL)；
> - 实现业务逻辑 (支持内置 AI 助手自动生成并合并业务逻辑代码)。
>
> 包括 **CRUD API、服务框架及胶水代码** 在内的所有基础代码均由 **Sponge 自动生成**，让开发者聚焦核心业务，全面提升开发效率。
</details>

<details>
<summary> <b>支持自定义模板，灵活扩展。</b> </summary>

>  Sponge 支持通过自定义模板生成项目所需的多种代码类型，不局限于 `Go` 语言。例如`后端代码`、`前端代码`、`测试代码`、`构建和部署脚本`等。
</details>

<details>
<summary> <b>在页面生成代码，简单易用。</b> </summary>

> Sponge 提供在页面生成代码，避免了复杂的命令行操作，只需在页面上简单的填写参数即可一键生成代码。
</details>

<details>
<summary> <b>Sponge 与 AI 助手协同开发，实现基础设施自动化和业务逻辑智能化。</b> </summary>

> Sponge 搭配内置的 AI 助手(DeepSeek、ChatGPT、Gemini)，构建出一套完整、高效、智能的开发解决方案：
> - **Sponge**：负责基础设施代码自动生成，包括 `服务框架`、`CRUD API`、`自定义 API(不含业务逻辑)` 等代码，确保架构统一、规范化。
> - **AI 助手**：专注于业务逻辑实现，辅助完成 `数据表设计`、`Protobuf API 定义`、`业务逻辑编写` 等任务，减少重复劳动、提升研发效率。

</details>

<br>

### 适用场景

Sponge 适用于快速构建多种类型的高性能后端服务，适用场景如下：

- **开发 RESTful API 服务**
- **构建微服务项目**
- **云原生项目开发**
- **快速重构旧有项目**
- **作为 Go 初学者或团队学习的最佳实践的起点**

此外，开发者还可以通过自定义模板，生成满足业务需求的各类代码。

<br>

### 在线体验

Sponge 提供在线体验生成代码：[**Code Generation**](https://go-sponge.com/ui/)

> 注：若需在本地运行下载的服务代码，需先完成 Sponge 的本地安装。

<br>

### 快速上手

1. **安装 Sponge**：支持 Windows/macOS/Linux/Docker，查看 [**Sponge 安装指南**](https://github.com/go-dev-frame/sponge/blob/main/assets/install-cn.md)。

2. **打开生成代码 UI 页面**

   ```bash
   sponge run
   ```

   在本地浏览器访问 `http://localhost:24631`生成代码。

3. **示例：一键生成完整的 Web 服务后端代码**

   <p align="center">
   <img width="1500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/sponge-ui.png">
   </p>

   - 项目本地运行与测试步骤：
     - **填写参数**：连接数据库，选择表名。
     - **下载代码**：获取完整项目代码。
     - **生成 swagger 文档**：执行命令 `make docs`。
     - **启动服务**：执行命令 `make run`。
     - **接口测试**：在浏览器中访问 `http://localhost:8080/swagger/index.html` 通过 Swagger 界面进行 API 测试。

<br>

### 组件

Sponge 内置了 Go 生态常见的 30+ 个组件(按需使用)，包括 **Gin**, **gRPC**, **GORM**, **MongoDB**, **Redis**, **Kafka**, **DTM**, **WebSocket**, **Prometheus** 等主流技术栈，[**查看所有组件**](https://go-sponge.com/zh/component/)。

<br>

### 代码生成引擎

Sponge 提供强大的代码生成能力，支持基于`内置模板`和`自定义模板`两种方式快速生成项目所需代码，同时集成`AI 助手`辅助生成业务逻辑代码。

1. Sponge 基于内置模板生成代码框架，如下图所示：

<p align="center">
<img width="1500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/sponge-framework.png">
</p>

<br>

2. Sponge 基于自定义模板生成代码框架，如下图所示：

<p align="center">
<img width="1200px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/template-framework.png">
</p>

<br>

3. Sponge 基于函数及注释生成业务逻辑代码框架，如下图所示：

<p align="center">
<img width="1200px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/ai-assistant-framework.png">
</p>

<br>

### 微服务框架

Sponge 是一个现代化的 Go 微服务框架，它采用典型的微服务分层架构，内置了丰富的服务治理功能，帮助开发者快速构建和维护复杂的微服务系统，框架结构如下图所示：

<p align="center">
<img width="1000px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/microservices-framework.png">
</p>

<br>

创建的 HTTP 和 gRPC 服务代码的性能测试： 50个并发，总共100万个请求。

![http-server](https://raw.githubusercontent.com/zhufuyi/microservices_framework_benchmark/main/test/assets/http-server.png)

![grpc-server](https://raw.githubusercontent.com/zhufuyi/microservices_framework_benchmark/main/test/assets/grpc-server.png)

点击查看 [**测试代码**](https://github.com/zhufuyi/microservices_framework_benchmark)。

<br>

### 目录结构

Sponge 创建的服务代码目录结构遵循 [project-layout](https://github.com/golang-standards/project-layout)。

Sponge 支持创建 `单体应用单体仓库(monolith)`、`微服务多仓库(multi-repo)`、`微服务单体仓库(mono-repo)`项目代码结构。

<details>
<summary> <b>1. 单体应用单体仓库(monolith)或微服务多仓库(multi-repo)目录结构说明。</b> </summary>

```bash
   .
   ├── api            # protobuf文件和生成的*pb.go目录
   ├── assets         # 其他与资源库一起使用的资产(图片、logo等)目录
   ├── cmd            # 程序入口目录
   ├── configs        # 配置文件的目录
   ├── deployments    # 裸机、docker、k8s部署脚本目录
   ├── docs           # 设计文档和界面文档目录
   ├── internal       # 项目内部代码目录
   │    ├── cache        # 基于业务包装的缓存目录
   │    ├── config       # Go结构的配置文件目录
   │    ├── dao          # 数据访问目录
   │    ├── database     # 数据库目录
   │    ├── ecode        # 自定义业务错误代码目录
   │    ├── handler      # http的业务功能实现目录
   │    ├── model        # 数据库模型目录
   │    ├── routers      # http路由目录
   │    ├── rpcclient    # 连接grpc服务的客户端目录
   │    ├── server       # 服务入口，包括http、grpc等
   │    ├── service      # grpc的业务功能实现目录
   │    └── types        # http的请求和响应类型目录
   ├── pkg            # 外部应用程序可以使用的库目录
   ├── scripts        # 执行脚本目录
   ├── test           # 额外的外部测试程序和测试数据
   ├── third_party    # 依赖第三方protobuf文件或其他工具的目录
   ├── Makefile       # 开发、测试、部署相关的命令集合
   ├── go.mod         # go 模块依赖关系和版本控制文件
   └── go.sum         # go 模块依赖项的密钥和校验文件
```
</details>

<details>
<summary> <b>2. 微服务单体仓库(mono-repo)(大仓库)目录结构说明。</b> </summary>

```bash
   .
   ├── api
   │    ├── server1       # 服务1的protobuf文件和生成的*pb.go目录
   │    ├── server2       # 服务2的protobuf文件和生成的*pb.go目录
   │    ├── server3       # 服务3的protobuf文件和生成的*pb.go目录
   │    └── ...
   ├── server1        # 服务1的代码目录，与微服务多仓库(multi-repo)目录结构基本一样
   ├── server2        # 服务2的代码目录，与微服务多仓库(multi-repo)目录结构基本一样
   ├── server3        # 服务3的代码目录，与微服务多仓库(multi-repo)目录结构基本一样
   ├── ...
   ├── third_party    # 依赖的第三方protobuf文件
   ├── go.mod         # go 模块依赖关系和版本控制文件
   └── go.sum         # go 模块依赖项的密钥和校验和文件
```
</details>

<br>

### 文档

点击查看 [Sponge 官方文档](https://go-sponge.com/zh/)，完整涵盖了开发指南、组件、服务配置与部署方案等核心内容。

<br>

### 示例

#### Sponge 创建服务代码示例

- [基于sql创建web服务(包括CRUD)](https://github.com/go-dev-frame/sponge_examples/tree/main/1_web-gin-CRUD)
- [基于sql创建grpc服务(包括CRUD)](https://github.com/go-dev-frame/sponge_examples/tree/main/2_micro-grpc-CRUD)
- [基于protobuf创建web服务](https://github.com/go-dev-frame/sponge_examples/tree/main/3_web-gin-protobuf)
- [基于protobuf创建grpc服务](https://github.com/go-dev-frame/sponge_examples/tree/main/4_micro-grpc-protobuf)
- [基于protobuf创建grpc网关服务](https://github.com/go-dev-frame/sponge_examples/tree/main/5_micro-gin-rpc-gateway)
- [基于protobuf创建grpc+http服务](https://github.com/go-dev-frame/sponge_examples/tree/main/_10_micro-grpc-http-protobuf)

#### 分布式事务示例

- [简单的分布式订单系统](https://github.com/go-dev-frame/sponge_examples/tree/main/9_order-grpc-distributed-transaction)
- [秒杀抢购活动](https://github.com/go-dev-frame/sponge_examples/tree/main/_12_sponge-dtm-flashSale)
- [电商系统](https://github.com/go-dev-frame/sponge_examples/tree/main/_14_eshop)

####  Sponge+AI 助手协同开发示例

- [家电零售管理](https://github.com/go-dev-frame/sponge_examples/tree/main/_15_appliance_store)

#### Sponge 开发项目示例

- [社区后端服务](https://github.com/go-dev-frame/sponge_examples/tree/main/7_community-single)
- [单体服务拆分为微服务](https://github.com/go-dev-frame/sponge_examples/tree/main/8_community-cluster)


<br>

### 贡献

欢迎 Issue/PR！[贡献指南](https://go-sponge.com/zh/community/contribution.html)。

如果 Sponge 对您有帮助，请给个 ⭐ Star！这将激励我们继续迭代。

<br>

### 社区交流

欢迎加入**go sponge微信群交流**，加微信(备注`sponge`)进群。

<img width="300px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/wechat-group.jpg">

<br>
