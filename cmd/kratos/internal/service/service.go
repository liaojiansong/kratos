package service

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/emicklei/proto"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CmdServer the service command.
var CmdService = &cobra.Command{
	Use:   "service",
	Short: "生成服务端代码",
	Long:  "切换到指定服务根目录,执行当前命令",
	Run:   run,
}

type Env struct {
	protoPath         string
	targetDir         string
	workDir           string
	appDirName        string
	methPrefix        string
	appStructName     string
	firsServiceGoFile string
}

func loadProtoPath(workDir, appDirName string) (string, error) {
	dir := path.Join(workDir, fmt.Sprintf("../../api/%s/v1", appDirName))
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, file := range fileInfos {
		hasSuffix := strings.HasSuffix(file.Name(), ".proto")
		if hasSuffix {
			return path.Join(dir, file.Name()), nil
		}
	}
	return "", nil
}

func loadTargetDir(workDir string) (string, error) {
	dir := path.Join(workDir, "internal/service")
	_, err := os.Stat(dir)
	if err != nil {
		panic(err)
	}
	return dir, nil
}

func newEnv(workdir string) (*Env, error) {
	e := &Env{
		workDir:    workdir,
		appDirName: path.Base(workdir),
	}

	targetDir, err := loadTargetDir(workdir)
	if err != nil {
		return nil, err
	}
	e.targetDir = targetDir

	protoPath, err := loadProtoPath(workdir, e.appDirName)
	if err != nil {
		return nil, err
	}
	e.protoPath = protoPath

	PinkLog("work dir:%s", workdir)
	PinkLog("proto file:%s", protoPath)
	PinkLog("target dir:%s", targetDir)

	err = e.pickMethPrefixAndGoFile()
	if err != nil {
		return nil, err
	}

	return e, nil

}

func NewEnv() *Env {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	env, err := newEnv(dir)
	if err != nil {
		panic(err)
	}
	return env
}

func (e *Env) parserProto() (*proto.Proto, error) {
	reader, err := os.Open(e.protoPath)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	parser := proto.NewParser(reader)
	definition, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return definition, nil
}

func (e *Env) pickMethPrefixAndGoFile() error {
	files, err := ioutil.ReadDir(e.targetDir)
	if err != nil {
		panic(err)
	}
	if len(files) == 0 {
		return nil
	}

	// 填充第一个go文件路径,后续新增的方法都在这里
	e.firsServiceGoFile = path.Join(e.targetDir, "/", files[0].Name())

	funcCounter := make(map[string]int, 64)
	for _, file := range files {
		name := file.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		filePath := path.Join(e.targetDir, "/", name)
		open, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer open.Close()

		scanner := bufio.NewScanner(open)
		for {
			if !scanner.Scan() {
				break //文件读完了,退出for
			}
			line := scanner.Text() //获取每一行
			hasPrefix := strings.HasPrefix(line, "func")
			if !hasPrefix {
				continue
			}
			ss := strings.SplitN(line, ")", 2)
			if len(ss) != 2 {
				continue
			}
			// ss[0] => func (s *AssetService:0
			funcCounter[ss[0]] += 1
		}
	}

	// 出现频率最高的就是methPrefix
	max := 0
	f := ""
	for k, v := range funcCounter {
		if v > max {
			max = v
			f = k
		}
	}

	e.methPrefix = strings.ReplaceAll(f, " ", "") + ")"

	ss := strings.SplitN(f, "*", 2)
	if len(ss) != 2 {
		return errors.New("parser app struct name fail")
	}
	e.appStructName = ss[1]
	return nil
}

func (e *Env) collectFileMethods(filePath string) ([]string, error) {
	open, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer open.Close()

	list := make([]string, 0, 16)
	scanner := bufio.NewScanner(open)
	for {
		if !scanner.Scan() {
			break //文件读完了,退出for
		}
		line := scanner.Text() //获取每一行
		name := e.pickMethName(line, e.methPrefix)
		if name != "" {
			list = append(list, name)
		}
	}
	return list, nil
}

