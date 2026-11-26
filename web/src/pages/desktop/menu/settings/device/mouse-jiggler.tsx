import { Select, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import { useMouseJiggler } from '@/hooks/useMouseJiggler.ts';

export const MouseJiggler = () => {
  const { t } = useTranslation();

  const { config, blocked, busy, update } = useMouseJiggler();

  const options = [
    { value: 'disable', label: t('settings.device.mouseJiggler.disable') },
    { value: 'relative', label: t('settings.device.mouseJiggler.relative') },
    { value: 'absolute', label: t('settings.device.mouseJiggler.absolute') }
  ];

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-col space-y-1">
        <span>{t('settings.device.mouseJiggler.title')}</span>
        <span className="text-xs text-neutral-500">
          {t(blocked ? 'settings.device.mouseJiggler.noMouse' : 'settings.device.mouseJiggler.description')}
        </span>
      </div>

      {config?.enabled ? (
        <Select
          style={{ width: 150 }}
          value={config.mode}
          options={blocked ? options.filter((item) => item.value === 'disable') : options}
          loading={busy}
          onChange={(value: 'relative' | 'absolute' | 'disable') => void update(value)}
        />
      ) : (
        <Switch disabled={!config || blocked} checked={false} loading={busy} onChange={() => void update('toggle')} />
      )}
    </div>
  );
};
