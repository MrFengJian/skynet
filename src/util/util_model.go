package util

import (
	"net"
	"github.com/containernetworking/cni/pkg/types"
)

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

// Kubernetes a K8s specific struct to hold config
type Kubernetes struct {
	K8sAPIRoot string `json:"k8s_api_root"`
	Kubeconfig string `json:"kubeconfig"`
	NodeName   string `json:"node_name"`
}

//Neutron stores config used to communicate with neutron server
type NeutronConfig struct {
	NeutronUrl              string `json:"neutron_url"`
	DefaultNetworkId        string `json:"default_network_id"`
	DefaultSubnetId         string `json:"default_subnet_id"`
	DefaultSecurityGroupIds string `json:"default_security_group_ids"`
	ExternalRouterGatewayIp string `json:"external_gateway_ip"`
	ServiceClusterIpRange   string `json:"service_cluster_ip_range"`
	ExternalRouteNic string `json:"external_gateway_nic"`
}

//Plugin stores the plugin used parameters to implment plysical network devices
type Plugin struct {
	Type      string `json:"plugin_type"`
	TrunkNic  string `json:"trunk_nic"`
	TunnelNic string `json:"tunnel_nic"`
}

// NetConf stores the common network config for Skynet CNI plugin
type NetConf struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IPAM       struct {
			   Name       string
			   Type       string  `json:"type"`
			   Subnet     string  `json:"subnet"`
			   AssignIpv4 *string `json:"assign_ipv4"`
			   AssignIpv6 *string `json:"assign_ipv6"`
		   } `json:"ipam,omitempty"`
	Hostname   string     `json:"hostname"`
	Neutron    NeutronConfig     `json:"neutron"`
	LogLevel   string     `json:"log_level"`
	Kubernetes Kubernetes `json:"kubernetes"`
	Plugin     Plugin `json:"plugin"`
}
