[TOC]

# 概述

skynet插件实现了[cni](https://github.com/containernetworking/cni)接口，实现了cni与neutron的对接。并可以利用neutron的网络管理功能、安全组功能实现容器网络的进一步隔离定制。

# skynet配置

​	遵循cni接口协议的规范，skynet网络的配置文件如下所示：

```json
{
	"name": "skynet",
	"type": "skynet",
	"neutron": {
		"neutron_url": "http://192.168.7.211:9696",
		"default_network_id": "e5e7478a-53ec-4063-a4dc-f3ca3ba3e98b",
		"default_security_group_ids": "31a8f352-580e-4136-9f91-fa8829aa179c",
		"external_router_gateway_ip": "192.168.7.160",
		"service_cluster_ip_cidr": "10.254.0.0/16"
	},
	"plugin": {
		"plugin_type": "linuxbridge",
		"trunk_nic": "ens192",
		"tunnel_nic": "ens160"
	},
	"kubernetes": {
		"k8s_api_root": "http://192.168.7.203:8080"
	},
	"log_level": "info",
	"cniVersion": "1.0"
}
```

​	其中：

+ name：插件名称定义。

+ type：插件类型，cni插件机制根据此值来检索插件可执行文件。

+ neutron：连接网络节点的方式，以及默认的网络配置。

  > TBD：现在neutron网络节点为noauth模式，支持keystone方式的待开发。

+ plugin：网络物理实现的定义。

+ kubernetes：访问kubernetes的方式。

  > TBD：目前仅支持http无验证方式访问，基于认证和https的访问待开发。

+ log_level：日志级别。

+ cniVersion：调用的cni实现版本。

# 功能支持矩阵

下面是几种skynet实现的基于不同网络实现的功能支持矩阵。<u>其中linuxbridge通过与neutron的linuxbridge-agent结合，来实现vxlan和安全组功能。</u>

|         | linuxbridge | openvswitch | macvlan |
| ------- | ----------- | :---------- | ------- |
| flat网络  | √           | √           | √       |
| vlan网络  | √           | √           | √       |
| vxlan网络 | √           | ×           | ×       |
| 安全组     | √           | ×           | ×       |



# 编译

​	基于go语言实现，直接通过go就可以build。

```shell
git clone https://github.com/swordboy/skynet.git
export GOPATH=$(pwd)/skynet
cd skynet/src &&go build skynet.go
```

 	生成的二进制文件即可使用。

# 如何使用skynet

​	由于skynet基于skynet实现了cni接口，但目前仅针对kubernetes的配置进行实现，因此必须与kubernetes集群结合使用。

## 配置neutron网络节点

​	基于下列命令安装neutron网络节点

```shell
yum install centos-release-openstack-mitaka -y
yum install bridge-utils net-tools openstack-neutron-linuxbridge ebtables ipset \
openstack-neutron-lbaas haproxy -y
```

​	可以参考OpenStack[官方安装文档](http://docs.openstack.org/mitaka/install-guide-rdo/neutron.html)，将neutron-server配置为noauth模式。

​	最终配置可以参考下面启用安全组时的配置。

## 配置kubelet

1. 修改kubelet的启动参数，增加下面两项，指定启用cni。

   `--network-plugin-dir=/etc/cni/net.d --network-plugin=cni`

2. 将skynet可执行文件放到`/opt/cni/bin`目录中。

3. 将`20-skynet.conf`放到目录`/etc/cni/net.d`中，其内容参考skynet配置章节的说明，根据实际环境修改。

4. 重启kubelet即可启用。

## 配置Pod的网络

​	skynet通过Pod上的以下注解来指定网络配置。

+ skynet/network_id：指定网络。

+ skynet/subnet_id：指定子网，如果网络和子网都指定，则子网会覆盖掉指定的网络。

+ skynet/ip：指定IP。如果IP已使用，自动尝试删除对应端口，不保证删除成功。

+ skynet/security_group_ids：指定Pod安全组列表，多个安全组ID以英文逗号分割。默认使用`20-skynet.conf`中指定的`default_security_group_ids`。

  pod指定子网示例：

  ```yaml
  apiVersion: v1
  kind: ReplicationController
  metadata:
    name: neutron-test
    labels:
      app: neutron-test
  spec:
    replicas: 1
    template:
      metadata:
        name: neutron-test
        annotations:
          skynet/subnet_id: 76aa33bc-c9c1-4834-bcfc-aefd28206997
        labels:
          app: neutron-test
      spec:
        terminationGracePeriodSeconds: 0
        containers:
          - image: ubuntu:14.04.4
            env:
            - name: ROOT_PASS
              value: password
            name: busybox
            imagePullPolicy: IfNotPresent
  ```

  ​

# 启用安全组

在使用linuxbridge实现的情况下，可以与neutron的linuxbridge-agent结合，实现容器基于安全组的网络隔离。并且在使用vxlan网络时，也需要neutron的linuxbridge-agent生成的fdb表，用于实现容器通信。

说明怎么在skynet扩展的基础上，实现安全组隔离和vxlan通信

##安装linuxbridge-agent
    yum install http://rdo.fedorapeople.org/openstack-kilo/rdo-release-kilo.rpm
    yum install openstack-neutron-linuxbridge ebtables ipset -y

由于openstack rpm包的问题，我们需要手动创建/etc/neutron/policy.json文件，内容如下：

```json
{
    "context_is_admin":  "role:admin",
    "admin_or_owner": "rule:context_is_admin or tenant_id:%(tenant_id)s",
    "context_is_advsvc":  "role:advsvc",
    "admin_or_network_owner": "rule:context_is_admin or tenant_id:%(network:tenant_id)s",
    "admin_only": "rule:context_is_admin",
    "regular_user": "",
    "shared": "field:networks:shared=True",
    "shared_firewalls": "field:firewalls:shared=True",
    "shared_firewall_policies": "field:firewall_policies:shared=True",
    "shared_subnetpools": "field:subnetpools:shared=True",
    "external": "field:networks:router:external=True",
    "default": "rule:admin_or_owner",

    "create_subnet": "rule:admin_or_network_owner",
    "get_subnet": "rule:admin_or_owner or rule:shared",
    "update_subnet": "rule:admin_or_network_owner",
    "delete_subnet": "rule:admin_or_network_owner",

    "create_subnetpool": "",
    "create_subnetpool:shared": "rule:admin_only",
    "get_subnetpool": "rule:admin_or_owner or rule:shared_subnetpools",
    "update_subnetpool": "rule:admin_or_owner",
    "delete_subnetpool": "rule:admin_or_owner",

    "create_network": "",
    "get_network": "rule:admin_or_owner or rule:shared or rule:external or rule:context_is_advsvc",
    "get_network:router:external": "rule:regular_user",
    "get_network:segments": "rule:admin_only",
    "get_network:provider:network_type": "rule:admin_only",
    "get_network:provider:physical_network": "rule:admin_only",
    "get_network:provider:segmentation_id": "rule:admin_only",
    "get_network:queue_id": "rule:admin_only",
    "create_network:shared": "rule:admin_only",
    "create_network:router:external": "rule:admin_only",
    "create_network:segments": "rule:admin_only",
    "create_network:provider:network_type": "rule:admin_only",
    "create_network:provider:physical_network": "rule:admin_only",
    "create_network:provider:segmentation_id": "rule:admin_only",
    "update_network": "rule:admin_or_owner",
    "update_network:segments": "rule:admin_only",
    "update_network:shared": "rule:admin_only",
    "update_network:provider:network_type": "rule:admin_only",
    "update_network:provider:physical_network": "rule:admin_only",
    "update_network:provider:segmentation_id": "rule:admin_only",
    "update_network:router:external": "rule:admin_only",
    "delete_network": "rule:admin_or_owner",

    "network_device": "field:port:device_owner=~^network:",
    "create_port": "",
    "create_port:device_owner": "not rule:network_device or rule:admin_or_network_owner or rule:context_is_advsvc",
    "create_port:mac_address": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "create_port:fixed_ips": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "create_port:port_security_enabled": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "create_port:binding:host_id": "rule:admin_only",
    "create_port:binding:profile": "rule:admin_only",
    "create_port:mac_learning_enabled": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "create_port:allowed_address_pairs": "rule:admin_or_network_owner",
    "get_port": "rule:admin_or_owner or rule:context_is_advsvc",
    "get_port:queue_id": "rule:admin_only",
    "get_port:binding:vif_type": "rule:admin_only",
    "get_port:binding:vif_details": "rule:admin_only",
    "get_port:binding:host_id": "rule:admin_only",
    "get_port:binding:profile": "rule:admin_only",
    "update_port": "rule:admin_or_owner or rule:context_is_advsvc",
    "update_port:device_owner": "not rule:network_device or rule:admin_or_network_owner or rule:context_is_advsvc",
    "update_port:mac_address": "rule:admin_only or rule:context_is_advsvc",
    "update_port:fixed_ips": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "update_port:port_security_enabled": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "update_port:binding:host_id": "rule:admin_only",
    "update_port:binding:profile": "rule:admin_only",
    "update_port:mac_learning_enabled": "rule:admin_or_network_owner or rule:context_is_advsvc",
    "update_port:allowed_address_pairs": "rule:admin_or_network_owner",
    "delete_port": "rule:admin_or_owner or rule:context_is_advsvc",

    "get_router:ha": "rule:admin_only",
    "create_router": "rule:regular_user",
    "create_router:external_gateway_info:enable_snat": "rule:admin_only",
    "create_router:distributed": "rule:admin_only",
    "create_router:ha": "rule:admin_only",
    "get_router": "rule:admin_or_owner",
    "get_router:distributed": "rule:admin_only",
    "update_router:external_gateway_info:enable_snat": "rule:admin_only",
    "update_router:distributed": "rule:admin_only",
    "update_router:ha": "rule:admin_only",
    "delete_router": "rule:admin_or_owner",

    "add_router_interface": "rule:admin_or_owner",
    "remove_router_interface": "rule:admin_or_owner",

    "create_router:external_gateway_info:external_fixed_ips": "rule:admin_only",
    "update_router:external_gateway_info:external_fixed_ips": "rule:admin_only",

    "create_firewall": "",
    "get_firewall": "rule:admin_or_owner",
    "create_firewall:shared": "rule:admin_only",
    "get_firewall:shared": "rule:admin_only",
    "update_firewall": "rule:admin_or_owner",
    "update_firewall:shared": "rule:admin_only",
    "delete_firewall": "rule:admin_or_owner",

    "create_firewall_policy": "",
    "get_firewall_policy": "rule:admin_or_owner or rule:shared_firewall_policies",
    "create_firewall_policy:shared": "rule:admin_or_owner",
    "update_firewall_policy": "rule:admin_or_owner",
    "delete_firewall_policy": "rule:admin_or_owner",

    "create_firewall_rule": "",
    "get_firewall_rule": "rule:admin_or_owner or rule:shared_firewalls",
    "update_firewall_rule": "rule:admin_or_owner",
    "delete_firewall_rule": "rule:admin_or_owner",

    "create_qos_queue": "rule:admin_only",
    "get_qos_queue": "rule:admin_only",

    "update_agent": "rule:admin_only",
    "delete_agent": "rule:admin_only",
    "get_agent": "rule:admin_only",

    "create_dhcp-network": "rule:admin_only",
    "delete_dhcp-network": "rule:admin_only",
    "get_dhcp-networks": "rule:admin_only",
    "create_l3-router": "rule:admin_only",
    "delete_l3-router": "rule:admin_only",
    "get_l3-routers": "rule:admin_only",
    "get_dhcp-agents": "rule:admin_only",
    "get_l3-agents": "rule:admin_only",
    "get_loadbalancer-agent": "rule:admin_only",
    "get_loadbalancer-pools": "rule:admin_only",
    "get_agent-loadbalancers": "rule:admin_only",
    "get_loadbalancer-hosting-agent": "rule:admin_only",

    "create_floatingip": "rule:regular_user",
    "create_floatingip:floating_ip_address": "rule:admin_only",
    "update_floatingip": "rule:admin_or_owner",
    "delete_floatingip": "rule:admin_or_owner",
    "get_floatingip": "rule:admin_or_owner",

    "create_network_profile": "rule:admin_only",
    "update_network_profile": "rule:admin_only",
    "delete_network_profile": "rule:admin_only",
    "get_network_profiles": "",
    "get_network_profile": "",
    "update_policy_profiles": "rule:admin_only",
    "get_policy_profiles": "",
    "get_policy_profile": "",

    "create_metering_label": "rule:admin_only",
    "delete_metering_label": "rule:admin_only",
    "get_metering_label": "rule:admin_only",

    "create_metering_label_rule": "rule:admin_only",
    "delete_metering_label_rule": "rule:admin_only",
    "get_metering_label_rule": "rule:admin_only",

    "get_service_provider": "rule:regular_user",
    "get_lsn": "rule:admin_only",
    "create_lsn": "rule:admin_only"
}
```

##配置sysctl
  由于centos下安装的bug，需要手工增加配置才能使安全组规则生效。修改/etc/sysctl.conf，增加如下三行配置
    net.bridge.bridge-nf-call-arptables = 1
    net.bridge.bridge-nf-call-iptables = 1
    net.bridge.bridge-nf-call-ip6tables = 1
  然后执行`systctl -p`使配置生效

##配置rootwrap
  手工在文件`/usr/share/neutron/rootwrap/iptables-firewall.filters`中，增加如下配置，使ebtables命令能够正常被neutron调用。
    ebtables: CommandFilter, ebtables, root

##配置linuxbridge-agent
  按正常配置，修改/etc/neutron/neutron.conf和/etc/neutron/plugins/linuxbridge/linuxbridge_conf.ini。
###neutron.conf示例配置
  示例配置内容如下：

    [DEFAULT]
    verbose = True
    debug = True
    core_plugin=ml2
    host=slave1
    notification_driver = neutron.openstack.common.notifier.rpc_notifier
    interface_driver = neutron.agent.linux.interface.BridgeInterfaceDriver
    rpc_backend=rabbit
    [agent]
    root_helper=sudo neutron-rootwrap /etc/neutron/rootwrap.conf
    [oslo_concurrency]
    lock_path = $state_path/lock
    [oslo_messaging_rabbit]
    rabbit_host=192.168.7.213
    rabbit_port=5672
    rabbit_userid = openstack
    rabbit_password = password
    [keystone_authtoken]
    auth_uri = http://192.168.7.213:5000
    auth_url = http://192.168.7.213:35357
    auth_plugin = password
    project_domain_id = default
    user_domain_id = default
    project_name = service
    username = neutron
    password = password

###linuxbridge_conf.ini示例配置
  linuxbridge_conf.ini的示例配置如下：

    [ml2]
    type_drivers= local,flat,vlan,vxlan
    tenant_network_types = vlan,vxlan,flat
    mechanism_drivers = linuxbridge,l2population
    [vlans]
    tenant_network_type=vlan
    network_vlan_ranges = physnet2:100:2999
    [linux_bridge]
    physical_interface_mappings=physnet2:eth1
    [vxlan]
    enable_vxlan=True
    vxlan_group = None
    local_ip=192.168.7.207
    l2_population = True
    [agent]
    prevent_arp_spoofing = True
    [securitygroup]
    firewall_driver = neutron.agent.linux.iptables_firewall.IptablesFirewallDriver
    enable_security_group = True
    enable_ipset=True


​	