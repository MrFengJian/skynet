package plugin

import (
        "github.com/containernetworking/cni/pkg/skel"
        "neutron"
        "util"
        "github.com/containernetworking/cni/pkg/ns"
        "github.com/containernetworking/cni/pkg/ip"
        "github.com/vishvananda/netlink"
        "errors"
        "net"
        "fmt"
)

const LINUX_BRIDGE = "linuxbridge"
const MAC_VLAN = "macvlan"
const OPEN_VSWITCH = "openvswitch"
const FLAT_TYPE = "flat"
const VXLAN_TYPE = "vxlan"
const VLAN_TYPE = "vlan"

func SetupInterface(args *skel.CmdArgs, podName, namespace string, network neutron.OpenStackNetwork, subnet neutron.OpenStackSubnet, podPort neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        pluginType := conf.Plugin.Type
        switch pluginType {
        case LINUX_BRIDGE:
                err = SetupBridgeInterface(args, podName, namespace, network, subnet, podPort, podIp, conf)
        case MAC_VLAN:
                err = SetupMacVlanInterface(args, podName, namespace, network, subnet, podPort, podIp, conf)
        case OPEN_VSWITCH:
                err = SetupOvsInterface(args, podName, namespace, network, subnet, podPort, podIp, conf)
        default:
                err = errors.New("unsupported plugin type: " + pluginType)
        }
        return err
}

func DeleteInterface(netnsPath, interfaceName string) (err error) {
        err = ns.WithNetNSPath(netnsPath, func(_ ns.NetNS) error {
                _, err = ip.DelLinkByNameAddr(interfaceName, netlink.FAMILY_V4)
                return err
        })
        return err
}
//methods below are used to communicate with system to create network devices for all physical network implementations

//func ensureVlanDevice(segmentId int, nic string) (link netlink.Link, err error) {
//        deviceName := fmt.Sprintf("%s.%s", nic, segmentId)
//        link, existed := isDeviceExists(deviceName)
//        if !existed {
//                parent, err := netlink.LinkByName(nic)
//                if err != nil {
//                        return link, err
//                }
//                vlan := &netlink.Vlan{VlanId:segmentId,
//                        LinkAttrs:netlink.LinkAttrs{
//                                Name:deviceName,
//                                ParentIndex:parent.Attrs().Index,
//                        }}
//                if err = netlink.LinkAdd(vlan); err != nil {
//                        return link, err
//                }
//        }
//        return link, nil
//}

func ensureBridge(bridgeName string, segmentId int, networkType, parentNic string) (bridge *netlink.Bridge, err error) {
        link, existed := isDeviceExists(bridgeName)
        if existed {
                bridge = link.(*netlink.Bridge)
        }else {
                bridge = &netlink.Bridge{LinkAttrs:netlink.LinkAttrs{
                        Name:bridgeName,
                }}
                netlink.LinkAdd(bridge)
        }
        ensureDeviceUp(bridge)
        var deviceLink netlink.Link
        switch networkType {
        case VXLAN_TYPE:
                deviceName := fmt.Sprintf("vxlan-%d", segmentId)
                deviceLink, existed = isDeviceExists(deviceName);
                if !existed {
                        parent, err := netlink.LinkByName(parentNic)
                        if err != nil {
                                return bridge, err
                        }
                        vxlan := &netlink.Vxlan{
                                LinkAttrs:netlink.LinkAttrs{
                                        Name:deviceName,
                                },
                                VxlanId:segmentId,
                                VtepDevIndex:parent.Attrs().Index,
                        }
                        _ = netlink.LinkAdd(vxlan)
                        deviceLink, _ = netlink.LinkByName(deviceName)
                }
        case VLAN_TYPE:
                deviceName := fmt.Sprintf("%s.%d", parentNic, segmentId)
                deviceLink, existed = isDeviceExists(deviceName)
                if !existed {
                        parent, err := netlink.LinkByName(parentNic)
                        if err != nil {
                                return bridge, err
                        }
                        vlan := &netlink.Vlan{VlanId:segmentId,
                                LinkAttrs:netlink.LinkAttrs{
                                        Name:deviceName,
                                        ParentIndex:parent.Attrs().Index,
                                }}
                        if err = netlink.LinkAdd(vlan); err != nil {
                                return bridge, err
                        }
                }
        case FLAT_TYPE:
                deviceLink, existed = isDeviceExists(parentNic)
                if !existed {
                        return bridge, errors.New("unable to find parent nic " + parentNic)
                }
        }
        ensureDeviceUp(deviceLink)
        netlink.LinkSetMaster(link, bridge)
        return bridge, nil
}

func setVethMac(veth netlink.Link, mac string) (error) {
        addr, _ := net.ParseMAC(mac)
        return netlink.LinkSetHardwareAddr(veth, addr)
}

func ensureDeviceUp(link netlink.Link) (err error) {
        _ = netlink.LinkSetUp(link)
        return nil
}

func isDeviceExists(device string) (link netlink.Link, existed bool) {
        link, err := netlink.LinkByName(device)
        if err != nil {
                return nil, false
        }
        return link, true
}

func setupContainer(vifLink netlink.Link, containerInterface string, portMac string, podIp neutron.FixIP, gateway string) (err error) {
        if err = setVethMac(vifLink, portMac); err != nil {
                return err
        }
        hwAddr, _ := net.ParseMAC(portMac)
        if err = netlink.LinkSetHardwareAddr(vifLink, hwAddr); err != nil {
                return err
        }
        netlink.LinkSetName(vifLink, containerInterface)
        vifLink, _ = netlink.LinkByName(containerInterface)
        ipCidr := podIp.IpAddress + "/24"
        _, ipNet, _ := net.ParseCIDR(ipCidr)
        if err = netlink.AddrAdd(vifLink, &netlink.Addr{IPNet:ipNet}); err != nil {
                return err
        }
        gwIp := net.ParseIP(gateway)
        if err = ip.AddDefaultRoute(gwIp, vifLink); err != nil {
                return nil
        }
        return nil
}

func createVethPair(hostVethName, contVethNameTemp string) (hostVeth, tempVeth netlink.Link, err error) {
        veth := &netlink.Veth{
                LinkAttrs: netlink.LinkAttrs{
                        Name:  contVethNameTemp,
                        Flags: net.FlagUp,
                },
                PeerName: hostVethName,
        }

        if err := netlink.LinkAdd(veth); err != nil {
                return nil, nil, err
        }

        hostVeth, err = netlink.LinkByName(hostVethName)
        if err != nil {
                err = fmt.Errorf("failed to lookup %q: %v", hostVethName, err)
                return nil, nil, err
        }

        tempVeth, err = netlink.LinkByName(contVethNameTemp)
        if err != nil {
                err = fmt.Errorf("failed to lookup %q: %v", contVethNameTemp, err)
                return nil, nil, err
        }
        return hostVeth, tempVeth, nil
}