#!/usr/bin/env bash

#!/bin/bash

# Navigate to the top-level directory of the Git repository
cd "$(git rev-parse --show-toplevel)"

# Step 1: Add all changes to git
git add .

# Step 2: Commit the changes with a message
git commit -m ":rocket:"

# Step 3: Generate a new tag based on the latest tag, incrementing the patch version
# Fetch all tags
git fetch --tags

# Get the latest tag name
latest_tag=$(git describe --tags "$(git rev-list --tags --max-count=1)")

# If there are no tags yet, start with v0.0.0-beta

# Break the tag into parts
base=$(echo $latest_tag | sed 's/-beta//') # Remove -beta suffix if present
major=$(echo "$base" | cut -d. -f1)
minor=$(echo "$base" | cut -d. -f2)
patch=$(echo "$base" | cut -d. -f3)

# Increment the patch version
new_patch=$((patch + 1))

# Construct the new tag
new_tag="${major}.${minor}.${new_patch}-beta"

# Create the new tag
git tag "$new_tag"

# Step 4: Push the new tag to the repository
git push origin "$new_tag"

echo "Tag $new_tag created and pushed successfully."
