FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       build-essential \
       binutils-arm-linux-gnueabihf \
       debhelper \
       dpkg-dev \
       fakeroot \
       devscripts \
       file \
       ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
