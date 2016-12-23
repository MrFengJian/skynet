package main

import (
        "runtime"
        "os"
        "flag"
        "fmt"
        "github.com/containernetworking/cni/pkg/skel"
        "encoding/json"
        log "github.com/Sirupsen/logrus"
        "neutron"
        "net/http"
        "io/ioutil"
        "strings"
        "errors"
        plugin "pluginv2"
        "util"
        "github.com/containernetworking/cni/pkg/version"
        "github.com/containernetworking/cni/pkg/types"
)

var hostname string

//consts in pod spec to specify pod network config
const NETWORK_ID_KEY = "skynet/network_id"
const SUBNET_ID_KEY = "skynet/subnet_id"
const SECURITY_GROUP_IDS_KEY = "skynet/security_group_ids"
const IP_KEY = "skynet/ip"

const VERSION = "1.0"

func init() {
        runtime.LockOSThread()
        hostname, _ = os.Hostname()
}

func getPodSpec(k8sUrl, podName, namespace string) (pod neutron.K8sPod, err error) {
        url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", k8sUrl, namespace, podName)
        resp, err := http.Get(url)
        if err != nil {
                return pod, err
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return pod, err
        }
        err = json.Unmarshal(body, &pod)
        if err != nil {
                return pod, err
        }
        return pod, nil
}

func getPodNetwork(conf util.NetConf, neutronApi *neutron.NeutronApi, podName, namespace string) (network *neutron.OpenStackNetwork, subnet *neutron.OpenStackSubnet, ipSpecified string, securityGroupIds []string, err error) {
        k8sUrl := conf.Kubernetes.K8sAPIRoot

        pod, err := getPodSpec(k8sUrl, podName, namespace)
        if err != nil {
                return nil, nil, ipSpecified, securityGroupIds, err
        }
        annotations := pod.Metadata.Annotations

        if value, ok := annotations[SECURITY_GROUP_IDS_KEY]; ok {
                securityGroupIds = strings.Split(value, ",")
        }else if conf.Neutron.DefaultSecurityGroupIds != "" {
                securityGroupIds = strings.Split(conf.Neutron.DefaultSecurityGroupIds, ",")
        }else {
                securityGroupIds = make([]string, 0)
        }

        var networkId string
        if value, ok := annotations[NETWORK_ID_KEY]; ok {
                networkId = value
        }else {
                networkId = conf.Neutron.DefaultNetworkId
        }

        var subnetId string
        if value, ok := annotations[SUBNET_ID_KEY]; ok {
                subnetId = value
        }else if conf.Neutron.DefaultSubnetId != "" {
                subnetId = conf.Neutron.DefaultSubnetId
        }

        if value, ok := annotations[IP_KEY]; ok {
                ipSpecified = value
        }

        if subnetId != "" {
                subnet, err = neutronApi.GetSubnet(subnetId)
                fmt.Fprintln(os.Stderr,subnet,&subnet,err)
                if err != nil {
                        return nil, nil, ipSpecified, securityGroupIds, err
                }
                network, err = neutronApi.GetNetwork(subnet.NetworkId)
                if err != nil {
                        return nil, nil, ipSpecified, securityGroupIds, err
                }
                return network, subnet, ipSpecified, securityGroupIds, nil
        }else if networkId != "" {
                network, err = neutronApi.GetNetwork(subnet.NetworkId)
                if err != nil {
                        return nil, nil, ipSpecified, securityGroupIds, err
                }
                return network, subnet, ipSpecified, securityGroupIds, nil
        }
        return nil, nil, ipSpecified, securityGroupIds, errors.New("unable to get pod network config")

}

