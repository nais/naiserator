package deployment_generator

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)


func Create(source resource.Source,ast *resource.Ast, app skatteetaten_no_v1alpha1.ApplicationSpec,dbVars []corev1.EnvVar) {
	generateDeployment(source, ast,app, dbVars)
}

func generateDeployment(source resource.Source,ast *resource.Ast, app skatteetaten_no_v1alpha1.ApplicationSpec,dbVars []corev1.EnvVar){

	standardEnvVars := generateStandardEnv()
	standardEnvVars = append(standardEnvVars, dbVars...)

	probe := getProbe()
	deployment := getDeploymentSpec(source, app, standardEnvVars, probe)
	ast.AppendOperation(resource.OperationCreateOrUpdate, deployment)

}

func generateStandardEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}
}

func getProbe() *corev1.Probe{
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.IntOrString{
					IntVal: 8080,
				},
			},
		},
		FailureThreshold: 30,
		PeriodSeconds:    10,
	}
}

func getDeploymentSpec(source resource.Source, appSpec skatteetaten_no_v1alpha1.ApplicationSpec, standardEnvVars []corev1.EnvVar, probe *corev1.Probe) *v1.Deployment {
	replicas := appSpec.Replicas.Min
	image := appSpec.Pod.Image

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: v1.DeploymentSpec{
			Strategy: v1.DeploymentStrategy{
				Type: "Recreate",
			},
			Replicas: pointer.Int32Ptr(int32(*replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: source.GetLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: source.GetLabels(),
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{{
										Key:      "app",
										Operator: "In",
										Values:   []string{source.GetName()},
									}},
								},
								TopologyKey: "kubernetes.io/hostname",
							}},
						},
					},
					Containers: []corev1.Container{{
						Name:      source.GetName(),
						Image:     image,
						Env:       standardEnvVars,
						Resources: appSpec.Pod.Resource,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "http",
							Protocol:      "TCP",
						}},
						StartupProbe:   probe,
						ReadinessProbe: probe,
					}},
					ServiceAccountName: source.GetName(),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  pointer.Int64Ptr(10000),
						RunAsGroup: pointer.Int64Ptr(30000),
						FSGroup:    pointer.Int64Ptr(20000),
					},
				},
			},
		},
	}

}