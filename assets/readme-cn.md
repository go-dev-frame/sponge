## [English](../README.md) | 简体中文

<br>

### 简介

**Sponge** 是一款功能强大且易用的 Go 开发框架，它整合了 **代码自动生成**、**Web 框架 (Gin)** 和 **微服务框架 (gRPC)**，覆盖了从项目生成、开发、测试、API 文档到部署的全生命周期。

Sponge 的核心理念是 **"Definition is Code" (定义即代码)**。通过解析 SQL、Protobuf 和 JSON 配置文件生成模块化的后端服务代码。你可以像搭积木一样，灵活组合这些代码模块，快速构建从单体应用到微服务集群的各种后端系统（包括 RESTful API、gRPC、HTTP+gRPC、gRPC Gateway 等）。

Sponge 旨在提升后端服务的开发效率与代码质量，通过 **低代码** 方式，帮助开发者彻底消除繁琐的重复性工作（如 CRUD、路由配置、项目结构搭建等），只需专注于核心业务逻辑的实现。

<br>

### 为什么选择 Sponge？

- **极致的开发效率**： 自动生成 CRUD API、项目脚手架及大量 Boilerplate（样板）代码，让开发者告别重复劳动。
- **开箱即用**：提供完整的开发工具链（生成 → 开发 → 测试 → 部署 → 监控），无需拼凑碎片化工具。  
- **行业最佳实践**：基于 Go 社区主流技术栈（Gin/GORM/gRPC/Protobuf 等）构建，架构规范，降低技术选型风险。 
* **灵活可扩展**：支持自定义模板，不仅限于 Go，还可扩展前端、测试脚本等任意代码生成。
- **学习曲线平缓**：入门门槛低，易于上手。
- **适合团队协作**：统一项目结构，提升团队协作效率和代码可维护性。
- **低代码与 AI 融合**：
    - **Sponge** 负责生成标准化的基础设施代码（API、数据层、服务框架）。
    - **AI 模型** 感知项目上下文，辅助生成核心业务逻辑，实现“基建自动化，业务智能化”。

<br>

### 关键特性

<details>
<summary> <b>一键生成完整后端服务 (Zero-Code)。</b> </summary>

> 针对仅需基础 `CRUD` 能力的场景（Web、gRPC 或 HTTP+gRPC），**无需编写一行 Go 代码**。只需连接数据库（MySQL、MongoDB、PostgreSQL、SQLite），即可一键生成完整的后端服务，并支持直接部署到 Linux、Docker 或 Kubernetes。
</details>

<details>
<summary> <b>协议驱动开发通用后端服务，从定义到实现一步到位。</b> </summary>

> 针对通用的 `Web`、`gRPC`、`HTTP+gRPC` 或 `gRPC Gateway` 服务，采用标准化的开发流程：
> 1. **定义数据**：设计 SQL DDL。
> 2. **定义接口**：编写 Protobuf IDL。
> 3. **生成代码**：Sponge 自动生成 CRUD、路由、文档及胶水代码。
> 4. **填充业务**：利用 Sponge 内置的 AI 模型或手动补全核心逻辑。

</details>

<details>
<summary> <b>可视化代码生成，简单易用。</b> </summary>

> 告别复杂的命令行参数。Sponge 提供直观的 Web UI，通过简单的表单配置即可完成代码生成，降低上手门槛。
</details>

<details>
<summary> <b>AI 协同开发。</b> </summary>

> Sponge 本身集成 DeepSeek、ChatGPT、Gemini 等 AI 模型。Sponge 处理繁琐的工程结构与标准化代码，AI 模型专注于理解需求并生成具体的业务逻辑代码，显著提升代码一致性与研发速度。

</details>

<details>
<summary> <b>支持自定义模板，灵活扩展。</b> </summary>

>  不局限于 Go 语言，你可以通过自定义模板生成前端代码、测试脚本、CI/CD 配置等任何文本文件，满足团队特定的工程化需求。

</details>

<br>

### 适用场景

Sponge 适用于构建高性能、可维护的后端系统，特别适合：

- **快速开发 RESTful API 服务**
- **构建大规模微服务架构**
- **云原生应用开发**
- **旧有项目的快速重构与迁移**
- **Go 语言初学者或团队的标准化工程模板**

<br>

### 在线体验

