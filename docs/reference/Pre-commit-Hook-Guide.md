# Pre-commit Hook: Large File and Binary Detection

## Overview

This repository includes a pre-commit hook that automatically detects and blocks commits containing large files and binary files. This helps maintain repository health and prevents accidental commits of files that shouldn't be in version control.

## Configuration

The pre-commit hook is configured with the following defaults:

- **Maximum file size**: 10 MB
- **Binary file detection**: Automatic detection using file signatures and content analysis
- **Suspicious file extensions**: Comprehensive list of extensions typically not suitable for version control

## What Gets Blocked

### Large Files (>10MB)
Any file exceeding the size limit will be blocked, regardless of type.

### Binary Files
- Executables (`.exe`, `.dll`, `.so`, `.dylib`, `.app`)
- Compiled objects (`.o`, `.obj`, `.class`, `.pyc`)
- Archives (`.zip`, `.rar`, `.7z`, `.tar.gz`)
- Media files (`.jpg`, `.png`, `.mp4`, `.mp3`)
- Documents (`.pdf`, `.doc`, `.xls`, `.ppt`)
- Database files (`.db`, `.sqlite`, `.mdb`)

### Suspicious Files
Files with extensions commonly associated with:
- Build artifacts
- IDE-specific files
- OS-generated files
- Log files
- Temporary files

## Hook Behavior

When you attempt to commit:

1. 🔍 **Scan Phase**: All staged files are analyzed
2. 📊 **Size Check**: Files exceeding 10MB are flagged
3. 🔧 **Binary Detection**: Files are checked for binary content
4. ⚡ **Extension Analysis**: Suspicious extensions are identified
5. 📝 **Report**: Detailed findings are displayed
6. ✅/❌ **Decision**: Commit is allowed or blocked

## Example Output

### Successful Commit
```
🔍 Pre-commit Hook: Checking for large files and binaries...
📁 Checking 5 files...
✅ All files passed size and binary checks
🎉 Commit approved!
```

### Blocked Commit
```
🔍 Pre-commit Hook: Checking for large files and binaries...
📁 Checking 3 files...

❌ LARGE FILES DETECTED (>10MB):
   📄 assets/large-dataset.csv (25 MB)

❌ BINARY FILES DETECTED:
   🔧 build/myapp.exe (2 MB)

⚠️  SUSPICIOUS FILES DETECTED:
   ⚡ logs/application.log (500 KB)

🚫 COMMIT BLOCKED

💡 SOLUTIONS:
   📦 For large/binary files:
      • Use Git LFS: git lfs track "*.ext" && git add .gitattributes
      • Store in external storage (S3, CDN, artifact repository)
      • Add to .gitignore if not needed in version control
      • Split large files into smaller chunks

   🗂️  To remove files from staging:
      git reset HEAD <file>

   🚀 To override this hook (NOT RECOMMENDED):
      git commit --no-verify

   📋 Add to .gitignore to permanently exclude:
      echo "filename" >> .gitignore
```

## Solutions for Different File Types

### Large Data Files
```bash
# Option 1: Use Git LFS
git lfs track "*.csv"
git lfs track "*.json"
git add .gitattributes
git add large-file.csv
git commit -m "Add large data file with LFS"

# Option 2: External storage
# Upload to S3, document URL in README
echo "s3://bucket/large-file.csv" > data-sources.md
```

### Build Artifacts
```bash
# Add to .gitignore
echo "build/" >> .gitignore
echo "dist/" >> .gitignore
echo "*.exe" >> .gitignore
git add .gitignore
```

### Media Assets
```bash
# Option 1: Asset CDN
# Upload to CDN, reference in code

# Option 2: Git LFS for essential assets
git lfs track "*.png"
git lfs track "*.jpg"
git add .gitattributes
```

### Database Files
```bash
# Use schema/migration files instead
echo "*.db" >> .gitignore
echo "*.sqlite" >> .gitignore
# Keep schema.sql and migrations/ instead
```

## Temporarily Bypassing the Hook

**⚠️ Warning**: Only bypass the hook when absolutely necessary and you understand the implications.

```bash
# For a single commit (NOT RECOMMENDED)
git commit --no-verify -m "Emergency commit with large file"

# Temporarily disable the hook
mv .git/hooks/pre-commit .git/hooks/pre-commit.disabled
# Remember to re-enable it!
mv .git/hooks/pre-commit.disabled .git/hooks/pre-commit
```

## Customizing the Hook

To modify the configuration, edit `.git/hooks/pre-commit`:

```bash
# Change maximum file size (in MB)
MAX_FILE_SIZE_MB=5  # Change from 10MB to 5MB

# Add custom suspicious extensions
suspicious_extensions+=(
    "custom" "internal" "temp"
)
```

## Git LFS Integration

For projects that legitimately need large files, consider Git LFS:

```bash
# Initialize LFS
git lfs install

# Track large file types
git lfs track "*.psd"
git lfs track "*.zip"
git lfs track "*.bin"

# Commit the .gitattributes file
git add .gitattributes
git commit -m "Configure Git LFS tracking"
```

## Continuous Integration

The pre-commit hook works locally. For CI/CD pipelines, consider:

```bash
# In your CI script
if [ -f .git/hooks/pre-commit ]; then
    ./.git/hooks/pre-commit
fi
```

## Troubleshooting

### Hook Not Running
```bash
# Check if hook is executable
ls -la .git/hooks/pre-commit
# Should show: -rwxr-xr-x

# Make executable if needed
chmod +x .git/hooks/pre-commit
```

### False Positives
If the hook incorrectly identifies a text file as binary:

1. Check file encoding (should be UTF-8)
2. Remove null bytes or control characters
3. Verify file isn't corrupted

### Performance Issues
For repositories with many files:

1. The hook only checks staged files (not the entire repository)
2. Consider increasing file size limits for faster scans
3. Use `.gitignore` to prevent staging unwanted files

## Best Practices

1. **Use .gitignore proactively**: Add patterns before files are created
2. **Regular cleanup**: Periodically review and clean staged files
3. **Team education**: Ensure all team members understand the hook's purpose
4. **Git LFS for assets**: Use LFS for legitimate large files
5. **External storage**: Use cloud storage for large datasets
6. **Documentation**: Keep URLs/references to external assets in code comments

## Files Managed by This Hook

- **Location**: `.git/hooks/pre-commit`
- **Type**: Bash script
- **Permissions**: Executable (`chmod +x`)
- **Scope**: Local repository only (not shared via git)

## Team Setup

Since git hooks aren't shared via the repository, each team member needs to set up the hook:

```bash
# Option 1: Manual setup (each developer)
chmod +x .git/hooks/pre-commit

# Option 2: Setup script (recommended)
# Create setup.sh in repository root:
#!/bin/bash
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
echo "Pre-commit hook installed successfully!"
```

## Related Files

- `.gitignore`: Preventive measure to avoid staging unwanted files
- `.gitattributes`: Git LFS configuration
- `README.md`: Project documentation should mention large file policies
- CI/CD configs: May include similar checks for remote validation