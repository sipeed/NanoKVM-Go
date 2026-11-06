import { Divider } from 'antd';
import { useTranslation } from 'react-i18next';

import { RemoteNetworkComponentCard } from './component-card.tsx';
import type { RemoteNetworkComponent } from './types.ts';

type RemoteNetworkHomeProps = {
  components: readonly RemoteNetworkComponent[];
  onOpen: (id: string) => void;
};

/** Home page of the remote network section: one card per registered component. */
export const RemoteNetworkHome = ({ components, onOpen }: RemoteNetworkHomeProps) => {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex items-center justify-between">
        <span className="text-base font-bold">{t('settings.remoteNetwork.title')}</span>
      </div>
      <Divider className="opacity-50" />

      <div className="h-10" />

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {components.map((component) => (
          <RemoteNetworkComponentCard key={component.id} component={component} onOpen={onOpen} />
        ))}
      </div>
    </>
  );
};
