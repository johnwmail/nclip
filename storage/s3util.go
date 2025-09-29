package storage

func applyS3Prefix(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + name
}
