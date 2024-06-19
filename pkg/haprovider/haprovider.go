// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package haprovider

import (
	"context"
	"net"
	"sync"

	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/utils"

	"github.com/pkg/errors"

	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	akov1beta1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1beta1"
)

const (
	IPv4IpFamily = "IPv4"
	IPv6IpFamily = "IPv6"
	IPv6IpType   = "V6"
)

type HAProvider struct {
	client.Client
	log logr.Logger
}

var (
	instance  *HAProvider
	once      sync.Once
	QueryFQDN = queryFQDNEndpoint
)

// NewProvider make HAProvider as a singleton
func NewProvider(c client.Client, log logr.Logger) *HAProvider {
	once.Do(func() {
		instance = &HAProvider{
			Client: c,
			log:    log,
		}
	})
	return instance
}

func (r *HAProvider) getHAServiceName(cluster *clusterv1.Cluster) string {
	return cluster.Namespace + "-" + cluster.Name + "-" + akoov1alpha1.HAServiceName
}

func (r *HAProvider) CreateOrUpdateHAService(ctx context.Context, cluster *clusterv1.Cluster) error {
	serviceName := r.getHAServiceName(cluster)
	service := &corev1.Service{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      serviceName,
		Namespace: cluster.Namespace,
	}, service); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Info(serviceName + " service doesn't exist, start creating it...")
			service, err = r.createService(ctx, cluster)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if err := r.updateClusterControlPlaneEndpoint(cluster, service); err != nil {
		return err
	}

	if err := r.updateControlPlaneEndpointToService(ctx, cluster, service); err != nil {
		return err
	}

	if _, err := r.ensureEndpoints(ctx, serviceName, service.Namespace); err != nil {
		return err
	}
	return nil
}

func (r *HAProvider) createService(
	ctx context.Context,
	cluster *clusterv1.Cluster,
) (*corev1.Service, error) {
	serviceName := r.getHAServiceName(cluster)

	serviceAnnotations, err := r.annotateService(ctx, cluster)
	if err != nil {
		return nil, err
	}

	port, err := ako_operator.GetControlPlaneEndpointPort(cluster)
	if err != nil {
		return nil, err
	}

	// Get cluster primary ip family, which is used for HA service
	primaryIPFamily, err := utils.GetPrimaryIPFamily(cluster)
	if err != nil {
		return nil, err
	}
	if primaryIPFamily == IPv6IpType {
		primaryIPFamily = IPv6IpFamily
	} else {
		primaryIPFamily = IPv4IpFamily
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   cluster.Namespace,
			Annotations: serviceAnnotations,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			// TODO:(chenlin) Add two ip families after AKO fully supports dual-stack load balancer type of service
			IPFamilies: []corev1.IPFamily{corev1.IPFamily(primaryIPFamily)},
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       port,
					TargetPort: intstr.FromInt(int(6443)),
				},
			},
		},
	}
	// Add Finalizer on Management Cluster's service to avoid being deleted.
	if cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		ctrlutil.AddFinalizer(service, akoov1alpha1.HAServiceBootstrapClusterFinalizer)
	} else {
		service.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: cluster.APIVersion,
				Kind:       cluster.Kind,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		}
	}

	if endpoint, err := ako_operator.GetControlPlaneEndpoint(cluster); err != nil {
		r.log.Error(err, "can't unmarshal cluster variables ", "endpoint", endpoint)
		return nil, err
	} else if endpoint != "" {
		// "endpoint" can be ipv4/ipv6 or hostname, add ipv4/ipv6 or hostname as annotation: ako.vmware.com/load-balancer-ip:<ip>
		// doesn't support ipv6 endpoint because of AKO limitation: https://avinetworks.com/docs/ako/1.10/support-for-ipv6-in-ako/
		if net.ParseIP(endpoint) == nil {
			endpoint, err = QueryFQDN(endpoint)
			if err != nil {
				r.log.Error(err, "Failed to resolve control plane endpoint ", "endpoint", endpoint)
				return nil, err
			}
		}
		// update the load balancer ip spec & annotation as intermediate plan to
		// tolerant older and newer version TKr
		service.Spec.LoadBalancerIP = endpoint
		service.Annotations[akoov1alpha1.AkoPreferredIPAnnotation] = endpoint
	}
	r.log.Info("Creating " + serviceName + " Service")
	err = r.Create(ctx, service)
	return service, err
}

