package generate

import (
	"errors"
	"fmt"

	"github.com/fatih/color"
	"github.com/huandu/xstrings"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/pkg/replacer"
)

// RPCGwPbCommand generate grpc gateway server code base on protobuf file
func RPCGwPbCommand() *cobra.Command {
	var (
		moduleName   string // module name for go.mod
		serverName   string // server name
		projectName  string // project name for deployment name
		repoAddr     string // image repo address
		outPath      string // output directory
		protobufFile string // protobuf file, support * matching

		suitedMonoRepo bool // whether the generated code is suitable for mono-repo
	)

	cmd := &cobra.Command{
		Use:   "rpc-gw-pb",
		Short: "Generate grpc gateway server code based on protobuf file",
		Long:  "Generate grpc gateway server code based on protobuf file.",
		Example: color.HiBlackString(`  # Generate grpc gateway server code.
  sponge micro rpc-gw-pb --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --protobuf-file=./demo.proto

  # Generate grpc gateway server code and specify the output directory, Note: code generation will be canceled when the latest generated file already exists.
  sponge micro rpc-gw-pb --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --protobuf-file=./demo.proto --out=./yourServerDir

  # Generate grpc gateway server code and specify the docker image repository address.
  sponge micro rpc-gw-pb --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --repo-addr=192.168.3.37:9443/user-name --protobuf-file=./demo.proto

  # If you want the generated code to suited to mono-repo, you need to set the parameter --suited-mono-repo=true`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			projectName, serverName, err = convertProjectAndServerName(projectName, serverName)
			if err != nil {
				return err
			}

			if suitedMonoRepo {
				outPath = changeOutPath(outPath, serverName)
			}

			g := &rpcGwPbGenerator{
				moduleName:   moduleName,
				serverName:   serverName,
				projectName:  projectName,
				protobufFile: protobufFile,
				repoAddr:     repoAddr,
				outPath:      outPath,

				suitedMonoRepo: suitedMonoRepo,
			}
			err = g.generateCode()
			if err != nil {
				return err
			}

			_ = generateConfigmap(serverName, outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module-name", "m", "", "module-name is the name of the module in the go.mod file")
	_ = cmd.MarkFlagRequired("module-name")
	cmd.Flags().StringVarP(&serverName, "server-name", "s", "", "server name")
	_ = cmd.MarkFlagRequired("server-name")
	cmd.Flags().StringVarP(&projectName, "project-name", "p", "", "project name")
	_ = cmd.MarkFlagRequired("project-name")
	cmd.Flags().StringVarP(&protobufFile, "protobuf-file", "f", "", "proto file")
	_ = cmd.MarkFlagRequired("protobuf-file")
	cmd.Flags().BoolVarP(&suitedMonoRepo, "suited-mono-repo", "l", false, "whether the generated code is suitable for mono-repo")
	cmd.Flags().StringVarP(&repoAddr, "repo-addr", "r", "", "docker image repository address, excluding http and repository names")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output directory, default is ./serverName_rpc-gw-pb_<time>")

	return cmd
}

type rpcGwPbGenerator struct {
	moduleName   string
	serverName   string
	projectName  string
	protobufFile string
	repoAddr     string
	outPath      string

	suitedMonoRepo bool
}

func (g *rpcGwPbGenerator) generateCode() error {
	protobufFiles, isImportTypes, err := parseProtobufFiles(g.protobufFile)
	if err != nil {
		return err
	}

	subTplName := codeNameGRPCGW
	r := Replacers[TplNameSponge]
	if r == nil {
		return errors.New("replacer is nil")
	}

	// specify the subdirectory and files
	subDirs := []string{
		"cmd/serverNameExample_grpcGwPbExample",
		"sponge/configs",
		"sponge/deployments", "sponge/scripts", "sponge/third_party",
	}
	subFiles := []string{
		"sponge/.gitignore", "sponge/.golangci.yml", "sponge/go.mod", "sponge/go.sum",
		"sponge/Jenkinsfile", "sponge/Makefile", "sponge/README.md",
	}

	if isImportTypes {
		subFiles = append(subFiles, "api/types/types.proto")
	}

	selectFiles := map[string][]string{
		"docs": {
			"apis.go", "apis.swagger.json",
		},
		"internal/config": {
			"serverNameExample.go",
		},
		"internal/ecode": {
			"systemCode_rpc.go",
		},
		"internal/routers": {
			"routers_pbExample.go",
		},
		"internal/server": {
			"http.go", "http_option.go",
		},
	}

	if g.suitedMonoRepo {
		subDirs = removeElements(subDirs, "sponge/third_party")
		subFiles = removeElements(subFiles, "sponge/go.mod", "sponge/go.sum", "api/types/types.proto")
	}

	replaceFiles := make(map[string][]string)
	subFiles = append(subFiles, getSubFiles(selectFiles, replaceFiles)...)

	// ignore some directories and files
	ignoreDirs := []string{"cmd/sponge"}
	ignoreFiles := []string{"configs/serverNameExample_cc.yml"}

	r.SetSubDirsAndFiles(subDirs, subFiles...)
	r.SetIgnoreSubDirs(ignoreDirs...)
	r.SetIgnoreSubFiles(ignoreFiles...)
	_ = r.SetOutputDir(g.outPath, g.serverName+"_"+subTplName)
	fields := g.addFields(r)
	r.SetReplacementFields(fields)
	if err = r.SaveFiles(); err != nil {
		return err
	}

	if err = saveProtobufFiles(g.moduleName, g.serverName, g.suitedMonoRepo, r.GetOutputDir(), protobufFiles); err != nil {
		return err
	}
	_ = saveGenInfo(g.moduleName, g.serverName, g.suitedMonoRepo, r.GetOutputDir())
	_ = saveEmptySwaggerJSON(r.GetOutputDir())

	fmt.Printf(`
using help:
  1. open a terminal and execute the command to generate code: make proto
  2. open internal/service/xxx.go file, replace panic("implement me") according to template code example.
  3. compile and run server: make run
  4. access http://localhost:8080/apis/swagger/index.html in your browser, and test the http api.

`)
	fmt.Printf("generate %s's grpc gateway server code successfully, out = %s\n", g.serverName, r.GetOutputDir())

	return nil
}

func (g *rpcGwPbGenerator) addFields(r replacer.Replacer) []replacer.Field {
	var fields []replacer.Field

	repoHost, _ := parseImageRepoAddr(g.repoAddr)

	fields = append(fields, deleteFieldsMark(r, httpFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, dockerFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, dockerFileBuild, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, dockerComposeFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, k8sDeploymentFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, k8sServiceFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, imageBuildFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, imageBuildLocalFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteAllFieldsMark(r, makeFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, gitIgnoreFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteAllFieldsMark(r, protoShellFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteAllFieldsMark(r, appConfigFile, wellStartMark, wellEndMark)...)
	//fields = append(fields, deleteFieldsMark(r, deploymentConfigFile, wellStartMark, wellEndMark)...)
	fields = append(fields, replaceFileContentMark(r, readmeFile,
		getReadmeContent(g.moduleName, g.serverName, codeNameGRPCGW, "", g.suitedMonoRepo))...)
	fields = append(fields, []replacer.Field{
		{ // replace the configuration of the *.yml file
			Old: appConfigFileMark,
			New: rpcGwServerConfigCode,
		},
		{ // replace the configuration of the *.yml file
			Old: appConfigFileMark2,
			New: getDBConfigCode(""), // no db config
		},
		//{ // replace the contents of the model/init.go file
		//	Old: modelInitDBFileMark,
		//	New: getInitDBCode(DBDriverMysql), // default is mysql
		//},
		{ // replace the contents of the Dockerfile file
			Old: dockerFileMark,
			New: dockerFileHTTPCode,
		},
		{ // replace the contents of the Dockerfile_build file
			Old: dockerFileBuildMark,
			New: dockerFileBuildHTTPCode,
		},
		{ // replace the contents of the image-build.sh file
			Old: imageBuildFileMark,
			New: imageBuildFileHTTPCode,
		},
		{ // replace the contents of the image-build-local.sh file
			Old: imageBuildLocalFileMark,
			New: imageBuildLocalFileHTTPCode,
		},
		{ // replace the contents of the docker-compose.yml file
			Old: dockerComposeFileMark,
			New: dockerComposeFileHTTPCode,
		},
		//{ // replace the contents of the *-configmap.yml file
		//	Old: deploymentConfigFileMark,
		//	New: getDBConfigCode(DBDriverMysql, true),
		//},
		{ // replace the contents of the *-deployment.yml file
			Old: k8sDeploymentFileMark,
			New: k8sDeploymentFileHTTPCode,
		},
		{ // replace the contents of the *-svc.yml file
			Old: k8sServiceFileMark,
			New: k8sServiceFileHTTPCode,
		},
		{ // replace the contents of the proto.sh file
			Old: protoShellFileGRPCMark,
			New: protoShellGRPCMark,
		},
		{ // replace the contents of the proto.sh file
			Old: protoShellFileMark,
			New: protoShellServiceCode,
		},
		{
			Old: "github.com/go-dev-frame/sponge",
			New: g.moduleName,
		},
		{
			Old: g.moduleName + pkgPathSuffix,
			New: "github.com/go-dev-frame/sponge/pkg",
		},
		{ // replace the sponge version of the go.mod file
			Old: spongeTemplateVersionMark,
			New: getLocalSpongeTemplateVersion(),
		},
		{
			Old: "sponge api docs",
			New: g.serverName + " api docs",
		},
		{
			Old: defaultGoModVersion,
			New: getLocalGoVersion(),
		},
		{
			Old: defaultImageGoModVersion,
			New: extractImageGoVersion(),
		},
		{
			Old:             "serverNameExample",
			New:             g.serverName,
			IsCaseSensitive: true,
		},
		// docker image and k8s deployment script replacement
		{
			Old: "server-name-example",
			New: xstrings.ToKebabCase(g.serverName), // snake_case to kebab_case
		},
		// docker image and k8s deployment script replacement
		{
			Old: "project-name-example",
			New: g.projectName,
		},
		{
			Old: "projectNameExample",
			New: g.projectName,
		},
		{
			Old: "repo-addr-example",
			New: g.repoAddr,
		},
		{
			Old: "image-repo-host",
			New: repoHost,
		},
		{
			Old: "_grpcGwPbExample",
			New: "",
		},
		{
			Old: "_mixExample",
			New: "",
		},
		{
			Old: "_pbExample",
			New: "",
		},
	}...)

	fields = append(fields, getGRPCServiceFields()...)

	if g.suitedMonoRepo {
		fs := serverCodeFields(codeNameGRPCGW, g.moduleName, g.serverName)
		fields = append(fields, fs...)
	}

	return fields
}
