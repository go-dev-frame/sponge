
## Changelog

### 1. **Enhanced TLS Encryption Support**

The web service now supports multiple TLS encryption methods, including automatic certificate generation, manual certificate upload, Let's Encrypt auto-renewal, and dynamic certificate retrieval via remote APIs. Additionally, it features extensible architecture for dynamically obtaining certificates from configuration centers like etcd and Consul.

### 2. **Route Reverse Proxy Capability**

The web services support reverse proxy capabilities similar to Nginx, supporting flexible route configuration and request forwarding.

### 3. **Static Resource Serving**

Efficient serving of static files and web pages through route mapping, enabling one-stop hosting for frontend resources.

### 4. **Customizable CORS Middleware**

CORS middleware now supports fully customizable configurations to meet diverse cross-origin access requirements.

### 5. **Bug Fix**

Resolved PostgreSQL `timestamptz` type mapping issue [#141](https://github.com/go-dev-frame/sponge/issues/141).
