import { useEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { getPortalConfig, reportPortalDetectedArea, setPortalConfig } from '@/api/vm.ts';
import * as storage from '@/lib/localstorage.ts';
import {
  defaultPortalConfig,
  getScreenCropFromManualRegion,
  getScreenCropFromResolution,
  initialScreenCropState,
  PortalConfigResponse
} from '@/lib/screen-crop.ts';
import { portalAreaReport, portalAreaReportKey } from '@/lib/portal-report.ts';
import {
  detectedScreenAreaAtom,
  manualScreenAreaAtom,
  portalConfigAtom,
  portalDetectionPhaseAtom,
  screenSourceSizeAtom
} from '@/jotai/screen.ts';

export const PortalController = () => {
  const config = useAtomValue(portalConfigAtom);
  const sourceSize = useAtomValue(screenSourceSizeAtom);
  const detectedArea = useAtomValue(detectedScreenAreaAtom);
  const detectionPhase = useAtomValue(portalDetectionPhaseAtom);
  const setConfig = useSetAtom(portalConfigAtom);
  const setManualArea = useSetAtom(manualScreenAreaAtom);
  const loadedRef = useRef(false);
  const reportKeyRef = useRef('');
  const observedCropRef = useRef(false);
  const detectionTicksRef = useRef(0);

  useEffect(() => {
    if (loadedRef.current) return;
    loadedRef.current = true;

    void getPortalConfig()
      .then(async (rsp) => {
        if (rsp.code !== 0 || !rsp.data) return;
        const response = rsp.data as PortalConfigResponse;
        const legacyMode = storage.getStoredPortalMode();
        const nextConfig = {
          ...defaultPortalConfig,
          ...response,
          mode: response.configured ? response.mode : (legacyMode ?? 'auto')
        };
        setConfig(nextConfig);
        storage.setPortalMode(nextConfig.mode);

        if (!response.configured && legacyMode === 'off') {
          await setPortalConfig(nextConfig);
        }
      })
      .catch(() => undefined);
  }, [setConfig]);

  useEffect(() => {
    if (config.mode !== 'manual' || sourceSize.width <= 0 || sourceSize.height <= 0) {
      setManualArea(initialScreenCropState);
      return;
    }

    if (config.selectedRegionId) {
      const region = config.manualRegions.find((item) => item.id === config.selectedRegionId);
      setManualArea(
        region
          ? getScreenCropFromManualRegion(region, sourceSize.width, sourceSize.height)
          : initialScreenCropState
      );
      return;
    }

    const resolution = config.resolutions.find(
      (item) => `${item.width}x${item.height}` === config.selectedResolution
    );
    setManualArea(
      resolution
        ? getScreenCropFromResolution(resolution, sourceSize.width, sourceSize.height)
        : initialScreenCropState
    );
  }, [config, setManualArea, sourceSize]);

  // A new stream resolution invalidates the detector's previous coordinates.
  // Reset these before the reporting effect below runs for the new source size.
  useEffect(() => {
    reportKeyRef.current = '';
    observedCropRef.current = false;
    detectionTicksRef.current = 0;
  }, [config.mode, sourceSize.width, sourceSize.height]);

  // The server persists the auto-detected area so MCP screenshots and clicks
  // use the same crop the page shows. Only stable detections are reported.
  useEffect(() => {
    detectionTicksRef.current += 1;

    const areaMatchesSource =
      detectedArea.sourceWidth === sourceSize.width && detectedArea.sourceHeight === sourceSize.height;
    const report = portalAreaReport({
      mode: config.mode,
      phase: detectionPhase,
      contentRect: areaMatchesSource ? detectedArea.contentRect : null,
      sourceWidth: sourceSize.width,
      sourceHeight: sourceSize.height,
      observedCrop: areaMatchesSource && observedCropRef.current,
      detectionTicks: detectionTicksRef.current
    });
    const key = portalAreaReportKey(report);
    if (!report || !key || key === reportKeyRef.current) {
      return;
    }

    reportKeyRef.current = key;
    if (report.contentRect) {
      observedCropRef.current = true;
    }
    void reportPortalDetectedArea(report.sourceWidth, report.sourceHeight, report.contentRect).catch(
      () => {
        reportKeyRef.current = '';
      }
    );
  }, [config.mode, detectedArea, detectionPhase, sourceSize]);

  return null;
};