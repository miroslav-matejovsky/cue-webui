<#
.SYNOPSIS
    Creates and pushes a release tag to trigger the GitHub Actions release workflow.

.DESCRIPTION
    This script validates the provided version string, checks that the current
    branch is 'main', creates a Git tag in the form 'v<version>', and pushes
    it to the upstream remote. Pushing the tag triggers the release GitHub
    Actions workflow, which builds binaries for all supported platforms and
    publishes them as a GitHub release.

.PARAMETER Version
    The semantic version to release, e.g. "1.2.3". The 'v' prefix is added
    automatically. Must follow Semantic Versioning (MAJOR.MINOR.PATCH).

.EXAMPLE
    .\scripts\release.ps1 -Version 1.0.0

    Creates and pushes the tag 'v1.0.0'.
#>
param(
    [Parameter(Mandatory)]
    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Validate semantic version format (MAJOR.MINOR.PATCH, all numeric parts).
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    Write-Error "Invalid version '$Version'. Expected format: MAJOR.MINOR.PATCH (e.g. 1.2.3)"
    exit 1
}

$tag = "v$Version"

# Ensure we are on the main branch.
$branch = git rev-parse --abbrev-ref HEAD
if ($branch -ne 'main') {
    Write-Error "Releases must be created from the 'main' branch. Current branch: '$branch'"
    exit 1
}

# Check there are no uncommitted changes.
$status = git status --porcelain
if ($status) {
    Write-Error "Working tree is not clean. Commit or stash your changes before releasing."
    exit 1
}

# Make sure local main is up-to-date with the remote.
Write-Host "Fetching remote..."
git fetch origin main

$local  = git rev-parse HEAD
$remote = git rev-parse FETCH_HEAD
if ($local -ne $remote) {
    Write-Error "Local 'main' is not up-to-date with 'origin/main'. Pull the latest changes first."
    exit 1
}

# Check the tag doesn't already exist locally or remotely.
$existingTag = git tag --list $tag
if ($existingTag) {
    Write-Error "Tag '$tag' already exists locally."
    exit 1
}

Write-Host "Creating tag $tag..."
git tag $tag

Write-Host "Pushing tag $tag to origin..."
git push origin $tag

Write-Host "Tag $tag pushed. The GitHub Actions release workflow will now build and publish the release."
