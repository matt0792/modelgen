package types

type MappingConfig struct {
	SourceType *StructInfo
	TargetType *StructInfo
	OmitFields map[string]bool
	FieldMap   map[string]string
}
