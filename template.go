package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const initFunc = `
package main

import (
	"devops-gotemplate/handlers"
	"devops-gotemplate/log"
	"devops-gotemplate/middlewares"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
)

func init() {
	env := os.Getenv("envID")
	if env == "" {
		env = "dev"
	}

	// 生产环境关闭debug模式
	if env != "dev" {
		gin.SetMode(gin.ReleaseMode)
	}

	// viper是单实例的
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	// 生成zap logger对象
	project := viper.GetString("project.name")
	app := viper.GetString("app.name")
	logFile := viper.GetString("log.file")
	logLevel := viper.GetString("log.level")
	maxSize := viper.GetInt("log.maxsize")
	maxAge := viper.GetInt("log.max_age")
	maxBackup := viper.GetInt("log.max_backup")
	compress := viper.GetBool("log.compress")
    consoleEnable := viper.GetBool("log.console_enable")

	if logLevel == "" {
		logLevel = "info"
	}
	if maxSize == 0 {
		maxSize = 100
	}
	if maxAge == 0 {
		maxAge = 15
	}
	if maxBackup == 0 {
		maxBackup = 30
	}
	if !compress {
		compress = true
	}
	log.NewLogger(project, app, env, logFile, logLevel, maxSize, maxAge, maxBackup, compress, consoleEnable)

	// 监控配置文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		log.Info("配置文件已更新:", zap.String("event", in.Name))
	})
}

func main() {
	engine := gin.New()
	engine.Use(middlewares.Logger(), middlewares.Cors(), gin.Recovery(), middlewares.Authenticate())

	vms := engine.Group("/vms")
	{
		vms.GET("/", handlers.HomeHandler)

	}

	if err := engine.Run(fmt.Sprintf(":%s", viper.GetString("app.port"))); err != nil {
		panic(err)
	}
}
`

// exeCmd 用于执行命令
func exeCmd(dir string, command string, args ...string) error {
	cmd := exec.Command(command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if dir != "" {
		cmd.Dir = dir
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// copyFile 用于复制文件
// 代码文件不大，直接copy
func copyFile(dst, src string) error {
	srcFile, err := ioutil.ReadFile(src)
	if err != nil {
		return errors.New(fmt.Sprintf("源文件 %s 打开失败, err: %v+", src, err))
	}

	err = ioutil.WriteFile(dst, srcFile, 666)
	if err != nil {
		return errors.New(fmt.Sprintf("创建目标文件 %s 失败, err: %v+", dst, err))
	}
	return nil
}

// goCheck 用于检查golang是否已经安装
func goCheck() error {
	if err := exeCmd("", "go", "version"); err != nil {
		return err
	}
	return nil
}

// gitCheck 用于检查git是否已经安装
func gitCheck() error {
	if err := exeCmd("", "git", "version"); err != nil {
		return err
	}
	return nil
}

// replaceFile 用于替换文件内容
func replaceFile(filePath, oldStr, newStr string, n int) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("读取文件 %s 失败", filePath))
	}

	newContent := strings.Replace(string(content), oldStr, newStr, n)
	err = ioutil.WriteFile(filePath, []byte(newContent), 664)
	if err != nil {
		return errors.New(fmt.Sprintf("修改文件 %s 失败", filePath))
	}
	return nil
}

// createProject 创建项目工程目录
func createProject(projectDir string) error {
	_, err := os.Stat(projectDir)
	if err == nil || os.IsExist(err) {
		return errors.New("工程目录已经存在")
	}

	// 创建工程目录
	if err := os.MkdirAll(projectDir, 755); err != nil {
		return errors.New(fmt.Sprintf("创建工程目录 %s 失败, err: %v", projectDir, err))
	}

	// 执行go mod
	projectName := filepath.Base(projectDir)
	err = exeCmd(projectDir, "go", "mod", "init", projectName)
	if err != nil {
		return errors.New(fmt.Sprintf("执行go mod失败，err: %v", err))
	}

	// 创建功能子目录
	dirs := []string{"log", "middlewares", "handlers"}
	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		if err := os.Mkdir(path, 755); err != nil {
			fmt.Println(fmt.Sprintf("创建目录 %s 失败, err: %v", path, err))
		}
	}
	return nil
}

