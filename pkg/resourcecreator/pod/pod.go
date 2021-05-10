package pod

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/spf13/viper"
	"k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	cloudSQLProxyTermTimeout       = "30s"
	kafkaCredentialFilesVolumeName = "kafka-credentials"
	kafkaCertificatePathKey        = "KAFKA_CERTIFICATE_PATH"
	kafkaPrivateKeyPathKey         = "KAFKA_PRIVATE_KEY_PATH"
	kafkaCAPathKey                 = "KAFKA_CA_PATH"
	kafkaKeystorePathKey           = "KAFKA_KEYSTORE_PATH"
	kafkaTruststorePathKey         = "KAFKA_TRUSTSTORE_PATH"
	kafkaCertificateKey            = "KAFKA_CERTIFICATE"
	kafkaPrivateKeyKey             = "KAFKA_PRIVATE_KEY"
	kafkaCAKey                     = "KAFKA_CA"
	kafkaBrokersKey                = "KAFKA_BROKERS"
	kafkaSchemaRegistryKey         = "KAFKA_SCHEMA_REGISTRY"
	kafkaSchemaRegistryUserKey     = "KAFKA_SCHEMA_REGISTRY_USER"
	kafkaSchemaRegistryPasswordKey = "KAFKA_SCHEMA_REGISTRY_PASSWORD"
	kafkaCredStorePasswordKey      = "KAFKA_CREDSTORE_PASSWORD"
	kafkaCertificateFilename       = "kafka.crt"
	kafkaPrivateKeyFilename        = "kafka.key"
	kafkaCAFilename                = "ca.crt"
	kafkaKeystoreFilename          = "client.keystore.p12"
	kafkaTruststoreFilename        = "client.truststore.jks"
	naisAppNameEnv                 = "NAIS_APP_NAME"
	naisNamespaceEnv               = "NAIS_NAMESPACE"
	naisAppImageEnv                = "NAIS_APP_IMAGE"
	naisClusterNameEnv             = "NAIS_CLUSTER_NAME"
	naisClientId                   = "NAIS_CLIENT_ID"
)

func Spec(resourceOptions resource.Options, app *nais_io_v1alpha1.Application) (*v1.PodSpec, error) {
	var err error

	podSpec := podSpecBase(app)

	if app.Spec.GCP != nil && app.Spec.GCP.SqlInstances != nil {
		podSpec = appendGoogleSQLUserSecretEnvs(podSpec, app)
		for _, instance := range app.Spec.GCP.SqlInstances {
			podSpec.Containers = append(podSpec.Containers, cloudSqlProxyContainer(instance, 5432, resourceOptions.GoogleTeamProjectId))
		}
	}

	if len(resourceOptions.HostAliases) > 0 {
		podSpec.HostAliases = hostAliases(resourceOptions)
	}

	if !app.Spec.SkipCaBundle {
		podSpec = certificateauthority.CABundle(podSpec)
	}

	podSpec = filesFrom(app, podSpec, resourceOptions.NativeSecrets)

	podSpec = envFrom(app, podSpec, resourceOptions.NativeSecrets)

	if len(resourceOptions.JwkerSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.JwkerSecretName, nais_io_v1alpha1.DefaultJwkerMountPath)
		if app.Spec.TokenX.Enabled && !app.Spec.TokenX.MountSecretsAsFilesOnly {
			podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.JwkerSecretName)
		}
	}

	if len(resourceOptions.AzureratorSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.AzureratorSecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
		podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.AzureratorSecretName)
	}

	if len(resourceOptions.DigdiratorIDPortenSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.DigdiratorIDPortenSecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
		podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.DigdiratorIDPortenSecretName)
	}

	if len(resourceOptions.DigdiratorMaskinportenSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.DigdiratorMaskinportenSecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
		podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.DigdiratorMaskinportenSecretName)
	}

	if len(resourceOptions.KafkaratorSecretName) > 0 {
		podSpec = podSpecWithKafka(podSpec, resourceOptions)
	}

	if vault.Enabled() && app.Spec.Vault.Enabled {
		podSpec, err = vaultSidecar(app, podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.WebProxy && len(resourceOptions.GoogleProjectId) == 0 {
		podSpec, err = proxyopts.ProxyOpts(podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.SecureLogs.Enabled {
		podSpec = secureLogs(podSpec)
	}

	return podSpec, err
}

func makeKafkaSecretEnvVar(key, secretName string) v1.EnvVar {
	return v1.EnvVar{
		Name: key,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}

func appendGoogleSQLUserSecretEnvs(podSpec *v1.PodSpec, app *nais_io_v1alpha1.Application) *v1.PodSpec {
	for _, instance := range app.Spec.GCP.SqlInstances {
		for _, db := range instance.Databases {
			googleSQLUsers := google_sql.MergeAndFilterSQLUsers(db.Users, instance.Name)
			for _, user := range googleSQLUsers {
				podSpec.Containers[0].EnvFrom = append(podSpec.Containers[0].EnvFrom, envFromSecret(google_sql.GoogleSQLSecretName(app, instance.Name, user.Name)))
			}
		}
	}
	return podSpec
}

func podSpecWithKafka(podSpec *v1.PodSpec, resourceOptions resource.Options) *v1.PodSpec {
	// Mount specific secret keys as credential files
	credentialFilesVolume := fromFilesSecretVolume(kafkaCredentialFilesVolumeName, resourceOptions.KafkaratorSecretName, []v1.KeyToPath{
		{
			Key:  kafkaCertificateKey,
			Path: kafkaCertificateFilename,
		},
		{
			Key:  kafkaPrivateKeyKey,
			Path: kafkaPrivateKeyFilename,
		},
		{
			Key:  kafkaCAKey,
			Path: kafkaCAFilename,
		},
		{
			Key:  kafkaKeystoreFilename,
			Path: kafkaKeystoreFilename,
		},
		{
			Key:  kafkaTruststoreFilename,
			Path: kafkaTruststoreFilename,
		},
	})
	podSpec = podSpecWithVolume(podSpec, credentialFilesVolume)

	// Add environment variables for string data
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, []v1.EnvVar{
		makeKafkaSecretEnvVar(kafkaCertificateKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaPrivateKeyKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaBrokersKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaSchemaRegistryKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaSchemaRegistryUserKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaSchemaRegistryPasswordKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaCAKey, resourceOptions.KafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaCredStorePasswordKey, resourceOptions.KafkaratorSecretName),
	}...)

	// Inject path environment variables to refer to mounted secrets
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, []v1.EnvVar{
		{
			Name:  kafkaCertificatePathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaCertificateFilename),
		},
		{
			Name:  kafkaPrivateKeyPathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaPrivateKeyFilename),
		},
		{
			Name:  kafkaCAPathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaCAFilename),
		},
		{
			Name:  kafkaKeystorePathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaKeystoreFilename),
		},
		{
			Name:  kafkaTruststorePathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaTruststoreFilename),
		},
	}...)

	return podSpec
}

