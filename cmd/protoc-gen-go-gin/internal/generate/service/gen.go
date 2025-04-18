// Package service is to generate template code, router code, and error code.
package service

import (
	"bytes"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/go-dev-frame/sponge/cmd/protoc-gen-go-gin/internal/parse"
)

// GenerateFiles generate service logic, router, error code files.
func GenerateFiles(file *protogen.File, moduleName string) (logicContent []byte,
	routerFileContent []byte, errCodeFileContent []byte) {
	if len(file.Services) == 0 {
		return nil, nil, nil
	}

	pss := parse.GetServices(file, moduleName)
	logicContent = genServiceLogicFile(pss)
	routerFileContent = genRouterFile(pss)
	errCodeFileContent = genErrCodeFile(pss)

	return logicContent, routerFileContent, errCodeFileContent
}

func genServiceLogicFile(fields []*parse.PbService) []byte {
	slf := &serviceLogicFields{PbServices: fields}
	return slf.execute()
}

func genRouterFile(fields []*parse.PbService) []byte {
	rf := &routerFields{PbServices: fields}
	return rf.execute()
}

func genErrCodeFile(fields []*parse.PbService) []byte {
	cf := &errCodeFields{PbServices: fields}
	return cf.execute()
}

type serviceLogicFields struct {
	PbServices []*parse.PbService
}

func (f *serviceLogicFields) execute() []byte {
	buf := new(bytes.Buffer)
	if err := serviceLogicTmpl.Execute(buf, f); err != nil {
		panic(err)
	}
	content := buf.Bytes()
	return bytes.ReplaceAll(content, []byte(importPkgPathMark), parse.GetImportPkg(f.PbServices))
}

type routerFields struct {
	PbServices []*parse.PbService
}

func (f *routerFields) execute() []byte {
	buf := new(bytes.Buffer)
	if err := routerTmpl.Execute(buf, f); err != nil {
		panic(err)
	}
	content := buf.Bytes()
	return bytes.ReplaceAll(content, []byte(importPkgPathMark), parse.GetSourceImportPkg(f.PbServices))
}

type errCodeFields struct {
	PbServices []*parse.PbService
}

func (f *errCodeFields) execute() []byte {
	buf := new(bytes.Buffer)
	if err := rpcErrCodeTmpl.Execute(buf, f); err != nil {
		panic(err)
	}
	data := bytes.ReplaceAll(buf.Bytes(), []byte("// --blank line--"), []byte{})
	return data
}

const importPkgPathMark = "// import api service package here"
