// Package main generate *.go(tmpl), *_router.go, *_http.go, *_router.pb.go code based on proto files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/go-dev-frame/sponge/cmd/protoc-gen-go-gin/internal/generate/handler"
	"github.com/go-dev-frame/sponge/cmd/protoc-gen-go-gin/internal/generate/router"
	"github.com/go-dev-frame/sponge/cmd/protoc-gen-go-gin/internal/generate/service"
	"github.com/go-dev-frame/sponge/pkg/gofile"
)

const (
	handlerPlugin = "handler"
	servicePlugin = "service"
	mixPlugin     = "mix" // code generated for the http+grpc approach

	helpInfo = `
# generate *_router.pb.go file
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative *.proto

# generate *_router.pb.go, *.go(tmpl), *_router.go, *_http.go files
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative --go-gin_opt=plugin=handler \
  --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName *.proto

# generate *_router.pb.go, *.go(tmpl), *_router.go, *_rpc.go files
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative --go-gin_opt=plugin=service \
  --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName *.proto

# generate *_router.pb.go, *.go(tmpl), *_router.go, *_rpc.go files
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative --go-gin_opt=plugin=mix \
  --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName *.proto

# if you want the generated code to suited to mono-repo, you need to set the parameter --go-gin_opt=suitedMonoRepo=true

Tip:
    If you want to merge the code, after generating the code, execute the command "sponge merge http-pb" or
    "sponge merge rpc-gw-pb", you don't worry about it affecting the logic code you have already written,
    in case of accidents, you can find the pre-merge code in the directory /tmp/sponge_merge_backup_code.
`

	optErrFormat = `--go-gin_opt error, '%s' cannot be empty.

Usage example: 
    protoc --proto_path=. --proto_path=./third_party \
      --go-gin_out=. --go-gin_opt=paths=source_relative \
      --go-gin_opt=plugin=%s --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName \
      *.proto
`
)

