package ingress

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/africhild/fleet-infra/src/common"
	"github.com/africhild/fleet-infra/src/config"
	"gopkg.in/yaml.v2"
)

type IngressYAML struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		IngressClassName string `yaml:"ingressClassName"`
		Rules            []struct {
			Host string `yaml:"host"`
			Http struct {
				Paths []struct {
					Path     string `yaml:"path"`
					PathType string `yaml:"pathType"`
					Backend  struct {
						Service struct {
							Name string `yaml:"name"`
							Port struct {
								Number int `yaml:"number"`
							} `yaml:"port"`
						} `yaml:"service"`
					} `yaml:"backend"`
				} `yaml:"paths"`
			} `yaml:"http"`
		} `yaml:"rules"`
	} `yaml:"spec"`
}

func ManageIngressRule(env, serviceName, subdomain string, add bool) error {
	ingressPath := filepath.Join(config.AppTemplatePath, env, "common", "ingress.yaml")
	// Check if the ingress file exists
	fileStatus, err := common.CheckFileExists(ingressPath)
	var host string
	if subdomain == "@" {
		host = config.UrlSuffix
	} else {
		host = fmt.Sprintf("%s.%s", subdomain, config.UrlSuffix)
	}
	if err != nil {
		fmt.Println("Error checking file:", err)
		os.Exit(1)
	}
	if !fileStatus {
		fmt.Println("Ingress file does not exist:", ingressPath)
		os.Exit(1)
	}
	// Read the YAML file
	data, err := ioutil.ReadFile(ingressPath)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	var ingress IngressYAML
	err = yaml.Unmarshal(data, &ingress)
	if err != nil {
		return fmt.Errorf("error unmarshaling YAML: %v", err)
	}

	// Check if the rule already exists
	ruleExists := false
	ruleIndex := -1
	for i, rule := range ingress.Spec.Rules {
		if strings.HasPrefix(rule.Host, subdomain+".") {
			ruleExists = true
			ruleIndex = i
			break
		}
	}

	if add {
		if ruleExists {
			// Rule already exists, do nothing
			fmt.Printf("Rule for %s already exists\n", serviceName)
			return nil
		}
		// Add new rule
		newRule := struct {
			Host string `yaml:"host"`
			Http struct {
				Paths []struct {
					Path     string `yaml:"path"`
					PathType string `yaml:"pathType"`
					Backend  struct {
						Service struct {
							Name string `yaml:"name"`
							Port struct {
								Number int `yaml:"number"`
							} `yaml:"port"`
						} `yaml:"service"`
					} `yaml:"backend"`
				} `yaml:"paths"`
			} `yaml:"http"`
		}{
			Host: host,
		}
		newRule.Http.Paths = []struct {
			Path     string `yaml:"path"`
			PathType string `yaml:"pathType"`
			Backend  struct {
				Service struct {
					Name string `yaml:"name"`
					Port struct {
						Number int `yaml:"number"`
					} `yaml:"port"`
				} `yaml:"service"`
			} `yaml:"backend"`
		}{
			{
				Path:     "/",
				PathType: "Prefix",
				Backend: struct {
					Service struct {
						Name string `yaml:"name"`
						Port struct {
							Number int `yaml:"number"`
						} `yaml:"port"`
					} `yaml:"service"`
				}{
					Service: struct {
						Name string `yaml:"name"`
						Port struct {
							Number int `yaml:"number"`
						} `yaml:"port"`
					}{
						Name: serviceName,
						Port: struct {
							Number int `yaml:"number"`
						}{Number: 80},
					},
				},
			},
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, newRule)
		fmt.Printf("Added rule for %s\n", serviceName)
	} else {
		if !ruleExists {
			// Rule doesn't exist, do nothing
			fmt.Printf("Rule for %s doesn't exist\n", serviceName)
			return nil
		}
		// Remove the rule
		ingress.Spec.Rules = append(ingress.Spec.Rules[:ruleIndex], ingress.Spec.Rules[ruleIndex+1:]...)
		fmt.Printf("Removed rule for %s\n", serviceName)
	}

	// Marshal the updated struct back to YAML
	updatedData, err := yaml.Marshal(&ingress)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %v", err)
	}

	// Write the updated YAML back to the file
	err = ioutil.WriteFile(ingressPath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}
