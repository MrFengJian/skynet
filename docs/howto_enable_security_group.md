# ���ð�ȫ��

��ʹ��linuxbridgeʵ�ֵ�����£�������neutron��linuxbridge-agent��ϣ�ʵ���������ڰ�ȫ���������롣������ʹ��vxlan����ʱ��Ҳ��Ҫneutron��linuxbridge-agent���ɵ�fdb������ʵ������ͨ�š�

˵����ô��skynet��չ�Ļ����ϣ�ʵ�ְ�ȫ������vxlanͨ��

##��װlinuxbridge-agent
    yum install http://rdo.fedorapeople.org/openstack-kilo/rdo-release-kilo.rpm
    yum install openstack-neutron-linuxbridge ebtables ipset -y

����openstack rpm�������⣬������Ҫ�ֶ�����/etc/neutron/policy.json�ļ����������£�

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



##����sysctl
  ����centos�°�װ��bug����Ҫ�ֹ��������ò���ʹ��ȫ�������Ч���޸�/etc/sysctl.conf������������������
    net.bridge.bridge-nf-call-arptables = 1
    net.bridge.bridge-nf-call-iptables = 1
    net.bridge.bridge-nf-call-ip6tables = 1
  Ȼ��ִ��`systctl -p`ʹ������Ч

##����rootwrap
  �ֹ����ļ�`/usr/share/neutron/rootwrap/iptables-firewall.filters`�У������������ã�ʹebtables�����ܹ�������neutron���á�
    ebtables: CommandFilter, ebtables, root

##����linuxbridge-agent
  ���������ã��޸�/etc/neutron/neutron.conf��/etc/neutron/plugins/linuxbridge/linuxbridge_conf.ini��
###neutron.confʾ������
  ʾ�������������£�

```ini
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

###linuxbridge_conf.iniʾ������
  linuxbridge_conf.ini��ʾ���������£�

```ini
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