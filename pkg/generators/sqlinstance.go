package generators

import (
	"context"
	"fmt"

	"github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func prepareSqlInstance(ctx context.Context, kube client.Client, key client.ObjectKey, o *Options) error {
	sqlinstance := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	err := kube.Get(ctx, key, sqlinstance)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("query existing sqlinstance: %s", err)
		}
		o.SqlInstance.exists = false
	} else {
		o.SqlInstance.exists = true
		o.SqlInstance.hasPrivateIp = sqlinstance.Spec.Settings.IpConfiguration.PrivateNetworkRef != nil
	}
	return nil
}
