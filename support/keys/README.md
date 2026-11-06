# Release update keys

Two Ed25519 public keys are pinned in every device key ring
(`/usr/share/nanokvm/update-keys.json`). Neither file is a secret.

| File | Role | Key id |
|---|---|---|
| `nanokvm-go-release.pub.pem` | Offline root key. Every shipped device pins it. Signs only key-ring changes, channel moves, and recovery releases. | `sha256:6bb54efbf99febb6f616292ae21d9a3ac906b9cd3ea57e1d599cbe1fefa0999c` |
| `nanokvm-go-ci.pub.pem` | CI key. Signs routine releases from CI (planned: a GitHub Actions `release` environment with manual approval). | `sha256:0fb49708d267aa1825610fcbee7250110b278f3109fe2ee0e62824e18635e154` |

The key id is the lowercase SHA-256 of the raw 32-byte public key.

They are checked in so that a build without access to a private key still
produces the correct key ring. Unless `--trusted-public-key` is passed,
`build.py` pins both keys, both for unsigned builds and for builds signed by
either official key, and derives `build/update-keys.json` from them;
`kvmcomm.install` ships that file to `/usr/share/nanokvm/update-keys.json`.
A build signed by any other key pins only that key. `--trusted-public-key`
replaces the defaults; a ring that keeps some official keys but drops others
is refused unless `--allow-partial-official-keys` is passed.

Without them a CI build emits an **empty** key ring, and installing that package
overwrites the device key ring with the empty one. The device then rejects
every official signed update as well, permanently, unless signature
verification stays disabled. That is why package_kvmcomm now refuses to build
against an empty key ring unless --allow-empty-keyring is passed explicitly.

Devices that predate the CI key pin only the root key. They poll the legacy
CDN prefix, whose manifests stay frozen on a root-signed release that adds the
CI key and moves the update channels to the `v2/` prefix; only `v2/` carries
CI-signed bundles.

If the CI key is compromised, delete the CI secret, then publish a release
signed by the root key whose key ring replaces `nanokvm-go-ci.pub.pem` with a
new CI public key (see the rotation section of `docs/build-packaging.md`).

To confirm a file matches its private key:

    openssl pkey -in <private-key>.pem -pubout | diff - nanokvm-go-release.pub.pem

The private keys never belong in this repository.
