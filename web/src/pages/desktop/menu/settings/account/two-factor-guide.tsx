import { useState, type JSX } from 'react';
import { Modal } from 'antd';
import {
  CircleDotIcon,
  CircleHelpIcon,
  KeyRoundIcon,
  LockIcon,
  RefreshCwIcon,
  ShieldIcon,
  SmartphoneIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

type GuideItem = {
  key: string;
  icon: JSX.Element;
  title: string;
  desc: string;
};

export const TwoFactorGuide = () => {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);

  const items: GuideItem[] = [
    {
      key: 'what',
      icon: <ShieldIcon size={18} />,
      title: t('settings.account.twoFactorGuide.whatTitle'),
      desc: t('settings.account.twoFactorGuide.whatDesc')
    },
    {
      key: 'apps',
      icon: <SmartphoneIcon size={18} />,
      title: t('settings.account.twoFactorGuide.appsTitle'),
      desc: t('settings.account.twoFactorGuide.appsDesc')
    },
    {
      key: 'setup',
      icon: <KeyRoundIcon size={18} />,
      title: t('settings.account.twoFactorGuide.setupTitle'),
      desc: t('settings.account.twoFactorGuide.setupDesc')
    },
    {
      key: 'recovery',
      icon: <LockIcon size={18} />,
      title: t('settings.account.twoFactorGuide.recoveryTitle'),
      desc: t('settings.account.twoFactorGuide.recoveryDesc')
    }
  ];

  return (
    <>
      <button
        type="button"
        className="flex h-[15px] w-[15px] shrink-0 appearance-none items-center justify-center rounded border-0 bg-transparent p-0 leading-none text-neutral-500/60 hover:bg-transparent hover:text-neutral-400 focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
        onClick={() => setIsOpen(true)}
        aria-label={t('settings.account.twoFactorGuide.title')}
      >
        <CircleHelpIcon size={15} />
      </button>

      <Modal
        open={isOpen}
        title={t('settings.account.twoFactorGuide.title')}
        width={680}
        footer={false}
        onCancel={() => setIsOpen(false)}
      >
        <div className="space-y-4 pt-2 text-neutral-200">
          <div className="rounded border border-neutral-800 bg-neutral-950/70 px-4 py-3 text-sm text-neutral-300">
            <div className="flex items-start gap-2">
              <CircleDotIcon className="mt-0.5 shrink-0 text-sky-400" size={16} />
              <span>{t('settings.account.twoFactorGuide.scope')}</span>
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            {items.map((item) => (
              <div
                key={item.key}
                className="rounded border border-neutral-800 bg-neutral-900/80 p-3"
              >
                <div className="mb-2 flex items-center gap-2 text-neutral-100">
                  <div className="flex h-7 w-7 items-center justify-center rounded bg-neutral-800 text-sky-300">
                    {item.icon}
                  </div>
                  <div className="text-sm font-medium">{item.title}</div>
                </div>
                <p className="mb-0 text-sm leading-5 text-neutral-400">{item.desc}</p>
              </div>
            ))}
          </div>

          <div className="flex items-start gap-2 rounded border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
            <RefreshCwIcon className="mt-0.5 shrink-0" size={16} />
            <span>{t('settings.account.twoFactorGuide.logoutNotice')}</span>
          </div>
        </div>
      </Modal>
    </>
  );
};
