package main

import (
	"archive/zip"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed resources/example_app.zip
var exampleZip embed.FS

const Version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]
	switch command {
	case "create":
		if len(os.Args) < 3 {
			printError("请指定要创建的 APP 名称")
			return
		}
		appName := os.Args[2]
		createApp(appName)
	case "version":
		printVersion()
	case "help", "-h", "--help":
		printHelp()
	default:
		runNyxShell(command)
	}
}

// 创建 APP 目录结构并复制资源
func createApp(appName string) {
	// 检查目录是否已存在
	if _, err := os.Stat(appName); !os.IsNotExist(err) {
		printError("目录 %s 已存在", appName)
		return
	}

	// 确认创建
	fmt.Printf("即将创建新项目: %s\n", colorize(appName, "green"))
	fmt.Print("确定要创建吗? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		printInfo("已取消创建")
		return
	}

	// 创建目录
	printInfo("创建目录结构...")
	err := os.MkdirAll(appName, 0755)
	if err != nil {
		printError("创建目录失败: %v", err)
		return
	}

	// 解压并复制嵌入式资源
	printInfo("复制项目文件...")
	err = extractAndCopyFiles(appName)
	if err != nil {
		printError("复制资源失败: %v", err)
		os.RemoveAll(appName) // 清理失败的项目
		return
	}

	// 创建log目录
	logpath := filepath.Join(appName, "logs")
	err = os.MkdirAll(logpath, 0755)
	if err != nil {
		printError("创建logs目录失败: %v", err)
		return
	}

	// 创建workers目录
	workerspath := filepath.Join(appName, "workers")
	err = os.MkdirAll(workerspath, 0755)
	if err != nil {
		printError("创建workers目录失败: %v", err)
		return
	}

	// 设置 nyx 文件可执行权限
	nyxPath := filepath.Join(appName, "nyx")
	err = os.Chmod(nyxPath, 0755)
	if err != nil {
		printError("设置权限失败: %v", err)
		return
	}

	printSuccess("成功创建项目: %s", appName)
	printInfo("进入项目并执行初始化命令: \ncd %s\ngo mod init [module-path]\ngo mod tidy\n", appName)
	printNotice(" (提示: 如果出现有关 grpc 的 import 冲突问题， 请尝试使用 go get google.golang.org/grpc )")
	printInfo("启动项目: ./nyx run")
}

// 解压ZIP文件并复制到目标目录，处理模板替换，跳过ZIP中的顶层目录
func extractAndCopyFiles(targetDir string) error {
	// 读取嵌入的ZIP文件
	zipData, err := exampleZip.ReadFile("resources/example_app.zip")
	if err != nil {
		return fmt.Errorf("读取ZIP文件失败: %w", err)
	}

	// 创建一个临时文件来保存ZIP内容
	tmpFile, err := os.CreateTemp("", "nyx-example-*.zip")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 写入ZIP数据到临时文件
	if _, err := tmpFile.Write(zipData); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 打开ZIP文件
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("打开ZIP文件失败: %w", err)
	}
	defer zipReader.Close()

	replacement := map[string]any{
		"AppName": filepath.Base(targetDir),
	}

	// 遍历ZIP文件中的每个文件/目录
	for _, file := range zipReader.File {
		// 跳过ZIP中的顶层目录
		relPath := strings.SplitN(file.Name, "/", 2)
		if len(relPath) < 2 {
			continue // 跳过顶层目录本身
		}

		// 处理目标路径（去掉顶层目录）
		targetPath := filepath.Join(targetDir, relPath[1])

		// 如果是目录，创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		// 打开ZIP中的文件
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开ZIP内文件失败: %w", err)
		}

		// 确保目标目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			srcFile.Close()
			return fmt.Errorf("创建目标目录失败: %w", err)
		}

		// 读取文件内容
		content, err := io.ReadAll(srcFile)
		srcFile.Close()
		if err != nil {
			return fmt.Errorf("读取ZIP内文件失败: %w", err)
		}

		// 处理模板替换
		processedContent, err := processTemplate(string(content), replacement)
		if err != nil {
			return fmt.Errorf("处理模板失败: %w", err)
		}

		// 创建目标文件
		dstFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("创建目标文件失败: %w", err)
		}

		// 写入处理后的内容
		if _, err := dstFile.Write([]byte(processedContent)); err != nil {
			dstFile.Close()
			return fmt.Errorf("写入目标文件失败: %w", err)
		}
		dstFile.Close()

		fmt.Println("创建文件:", targetPath)
	}

	return nil
}

