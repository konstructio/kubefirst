package flagset

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Test flags scenarions on init
// Meant to validate init scenarios
// based on: https://gianarb.it/blog/golang-mockmania-cli-command-with-cobra

// success
// Expected result form test that works as expected.
const success = "\nDONE"

func FakeInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fake-init",
		Short: "Let's test init",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := configs.ReadConfig()
			config.KubefirstConfigFilePath = "./logs/.k1_test"
			_ = os.Remove(config.KubefirstConfigFilePath)
			pkg.SetupViper(config)
			log.Println(viper.AllSettings())
			_, _, _, _, err := InitFlags(cmd)
			log.Println(viper.AllSettings())
			if err != nil {
				fmt.Fprint(cmd.OutOrStdout(), err.Error())
			}
			fmt.Fprint(cmd.OutOrStdout(), success)
			return nil
		},
	}
	DefineGlobalFlags(cmd)
	DefineGithubCmdFlags(cmd)
	DefineAWSFlags(cmd)
	DefineInstallerGenericFlags(cmd)
	return cmd
}

func FakeInitAddonsTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fake-init-addons",
		Short: "Let's test init with addons",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := ProcessGlobalFlags(cmd)
			if err != nil {
				fmt.Fprint(cmd.OutOrStdout(), err.Error())
			}

			_, err = ProcessGithubAddCmdFlags(cmd)
			if err != nil {
				fmt.Fprint(cmd.OutOrStdout(), err.Error())
			}

			_, err = ProcessInstallerGenericFlags(cmd)
			if err != nil {
				fmt.Fprint(cmd.OutOrStdout(), err.Error())
			}

			_, err = ProcessAwsFlags(cmd)
			if err != nil {
				fmt.Fprint(cmd.OutOrStdout(), err.Error())
			}
			addons := viper.GetStringSlice("addons")
			//convert to string..
			addonsStr := strings.Join(addons, ",")
			fmt.Fprint(cmd.OutOrStdout(), addonsStr)
			return nil
		},
	}
	DefineGlobalFlags(cmd)
	DefineGithubCmdFlags(cmd)
	DefineAWSFlags(cmd)
	DefineInstallerGenericFlags(cmd)
	return cmd
}

// Test_Init_k3d_basic - not supported on gitlab
// simulates: `kubefirst --admin-email user@domain.com --cloud k3d
//func Test_Init_k3d_gitlab(t *testing.T) {
//	cmd := FakeInitCmd()
//	b := bytes.NewBufferString("")
//	cmd.SetOut(b)
//	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "k3d"})
//	err := cmd.Execute()
//	if err != nil {
//		t.Error(err)
//	}
//	out, err := io.ReadAll(b)
//	if err != nil {
//		t.Error(err)
//	}
//	if string(out) == success {
//		t.Errorf("expected \"%s\" got \"%s\"", "set-by-flag", string(out))
//	}
//}

// Test_Init_k3d_basic
// simulates: `kubefirst --admin-email user@domain.com --cloud k3d --github-user ghuser --github-org ghorg
func Test_Init_k3d_basic_github(t *testing.T) {
	os.Setenv("GITHUB_AUTH_TOKEN", "ghp_fooBARfoo")
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "k3d", "--github-user", "ghuser", "--github-org", "ghorg"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected \"%s\" got \"%s\"", "set-by-flag", string(out))
	}
}

// Test_Init_aws_basic_missing_hostzone
// simulates: `kubefirst --admin-email user@domain.com --cloud aws