无需安装，直接在浏览器中体验代码生成功能：[**Code Generation**](https://go-sponge.com/ui/)

> 注：若需在本地运行下载的服务代码，需先完成 Sponge 的本地安装。

<br>

### 快速上手

1. **安装 Sponge**：支持 Windows、macOS、Linux 及 Docker 环境，查看 [**Sponge 安装指南**](https://github.com/go-dev-frame/sponge/blob/main/assets/install-cn.md)。

2. **打开生成代码 UI 页面**

   ```bash
   sponge run
   ```

   在本地浏览器访问 `http://localhost:24631`生成代码。

3. **示例：基于 SQL 一键生成 Web 服务后端代码**

   <p align="center">
   <img width="1500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/sponge-ui.png">
   </p>

   - 操作流程：
     - **填写参数**：在 UI 中连接数据库并选择表。
     - **下载代码**：点击生成并下载完整项目包。
     - **生成 swagger 文档**：在项目根目录执行 `make docs`。
     - **启动服务**：执行 `make run`。
     - **测试接口**：在浏览器中访问 `http://localhost:8080/swagger/index.html` 通过 Swagger 界面进行 API 测试。

<br>

### 技术栈与组件

Sponge 遵循“电池内置”原则，集成了 30+ 个 Go 生态主流组件，按需加载：

| 类别 | 组件                                                    |
| :--- |:------------------------------------------------------|
| **框架** | Gin, gRPC                                             |
| **数据库** | GORM (MySQL, PostgreSQL, SQLite等), MongoDB            |
| **缓存/消息** | Redis, Kafka, RabbitMQ                                |
| **服务治理** | Etcd, Consul, Nacos, Jaeger, Prometheus, OpenTelemetry               |
| **其他** | DTM (分布式事务), WebSocket, Swagger, PProf |

👉 [**查看完整的技术栈与组件列表**](https://go-sponge.com/zh/component/)。

<br>

### 架构与原理

#### 代码生成引擎

Sponge 提供多种代码生成引擎，支持**内置模板**、**自定义模板**以及**AI 辅助生成**。

1. 基于**内置模板**的代码生成引擎，如下图所示：

<p align="center">
<img width="1500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/sponge-framework.png">
</p>

<br>

2. 基于**自定义模板**代码生成引擎，如下图所示：

<p align="center">
<img width="1200px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/template-framework.png">
</p>

<br>

3. **AI 辅助业务逻辑**代码生成引擎，如下图所示：

<p align="center">
<img width="1200px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/ai-assistant-framework.png">
</p>

<br>

#### 微服务架构

Sponge 生成的代码遵循典型的分层架构，内置服务治理能力，结构清晰，易于维护。一个现代的微服务框架结构如下图所示：

<p align="center">
<img width="1000px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/microservices-framework.png">
</p>

<br>

### 性能基准测试

基于 50 并发、100 万次请求的测试结果显示，Sponge 生成的服务具有优异的性能表现：

1. **HTTP 服务**
![http-server](https://raw.githubusercontent.com/zhufuyi/microservices_framework_benchmark/main/test/assets/http-server.png)

2. **gRPC 服务**
![grpc-server](https://raw.githubusercontent.com/zhufuyi/microservices_framework_benchmark/main/test/assets/grpc-server.png)

👉 [**查看详细测试代码与环境**](https://github.com/zhufuyi/microservices_framework_benchmark)。

<br>

### 目录结构

Sponge 创建的服务代码目录结构遵循 [project-layout](https://github.com/golang-standards/project-layout)，支持 **Monolith (单体)**、**Multi-Repo (多仓微服务)** 及 **Mono-Repo (单仓微服务)** 模式。

<details>
<summary> <b>1. Monolith / Multi-Repo 结构详解。</b> </summary>

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
<summary> <b>2. Mono-Repo (大仓) 结构详解。</b> </summary>

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

### 示例项目

#### 基础示例

- [Web 服务 (基于 SQL, 含 CRUD)](https://github.com/go-dev-frame/sponge_examples/tree/main/1_web-gin-CRUD)
- [gRPC 微服务 (基于 SQL, 含 CRUD)](https://github.com/go-dev-frame/sponge_examples/tree/main/2_micro-grpc-CRUD)
- [Web 服务 (基于 Protobuf)](https://github.com/go-dev-frame/sponge_examples/tree/main/3_web-gin-protobuf)
- [gRPC 微服务 (基于 Protobuf)](https://github.com/go-dev-frame/sponge_examples/tree/main/4_micro-grpc-protobuf)
- [gRPC 网关服务 (Gateway)](https://github.com/go-dev-frame/sponge_examples/tree/main/5_micro-gin-rpc-gateway)
- [gRPC+HTTP 微服务 (基于 Protobuf)](https://github.com/go-dev-frame/sponge_examples/tree/main/_10_micro-grpc-http-protobuf)

#### 进阶实战

- [社区后端服务](https://github.com/go-dev-frame/sponge_examples/tree/main/7_community-single)
- [社区服务微服务拆分](https://github.com/go-dev-frame/sponge_examples/tree/main/8_community-cluster)
- [电商分布式订单系统](https://github.com/go-dev-frame/sponge_examples/tree/main/9_order-grpc-distributed-transaction)
- [电商秒杀系统 (DTM + FlashSale)](https://github.com/go-dev-frame/sponge_examples/tree/main/_12_sponge-dtm-flashSale)
- [电商系统](https://github.com/go-dev-frame/sponge_examples/tree/main/_14_eshop)
- [家电零售管理](https://github.com/go-dev-frame/sponge_examples/tree/main/_15_appliance_store)

<br>

### 社区与贡献

欢迎 Issue/PR！[贡献指南](https://go-sponge.com/zh/community/contribution.html)。

欢迎加入**go sponge微信群交流**，加微信(备注`sponge`)进群。

<img width="300px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/wechat-group.jpg">

<br>

如果 Sponge 对您有帮助，请给个 ⭐ Star！这将是我们持续更新的动力。

<br>