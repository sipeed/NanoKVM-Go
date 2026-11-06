# syntax=docker/dockerfile:1.7

ARG DEBIAN_VERSION=12.11-slim
ARG GO_VERSION=1.26.5
ARG NODE_VERSION=22.17.1

FROM golang:${GO_VERSION}-bookworm AS go

FROM node:${NODE_VERSION}-bookworm-slim AS node

FROM debian:${DEBIAN_VERSION}

ARG GO_VERSION
ARG NODE_VERSION
ARG PNPM_VERSION=11.3.0
ARG CONAN_VERSION=2.30.0
ARG CMAKE_VERSION=3.31.6
ARG ARM_TOOLCHAIN_URL=https://developer.arm.com/-/media/Files/downloads/gnu/12.2.rel1/binrel/arm-gnu-toolchain-12.2.rel1-x86_64-arm-none-linux-gnueabihf.tar.xz
ARG ARM_TOOLCHAIN_SHA256=d73f230bb946231b648a960b719f2cc1afc792ec2e36f9abc25552f00923a926
ARG LIBOPUS_RUNTIME_URL=https://ports.ubuntu.com/ubuntu-ports/pool/main/o/opus/libopus0_1.3.1-0.1build2_armhf.deb
ARG LIBOPUS_RUNTIME_SHA256=316e426d14511b36d3a5dda02d73621b0028005fd3a7c464286ada7bca43db91
ARG LIBOPUS_DEV_URL=https://ports.ubuntu.com/ubuntu-ports/pool/main/o/opus/libopus-dev_1.3.1-0.1build2_armhf.deb
ARG LIBOPUS_DEV_SHA256=9fba0a8f0010bc10582b3872b8cb0c5ca702a373b5f40a50faf393629dca9117
# kvm_ui talks to systemd over sd-bus on the device, so the cross sysroot needs
# armhf libsystemd. 252.x ships libsystemd.so.0.35.0, which is the ABI the device
# image and the existing developer sysroots already build against.
#
# libsystemd.so also records libcap/libgcrypt/libgpg-error/liblz4/liblzma/libzstd
# as DT_NEEDED, so the linker has to resolve them even though kvm_ui calls none of
# them and the produced binary only needs libsystemd.so.0. Any version satisfies
# the link and none of them reach the device, so only libsystemd itself is pinned.
ARG LIBSYSTEMD_VERSION=252.39-1~deb12u2

LABEL org.opencontainers.image.title="NanoKVM-Go ARMv7 CI build image" \
      org.opencontainers.image.description="Pinned Linux amd64 environment for ARMv7 builds and Debian packaging"

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ENV DEBIAN_FRONTEND=noninteractive \
    HOME=/opt/ci \
    CONAN_HOME=/opt/ci/.conan2 \
    GOPATH=/opt/ci/go \
    GOCACHE=/opt/ci/go-build \
    GOMODCACHE=/opt/ci/go/pkg/mod \
    PNPM_HOME=/opt/ci/pnpm \
    PATH=/opt/python-tools/bin:/opt/ci/pnpm:/usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin

COPY --from=go /usr/local/go /usr/local/go
COPY --from=node /usr/local/ /usr/local/