func hostAliases(resourceOptions resource.Options) []v1.HostAlias {
	var hostAliases []v1.HostAlias

	for _, hostAlias := range resourceOptions.HostAliases {
		hostAliases = append(hostAliases, v1.HostAlias{Hostnames: []string{hostAlias.Host}, IP: hostAlias.Address})
	}
	return hostAliases
}

func cloudSqlProxyContainer(sqlInstance nais_io_v1alpha1.CloudSqlInstance, port int32, projectId string) v1.Container {
	connectionName := fmt.Sprintf("%s:%s:%s", projectId, google.GoogleRegion, sqlInstance.Name)
	var runAsUser int64 = 2
	allowPrivilegeEscalation := false
	cloudSqlProxyContainerResourceSpec := nais_io_v1alpha1.ResourceRequirements{
		Limits: &nais_io_v1alpha1.ResourceSpec{
			Cpu:    "250m",
			Memory: "256Mi",
		},
		Requests: &nais_io_v1alpha1.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}
	return v1.Container{
		Name:            "cloudsql-proxy",
		Image:           viper.GetString(config.GoogleCloudSQLProxyContainerImage),
		ImagePullPolicy: v1.PullIfNotPresent,
		Ports: []v1.ContainerPort{{
			ContainerPort: port,
			Protocol:      v1.ProtocolTCP,
		}},
		Command: []string{
			"/cloud_sql_proxy",
			fmt.Sprintf("-term_timeout=%s", cloudSQLProxyTermTimeout),
			fmt.Sprintf("-instances=%s=tcp:%d", connectionName, port),
		},
		Resources: resourceLimits(cloudSqlProxyContainerResourceSpec),
		SecurityContext: &v1.SecurityContext{
			RunAsUser:                &runAsUser,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
	}
}

func envFrom(app *nais_io_v1alpha1.Application, spec *v1.PodSpec, nativeSecrets bool) *v1.PodSpec {
	for _, env := range app.Spec.EnvFrom {
		if len(env.ConfigMap) > 0 {
			spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, fromEnvConfigmap(env.ConfigMap))
		} else if nativeSecrets && len(env.Secret) > 0 {
			spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, envFromSecret(env.Secret))
		}
	}

	return spec
}