func (r *HAProvider) annotateService(ctx context.Context, cluster *clusterv1.Cluster) (map[string]string, error) {
	serviceAnnotation := map[string]string{
		akoov1alpha1.HAServiceAnnotationsKey:  "true",
		akoov1alpha1.TKGClusterNameLabel:      cluster.Name,
		akoov1alpha1.TKGClusterNameSpaceLabel: cluster.Namespace,
	}

	adcForCluster, err := r.getADCForCluster(ctx, cluster)
	if err != nil {
		return serviceAnnotation, err
	}
	// no adc is selected for cluster, no annotation is needed.
	if adcForCluster == nil {
		// for the management cluster, it needs to requeue until the install-ako-for-management-cluster AKODeploymentConfig created
		if _, ok := cluster.Labels[akoov1alpha1.TKGManagememtClusterRoleLabel]; ok {
			return serviceAnnotation, errors.New("management cluster's AKODeploymentConfig didn't find, requeue to wait for AKODeploymentConfig created")
		}
		return serviceAnnotation, nil
	}

	aviInfraSetting, err := r.getAviInfraSettingFromAdc(ctx, adcForCluster)
	if err != nil {
		return serviceAnnotation, err
	}

	if _, ok := cluster.Labels[akoov1alpha1.TKGManagememtClusterRoleLabel]; ok {
		if adcForCluster.Spec.ControlPlaneNetwork.CIDR != "" && adcForCluster.Spec.ControlPlaneNetwork.CIDR != adcForCluster.Spec.DataNetwork.CIDR {
			if aviInfraSetting == nil {
				return serviceAnnotation, errors.New("management cluster control plane network set, but corresponding AVIInfraSetting not found, requeue to wait for AVIInfraSetting created")
			}
		}
	}
	if aviInfraSetting != nil {
		// add AVIInfraSetting annotation when creating HA svc
		serviceAnnotation[akoov1alpha1.HAAVIInfraSettingAnnotationsKey] = aviInfraSetting.Name
	}
	return serviceAnnotation, nil
}

func (r *HAProvider) getADCForCluster(ctx context.Context, cluster *clusterv1.Cluster) (*akoov1alpha1.AKODeploymentConfig, error) {
	adcForCluster, err := ako_operator.GetAKODeploymentConfigForCluster(ctx, r.Client, r.log, cluster)
	if err != nil {
		return nil, err
	}
	if adcForCluster == nil {
		r.log.Info("Current cluster is not selected by any akoDeploymentConfig, skip adding AviInfraSetting annotation")
	}
	return adcForCluster, nil
}

func (r *HAProvider) getAviInfraSettingFromAdc(ctx context.Context, adcForCluster *akoov1alpha1.AKODeploymentConfig) (*akov1beta1.AviInfraSetting, error) {
	aviInfraSetting := &akov1beta1.AviInfraSetting{}
	aviInfraSettingName := GetAviInfraSettingName(adcForCluster)
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: aviInfraSettingName,
	}, aviInfraSetting); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Info(aviInfraSettingName + " not found, skip adding annotation to HA service...")
			return nil, nil
		} else {
			r.log.Error(err, "Failed to get AVIInfraSetting, requeue")
			return nil, err
		}
	} else {
		return aviInfraSetting, nil
	}
}

func (r *HAProvider) updateClusterControlPlaneEndpoint(cluster *clusterv1.Cluster, service *corev1.Service) error {
	endpoint, _ := ako_operator.GetControlPlaneEndpoint(cluster)
	// Dakar Limitation: customers ensure the service engine is running
	ingress := service.Status.LoadBalancer.Ingress
	if len(ingress) > 0 && net.ParseIP(ingress[0].IP) != nil {
		if endpoint != "" && net.ParseIP(endpoint) == nil {
			cluster.Spec.ControlPlaneEndpoint.Host = endpoint
		} else {
			cluster.Spec.ControlPlaneEndpoint.Host = service.Status.LoadBalancer.Ingress[0].IP
			ako_operator.SetControlPlaneEndpoint(cluster, service.Status.LoadBalancer.Ingress[0].IP)
		}
		port, err := ako_operator.GetControlPlaneEndpointPort(cluster)
		cluster.Spec.ControlPlaneEndpoint.Port = port
		return err
	}
	return errors.New(service.Name + " service external ip is not ready")
}

