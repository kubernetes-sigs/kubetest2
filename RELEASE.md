# Release Process

This project uses [GoReleaser] to manage releases. This allows for automated changelog creation (in the release notes)
and uploading of platform specific archives.

This is handled via a GitHub action (_see .github/workflows/release.yaml_). In order to work it requires a repo owner to
create a GitHub Token with the `repo` scope and store that as a repository secret called `GH_RELEASE_TOKEN`.

## Cutting a New Release

Make sure you're on an up to date master, and run the following:

* Create a new tag `git tag -s v<VERSION>`
* Push to GitHub `git push origin v<VERSION>`
* [GoReleaser] will take care of creating the release, adding the changelog, and attaching the binaries.
* Send an announcement email to `kubernetes-dev@googlegroups.com` with the subject `[ANNOUNCE] kubetest2 v<VERSION> is
	released`.

## Local Snapshots

Sometimes you want to verify the builds before cutting a release. This can be done locally by running:

    make snapshot

This will build a snapshot release (`v<VERSION+PATCH>-next`) locally and put all of the files in _./dist_. This is used
for local testing and therefore the dist directory is git ignored.

[GoReleaser]: https://goreleaser.com/
