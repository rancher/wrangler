package generator

var RegisterTemplate = `package {{.Package}}

const (
	// Package-wide consts from generator "zz_generated_register".
	GroupName = "{{.Groupversion}}"
)`
