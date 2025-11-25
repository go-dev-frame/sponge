## English | [简体中文](assets/readme-cn.md)

<p align="center">
<img width="500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/logo.png">
</p>

<div align=center>

[![Go Report](https://goreportcard.com/badge/github.com/go-dev-frame/sponge)](https://goreportcard.com/report/github.com/go-dev-frame/sponge)
[![codecov](https://codecov.io/gh/go-dev-frame/sponge/branch/main/graph/badge.svg)](https://codecov.io/gh/go-dev-frame/sponge)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-dev-frame/sponge.svg)](https://pkg.go.dev/github.com/go-dev-frame/sponge)
[![Go](https://github.com/go-dev-frame/sponge/workflows/Go/badge.svg)](https://github.com/go-dev-frame/sponge/actions)
[![Awesome Go](https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg)](https://github.com/avelino/awesome-go)
[![License: MIT](https://img.shields.io/github/license/go-dev-frame/sponge)](https://img.shields.io/github/license/go-dev-frame/sponge)

</div>

---

<br>

### Introduction

**Sponge** is a powerful and easy-to-use **Go development framework**. Built on the core philosophy of **"Definition is Code"**, it aims to reshape the backend development experience through automatic generation technology, unleashing productivity and boosting development efficiency.

Sponge deeply integrates a **code generation engine**, **Gin (Web framework)**, and **gRPC (microservice framework)**, covering the full software lifecycle from project generation, development, and testing to API documentation and deployment.

**Core Features:**

*   **Definition-Driven Development**: Automatically parses SQL, Protobuf, and JSON configuration files to generate high-quality modular code.
*   **LEGO-style Assembly**: Flexibly combines code modules in a loosely coupled manner, supporting the rapid construction of **monolithic applications**, **microservice clusters**, and **gateway services**, including `RESTful API`, `gRPC`, `HTTP+gRPC`, `gRPC Gateway`, etc.
*   **Low Code, High Efficiency**: Completely eliminates tedious repetitive work such as CRUD, routing configuration, and service governance. Developers only need to focus on core business logic to deliver standardized, high-quality backend services.

<br>

### Why Choose Sponge?

- **Extreme Development Efficiency**: One-click generation of complete, production-ready backend services (Zero-Code), including CRUD, routing, documentation, and service framework code, multiplying development efficiency.
- **Out-of-the-box Toolchain**: Provides a complete development toolchain (Generation → Business Logic → Testing → Deployment → Monitoring), eliminating the need to piece together fragmented tools.
- **Industry Best Practices**: Built on the mainstream Go community tech stack (Gin/GORM/gRPC/Protobuf, etc.), with a standardized architecture to reduce technology selection risks.
- **Extremely Low Learning Curve**: Easy to get started and beginner-friendly, while also meeting the customization needs of senior developers.
- **Ideal for Team Collaboration**: Unifies project structure, improving team collaboration efficiency and code maintainability.
* **Flexible and Extensible**: Supports custom templates, not limited to Go; capable of extending to frontend, test scripts, and other arbitrary code generation.
- **AI-Driven Development (AI-Native)**:
   - **Sponge:** automatically builds standardized infrastructure (API, data layer, service framework).
   - **AI Assistant:** analyzes project context and generates code intelligently fill in core business logic based on project context, achieving "Infrastructure Automation, Business Intelligence".

<br>

### Applicable Scenarios

Sponge is suitable for building high-performance, maintainable backend systems, specifically for:

- **Rapid development of RESTful API services.**
- **Building large-scale microservice architectures.**
- **Cloud-native application development.**
- **Rapid refactoring and migration of legacy projects.**
- **Standardized engineering templates for Go beginners or teams.**

<br>

### Online Demo

No installation required, experience the code generation feature directly in your browser: [**Code Generation**](https://go-sponge.com/en/ui)

> Note: If you need to run the downloaded service code locally, you must first complete the local installation of Sponge.

<br>

### Quick Start

1. **Install Sponge**: Supports Windows, macOS, Linux, and Docker environments. View the [**Sponge Installation Guide**](https://github.com/go-dev-frame/sponge/blob/main/assets/install-en.md).

2. **Open the Code Generation UI Page**

   ```bash
   sponge run
   ```

   Access `http://localhost:24631` in your local browser to generate code.

3. **Example: One-click Generation of Web Service Backend Code Based on SQL**

   <p align="center">
   <img width="1500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/en_sponge-ui.png">
   </p>

    - Operation Process:
        - **Fill Parameters**: Connect to the database and select tables in the UI.
        - **Download Code**: Click generate and download the complete project package.
        - **Generate Swagger Documentation**: Execute `make docs` in the project root directory.
        - **Start Service**: Execute `make run`.
        - **Test Interface**: Access `http://localhost:8080/swagger/index.html` in the browser to test APIs via the Swagger interface.

<br>

### Tech Stack & Components

Sponge follows the "batteries included" principle, integrating 30+ mainstream Go ecosystem components, loaded on demand:

| Category               | Component                                                |
|:-----------------------|:---------------------------------------------------------|
| **Frameworks**         | Gin, gRPC                                                |
| **Database**           | GORM (MySQL, PostgreSQL, SQLite, etc.), MongoDB          |
| **Cache/Messaging**    | Redis, Kafka, RabbitMQ                                   |
| **Service Governance** | Etcd, Consul, Nacos, Jaeger, Prometheus, OpenTelemetry   |
| **Others**             | DTM (Distributed Transaction), WebSocket, Swagger, PProf |
| ...                    | ...                                                      |

👉 [**View the complete list of tech stacks and components**](https://go-sponge.com/component/).

<br>

### Architecture & Principles

#### Code Generation Engine

Sponge provides multiple code generation engines, supporting **built-in templates**, **custom templates**, and **AI-assisted generation**.

1. Code generation engine based on **built-in templates**, as shown below:

<p align="center">
<img width="1500px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/sponge-framework.png">
</p>

<br>

2. Code generation engine based on **custom templates**, as shown below:

<p align="center">
<img width="1200px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/template-framework.png">
</p>

<br>

3. Code generation engine based on **AI-assisted business logic**, as shown below:

<p align="center">
<img width="1200px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/ai-assistant-framework.png">
</p>

<br>

#### Microservice Architecture

The code generated by Sponge follows a typical layered architecture with built-in service governance capabilities. The structure is clear and easy to maintain. A modern microservice framework structure is shown below:

<p align="center">
<img width="1000px" src="https://raw.githubusercontent.com/go-dev-frame/sponge/main/assets/en_microservices-framework.png">
</p>

<br>

### Performance Benchmarks

Based on tests with 50 concurrency and 1 million requests, services generated by Sponge demonstrate excellent performance:

1. **HTTP Service**
   ![http-server](https://raw.githubusercontent.com/zhufuyi/microservices_framework_benchmark/main/test/assets/http-server.png)

2. **gRPC Service**
   ![grpc-server](https://raw.githubusercontent.com/zhufuyi/microservices_framework_benchmark/main/test/assets/grpc-server.png)

👉 [**View detailed test code and environment**](https://github.com/zhufuyi/microservices_framework_benchmark).

<br>

### Directory Structure

The service code directory structure created by Sponge follows the [project-layout](https://github.com/golang-standards/project-layout), supporting **Monolith**, **Multi-Repo**, and **Mono-Repo** patterns.

<details>
<summary> <b>1. Monolith / Multi-Repo Structure Details.</b> </summary>

```bash
   .
   ├── api            # Directory for protobuf files and generated *pb.go files
   ├── assets         # Directory for other assets used with the repository (images, logos, etc.)
   ├── cmd            # Main application entry directory
   ├── configs        # Directory for configuration files
   ├── deployments    # Deployment scripts directory for bare metal, docker, k8s
   ├── docs           # Directory for design documents and interface documentation
   ├── internal       # Private application and library code directory
   │    ├── cache        # Cache directory wrapped based on business logic
   │    ├── config       # Go structure configuration file directory
   │    ├── dao          # Data access directory
   │    ├── database     # Database directory
   │    ├── ecode        # Custom business error code directory
   │    ├── handler      # HTTP business functionality implementation directory
   │    ├── model        # Database model directory
   │    ├── routers      # HTTP routing directory
   │    ├── rpcclient    # Client directory for connecting to gRPC services
   │    ├── server       # Service entry, including http, grpc, etc.
   │    ├── service      # gRPC business functionality implementation directory
   │    └── types        # HTTP request and response types directory
   ├── pkg            # Library directory that can be used by external applications
   ├── scripts        # Execution scripts directory
   ├── test           # Additional external test applications and test data
   ├── third_party    # Directory for dependent third-party protobuf files or other tools
   ├── Makefile       # Collection of commands related to development, testing, and deployment
   ├── go.mod         # Go module dependency and version control file
   └── go.sum         # Go module dependency checksum file
```
</details>

<details>
<summary> <b>2. Mono-Repo Structure Details.</b> </summary>

```bash
   .
   ├── api
   │    ├── server1       # Protobuf files and generated *pb.go directory for Service 1
   │    ├── server2       # Protobuf files and generated *pb.go directory for Service 2
   │    ├── server3       # Protobuf files and generated *pb.go directory for Service 3
   │    └── ...
   ├── server1        # Code directory for Service 1, basically same structure as multi-repo
   ├── server2        # Code directory for Service 2, basically same structure as multi-repo
   ├── server3        # Code directory for Service 3, basically same structure as multi-repo
   ├── ...
   ├── third_party    # Dependent third-party protobuf files
   ├── go.mod         # Go module dependency and version control file
   └── go.sum         # Go module dependency checksum file
```
</details>

<br>

### Documentation

Click to view the [Official Sponge Documentation](https://go-sponge.com/), covering core content such as development guides, components, service configuration, and deployment solutions.

<br>

### Example Projects

#### Basic Examples

- [Web Service (based on SQL, includes CRUD)](https://github.com/go-dev-frame/sponge_examples/tree/main/1_web-gin-CRUD)
- [gRPC Microservice (based on SQL, includes CRUD)](https://github.com/go-dev-frame/sponge_examples/tree/main/2_micro-grpc-CRUD)
- [Web Service (based on Protobuf)](https://github.com/go-dev-frame/sponge_examples/tree/main/3_web-gin-protobuf)
- [gRPC Microservice (based on Protobuf)](https://github.com/go-dev-frame/sponge_examples/tree/main/4_micro-grpc-protobuf)
- [gRPC Gateway Service (Gateway)](https://github.com/go-dev-frame/sponge_examples/tree/main/5_micro-gin-rpc-gateway)
- [gRPC+HTTP Microservice (based on Protobuf)](https://github.com/go-dev-frame/sponge_examples/tree/main/_10_micro-grpc-http-protobuf)

#### Advanced Projects

- [Community Backend Service](https://github.com/go-dev-frame/sponge_examples/tree/main/7_community-single)
- [Community Service Microservice Split](https://github.com/go-dev-frame/sponge_examples/tree/main/8_community-cluster)
- [E-commerce Distributed Order System](https://github.com/go-dev-frame/sponge_examples/tree/main/9_order-grpc-distributed-transaction)
- [E-commerce Flash Sale System (DTM + FlashSale)](https://github.com/go-dev-frame/sponge_examples/tree/main/_12_sponge-dtm-flashSale)
- [E-shop System](https://github.com/go-dev-frame/sponge_examples/tree/main/_14_eshop)
- [Appliance Store Management](https://github.com/go-dev-frame/sponge_examples/tree/main/_15_appliance_store)

<br>

### Contribution

Issues and PRs are welcome! [Contribution Guide](https://go-sponge.com/community/contribution.html).

<br>

If Sponge is helpful to you, please give it a ⭐ Star! This will be our motivation for continuous updates.

<br>
