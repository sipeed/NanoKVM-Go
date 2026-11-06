import { PointerEvent as ReactPointerEvent, useEffect, useRef, useState } from 'react';
import { Button, Space, Typography } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { setPortalConfig } from '@/api/vm.ts';
import { LAYER_OVERLAY } from '@/lib/layers.ts';
import * as storage from '@/lib/localstorage.ts';
import { getRenderedScreenRect, PortalManualRegion, ScreenContentRect } from '@/lib/screen-crop.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { portalConfigAtom, portalSelectingAtom, screenSourceSizeAtom } from '@/jotai/screen.ts';

type SelectionRect = ScreenContentRect;
type ResizeHandle = 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w' | 'nw';
type MagnifierState = {
  left: number;
  top: number;
  focusX: number;
  focusY: number;
};
type Operation =
  | { type: 'create'; startX: number; startY: number }
  | { type: 'move'; startX: number; startY: number; initial: SelectionRect }
  | {
      type: 'resize';
      startX: number;
      startY: number;
      initial: SelectionRect;
      handle: ResizeHandle;
      sensitivity: number;
    };

const edgeSnapDistance = 12;
const minimumSelectionSize = 12;
const precisionHoldDelay = 450;
const precisionHoldTolerance = 8;
const precisionSensitivity = 0.25;
const magnifierSize = 144;
const magnifierZoom = 4;
const magnifierOffset = 24;
const magnifierMargin = 8;

