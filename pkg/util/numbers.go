package util

//go:fix inline
func Int32p(i int32) *int32 {
	return new(i)
}

//go:fix inline
func Intp(i int) *int {
	return new(i)
}
