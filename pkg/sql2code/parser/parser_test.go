package parser

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jinzhu/inflection"
	"github.com/stretchr/testify/assert"
	"github.com/zhufuyi/sqlparser/dependency/mysql"
	"github.com/zhufuyi/sqlparser/dependency/types"
)

func TestParseMysqlSQL(t *testing.T) {
	sql := `CREATE TABLE orders (
	   order_id bigint NOT NULL AUTO_INCREMENT COMMENT 'order id',
	   user_id bigint NOT NULL COMMENT 'user id',
	   total_amount decimal(10,2) NOT NULL COMMENT 'total amount',
	   order_remark json NOT NULL COMMENT 'order remark',
	   order_status ENUM('Pending Payment', 'Paid', 'Shipped', 'Completed', 'Cancelled') NOT NULL DEFAULT 'active' COMMENT 'order status',
	   pay_type SET('Alipay', 'WeChat Pay', 'Bank Card') NOT NULL DEFAULT '' COMMENT 'pay type',
	   is_deleted bit(1) NOT NULL DEFAULT b'0' COMMENT '0-no，1-yes',
	   created_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created time'
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='orders table';`

	codes, err := ParseSQL(sql, WithJSONTag(1), WithDBDriver(DBDriverMysql))
	assert.Nil(t, err)
	assert.NotEmpty(t, codes)
	printCode(codes)
}

func TestParseSQL(t *testing.T) {
	sqls := []string{`create table user (
    id          bigint unsigned auto_increment,
    created_at  datetime        null,
    updated_at  datetime        null,
    deleted_at  datetime        null,
    name        char(50)        not null comment '用户名',
    password    char(100)       not null comment '密码',
    email       char(50)        not null comment '邮件',
    phone       bigint unsigned not null comment '手机号码',
    age         tinyint         not null comment '年龄',
    gender      tinyint         not null comment '性别，1:男，2:女，3:未知',
    status      tinyint         not null comment '账号状态，1:未激活，2:已激活，3:封禁',
    login_state tinyint         not null comment '登录状态，1:未登录，2:已登录',
    primary key (id),
    constraint user_email_uindex
        unique (email)
);`,

		`create table user_order (
    id         varchar(36)     not null comment '订单id',
    product_id varchar(36)     not null comment '商品id',
    user_id    bigint unsigned not null comment '用户id',
    status     smallint        null comment '0:未支付, 1:已支付, 2:已取消',
    created_at timestamp       null comment '创建时间',
    updated_at timestamp       null comment '更新时间',
    primary key (id)
);`,

		`create table user_str (
    user_id    varchar(36)  not null comment '用户id',
    username   varchar(50)  not null comment '用户名',
    email      varchar(100) not null comment '邮箱',
    created_at datetime     null comment '创建时间',
    primary key (user_id),
    constraint email
        unique (email)
);`,

		`create table user_no_primary (
    username   varchar(50)  not null comment '用户名',
    email      varchar(100) not null comment '邮箱',
    user_id    varchar(36)  not null comment '用户id',
    created_at datetime     null comment '创建时间',
    constraint email
        unique (email)
);`}

	for _, sql := range sqls {
		codes, err := ParseSQL(sql, WithJSONTag(0), WithEmbed())
		assert.Nil(t, err)
		for k, v := range codes {
			if k == CodeTypeTableInfo {
				continue
			}
			assert.NotEmpty(t, k)
			assert.NotEmpty(t, v)
		}
		//printCode(codes)

		codes, err = ParseSQL(sql, WithJSONTag(1), WithWebProto(), WithDBDriver(DBDriverMysql))
		assert.Nil(t, err)
		for k, v := range codes {
			if k == CodeTypeTableInfo {
				continue
			}
			assert.NotEmpty(t, k)
			assert.NotEmpty(t, v)
		}
		//printCode(codes)

		codes, err = ParseSQL(sql, WithJSONTag(0), WithDBDriver(DBDriverPostgresql))
		assert.Nil(t, err)
		for k, v := range codes {
			if k == CodeTypeTableInfo {
				continue
			}
			assert.NotEmpty(t, k)
			assert.NotEmpty(t, v)
		}
		//printCode(codes)

		codes, err = ParseSQL(sql, WithJSONTag(0), WithDBDriver(DBDriverSqlite))
		assert.Nil(t, err)
		for k, v := range codes {
			if k == CodeTypeTableInfo {
				continue
			}
			assert.NotEmpty(t, k)
			assert.NotEmpty(t, v)
		}
		//printCode(codes)

		codes, err = ParseSQL(sql, WithDBDriver(DBDriverSqlite), WithCustomTemplate())
		assert.Nil(t, err)
		for k, v := range codes {
			if k == CodeTypeTableInfo {
				assert.NotEmpty(t, k)
				assert.NotEmpty(t, v)
				break
			}
		}
		//printCode(codes)
	}
}

