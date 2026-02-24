# Creating a GitHub Release (Nitro-Core-DX)

This project is set up to build release binaries and publish a GitHub Release page automatically when you push a version tag.

## What Happens Automatically

On tag push (example: `v0.1.0`):

- GitHub Actions builds **Linux amd64** and **Windows amd64** packages for the integrated Nitro-Core-DX app
- A GitHub Release page is created/updated for that tag
- Release binaries are attached to the release page
- GitHub generated release notes are enabled (with optional category grouping via `.github/release.yml`)

Workflow:

- `.github/workflows/release-binaries.yml`

## Release Steps (Maintainer)

1. Commit your final changes
2. Push `main`
3. Create and push a tag

```bash
git tag v0.1.0
git push origin v0.1.0
```

4. Open GitHub -> **Actions** and confirm the workflow succeeds
5. Open GitHub -> **Releases** and verify:
   - Linux archive attached (`.tar.gz`)
   - Windows archive attached (`.zip`)
   - Release notes look reasonable

## Manual (No Tag) Test

You can run the workflow manually with **Actions -> Build Nitro-Core-DX Release Binaries -> Run workflow**.

- This uploads build artifacts to the workflow run
- It does **not** publish a release page unless the run is for a tag ref

## Local Linux Packaging (Optional)

For local testing of the package format:

```bash
make release-linux
```

This creates a Linux release archive in `dist/`.
