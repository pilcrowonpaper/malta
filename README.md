# Malta

### Preview Malta Docs

- `curl -o malta.tgz -L https://github.com/James-Kua/Malta/releases/latest/download/darwin-amd64.tgz`
- `tar -xvzf malta.tgz`
- `install darwin-arm64/malta /usr/local/bin`
- `malta build && malta preview`

### Notes
- Files are served from the `/pages` directory

### Example `malta.config.json`

```json
{
  "name": "your-page-title",
  "description": "your-page-description",
  "domain": "your-domain",
  "sidebar": [
    {
      "title": "your-sidebar-title",
      "pages": [
        ["content-title", "/index"]
      ]
    },
    {
      "title": "Links",
      "pages": [
        ["GitHub", "https://github.com/James-Kua/your-repo-link"]
      ]
    }
  ]
}
```

### Sample Cloudflare Pages GitHub Actions

```yaml
name: "Publish"
on:
  push:
    branches:
      - main

env:
  CLOUDFLARE_API_TOKEN: ${{secrets.CLOUDFLARE_PAGES_API_TOKEN}}

jobs:
  publish:
    name: Publish
    runs-on: ubuntu-latest
    steps:
      - name: setup actions
        uses: actions/checkout@v3
      - name: setup node
        uses: actions/setup-node@v3
        with:
          node-version: 20.5.1
          registry-url: https://registry.npmjs.org
      - name: install malta
        run: |
          curl -o malta.tgz -L https://github.com/James-Kua/Malta/releases/latest/download/linux-amd64.tgz
          tar -xvzf malta.tgz
      - name: build
        run: ./linux-amd64/malta build
      - name: install wrangler
        run: npm i -g wrangler
      - name: deploy
        run: wrangler pages deploy dist --project-name [cloudflare-project-name] --branch main
```

## Developer Guide

### Build and test

- `cd src`
- `go build && install malta /usr/local/bin`

### Build Binaries

`./scripts/build.sh`