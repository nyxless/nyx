package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
)

var appname string
var sqlFile string
var tableNames string
var confName string
var force bool
var genMode string
var group string
var hashNum string
var hashTable map[string]int

// Table 表示数据库表结构
type Table struct {
	Name       string
	Comment    string
	Fields     []Field
	HasTime    bool
	PrimaryKey string
	PrimaryInt bool
	HashNum    int
}

// Field 表示表字段结构
type Field struct {
	Name       string
	Type       string
	OrmType    string
	IsPrimary  bool
	IsNullable bool
	Comment    string
	Default    string
	Extra      string
}

var intTypes = map[string]string{
	"int":    "",
	"int8":   "",
	"int16":  "",
	"int32":  "",
	"int64":  "",
	"uint":   "",
	"uint32": "",
	"uint64": "",
}

// 映射 字段 类型到 Go 类型
var typeMappings = map[string]string{ // {{{
	"int":        "int",
	"int32":      "int32",
	"int64":      "int64",
	"uint":       "uint",
	"uint32":     "uint32",
	"uint64":     "uint64",
	"integer":    "int",
	"tinyint":    "int8",
	"smallint":   "int16",
	"mediumint":  "int32",
	"bigint":     "int64",
	"string":     "string",
	"varchar":    "string",
	"char":       "string",
	"text":       "string",
	"longtext":   "string",
	"mediumtext": "string",
	"tinytext":   "string",
	"date":       "time.Time",
	"datetime":   "time.Time",
	"timestamp":  "time.Time",
	"time":       "time.Time",
	"float":      "float32",
	"double":     "float64",
	"decimal":    "float64",
	"numeric":    "float64",
	"bool":       "bool",
	"boolean":    "bool",
	"bit":        "uint",
	"binary":     "[]byte",
	"varbinary":  "[]byte",
	"blob":       "[]byte",
	"longblob":   "[]byte",
	"tinyblob":   "[]byte",
	"json":       "string",
	"enum":       "string",
	"set":        "string",
	"geometry":   "[]byte",
	"point":      "[]byte",
	"linestring": "[]byte",
	"polygon":    "[]byte",

	"datetime64":     "time.Time",
	"decimal32":      "float32",
	"decimal64":      "float64",
	"decimal128":     "float64",
	"fixedstring":    "string",
	"enum8":          "string",
	"enum16":         "string",
	"lowcardinality": "string",
	"nullable":       "interface{}",
} // }}}

// 映射 字段类型到 ORM 类型
var ormTypeMappings = map[string]string{ // {{{
	"int":        "integer",
	"int32":      "integer",
	"int64":      "integer",
	"uint":       "integer",
	"uint32":     "integer",
	"uint64":     "integer",
	"integer":    "integer",
	"tinyint":    "tinyint",
	"smallint":   "smallint",
	"mediumint":  "mediumint",
	"bigint":     "bigint",
	"string":     "string",
	"varchar":    "string",
	"char":       "char",
	"text":       "text",
	"longtext":   "longtext",
	"mediumtext": "mediumtext",
	"tinytext":   "tinytext",
	"date":       "date",
	"datetime":   "datetime",
	"timestamp":  "timestamp",
	"time":       "time",
	"float":      "float",
	"double":     "double",
	"decimal":    "decimal",
	"numeric":    "numeric",
	"bool":       "boolean",
	"boolean":    "boolean",
	"bit":        "bit",
	"binary":     "binary",
	"varbinary":  "varbinary",
	"blob":       "blob",
	"longblob":   "longblob",
	"tinyblob":   "tinyblob",
	"json":       "json",
	"enum":       "enum",
	"set":        "set",

	"datetime64":  "datetime64",
	"decimal32":   "decimal32",
	"decimal64":   "decimal64",
	"decimal128":  "decimal128",
	"fixedstring": "fixedstring",
	"enum8":       "enum8",
	"enum16":      "enum16",
} // }}}

func showHelp() { // {{{
	fmt.Println("Usage: nyx gen orm <options>")
	fmt.Println(`Options:
	-f        强制生成，如果已存在则覆盖
	-c        在配置文件中的配置名, 默认: db_master,db_slave 
	-s        sql文件名 
	-t        指定表名, 需要在 sql 文件中存在, 多个逗号分隔，默认全部生成 
	-h        哈希分表, 如: 'user:10' 表示user表按主键哈希分成10个表，若主键为数值类型则取模，若为字符串类型则crc32后再取模，多个表用逗号分隔
	-a        同时生成 controller,svc 文件, 默认不生成
	-g        controller 分组路径, 默认无`)
} // }}}

