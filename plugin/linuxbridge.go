package plugin

import (
        "skynet/util"
        "skynet/neutron"
        "github.com/containernetworking/cni/pkg/skel"
        "errors"
        "github.com/vishvananda/netlink"
        "github.com/containernetworking/cni/pkg/ip"
        "net"
        "github.com/containernetworking/cni/pkg/ns"
)

func SetupBridgeInterface(args *skel.CmdArgs, podName, namespace string, network neutron.OpenStackNetwork, subnet neutron.OpenStackSubnet, port neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        portId := port.Id
        portMac := port.MacAddress
        tapName, vifName := "tap" + portId[:11], "vif" + portId[:11]
        containerInterface := args.IfName
        if containerInterface == "" {
                containerInterface = "eth0"
        }
        networkType := network.NetworkType
        segmentId := network.SegmentId
        bridgeName := "brq" + network.Id[:11]
        var bridgeCommonLink netlink.Link
        containerNs := args.Netns
        var parentNic string
        switch networkType {
        case VXLAN_TYPE:
                parentNic=conf.Plugin.TunnelNic
        case VLAN_TYPE:
                parentNic=conf.Plugin.TrunkNic
        case FLAT_TYPE:
                parentNic = conf.Plugin.TrunkNic
        default:
                err = errors.New("linuxbridge DO NOT support network type: " + networkType)
        }
        if err != nil {
                return err
        }
        bridge, err := ensureBridge(bridgeName,segmentId,networkType,parentNic)
        err = ns.WithNetNSPath(containerNs, func(hostNS ns.NetNS) error {
                var vifLink netlink.Link
                tapLink, existed := isDeviceExists(tapName)
                if !existed {
                        tapLink, vifLink, err = createVethPair(tapName, vifName)
                }
                _ = netlink.LinkSetMaster(bridgeCommonLink, bridge)
                _ = netlink.LinkSetMaster(tapLink, bridge)
                if vifLink == nil {
                        return errors.New("unable to find container vif device " + vifName)
                }
                if err = setupContainer(vifLink, containerInterface, portMac, podIp, subnet.GatewayIp); err != nil {
                        _ = netlink.LinkDel(tapLink)
                        return err
                }
                if err := netlink.LinkSetNsFd(tapLink, int(hostNS.Fd())); err != nil {
                        _ = netlink.LinkDel(tapLink)
                        return err
                }
                return nil
        })
        if err != nil {
                return err
        }
        tapLink, err := netlink.LinkByName(tapName)
        if err != nil {
                return errors.New("unable to lookup host end device " + tapName + "for container ")
        }
        if err = ensureDeviceUp(tapLink); err != nil {
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

//func ensureVxlanDevice(segmentId int, nic string) (vxlan netlink.Vxlan, err error) {
//        deviceName := fmt.Sprintf("vxlan-%s", segmentId)
//        link, existed := isDeviceExists(deviceName);
//        if !existed {
//                parent, err := netlink.LinkByName(nic)
//                if err != nil {
//                        return vxlan, err
//                }
//                vxlan := &netlink.Vxlan{
//                        LinkAttrs:netlink.LinkAttrs{
//                                Name:deviceName,
//                        },
//                        VxlanId:segmentId,
//                        VtepDevIndex:parent.Attrs().Index,
//                }
//                _ = netlink.LinkAdd(vxlan)
//                link, _ = netlink.LinkByName(deviceName)
//        }
//        if vxlan, ok := link.(*netlink.Vxlan); ok {
//                ensureDeviceUp(vxlan)
//                return vxlan, nil
//        }else {
//                return vxlan, errors.New("device " + deviceName + " is not vxlan device")
//        }
//}

func ensureCidrRoute(cidr, nextHopIp, devName string) (err error) {
        nextHop := net.ParseIP(nextHopIp)
        _, destDet, err := net.ParseCIDR(cidr)
        if err != nil {
                return err
        }
        devLink, _ := netlink.LinkByName(devName)
        err = ip.AddRoute(destDet, nextHop, devLink)
        return err
}
                                                                                                                                                                                                                       