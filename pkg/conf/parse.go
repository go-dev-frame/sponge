// Package conf is parsing yaml, json, toml configuration files to go struct.
package conf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Parse configuration files to struct, including yaml, toml, json, etc., and turn on listening for configuration file changes if fs is not empty
func Parse(configFile string, obj interface{}, reloads ...func()) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("obj must be a non-nil pointer")
	}

	confFileAbs, err := filepath.Abs(configFile)
	if err != nil {
		return err
	}

	filePathStr, filename := filepath.Split(confFileAbs)
	ext := strings.TrimLeft(path.Ext(filename), ".")
	if ext != "toml" {
		filename = strings.ReplaceAll(filename, "."+ext, "") // excluding suffix names
	}

	viper.AddConfigPath(filePathStr) // path
	viper.SetConfigName(filename)    // file name
	viper.SetConfigType(ext)         // get the configuration type from the file name
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	err = viper.Unmarshal(obj)
	if err != nil {
		return err
	}

	if len(reloads) > 0 {
		watchConfig(obj, reloads...)
	}

	return nil
}

// ParseConfigData parse data to struct, parameter format is the configuration file format, such as "yaml", "json", "toml"
func ParseConfigData(data []byte, format string, obj interface{}) error {
	viper.SetConfigType(format)
	err := viper.ReadConfig(bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	return viper.Unmarshal(obj)
}

// listening for profile updates
func watchConfig(obj interface{}, reloads ...func()) {
	viper.WatchConfig()

	// Note: OnConfigChange is called twice on Windows
	viper.OnConfigChange(func(e fsnotify.Event) {
		t := reflect.TypeOf(obj).Elem()
		v := reflect.New(t)
		reflect.ValueOf(obj).Elem().Set(v.Elem()) // reset object

		err := viper.Unmarshal(obj)
		if err != nil {
			fmt.Println("viper.Unmarshal error: ", err)
		} else {
			for _, reload := range reloads {
				reload()
			}
		}
	})
}

// Show print configuration information (hide sensitive fields)
func Show(obj interface{}, fields ...string) string {
	var out string

	data, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		fmt.Println("json.MarshalIndent error: ", err)
		return ""
	}

	buf := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			break
		}
		fields = append(fields, `"dsn"`, `"password"`, `"pwd"`)

		out += hideSensitiveFields(line, fields...)
	}

	return out
}

func hideSensitiveFields(line string, fields ...string) string {
	for _, field := range fields {
		if strings.Contains(line, field) {
			index := strings.Index(line, field)
			if strings.Contains(line, "@") && strings.Contains(line, ":") {
				return replaceDSN(line)
			}
			return fmt.Sprintf("%s: \"******\",\n", line[:index+len(field)])
		}
	}

	// replace dsn
	if strings.Contains(line, "@") && strings.Contains(line, ":") {
		return replaceDSN(line)
	}

	return line
}

// replace dsn password
func replaceDSN(str string) string {
	data := []byte(str)
	start, end := 0, 0
	for k, v := range data {
		if v == ':' {
			start = k
		}
		if v == '@' {
			end = k
			break
		}
	}

	if start >= end {
		return str
	}

	return fmt.Sprintf("%s******%s", data[:start+1], data[end:])
}
