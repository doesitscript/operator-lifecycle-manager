package install

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	clientfakes "github.com/operator-framework/operator-lifecycle-manager/pkg/api/wrappers/wrappersfakes"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
)

var (
	Controller         = false
	BlockOwnerDeletion = false
)

func testDeployment(name, namespace string, mockOwner ownerutil.Owner) appsv1.Deployment {
	testDeploymentLabels := map[string]string{"olm.owner": mockOwner.GetName(), "olm.owner.namespace": mockOwner.GetNamespace()}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         v1alpha1.SchemeGroupVersion.String(),
					Kind:               v1alpha1.ClusterServiceVersionKind,
					Name:               mockOwner.GetName(),
					UID:                mockOwner.GetUID(),
					Controller:         &Controller,
					BlockOwnerDeletion: &BlockOwnerDeletion,
				},
			},
			Labels: testDeploymentLabels,
		},
	}
	return deployment
}

func testServiceAccount(name string, mockOwner ownerutil.Owner) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{}
	serviceAccount.SetName(name)
	serviceAccount.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.SchemeGroupVersion.String(),
			Kind:               v1alpha1.ClusterServiceVersionKind,
			Name:               mockOwner.GetName(),
			UID:                mockOwner.GetUID(),
			Controller:         &Controller,
			BlockOwnerDeletion: &BlockOwnerDeletion,
		},
	})
	return serviceAccount
}

func strategy(n int, namespace string, mockOwner ownerutil.Owner) *StrategyDetailsDeployment {
	var deploymentSpecs = []StrategyDeploymentSpec{}
	var permissions = []StrategyDeploymentPermissions{}
	for i := 1; i <= n; i++ {
		dep := testDeployment(fmt.Sprintf("olm-dep-%d", i), namespace, mockOwner)
		spec := StrategyDeploymentSpec{Name: dep.GetName(), Spec: dep.Spec}
		deploymentSpecs = append(deploymentSpecs, spec)
		serviceAccount := testServiceAccount(fmt.Sprintf("olm-sa-%d", i), mockOwner)
		permissions = append(permissions, StrategyDeploymentPermissions{
			ServiceAccountName: serviceAccount.Name,
			Rules: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"list", "delete"},
					APIGroups: []string{""},
					Resources: []string{"pods"},
				},
			},
		})
	}
	return &StrategyDetailsDeployment{
		DeploymentSpecs: deploymentSpecs,
		Permissions:     permissions,
	}
}

func TestInstallStrategyDeploymentInstallDeployments(t *testing.T) {
	var (
		mockOwner = v1alpha1.ClusterServiceVersion{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1alpha1.ClusterServiceVersionKind,
				APIVersion: v1alpha1.ClusterServiceVersionAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "clusterserviceversion-owner",
				Namespace: "olm-test-deployment",
			},
		}
		mockOwnerRefs = []metav1.OwnerReference{{
			APIVersion:         v1alpha1.ClusterServiceVersionAPIVersion,
			Kind:               v1alpha1.ClusterServiceVersionKind,
			Name:               mockOwner.GetName(),
			UID:                mockOwner.UID,
			Controller:         &Controller,
			BlockOwnerDeletion: &BlockOwnerDeletion,
		}}
	)

	type inputs struct {
		strategyDeploymentSpecs []StrategyDeploymentSpec
	}
	type setup struct {
		existingDeployments []*appsv1.Deployment
	}
	type createOrUpdateMock struct {
		expectedDeployment appsv1.Deployment
		returnError        error
	}
	tests := []struct {
		description         string
		inputs              inputs
		setup               setup
		createOrUpdateMocks []createOrUpdateMock
		output              error
	}{
		{
			description: "updates/creates correctly",
			inputs: inputs{
				strategyDeploymentSpecs: []StrategyDeploymentSpec{
					{
						Name: "test-deployment-1",
						Spec: appsv1.DeploymentSpec{},
					},
					{
						Name: "test-deployment-2",
						Spec: appsv1.DeploymentSpec{},
					},
					{
						Name: "test-deployment-3",
						Spec: appsv1.DeploymentSpec{},
					},
				},
			},
			setup: setup{
				existingDeployments: []*appsv1.Deployment{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-deployment-1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-deployment-3",
						},
						Spec: appsv1.DeploymentSpec{
							Paused: false, // arbitrary spec difference
						},
					},
				},
			},
			createOrUpdateMocks: []createOrUpdateMock{
				{
					expectedDeployment: appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-deployment-1",
							Namespace:       mockOwner.GetNamespace(),
							OwnerReferences: mockOwnerRefs,
							Labels: map[string]string{
								"olm.owner":           mockOwner.GetName(),
								"olm.owner.namespace": mockOwner.GetNamespace(),
							},
						},
					},
					returnError: nil,
				},
				{
					expectedDeployment: appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-deployment-2",
							Namespace:       mockOwner.GetNamespace(),
							OwnerReferences: mockOwnerRefs,
							Labels: map[string]string{
								"olm.owner":           mockOwner.GetName(),
								"olm.owner.namespace": mockOwner.GetNamespace(),
							},
						},
					},
					returnError: nil,
				},
				{
					expectedDeployment: appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-deployment-3",
							Namespace:       mockOwner.GetNamespace(),
							OwnerReferences: mockOwnerRefs,
							Labels: map[string]string{
								"olm.owner":           mockOwner.GetName(),
								"olm.owner.namespace": mockOwner.GetNamespace(),
							},
						},
					},
					returnError: nil,
				},
			},
			output: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			fakeClient := new(clientfakes.FakeInstallStrategyDeploymentInterface)

			for i, m := range tt.createOrUpdateMocks {
				fakeClient.CreateDeploymentReturns(nil, m.returnError)
				defer func(i int, expectedDeployment appsv1.Deployment) {
					dep := fakeClient.CreateOrUpdateDeploymentArgsForCall(i)
					require.Equal(t, expectedDeployment, *dep)
				}(i, m.expectedDeployment)
			}

			installer := &StrategyDeploymentInstaller{
				strategyClient: fakeClient,
				owner:          &mockOwner,
			}
			result := installer.installDeployments(tt.inputs.strategyDeploymentSpecs)
			assert.Equal(t, tt.output, result)
		})
	}
}

