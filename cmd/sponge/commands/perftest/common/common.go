package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

const (
	BodyTypeJSON = "application/json"
	BodyTypeForm = "application/x-www-form-urlencoded"
	BodyTypeText = "text/plain"
)

// nolint
func ParseHTTPParams(method string, headers []string, body string, bodyFile string) ([]byte, map[string]string, error) {
	var bodyType string
	var bodyBytes []byte
	headerMap := make(map[string]string)
	for _, h := range headers {
		kvs := strings.SplitN(h, ":", 2)
		if len(kvs) == 2 {
			key := trimString(kvs[0])
			value := trimString(kvs[1])
			headerMap[key] = value
		}
	}

	if strings.ToUpper(method) == "GET" {
		return bodyBytes, headerMap, nil
	}

	if body != "" {
		body = strings.Trim(body, "'")
		bodyBytes = []byte(body)
	} else {
		if bodyFile != "" {
			data, err := os.ReadFile(bodyFile)
			if err != nil {
				return nil, nil, err
			}
			bodyBytes = data
		}
	}
	if len(bodyBytes) == 0 {
		return bodyBytes, headerMap, nil
	}

	for k, v := range headerMap {
		if strings.ToLower(k) == "content-type" && v != "" {
			bodyType = strings.ToLower(v)
		}
	}

	switch bodyType {
	case BodyTypeJSON:
		ok, err := isValidJSON(bodyBytes)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, fmt.Errorf("invalid JSON format")
		}
		return bodyBytes, headerMap, nil

	case BodyTypeForm:
		if bytes.Count(bodyBytes, []byte("=")) == 0 {
			return nil, nil, fmt.Errorf("invalid body form data format")
		}
		return bodyBytes, headerMap, nil

	case BodyTypeText:
		return bodyBytes, headerMap, nil

	default:
		if bodyType != "" {
			return bodyBytes, headerMap, nil
		}

		ok, err := isValidJSON(bodyBytes)
		if err == nil && ok {
			headerMap["Content-Type"] = BodyTypeJSON
			return bodyBytes, headerMap, nil
		}

		equalNun := bytes.Count(bodyBytes, []byte("="))
		andNun := bytes.Count(bodyBytes, []byte("&"))
		if equalNun == andNun+1 {
			headerMap["Content-Type"] = BodyTypeForm
		} else {
			headerMap["Content-Type"] = BodyTypeText
		}

		return bodyBytes, headerMap, nil
	}
}

func trimString(s string) string {
	return strings.Trim(s, " \t\r\n\"'")
}

// CheckBodyParam checks if the body parameter is provided in JSON or file format.
func CheckBodyParam(bodyJSON string, bodyFile string) (string, error) {
	var body []byte
	if bodyJSON != "" {
		bodyJSON = strings.Trim(bodyJSON, "'")
		body = []byte(bodyJSON)
	} else {
		if bodyFile != "" {
			data, err := os.ReadFile(bodyFile)
			if err != nil {
				return "", err
			}
			body = data
		}
	}

	if len(body) == 0 {
		return "", nil
	}

	ok, err := isValidJSON(body)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("invalid JSON format")
	}

	return string(body), nil
}

func isValidJSON(data []byte) (bool, error) {
	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		return false, err
	}
	return true, nil
}

// NewID generates a new ID for each request.
func NewID() int64 {
	ns := time.Now().UnixMilli() * 1000000
	return ns + rand.Int63n(1000000)
}

// NewStringID Generate a string ID, the hexadecimal form of NewID(), total 16 bytes.
func NewStringID() string {
	return strconv.FormatInt(NewID(), 16)
}

// CheckPortInUse checks if the given port is in use, if not, it returns a new available port.
func CheckPortInUse(port string) string {
	addr := fmt.Sprintf(":%s", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		portInt, err2 := utils.GetAvailablePort()
		if err2 != nil {
			return ""
		}
		return strconv.Itoa(portInt)
	}
	_ = ln.Close()
	return port
}
