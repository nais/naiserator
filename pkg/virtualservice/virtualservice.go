package virtualservice

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/nais/naiserator/pkg/util"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/naiserator/pkg/naiserator/config"
)

const IstioVirtualServiceTotalWeight int32 = 100 // The total weight of all routes must equal 100

const regexSuffix = "(/.*)?"

const ServiceSuffix = ".svc.cluster.local"

type Gateway string

type Route struct {
	Gateway Gateway
	Route   networking_istio_io_v1alpha3.HTTPRoute
}

type RouteMap map[url.URL]networking_istio_io_v1alpha3.HTTPRoute

type Registry struct {
	routes    RouteMap
	mappings  []config.GatewayMapping
	gateways  map[string][]string
	namespace string
}

func (r *Registry) All() []*networking_istio_io_v1alpha3.VirtualService {
	services := make([]*networking_istio_io_v1alpha3.VirtualService, 0)
	for host := range r.gateways {
		services = append(services, r.VirtualService(host))
	}
	return services
}

func (r *Registry) Populate(ctx context.Context, client client.Client) error {
	log.Infof("Building virtual service registry...")

	timer := time.Now()
	errors := 0

	appList := &nais_io_v1alpha1.ApplicationList{}
	err := client.List(ctx, appList)

	if err != nil {
		return fmt.Errorf("get all applications: %w", err)
	}

	for _, app := range appList.Items {
		if err = nais_io_v1alpha1.ApplyDefaults(&app); err != nil {
			return fmt.Errorf("BUG: merge default values into application: %s", err)
		}
		err = r.Add(&app)
		if err != nil {
			log.WithFields(app.LogFields()).Errorf("unable to add to virtual service registry: %s", err)
			errors++
		}
	}
	log.Infof("Built virtual service registry with %d errors in %s", errors, time.Now().Sub(timer).String())
	services := r.All()
	log.Infof("Virtual service registry has %d URLs across %d domains", len(r.routes), len(services))

	return nil
}

func (r *Registry) VirtualServices(app *nais_io_v1alpha1.Application) ([]*networking_istio_io_v1alpha3.VirtualService, error) {
	services := make([]*networking_istio_io_v1alpha3.VirtualService, 0)
	hostSet := make(map[string]interface{})
	for _, ingress := range app.Spec.Ingresses {
		ingressUrl, err := url.Parse(string(ingress))
		if err != nil {
			continue
		}
		hostSet[ingressUrl.Host] = new(interface{})
	}

	for host := range hostSet {
		services = append(services, r.VirtualService(host))
	}
	return services, nil
}

func NewRegistry(gatewayMapping []config.GatewayMapping, namespace string) *Registry {
	return &Registry{
		routes:    make(RouteMap),
		mappings:  gatewayMapping,
		gateways:  make(map[string][]string),
		namespace: namespace,
	}
}

func (r *Registry) ResolveAndCacheGateway(host string) []string {
	if gateways, ok := r.gateways[host]; ok {
		return gateways
	}
	for _, mapping := range r.mappings {
		if strings.HasSuffix(host, mapping.DomainSuffix) {
			r.gateways[host] = strings.Split(mapping.GatewayName, ",")
			return r.gateways[host]
		}
	}
	return nil
}

func (r *Registry) Routes(host string) []networking_istio_io_v1alpha3.HTTPRoute {
	routes := make([]networking_istio_io_v1alpha3.HTTPRoute, 0, len(r.routes))
	for parsedURL, route := range r.routes {
		if parsedURL.Host != host {
			continue
		}
		routes = append(routes, route)
	}

	SortRoutes(routes)

	return routes
}

func (r *Registry) VirtualService(host string) *networking_istio_io_v1alpha3.VirtualService {
	return &networking_istio_io_v1alpha3.VirtualService{
		TypeMeta: v1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: networking_istio_io_v1alpha3.GroupVersion.Identifier(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      MangleName(host),
			Namespace: r.namespace,
		},
		Spec: networking_istio_io_v1alpha3.VirtualServiceSpec{
			Gateways: r.ResolveAndCacheGateway(host),
			Hosts:    []string{host},
			HTTP:     r.Routes(host),
		},
	}
}

