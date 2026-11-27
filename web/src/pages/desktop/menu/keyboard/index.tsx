import { Divider } from 'antd';
import { useAtomValue } from 'jotai';
import { KeyboardIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { isPreviewModeAtom } from '@/jotai/preview.ts';
import { MenuItem } from '@/components/menu-item.tsx';

import { KeyboardLayouts } from './layouts.tsx';
import { LeaderKey } from './leader-key.tsx';
import { Paste } from './paste.tsx';
import { Shortcuts } from './shortcuts';
import { SwapCommandCtrl } from './swap-command-ctrl.tsx';

export const Keyboard = () => {
  const { t } = useTranslation();
  const isPreviewMode = useAtomValue(isPreviewModeAtom);

  const content = (
    <div className="flex flex-col space-y-1">
      <Paste />
      <KeyboardLayouts />
      <SwapCommandCtrl />
      <Shortcuts />

      <Divider style={{ margin: '5px 0', opacity: 0.5 }} />

      <LeaderKey />
    </div>
  );

  return (
    <MenuItem
      title={t('keyboard.title')}
      icon={<KeyboardIcon size={18} />}
      content={content}
      disabled={isPreviewMode}
    />
  );
};
