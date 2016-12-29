package pluginv2

import (
        "skynet/util"
        "skynet/neutron"
        "github.com/containernetworking/cni/pkg/skel"
        "errors"
        "os"
        "fmt"
)

func SetupBridgeInterface(args *skel.CmdArgs, podName, namespace string, network *neutron.OpenStackNetwork, subnet *neutron.OpenStackSubnet, port *neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        portId := port.Id
        portMac := port.MacAddress
        fmt.Fprintf(os.Stderr, "Skynet set container interface with portid  %s\n", portId)
        tapName, vifName := "tap" + portId[:11], "vif" + portId[:11]
        containerInterface := args.IfName
        if containerInterface == "" {
                containerInterface = "eth0"
        }
        networkType := network.NetworkType
        segmentId := network.SegmentId
        bridgeName := "brq" + network.Id[:11]
        containerNs := args.Netns
        var parentNic string
        switch networkType {
        case VXLAN_TYPE:
                parentNic = conf.Plugin.TunnelNic
        case VLAN_TYPE:
                parentNic = conf.Plugin.TrunkNic
        case FLAT_TYPE:
                parentNic = conf.Plugin.TrunkNic
        default:
                err = errors.New("linuxbridge DO NOT support network type: " + networkType)
        }
        if err != nil {
                return err
        }
        fmt.Fprintf(os.Stderr, "Skynet set container interface with %s\n", containerInterface)
        if err = ensureBridge(bridgeName, segmentId, networkType, parentNic); err != nil {
                return err
        }
        existed := isDeviceExists(tapName)
        if !existed {
                if err = createVethPair(tapName, vifName); err != nil {
                        return err
                }
        }
        if err = addIfToBridge(tapName, bridgeName); err != nil {
                deleteVeth(tapName)
                return err
        }
        if err = setupContainer(containerNs, vifName, containerInterface, portMac, podIp, subnet.GatewayIp); err != nil {
                deleteVeth(tapName)
                return err
        }
        if err = ensureDeviceUp(tapName); err != nil {
                return err
        }
        if networkType == VXLAN_TYPE {
                //TODO: this is just a temporary solution,may be changed later
                if conf.Neutron.ExternalRouterGatewayIp != "" {
                        ensureCidrRoute(subnet.Cidr, conf.Neutron.ExternalRouterGatewayIp, conf.Neutron.ExternalRouteNic)
                        if conf.Neutron.ServiceClusterIpRange != "" {
                                ensureCidrRoute(conf.Neutron.ServiceClusterIpRange, conf.Neutron.ExternalRouterGatewayIp, conf.Plugin.TunnelNic)
                        }
                }
        }
        return nil
}

func addIfToBridge(ifName, bridgeName string) error {
        return Exec([]string{"brctl", "addif", bridgeName, ifName})
}

func deleteVeth(veth string) error {
        return Exec([]string{"ip", "link", "delete", veth})
}

func ensureCidrRoute(cidr, nextHopIp, devName string) (err error) {
        return Exec([]string{"ip", "-4", "route", "add", cidr, "via", nextHopIp, "dev", devName})
}
                                                                                                                                                                                                                       