package staticfs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTestServer Create a gin engine and temporary directory structure for testing purposes
func setupTestServer(t *testing.T, opts ...ListDirOption) (*gin.Engine, string) {
	tmpDir, err := os.MkdirTemp("", "test-staticfs-")
	assert.NoError(t, err)

	// Create test files and directory structure
	//- /
	//  - sub_dir/
	//    - file3.txt (older)
	//  - .git/ (sensitive)
	//  - file1.txt (10 bytes)
	//  - file2.log (20 bytes)
	//  - .env (sensitive)
	assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, "sub_dir"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sub_dir", "file3.txt"), []byte("sub"), 0644))
	// Modify file time for testing sorting
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	assert.NoError(t, os.Chtimes(filepath.Join(tmpDir, "sub_dir", "file3.txt"), twoDaysAgo, twoDaysAgo))

	assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".git"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("0123456789"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.log"), []byte("01234567890123456789"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("secret"), 0644))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ListDir(r, opts...)

	// Use t.Cleanup to automatically delete temporary directories
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return r, tmpDir
}

func newRequest(t *testing.T, r *gin.Engine, method, url string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, url, nil)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListDirOptions(t *testing.T) {
	t.Run("DefaultOptions", func(t *testing.T) {
		o := defaultListDirOptions()
		assert.True(t, o.enableFilter)
		assert.False(t, o.enableDownload)
		assert.Empty(t, o.prefixPath)
	})

	t.Run("WithOptions", func(t *testing.T) {
		o := defaultListDirOptions()
		opts := []ListDirOption{
			WithListDirPrefixPath("/static"),
			WithListDirDownload(),
			WithListDirFilter(false),
			WithListDirFilesFilter(".tmp"),
			WithListDirDirsFilter("/secret"),
		}
		o.apply(opts...)

		assert.Equal(t, "/static", o.prefixPath)
		assert.True(t, o.enableDownload)
		assert.False(t, o.enableFilter)
		assert.Contains(t, sensitiveFiles, ".tmp")
		assert.Contains(t, sensitiveDirs, "/secret")
	})
}

func TestIsAllowedPath(t *testing.T) {
	originalSensitiveDirs := sensitiveDirs
	originalSensitiveFiles := sensitiveFiles
	t.Cleanup(func() {
		sensitiveDirs = originalSensitiveDirs
		sensitiveFiles = originalSensitiveFiles
	})
	sensitiveDirs = append(sensitiveDirs, "/test_dir")
	sensitiveFiles = append(sensitiveFiles, ".test_file")

	testCases := []struct {
		name         string
		path         string
		enableFilter bool
		expected     bool
	}{
		{"Allowed", "/home/user/file.txt", true, true},
		{"DeniedDir", "/proc/1/status", true, false},
		{"DeniedFile", "/home/user/.git/config", true, false},
		{"CustomDeniedDir", "/test_dir/somefile", true, false},
		{"CustomDeniedFile", "config.test_file", true, false},
		{"FilterDisabled", "/proc/version", false, true},
		{"RootDenied", "/root", true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isAllowedPath(tc.path, tc.enableFilter))
		})
	}
}

func TestFormatSize(t *testing.T) {
	testCases := []struct {
		name     string
		size     int64
		expected string
	}{
		{"Bytes", 100, "100 B"},
		{"KB", 2048, "2.00 KB"},
		{"MB", 3 * 1024 * 1024, "3.00 MB"},
		{"GB", 4 * 1024 * 1024 * 1024, "4.00 GB"},
		{"FractionalKB", 1536, "1.50 KB"},
		{"Zero", 0, "0 B"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, formatSize(tc.size))
		})
	}
}

func TestListDirectory(t *testing.T) {
	_, tmpDir := setupTestServer(t)

	t.Run("ListWithFilter", func(t *testing.T) {
		files, err := listDirectory(tmpDir, true)
		assert.NoError(t, err)
		assert.Len(t, files, 3) // sub_dir, file1.txt, file2.log
		names := []string{}
		for _, f := range files {
			names = append(names, f.Name)
		}
		assert.Contains(t, names, "sub_dir")
		assert.Contains(t, names, "file1.txt")
		assert.Contains(t, names, "file2.log")
		assert.NotContains(t, names, ".git")
		assert.NotContains(t, names, ".env")
	})

	t.Run("ListWithoutFilter", func(t *testing.T) {
		files, err := listDirectory(tmpDir, false)
		assert.NoError(t, err)
		assert.Len(t, files, 5) // all files and dirs
		names := []string{}
		for _, f := range files {
			names = append(names, f.Name)
		}
		assert.Contains(t, names, ".git")
		assert.Contains(t, names, ".env")
	})

	t.Run("NonExistentDir", func(t *testing.T) {
		_, err := listDirectory(filepath.Join(tmpDir, "non-existent"), true)
		assert.Error(t, err)
	})
}

