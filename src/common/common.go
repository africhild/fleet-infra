package common

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

// parseEnvFile reads an .env file and returns a map of key-value pairs
func ParseEnvFile(envFile string, skipValue bool) (map[string]string, error) {
	file, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "---" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line in env file: %s", line)
		}
		key := strings.TrimSpace(parts[0])
		if skipValue {
			envMap[key] = ""
		} else {
			value := strings.TrimSpace(parts[1])
			envMap[key] = base64.StdEncoding.EncodeToString([]byte(value))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return envMap, nil
}

func RemoveComments(content *string) error {
    lines := strings.Split(*content, "\n")
    var newContent []string
    for _, line := range lines {
        trimmedLine := strings.TrimSpace(line)
        if !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "//") && trimmedLine != "" {
            newContent = append(newContent, line)
        }
    }
    *content = strings.Join(newContent, "\n")
    return nil
}

func MakeDir(first_path, second_path string) (string, error) {
	// appName, _ := cmd.Flags().GetString("app")
	// env, _ := cmd.Flags().GetString("env")

	// sealedSecretFileName := fmt.Sprintf("sealed.%s.%s.secret.yaml", appName, env)

	err := os.MkdirAll(first_path, 0755)
	if err != nil {
		fmt.Println("Error creating output directory:", err)
		return "", err
	}

	// err = os.Rename(sealedSecretFileName, filepath.Join(outputDir, sealedSecretFileName))
	// if err != nil {
	// 	fmt.Println("Error moving sealed secret file:", err)
	// 	return "", err
	// }
	first_path = filepath.Join(first_path, second_path)

	fmt.Println("Sealed secret successfully deployed to:", first_path)
	return first_path, nil
}
func EnsureDirectoryExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		// Path already exists
		return nil
	}
	if os.IsNotExist(err) {
		// Create the directory with permissions set to 0755
		return os.MkdirAll(path, 0755)
	}
	// Some other error occurred
	return err
}

func CheckFileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil || os.IsExist(err) {
		return true, nil
		//return fmt.Errorf("file does exist: %s", filePath)
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetPath(base, longFile string) string {
    fileComponents := strings.Split(longFile, "/")
    if len(fileComponents) > 0 {
        lastIndex := len(fileComponents) - 1
        fileComponents[lastIndex] += ".yaml" // Append .yaml to the last component
    } else {
        return filepath.Join(base, ".yaml") // Handle the case where longFile is empty
    }
    return filepath.Join(base, filepath.Join(fileComponents...))
}

