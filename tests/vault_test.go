/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// TestVaultLoginEndToEnd tests the end to end flow of logging into the cloud and local vault and retrieving a secret
// from it. This test is not run by default because it requires a cloud vault to be running.
// This test does:
//   - login to the cloud vault
//   - make sure the login was successful
//   - retrieve kbot secret
//   - logout of the cloud vault
//   - login to the cloud vault again using kbot credentials and userpass flow
//   - make sure the kbot is logged in
//
// prerequisites: this test requires E2E_VAULT_USERNAME to be set to "aone" in case we want to test a new created user
// and "kbot" in case we want to test the login for the initial account
func TestVaultLoginEndToEnd(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping end to tend test")
	}

	username := os.Getenv("E2E_VAULT_USERNAME")
	if username == "" {
		t.Error("E2E_VAULT_USERNAME is not set")
		return
	}

	config := configs.ReadConfig()

	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err)
	}

	initialVaultToken := viper.GetString("vault.token")
	if initialVaultToken == "" {
		t.Error("Vault token is empty")
	}

	// Headless is active by default
	opts := append(chromedp.DefaultExecAllocatorOptions[3:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.IgnoreCertErrors,
		chromedp.Headless,
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// create chrome instance
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// find Vault url
	var vaultURL string
	switch viper.GetString("cloud") {
	case pkg.CloudK3d:
		vaultURL = viper.GetString("vault.local.service")
	default:
		// cloud default
		vaultURL = "https://vault." + viper.GetString("aws.hostedzonename")
	}

	if err = chromedp.Run(
		ctx,
		chromedp.Navigate(vaultURL),
		chromedp.WaitVisible("//h1[text()='Sign in to Vault']"),
	); err != nil {
		t.Error(err.Error())
	}

	if err = chromedp.Run(ctx, chromedp.SendKeys(`//form/div/div/input`, initialVaultToken)); err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	if err = chromedp.Run(ctx, chromedp.Click(`//button[@id='auth-submit']`)); err != nil {
		t.Error(err)
	}

	// confirm its logged in
	if err = chromedp.Run(ctx,
		chromedp.WaitVisible(`//div[@class='level-left']/h1[contains(text(),'Secrets Engines')]`),
	); err != nil {
		t.Error(err)
	}

	if err = chromedp.Run(ctx,
		chromedp.Click(`(//div[@class='linkable-item-content'])[3]//a`),
	); err != nil {
		t.Error(err)
	}

	if err = chromedp.Run(ctx, chromedp.Click(`(//a)[10]`)); err != nil {
		t.Error(err)
	}

	// show secret
	if err = chromedp.Run(ctx, chromedp.Click(`//button[@class=' masked-input-toggle button']`)); err != nil {
		t.Error(err)
	}

	var initialPassword string
	if err = chromedp.Run(ctx,
		chromedp.Text(`//pre[@class='masked-value display-only is-word-break']`, &initialPassword),
	); err != nil {
		t.Error(err)
	}

	if initialPassword == "" {
		t.Error("initial user password is empty")
	}

	vaultLogoutURL := vaultURL + "/ui/vault/logout"
	if err = chromedp.Run(ctx,
		chromedp.Navigate(vaultLogoutURL),
	); err != nil {
		t.Error(err)
	}

	// select
	if err = chromedp.Run(ctx, chromedp.SetValue(`//select[@class="select"]`, "userpass", chromedp.BySearch)); err != nil {
		t.Error(err)
	}
	// force wait above select update
	time.Sleep(1 * time.Second)

	if err = chromedp.Run(ctx, chromedp.SendKeys(`//input[@id="username"]`, username)); err != nil {
		t.Error(err)
	}
	if err = chromedp.Run(ctx, chromedp.SendKeys(`//input[@id="password"]`, initialPassword)); err != nil {
		t.Error(err)
	}

	// click sign in
	if err = chromedp.Run(ctx, chromedp.Click(`//button[@id='auth-submit']`)); err != nil {
		t.Error(err)
	}

	if err = chromedp.Run(ctx,
		chromedp.WaitVisible(`//div[@class='level-left']/h1[contains(text(),'Secrets Engines')]`),
	); err != nil {
		t.Error(err)
	}

}
