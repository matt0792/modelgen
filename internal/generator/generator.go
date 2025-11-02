package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"

	"github.com/matt0792/modelgen/internal/types"
)

type Generator struct {
	buf              *bytes.Buffer
	generatedStructs map[string]bool   // track structs that have already been generated
	nestedStructs    []types.FieldInfo // track nested that need generation
}

func New() *Generator {
	return &Generator{
		generatedStructs: make(map[string]bool),
	}
}

func (g *Generator) Generate(config types.MappingConfig) (string, error) {
	g.buf = &bytes.Buffer{}
	g.generatedStructs = make(map[string]bool)
	g.nestedStructs = []types.FieldInfo{}

	// pkg & imports
	g.writePackage(config.TargetType.PackageName)
	g.writeImports(config)

	// generate definition
	g.generateStructDef(config)

	// generate methods
	g.generateFromMethod(config)
	g.generateToMethod(config)

	// format
	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("format error: %w\nGenerated code:\n%s", err, g.buf.String())
	}

	return string(formatted), nil
}

// GenerateStructAndMethods generates only the struct and methods without package/imports
func (g *Generator) GenerateStructAndMethods(config types.MappingConfig) (string, error) {
	g.buf = &bytes.Buffer{}
	g.generatedStructs = make(map[string]bool)
	g.nestedStructs = []types.FieldInfo{}

	// Generate struct definition
	g.generateStructDef(config)

	// Generate From method
	g.generateFromMethod(config)

	// Generate To method
	g.generateToMethod(config)

	return g.buf.String(), nil
}

func (g *Generator) writePackage(pkgName string) {
	fmt.Fprintf(g.buf, "package %s\n\n", pkgName)
}

func (g *Generator) writeImports(config types.MappingConfig) {
	imports := g.collectImports(config)
	if len(imports) == 0 {
		return
	}

	g.buf.WriteString("import (\n")
	for _, imp := range imports {
		fmt.Fprintf(g.buf, "\t\"%s\"\n", imp)
	}
	g.buf.WriteString(")\n\n")
}

func (g *Generator) collectImports(config types.MappingConfig) []string {
	importsMap := make(map[string]bool)

	// generate source package import (needed for From(), To() methods)
	if config.SourceType.PackagePath != "" {
		importsMap[config.SourceType.PackagePath] = true
	}

	for _, field := range config.SourceType.Fields {
		if strings.Contains(field.Type, "time.Time") {
			importsMap["time"] = true
		}
	}
	for _, field := range config.TargetType.Fields {
		if strings.Contains(field.Type, "time.Time") {
			importsMap["time"] = true
		}
	}

	imports := make([]string, 0, len(importsMap))
	for imp := range importsMap {
		imports = append(imports, imp)
	}
	return imports
}

func (g *Generator) generateStructDef(config types.MappingConfig) {
	targetTypeName := config.TargetType.TypeName

	fmt.Fprintf(g.buf, "type %s struct {\n", targetTypeName)

	// generate fields from source
	for _, sourceField := range config.SourceType.Fields {
		if config.OmitFields[sourceField.Name] {
			continue
		}

		// check for custom mapping
		targetFieldName := sourceField.Name
		for srcName, tgtName := range config.FieldMap {
			if srcName == sourceField.Name {
				targetFieldName = tgtName
				break
			}
		}

		// clean type name
		typeStr := g.cleanTypeName(sourceField.Type, config)
		fmt.Fprintf(g.buf, "\t%s %s\n", targetFieldName, typeStr)
	}

	g.buf.WriteString("}\n\n")
	g.generatedStructs[targetTypeName] = true
}

func (g *Generator) cleanTypeName(typeStr string, config types.MappingConfig) string {
	sourcePrefix := config.SourceType.PackageName + "."
	return strings.ReplaceAll(typeStr, sourcePrefix, "")
}

func (g *Generator) generateFromMethod(config types.MappingConfig) {
	targetType := config.TargetType.TypeName
	sourceType := config.SourceType.TypeName
	sourcePackage := config.SourceType.PackageName

	g.buf.WriteString("// From maps from an external struct to a local\n")
	g.buf.WriteString("//\n")
	fmt.Fprintf(g.buf, "// Usage: local%s := (&%s{}).From(&external%s)\n", targetType, targetType, sourceType)
	fmt.Fprintf(g.buf, "func (t *%s) From(src *%s.%s) *%s {\n", targetType, sourcePackage, sourceType, targetType)
	g.buf.WriteString("\tif src == nil {\n")
	g.buf.WriteString("\t\treturn nil\n")
	g.buf.WriteString("\t}\n\n")

	fmt.Fprintf(g.buf, "\treturn &%s{\n", targetType)

	// generate fields from source
	for _, sourceField := range config.SourceType.Fields {
		if config.OmitFields[sourceField.Name] {
			continue
		}

		// get target field name (derive if not overridden)
		targetFieldName := sourceField.Name
		for srcName, tgtName := range config.FieldMap {
			if srcName == sourceField.Name {
				targetFieldName = tgtName
				break
			}
		}

		// create pseudo target field (for type comparison)
		targetFieldType := g.cleanTypeName(sourceField.Type, config)
		targetField := types.FieldInfo{
			Name:      targetFieldName,
			Type:      targetFieldType,
			IsPointer: sourceField.IsPointer,
			IsSlice:   sourceField.IsSlice,
			IsNested:  sourceField.IsNested,
		}

		// generate mapping expression
		mappingExpr := g.generateFieldMapping(sourceField, targetField, config)
		fmt.Fprintf(g.buf, "\t\t%s: %s,\n", targetFieldName, mappingExpr)
	}

	g.buf.WriteString("\t}\n")
	g.buf.WriteString("}\n\n")
}

