[TOC]

# 概述

skynet插件实现了[cni](https://github.com/containernetworking/cni)接口，实现了cni与neutron的对接。并可以利用neutron的网络管理功能、安全组功能实现容器网络的进一步隔离定制。skynet的整体网络实现方案参考[设计文档](./docs/design.md)。

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
         "service_subnet_enabled": false,
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

  + neutron_url：neutron-server的访问地址。
  + default_network_id：在不指定网络的情况下，默认为POD设置的网络。
  + default_subnet_id：在不指定网络情况下，默认使用的POD子网，与default_subnet_id同时指定时会覆盖default_network_id的配置。
  + default_security_group_ids：在不指定安全组的情况下，默认使用的安全组。为空字符串时，使用对应租户的所有安全组。
  + service_subnet_enabled：是否启用将kubernetes的服务IP映射为真实IP，默认为false。最好在网络为vxlan的情况下进行配置。
  + external_router_gateway_ip：启用映射服务IP为真实IP情况下，所有子网都连接到一个开放了外网网关的router上，external_router_gateway即为router网关地址，并且要求计算节点上有一张网卡与网关地址在同一个网段。
  + service_cluster_ip_cidr：kubernetes服务IP的CIDR范围。

  > TBD：现在neutron网络节点为noauth模式，支持keystone方式的待开发。

+ plugin：网络物理实现的定义。

  + plugin_type：网络的物理实现，支持linuxbridge、macvlan、openvswitch三个值。
  + trunk_nic：物理机上的trunk网卡名称，用于支持多vlan情况。
  + tunnel_nic：物理机上的tunnel网卡名称，用于vxlan隧道的配置。

+ kubernetes：访问kubernetes的方式。

  + k8s_api_root：kubernetes apiserver的访问地址。

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



# how to build

​	基于go语言实现，直接通过go就可以build。

```shell
mkdir src&&git clone https://github.com/swordboy/skynet.git
export GOPATH=$(pwd)
cd src/skynet &&go build skynet.go
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

+    skynet/network_id：指定网络。

+    skynet/subnet_id：指定子网，如果网络和子网都指定，则子网会覆盖掉指定的网络。

+    skynet/ip：指定IP。如果IP已使用，自动尝试删除对应端口，不保证删除成功。

+    skynet/security_group_ids：指定Pod安全组列表，多个安全组ID以英文逗号分割。默认使用`20-skynet.conf`中指定的`default_security_group_ids`。

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
# 如何启用安全组

​	在skynet实现方案中，安全组功能的实现需要依赖neutron的linuxbridge-agent，参考[如何启用安全组](./docs/howto_enable_security_group.md)。

​	