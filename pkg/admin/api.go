package admin

func DerefOrZero[T any](a *T) T {
	if a == nil {
		var aZero T
		return aZero
	}
	return *a
}
