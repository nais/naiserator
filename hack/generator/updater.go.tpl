// generated by your friendly code generator. DO NOT EDIT.
// to refresh this file, run `go generate` in your shell.

package updater

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	istioClientSet "istio.io/client-go/pkg/clientset/versioned"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typed_core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	typed_apps_v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typed_autoscaling_v1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	typed_networking_v1beta1 "k8s.io/client-go/kubernetes/typed/networking/v1beta1"
	typed_networking_v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	rbacv1 "k8s.io/api/rbac/v1"
	typed_rbac_v1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	typed_networking_istio_io_v1alpha3 "github.com/nais/naiserator/pkg/client/clientset/versioned/typed/networking.istio.io/v1alpha3"
	networking_istio_io_v1alpha3 "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	typed_iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/naiserator/pkg/client/clientset/versioned/typed/iam.cnrm.cloud.google.com/v1beta1"
	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	typed_storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/naiserator/pkg/client/clientset/versioned/typed/storage.cnrm.cloud.google.com/v1beta1"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	typed_sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/naiserator/pkg/client/clientset/versioned/typed/sql.cnrm.cloud.google.com/v1beta1"
	istio_security_v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	typed_istio_security_v1beta1 "istio.io/client-go/pkg/clientset/versioned/typed/security/v1beta1"
)

{{range .}}

func {{.Name}}(client {{.Interface}}, old, new {{.Type}}) func() error {
	log.Infof("creating or updating {{ .Type }} for %s", new.Name)
	if old == nil {
		return func() error {
			_, err := client.Create(new)
			if err != nil {
				return fmt.Errorf("%s: %s", "{{ .Name }}", err)
			}
			return err
		}
	}

	CopyMeta(old, new)
	{{if .TransformFunc}}
		{{.TransformFunc}}(old, new)
	{{end}}

	return func() error {
		_, err := client.Update(new)
		if err != nil {
			return fmt.Errorf("%s: %s", "{{ .Name }}", err)
		}
		return err
	}
}

{{end}}

func CreateOrUpdate(clientSet kubernetes.Interface, customClient clientV1Alpha1.Interface, istioClient istioClientSet.Interface, resource runtime.Object) func() error {
	switch new := resource.(type) {
	{{range .}}
		case {{.Type}}:
		c := {{.Client}}(new.Namespace)
		old, err := c.Get(new.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return func() error { return fmt.Errorf("%s: %s", "{{ .Name }}", err) }
			}
			return {{.Name}}(c, nil, new)
		}
		return {{.Name}}(c, old, new)
	{{end}}
	default:
		panic(fmt.Errorf("BUG! You didn't specify a case for type '%T' in the file hack/generator/updater.go", new))
	}
}

func CreateOrRecreate(clientSet kubernetes.Interface, customClient clientV1Alpha1.Interface, istioClient istioClientSet.Interface, resource runtime.Object) func() error {
	switch new := resource.(type) {
	{{range .}}
		case {{.Type}}:
		c := {{.Client}}(new.Namespace)
		return func() error {
			log.Infof("pre-deleting {{ .Type }} for %s", new.Name)
			err := c.Delete(new.Name, &metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("%s: %s", "{{ .Name }}", err)
			}
			log.Infof("creating new {{ .Type }} for %s", new.Name)
			_, err = c.Create(new)
			if err != nil {
				return fmt.Errorf("%s: %s", "{{ .Name }}", err)
			} else {
				return nil
			}
		}
	{{end}}
	default:
		panic(fmt.Errorf("BUG! You didn't specify a case for type '%T' in the file hack/generator/updater.go", new))
	}
}

func CreateIfNotExists(clientSet kubernetes.Interface, customClient clientV1Alpha1.Interface, istioClient istioClientSet.Interface, resource runtime.Object) func() error {
	switch new := resource.(type) {
	{{range .}}
		case {{.Type}}:
		c := {{.Client}}(new.Namespace)
		return func() error {
			log.Infof("creating new {{ .Type }} for %s", new.Name)
			_, err := c.Create(new)
			if err != nil && !errors.IsAlreadyExists(err) {
				return fmt.Errorf("%s: %s", "{{ .Name }}", err)
			}
			return nil
		}
	{{end}}
	default:
		panic(fmt.Errorf("BUG! You didn't specify a case for type '%T' in the file hack/generator/updater.go", new))
	}
}

func FindAll(clientSet kubernetes.Interface, customClient clientV1Alpha1.Interface, istioClient istioClientSet.Interface, name, namespace string) ([]runtime.Object, error) {
	resources := make([]runtime.Object, 0)

	{{range .}}
	{
		c := {{.Client}}(namespace)
		existing, err := c.List(metav1.ListOptions{LabelSelector: "app=" + name})
		if err != nil && !errors.IsNotFound(err) {
			return nil, fmt.Errorf("discover %s: %s", "{{.Type}}", err)
		} else if existing != nil {
			items, err := meta.ExtractList(existing)
			if err != nil {
				return nil, fmt.Errorf("extract list of %s: %s", "{{.Type}}", err)
			}
			resources = append(resources, items...)
        }
	}
	{{end}}

	return resources, nil
}

func DeleteIfExists(clientSet kubernetes.Interface, customClient clientV1Alpha1.Interface, istioClient istioClientSet.Interface, resource runtime.Object) func() error {
	switch new := resource.(type) {
	{{range .}}
		case {{.Type}}:
		c := {{.Client}}(new.Namespace)
		return func() error {
			log.Infof("deleting {{ .Type }} for %s", new.Name)
			err := c.Delete(new.Name, &metav1.DeleteOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("%s: %s", "{{ .Name }}", err)
			}

			return err
		}
	{{end}}
	default:
		panic(fmt.Errorf("BUG! You didn't specify a case for type '%T' in the file hack/generator/updater.go", new))
	}
}
