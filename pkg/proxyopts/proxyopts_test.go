package proxyopts

import (
	"testing"
)

var testCases = []struct {
	proxyUrl string
	noProxy  string
	Output   string
	Error    bool
}{
	{
		proxyUrl: "http://foo.bar:1234",
		Output:   `-Dhttp.proxyHost=foo.bar -Dhttps.proxyHost=foo.bar -Dhttp.proxyPort=1234 -Dhttps.proxyPort=1234`,
	},
	{
		Output: ``,
	},
	{
		proxyUrl: "http://foo.bar:1234",
		Output:   `-Dhttp.proxyHost=foo.bar -Dhttps.proxyHost=foo.bar -Dhttp.proxyPort=1234 -Dhttps.proxyPort=1234`,
	},
	{
		noProxy: "internalhost",
		Output:  `-Dhttp.nonProxyHosts=internalhost`,
	},
	{
		noProxy: "host1,host2,.wildcard.local,.local,foo",
		Output:  `-Dhttp.nonProxyHosts=host1|host2|*.wildcard.local|*.local|foo`,
	},
	{
		proxyUrl: "http://foo.bar:1234",
		noProxy:  "host1,host2,.wildcard.local,.local,foo",
		Output:   `-Dhttp.proxyHost=foo.bar -Dhttps.proxyHost=foo.bar -Dhttp.proxyPort=1234 -Dhttps.proxyPort=1234 -Dhttp.nonProxyHosts=host1|host2|*.wildcard.local|*.local|foo`,
	},
	{
		proxyUrl: "foo.bar:1234",
		Error:    true,
	},
	{
		proxyUrl: "http://proxy",
		Error:    true,
	},
}

func TestSuccess(t *testing.T) {
	for i, test := range testCases {
		output, err := JavaProxyOptions(test.proxyUrl, test.noProxy)

		if test.Error {
			if err == nil {
				t.Fatalf("Test #%d: expected error, got success instead", i)
			}
		} else {
			if err != nil {
				t.Fatalf("Test #%d: expected success, got error: %s", i, err)
			}
			if output != test.Output {
				t.Fatalf("Test #%d: expected output \"%s\", got \"%s\"", i, test.Output, output)
			}
		}
	}
}
