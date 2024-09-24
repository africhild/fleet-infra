package main

import (
	// "flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/africhild/fleet-infra/src/application"
	"github.com/africhild/fleet-infra/src/common"
	"github.com/africhild/fleet-infra/src/config"
	"github.com/africhild/fleet-infra/src/infrastructure"
	"github.com/africhild/fleet-infra/src/ingress"
	"github.com/africhild/fleet-infra/src/secret"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "fleet"}

	var newSetupCmd = &cobra.Command{
		Use:   "setup:new",
		Short: "Setup new infrastructure",
		Run:   newSetup,
	}
	// a flag for enum values(single cluster for all environments, separate clusters for each environment)
	// newSetupCmd.Flags().StringP("cluster_to_env", "c2e", "single", "Cluster to environment mapping (single|separate)")
	newSetupCmd.Flags().StringP("file", "f", "", "Path to the setup file")

	var genSecretCmd = &cobra.Command{
		Use:   "secret:create",
		Short: "Generate and seal a Kubernetes secret",
		Run:   genSecret,
	}
	genSecretCmd.Flags().StringP("env", "e", "", "Environment (staging|production)")
	genSecretCmd.Flags().StringP("file", "f", "", "Path to the .env file")
	genSecretCmd.Flags().StringP("app", "a", "", "Application name")
	genSecretCmd.MarkFlagRequired("env")
	genSecretCmd.MarkFlagRequired("file")
	genSecretCmd.MarkFlagRequired("app")

	var createNewAppCmd = &cobra.Command{
		Use:   "app:create",
		Short: "Create a new application",
		Run:   createNewApp,
	}
	createNewAppCmd.Flags().StringP("app", "a", "", "Application name")
	createNewAppCmd.Flags().StringP("env", "e", "", "Environment (staging|production)")
	createNewAppCmd.Flags().IntP("port", "p", 80, "Port")
	createNewAppCmd.Flags().IntP("replicas", "r", 1, "Number of replicas")
	createNewAppCmd.MarkFlagRequired("app")
	createNewAppCmd.MarkFlagRequired("env")
	createNewAppCmd.MarkFlagRequired("port")

	var updateIngressCmd = &cobra.Command{
		Use:   "ingress",
		Short: "Update the ingress",
		Run:   updateIngress,
	}
	updateIngressCmd.Flags().StringP("env", "e", "", "Environment (staging|production)")
	updateIngressCmd.Flags().StringP("app", "a", "", "Application name")
	updateIngressCmd.Flags().StringP("subdomain", "s", "", "Subdomain")
	// add or remove flag
	updateIngressCmd.Flags().BoolP("add", "", false, "Add to ingress")
	updateIngressCmd.Flags().BoolP("remove", "", false, "Remove from ingress")
	updateIngressCmd.MarkFlagRequired("env")
	updateIngressCmd.MarkFlagRequired("app")
	updateIngressCmd.MarkFlagRequired("subdomain")

	rootCmd.AddCommand(genSecretCmd, createNewAppCmd, updateIngressCmd, newSetupCmd)
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("Error executing command:", err)
		os.Exit(1)

	}
}

func updateIngress(cmd *cobra.Command, args []string) {
	env, _ := cmd.Flags().GetString("env")
	appName, _ := cmd.Flags().GetString("app")
	add, _ := cmd.Flags().GetBool("add")
	remove, _ := cmd.Flags().GetBool("remove")
	subdomain, _ := cmd.Flags().GetString("subdomain")
	if add == remove {
		fmt.Println("Specify either --add or --remove")
		os.Exit(1)
	}
	addStatus := add == true

	err := ingress.ManageIngressRule(env, appName, subdomain, addStatus)
	if err != nil {
		fmt.Println("Error updating ingress:", err)
		os.Exit(1)
	}
	fmt.Println("Ingress successfully updated")
}