func TestParseSqlWithTablePrefix(t *testing.T) {
	sql := `CREATE TABLE t_person_info (
  id BIGINT(11) AUTO_INCREMENT NOT NULL COMMENT 'id',
  age INT(11) unsigned NULL,
  name VARCHAR(30) NOT NULL DEFAULT 'default_name' COMMENT 'name',
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  login_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gender INT(8) NULL,
  num INT(11) DEFAULT 3 NULL,
  comment TEXT,
  PRIMARY KEY (id)
  ) COMMENT="person info";`

	codes, err := ParseSQL(sql, WithTablePrefix("t_"), WithJSONTag(0), WithNullStyle(NullDisable))
	assert.Nil(t, err)
	for k, v := range codes {
		if k == CodeTypeTableInfo {
			continue
		}
		assert.NotEmpty(t, k)
		assert.NotEmpty(t, v)
	}
	//printCode(codes)

	codes, err = ParseSQL(sql, WithTablePrefix("t_"), WithJSONTag(0), WithCustomTemplate())
	assert.Nil(t, err)
	for k, v := range codes {
		if k != CodeTypeTableInfo {
			continue
		}
		assert.NotEmpty(t, k)
		assert.NotEmpty(t, v)
	}
	jsonData := codes[CodeTypeTableInfo]
	t.Log(jsonData)
	t.Log(UnMarshalTableInfo(jsonData))

	codes, err = ParseSQL(sql, WithTablePrefix("t_"), WithJSONTag(0), WithEmbed())
	assert.Nil(t, err)
	for k, v := range codes {
		if k == CodeTypeTableInfo {
			continue
		}
		assert.NotEmpty(t, k)
		assert.NotEmpty(t, v)
	}
	//printCode(codes)

	codes, err = ParseSQL(sql, WithTablePrefix("t_"), WithJSONTag(0), WithWebProto())
	assert.Nil(t, err)
	for k, v := range codes {
		if k == CodeTypeTableInfo {
			continue
		}
		assert.NotEmpty(t, k)
		assert.NotEmpty(t, v)
	}
	//printCode(codes)

	codes, err = ParseSQL(sql, WithTablePrefix("t_"), WithJSONTag(0), WithDBDriver(DBDriverPostgresql))
	assert.Nil(t, err)
	for k, v := range codes {
		if k == CodeTypeTableInfo {
			continue
		}
		assert.NotEmpty(t, k)
		assert.NotEmpty(t, v)
	}
	//printCode(codes)
}

var testData = [][]string{
	{
		"CREATE TABLE information (age INT(11) NULL);",
		"Age int `gorm:\"column:age\"`", "",
	},
	{
		"CREATE TABLE information (age BIGINT(11) NULL COMMENT 'is age');",
		"Age int64 `gorm:\"column:age\"` // is age", "",
	},
	{
		"CREATE TABLE information (id BIGINT(11) PRIMARY KEY AUTO_INCREMENT);",
		"ID int64 `gorm:\"column:id;primary_key;AUTO_INCREMENT\"`", "",
	},
	{
		"CREATE TABLE information (user_ip varchar(20));",
		"UserIP string `gorm:\"column:user_ip\"`", "",
	},
	{
		"CREATE TABLE information (created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP);",
		"CreatedAt time.Time `gorm:\"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL\"`", "time",
	},
	{
		"CREATE TABLE information (num INT(11) DEFAULT 3 NULL);",
		"Num int `gorm:\"column:num;default:3\"`", "",
	},
	{
		"CREATE TABLE information (num double(5,6) DEFAULT 31.50 NULL);",
		"Num float64 `gorm:\"column:num;default:31.50\"`", "",
	},
	{
		"CREATE TABLE information (comment TEXT);",
		"Comment string `gorm:\"column:comment\"`", "",
	},
	{
		"CREATE TABLE information (comment TINYTEXT);",
		"Comment string `gorm:\"column:comment\"`", "",
	},
	{
		"CREATE TABLE information (comment LONGTEXT);",
		"Comment string `gorm:\"column:comment\"`", "",
	},
}

