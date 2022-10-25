package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/reports"

	"github.com/kubefirst/kubefirst/pkg"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var postInstallCmd = &cobra.Command{
	Use:   "post-install",
	Short: "starts post install process",
	Long:  "Starts post install process to open the Console UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		createFlags, err := flagset.ProcessCreateFlags(cmd)
		if err != nil {
			return err
		}

		cloud := viper.GetString("cloud")
		if createFlags.EnableConsole && cloud != pkg.CloudK3d {
			log.Println("Starting the presentation of console and api for the handoff screen")
			go func() {
				errInThread := api.RunE(cmd, args)
				if errInThread != nil {
					log.Println(errInThread)
				}
			}()
			go func() {
				errInThread := console.RunE(cmd, args)
				if errInThread != nil {
					log.Println(errInThread)
				}
			}()

			log.Println("Kubefirst Console available at: http://localhost:9094", globalFlags.SilentMode)

			openbrowser(pkg.LocalConsoleUI)

		} else {
			log.Println("Skipping the presentation of console and api for the handoff screen")
		}

		// open all port forwards, wait console ui be ready, and open console ui in the browser
		if cloud == pkg.CloudK3d {
			err := openPortForwardForKubeConConsole()
			if err != nil {
				log.Println(err)
			}

			err = isConsoleUIAvailable(pkg.LocalConsoleUI)
			if err != nil {
				log.Println(err)
			}
			openbrowser(pkg.LocalConsoleUI)
		}

		if viper.GetString("cloud") == flagset.CloudK3d {
			reports.LocalHandoffScreen(globalFlags.DryRun, globalFlags.SilentMode)
		} else {
			reports.HandoffScreen(globalFlags.DryRun, globalFlags.SilentMode)
		}

		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(postInstallCmd)

	currentCommand := postInstallCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineCreateFlags(currentCommand)
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		_, _, err = pkg.ExecShellReturnStrings("xdg-open", url)
	case "windows":
		_, _, err = pkg.ExecShellReturnStrings("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		_, _, err = pkg.ExecShellReturnStrings("open", url)
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Println(err)
	}
}

// todo: this is temporary
func isConsoleUIAvailable(url string) error {
	attempts := 10
	httpClient := http.DefaultClient
	for i := 0; i < attempts; i++ {

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			log.Println("console UI is up and running")
			return nil
		}

		log.Println("waiting UI console to be ready")
		time.Sleep(5 * time.Second)
	}

	return nil
}

// todo: this is temporary
func openPortForwardForKubeConConsole() error {

	var wg sync.WaitGroup
	wg.Add(7)
	// argocd
	go func() {
		_, err := k8s.PortForward(false, "argocd", "svc/argocd-server", "8080:80")
		if err != nil {
			log.Println("error opening ArgoCD port forward")
		}
		wg.Done()
	}()

	// atlantis
	go func() {
		_, err := k8s.PortForward(false, "atlantis", "svc/atlantis", "4141:80")
		if err != nil {
			log.Println("error opening Atlantis port forward")
		}
		wg.Done()
	}()

	// chartmuseum
	go func() {
		_, err := k8s.PortForward(false, "chartmuseum", "svc/chartmuseum", "8181:8080")
		if err != nil {
			log.Println("error opening Chartmuseum port forward")
		}
		wg.Done()
	}()

	// vault
	go func() {
		_, err := k8s.PortForward(false, "vault", "svc/vault", "8200:8200")
		if err != nil {
			log.Println("error opening Vault port forward")
		}
		wg.Done()
	}()

	// minio
	go func() {
		_, err := k8s.PortForward(false, "minio", "svc/minio", "9000:9000")
		if err != nil {
			log.Println("error opening Minio port forward")
		}
		wg.Done()
	}()

	// minio console
	go func() {
		_, err := k8s.PortForward(false, "minio", "svc/minio-console", "9001:9001")
		if err != nil {
			log.Println("error opening Minio-console port forward")
		}
		wg.Done()
	}()

	// Kubecon console ui
	go func() {
		_, err := k8s.PortForward(false, "kubefirst", "svc/kubefirst-console", "9094:80")
		if err != nil {
			log.Println("error opening Kubefirst-console port forward")
		}
		wg.Done()
	}()

	wg.Wait()

	return nil
}
