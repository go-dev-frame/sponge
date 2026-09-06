Requires [Go 1.27.1+](https://go.dev/doc/install) version.

<br>

---

<br>

### Windows Environment

#### Preparations

Ensure Go is installed and the `GOBIN` environment variable is configured:

```bash
# Check if GOBIN is set (empty output means not set)
go env GOBIN

# If not set, configure it (modify the example path according to your actual setup)
go env -w GOBIN=D:\go\bin
# Then add the GOBIN path to the system PATH environment variable
```

<br>

#### Quick Install sponge

We provide an integrated installation package that includes all dependencies, download address: [**sponge-install.zip**](https://drive.google.com/drive/folders/1T55lLXDBIQCnL5IQ-i1hWJovgLI2l0k1?usp=sharing)

<br>

**Installation Steps**:

1. After unzipping, run `install.bat`
    - Keep default options during Git installation (skip if already installed)
2. Open Git Bash terminal, right-click → [Open Git Bash here]
   ```bash
   sponge init          # Initialize and install dependencies
   sponge plugins       # View installed plugins
   sponge -v            # View version
   ```

   **Note**: Always use Git Bash. Do not use the default Windows cmd terminal, otherwise you may encounter 'command not found' errors.

<br>

> The above is the installation method for the integrated package. Native installation of sponge is also supported, see details at:
> 👉 **[[Install Sponge] → [Windows Environment]](https://go-sponge.com/getting-started/install.html#install-sponge)**

<br>

---

<br>

### Linux/MacOS Environment

#### Environment Configuration

Configure environment variables (skip if already configured):

```bash
vim ~/.bashrc
```

Add the following content (modify paths according to your actual setup):
```bash
export GOROOT="/opt/go"       # Go installation directory
export GOPATH=$HOME/go        # Go module download directory
export GOBIN=$GOPATH/bin      # Directory for executable files
export PATH=$PATH:$GOBIN:$GOROOT/bin
```

Apply configuration:

```bash
source ~/.bashrc
go env GOBIN  # Verify configuration success, non-empty output indicates success
```

<br>

#### Installation Steps

1. Install protoc:
    - Download link: [protoc v31.1](https://github.com/protocolbuffers/protobuf/releases/tag/v31.1)
    - Place the `protoc` executable file into the `GOBIN` directory

2. Install sponge:
   ```bash
   go install github.com/go-dev-frame/sponge/cmd/sponge@latest
   sponge init          # Initialize and install dependencies
   sponge plugins       # View installed plugins
   sponge -v            # View version
   ```

<br>

---

<br>

### Docker Environment

> ⚠ Warning: The Docker version only supports UI code generation. If you need to develop based on the generated service code, you still need to install the full sponge environment locally.

#### Quick Start

**Option 1: Docker Command**
```bash
docker run -d --name sponge -p 24631:24631 zhufuyi/sponge:latest -a http://<Host IP>:24631
```

<br>

**Option 2: docker-compose**
```yaml
version: "3.7"
services:
  sponge:
    image: zhufuyi/sponge:latest
    container_name: sponge
    restart: always
    command: ["-a","http://<Host IP>:24631"]
    ports:
      - "24631:24631"
```
Start command:
```bash
docker-compose up -d
```

Access URL: `http://<Host IP>:24631`
