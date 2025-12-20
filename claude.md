# Claude Code Instructions

This file contains important instructions and conventions for Claude Code when working on this repository.

## Deployment Guidelines

### Never Manually Edit Remote Configuration Files

When deploying to BeagleBone devices (bbb1.wegman, bbb2.wegman, etc.):

1. **Always use Makefiles for deployments** - Each component has a Makefile with proper deployment targets
2. **Never SSH and manually edit files** - Configuration should be tracked in the repository and deployed via Makefile
3. **Configuration files are tracked in the repo** - Edit the local version, commit, then deploy

### Key Deployment Files and Their Makefiles

| Local File | Makefile Location | Deploy Command |
|------------|-------------------|----------------|
| `hardware/device-tree/uEnv.txt` | `hardware/device-tree/Makefile` | `make deploy-uenv` |
| `hardware/device-tree/*.dts` | `hardware/device-tree/Makefile` | `make deploy-full` |
| `pru/remote/*.c` | `api/Makefile` | `make deploy-pru` |
| `pru/audio/*.c` (I2S) | `pru/audio/Makefile` | `make i2s-deploy-all` |
| `pru/audio/*.c` (ADC) | `pru/audio/Makefile` | `make deploy-all` |
| Go API binary | `api/Makefile` | `make deploy` |

### Deployment Workflow

1. Edit the file in the local repository
2. Commit the change with a descriptive message
3. Use the appropriate `make deploy-*` command
4. If a reboot is required (e.g., uEnv.txt changes), inform the user

### Example: Updating uEnv.txt

```bash
# 1. Edit the local file
vim hardware/device-tree/uEnv.txt

# 2. Commit the change
git add hardware/device-tree/uEnv.txt
git commit -m "Update overlay configuration in uEnv.txt"

# 3. Deploy using Makefile
cd hardware/device-tree
make deploy-uenv DEPLOY_HOST=bbb1.wegman

# 4. Reboot if needed
ssh bbb1.wegman 'sudo reboot'
```

## PRU Shared Memory

The PRU shared memory layout is defined in `pru/pru_shared_memory.h`. When deploying PRU firmware, this header must be copied alongside the firmware files. The Makefiles handle this automatically.

## Commit Guidelines

- Commit frequently as changes are made so progress can be tracked
- Use descriptive commit messages explaining the "why" not just the "what"
- Include the Claude Code signature in commits
