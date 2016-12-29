package plugin

import (
        "neutron"
        "github.com/containernetworking/cni/pkg/skel"
        "util"
)

func SetupOvsInterface(args *skel.CmdArgs, podName, namespace string, network neutron.OpenStackNetwork, subnet neutron.OpenStackSubnet, podPort neutron.OpenStackPort,
        podIp neutron.FixIP, conf util.NetConf) (err error) {
        return nil
}
