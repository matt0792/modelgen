package reader

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"reflect"
	"strings"

	"github.com/matt0792/modelgen/internal/types"
)

type Reader struct {
	fset    *token.FileSet
	pkgPath string
}

func NewReader(pkgPath string) *Reader {
	return &Reader{
		fset:    token.NewFileSet(),
		pkgPath: pkgPath,
	}
}

func (r *Reader) Read(structType interface{}) (*types.StructInfo, error) {
	t := reflect.TypeOf(structType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	pkgPath := t.PkgPath()
	typeName := t.Name()

	// parse source file to get ast info
	info, err := r.parseStructFromSource(pkgPath, typeName)
	if err != nil {
		return nil, err
	}

	// Add the full package import path to the struct info
	info.PackagePath = pkgPath

	return info, nil
}

func (r *Reader) parseStructFromSource(pkgPath, typeName string) (*types.StructInfo, error) {
	// 1. find package dir from packagePath
	// 2. parse all .go files in package
	// 3. find matching struct declartion
	// 4. extract field info and nested structs

	// Convert package path to directory path using go list
	dirPath, err := r.pkgPathToDir(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find directory for package %s: %w", pkgPath, err)
	}

	pkgs, err := parser.ParseDir(r.fset, dirPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			info := r.findStructInFile(file, typeName)
			if info != nil {
				return info, nil
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found", typeName)
}

func (a *Reader) findStructInFile(file *ast.File, typeName string) *types.StructInfo {
	var result *types.StructInfo

	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != typeName {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		result = &types.StructInfo{
			PackageName: file.Name.Name,
			TypeName:    typeName,
			Fields:      a.extractFields(structType),
		}

		return false
	})

	return result
}

func (r *Reader) extractFields(structType *ast.StructType) []types.FieldInfo {
	var fields []types.FieldInfo

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fieldInfo := types.FieldInfo{
				Name: name.Name,
				Type: r.exprToString(field.Type),
			}

			// type characteristics
			fieldInfo.IsPointer = r.isPointer(field.Type)
			fieldInfo.IsSlice = r.isSlice(field.Type)
			fieldInfo.IsNested = r.isStructType(field.Type)

			fields = append(fields, fieldInfo)
		}
	}

	return fields
}

func (a *Reader) exprToString(expr ast.Expr) string {
	return getTypeName(expr)
}

func (r *Reader) isPointer(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func (r *Reader) isSlice(expr ast.Expr) bool {
	_, ok := expr.(*ast.ArrayType)
	return ok
}

func (r *Reader) isStructType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.StructType:
		return true
	case *ast.Ident:
		// assume custom types are structs
		basicTypes := map[string]bool{
			"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
			"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
			"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
			"complex64": true, "complex128": true, "error": true,
		}
		return !basicTypes[t.Name]
	case *ast.SelectorExpr:
		// check if is a stdlib type
		typeName := getTypeName(t)
		stdlibTypes := map[string]bool{
			"time.Time":     true,
			"time.Duration": true,
			// TODO add more (?)
		}
		// if a stdlib type, don't treat as convertible
		if stdlibTypes[typeName] {
			return false
		}
		// if we got here, it's a custom struct type
		return true
	case *ast.StarExpr:
		// pointer - check the underlying type
		return r.isStructType(t.X)
	case *ast.ArrayType:
		// slice - check the element type
		return r.isStructType(t.Elt)
	default:
		return false
	}
}

// pkgPathToDir converts a Go package import path to its directory path
func (r *Reader) pkgPathToDir(pkgPath string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", pkgPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// --- Helpers ---

func getStructType(expr ast.Expr) (*ast.StructType, bool) {
	switch t := expr.(type) {
	case *ast.StructType:
		return t, true
	case *ast.StarExpr:
		return getStructType(t.X)
	default:
		return nil, false
	}
}

func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeName(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeName(t.Elt)
	case *ast.MapType:
		return "map[" + getTypeName(t.Key) + "]" + getTypeName(t.Value)
	case *ast.SelectorExpr:
		return getTypeName(t.X) + "." + t.Sel.Name
	case *ast.StructType:
		return "struct{...}"
	case *ast.InterfaceType:
		// empty interface or interface with methods
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + getTypeName(t.Value)
	default:
		// fallback - ideally should never get here
		return fmt.Sprintf("unknown<%T>", t)
	}
}