func TestSortFiles(t *testing.T) {
	now := time.Now()
	files := []FileInfo{
		{Name: "C", Size: 300, ModTime: now.Add(-time.Hour)},
		{Name: "a", Size: 100, ModTime: now},
		{Name: "B", Size: 200, ModTime: now.Add(-2 * time.Hour)},
	}

	t.Run("SortByNameAsc", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "name", "asc")
		assert.Equal(t, "a", sorted[0].Name)
		assert.Equal(t, "B", sorted[1].Name)
		assert.Equal(t, "C", sorted[2].Name)
	})

	t.Run("SortByNameDesc", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "name", "desc")
		assert.Equal(t, "C", sorted[0].Name)
		assert.Equal(t, "B", sorted[1].Name)
		assert.Equal(t, "a", sorted[2].Name)
	})

	t.Run("SortBySizeAsc", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "size", "asc")
		assert.Equal(t, int64(100), sorted[0].Size)
		assert.Equal(t, int64(200), sorted[1].Size)
		assert.Equal(t, int64(300), sorted[2].Size)
	})

	t.Run("SortBySizeDesc", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "size", "desc") // default sort
		assert.Equal(t, int64(300), sorted[0].Size)
		assert.Equal(t, int64(200), sorted[1].Size)
		assert.Equal(t, int64(100), sorted[2].Size)
	})

	t.Run("SortByTimeAsc", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "time", "asc")
		assert.Equal(t, "B", sorted[0].Name) // oldest
		assert.Equal(t, "C", sorted[1].Name)
		assert.Equal(t, "a", sorted[2].Name) // newest
	})

	t.Run("SortByTimeDesc", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "time", "desc")
		assert.Equal(t, "a", sorted[0].Name) // newest
		assert.Equal(t, "C", sorted[1].Name)
		assert.Equal(t, "B", sorted[2].Name) // oldest
	})

	t.Run("DefaultSort (name asc)", func(t *testing.T) {
		sorted := make([]FileInfo, len(files))
		copy(sorted, files)
		sortFiles(sorted, "invalid-sort-key", "asc")
		assert.Equal(t, "a", sorted[0].Name)
	})
}

func TestToggleOrder(t *testing.T) {
	assert.Equal(t, "desc", toggleOrder("asc"))
	assert.Equal(t, "asc", toggleOrder("desc"))
	assert.Equal(t, "asc", toggleOrder("something-else"))
}

func TestHandleList(t *testing.T) {
	r, tmpDir := setupTestServer(t)

	t.Run("ValidRequest", func(t *testing.T) {
		url := fmt.Sprintf("/dir/list?dir=%s", tmpDir)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)
		body, _ := io.ReadAll(w.Body)
		assert.Contains(t, string(body), "<h1>Listing Directory</h1>")
		assert.Contains(t, string(body), "file1.txt")
		assert.Contains(t, string(body), "20 B") // size of file2.log
		assert.NotContains(t, string(body), ".env")
	})

	t.Run("MissingDirParam", func(t *testing.T) {
		w := newRequest(t, r, "GET", "/dir/list")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		body, _ := io.ReadAll(w.Body)
		assert.Contains(t, string(body), "dir parameter is required")
	})

	t.Run("NonExistentDir", func(t *testing.T) {
		url := fmt.Sprintf("/dir/list?dir=%s", filepath.Join(tmpDir, "non-existent"))
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		body, _ := io.ReadAll(w.Body)
		assert.Contains(t, string(body), "failed to read directory")
	})

	t.Run("Pagination", func(t *testing.T) {
		// create more files to test pagination
		for i := 0; i < 25; i++ {
			assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("page_test_%02d.txt", i)), []byte(""), 0644))
		}

		url := fmt.Sprintf("/dir/list?dir=%s&page=1", tmpDir)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)
		body, _ := io.ReadAll(w.Body)
		assert.Contains(t, string(body), "Page 1 / 2")       // 3 + 25 = 28 files, pageSize 20 -> 2 pages
		assert.Contains(t, string(body), "page_test_15.txt") // last file on page 1, default sort by size=desc
		assert.NotContains(t, string(body), "page_test_20.txt")

		url = fmt.Sprintf("/dir/list?dir=%s&page=2", tmpDir)
		w = newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)
		body, _ = io.ReadAll(w.Body)
		assert.Contains(t, string(body), "Page 2 / 2")
		assert.Contains(t, string(body), "page_test_24.txt")
	})

	t.Run("ParentDirLink", func(t *testing.T) {
		subDirPath := filepath.Join(tmpDir, "sub_dir")
		url := fmt.Sprintf("/dir/list?dir=%s&root=%s", subDirPath, tmpDir)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)
		body, _ := io.ReadAll(w.Body)
		// Expect a "Back to Parent" link that points to tmpDir
		assert.Contains(t, string(body), "ctest-staticfs-")
	})
}

