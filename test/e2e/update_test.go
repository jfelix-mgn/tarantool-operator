package e2e

import (
	goctx "context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestOperatorMustUpdateStatefulSetPerCartridgeRole(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	clenupOpts := &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       time.Second * 60,
		RetryInterval: time.Second * 1,
	}
	if err := ctx.InitializeClusterResources(clenupOpts); err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("failed to get namespace %s", err)
	}

	kubeClient := framework.Global.KubeClient
	err = e2eutil.WaitForOperatorDeployment(t, kubeClient, namespace, "tarantool-operator", 1, time.Second*1, time.Second*60)
	if err != nil {
		t.Fatalf("failed to deploy operator %s", err)
	}

	t.Log("Install resources version 0.0.1")
	if err = InitializeScenario(ctx, "first_install"); err != nil {
		t.Fatalf("failed to initialize scenario %s", err)
	}

	expectedRoles := 2
	err = wait.Poll(time.Second*1, time.Second*60, func() (done bool, err error) {
		sts, err := kubeClient.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if len(sts.Items) == expectedRoles {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Install resources version 0.0.2")
	ress, err := UpdateScenario(ctx, "update_install", t)
	if err != nil {
		t.Fatalf("failed to initialize scenario %s", err)
	}

	namespace, err = ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	err = wait.Poll(time.Second*1, time.Second*60, func() (done bool, err error) {

		versionCheckComplete := true

		for _, obj := range ress {
			namespaceName := types.NamespacedName{Namespace: namespace, Name: obj.GetName()}
			t.Log(fmt.Sprintf("Get resource %s for check version annotation", namespaceName))
			err = framework.Global.Client.Get(goctx.TODO(), namespaceName, &obj)
			if err != nil {
				t.Fatal(err)
			}
			versionAnnotation := obj.GetAnnotations()["example.local/version"]
			t.Log(fmt.Sprintf("Check resource %s for check version annotation. Expected - %s, Found - %s", namespaceName, "0.0.2", versionAnnotation))
			if versionAnnotation != "0.0.2" {
				versionCheckComplete = false
			}
		}
		t.Log(fmt.Sprintf("Check version check result (%v)", versionCheckComplete))
		if !versionCheckComplete {
			t.Log(fmt.Sprintf("Version check failed"))
			return false, nil
		}
		t.Log(fmt.Sprintf("Check count statefullsets in namespace %s", namespace))
		sts, err := kubeClient.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if len(sts.Items) != expectedRoles {
			return false, nil
		}
		versionCheckComplete = true
		for _, obj := range sts.Items {
			namespaceName := types.NamespacedName{Namespace: namespace, Name: obj.GetName()}
			versionAnnotation := obj.Spec.Template.GetAnnotations()["example.local/version"]
			t.Log(fmt.Sprintf("Check resource %s for check version annotation. Expected - %s, Found - %s", namespaceName, "0.0.2", versionAnnotation))
			if versionAnnotation != "0.0.2" {
				versionCheckComplete = false
			}
		}
		t.Log(fmt.Sprintf("Check version check result (%v)", versionCheckComplete))
		if versionCheckComplete {
			return true, nil
		}
		t.Log(fmt.Sprintf("Version check failed"))
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = wait.Poll(time.Second*1, time.Second*300, func() (done bool, err error) {
		return false, nil
	})
}
