# Limitations running Kubefirst Locally 

## Overview

`Kubefirst local` bring to our local machine the experience of having a cloud environment running locally. This is the best effort to abstract all the convenience of cloud services running inside containers. 
Thank you for all the projects listed in the [Credit](../local/credit.md) section. 

Unfortunately, we have some limitations running kubefirst locally, and we listed below these limitations and possible ways to solve them.

## Hardware recommendation

- OS: macOS (Intel or Apple Silicon M1/M2) and Linux AMD64
- CPU: A Quad Core CPU or Apple Silicon with M1 or M2 chip
- RAM: 16GB RAM
- HDD: 10GB HD space (docker images)

## Features Limitations

**Gitlab:** to keep the local install "slim" we couldn't offer Gitlab as a Git provider option.

**Ngrok:** to allow Github Webhook to reach your machine without exposing them directly to the internet, we use Ngrok to create a tunnel and assign the Ngrok endpoint to Github Webhook.

We use a free tier of this service and have rate limits for data transfer and limited session duration of the tunnel. If the tunnel was closed, we didn't support the reconnect process. If you want to reconnect, you should use the Ngrok tool and update the webhook on GitHub to keep the Atlantis working.

## Known issues

- Disk: During the provisioning of the local environment, the kubefirst cli bootstrap a k3d cluster which starts downloading all docker images simultaneously. So you can experience some issues related to disk performance.

- Network bandwidth: As described above, the network bandwidth could be throttled due to downloading all docker images simultaneously.

## Problematic use cases

- Conventions: you are demoing the kubefirst using the kubefirst local installation at the convention. You could suffer issues with networking, mainly if the convention facility's network is poor.
- Mobile Connection: if you use the mobile connection routed to your laptop, the downloads may spend much of your data plan and suffer from the poor mobile connection.

## Tips

### Avoiding tools re-download

The kubefirstCLI download some tools used during cluster provisioning, for example, Terraform, Helm, and Kubectl, in versions compatible with Kubefirst and stores them in the K1 folder. 
If you are using Kufibefirst to demo in conferences or using poor connections (mobile, hotels) you should consider using this additional flag `--preserve-tools` for each cycle of create/destroy. 
This will preserve tools downloaded and will save time and network bandwidth during cluster provisioning.