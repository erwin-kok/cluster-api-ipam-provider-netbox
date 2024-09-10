# Cluster API IPAM Provider In Cluster

This is an [IPAM provider](https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20220125-ipam-integration.md#ipam-provider) for [Cluster API](https://github.com/kubernetes-sigs/cluster-api) that manages pools of IP addresses
using [Netbox](https://netboxlabs.com/).

go mod init github.com/erwin-kok/cluster-api-ipam-provider-netbox

kubebuilder init --domain cluster.x-k8s.io --repo github.com/erwin-kok/cluster-api-ipam-provider-netbox --project-name cluster-api-ipam-provider-netbox

kubebuilder create api --group ipam --version v1alpha1 --kind NetboxIPPool --controller=true --resource=true
kubebuilder create webhook --group ipam --version v1alpha1 --kind NetboxIPPool --programmatic-validation --defaulting

kubebuilder create api --group ipam --version v1alpha1 --kind NetboxGlobalIPPool --controller=true --resource=true
kubebuilder create webhook --group ipam --version v1alpha1 --kind NetboxGlobalIPPool --programmatic-validation --defaulting







// resourceNetboxIPAddress
// Create: resourceNetboxIPAddressCreate,
// Read:   resourceNetboxIPAddressRead,
// Update: resourceNetboxIPAddressUpdate,
// Delete: resourceNetboxIPAddressDelete,
// resource "netbox_ip_address" "node_management_ip" {
// 	ip_address   = format("%s/%d", local.node.static_management_ip, split("/", local.netbox.management_network_prefix)[1])
// 	status       = local.netbox_ip_address_status
// 	tenant_id    = data.netbox_tenant.management_tenant.id
// 	interface_id = netbox_interface.node_management_nic.id
// 	object_type  = local.netbox_ip_address_object_type
// 	description  = format(local.netbox_ip_description_template, "management", local.node.name)
// 	tags         = concat(local.netbox_common_tags, [local.netbox_management_tag])
// }
//
// // resourceNetboxAvailableIPAddress
// // Create: resourceNetboxAvailableIPAddressCreate,
//
//
//
// // Read:   resourceNetboxAvailableIPAddressRead,
// // Update: resourceNetboxAvailableIPAddressUpdate,
// // Delete: resourceNetboxAvailableIPAddressDelete,
// resource "netbox_available_ip_address" "node_management_ip" {
// 	prefix_id    = data.netbox_prefix.management_network_prefix.id
// 	status       = local.netbox_ip_address_status
// 	tenant_id    = data.netbox_tenant.management_tenant.id
// 	interface_id = netbox_interface.node_management_nic.id
// 	object_type  = local.netbox_ip_address_object_type
// 	description  = format(local.netbox_ip_description_template, "management", local.node.name)
// 	tags         = concat(local.netbox_common_tags, [local.netbox_management_tag])
// }
//
//
//
//
// IpamPrefixesAvailableIpsCreate
// IpamIPRangesAvailableIpsCreate
//
// IpamIPAddressesList
