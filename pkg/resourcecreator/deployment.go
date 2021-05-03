package resourcecreator

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/resourceutils"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/nais/naiserator/pkg/util"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/spf13/viper"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
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
	cloudSQLProxyTermTimeout       = "30s"
)

func Deployment(app *nais.Application, resourceOptions resourceutils.Options) (*appsv1.Deployment, error) {
	spec, err := deploymentSpec(app, resourceOptions)
	if err != nil {
		return nil, err
	}

	objectMeta := app.CreateObjectMeta()
	if val, ok := app.Annotations["kubernetes.io/change-cause"]; ok {
		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}

		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objectMeta,
		Spec:       *spec,
	}, nil
}

func deploymentSpec(app *nais.Application, resourceOptions resourceutils.Options) (*appsv1.DeploymentSpec, error) {
	podSpec, err := podSpec(resourceOptions, app)
	if err != nil {
		return nil, err
	}

	var strategy appsv1.DeploymentStrategy

	if app.Spec.Strategy == nil {
		log.Error("BUG: strategy is nil; should be fixed by NilFix")
		app.Spec.Strategy = &nais.Strategy{Type: nais.DeploymentStrategyRollingUpdate}
	}

	if app.Spec.Strategy.Type == nais.DeploymentStrategyRecreate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	} else if app.Spec.Strategy.Type == nais.DeploymentStrategyRollingUpdate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(0),
				},
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(1),
				},
			},
		}
	}

	return &appsv1.DeploymentSpec{
		Replicas: util.Int32p(resourceOptions.NumReplicas),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": app.Name},
		},
		Strategy:                strategy,
		ProgressDeadlineSeconds: util.Int32p(300),
		RevisionHistoryLimit:    util.Int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: podObjectMeta(app),
			Spec:       *podSpec,
		},
	}, nil
}

