package core

import "github.com/joho/godotenv"

// ParseEnv parses environment file content from bytes and returns variables
func ParseEnv(data []byte) (map[string][]byte, error) {
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

// FormatEnv formats environment variables from []byte back to file format
func FormatEnv(vars map[string][]byte) []byte {
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
