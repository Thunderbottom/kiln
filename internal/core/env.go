package core

import "github.com/joho/godotenv"

// ParseEnvData parses environment file content from bytes and returns variables
func ParseEnvData(data []byte) (map[string][]byte, error) {
	stringVars, err := godotenv.Unmarshal(string(data))
	if err != nil {
		return nil, err
	}

	// Convert to []byte values for secure handling
	vars := make(map[string][]byte)
	for key, value := range stringVars {
		vars[key] = []byte(value)
	}

	return vars, nil
}

// ParseEnvFile parses environment file content from string and returns string variables
// Kept for backwards compatibility where string handling is required
func ParseEnvFile(content string) (map[string]string, error) {
	return godotenv.Unmarshal(content)
}

// FormatEnvData formats environment variables from []byte back to file format
func FormatEnvData(vars map[string][]byte) []byte {
	stringVars := make(map[string]string)
	for key, value := range vars {
		stringVars[key] = string(value)
	}

	content, err := godotenv.Marshal(stringVars)
	if err != nil {
		return nil
	}
	return []byte(content)
}

// FormatEnvFile formats environment variables from strings back to file format
// Kept for backwards compatibility
func FormatEnvFile(vars map[string]string) string {
	content, err := godotenv.Marshal(vars)
	if err != nil {
		return ""
	}
	return content
}
