# Install

## Go

Install the latest published CLI:

```bash
go install github.com/johnmonarch/ediforge/cmd/edi-json@latest
```

Install a specific release tag:

```bash
go install github.com/johnmonarch/ediforge/cmd/edi-json@v0.1.0-alpha.3
```

Verify the installed version:

```bash
edi-json version
```

## GitHub Releases

Prebuilt archives and `checksums.txt` are published on the
[GitHub Releases](https://github.com/johnmonarch/ediforge/releases) page.
Download the archive for your operating system and architecture, then verify it
against `checksums.txt` before placing `edi-json` on your `PATH`.

## Docker

Official images publish to GitHub Container Registry:

```bash
docker pull ghcr.io/johnmonarch/ediforge:0.1.0-alpha.3
```

See [Docker](docker.md) for build and run examples.

## Homebrew

Tap and install:

```bash
brew tap johnmonarch/tap
brew install edi-json
```

The tap lives at [johnmonarch/homebrew-tap](https://github.com/johnmonarch/homebrew-tap).
GoReleaser formula generation is configured but currently uses `skip_upload:
true`; fully automated tap updates need a release workflow token with write
access to that tap.
