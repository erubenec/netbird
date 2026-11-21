# fork-ci

Default branch of this fork. It only holds the `fork build` workflow, which
builds images from `netbirdio/netbird` main plus the branches listed in
`features.txt` and publishes them to ghcr.io. Nothing is written back to the
repository.

- Feature work lives on its own branch; add its name to `features.txt` to ship it.
- The fork also carries every upstream branch, which is why features are
  listed instead of matched by name.
- `main` is not pushed to this fork on purpose: upstream's own workflows
  would run on every push to it.
