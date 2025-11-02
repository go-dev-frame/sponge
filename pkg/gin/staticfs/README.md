## StaticFS

StaticFS is a high-performance static file service designed for the Gin framework. It improves the access speed of static resources and reduces server load through various optimization methods. The main features include:

-   Automatically processes index files (such as `index.html`)
-   File system caching to reduce disk I/O operations
-   HTTP Cache Control to reduce client requests
-   Flexible configuration options to adapt to different scenario requirements
-   Advanced directory listing and file browser (HTML and JSON API)
-   File sorting (by name, size, modification time) and pagination
-   Optional file download functionality
-   Built-in security filter to hide sensitive files and directories

### 1. Static File Service Usage

This functionality is used as a FileServer in Gin. It allows you to map a URL path to a static file directory on the server.

**Directory Structure:**

```
.
├── static/
│   ├── index.html
│   ├── css/
│   │   └── style.css
│   └── js/
│       └── main.js
└── main.go
```


#### Example of Usage

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    "github.com/go-dev-frame/sponge/pkg/gin/staticfs" 
)

func main() {
    r := gin.Default()

   // Maps the URL prefix /user/ to the local /var/www/dist/index.html
   staticfs.StaticFS(r, "/user/", "/var/www/dist")

    log.Println("Server is running on http://localhost:8080")
    log.Println("Access static files at http://localhost:8080/user/")

    r.Run(":8080")
}
```

Now, you can access your static files at the following URLs:
-   `http://localhost:8080/user/` -> Serves the content of `static/index.html`
-   `http://localhost:8080/user/css/style.css` -> Serves the content of `static/css/style.css`

#### StaticFS Configuration Options

You can customize the behavior of `StaticFS` using the following option functions:

-   `WithStaticFSIndexFile(indexFile string)`: Sets the name of the index file to be processed automatically.
-   `WithCacheExpiration(duration time.Duration)`: sets the cache expiration time.
-   `WithCacheMaxAge(duration time.Duration)`: sets the Cache-Control max-age header value.
-   `WithCacheSize(size int)`:  sets the maximum number of entries in the file existence cache.
-   `WithMiddlewares(middlewares ...gin.HandlerFunc)`: sets middlewares for staticFS.

<br>

### 2. Directory Listing and File Browser

In addition to serving static files, StaticFS provides a powerful directory listing feature that can be used as a simple file browser. This feature registers its routes directly on the Gin engine and offers both an HTML interface and a JSON API.

#### Key Features

-   **Web UI Interface**: Browse server directories through a web page.
-   **JSON API**: Provides an API for programmatic access to directory contents.
-   **File Sorting**: Supports sorting by file name, size, or modification time in ascending or descending order.
-   **Pagination**: Automatically paginates the file list when a directory contains many files.
-   **File Download**: Optionally enables a file download feature.
-   **Security Filtering**: By default, it filters sensitive files and directories (e.g., `.git`, `.env`, `/proc`). Custom filter rules are also supported.

#### Example of Usage

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    "github.com/go-dev-frame/sponge/pkg/gin/staticfs"
)

func main() {
    r := gin.Default()

    // Register the directory listing routes /dir/list and /dir/list/api
    staticfs.ListDir(r)

    log.Println("Server is running on http://localhost:8080")
    log.Println("Access the file browser at: http://localhost:8080/dir/list?dir=/path/to/your/directory")
    
    r.Run(":8080")
}
```

#### Accessing URLs

-   **HTML Browser Interface**:
    `http://localhost:8080/dir/list?dir=/path/to/your/directory`
    -   `dir`: Required parameter. Specifies the directory path to browse.
    -   `sort`: Optional parameter. The basis for sorting (`name`, `size`, `time`).
    -   `order`: Optional parameter. The sort order (`asc`, `desc`).
    -   `page`: Optional parameter. The page number.

-   **File Download Endpoint** (requires `WithListDirDownload()`):
    `http://localhost:8080/dir/file/download?path=/path/to/your/file.txt`
    -   `path`: Required parameter. The full path of the file to download.

-   **JSON API Endpoint**:
    `http://localhost:8080/dir/list/api?dir=/path/to/your/directory`
    -   Returns JSON data of the files and subdirectories within the specified directory.

#### ListDir Configuration Options

You can customize the behavior of `ListDir` using the following option functions:

-   `WithListDirPrefixPath(prefix string)`: Sets a common URL prefix for all related routes.
-   `WithListDirDownload()`: Enables the file download feature and its corresponding `/dir/file/download` route.
-   `WithListDirFilter(enable bool)`: Enables or disables the security filter. It is `true` (enabled) by default.
-   `WithListDirFilesFilter(filters ...string)`: Adds custom file name filters. Matched files will be hidden.
-   `WithListDirDirsFilter(filters ...string)`: Adds custom directory path filters. Matched directories will be hidden.
