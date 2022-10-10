package flagset

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
			addons_str := strings.Join(addons, ",")
			fmt.Fprint(cmd.OutOrStdout(), addons_str)
			return nil
		},
	}
	DefineGlobalFlags(cmd)
	DefineGithubCmdFlags(cmd)
	DefineAWSFlags(cmd)
	DefineInstallerGenericFlags(cmd)
	return cmd
}

// Test_Init_k3d_basic
// simulates: `kubefirst --admin-email user@domain.com --cloud k3d
func Test_Init_k3d_basic(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "k3d"})
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
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) == success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
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
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) == success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_aws_basic_with_profile
// simulates: `kubefirst --admin-email user@domain.com --cloud aws --cloud aws --hosted-zone-name my.domain.com --profile default
func Test_Init_aws_basic_with_profile(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_aws_basic_with_arn
// simulates: `kubefirst --admin-email user@domain.com --cloud aws --cloud aws --hosted-zone-name my.domain.com --aws-assume-role role
func Test_Init_aws_basic_with_arn(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--aws-assume-role", "role"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
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
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) == success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
	}
}

// Test_Init_by_var_k3d
func Test_Init_by_var_k3d(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	os.Setenv("KUBEFIRST_ADMIN_EMAIL", "user@domain.com")
	os.Setenv("KUBEFIRST_CLOUD", "k3d")
	cmd.SetOut(b)
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
	}
	os.Unsetenv("KUBEFIRST_ADMIN_EMAIL")
	os.Unsetenv("KUBEFIRST_CLOUD")
}

// Test_Init_by_var_aws_profile
func Test_Init_by_var_aws_profile(t *testing.T) {
	cmd := FakeInitCmd()
	b := bytes.NewBufferString("")
	os.Setenv("KUBEFIRST_ADMIN_EMAIL", "user@domain.com")
	os.Setenv("KUBEFIRST_CLOUD", "aws")
	os.Setenv("KUBEFIRST_PROFILE", "default")
	os.Setenv("KUBEFIRST_HOSTED_ZONE_NAME", "mydomain.com")
	cmd.SetOut(b)
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
	}
	os.Unsetenv("KUBEFIRST_ADMIN_EMAIL")
	os.Unsetenv("KUBEFIRST_CLOUD")
	os.Unsetenv("KUBEFIRST_PROFILE")
	os.Unsetenv("KUBEFIRST_HOSTED_ZONE_NAME")
}

func Test_Init_Addons(t *testing.T) {
	// viper.SetConfigName("configTest") // name of config file (without extension)
	// viper.SetConfigType("yaml")       // REQUIRED if the config file does not have the extension in the name

	// viper.AddConfigPath(".")    // call multiple times to add many search paths
	// err := viper.ReadInConfig() // Find and read the config file
	// if err != nil {             // Handle errors reading the config file
	// 	panic(fmt.Errorf("fatal error config file: %w", err))
	// }
	viper.New()
	config := configs.ReadConfig()
	viperConfigFile := config.KubefirstConfigFilePath
	os.Remove(viperConfigFile)

	pkg.SetupViper(config)

	cmd := FakeInitAddonsTestCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--admin-email", "user@domain.com", "--cloud", "aws", "--hosted-zone-name", "my.domain.com", "--profile", "default"})
	err := cmd.Execute()
	if err != nil {
		t.Error(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Error(err)
	}
	if string(out) != success {
		t.Errorf("expected  to fail validation, but got \"%s\"", string(out))
	}
}