func TestHandleDownload(t *testing.T) {
	r, tmpDir := setupTestServer(t, WithListDirDownload())

	t.Run("ValidDownload", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "file1.txt")
		url := fmt.Sprintf("/dir/file/download?path=%s", filePath)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, `attachment; filename="file1.txt"`, w.Header().Get("Content-Disposition"))
		body, _ := io.ReadAll(w.Body)
		assert.Equal(t, "0123456789", string(body))
	})

	t.Run("InvalidPath", func(t *testing.T) {
		w := newRequest(t, r, "GET", "/dir/file/download?path=")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("DisallowedPath", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, ".env")
		url := fmt.Sprintf("/dir/file/download?path=%s", filePath)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPIList(t *testing.T) {
	r, tmpDir := setupTestServer(t)

	t.Run("ValidAPIRequest", func(t *testing.T) {
		url := fmt.Sprintf("/dir/list/api?dir=%s&sort=name&order=asc", tmpDir)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)

		var respData struct {
			Dir   string     `json:"dir"`
			Sort  string     `json:"sort"`
			Order string     `json:"order"`
			Files []FileInfo `json:"files"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &respData)
		assert.NoError(t, err)

		assert.Equal(t, tmpDir, respData.Dir)
		assert.Equal(t, "name", respData.Sort)
		assert.Equal(t, "asc", respData.Order)
		assert.Len(t, respData.Files, 3)
		assert.Equal(t, "file1.txt", respData.Files[0].Name) // sorted by name asc
	})

	t.Run("APINonExistentDir", func(t *testing.T) {
		url := fmt.Sprintf("/dir/list/api?dir=%s", filepath.Join(tmpDir, "non-existent"))
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestListDirRouterSetup(t *testing.T) {
	t.Run("NoPrefix", func(t *testing.T) {
		r, _ := setupTestServer(t)
		w := newRequest(t, r, "GET", "/dir/list?dir=.")
		assert.Equal(t, http.StatusOK, w.Code)
		wApi := newRequest(t, r, "GET", "/dir/list/api?dir=.")
		assert.Equal(t, http.StatusOK, wApi.Code)
		// Download route should not be registered by default
		wDownload := newRequest(t, r, "GET", "/dir/file/download?path=.")
		assert.Equal(t, http.StatusNotFound, wDownload.Code)
	})

	t.Run("WithPrefix", func(t *testing.T) {
		r, _ := setupTestServer(t, WithListDirPrefixPath("/static/files"))
		w := newRequest(t, r, "GET", "/static/files/dir/list?dir=.")
		assert.Equal(t, http.StatusOK, w.Code)
		wApi := newRequest(t, r, "GET", "/static/files/dir/list/api?dir=.")
		assert.Equal(t, http.StatusOK, wApi.Code)
	})

	t.Run("WithDownloadEnabled", func(t *testing.T) {
		r, tmpDir := setupTestServer(t, WithListDirDownload())
		filePath := filepath.Join(tmpDir, "file1.txt")
		url := fmt.Sprintf("/dir/file/download?path=%s", filePath)
		w := newRequest(t, r, "GET", url)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("WithPrefixSlashVariations", func(t *testing.T) {
		testCases := []string{"/prefix", "prefix/", "/prefix/", "prefix"}
		for _, prefix := range testCases {
			r, _ := setupTestServer(t, WithListDirPrefixPath(prefix))
			w := newRequest(t, r, "GET", "/prefix/dir/list?dir=.")
			assert.Equal(t, http.StatusOK, w.Code, fmt.Sprintf("Failed with prefix: %s", prefix))
		}
	})
}