// nolint
func main() {
	var h bool
	flag.BoolVar(&h, "h", false, "help information")
	flag.Parse()
	if h {
		fmt.Printf("%s", helpInfo)
		return
	}

	var flags flag.FlagSet

	var plugin, moduleName, serverName, logicOut, routerOut, ecodeOut string
	var suitedMonoRepo bool
	flags.StringVar(&plugin, "plugin", "", "plugin name, supported values: handler, service and mix")
	flags.StringVar(&moduleName, "moduleName", "", "module name for plugin")
	flags.StringVar(&serverName, "serverName", "", "server name for plugin")
	flags.StringVar(&logicOut, "logicOut", "", "directory of logical template code generated by the plugin, "+
		"the default value is internal/handler if the plugin is a handler, or internal/service if it is a service")
	flags.StringVar(&routerOut, "routerOut", "", "directory of routing code generated by the plugin, default is internal/routers")
	flags.StringVar(&ecodeOut, "ecodeOut", "", "directory of error code generated by the plugin, default is internal/ecode")
	flags.BoolVar(&suitedMonoRepo, "suitedMonoRepo", false, "whether the generated code is suitable for mono-repo")

	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(gen *protogen.Plugin) error {
		handlerFlag, serviceFlag, mixFlag := false, false, false
		pluginName := strings.ReplaceAll(plugin, " ", "")
		dirName := "internal"
		if suitedMonoRepo {
			dirName = serverName + "/internal"
		}
		switch pluginName {
		case handlerPlugin:
			handlerFlag = true
			if logicOut == "" {
				logicOut = dirName + "/handler"
			}
			if routerOut == "" {
				routerOut = dirName + "/routers"
			}
			if ecodeOut == "" {
				ecodeOut = dirName + "/ecode"
			}
		case servicePlugin:
			serviceFlag = true
			if logicOut == "" {
				logicOut = dirName + "/service"
			}
			if routerOut == "" {
				routerOut = dirName + "/routers"
			}
			if ecodeOut == "" {
				ecodeOut = dirName + "/ecode"
			}
		case mixPlugin:
			mixFlag = true
			handlerFlag = true
			if logicOut == "" {
				logicOut = dirName + "/handler"
			}
			if routerOut == "" {
				routerOut = dirName + "/routers"
			}
		default:
			return fmt.Errorf("protoc-gen-go-gin: unknown plugin name '%q', only 'service', 'handler' and 'mix' are supported", plugin)
		}

		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			_, checkFilename := filepath.Split(f.GeneratedFilenamePrefix + ".proto")
			if strings.HasSuffix(checkFilename, "_test.proto") {
				return fmt.Errorf(`the proto file name (%s) suffix "_test" is not supported for code generation, please delete suffix "_test" or change it to another name. `, checkFilename)
			}

			if err := saveGinRouterFiles(f); err != nil {
				return err
			}

			if handlerFlag {
				err := saveHandlerAndRouterFiles(f, moduleName, serverName, logicOut, routerOut, ecodeOut, suitedMonoRepo, mixFlag)
				if err != nil {
					return err
				}
			} else if serviceFlag {
				err := saveServiceAndRouterFiles(f, moduleName, serverName, logicOut, routerOut, ecodeOut, suitedMonoRepo)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func saveGinRouterFiles(f *protogen.File) error {
	ginRouterFileContent := router.GenerateFiles(f)
	if len(ginRouterFileContent) == 0 {
		return nil
	}
	if !bytes.Contains(ginRouterFileContent, []byte("errors.")) {
		ginRouterFileContent = bytes.Replace(ginRouterFileContent, []byte(`"errors"`), []byte(""), 1)
	}
	if !bytes.Contains(ginRouterFileContent, []byte("middleware.")) {
		ginRouterFileContent = bytes.Replace(ginRouterFileContent, []byte(`"github.com/go-dev-frame/sponge/pkg/gin/middleware"`), []byte(""), 1)
	}
	filePath := f.GeneratedFilenamePrefix + "_router.pb.go"
	return os.WriteFile(filePath, ginRouterFileContent, 0666)
}

func saveHandlerAndRouterFiles(f *protogen.File, moduleName string, serverName string,
	logicOut string, routerOut string, ecodeOut string, suitedMonoRepo bool, isMixType bool) error {
	filenamePrefix := f.GeneratedFilenamePrefix
	handlerLogicContent, routerContent, errCodeFileContent := handler.GenerateFiles(f, isMixType, moduleName)

	filePath := filenamePrefix + ".go"
	err := saveFile(moduleName, serverName, logicOut, filePath, handlerLogicContent, false, handlerPlugin, suitedMonoRepo)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_router.go"
	err = saveFile(moduleName, serverName, routerOut, filePath, routerContent, false, handlerPlugin, suitedMonoRepo)
	if err != nil {
		return err
	}

	if !isMixType {
		filePath = filenamePrefix + "_http.go"
		err = saveFileSimple(ecodeOut, filePath, errCodeFileContent, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func saveServiceAndRouterFiles(f *protogen.File, moduleName string, serverName string,
	logicOut string, routerOut string, ecodeOut string, suitedMonoRepo bool) error {
	filenamePrefix := f.GeneratedFilenamePrefix
	serviceLogicContent, routerContent, errCodeFileContent := service.GenerateFiles(f, moduleName)

	filePath := filenamePrefix + ".go"
	err := saveFile(moduleName, serverName, logicOut, filePath, serviceLogicContent, false, servicePlugin, suitedMonoRepo)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_router.go"
	err = saveFile(moduleName, serverName, routerOut, filePath, routerContent, false, servicePlugin, suitedMonoRepo)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_rpc.go"
	err = saveFileSimple(ecodeOut, filePath, errCodeFileContent, false)
	if err != nil {
		return err
	}

	return nil
}

func saveFile(moduleName string, serverName string, out string, filePath string, content []byte, isNeedCovered bool, pluginName string, suitedMonoRepo bool) error {
	if len(content) == 0 {
		return nil
	}

	if moduleName == "" {
		panic(fmt.Sprintf(optErrFormat, "moduleName", pluginName))
	}
	if serverName == "" {
		panic(fmt.Sprintf(optErrFormat, "serverName", pluginName))
	}

	_ = os.MkdirAll(out, 0766)
	_, name := filepath.Split(filePath)
	file := out + "/" + name
	if !isNeedCovered && isExists(file) {
		removeOldGenFile(file)
		file += ".gen" + time.Now().Format("20060102T150405")
	}

	content = bytes.ReplaceAll(content, []byte("moduleNameExample"), []byte(moduleName))
	content = bytes.ReplaceAll(content, []byte("serverNameExample"), []byte(serverName))
	content = bytes.ReplaceAll(content, firstLetterToUpper("serverNameExample"), firstLetterToUpper(serverName))
	if suitedMonoRepo {
		content = adaptMonoRepo(moduleName, serverName, content)
	}

	return os.WriteFile(file, content, 0666)
}

func saveFileSimple(out string, filePath string, content []byte, isNeedCovered bool) error {
	if len(content) == 0 {
		return nil
	}

	_ = os.MkdirAll(out, 0766)
	_, name := filepath.Split(filePath)
	file := out + "/" + name
	if !isNeedCovered && isExists(file) {
		removeOldGenFile(file)
		file += ".gen" + time.Now().Format("20060102T150405")
	}

	return os.WriteFile(file, content, 0666)
}

func isExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func removeOldGenFile(file string) {
	oldGenFiles := gofile.FuzzyMatchFiles(file + ".gen*")
	for _, oldGenFile := range oldGenFiles {
		_ = os.Remove(oldGenFile)
	}
}

func firstLetterToUpper(s string) []byte {
	if s == "" {
		return []byte{}
	}

	return []byte(strings.ToUpper(s[:1]) + s[1:])
}

func adaptMonoRepo(moduleName string, serverName string, data []byte) []byte {
	matchStr := map[string]string{
		fmt.Sprintf("\"%s/internal/", moduleName): fmt.Sprintf("\"%s/internal/", moduleName+"/"+serverName),
		fmt.Sprintf("\"%s/configs", moduleName):   fmt.Sprintf("\"%s/configs", moduleName+"/"+serverName),
		fmt.Sprintf("\"%s/api", moduleName):       fmt.Sprintf("\"%s/api", moduleName+"/"+serverName),
	}
	for oldStr, newStr := range matchStr {
		data = bytes.ReplaceAll(data, []byte(oldStr), []byte(newStr))
	}
	return data
}
