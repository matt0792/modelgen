package mapper

import "github.com/matt0792/modelgen/internal/types"

type FieldMatcher struct{}

func (m *FieldMatcher) MatchFields(source, target *types.StructInfo) map[string]string {
	matches := make(map[string]string)
	targetFields := make(map[string]bool)

	for _, tf := range target.Fields {
		targetFields[tf.Name] = true
	}

	for _, sf := range source.Fields {
		if targetFields[sf.Name] {
			matches[sf.Name] = sf.Name
		}
	}

	return matches
}

func (m *FieldMatcher) NeedsRecursiveMapping(sourceField, targetField types.FieldInfo) bool {
	// if both are struct types aith the same name, we need to generate recursive mapping
	return sourceField.IsNested && targetField.IsNested
}
