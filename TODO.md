# TODO

Open tasks of the whole project. Tasks of a single part are in its own file:

- [Server](internal/server/TODO.md)
- [Camera](internal/cam/TODO.md)
- [Camera access](internal/cam/camera/TODO.md)

## Whole project

- **Nix flake** (`flake.nix`): builds only once nixpkgs has Go 1.27.4 (`go.mod`; nixos-unstable has 1.27.3, 2026-11-22), then `nix flake update` and commit the lock, and add a job to `ci.yml` that runs `nix build .#darts-o-cam .#darts-o-mat`, so an outdated `vendorHash` (after a change of `go.sum`) shows at once instead of breaking the flake unnoticed; not before, since it would fail until then (2026-11-22). Decided against building Go 1.27.4 in the flake meanwhile (an override plus minutes of build time for everyone).
- Rate limits beyond [failed logins](internal/server/AGENTS.md#failed-logins): registering, creating games and tournaments, friend requests, the API in general.