// initProject 用于初始化工程
func initProject(projectDir string, appName string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return errors.New(fmt.Sprintf("获取当前目录失败, err :%v", err))
	}

	// 偷个懒，clone devops-template到当前目录，用于创建log、middleware等，意味着必须有该仓库的权限才能使用
	// 正常做法应该是通过api接口，登录gitlab去获取文件
	err = exeCmd("", "git", "clone", "http://git.baozun.com/devops/devops-gotemplate.git")
	if err != nil {
		return errors.New(fmt.Sprintf("获取模板文件失败, err: %v", err))
	}

	// 复制logger.go文件
	logDstFile := filepath.Join(projectDir, "log/logger.go")
	logSrcFile := filepath.Join(pwd, "devops-gotemplate/log/logger.go")
	if err := copyFile(logDstFile, logSrcFile); err != nil {
		return errors.New(fmt.Sprintf("创建logger.go文件失败, err: %v", err))
	}

	// 复制middlewares.go文件
	midDstFile := filepath.Join(projectDir, "middlewares/middlewares.go")
	midSrcFile := filepath.Join(pwd, "devops-gotemplate/middlewares/middlewares.go")
	if err := copyFile(midDstFile, midSrcFile); err != nil {
		return errors.New(fmt.Sprintf("创建middlewares.go文件失败, err: %v", err))
	}

	// 复制handlers.go文件
	handlerDstFile := filepath.Join(projectDir, "handlers/handlers.go")
	handlerSrcFile := filepath.Join(pwd, "devops-gotemplate/handlers/handlers.go")
	if err := copyFile(handlerDstFile, handlerSrcFile); err != nil {
		return errors.New(fmt.Sprintf("创建handlers.go文件失败, err: %v", err))
	}

	// 复制conf.yaml文件
	confDstFile := filepath.Join(projectDir, "conf.yaml")
	confSrcFile := filepath.Join(pwd, "devops-gotemplate/conf.yaml")
	if err := copyFile(confDstFile, confSrcFile); err != nil {
		return errors.New(fmt.Sprintf("创建conf.yaml文件失败, err: %v", err))
	}

	// 获取项目名称
	projectName := filepath.Base(projectDir)

	// 替换middlewares.go中的路径
	midFilePath := filepath.Join(projectDir, "middlewares/middlewares.go")
	err = replaceFile(midFilePath, "devops-gotemplate", projectName, 1)
	if err != nil {
		return errors.New(fmt.Sprintf("初始化middlewares.go文件失败, 原因:%v", err))
	}

	// 替换handlers.go中的路径
	handlerFilePath := filepath.Join(projectDir, "handlers/handlers.go")
	err = replaceFile(handlerFilePath, "devops-gotemplate", projectName, 1)
	if err != nil {
		return errors.New(fmt.Sprintf("初始化handlers.go文件失败, 原因:%v", err))
	}

	// 替换conf.yaml中的内容
	confFilePath := filepath.Join(projectDir, "conf.yaml")
	err = replaceFile(confFilePath, "devops", projectName, 1)
	if err != nil {
		return errors.New(fmt.Sprintf("初始化conf.yaml文件失败, 原因:%v", err))
	}
	err = replaceFile(confFilePath, "gotemplate", appName, 1)
	if err != nil {
		return errors.New(fmt.Sprintf("初始化conf.yaml文件失败, 原因:%v", err))
	}

	// 创建main.go
	mainFile := filepath.Join(projectDir, fmt.Sprintf("%s.go", projectName))
	f, err := os.Create(mainFile)
	if err != nil {
		fmt.Println(fmt.Sprintf("创建项目入口文件 %s 失败", mainFile))
		return errors.New("初始化工程失败")
	}

	mainFunc := strings.Replace(initFunc, "devops-gotemplate", projectName, 3)
	_, err = f.WriteString(mainFunc)
	if err != nil {
		fmt.Println(fmt.Sprintf("写入项目入口文件 %s 失败", mainFile))
		return errors.New("初始化工程失败")
	}

	// 执行go mod tidy，下载依赖包
	goProxy := "export GOPROXY=https://goproxy.cn"
	if runtime.GOOS == "windows" {
		goProxy = "set GOPROXY=https://goproxy.cn"
	}

	proxyCmd := exec.Command(goProxy)
	proxyCmd.Dir = projectDir
	if err := proxyCmd.Run(); err != nil {
		fmt.Println()
		fmt.Printf("设置GOPROXY失败，依赖包下载可能失败，如遇失败，请手工设置，用法如下:\n当前操作系统为%s, 执行:%s\n\n", runtime.GOOS, goProxy)
	}

	// 下载过程显示到console
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = projectDir

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func main() {
	// 检查参数
	fmt.Println("进行参数检查...")
	if len(os.Args) < 4 {
		fmt.Println("参数错误")
		fmt.Println("用法：template PROJECT_PATH PROJECT_NAME APP_NAME")
		os.Exit(1)
	}

	// 检查golang是否安装
	fmt.Println("进行golang检查...")
	if err := goCheck(); err != nil {
		fmt.Println("golang未安装")
		os.Exit(1)
	}

	// 检查git是否安装
	fmt.Println("进行git检查...")
	if err := gitCheck(); err != nil {
		fmt.Println("git未安装")
		os.Exit(1)
	}

	projectPath := os.Args[1]
	projectName := os.Args[2]
	appName := os.Args[3]
	projectDir := filepath.Join(projectPath, projectName)

	// 创建工程目录
	fmt.Println("进行创建工程...")
	if err := createProject(projectDir); err != nil {
		fmt.Printf("创建golang工程失败. 原因:%v\n", err)
		os.Exit(1)
	}

	// 初始化工程
	fmt.Println("进行初始化工程...")
	if err := initProject(projectDir, appName); err != nil {
		fmt.Printf("初始化golang工程失败. 原因:%v\n", err)
		os.Exit(1)
	}

	// 删除目录
	if err := os.RemoveAll("devops-gotemplate"); err != nil {

	}
	fmt.Println("golang工程模板创建成功...")
}
