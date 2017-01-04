package neutron

type FixIP struct {
        SubnetId  string `json:"subnet_id"`
        IpAddress string `json:"ip_address"`
}

type OpenStackPort struct {
        Status           string `json:"status,omitempty"`
        Name             string `json:"name,omitempty"`
        Id               string `json:"id,omitempty"`
        DnsName          string `json:"dns_name,omitempty"`
        NetworkId        string `json:"network_id,omitempty"`
        MacAddress       string `json:"mac_address,omitempty"`
        FixedIPS         []FixIP `json:"fixed_ips,omitempty"`
        AdminStateUp     bool `json:"admin_state_up,omitempty"`
        TenantId         string `json:"tenant_id,omitempty"`
        SecurityGroupIds []string `json:"security_groups,omitempty"`
        BindingHost      string `json:"binding:host_id,omitempty"`
        DeviceOwner      string `json:"device_owner,omitempty"`
        DeviceId         string `json:"device_id,omitempty"`
}

type PortsResponse struct {
        Ports []OpenStackPort `json:"ports"`
}

type PortResponse struct {
        Port OpenStackPort `json:"port"`
}

type OpenStackSubnet struct {
        Name      string `json:"name"`
        NetworkId string `json:"network_id"`
        TenantId  string `json:"tenant_id"`
        Id        string `json:"id"`
        GatewayIp string `json:"gateway_ip"`
        Cidr      string `json:"cidr"`
}
type SubnetResponse struct {
        Subnet OpenStackSubnet `json:"subnet"`
}

type OpenStackNetwork struct {
        Name        string `json:"name"`
        Id          string `json:"id"`
        TenantId    string `json:"tenant_id"`
        NetworkType string `json:"provider:network_type"`
        SegmentId   int `json:"provider:segmentation_id"`
}

type NetworkResponse struct {
        Network OpenStackNetwork `json:"network"`
}

type OpenStackSecurityGroup struct {
        Id       string `json:"id"`
        Name     string `json:"name"`
        TenantId string `json:"tenant_id"`
}

type SecurityGroupsResponse struct {
        SecurityGroups []OpenStackSecurityGroup `json:"security_groups"`
}