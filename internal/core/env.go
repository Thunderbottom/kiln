package core

import "github.com/joho/godotenv"

// ParseEnvFile parses environment file content and returns variables
func ParseEnvFile(content string) (map[string]string, error) {
	return godotenv.Unmarshal(content)
}

// FormatEnvFile formats environment variables back to file format
func FormatEnvFile(vars map[string]string) string {
	content, err := godotenv.Marshal(vars)
	if err != nil {
		return ""
	}
	return content
}
