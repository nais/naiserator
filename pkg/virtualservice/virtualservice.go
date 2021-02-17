package virtualservice

import (
    "fmt"
    "net/url"
    "sort"
    "strings"

    "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
    networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
    v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    "github.com/nais/naiserator/pkg/naiserator/config"
)

const IstioVirtualServiceTotalWeight int32 = 100 // The total weight of all routes must equal 100

const regexSuffix = "(/.*)?"

type Gateway string

type Route struct {
    Gateway Gateway
    Route   networking_istio_io_v1alpha3.HTTPRoute
}

type RouteMap map[url.URL]networking_istio_io_v1alpha3.HTTPRoute

type Registry struct {
    routes   RouteMap
    mappings []config.GatewayMapping
}

func (r *Registry) VirtualServices(app *nais_io_v1alpha1.Application) []networking_istio_io_v1alpha3.VirtualService {
    services := make([]networking_istio_io_v1alpha3.VirtualService, 0)
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
    return services
}

func New(gatewayMapping []config.GatewayMapping) *Registry {
    return &Registry{
        routes:   make(RouteMap),
        mappings: gatewayMapping,
    }
}

func ResolveGateway(host string, mappings []config.GatewayMapping) []string {
    for _, mapping := range mappings {
        if strings.HasSuffix(host, mapping.DomainSuffix) {
            return strings.Split(mapping.GatewayName, ",")
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

func (r *Registry) VirtualService(host string) networking_istio_io_v1alpha3.VirtualService {
    return networking_istio_io_v1alpha3.VirtualService{
        TypeMeta: v1.TypeMeta{
            Kind:       "VirtualService",
            APIVersion: networking_istio_io_v1alpha3.GroupVersion.Version,
        },
        ObjectMeta: v1.ObjectMeta{
            Name:      host,
            Namespace: "namespace",
        },
        Spec: networking_istio_io_v1alpha3.VirtualServiceSpec{
            Gateways: ResolveGateway(host, r.mappings),
            Hosts:    []string{host},
            HTTP:     r.Routes(host),
        },
    }
}

func (r *Registry) Add(app *nais_io_v1alpha1.Application) error {
    routes, err := httpRoutes(app)
    if err != nil {
        return err
    }

    // fixme: collisions: check same app

    for parsedURL, route := range routes {
        r.routes[parsedURL] = route
    }

    return nil
}

func httpRoutes(app *nais_io_v1alpha1.Application) (RouteMap, error) {
    routes := make(RouteMap, 0)
    for _, ingress := range app.Spec.Ingresses {

        parsedUrl, err := url.Parse(string(ingress))
        if err != nil {
            return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
        }

        if len(parsedUrl.Path) == 0 {
            parsedUrl.Path = "/"
        }

        /* FIXME
           err = validateUrl(parsedUrl)
           if err != nil {
           	return nil, err
           }

        */

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
                    Host: app.Name,
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
