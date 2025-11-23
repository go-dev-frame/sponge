package staticfs

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListDirOption set options.
type ListDirOption func(*listDirOptions)

type listDirOptions struct {
	prefixPath     string
	enableDownload bool // default: false
	enableFilter   bool // default: true
	middlewares    []gin.HandlerFunc
}

func (o *listDirOptions) apply(opts ...ListDirOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultListDirOptions() *listDirOptions {
	return &listDirOptions{
		enableFilter: true,
	}
}

// WithListDirPrefixPath sets prefix path.
func WithListDirPrefixPath(prefixPath string) ListDirOption {
	return func(o *listDirOptions) {
		o.prefixPath = prefixPath
	}
}

// WithListDirDownload enables download feature.
func WithListDirDownload() ListDirOption {
	return func(o *listDirOptions) {
		o.enableDownload = true
	}
}

// WithListDirFilter enables file filter feature.
func WithListDirFilter(enable bool) ListDirOption {
	return func(o *listDirOptions) {
		o.enableFilter = enable
	}
}

// WithListDirFilesFilter sets file name filter.
func WithListDirFilesFilter(filters ...string) ListDirOption {
	return func(o *listDirOptions) {
		sensitiveFiles = append(sensitiveFiles, filters...)
	}
}

// WithListDirDirsFilter sets directory name filter.
func WithListDirDirsFilter(filters ...string) ListDirOption {
	return func(o *listDirOptions) {
		sensitiveDirs = append(sensitiveDirs, filters...)
	}
}

// WithListDirMiddlewares sets middlewares.
func WithListDirMiddlewares(middlewares ...gin.HandlerFunc) ListDirOption {
	return func(o *listDirOptions) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// -------------------------------------------------------------------------------------------

// FileInfo is a struct that represents a file or directory in the file system.
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size,omitempty"`
	ModTime time.Time `json:"mod_time,omitempty"`
}

// default file filters
var sensitiveDirs = []string{"/proc", "/sys", "/dev", "/run", "/boot", "/root", "/etc"}
var sensitiveFiles = []string{".git", ".env", ".DS_Store"}

func isAllowedPath(p string, enableFilter bool) bool {
	if !enableFilter {
		return true
	}
	for _, s := range sensitiveDirs {
		if strings.HasPrefix(p, s) {
			return false
		}
	}
	for _, f := range sensitiveFiles {
		if strings.Contains(p, f) {
			return false
		}
	}
	return true
}

// nolint
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func listDirectory(dir string, enableFilter bool) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if !isAllowedPath(fullPath, enableFilter) {
			continue
		}

		info, _ := entry.Info()
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    fullPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return files, nil
}

func sortFiles(files []FileInfo, sortBy, order string) {
	desc := order != "asc" // default: desc
	switch sortBy {
	case "time":
		sort.Slice(files, func(i, j int) bool {
			if desc {
				return files[i].ModTime.After(files[j].ModTime)
			}
			return files[i].ModTime.Before(files[j].ModTime)
		})
	case "size":
		sort.Slice(files, func(i, j int) bool {
			if desc {
				return files[i].Size > files[j].Size
			}
			return files[i].Size < files[j].Size
		})
	default: // name
		sort.Slice(files, func(i, j int) bool {
			if desc {
				return strings.ToLower(files[i].Name) > strings.ToLower(files[j].Name)
			}
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})
	}
}

func toggleOrder(current string) string {
	if current == "asc" {
		return "desc"
	}
	return "asc"
}

func badRequestData(data any) gin.H {
	return gin.H{"code": 400, "msg": "not found", "data": data}
}

