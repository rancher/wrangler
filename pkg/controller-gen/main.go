package controllergen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	cgargs "github.com/rancher/wrangler/pkg/controller-gen/args"
	ctemplate "github.com/rancher/wrangler/pkg/controller-gen/template"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/imports"
	"k8s.io/apimachinery/pkg/runtime/schema"
	csargs "k8s.io/code-generator/cmd/client-gen/args"
	cs "k8s.io/code-generator/cmd/client-gen/generators"
	types2 "k8s.io/code-generator/cmd/client-gen/types"
	dpargs "k8s.io/code-generator/cmd/deepcopy-gen/args"
	infargs "k8s.io/code-generator/cmd/informer-gen/args"
	inf "k8s.io/code-generator/cmd/informer-gen/generators"
	lsargs "k8s.io/code-generator/cmd/lister-gen/args"
	ls "k8s.io/code-generator/cmd/lister-gen/generators"
	"k8s.io/gengo/args"
	dp "k8s.io/gengo/examples/deepcopy-gen/generators"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/types"
)

var (
	t            = true
	outputDir    = "./pkg/generated"
	outputAPIDir = "./pkg/apis"
)

func funcs() template.FuncMap {
	return template.FuncMap{
		"capitalize":   convert.Capitalize,
		"unCapitalize": convert.Uncapitalize,
		"upper":        strings.ToUpper,
		"toLower":      strings.ToLower,
		"toPlural":     name.GuessPluralName,
	}
}

func Run(opts cgargs.Options) {
	customArgs := &cgargs.CustomArgs{
		Options:      opts,
		TypesByGroup: map[schema.GroupVersion][]*cgargs.Name{},
		Package:      opts.OutputPackage,
		OutputBase:   args.DefaultSourceTree(),
	}
	parseTypes(customArgs)

	if customArgs.OutputBase == "./" { //go modules
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			return
		}

		defer os.RemoveAll(tempDir)
		customArgs.OutputBase = tempDir
	}

	if err := generateKubernetes(customArgs); err != nil {
		panic(err)
	}

	groups := map[string]bool{}
	listerGroups := map[string]bool{}
	informerGroups := map[string]bool{}
	deepCopygroups := map[string]bool{}
	for groupName, group := range customArgs.Options.Groups {
		if group.GenerateTypes {
			deepCopygroups[groupName] = true
		}
		if group.GenerateClients {
			groups[groupName] = true
		}
		if group.GenerateListers {
			listerGroups[groupName] = true
		}
		if group.GenerateInformers {
			informerGroups[groupName] = true
		}
	}

	if err := generateDeepcopy(deepCopygroups, customArgs); err != nil {
		logrus.Fatalf("deepcopy failed: %v", err)
	}

	if err := generateClientset(groups, customArgs); err != nil {
		logrus.Fatalf("clientset failed: %v", err)
	}

	if err := generateListers(listerGroups, customArgs); err != nil {
		logrus.Fatalf("listers failed: %v", err)
	}

	if err := generateInformers(informerGroups, customArgs); err != nil {
		logrus.Fatalf("informers failed: %v", err)
	}

	if err := copyGoPathToModules(customArgs); err != nil {
		logrus.Fatalf("copy to go module failed: %v", err)
	}

}

func sourcePackagePath(customArgs *cgargs.CustomArgs, pkgName string) string {
	pkgSplit := strings.Split(pkgName, string(os.PathSeparator))
	pkg := filepath.Join(customArgs.OutputBase, strings.Join(pkgSplit[:3], string(os.PathSeparator)))
	return pkg
}

