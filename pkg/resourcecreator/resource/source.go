package resource

import (
	"encoding/base32"
	"encoding/binary"
	"hash/crc32"
	"strings"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/stringutil"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	metav1.Object
	metav1.ObjectMetaAccessor
	metav1.Common
	GetObjectReference() corev1.ObjectReference
	GetOwnerReference() metav1.OwnerReference
	CorrelationID() string
	SkipDeploymentMessage() bool
	LogFields() log.Fields
}

func CreateObjectMeta(source Source) metav1.ObjectMeta {
	labels := map[string]string{}

	for k, v := range source.GetLabels() {
		labels[k] = v
	}

	labels["app"] = source.GetName()

	return metav1.ObjectMeta{
		Name:      source.GetName(),
		Namespace: source.GetNamespace(),
		Labels:    labels,
		Annotations: map[string]string{
			nais_io_v1.DeploymentCorrelationIDAnnotation: source.CorrelationID(),
		},
		OwnerReferences: []metav1.OwnerReference{source.GetOwnerReference()},
	}
}

// We concatenate name, namespace and add a hash in order to avoid duplicate names when creating service accounts in common service accounts namespace.
// Also making sure to not exceed name length restrictions of 30 characters
func CreateAppNamespaceHash(source Source) string {
	name := source.GetName()
	namespace := source.GetNamespace()
	if len(name) > 11 {
		name = source.GetName()[:11]
	}
	if len(namespace) > 10 {
		namespace = source.GetNamespace()[:10]
	}
	appNameSpace := name + "-" + namespace

	checksum := crc32.ChecksumIEEE([]byte(source.GetName() + "-" + source.GetNamespace()))
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, checksum)

	return appNameSpace + "-" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bs))
}

func CreateEvent(source Source, reason, message, typeStr string) *corev1.Event {
	objectMeta := CreateObjectMeta(source)
	objectMeta.GenerateName = "naiserator-event-"
	objectMeta.Name = ""

	return &corev1.Event{
		ObjectMeta:          objectMeta,
		ReportingController: "naiserator",
		ReportingInstance:   "naiserator",
		Action:              reason,
		Reason:              reason,
		InvolvedObject:      source.GetObjectReference(),
		Source:              corev1.EventSource{Component: "naiserator"},
		Message:             stringutil.StrTrimMiddle(message, 1024),
		EventTime:           metav1.MicroTime{Time: time.Now()},
		FirstTimestamp:      metav1.Time{Time: time.Now()},
		LastTimestamp:       metav1.Time{Time: time.Now()},
		Type:                typeStr,
		Count:               1,
	}
}
