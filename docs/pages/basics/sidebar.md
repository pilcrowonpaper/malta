---
title: "Configuring the sidebar"
---

# Configuring the sidebar

Malta does not automatically generate the sidebar (navigation bar). You can define sections and pages with the `sidebar` config.

```json
{
  "sidebar": [
    {
      "title": "Basics",
      "pages": [
        ["Getting started", "/basics/setup"],
        ["Writing pages", "/basics/pages"],
        ["Configuring the sidebar", "/basics/sidebar"],
        ["Commands", "/basics/commands"]
      ]
    },
    {
      "title": "Guides",
      "pages": [["Deploying with GitHub Actions", "/guides/github-actions"]]
    }
  ]
}
```
