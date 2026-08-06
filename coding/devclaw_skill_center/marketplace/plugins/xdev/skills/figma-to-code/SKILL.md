---
name: figma-to-code
description: "Convert Figma design URLs to production-ready JSX/HTML code with externalized assets. Use this skill whenever the user provides a Figma link and wants to generate frontend code, create a React component from a design, preview a Figma design as HTML, or mentions anything about converting/exporting Figma designs to code."
argument-hint: "[figma-url] [--html] [--output path]"
allowed-tools: Read, Write, Bash, Glob, Grep
---

# Figma to Code

Convert Figma designs to production-ready JSX components or HTML preview pages, with images and SVGs saved as separate asset files.

## Prerequisites

The `FIGMA_ACCESS_TOKEN` environment variable must be set. If it's missing, tell the user to get a token at https://www.figma.com/developers/api#access-tokens and set it:

```bash
export FIGMA_ACCESS_TOKEN="figd_xxxxx"
```

## CLI

The conversion CLI lives in this repo:

```bash
npx -p @byted/x-figma-to-code figma-to-code <figma-url> [options]
```

## Workflow

### 1. Parse the user request

Extract from `$ARGUMENTS`:
- **Figma URL** (required) — a `figma.com/design/...` or `figma.com/file/...` URL
- **Output mode** — JSX (default) or HTML preview (`--html` flag or user says "html" / "preview")
- **Output path** — where to write files (default: `./figma-output/`)

### 2. Determine output mode

| User says | html-mode | File extension | Wrap |
|-----------|-----------|----------------|------|
| _(default)_ / "jsx" / "react" / "component" | `jsx` | `.jsx` | `--full-file` (React component) |
| "html" / "preview" / "page" | `html` | `.html` | `--full-file` (full HTML document) |

### 3. Run the conversion

```bash
npx -p @byted/x-figma-to-code figma-to-code "<figma-url>" \
  --framework HTML \
  --html-mode <jsx|html> \
  --full-file \
  --no-embed-vectors \
  --no-embed-images \
  --output <output-path>
```

Key flags explained:
- `--framework HTML` — use the HTML/JSX converter (covers both jsx and html modes)
- `--full-file` — wrap output in a complete file structure (React component for jsx, HTML document for html)
- `--no-embed-vectors` — save SVGs as separate files in `assets/` instead of inlining
- `--no-embed-images` — save images as separate files in `assets/` instead of base64

The CLI outputs a JSON object to **stdout** containing both the file metadata and conversion details. Progress messages go to stderr.

The CLI will create:
- The entry file at `<output-path>` (e.g. `./figma-output/MyComponent.jsx`)
- An `assets/` directory next to it with SVG and image files

### 4. Parse the JSON output

The stdout output is a JSON object with this structure:

```json
{
  "fileKey": "abc123XYZ",
  "fileName": "My Design File",
  "nodeId": "1:2",
  "version": "1234567890",
  "lastModified": "2025-01-15T10:30:00Z",
  "mode": "code",
  "outputPath": "/absolute/path/to/MyComponent.jsx",
  "framework": "HTML",
  "assets": ["/absolute/path/to/assets/icon.svg"]
}
```

Fields:
- `fileKey` — the Figma file identifier extracted from the URL
- `fileName` — the human-readable name of the Figma file
- `nodeId` — the specific node that was converted
- `version` — the current file version string (use this for version tracking/comparison)
- `lastModified` — ISO timestamp of when the file was last modified
- `mode` — always `"code"` in this context
- `outputPath` — absolute path to the generated code file
- `framework` — the framework used for conversion
- `assets` — array of absolute paths to asset files (SVGs, images)

### 5. Present the result

After conversion succeeds:
1. Read the generated entry file and show a brief summary (component name, line count)
2. List the asset files created in `assets/`
3. Report the file metadata for version management: fileName, fileKey, nodeId, version, lastModified
4. If html mode was used, tell the user they can open the `.html` file directly in a browser for a high-fidelity preview

## User Request

$ARGUMENTS

Now execute the conversion following the workflow above. If the user didn't specify a mode, default to JSX. If the user didn't specify an output path, use `./figma-output/` with a filename derived from the Figma node name.
