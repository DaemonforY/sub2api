# Model Pricing Data

This directory contains a local copy of the mirrored model pricing data as a fallback mechanism.

## Source
The original file is maintained by the LiteLLM project and mirrored into the `price-mirror` branch of this repository via GitHub Actions:
- Mirror branch (configurable via `PRICE_MIRROR_REPO`): https://raw.githubusercontent.com/<your-repo>/price-mirror/model_prices_and_context_window.json
- Upstream source: https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json

## Purpose
This local copy serves as a fallback when the remote file cannot be downloaded due to:
- Network restrictions
- Firewall rules
- DNS resolution issues
- GitHub being blocked in certain regions
- Docker container network limitations

## Update Process
The pricingService will:
1. First attempt to download the latest version from GitHub
2. If download fails, use this local copy as fallback
3. Log a warning when using the fallback file

## Manual Update
To manually update this file with the latest pricing data (if automation is unavailable):
```bash
curl -s https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json -o model_prices_and_context_window.json
```

## File Format
The file contains JSON data with model pricing information including:
- Model names and identifiers
- Input/output token costs
- Context window sizes
- Model capabilities

Last updated: 2025-08-10

## HiveGPT price patches (`hivegpt_overrides.json`)

`deploy/docker-compose.yml` points `PRICING_OVERRIDE_FILE` at this file. Its entries are merged field by field
over the synced catalog and win over it; the file is re-read when it changes.

- `gpt-5.6-sol` (also billed for its alias `gpt-5.6`): OpenAI's promotional price ($4 in / $0.40 cached / $20 out
  per 1M, >272K $8 / $0.80 / $30, Fast, Batch and Flex tiers to match), which the catalog lists only under
  `gpt-5.6`. OpenAI says the promotion runs at least through 2026-11-21 — **delete this entry when it ends**,
  and the catalog's `gpt-5.6-sol` price applies again.
