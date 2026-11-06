#!/usr/bin/env python3
"""
Raw framebuffer + evdev touch diagnostic board.

This script usually needs root permissions, or read/write access to the
framebuffer device and read access to the input event device. It deliberately
does not rotate, swap axes, scale, crop, apply LVGL mapping, or compensate for
dead areas. Raw touch coordinates are used directly as framebuffer x/y pixels.
"""

from __future__ import annotations

import argparse
import datetime as dt
import fcntl
import json
import os
import struct
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional


EV_SYN = 0x00
EV_KEY = 0x01
EV_ABS = 0x03

SYN_REPORT = 0x00
BTN_TOUCH = 0x014A

ABS_X = 0x00
ABS_Y = 0x01
ABS_MT_POSITION_X = 0x35
ABS_MT_POSITION_Y = 0x36

DEFAULT_FB = "/dev/fb0"
DEFAULT_EVENT = "/dev/input/event0"
DEFAULT_WIDTH = 284
DEFAULT_HEIGHT = 240
DEFAULT_BPP = 16
DEFAULT_TARGET_SIZE = 24
DEAD_AREA_BOUNDARY_X = 14
DEAD_AREA_BOUNDARY_Y = 14

ABS_CODES_TO_QUERY = (ABS_X, ABS_Y, ABS_MT_POSITION_X, ABS_MT_POSITION_Y)


@dataclass(frozen=True)
class Rect:
    x: int
    y: int
    w: int
    h: int

    def contains(self, x: int, y: int) -> bool:
        return self.x <= x < self.x + self.w and self.y <= y < self.y + self.h

    def to_json(self) -> Dict[str, int]:
        return {"x": self.x, "y": self.y, "w": self.w, "h": self.h}


@dataclass(frozen=True)
class Target:
    id: str
    name: str
    rect: Rect
    color: str

    def to_json(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "name": self.name,
            "rect": self.rect.to_json(),
            "color": self.color,
        }


@dataclass(frozen=True)
class InputEvent:
    sec: int
    usec: int
    event_type: int
    code: int
    value: int


@dataclass(frozen=True)
class FrameBufferGeometry:
    path: str
    width: int
    height: int
    bpp: int
    stride: int
    source: Dict[str, str]
    sysfs: Dict[str, Any]

    @property
    def bytes_per_pixel(self) -> int:
        return self.bpp // 8


@dataclass(frozen=True)
class InputEventLayout:
    fmt: str
    size: int
    timeval_bits: int

    @classmethod
    def native(cls) -> "InputEventLayout":
        # input_event uses native long timeval fields: 16 bytes on 32-bit,
        # 24 bytes on 64-bit Linux. Native struct mode keeps that ABI layout.
        fmt = "@llHHi"
        size = struct.calcsize(fmt)
        timeval_bits = 64 if size == 24 else 32
        return cls(fmt=fmt, size=size, timeval_bits=timeval_bits)

    def unpack(self, data: bytes) -> InputEvent:
        sec, usec, event_type, code, value = struct.unpack(self.fmt, data)
        return InputEvent(sec=sec, usec=usec, event_type=event_type, code=code, value=value)


def positive_int(value: str) -> int:
    try:
        parsed = int(value, 10)
    except ValueError as exc:
        raise argparse.ArgumentTypeError(f"not an integer: {value}") from exc
    if parsed <= 0:
        raise argparse.ArgumentTypeError("must be greater than zero")
    return parsed


