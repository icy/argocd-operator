package workloads

import (
	"context"
	"fmt"

	"github.com/argoproj-labs/argocd-operator/common"
	"github.com/argoproj-labs/argocd-operator/pkg/argoutil"
	"github.com/argoproj-labs/argocd-operator/pkg/mutation"
	oappsv1 "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentConfigRequest objects contain all the required information to produce a deploymentConfig object in return
type DeploymentConfigRequest struct {
	Name              string
	InstanceName      string
	InstanceNamespace string
	Component         string
	Labels            map[string]string
	Annotations       map[string]string

	// array of functions to mutate role before returning to requester
	Mutations []mutation.MutateFunc
	Client    interface{}
}

// newDeploymentConfig returns a new DeploymentConfig instance for the given ArgoCD.
func newDeploymentConfig(name, instanceName, instanceNamespace, component string, labels, annotations map[string]string) *oappsv1.DeploymentConfig {
	var deploymentConfigName string
	if name != "" {
		deploymentConfigName = name
	} else {
		deploymentConfigName = argoutil.GenerateResourceName(instanceName, component)

	}
	return &oappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentConfigName,
			Namespace:   instanceNamespace,
			Labels:      argoutil.MergeMaps(common.DefaultLabels(deploymentConfigName, instanceName, component), labels),
			Annotations: argoutil.MergeMaps(common.DefaultAnnotations(instanceName, instanceNamespace), annotations),
		},
	}
}

func CreateDeploymentConfig(deploymentConfig *oappsv1.DeploymentConfig, client ctrlClient.Client) error {
	return client.Create(context.TODO(), deploymentConfig)
}

// UpdateDeploymentConfig updates the specified DeploymentConfig using the provided client.
func UpdateDeploymentConfig(deploymentConfig *oappsv1.DeploymentConfig, client ctrlClient.Client) error {
	_, err := GetDeploymentConfig(deploymentConfig.Name, deploymentConfig.Namespace, client)
	if err != nil {
		return err
	}

	if err = client.Update(context.TODO(), deploymentConfig); err != nil {
		return err
	}
	return nil
}

func DeleteDeploymentConfig(name, namespace string, client ctrlClient.Client) error {
	existingDeploymentConfig, err := GetDeploymentConfig(name, namespace, client)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return nil
	}

	if err := client.Delete(context.TODO(), existingDeploymentConfig); err != nil {
		return err
	}
	return nil
}

func GetDeploymentConfig(name, namespace string, client ctrlClient.Client) (*oappsv1.DeploymentConfig, error) {
	existingDeploymentConfig := &oappsv1.DeploymentConfig{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, existingDeploymentConfig)
	if err != nil {
		return nil, err
	}
	return existingDeploymentConfig, nil
}

func ListDeploymentConfigs(namespace string, client ctrlClient.Client, listOptions []ctrlClient.ListOption) (*oappsv1.DeploymentConfigList, error) {
	existingDeploymentConfigs := &oappsv1.DeploymentConfigList{}
	err := client.List(context.TODO(), existingDeploymentConfigs, listOptions...)
	if err != nil {
		return nil, err
	}
	return existingDeploymentConfigs, nil
}

func RequestDeploymentConfig(request DeploymentConfigRequest) (*oappsv1.DeploymentConfig, error) {
	var (
		mutationErr error
	)
	deploymentConfig := newDeploymentConfig(request.Name, request.InstanceName, request.InstanceNamespace, request.Component, request.Labels, request.Annotations)

	if len(request.Mutations) > 0 {
		for _, mutation := range request.Mutations {
			err := mutation(nil, deploymentConfig, request.Client)
			if err != nil {
				mutationErr = err
			}
		}
		if mutationErr != nil {
			return deploymentConfig, fmt.Errorf("RequestDeploymentConfig: one or more mutation functions could not be applied: %s", mutationErr)
		}
	}

	return deploymentConfig, nil
}