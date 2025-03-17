package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func mapImageToApplicationOrNaisjob(_ context.Context, object client.Object) []controllerruntime.Request {
	req := controllerruntime.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	return []controllerruntime.Request{req}
}