func (g *Generator) generateToMethod(config types.MappingConfig) {
	targetType := config.TargetType.TypeName
	sourceType := config.SourceType.TypeName
	sourcePackage := config.SourceType.PackageName

	fmt.Fprintf(g.buf, "// Usage: external%s := %s.To()\n", sourceType, targetType)

	fmt.Fprintf(g.buf, "func (t *%s) To() %s.%s {\n", targetType, sourcePackage, sourceType)

	fmt.Fprintf(g.buf, "\treturn %s.%s{\n", sourcePackage, sourceType)

	// Generate field mappings (reverse of From)
	for _, sourceField := range config.SourceType.Fields {
		// If field was omitted in target, we still need to provide a value in source
		if config.OmitFields[sourceField.Name] {
			// Use zero value for omitted fields
			fmt.Fprintf(g.buf, "\t\t%s: %s,\n", sourceField.Name, g.zeroValue(sourceField.Type))
			continue
		}

		// Get target field name (check for custom mapping)
		targetFieldName := sourceField.Name
		for srcName, tgtName := range config.FieldMap {
			if srcName == sourceField.Name {
				targetFieldName = tgtName
				break
			}
		}

		// Create pseudo target field
		targetFieldType := g.cleanTypeName(sourceField.Type, config)
		targetField := types.FieldInfo{
			Name:      targetFieldName,
			Type:      targetFieldType,
			IsPointer: sourceField.IsPointer,
			IsSlice:   sourceField.IsSlice,
			IsNested:  sourceField.IsNested,
		}

		// Generate reverse mapping expression
		mappingExpr := g.generateReverseFieldMapping(targetField, sourceField, config)
		fmt.Fprintf(g.buf, "\t\t%s: %s,\n", sourceField.Name, mappingExpr)
	}

	g.buf.WriteString("\t}\n")
	g.buf.WriteString("}\n\n")
}

func (g *Generator) findTargetField(sourceField types.FieldInfo, config types.MappingConfig) *types.FieldInfo {
	// Check if field is omitted
	if config.OmitFields[sourceField.Name] {
		return nil
	}

	// Check if there's a custom field mapping
	if targetFieldName, ok := config.FieldMap[sourceField.Name]; ok {
		for _, tf := range config.TargetType.Fields {
			if tf.Name == targetFieldName {
				return &tf
			}
		}
	}

	// Otherwise, look for field with same name
	for _, tf := range config.TargetType.Fields {
		if tf.Name == sourceField.Name {
			return &tf
		}
	}

	return nil
}

func (g *Generator) generateFieldMapping(sf, tf types.FieldInfo, config types.MappingConfig) string {
	if config.OmitFields[sf.Name] {
		return g.zeroValue(tf.Type)
	}

	// Handle slices first (including slices of nested structs)
	if sf.IsSlice && tf.IsSlice {
		return g.generateSliceMapping(sf, tf, config)
	}

	// Handle nested structs
	if sf.IsNested && tf.IsNested {
		return g.generateNestedMapping(sf, tf, config)
	}

	// For primitive types, direct assignment
	return fmt.Sprintf("src.%s", sf.Name)
}

func (g *Generator) generateReverseFieldMapping(tf, sf types.FieldInfo, config types.MappingConfig) string {
	// This is the reverse mapping for To() method
	// tf is target field (in our generated struct), sf is source field (in external struct)

	// Clean types for comparison
	sourceTypeClean := g.cleanTypeName(sf.Type, config)
	targetTypeClean := g.cleanTypeName(tf.Type, config)

	if sourceTypeClean == targetTypeClean {
		return fmt.Sprintf("t.%s", tf.Name)
	}

	if tf.IsSlice && sf.IsSlice {
		return g.generateReverseSliceMapping(tf, sf, config)
	}

	if tf.IsNested && sf.IsNested {
		return g.generateReverseNestedMapping(tf, sf, config)
	}

	// Direct conversion
	return fmt.Sprintf("t.%s", tf.Name)
}