func handleList(prefixPath string, o *listDirOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		dir := c.Query("dir")
		root := c.Query("root")
		sortBy := c.DefaultQuery("sort", "size")
		order := c.DefaultQuery("order", "desc")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		pageSize := 20 // Default display of 20 files per page.

		if dir == "" {
			c.JSON(http.StatusBadRequest, badRequestData("dir parameter is required, e.g. /list?dir=/tmp/dist"))
			return
		}
		if root == "" {
			root = dir
		}

		files, err := listDirectory(dir, o.enableFilter)
		if err != nil {
			c.JSON(http.StatusBadRequest, badRequestData(fmt.Sprintf("failed to read directory: %v", err)))
			return
		}
		sortFiles(files, sortBy, order)

		// Calculate pagination information
		totalFiles := len(files)
		totalPages := (totalFiles + pageSize - 1) / pageSize
		if page > totalPages && totalPages > 0 {
			page = totalPages
		}

		// Pagination
		startIndex := (page - 1) * pageSize
		endIndex := startIndex + pageSize
		if endIndex > totalFiles {
			endIndex = totalFiles
		}

		// Retrieve the file of the current page
		var pagedFiles []FileInfo
		if startIndex < totalFiles {
			pagedFiles = files[startIndex:endIndex]
		}

		var parentDir string
		if dir != root {
			parentDir = filepath.Dir(strings.TrimRight(dir, "/"))
			if parentDir == "" {
				parentDir = "/"
			}
		}

		// Template FuncMap
		funcMap := template.FuncMap{
			"ToUpper":    strings.ToUpper,
			"FormatSize": formatSize,
		}

		tmpl := template.Must(template.New("list-dir").Funcs(funcMap).Parse(htmlTextSrc))

		c.Header("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.Execute(c.Writer, gin.H{
			"Dir":            dir,
			"Root":           root,
			"ParentDir":      parentDir,
			"Files":          pagedFiles,
			"SortBy":         sortBy,
			"Order":          order,
			"NextOrder":      toggleOrder(order),
			"EnableFileMeta": true,
			"EnableDownload": o.enableDownload,
			"ListPath":       prefixPath + "/dir/list",
			"DownloadPath":   prefixPath + "/dir/file/download",
			"CurrentPage":    page,
			"TotalPages":     totalPages,
			"HasPrevPage":    page > 1,
			"HasNextPage":    page < totalPages,
			"PrevPage":       page - 1,
			"NextPage":       page + 1,
		})
	}
}

func handleDownload(c *gin.Context) {
	path := c.Query("path")
	if path == "" || !isAllowedPath(path, true) {
		c.JSON(http.StatusBadRequest, badRequestData("invalid file path"))
		return
	}
	c.FileAttachment(path, filepath.Base(path))
}

func handleAPIList(enableFilter bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		dir := c.Query("dir")
		sortBy := c.DefaultQuery("sort", "name")
		order := c.DefaultQuery("order", "desc")

		files, err := listDirectory(dir, enableFilter)
		if err != nil {
			c.JSON(http.StatusBadRequest, badRequestData(fmt.Sprintf("failed to read directory: %v", err)))
			return
		}

		sortFiles(files, sortBy, order)
		c.JSON(http.StatusOK, gin.H{
			"dir":   dir,
			"sort":  sortBy,
			"order": order,
			"files": files,
		})
	}
}

// ListDir registers the routes for serving static files.
func ListDir(r *gin.Engine, opts ...ListDirOption) {
	o := defaultListDirOptions()
	o.apply(opts...)

	prefixPath := o.prefixPath
	if prefixPath != "" {
		if !strings.HasPrefix(prefixPath, "/") {
			prefixPath = "/" + prefixPath
		}
		prefixPath = strings.TrimSuffix(prefixPath, "/")
	}
	if prefixPath == "/" {
		prefixPath = ""
	}

	if len(o.middlewares) > 0 {
		group := r.Group("", o.middlewares...)
		group.GET(prefixPath+"/dir/list", handleList(prefixPath, o))
		if o.enableDownload {
			group.GET(prefixPath+"/dir/file/download", handleDownload)
		}
		group.GET(prefixPath+"/dir/list/api", handleAPIList(o.enableFilter))
	} else {
		r.GET(prefixPath+"/dir/list", handleList(prefixPath, o))
		if o.enableDownload {
			r.GET(prefixPath+"/dir/file/download", handleDownload)
		}
		r.GET(prefixPath+"/dir/list/api", handleAPIList(o.enableFilter))
	}
}

