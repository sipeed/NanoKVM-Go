#include "ax_utils.hpp"

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <fstream>
#include <iterator>
#include <vector>

#include <ax_buffer_tool.h>
#include <ax_sys_api.h>
#include <ax_vdec_api.h>

namespace {

constexpr int VDEC_OK = 0;
constexpr int VDEC_ERR_NOT_INIT = -20;
constexpr int VDEC_ERR_BAD_JPEG = -21;
constexpr int VDEC_ERR_BAD_SIZE = -22;
constexpr int VDEC_ERR_OUTPUT_BUFFER = -23;
constexpr int VDEC_ERR_ALLOC = -24;
constexpr int VDEC_ERR_DECODE = -25;
constexpr int VDEC_ERR_FORMAT = -26;
constexpr int VDEC_ERR_COPY = -27;
constexpr int VDEC_ERR_READ_FILE = -28;

constexpr int JDEC_MIN_WIDTH = 48;
constexpr int JDEC_MIN_HEIGHT = 48;
constexpr int JDEC_MAX_WIDTH = 16384;
constexpr int JDEC_MAX_HEIGHT = 16384;

uint16_t read_be16(const uint8_t* data)
{
    return static_cast<uint16_t>((static_cast<uint16_t>(data[0]) << 8) | data[1]);
}

bool is_sof_marker(uint8_t marker)
{
    switch (marker) {
        case 0xC0:
        case 0xC1:
        case 0xC2:
        case 0xC3:
        case 0xC5:
        case 0xC6:
        case 0xC7:
        case 0xC9:
        case 0xCA:
        case 0xCB:
        case 0xCD:
        case 0xCE:
        case 0xCF:
            return true;
        default:
            return false;
    }
}

bool is_standalone_marker(uint8_t marker)
{
    return marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7);
}

bool parse_jpeg_dimensions(const uint8_t* data, size_t size, int& width, int& height)
{
    width = 0;
    height = 0;
    if (data == nullptr || size < 4 || data[0] != 0xFF || data[1] != 0xD8) {
        return false;
    }

    size_t pos = 2;
    while (pos + 1 < size) {
        if (data[pos] != 0xFF) {
            ++pos;
            continue;
        }

        while (pos < size && data[pos] == 0xFF) {
            ++pos;
        }
        if (pos >= size) {
            return false;
        }

        const uint8_t marker = data[pos++];
        if (marker == 0xD9 || marker == 0xDA) {
            return false;
        }
        if (is_standalone_marker(marker)) {
            continue;
        }
        if (pos + 2 > size) {
            return false;
        }

        const uint16_t segment_len = read_be16(data + pos);
        if (segment_len < 2 || pos + segment_len > size) {
            return false;
        }

        if (is_sof_marker(marker)) {
            if (segment_len < 7) {
                return false;
            }
            height = read_be16(data + pos + 3);
            width = read_be16(data + pos + 5);
            return width > 0 && height > 0;
        }

        pos += segment_len;
    }

    return false;
}

bool validate_jpeg_dimensions(int width, int height)
{
    if (width <= 0 || height <= 0) {
        return false;
    }
    if ((width % 2) != 0 || (height % 2) != 0) {
        return false;
    }
    if (width < JDEC_MIN_WIDTH || height < JDEC_MIN_HEIGHT) {
        return false;
    }
    if (width > JDEC_MAX_WIDTH || height > JDEC_MAX_HEIGHT) {
        return false;
    }
    return true;
}

AX_U32 align_up(AX_U32 value, AX_U32 align)
{
    return (value + align - 1) & ~(align - 1);
}

size_t nv12_size(int y_stride, int uv_stride, int height)
{
    return static_cast<size_t>(y_stride) * static_cast<size_t>(height) +
           static_cast<size_t>(uv_stride) * static_cast<size_t>(height / 2);
}

bool copy_plane(uint8_t* dst,
                size_t dst_size,
                size_t dst_offset,
                AX_U64 src_addr,
                AX_U32 stride,
                AX_U32 rows)
{
    const size_t bytes = static_cast<size_t>(stride) * static_cast<size_t>(rows);
    if (bytes == 0) {
        return true;
    }
    if (src_addr == 0 || dst == nullptr || dst_offset > dst_size || bytes > dst_size - dst_offset) {
        return false;
    }

    const auto* src = reinterpret_cast<const uint8_t*>(static_cast<uintptr_t>(src_addr));
    std::memcpy(dst + dst_offset, src, bytes);
    return true;
}

}  // namespace

