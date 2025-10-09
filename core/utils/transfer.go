package utils

func ConvertGBToBytes(gb int64) int64 {
	// 1 GB = 1024 * 1024 * 1024 bytes
	return gb * 1024 * 1024 * 1024
}

func ConvertBytesToGB(bytes int64) int64 {
	// 1 GB = 1024 * 1024 * 1024 bytes
	return bytes / (1024 * 1024 * 1024)
}
