import { type ReactNode } from 'react';

import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

type SettingRowProps = {
  label: ReactNode;
  description?: ReactNode;
  control: ReactNode;
  desktopControlWidth?: number;
};

export const SettingRow = ({
  label,
  description,
  control,
  desktopControlWidth = 240
}: SettingRowProps) => {
  const { isMobilePortrait } = useResponsiveDevice();

  if (isMobilePortrait) {
    return (
      <div className="flex flex-col space-y-3">
        <div className="flex flex-col space-y-1">
          {label}
          {description && <span className="text-xs text-neutral-500">{description}</span>}
        </div>
        <div className="box-border w-full px-1">{control}</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col space-y-3 sm:flex-row sm:items-center sm:justify-between sm:space-x-5 sm:space-y-0">
      <div className="flex min-w-0 flex-1 flex-col space-y-1">
        {label}
        {description && <span className="text-xs text-neutral-500">{description}</span>}
      </div>
      <div
        className="box-border w-full max-w-full px-1 sm:max-w-none sm:shrink-0 sm:px-0"
        style={{ width: desktopControlWidth }}
      >
        {control}
      </div>
    </div>
  );
};