bool AxeraUtils::vdec_init(void)
{
    AX_VDEC_MOD_ATTR_T mod_attr {};
    mod_attr.u32MaxGroupCount = 16;

    const AX_S32 ret = AX_VDEC_Init(&mod_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VDEC_Init failed, ret=0x{:X}", static_cast<AX_U32>(ret));
        return false;
    }

    return true;
}

bool AxeraUtils::vdec_deinit(void)
{
    const AX_S32 ret = AX_VDEC_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VDEC_Deinit failed, ret=0x{:X}", static_cast<AX_U32>(ret));
        return false;
    }

    return true;
}

int AxeraUtils::vdec_get_jpeg_info(const uint8_t* data,
                                   size_t size,
                                   int& width,
                                   int& height,
                                   int& y_stride,
                                   int& uv_stride,
                                   size_t& required_size)
{
    width = 0;
    height = 0;
    y_stride = 0;
    uv_stride = 0;
    required_size = 0;

    if (!parse_jpeg_dimensions(data, size, width, height)) {
        (*this->_logger)->error("[vdec] failed to parse JPEG dimensions");
        return VDEC_ERR_BAD_JPEG;
    }
    if (!validate_jpeg_dimensions(width, height)) {
        (*this->_logger)->error("[vdec] invalid JPEG dimensions: {}x{}", width, height);
        return VDEC_ERR_BAD_SIZE;
    }

    y_stride = static_cast<int>(align_up(static_cast<AX_U32>(width), 16));
    uv_stride = y_stride;
    required_size = nv12_size(y_stride, uv_stride, height);
    return VDEC_OK;
}

