// Package informer keeps up with changes in Kubernetes and reports back when an application is changed.
package informer

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	nais_informers "github.com/nais/naiserator/pkg/client/informers/externalversions"
	nais_informers_typed "github.com/nais/naiserator/pkg/client/informers/externalversions/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

type Synchronizer interface {
	Enqueue(*nais.Application)
}

type Informer struct {
	stopChannel     chan struct{}
	informer        nais_informers_typed.ApplicationInformer
	synchronizer    Synchronizer
	informerFactory nais_informers.SharedInformerFactory
}

func New(synchronizer Synchronizer, sharedInformerFactory nais_informers.SharedInformerFactory) *Informer {
	i := &Informer{
		synchronizer:    synchronizer,
		informerFactory: sharedInformerFactory,
		stopChannel:     make(chan struct{}, 1),
	}

	sharedInformerFactory.Nais().V1alpha1().Applications().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(newPod interface{}) {
				i.watchCallback(newPod)
			},
			UpdateFunc: func(oldPod, newPod interface{}) {
				i.watchCallback(newPod)
			},
		},
	)

	return i
}

// This function is passed to the Application resource watcher.
// It will be called once for every application resource, which
// must be type cast from interface{} to Application.
func (informer *Informer) watchCallback(unstructured interface{}) {
	var app *nais.Application
	var ok bool

	if unstructured == nil {
		return
	}

	app, ok = unstructured.(*nais.Application)
	if !ok {
		// type cast failed; discard
		log.Errorf("watchCallback encountered invalid Application resource of type %T", unstructured)
		return
	}

	informer.synchronizer.Enqueue(app)
}

func (informer *Informer) Stop() {
	informer.stopChannel <- struct{}{}
}

func (informer *Informer) Run() error {
	log.Info("Starting application informer")

	informer.informerFactory.Start(informer.stopChannel)

	i := informer.informerFactory.Nais().V1alpha1().Applications().Informer()
	if !cache.WaitForCacheSync(informer.stopChannel, i.HasSynced) {
		return fmt.Errorf("timed out waiting for cache sync")
	}

	log.Infof("Cache has been populated")

	return nil
}
