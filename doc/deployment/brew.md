# Homebrew deployment

- Checked: **2026-10-03** against the official refs below.
- Package names, owner, and URLs below are placeholders.

## Publishing routes

| Route | Install cmd | Approval |
|---|---|---|
| Public tap | `brew install <owner>/tap/ai-cross` | Publisher manages publication |
| Default `homebrew/core` catalog | `brew install ai-cross` | Homebrew review required |

- References: [Tap guide](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap), [Submission guide](https://docs.brew.sh/Adding-Software-to-Homebrew)

## Public tap

- Setup: Create a public GitHub repo named `<owner>/homebrew-tap`
  - `brew tap-new <owner>/tap` can generate the local tap structure
- Package: Add `Formula/ai-cross.rb`
  - Include release URL, SHA-256, license, dependencies, and build / install steps
  - Add a meaningful formula test
- Publish: Make the release download available, then push the formula to the public tap
  - No `homebrew/core` approval required
- Verify: Run on a clean supported machine
  - `brew install <owner>/tap/ai-cross`
  - Confirm the installed CLI reports the expected release version
- Update: Publish each release, then update the formula URL and checksum
  - Users receive formula updates through `brew update` and package updates through `brew upgrade`
- References: [Create and maintain a tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap), [Formula cookbook](https://docs.brew.sh/Formula-Cookbook)

## Default catalog: homebrew/core

- Setup: Check acceptance rules and search existing formulae / PRs before starting
- Release: Publish a stable, immutable source release
- Package: Fork `Homebrew/homebrew-core` and create a formula from the release URL
  - `brew create <release-url>` generates a starting template
  - Complete metadata, dependencies, build / install steps, and a meaningful test
- Validate: Run the local checks below and inspect installed files / output
- Submit: Open a PR to `Homebrew/homebrew-core`
  - Complete the PR template and disclose AI assistance when applicable
  - Resolve CI failures and reviewer feedback
- Publish: Wait for acceptance and catalog publication
- Verify: Run `brew update`, `brew info ai-cross`, and `brew install ai-cross`
  - Confirm the published and installed versions match the release
- References: [Submission guide](https://docs.brew.sh/Adding-Software-to-Homebrew), [PR workflow](https://docs.brew.sh/How-To-Open-a-Homebrew-Pull-Request)

## Core requirements

- License: Open source under a license compatible with Debian Free Software Guidelines
- Build: Native CLI software must build from source
- Release: Stable version with immutable source archive, tag, or revision
- Integrity: Verify downloaded archives with SHA-256
- Dependencies: Use reproducible versions; avoid moving or unchecksummed sources
- Platforms: Build and pass tests on the supported CI matrix; justified platform restrictions may be eligible
- Installation: Automate dependency resolution and installation
- Updates: Disable self-update behavior where possible so Homebrew manages versions
- Acceptance: Formula-specific rules and the shared package acceptance policy both apply
- References: [Acceptable formulae](https://docs.brew.sh/Acceptable-Formulae), [Acceptance policy links](https://docs.brew.sh/Adding-Software-to-Homebrew#choose-a-package-type)

## Local verification cmds

- Run these yourself from the contribution environment before submitting
  - `HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source ai-cross`
  - `brew test ai-cross`
  - `brew audit --strict --new --online ai-cross`
  - `brew style --fix --formula ai-cross`
  - `brew lgtm --online`
- Inspect installed files and CLI output in addition to command exit status
- References: [Local validation](https://docs.brew.sh/Adding-Software-to-Homebrew#test-locally)