func main() { // {{{

	modelDir := "./model"
	daoDir := "./dao"
	apiDir := "./controller/api"
	rpcDir := "./controller/rpc"
	svcDir := "./svc"

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前工作目录失败:", err)
		return
	}

	flag.StringVar(&appname, "n", path.Base(currentDir), "application name")
	flag.StringVar(&sqlFile, "s", "", "sql 文件名")
	flag.StringVar(&confName, "c", "", "配置文件名, 默认: db_master,db_slave")
	flag.BoolVar(&force, "f", false, "强制生成, 如果已存在则覆盖")
	flag.StringVar(&genMode, "m", "", " 指定生成[api | rpc | svc | dao | model]文件, 多个逗号分隔，默认全部生成")
	flag.StringVar(&group, "g", "", "controller 分组名, 默认无")
	flag.StringVar(&tableNames, "t", "", "表名, 默认全部")
	flag.StringVar(&hashNum, "h", "", "哈希分表配置, 默认无")
	flag.Parse()

	if appname == "" || sqlFile == "" {
		showHelp()
		return
	}

	if confName != "" {
		confName = strings.ReplaceAll(strings.ReplaceAll(strings.Trim(confName, ""), " ", ""), ",", `", "`)
	}

	tableNamesArr := []string{}
	if tableNames != "" {
		tableNamesArr = strings.Split(tableNames, ",")
	}

	genModeArr := []string{}
	if genMode != "" {
		genModeArr = strings.Split(genMode, ",")
	}

	hashTable = map[string]int{}
	if hashNum != "" {
		hashs := strings.Split(hashNum, ",")
		for _, hash := range hashs {
			hts := strings.Split(hash, ":")
			if len(hts) < 2 {
				fmt.Println("\033[31m哈希分表参数不正确!\033[0m\n")
				showHelp()
				return
			}

			table := strings.TrimSpace(hts[0])
			if table == "" {
				fmt.Println("\033[31m哈希分表参数不正确!\033[0m\n")
				showHelp()
				return
			}

			num := strings.TrimSpace(hts[1])
			numint, err := strconv.Atoi(num)
			if err != nil {
				fmt.Println("\033[31m哈希分表参数不正确!\033[0m\n")
				showHelp()
				return
			}

			hashTable[table] = numint
		}
	}

	// 读取 SQL 文件
	sqlContent, err := readFile(sqlFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// 解析 SQL 文件获取表结构
	tables := parseSQL(sqlContent)

	total := len(tables)
	for _, table := range tables {
		gen := true
		if len(tableNamesArr) > 0 {
			gen = false
			for _, tname := range tableNamesArr {
				if strings.ToLower(tname) == toSnakeCase(table.Name) {
					gen = true
				}
			}
		}

		if !gen {
			total--
			continue
		}

		// 生成 model 代码
		if genMode == "" || contains(genModeArr, "model") || contains(genModeArr, "m") {
			modelCode := genModelCode(table)

			modelFile := filepath.Join(modelDir, table.Name+"Model.go")
			// 写入输出文件
			err = writeFile(modelFile, modelCode)
			if err != nil {
				fmt.Printf("Error writing file: %v\n", err)
				return
			}
		}

		// 生成 dao 代码
		if genMode == "" || contains(genModeArr, "dao") || contains(genModeArr, "d") {
			daoCode := genDaoCode(table)

			daoFile := filepath.Join(daoDir, table.Name+"Dao.go")
			// 写入输出文件
			err = writeFile(daoFile, daoCode)
			if err != nil {
				fmt.Printf("Error writing file: %v\n", err)
				return
			}
		}

		if genMode == "" || contains(genModeArr, "api") || contains(genModeArr, "a") {
			hasBase, _ := IsFile(filepath.Join(apiDir, "BaseController.go"))
			// 生成 controller 代码
			controllerCode := genControllerCode(table, true, hasBase)

			controllerFile := filepath.Join(apiDir, group, table.Name+"Controller.go")
			// 写入输出文件
			err = writeFile(controllerFile, controllerCode)
			if err != nil {
				fmt.Printf("Error writing file: %v\n", err)
				return
			}
		}

		if genMode == "" || contains(genModeArr, "rpc") || contains(genModeArr, "r") {
			hasBase, _ := IsFile(filepath.Join(rpcDir, "BaseController.go"))
			// 生成 controller 代码
			controllerCode := genControllerCode(table, false, hasBase)

			controllerFile := filepath.Join(rpcDir, group, table.Name+"Controller.go")
			// 写入输出文件
			err = writeFile(controllerFile, controllerCode)
			if err != nil {
				fmt.Printf("Error writing file: %v\n", err)
				return
			}
		}

		if genMode == "" || contains(genModeArr, "svc") || contains(genModeArr, "s") {
			// 生成 svc 代码
			svcCode := genSvcCode(table)

			svcFile := filepath.Join(svcDir, group, table.Name+"Svc.go")
			// 写入输出文件
			err = writeFile(svcFile, svcCode)
			if err != nil {
				fmt.Printf("Error writing file: %v\n", err)
				return
			}
		}
	}

	fmt.Printf("扫描到 %d 个表\n", total)

} // }}}

func readFile(filename string) (string, error) { // {{{
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return strings.Join(lines, "\n"), scanner.Err()
} // }}}