func createNewApp(cmd *cobra.Command, args []string) {
	appName, _ := cmd.Flags().GetString("app")
	env, _ := cmd.Flags().GetString("env")
	port, _ := cmd.Flags().GetInt("port")
	replicas, _ := cmd.Flags().GetInt("replicas")

	fleet_app_path := filepath.Join(config.AppTemplatePath, env)
	fmt.Println("Creating new app:", fleet_app_path)
	application := application.App{
		Name:      appName,
		Namespace: env,
		Env:       env,
		Port:      port,
		ImageHost: config.ImageHost,
		Image:     fmt.Sprintf("%s/%s:latest", config.ImageHost, appName),
		Replicas:  replicas,
		Templates: application.Templates,
	}
	err := application.Create(fleet_app_path)
	if err != nil {
		fmt.Println("Error creating app:", err)
		os.Exit(1)
	}
	fmt.Println("App successfully created:", appName)
}

func genSecret(cmd *cobra.Command, args []string) {
	env, _ := cmd.Flags().GetString("env")
	envFile, _ := cmd.Flags().GetString("file")
	appName, _ := cmd.Flags().GetString("app")
	fleet_app_path := filepath.Join(config.AppTemplatePath, env, appName)
	envMap, err := common.ParseEnvFile(envFile, false)
	if err != nil {
		fmt.Println("Error reading .env file:", err)
		os.Exit(1)
	}

	// Create Kubernetes Secret YAML
	secretYaml := secret.CreateSecretYaml(appName, env, envMap)
	secretFileName := fmt.Sprintf("%s.%s.secret.yaml", appName, env)
	err = ioutil.WriteFile(secretFileName, []byte(secretYaml), 0644)
	if err != nil {
		fmt.Println("Error writing secret file:", err)
		os.Exit(1)
	}

	// Seal the secret using kubeseal
	// sealedSecretFileName := fmt.Sprintf("sealed.%s.%s.secret.yaml", appName, env)
	sealedSecretFileName := "sealed-secret.yaml"
	err = secret.SealSecret(fleet_app_path, secretFileName, sealedSecretFileName)
	if err != nil {
		fmt.Println("Error sealing secret:", err)
		os.Exit(1)
	}
	// Update the kustomization.yaml file
	kustomizationFile := filepath.Join(fleet_app_path, "kustomization.yaml")
	err = secret.AddSealedSecretToKustomization(sealedSecretFileName, kustomizationFile)
	if err != nil {
		fmt.Println("Error updating kustomization.yaml:", err)
		os.Exit(1)
	}
	// Update the deployment.yaml file with secret keys
	deploymentFile := filepath.Join(fleet_app_path, "deployment.yaml")
	//    sealedSecretFile := filepath.Join(fleet_app_path, "secrets", sealedSecretFileName)
	err = secret.AddSecretKeysToDeployment(secretFileName, deploymentFile, envFile)
	if err != nil {
		fmt.Println("Error updating deployment.yaml:", err)
		os.Exit(1)
	}
	err = common.DeleteFile(secretFileName)
	if err != nil {
		fmt.Printf("Error deleting file: %s", secretFileName)
	}
	err = common.DeleteFile(envFile)
	if err != nil {
		fmt.Printf("Error deleting file: %s", secretFileName)
	}

	fmt.Println("Secret successfully created and sealed:", sealedSecretFileName)
}

func newSetup(cmd *cobra.Command, args []string) {
	// clusterToEnv, _ := cmd.Flags().GetString("cluster_to_env")
	setupFile, _ := cmd.Flags().GetString("file")
	// if clusterToEnv != "single" && clusterToEnv != "separate" {
	// 	fmt.Println("Invalid cluster to environment mapping. Use 'single' or 'separate'")
	// 	os.Exit(1)
	// }
	if setupFile == "" {
		fmt.Println("Specify the config file")
		os.Exit(1)
	}
	err := infrastructure.SetupInfrastructure(setupFile)
	if err != nil {
		fmt.Println("Error setting up infrastructure:", err)
		os.Exit(1)
	} else {
		fmt.Println("Infrastructure successfully setup")
	}
}