int AxeraUtils::vdec_decode_jpeg(const uint8_t* data,
                                 size_t size,
                                 uint8_t* output,
                                 size_t output_size,
                                 int& width,
                                 int& height,
                                 int& y_stride,
                                 int& uv_stride,
                                 size_t& written_size)
{
    width = 0;
    height = 0;
    y_stride = 0;
    uv_stride = 0;
    written_size = 0;

    size_t required_size = 0;
    int ret_code = vdec_get_jpeg_info(data, size, width, height, y_stride, uv_stride, required_size);
    if (ret_code != VDEC_OK) {
        return ret_code;
    }
    if (output == nullptr || output_size < required_size) {
        (*this->_logger)->error("[vdec] output buffer too small: have={} need={}",
                                output_size, required_size);
        return VDEC_ERR_OUTPUT_BUFFER;
    }

    AX_U64 jpg_phy = 0;
    AX_U64 out_phy = 0;
    AX_VOID* jpg_vir = nullptr;
    AX_VOID* out_vir = nullptr;

    AX_S32 ret = AX_SYS_MemAlloc(&jpg_phy,
                                 &jpg_vir,
                                 static_cast<AX_U32>(size),
                                 0x100,
                                 reinterpret_cast<const AX_S8*>("kvm_vdec_jpeg"));
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[vdec] AX_SYS_MemAlloc(jpeg) failed, ret=0x{:X}",
                                static_cast<AX_U32>(ret));
        return VDEC_ERR_ALLOC;
    }
    std::memcpy(jpg_vir, data, size);

    const AX_U32 aligned_width = align_up(static_cast<AX_U32>(width), 16);
    const AX_U32 aligned_height = align_up(static_cast<AX_U32>(height), 16);
    const AX_U32 y_plane_size = aligned_width * aligned_height;
    const AX_U32 out_size = AX_JDEC_GetYuvBufferSize(
        aligned_width, aligned_height, AX_FORMAT_YUV420_SEMIPLANAR);

    ret = AX_SYS_MemAlloc(&out_phy,
                          &out_vir,
                          out_size,
                          0x1000,
                          reinterpret_cast<const AX_S8*>("kvm_vdec_nv12"));
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[vdec] AX_SYS_MemAlloc(nv12) failed, ret=0x{:X}",
                                static_cast<AX_U32>(ret));
        AX_SYS_MemFree(jpg_phy, jpg_vir);
        return VDEC_ERR_ALLOC;
    }

    AX_VDEC_DEC_ONE_FRM_T param {};
    param.stStream.pu8Addr = static_cast<AX_U8*>(jpg_vir);
    param.stStream.u64PhyAddr = jpg_phy;
    param.stStream.u32StreamPackLen = static_cast<AX_U32>(size);

    param.stFrame.u64PhyAddr[0] = out_phy;
    param.stFrame.u64VirAddr[0] = reinterpret_cast<AX_U64>(out_vir);
    param.stFrame.u64PhyAddr[1] = out_phy + y_plane_size;
    param.stFrame.u64VirAddr[1] = reinterpret_cast<AX_U64>(out_vir) + y_plane_size;

    ret = AX_VDEC_JpegDecodeOneFrame(&param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[vdec] AX_VDEC_JpegDecodeOneFrame failed, ret=0x{:X}",
                                static_cast<AX_U32>(ret));
        AX_SYS_MemFree(out_phy, out_vir);
        AX_SYS_MemFree(jpg_phy, jpg_vir);
        return VDEC_ERR_DECODE;
    }

    if (param.stFrame.enImgFormat != AX_FORMAT_YUV420_SEMIPLANAR) {
        (*this->_logger)->error("[vdec] unsupported output format: {}",
                                static_cast<int>(param.stFrame.enImgFormat));
        AX_SYS_MemFree(out_phy, out_vir);
        AX_SYS_MemFree(jpg_phy, jpg_vir);
        return VDEC_ERR_FORMAT;
    }

    width = static_cast<int>(param.stFrame.u32Width);
    height = static_cast<int>(param.stFrame.u32Height);
    y_stride = static_cast<int>(param.stFrame.u32PicStride[0] > 0
                                    ? param.stFrame.u32PicStride[0]
                                    : aligned_width);
    uv_stride = static_cast<int>(param.stFrame.u32PicStride[1] > 0
                                     ? param.stFrame.u32PicStride[1]
                                     : y_stride);
    written_size = nv12_size(y_stride, uv_stride, height);

    if (output_size < written_size) {
        (*this->_logger)->error("[vdec] output buffer too small after decode: have={} need={}",
                                output_size, written_size);
        AX_SYS_MemFree(out_phy, out_vir);
        AX_SYS_MemFree(jpg_phy, jpg_vir);
        return VDEC_ERR_OUTPUT_BUFFER;
    }
    if (y_stride < width || uv_stride < width) {
        (*this->_logger)->error("[vdec] invalid decoded stride: y={} uv={} width={}",
                                y_stride, uv_stride, width);
        AX_SYS_MemFree(out_phy, out_vir);
        AX_SYS_MemFree(jpg_phy, jpg_vir);
        return VDEC_ERR_COPY;
    }

    const size_t uv_offset = static_cast<size_t>(y_stride) * static_cast<size_t>(height);
    const bool copied_y = copy_plane(output,
                                     output_size,
                                     0,
                                     param.stFrame.u64VirAddr[0],
                                     static_cast<AX_U32>(y_stride),
                                     static_cast<AX_U32>(height));
    const bool copied_uv = copy_plane(output,
                                      output_size,
                                      uv_offset,
                                      param.stFrame.u64VirAddr[1],
                                      static_cast<AX_U32>(uv_stride),
                                      static_cast<AX_U32>(height / 2));

    AX_SYS_MemFree(out_phy, out_vir);
    AX_SYS_MemFree(jpg_phy, jpg_vir);

    if (!copied_y || !copied_uv) {
        (*this->_logger)->error("[vdec] failed to copy decoded planes");
        return VDEC_ERR_COPY;
    }

    return VDEC_OK;
}

int AxeraUtils::vdec_decode_jpeg_file(const std::string& path,
                                      uint8_t* output,
                                      size_t output_size,
                                      int& width,
                                      int& height,
                                      int& y_stride,
                                      int& uv_stride,
                                      size_t& written_size)
{
    std::ifstream file(path, std::ios::binary);
    if (!file) {
        (*this->_logger)->error("[vdec] failed to open JPEG file: {}", path);
        return VDEC_ERR_READ_FILE;
    }

    std::vector<uint8_t> jpeg((std::istreambuf_iterator<char>(file)),
                              std::istreambuf_iterator<char>());
    if (!file.eof() && file.fail()) {
        (*this->_logger)->error("[vdec] failed to read JPEG file: {}", path);
        return VDEC_ERR_READ_FILE;
    }
    if (jpeg.empty()) {
        (*this->_logger)->error("[vdec] JPEG file is empty: {}", path);
        return VDEC_ERR_BAD_JPEG;
    }

    return vdec_decode_jpeg(jpeg.data(),
                            jpeg.size(),
                            output,
                            output_size,
                            width,
                            height,
                            y_stride,
                            uv_stride,
                            written_size);
}
