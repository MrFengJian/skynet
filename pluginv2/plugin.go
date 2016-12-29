package pluginv2

import (
        "github.com/containernetworking/cni/pkg/skel"
        "skynet/neutron"
        "skynet/util"
        "github.com/containernetworking/cni/pkg/ns"
        "github.com/containernetworking/cni/pkg/ip"
        "github.com/vishvananda/netlink"
        "errors"
        "fmt"
        "strconv"
        "github.com/pborman/uuid"
        "os"
)

const LINUX_BRIDGE = "linuxbridge"
const MAC_VLAN = "macvlan"
const OPEN_VSWITCH = "openvswitch"
const FLAT_TYPE = "flat"
const VXLAN_TYPE = "vxlan"
const VLAN_TYPE = "vlan"

func SetupInterface(args *skel.CmdArgs, podName, namespace string, network *neutron.OpenStackNetwork, subnet *neutron.OpenStackSubnet, podPort *neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        pluginType := conf.Plugin.Type
        fmt.Fprintln(os.Stderr, "Skynet setup container with  plugin type" + pluginType)
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

func DeleteInterface(netnsPath, interfaceName ,portId string,conf util.NetConf) (err error) {
        if conf.Plugin.Type==OPEN_VSWITCH{
                deleteOvsInterface(portId,conf)
        }
        err = ns.WithNetNSPath(netnsPath, func(_ ns.NetNS) error {
                _, err = ip.DelLinkByNameAddr(interfaceName, netlink.FAMILY_V4)
                return err
        })
        return err
}
//methods below are used to communicate with system to create network devices for all physical network implementations

func ensureBridge(bridgeName string, segmentId int, networkType, parentNic string) (err error) {
        existed := isDeviceExists(bridgeName)
        if !existed {
                Exec([]string{"brctl", "addbr", bridgeName})
        }
        ensureDeviceUp(bridgeName)
        var bridgeDevice string
        existed = isDeviceExists(parentNic)
        if !existed {
                return errors.New("unable to find device " + parentNic)
        }
        switch networkType {
        case VXLAN_TYPE:
                bridgeDevice = fmt.Sprintf("vxlan-%d", segmentId)
                existed = isDeviceExists(bridgeDevice);
                if !existed {
                        Exec([]string{"ip", "link", "add", bridgeDevice, "type", "vxlan", "id", strconv.Itoa(segmentId), "dev", parentNic})
                }
        case VLAN_TYPE:
                bridgeDevice, _ = ensureVlanDevice(parentNic, segmentId)
        case FLAT_TYPE:
                bridgeDevice = parentNic
        }
        ensureDeviceUp(bridgeDevice)
        Exec([]string{"brctl", "addif", bridgeName, bridgeDevice})
        return nil
}

func ensureVlanDevice(parentNic string, segmentId int) (string, error) {
        bridgeDevice := fmt.Sprintf("%s.%d", parentNic, segmentId)
        existed := isDeviceExists(bridgeDevice);
        if !existed {
                if err := Exec([]string{"ip", "link", "add", "link", parentNic, "name", bridgeDevice, "type", "vlan", "id", strconv.Itoa(segmentId)}); err != nil {
                        return "", err
                }
        }
        return bridgeDevice, nil
}

func ensureDeviceUp(device string) (err error) {
        return Exec([]string{"ip", "link", "set", device, "up"})
}

func isDeviceExists(device string) (existed bool) {
        err := Exec([]string{"ip", "link", "show", device})
        return err == nil
}

func setupContainer(containerNs, vifName, containerInterface string, portMac string, podIp neutron.FixIP, gateway string) (err error) {
        //move vif to container ns
        nsDir := "/var/run/netns/"
        nsName := uuid.New()
        nsNormalPath := nsDir + nsName
        if err = os.MkdirAll(nsDir, 0755); err != nil {
                return err
        }
        os.Symlink(containerNs, nsNormalPath)
        defer os.Remove(nsNormalPath)
        if err = Exec([]string{"ip", "link", "set", vifName, "netns", nsName}); err != nil {
                return err
        }
        if err = Exec(changeTonsArgs(nsName, []string{"ip", "link", "set", "dev", vifName, "name", containerInterface})); err != nil {
                return err
        }
        if err = Exec(changeTonsArgs(nsName, []string{"ip", "link", "set", containerInterface, "up"})); err != nil {
                return err
        }
        if err = Exec(changeTonsArgs(nsName, []string{"ip", "link", "set", containerInterface, "address", portMac})); err != nil {
                return err
        }
        if err = Exec(changeTonsArgs(nsName, []string{"ip", "-4", "addr", "add", podIp.IpAddress + "/24", "dev", containerInterface})); err != nil {
                return err
        }
        if err = Exec(changeTonsArgs(nsName, []string{"ip", "-4", "route", "replace", gateway, "dev", containerInterface})); err != nil {
                return err
        }
        if err = Exec(changeTonsArgs(nsName, []string{"ip", "-4", "route", "replace", "default", "via", gateway, "dev", containerInterface})); err != nil {
                return err
        }
        return nil
}

func createVethPair(hostVethName, contVethNameTemp string) (err error) {
        return Exec([]string{"ip", "link", "add", hostVethName, "type", "veth", "peer", "name", contVethNameTemp})
}