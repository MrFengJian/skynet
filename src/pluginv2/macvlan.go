package pluginv2

import (
        "util"
        "neutron"
        "github.com/containernetworking/cni/pkg/skel"
        "os"
        "fmt"
        "errors"
)

func SetupMacVlanInterface(args *skel.CmdArgs, podName string, namespace string, network *neutron.OpenStackNetwork, subnet *neutron.OpenStackSubnet, port *neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        portId := port.Id
        portMac := port.MacAddress
        fmt.Fprintf(os.Stderr, "Skynet set container interface with port id  %s\n", portId)
        vifName := "vif" + portId[:11]
        containerInterface := args.IfName
        if containerInterface == "" {
                containerInterface = "eth0"
        }
        networkType := network.NetworkType
        segmentId := network.SegmentId
        containerNs := args.Netns
        var parentNic string
        if networkType != VLAN_TYPE&&networkType != FLAT_TYPE {
                return errors.New("macvlan DO NOT support network type: " + networkType)
        }
        switch networkType {
        case VLAN_TYPE:
                parentNic, err = ensureVlanDevice(conf.Plugin.TrunkNic, segmentId)
                ensureDeviceUp(parentNic)
        case FLAT_TYPE:
                parentNic = conf.Plugin.TrunkNic
        default:
                err = errors.New("macvlan DO NOT support network type: " + networkType)
        }
        if err != nil {
                return err
        }
        fmt.Fprintf(os.Stderr, "Skynet set container interface with %s\n", containerInterface)
        if err = createMacvlanDevice(vifName, parentNic); err != nil {
                return err
        }
        if err = setupContainer(containerNs, vifName, containerInterface, portMac, podIp, subnet.GatewayIp); err != nil {
                deleteVeth(vifName)
                return err
        }
        return nil
}

func createMacvlanDevice(vifName, parentNic string) error {
        return Exec([]string{"ip", "link", "add", "link", parentNic, vifName, "type", "macvlan", "mode", "bridge"})
}