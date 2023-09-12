package controllers

import "sigs.k8s.io/controller-runtime/pkg/controller"

func options(opts []func(*controller.Options)) controller.Options {
	o := &controller.Options{}

	for _, opt := range opts {
		opt(o)
	}

	return *o
}

func WithMaxConcurrentReconciles(n int) func(*controller.Options) {
	return func(o *controller.Options) {
		o.MaxConcurrentReconciles = n
	}
}