RUN set -eux; \
    test "$(dpkg --print-architecture)" = "amd64"; \
    apt-get update; \
    apt-get install --yes --no-install-recommends \
        autoconf \
        automake \
        bash \
        binutils \
        binutils-arm-linux-gnueabihf \
        bison \
        build-essential \
        ca-certificates \
        curl \
        debhelper \
        devscripts \
        dpkg-dev \
        fakeroot \
        file \
        flex \
        gawk \
        gettext \
        git \
        libffi-dev \
        libssl-dev \
        libtool \
        libtool-bin \
        m4 \
        ninja-build \
        patch \
        patchelf \
        perl \
        pkg-config \
        python3 \
        python3-pip \
        python3-venv \
        rsync \
        unzip \
        xz-utils \
        zip \
        zlib1g-dev \
        zstd; \
    rm -rf /var/lib/apt/lists/*; \
    mkdir -p \
        "${CONAN_HOME}" \
        "${GOCACHE}" \
        "${GOMODCACHE}" \
        "${PNPM_HOME}"; \
    python3 -m venv /opt/python-tools; \
    /opt/python-tools/bin/pip install --no-cache-dir \
        "cmake==${CMAKE_VERSION}" \
        "conan==${CONAN_VERSION}"; \
    npm install --global "pnpm@${PNPM_VERSION}"; \
    test "$(go env GOVERSION)" = "go${GO_VERSION}"; \
    test "$(node --version)" = "v${NODE_VERSION}"; \
    test "$(pnpm --version)" = "${PNPM_VERSION}"; \
    test "$(conan --version)" = "Conan version ${CONAN_VERSION}"; \
    test "$(cmake --version | head -n 1)" = "cmake version ${CMAKE_VERSION}"; \
    python3 --version

RUN set -eux; \
    work_dir="$(mktemp -d)"; \
    trap 'rm -rf "${work_dir}"' EXIT; \
    curl --fail --location --retry 3 \
        --output "${work_dir}/arm-toolchain.tar.xz" \
        "${ARM_TOOLCHAIN_URL}"; \
    echo "${ARM_TOOLCHAIN_SHA256}  ${work_dir}/arm-toolchain.tar.xz" | sha256sum --check --strict; \
    mkdir -p /opt/armv7-toolchain; \
    tar --extract --xz --file "${work_dir}/arm-toolchain.tar.xz" \
        --directory /opt/armv7-toolchain --strip-components=1; \
    /opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-gcc --version; \
    test "$(/opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-gcc -dumpmachine)" = \
        "arm-none-linux-gnueabihf"; \
    sysroot=/opt/armv7-toolchain/arm-none-linux-gnueabihf/libc; \
    test -d "${sysroot}"; \
    curl --fail --location --retry 3 \
        --output "${work_dir}/libopus0.deb" \
        "${LIBOPUS_RUNTIME_URL}"; \
    curl --fail --location --retry 3 \
        --output "${work_dir}/libopus-dev.deb" \
        "${LIBOPUS_DEV_URL}"; \
    echo "${LIBOPUS_RUNTIME_SHA256}  ${work_dir}/libopus0.deb" | sha256sum --check --strict; \
    echo "${LIBOPUS_DEV_SHA256}  ${work_dir}/libopus-dev.deb" | sha256sum --check --strict; \
    dpkg-deb --extract "${work_dir}/libopus0.deb" "${work_dir}/libopus0"; \
    dpkg-deb --extract "${work_dir}/libopus-dev.deb" "${work_dir}/libopus-dev"; \
    mkdir -p "${sysroot}/usr/include" "${sysroot}/usr/lib"; \
    cp -a "${work_dir}/libopus-dev/usr/include/." "${sysroot}/usr/include/"; \
    cp -a "${work_dir}/libopus0/usr/lib/arm-linux-gnueabihf/"libopus.so* "${sysroot}/usr/lib/"; \
    cp -a "${work_dir}/libopus-dev/usr/lib/arm-linux-gnueabihf/libopus.so" "${sysroot}/usr/lib/"; \
    cp -a "${work_dir}/libopus-dev/usr/lib/arm-linux-gnueabihf/libopus.a" "${sysroot}/usr/lib/"; \
    cp -a "${work_dir}/libopus-dev/usr/lib/arm-linux-gnueabihf/pkgconfig" "${sysroot}/usr/lib/"; \
    test -f "${sysroot}/usr/include/opus/opus.h"; \
    test -e "${sysroot}/usr/lib/libopus.so"; \
    opus_so="$(readlink -f "${sysroot}/usr/lib/libopus.so")"; \
    /opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-readelf -h "${opus_so}" \
        | grep --quiet 'Machine:.*ARM'; \
    /opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-readelf -A "${opus_so}" \
        | grep --quiet 'Tag_ABI_VFP_args: VFP registers'; \
    dpkg --add-architecture armhf; \
    apt-get update; \
    systemd_dir="${work_dir}/systemd"; \
    mkdir -p "${systemd_dir}"; \
    chmod 0755 "${work_dir}" "${systemd_dir}"; \
    (cd "${systemd_dir}"; \
        apt-get download \
            "libsystemd0:armhf=${LIBSYSTEMD_VERSION}" \
            "libsystemd-dev:armhf=${LIBSYSTEMD_VERSION}" \
            libcap2:armhf libgcrypt20:armhf libgpg-error0:armhf \
            liblz4-1:armhf liblzma5:armhf libzstd1:armhf); \
    rm -rf /var/lib/apt/lists/*; \
    for systemd_deb in "${systemd_dir}"/*.deb; do \
        dpkg-deb --extract "${systemd_deb}" "${systemd_dir}/root"; \
    done; \
    cp -a "${systemd_dir}/root/usr/include/." "${sysroot}/usr/include/"; \
    for systemd_lib in "${systemd_dir}/root/usr/lib/arm-linux-gnueabihf/"lib*.so* \
                       "${systemd_dir}/root/lib/arm-linux-gnueabihf/"lib*.so*; do \
        test -e "${systemd_lib}" || continue; \
        cp -a "${systemd_lib}" "${sysroot}/usr/lib/"; \
    done; \
    cp -a "${systemd_dir}/root/usr/lib/arm-linux-gnueabihf/pkgconfig/libsystemd.pc" \
        "${sysroot}/usr/lib/pkgconfig/"; \
    for systemd_soname in libsystemd.so libcap.so.2 libgcrypt.so.20 \
                          libgpg-error.so.0 liblz4.so.1 liblzma.so.5 libzstd.so.1; do \
        test -e "${sysroot}/usr/lib/${systemd_soname}"; \
    done; \
    test -f "${sysroot}/usr/include/systemd/sd-bus.h"; \
    test -e "${sysroot}/usr/lib/libsystemd.so"; \
    systemd_so="$(readlink -f "${sysroot}/usr/lib/libsystemd.so")"; \
    /opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-readelf -h "${systemd_so}" \
        | grep --quiet 'Machine:.*ARM'; \
    /opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-readelf -A "${systemd_so}" \
        | grep --quiet 'Tag_ABI_VFP_args: VFP registers'

# Prime the immutable image layer with the exact cross-build dependencies used
# by the component recipes. Source/build/download leftovers are removed while
# the resulting Conan packages remain available to every CI container.
COPY support/scripts/getconfig.py \
     support/scripts/nanokvm-go.ini \
     support/scripts/toolchain_setup_components.sh \
     /tmp/nanokvm-go/support/scripts/
COPY libkvm/conanfile.py /tmp/nanokvm-go/libkvm/conanfile.py
COPY kvm_vin/conanfile.py /tmp/nanokvm-go/kvm_vin/conanfile.py
COPY kvm_ui/conanfile.py /tmp/nanokvm-go/kvm_ui/conanfile.py
COPY nano_ocr/conanfile.py /tmp/nanokvm-go/nano_ocr/conanfile.py

RUN set -eux; \
    mkdir -p /tmp/nanokvm-go/support/toolchains; \
    ln -s /opt/armv7-toolchain \
        /tmp/nanokvm-go/support/toolchains/armv7-toolchain; \
    chmod +x \
        /tmp/nanokvm-go/support/scripts/getconfig.py \
        /tmp/nanokvm-go/support/scripts/toolchain_setup_components.sh; \
    /tmp/nanokvm-go/support/scripts/toolchain_setup_components.sh \
        --config /tmp/nanokvm-go/support/scripts/nanokvm-go.ini \
        --switch; \
    conan profile detect --force; \
    profile=/tmp/nanokvm-go/support/scripts/nanokvm-go; \
    conan install /tmp/nanokvm-go/libkvm \
        --build=missing --profile:build=default --profile:host="${profile}"; \
    conan install /tmp/nanokvm-go/kvm_vin \
        --build=missing --profile:build=default --profile:host="${profile}"; \
    conan install /tmp/nanokvm-go/kvm_ui \
        --build=missing --profile:build=default --profile:host="${profile}"; \
    conan install /tmp/nanokvm-go/nano_ocr \
        --build=missing --profile:build=default --profile:host="${profile}" \
        --settings:host compiler.libcxx=libstdc++11 \
        --options:host 'libcurl/*:shared=True'; \
    conan cache clean '*' --source --build --download; \
    conan install /tmp/nanokvm-go/libkvm \
        --build=never --no-remote \
        --profile:build=default --profile:host="${profile}"; \
    conan install /tmp/nanokvm-go/kvm_vin \
        --build=never --no-remote \
        --profile:build=default --profile:host="${profile}"; \
    conan install /tmp/nanokvm-go/kvm_ui \
        --build=never --no-remote \
        --profile:build=default --profile:host="${profile}"; \
    conan install /tmp/nanokvm-go/nano_ocr \
        --build=never --no-remote \
        --profile:build=default --profile:host="${profile}" \
        --settings:host compiler.libcxx=libstdc++11 \
        --options:host 'libcurl/*:shared=True'; \
    rm -rf /tmp/nanokvm-go; \
    test -f "${CONAN_HOME}/profiles/default"

WORKDIR /workspace

CMD ["bash"]
