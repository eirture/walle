package utils

func InStringArray(v string, vs []string) bool {
	for _, i := range vs {
		if i == v {
			return true
		}
	}
	return false
}
