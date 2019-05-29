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
	Client        string // Function to call on the clientset to get a REST client
}

var i = []Service{
	{
		Name:          "service",
		Interface:     "typed_core_v1.ServiceInterface",
		Type:          "*corev1.Service",
		TransformFunc: "CopyService",
		Client:        "clientSet.CoreV1().Services",
	},
	{
		Name:      "serviceAccount",
		Interface: "typed_core_v1.ServiceAccountInterface",
		Type:      "*corev1.ServiceAccount",
		Client:    "clientSet.CoreV1().ServiceAccounts",
	},
	{
		Name:      "deployment",
		Interface: "typed_apps_v1.DeploymentInterface",
		Type:      "*appsv1.Deployment",
		Client:    "clientSet.AppsV1().Deployments",
	},
	{
		Name:      "ingress",
		Interface: "typed_extensions_v1beta1.IngressInterface",
		Type:      "*extensionsv1beta1.Ingress",
		Client:    "clientSet.ExtensionsV1beta1().Ingresses",
	},
	{
		Name:      "horizontalPodAutoscaler",
		Interface: "typed_autoscaling_v1.HorizontalPodAutoscalerInterface",
		Type:      "*autoscalingv1.HorizontalPodAutoscaler",
		Client:    "clientSet.AutoscalingV1().HorizontalPodAutoscalers",
	},
	{
		Name:      "networkPolicy",
		Interface: "typed_networking_v1.NetworkPolicyInterface",
		Type:      "*networkingv1.NetworkPolicy",
		Client:    "clientSet.NetworkingV1().NetworkPolicies",
	},
	{
		Name:      "serviceRole",
		Interface: "istio_v1alpha1.ServiceRoleInterface",
		Type:      "*v1alpha1.ServiceRole",
		Client:    "customClient.RbacV1alpha1().ServiceRoles",
	},
	{
		Name:      "serviceRoleBinding",
		Interface: "istio_v1alpha1.ServiceRoleBindingInterface",
		Type:      "*v1alpha1.ServiceRoleBinding",
		Client:    "customClient.RbacV1alpha1().ServiceRoleBindings",
	},
	{
		Name:      "virtualService",
		Interface: "typed_networking_istio_io_v1alpha3.VirtualServiceInterface",
		Type:      "*networking_istio_io_v1alpha3.VirtualService",
		Client:    "customClient.NetworkingV1alpha3().VirtualServices",
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