//until k8s code-gen supports gopath
func copyGoPathToModules(customArgs *cgargs.CustomArgs) error {

	pathsToCopy := map[string]bool{}
	for _, types := range customArgs.TypesByGroup {
		for _, names := range types {
			pkg := sourcePackagePath(customArgs, names.Package)
			pathsToCopy[pkg] = true
		}
	}

	pkg := sourcePackagePath(customArgs, customArgs.Package)
	pathsToCopy[pkg] = true

	for pkg, _ := range pathsToCopy {
		if _, err := os.Stat(pkg); os.IsNotExist(err) {
			continue
		}

		return filepath.Walk(pkg, func(path string, info os.FileInfo, err error) error {
			newPath := strings.Replace(path, pkg, ".", 1)
			if info.IsDir() {
				return os.MkdirAll(newPath, info.Mode())
			}

			return copyFile(path, newPath)
		})
	}

	return nil
}

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func generateDeepcopy(groups map[string]bool, customArgs *cgargs.CustomArgs) error {
	if len(groups) == 0 {
		return nil
	}

	deepCopyCustomArgs := &dpargs.CustomArgs{}

	args := args.Default().WithoutDefaultFlagParsing()
	args.CustomArgs = deepCopyCustomArgs
	args.OutputBase = customArgs.OutputBase
	args.OutputPackagePath = outputAPIDir
	args.OutputFileBaseName = "zz_generated_deepcopy"
	args.GoHeaderFilePath = customArgs.Options.Boilerplate

	for gv, names := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		args.InputDirs = append(args.InputDirs, names[0].Package)
		deepCopyCustomArgs.BoundingDirs = append(deepCopyCustomArgs.BoundingDirs, names[0].Package)
	}

	return args.Execute(dp.NameSystems(),
		dp.DefaultNameSystem(),
		dp.Packages)
}

func generateClientset(groups map[string]bool, customArgs *cgargs.CustomArgs) error {
	if len(groups) == 0 {
		return nil
	}

	args, clientSetArgs := csargs.NewDefaults()
	clientSetArgs.ClientsetName = "versioned"
	args.OutputBase = customArgs.OutputBase
	args.OutputPackagePath = filepath.Join(customArgs.Package, "clientset")
	args.GoHeaderFilePath = customArgs.Options.Boilerplate

	var order []schema.GroupVersion

	for gv := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		order = append(order, gv)
	}
	sort.Slice(order, func(i, j int) bool {
		return order[i].Group < order[j].Group
	})

	for _, gv := range order {
		packageName := customArgs.Options.Groups[gv.Group].PackageName
		if packageName == "" {
			packageName = gv.Group
		}
		names := customArgs.TypesByGroup[gv]
		args.InputDirs = append(args.InputDirs, names[0].Package)
		clientSetArgs.Groups = append(clientSetArgs.Groups, types2.GroupVersions{
			PackageName: packageName,
			Group:       types2.Group(gv.Group),
			Versions: []types2.PackageVersion{
				{
					Version: types2.Version(gv.Version),
					Package: names[0].Package,
				},
			},
		})
	}

	return args.Execute(cs.NameSystems(nil),
		cs.DefaultNameSystem(),
		setGenClient(groups, convertTypesByGroup(customArgs.TypesByGroup), cs.Packages))
}

func convertTypesByGroup(in map[schema.GroupVersion][]*cgargs.Name) (result map[schema.GroupVersion][]*types.Name) {
	result = map[schema.GroupVersion][]*types.Name{}
	for gv, names := range in {
		for _, name := range names {
			result[gv] = append(result[gv], &types.Name{
				Name: name.Name,
				Package: name.Package,
				Path: name.Path,
			})
		}
	}
	return result
}

