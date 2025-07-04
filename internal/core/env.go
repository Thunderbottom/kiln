package core

import "github.com/joho/godotenv"

// ParseEnv parses environment file content
func ParseEnv(data []byte) (map[string][]byte, error) {
	stringVars, err := godotenv.Unmarshal(string(data))
	if err != nil {
		return nil, err
	}

	// Convert to []byte for kiln's secure handling
	vars := make(map[string][]byte, len(stringVars))

	for key, value := range stringVars {
		valueBytes := []byte(value)
		vars[key] = valueBytes
	}

	return vars, nil
}

// FormatEnv formats environment variables
func FormatEnv(vars map[string][]byte) []byte {
	if len(vars) == 0 {
		return []byte{}
	}

	stringVars := make(map[string]string, len(vars))
	for key, value := range vars {
		stringVars[key] = string(value)
	}

	content, err := godotenv.Marshal(stringVars)
	if err != nil {
		return []byte{}
	}

	return []byte(content)
}
