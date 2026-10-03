# WinGet deployment

- Checked: **2026-10-03** against the official refs below.
- Package IDs and paths below are placeholders.

## How it works

- `microsoft/winget-pkgs` stores YAML manifests describing apps and their download URLs; the publisher hosts release files
- Flow: Submit PR → automated validation / security scans → review / acceptance → publication to the WinGet catalog
- GitHub merge and catalog publication are separate steps; verify availability through the WinGet client
- Published packages can be discovered and installed through the default `winget` source
- References: [Repo overview](https://github.com/microsoft/winget-pkgs), [Submission process](https://learn.microsoft.com/en-us/windows/package-manager/package/repository)

## Package requirements

- Downloads: Use the publisher's official release location; prefer stable, version-specific URLs
  - Replacing files at an existing URL can cause SHA-256 mismatches
- Metadata: Include accurate package ID, version, publisher, license, architecture, installer type, URL, and SHA-256
- Install: Must work unattended; installation / uninstallation must behave correctly
- Installer: Use a supported installer type; `.bat` / `.ps1` scripts cannot serve as installers
- References: [Manifest schema](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest), [Submission expectations](https://learn.microsoft.com/en-us/windows/package-manager/package/repository#submission-expectations), [Installer and URL rules](https://github.com/microsoft/winget-pkgs/blob/master/doc/Policies.md)

## PR rules

- Check for an existing package and open PR for the same version before submitting
- Scope: One package version per PR; manifest files only
- Format: Multi-file manifests required; singleton manifests are not accepted
  - Minimum files: version, default locale, installer
  - Include schema headers and use the latest schema supported by the repo
- Contribution: Follow the CLA bot's instructions and Microsoft's code of conduct
- Review: Respond to validation failures and reviewer requests; inactive PRs may be closed
- References: [Contributor checklist](https://github.com/microsoft/winget-pkgs/blob/master/doc/FirstContribution.md), [Authoring guide](https://github.com/microsoft/winget-pkgs/blob/master/doc/Authoring.md), [Contribution terms](https://github.com/microsoft/winget-pkgs#contributing), [Contributor guide](https://github.com/microsoft/winget-pkgs/blob/master/CONTRIBUTING.md)

## Acceptance rules

- App must be testable and accurately described
- Follow Microsoft's security, privacy, licensing, and content policies
- Malware / potentially unwanted application (PUA) scan flags block acceptance until resolved
- Passing local validation does not guarantee approval; Microsoft can reject submissions
- References: [Microsoft policies](https://learn.microsoft.com/en-us/windows/package-manager/package/windows-package-manager-policies), [Security rules](https://github.com/microsoft/winget-pkgs/blob/master/doc/Policies.md#security-scans-and-potentially-unwanted-applications-pua), [Submission process](https://learn.microsoft.com/en-us/windows/package-manager/package/repository)

## Deployment steps

- Publish: Make the Windows release files available at official, version-specific URLs
- Generate: Run `wingetcreate new <installer-url>` and review the generated manifests
- Validate: Run `winget validate --manifest <manifest-dir>`
- Test: Verify locally before submitting, preferably in Windows Sandbox
  - Enable local manifests: `winget settings --enable LocalManifestFiles`
  - Install: `winget install --manifest <manifest-dir>`
  - Confirm unattended installation, correct app / publisher / version metadata, and successful uninstallation
- Submit: Open a manifest PR in `microsoft/winget-pkgs`
- Review: Resolve checks and feedback; wait for acceptance and catalog publication
- Verify availability: Run on Windows after publication
  - Refresh sources: `winget source update`
  - Check listing: `winget show --id <Publisher>.AiCross --exact --source winget`
  - Install: `winget install --id <Publisher>.AiCross --exact --source winget`
  - Confirm the listed and installed versions match the release
- Update: Submit new version manifests for each release and repeat verification
- References: [Authoring and testing](https://github.com/microsoft/winget-pkgs/blob/master/doc/Authoring.md), [Submission process](https://learn.microsoft.com/en-us/windows/package-manager/package/repository), [WinGet commands](https://learn.microsoft.com/en-us/windows/package-manager/winget/)
