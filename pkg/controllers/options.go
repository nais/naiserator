package controllers

import "sigs.k8s.io/controller-runtime/pkg/controller"

type Option func(*controller.Options)

func asControllerOptions(opts []Option) controller.Options {
	o := &controller.Options{}

	for _, opt := range opts {
		opt(o)
	}

	return *o
}

func WithMaxConcurrentReconciles(n int) Option {
	return func(o *controller.Options) {
		o.MaxConcurrentReconciles = n
	}
}
