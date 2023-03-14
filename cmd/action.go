package cmd

import (
	"context"
	"fmt"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argo "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
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
		//config := aws.GetConfig("kubefirst-tech")

		kubeconfig, err := k8s.GetClientConfig(false, "/Users/scott/.kube/config")
		if err != nil {
			fmt.Println(err)
			return err
		}

		argoClient, err := argo.NewForConfig(kubeconfig)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fmt.Println("installing argocd app")
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
					//RepoURL:        "gitops_repo_git_url",
					RepoURL: "https://github.com/jarededwards/vault-spike.git",
					Path:    "registry/k1-vault-spike",
					// Path:           "registry/clustername",
					TargetRevision: "HEAD",
				},
				Destination: v1alpha1.ApplicationDestination{
					Server:    "https://kubernetes.default.svc",
					Namespace: "argocd",
				},
				Project: "default",
				SyncPolicy: &v1alpha1.SyncPolicy{
					Automated: &v1alpha1.SyncPolicyAutomated{
						Prune:    true,
						SelfHeal: true,
					},
					SyncOptions: []string{"CreateNamespace=true"},
					Retry: &v1alpha1.RetryStrategy{
						Limit: 5,
						Backoff: &v1alpha1.Backoff{
							Duration:    "5s",
							Factor:      new(int64),
							MaxDuration: "5m0s",
						},
					},
				},
			},
		}
		_, err = argoClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), app, v1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("app created")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
