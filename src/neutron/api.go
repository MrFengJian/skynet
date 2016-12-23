package neutron

import (
        "net/http"
        "io/ioutil"
        "encoding/json"
        "fmt"
        "strings"
        "os"
        "github.com/pborman/uuid"
)

const JSON_TYPE = "application/json"

var hostName string

func init() {
        hostName, _ = os.Hostname()
}

type NeutronApi struct {
        NeutronUrl string
}

func NewNeutronApi(neutronUrl string) (api *NeutronApi) {
        return &NeutronApi{NeutronUrl:neutronUrl}
}

func (api *NeutronApi) GetSubnet(subnetId string) (subnet *OpenStackSubnet, err error) {
        url := api.NeutronUrl + "/v2.0/subnets/" + subnetId
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response SubnetResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.Subnet, nil
}

func (api *NeutronApi) GetNetwork(networkId string) (network *OpenStackNetwork, err error) {
        url := api.NeutronUrl + "/v2.0/networks/" + networkId
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response NetworkResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.Network, nil
}

func (api *NeutronApi) GetPorts(portName string) (ports []OpenStackPort, err error) {
        url := api.NeutronUrl + "/v2.0/ports?name=" + portName
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response PortsResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return response.Ports, nil
}

func (api *NeutronApi) DeletePort(portId string) (err error) {
        url := api.NeutronUrl + "/v2.0/ports/" + portId
        err = callDelete(url)
        if err != nil {
                return err
        }
        return nil
}

func (api *NeutronApi) EnsureIpFree(networkId, ip string) (err error) {
        url := fmt.Sprintf("%s/v2.0/ports?fix_ips=ip_address=%s&network_id=%s", api.NeutronUrl, networkId, ip)
        resp, err := http.Get(url)
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return err
        }
        var response PortsResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return err
        }
        if len(response.Ports) > 0 {
                for _, port := range response.Ports {
                        err = api.DeletePort(port.Id)
                        if err != nil {
                                return err
                        }
                }
        }
        return nil
}

func (api *NeutronApi) getTenantSecutriyGroups(tenantId string) (securityGroups []OpenStackSecurityGroup, err error) {
        url := api.NeutronUrl + "/v2.0/security-groups?tenant_id=" + tenantId
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response SecurityGroupsResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return response.SecurityGroups, nil
}

func (api *NeutronApi) CreatePort(network *OpenStackNetwork, workload, ipSpecified, podName, namespace string, subnet *OpenStackSubnet, securityGroupIds []string) (port *OpenStackPort, err error) {
        portRequest := OpenStackPort{
                NetworkId: network.Id,
                Name:workload,
                TenantId:network.TenantId,
                DnsName:podName + "." + namespace,
                DeviceOwner:"compute:" + hostName,
                BindingHost:hostName,
                DeviceId:uuid.New(),
        }
        if len(securityGroupIds) == 0 {
                securityGroups, err := api.getTenantSecutriyGroups(network.TenantId)
                if err != nil {
                        return nil, err
                }
                for _, securityGroup := range securityGroups {
                        securityGroupIds = append(securityGroupIds, securityGroup.Id)
                }
        }
        portRequest.SecurityGroupIds = securityGroupIds
        if ipSpecified != "" {
                fixedIp := FixIP{
                        IpAddress:ipSpecified,
                }
                if subnet.Id != "" {
                        fixedIp.SubnetId = subnet.Id
                }
                portRequest.FixedIPS = []FixIP{fixedIp}
        }
        url := api.NeutronUrl + "/v2.0/ports"
        responseBody := PortResponse{Port:portRequest}
        data, err := json.Marshal(responseBody)
        if err != nil {
                return nil, err
        }
        resp, err := http.Post(url, JSON_TYPE, strings.NewReader(string(data)))
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response PortResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.Port, nil
}

func (api *NeutronApi) DeletePortByName(portName string) (portId string, err error) {
        ports, err := api.GetPorts(portName)
        if err != nil {
                return "", nil
        }
        if len(ports) > 0 {
                for _, port := range ports {
                        err = api.DeletePort(port.Id)
                        portId = port.Id
                        if err != nil {
                                return "", err
                        }
                }
        }
        return portId, nil
}

//UpdatePort updates port's binding host name
func (api *NeutronApi) UpdatePort(portId, hostName string) (port *OpenStackPort, err error) {
        portRequest := OpenStackPort{
                BindingHost:hostName,
                DeviceOwner:"compute:" + hostName,
        }
        url := api.NeutronUrl + "/v2.0/ports/" + portId
        requestBody := PortResponse{Port:portRequest}
        data, err := json.Marshal(requestBody)
        if err != nil {
                return nil, err
        }
        req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(string(data)))
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response PortResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.Port, nil
}

//GetDnsServersInNetwork query all dhcp ports in network and extract ip as dns servers for container
func (api *NeutronApi) GetDnsServersInNetwork(networkId string) (dnsServers []string, err error) {
        url := api.NeutronUrl + "/v2.0/ports?network_id=" + networkId + "&device_owner=network:dhcp"
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var portsArr PortsResponse
        err = json.Unmarshal(body, &portsArr)
        if err != nil {
                return nil, err
        }
        ports := portsArr.Ports
        for _, port := range ports {
                for _, fixIp := range port.FixedIPS {
                        dnsServers = append(dnsServers, fixIp.IpAddress)
                }
        }
        return dnsServers, nil
}

//callDelete wraps http delete request
func callDelete(url string) (err error) {
        req, err := http.NewRequest(http.MethodDelete, url, nil)
        if err != nil {
                return nil
        }
        _, err = http.DefaultClient.Do(req)
        if err != nil {
                return err
        }
        return nil
}