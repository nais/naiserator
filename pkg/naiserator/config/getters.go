package config

func (c *Config) IsKafkaratorEnabled() bool {
	return c.Features.Kafkarator
}

func (c *Config) IsNetworkPolicyEnabled() bool {
	return c.Features.NetworkPolicy
}

func (c *Config) IsLinkerdEnabled() bool {
	return c.Features.Linkerd
}

func (c *Config) GetAPIServerIP() string {
	return c.ApiServerIp
}

func (c *Config) GetAccessPolicyNotAllowedCIDRs() []string {
	return c.Features.AccessPolicyNotAllowedCIDRs
}

func (c *Config) GetGoogleProjectID() string {
	return c.GoogleProjectId
}

func (c *Config) GetClusterName() string {
	return c.ClusterName
}

func (c *Config) GetGatewayMappings() []GatewayMapping {
	return c.GatewayMappings
}
