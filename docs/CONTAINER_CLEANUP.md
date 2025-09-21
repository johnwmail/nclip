# Container Image Cleanup

This document explains the automated container image cleanup system for the nclip repository.

## Overview

The nclip repository uses GitHub Container Registry (ghcr.io) to store Docker images. Over time, these images can accumulate and consume significant storage space. The container cleanup workflow automatically removes old images while preserving recent and important versions.

## Cleanup Workflow

The cleanup is handled by the `.github/workflows/cleanup-container-images.yml` workflow.

### Schedule

- **Automatic**: Runs monthly on the 1st day at 02:00 UTC
- **Manual**: Can be triggered manually via GitHub Actions UI with custom parameters

### Retention Policy

The cleanup workflow follows these rules:

1. **Always Keep Latest**: Images tagged as `latest` are never deleted
2. **Keep Recent Versions**: Always preserve the newest N versions (default: 5) regardless of age
3. **Age-Based Cleanup**: Delete images older than the retention period (default: 30 days)
4. **Safe Deletion**: Only removes container image versions, never affects source code or releases

### Configuration Options

When running manually, you can customize these parameters:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `retention_days` | 30 | Number of days to keep images |
| `dry_run` | true | Preview changes without deleting |
| `keep_latest_count` | 5 | Number of newest versions to always keep |

## How to Use

### Automatic Cleanup

The workflow runs automatically every month. No action is required.

### Manual Cleanup

1. Go to **Actions** tab in the GitHub repository
2. Select **Cleanup Container Images** workflow
3. Click **Run workflow**
4. Optionally adjust parameters:
   - Set `dry_run` to `false` to actually delete images
   - Adjust `retention_days` for different retention periods
   - Modify `keep_latest_count` to preserve more/fewer recent versions

### Dry Run Mode

**Always test with dry run first!** The workflow defaults to dry run mode, which:
- Shows what would be deleted
- Doesn't actually remove any images
- Helps you verify the cleanup plan before execution

## Example Scenarios

### Scenario 1: Weekly Development Images
```
retention_days: 7
keep_latest_count: 3
dry_run: false
```
Keeps only the last 3 versions plus any images from the last week.

### Scenario 2: Long-term Stable Releases
```
retention_days: 90
keep_latest_count: 10
dry_run: false
```
Keeps 10 most recent versions plus anything from the last 3 months.

### Scenario 3: Aggressive Cleanup
```
retention_days: 14
keep_latest_count: 2
dry_run: false
```
Only keeps 2 most recent versions plus images from the last 2 weeks.

## What Gets Cleaned Up

### Images That May Be Deleted
- Container images older than the retention period
- Images without the `latest` tag
- Images beyond the "keep latest count" threshold

### Images That Are Always Preserved
- Any image tagged as `latest`
- The N most recent images (based on `keep_latest_count`)
- Images newer than the retention period

## Monitoring

The workflow provides detailed logs showing:
- Number of images found
- Which images are kept and why
- Which images are deleted
- Summary of the cleanup operation
- Any errors encountered

## Security

The cleanup workflow:
- Uses the repository's `GITHUB_TOKEN` with minimal required permissions
- Only has access to packages in the same repository
- Cannot delete source code, releases, or other repository content
- Runs with `packages: write` permission only

## Troubleshooting

### Permission Errors
If you see 403 (Forbidden) errors:
- Ensure the workflow has `packages: write` permission
- Check that the repository is configured to allow package deletion

### Package Not Found
404 errors are normal if:
- No container images exist yet
- Images were already deleted
- Package names don't match the repository

### Failed Deletions
Some deletions might fail if:
- Images are being pulled during deletion
- There are temporary GitHub API issues
- Images are referenced by other systems

The workflow will continue and report which deletions failed.

## Best Practices

1. **Start with Dry Run**: Always test your configuration with `dry_run: true`
2. **Conservative Retention**: Start with longer retention periods and adjust based on needs
3. **Monitor Storage**: Keep an eye on your package storage usage in GitHub
4. **Regular Cleanup**: Don't let images accumulate for too long before running cleanup
5. **Preserve Important Tags**: Ensure critical images are tagged appropriately

## Related Workflows

- **Container Build** (`.github/workflows/container.yml`): Builds and pushes new images
- **Release** (`.github/workflows/release.yml`): Creates releases and triggers container builds

## Further Reading

- [GitHub Packages Documentation](https://docs.github.com/en/packages)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Managing Package Versions](https://docs.github.com/en/packages/learn-github-packages/deleting-and-restoring-a-package)