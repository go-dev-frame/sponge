// Package parser is parsed proto file to struct
package parser

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// PbService service fields
type PbService struct {
	Name      string           // Greeter
	LowerName string           // greeter first character to lower
	Methods   []*ServiceMethod // service methods

	CutServiceName      string // GreeterService --> Greeter
	LowerCutServiceName string // GreeterService --> greeter

	ImportPkgMap      map[string]string // e.g. `userV1`:`userV1 "user/api/user/v1"`
	FieldImportPkgMap map[string]string // e.g. `userV1`:`userV1 "user/api/user/v1"`

	ProtoFileDir string // e.g. api/user/v1
	ProtoPkgName string // e.g. userV1
}

// ServiceMethod RPCMethod fields
type ServiceMethod struct {
	MethodName    string // Create
	Request       string // CreateRequest
	RequestFields []*Field
	Reply         string // CreateReply
	ReplyFields   []*Field
	Comment       string // e.g. Create a record
	InvokeType    int    // 0:unary, 1: client-side streaming, 2: server-side streaming, 3: bidirectional streaming

	ServiceName         string // Greeter
	LowerServiceName    string // greeter first character to lower
	LowerCutServiceName string // GreeterService --> greeter

	// http_rule
	Path   string // rule
	Method string // HTTP Method
	Body   string

	IsPassGinContext   bool
	IsIgnoreShouldBind bool

	RequestImportPkgName string // e.g. userV1
	ReplyImportPkgName   string // e.g. userV1
	ProtoPkgName         string // e.g. userV1
}

// Field message field
type Field struct {
	Name           string // field name
	GoType         string // field go type
	GoTypeCrossPkg string // field go type cross package
	Comment        string // field comment
	FieldType      string // field type
	IsList         bool   // is list
	IsMap          bool   // is map
	ImportPkgName  string // e.g. anypb
	ImportPkgPath  string // import path e.g. google.golang.org/protobuf/types/known/anypb
}

// GoTypeZero default zero value for type
func (r Field) GoTypeZero() string {
	switch r.FieldType {
	case "bool":
		return "false"
	case "int32", "uint32", "sint32", "int64", "uint64", "sint64", "sfixed32", "fixed32", "sfixed64", "fixed64": //nolint
		return "0"
	case "float", "double":
		return "0.0"
	case "string":
		return `""`
	default:
		return "nil"
	}
}

// AddOne counter
func (t *ServiceMethod) AddOne(i int) int {
	return i + 1
}

// RandNumber rand number 1~100
func (s *PbService) RandNumber() int {
	return rand.Intn(99) + 1
}

func parsePbService(s *protogen.Service, protoFileDir string) *PbService {
	protoPkgName := convertToPkgName(protoFileDir)
	cutServiceName := getCutServiceName(s.GoName)
	importPkgMap := make(map[string]string)
	fieldImportPkgMap := make(map[string]string)

	var methods []*ServiceMethod
	for _, m := range s.Methods {
		rpcMethod := &RPCMethod{} //nolint
		rule, ok := proto.GetExtension(m.Desc.Options(), annotations.E_Http).(*annotations.HttpRule)
		if rule != nil && ok {
			rpcMethod = buildHTTPRule(m, rule, protoPkgName)
		}

		requestImportPkgName := convertToPkgName(m.Input.GoIdent.GoImportPath.String())
		replyImportPkgName := convertToPkgName(m.Output.GoIdent.GoImportPath.String())
		if requestImportPkgName != "" {
			importPkgMap[requestImportPkgName] = requestImportPkgName + " " + m.Input.GoIdent.GoImportPath.String()
		}
		if replyImportPkgName != "" {
			importPkgMap[replyImportPkgName] = replyImportPkgName + " " + m.Output.GoIdent.GoImportPath.String()
		}

		methods = append(methods, &ServiceMethod{
			MethodName:    m.GoName,
			Request:       m.Input.GoIdent.GoName,
			RequestFields: getFields(m.Input, fieldImportPkgMap),
			Reply:         m.Output.GoIdent.GoName,
			ReplyFields:   getFields(m.Output, fieldImportPkgMap),
			Comment:       getMethodComment(m),
			InvokeType:    getInvokeType(m.Desc.IsStreamingClient(), m.Desc.IsStreamingServer()),

			ServiceName:         s.GoName,
			LowerServiceName:    strings.ToLower(s.GoName[:1]) + s.GoName[1:],
			LowerCutServiceName: strings.ToLower(cutServiceName[:1]) + cutServiceName[1:],

			Path:   rpcMethod.Path,
			Method: rpcMethod.Method,
			Body:   rpcMethod.Body,

			IsPassGinContext:   rpcMethod.IsPassGinContext,
			IsIgnoreShouldBind: rpcMethod.IsIgnoreShouldBind,

			RequestImportPkgName: requestImportPkgName,
			ReplyImportPkgName:   replyImportPkgName,
			ProtoPkgName:         protoPkgName,
		})
	}

	return &PbService{
		Name:                s.GoName,
		LowerName:           strings.ToLower(s.GoName[:1]) + s.GoName[1:],
		Methods:             methods,
		CutServiceName:      cutServiceName,
		LowerCutServiceName: strings.ToLower(cutServiceName[:1]) + cutServiceName[1:],
		ImportPkgMap:        importPkgMap,
		FieldImportPkgMap:   fieldImportPkgMap,
		ProtoFileDir:        protoFileDir,
		ProtoPkgName:        protoPkgName,
	}
}