func fromEnvConfigmap(name string) v1.EnvFromSource {
	return v1.EnvFromSource{
		ConfigMapRef: &v1.ConfigMapEnvSource{
			LocalObjectReference: v1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func envFromSecret(name string) v1.EnvFromSource {
	return v1.EnvFromSource{
		SecretRef: &v1.SecretEnvSource{
			LocalObjectReference: v1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func podSpecWithVolume(spec *v1.PodSpec, volume v1.Volume) *v1.PodSpec {
	spec.Volumes = append(spec.Volumes, volume)
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, fromFilesVolumeMount(volume.Name, nais_io_v1alpha1.DefaultKafkaratorMountPath, ""))
	return spec
}

func podSpecWithAdditionalSecret(spec *v1.PodSpec, secretName, mountPath string) *v1.PodSpec {
	spec.Volumes = append(spec.Volumes, fromFilesSecretVolume(secretName, secretName, nil))
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
		fromFilesVolumeMount(secretName, "", mountPath))
	return spec
}

func podSpecWithAdditionalEnvFromSecret(spec *v1.PodSpec, secretName string) *v1.PodSpec {
	spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, envFromSecret(secretName))
	return spec
}

func filesFrom(app *nais_io_v1alpha1.Application, spec *v1.PodSpec, nativeSecrets bool) *v1.PodSpec {
	for _, file := range app.Spec.FilesFrom {
		if len(file.ConfigMap) > 0 {
			name := file.ConfigMap
			spec.Volumes = append(spec.Volumes, fromFilesConfigmapVolume(name))
			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
				fromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultMountPath(name)))
		} else if nativeSecrets && len(file.Secret) > 0 {
			name := file.Secret
			spec.Volumes = append(spec.Volumes, fromFilesSecretVolume(name, name, nil))
			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
				fromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.DefaultSecretMountPath))
		}
	}

	return spec
}

func fromFilesVolumeMount(name string, mountPath string, defaultMountPath string) v1.VolumeMount {
	if len(mountPath) == 0 {
		mountPath = defaultMountPath
	}

	return v1.VolumeMount{
		Name:      name,
		ReadOnly:  true,
		MountPath: mountPath,
	}
}

func fromFilesSecretVolume(volumeName, secretName string, items []v1.KeyToPath) v1.Volume {
	return v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
				Items:      items,
			},
		},
	}
}

func fromFilesConfigmapVolume(name string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}
}
func podSpecBase(app *nais_io_v1alpha1.Application) *v1.PodSpec {
	return &v1.PodSpec{
		Containers:         []v1.Container{appContainer(app)},
		ServiceAccountName: app.Name,
		RestartPolicy:      v1.RestartPolicyAlways,
		DNSPolicy:          v1.DNSClusterFirst,
		ImagePullSecrets: []v1.LocalObjectReference{
			{Name: "gpr-credentials"},
			{Name: "ghcr-credentials"},
		},
	}
}

func appContainer(app *nais_io_v1alpha1.Application) v1.Container {
	c := v1.Container{
		Name:  app.Name,
		Image: app.Spec.Image,
		Ports: []v1.ContainerPort{
			{ContainerPort: int32(app.Spec.Port), Protocol: v1.ProtocolTCP, Name: nais_io_v1alpha1.DefaultPortName},
		},
		Resources:       resourceLimits(*app.Spec.Resources),
		ImagePullPolicy: v1.PullIfNotPresent,
		Lifecycle:       lifeCycle(app.Spec.PreStopHookPath),
		Env:             envVars(app),
	}

	if app.Spec.Liveness != nil && len(app.Spec.Liveness.Path) > 0 {
		c.LivenessProbe = probe(app, *app.Spec.Liveness)
	}

	if app.Spec.Readiness != nil && len(app.Spec.Readiness.Path) > 0 {
		c.ReadinessProbe = probe(app, *app.Spec.Readiness)
	}

	if app.Spec.Startup != nil && len(app.Spec.Startup.Path) > 0 {
		c.StartupProbe = probe(app, *app.Spec.Startup)
	}

	return c
}

func secureLogs(podSpec *v1.PodSpec) *v1.PodSpec {
	spec := podSpec.DeepCopy()
	spec.Containers = append(spec.Containers, securelogs.FluentdSidecar())
	spec.Containers = append(spec.Containers, securelogs.ConfigmapReloadSidecar())

	spec.Volumes = append(spec.Volumes, securelogs.Volumes()...)

	volumeMount := v1.VolumeMount{
		Name:      "secure-logs",
		MountPath: "/secure-logs",
	}
	mainContainer := spec.Containers[0].DeepCopy()
	mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, volumeMount)
	spec.Containers[0] = *mainContainer

	return spec
}

