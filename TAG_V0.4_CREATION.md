# Tag v0.4 Creation Documentation

This document records the creation of tag v0.4 from the main branch.

## Tag Details
- **Tag Name**: v0.4
- **Target Commit**: 9ab32cd97b88e597fe266d9adea0405f4a207888 (main branch)
- **Tag Message**: "Release version 0.4 from main branch"
- **Created**: $(date)

## Verification
The tag was created locally and points to the correct commit from the main branch as requested.

```bash
git tag -n1 v0.4
# Output: v0.4            Release version 0.4 from main branch

git rev-parse v0.4^{commit}
# Output: 9ab32cd97b88e597fe266d9adea0405f4a207888
```

This tag represents version 0.4 of the nclip project.