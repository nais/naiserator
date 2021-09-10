package postgres_env

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func GenerateDbEnv(prefix string, secretName string) []corev1.EnvVar {

	//TODO: støtte go? 			"GO_DATASOURCE":              fmt.Sprintf("host=%s user=%s password=%s port=%d dbname=%s sslmode=require connect_timeout=30", fullyQualifiedServerName, user.Name, userPassword, azure.PostgresPort, databaseNameInAzure),
	vars := []corev1.EnvVar{
		{
			Name:  fmt.Sprintf("%s_URL", prefix),
			Value: fmt.Sprintf("jdbc:postgresql://${%s_DATABASESERVER_NAME}.postgres.database.azure.com:5432/${%s_DATABASE_NAME}?sslmode=require", prefix, prefix),
		},
		{
			Name:  fmt.Sprintf("%s_USERNAME", prefix),
			Value: fmt.Sprintf("${%s_LOCAL_USERNAME}@${%s_DATABASESERVER_NAME}", prefix, prefix),
		},
		{
			Name: fmt.Sprintf("%s_LOCAL_USERNAME", prefix),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "username",
				},
			},
		},
		{
			Name: fmt.Sprintf("%s_PASSWORD", prefix),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "password",
				},
			},
		},
		{
			Name: fmt.Sprintf("%s_DATABASESERVER_NAME", prefix),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "PSqlServerName",
				},
			},
		},
		{
			Name: fmt.Sprintf("%s_DATABASE_NAME", prefix),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "PSqlDatabaseName",
				},
			},
		},
	}
	return vars
}
