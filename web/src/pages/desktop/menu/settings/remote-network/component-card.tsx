import { ChevronRightIcon } from 'lucide-react';

import { ButtonBase } from '@/components/ui/button-base.tsx';

import type { RemoteNetworkComponent } from './types.ts';

type RemoteNetworkComponentCardProps = {
  component: RemoteNetworkComponent;
  onOpen: (id: string) => void;
};

export const RemoteNetworkComponentCard = ({
  component,
  onOpen
}: RemoteNetworkComponentCardProps) => {
  const { Icon, Status } = component;

  return (
    <ButtonBase
      className="flex items-center gap-3 rounded-[10px] border border-neutral-700 bg-neutral-800 p-3.5 text-left shadow-sm shadow-black/30 transition-colors enabled:hover:border-neutral-600 enabled:hover:bg-neutral-700/80 enabled:active:bg-neutral-700"
      onClick={() => onOpen(component.id)}
    >
      <span
        aria-hidden="true"
        className="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-neutral-700 text-neutral-200"
      >
        <Icon size={22} />
      </span>

      <span className="flex min-w-0 flex-1 flex-col items-start gap-0.5">
        <span className="max-w-full truncate text-[15px] font-semibold text-neutral-100">
          {component.name}
        </span>
        {Status && <Status />}
      </span>

      <ChevronRightIcon aria-hidden="true" size={18} className="shrink-0 text-neutral-500" />
    </ButtonBase>
  );
};
