package generate

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/fatih/color"
	"github.com/huandu/xstrings"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/pkg/gofile"
	"github.com/go-dev-frame/sponge/pkg/replacer"
	"github.com/go-dev-frame/sponge/pkg/sql2code"
	"github.com/go-dev-frame/sponge/pkg/sql2code/parser"
)

// RPCCommand generate grpc server code
func RPCCommand() *cobra.Command {
	var (
		moduleName  string // module name for go.mod
		serverName  string // server name
		projectName string // project name for deployment name
		repoAddr    string // image repo address
		outPath     string // output directory
		dbTables    string // table names
		sqlArgs     = sql2code.Args{
			Package:  "model",
			JSONTag:  true,
			GormType: true,
		}

		suitedMonoRepo bool // whether the generated code is suitable for mono-repo
	)

	//nolint
	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "Generate grpc server code based on sql",
		Long:  "Generate grpc server code based on sql.",
		Example: color.HiBlackString(`  # Generate grpc server code.
  sponge micro rpc --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # Generate grpc server code with multiple table names.
  sponge micro rpc --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=t1,t2

  # Generate grpc server code with extended api.
  sponge micro rpc --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --extended-api=true

  # Generate grpc server code and specify the output directory, Note: code generation will be canceled when the latest generated file already exists.
  sponge micro rpc --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --out=./yourServerDir

  # Generate grpc server code and specify the docker image repository address.
  sponge micro rpc --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --repo-addr=192.168.3.37:9443/user-name --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # If you want the generated code to suited to mono-repo, you need to set the parameter --suited-mono-repo=true`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var firstTable string
			var servicesTableNames []string
			tableNames := strings.Split(dbTables, ",")
			if len(tableNames) == 1 {
				firstTable = tableNames[0]
			} else if len(tableNames) > 1 {
				firstTable = tableNames[0]
				servicesTableNames = tableNames[1:]
			}

			projectName, serverName, err = convertProjectAndServerName(projectName, serverName)
			if err != nil {
				return err
			}

			if suitedMonoRepo {
				outPath = changeOutPath(outPath, serverName)
			}
			if sqlArgs.DBDriver == DBDriverMongodb {
				sqlArgs.IsEmbed = false
			}

			sqlArgs.DBTable = firstTable
			codes, err := sql2code.Generate(&sqlArgs)
			if err != nil {
				return err
			}
			g := &rpcGenerator{
				moduleName:    moduleName,
				serverName:    serverName,
				projectName:   projectName,
				repoAddr:      repoAddr,
				dbDSN:         sqlArgs.DBDsn,
				dbDriver:      sqlArgs.DBDriver,
				isExtendedAPI: sqlArgs.IsExtendedAPI,
				isEmbed:       sqlArgs.IsEmbed,
				codes:         codes,
				outPath:       outPath,

				suitedMonoRepo: suitedMonoRepo,
			}
			outPath, err = g.generateCode()
			if err != nil {
				return err
			}

			for _, serviceTableName := range servicesTableNames {
				if serviceTableName == "" {
					continue
				}

				sqlArgs.DBTable = serviceTableName
				codes, err := sql2code.Generate(&sqlArgs)
				if err != nil {
					return err
				}

				sg := &serviceGenerator{
					moduleName:     moduleName,
					serverName:     serverName,
					dbDriver:       sqlArgs.DBDriver,
					isExtendedAPI:  sqlArgs.IsExtendedAPI,
					isEmbed:        sqlArgs.IsEmbed,
					codes:          codes,
					outPath:        outPath,
					suitedMonoRepo: suitedMonoRepo,
				}
				outPath, err = sg.generateCode()
				if err != nil {
					return err
				}
			}

			fmt.Printf(`
using help:
  1. open a terminal and execute the command to generate code:  make proto
  2. compile and run server: make run
  3. open the file internal/service/xxx_client_test.go using Goland or VSCode, and test the grpc CRUD api.

`)
			fmt.Printf("generate %s's grpc server code successfully, out = %s\n", serverName, outPath)

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
	cmd.Flags().StringVarP(&sqlArgs.DBDriver, "db-driver", "k", "mysql", "database driver, support mysql, mongodb, postgresql, sqlite")
	cmd.Flags().StringVarP(&sqlArgs.DBDsn, "db-dsn", "d", "", "database content address, e.g. user:password@(host:port)/database. Note: if db-driver=sqlite, db-dsn must be a local sqlite db file, e.g. --db-dsn=/tmp/sponge_sqlite.db") //nolint
	_ = cmd.MarkFlagRequired("db-dsn")
	cmd.Flags().StringVarP(&dbTables, "db-table", "t", "", "table name, multiple names separated by commas")
	_ = cmd.MarkFlagRequired("db-table")
	cmd.Flags().BoolVarP(&sqlArgs.IsEmbed, "embed", "e", false, "whether to embed gorm.model struct")
	cmd.Flags().BoolVarP(&sqlArgs.IsExtendedAPI, "extended-api", "a", false, "whether to generate extended crud api, additional includes: DeleteByIDs, GetByCondition, ListByIDs, ListByLatestID")
	cmd.Flags().BoolVarP(&suitedMonoRepo, "suited-mono-repo", "l", false, "whether the generated code is suitable for mono-repo")
	cmd.Flags().IntVarP(&sqlArgs.JSONNamedType, "json-name-type", "j", 1, "json tags name type, 0:snake case, 1:camel case")
	cmd.Flags().StringVarP(&repoAddr, "repo-addr", "r", "", "docker image repository address, excluding http and repository names")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output directory, default is ./serverName_rpc_<time>")

	return cmd
}

type rpcGenerator struct {
	moduleName     string
	serverName     string
	projectName    string
	repoAddr       string
	dbDSN          string
	dbDriver       string
	isEmbed        bool
	isExtendedAPI  bool
	codes          map[string]string
	outPath        string
	suitedMonoRepo bool

	fields        []replacer.Field
	isCommonStyle bool
}

func (g *rpcGenerator) generateCode() (string, error) {
	subTplName := codeNameGRPC
	r, _ := replacer.New(SpongeDir)
	if r == nil {
		return "", errors.New("replacer is nil")
	}

	// specify the subdirectory and files
	subDirs := []string{
		"cmd/serverNameExample_grpcExample", "sponge/configs",
		"sponge/deployments", "sponge/scripts", "sponge/third_party",
	}
	subFiles := []string{
		"sponge/.gitignore", "sponge/.golangci.yml", "sponge/go.mod", "sponge/go.sum",
		"sponge/Jenkinsfile", "sponge/Makefile", "sponge/README.md",
	}

	selectFiles := map[string][]string{
		"api/serverNameExample/v1": {
			"userExample.proto",
		},
		"api/types": {
			"types.proto",
		},
		"internal/cache": {
			"userExample.go", "userExample_test.go",
		},
		"internal/config": {
			"serverNameExample.go",
		},
		"internal/dao": {
			"userExample.go", "userExample_test.go",
		},
		"internal/database": {
			"init.go",
		},
		"internal/ecode": {
			"systemCode_rpc.go", "userExample_rpc.go",
		},
		"internal/model": {
			"userExample.go",
		},
		"internal/server": {
			"grpc.go", "grpc_option.go",
		},
		"internal/service": {
			"service.go", "service_test.go", "userExample.go", /*"userExample_client_test.go",*/
		},
	}
	err := SetSelectFiles(g.dbDriver, selectFiles)
	if err != nil {
		return "", err
	}

	info := g.codes[parser.CodeTypeCrudInfo]
	crudInfo, _ := unmarshalCrudInfo(info)
	if crudInfo.CheckCommonType() {
		g.isCommonStyle = true
		selectFiles["internal/cache"] = []string{"userExample.go.tpl"}
		selectFiles["internal/dao"] = []string{"userExample.go.tpl"}
		selectFiles["internal/ecode"] = []string{"systemCode_rpc.go", "userExample_rpc.go.tpl"}
		selectFiles["internal/service"] = []string{"service.go", "service_test.go", "userExample.go.tpl"}
		var fields []replacer.Field
		if g.isExtendedAPI {
			selectFiles["internal/dao"] = []string{"userExample.go.exp.tpl"}
			selectFiles["internal/ecode"] = []string{"systemCode_rpc.go", "userExample_rpc.go.exp.tpl"}
			selectFiles["internal/service"] = []string{"service.go", "service_test.go", "userExample.go.exp.tpl"}
			fields = commonGRPCExtendedFields(r)
		} else {
			fields = commonGRPCFields(r)
		}
		contentFields, err := replaceFilesContent(r, getTemplateFiles(selectFiles), crudInfo)
		if err != nil {
			return "", err
		}
		g.fields = append(g.fields, contentFields...)
		g.fields = append(g.fields, fields...)
	}

	if g.suitedMonoRepo {
		subDirs = removeElements(subDirs, "sponge/third_party")
		subFiles = removeElements(subFiles, "sponge/go.mod", "sponge/go.sum")
		delete(selectFiles, "api/types")
	}

	replaceFiles := make(map[string][]string)
	switch strings.ToLower(g.dbDriver) {
	case DBDriverMysql, DBDriverPostgresql, DBDriverTidb, DBDriverSqlite:
		g.fields = append(g.fields, getExpectedSQLForDeletionField(g.isEmbed)...)
		if g.isExtendedAPI {
			var fields []replacer.Field
			replaceFiles, fields = serviceExtendedAPI(r, codeNameGRPC)
			g.fields = append(g.fields, fields...)
		}

	case DBDriverMongodb:
		if g.isExtendedAPI {
			var fields []replacer.Field
			replaceFiles, fields = serviceMongoDBExtendedAPI(r, codeNameHTTP)
			g.fields = append(g.fields, fields...)
		} else {
			replaceFiles = map[string][]string{
				"internal/cache": {
					"userExample.go.mgo",
				},
				"internal/dao": {
					"userExample.go.mgo",
				},
				"internal/service": {
					"service.go", "service_test.go", "userExample.go.mgo", /*"userExample_client_test.go.mgo",*/
				},
			}
			g.fields = append(g.fields, deleteFieldsMark(r, serviceLogicFile+mgoSuffix, startMark, endMark)...)
		}

	default:
		return "", dbDriverErr(g.dbDriver)
	}

	subFiles = append(subFiles, getSubFiles(selectFiles, replaceFiles)...)

	// ignore some directories and files
	ignoreDirs := []string{"cmd/sponge"}
	ignoreFiles := []string{"scripts/swag-docs.sh", "configs/serverNameExample_cc.yml"}

	r.SetSubDirsAndFiles(subDirs, subFiles...)
	r.SetIgnoreSubDirs(ignoreDirs...)
	r.SetIgnoreSubFiles(ignoreFiles...)
	_ = r.SetOutputDir(g.outPath, g.serverName+"_"+subTplName)
	fields := g.addFields(r)
	r.SetReplacementFields(fields)
	if err := r.SaveFiles(); err != nil {
		return "", err
	}

	if g.suitedMonoRepo {
		if err := moveProtoFileToAPIDir(g.moduleName, g.serverName, g.suitedMonoRepo, r.GetOutputDir()); err != nil {
			return "", err
		}
	}
	_ = saveGenInfo(g.moduleName, g.serverName, g.suitedMonoRepo, r.GetOutputDir())

	return r.GetOutputDir(), nil
}

func (g *rpcGenerator) addFields(r replacer.Replacer) []replacer.Field {
	repoHost, _ := parseImageRepoAddr(g.repoAddr)

	var fields []replacer.Field
	fields = append(fields, g.fields...)
	fields = append(fields, deleteFieldsMark(r, modelFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, databaseInitDBFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoMgoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoTestFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, protoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, serviceLogicFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, serviceClientFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, serviceClientMgoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, serviceTestFile, startMark, endMark)...)
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
		getReadmeContent(g.moduleName, g.serverName, codeNameGRPC, g.dbDriver, g.suitedMonoRepo))...)
	fields = append(fields, []replacer.Field{
		{ // replace the configuration of the *.yml file
			Old: appConfigFileMark,
			New: rpcServerConfigCode,
		},
		{ // replace the configuration of the *.yml file
			Old: appConfigFileMark2,
			New: getDBConfigCode(g.dbDriver),
		},
		{ // replace the contents of the model/userExample.go file
			Old: modelFileMark,
			New: g.codes[parser.CodeTypeModel],
		},
		{ // replace the contents of the database/init.go file
			Old: databaseInitDBFileMark,
			New: getInitDBCode(g.dbDriver),
		},
		{ // replace the contents of the dao/userExample.go file
			Old: daoFileMark,
			New: g.codes[parser.CodeTypeDAO],
		},
		{ // replace the contents of the service/userExample.go file
			Old: embedTimeMark,
			New: getEmbedTimeCode(g.isEmbed),
		},
		{ // replace the contents of the v1/userExample.proto file
			Old: protoFileMark,
			New: g.codes[parser.CodeTypeProto],
		},
		{ // replace the contents of the proto.sh file
			Old: protoShellFileGRPCMark,
			New: protoShellGRPCMark,
		},
		{ // replace the contents of the scripts/proto.sh file
			Old: protoShellFileMark,
			New: protoShellServiceTmplCode,
		},
		{ // replace the contents of the service/userExample_client_test.go file
			Old: serviceFileMark,
			New: adjustmentOfIDType(g.codes[parser.CodeTypeService], g.dbDriver, g.isCommonStyle),
		},
		{ // replace the contents of the Dockerfile file
			Old: dockerFileMark,
			New: dockerFileGrpcCode,
		},
		{ // replace the contents of the Dockerfile_build file
			Old: dockerFileBuildMark,
			New: dockerFileBuildGrpcCode,
		},
		{ // replace the contents of the image-build.sh file
			Old: imageBuildFileMark,
			New: imageBuildFileGrpcCode,
		},
		{ // replace the contents of the image-build-local.sh file
			Old: imageBuildLocalFileMark,
			New: imageBuildLocalFileGrpcCode,
		},
		{ // replace the contents of the docker-compose.yml file
			Old: dockerComposeFileMark,
			New: dockerComposeFileGrpcCode,
		},
		//{ // replace the contents of the *-configmap.yml file
		//	Old: deploymentConfigFileMark,
		//	New: getDBConfigCode(g.dbDriver, true),
		//},
		{ // replace the contents of the *-deployment.yml file
			Old: k8sDeploymentFileMark,
			New: k8sDeploymentFileGrpcCode,
		},
		{ // replace the contents of the *-svc.yml file
			Old: k8sServiceFileMark,
			New: k8sServiceFileGrpcCode,
		},
		{ // replace github.com/go-dev-frame/sponge/templates/sponge
			Old: selfPackageName + "/" + r.GetSourcePath(),
			New: g.moduleName,
		},
		// replace directory name
		{
			Old: strings.Join([]string{"api", "userExample", "v1"}, gofile.GetPathDelimiter()),
			New: strings.Join([]string{"api", g.serverName, "v1"}, gofile.GetPathDelimiter()),
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
			Old: "api/userExample/v1",
			New: fmt.Sprintf("api/%s/v1", g.serverName),
		},
		{
			Old: "api.userExample.v1",
			New: fmt.Sprintf("api.%s.v1", g.serverName), // protobuf package no "-" signs allowed
		},
		{
			Old: "sponge api docs",
			New: g.serverName + apiDocsSuffix,
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
			Old: "_userExampleNO       = 2",
			New: fmt.Sprintf("_userExampleNO       = %d", rand.Intn(99)+1),
		},
		{
			Old: "serverNameExample",
			New: g.serverName,
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
			Old: "_grpcExample",
			New: "",
		},
		{
			Old: "_mixExample",
			New: "",
		},
		{
			Old: "root:123456@(192.168.3.37:3306)/account",
			New: g.dbDSN,
		},
		{
			Old: "root:123456@192.168.3.37:27017/account",
			New: g.dbDSN,
		},
		{
			Old: "root:123456@192.168.3.37:5432/account?sslmode=disable",
			New: adaptPgDsn(g.dbDSN),
		},
		{
			Old: "test/sql/sqlite/sponge.db",
			New: sqliteDSNAdaptation(g.dbDriver, g.dbDSN),
		},
		{
			Old: showDbNameMark,
			New: CurrentDbDriver(g.dbDriver),
		},
		{
			Old: "init.go.mgo",
			New: "init.go",
		},
		{
			Old: "mongodb.go.mgo",
			New: "mongodb.go",
		},
		{
			Old: "userExample_client_test.go.mgo",
			New: "userExample_client_test.go",
		},
		{
			Old: "userExample.go.mgo",
			New: "userExample.go",
		},
		{
			Old:             "UserExample",
			New:             g.codes[parser.TableName],
			IsCaseSensitive: true,
		},
	}...)

	fields = append(fields, getGRPCServiceFields()...)

	if g.suitedMonoRepo {
		fs := serverCodeFields(codeNameGRPC, g.moduleName, g.serverName)
		fields = append(fields, fs...)
	}

	return fields
}

func commonGRPCFields(r replacer.Replacer) []replacer.Field {
	return commonServiceFields(r)
}

func commonGRPCExtendedFields(r replacer.Replacer) []replacer.Field {
	return commonServiceExtendedFields(r)
}
