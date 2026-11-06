# NanoKVM-Go Support Resources

`support/` contains auxiliary scripts, configuration, and local generated
toolchain state used by the root `build.py` driver.

The formal build and packaging entry point is always the repository root:

```bash
python3 build.py build
python3 build.py release --version <version>
```

`support/scripts/` may be used for setup tasks such as installing or refreshing
the ARMv7 toolchain, but it is not the release packaging entry point.

`support/toolchains/` is local generated state and should not be committed.
