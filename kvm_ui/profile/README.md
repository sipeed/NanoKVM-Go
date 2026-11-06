# Conan host profile

`builder.sh` cross-compiles `kvm_ui` for armv7hf through Conan 2 and
expects a machine-specific host profile at `profile/nanokvm-go`. The file
is intentionally git-ignored because compiler paths and versions are
machine-specific: never commit a local toolchain path.

## Notes

- Requires Conan 2 on `PATH` and an ARM hard-float GCC toolchain
  (`arm-none-linux-gnueabihf-gcc`/`g++`).
- The profile compiler version must match the cross toolchain the build
  image provides.
- `minibuild` assumes the profile and toolchain already exist.

See [README.md](../README.md) for the build commands.