// GetServices parse protobuf services
func GetServices(file *protogen.File) []*PbService {
	if len(file.Services) == 0 {
		return []*PbService{}
	}

	protoFileDir := GetProtoFileDir(file.GeneratedFilenamePrefix)
	var pss []*PbService
	for _, s := range file.Services {
		pss = append(pss, parsePbService(s, protoFileDir))
	}
	return pss
}

func getCutServiceName(name string) string {
	service := "Service"
	if len(name) < len(service) {
		return name
	}
	l := len(name) - len(service)
	if name[l:] == service {
		if name[:l] == "" {
			return name
		}
		return name[:l]
	}
	return name
}

type fieldPkgInfo struct {
	fieldType      string
	importPkgPath  string
	importPkgName  string
	goType         string
	goTypeCrossPkg string
}

func newFieldPkgInfo(ident protogen.GoIdent, importPkgMap map[string]string) *fieldPkgInfo {
	fieldType := ident.GoName
	importPkgPath := ident.GoImportPath.String()
	importPkgName := convertToPkgName(ident.GoImportPath.String())
	goType := "*" + fieldType
	goTypeCrossPkg := "*" + importPkgName + "." + fieldType
	if importPkgMap == nil {
		importPkgMap = make(map[string]string)
	}
	if importPkgName != "" {
		importPkgMap[importPkgName] = importPkgName + " " + importPkgPath
	}
	return &fieldPkgInfo{
		fieldType:      fieldType,
		importPkgPath:  importPkgPath,
		importPkgName:  importPkgName,
		goType:         goType,
		goTypeCrossPkg: goTypeCrossPkg,
	}
}

func getFields(m *protogen.Message, fieldImportPkgMap map[string]string) []*Field {
	var fields []*Field
	for _, f := range m.Fields {
		var (
			fieldType      string
			goType         string
			isList         bool
			isMap          bool
			goTypeCrossPkg string
			importPkgName  string
			importPkgPath  string
		)

		if f.Desc.IsList() {
			isList = true
		}
		if f.Desc.IsMap() {
			isMap = true
		}

		if f.Message != nil {
			fpi := newFieldPkgInfo(f.Message.GoIdent, fieldImportPkgMap)
			if isMap {
				// map value is message
				if f.Desc.MapValue().Kind() == protoreflect.MessageKind {
					for _, fSub := range f.Message.Fields {
						if fSub.Message != nil {
							fpi = newFieldPkgInfo(fSub.Message.GoIdent, fieldImportPkgMap)
							fieldType = fpi.fieldType
							importPkgPath = fpi.importPkgPath
							importPkgName = fpi.importPkgName
							goType = "map[" + toGoType(f.Desc.MapKey().Kind()) + "]" + fpi.goType
							goTypeCrossPkg = "map[" + toGoType(f.Desc.MapKey().Kind()) + "]" + fpi.goTypeCrossPkg
						}
					}
				} else {
					// map value is not message
					goType = "map[" + toGoType(f.Desc.MapKey().Kind()) + "]" + toGoType(f.Desc.MapValue().Kind())
					goTypeCrossPkg = goType
				}
			} else {
				// field is message
				fieldType = fpi.fieldType
				importPkgPath = fpi.importPkgPath
				importPkgName = fpi.importPkgName
				goType = fpi.goType
				goTypeCrossPkg = fpi.goTypeCrossPkg
				if isList {
					// field is list of message
					goType = "[]" + fpi.goType
					goTypeCrossPkg = "[]" + fpi.goTypeCrossPkg
				}
			}
		} else {
			// field is not message
			fieldType = f.Desc.Kind().String()
			goType = toGoType(f.Desc.Kind())
			goTypeCrossPkg = goType
			if isList {
				// field is list of not message
				goType = "[]" + goType
				goTypeCrossPkg = "[]" + goTypeCrossPkg
			}
		}

		fields = append(fields, &Field{
			Name:           f.GoName,
			GoType:         goType,
			GoTypeCrossPkg: goTypeCrossPkg,
			FieldType:      fieldType,
			Comment:        getFieldComment(f.Comments),
			IsList:         isList,
			IsMap:          isMap,
			ImportPkgName:  importPkgName,
			ImportPkgPath:  importPkgPath,
		})
	}

	return fields
}

