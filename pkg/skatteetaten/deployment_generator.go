package generator

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func GenerateDeployment(application skatteetaten_no_v1alpha1.Application, dbVars []corev1.EnvVar) *v1.Deployment {

	standardEnvVars := []corev1.EnvVar{
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
	standardEnvVars = append(standardEnvVars, dbVars...)

	probe := &corev1.Probe{
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

	replicas := application.Spec.Replicas.Min
	image := application.Spec.Pod.Image
	deployment := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: v1.DeploymentSpec{
			Strategy: v1.DeploymentStrategy{
				Type: "Recreate",
			},
			Replicas: pointer.Int32(int32(replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: application.StandardLabelSelector(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: application.StandardLabels(),
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{{
										Key:      "app",
										Operator: "In",
										Values:   []string{application.Name},
									}},
								},
								TopologyKey: "kubernetes.io/hostname",
							}},
						},
					},
					Containers: []corev1.Container{{
						Name:      application.Name,
						Image:     image,
						Env:       standardEnvVars,
						Resources: application.Spec.Pod.Resource,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "http",
							Protocol:      "TCP",
						}},
						StartupProbe:   probe,
						ReadinessProbe: probe,
					}},
					ServiceAccountName: application.Name,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  pointer.Int64(10000),
						RunAsGroup: pointer.Int64(30000),
						FSGroup:    pointer.Int64(20000),
					},
				},
			},
		},
	}
	return deployment
}
