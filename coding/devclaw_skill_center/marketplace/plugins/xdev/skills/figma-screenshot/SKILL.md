---
name: figma-screenshot
description: "Take a screenshot of a Figma design and return file metadata for version management. Use this skill whenever the user wants to capture, screenshot, snapshot, or export a Figma design as an image (PNG/JPG/SVG/PDF), or when the user wants to get the current version/metadata of a Figma file for tracking changes. Also triggers when the user provides a Figma link and asks for a 'picture', 'image', 'preview image', or 'visual snapshot' of the design."
argument-hint: "[figma-url] [--format png|jpg|svg|pdf] [--scale N] [--output path]"
allowed-tools: Read, Bash, Glob
---

# Figma Screenshot

Capture a screenshot of a Figma design node and return structured metadata (fileKey, fileName, nodeId, version) for version management.

## CLI

```bash
npx -p @byted/x-figma-to-code figma-to-code <figma-url> --screenshot [options]
```

## Workflow

### 1. Parse the user request

Extract from `$ARGUMENTS`:
- **Figma URL** (required) — a `figma.com/design/...` or `figma.com/file/...` URL. The URL must contain a `node-id` query parameter (e.g., `?node-id=1:2`).
- **Format** — `png` (default), `jpg`, `svg`, or `pdf`
- **Scale** — integer scale factor, default `1` (1x resolution)
- **Output path** — where to write the image file (default: auto-generated based on node ID)

### 2. Run the screenshot command

```bash
npx -p @byted/x-figma-to-code figma-to-code "<figma-url>" \
  --screenshot \
  --format <format> \
  --scale <scale> \
  --output <output-path>
```

The CLI outputs a JSON object to **stdout** containing both the file metadata and screenshot details. Progress messages go to stderr.

### 3. Parse the JSON output

The stdout output is a JSON object with this structure:

```json
{
  "fileKey": "abc123XYZ",
  "fileName": "My Design File",
  "nodeId": "1:2",
  "version": "1234567890",
  "lastModified": "2025-01-15T10:30:00Z",
  "mode": "screenshot",
  "outputPath": "/absolute/path/to/screenshot.png",
  "format": "png",
  "scale": 1
}
```

Fields:
- `fileKey` — the Figma file identifier extracted from the URL
- `fileName` — the human-readable name of the Figma file
- `nodeId` — the specific node that was captured
- `version` — the current file version string (use this for version tracking/comparison)
- `lastModified` — ISO timestamp of when the file was last modified
- `mode` — always `"screenshot"` in this context
- `outputPath` — absolute path to the saved image file
- `format` — the image format used
- `scale` — the scale factor used

### 4. Present the result

After the command succeeds:
1. Show the user where the image was saved (the `outputPath` from the JSON)
2. Report the file metadata for version management: fileName, fileKey, nodeId, version, lastModified
3. If the user needs to track versions, they can compare the `version` and `lastModified` values across runs to detect design changes

## Error handling

- If the Figma URL has no `node-id` parameter, the CLI will error. Tell the user to include a specific node in the URL (they can get this by right-clicking a frame in Figma and choosing "Copy link").

## User Request

$ARGUMENTS

Now execute the screenshot following the workflow above. Default to PNG format at 1x scale if the user doesn't specify.
