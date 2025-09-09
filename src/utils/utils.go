package utils

func Keys(m map[string]string) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func In(slice []string, element string) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}
	return false
}
