package action

func optionalFloat(args map[string]any, key string) *float64 {
	raw, ok := args[key]
	if !ok {
		return nil
	}
	val, ok := raw.(float64)
	if !ok {
		return nil
	}
	return &val
}

func floatOrDefault(args map[string]any, key string, defaultVal float64) float64 {
	ptr := optionalFloat(args, key)
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}
