package utilspackage utils


import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateSlug creates a random alphanumeric slug of the specified length
func GenerateSlug(length int) (string, error) {
	if length <= 0 {
		length = 5
	}
	
	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))
	
	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		result[i] = charset[randomIndex.Int64()]
	}
	
	return string(result), nil
}

// IsValidSlug checks if a slug contains only valid characters
func IsValidSlug(slug string) bool {
	if len(slug) == 0 {
		return false
	}
	
	for _, char := range slug {
		valid := false
		for _, validChar := range charset {
			if char == validChar {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}
	
	return true
}