func TestParseSQLs(t *testing.T) {
	for i, test := range testData {
		msg := fmt.Sprintf("sql-%d", i)
		codes, err := ParseSQL(test[0], WithNoNullType())
		if !assert.NoError(t, err, msg) {
			continue
		}
		for k, v := range codes {
			if len(v) > 100 {
				v = v[:100]
			}
			t.Log(i+1, k, v)
		}
	}
}

func Test_parseOption(t *testing.T) {
	opts := []Option{
		WithDBDriver("foo"),
		WithFieldTypes(map[string]string{"foo": "bar"}),
		WithCharset("foo"),
		WithCollation("foo"),
		WithTablePrefix("foo"),
		WithColumnPrefix("foo"),
		WithJSONTag(1),
		WithNoNullType(),
		WithNullStyle(1),
		WithPackage("model"),
		WithGormType(),
		WithForceTableName(),
		WithEmbed(),
	}
	o := parseOption(opts)
	assert.NotNil(t, o)
}

func Test_mysqlToGoType(t *testing.T) {
	fields := []*types.FieldType{
		{Tp: uint8('n')},
		{Tp: mysql.TypeTiny},
		{Tp: mysql.TypeLonglong},
		{Tp: mysql.TypeFloat},
		{Tp: mysql.TypeString},
		{Tp: mysql.TypeTimestamp},
		{Tp: mysql.TypeDecimal},
		{Tp: mysql.TypeJSON},
		{Tp: mysql.TypeBit},
	}
	var names []string
	for _, d := range fields {
		name1, _, _ := mysqlToGoType(d, NullInSql)
		name2, _, _ := mysqlToGoType(d, NullInPointer)
		names = append(names, name1, name2)
	}
	t.Log(names)
}

func Test_goTypeToProto(t *testing.T) {
	fields := []tmplField{
		{GoType: "int"},
		{GoType: "uint"},
		{GoType: "time.Time"},
	}
	v := goTypeToProto(fields, 1, false)
	assert.NotNil(t, v)
}

func Test_initTemplate(t *testing.T) {
	initTemplate()

	defer func() { recover() }()
	modelStructTmplRaw = "{{if .foo}}"
	modelTmplRaw = "{{if .foo}}"
	updateFieldTmplRaw = "{{if .foo}}"
	handlerCreateStructTmplRaw = "{{if .foo}}"
	handlerUpdateStructTmplRaw = "{{if .foo}}"
	handlerDetailStructTmplRaw = "{{if .foo}}"
	modelJSONTmplRaw = "{{if .foo}}"
	protoFileTmplRaw = "{{if .foo}}"
	protoFileSimpleTmplRaw = "{{if .foo}}"
	protoFileForWebTmplRaw = "{{if .foo}}"
	protoFileForSimpleWebTmplRaw = "{{if .foo}}"
	protoMessageCreateTmplRaw = "{{if .foo}}"
	protoMessageUpdateTmplRaw = "{{if .foo}}"
	protoMessageDetailTmplRaw = "{{if .foo}}"
	serviceCreateStructTmplRaw = "{{if .foo}}"
	serviceUpdateStructTmplRaw = "{{if .foo}}"
	serviceStructTmplRaw = "{{if .foo}}"
	initTemplate()
}

func TestGetMysqlTableInfo(t *testing.T) {
	info, err := GetMysqlTableInfo("root:123456@(192.168.3.37:3306)/account", "user_order")
	t.Log(err, info)
}

func TestGetPostgresqlTableInfo(t *testing.T) {
	var (
		dbname    = "account"
		tableName = "user_order"
		dsn       = fmt.Sprintf("host=192.168.3.37 port=5432 user=root password=123456 dbname=%s sslmode=disable", dbname)
	)

	fields, err := GetPostgresqlTableInfo(dsn, tableName)
	if err != nil {
		t.Log(err)
		return
	}
	printPGFields(fields)
	sql, fieldTypes := ConvertToSQLByPgFields(tableName, fields)
	t.Log(sql)
	t.Log(fieldTypes)
}

