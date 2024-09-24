package application

import (
	"fmt"
)

// Template constants
const (
	BaseDeploymentTmpl = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.Name}}
  template:
    metadata:
      labels:
        app: {{.Name}}
    spec:
      containers:
      - name: {{.Name}}
        ports:
        - containerPort: {{.Port}}
`

	BaseServiceTmpl = `
apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
  namespace: default
spec:
  selector:
    app: {{.Name}}
  ports:
  - protocol: TCP
    port: 80
    targetPort: {{.Port}}
  type: ClusterIP
`

BaseKustomizationTmpl = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
- service.yaml
`

DeploymentTmpl = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
spec:
  replicas: {{.Replicas}}
  selector:
    matchLabels:
      app: {{.Name}}
  template:
    spec:
      containers:
      - name: {{.Name}}
        image: {{.Image}}
      imagePullSecrets:
        - name: registry-secret
`

KustomizationTmpl = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {{.Namespace}}
resources:
- ../../../../../base/{{.Name}}
patches:
  - path: deployment.yaml
`
NamespaseTmpl = `
apiVersion: v1
kind: Namespace
metadata: 
  name: {{.Namespace}}
`
IngressKustomizationTmpl = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {{.Namespace}}
resources:
- ingress.yaml
`
IngressTmpl = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{.Namespace}}-ingress
  namespace: {{.Namespace}}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules: []
`
)

// ResourceType represents the type of resource
type ResourceType int

// ResourceType constants
const (
	Base ResourceType = iota
	Application
	Common
)

// ResourceInfo contains information about a resource
type ResourceInfo struct {
	//Path string
	Name string
}

// resources maps ResourceType to ResourceInfo
var resources = map[ResourceType]ResourceInfo{
	Base:        {Name: "Base"},
	Application: {Name: "Application"},
	Common:      {Name: "Common"},
}

// Path returns the path for a given ResourceType
//func (r ResourceType) Path() string {
//	if resource, ok := resources[r]; ok {
//		return resource.Path
//	}
//	return "unknown"
//}

// String returns the string representation of a ResourceType
func (r ResourceType) String() string {
	if resource, ok := resources[r]; ok {
		return resource.Name
	}
	return "unknown"
}

// Template represents a template configuration
type Template struct {
	Name    string
	Content string
	Type    string
	//Path    string
}

// Templates is a slice of Template configurations
var Templates = []Template{
	{
		Name:    "deployment",
		Content: BaseDeploymentTmpl,
		Type:    Base.String(),
		//Path:    Base.Path(),
	},
	{
		Name:    "service",
		Content: BaseServiceTmpl,
		Type:    Base.String(),
		//Path:    Base.Path(),
	},
	{
		Name:    "kustomization",
		Content: BaseKustomizationTmpl,
		Type:    Base.String(),
		//Path:    Base.Path(),
	},
	{
		Name:    "deployment",
		Content: DeploymentTmpl,
		Type:    Application.String(),
		//Path:    Application.Path(),
	},
	{
		Name:    "kustomization",
		Content: KustomizationTmpl,
		Type:    Application.String(),
		//Path:    Application.Path(),
	},
	{	Name: "namespace",
		Content: NamespaseTmpl,
		Type: Common.String(),
	},
	{   Name: "common/kustomization",
		Content: IngressKustomizationTmpl,
		Type: Common.String(),
	},
	{   Name: "common/ingress",
		Content: IngressTmpl,
		Type: Common.String(),
	},
}

// ValidateTemplates checks if all templates are valid
func ValidateTemplates() error {
	for _, tmpl := range Templates {
		if tmpl.Name == "" || tmpl.Content == "" || tmpl.Type == "" {
			return fmt.Errorf("invalid template configuration: %+v", tmpl)
		}
	}
	return nil
}
