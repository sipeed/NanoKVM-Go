import { useEffect, useState } from 'react';
import { Modal, Tooltip } from 'antd';
import { FolderSyncIcon, HardDriveIcon, XIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/media.ts';
import { MenuItem } from '@/components/menu-item.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';
import { StorageView } from '@/pages/storage/index.tsx';

type View = 'mount' | 'files';

// The menu entry is only a chooser: disk mounting and file sharing each open
// their own settings-style window, so the popover stays a small picker.
export const Storage = () => {
  const { t } = useTranslation();

  const [state, setState] = useState<api.MediaStateInfo>({
    sharedDiskInitialized: false,
    state: 'none',
    hidOnly: false,
    open: [],
    cdrom: false,
    readOnly: false,
    localWritable: false
  });
  const [modalView, setModalView] = useState<View | null>(null);

  async function refresh() {
    try {
      const stateRsp = await api.getMediaState();
      if (stateRsp.code === 0) {
        setState(stateRsp.data);
      }
    } catch {
      // Retried on the next open or poll.
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  function handleOpenChange(open: boolean) {
    if (open) void refresh();
  }

  const unavailable = state.state === 'unsupported';

  const cards: { view: View; icon: typeof HardDriveIcon; label: string }[] = [
    { view: 'mount', icon: HardDriveIcon, label: t('storage.cards.mount') },
    { view: 'files', icon: FolderSyncIcon, label: t('storage.cards.share') }
  ];

  const closeButton = (
    <ButtonBase
      aria-label={t('settings.close')}
      onClick={() => setModalView(null)}
      className="flex h-8 w-8 items-center justify-center rounded-md text-white/45 transition-colors hover:bg-white/[0.06] hover:text-white/85"
    >
      <XIcon size={16} />
    </ButtonBase>
  );

  return (
    <>
      <MenuItem
        title={t('storage.title')}
        icon={<HardDriveIcon size={18} />}
        fresh={state.state !== 'none'}
        onOpenChange={handleOpenChange}
        content={
          <div className="flex w-[320px] flex-col gap-3">
            {unavailable && (
              <span className="text-xs text-neutral-500">
                {state.hidOnly ? t('storage.hidOnly') : t('storage.disabled')}
              </span>
            )}

            {/* view picker */}
            <div className="grid grid-cols-2 gap-2.5">
              {cards.map((card) => {
                const needsInitialization = card.view === 'files' && !state.sharedDiskInitialized;
                const disabled = unavailable || needsInitialization;
                return (
                  <Tooltip
                    key={card.view}
                    title={
                      needsInitialization ? t('storage.files.reasonNotInitialized') : undefined
                    }
                  >
                    <span className={disabled ? 'block cursor-not-allowed' : 'block'}>
                      <ButtonBase
                        disabled={disabled}
                        onClick={() => setModalView(card.view)}
                        className="flex w-full flex-col items-center gap-2.5 rounded-xl border border-white/10 bg-white/[0.04] py-5 text-neutral-300 transition-colors enabled:hover:border-white/20 enabled:hover:bg-white/[0.08] enabled:hover:text-neutral-100 disabled:pointer-events-none"
                      >
                        <card.icon size={24} strokeWidth={1.8} className="text-blue-400" />
                        <span className="text-sm">{card.label}</span>
                      </ButtonBase>
                    </span>
                  </Tooltip>
                );
              })}
            </div>
          </div>
        }
      />

      {/* Keep the mount view alive so closing the window preserves uploads and their progress. */}
      <Modal
        open={modalView === 'mount'}
        width={560}
        centered
        footer={null}
        closable={false}
        forceRender
        destroyOnHidden={false}
        onCancel={() => setModalView(null)}
        styles={{ content: { padding: 0, overflow: 'hidden' } }}
      >
        <StorageView view="mount" active={modalView === 'mount'} headerExtra={closeButton} />
      </Modal>

      <Modal
        open={modalView === 'files'}
        width="80%"
        centered
        footer={null}
        closable={false}
        forceRender
        destroyOnHidden={false}
        onCancel={() => setModalView(null)}
        style={{ maxWidth: '1080px' }}
        styles={{ content: { padding: 0, overflow: 'hidden' } }}
      >
        <div className="h-[80vh] max-h-[700px]">
          <StorageView view="files" active={modalView === 'files'} headerExtra={closeButton} />
        </div>
      </Modal>
    </>
  );
};
