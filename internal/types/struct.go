package types

type StructInfo struct {
	PackageName string // eg: "externalservice"
	PackagePath string // eg: "github.com/matt0792/modelgen/externalservice"
	TypeName    string
	Fields      []FieldInfo
}
