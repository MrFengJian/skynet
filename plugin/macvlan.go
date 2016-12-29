package plugin

import (
        "skynet/util"
        "skynet/neutron"
        "github.com/containernetworking/cni/pkg/skel"
)

func SetupMacVlanInterface(args *skel.CmdArgs, podName string, namespace string, network neutron.OpenStackNetwork, subnet neutron.OpenStackSubnet, podPort neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        return nil
}