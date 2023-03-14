package cmd

import (
	"context"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argo "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := aws.GetConfig("kubefirst-tech")

		kubeconfig, err := k8s.GetClientConfig(false, config.Kubeconfig)
		if err != nil {
			return err
		}

		argoClient, err := argo.NewForConfig(kubeconfig)
		if err != nil {
			return err
		}

		app := &v1alpha1.Application{
			TypeMeta: v1.TypeMeta{
				Kind:       "Application",
				APIVersion: "argoproj.io/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        "registry",
				Namespace:   "argocd",
				Annotations: map[string]string{"argocd.argoproj.io/sync-wave": "1"},
			},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL:        "",
					Path:           "",
					TargetRevision: "",
					Helm:           &v1alpha1.ApplicationSourceHelm{},
					Kustomize:      &v1alpha1.ApplicationSourceKustomize{},
					Directory:      &v1alpha1.ApplicationSourceDirectory{},
					Plugin:         &v1alpha1.ApplicationSourcePlugin{},
					Chart:          "",
					Ref:            "",
				},
				Destination: v1alpha1.ApplicationDestination{
					Server:    "",
					Namespace: "",
					Name:      "",
				},
				Project: "default",
				SyncPolicy: &v1alpha1.SyncPolicy{
					Automated:                &v1alpha1.SyncPolicyAutomated{},
					SyncOptions:              []string{},
					Retry:                    &v1alpha1.RetryStrategy{},
					ManagedNamespaceMetadata: &v1alpha1.ManagedNamespaceMetadata{},
				},
				IgnoreDifferences:    []v1alpha1.ResourceIgnoreDifferences{},
				Info:                 []v1alpha1.Info{},
				RevisionHistoryLimit: new(int64),
				Sources:              []v1alpha1.ApplicationSource{},
			},
		}
		_, err = argoClient.ArgoprojV1alpha1().Applications("default").Create(context.Background(), app, v1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
