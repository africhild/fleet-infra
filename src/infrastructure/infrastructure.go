package infrastructure

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type ClusterConfig struct {
	ReadWriteKey    bool     `yaml:"readWriteKey"`
	ComponentsExtra []string `yaml:"componentsExtra"`
}
type Namespace struct {
	Namespace  string        `yaml:"namespace"`
	Repository string        `yaml:"repository"`
	Branch     string        `yaml:"branch"`
	Config     ClusterConfig `yaml:"config"`
}
type Cluster struct {
	Name       string      `yaml:"name"`
	Namespaces []Namespace `yaml:"namespaces"`
}
type Config struct {
	Owner                string    `yaml:"owner"`
	ScriptPath           string    `yaml:"scriptPath"`
	Kind                 string    `yaml:"kind"`
	Provider             string    `yaml:"provider"`
	Team                 string    `yaml:"team"`
	DefaultCluster       string    `yaml:"defaultCluster"`
	DefaultNamespace     string    `yaml:"defaultNamespace"`
	Clusters             []Cluster `yaml:"clusters"`
}

type FlattenedConfig struct {
	Owner                string   `yaml:"owner"`
	Kind                 string   `yaml:"kind"`
	Provider             string   `yaml:"provider"`
	Team                 string   `yaml:"team"`
	ClusterName          string   `yaml:"clusterName"`
	Namespace            string   `yaml:"namespace"`
	Repository           string   `yaml:"repository"`
	Branch               string   `yaml:"branch"`
	ReadWriteKey         bool     `yaml:"readWriteKey"`
	ComponentsExtra      []string `yaml:"componentsExtra"`
}

func SetupInfrastructure(cofigFile string) error {
	// Read the setup file
	config, err := readConfigFile(cofigFile)
	if err != nil {
		return err
	}


	cluster, err := getConfig(config)
	if err != nil {
		return err
	}
	if config.ScriptPath == "" {
		return fmt.Errorf("scriptPath is required")
	}
	scriptPath := config.ScriptPath
	err = runSetupCommands(cluster, scriptPath)
	if err != nil {
		return err
	}
	return nil
}
func readConfigFile(configFile string) (*Config, error) {
	config := &Config{}
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		// return nil, fmt.Errorf("failed to read setup file: %v", err)
		return nil, err
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		// return nil, fmt.Errorf("failed to unmarshal setup file: %v", err)
		return nil, err
	}
	return config, nil
}
func getConfig(config *Config) (FlattenedConfig, error) {
	// validate required fields
	var flattenObj FlattenedConfig

	if config.Owner == "" {
		return flattenObj, fmt.Errorf("owner is required")
	}
	if config.DefaultCluster == "" {
		return flattenObj, fmt.Errorf("defaultCluster is required")
	}
	if config.DefaultNamespace == "" {
		return flattenObj, fmt.Errorf("defaultNamespace is required")
	}

	if config.Kind == "" {
		return flattenObj, fmt.Errorf("kind is required")
	}
	if config.Provider == "" {
		return flattenObj, fmt.Errorf("provider is required")
	}
	clusterMap := make(map[string]bool)
	// validate clusters
	for _, cluster := range config.Clusters {
		if cluster.Name == "" {
			return flattenObj, fmt.Errorf("cluster name is required")
		}
		for _, namespace := range cluster.Namespaces {
			if namespace.Namespace == "" {
				return flattenObj, fmt.Errorf("namespace is required")
			}
			if namespace.Repository == "" {
				return flattenObj, fmt.Errorf("repository is required")
			}
			if namespace.Branch == "" {
				return flattenObj, fmt.Errorf("branch is required")
			}
			// concat cluster.Name+namespace.Namespace as key
			fullName := cluster.Name + namespace.Namespace
			defaultName := config.DefaultCluster + config.DefaultNamespace
			_, ok := clusterMap[fullName];
			if ok {
				return flattenObj, fmt.Errorf("duplicate cluster name and namespace")
			}
			clusterMap[fullName] = true
			if fullName == defaultName {
				flattenObj = FlattenedConfig{
					Owner:                config.Owner,
					Kind:                 config.Kind,
					Provider:             config.Provider,
					Team:                 config.Team,
					ClusterName:          cluster.Name,
					Namespace:            namespace.Namespace,
					Repository:           namespace.Repository,
					Branch:               namespace.Branch,
					ReadWriteKey:         namespace.Config.ReadWriteKey,
					ComponentsExtra:      namespace.Config.ComponentsExtra,
				}
				break
			}
		}
	}
	return flattenObj,  nil
}

