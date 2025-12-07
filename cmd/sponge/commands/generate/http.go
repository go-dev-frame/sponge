package generate

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/fatih/color"
	"github.com/huandu/xstrings"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/pkg/replacer"
	"github.com/go-dev-frame/sponge/pkg/sql2code"
	"github.com/go-dev-frame/sponge/pkg/sql2code/parser"
)

// HTTPCommand generate web server code
func HTTPCommand() *cobra.Command {
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
		Use:   "http",
		Short: "Generate web server code based on sql",
		Long:  "Generate web server code based on sql.",
		Example: color.HiBlackString(`  # Generate web server code.
  sponge web http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # Generate web server code with multiple table names.
  sponge web http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=t1,t2

  # Generate web server code with extended api.
  sponge web http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --extended-api=true

  # Generate web server code and specify the output directory, Note: code generation will be canceled when the latest generated file already exists.
  sponge web http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --out=./yourServerDir

  # Generate web server code and specify the docker image repository address.
  sponge web http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --repo-addr=192.168.3.37:9443/user-name --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # If you want the generated code to suited to mono-repo, you need to set the parameter --suited-mono-repo=true`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var firstTable string
			var handlerTableNames []string
			tableNames := strings.Split(dbTables, ",")
			if len(tableNames) == 1 {
				firstTable = tableNames[0]
			} else if len(tableNames) > 1 {
				firstTable = tableNames[0]
				handlerTableNames = tableNames[1:]
			}

			projectName, serverName, err = convertProjectAndServerName(projectName, serverName)
			if err != nil {
				return err
			}

			if sqlArgs.DBDriver == DBDriverMongodb {
				sqlArgs.IsEmbed = false
			}

			if suitedMonoRepo {
				outPath = changeOutPath(outPath, serverName)
			}

			sqlArgs.DBTable = firstTable
			codes, err := sql2code.Generate(&sqlArgs)
			if err != nil {
				return err
			}
			g := &httpGenerator{
				moduleName:     moduleName,
				serverName:     serverName,
				projectName:    projectName,
				repoAddr:       repoAddr,
				dbDSN:          sqlArgs.DBDsn,
				dbDriver:       sqlArgs.DBDriver,
				codes:          codes,
				outPath:        outPath,
				isExtendedAPI:  sqlArgs.IsExtendedAPI,
				isEmbed:        sqlArgs.IsEmbed,
				suitedMonoRepo: suitedMonoRepo,
			}
			outPath, err = g.generateCode()
			if err != nil {
				return err
			}

			for _, handlerTableName := range handlerTableNames {
				if handlerTableName == "" {
					continue
				}

				sqlArgs.DBTable = handlerTableName
				codes, err := sql2code.Generate(&sqlArgs)
				if err != nil {
					return err
				}

				hg := &handlerGenerator{
					moduleName:    moduleName,
					dbDriver:      sqlArgs.DBDriver,
					codes:         codes,
					outPath:       outPath,
					isEmbed:       sqlArgs.IsEmbed,
					isExtendedAPI: sqlArgs.IsExtendedAPI,
					serverName:    serverName,

					suitedMonoRepo: suitedMonoRepo,
				}
				outPath, err = hg.generateCode()
				if err != nil {
					return err
				}
			}

			fmt.Printf(`
using help:
  1. open a terminal and execute the command to generate the swagger documentation: make docs
  2. compile and run server: make run
  3. access http://localhost:8080/swagger/index.html in your browser, and test the http CRUD api.

`)
			fmt.Printf("generate %s's web server code successfully, out = %s\n", serverName, outPath)

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
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output directory, default is ./serverName_http_<time>, if suited-mono-repo = true, output directory is serverName")

	return cmd
}

type httpGenerator struct {
	moduleName     string
	serverName     string
	projectName    string
	repoAddr       string
	dbDSN          string
	dbDriver       string
	codes          map[string]string
	outPath        string
	isEmbed        bool
	isExtendedAPI  bool
	suitedMonoRepo bool

	fields        []replacer.Field
	isCommonStyle bool
}

func (g *httpGenerator) generateCode() (string, error) {
	subTplName := codeNameHTTP
	r, _ := replacer.New(SpongeDir)
	if r == nil {
		return "", errors.New("replacer is nil")
	}

	// specify the subdirectory and files
	subDirs := []string{
		"cmd/serverNameExample_httpExample", "sponge/configs", "sponge/deployments", "sponge/scripts",
	}
	subFiles := []string{
		"sponge/.gitignore", "sponge/.golangci.yml", "sponge/go.mod", "sponge/go.sum",
		"sponge/Jenkinsfile", "sponge/Makefile-for-http", "sponge/README.md",
	}
	if g.suitedMonoRepo {
		subFiles = removeElements(subFiles, "sponge/go.mod", "sponge/go.sum")
	}

	selectFiles := map[string][]string{
		"docs": {
			"docs.go",
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
			"systemCode_http.go", "userExample_http.go",
		},
		"internal/handler": {
			"userExample.go", "userExample_test.go",
		},
		"internal/model": {
			"userExample.go",
		},
		"internal/routers": {
			"routers.go", "userExample.go",
		},
		"internal/server": {
			"http.go.noregistry", "http_option.go.noregistry",
		},
		"internal/types": {
			"swagger_types.go", "userExample_types.go",
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
		selectFiles["internal/ecode"] = []string{"systemCode_http.go", "userExample_http.go.tpl"}
		selectFiles["internal/handler"] = []string{"userExample.go.tpl"}
		selectFiles["internal/routers"] = []string{"routers.go", "userExample.go.tpl"}
		selectFiles["internal/types"] = []string{"swagger_types.go", "userExample_types.go.tpl"}
		var fields []replacer.Field
		if g.isExtendedAPI {
			selectFiles["internal/dao"] = []string{"userExample.go.exp.tpl"}
			selectFiles["internal/ecode"] = []string{"systemCode_http.go", "userExample_http.go.exp.tpl"}
			selectFiles["internal/handler"] = []string{"userExample.go.exp.tpl"}
			selectFiles["internal/routers"] = []string{"routers.go", "userExample.go.exp.tpl"}
			selectFiles["internal/types"] = []string{"swagger_types.go", "userExample_types.go.exp.tpl"}
			fields = commonHTTPExtendedFields(r)
		} else {
			fields = commonHTTPFields(r)
		}
		contentFields, err := replaceFilesContent(r, getTemplateFiles(selectFiles), crudInfo)
		if err != nil {
			return "", err
		}
		g.fields = append(g.fields, contentFields...)
		g.fields = append(g.fields, fields...)
	}

	replaceFiles := make(map[string][]string)
	switch strings.ToLower(g.dbDriver) {
	case DBDriverMysql, DBDriverPostgresql, DBDriverTidb, DBDriverSqlite:
		g.fields = append(g.fields, getExpectedSQLForDeletionField(g.isEmbed)...)
		if g.isExtendedAPI {
			var fields []replacer.Field
			if !crudInfo.CheckCommonType() {
				replaceFiles, fields = handlerExtendedAPI(r, codeNameHTTP)
			}
			g.fields = append(g.fields, fields...)
		}

	case DBDriverMongodb:
		if g.isExtendedAPI {
			var fields []replacer.Field
			replaceFiles, fields = handlerMongoDBExtendedAPI(r, codeNameHTTP)
			g.fields = append(g.fields, fields...)
		} else {
			replaceFiles = map[string][]string{
				"internal/cache": {
					"userExample.go.mgo",
				},
				"internal/dao": {
					"userExample.go.mgo",
				},
				"internal/handler": {
					"userExample.go.mgo",
				},
				"internal/types": {
					"swagger_types.go", "userExample_types.go.mgo",
				},
			}
		}

	default:
		return "", dbDriverErr(g.dbDriver)
	}

	subFiles = append(subFiles, getSubFiles(selectFiles, replaceFiles)...)

	// ignore some directories and files
	ignoreDirs := []string{"cmd/sponge"}
	ignoreFiles := []string{"scripts/image-rpc-test.sh", "scripts/patch.sh", "scripts/protoc.sh",
		"scripts/proto-doc.sh", "configs/serverNameExample_cc.yml"}

	r.SetSubDirsAndFiles(subDirs, subFiles...)
	r.SetIgnoreSubDirs(ignoreDirs...)
	r.SetIgnoreSubFiles(ignoreFiles...)
	_ = r.SetOutputDir(g.outPath, g.serverName+"_"+subTplName)
	fields := g.addFields(r)
	r.SetReplacementFields(fields)
	if err := r.SaveFiles(); err != nil {
		return "", err
	}
	_ = saveGenInfo(g.moduleName, g.serverName, g.suitedMonoRepo, r.GetOutputDir())

	return r.GetOutputDir(), nil
}

func (g *httpGenerator) addFields(r replacer.Replacer) []replacer.Field {
	repoHost, _ := parseImageRepoAddr(g.repoAddr)

	var fields []replacer.Field
	fields = append(fields, g.fields...)
	fields = append(fields, deleteFieldsMark(r, modelFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, databaseInitDBFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoMgoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoTestFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, typesFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, typesMgoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, handlerTestFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, httpFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, httpFile+".noregistry", startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, dockerFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, dockerFileBuild, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, dockerComposeFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, k8sDeploymentFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, k8sServiceFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, imageBuildFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, imageBuildLocalFile, wellStartMark, wellEndMark)...)
	//fields = append(fields, deleteAllFieldsMark(r, makeFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteFieldsMark(r, gitIgnoreFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteAllFieldsMark(r, protoShellFile, wellStartMark, wellEndMark)...)
	fields = append(fields, deleteAllFieldsMark(r, appConfigFile, wellStartMark, wellEndMark)...)
	//fields = append(fields, deleteFieldsMark(r, deploymentConfigFile, wellStartMark, wellEndMark)...)
	fields = append(fields, replaceFileContentMark(r, readmeFile,
		getReadmeContent(g.moduleName, g.serverName, codeNameHTTP, g.dbDriver, g.suitedMonoRepo))...)
	fields = append(fields, []replacer.Field{
		{ // replace the configuration of the *.yml file
			Old: appConfigFileMark,
			New: httpServerConfigCode,
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
		{ // replace the contents of the handler/userExample.go file
			Old: handlerFileMark,
			New: adjustmentOfIDType(g.codes[parser.CodeTypeHandler], g.dbDriver, g.isCommonStyle),
		},
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
		//	New: getDBConfigCode(g.dbDriver, true),
		//},
		{ // replace the contents of the *-deployment.yml file
			Old: k8sDeploymentFileMark,
			New: k8sDeploymentFileHTTPCode,
		},
		{ // replace the contents of the *-svc.yml file
			Old: k8sServiceFileMark,
			New: k8sServiceFileHTTPCode,
		},
		{ // replace github.com/go-dev-frame/sponge/templates/sponge
			Old: selfPackageName + "/" + r.GetSourcePath(),
			New: g.moduleName,
		},
		{
			Old: protoShellFileGRPCMark,
			New: "",
		},
		{
			Old: protoShellFileMark,
			New: "",
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
			Old: "userExampleNO       = 1",
			New: fmt.Sprintf("userExampleNO = %d", rand.Intn(99)+1),
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
			Old: "_httpExample",
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
			Old: "Makefile-for-http",
			New: "Makefile",
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
			Old: "userExample_types.go.mgo",
			New: "userExample_types.go",
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

	fields = append(fields, getHTTPServiceFields()...)

	if g.suitedMonoRepo {
		fs := serverCodeFields(codeNameHTTP, g.moduleName, g.serverName)
		fields = append(fields, fs...)
	}

	return fields
}

func commonHTTPFields(r replacer.Replacer) []replacer.Field {
	return commonHandlerFields(r)
}

func commonHTTPExtendedFields(r replacer.Replacer) []replacer.Field {
	return commonHandlerExtendedFields(r)
}