func (r *HAProvider) updateControlPlaneEndpointToService(ctx context.Context, cluster *clusterv1.Cluster, service *corev1.Service) error {
	host := cluster.Spec.ControlPlaneEndpoint.Host
	var err error
	if net.ParseIP(host) == nil {
		host, err = QueryFQDN(host)
		if err != nil {
			r.log.Error(err, "Failed to resolve control plane endpoint ", "endpoint", host)
			return err
		}
	}
	service.Spec.LoadBalancerIP = host
	if service.Annotations == nil {
		service.Annotations = make(map[string]string)
	}
	service.Annotations[akoov1alpha1.AkoPreferredIPAnnotation] = host
	if err := r.Update(ctx, service); err != nil {
		return errors.Wrapf(err, "Failed to update cluster endpoint to cluster control plane load balancer type of service <%s>\n", service.Name)
	}
	return nil
}

func (r *HAProvider) syncEndpointMachineIP(address corev1.EndpointAddress, machine *clusterv1.Machine) corev1.EndpointAddress {
	// check if ip is still in the machine's address
	found := false
	for _, machineAddress := range machine.Status.Addresses {
		if machineAddress.Type != clusterv1.MachineExternalIP {
			continue
		}
		if net.ParseIP(machineAddress.Address) != nil && address.IP == machineAddress.Address {
			found = true
		}
	}

	if !found {
		// machine address is not sync with the ip address in endpoints object
		// update endpoints object
		for _, machineAddress := range machine.Status.Addresses {
			if machineAddress.Type != clusterv1.MachineExternalIP {
				continue
			}
			if net.ParseIP(machineAddress.Address) != nil {
				address.IP = machineAddress.Address
				r.log.Info("sync endpoints object, update machine: " + machine.Name + "'s ip to:" + address.IP)
			} else {
				r.log.Info(machineAddress.Address + " is not a valid IP address")
			}
		}
	}
	return address
}

func (r *HAProvider) removeMachineIpFromEndpoints(endpoints *corev1.Endpoints, machine *clusterv1.Machine) {
	if endpoints.Subsets == nil || len(endpoints.Subsets) == 0 {
		r.log.Info("currentEndpoints.Subsets is already empty, skip")
		return
	}
	newAddresses := make([]corev1.EndpointAddress, 0)
	for _, address := range endpoints.Subsets[0].Addresses {
		// skip the machine should be deleted
		if address.NodeName != nil && *address.NodeName == machine.Name {
			continue
		}
		newAddresses = append(newAddresses, address)
	}
	endpoints.Subsets[0].Addresses = newAddresses
	// remove the Subset if "Addresses" is emtpy
	if len(endpoints.Subsets[0].Addresses) == 0 {
		endpoints.Subsets = nil
	}
}