func Test_getPostgresqlTableFields(t *testing.T) {
	defer func() { _ = recover() }()
	_, _ = getPostgresqlTableFields(nil, "foobar")
}

func TestGetSqliteTableInfo(t *testing.T) {
	dbFile := filepath.Join("..", "..", "..", "test", "sql", "sqlite", "sponge.db")
	info, err := GetSqliteTableInfo(dbFile, "user_order")
	t.Log(err, info)
}

func TestGetMongodbTableInfo(t *testing.T) {
	var (
		dbname    = "account"
		tableName = "people"
		dsn       = fmt.Sprintf("mongodb://root:123456@192.168.3.37:27017/%s", dbname)
	)

	fields, err := GetMongodbTableInfo(dsn, tableName)
	if err != nil {
		t.Log(err)
		return
	}
	sql, fieldTypes := ConvertToSQLByMgoFields(tableName, fields)
	t.Log(sql)
	t.Log(fieldTypes)
}

func TestConvertToSQLByPgFields(t *testing.T) {
	fields := []*PGField{
		{Name: "id", Type: "smallint"},
		{Name: "name", Type: "character", Lengthvar: 24, Notnull: false},
		{Name: "age", Type: "smallint", Notnull: true},
	}
	sql, tps := ConvertToSQLByPgFields("foobar", fields)
	t.Log(sql, tps)
}

func Test_PGField_getMysqlType(t *testing.T) {
	fields := []*PGField{
		{Type: "smallint"},
		{Type: "bigint"},
		{Type: "real"},
		{Type: "decimal"},
		{Type: "double precision"},
		{Type: "money"},
		{Type: "character", Lengthvar: 24},
		{Type: "text"},
		{Type: "timestamp"},
		{Type: "date"},
		{Type: "time"},
		{Type: "interval"},
		{Type: "boolean"},
	}
	for _, field := range fields {
		t.Log(field.getMysqlType(), getType(field))
	}
}

func Test_SqliteField_getMysqlType(t *testing.T) {
	fields := []*SqliteField{
		{Type: "integer"},
		{Type: "text"},
		{Type: "real"},
		{Type: "numeric"},
		{Type: "blob"},
		{Type: "datetime"},
		{Type: "boolean"},
		{Type: "unknown_type"},
	}
	for _, field := range fields {
		t.Log(field.getMysqlType())
	}
}

func printCode(code map[string]string) {
	for k, v := range code {
		fmt.Printf("\n\n----------------- %s --------------------\n%s\n", k, v)
	}
}

func printPGFields(fields []*PGField) {
	fmt.Printf("%-20v %-20v %-20v %-20v %-20v %-20v %-20v\n", "Name", "Type", "Length", "Lengthvar", "Notnull", "Comment", "IsPrimaryKey")
	for _, p := range fields {
		fmt.Printf("%-20v %-20v %-20v %-20v %-20v %-20v %-20v\n", p.Name, p.Type, p.Length, p.Lengthvar, p.Notnull, p.Comment, p.IsPrimaryKey)
	}
}