func setGenClient(groups map[string]bool, typesByGroup map[schema.GroupVersion][]*types.Name, f func(*generator.Context, *args.GeneratorArgs) generator.Packages) func(*generator.Context, *args.GeneratorArgs) generator.Packages {
	return func(context *generator.Context, generatorArgs *args.GeneratorArgs) generator.Packages {
		for gv, names := range typesByGroup {
			if !groups[gv.Group] {
				continue
			}
			for _, name := range names {
				var (
					p           = context.Universe.Package(name.Package)
					t           = p.Type(name.Name)
					status      bool
					nsed        bool
					kubebuilder bool
				)

				for _, line := range append(t.SecondClosestCommentLines, t.CommentLines...) {
					switch {
					case strings.Contains(line, "+kubebuilder:object:root=true"):
						kubebuilder = true
						t.SecondClosestCommentLines = append(t.SecondClosestCommentLines, "+genclient")
					case strings.Contains(line, "+kubebuilder:subresource:status"):
						status = true
					case strings.Contains(line, "+kubebuilder:resource:") && strings.Contains(line, "scope=Namespaced"):
						nsed = true
					}
				}

				if kubebuilder {
					if !nsed {
						t.SecondClosestCommentLines = append(t.SecondClosestCommentLines, "+genclient:nonNamespaced")
					}
					if !status {
						t.SecondClosestCommentLines = append(t.SecondClosestCommentLines, "+genclient:noStatus")
					}

					foundGroup := false
					for _, comment := range p.DocComments {
						if strings.Contains(comment, "+groupName=") {
							foundGroup = true
							break
						}
					}

					if !foundGroup {
						p.DocComments = append(p.DocComments, "+groupName="+gv.Group)
						p.Comments = append(p.Comments, "+groupName="+gv.Group)
						fmt.Println(gv.Group, p.DocComments, p.Comments, p.Path)
					}
				}
			}
		}
		return f(context, generatorArgs)
	}
}

func generateInformers(groups map[string]bool, customArgs *cgargs.CustomArgs) error {
	if len(groups) == 0 {
		return nil
	}

	args, clientSetArgs := infargs.NewDefaults()
	clientSetArgs.VersionedClientSetPackage = filepath.Join(customArgs.Package, "clientset/versioned")
	clientSetArgs.ListersPackage = filepath.Join(customArgs.Package, "listers")
	args.OutputBase = customArgs.OutputBase
	args.OutputPackagePath = filepath.Join(customArgs.Package, "informers")
	args.GoHeaderFilePath = customArgs.Options.Boilerplate

	for gv, names := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		args.InputDirs = append(args.InputDirs, names[0].Package)
	}

	return args.Execute(inf.NameSystems(nil),
		inf.DefaultNameSystem(),
		setGenClient(groups, convertTypesByGroup(customArgs.TypesByGroup), inf.Packages))
}

func generateListers(groups map[string]bool, customArgs *cgargs.CustomArgs) error {
	if len(groups) == 0 {
		return nil
	}

	args, _ := lsargs.NewDefaults()
	args.OutputBase = customArgs.OutputBase
	args.OutputPackagePath = filepath.Join(customArgs.Package, "listers")
	args.GoHeaderFilePath = customArgs.Options.Boilerplate

	for gv, names := range customArgs.TypesByGroup {
		if !groups[gv.Group] {
			continue
		}
		args.InputDirs = append(args.InputDirs, names[0].Package)
	}

	return args.Execute(ls.NameSystems(nil),
		ls.DefaultNameSystem(),
		setGenClient(groups, convertTypesByGroup(customArgs.TypesByGroup), ls.Packages))
}

func parseTypes(customArgs *cgargs.CustomArgs) []string {
	fset := token.NewFileSet()

	for groupName, group := range customArgs.Options.Groups {
		if group.GenerateTypes || group.GenerateClients {
			if group.InformersPackage == "" {
				group.InformersPackage = filepath.Join(customArgs.Package, "informers/externalversions")
			}
			if group.ClientSetPackage == "" {
				group.ClientSetPackage = filepath.Join(customArgs.Package, "clientset/versioned")
			}
			if group.ListersPackage == "" {
				group.ListersPackage = filepath.Join(customArgs.Package, "listers")
			}
			customArgs.Options.Groups[groupName] = group
		}
	}

	for groupName, group := range customArgs.Options.Groups {
		if err := cgargs.ObjectsToGroupVersion(groupName, group.Types, customArgs.TypesByGroup); err != nil {
			// sorry, should really handle this better
			panic(err)
		}
	}

	for gv, names := range customArgs.TypesByGroup {
		for _, name := range names {
			pkgs, commments, err := loadComments(fset, gv)
			if err != nil {
				panic(err)
			}

			name.Namespaced = checkNamespaced(fset, pkgs[gv.Version], name, commments)
		}
	}

	var inputDirs []string
	for _, names := range customArgs.TypesByGroup {
		inputDirs = append(inputDirs, names[0].Package)
	}

	return inputDirs
}

