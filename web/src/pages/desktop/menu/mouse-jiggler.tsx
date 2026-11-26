import { useEffect, useReducer, useState } from 'react';
import { ConfigProvider, InputNumber } from 'antd';
import { MousePointerClickIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { useMouseJiggler } from '@/hooks/useMouseJiggler.ts';
import { MenuItem } from '@/components/menu-item.tsx';

function remainingTime(hours: number, elapsedMs: number) {
  if (hours === 0) return '∞';
  const seconds = Math.max(0, Math.ceil(hours * 3600 - elapsedMs / 1000));
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, seconds % 60]
    .map((n) => String(n).padStart(2, '0'))
    .join(':');
}

export function MouseJigglerButton() {
  const { t } = useTranslation();
  const { config, blocked, busy, hours, editingHours, update } = useMouseJiggler();
  const [open, setOpen] = useState(false);
  const [, tick] = useReducer((value: number) => value + 1, 0);
  const active = !!config?.enabled;
  useEffect(() => {
    if (!open || !active || !hours) return;
    const timer = window.setInterval(tick, 1000);
    return () => window.clearInterval(timer);
  }, [open, active, hours]);

  const elapsed = config?.enabled && !editingHours ? config.elapsedMs + performance.now() - config.sampledAt : 0;
  return (
    <MenuItem
      title={t('settings.device.mouseJiggler.title')}
      icon={<MousePointerClickIcon size={18} className={active && !blocked ? 'text-green-400' : undefined} />}
      action={{ pressed: active, disabled: busy || !config || (blocked && !active), toggle: () => void update('toggle') }}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) void update('refresh');
      }}
      content={
        <div
          className="flex flex-col gap-1"
          onKeyDown={(event) => event.stopPropagation()}
          onKeyUp={(event) => event.stopPropagation()}
        >
          {blocked && <span role="status">{t('settings.device.mouseJiggler.noMouse')}</span>}
          <label className="flex items-center gap-2 whitespace-nowrap">
            {t('settings.device.mouseJiggler.maxRuntime', 'max runtime:')}
            <ConfigProvider theme={{ components: { InputNumber: { controlWidth: 54, handleVisible: true } } }}>
              <InputNumber
                size="small"
                className="[&_.ant-input-number-input-wrap]:me-[22px]"
                aria-label={t('settings.device.mouseJiggler.runtimeHours', 'Max runtime in hours')}
                min={0}
                max={2_147_483_647}
                step={1}
                precision={0}
                value={hours}
                disabled={!config}
                onChange={(value) => {
                  if (value != null && Number.isInteger(value) && value >= 0 && value <= 2_147_483_647 && value !== hours) {
                    void update({ maxHours: value });
                  }
                }}
              />
            </ConfigProvider>
            <span>{t('settings.device.mouseJiggler.hoursUnit', 'hour(s).')}</span>
          </label>
          {!blocked && hours != null && (
            <span role="timer" className="text-xs tabular-nums text-neutral-400">
              {t('settings.device.mouseJiggler.remaining', '{{time}} remaining', { time: remainingTime(hours, elapsed) })}
            </span>
          )}
        </div>
      }
    />
  );
}