func podSpec(resourceOptions resourceutils.Options, app *nais.Application) (*corev1.PodSpec, error) {
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

	if app.Spec.LeaderElection {
		podSpec = LeaderElection(app, podSpec)
	}

	if !app.Spec.SkipCaBundle {
		podSpec = caBundle(podSpec)
	}

	podSpec = filesFrom(app, podSpec, resourceOptions.NativeSecrets)

	podSpec = envFrom(app, podSpec, resourceOptions.NativeSecrets)

	if len(resourceOptions.JwkerSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.JwkerSecretName, nais.DefaultJwkerMountPath)
		if app.Spec.TokenX.Enabled && !app.Spec.TokenX.MountSecretsAsFilesOnly {
			podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.JwkerSecretName)
		}
	}

	if len(resourceOptions.AzureratorSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.AzureratorSecretName, nais.DefaultAzureratorMountPath)
		podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.AzureratorSecretName)
	}

	if len(resourceOptions.DigdiratorIDPortenSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.DigdiratorIDPortenSecretName, nais.DefaultDigdiratorIDPortenMountPath)
		podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.DigdiratorIDPortenSecretName)
	}

	if len(resourceOptions.DigdiratorMaskinportenSecretName) > 0 {
		podSpec = podSpecWithAdditionalSecret(podSpec, resourceOptions.DigdiratorMaskinportenSecretName, nais.DefaultDigdiratorMaskinportenMountPath)
		podSpec = podSpecWithAdditionalEnvFromSecret(podSpec, resourceOptions.DigdiratorMaskinportenSecretName)
	}

	if len(resourceOptions.KafkaratorSecretName) > 0 {
		podSpec = podSpecWithKafka(podSpec, resourceOptions)
	}

	if resourceOptions.Linkerd {
		podSpec = podSpecWithEnv(podSpec, corev1.EnvVar{Name: "START_WITHOUT_ENVOY", Value: "true"})
	}

	if vault.Enabled() && app.Spec.Vault.Enabled {
		podSpec, err = vaultSidecar(app, podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.WebProxy && len(resourceOptions.GoogleProjectId) == 0 {
		podSpec, err = proxyOpts(podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.SecureLogs.Enabled {
		podSpec = secureLogs(podSpec)
	}

	return podSpec, err
}

func podSpecWithEnv(spec *corev1.PodSpec, envVar corev1.EnvVar) *corev1.PodSpec {
	spec.Containers[0].Env = append(spec.Containers[0].Env, envVar)
	return spec
}

func makeKafkaSecretEnvVar(key, secretName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: key,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}

func appendGoogleSQLUserSecretEnvs(podSpec *corev1.PodSpec, app *nais.Application) *corev1.PodSpec {
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

func podSpecWithKafka(podSpec *corev1.PodSpec, resourceOptions resourceutils.Options) *corev1.PodSpec {
	// Mount specific secret keys as credential files
	credentialFilesVolume := fromFilesSecretVolume(kafkaCredentialFilesVolumeName, resourceOptions.KafkaratorSecretName, []corev1.KeyToPath{
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
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, []corev1.EnvVar{
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
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, []corev1.EnvVar{
		{
			Name:  kafkaCertificatePathKey,
			Value: filepath.Join(nais.DefaultKafkaratorMountPath, kafkaCertificateFilename),
		},
		{
			Name:  kafkaPrivateKeyPathKey,
			Value: filepath.Join(nais.DefaultKafkaratorMountPath, kafkaPrivateKeyFilename),
		},
		{
			Name:  kafkaCAPathKey,
			Value: filepath.Join(nais.DefaultKafkaratorMountPath, kafkaCAFilename),
		},
		{
			Name:  kafkaKeystorePathKey,
			Value: filepath.Join(nais.DefaultKafkaratorMountPath, kafkaKeystoreFilename),
		},
		{
			Name:  kafkaTruststorePathKey,
			Value: filepath.Join(nais.DefaultKafkaratorMountPath, kafkaTruststoreFilename),
		},
	}...)

	return podSpec
}

func hostAliases(resourceOptions resourceutils.Options) []corev1.HostAlias {
	var hostAliases []corev1.HostAlias

	for _, hostAlias := range resourceOptions.HostAliases {
		hostAliases = append(hostAliases, corev1.HostAlias{Hostnames: []string{hostAlias.Host}, IP: hostAlias.Address})
	}
	return hostAliases
}

func cloudSqlProxyContainer(sqlInstance nais.CloudSqlInstance, port int32, projectId string) corev1.Container {
	connectionName := fmt.Sprintf("%s:%s:%s", projectId, google.GoogleRegion, sqlInstance.Name)
	var runAsUser int64 = 2
	allowPrivilegeEscalation := false
	cloudSqlProxyContainerResourceSpec := nais.ResourceRequirements{
		Limits: &nais.ResourceSpec{
			Cpu:    "250m",
			Memory: "256Mi",
		},
		Requests: &nais.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}
	return corev1.Container{
		Name:            "cloudsql-proxy",
		Image:           viper.GetString(config.GoogleCloudSQLProxyContainerImage),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{{
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		}},
		Command: []string{
			"/cloud_sql_proxy",
			fmt.Sprintf("-term_timeout=%s", cloudSQLProxyTermTimeout),
			fmt.Sprintf("-instances=%s=tcp:%d", connectionName, port),
		},
		Resources: resourceLimits(cloudSqlProxyContainerResourceSpec),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsUser,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
	}
}

func envFrom(app *nais.Application, spec *corev1.PodSpec, nativeSecrets bool) *corev1.PodSpec {
	for _, env := range app.Spec.EnvFrom {
		if len(env.ConfigMap) > 0 {
			spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, fromEnvConfigmap(env.ConfigMap))
		} else if nativeSecrets && len(env.Secret) > 0 {
			spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, envFromSecret(env.Secret))
		}
	}

	return spec
}

func fromEnvConfigmap(name string) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func envFromSecret(name string) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func podSpecWithVolume(spec *corev1.PodSpec, volume corev1.Volume) *corev1.PodSpec {
	spec.Volumes = append(spec.Volumes, volume)
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, fromFilesVolumeMount(volume.Name, nais.DefaultKafkaratorMountPath, ""))
	return spec
}

func podSpecWithAdditionalSecret(spec *corev1.PodSpec, secretName, mountPath string) *corev1.PodSpec {
	spec.Volumes = append(spec.Volumes, fromFilesSecretVolume(secretName, secretName, nil))
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
		fromFilesVolumeMount(secretName, "", mountPath))
	return spec
}

func podSpecWithAdditionalEnvFromSecret(spec *corev1.PodSpec, secretName string) *corev1.PodSpec {
	spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, envFromSecret(secretName))
	return spec
}

func filesFrom(app *nais.Application, spec *corev1.PodSpec, nativeSecrets bool) *corev1.PodSpec {
	for _, file := range app.Spec.FilesFrom {
		if len(file.ConfigMap) > 0 {
			name := file.ConfigMap
			spec.Volumes = append(spec.Volumes, fromFilesConfigmapVolume(name))
			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
				fromFilesVolumeMount(name, file.MountPath, nais.GetDefaultMountPath(name)))
		} else if nativeSecrets && len(file.Secret) > 0 {
			name := file.Secret
			spec.Volumes = append(spec.Volumes, fromFilesSecretVolume(name, name, nil))
			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
				fromFilesVolumeMount(name, file.MountPath, nais.DefaultSecretMountPath))
		}
	}

	return spec
}

