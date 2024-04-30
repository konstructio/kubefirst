---
name: Validate if commits are signed
on: [pull_request, pull_request_target]

jobs:
  signed-commits-check:
    runs-on: ubuntu-latest
    steps:

      - name: Check out the repository code
        uses: actions/checkout@v4.1.4

      - name: Check if the commits are signed
        uses: 1Password/check-signed-commits-action@v1