// processTemplate 处理模板替换
func processTemplate(input string, data interface{}) (string, error) {
	tmpl, err := template.New("").Delims("#[", "]#").Parse(input)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// 检查并执行 nyx shell 文件
func runNyxShell(command string) {
	nyxCmd := "nyx"
	if _, err := os.Stat(nyxCmd); os.IsNotExist(err) {
		nyxCmd = "tools/nyx"
		if _, err = os.Stat(nyxCmd); os.IsNotExist(err) {
			printError("没有找到文件 nyx 或 tools/nyx, 请确认当前是否在项目根目录")
			printNotice(" (若要创建项目，请执行 nyx create AppName)")
			return
		}
	}

	cmd := exec.Command("sh", nyxCmd, command)
	cmd.Args = append(cmd.Args, os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		printError("执行命令失败: %v", err)
		printNotice(" (支持的命令: create | run | build | init | gen | server | help, 详情使用 help 查看)")
	}
}

// 打印帮助信息
func printHelp() {

	help := colorize("Nyx 脚手架", "cyan+bold") + "\n"
	help += colorize("版本: "+Version, "yellow") + "\n"
	help += `
Usage: nyx <command> [options]

命令列表:
  create <appname>  创建新项目

项目命令 (需要在项目目录中执行):
  run       运行应用程序
    -c          配置文件路径
    -t          构建标签(逗号分隔)
    -d          调试模式
    -m          运行模式(http,rpc,cli,tcp,ws 可组合, 使用逗号分隔)
    -p          cli模式时，访问URI路径,  格式同 http uri, 如: user/getUserInfo
    -q          cli模式时，参数列表,  格式同 http query, 如: uid=1&username=test
                
  build         编译应用程序
    -t          构建标签
    -o          输出路径
    --cross     跨平台编译(如 linux/amd64)
    --debug     保留调试信息
    --ldflags   自定义链接器标志
                (默认: ${DEFAULT_LDFLAGS})

  init      初始化应用程序, 生成路由配置代码(也可以手动在代码里配置)

  gen       生成示例代码 
    orm       从 sql 文件生成，具有 orm 功能的 model/dao/svc/controller 文件 
      -f        强制生成，如果已存在则覆盖
      -c        在配置文件中的配置名, 默认: db_master,db_slave 
      -s        sql文件名 
      -t        指定表名, 需要在 sql 文件中存在, 多个用逗号分隔, 如果指定了-p(表名前缀), 此处为去除前缀的部分，默认全部生成 
      -h        哈希分表, 格式 [{table}:{num}] 如: 'user:10' 表示user表按主键哈希分成10个表，若主键为数值类型则取模，若为字符串类型则crc32后再取模，多个表用逗号分隔, 如果指定了-p(表名前缀), 此处为去除前缀的部分
      -m        指定生成[api | rpc | svc | dao | model]文件(api、rpc 指controller文件), 多个用逗号分隔，默认全部生成
      -g        controller 分组路径, 默认无
      -p        表名前缀路径, 默认无
    dao       生成dao文件 
      -f        强制生成，如果已存在则覆盖
      -c        配置文件名, 默认: db_master,db_slave 
      -t        表名 
      -p        主键名，默认: id
      -n        哈希分表数值, 默认: 1"
    svc       生成svc文件
      -f        强制生成，如果已存在则覆盖
      -m        模块名
    rpc       生成rpc svc 文件 
      -f        强制生成，如果已存在则覆盖
      -m        模块名 
      -r        rpc方法名，多个逗号分隔, 使用basename作为函数名
      -p        rpc方法名前缀，多个相同路径的方法，可统一指定前缀

   
  server    服务管理
    start     启动服务
      -i        应用程序文件, 默认 bin/${APP_NAME}
      -c        配置文件
      -t        构建标签
      -d        调试模式
      -m        运行模式
      -p        cli模式时，访问URI路径,  格式同 http uri, 如: user/getUserInfo
      -q        cli模式时，参数列表,  格式同 http query, 如: uid=1&username=test
      -b        后台运行
      -o        运行日志
    stop      停止服务
    restart   重启服务
    reload    平滑重启 

示例:
  nyx create myapp    # 创建新项目
  cd myapp            # 进入项目目录
  nyx run           # 运行项目
`
	cmd := exec.Command("less", "-R")

	// 获取标准输入
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, help)
	}()

	// 连接标准输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 运行命令
	if err := cmd.Run(); err != nil {
		fmt.Println("Error:", err)
	}

}

func printVersion() {
	fmt.Printf("Nyx 脚手架版本: %s\n", colorize(Version, "green"))
}

// 颜色输出工具函数
func colorize(text string, color string) string {
	colorCodes := map[string]string{
		"reset":     "\033[0m",
		"red":       "\033[31m",
		"green":     "\033[32m",
		"yellow":    "\033[33m",
		"blue":      "\033[34m",
		"magenta":   "\033[35m",
		"cyan":      "\033[36m",
		"white":     "\033[37m",
		"bold":      "\033[1m",
		"underline": "\033[4m",
	}

	// 处理组合颜色如 "green+bold"
	parts := strings.Split(color, "+")
	var codes []string
	for _, part := range parts {
		if code, ok := colorCodes[part]; ok {
			codes = append(codes, code)
		}
	}

	if len(codes) == 0 {
		return text
	}

	return strings.Join(codes, "") + text + colorCodes["reset"]
}

func printError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(colorize("错误: "+msg, "red"))
}

func printNotice(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(colorize(" "+msg, "yellow"))
}

func printSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(colorize("✓ "+msg, "green"))
}

func printInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(colorize("> "+msg, "cyan"))
}
