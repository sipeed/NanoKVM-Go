import { type ReactNode } from 'react';

type AccountSettingRowProps = {
  label: ReactNode;
  children: ReactNode;
};

export const AccountSettingRow = ({ label, children }: AccountSettingRowProps) => (
  <div className="flex items-center justify-between gap-4">
    <span className="min-w-0">{label}</span>
    <div className="shrink-0">{children}</div>
  </div>
);
