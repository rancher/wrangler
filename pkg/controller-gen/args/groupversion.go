package args

import (
	gotypes "go/types"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/code-generator/cmd/client-gen/generators/util"
	"k8s.io/code-generator/cmd/client-gen/path"
	"k8s.io/gengo/types"
)

const (
	needsComment = `
		// +genclient
		// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	`
	objectComment = "+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object"
)

type Name struct {
	// Empty if embedded or builtin. This is the package path unless Path is specified.
	Package string
	// The type name.
	Name string
	// An optional location of the type definition for languages that can have disjoint
	// packages and paths.
	Path string

	Namespaced bool
	HasStatus  bool
	StatusType string
}

func translate(types []Type, err error) (result []interface{}, _ error) {
	for _, v := range types {
		result = append(result, v)
	}
	return result, err
}

func ObjectsToGroupVersion(group string, objs []interface{}, ret map[schema.GroupVersion][]*Name) error {
	for _, obj := range objs {
		if s, ok := obj.(string); ok {
			types, err := translate(ScanDirectory(s))
			if err != nil {
				return err
			}
			if err := ObjectsToGroupVersion(group, types, ret); err != nil {
				return err
			}
			continue
		}

		version, t := toVersionType(obj)
		gv := schema.GroupVersion{
			Group:   group,
			Version: version,
		}
		ret[gv] = append(ret[gv], t)
	}

	return nil
}

func toVersionType(obj interface{}) (string, *Name) {
	switch v := obj.(type) {
	case Type:
		return v.Version, &Name{
			Package:    v.Package,
			Name:       v.Name,
			HasStatus:  v.HasStatus,
			Namespaced: v.Namespaced,
			StatusType: v.StatusType,
		}
	case *Type:
		return v.Version, &Name{
			Package:    v.Package,
			Name:       v.Name,
			HasStatus:  v.HasStatus,
			StatusType: v.StatusType,
			Namespaced: v.Namespaced,
		}
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	statusField, hasStatus := t.FieldByName("Status")
	pkg := path.Vendorless(t.PkgPath())
	return versionFromPackage(pkg), &Name{
		Package:   pkg,
		Name:      t.Name(),
		HasStatus: hasStatus,
		StatusType: statusField.Type.Name(),
	}
}

func versionFromPackage(pkg string) string {
	parts := strings.Split(pkg, "/")
	return parts[len(parts)-1]
}

func CheckType(passedType *types.Type) {
	tags := util.MustParseClientGenTags(passedType.SecondClosestCommentLines)
	if !tags.GenerateClient {
		panic("Type " + passedType.String() + " is missing comment " + needsComment)
	}
	found := false
	for _, line := range passedType.SecondClosestCommentLines {
		if strings.Contains(line, objectComment) {
			found = true
		}
	}
	if !found {
		panic("Type " + passedType.String() + " is missing comment " + objectComment)
	}
}

func ScanDirectory(pkgPath string) (result []Type, err error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  pkgPath,
	})
	if err != nil {
		return nil, err
	}
	for _, v := range pkgs[0].TypesInfo.Defs {
		typeAndName, ok := v.(*gotypes.TypeName)
		if !ok {
			continue
		}
		s, ok := typeAndName.Type().Underlying().(*gotypes.Struct)
		if !ok {
			continue
		}

		hasStatus := false
		statusType := ""
		for i := 0; i < s.NumFields(); i++ {
			f := s.Field(i)
			if f.Name() == "Status" {
				hasStatus = true
				if _, ok := f.Type().(*gotypes.Named); !ok {
					hasStatus = false
					continue
				}
				statusType = f.Type().(*gotypes.Named).Obj().Name()
			}
		}
		for i := 0; i < s.NumFields(); i++ {
			f := s.Field(i)
			if f.Embedded() && f.Type().String() == "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta" {
				pkgPath := path.Vendorless(pkgs[0].PkgPath)
				result = append(result, Type{
					Package:    pkgPath,
					Version:    versionFromPackage(pkgPath),
					Name:       typeAndName.Name(),
					HasStatus:  hasStatus,
					StatusType: statusType,
				})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return
}
