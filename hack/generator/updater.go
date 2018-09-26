package main

import (
	"os"
	"text/template"
)

//go:generate go run updater.go

type Service struct {
	Name          string // Resource type.
	Interface     string // Kubernetes client interface for using REST.
	Type          string // Real type of the object being modified.
	TransformFunc string // Optional name of function that will be called to copy the old object into the new.
	ClientType    string // Function to call on the clientset to get a REST client
}

var i = []Service{
	{
		Name:          "service",
		Interface:     "typed_core_v1.ServiceInterface",
		Type:          "*corev1.Service",
		TransformFunc: "CopyService",
		ClientType:    "CoreV1().Services",
	},
	{
		Name:       "serviceAccount",
		Interface:  "typed_core_v1.ServiceAccountInterface",
		Type:       "*corev1.ServiceAccount",
		ClientType: "CoreV1().ServiceAccounts",
	},
	{
		Name:       "deployment",
		Interface:  "typed_apps_v1.DeploymentInterface",
		Type:       "*appsv1.Deployment",
		ClientType: "AppsV1().Deployments",
	},
	{
		Name:       "ingress",
		Interface:  "typed_extensions_v1beta1.IngressInterface",
		Type:       "*extensionsv1beta1.Ingress",
		ClientType: "ExtensionsV1beta1().Ingresses",
	},
	{
		Name:       "horizontalPodAutoscaler",
		Interface:  "typed_autoscaling_v1.HorizontalPodAutoscalerInterface",
		Type:       "*autoscalingv1.HorizontalPodAutoscaler",
		ClientType: "AutoscalingV1().HorizontalPodAutoscalers",
	},
}

func main() {
	t, err := template.ParseFiles("updater.go.tpl")
	if err != nil {
		panic(err)
	}
	err = t.Execute(os.Stdout, i)
	if err != nil {
		panic(err)
	}
}
