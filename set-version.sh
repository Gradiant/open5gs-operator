#!/bin/bash
# Script to update the version in Makefile, kustomization.yaml, Chart.yaml, and values.yaml
# Usage: ./set-version.sh <new_version>

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <new_version>"
    exit 1
fi

NEW_VERSION="$1"

# Update Makefile
sed -i "s/^VERSION ?=.*/VERSION ?= $NEW_VERSION/" Makefile

# Update kustomization.yaml
sed -i "s/newTag: .*/newTag: $NEW_VERSION/" config/manager/kustomization.yaml

# Update Chart.yaml
sed -i "s/^version: .*/version: $NEW_VERSION/" charts/open5gs-operator/Chart.yaml
sed -i "s/^appVersion: .*/appVersion: \"$NEW_VERSION\"/" charts/open5gs-operator/Chart.yaml

# Update values.yaml (regardless of indentation)
sed -i "s/^\s*tag: .*/      tag: $NEW_VERSION/" charts/open5gs-operator/values.yaml

echo "Versions updated to $NEW_VERSION!"