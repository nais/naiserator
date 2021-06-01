package resource

import "k8s.io/apimachinery/pkg/apis/meta/v1"

type Source interface {
	v1.Object
	CreateObjectMeta() v1.ObjectMeta
	CreateObjectMetaWithName(string) v1.ObjectMeta
	CreateAppNamespaceHash() string
}
