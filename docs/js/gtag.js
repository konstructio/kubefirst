app.location$.subscribe(function (url) {
	window.dataLayer = window.dataLayer || [];
	function gtag() {
	  dataLayer.push(arguments);
	}
	gtag("js", new Date());

	gtag("config", "GTM-KZLF3TJ");
});