import { useCallback, useContext, useEffect, useRef, useState } from 'react';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import {
  Button,
  InputNumber,
  Modal,
  Popover,
  Segmented,
  Select,
  Space,
  Tooltip,
  Typography
} from 'antd';
import clsx from 'clsx';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { ScanSearchIcon, SmartphoneIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { setPortalConfig } from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';
import { LAYER_DESKTOP_MENU_POPOVER, LAYER_MOBILE_MENU_POPOVER } from '@/lib/layers.ts';
import * as storage from '@/lib/localstorage.ts';
import { PortalConfig, PortalMode } from '@/lib/screen-crop.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { portalConfigAtom, portalDetectionPhaseAtom, portalSelectingAtom } from '@/jotai/screen.ts';
import { useSubmenuOpenCounter } from '@/hooks/useSubmenuOpenCounter.ts';
import { MobileMenuPanel } from '@/components/menu-item.tsx';
import { MobileMenuItemContext } from '@/components/mobile-menu-context.ts';

type PortalProps = {
  tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
};

export const Portal = ({ tooltipPlacement = 'bottom' }: PortalProps) => {
  const { t } = useTranslation();
  const [config, setConfig] = useAtom(portalConfigAtom);
  const phase = useAtomValue(portalDetectionPhaseAtom);
  const setSelecting = useSetAtom(portalSelectingAtom);
  const setKeyboardEnabled = useSetAtom(isKeyboardEnableAtom);
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const [resolutionModalOpen, setResolutionModalOpen] = useState(false);
  const [resolutionWidth, setResolutionWidth] = useState<number | null>(null);
  const [resolutionHeight, setResolutionHeight] = useState<number | null>(null);
  const { setOpen: setPopoverVisibility } = useSubmenuOpenCounter();
  const { mobilePlacement, mobilePopoverClassName, dismissMenuKey } =
    useContext(MobileMenuItemContext);
  const previousDismissMenuKey = useRef(dismissMenuKey);

  useEffect(() => {
    setKeyboardEnabled(!popoverOpen && !resolutionModalOpen);
    return () => setKeyboardEnabled(true);
  }, [popoverOpen, resolutionModalOpen, setKeyboardEnabled]);

  async function save(nextConfig: PortalConfig) {
    try {
      const rsp = await setPortalConfig(nextConfig);
      if (rsp.code !== 0) {
        feedback.error(t('portal.saveFailed'));
        return false;
      }
      setConfig(nextConfig);
      storage.setPortalMode(nextConfig.mode);
      return true;
    } catch {
      feedback.error(t('portal.saveFailed'));
      return false;
    }
  }

  async function updateMode(mode: PortalMode) {
    await save({ ...config, mode });
  }

  async function selectManualRegion(selectedRegionId: string) {
    await save({
      ...config,
      mode: 'manual',
      selectedRegionId,
      selectedResolution: ''
    });
  }

  async function deleteManualRegion(regionId: string) {
    const selectedRegionId = config.selectedRegionId === regionId ? '' : config.selectedRegionId;
    await save({
      ...config,
      manualRegions: config.manualRegions.filter((region) => region.id !== regionId),
      selectedRegionId
    });
  }

  async function selectResolution(selectedResolution: string) {
    await save({
      ...config,
      mode: 'manual',
      selectedRegionId: '',
      selectedResolution
    });
  }

  async function addResolution() {
    if (!resolutionWidth || !resolutionHeight) {
      feedback.error(t('portal.invalidResolution'));
      return;
    }
    const key = `${resolutionWidth}x${resolutionHeight}`;
    if (config.resolutions.some((item) => `${item.width}x${item.height}` === key)) {
      feedback.warning(t('portal.duplicateResolution'));
      return;
    }

    if (
      await save({
        ...config,
        resolutions: [...config.resolutions, { width: resolutionWidth, height: resolutionHeight }]
      })
    ) {
      setResolutionWidth(null);
      setResolutionHeight(null);
      setResolutionModalOpen(false);
    }
  }

  async function deleteResolution(key: string) {
    const selectedResolution = config.selectedResolution === key ? '' : config.selectedResolution;
    await save({
      ...config,
      resolutions: config.resolutions.filter((item) => `${item.width}x${item.height}` !== key),
      selectedResolution
    });
  }

  function startSelection() {
    updatePopoverOpen(false);
    setSelecting(true);
  }

  const updatePopoverOpen = useCallback(
    (open: boolean) => {
      setTooltipOpen(false);
      setPopoverOpen(open);
      setPopoverVisibility(open);
    },
    [setPopoverVisibility]
  );

  function updateTooltipOpen(open: boolean) {
    if (popoverOpen) return;
    setTooltipOpen(open);
  }

  // Match MenuItem: close the popover when the menu rail asks items to dismiss.
  useEffect(() => {
    if (dismissMenuKey === previousDismissMenuKey.current) {
      return;
    }

    previousDismissMenuKey.current = dismissMenuKey;
    updatePopoverOpen(false);
  }, [dismissMenuKey, updatePopoverOpen]);

  const status =
    config.mode === 'auto'
      ? phase === 'active'
        ? t('portal.statusActive')
        : phase === 'learning'
          ? t('portal.statusLearning')
          : t('portal.statusFull')
      : config.mode === 'manual'
        ? t('portal.manual')
        : t('portal.off');
  const iconClassName =
    config.mode === 'off'
      ? 'text-red-400'
      : config.mode === 'manual'
        ? 'text-green-400'
        : phase === 'active'
          ? 'text-blue-400'
          : 'text-red-400';

  const content = (
    <div className="w-[280px] max-w-full">
      <Space direction="vertical" size="small" className="w-full">
        <Segmented<PortalMode>
          block
          value={config.mode}
          options={[
            { label: t('portal.auto'), value: 'auto' },
            { label: t('portal.manual'), value: 'manual' },
            { label: t('portal.off'), value: 'off' }
          ]}
          onChange={updateMode}
        />
        <Typography.Text type="secondary" className="text-xs">
          {config.mode === 'auto' ? t('portal.autoDescription') : t('portal.description')}
        </Typography.Text>
        {config.mode === 'auto' && <Typography.Text className="text-xs">{status}</Typography.Text>}
        {config.mode === 'manual' && (
          <>
            <Button
              block
              type="primary"
              icon={<ScanSearchIcon size={14} />}
              onClick={startSelection}
            >
              {t('portal.selectArea')}
            </Button>
            <Typography.Text>{t('portal.savedAreas')}</Typography.Text>
            <Select
              className="w-full"
              value={config.selectedRegionId}
              options={[
                { label: t('portal.unused'), value: '' },
                ...config.manualRegions.map((region, index) => ({
                  label: `${region.contentRect.width}×${region.contentRect.height} #${index + 1}`,
                  value: region.id
                }))
              ]}
              optionRender={(option) => (
                <div className="flex items-center justify-between">
                  <span>{option.label}</span>
                  {option.value && (
                    <Button
                      type="text"
                      danger
                      size="small"
                      icon={<DeleteOutlined />}
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={(event) => {
                        event.stopPropagation();
                        void deleteManualRegion(String(option.value));
                      }}
                    />
                  )}
                </div>
              )}
              onChange={selectManualRegion}
            />
            <Typography.Text>{t('portal.originalResolution')}</Typography.Text>
            <Space.Compact block>
              <Select
                className="w-full"
                value={config.selectedResolution}
                options={[
                  { label: t('portal.unused'), value: '' },
                  ...config.resolutions.map((resolution) => ({
                    label: `${resolution.width}×${resolution.height}`,
                    value: `${resolution.width}x${resolution.height}`
                  }))
                ]}
                optionRender={(option) => (
                  <div className="flex items-center justify-between">
                    <span>{option.label}</span>
                    {option.value && (
                      <Button
                        type="text"
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={(event) => {
                          event.stopPropagation();
                          void deleteResolution(String(option.value));
                        }}
                      />
                    )}
                  </div>
                )}
                onChange={selectResolution}
              />
              <Tooltip title={t('portal.addResolution')}>
                <Button icon={<PlusOutlined />} onClick={() => setResolutionModalOpen(true)} />
              </Tooltip>
            </Space.Compact>
          </>
        )}
      </Space>
    </div>
  );

  return (
    <>
      <Popover
        trigger="click"
        arrow={false}
        placement={
          tooltipPlacement === 'left'
            ? 'leftTop'
            : tooltipPlacement === 'right'
              ? 'rightTop'
              : 'bottomLeft'
        }
        zIndex={mobilePlacement ? LAYER_MOBILE_MENU_POPOVER : LAYER_DESKTOP_MENU_POPOVER}
        overlayClassName={mobilePopoverClassName}
        content={
          mobilePlacement ? (
            <MobileMenuPanel
              content={content}
              open={popoverOpen}
              onClose={() => updatePopoverOpen(false)}
            />
          ) : (
            content
          )
        }
        open={popoverOpen}
        onOpenChange={updatePopoverOpen}
      >
        <Tooltip
          title={`${t('portal.title')}: ${status}`}
          placement={tooltipPlacement}
          mouseEnterDelay={0.6}
          open={tooltipOpen}
          onOpenChange={updateTooltipOpen}
        >
          <div
            role="button"
            tabIndex={0}
            className={clsx(
              'flex h-[30px] w-[30px] cursor-pointer items-center justify-center rounded hover:bg-neutral-700/80 hover:text-white',
              iconClassName
            )}
            aria-label={`${t('portal.title')}: ${status}`}
            aria-expanded={popoverOpen}
            onKeyDown={(event) => {
              if (event.key !== 'Enter' && event.key !== ' ') return;
              event.preventDefault();
              updatePopoverOpen(!popoverOpen);
            }}
          >
            <SmartphoneIcon size={18} />
          </div>
        </Tooltip>
      </Popover>
      <Modal
        title={t('portal.addResolution')}
        open={resolutionModalOpen}
        okText={t('portal.add')}
        cancelText={t('portal.cancel')}
        onOk={addResolution}
        onCancel={() => setResolutionModalOpen(false)}
      >
        <Space.Compact block>
          <InputNumber<number>
            className="w-full"
            min={1}
            max={32768}
            precision={0}
            value={resolutionWidth}
            placeholder={t('portal.width')}
            onChange={setResolutionWidth}
          />
          <InputNumber<number>
            className="w-full"
            min={1}
            max={32768}
            precision={0}
            value={resolutionHeight}
            placeholder={t('portal.height')}
            onChange={setResolutionHeight}
          />
        </Space.Compact>
      </Modal>
    </>
  );
};