// nolint
var htmlTextSrc = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Directory Listing: {{.Dir}}</title>
    <style>
        :root {
            --primary-color: #3498db;
            --secondary-color: #2980b9;
            --background-color: #f8f9fa;
            --card-color: #ffffff;
            --text-color: #333333;
            --border-color: #e0e0e0;
            --hover-color: #f1f7fc;
            --folder-color: #f39c12;
            --file-color: #7f8c8d;
            --header-bg: #f5f7fa;
            --sort-indicator-color: #3498db;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        body {
            background-color: var(--background-color);
            color: var(--text-color);
            line-height: 1.1;
            padding: 20px;
            max-width: 1200px;
            margin: 0 auto;
        }

        .container {
            background-color: var(--card-color);
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
            padding: 30px;
        }

        .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 20px;
            padding-bottom: 15px;
            border-bottom: 1px solid var(--border-color);
        }

        h1 {
            font-size: 24px;
            font-weight: 500;
            color: var(--primary-color);
        }

        .path-display {
            background-color: rgba(52, 152, 219, 0.1);
            padding: 10px 15px;
            border-radius: 6px;
            margin-bottom: 20px;
            overflow-x: auto;
            white-space: nowrap;
            font-family: monospace;
            font-size: 14px;
            border-left: 4px solid var(--primary-color);
        }

        .back-link {
            display: inline-flex;
            align-items: center;
            color: var(--primary-color);
            text-decoration: none;
            font-weight: 500;
            padding: 8px 16px;
            border-radius: 4px;
            transition: all 0.2s ease;
            margin-bottom: 20px;
            border: 1px solid var(--primary-color);
        }

        .back-link:hover {
            background-color: var(--primary-color);
            color: white;
        }

        .back-icon {
            margin-right: 8px;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
            table-layout: fixed;
        }

        th {
            background-color: var(--header-bg);
            text-align: left;
            padding: 12px 15px;
            font-weight: 600;
            color: var(--text-color);
            border-bottom: 2px solid var(--border-color);
            position: sticky;
            top: 0;
        }

        td {
            padding: 12px 15px;
            border-bottom: 1px solid var(--border-color);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        tr:hover {
            background-color: var(--hover-color);
        }

        th a {
            color: var(--text-color);
            text-decoration: none;
            align-items: center;
            justify-content: space-between;
        }

        th a:hover {
            color: var(--primary-color);
        }

        .sort-indicator {
            color: var(--sort-indicator-color);
            font-weight: bold;
            margin-left: 5px;
        }

        .file-link {
            display: flex;
            align-items: center;
            text-decoration: none;
            color: var(--text-color);
        }

        .file-link:hover {
            color: var(--primary-color);
        }

        .file-text {
            display: flex;
            align-items: center;
            color: var(--text-color);
        }

        .file-icon {
            margin-right: 10px;
            font-size: 18px;
        }

        .folder-icon {
            color: var(--folder-color);
        }

        .file-icon-regular {
            color: var(--file-color);
        }

        .size-cell {
            color: #666;
            font-size: 0.9em;
        }

        .date-cell {
            color: #666;
            font-size: 0.9em;
        }

        .empty-message {
            text-align: center;
            padding: 30px;
            color: #7f8c8d;
            font-style: italic;
        }

        .pagination {
            display: flex;
            justify-content: center;
            align-items: center;
            margin-top: 30px;
            flex-wrap: wrap;
        }

        .pagination-item {
            margin: 0 5px;
            padding: 8px 15px;
            border-radius: 4px;
            background-color: var(--background-color);
            color: var(--text-color);
            text-decoration: none;
            border: 1px solid var(--border-color);
            transition: all 0.2s ease;
        }

        .pagination-item:hover {
            background-color: var(--hover-color);
            border-color: var(--primary-color);
        }

        .pagination-item.active {
            background-color: var(--primary-color);
            color: white;
            border-color: var(--primary-color);
        }

        .pagination-item.disabled {
            opacity: 0.5;
            cursor: not-allowed;
            pointer-events: none;
        }

        .pagination-info {
            margin: 0 15px;
            color: var(--text-color);
        }

        .pagination-form {
            display: flex;
            align-items: center;
            margin-left: 15px;
        }

        .pagination-input {
            width: 60px;
            padding: 6px 10px;
            border-radius: 4px;
            border: 1px solid var(--border-color);
            margin: 0 5px;
        }

        .pagination-button {
            padding: 6px 12px;
            border-radius: 4px;
            background-color: var(--primary-color);
            color: white;
            border: none;
            cursor: pointer;
        }

        .pagination-button:hover {
            background-color: var(--secondary-color);
        }

        @media (max-width: 768px) {
            .container {
                padding: 20px 10px;
            }

            h1 {
                font-size: 20px;
            }

            th, td {
                padding: 8px;
            }

            .date-cell {
                display: none;
            }
            
            .pagination {
                flex-direction: column;
                gap: 10px;
            }
            
            .pagination-form {
                margin-top: 10px;
                margin-left: 0;
            }
        }

        @media (max-width: 480px) {
            .size-cell {
                display: none;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Listing Directory</h1>
        </div>

        <div class="path-display">
            {{.Dir}}
        </div>

        {{if .ParentDir}}
        <a href="{{$.ListPath}}?dir={{.ParentDir}}&root={{.Root}}&sort={{.SortBy}}&order={{.Order}}" class="back-link">
            <span class="back-icon">⬅</span> Back to Parent
        </a>
        {{end}}

        <table>
            <thead>
                <tr>
                    <th style="width: 60%">
                        <a href="?dir={{.Dir}}&root={{.Root}}&sort=name&order={{.NextOrder}}&page={{.CurrentPage}}">
                            Name
                            {{if eq .SortBy "name"}}
                                <span class="sort-indicator">{{if eq .Order "desc"}}⬇️{{else}}⬆️{{end}}</span>
                            {{end}}
                        </a>
                    </th>
                    {{if $.EnableFileMeta}}
                        <th style="width: 15%">
                            <a href="?dir={{.Dir}}&root={{.Root}}&sort=size&order={{.NextOrder}}&page={{.CurrentPage}}">
                                Size
                                {{if eq .SortBy "size"}}
                                    <span class="sort-indicator">{{if eq .Order "desc"}}⬇️{{else}}⬆️{{end}}</span>
                                {{end}}
                            </a>
                        </th>
                        <th style="width: 25%">
                            <a href="?dir={{.Dir}}&root={{.Root}}&sort=time&order={{.NextOrder}}&page={{.CurrentPage}}">
                                Modified Time
                                {{if eq .SortBy "time"}}
                                    <span class="sort-indicator">{{if eq .Order "desc"}}⬇️{{else}}⬆️{{end}}</span>
                                {{end}}
                            </a>
                        </th>
                    {{end}}
                </tr>
            </thead>
            <tbody>
                {{if .Files}}
                    {{range .Files}}
                    <tr>
                        <td>
                            {{if .IsDir}}
                                <a href="{{$.ListPath}}?dir={{.Path}}&root={{$.Root}}&sort={{$.SortBy}}&order={{$.Order}}" class="file-link">
                                    <span class="file-icon folder-icon">📁</span> {{.Name}}
                                </a>
                            {{else if $.EnableDownload}}
                                <a href="{{$.DownloadPath}}?path={{.Path}}" class="file-link">
                                    <span class="file-icon file-icon-regular">📄</span> {{.Name}}
                                </a>
                            {{else}}
                                <div class="file-text">
                                    <span class="file-icon file-icon-regular">📄</span> {{.Name}}
                                </div>
                            {{end}}
                        </td>
                        {{if $.EnableFileMeta}}
                            <td class="size-cell">{{if not .IsDir}}{{.Size | FormatSize}}{{end}}</td>
                            <td class="date-cell">{{.ModTime.Format "2006-01-02 15:04:05"}}</td>
                        {{end}}
                    </tr>
                    {{end}}
                {{else}}
                    <tr>
                        <td colspan="{{if $.EnableFileMeta}}3{{else}}1{{end}}" class="empty-message">
                            This directory is empty.
                        </td>
                    </tr>
                {{end}}
            </tbody>
        </table>
        
        {{if gt .TotalPages 1}}
        <div class="pagination">
            {{if .HasPrevPage}}
            <a href="?dir={{.Dir}}&root={{.Root}}&sort={{.SortBy}}&order={{.Order}}&page={{.PrevPage}}" class="pagination-item">
                Previous
            </a>
            {{else}}
            <span class="pagination-item disabled">Previous</span>
            {{end}}
            
            <span class="pagination-info">
                Page {{.CurrentPage}} / {{.TotalPages}}
            </span>
            
            {{if .HasNextPage}}
            <a href="?dir={{.Dir}}&root={{.Root}}&sort={{.SortBy}}&order={{.Order}}&page={{.NextPage}}" class="pagination-item">
                Next
            </a>
            {{else}}
            <span class="pagination-item disabled">Next</span>
            {{end}}
            
            <form class="pagination-form" action="{{$.ListPath}}" method="get">
                <input type="hidden" name="dir" value="{{.Dir}}">
                <input type="hidden" name="root" value="{{.Root}}">
                <input type="hidden" name="sort" value="{{.SortBy}}">
                <input type="hidden" name="order" value="{{.Order}}">
                <label>Go to: </label>
                <input type="number" name="page" min="1" max="{{.TotalPages}}" value="{{.CurrentPage}}" class="pagination-input">
                <button type="submit" class="pagination-button">Go</button>
            </form>
        </div>
        {{end}}
    </div>
</body>
</html>
`