func generateKubernetes(customArgs *cgargs.CustomArgs) error {
	for gv, names := range customArgs.TypesByGroup {
		if err := generateRegister(gv); err != nil {
			return err
		}

		if err := generateRegisterGvk(gv, names); err != nil {
			return err
		}

		if err := generateList(gv, names); err != nil {
			return err
		}

		if err := generateFactory(gv); err != nil {
			return err
		}

		if err := generateFactory(gv); err != nil {
			return err
		}

		if err := generateInterface(gv); err != nil {
			return err
		}

		if err := generateGroupVersionInterface(gv, names); err != nil {
			return err
		}

		if err := generateController(gv, names); err != nil {
			return err
		}
	}

	if err := gofmt(outputDir, "controllers"); err != nil {
		return err
	}
	if err := gofmt(outputAPIDir, ""); err != nil {
		return err
	}
	return nil
}

func gofmt(workDir, pkg string) error {
	return filepath.Walk(filepath.Join(workDir, pkg), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := imports.Process(path, content, &imports.Options{
			Fragment:   true,
			Comments:   true,
			TabIndent:  true,
			TabWidth:   8,
		})
		if err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(formatted)
		return err
	})
}

func generateRegisterGvk(gv schema.GroupVersion, names []*cgargs.Name) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputAPIDir, gv.Group, gv.Version)
	filePath := "zz_generated_register.go"
	output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
	if err != nil {
		return err
	}
	defer output.Close()

	packageName := strings.SplitN(gv.Group, ".", 2)[0]
	typeTemplate, err := template.New("register_gvk.template").
		Funcs(funcs()).
		Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.RegisterGroupVersionTemplate, "%BACK%", "`", -1))
	if err != nil {
		return err
	}

	if err := typeTemplate.Execute(output, map[string]interface{}{
		"Package": packageName,
		"group":   gv.Group,
		"Names":   names,
		"version": gv.Version,
	}); err != nil {
		return err
	}
	return output.Close()
}

func generateRegister(gv schema.GroupVersion) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputAPIDir, gv.Group)
	filePath := "zz_generated_register.go"
	output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
	if err != nil {
		return err
	}
	defer output.Close()

	typeTemplate, err := template.New("register.template").
		Funcs(funcs()).
		Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.RegisterTemplate, "%BACK%", "`", -1))
	if err != nil {
		return err
	}

	group := strings.SplitN(gv.Group, ".", 2)[0]
	if err := typeTemplate.Execute(output, map[string]interface{}{
		"Package":      group,
		"Groupversion": gv.Group,
	}); err != nil {
		return err
	}
	return output.Close()
}

func generateList(gv schema.GroupVersion, names []*cgargs.Name) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputAPIDir, gv.Group, gv.Version)
	filePath := "zz_generated_list_types.go"
	output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
	if err != nil {
		return err
	}
	defer output.Close()

	typeTemplate, err := template.New("list.template").
		Funcs(funcs()).
		Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.ListTemplate, "%BACK%", "`", -1))
	if err != nil {
		return err
	}

	if err := typeTemplate.Execute(output, map[string]interface{}{
		"Group":   gv.Group,
		"Names":   names,
		"Version": gv.Version,
	}); err != nil {
		return err
	}
	return output.Close()
}

func generateFactory(gv schema.GroupVersion) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputDir, "controllers", gv.Group)
	if err := os.MkdirAll(outputRegisterSchemaDir, 0755); err != nil {
		return err
	}
	filePath := "factory.go"
	output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
	if err != nil {
		return err
	}
	defer output.Close()

	typeTemplate, err := template.New("factory.template").
		Funcs(funcs()).
		Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.FactoryTemplate, "%BACK%", "`", -1))
	if err != nil {
		return err
	}

	packageName := strings.SplitN(gv.Group, ".", 2)[0]
	if err := typeTemplate.Execute(output, map[string]interface{}{
		"Package": packageName,
	}); err != nil {
		return err
	}
	return output.Close()
}

