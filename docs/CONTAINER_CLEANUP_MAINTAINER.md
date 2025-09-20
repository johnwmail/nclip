# Container Cleanup Workflow - Maintainer Guide

This document is for maintainers who need to understand, modify, or troubleshoot the container cleanup workflow.

## Workflow Architecture

### File Location
`.github/workflows/cleanup-container-images.yml`

### Workflow Components

1. **Triggers**
   - **Scheduled**: Monthly on 1st day at 02:00 UTC (`0 2 1 * *`)
   - **Manual**: Via GitHub Actions UI with customizable parameters

2. **Jobs**
   - **cleanup**: Single job that handles validation, API calls, and cleanup

3. **Steps**
   - **Input Validation**: Validates parameters and sets environment variables
   - **Cleanup Logic**: Uses `actions/github-script@v7` to interact with GitHub Packages API
   - **Summary**: Provides final status and tips

## Key Design Decisions

### Safety First
- **Dry Run Default**: All manual triggers default to dry run mode
- **Multiple Safeguards**: Latest tag preservation, keep latest count, age-based retention
- **Comprehensive Logging**: Detailed output for every decision made

### Flexibility
- **Configurable Parameters**: Retention days, keep count, dry run mode
- **Manual Override**: Can be triggered manually with custom settings
- **Environment Agnostic**: Works with any repository using GitHub Container Registry

### Error Handling
- **Graceful Degradation**: Continues processing even if some deletions fail
- **Clear Error Messages**: Specific error handling for permissions, API issues
- **Detailed Logging**: Every action is logged for troubleshooting

## API Integration

### GitHub Packages API
The workflow uses the following endpoints:
- `GET /orgs/{org}/packages/container/{package_name}/versions` - List package versions
- `DELETE /orgs/{org}/packages/container/{package_name}/versions/{package_version_id}` - Delete specific version

### Authentication
- Uses `GITHUB_TOKEN` with `packages: write` permission
- Scoped to the repository where the workflow runs
- No additional secrets or configuration required

## Configuration Parameters

### Environment Variables
Set in the workflow file:
- `REGISTRY`: `ghcr.io` (GitHub Container Registry)
- `IMAGE_NAME`: `${{ github.repository }}` (automatically set to repo name)

### Input Parameters (Manual Triggers)
- `retention_days`: Number of days to keep images (default: 30)
- `dry_run`: Preview mode without actual deletion (default: true)
- `keep_latest_count`: Number of newest versions to always preserve (default: 5)

## Logic Flow

1. **Parameter Validation**
   - Validate numeric inputs are positive integers
   - Set defaults for scheduled runs
   - Export validated values to environment

2. **Package Discovery**
   - Query GitHub Packages API for all container versions
   - Sort by creation date (newest first)
   - Log package details for transparency

3. **Retention Analysis**
   - Calculate cutoff date based on retention period
   - Identify packages to always keep (latest N versions)
   - Analyze remaining packages against retention rules

4. **Safe Deletion**
   - Never delete packages with 'latest' tag
   - Never delete packages within keep latest count
   - Only delete packages older than retention period
   - Log every decision (keep/delete) with reasoning

5. **Execution**
   - In dry run mode: only log what would be deleted
   - In live mode: actually delete packages via API
   - Handle and log any errors without stopping execution

## Monitoring and Alerts

### Built-in Monitoring
- Detailed console output for every execution
- Summary statistics (kept/deleted/total)
- Error reporting with context

### Recommended Monitoring
- Monitor workflow execution in GitHub Actions
- Set up alerts for workflow failures
- Review cleanup summaries periodically

## Common Maintenance Tasks

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

## Troubleshooting

### Common Issues

#### Permission Denied (403)
- Check workflow has `packages: write` permission
- Verify repository settings allow package deletion
- Ensure actor has appropriate permissions

#### Package Not Found (404)
- Normal if no container images exist
- May indicate package name mismatch
- Check repository has published container images

#### API Rate Limiting
- GitHub API has rate limits for package operations
- Workflow includes appropriate delays
- Consider reducing frequency for repositories with many images

#### Partial Failures
- Some deletions may fail due to timing (image being pulled)
- Workflow continues and reports which deletions failed
- Failed deletions can be retried in next run

### Debugging Steps

1. **Review Workflow Logs**
   - Check GitHub Actions logs for detailed output
   - Look for specific error messages and package IDs

2. **Manual API Testing**
   - Use GitHub CLI or API to manually query packages
   - Verify package names and IDs match expectations

3. **Test with Dry Run**
   - Always use dry run mode first when debugging
   - Verify logic before enabling actual deletion

## Security Considerations

### Permissions
- Workflow only has access to packages in the same repository
- Cannot delete source code, releases, or other repository content
- Uses minimal required permissions (`packages: write`)

### Access Control
- Only repository collaborators can trigger manual runs
- Scheduled runs cannot be modified without repository access
- All actions are logged and auditable

### Safety Mechanisms
- Multiple layers of protection against accidental deletion
- Dry run mode for testing
- Comprehensive logging for accountability

## Testing Changes

Before modifying the workflow:

1. **Test in Fork**: Create a fork and test changes there first
2. **Use Dry Run**: Always test with dry run mode enabled
3. **Review Logs**: Carefully review all log output before going live
4. **Start Conservative**: Use longer retention periods initially
5. **Monitor Results**: Watch the first few executions closely

## Related Documentation

- [Container Cleanup User Guide](CONTAINER_CLEANUP.md)
- [GitHub Packages API Documentation](https://docs.github.com/en/rest/packages)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)