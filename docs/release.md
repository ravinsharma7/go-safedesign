# release process

## alpha branch and tag release process
Use this process when publishing an alpha snapshot branch such as `alpha-0.1.1` from `dev`.

Commit the release state on `dev`, then point both the branch and annotated tag at that commit:

```sh
git switch dev
git status --short
git add -A
git commit -m "add go tool deployment support"
release_commit=$(git rev-parse HEAD)
git tag -f -a alpha-0.1.1 -m "alpha-0.1.1" "$release_commit"
git branch -f alpha-0.1.1 "$release_commit"
```

Push with fully qualified refs because the branch and tag intentionally share the same name:

```sh
git push -u origin refs/heads/dev:refs/heads/dev
git push --force-with-lease origin refs/heads/alpha-0.1.1:refs/heads/alpha-0.1.1
git push --force origin refs/tags/alpha-0.1.1:refs/tags/alpha-0.1.1
```

Verify the branch and tag resolve to the same commit:

```sh
git rev-parse refs/heads/alpha-0.1.1
git rev-parse 'refs/tags/alpha-0.1.1^{}'
git ls-remote --symref origin HEAD refs/heads/alpha-0.1.1 refs/tags/alpha-0.1.1 'refs/tags/alpha-0.1.1^{}'
```

Set the GitHub repository default branch to the latest `alpha-*` branch in repository settings, or with an owner/admin token:

```sh
curl -fsS -X PATCH \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/ravinsharma7/go-safedesign \
  -d '{"default_branch":"alpha-0.1.1"}'
```

Keep the long-lived `dev` branch protected from accidental deletion. In GitHub repository settings, add a branch protection rule for `dev` and keep `Allow deletions` disabled.
