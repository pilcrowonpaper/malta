---
title: "Commands"
---

# Commands

## `build`

Generates HTML files to the `dist` directory.

```
malta build
```

## `preview`

Runs a preview server on localhost (port 3000) for the generated site.

```
malta preview
```

```
malta preview --port 5000
```

**Note**: It's important to ensure you build your project before running the preview server to ensure that the generated site reflects the latest changes.

### Options

- `--port` (`-p`): Localhost port (number - `3000` by default)
