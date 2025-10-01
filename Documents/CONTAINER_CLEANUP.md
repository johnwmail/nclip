# Container Image Cleanup & Maintainer Guide

This document explains the automated container image cleanup system for the nclip repository and provides a maintainer reference for the workflow.

---

## Overview

The nclip repository uses GitHub Container Registry (ghcr.io) to store Docker images. Over time, these images can accumulate and consume significant storage space. The container cleanup workflow automatically removes old images while preserving recent and important versions.

---

## Cleanup Workflow

The cleanup is handled by the `.github/workflows/cleanup-container-images.yml` workflow.

### Schedule & Triggers
- **Automatic**: Runs monthly on the 1st day at 02:00 UTC
- **Manual**: Can be triggered manually via GitHub Actions UI with custom parameters

### Retention Policy
1. **Always Keep Latest**: Images tagged as `latest` are never deleted
2. **Keep Recent Versions**: Always preserve the newest N versions (default: 5) regardless of age
3. **Age-Based Cleanup**: Delete images older than the retention period (default: 30 days)
4. **Safe Deletion**: Only removes container image versions, never affects source code or releases

### Configuration Options
| Parameter | Default | Description |
|-----------|---------|-------------|
| `retention_days` | 30 | Number of days to keep images |
| `dry_run` | true | Preview changes without deleting |
| `keep_latest_count` | 5 | Number of newest versions to always keep |

---

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

---

## Example Scenarios

### Scenario 1: Weekly Development Images
```yaml
retention_days: 7
keep_latest_count: 3
dry_run: false
```
Keeps only the last 3 versions plus any images from the last week.

### Scenario 2: Long-term Stable Releases
```yaml
retention_days: 90
keep_latest_count: 10
dry_run: false
```
Keeps 10 most recent versions plus anything from the last 3 months.

### Scenario 3: Aggressive Cleanup
```yaml
retention_days: 14
keep_latest_count: 2
dry_run: false
```
Only keeps 2 most recent versions plus images from the last 2 weeks.

---

## Maintainer Reference: Workflow Architecture & Logic

### File Location
`.github/workflows/cleanup-container-images.yml`

### Workflow Components
1. **Triggers**: Scheduled (monthly) and manual (with parameters)
2. **Jobs**: Single `cleanup` job for validation, API calls, and cleanup
3. **Steps**:
   - Input validation
   - Cleanup logic (via `actions/github-script@v7`)
   - Summary/status output

### Key Design Decisions
- **Safety First**: Dry run by default, multiple safeguards, comprehensive logging
- **Flexibility**: Configurable parameters, manual override, environment agnostic
- **Error Handling**: Graceful degradation, clear error messages, detailed logging

### API Integration
- Uses GitHub Packages API for listing and deleting images
- Authenticated with `GITHUB_TOKEN` (`packages: write` permission)

### Logic Flow
1. Parameter validation
2. Package discovery and sorting
3. Retention analysis (cutoff date, keep latest, age-based)
4. Safe deletion (never delete `latest`, always keep N newest, only delete old)
5. Execution (dry run or live)
6. Error handling and logging

---

## What Gets Cleaned Up

### Images That May Be Deleted
- Container images older than the retention period
- Images without the `latest` tag
- Images beyond the "keep latest count" threshold

### Images That Are Always Preserved
- Any image tagged as `latest`
- The N most recent images (based on `keep_latest_count`)
- Images newer than the retention period

---

## Monitoring & Troubleshooting

### Monitoring
- Detailed logs: images found, kept, deleted, summary, errors
- Review workflow logs in GitHub Actions

### Troubleshooting
- **Permission Errors (403)**: Ensure `packages: write` permission, repo allows deletion
- **Package Not Found (404)**: Normal if no images exist or already deleted
- **Failed Deletions**: May occur if images are in use or API issues; workflow continues and reports failures
- **API Rate Limiting**: Workflow includes delays; reduce frequency if needed
- **Debugging**: Review logs, test with dry run, use GitHub CLI/API for manual checks

---

## Best Practices
1. **Start with Dry Run**: Always test your configuration with `dry_run: true`
2. **Conservative Retention**: Start with longer retention periods and adjust as needed
3. **Monitor Storage**: Watch your package storage usage in GitHub
4. **Regular Cleanup**: Don't let images accumulate for too long
5. **Preserve Important Tags**: Ensure critical images are tagged appropriately

---

## Maintenance Tasks

### Changing Default Retention
Edit the workflow file:
```yaml
retention_days:
  description: 'Number of days to keep images (default: 30)'
  default: '30'  # Change this value
```

### Changing Schedule
Edit the cron expression:
```yaml
schedule:
  - cron: '0 2 1 * *'  # Monthly
  # Examples:
  # - cron: '0 2 * * 0'    # Weekly (Sundays)
  # - cron: '0 2 1,15 * *' # Bi-monthly (1st and 15th)
```

### Adding New Safety Checks
Modify the JavaScript logic in the `actions/github-script` step to add additional conditions.

---

## Security Considerations
- Workflow only has access to packages in the same repository
- Cannot delete source code, releases, or other repository content
- Uses minimal required permissions (`packages: write`)
- Only repository collaborators can trigger manual runs
- All actions are logged and auditable
- Multiple layers of protection: dry run, logging, tag/age safeguards

---

## Related Workflows & Documentation
- **Container Build** (`.github/workflows/container.yml`): Builds and pushes new images
- **Release** (`.github/workflows/release.yml`): Creates releases and triggers container builds
- [GitHub Packages Documentation](https://docs.github.com/en/packages)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Managing Package Versions](https://docs.github.com/en/packages/learn-github-packages/deleting-and-restoring-a-package)

---

## See Also
- [Container Cleanup User Guide](CONTAINER_CLEANUP.md)
- [GitHub Packages API Documentation](https://docs.github.com/en/rest/packages)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)