#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
PROJECT_ROOT=$(cd "${SCRIPT_DIR}/../../.." && pwd -P)
IMAGE_NAME="nanokvm-go-deb-packager:trixie"
CONTAINER_WORKDIR="/workspace"
PKG_NAME="${1:-kvmcomm}"
PKG_DIR="${PROJECT_ROOT}/debhelper/packages/${PKG_NAME}"

if ! command -v docker >/dev/null 2>&1; then
	echo "docker is required but not found in PATH"
	exit 1
fi

if [[ ! -d "${PKG_DIR}" ]]; then
	echo "package dir not found: ${PKG_DIR}"
	echo "expected layout: debhelper/packages/<package>/debian"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/build/kvm_vin" ]]; then
	echo "missing ${PROJECT_ROOT}/build/kvm_vin"
	echo "please run: python3 build.py kvm_vin"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/build/kvm_ui" ]]; then
	echo "missing ${PROJECT_ROOT}/build/kvm_ui"
	echo "please run: python3 build.py kvm_ui"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/build/nano_ocr_app" ]]; then
	echo "missing ${PROJECT_ROOT}/build/nano_ocr_app"
	echo "please run: python3 build.py nano_ocr"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/usr/bin/ax_efuse_tool" ]]; then
	echo "missing ${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/usr/bin/ax_efuse_tool"
	echo "please add ax_efuse_tool to kvmcomm rootfs"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ch_PP_OCRv3_det_npu.axmodel" ]]; then
	echo "missing ${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ch_PP_OCRv3_det_npu.axmodel"
	echo "please add OCR detection model to kvmcomm rootfs"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ch_PP_OCRv4_rec_npu.axmodel" ]]; then
	echo "missing ${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ch_PP_OCRv4_rec_npu.axmodel"
	echo "please add OCR recognition model to kvmcomm rootfs"
	exit 1
fi

if [[ "${PKG_NAME}" == "kvmcomm" ]] && [[ ! -f "${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ppocr_keys_v1.txt" ]]; then
	echo "missing ${PROJECT_ROOT}/debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ppocr_keys_v1.txt"
	echo "please add OCR dictionary to kvmcomm rootfs"
	exit 1
fi

if [[ "${PKG_NAME}" == "nanokvm-go" ]] && [[ ! -f "${PROJECT_ROOT}/build/memfab" ]]; then
	echo "missing ${PROJECT_ROOT}/build/memfab"
	echo "please run: python3 build.py memfab"
	exit 1
fi

if docker image inspect "${IMAGE_NAME}" >/dev/null 2>&1; then
	echo "reuse existing image: ${IMAGE_NAME}"
else
	echo "image not found, building: ${IMAGE_NAME}"
	docker build -t "${IMAGE_NAME}" "${SCRIPT_DIR}"
fi

docker run --rm \
	--user "$(id -u):$(id -g)" \
	-v "${PROJECT_ROOT}:${CONTAINER_WORKDIR}" \
	-w "${CONTAINER_WORKDIR}/debhelper/packages/${PKG_NAME}" \
	"${IMAGE_NAME}" \
	dpkg-buildpackage -d -us -uc -b -aarmhf
