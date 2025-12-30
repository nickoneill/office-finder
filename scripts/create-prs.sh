#!/bin/bash
set -e

# Configuration
FORK_REPO="5calls/congress-legislators"
UPSTREAM_REPO="unitedstates/congress-legislators"
CHANGES_FILE="changes.json"
RESULTS_DIR="results"

# Check if changes.json exists
if [ ! -f "$CHANGES_FILE" ]; then
    echo "No changes.json found. Run 'office-finder compare' first."
    exit 0
fi

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo "GitHub CLI (gh) is required but not installed."
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is required but not installed."
    exit 1
fi

# Read changes
CHANGES=$(cat "$CHANGES_FILE")
NUM_CHANGES=$(echo "$CHANGES" | jq '.changes | length')

if [ "$NUM_CHANGES" -eq 0 ]; then
    echo "No changes to process."
    exit 0
fi

echo "Processing $NUM_CHANGES changes..."

# Clone fork if not already present
if [ ! -d "congress-legislators" ]; then
    echo "Cloning fork repository..."
    gh repo clone "$FORK_REPO" congress-legislators
fi

cd congress-legislators

# Configure git to use GH_TOKEN for push authentication
git remote set-url origin "https://x-access-token:${GH_TOKEN}@github.com/${FORK_REPO}.git"

# Sync with upstream
git remote add upstream "https://github.com/$UPSTREAM_REPO.git" 2>/dev/null || true
git fetch upstream
git checkout main
git reset --hard upstream/main

# Process each change
echo "$CHANGES" | jq -r '.changes[].bioguide' | while read -r BIOGUIDE; do
    echo "Processing $BIOGUIDE..."

    BRANCH_NAME="update-offices/$BIOGUIDE"
    RESULT_FILE="../$RESULTS_DIR/$BIOGUIDE.yaml"

    # Check if result file exists
    if [ ! -f "$RESULT_FILE" ]; then
        echo "  No result file for $BIOGUIDE, skipping..."
        continue
    fi

    # Check if PR already exists
    EXISTING_PR=$(gh pr list --repo "$FORK_REPO" --head "$BRANCH_NAME" --json number --jq '.[0].number' 2>/dev/null || echo "")
    if [ -n "$EXISTING_PR" ]; then
        echo "  PR #$EXISTING_PR already exists for $BIOGUIDE, skipping..."
        continue
    fi

    # Create branch and merge changes
    git checkout -B "$BRANCH_NAME" main

    # Merge the result into legislators-district-offices.yaml
    ../office-finder merge --bioguide "$BIOGUIDE" --target legislators-district-offices.yaml --results "../$RESULTS_DIR"

    # Check if there are actual changes to commit
    if ! git diff --quiet legislators-district-offices.yaml; then
        git add legislators-district-offices.yaml
        git commit -m "Update district offices for $BIOGUIDE"
        git push --force origin "$BRANCH_NAME"
        gh pr create --repo "$FORK_REPO" \
            --head "$BRANCH_NAME" \
            --base main \
            --title "Update district offices for $BIOGUIDE" \
            --body "Updated office information scraped from official website."
        echo "  Created PR for $BIOGUIDE"
    else
        echo "  No changes detected for $BIOGUIDE, skipping PR"
    fi

    git checkout main
done

echo "PR creation complete."
