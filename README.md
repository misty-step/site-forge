# Site Forge

Quality verification harness for static websites — Lighthouse, asset checks, vision grading.

## Overview

Site Forge is a Go CLI tool that verifies the quality of a built static website. It's designed as the hard gate in the Misty Step site pipeline — nothing deploys without passing Site Forge checks.

## Installation

```bash
# Clone the repository
git clone https://github.com/misty-step/site-forge.git
cd site-forge

# Build the binary
make build

# Or install to your PATH
make install
```

## Usage

```bash
site-forge verify ./dist [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | `./dist` | Directory to verify (required) |
| `--baseline` | - | Baseline directory for vision comparison |
| `--threshold` | `7` | Vision score threshold (1-10) |
| `--lighthouse-perf` | `90` | Lighthouse performance threshold |
| `--lighthouse-a11y` | `90` | Lighthouse accessibility threshold |
| `--lighthouse-seo` | `90` | Lighthouse SEO threshold |

### Examples

```bash
# Basic verification (no vision check)
site-forge verify ./dist

# Full verification with vision comparison
site-forge verify ./dist --baseline ./reference/original

# Custom thresholds
site-forge verify ./dist --lighthouse-perf 95 --threshold 8
```

## Check Pipeline

Site Forge runs these checks in order, failing fast on critical checks:

1. **ASSETS** - Verifies all referenced files (images, CSS, JS) exist
2. **BUILD** - Validates HTML structure and meta tags
3. **LIGHTHOUSE** - Runs Lighthouse audit for performance, accessibility, SEO
4. **SCREENSHOTS** - Captures desktop (1280x900) and mobile (390x844) screenshots
5. **VISION** - Compares redesign with baseline using AI vision model

## Exit Codes

- `0` - All checks passed
- `1` - At least one check failed

## Output

A JSON report is written to `forge-report.json`:

```json
{
  "timestamp": "2026-02-15T05:30:00Z",
  "directory": "./dist",
  "overall": "PASS",
  "checks": {
    "assets": { "status": "PASS", "total": 42 },
    "build": { "status": "PASS", "pages": 1 },
    "lighthouse": { "status": "PASS", "performance": 95, "accessibility": 98, "seo": 100 },
    "screenshots": { "status": "PASS", "desktop": "screenshots/desktop.png", "mobile": "screenshots/mobile.png" },
    "vision": { "status": "SKIP" }
  }
}
```

## Requirements

- **Go 1.23+**
- **Node.js** (for Lighthouse via `npx`)
- **Chrome/Chromium** (for screenshots)
- **OPENROUTER_API_KEY** environment variable (for vision check)

## Development

```bash
# Run tests
make test

# Build
make build

# Install
make install
```

## License

MIT