func writeFile(filename, content string) error { // {{{
	isfile, err := IsFile(filename)
	if err != nil {
		return err
	}

	if isfile {
		fmt.Printf("\033[33m文件 [%s] 已存在!\033[0m\n", filename)

		if !force {
			return nil
		}

		bkfile := filename + ".bak." + time.Now().Format("2006-01-02-150405")
		err = os.Rename(filename, bkfile)
		if err != nil {
			return err
		}

		fmt.Printf("\033[32m文件 [%s] 已备份为 [%s]!\033[0m\n", filename, bkfile)
	}

	if err := os.MkdirAll(path.Dir(filename), 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	fmt.Printf("\033[32m成功生成文件 [%s]!\033[0m\n", filename)

	return nil
} // }}}

func IsFile(path string) (bool, error) { // {{{
	m, err := os.Stat(path)
	if err == nil {
		return m.Mode().IsRegular(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
} // }}}

func parseSQL(sql string) []Table { // {{{
	var tables []Table

	// 移除注释和多余空格
	sql = removeSQLComments(sql)
	sql = strings.ReplaceAll(sql, "\r\n", "\n")
	sql = strings.ReplaceAll(sql, "\r", "\n")

	// 分割 SQL 语句
	statements := splitSQLStatements(sql)

	for _, stmt := range statements {
		//	if !isCreateTableStatement(stmt) {
		//		continue
		//	}

		table := parseCreateTableStatement(stmt)
		if table != nil {
			tables = append(tables, *table)
		}
	}

	return tables
} // }}}

func removeSQLComments(sql string) string { // {{{
	// 移除单行注释
	sql = regexp.MustCompile(`--.*`).ReplaceAllString(sql, "")
	// 移除多行注释
	sql = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(sql, "")
	// 移除多余空格和换行
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")
	return sql
} // }}}

func splitSQLStatements(sql string) []string {
	// 简单按分号分割，不考虑分号在字符串中的情况
	return strings.Split(sql, ";")
}

func isCreateTableStatement(stmt string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(stmt)), "create table")
}

func parseCreateTableStatement(stmt string) *Table { // {{{
	stmt = strings.TrimSpace(stmt)
	if len(stmt) == 0 {
		return nil
	}

	// 提取表名
	tableName, tableComment, body := extractTableInfo(stmt)
	if tableName == "" {
		return nil
	}
	tableName = strings.ToLower(tableName)

	// 提取字段和主键
	fields, primaryKeys := parseTableBody(body)

	hashNum := 1
	if num, ok := hashTable[tableName]; ok {
		hashNum = num
	}

	// 构建表结构
	table := &Table{
		Name:    toPascalCase(tableName),
		Comment: tableComment,
		HashNum: hashNum,
	}

	for _, field := range fields {
		// 检查是否是主键
		isPrimary := false
		for _, pk := range primaryKeys {
			if pk == field.Name {
				isPrimary = true
				table.PrimaryKey = pk
				break
			}
		}

		// 处理Nullable类型
		fieldType := field.Type
		if strings.HasPrefix(strings.ToLower(fieldType), "nullable(") {
			fieldType = strings.TrimSuffix(strings.TrimPrefix(fieldType, "Nullable("), ")")
			field.IsNullable = true
		}

		goType, ok := typeMappings[strings.ToLower(fieldType)]
		if !ok {
			goType = "interface{}"
		}

		if _, ok := intTypes[goType]; ok {
			table.PrimaryInt = true
		}

		ormType, ok := ormTypeMappings[strings.ToLower(fieldType)]
		if !ok {
			ormType = fieldType
		}

		table.Fields = append(table.Fields, Field{
			Name:       toPascalCase(field.Name),
			Type:       goType,
			OrmType:    ormType,
			IsPrimary:  isPrimary,
			IsNullable: field.IsNullable,
			Comment:    field.Comment,
			Default:    field.Default,
			Extra:      field.Extra,
		})
	}

	if table.PrimaryKey == "" {
		table.PrimaryKey = toSnakeCase(table.Fields[0].Name)
	}

	return table
} // }}}

func extractTableInfo(stmt string) (name, comment, body string) { // {{{
	// 先提取表名
	//nameRe := regexp.MustCompile(`(?i)create\s+table\s+(?:if\s+not\s+exists\s+)?([^\s(]+)`)
	//nameRe := regexp.MustCompile(`(?i)create\s+table\s+(?:if\s+not\s+exists\s+)?([\w.]+)(?:\s+on\s+cluster\s+\w+)?`)
	nameRe := regexp.MustCompile(`(?i)create\s+(?:table|dictionary)\s+(?:if\s+not\s+exists\s+)?(?:[\w.]+\.)?` + "(?:[`])?" + `(\w+)` + "(?:[`])?" + `(?:\s+on\s+cluster\s+\w+)?`)
	nameMatch := nameRe.FindStringSubmatch(stmt)

	if len(nameMatch) < 2 {
		return "", "", ""
	}
	name = strings.Trim(nameMatch[1], "`'\" ")

	// 提取注释
	commentRe := regexp.MustCompile(`(?i)comment\s*=\s*['"](.*?)['"]`)
	commentMatch := commentRe.FindStringSubmatch(stmt)
	if len(commentMatch) > 1 {
		comment = strings.Trim(commentMatch[1], "'\"")
	}

	// 提取表定义主体 - 找到第一个左括号和匹配的右括号
	start := strings.Index(stmt, "(")
	if start == -1 {
		return name, comment, ""
	}

	// 从第一个左括号开始找匹配的右括号
	parenCount := 1
	end := start + 1
	for end < len(stmt) && parenCount > 0 {
		switch stmt[end] {
		case '(':
			parenCount++
		case ')':
			parenCount--
		}
		end++
	}

	if parenCount != 0 {
		return name, comment, ""
	}

	body = stmt[start+1 : end-1]
	return name, comment, body
} // }}}

func parseTableBody(body string) ([]Field, []string) { // {{{
	var fields []Field
	var primaryKeys []string

	// 分割字段定义
	fieldDefs := splitFieldDefinitions(body)

	for _, def := range fieldDefs {
		def = strings.TrimSpace(def)
		if len(def) == 0 {
			continue
		}

		// 检查是否是主键定义
		if strings.HasPrefix(strings.ToUpper(def), "PRIMARY KEY") {
			pks := extractPrimaryKeys(def)
			primaryKeys = append(primaryKeys, pks...)
			continue
		}

		if strings.HasPrefix(strings.ToUpper(def), "UNIQUE") || strings.HasPrefix(strings.ToUpper(def), "KEY") || strings.HasPrefix(strings.ToUpper(def), "INDEX") {
			continue
		}

		// 解析字段定义
		field := parseFieldDefinition(def)
		if field != nil {
			fields = append(fields, *field)
		}
	}

	return fields, primaryKeys
} // }}}

func splitFieldDefinitions(body string) []string { // {{{
	var definitions []string
	var current strings.Builder
	parenLevel := 0

	commentRe := regexp.MustCompile(`(?i)comment\s+'(.*?)'`)
	commentMatch := commentRe.FindAllStringSubmatch(body, -1)
	for _, match := range commentMatch {

		comment := match[1]
		body = strings.ReplaceAll(body, comment, base64.StdEncoding.EncodeToString([]byte(comment)))
	}

	for _, r := range body {
		switch r {
		case '(':
			parenLevel++
			current.WriteRune(r)
		case ')':
			parenLevel--
			current.WriteRune(r)
		case ',':
			if parenLevel == 0 {
				definitions = append(definitions, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		definitions = append(definitions, strings.TrimSpace(current.String()))
	}

	return definitions
} // }}}

func extractPrimaryKeys(def string) []string { // {{{
	re := regexp.MustCompile(`(?i)primary\s+key\s*\(([^)]+)\)`)
	matches := re.FindStringSubmatch(def)
	if len(matches) < 2 {
		return nil
	}

	keys := strings.Split(matches[1], ",")
	for i := range keys {
		keys[i] = strings.Trim(keys[i], "`'\" ")
	}

	return keys
} // }}}

func parseFieldDefinition(def string) *Field { // {{{
	// 先提取注释部分，防止注释中的逗号干扰解析
	comment := ""
	commentRe := regexp.MustCompile(`(?i)comment\s+'(.*?)'`)
	commentMatch := commentRe.FindStringSubmatch(def)
	if len(commentMatch) > 1 {
		comment = commentMatch[1]
		// 临时移除注释部分进行解析
		def = strings.Replace(def, commentMatch[0], "", 1)
	}

	// 改进的字段解析正则表达式，不依赖注释部分
	re := regexp.MustCompile(`^\s*([^\s,]+)\s+([a-zA-Z0-9]+)(?:\()?(?:\s+unsigned)?(?:\s+not\s+null)?(?:\s+default\s+(?:null|'[^']*'|\d+|\([^)]*\)))?(?:\s+(auto_increment|on\s+update\s+[^\s]+))?`)
	matches := re.FindStringSubmatch(strings.ToLower(def))
	if len(matches) < 3 {
		return nil
	}

	fieldName := strings.Trim(matches[1], "`'\" ")
	fieldType := strings.ToLower(matches[2])
	isNullable := !strings.Contains(strings.ToLower(def), "not null")
	defaultValue := ""
	if len(matches) > 3 && matches[3] != "" {
		defaultValue = strings.TrimSpace(strings.TrimPrefix(matches[3], "default "))
		defaultValue = strings.Trim(defaultValue, "'\"")
	}
	extra := ""
	if len(matches) > 4 && matches[4] != "" {
		extra = strings.TrimSpace(matches[4])
	}

	comment_d, err := base64.StdEncoding.DecodeString(comment)
	if err != nil {
		panic(err)
	}

	return &Field{
		Name:       fieldName,
		Type:       fieldType,
		IsNullable: isNullable,
		Comment:    string(comment_d),
		Default:    defaultValue,
		Extra:      extra,
	}
} // }}}

func genModelCode(table Table) string { // {{{
	const ormTemplate = `package model

//此文件是由 nyx 脚手架自动生成, 可按需要修改

import (
	"context"
	"github.com/nyxless/nyx/model"
	"github.com/nyxless/nyx/x"
    {{if .HasTime}}"time"{{end}}
)

{{if ne .Comment ""}}// {{.Comment}}{{end}}
type {{.Name}}Model struct {
	model.Model

    {{- range .Fields}}
    {{.Name}} {{.Type}} ` + "`" + `json:"{{.Name | toSnakeCase}}" orm:"column:{{.Name | toSnakeCase}}{{if ne .OrmType ""}};type:{{.OrmType}}{{end}}{{if .IsPrimary}};primaryKey{{end}}{{if not .IsNullable}};not null{{end}}{{if ne .Default ""}};default:{{.Default}}{{end}}{{if ne .Extra ""}};{{.Extra}}{{end}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
    {{- end}}
}

// 创建空的 {{.Name}}Model
func {{.Name}}() *{{.Name}}Model {
	return &{{.Name}}Model{}
}

// 从 map 初始化 {{.Name}}Model
func New{{.Name}}Model(m x.MAP) *{{.Name}}Model {
	c := {{.Name}}()
	c.Fill(m)

	return c
}

func (this *{{.Name}}Model) WithContext(ctx context.Context) *{{.Name}}Model {
	this.Model.WithContext(ctx)
	return this
}

func (this *{{.Name}}Model) Fill(m x.MAP) {
{{- range .Fields}}
	if {{.Name | toSnakeCase}}, ok := m["{{.Name | toSnakeCase}}"]; ok {
		this.{{.Name}} = {{convertFunc .Type (.Name | toSnakeCase)}}
	}
    {{- end}}
}

func (this *{{.Name}}Model) ToMap() x.MAP {
	return x.MAP{
{{- range .Fields}}
		"{{.Name | toSnakeCase}}":   this.{{.Name}},
    {{- end}}
	}
}

`

	funcMap := template.FuncMap{
		"toSnakeCase":  toSnakeCase,
		"toLowerCamel": toLowerCamel,
		"convertFunc":  convertFunc,
	}

	tmpl, err := template.New("orm").Funcs(funcMap).Parse(ormTemplate)
	if err != nil {
		panic(err)
	}

	if hasType(table, "time.Time") {
		table.HasTime = true
	}

	//table.Appname = appname

	var buf strings.Builder
	err = tmpl.Execute(&buf, struct {
		Table
		Appname string
	}{
		Table:   table,
		Appname: appname,
	})

	if err != nil {
		panic(err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		panic(err)
	}

	return string(formatted)
} // }}}

func genDaoCode(table Table) string { // {{{
	const ormTemplate = `package dao

//此文件是由 nyx 脚手架自动生成, 可按需要修改

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/dao"
	"github.com/nyxless/nyx/x"
	"{{.Appname}}/model"
)

func New{{.Name}}Dao({{if gt .HashNum 1}}id any,{{end}}tx ...*x.SqlClient) *{{.Name}}Dao {
	ins := &{{.Name}}Dao{}
	ins.Init({{if gt .HashNum 1}}id, {{end}}tx...)
	return ins
}

type {{.Name}}Dao struct {
	dao.Dao
}
{{if gt .HashNum 1}}//按主键(PrimaryKey)分表, id 为主键值{{end}}
func (this *{{.Name}}Dao) Init({{if gt .HashNum 1}}id any, {{end}}tx ...*x.SqlClient) {
	if len(tx) > 0 {
		this.Dao.InitTx(tx[0])
	} else {
		this.Dao.Init({{if ne .ConfName ""}}"{{.ConfName}}"{{end}})
	}
	this.SetTable("{{.Name | toSnakeCase}}{{if gt .HashNum 1}}_" + this.getHashNum(id){{else}}"{{end}})
	this.SetPrimary("{{.PrimaryKey}}")
}

{{if gt .HashNum 1}}
func (this *{{.Name}}Dao) getHashNum(id any) string {
	return x.ToString({{if .PrimaryInt}}x.AsInt(id){{else}}x.Crc32(x.AsString(id)){{end}} % {{.HashNum}})
}
{{end}}

/** 继承 Dao 链式调用方法 **/
func (this *{{.Name}}Dao) WithContext(ctx context.Context) *{{.Name}}Dao {
	this.Dao.WithContext(ctx)
	return this
}

func (this *{{.Name}}Dao) SetCountField(field string) *{{.Name}}Dao {
	this.Dao.SetCountField(field)
	return this
}

func (this *{{.Name}}Dao) SetDefaultFields(fields string) *{{.Name}}Dao {
	this.Dao.SetDefaultFields(fields)
	return this
}

func (this *{{.Name}}Dao) SetFields(fields ...string) *{{.Name}}Dao {
	this.Dao.SetFields(fields...)
	return this
}

func (this *{{.Name}}Dao) UseIndex(idx string) *{{.Name}}Dao {
	this.Dao.UseIndex(idx)
	return this
}

func (this *{{.Name}}Dao) UseMaster(flag ...bool) *{{.Name}}Dao {
	this.Dao.UseMaster(flag...)
	return this
}

func (this *{{.Name}}Dao) SetAutoOrder(flag ...bool) *{{.Name}}Dao {
	this.Dao.SetAutoOrder(flag...)
	return this
}

func (this *{{.Name}}Dao) Order(order ...string) *{{.Name}}Dao {
	this.Dao.Order(order...)
	return this
}

func (this *{{.Name}}Dao) Group(group ...string) *{{.Name}}Dao {
	this.Dao.Group(group...)
	return this
}

func (this *{{.Name}}Dao) Limit(limit int, limits ...int) *{{.Name}}Dao {
	this.Dao.Limit(limit, limits...)
	return this
}

func (this *{{.Name}}Dao) SetFilter(params ...any) *{{.Name}}Dao {
	this.Dao.SetFilter(params...)
	return this
}

func (this *{{.Name}}Dao) Alias(a string) *{{.Name}}Dao {
	this.Dao.Alias(a)
	return this
}

func (this *{{.Name}}Dao) LeftJoin(left_join ...*dao.JoinOn) *{{.Name}}Dao {
	this.Dao.LeftJoin(left_join...)
	return this
}

func (this *{{.Name}}Dao) InnerJoin(inner_join ...*dao.JoinOn) *{{.Name}}Dao {
	this.Dao.InnerJoin(inner_join...)
	return this
}


/**end**/

// 新增数据
func (this *{{.Name}}Dao) Add{{.Name}}(t *model.{{.Name}}Model) (int, error) {
	return this.AddRecord(x.AsMap(t))
}

// 更新数据
func (this *{{.Name}}Dao) Set{{.Name}}(t *model.{{.Name}}Model) (int, error ){
	return this.SetRecord(x.AsMap(t), t.{{.PrimaryKey | toPascalCase}})
}

// 先尝试新增, 数据重复时更新
func (this *{{.Name}}Dao) Reset{{.Name}}(t *model.{{.Name}}Model) (int, error) {
	return this.ResetRecord(x.AsMap(t))
}

// 删除数据
func (this *{{.Name}}Dao) Del{{.Name}}(id any) (int, error) {
	return this.DelRecord(id)
}

// 获取数据信息
func (this *{{.Name}}Dao) Get{{.Name}}(id any) (*model.{{.Name}}Model, error) {
	data, err := this.GetRecord(id)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("数据不存在: %v", id)
	}

	return model.New{{.Name}}Model(data), nil
}

// 返回数据列表
func (this *{{.Name}}Dao) Get{{.Name}}s(params ...any) ([]*model.{{.Name}}Model, error) {
	data, err := this.GetRecords(params...)
	if err != nil {
		return nil, err
	}

	var list []*model.{{.Name}}Model
	for _, row := range data {
		list = append(list, model.New{{.Name}}Model(row))
	}

	return list, nil
}

// 返回带总数的列表
func (this *{{.Name}}Dao) Get{{.Name}}List(params ...any) (int, []*model.{{.Name}}Model, error) {
	total, data, err := this.GetList(params...)
	if err != nil {
		return 0, nil, err
	}

	var list []*model.{{.Name}}Model
	for _, row := range data {
		list = append(list, model.New{{.Name}}Model(row))
	}

	return total, list, nil
}
`

	funcMap := template.FuncMap{
		"toPascalCase": toPascalCase,
		"toSnakeCase":  toSnakeCase,
		"toLowerCamel": toLowerCamel,
		"convertFunc":  convertFunc,
	}

	tmpl, err := template.New("orm").Funcs(funcMap).Parse(ormTemplate)
	if err != nil {
		panic(err)
	}

	if hasType(table, "time.Time") {
		table.HasTime = true
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, struct {
		Table
		Appname  string
		ConfName string
	}{
		Table:    table,
		Appname:  appname,
		ConfName: confName,
	})

	if err != nil {
		panic(err)
	}

	//return buf.String()
	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		panic(err)
	}

	return string(formatted)
} // }}}

func genControllerCode(table Table, is_api, hasBase bool) string { // {{{
	const ormTemplate = `package {{.Pkg}} 

//此文件是由 nyx 脚手架自动生成, 可按需要修改

import ({{if not .HasBase}}
	"github.com/nyxless/nyx/controller"{{end}}
	"github.com/nyxless/nyx/x"
	"{{.Appname}}/svc"
)

func init() {
	x.Add{{if .IsApi}}Api{{else}}Rpc{{end}}(&{{.Name}}Controller{}{{if ne .Group ""}}, "{{.Group}}"{{end}})
}

type {{.Name}}Controller struct {
	{{if .HasBase}}BaseController{{else}}controller.Controller{{end}}
}

func (this *{{.Name}}Controller) Get{{.Name}}InfoAction() {
	// 参数接收{{if .PrimaryInt}}
	{{.PrimaryKey}} := this.GetInt("{{.PrimaryKey}}")

	// 拦截器
	x.Interceptor({{.PrimaryKey}} > 0, x.ERR_PARAMS, "{{.PrimaryKey}}")
	{{else}}
	{{.PrimaryKey}}:= this.GetString("{{.PrimaryKey}}")

	// 拦截器
	x.Interceptor({{.PrimaryKey}} != "", x.ERR_PARAMS, "{{.PrimaryKey}}")
	{{end}}

	// 业务逻辑
	{{.Name | toSnakeCase}}_svc := svc.New{{.Name}}Svc()
	{{.Name | toSnakeCase}}_info, err := {{.Name | toSnakeCase}}_svc.Get{{.Name}}({{.PrimaryKey}})

	x.Interceptor(err == nil, x.ERR_OTHER, err)

	this.Render({{.Name | toSnakeCase}}_info)
}

func (this *{{.Name}}Controller) Get{{.Name}}ListAction() {
	// 参数接收
	page := this.GetInt("page", 1)
	num := this.GetInt("num", 50)

	// 拦截器
	x.Interceptor(page > 0, x.ERR_PARAMS, "page")
	x.Interceptor(num > 0, x.ERR_PARAMS, "num")

	// 业务逻辑
	{{.Name | toSnakeCase}}_svc := svc.New{{.Name}}Svc()
	total, {{.Name | toSnakeCase}}_list, _ := {{.Name | toSnakeCase}}_svc.Get{{.Name}}List(page, num)

	ret_data := x.MAP{
		"total":    total,
		"{{.Name | toSnakeCase}}_list": {{.Name | toSnakeCase}}_list,
	}

	this.Render(ret_data)
}

func (this *{{.Name}}Controller) Add{{.Name}}Action() {
	// 检查权限
	// this.CheckAuth()

	// 参数接收
	//{{.Name | toSnakeCase}}_id := this.GetInt("{{.Name | toSnakeCase}}_id")

	// 拦截器
	//x.Interceptor({{.Name | toSnakeCase}}_id > 0, x.ERR_PARAMS, "{{.Name | toSnakeCase}}_id")

	// 业务逻辑
	//{{.Name | toSnakeCase}}_svc := svc.New{{.Name}}Svc()
	//err := {{.Name | toSnakeCase}}_svc.Add{{.Name}}()
	//x.Interceptor(err == nil, x.ERR_OTHER, "新增数据出错")

	this.Render()

}

func (this *{{.Name}}Controller) Set{{.Name}}Action() {
	// 检查权限
	//this.CheckAuth()

	// 参数接收
	//{{.Name | toSnakeCase}}_id := this.GetInt("{{.Name | toSnakeCase}}_id")

	// 拦截器
	//x.Interceptor({{.Name | toSnakeCase}}_id > 0, x.ERR_PARAMS, "{{.Name | toSnakeCase}}_id")

	// 业务逻辑
	//{{.Name | toSnakeCase}}_svc := svc.New{{.Name}}Svc()
	//err := {{.Name | toSnakeCase}}_svc.Set{{.Name}}()
	//x.Interceptor(err == nil, x.ERR_OTHER, "更新数据出错")

	this.Render()

}

func (this *{{.Name}}Controller) Del{{.Name}}Action() {
	// 检查权限
	//this.CheckAuth()

	// 参数接收
	//{{.Name | toSnakeCase}}_id := this.GetInt("{{.Name | toSnakeCase}}_id")

	// 拦截器
	//x.Interceptor({{.Name | toSnakeCase}}_id > 0, x.ERR_PARAMS, "{{.Name | toSnakeCase}}_id")

	// 业务逻辑
	//{{.Name | toSnakeCase}}_svc := svc.New{{.Name}}Svc()
	//err := {{.Name | toSnakeCase}}_svc.Del{{.Name}}({{.Name | toSnakeCase}}_id)
	//x.Interceptor(err == nil, x.ERR_OTHER, "删除数据出错")

	this.Render()
}
`

	funcMap := template.FuncMap{
		"toSnakeCase":  toSnakeCase,
		"toLowerCamel": toLowerCamel,
		"convertFunc":  convertFunc,
	}

	tmpl, err := template.New("orm").Funcs(funcMap).Parse(ormTemplate)
	if err != nil {
		panic(err)
	}

	if hasType(table, "time.Time") {
		table.HasTime = true
	}

	//table.Appname = appname

	var pkg string
	if is_api {
		pkg = "api"
	} else {
		pkg = "rpc"
	}

	if group != "" {
		pkg = path.Base(group)
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, struct {
		Table
		Appname  string
		ConfName string
		Group    string
		Pkg      string
		HasBase  bool
		IsApi    bool
	}{
		Table:    table,
		Appname:  appname,
		ConfName: confName,
		Group:    group,
		Pkg:      pkg,
		HasBase:  hasBase,
		IsApi:    is_api,
	})

	if err != nil {
		panic(err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		panic(err)
	}

	return string(formatted)
} // }}}

func genSvcCode(table Table) string { // {{{
	const ormTemplate = `package svc

//此文件是由 nyx 脚手架自动生成, 可按需要修改

import (
	"context"
	"github.com/nyxless/nyx/svc"
	//"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/dao/tx"
	"{{.Appname}}/dao"
	"{{.Appname}}/model"
)

{{if ne .Comment ""}}// {{.Comment}}{{end}}
type {{.Name}}Svc struct {
	svc.Svc
}

func New{{.Name}}Svc() *{{.Name}}Svc{
	return &{{.Name}}Svc{}
}

func (this *{{.Name}}Svc) WithContext(ctx context.Context) *{{.Name}}Svc {
	this.Svc.WithContext(ctx)
	return this
}

func (this *{{.Name}}Svc) Get{{.Name}}(id any)(*model.{{.Name}}Model, error) {
	return dao.New{{.Name}}Dao({{if gt .HashNum 1}}id{{end}}).Get{{.Name}}(id)
}

func (this *{{.Name}}Svc) Add{{.Name}}() error {
	var err error
	{{.Name | toSnakeCase}} := &model.{{.Name}}Model{
		//	...
	}
	_, err = dao.New{{.Name}}Dao().Add{{.Name}}({{.Name | toSnakeCase}})
	return err
}

// 使用事务
func (this *{{.Name}}Svc) Add{{.Name}}Tx() error {
	trans, err := tx.TransBegin("db_manta")
	if err != nil {
		return err
	}

	defer trans.Rollback()

	{{.Name | toSnakeCase}} := &model.{{.Name}}Model{
		//	...
	}
	_, err = dao.New{{.Name}}Dao(trans).Add{{.Name}}({{.Name | toSnakeCase}})
	if err != nil {
		return err
	}

	// other sql

	trans.Commit()

	return nil
}

func (this *{{.Name}}Svc) Set{{.Name}}() error {
	{{.Name | toSnakeCase}} := &model.{{.Name}}Model{
		//...
	}
	_, err := dao.New{{.Name}}Dao().Set{{.Name}}({{.Name | toSnakeCase}})
	return err
}

func (this *{{.Name}}Svc) Del{{.Name}}({{.Name | toSnakeCase}}_id any) error {
	_, err := dao.New{{.Name}}Dao().Del{{.Name}}({{.Name | toSnakeCase}}_id)
	return err
}

func (this *{{.Name}}Svc) Get{{.Name}}s(params ...any) ([]*model.{{.Name}}Model, error) {
	return dao.New{{.Name}}Dao().Get{{.Name}}s(params...)
}

func (this *{{.Name}}Svc) Get{{.Name}}List(page, num int, params ...any) (int, []*model.{{.Name}}Model, error) {
	return dao.New{{.Name}}Dao().Limit((page-1)*num, num).Get{{.Name}}List(params...)
}
`

	funcMap := template.FuncMap{
		"toSnakeCase":  toSnakeCase,
		"toLowerCamel": toLowerCamel,
		"convertFunc":  convertFunc,
	}

	tmpl, err := template.New("orm").Funcs(funcMap).Parse(ormTemplate)
	if err != nil {
		panic(err)
	}

	if hasType(table, "time.Time") {
		table.HasTime = true
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, struct {
		Table
		Appname string
	}{
		Table:   table,
		Appname: appname,
	})

	if err != nil {
		panic(err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		panic(err)
	}

	return string(formatted)
} // }}}

func hasType(table Table, typeName string) bool {
	for _, field := range table.Fields {
		if field.Type == typeName {
			return true
		}
	}

	return false
}

func toPascalCase(s string) string {
	s = strings.Trim(s, "`'\" ")
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.Title(strings.ToLower(parts[i]))
		}
	}
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

func toLowerCamel(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func convertFunc(t, key string) string { // {{{
	switch t {
	case "[]byte":
		return "x.AsByte(" + key + ")"
	case "bool":
		return "x.AsBool(" + key + ")"
	case "float32":
		return "x.AsFloat32(" + key + ")"
	case "float64":
		return "x.AsFloat64(" + key + ")"
	case "int":
		return "x.AsInt(" + key + ")"
	case "int8":
		return "x.AsInt8(" + key + ")"
	case "int16":
		return "x.AsInt16(" + key + ")"
	case "int32":
		return "x.AsInt32(" + key + ")"
	case "int64":
		return "x.AsInt64(" + key + ")"
	case "interface{}":
		return key
	case "any":
		return key
	case "string":
		return "x.AsString(" + key + ")"
	case "uint":
		return "x.AsUint(" + key + ")"
	case "uint32":
		return "x.AsUint32(" + key + ")"
	case "uint64":
		return "x.AsUint64(" + key + ")"
	case "time.Time":
		return key + ".(time.Time)"
	default:
		return key
	}

} // }}}

// 检查字符串是否在切片中
func contains(slice []string, item string) bool { // {{{
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
} // }}}
