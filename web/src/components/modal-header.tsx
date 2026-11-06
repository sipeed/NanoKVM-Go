import type { ReactNode } from 'react';

type ModalHeaderProps = {
  icon: ReactNode;
  title: string;
  status?: ReactNode;
  extra?: ReactNode;
};

// Shared modal title bar for the image dialogs: a gradient icon tile, the
// title, an optional status line and an optional right-side action slot.
export function ModalHeader({ icon, title, status, extra }: ModalHeaderProps) {
  return (
    <div className="flex items-center gap-3 pr-6">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500/30 to-blue-600/10 text-blue-400 ring-1 ring-blue-500/30">
        {icon}
      </div>

      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-base font-medium leading-5 text-neutral-100">{title}</span>
        {status}
      </div>

      {extra}
    </div>
  );
}