export const PortalSelectionOverlay = () => {
  const { t } = useTranslation();
  const [selecting, setSelecting] = useAtom(portalSelectingAtom);
  const [config, setConfig] = useAtom(portalConfigAtom);
  const sourceSize = useAtomValue(screenSourceSizeAtom);
  const setKeyboardEnabled = useSetAtom(isKeyboardEnableAtom);
  const rootRef = useRef<HTMLDivElement>(null);
  const operationRef = useRef<Operation | null>(null);
  const frameRectRef = useRef<SelectionRect | null>(null);
  const selectionRef = useRef<SelectionRect | null>(null);
  const precisionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const precisionStartRef = useRef<{ pointerId: number; x: number; y: number } | null>(null);
  const latestPointerRef = useRef<{ x: number; y: number } | null>(null);
  const precisionActiveRef = useRef(false);
  const magnifierCanvasRef = useRef<HTMLCanvasElement>(null);
  const [frameRect, setFrameRect] = useState<SelectionRect | null>(null);
  const [selection, setSelectionState] = useState<SelectionRect | null>(null);
  const [magnifier, setMagnifier] = useState<MagnifierState | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  function setSelection(next: SelectionRect | null) {
    selectionRef.current = next;
    setSelectionState(next);
  }

  useEffect(() => {
    if (!selecting) return;
    setKeyboardEnabled(false);
    setError('');

    let animationFrame = 0;
    let resizeObserver: ResizeObserver | null = null;
    const updateFrame = () => {
      const screen = document.getElementById('screen');
      if (!screen || sourceSize.width <= 0 || sourceSize.height <= 0) return;
      const rendered = getRenderedScreenRect(
        screen.getBoundingClientRect(),
        sourceSize.width,
        sourceSize.height
      );
      if (!rendered) return;

      frameRectRef.current = rendered;
      setFrameRect(rendered);
      if (!selectionRef.current) {
        const selected = config.manualRegions.find(
          (region) => region.id === config.selectedRegionId
        );
        if (
          selected &&
          selected.sourceWidth === sourceSize.width &&
          selected.sourceHeight === sourceSize.height
        ) {
          setSelection({
            left: rendered.left + (selected.contentRect.left / sourceSize.width) * rendered.width,
            top: rendered.top + (selected.contentRect.top / sourceSize.height) * rendered.height,
            width: (selected.contentRect.width / sourceSize.width) * rendered.width,
            height: (selected.contentRect.height / sourceSize.height) * rendered.height
          });
        }
      }
    };

    animationFrame = requestAnimationFrame(() => {
      animationFrame = requestAnimationFrame(() => {
        updateFrame();
        const screen = document.getElementById('screen');
        if (screen && typeof ResizeObserver !== 'undefined') {
          resizeObserver = new ResizeObserver(updateFrame);
          resizeObserver.observe(screen);
        }
      });
    });
    window.addEventListener('resize', updateFrame);

    return () => {
      cancelAnimationFrame(animationFrame);
      resizeObserver?.disconnect();
      window.removeEventListener('resize', updateFrame);
      stopPrecisionMode();
      operationRef.current = null;
      frameRectRef.current = null;
      setFrameRect(null);
      setSelection(null);
      setKeyboardEnabled(true);
    };
  }, [config.manualRegions, config.selectedRegionId, selecting, setKeyboardEnabled, sourceSize]);

  useEffect(() => {
    if (!selecting) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      setSelecting(false);
    };
    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [selecting, setSelecting]);

  useEffect(() => {
    if (!magnifier) return;
    drawMagnifier(
      magnifierCanvasRef.current,
      magnifier,
      frameRectRef.current,
      sourceSize.width,
      sourceSize.height
    );
  }, [magnifier, sourceSize.height, sourceSize.width]);

  if (!selecting) return null;

  function capturePointer(event: ReactPointerEvent) {
    rootRef.current?.setPointerCapture(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
  }

  function handleCreateStart(event: ReactPointerEvent<HTMLDivElement>) {
    if (event.target !== event.currentTarget || !frameRectRef.current) return;
    stopPrecisionMode();
    const point = clampPoint(event.clientX, event.clientY, frameRectRef.current);
    operationRef.current = { type: 'create', startX: point.x, startY: point.y };
    setSelection({ left: point.x, top: point.y, width: 0, height: 0 });
    capturePointer(event);
  }

  function handleMoveStart(event: ReactPointerEvent<HTMLDivElement>) {
    const current = selectionRef.current;
    if (!current) return;
    stopPrecisionMode();
    operationRef.current = {
      type: 'move',
      startX: event.clientX,
      startY: event.clientY,
      initial: current
    };
    capturePointer(event);
  }

  function handleResizeStart(event: ReactPointerEvent<HTMLDivElement>, handle: ResizeHandle) {
    const current = selectionRef.current;
    if (!current) return;
    stopPrecisionMode();
    operationRef.current = {
      type: 'resize',
      startX: event.clientX,
      startY: event.clientY,
      initial: current,
      handle,
      sensitivity: 1
    };
    latestPointerRef.current = { x: event.clientX, y: event.clientY };
    precisionStartRef.current = {
      pointerId: event.pointerId,
      x: event.clientX,
      y: event.clientY
    };
    precisionTimerRef.current = setTimeout(activatePrecisionMode, precisionHoldDelay);
    capturePointer(event);
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    const operation = operationRef.current;
    const frame = frameRectRef.current;
    if (!operation || !frame) return;
    latestPointerRef.current = { x: event.clientX, y: event.clientY };
    cancelPrecisionHoldAfterMovement(event);

    if (operation.type === 'create') {
      const point = clampPoint(event.clientX, event.clientY, frame);
      setSelection(
        snapSelection(
          {
            left: Math.min(operation.startX, point.x),
            top: Math.min(operation.startY, point.y),
            width: Math.abs(point.x - operation.startX),
            height: Math.abs(point.y - operation.startY)
          },
          frame
        )
      );
      return;
    }

    if (operation.type === 'move') {
      const left = clamp(
        operation.initial.left + event.clientX - operation.startX,
        frame.left,
        frame.left + frame.width - operation.initial.width
      );
      const top = clamp(
        operation.initial.top + event.clientY - operation.startY,
        frame.top,
        frame.top + frame.height - operation.initial.height
      );
      setSelection(snapSelection({ ...operation.initial, left, top }, frame));
      return;
    }

    const adjustedX = operation.startX + (event.clientX - operation.startX) * operation.sensitivity;
    const adjustedY = operation.startY + (event.clientY - operation.startY) * operation.sensitivity;
    const next = resizeSelection(
      operation,
      adjustedX,
      adjustedY,
      frame,
      precisionActiveRef.current ? edgeSnapDistance * precisionSensitivity : edgeSnapDistance
    );
    setSelection(next);
    if (precisionActiveRef.current) {
      updateMagnifier(event.clientX, event.clientY, getResizeHandlePoint(next, operation.handle));
    }
  }

  function handlePointerEnd(event: ReactPointerEvent<HTMLDivElement>) {
    stopPrecisionMode();
    operationRef.current = null;
    if (rootRef.current?.hasPointerCapture(event.pointerId)) {
      rootRef.current.releasePointerCapture(event.pointerId);
    }
    const current = selectionRef.current;
    if (
      current &&
      (current.width < minimumSelectionSize || current.height < minimumSelectionSize)
    ) {
      setSelection(null);
    }
  }

  function activatePrecisionMode() {
    precisionTimerRef.current = null;
    const operation = operationRef.current;
    const current = selectionRef.current;
    const pointer = latestPointerRef.current;
    if (!operation || operation.type !== 'resize' || !current || !pointer) return;

    operationRef.current = {
      ...operation,
      startX: pointer.x,
      startY: pointer.y,
      initial: { ...current },
      sensitivity: precisionSensitivity
    };
    precisionActiveRef.current = true;
    updateMagnifier(pointer.x, pointer.y, getResizeHandlePoint(current, operation.handle));
  }

  function cancelPrecisionHoldAfterMovement(event: ReactPointerEvent<HTMLDivElement>) {
    const start = precisionStartRef.current;
    if (!precisionTimerRef.current || !start || start.pointerId !== event.pointerId) return;
    if (Math.hypot(event.clientX - start.x, event.clientY - start.y) <= precisionHoldTolerance) {
      return;
    }
    clearTimeout(precisionTimerRef.current);
    precisionTimerRef.current = null;
    precisionStartRef.current = null;
  }

  function updateMagnifier(pointerX: number, pointerY: number, focus: { x: number; y: number }) {
    const position = getMagnifierPosition(
      pointerX,
      pointerY,
      window.innerWidth,
      window.innerHeight
    );
    setMagnifier({ ...position, focusX: focus.x, focusY: focus.y });
  }

  function stopPrecisionMode() {
    if (precisionTimerRef.current) clearTimeout(precisionTimerRef.current);
    precisionTimerRef.current = null;
    precisionStartRef.current = null;
    latestPointerRef.current = null;
    precisionActiveRef.current = false;
    setMagnifier(null);
  }

  async function confirm() {
    const frame = frameRectRef.current;
    const current = selectionRef.current;
    if (!frame || !current || sourceSize.width <= 0 || sourceSize.height <= 0) return;

    const left = clamp(
      Math.round(((current.left - frame.left) / frame.width) * sourceSize.width),
      0,
      sourceSize.width - 1
    );
    const top = clamp(
      Math.round(((current.top - frame.top) / frame.height) * sourceSize.height),
      0,
      sourceSize.height - 1
    );
    const right = clamp(
      Math.round(((current.left + current.width - frame.left) / frame.width) * sourceSize.width),
      left + 1,
      sourceSize.width
    );
    const bottom = clamp(
      Math.round(((current.top + current.height - frame.top) / frame.height) * sourceSize.height),
      top + 1,
      sourceSize.height
    );
    const contentRect = { left, top, width: right - left, height: bottom - top };
    const id = createRegionID();
    const region: PortalManualRegion = {
      id,
      sourceWidth: sourceSize.width,
      sourceHeight: sourceSize.height,
      contentRect
    };
    const nextConfig = {
      ...config,
      mode: 'manual' as const,
      manualRegions: [...config.manualRegions, region],
      selectedRegionId: id,
      selectedResolution: ''
    };

    setSaving(true);
    setError('');
    try {
      const rsp = await setPortalConfig(nextConfig);
      if (rsp.code !== 0) {
        setError(t('portal.saveFailed'));
        return;
      }
      setConfig(nextConfig);
      storage.setPortalMode('manual');
      setSelecting(false);
    } catch {
      setError(t('portal.saveFailed'));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      ref={rootRef}
      className="fixed inset-0 cursor-crosshair touch-none overflow-hidden bg-black/35"
      style={{ zIndex: LAYER_OVERLAY }}
      onPointerDown={handleCreateStart}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerEnd}
      onPointerCancel={handlePointerEnd}
    >
      {frameRect && (
        <div
          className="pointer-events-none fixed border border-dashed border-white/70"
          style={frameRect}
        />
      )}
      {selection && (
        <div
          className="fixed cursor-move border-2 border-sky-400 bg-sky-400/10 shadow-[0_0_0_9999px_rgba(0,0,0,0.45)]"
          style={selection}
          onPointerDown={handleMoveStart}
        >
          {resizeHandles.map(({ handle, className, cursor }) => (
            <div
              key={handle}
              className={`absolute h-4 w-4 rounded-full border-2 border-white bg-sky-500 ${className}`}
              style={{ cursor }}
              onPointerDown={(event) => handleResizeStart(event, handle)}
            />
          ))}
        </div>
      )}
      {magnifier && (
        <div
          className="pointer-events-none fixed overflow-hidden rounded-full border-4 border-white bg-black shadow-[0_6px_28px_rgba(0,0,0,0.75)]"
          style={{
            left: magnifier.left,
            top: magnifier.top,
            width: magnifierSize,
            height: magnifierSize
          }}
        >
          <canvas ref={magnifierCanvasRef} className="block h-full w-full" />
          <div className="absolute bottom-2 right-3 rounded bg-black/70 px-1.5 py-0.5 text-xs font-bold text-white">
            {magnifierZoom}×
          </div>
        </div>
      )}
      <div className="fixed left-1/2 top-5 -translate-x-1/2 rounded-lg bg-neutral-900/95 px-4 py-3 text-center shadow-xl">
        <Typography.Text className="text-white">{t('portal.selectionHint')}</Typography.Text>
      </div>
      <div className="fixed bottom-6 left-1/2 -translate-x-1/2 rounded-lg bg-neutral-900/95 px-4 py-3 shadow-xl">
        <Space direction="vertical" align="center">
          {error && <Typography.Text type="danger">{error}</Typography.Text>}
          <Space>
            <Button onClick={() => setSelecting(false)}>{t('portal.cancel')}</Button>
            <Button type="primary" disabled={!selection} loading={saving} onClick={confirm}>
              {t('portal.confirm')}
            </Button>
          </Space>
        </Space>
      </div>
    </div>
  );
};

const resizeHandles: Array<{
  handle: ResizeHandle;
  className: string;
  cursor: string;
}> = [
  { handle: 'nw', className: '-left-2 -top-2', cursor: 'nwse-resize' },
  { handle: 'n', className: 'left-1/2 -top-2 -translate-x-1/2', cursor: 'ns-resize' },
  { handle: 'ne', className: '-right-2 -top-2', cursor: 'nesw-resize' },
  { handle: 'e', className: '-right-2 top-1/2 -translate-y-1/2', cursor: 'ew-resize' },
  { handle: 'se', className: '-bottom-2 -right-2', cursor: 'nwse-resize' },
  { handle: 's', className: '-bottom-2 left-1/2 -translate-x-1/2', cursor: 'ns-resize' },
  { handle: 'sw', className: '-bottom-2 -left-2', cursor: 'nesw-resize' },
  { handle: 'w', className: '-left-2 top-1/2 -translate-y-1/2', cursor: 'ew-resize' }
];

function clampPoint(x: number, y: number, frame: SelectionRect) {
  return {
    x: clamp(x, frame.left, frame.left + frame.width),
    y: clamp(y, frame.top, frame.top + frame.height)
  };
}

function snapSelection(
  selection: SelectionRect,
  frame: SelectionRect,
  snapDistance = edgeSnapDistance
) {
  let left = selection.left;
  let top = selection.top;
  let right = selection.left + selection.width;
  let bottom = selection.top + selection.height;
  if (Math.abs(left - frame.left) <= snapDistance) left = frame.left;
  if (Math.abs(top - frame.top) <= snapDistance) top = frame.top;
  if (Math.abs(right - (frame.left + frame.width)) <= snapDistance) {
    right = frame.left + frame.width;
  }
  if (Math.abs(bottom - (frame.top + frame.height)) <= snapDistance) {
    bottom = frame.top + frame.height;
  }
  return { left, top, width: right - left, height: bottom - top };
}

function resizeSelection(
  operation: Extract<Operation, { type: 'resize' }>,
  clientX: number,
  clientY: number,
  frame: SelectionRect,
  snapDistance = edgeSnapDistance
) {
  const deltaX = clientX - operation.startX;
  const deltaY = clientY - operation.startY;
  let left = operation.initial.left;
  let top = operation.initial.top;
  let right = operation.initial.left + operation.initial.width;
  let bottom = operation.initial.top + operation.initial.height;

  if (operation.handle.includes('w')) left += deltaX;
  if (operation.handle.includes('e')) right += deltaX;
  if (operation.handle.includes('n')) top += deltaY;
  if (operation.handle.includes('s')) bottom += deltaY;

  left = clamp(left, frame.left, right - minimumSelectionSize);
  right = clamp(right, left + minimumSelectionSize, frame.left + frame.width);
  top = clamp(top, frame.top, bottom - minimumSelectionSize);
  bottom = clamp(bottom, top + minimumSelectionSize, frame.top + frame.height);

  return snapSelection(
    { left, top, width: right - left, height: bottom - top },
    frame,
    snapDistance
  );
}

function getResizeHandlePoint(selection: SelectionRect, handle: ResizeHandle) {
  const right = selection.left + selection.width;
  const bottom = selection.top + selection.height;
  return {
    x: handle.includes('w')
      ? selection.left
      : handle.includes('e')
        ? right
        : selection.left + selection.width / 2,
    y: handle.includes('n')
      ? selection.top
      : handle.includes('s')
        ? bottom
        : selection.top + selection.height / 2
  };
}

function getMagnifierPosition(pointerX: number, pointerY: number, width: number, height: number) {
  let left = pointerX + magnifierOffset;
  if (left + magnifierSize > width - magnifierMargin) {
    left = pointerX - magnifierSize - magnifierOffset;
  }
  let top = pointerY - magnifierSize - magnifierOffset;
  if (top < magnifierMargin) {
    top = pointerY + magnifierOffset;
  }
  return {
    left: clamp(
      left,
      magnifierMargin,
      Math.max(magnifierMargin, width - magnifierSize - magnifierMargin)
    ),
    top: clamp(
      top,
      magnifierMargin,
      Math.max(magnifierMargin, height - magnifierSize - magnifierMargin)
    )
  };
}

function drawMagnifier(
  canvas: HTMLCanvasElement | null,
  magnifier: MagnifierState,
  frame: SelectionRect | null,
  sourceWidth: number,
  sourceHeight: number
) {
  const screen = document.getElementById('screen');
  if (
    !canvas ||
    !frame ||
    !(screen instanceof HTMLVideoElement || screen instanceof HTMLCanvasElement) ||
    sourceWidth <= 0 ||
    sourceHeight <= 0
  ) {
    return;
  }

  const outputScale = 2;
  const outputSize = magnifierSize * outputScale;
  if (canvas.width !== outputSize || canvas.height !== outputSize) {
    canvas.width = outputSize;
    canvas.height = outputSize;
  }
  const context = canvas.getContext('2d');
  if (!context) return;

  context.clearRect(0, 0, outputSize, outputSize);
  context.fillStyle = '#000';
  context.fillRect(0, 0, outputSize, outputSize);

  const sampleSize = magnifierSize / magnifierZoom;
  const sampleLeft = magnifier.focusX - sampleSize / 2;
  const sampleTop = magnifier.focusY - sampleSize / 2;
  const visibleLeft = Math.max(sampleLeft, frame.left);
  const visibleTop = Math.max(sampleTop, frame.top);
  const visibleRight = Math.min(sampleLeft + sampleSize, frame.left + frame.width);
  const visibleBottom = Math.min(sampleTop + sampleSize, frame.top + frame.height);

  if (visibleRight > visibleLeft && visibleBottom > visibleTop) {
    const sourceLeft = ((visibleLeft - frame.left) / frame.width) * sourceWidth;
    const sourceTop = ((visibleTop - frame.top) / frame.height) * sourceHeight;
    const sampledWidth = ((visibleRight - visibleLeft) / frame.width) * sourceWidth;
    const sampledHeight = ((visibleBottom - visibleTop) / frame.height) * sourceHeight;
    const destinationLeft = (visibleLeft - sampleLeft) * magnifierZoom * outputScale;
    const destinationTop = (visibleTop - sampleTop) * magnifierZoom * outputScale;
    const destinationWidth = (visibleRight - visibleLeft) * magnifierZoom * outputScale;
    const destinationHeight = (visibleBottom - visibleTop) * magnifierZoom * outputScale;
    try {
      context.imageSmoothingEnabled = false;
      context.drawImage(
        screen,
        sourceLeft,
        sourceTop,
        sampledWidth,
        sampledHeight,
        destinationLeft,
        destinationTop,
        destinationWidth,
        destinationHeight
      );
    } catch {
      context.fillStyle = '#111';
      context.fillRect(0, 0, outputSize, outputSize);
    }
  }

  const center = outputSize / 2;
  context.strokeStyle = '#38bdf8';
  context.lineWidth = 2 * outputScale;
  context.beginPath();
  context.moveTo(center, 0);
  context.lineTo(center, outputSize);
  context.moveTo(0, center);
  context.lineTo(outputSize, center);
  context.stroke();
  context.fillStyle = '#ef4444';
  context.beginPath();
  context.arc(center, center, 3 * outputScale, 0, Math.PI * 2);
  context.fill();
}

function createRegionID() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `region-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value));
}
