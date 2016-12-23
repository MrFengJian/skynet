package pluginv2

import (
        "neutron"
        "github.com/containernetworking/cni/pkg/skel"
        "util"
        "fmt"
        "os"
        "errors"
        "strconv"
)

func SetupOvsInterface(args *skel.CmdArgs, podName, namespace string, network *neutron.OpenStackNetwork, subnet *neutron.OpenStackSubnet, port *neutron.OpenStackPort,
podIp neutron.FixIP, conf util.NetConf) (err error) {
        portId := port.Id
        portMac := port.MacAddress
        fmt.Fprintf(os.Stderr, "Skynet set container interface with port id  %s\n", portId)
        tapName, vifName := getInterfaces(portId)
        containerInterface := args.IfName
        if containerInterface == "" {
                containerInterface = "eth0"
        }
        networkType := network.NetworkType
        segmentId := network.SegmentId
        containerNs := args.Netns
        if networkType != VLAN_TYPE&&networkType != FLAT_TYPE {
                return errors.New("openvswitch DO NOT support network type: " + networkType)
        }
        //for flat network do not use segment
        if networkType == FLAT_TYPE {
                segmentId = 0
        }
        fmt.Fprintf(os.Stderr, "Skynet set container interface with %s\n", containerInterface)
        existed := isDeviceExists(tapName)
        if !existed {
                if err = createVethPair(tapName, vifName); err != nil {
                        return err
                }
        }
        bridgeName := "br-int"
        if err = addTapToOvsBridge(bridgeName, tapName, conf.Plugin.TrunkNic, segmentId); err != nil {
                deleteVeth(tapName)
                return err;
        }
        if err = setupContainer(containerNs, vifName, containerInterface, portMac, podIp, subnet.GatewayIp); err != nil {
                deleteVeth(tapName)
                return err
        }
        if err = ensureDeviceUp(tapName); err != nil {
                return err
        }
        return nil
}

func addTapToOvsBridge(bridgeName, tapName, bridgeNic string, segmentId int) (err error) {
        if err = Exec([]string{"ovs-vsctl", "br-exists", bridgeName}); err != nil {
                Exec([]string{"ovs-vsctl", "add-br", bridgeName})
                Exec([]string{"ovs-vsctl", "add-port", bridgeName, bridgeNic})
        }
        ensureDeviceUp(bridgeName)
        if err = Exec([]string{"ovs-vsctl", "add-port", bridgeName, tapName}); err != nil {
                return err
        }
        if segmentId != 0 {
                Exec([]string{"ovs-vsctl", "set", "port", tapName, "tag=" + strconv.Itoa(segmentId)})
        }
        return nil
}

func deleteOvsInterface(portId string, conf util.NetConf) error {
        if portId != "" {
                tapName, _ := getInterfaces(portId)
                bridgeName := "br-int"
                return Exec([]string{"ovs-vsctl", "del-port", bridgeName, tapName})
        }
        return nil
}

func getInterfaces(portId string) (string, string) {
        return "qvo" + portId[:11], "tap" + portId[:11]
}
