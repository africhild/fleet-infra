package application

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/africhild/fleet-infra/src/common"
	"github.com/africhild/fleet-infra/src/config"
	"github.com/africhild/fleet-infra/src/storage"
	"github.com/sirupsen/logrus"
)

// App represents an application configuration
type App struct {
	Name      string
	Namespace string // out
	Env       string // out
	Port      int
	ImageHost string // ghcr.io or docker.io
	Image     string
	Templates []Template
	Replicas  int
}

var (
	basePath = config.BaseTemplatePath
	log      = logrus.New()
)

func init() {
	// Configure logging
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
}

// Create creates a new application directory
func (a *App) Create(appPath string) error {
	log.WithFields(logrus.Fields{
		"appPath": appPath,
		"appName": a.Name,
		"env":     a.Env,
		"port":    a.Port,
		"replica": a.Replicas,
	}).Info("Creating new application")
	_basePath := filepath.Join(basePath, a.Name)
	if err := common.EnsureDirectoryExists(_basePath); err != nil {
		return fmt.Errorf("failed to create base path: %w", err)
	}
	_appPath := filepath.Join(appPath, a.Name)
	if err := common.EnsureDirectoryExists(_appPath); err != nil {
		return fmt.Errorf("failed to create app path: %w", err)
	}
	_commonPath := filepath.Join(appPath, "common")
	if err := common.EnsureDirectoryExists(_commonPath); err != nil {
		return fmt.Errorf("failed to create common path: %w", err)
	}

	//if _, err := os.Stat(appPath); !os.IsNotExist(err) {
	//	return fmt.Errorf("application already exists at %s", appPath)
	//}
	//
	//if err := os.MkdirAll(appPath, 0755); err != nil {
	//	return fmt.Errorf("failed to create application directory: %w", err)
	//}

	return a.createYAML(appPath)
}

func (a *App) createYAML(appPath string) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(a.Templates))

	for _, tmpl := range a.Templates {
		wg.Add(1)
		go func(tmpl Template) {
			defer wg.Done()
			if err := a.createFile(tmpl, appPath); err != nil {
				errCh <- fmt.Errorf("error creating file for template %s: %w", tmpl.Name, err)
			}
		}(tmpl)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
			//log.WithError(err).Error("Error creating file")
		}
	}

	return nil
}

func (a *App) createFile(tmpl Template, appPath string) error {
	var tempFile string
	switch tmpl.Type {
	case "Base":
		tempFile = common.GetPath(filepath.Join(basePath, a.Name), tmpl.Name)
	case "Common":
		tempFile = common.GetPath(appPath, tmpl.Name)
	case "Application":
		tempFile = common.GetPath(filepath.Join(appPath, a.Name), tmpl.Name)
	default:
		return fmt.Errorf("invalid template type: %s", tmpl.Type)
	}

	fileExist, err := common.CheckFileExists(tempFile)
	if err != nil {
		return fmt.Errorf("error checking file %s: %w", tempFile, err)
	}
	if !fileExist {
		// remove lines with # or // from the tmpl.content
		// Check if the port is in use
		if tmpl.Type == "Base" && (tmpl.Name == "deployment" || tmpl.Name == "service") {
			if storage.IsPortUsed(a.Port) {
				return fmt.Errorf("port %d is already in use", a.Port)
			}
		}
		err := common.RemoveComments(&tmpl.Content)
		if err != nil {
			return fmt.Errorf("error removing comments from template %s: %w", tmpl.Name, err)
		}
		newTmpl, err := template.New(tmpl.Name).Parse(tmpl.Content)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", tmpl.Name, err)
		}

		file, err := os.Create(tempFile)
		if err != nil {
			return fmt.Errorf("error creating file %s: %w", tempFile, err)
		}
		defer file.Close()

		if err := newTmpl.Execute(file, a); err != nil {
			return fmt.Errorf("error executing template %s: %w", tmpl.Name, err)
		}
		if tmpl.Type == "Base" && (tmpl.Name == "deployment" || tmpl.Name == "service") {
			storage.AddPort(a.Name, a.Port)
		}
		log.WithField("file", tempFile).Info("File created successfully")
	}
	return nil
}
