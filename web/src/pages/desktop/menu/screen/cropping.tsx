import { useEffect, useState } from 'react';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, InputNumber, Modal, Select, Space, Tooltip, Typography } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { CheckIcon, ChevronRightIcon, CropIcon, ScanSearchIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useMediaQuery } from 'react-responsive';

import { setPortalConfig } from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';
import * as storage from '@/lib/localstorage.ts';
import { PortalConfig, PortalMode } from '@/lib/screen-crop.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { portalConfigAtom, portalDetectionPhaseAtom, portalSelectingAtom } from '@/jotai/screen.ts';
import { MenuSubmenu } from '@/components/menu-item.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

type CroppingProps = {
  open: boolean;
  setOpen: (open: boolean) => void;
  dismissMenus: () => void;
};

export const Cropping = ({ open, setOpen, dismissMenus }: CroppingProps) => {
  const { t } = useTranslation();
  const [config, setConfig] = useAtom(portalConfigAtom);
  const canHover = useMediaQuery({ query: '(hover: hover)' });
  const phase = useAtomValue(portalDetectionPhaseAtom);
  const setSelecting = useSetAtom(portalSelectingAtom);
  const setKeyboardEnabled = useSetAtom(isKeyboardEnableAtom);
  const [resolutionModalOpen, setResolutionModalOpen] = useState(false);
  const [resolutionWidth, setResolutionWidth] = useState<number | null>(null);
  const [resolutionHeight, setResolutionHeight] = useState<number | null>(null);
  const editing = open || resolutionModalOpen;

  useEffect(() => {
    if (!editing) return;
    setKeyboardEnabled(false);
    return () => setKeyboardEnabled(true);
  }, [editing, setKeyboardEnabled]);

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
    dismissMenus();
    setSelecting(true);
  }

  const autoStatus =
    phase === 'active' ? 'portal.statusActive' :
    phase === 'learning' ? 'portal.statusLearning' : 'portal.statusFull';
  const title = t('portal.title');
  const content = (
    <div className={config.mode === 'manual' ? 'w-[260px] max-w-full' : 'max-w-[230px]'}>
      <Space direction="vertical" size="small" className="w-full">
        <div role="group" aria-label={title} className="flex flex-col">
          {(['auto', 'manual', 'off'] as const).map((mode) => (
            <ButtonBase
              key={mode}
              type="button"
              aria-pressed={config.mode === mode}
              onClick={() => void updateMode(mode)}
              className="flex h-[30px] items-center gap-2 rounded px-2 text-left text-sm text-neutral-200 hover:bg-neutral-700/70 hover:text-white"
            >
              <span className="w-4">
                {config.mode === mode && <CheckIcon size={16} className="text-blue-500" />}
              </span>
              {t(`portal.${mode}`)}
            </ButtonBase>
          ))}
        </div>
        <Typography.Text type="secondary" className="text-xs">
          {config.mode === 'auto' ? t('portal.autoDescription') : t('portal.description')}
        </Typography.Text>
        {config.mode === 'auto' && <Typography.Text className="text-xs">{t(autoStatus)}</Typography.Text>}
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
      <MenuSubmenu
        title={title}
        content={content}
        popoverProps={{
          placement: 'rightTop',
          arrow: false,
          trigger: canHover ? ['hover', 'click'] : 'click',
          mouseLeaveDelay: 0.3,
          open,
          onOpenChange: setOpen
        }}
      >
        <ButtonBase
          type="button"
          aria-haspopup="dialog"
          aria-expanded={open}
          className="flex h-[30px] items-center gap-2 rounded pl-3 pr-2 text-left text-neutral-200 hover:bg-neutral-700/70 hover:text-white"
        >
          <CropIcon size={18} />
          <span className="flex-1 text-sm">{title}</span>
          <ChevronRightIcon size={14} />
        </ButtonBase>
      </MenuSubmenu>
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