func vaultSidecar(app *nais_io_v1alpha1.Application, podSpec *v1.PodSpec) (*v1.PodSpec, error) {
	creator, err := vault.NewVaultContainerCreator(*app)
	if err != nil {
		return nil, fmt.Errorf("while creating Vault container: %s", err)
	}
	return creator.AddVaultContainer(podSpec)
}

func defaultEnvVars(app *nais_io_v1alpha1.Application) []v1.EnvVar {
	cluster := viper.GetString(config.ClusterName)
	return []v1.EnvVar{
		{Name: naisAppNameEnv, Value: app.ObjectMeta.Name},
		{Name: naisNamespaceEnv, Value: app.ObjectMeta.Namespace},
		{Name: naisAppImageEnv, Value: app.Spec.Image},
		{Name: naisClusterNameEnv, Value: cluster},
		{Name: naisClientId, Value: app.ClientID(cluster)},
	}
}

// Maps environment variables from ApplicationSpec to the ones we use in Spec
func envVars(app *nais_io_v1alpha1.Application) []v1.EnvVar {
	newEnvVars := defaultEnvVars(app)

	for _, envVar := range app.Spec.Env {
		if envVar.ValueFrom != nil && envVar.ValueFrom.FieldRef.FieldPath != "" {
			newEnvVars = append(newEnvVars, v1.EnvVar{
				Name: envVar.Name,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{FieldPath: envVar.ValueFrom.FieldRef.FieldPath},
				},
			})
		} else {
			newEnvVars = append(newEnvVars, v1.EnvVar{Name: envVar.Name, Value: envVar.Value})
		}
	}

	return newEnvVars
}

func ObjectMeta(app *nais_io_v1alpha1.Application) v12.ObjectMeta {
	objectMeta := app.CreateObjectMeta()

	port := app.Spec.Prometheus.Port
	if len(port) == 0 {
		port = strconv.Itoa(app.Spec.Port)
	}

	objectMeta.Annotations = map[string]string{}
	if app.Spec.Prometheus.Enabled {
		objectMeta.Annotations["prometheus.io/scrape"] = "true"
		objectMeta.Annotations["prometheus.io/port"] = port
		objectMeta.Annotations["prometheus.io/path"] = app.Spec.Prometheus.Path
	}

	if len(app.Spec.Logformat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = app.Spec.Logformat
	}

	if len(app.Spec.Logtransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = app.Spec.Logtransform
	}

	if app.Spec.Elastic != nil {
		objectMeta.Labels["elastic"] = "enabled"
	}

	if app.Spec.Kafka != nil {
		objectMeta.Labels["kafka"] = "enabled"
	}

	return objectMeta
}

func resourceLimits(reqs nais_io_v1alpha1.ResourceRequirements) v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    k8sResource.MustParse(reqs.Requests.Cpu),
			v1.ResourceMemory: k8sResource.MustParse(reqs.Requests.Memory),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    k8sResource.MustParse(reqs.Limits.Cpu),
			v1.ResourceMemory: k8sResource.MustParse(reqs.Limits.Memory),
		},
	}
}

func lifeCycle(path string) *v1.Lifecycle {
	if len(path) > 0 {
		return &v1.Lifecycle{
			PreStop: &v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path: strings.TrimLeft(path, "/"),
					Port: intstr.FromString(nais_io_v1alpha1.DefaultPortName),
				},
			},
		}
	}

	return &v1.Lifecycle{
		PreStop: &v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{"sleep", "5"},
			},
		},
	}
}

func probe(app *nais_io_v1alpha1.Application, probe nais_io_v1alpha1.Probe) *v1.Probe {
	port := probe.Port
	if port == 0 {
		port = app.Spec.Port
	}

	k8sprobe := &v1.Probe{
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: leadingSlash(probe.Path),
				Port: intstr.FromInt(port),
			},
		},
		InitialDelaySeconds: int32(probe.InitialDelay),
		PeriodSeconds:       int32(probe.PeriodSeconds),
		FailureThreshold:    int32(probe.FailureThreshold),
		TimeoutSeconds:      int32(probe.Timeout),
	}

	if probe.Port != 0 {
		k8sprobe.Handler.HTTPGet.Port = intstr.FromInt(probe.Port)
	}

	return k8sprobe
}

func leadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func GetContainerByName(containers []v1.Container, name string) *v1.Container {
	for i, v := range containers {
		if v.Name == name {
			return &containers[i]
		}
	}

	return nil
}