func generateInterface(gv schema.GroupVersion) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputDir, "controllers", gv.Group)
	if err := os.MkdirAll(outputRegisterSchemaDir, 0755); err != nil {
		return err
	}
	filePath := "interface.go"
	output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
	if err != nil {
		return err
	}
	defer output.Close()

	typeTemplate, err := template.New("interface.template").
		Funcs(funcs()).
		Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.InterfaceTemplate, "%BACK%", "`", -1))
	if err != nil {
		return err
	}

	packageName := strings.SplitN(gv.Group, ".", 2)[0]
	if err := typeTemplate.Execute(output, map[string]interface{}{
		"Package": packageName,
		"Group":   gv.Group,
		"Version": gv.Version,
	}); err != nil {
		return err
	}
	return output.Close()
}

func generateGroupVersionInterface(gv schema.GroupVersion, names []*cgargs.Name) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputDir, "controllers", gv.Group, gv.Version)
	if err := os.MkdirAll(outputRegisterSchemaDir, 0755); err != nil {
		return err
	}
	filePath := "interface.go"
	output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
	if err != nil {
		return err
	}
	defer output.Close()

	typeTemplate, err := template.New("interface.template").
		Funcs(funcs()).
		Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.GroupInterfaceTemplate, "%BACK%", "`", -1))
	if err != nil {
		return err
	}

	if err := typeTemplate.Execute(output, map[string]interface{}{
		"Group":   gv.Group,
		"Names":   names,
		"Version": gv.Version,
	}); err != nil {
		return err
	}
	return output.Close()
}

func generateController(gv schema.GroupVersion, names []*cgargs.Name) error {
	outputRegisterSchemaDir := filepath.Join(args.DefaultSourceTree(), outputDir, "controllers", gv.Group, gv.Version)
	if err := os.MkdirAll(outputRegisterSchemaDir, 0755); err != nil {
		return err
	}

	for _, name := range names {
		filePath := fmt.Sprintf("%v.go", strings.ToLower(name.Name))
		output, err := os.Create(path.Join(outputRegisterSchemaDir, filePath))
		if err != nil {
			return err
		}

		typeTemplate, err := template.New("controller.template").
			Funcs(funcs()).
			Parse(strings.Replace(ctemplate.Boilerplate+ctemplate.ControllerTemplate, "%BACK%", "`", -1))
		if err != nil {
			return err
		}

		if err := typeTemplate.Execute(output, map[string]interface{}{
			"Name":       name.Name,
			"Group":      gv.Group,
			"Version":    gv.Version,
			"namespaced": name.Namespaced,
			"hasStatus":  name.HasStatus,
			"statusType": name.StatusType,
		}); err != nil {
			return err
		}
		if err := output.Close(); err != nil {
			return err
		}
	}
	return nil
}

func loadComments(fset *token.FileSet, gv schema.GroupVersion) (map[string]*ast.Package, map[fileline]*ast.CommentGroup, error) {
	pkgs, err := parser.ParseDir(fset, fmt.Sprintf("./pkg/apis/%v/%v", gv.Group, gv.Version), func(info os.FileInfo) bool {
		if strings.HasPrefix(info.Name(), "zz_generated") {
			return false
		}
		return true
	}, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	endlineToComment := map[fileline]*ast.CommentGroup{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, comment := range file.Comments {
				position := fset.Position(comment.End())
				endlineToComment[fileline{filename: position.Filename, line: position.Line}] = comment
			}
		}
	}
	return pkgs, endlineToComment, nil
}

func checkNamespaced(fset *token.FileSet, root ast.Node, nameToLook *cgargs.Name, endlineToComment map[fileline]*ast.CommentGroup) bool {
	isNamespaced := true

	ast.Inspect(root, func(node ast.Node) bool {
		switch t := node.(type) {
		case *ast.TypeSpec:
			if t.Name.Name == nameToLook.Name {
				position := fset.Position(node.Pos())
				filename := position.Filename
				line := fset.Position(node.Pos()).Line - 2
				comments := endlineToComment[fileline{filename: filename, line: line}]
				if comments != nil {
					for _, comment := range comments.List {
						if comment.Text == "// +genclient:nonNamespaced" {
							isNamespaced = false
						}
					}
				}
			}
		}
		return true
	})

	return isNamespaced
}

type fileline struct {
	filename string
	line     int
}
