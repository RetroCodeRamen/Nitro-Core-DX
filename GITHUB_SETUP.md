# GitHub Repository Setup Guide

This guide will help you set up a public GitHub repository for this project.

## Prerequisites

1. **Install Git** (if not already installed):
   - Download from: https://git-scm.com/download/win
   - Or install via GitHub Desktop: https://desktop.github.com/
   - Or use winget: `winget install Git.Git`

2. **Create a GitHub Account** (if you don't have one):
   - Go to: https://github.com
   - Sign up for a free account

## Step 1: Initialize Git Repository

Open a terminal in the project directory and run:

```bash
# Initialize git repository
git init

# Configure git (if not already done globally)
git config user.name "Your Name"
git config user.email "your.email@example.com"
```

## Step 2: Add Files and Create Initial Commit

```bash
# Add all files (respects .gitignore)
git add .

# Create initial commit
git commit -m "Initial commit: Fantasy Console Emulator with Matrix Mode"
```

## Step 3: Create GitHub Repository

1. Go to https://github.com/new
2. Repository name: `fantasy-console-emulator` (or your preferred name)
3. Description: "A fantasy console emulator with custom 16-bit CPU, tile-based graphics, and Matrix Mode for 3D-style effects"
4. Set to **Public**
5. **DO NOT** initialize with README, .gitignore, or license (we already have these)
6. Click "Create repository"

## Step 4: Connect Local Repository to GitHub

After creating the repository, GitHub will show you commands. Use these:

```bash
# Add remote repository (replace YOUR_USERNAME with your GitHub username)
git remote add origin https://github.com/YOUR_USERNAME/fantasy-console-emulator.git

# Rename default branch to main (if needed)
git branch -M main

# Push to GitHub
git push -u origin main
```

## Step 5: Add Repository Topics (Optional)

On your GitHub repository page:
1. Click the gear icon next to "About"
2. Add topics like: `emulator`, `python`, `game-development`, `retro`, `fantasy-console`, `matrix-mode`

## Step 6: Update README (Optional)

Consider adding:
- Badges (build status, license, etc.)
- Screenshots/GIFs of the emulator in action
- Link to the programming manual
- Contributing guidelines

## Future Updates

To push future changes:

```bash
# Stage changes
git add .

# Commit with message
git commit -m "Description of changes"

# Push to GitHub
git push
```

## Troubleshooting

**If git is not recognized:**
- Make sure Git is installed and added to PATH
- Restart your terminal after installation
- Try using GitHub Desktop instead

**If you get authentication errors:**
- GitHub no longer accepts passwords for HTTPS
- Use a Personal Access Token: https://github.com/settings/tokens
- Or use SSH keys: https://docs.github.com/en/authentication/connecting-to-github-with-ssh

**If you want to use GitHub Desktop:**
1. Install GitHub Desktop
2. File → Add Local Repository
3. Select this project folder
4. Publish repository to GitHub
