# 网络实现方案

skynet的网络实现方案如下图所示

![skynet网络方案](./skynet.png)

+ 使用neutron作为IPAM，可以根据网络的物理实现定义各种网络。
+ [**k8s2lb**](https://github.com/swordboy/k8s2lb)是与skynet配合使用的，用于将kubernetes的service映射为neutron lbaasv2中的loadbalancer，对应实现kubernetes中的负载均衡和外网连接通信。
+ **kubelet**根据实际需要进行了简单的定制，根据POD的网络，为容器注入dns server，使pod内部能够正常使用服务名、pod名称进行通信。
+ **dns-forwarder**是作为一个总的dns转发服务器存在，一旦POD的dns请求跨越了网络，则通过dns-forwarder将请求转发到对应的dhcp服务上，实现跨网络访问的能力。

# 方案限制

## neutron资源配额

neutron的port、loadbalancer等资源配额的限制，需要预先手工通过neutron修改，避免因为配额不足，导致无法创建port。示例命令

`neutron quota-update <tenant-id> --pool=-1 --network=-1 --port=-1 --loadbalancer=-1`

## dhcp端口变化

**偶尔**发生的情况：pod已经创建，但dhcp端口发生了变化。这样作为pod内部的dnsserver的地址有可能不可达，导致容器无法正常以名称与外界通信。而容器不像虚拟机，会通过dhcp服务来刷新网关地址。

***目前还没找到问题出现的原因！！！***





