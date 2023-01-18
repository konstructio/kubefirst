# ngrok

`ngrok` ngrok is a tool that allows developers to expose a local web server to the internet. It creates a secure tunnel to the local web server, which can then be accessed remotely via a public URL. This is useful for testing webhooks, debugging web applications, and sharing local development environments with others. ngrok can also be used to inspect and debug web traffic and can capture and analyze HTTP requests and responses. Additionally, ngrok supports both HTTP and HTTPS and can also inspect the traffic in WebSocket, HTTP/2 and TCP protocols.

### Releases

Atlantis announces their releases page:
[https://ngrok.com/download](https://ngrok.com/download)

### Installation Instructions

Kubefirst uses ngrok Go library, so you don't need to install it. In case you want to try it out, you can install it with brew (mac):

```bash
brew install ngrok/ngrok/ngrok
```

### Checking Your ngrok version

```bash
ngrok version
```

### Opening a testing tunnel

```bash
ngrok http 8080
```
