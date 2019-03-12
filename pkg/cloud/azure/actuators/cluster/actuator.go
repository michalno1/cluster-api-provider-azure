/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"github.com/pkg/errors"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api-provider-azure/pkg/cloud/azure/actuators"
	"sigs.k8s.io/cluster-api-provider-azure/pkg/cloud/azure/services/certificates"
	"sigs.k8s.io/cluster-api-provider-azure/pkg/cloud/azure/services/network"
	"sigs.k8s.io/cluster-api-provider-azure/pkg/cloud/azure/services/resources"
	"sigs.k8s.io/cluster-api-provider-azure/pkg/deployer"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	controllerError "sigs.k8s.io/cluster-api/pkg/controller/error"
)

//+kubebuilder:rbac:groups=azureprovider.k8s.io,resources=azureclusterproviderconfigs;azureclusterproviderstatuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.k8s.io,resources=clusters;clusters/status,verbs=get;list;watch;create;update;patch;delete

// Actuator is responsible for performing cluster reconciliation
type Actuator struct {
	*deployer.Deployer

	client client.ClusterV1alpha1Interface
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	Client client.ClusterV1alpha1Interface
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) *Actuator {
	return &Actuator{
		Deployer: deployer.New(deployer.Params{ScopeGetter: actuators.DefaultScopeGetter}),
		client:   params.Client,
	}
}

// Reconcile reconciles a cluster and is invoked by the Cluster Controller
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) error {
	klog.Infof("Reconciling cluster %v", cluster.Name)

	scope, err := actuators.NewScope(actuators.ScopeParams{Cluster: cluster, Client: a.client})
	if err != nil {
		return errors.Errorf("failed to create scope: %+v", err)
	}

	defer scope.Close()

	networkSvc := network.NewService(scope)
	resourcesSvc := resources.NewService(scope)
	certSvc := certificates.NewService(scope)

	// Store cert material in spec.
	if err := certSvc.ReconcileCertificates(); err != nil {
		return errors.Wrapf(err, "failed to reconcile certificates for cluster %q", cluster.Name)
	}

	if err := resourcesSvc.ReconcileResourceGroup(); err != nil {
		return errors.Wrapf(err, "failed to reconcile resource group for cluster %q", cluster.Name)
	}

	if err := networkSvc.ReconcileNetwork(); err != nil {
		return errors.Wrapf(err, "failed to reconcile network for cluster %q", cluster.Name)
	}

	// TODO: Add bastion method
	/*
		if err := resourcesSvc.ReconcileBastion(); err != nil {
			return errors.Wrapf(err, "failed to reconcile bastion host for cluster %q", cluster.Name)
		}
	*/

	if err := networkSvc.ReconcileLoadBalancer("api"); err != nil {
		return errors.Wrapf(err, "failed to reconcile load balancers for cluster %q", cluster.Name)
	}

	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller.
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	klog.Infof("Reconciling cluster %v", cluster.Name)

	scope, err := actuators.NewScope(actuators.ScopeParams{Cluster: cluster, Client: a.client})
	if err != nil {
		return errors.Errorf("failed to create scope: %+v", err)
	}

	defer scope.Close()

	//networkSvc := network.NewService(scope)
	resourcesSvc := resources.NewService(scope)

	// TODO: Add load balancer method
	/*
		if err := networkSvc.DeleteLoadBalancers(); err != nil {
			return errors.Errorf("unable to delete load balancers: %+v", err)
		}
	*/

	// TODO: Add bastion method
	/*
		if err := resourcesSvc.DeleteBastion(); err != nil {
			return errors.Errorf("unable to delete bastion: %+v", err)
		}
	*/

	// TODO: Add network method
	/*
		if err := resourcesSvc.DeleteNetwork(); err != nil {
			klog.Errorf("Error deleting cluster %v: %v.", cluster.Name, err)
			return &controllerError.RequeueAfterError{
				RequeueAfter: 5 * 1000 * 1000 * 1000,
			}
		}
	*/

	if err := resourcesSvc.DeleteResourceGroup(); err != nil {
		klog.Errorf("Error deleting resource group: %v.", err)
		return &controllerError.RequeueAfterError{
			RequeueAfter: 5 * 1000 * 1000 * 1000,
		}
	}

	return nil
}