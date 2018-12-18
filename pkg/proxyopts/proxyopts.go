package proxyopts

import (
	"fmt"
	"net/url"
	"strings"
)

type javaOption struct {
	Key   string
	Value string
}

type javaOptions []javaOption

func (o javaOption) Format() string {
	return fmt.Sprintf("-D%s=%s", o.Key, o.Value)
}

func newJavaOption(key string, value string) javaOption {
	return javaOption{
		Key:   key,
		Value: value,
	}
}

func (o javaOptions) Format() string {
	s := make([]string, len(o))
	for i, opt := range o {
		s[i] = opt.Format()
	}
	return strings.Join(s, " ")
}

func httpOpts(flags javaOptions, proxyURL string) (javaOptions, error) {
	if len(proxyURL) == 0 {
		return flags, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return flags, err
	}

	if len(u.Hostname()) == 0 || len(u.Port()) == 0 {
		return flags, fmt.Errorf("if specifying a proxy URL, both hostname and port is required")
	}

	flags = append(flags, newJavaOption("http.proxyHost", u.Hostname()))
	flags = append(flags, newJavaOption("https.proxyHost", u.Hostname()))
	flags = append(flags, newJavaOption("http.proxyPort", u.Port()))
	flags = append(flags, newJavaOption("https.proxyPort", u.Port()))

	return flags, nil
}

// mangleWildcard takes a list of hostnames and prepends '*' if the hostname
// starts with '.', then returns a new slice with the modified hostnames.
func mangleWildcard(hosts []string) []string {
	mangled := make([]string, len(hosts))
	for i, host := range hosts {
		if len(host) > 0 && host[0] == '.' {
			host = "*" + host
		}
		mangled[i] = host
	}
	return mangled
}

func noProxyOpts(flags javaOptions, noProxy string) (javaOptions, error) {
	if len(noProxy) == 0 {
		return flags, nil
	}

	hosts := mangleWildcard(strings.Split(noProxy, ","))
	flags = append(flags, newJavaOption("http.nonProxyHosts", strings.Join(hosts, "|")))

	return flags, nil
}

// JavaProxyOptions converts *NIX style http proxy environment variables
// $HTTP_PROXY and $NO_PROXY to JVM startup flag equivalents.
func JavaProxyOptions(proxyURL, noProxy string) (s string, err error) {
	flags := make(javaOptions, 0)

	flags, err = httpOpts(flags, proxyURL)
	if err != nil {
		err = fmt.Errorf("error in parsing proxy URL: %s", err)
		return
	}

	flags, _ = noProxyOpts(flags, noProxy)

	s = flags.Format()

	return
}
