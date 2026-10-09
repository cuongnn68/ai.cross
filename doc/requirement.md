# General
- DEAD SIMPLE
- CLI tool
- use https://github.com/charmbracelet/bubbletea
- use [ai-agent-instruction-files.md](./ai-agent-instruction-files.md) as baseline knowledge
- use a hardcoded dictionary for this info
  - or switch statements with logic to determine instruction locations
- store data of the tool at ~/.ai.cross
- easy to extend when adding more AI tools
- easy to update when AI tools change their config
- should work on all platforms: macOS/Windows/Linux

# Function
## status
### version
### recognize which agents are installed on the machine
- have a flag to specify global or local repo scope
- based on a hardcoded dictionary
- include list of coding agents
- and their instruction file locations (both local and global)
## histories
- include both user instructions before applying and all files after applying
- history folder and file names should use the backup date and time
- eg: 20261003130759 means 2026-10-03 at 13:07:59
- 20261003113058.md is a history version after applying
- 20261003113058 is a history version before applying
- preserve the original directory structure of all old instruction files

### global
- inside folder ~/.ai.cross/instructions/global format:
```
20261003113058
20261003113058.md
```
- eg: `~/.claude/CLAUDE.md` should be stored at `~/.ai.cross/instructions/global/20261003113058/.claude/CLAUDE.md`

### local
- inside folder ~/.ai.cross/instructions/project_location format:
```
20261003113058
20261003113058.md
```
- preserve the project's full directory path under `~/.ai.cross/instructions/project_location`
- eg: project `/usr/code/mock-project` has history records under `~/.ai.cross/instructions/project_location/usr/code/mock-project`
- preserve each file's project-relative path inside the timestamped backup folder
- eg: `/usr/code/mock-project/.team/VIBE.md` should be backed up at `~/.ai.cross/instructions/project_location/usr/code/mock-project/20261003113058/.team/VIBE.md`

## config
- ~/.ai.cross/config.yml
- or ~/.ai.cross/config.yaml
format:
```yaml
histories:
  global:
    additional:
      - "~/VIBE.md"
    ignore:
      - "~/CLAUDE.md"
  local:
    additional:
      - "./.team/VIBE.md"
      - "./.user/VIBE.md"
    ignore:
      - "./CLAUDE.md"
```

## apply instruction file
- require a flag to specify global or local repo scope
- apply across all AI agents
- include all AI agents in the hardcoded dictionary
  - include additional paths specified in config
  - exclude ignored paths specified in config
- automatically back up current files before applying unless a flag disables backups (`--no-backup` or similar)
- have a folder of apply history in ~/.ai.cross (check ## histories)
- add a history record to that folder each time apply succeeds
- apply should be atomic
- apply the instruction (overwrite files)
- have a flag to determine the source of new instructions: input, clipboard, file, HTTP(S) URL
- default to input when no source flag is specified
## restore history
### list
- list all
- flag to list only tool applies
- flag to list only pre-applies backups
- show the backup file/folder path next to its name
- show the backup date and time in a human-readable format
### restore
- have a flag to specify global or local repo scope
- have a flag to specify the history record by its name
## help
- show instructions on how to use this tool

# Deployment

- Package names, publisher, and repo URLs below are placeholders.
- Release is complete when the published version can be discovered and installed from each package manager on a clean supported machine.

## brew

- Setup: Create a public GitHub tap repo, eg `<owner>/homebrew-tap`
- Package: Add `Formula/ai-cross.rb`
  - Include release URL, SHA-256 checksum, license, dependencies, and build / install steps
  - Support the intended macOS / Linux architectures
- Publish: Push the formula to the tap after the release download is available
  - A custom tap is available directly; no Homebrew core approval required
  - Default catalog distribution requires a separate `homebrew/core` submission and acceptance
- Verify: Run on a clean supported machine
  - `brew tap <owner>/tap`
  - `brew info <owner>/tap/ai-cross`
  - `brew install <owner>/tap/ai-cross`
  - Confirm the installed CLI reports the expected release version
- Next release: Publish the release first, then update the formula URL and checksum in the tap
- References: [Create and maintain a tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap), [Formula cookbook](https://docs.brew.sh/Formula-Cookbook), [Core submission](https://docs.brew.sh/Adding-Software-to-Homebrew)

## winget

- Setup: Choose a stable package ID, eg `<Publisher>.AiCross`
- Package: Publish Windows release files at public, versioned URLs
  - Create YAML manifests with package ID, version, installer type, architecture, URLs, and SHA-256 checksums
  - Validate locally: `winget validate <manifest-dir>`
- Publish: Open a PR in `microsoft/winget-pkgs` with the version manifests
  - Resolve automated validation failures and review feedback
  - Wait for acceptance and publication to the WinGet source; a GitHub release alone does not list the package
- Verify: Run on a clean Windows machine after publication
  - `winget source update`
  - `winget show --id <Publisher>.AiCross --exact --source winget`
  - `winget install --id <Publisher>.AiCross --exact --source winget`
  - Confirm the source and installed CLI report the expected release version
- Next release: Publish new release files, then submit new version manifests and repeat verification
- References: [Create manifests](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest), [Submit manifests](https://learn.microsoft.com/en-us/windows/package-manager/package/repository), [Commands](https://learn.microsoft.com/en-us/windows/package-manager/winget/)

## apt / apt-get

- Setup: Create an HTTPS APT repo and a repo signing key
  - Both commands use the same repo; one publishing process supports both
  - Users must add this repo; it is not automatically included in Debian / Ubuntu default sources
- Package: Build `.deb` files for each supported distro / architecture
  - Include package name, version, architecture, dependencies, and the CLI binary
- Publish: Upload packages and generate APT repo metadata
  - Generate `Packages` indexes and a `Release` file containing index checksums
  - Sign release metadata as `InRelease`, or `Release` plus `Release.gpg`
  - Publish packages before indexes and signed metadata so downloads exist when discovered
- Setup guide: Publish the repo URL, public signing key, and exact repo setup commands
  - Save the public key under `/etc/apt/keyrings/`
  - Add a `.sources` file under `/etc/apt/sources.list.d/` with repo URI, suite, components, and `Signed-By`
  - Keep the private signing key in release secrets
- Verify: Add the repo on a clean supported Debian / Ubuntu machine
  - `sudo apt-get update` must succeed without signature errors
  - `apt-cache policy ai-cross` must show the expected version and repo
  - `sudo apt-get install ai-cross`
  - Confirm the installed CLI reports the expected release version
- Next release: Upload new `.deb` files, regenerate indexes, sign and publish metadata, then repeat verification
- References: [Repo format](https://wiki.debian.org/DebianRepository/Format), [Third-party repo setup](https://wiki.debian.org/DebianRepository/UseThirdParty), [Debian package fields](https://www.debian.org/doc/debian-policy/ch-controlfields.html), [APT commands](https://www.debian.org/doc/manuals/debian-handbook/apt.en.html)
