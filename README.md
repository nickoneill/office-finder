# office-finder
### Find office locations and contact info for US representatives using Claude AI

Keeping updated lists of office locations and phone numbers for US representatives is a difficult job. A large portion of the US House changes every two years, and offices are usually added well after a representative gets their first official website on house.gov, and that's aside from the normal operational changes for existing House or Senate members as offices move around or change phone numbers.

In the past we've relied on humans to update these numbers, either through trial and error (disconnected numbers are often reported to [5 Calls](https://5calls.org)) or attempts to automate human discovery via systems like [mechanical turk](https://github.com/TheWalkers/congress-turk).

Large language models give us a compelling tool to gather this information quickly *and* accurately. By asking a generally trained language model to extract addresses and phone numbers from websites, we can recheck these websites frequently and maintain a more up-to-date list of office information.

This tool uses a **two-tier model approach** for cost efficiency:
- **Claude Haiku** for cheap navigation to find contact pages
- **Claude Sonnet** for accurate office data extraction

Results are cached to avoid redundant navigation, and the tool is designed to contribute data back to the [unitedstates/congress-legislators](https://github.com/unitedstates/congress-legislators/blob/main/legislators-district-offices.yaml) repo.

## Setup

1. Install Go 1.21+ on your machine
2. Copy `.env.example` to `.env` and add your Anthropic API key:
   ```
   ANTHROPIC_API_KEY=your-key-here
   ```
3. Build the tool:
   ```bash
   go build -o office-finder .
   ```

## Commands

### navigate
Find the contact/offices page URL for a representative website:
```bash
./office-finder navigate pelosi.house.gov
# Output: {"contact_url":"https://pelosi.house.gov/contact/offices"}
```

### extract
Extract office information from a specific page:
```bash
./office-finder extract --url "https://pelosi.house.gov/contact/offices" --bioguide P000197
# Output: YAML in district-offices format
```

### batch
Process multiple legislators with caching:
```bash
# Process all legislators
./office-finder batch

# Filter by state
./office-finder batch --state CA

# Process a single legislator
./office-finder batch --bioguide P000197

# Enable debug output
./office-finder batch --bioguide P000197 --debug
```

Results are saved to `results/{bioguide}.yaml` and contact URLs are cached in `cache/contact-urls.json`.

### compare
Compare results with upstream district-offices.yaml:
```bash
./office-finder compare
# Output: changes.json with list of changed bioguides
```

### lintYAML
Re-sort the YAML file and fill in missing IDs:
```bash
./office-finder lintYAML
```

## Automated Workflow

The `.github/workflows/update-offices.yml` workflow runs regularly to:
1. Process all legislators in batch mode
2. Compare results with upstream
3. Create PRs for any changes

Required secrets:
- `ANTHROPIC_API_KEY` - For Claude API calls
- `GH_PAT` - GitHub Personal Access Token for creating PRs
