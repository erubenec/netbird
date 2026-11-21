# fork-ci

Default branch of this fork. It only holds the `fork build` workflow, which
builds images from `netbirdio/netbird` main plus every `feat/*` branch of this
fork and publishes them to ghcr.io. Nothing is written back to the repository.

- Feature work lives on `feat/*` branches.
- `main` is not pushed to this fork on purpose: upstream's own workflows
  would run on every push to it.