// toGoType convert protobuf type to go type, not support message type
func toGoType(protoKind protoreflect.Kind) string {
	switch protoKind {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.EnumKind:
		return "int32"
	case protoreflect.Int32Kind:
		return "int32"
	case protoreflect.Sint32Kind:
		return "int32"
	case protoreflect.Uint32Kind:
		return "uint32"
	case protoreflect.Int64Kind:
		return "int64"
	case protoreflect.Sint64Kind:
		return "int64"
	case protoreflect.Uint64Kind:
		return "uint64"
	case protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.FloatKind:
		return "float32"
	case protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Fixed64Kind:
		return "uint64"
	case protoreflect.DoubleKind:
		return "float64"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "[]byte"
	default:
		return fmt.Sprintf("unsupported(%s)", protoKind.String())
	}
}

func getMethodComment(m *protogen.Method) string {
	symbol := "// "
	symbolLen := len(symbol)
	commentPrefix := symbol + m.GoName + " "
	comment := m.Comments.Leading.String()

	if len(comment) >= symbolLen {
		if comment[:symbolLen] == symbol {
			if comment[len(comment)-1] == '\n' {
				comment = comment[:len(comment)-1]
			}
			if len(comment) >= symbolLen {
				if len(comment[symbolLen:]) > len(m.GoName) {
					commentPrefixLower := strings.ToLower(comment[symbolLen : len(m.GoName)+symbolLen+1])
					if commentPrefixLower == strings.ToLower(m.GoName+" ") {
						return commentPrefix + comment[symbolLen+len(m.GoName)+1:]
					}
				}
				return commentPrefix + comment[symbolLen:]
			}
		}
	}

	return commentPrefix + "......"
}

func getFieldComment(commentSet protogen.CommentSet) string {
	comment1 := getFieldCommentStr(commentSet.Leading.String())
	comment2 := getFieldCommentStr(commentSet.Trailing.String())
	if comment1 == "" {
		return comment2
	}
	return comment1 + " " + comment2
}

func getFieldCommentStr(comment string) string {
	if len(comment) > 2 && comment[len(comment)-1] == '\n' {
		return comment[:len(comment)-1]
	}
	return comment
}

func getInvokeType(isStreamingClient bool, isStreamingServer bool) int {
	if isStreamingClient {
		if isStreamingServer {
			return 3 // bidirectional streaming
		}
		return 1 // client-side streaming
	}

	if isStreamingServer {
		return 2 // server-side streaming
	}

	return 0 // unary
}

func GetProtoFileDir(protoPath string) string {
	ss := strings.Split(protoPath, "/")
	if len(ss) > 1 {
		return strings.Join(ss[:len(ss)-1], "/")
	}
	return protoPath
}

func GetProtoPkgName(importPath string) string {
	return convertToPkgName(importPath)
}

func convertToPkgName(importPath string) string {
	importPath = strings.ReplaceAll(importPath, `"`, "")
	ss := strings.Split(importPath, "/")
	l := len(ss)
	if l > 1 {
		pkgName := strings.ToLower(ss[l-1])
		if isVersionNum(pkgName) || pkgName == "pb" || len(pkgName) < 2 {
			return removeMiddleLine(ss[l-2]) + strings.ToUpper(pkgName[:1]) + pkgName[1:]
		}
		return removeMiddleLine(ss[l-1])
	}
	return ""
}

func isVersionNum(pkgName string) bool {
	pattern := `^v\d+$`
	matched, err := regexp.MatchString(pattern, pkgName)
	if err != nil {
		return false
	}
	return matched
}

func removeMiddleLine(str string) string {
	return strings.ReplaceAll(str, "-", "")
}