func RouteOwnedBy(destinations []networking_istio_io_v1alpha3.HTTPRouteDestination, name, namespace string) error {
	for _, dest := range destinations {
		if dest.Destination.Host != name+"."+namespace+ServiceSuffix {
			return fmt.Errorf("already in use by %s", dest.Destination.Host)
		}
	}
	return nil
}

func (r *Registry) Add(app *nais_io_v1alpha1.Application) error {
	routes, err := r.httpRoutes(app)
	if err != nil {
		return err
	}

	// Remove old ingresses before adding new ones
	r.Remove(app.Name, app.Namespace)

	for parsedURL, route := range routes {
		existing, found := r.routes[parsedURL]
		if found {
			err := RouteOwnedBy(existing.Route, app.Name, app.Namespace)
			if err != nil {
				return fmt.Errorf("the ingress %s is %s", parsedURL.String(), err)
			}
		}
		r.routes[parsedURL] = route
	}

	return nil
}

// Remove an application from the registry, and return the affected VirtualService resources
func (r *Registry) Remove(name, namespace string) []*networking_istio_io_v1alpha3.VirtualService {
	hosts := make(map[string]interface{})

	for parsedURL, routes := range r.routes {
		if RouteOwnedBy(routes.Route, name, namespace) == nil {
			hosts[parsedURL.Host] = new(interface{})
			delete(r.routes, parsedURL)
		}
	}

	services := make([]*networking_istio_io_v1alpha3.VirtualService, 0)
	for host := range hosts {
		services = append(services, r.VirtualService(host))
	}
	return services
}

func (r *Registry) httpRoutes(app *nais_io_v1alpha1.Application) (RouteMap, error) {
	routes := make(RouteMap, 0)
	for _, ingress := range app.Spec.Ingresses {

		parsedUrl, err := url.Parse(string(ingress))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
		}

		err = util.ValidateUrl(parsedUrl)
		if err != nil {
			return nil, err
		}

		if r.ResolveAndCacheGateway(parsedUrl.Host) == nil {
			return nil, fmt.Errorf("'%s' is not a supported domain", parsedUrl.Host)
		}

		route := httpRoute(parsedUrl.Path, app)
		routes[*parsedUrl] = route

	}
	return routes, nil
}

func httpRoute(path string, app *nais_io_v1alpha1.Application) networking_istio_io_v1alpha3.HTTPRoute {
	return networking_istio_io_v1alpha3.HTTPRoute{
		Match: []networking_istio_io_v1alpha3.HTTPMatchRequest{
			{
				URI: networking_istio_io_v1alpha3.StringMatch{
					Regex: path + regexSuffix,
				},
			},
		},
		Route: []networking_istio_io_v1alpha3.HTTPRouteDestination{
			{
				Destination: networking_istio_io_v1alpha3.Destination{
					Host: app.Name + "." + app.Namespace + ServiceSuffix,
					Port: networking_istio_io_v1alpha3.PortSelector{
						Number: uint32(app.Spec.Service.Port),
					},
				},
				Weight: IstioVirtualServiceTotalWeight,
			},
		},
	}
}

func SortRoutes(routes []networking_istio_io_v1alpha3.HTTPRoute) {
	// Sort by name first
	sort.Slice(routes, func(a, b int) bool {
		return routes[a].Match[0].URI.Regex < routes[b].Match[0].URI.Regex
	})

	// Sort by actual path length
	sort.SliceStable(routes, func(a, b int) bool {
		return len(routes[a].Match[0].URI.Regex) > len(routes[b].Match[0].URI.Regex)
	})

	// Sort by URL path length
	sort.SliceStable(routes, func(a, b int) bool {
		segA := strings.Split(routes[a].Match[0].URI.Regex, "/")
		segB := strings.Split(routes[b].Match[0].URI.Regex, "/")
		return len(segA) > len(segB)
	})
}
