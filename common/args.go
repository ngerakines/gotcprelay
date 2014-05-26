package common

func ArgOrDefault(key, defaultValue string, arguments map[string]interface{}) string {
	rawValue, hasValue := arguments[key]
	if hasValue {
		stringValue, isString := rawValue.(string)
		if isString && len(stringValue) > 0 {
			return stringValue
		}
	}
	return defaultValue
}