func cmdAdd(args *skel.CmdArgs) error {
        // Unmarshall the network config, and perform validation
        conf := util.NetConf{}
        if err := json.Unmarshal(args.StdinData, &conf); err != nil {
                return fmt.Errorf("failed to load netconf: %v", err)
        }
        util.ConfigureLogging(conf.LogLevel)

        workload, podName, namespace, err := util.GetPortIdentifier(args)
        if err != nil {
                return err
        }

        logger := util.CreateContextLogger(workload)

        // Allow the hostname to be overridden by the network config
        if conf.Hostname != "" {
                hostname = conf.Hostname
        }

        logger.WithFields(log.Fields{
                "WorkLoad": workload,
                "Node":         hostname,
        }).Info("Extracted identifiers")

        logger.WithFields(log.Fields{"NetConfg": conf, "Args":args}).Info("Loaded CNI NetConf")
        //create port from neutron
        neutronApi := neutron.NewNeutronApi(conf.Neutron.NeutronUrl)
        network, subnet, ipSpecified, securityGroupIds, err := getPodNetwork(conf, neutronApi, podName, namespace)
        if err != nil {
                return err
        }
        ports, err := neutronApi.GetPorts(workload)
        var podPort *neutron.OpenStackPort
        var needUpdatePort bool
        if len(ports) >= 1 {
                podPort = &ports[0]
                for _, port := range ports[1:] {
                        neutronApi.DeletePort(port.Id)
                }
                needUpdatePort = true
        }
        //use Id to check port is valid or not
        if podPort==nil {
                if ipSpecified != "" {
                        err = neutronApi.EnsureIpFree(network.Id, ipSpecified)
                        if err != nil {
                                return err
                        }
                }
                podPort, err = neutronApi.CreatePort(network, workload, ipSpecified, podName, namespace, subnet, securityGroupIds)
                if err != nil {
                        return err
                }
        }
        if needUpdatePort {
                podPort, err = neutronApi.UpdatePort(podPort.Id, hostname)
                if err != nil {
                        return nil
                }
        }
        fmt.Fprintf(os.Stderr, "Skynet pod port is %v\n", podPort)
        podIP := podPort.FixedIPS[0]
        if subnet==nil {
                subnet, err = neutronApi.GetSubnet(podIP.SubnetId)
                fmt.Fprintln(os.Stderr,subnet,&subnet)
        }
        fmt.Fprintf(os.Stderr, "Skynet begin to setup container interface\n")
        err = plugin.SetupInterface(args, podName, namespace, network, subnet, podPort, podIP, conf)
        if err != nil {
                fmt.Fprintf(os.Stderr, "Skynet setup container with err %v\n", err)
                neutronApi.DeletePort(podPort.Id)
                return err
        }
        //_, ipNet, _ := net.ParseCIDR(podIP.IpAddress + "/24")
        result := types.Result{}
        //kubelt 1.3.2 requires must output to stdout
        data, _ := json.Marshal(result)
        fmt.Fprintf(os.Stdout, string(data))
        return nil
}

func cmdDel(args *skel.CmdArgs) error {
        conf := util.NetConf{}
        if err := json.Unmarshal(args.StdinData, &conf); err != nil {
                return fmt.Errorf("failed to load netconf: %v", err)
        }

        util.ConfigureLogging(conf.LogLevel)

        workload, _, _, err := util.GetPortIdentifier(args)
        if err != nil {
                return err
        }

        logger := util.CreateContextLogger(workload)

        // Allow the hostname to be overridden by the network config
        if conf.Hostname != "" {
                hostname = conf.Hostname
        }

        logger.WithFields(log.Fields{
                "Workload":     workload,
                "Node":         hostname,
                "Conf":conf,
        }).Info("Extracted identifiers")

        // Always try to release the address. Don't deal with any errors till the endpoints are cleaned up.
        fmt.Fprintf(os.Stderr, "Skynet CNI releasing IP address\n")
        // Only try to delete the device if a namespace was passed in.
        neutronApi := neutron.NewNeutronApi(conf.Neutron.NeutronUrl)
        portId, err := neutronApi.DeletePortByName(workload)
        if err != nil {
                return err
        }
        if args.Netns != "" {
                fmt.Fprintf(os.Stderr, "Calico CNI deleting device in netns %s\n", args.Netns)
                err = plugin.DeleteInterface(args.Netns, args.IfName, portId, conf)
                if err != nil {
                        return err
                }
        }
        result := types.Result{}
        //kubelt 1.3.2 requires must output to stdout
        data, _ := json.Marshal(result)
        fmt.Fprintf(os.Stdout, string(data))
        return nil
}

func main() {
        flagSet := flag.NewFlagSet("Skynet", flag.ExitOnError)
        versionFlag := flagSet.Bool("v", false, "Display version")
        err := flagSet.Parse(os.Args[1:])
        if err != nil {
                fmt.Println(err)
                os.Exit(1)
        }
        if *versionFlag {
                fmt.Println(VERSION)
                os.Exit(0)
        }

        if err := util.AddIgnoreUnknownArgs(); err != nil {
                os.Exit(1)
        }
        versionInfo := version.PluginSupports("1.0")
        skel.PluginMain(cmdAdd, cmdDel, versionInfo)
}
