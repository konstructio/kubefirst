package tests

import (
	"context"
	"github.com/chromedp/chromedp"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"testing"
	"time"
)

// TestCloudVaultLoginEndToEnd tests the end to end flow of logging into the cloud vault and retrieving a secret from it.
// This test is not run by default because it requires a cloud vault to be running.
// This test does:
//   - login to the cloud vault
//   - make sure the login was successful
//   - retrieve kbot secret
//   - logout of the cloud vault
//   - login to the cloud vault again using kbot credentials and userpass flow
//   - make sure the kbot is logged in
//
// todo: remove sleeps
func TestCloudVaultLoginEndToEnd(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping end to tend test")
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

	opts := append(chromedp.DefaultExecAllocatorOptions[3:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.IgnoreCertErrors,
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// create chrome instance
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	vaultURL := "https://vault." + viper.GetString("aws.hostedzonename")
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

	var initialKBotPassword string
	if err = chromedp.Run(ctx,
		chromedp.Text(`//pre[@class='masked-value display-only is-word-break']`, &initialKBotPassword),
	); err != nil {
		t.Error(err)
	}

	if initialKBotPassword == "" {
		t.Error("initial kbot password is empty")
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

	if err = chromedp.Run(ctx, chromedp.SendKeys(`//input[@id="username"]`, "kbot")); err != nil {
		t.Error(err)
	}
	if err = chromedp.Run(ctx, chromedp.SendKeys(`//input[@id="password"]`, initialKBotPassword)); err != nil {
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
