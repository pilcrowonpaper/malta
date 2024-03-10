---
title: "Deploying with GitHub Actions"
---

# Deploying with GitHub Actions

You can run Malta by installing the binaries from GitHub Releases.

```yaml
name: Publish docs
on:
  push:
    branches:
      - main

jobs:
  v3-docs:
    runs-on: ubuntu-latest
    steps:
      - name: setup actions
        uses: actions/checkout@v3
      - name: install malta
        run: |
          curl -o malta.tgz -L https://github.com/pilcrowonpaper/malta/releases/latest/download/linux-amd64.tgz
          tar -xvzf malta.tgz
      - name: build
        run: ./linux-amd64/malta build
```

## GitHub Pages

To deploy your docs to GitHub Pages, you need to upload the generated `dist` directory as an artifact.

```yaml
name: Publish docs

on:
  push:
    branches:
      - main

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: setup actions
        uses: actions/checkout@v3
      - name: install malta
        run: |
          curl -o malta.tgz -L https://github.com/pilcrowonpaper/malta/releases/latest/download/linux-amd64.tgz
          tar -xvzf malta.tgz
      - name: build
        run: ./linux-amd64/malta build
      - name: upload pages artifact
        uses: actions/upload-pages-artifact@v1
        with:
          path: dist

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v1
```
