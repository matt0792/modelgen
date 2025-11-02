package types

type FieldInfo struct {
	Name      string
	Type      string
	IsOmitted bool
	IsNested  bool
	IsSlice   bool
	IsPointer bool
}
