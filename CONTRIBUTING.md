# Contributing to goat-client

goat-client is the implementation; its design and decision records live in
[USAREUR-AF/goatcore](https://github.com/USAREUR-AF/goatcore) (see
`docs/design/goat-client.md` there). Link the relevant goatcore document by
URL when a change is driven by a design decision.

## First-time setup

```bash
git clone https://github.com/USAREUR-AF/goat-client.git
cd goat-client

# Sign commits (SSH or GPG; the policy is signed commits, not a key format)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true

# DCO sign-off on every commit
git config --global format.signoff true
```

Register the public key with GitHub as a *signing* key (Settings → SSH and GPG
keys → New SSH key → Key type: Signing Key).

## Branching

- One long-lived branch, `main`; direct push is blocked.
- Short-lived branches named by Conventional Commit type: `feat/<slug>`,
  `fix/<slug>`, `docs/<slug>`, `chore/<slug>`.
- No `develop` and no permanent release branches. Hotfixes branch from `main`
  and merge back to `main`.

## Commits

- **Signed**, and **DCO sign-off** (`git commit -s`). The `Signed-off-by:`
  trailer must match the commit author.
- **Conventional Commits** for PR titles, because the title becomes the
  squash-merge commit on `main`:

  ```text
  feat(tunnel): single-peer wg-cp0 interface manager
  fix(ipc): handle truncated bundle CBOR gracefully
  ```

  Allowed types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `perf`,
  `ci`, `build`, `revert`, `ui`.
- The commit body explains the *why* the diff does not already make obvious.

## Pull requests

The PR body covers what changed and why, the test plan, and any linked
goatcore change that must land with it.

Merge gates:

- Required CI checks green (vet, test, build, cross-compile, actionlint).
- Signed commits and DCO sign-off.
- At least one approving review from [`CODEOWNERS`](CODEOWNERS).
- Conversations resolved; branch up to date with `main`.

Merge style is squash-and-merge. Never bypass a gate: if a required check is
flaky, fix the test or remove it from the required list.

## Releases

Tags are cut from `main` on a green build, formatted
`goat-client-v<MAJOR>.<MINOR>.<PATCH>`. The release workflow builds the six
desktop targets and attaches the binaries, per-asset `.sha256`, an aggregate
`SHA256SUMS` and cosign signatures to the GitHub Release.

## Where to ask

- Implementation questions: comment on the PR, or open a draft PR with the
  question in the body.
- Design questions: open an issue or draft ADR in goatcore and link it.
- Security concerns: see [`SECURITY.md`](SECURITY.md); never a public issue.