func fromFilesVolumeMount(name string, mountPath string, defaultMountPath string) corev1.VolumeMount {
	if len(mountPath) == 0 {
		mountPath = defaultMountPath
	}

	return corev1.VolumeMount{
		Name:      name,
		ReadOnly:  true,
		MountPath: mountPath,
	}
}

func fromFilesSecretVolume(volumeName, secretName string, items []corev1.KeyToPath) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Items:      items,
			},
		},
	}
}

func fromFilesConfigmapVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}
}
func podSpecBase(app *nais.Application) *corev1.PodSpec {
	return &corev1.PodSpec{
		Containers:         []corev1.Container{appContainer(app)},
		ServiceAccountName: app.Name,
		RestartPolicy:      corev1.RestartPolicyAlways,
		DNSPolicy:          corev1.DNSClusterFirst,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "gpr-credentials"},
			{Name: "ghcr-credentials"},
		},
	}
}

func appContainer(app *nais.Application) corev1.Container {
	c := corev1.Container{
		Name:  app.Name,
		Image: app.Spec.Image,
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(app.Spec.Port), Protocol: corev1.ProtocolTCP, Name: nais.DefaultPortName},
		},
		Resources:       resourceLimits(*app.Spec.Resources),
		ImagePullPolicy: corev1.PullIfNotPresent,
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

func secureLogs(podSpec *corev1.PodSpec) *corev1.PodSpec {
	spec := podSpec.DeepCopy()
	spec.Containers = append(spec.Containers, securelogs.FluentdSidecar())
	spec.Containers = append(spec.Containers, securelogs.ConfigmapReloadSidecar())

	spec.Volumes = append(spec.Volumes, securelogs.Volumes()...)

	volumeMount := corev1.VolumeMount{
		Name:      "secure-logs",
		MountPath: "/secure-logs",
	}
	mainContainer := spec.Containers[0].DeepCopy()
	mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, volumeMount)
	spec.Containers[0] = *mainContainer

	return spec
}

func vaultSidecar(app *nais.Application, podSpec *corev1.PodSpec) (*corev1.PodSpec, error) {
	creator, err := vault.NewVaultContainerCreator(*app)
	if err != nil {
		return nil, fmt.Errorf("while creating Vault container: %s", err)
	}
	return creator.AddVaultContainer(podSpec)
}

func defaultEnvVars(app *nais.Application) []corev1.EnvVar {
	cluster := viper.GetString(config.ClusterName)
	return []corev1.EnvVar{
		{Name: NaisAppNameEnv, Value: app.ObjectMeta.Name},
		{Name: NaisNamespaceEnv, Value: app.ObjectMeta.Namespace},
		{Name: NaisAppImageEnv, Value: app.Spec.Image},
		{Name: NaisClusterNameEnv, Value: cluster},
		{Name: NaisClientId, Value: app.ClientID(cluster)},
	}
}

// Maps environment variables from ApplicationSpec to the ones we use in PodSpec
func envVars(app *nais.Application) []corev1.EnvVar {
	newEnvVars := defaultEnvVars(app)

	for _, envVar := range app.Spec.Env {
		if envVar.ValueFrom != nil && envVar.ValueFrom.FieldRef.FieldPath != "" {
			newEnvVars = append(newEnvVars, corev1.EnvVar{
				Name: envVar.Name,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: envVar.ValueFrom.FieldRef.FieldPath},
				},
			})
		} else {
			newEnvVars = append(newEnvVars, corev1.EnvVar{Name: envVar.Name, Value: envVar.Value})
		}
	}

	return newEnvVars
}

func podObjectMeta(app *nais.Application) metav1.ObjectMeta {
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

func resourceLimits(reqs nais.ResourceRequirements) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(reqs.Requests.Cpu),
			corev1.ResourceMemory: resource.MustParse(reqs.Requests.Memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(reqs.Limits.Cpu),
			corev1.ResourceMemory: resource.MustParse(reqs.Limits.Memory),
		},
	}
}

func lifeCycle(path string) *corev1.Lifecycle {
	if len(path) > 0 {
		return &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: strings.TrimLeft(path, "/"),
					Port: intstr.FromString(nais.DefaultPortName),
				},
			},
		}
	}

	return &corev1.Lifecycle{
		PreStop: &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"sleep", "5"},
			},
		},
	}
}

func probe(app *nais.Application, probe nais.Probe) *corev1.Probe {
	port := probe.Port
	if port == 0 {
		port = app.Spec.Port
	}

	k8sprobe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
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

func GetContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for i, v := range containers {
		if v.Name == name {
			return &containers[i]
		}
	}

	return nil
}
