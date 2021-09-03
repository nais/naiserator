package generator

import (
	"encoding/json"
	"fmt"
	azurev1alpha1 "github.com/Azure/azure-service-operator/api/v1alpha1"
	azurev1alpha2 "github.com/Azure/azure-service-operator/api/v1alpha2"
	imagev1beta1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1beta12 "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"testing"
)

var PostgresServer = azurev1alpha2.PostgreSQLServer{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pgs-orion-utv-postgres",
		Namespace: "orion-utv",
	},
	Spec: azurev1alpha2.PostgreSQLServerSpec{
		ResourceGroup: "rg-orion",
	},
}

func AssertEqualToFixtures(t *testing.T, obj runtime.Object, fixtureFile string) {
	jsonString, err := json.MarshalIndent(obj, "", "  ")
	assert.NoError(t, err)

	dat, err := ioutil.ReadFile(fmt.Sprintf("../../fixtures/%s", fixtureFile))
	assert.NoError(t, err)
	assert.Equal(t, string(dat), string(jsonString), fixtureFile)
}

var Scheme = runtime.NewScheme()

func TestApp(name string) *v1alpha1.Application {

	app := &v1alpha1.Application{}
	dat, err := ioutil.ReadFile(fmt.Sprintf("../../fixtures/%s", name))
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(dat, &app)
	if err != nil {
		panic(err)
	}
	return app
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(imagev1beta1.AddToScheme(Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(Scheme))
	utilruntime.Must(networkingv1beta1.AddToScheme(Scheme))
	utilruntime.Must(azurev1alpha1.AddToScheme(Scheme))
	utilruntime.Must(azurev1alpha2.AddToScheme(Scheme))
	utilruntime.Must(v1beta12.AddToScheme(Scheme))
}