func Test_Init_aws_basic_missing_hostzone(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "aws"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) == success {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_aws_basic_missing_profile
// simulates: `kubefirst --admin-email user@domain.com --cloud aws --cloud aws --hosted-zone-name my.domain.com

func Test_Init_aws_basic_missing_profile(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) == success {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_aws_basic_with_profile
// simulates: `kubefirst --admin-email user@domain.com --cloud aws --cloud aws --hosted-zone-name my.domain.com --profile default

func Test_Init_aws_basic_with_profile(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--region", "eu-central-1", "--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_aws_basic_with_arn
// simulates: `kubefirst --admin-email user@domain.com --cloud aws --cloud aws --hosted-zone-name my.domain.com --aws-assume-role role

func Test_Init_aws_basic_with_arn(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--region", "eu-central-1", "--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--aws-assume-role", "role"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_aws_basic_with_profile_and_arn

func Test_Init_aws_basic_with_profile_and_arn(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--aws-assume-role", "role", "--profile", "default"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) == success {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_by_var_k3d
// this scenario to test to fail gitlab with k3d as it is not supported

//func Test_Init_by_var_k3d(t *testing.T) {
//	cmd := FakeInitCmd()
//	b := bytes.NewBufferString("")
//	os.Setenv("KUBEFIRST_ADMIN_EMAIL", "user@domain.com")
//	os.Setenv("KUBEFIRST_CLOUD", "k3d")
//	cmd.SetOut(b)
//	err := cmd.Execute()
//	if err != nil {
//		t.Error(err)
//	}
//	out, err := io.ReadAll(b)
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Println("---debug---")
//	fmt.Println(string(out))
//	fmt.Println("---debug---")
//
//	if string(out) == success {
//		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
//	}
//	os.Unsetenv("KUBEFIRST_ADMIN_EMAIL")
//	os.Unsetenv("KUBEFIRST_CLOUD")
//}

// Test_Init_by_var_aws_profile
func Test_Init_by_var_aws_profile(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	os.Setenv("KUBEFIRST_ADMIN_EMAIL", "user@domain.com")
	defer os.Unsetenv("KUBEFIRST_ADMIN_EMAIL")
	os.Setenv("KUBEFIRST_CLOUD", "aws")
	defer os.Unsetenv("KUBEFIRST_CLOUD")
	os.Setenv("KUBEFIRST_PROFILE", "default")
	defer os.Unsetenv("KUBEFIRST_PROFILE")
	os.Setenv("KUBEFIRST_HOSTED_ZONE_NAME", "mydomain.com")
	defer os.Unsetenv("KUBEFIRST_HOSTED_ZONE_NAME")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--region", "eu-central-1", "--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}

}

// Test_Init_aws_basic_with_profile
// simulates: `kubefirst --admin-email user@domain.com --cloud aws --cloud aws --hosted-zone-name my.domain.com --profile default
//
//	func Test_Init_aws_basic_with_profile_config(t *testing.T) {
//		cmd := FakeInitCmd()
//		b := bytes.NewBufferString("")
//		artifactsDir := os.Getenv("ARTIFACTS_SOURCE")
//		cmd.SetOut(b)
//		cmd.SetArgs([]string{"--config", artifactsDir + "/test/artifacts/init/aws_profile.yaml"})
//		err := cmd.Execute()
//		if err != nil {
//			t.Error(err)
//		}
//		out, err := ioutil.ReadAll(b)
//		if err != nil {
//			t.Error(err)
//		}
//		if string(out) != success {
//			t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
//		}
//	}
func Test_Init_Addons_Gitlab(t *testing.T) {
	viper.Set("addons", "")
	cmd := FakeInitAddonsTestCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--region", "eu-central-1", "--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != "gitlab,cloud" {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

func Test_Init_Addons_Github(t *testing.T) {
	os.Setenv("GITHUB_AUTH_TOKEN", "ghp_fooBARfoo")
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")
	viper.Set("addons", "")
	cmd := FakeInitAddonsTestCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--region", "eu-central-1", "--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default", "--github-user", "fake", "--github-org", "demo"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != "github,cloud" {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}

func Test_Init_Addons_Github_Kusk(t *testing.T) {
	os.Setenv("GITHUB_AUTH_TOKEN", "ghp_fooBARfoo")
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")
	viper.Set("addons", "")
	cmd := FakeInitAddonsTestCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--region", "eu-central-1", "--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default", "--github-user", "fake", "--github-org", "demo", "--addons", "kusk"})
	err := cmd.Execute()
	if err != nil {
		fmt.Println("---debug---")
		fmt.Println("here")
		fmt.Println("---debug---")
		t.Error(err)
	}
	out, err := io.ReadAll(b)
	if err != nil {
		fmt.Println("---debug---")
		fmt.Println("here2")
		fmt.Println("---debug---")
		t.Error(err)
	}

	if string(out) != "github,kusk,cloud" {
		t.Errorf("expected to fail validation, but got \"%s\"", string(out))
	}
}
