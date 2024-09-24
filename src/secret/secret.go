package secret

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/africhild/fleet-infra/src/common"
	"gopkg.in/yaml.v2"
)

// addSealedSecretToKustomization adds the sealed secret file to the kustomization.yaml file
func AddSealedSecretToKustomization(sealedSecretFileName, kustomizationFile string) error {
	file, err := os.OpenFile(kustomizationFile, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Check if the sealed secret is already in the resources
	for _, line := range lines {
		if strings.TrimSpace(line) == "- "+sealedSecretFileName {
			return nil // Already present, no need to add
		}
	}

	// Add the sealed secret to the resources section
	var buffer bytes.Buffer
	for _, line := range lines {
		buffer.WriteString(line + "\n")
		if strings.TrimSpace(line) == "resources:" {
			buffer.WriteString("- " + sealedSecretFileName + "\n")
		}
	}

	// Write the updated content back to the file
	err = ioutil.WriteFile(kustomizationFile, buffer.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

// addSecretKeysToDeployment adds the secret keys to the deployment.yaml file
func AddSecretKeysToDeployment(secretFileName, deploymentFile, envFile string) error {
	file, err := os.OpenFile(deploymentFile, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var deployment map[string]interface{}
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&deployment); err != nil {
		return err
	}

	// Prepare the secret keys to be added
	envMap, err := common.ParseEnvFile(envFile, true)
	if err != nil {
		return err
	}

	var envVars []map[string]interface{}
	secretFileName = strings.TrimSuffix(secretFileName, ".yaml")
	for key := range envMap {
		envVar := map[string]interface{}{
			"name": key,
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": secretFileName,
					"key":  key,
				},
			},
		}
		envVars = append(envVars, envVar)
	}

	// Traverse to the containers section in the deployment
	spec, ok := deployment["spec"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("invalid deployment.yaml structure")
	}
	template, ok := spec["template"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("invalid deployment.yaml structure")
	}
	templateSpec, ok := template["spec"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("invalid deployment.yaml structure")
	}
	containers, ok := templateSpec["containers"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid deployment.yaml structure")
	}

	// Add the env vars to the first container (assuming there's at least one container)
	firstContainer, ok := containers[0].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("invalid deployment.yaml structure")
	}
	firstContainer["env"] = envVars

	// Write the updated deployment back to the file
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	if err := encoder.Encode(deployment); err != nil {
		return err
	}
	encoder.Close()

	err = ioutil.WriteFile(deploymentFile, buffer.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

// createSecretYaml generates a Kubernetes Secret YAML string
func CreateSecretYaml(appName, env string, envMap map[string]string) string {
	var buffer bytes.Buffer
	buffer.WriteString("apiVersion: v1\n")
	buffer.WriteString("kind: Secret\n")
	buffer.WriteString(fmt.Sprintf("metadata:\n  name: %s.%s.secret\n  namespace: %s\n", appName, env, env))
	buffer.WriteString("type: Opaque\n")
	buffer.WriteString("data:\n")
	for key, value := range envMap {
		buffer.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
	}
	return buffer.String()
}

// sealSecret seals a Kubernetes Secret YAML file using kubeseal
func SealSecret(base_path, inputFile, outputFile string) error {
	cmd := exec.Command("kubeseal", "--format", "yaml", "--cert", "./pub-sealed-secrets.pem")
	input, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer input.Close()
	outputName, err := common.MakeDir(base_path, outputFile)
	if err != nil {
		return err
	}
	output, err := os.Create(outputName)
	if err != nil {
		return err
	}
	defer output.Close()

	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	// delete staging0secret.yaml
	return cmd.Run()
}
