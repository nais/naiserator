package generators

import (
	"context"
	"fmt"
	"strings"

	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func prepareSqlInstance(ctx context.Context, source resource.Source, kube client.Client, o *Options) error {
	if !o.IsGCPEnabled() || source.GetGCP() == nil {
		return nil
	}

	gcpSpec := source.GetGCP()
	if len(gcpSpec.SqlInstances) != 1 || !o.Config.Features.SqlInstanceInSharedVpc {
		return nil
	}

	instanceName := source.GetName()
	if len(gcpSpec.SqlInstances[0].Name) > 0 {
		instanceName = gcpSpec.SqlInstances[0].Name
	}

	sqlInstanceKey := client.ObjectKey{
		Name:      instanceName,
		Namespace: source.GetNamespace(),
	}

	sqlinstance := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	err := kube.Get(ctx, sqlInstanceKey, sqlinstance)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("query existing sqlinstance: %s", err)
		}
		o.SqlInstance.exists = false
	} else {
		o.SqlInstance.exists = true
		o.SqlInstance.hasPrivateIpInSharedVpc = hasPrivateIpInSharedVpc(sqlinstance, o)
	}
	return nil
}

func hasPrivateIpInSharedVpc(sqlinstance *sql_cnrm_cloud_google_com_v1beta1.SQLInstance, o *Options) bool {
	privateNetworkRef := sqlinstance.Spec.Settings.IpConfiguration.PrivateNetworkRef
	if privateNetworkRef == nil {
		return false
	}
	return strings.Contains(privateNetworkRef.External, o.GetGoogleProjectID())
}
