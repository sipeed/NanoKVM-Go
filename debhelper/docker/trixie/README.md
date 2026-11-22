# Debian 13 (trixie) Docker packaging

This directory provides a Docker-based build environment for packaging debhelper packages on Debian 13 (trixie).
It is an auxiliary packaging path. The normal repository entry point remains the
root `build.py` driver.

## Build package in container

From project root:

1. Build package inputs first:
   - `python3 build.py build`

   Or build only the required inputs:
   - `python3 build.py kvm_vin`
   - `python3 build.py kvm_ui`
   - `python3 build.py nano_ocr`
   - `python3 build.py webcam`
   - `python3 build.py memfab`
   - `python3 build.py server`
   - `python3 build.py web`
   - `python3 build.py libkvm`
2. Build package in Docker through the root driver:
   - `python3 build.py package_kvmcomm --docker`
   - `python3 build.py package_nanokvm_go --docker`
   - `python3 build.py release --version <version> --docker`

The lower-level helper can still be used directly when debugging the Docker
packaging environment:

- `./debhelper/docker/trixie/build_in_docker.sh kvmcomm`
- `./debhelper/docker/trixie/build_in_docker.sh nanokvm-go`

The package name maps to:
- `debhelper/packages/<package>/debian`

When using `build.py --docker`, package artifacts are moved into the root
`build/` directory. When using the lower-level helper directly, package
artifacts are generated in:

- `debhelper/packages/*.deb`
- `debhelper/packages/*.changes`
- `debhelper/packages/*.buildinfo`

For non-Docker packaging, use:

```bash
python3 build.py package_kvmcomm
python3 build.py package_nanokvm_go
```

Those commands move Debian artifacts into the root `build/` directory.