func (r *HAProvider) addMachineIpToEndpoints(endpoints *corev1.Endpoints, machine *clusterv1.Machine, ipFamily string) {
	if endpoints.Subsets == nil {
		// create a Subset if Endpoint doesn't have one
		endpoints.Subsets = []corev1.EndpointSubset{{
			Addresses: make([]corev1.EndpointAddress, 0),
			Ports: []corev1.EndpointPort{{
				Port:     6443,
				Protocol: "TCP",
			}},
		}}
	} else {
		// check if machine has already been added to Endpoints
		for i, address := range endpoints.Subsets[0].Addresses {
			if address.NodeName != nil && *address.NodeName == machine.Name {
				r.log.Info("machine is in Endpoints Object")
				endpoints.Subsets[0].Addresses[i] = r.syncEndpointMachineIP(address, machine)
				return
			}
		}
	}
	// add a new machine to Endpoints
	for _, machineAddress := range machine.Status.Addresses {
		if machineAddress.Type != clusterv1.MachineExternalIP {
			continue
		}
		// check machineAddress.Address is valid
		// Support IPv4 and IPv6
		if net.ParseIP(machineAddress.Address) != nil {
			newAddress := corev1.EndpointAddress{
				IP:       machineAddress.Address,
				NodeName: &machine.Name,
			}
			// Validate MachineIP before adding to Endpoint
			if ipFamily == "V6" {
				if net.ParseIP(machineAddress.Address).To4() == nil {
					endpoints.Subsets[0].Addresses = append(endpoints.Subsets[0].Addresses, newAddress)
					break
				}
			} else if ipFamily == "V4" {
				if net.ParseIP(machineAddress.Address).To4() != nil {
					endpoints.Subsets[0].Addresses = append(endpoints.Subsets[0].Addresses, newAddress)
					break
				}
			}
		} else {
			r.log.Info(machineAddress.Address + " is not a valid IP address")
		}
	}
}

func (r *HAProvider) CreateOrUpdateHAEndpoints(ctx context.Context, machine *clusterv1.Machine) error {
	// return if it's not a control plane machine
	if _, ok := machine.ObjectMeta.Labels[clusterv1.MachineControlPlaneLabel]; !ok {
		r.log.Info("not a control plane machine, skip")
		return nil
	}

	// get endpoint name (cluster namespace and name)
	cluster := &clusterv1.Cluster{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      machine.Spec.ClusterName,
		Namespace: machine.Namespace,
	}, cluster); err != nil {
		r.log.Error(err, "Failed to get the cluster of "+machine.Name)
		return err
	}

	endpoints, err := r.ensureEndpoints(ctx, r.getHAServiceName(cluster), cluster.Namespace)
	if err != nil {
		r.log.Error(err, "Failed to get the Endpoints object of current cluster HA Service")
		return err
	}

	// Get control plane endpoint ip family
	adcForCluster, err := r.getADCForCluster(ctx, cluster)
	if err != nil {
		r.log.Error(err, "Failed to get cluster AKODeploymentConfig")
		return err
	}
	ipFamily := "V4"
	if adcForCluster != nil && adcForCluster.Spec.ExtraConfigs.IpFamily != "" {
		ipFamily = adcForCluster.Spec.ExtraConfigs.IpFamily
	}
	if !machine.DeletionTimestamp.IsZero() {
		r.log.Info("machine" + machine.Name + " is being deleted, remove the endpoint of the machine from " + r.getHAServiceName(cluster) + " Endpoints")
		r.removeMachineIpFromEndpoints(endpoints, machine)
	} else {
		// Add machine ip to the Endpoints object no matter it's ready or not
		// Because avi controller checks the status of machine. If it's not ready, avi won't use it as an endpoint
		r.addMachineIpToEndpoints(endpoints, machine, ipFamily)
	}
	if err := r.Update(ctx, endpoints); err != nil {
		return errors.Wrapf(err, "Failed to update endpoints <%s>, control plane machine IP doesn't get allocated yet\n", endpoints.Name)
	}
	return nil
}

func (r *HAProvider) ensureEndpoints(ctx context.Context, serviceName, serviceNamespace string) (*corev1.Endpoints, error) {
	endpoints := &corev1.Endpoints{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      serviceName,
		Namespace: serviceNamespace,
	}, endpoints); err != nil {
		if apierrors.IsNotFound(err) {
			endpoints = &corev1.Endpoints{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Endpoints",
					APIVersion: "core/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: serviceNamespace,
				},
			}
			if err = r.Create(ctx, endpoints); err != nil {
				r.log.Error(err, "Failed to create Endpoints object")
				return nil, err
			}
		} else {
			r.log.Error(err, "Failed to get Endpoints object")
			return nil, err
		}
	}
	return endpoints, nil
}

func GetAviInfraSettingName(adc *akoov1alpha1.AKODeploymentConfig) string {
	return adc.Name + "-ais"
}

func queryFQDNEndpoint(fqdn string) (string, error) {
	ips, err := net.LookupIP(fqdn)
	if err == nil {
		return ips[0].String(), err
	}
	return "", err
}