func (e *Env) collectDirMethods(dir string) (map[string]bool, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	list := make(map[string]bool, len(files)*16)
	for _, file := range files {
		name := file.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		meths, err := e.collectFileMethods(path.Join(dir, "/", name))
		if err != nil {
			return nil, err
		}
		for _, m := range meths {
			list[m] = true
		}
	}

	return list, nil
}

func (e *Env) pickMethName(line, prefix string) string {
	// 先去除空格
	if !strings.HasPrefix(line, "func") {
		return ""
	}
	line = strings.ReplaceAll(line, " ", "")
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	line = strings.ReplaceAll(line, prefix, "")
	splitN := strings.SplitN(line, "(", 2)
	if len(splitN) != 2 {
		return ""
	}
	return splitN[0]
}

func (e *Env) appendServiceFile(s *Service) error {
	// 追加
	// 1. 提取现有的方法
	existMeths, err := e.collectDirMethods(e.targetDir)
	if err != nil {
		return err
	}
	// 2. 对比差异,获得差集
	fmt.Fprintln(os.Stdout, "新增的方法如下:")
	diff := make([]*Method, 0, 16)
	for _, method := range s.Methods {
		if existMeths[method.Name] {
			continue
		}
		GreenLog("%s", method.Name)
		// 因为go具体实现与proto中的可能不同,重写Service
		method.Service = e.appStructName
		diff = append(diff, method)
	}
	if len(diff) == 0 {
		YellowLog("%s", "no method to add!")
		return nil
	}
	// 3. 构建二进制数据
	s.Service = e.appStructName
	bytes, err := s.append(diff)
	if err != nil {
		return err
	}

	// 4. 追加到现有的文件中
	openFile, err := os.OpenFile(e.firsServiceGoFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer openFile.Close()
	_, err = openFile.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func createServiceFile(s *Service, targetFile string) error {
	b, err := s.create()
	if err != nil {
		return err
	}
	if err := os.WriteFile(targetFile, b, 0o644); err != nil {
		return err
	}
	return nil
}

func getMethodType(streamsRequest, streamsReturns bool) MethodType {
	if !streamsRequest && !streamsReturns {
		return unaryType
	} else if streamsRequest && streamsReturns {
		return twoWayStreamsType
	} else if streamsRequest {
		return requestStreamsType
	} else if streamsReturns {
		return returnsStreamsType
	}
	return unaryType
}

func serviceName(name string) string {
	return toUpperCamelCase(strings.Split(name, ".")[0])
}

func toUpperCamelCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = cases.Title(language.Und, cases.NoLower).String(s)
	return strings.ReplaceAll(s, " ", "")
}

func run(cmd *cobra.Command, args []string) {
	var (
		pkg string
		res []*Service
	)
	env := NewEnv()

	definition, err := env.parserProto()
	if err != nil {
		log.Fatal(err)
	}

	proto.Walk(definition,
		proto.WithOption(func(o *proto.Option) {
			if o.Name == "go_package" {
				pkg = strings.Split(o.Constant.Source, ";")[0]
			}
		}),
		proto.WithService(func(s *proto.Service) {
			cs := &Service{
				Package: pkg,
				Service: serviceName(s.Name),
			}
			for _, e := range s.Elements {
				r, ok := e.(*proto.RPC)
				if !ok {
					continue
				}
				cs.Methods = append(cs.Methods, &Method{
					Service: serviceName(s.Name),
					Name:    serviceName(r.Name),
					Request: r.RequestType,
					Reply:   r.ReturnsType,
					Type:    getMethodType(r.StreamsRequest, r.StreamsReturns),
				})
			}
			res = append(res, cs)
		}),
	)

	// 追加写的方式
	// 原则,提取方法名,发现存在就跳过,没有就追加

	// 还没有实现,走新增
	if env.appStructName == "" {
		for _, s := range res {
			to := path.Join(env.targetDir, strings.ToLower(s.Service)+".go")
			err := createServiceFile(s, to)
			if err != nil {
				RedLog("create service go err:%v", err)
				return
			}
		}
	}

	// 已经有实现,走追加
	for _, s := range res {
		err = env.appendServiceFile(s)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}

}