func (g *Generator) generateSliceMapping(sf, tf types.FieldInfo, config types.MappingConfig) string {
	sourceElemType := strings.TrimPrefix(sf.Type, "[]")
	targetElemType := strings.TrimPrefix(tf.Type, "[]")
	targetElemTypeClean := g.cleanTypeName(targetElemType, config)

	// check if needs conversion (is a struct)
	if g.needsConversion(sourceElemType) {
		return fmt.Sprintf(`func() []%s {
		if src.%s == nil {
			return nil
		}
		result := make([]%s, len(src.%s))
		for i, item := range src.%s {
			converted := (&%s{}).From(&item)
			if converted != nil {
				result[i] = *converted
			}
		}
		return result
	}()`, targetElemTypeClean, sf.Name, targetElemTypeClean, sf.Name, sf.Name, targetElemTypeClean)
	}

	// direct copy for primitive/builtin slices
	if sourceElemType == targetElemType {
		return fmt.Sprintf("src.%s", sf.Name)
	}

	// casting
	return fmt.Sprintf(`func() []%s {
		result := make([]%s, len(src.%s))
		for i, v := range src.%s {
			result[i] = %s(v)
		}
		return result
	}()`, targetElemTypeClean, targetElemTypeClean, sf.Name, sf.Name, targetElemTypeClean)
}

func (g *Generator) generateReverseSliceMapping(tf, sf types.FieldInfo, config types.MappingConfig) string {
	// Reverse of slice mapping for To() method
	sourceElemType := strings.TrimPrefix(sf.Type, "[]")
	targetElemType := strings.TrimPrefix(tf.Type, "[]")

	sourcePackage := config.SourceType.PackageName

	// Check if element is a struct type
	if tf.IsNested || strings.Contains(targetElemType, ".") {
		// Extract just the type name without package
		sourceElemTypeClean := strings.TrimPrefix(sourceElemType, sourcePackage+".")

		return fmt.Sprintf(`func() []%s.%s {
		if t.%s == nil {
			return nil
		}
		result := make([]%s.%s, len(t.%s))
		for i, item := range t.%s {
			result[i] = item.To()
		}
		return result
	}()`, sourcePackage, sourceElemTypeClean, tf.Name, sourcePackage, sourceElemTypeClean, tf.Name, tf.Name)
	}

	// For primitive slices
	return fmt.Sprintf("t.%s", tf.Name)
}

func (g *Generator) generateNestedMapping(sf, tf types.FieldInfo, config types.MappingConfig) string {
	targetTypeClean := g.cleanTypeName(tf.Type, config)
	targetTypeClean = strings.TrimPrefix(targetTypeClean, "*")

	if sf.IsPointer {
		// nil check for source if pointer
		return fmt.Sprintf(`func() *%s {
		if src.%s != nil {
			return (&%s{}).From(src.%s)
		}
		return nil
	}()`, targetTypeClean, sf.Name, targetTypeClean, sf.Name)
	}

	return fmt.Sprintf(`func() %s {
		result := (&%s{}).From(&src.%s)
		if result != nil {
			return *result
		}
		return %s{}
	}()`, targetTypeClean, sf.Name, targetTypeClean, targetTypeClean)
}

func (g *Generator) generateReverseNestedMapping(tf, sf types.FieldInfo, config types.MappingConfig) string {
	// reverse nested mapping for To() method
	sourcePackage := config.SourceType.PackageName
	sourceTypeClean := g.cleanTypeName(sf.Type, config)

	typeName := strings.TrimPrefix(sourceTypeClean, sourcePackage+".")
	typeName = strings.TrimPrefix(typeName, "*")

	if tf.IsPointer {
		return fmt.Sprintf(`func() *%s.%s {
		if t.%s != nil {
			result := t.%s.To()
			return &result
		}
		return nil
	}()`, sourcePackage, typeName, tf.Name, tf.Name)
	}

	return fmt.Sprintf(`func() %s.%s {
		return t.%s.To()
	}()`, sourcePackage, typeName, tf.Name)
}

// --- Helpers ---

func (g *Generator) isPrimitiveType(typeName string) bool {
	clean := strings.TrimLeft(typeName, "*[]")

	// remove package (probably don't need this anymore)
	if idx := strings.LastIndex(clean, "."); idx != -1 {
		clean = clean[idx+1:]
	}

	primitives := map[string]bool{
		"bool":       true,
		"string":     true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"byte":       true,
		"rune":       true,
		"float32":    true,
		"float64":    true,
		"complex64":  true,
		"complex128": true,
	}

	return primitives[clean]
}

func (g *Generator) isBuiltinType(typeName string) bool {
	clean := strings.TrimLeft(typeName, "*[]")

	builtins := map[string]bool{
		"time.Time":     true,
		"time.Duration": true,
		// TODO add more
	}

	return builtins[clean]
}

func (g *Generator) needsConversion(typeName string) bool {
	return !g.isPrimitiveType(typeName) && !g.isBuiltinType(typeName)
}

func (g *Generator) zeroValue(typeStr string) string {
	switch typeStr {
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64":
		return "0"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "0"
	case "float32", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		if strings.HasPrefix(typeStr, "*") || strings.HasPrefix(typeStr, "[]") || strings.HasPrefix(typeStr, "map[") {
			return "nil"
		}
		// return empty struct literal
		return fmt.Sprintf("%s{}", typeStr)
	}
}
