import { useEffect } from 'react';
import { Divider } from 'antd';
import { useAtomValue, useSetAtom } from 'jotai';
import { MouseIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as ls from '@/lib/localstorage';
import { mouseModeAtom, scrollDirectionAtom, scrollIntervalAtom } from '@/jotai/mouse';
import { isPreviewModeAtom } from '@/jotai/preview.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';
import { MenuItem } from '@/components/menu-item.tsx';

import { CalibrationProfiles } from './calibration-profiles.tsx';
import { Direction } from './direction.tsx';
import { HidMode } from './hid-mode.tsx';
import { MouseMode } from './mouse-mode.tsx';
import { RepairIphoneDrag } from './repair-iphone-drag.tsx';
import { ResetHid } from './reset-hid.tsx';
import { Speed } from './speed.tsx';
import { TouchpadGuide } from './touchpad-guide.tsx';

export const Mouse = () => {
  const { t } = useTranslation();
  const isPreviewMode = useAtomValue(isPreviewModeAtom);
  const mouseMode = useAtomValue(mouseModeAtom);
  const responsiveDevice = useResponsiveDevice();

  const setScrollDirection = useSetAtom(scrollDirectionAtom);
  const setScrollInterval = useSetAtom(scrollIntervalAtom);

  useEffect(() => {
    const direction = ls.getMouseScrollDirection();
    if (direction) {
      setScrollDirection(direction > 0 ? 1 : -1);
    }

    const interval = ls.getMouseScrollInterval();
    if (interval) {
      setScrollInterval(interval);
    }
  }, [setScrollDirection, setScrollInterval]);

  const content = (
    <div className="flex flex-col space-y-1">
      <MouseMode />
      <CalibrationProfiles />
      {mouseMode !== 'touch-screen' &&
        !(responsiveDevice.isMobilePortrait && mouseMode === 'digitizer-stylus') && (
          <>
            <Direction />
            <Speed />
          </>
        )}
      <Divider style={{ margin: '10px 0' }} />

      <HidMode />
      <ResetHid />
      <RepairIphoneDrag />
      <TouchpadGuide />
    </div>
  );

  return (
    <MenuItem
      title={t('mouse.title')}
      icon={<MouseIcon size={18} />}
      content={content}
      disabled={isPreviewMode}
    />
  );
};