func Test_getMongodbTableFields(t *testing.T) {
	fields := []*MgoField{
		{
			Name:           "_id",
			Type:           "primitive.ObjectID",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "age",
			Type:           "int",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "birthday",
			Type:           "time.Time",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:      "home_address",
			Type:      "HomeAddress",
			ObjectStr: "type HomeAddress struct { Street string `bson:\"street\" json:\"street\"`; City string `bson:\"city\" json:\"city\"`; State string `bson:\"state\" json:\"state\"`; Zip int `bson:\"zip\" json:\"zip\"` } ",
			ProtoObjectStr: `message HomeAddress {
			string street = 1;
			string city = 2;
			string state = 3;
			int32 zip = 4;
		}
		`,
		},
		{
			Name:           "interests",
			Type:           "[]string",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "is_child",
			Type:           "bool",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "name",
			Type:           "string",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "numbers",
			Type:           "[]int",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:      "shop_addresses",
			Type:      "[]ShopAddress",
			ObjectStr: "type ShopAddress  struct { CityO string `bson:\"city_o\" json:\"cityO\"`; StateO string `bson:\"state_o\" json:\"stateO\"` }",
			ProtoObjectStr: `message ShopAddress  {
		string city_o = 1;
		string state_o = 2;
		}
		`,
		},
		{
			Name:           "created_at",
			Type:           "time.Time",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "updated_at",
			Type:           "time.Time",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
		{
			Name:           "deleted_at",
			Type:           "*time.Time",
			ObjectStr:      "",
			ProtoObjectStr: "",
		},
	}

	SetJSONTagCamelCase()
	goStructs := MgoFieldToGoStruct("foobar", fields)
	t.Log(goStructs)

	sql, fieldsMap := ConvertToSQLByMgoFields("foobar", fields)
	t.Log(sql)
	opts := []Option{
		WithDBDriver(DBDriverMongodb),
		WithFieldTypes(fieldsMap),
		WithJSONTag(1),
	}
	codes, err := ParseSQL(sql, opts...)
	if err != nil {
		t.Error(err)
		return
	}
	_ = codes
	//printCode(codes)

	SetJSONTagSnakeCase()
	sql, fieldsMap = ConvertToSQLByMgoFields("foobar", fields)
	t.Log(sql)
	opts = []Option{
		WithDBDriver(DBDriverMongodb),
		WithFieldTypes(fieldsMap),
		WithJSONTag(1),
		WithWebProto(),
		WithExtendedAPI(),
	}
	codes, err = ParseSQL(sql, opts...)
	if err != nil {
		t.Error(err)
		return
	}
	//printCode(codes)
}

func Test_toSingular(t *testing.T) {
	strs := []string{
		"users",
		"address",
		"addresses",
	}
	for _, str := range strs {
		t.Log(str, toSingular(str))
	}
}

func Test_embedTimeFields(t *testing.T) {
	names := []string{"age"}

	fields := embedTimeField(names, []*MgoField{})
	t.Log(fields)

	names = []string{
		"created_at",
		"updated_at",
		"deleted_at",
	}
	fields = embedTimeField(names, []*MgoField{})
	t.Log(fields)
}

func TestCrudInfo(t *testing.T) {
	data := tmplData{
		TableName:    "User",
		TName:        "user",
		NameFunc:     false,
		RawTableName: "user",
		Fields: []tmplField{
			{
				ColName:  "name",
				Name:     "Name",
				GoType:   "string",
				Tag:      "json:\"name\"",
				Comment:  "姓名",
				JSONName: "name",
				DBDriver: "mysql",
			},
			{
				ColName:  "age",
				Name:     "Age",
				GoType:   "int",
				Tag:      "json:\"age\"",
				Comment:  "年龄",
				JSONName: "age",
				DBDriver: "mysql",
			},
			{
				ColName:  "created_at",
				Name:     "CreatedAt",
				GoType:   "time.Time",
				Tag:      "json:\"created_at\"",
				Comment:  "创建时间",
				JSONName: "createdAt",
				DBDriver: "mysql",
			},
		},
		Comment:         "用户信息",
		SubStructs:      "",
		ProtoSubStructs: "",
		DBDriver:        "mysql",
	}

	info := newCrudInfo(data)

	isPrimary := info.isIDPrimaryKey()
	assert.Equal(t, false, isPrimary)

	code := info.getCode()
	assert.Contains(t, code, `"tableNameCamel":"User","tableNameCamelFCL":"user"`)

	grpcValidation := info.GetGRPCProtoValidation()
	assert.Contains(t, grpcValidation, "validate.rules")

	webValidation := info.GetWebProtoValidation()
	assert.Contains(t, webValidation, "validate.rules")

	info = nil
	_ = info.isIDPrimaryKey()
	_ = info.getCode()
	_ = info.GetGRPCProtoValidation()
	_ = info.GetWebProtoValidation()
}

func Test_customEndOfLetterToLower(t *testing.T) {
	names := []string{
		"ID",
		"IP",
		"userID",
		"orderID",
		"LocalIP",
		"bus",
		"BUS",
		"x",
		"s",
	}
	for _, name := range names {
		t.Log(customEndOfLetterToLower(name, inflection.Plural(name)))
	}
}