func runSetupCommands(cluster FlattenedConfig, scriptPath string) error {

	configLiterals := []string{
		"Owner=" + cluster.Owner,
		"Kind=" + cluster.Kind,
		"Provider=" + cluster.Provider,
		"Team=" + cluster.Team,
		"ClusterName=" + cluster.ClusterName,
		"Namespace=" + cluster.Namespace,
		"Repository=" + cluster.Repository,
		"Branch=" + cluster.Branch,
		"ReadWriteKey=" + fmt.Sprintf("%t", cluster.ReadWriteKey),
		"ComponentsExtra=" + strings.Join(cluster.ComponentsExtra, ","),
	}
	err := runCommand("hello", scriptPath, configLiterals)
	if err != nil {
		fmt.Println("Error running hello:", err)
	}
	err = runCommand("install_flux", scriptPath, configLiterals)
	if err != nil {
		fmt.Println("Error running instal_flux", err)
	}
	err = runCommand("bootstrap", scriptPath,  configLiterals)
	if err != nil {
		fmt.Println("Error running instal_flux", err)
	}
	err = runCommand("pull_git", scriptPath, configLiterals)
	if err != nil {
		fmt.Println("Error running pull_git", err)
	}
	return nil
}

func runCommand(functionName, scriptPath string, configLiterals []string) error {
	cmdArgs := append([]string{ scriptPath, functionName }, "")
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Env = append(os.Environ(), configLiterals...)
	// Create a pipe for the command's stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}
	// Create a pipe for the command's stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %v", err)
	}
	// Create a scanner for stdout
	outScanner := bufio.NewScanner(stdout)
	go func() {
		for outScanner.Scan() {
			fmt.Println(outScanner.Text())
		}
	}()

	errScanner := bufio.NewScanner(stderr)
	go func() {
		for errScanner.Scan() {
			fmt.Fprintln(os.Stderr, errScanner.Text())
		}
	}()
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command finished with error: %v", err)
	}

	// return cmd.Run()
	// out, err := cmd.CombinedOutput()
	// if err != nil {
	// 	return fmt.Errorf("error running command: %v, output: %s", err, string(out))
	// }
	// fmt.Printf("Output of %s:\n%s\n", functionName, out)
	return nil
}

// func parseBashFunctions(filePath string) (map[string]string, error) {
// 	file, err := os.Open(filePath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	functions := make(map[string]string)
// 	scanner := bufio.NewScanner(file)
// 	var currentFunction string
// 	var functionBody strings.Builder

// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.HasPrefix(line, "function ") || strings.HasSuffix(line, "() {") {
// 			if currentFunction != "" {
// 				functions[currentFunction] = functionBody.String()
// 				functionBody.Reset()
// 			}
// 			currentFunction = strings.TrimSuffix(strings.TrimPrefix(line, "function "), "() {")
// 			currentFunction = strings.TrimSpace(currentFunction)
// 		} else if line == "}" && currentFunction != "" {
// 			functions[currentFunction] = functionBody.String()
// 			currentFunction = ""
// 			functionBody.Reset()
// 		} else if currentFunction != "" {
// 			functionBody.WriteString(line + "\n")
// 		}
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	return functions, nil
// }

// 1. single or separate,
// 2 yaml file
// 3. what do i need in the file
// 4. run ssh command from ssh command function