type BadStrategy struct{}

func (b *BadStrategy) GetStrategyName() string {
	return "bad"
}

func TestNewStrategyDeploymentInstaller(t *testing.T) {
	mockOwner := v1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.ClusterServiceVersionKind,
			APIVersion: v1alpha1.ClusterServiceVersionAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clusterserviceversion-owner",
			Namespace: "ns",
		},
	}
	fakeClient := new(clientfakes.FakeInstallStrategyDeploymentInterface)
	strategy := NewStrategyDeploymentInstaller(fakeClient, map[string]string{"test": "annotation"}, &mockOwner, nil)
	require.Implements(t, (*StrategyInstaller)(nil), strategy)
	require.Error(t, strategy.Install(&BadStrategy{}))
	installed, err := strategy.CheckInstalled(&BadStrategy{})
	require.False(t, installed)
	require.Error(t, err)
}

func TestInstallStrategyDeploymentCheckInstallErrors(t *testing.T) {
	namespace := "olm-test-deployment"

	mockOwner := v1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.ClusterServiceVersionKind,
			APIVersion: v1alpha1.ClusterServiceVersionAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clusterserviceversion-owner",
			Namespace: namespace,
		},
	}

	tests := []struct {
		createDeploymentErr error
		description         string
	}{
		{
			createDeploymentErr: fmt.Errorf("error creating deployment"),
			description:         "ErrorCreatingDeployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			fakeClient := new(clientfakes.FakeInstallStrategyDeploymentInterface)
			strategy := strategy(1, namespace, &mockOwner)
			installer := NewStrategyDeploymentInstaller(fakeClient, map[string]string{"test": "annotation"}, &mockOwner, nil)

			dep := testDeployment("olm-dep-1", namespace, &mockOwner)
			dep.Spec.Template.SetAnnotations(map[string]string{"test": "annotation"})
			fakeClient.FindAnyDeploymentsMatchingNamesReturns(
				[]*appsv1.Deployment{
					&dep,
				}, nil,
			)
			defer func() {
				require.Equal(t, []string{dep.Name}, fakeClient.FindAnyDeploymentsMatchingNamesArgsForCall(0))
			}()

			installed, err := installer.CheckInstalled(strategy)
			require.NoError(t, err)
			require.True(t, installed)

			deployment := testDeployment("olm-dep-1", namespace, &mockOwner)
			deployment.Spec.Template.SetAnnotations(map[string]string{"test": "annotation"})
			fakeClient.CreateOrUpdateDeploymentReturns(&deployment, tt.createDeploymentErr)
			defer func() {
				require.Equal(t, &deployment, fakeClient.CreateOrUpdateDeploymentArgsForCall(0))
			}()

			if tt.createDeploymentErr != nil {
				err := installer.Install(strategy)
				require.Error(t, err)
				return
			}
		})
	}
}