def parse_args(argv: Optional[List[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Raw framebuffer + evdev touch diagnostic board. No coordinate mapping is applied."
    )
    parser.add_argument("--fb", default=DEFAULT_FB, help=f"framebuffer device, default: {DEFAULT_FB}")
    parser.add_argument("--event", default=DEFAULT_EVENT, help=f"evdev input device, default: {DEFAULT_EVENT}")
    parser.add_argument("--width", type=positive_int, help=f"framebuffer raw width; default auto-detects from sysfs, fallback: {DEFAULT_WIDTH}")
    parser.add_argument("--height", type=positive_int, help=f"framebuffer raw height; default auto-detects from sysfs, fallback: {DEFAULT_HEIGHT}")
    parser.add_argument("--bpp", type=positive_int, help=f"bits per pixel; default auto-detects from sysfs, fallback: {DEFAULT_BPP}")
    parser.add_argument("--stride", type=positive_int, help="framebuffer line length in bytes; default auto-detects from sysfs")
    parser.add_argument("--output", help="write final JSON report to this path on exit")
    parser.add_argument("--no-auto-fb", action="store_true", help="do not read /sys/class/graphics/fb* geometry")
    parser.add_argument("--info-only", action="store_true", help="print detected framebuffer/input metadata and exit without drawing")
    parser.add_argument(
        "--target-size",
        type=positive_int,
        default=DEFAULT_TARGET_SIZE,
        help=f"square target size in pixels, default: {DEFAULT_TARGET_SIZE}",
    )
    return parser.parse_args(argv)


def clamp(value: int, lower: int, upper: int) -> int:
    return max(lower, min(value, upper))


def make_block_rect(x: int, y: int, size: int, width: int, height: int) -> Rect:
    w = min(size, width)
    h = min(size, height)
    return Rect(x=clamp(x, 0, width - w), y=clamp(y, 0, height - h), w=w, h=h)


def centered_block_rect(cx: int, cy: int, size: int, width: int, height: int) -> Rect:
    return make_block_rect(cx - size // 2, cy - size // 2, size, width, height)


def make_targets(width: int, height: int, target_size: int) -> List[Target]:
    last_x = max(0, width - target_size)
    last_y = max(0, height - target_size)
    mid_x = width // 2
    mid_y = height // 2

    targets = [
        Target("corner_tl", "top-left corner", make_block_rect(0, 0, target_size, width, height), "#ff0000"),
        Target("corner_tr", "top-right corner", make_block_rect(last_x, 0, target_size, width, height), "#00ff00"),
        Target("corner_bl", "bottom-left corner", make_block_rect(0, last_y, target_size, width, height), "#0000ff"),
        Target("corner_br", "bottom-right corner", make_block_rect(last_x, last_y, target_size, width, height), "#ffff00"),
        Target("edge_top", "top edge midpoint", centered_block_rect(mid_x, 0, target_size, width, height), "#ff7f00"),
        Target("edge_bottom", "bottom edge midpoint", centered_block_rect(mid_x, height - 1, target_size, width, height), "#00ffff"),
        Target("edge_left", "left edge midpoint", centered_block_rect(0, mid_y, target_size, width, height), "#ff00ff"),
        Target("edge_right", "right edge midpoint", centered_block_rect(width - 1, mid_y, target_size, width, height), "#ffffff"),
        Target("center", "screen center", centered_block_rect(mid_x, mid_y, target_size, width, height), "#7fff00"),
    ]

    if DEAD_AREA_BOUNDARY_X < width:
        line_width = min(2, width - DEAD_AREA_BOUNDARY_X)
        targets.append(
            Target(
                "dead_area_x14",
                "14px dead-area boundary reference line",
                Rect(DEAD_AREA_BOUNDARY_X, 0, line_width, height),
                "#ff00ff",
            )
        )

    if DEAD_AREA_BOUNDARY_Y < height:
        line_height = min(2, height - DEAD_AREA_BOUNDARY_Y)
        targets.append(
            Target(
                "dead_area_y14",
                "14px raw-y reference line",
                Rect(0, DEAD_AREA_BOUNDARY_Y, width, line_height),
                "#00ff7f",
            )
        )

    right_reference_x = width - DEAD_AREA_BOUNDARY_X - 1
    if 0 <= right_reference_x < width:
        targets.append(
            Target(
                "raw_x_width_minus_15",
                "raw x=width-15 reference line",
                Rect(right_reference_x, 0, 1, height),
                "#7f00ff",
            )
        )

    bottom_reference_y = height - DEAD_AREA_BOUNDARY_Y - 1
    if 0 <= bottom_reference_y < height:
        targets.append(
            Target(
                "raw_y_height_minus_15",
                "raw y=height-15 reference line",
                Rect(0, bottom_reference_y, width, 1),
                "#ff007f",
            )
        )

    return targets


def rgb565_from_hex(color: str) -> int:
    if len(color) != 7 or not color.startswith("#"):
        raise ValueError(f"invalid color: {color}")
    r = int(color[1:3], 16)
    g = int(color[3:5], 16)
    b = int(color[5:7], 16)
    return ((r & 0xF8) << 8) | ((g & 0xFC) << 3) | (b >> 3)


class FrameBuffer:
    def __init__(self, geometry: FrameBufferGeometry) -> None:
        if geometry.bpp != 16:
            raise ValueError("only RGB565 little-endian framebuffer mode is supported; use --bpp 16")
        self.path = geometry.path
        self.width = geometry.width
        self.height = geometry.height
        self.bpp = geometry.bpp
        self.stride = geometry.stride
        self.bytes_per_pixel = geometry.bytes_per_pixel
        minimum_stride = self.width * self.bytes_per_pixel
        if self.stride < minimum_stride:
            raise ValueError(f"framebuffer stride {self.stride} is smaller than visible row bytes {minimum_stride}")
        self.byte_size = self.stride * self.height
        self._file = None  # type: Optional[Any]

    def __enter__(self) -> "FrameBuffer":
        self._file = open(self.path, "r+b", buffering=0)
        return self

    def __exit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        if self._file is not None:
            self._file.close()
            self._file = None

    def in_bounds(self, x: int, y: int) -> bool:
        return 0 <= x < self.width and 0 <= y < self.height

    def draw_board(self, targets: Iterable[Target]) -> None:
        buf = bytearray(self.byte_size)
        for target in targets:
            self._draw_rect(buf, target.rect, rgb565_from_hex(target.color))
        self._require_file().seek(0)
        self._require_file().write(buf)
        self._require_file().flush()

    def invert_cross(self, x: int, y: int, radius: int = 3) -> None:
        if not self.in_bounds(x, y):
            return
        seen = set()
        for delta in range(-radius, radius + 1):
            seen.add((x + delta, y))
            seen.add((x, y + delta))
        for px, py in seen:
            self.invert_pixel(px, py)
        self._require_file().flush()

    def invert_pixel(self, x: int, y: int) -> None:
        if not self.in_bounds(x, y):
            return
        file_obj = self._require_file()
        offset = y * self.stride + x * self.bytes_per_pixel
        file_obj.seek(offset)
        data = file_obj.read(self.bytes_per_pixel)
        if len(data) != self.bytes_per_pixel:
            return
        value = int.from_bytes(data, byteorder="little") ^ 0xFFFF
        file_obj.seek(offset)
        file_obj.write(value.to_bytes(self.bytes_per_pixel, byteorder="little"))

    def _draw_rect(self, buf: bytearray, rect: Rect, color565: int) -> None:
        x0 = clamp(rect.x, 0, self.width)
        y0 = clamp(rect.y, 0, self.height)
        x1 = clamp(rect.x + rect.w, 0, self.width)
        y1 = clamp(rect.y + rect.h, 0, self.height)
        if x0 >= x1 or y0 >= y1:
            return
        pixel = color565.to_bytes(self.bytes_per_pixel, byteorder="little")
        row = pixel * (x1 - x0)
        row_bytes = len(row)
        for y in range(y0, y1):
            start = y * self.stride + x0 * self.bytes_per_pixel
            buf[start : start + row_bytes] = row

    def _require_file(self) -> Any:
        if self._file is None:
            raise RuntimeError("framebuffer is not open")
        return self._file


class TouchSampler:
    def __init__(self, fb: FrameBuffer, targets: List[Target]) -> None:
        self.fb = fb
        self.targets = targets
        self.samples = []  # type: List[Dict[str, Any]]
        self.raw_x = None  # type: Optional[int]
        self.raw_y = None  # type: Optional[int]
        self.source_x = None  # type: Optional[str]
        self.source_y = None  # type: Optional[str]
        self.last_code_source = None  # type: Optional[str]
        self.pressed = False
        self.coordinate_dirty = False

    def handle(self, event: InputEvent) -> None:
        self.last_code_source = event_code_name(event.event_type, event.code)

        if event.event_type == EV_ABS:
            self._handle_abs(event)
            return
        if event.event_type == EV_KEY and event.code == BTN_TOUCH:
            self._handle_btn_touch(event)
            return
        if event.event_type == EV_SYN and event.code == SYN_REPORT:
            self._handle_syn_report(event)

    def _handle_abs(self, event: InputEvent) -> None:
        if event.code in (ABS_X, ABS_MT_POSITION_X):
            self.raw_x = event.value
            self.source_x = abs_code_name(event.code)
            self.coordinate_dirty = True
        elif event.code in (ABS_Y, ABS_MT_POSITION_Y):
            self.raw_y = event.value
            self.source_y = abs_code_name(event.code)
            self.coordinate_dirty = True

    def _handle_btn_touch(self, event: InputEvent) -> None:
        was_pressed = self.pressed
        self.pressed = event.value != 0
        if was_pressed and not self.pressed:
            self._record_if_ready(event, "release")

    def _handle_syn_report(self, event: InputEvent) -> None:
        if self.coordinate_dirty:
            self._record_if_ready(event, "syn_report")

    def _record_if_ready(self, event: InputEvent, reason: str) -> None:
        if self.raw_x is None or self.raw_y is None:
            return

        x = self.raw_x
        y = self.raw_y
        in_bounds = self.fb.in_bounds(x, y)
        hit_targets = [target for target in self.targets if target.rect.contains(x, y)] if in_bounds else []
        if in_bounds:
            self.fb.invert_cross(x, y)

        sample = {
            "index": len(self.samples) + 1,
            "timestamp": {
                "wall_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
                "event_sec": event.sec,
                "event_usec": event.usec,
            },
            "reason": reason,
            "raw_x": x,
            "raw_y": y,
            "in_screen_bounds": in_bounds,
            "hit_targets": [target.id for target in hit_targets],
            "hit_target_names": [target.name for target in hit_targets],
            "event_code_source": {
                "x": self.source_x,
                "y": self.source_y,
                "last": self.last_code_source,
            },
            "pressed": self.pressed,
        }
        self.samples.append(sample)
        self.coordinate_dirty = False
        print(json.dumps(sample, separators=(",", ":")), flush=True)


def event_code_name(event_type: int, code: int) -> str:
    if event_type == EV_SYN and code == SYN_REPORT:
        return "SYN_REPORT"
    if event_type == EV_KEY and code == BTN_TOUCH:
        return "BTN_TOUCH"
    if event_type == EV_ABS:
        return abs_code_name(code)
    return f"type_{event_type:#x}/code_{code:#x}"


def abs_code_name(code: int) -> str:
    names = {
        ABS_X: "ABS_X",
        ABS_Y: "ABS_Y",
        ABS_MT_POSITION_X: "ABS_MT_POSITION_X",
        ABS_MT_POSITION_Y: "ABS_MT_POSITION_Y",
    }
    return names.get(code, f"ABS_{code:#x}")


def read_exact(fd: int, size: int) -> Optional[bytes]:
    chunks = []
    remaining = size
    while remaining > 0:
        try:
            chunk = os.read(fd, remaining)
        except InterruptedError:
            continue
        if not chunk:
            return None
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)


def fb_sysfs_path(fb_path: str) -> Optional[Path]:
    name = Path(fb_path).name
    if not name.startswith("fb"):
        return None
    path = Path("/sys/class/graphics") / name
    return path if path.exists() else None


def read_sysfs_text(path: Path) -> Optional[str]:
    try:
        return path.read_text(encoding="utf-8").strip()
    except OSError:
        return None


def detect_fb_sysfs(fb_path: str) -> Dict[str, Any]:
    sysfs_path = fb_sysfs_path(fb_path)
    if sysfs_path is None:
        return {}

    info = {"sysfs_path": str(sysfs_path)}  # type: Dict[str, Any]
    for key in ("name", "virtual_size", "bits_per_pixel", "stride", "modes", "mode"):
        value = read_sysfs_text(sysfs_path / key)
        if value is not None:
            info[key] = value
    return info


def parse_virtual_size(value: Optional[str]) -> Optional[Dict[str, int]]:
    if not value:
        return None
    parts = value.replace("x", ",").split(",")
    if len(parts) != 2:
        return None
    try:
        return {"width": int(parts[0], 10), "height": int(parts[1], 10)}
    except ValueError:
        return None


def parse_optional_int(value: Any) -> Optional[int]:
    if value is None:
        return None
    try:
        return int(str(value), 10)
    except ValueError:
        return None


def choose_geometry_value(
    explicit: Optional[int], detected: Optional[int], fallback: int, source_name: str
) -> tuple[int, str]:
    if explicit is not None:
        return explicit, "argument"
    if detected is not None:
        return detected, source_name
    return fallback, "fallback"


def resolve_fb_geometry(args: argparse.Namespace) -> FrameBufferGeometry:
    sysfs = {} if args.no_auto_fb else detect_fb_sysfs(args.fb)
    virtual_size = parse_virtual_size(sysfs.get("virtual_size"))
    detected_width = virtual_size["width"] if virtual_size else None
    detected_height = virtual_size["height"] if virtual_size else None
    detected_bpp = parse_optional_int(sysfs.get("bits_per_pixel"))
    detected_stride = parse_optional_int(sysfs.get("stride"))

    width, width_source = choose_geometry_value(args.width, detected_width, DEFAULT_WIDTH, "sysfs.virtual_size")
    height, height_source = choose_geometry_value(args.height, detected_height, DEFAULT_HEIGHT, "sysfs.virtual_size")
    bpp, bpp_source = choose_geometry_value(args.bpp, detected_bpp, DEFAULT_BPP, "sysfs.bits_per_pixel")

    if bpp % 8 != 0:
        raise ValueError(f"unsupported bpp {bpp}; bpp must be byte-aligned")
    fallback_stride = width * (bpp // 8)
    stride, stride_source = choose_geometry_value(args.stride, detected_stride, fallback_stride, "sysfs.stride")

    return FrameBufferGeometry(
        path=args.fb,
        width=width,
        height=height,
        bpp=bpp,
        stride=stride,
        source={
            "width": width_source,
            "height": height_source,
            "bpp": bpp_source,
            "stride": stride_source,
        },
        sysfs=sysfs,
    )


def ioc(direction: int, ioctl_type: int, nr: int, size: int) -> int:
    return (direction << 30) | (size << 16) | (ioctl_type << 8) | nr


def eviocgabs(abs_code: int) -> int:
    return ioc(2, ord("E"), 0x40 + abs_code, struct.calcsize("iiiiii"))


def read_event_abs_info(event_path: str) -> Dict[str, Any]:
    result = {}  # type: Dict[str, Any]
    try:
        fd = os.open(event_path, os.O_RDONLY)
    except OSError as exc:
        return {"error": str(exc)}

    try:
        for code in ABS_CODES_TO_QUERY:
            name = abs_code_name(code)
            try:
                data = fcntl.ioctl(fd, eviocgabs(code), bytes(struct.calcsize("iiiiii")))
            except OSError:
                continue
            value, minimum, maximum, fuzz, flat, resolution = struct.unpack("iiiiii", data)
            result[name] = {
                "code": code,
                "value": value,
                "min": minimum,
                "max": maximum,
                "fuzz": fuzz,
                "flat": flat,
                "resolution": resolution,
            }
    finally:
        os.close(fd)

    return result


def read_touch_events(event_path: str, layout: InputEventLayout, sampler: TouchSampler) -> None:
    fd = os.open(event_path, os.O_RDONLY)
    try:
        while True:
            data = read_exact(fd, layout.size)
            if data is None:
                raise OSError(f"unexpected EOF from {event_path}")
            sampler.handle(layout.unpack(data))
    finally:
        os.close(fd)


def write_report(path: str, report: Dict[str, Any]) -> None:
    output_path = Path(path)
    if output_path.parent != Path(""):
        output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def build_target_hit_summary(targets: List[Target], samples: List[Dict[str, Any]]) -> Dict[str, Any]:
    target_hits = {target.id: [] for target in targets}
    misses = []
    out_of_bounds = []

    for sample in samples:
        compact_sample = {
            "index": sample["index"],
            "raw_x": sample["raw_x"],
            "raw_y": sample["raw_y"],
            "reason": sample["reason"],
            "pressed": sample["pressed"],
            "timestamp": sample["timestamp"],
            "event_code_source": sample["event_code_source"],
        }
        if not sample["in_screen_bounds"]:
            out_of_bounds.append(compact_sample)
            continue
        hit_targets = sample["hit_targets"]
        if not hit_targets:
            misses.append(compact_sample)
            continue
        for target_id in hit_targets:
            target_hits.setdefault(target_id, []).append(compact_sample)

    return {
        "by_target": target_hits,
        "misses": misses,
        "out_of_bounds": out_of_bounds,
    }


def build_report(
    args: argparse.Namespace,
    geometry: FrameBufferGeometry,
    layout: InputEventLayout,
    event_abs_info: Dict[str, Any],
    targets: List[Target],
    samples: List[Dict[str, Any]],
) -> Dict[str, Any]:
    return {
        "framebuffer": {
            "path": geometry.path,
            "width": geometry.width,
            "height": geometry.height,
            "bpp": geometry.bpp,
            "stride": geometry.stride,
            "pixel_format": "RGB565_LE",
            "raw_direction": "framebuffer memory order: x left-to-right, y top-to-bottom",
            "source": geometry.source,
            "sysfs": geometry.sysfs,
        },
        "event_device": {
            "path": args.event,
            "input_event_size": layout.size,
            "timeval_bits": layout.timeval_bits,
            "abs_info": event_abs_info,
        },
        "target_size": args.target_size,
        "targets": [target.to_json() for target in targets],
        "sample_count": len(samples),
        "target_hit_summary": build_target_hit_summary(targets, samples),
        "samples": samples,
    }


def main(argv: Optional[List[str]] = None) -> int:
    args = parse_args(argv)
    layout = InputEventLayout.native()
    geometry = resolve_fb_geometry(args)
    event_abs_info = read_event_abs_info(args.event)
    targets = make_targets(geometry.width, geometry.height, args.target_size)
    sampler = None  # type: Optional[TouchSampler]
    return_code = 0

    if args.info_only:
        print(json.dumps(build_report(args, geometry, layout, event_abs_info, targets, []), indent=2, sort_keys=True))
        return 0

    try:
        with FrameBuffer(geometry) as fb:
            fb.draw_board(targets)
            sampler = TouchSampler(fb, targets)
            print(
                f"raw_touch_board: fb={geometry.path} event={args.event} "
                f"size={geometry.width}x{geometry.height} stride={geometry.stride} "
                f"bpp={geometry.bpp} input_event_size={layout.size}",
                file=sys.stderr,
                flush=True,
            )
            print("touch samples are printed as compact JSON lines on stdout; press Ctrl+C to stop", file=sys.stderr, flush=True)
            read_touch_events(args.event, layout, sampler)
    except KeyboardInterrupt:
        print("interrupted; exiting", file=sys.stderr, flush=True)
    except (OSError, ValueError, RuntimeError) as exc:
        print(f"error: {exc}", file=sys.stderr, flush=True)
        return_code = 1
    finally:
        if args.output:
            samples = sampler.samples if sampler is not None else []
            report = build_report(args, geometry, layout, event_abs_info, targets, samples)
            try:
                write_report(args.output, report)
                print(f"wrote JSON report: {args.output}", file=sys.stderr, flush=True)
            except OSError as exc:
                print(f"error writing JSON report: {exc}", file=sys.stderr, flush=True)
                return_code = 1

    return return_code


if __name__ == "__main__":
    sys.exit(main())
