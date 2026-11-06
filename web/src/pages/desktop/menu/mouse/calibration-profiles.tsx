import { useEffect, useState } from 'react';
import { Button, Collapse, Input, Modal, Select } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { CopyIcon, CrosshairIcon, PencilIcon, PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { setPortalConfig as savePortalConfig } from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';
import {
  AUTO_INPUT_PROFILE_ID,
  createInputProfile,
  DEFAULT_CALIBRATION_TARGET_RECT,
  DEFAULT_INPUT_PROFILE_ID,
  getFourPointCalibrationMatrix,
  getNextInputProfileName
} from '@/lib/input-profile.ts';
import type {
  CalibratableMouseMode,
  CalibrationMatrix,
  InputProfile,
  InputProfileStore
} from '@/lib/input-profile.ts';
import * as ls from '@/lib/localstorage.ts';
import { activeCalibrationAtom } from '@/jotai/active-input-calibration.ts';
import {
  calibrationRawPointAtom,
  inputCalibrationSessionAtom,
  inputProfileStoreAtom,
  selectedInputProfileAtom,
  setProfileCalibration
} from '@/jotai/input-profile.ts';
import { mouseModeAtom } from '@/jotai/mouse.ts';
import { portalConfigAtom } from '@/jotai/screen.ts';

export const CalibrationProfiles = () => {
  const { t } = useTranslation();
  const [store, setStore] = useAtom(inputProfileStoreAtom);
  const [selectedProfile, setSelectedProfile] = useAtom(selectedInputProfileAtom);
  const mouseMode = useAtomValue(mouseModeAtom);
  const activeCalibration = useAtomValue(activeCalibrationAtom);
  const portalConfig = useAtomValue(portalConfigAtom);
  const setPortalConfig = useSetAtom(portalConfigAtom);
  const [modal, modalContextHolder] = Modal.useModal();
  const setCalibrationSession = useSetAtom(inputCalibrationSessionAtom);
  const setCalibrationRawPoint = useSetAtom(calibrationRawPointAtom);
  const [open, setOpen] = useState(false);
  const [renameOpen, setRenameOpen] = useState(false);
  const [renameProfileId, setRenameProfileId] = useState('');
  const [renameValue, setRenameValue] = useState('');
  const [matrixOpen, setMatrixOpen] = useState(false);
  const [pendingCalibration, setPendingCalibration] = useState<{
    profileId: string;
    mode: CalibratableMouseMode;
  } | null>(null);

  const mode = mouseMode === 'relative' ? null : mouseMode;
  const calibration = mode ? activeCalibration : null;
  const matrix = getFourPointCalibrationMatrix(calibration);
  const selectedProfileLabel =
    selectedProfile?.id === DEFAULT_INPUT_PROFILE_ID
      ? t('mouse.calibration.defaultProfile')
      : selectedProfile?.id === AUTO_INPUT_PROFILE_ID
        ? t('mouse.calibration.autoProfile')
        : selectedProfile?.name;

  function profileLabel(profile: InputProfile) {
    if (profile.id === DEFAULT_INPUT_PROFILE_ID) return t('mouse.calibration.defaultProfile');
    if (profile.id === AUTO_INPUT_PROFILE_ID) return t('mouse.calibration.autoProfile');
    return profile.name;
  }

  function persist(nextStore: InputProfileStore) {
    setStore(nextStore);
  }

  function createProfile() {
    const profile = createInputProfile(
      getNextInputProfileName(store.profiles, (index) =>
        t('mouse.calibration.newProfileName', { index })
      )
    );
    persist({
      ...store,
      selectedProfileId: profile.id,
      profiles: [...store.profiles, profile]
    });
    setMatrixOpen(false);
  }

  function openRenameProfile(profile: InputProfile) {
    if (profile.origin !== 'user') return;
    setRenameProfileId(profile.id);
    setRenameValue(profile.name);
    setRenameOpen(true);
  }

  function renameProfile() {
    const nextName = renameValue.trim();
    const profile = store.profiles.find((item) => item.id === renameProfileId);
    if (!profile || profile.origin !== 'user' || !nextName) return;
    persist({
      ...store,
      profiles: store.profiles.map((profile) =>
        profile.id === renameProfileId
          ? { ...profile, name: nextName, updatedAt: Date.now() }
          : profile
      )
    });
    setRenameOpen(false);
    setRenameProfileId('');
  }

  function removeProfile(profile: InputProfile) {
    if (profile.origin !== 'user') return;
    modal.confirm({
      title: t('mouse.calibration.deleteTitle'),
      content: profile.name,
      okText: t('auth.ok'),
      cancelText: t('auth.cancel'),
      okButtonProps: { danger: true },
      onOk: () => {
        const profiles = store.profiles.filter((item) => item.id !== profile.id);
        persist({
          ...store,
          selectedProfileId:
            store.selectedProfileId === profile.id
              ? AUTO_INPUT_PROFILE_ID
              : store.selectedProfileId,
          profiles
        });
      }
    });
  }

  function clearCalibration() {
    if (!mode || !selectedProfile || selectedProfile.origin !== 'user') return;
    persist(setProfileCalibration(store, selectedProfile.id, mode, null));
  }

  function beginCalibration() {
    if (!mode || !selectedProfile || selectedProfile.origin !== 'user') return;
    setCalibrationRawPoint(null);
    setCalibrationSession({
      phase: 'position',
      profileId: selectedProfile.id,
      mode,
      targetRect: { ...DEFAULT_CALIBRATION_TARGET_RECT },
      cornerIndex: 0,
      hidSamples: {}
    });
    setOpen(false);
  }

  useEffect(() => {
    if (!pendingCalibration || portalConfig.mode !== 'off') return;

    let firstFrame = 0;
    let secondFrame = 0;
    firstFrame = requestAnimationFrame(() => {
      secondFrame = requestAnimationFrame(() => {
        setCalibrationRawPoint(null);
        setCalibrationSession({
          phase: 'position',
          profileId: pendingCalibration.profileId,
          mode: pendingCalibration.mode,
          targetRect: { ...DEFAULT_CALIBRATION_TARGET_RECT },
          cornerIndex: 0,
          hidSamples: {}
        });
        setPendingCalibration(null);
        setOpen(false);
      });
    });

    return () => {
      cancelAnimationFrame(firstFrame);
      cancelAnimationFrame(secondFrame);
    };
  }, [pendingCalibration, portalConfig.mode, setCalibrationRawPoint, setCalibrationSession]);

  function startCalibration() {
    if (!mode || !selectedProfile || selectedProfile.origin !== 'user') return;
    if (portalConfig.mode === 'off') {
      beginCalibration();
      return;
    }

    modal.confirm({
      title: t('mouse.calibration.portalWarningTitle'),
      content: t('mouse.calibration.portalWarning'),
      okText: t('mouse.calibration.disablePortalAndStart'),
      cancelText: t('auth.cancel'),
      onOk: async () => {
        const nextConfig = { ...portalConfig, mode: 'off' as const };
        try {
          const response = await savePortalConfig(nextConfig);
          if (response.code !== 0) {
            feedback.error(t('portal.saveFailed'));
            return;
          }
          setPortalConfig(nextConfig);
          ls.setPortalMode('off');
          setPendingCalibration({ profileId: selectedProfile.id, mode });
        } catch {
          feedback.error(t('portal.saveFailed'));
        }
      }
    });
  }

  async function copyMatrix() {
    await navigator.clipboard.writeText(formatMatrixForSharing(matrix));
    feedback.success(t('mouse.calibration.matrixCopied'));
  }

  function openCalibrationSettings() {
    setMatrixOpen(false);
    setOpen(true);
  }

  function closeCalibrationSettings() {
    setMatrixOpen(false);
    setOpen(false);
  }

  return (
    <>
      {modalContextHolder}
      <div className="group rounded pb-1 transition-colors hover:bg-neutral-700/70">
        <div
          className="flex h-[30px] cursor-pointer select-none items-center space-x-2 px-3 text-neutral-300"
          onClick={openCalibrationSettings}
        >
          <CrosshairIcon size={18} />
          <span>{t('mouse.calibration.title')}</span>
        </div>
        <div
          role="status"
          aria-live="polite"
          title={selectedProfileLabel}
          className="ml-7 mr-2 flex h-[26px] cursor-default select-none items-center rounded bg-neutral-800/80 px-3 text-xs text-neutral-500 transition-colors group-hover:bg-neutral-800/60"
        >
          <span className="truncate">{selectedProfileLabel}</span>
        </div>
      </div>

      <Modal
        open={open}
        title={t('mouse.calibration.title')}
        width={720}
        footer={null}
        onCancel={closeCalibrationSettings}
      >
        <p className="mb-3 text-xs text-neutral-500">
          {t('mouse.calibration.description')}
          <br />
          {t('mouse.calibration.builtinHint')}
        </p>

        <div className="flex flex-wrap items-center gap-2">
          <Select
            className="min-w-[220px] flex-[1_1_260px]"
            value={selectedProfile?.id || undefined}
            placeholder={t('mouse.calibration.selectProfile')}
            options={store.profiles.map((profile) => ({
              value: profile.id,
              label: profileLabel(profile)
            }))}
            onChange={(profileId) => {
              setSelectedProfile(profileId);
              setMatrixOpen(false);
            }}
          />
          <div className="flex flex-wrap items-center gap-1">
            <Button icon={<PlusIcon size={14} />} onClick={createProfile}>
              {t('mouse.calibration.addProfile')}
            </Button>
            <Button
              icon={<PencilIcon size={14} />}
              disabled={selectedProfile?.origin !== 'user'}
              onClick={() => selectedProfile && openRenameProfile(selectedProfile)}
            >
              {t('mouse.calibration.rename')}
            </Button>
            <Button
              danger
              icon={<Trash2Icon size={14} />}
              disabled={selectedProfile?.origin !== 'user'}
              onClick={() => selectedProfile && removeProfile(selectedProfile)}
            >
              {t('mouse.calibration.delete')}
            </Button>
          </div>
        </div>

        {selectedProfile && (
          <>
            {selectedProfile.origin === 'user' && mode && (
              <div className="mt-4 flex flex-wrap items-center gap-2">
                <Button type="primary" onClick={startCalibration}>
                  {t('mouse.calibration.start')}
                </Button>
                {calibration && (
                  <Button type="primary" danger onClick={clearCalibration}>
                    {t('mouse.calibration.clear')}
                  </Button>
                )}
              </div>
            )}

            {selectedProfile.origin !== 'user' && (
              <p className="mt-3 text-xs text-neutral-500">
                {t('mouse.calibration.builtinReadonly')}
              </p>
            )}

            <div className="mt-4 overflow-hidden rounded-xl border border-neutral-700 bg-neutral-950 text-neutral-100 shadow-inner">
              <Collapse
                ghost
                expandIconPosition="end"
                activeKey={matrixOpen ? ['matrix'] : []}
                onChange={(keys) => {
                  const activeKeys = Array.isArray(keys) ? keys : [keys];
                  setMatrixOpen(activeKeys.includes('matrix'));
                }}
                items={[
                  {
                    key: 'matrix',
                    styles: {
                      header: { padding: '12px 16px' },
                      body: { padding: '0 16px 16px' }
                    },
                    label: (
                      <div>
                        <div className="font-medium leading-5 text-neutral-100">
                          {t('mouse.calibration.matrix')}
                        </div>
                        <div className="mt-1 text-xs leading-4 text-neutral-400">
                          {calibration
                            ? selectedProfile.id === AUTO_INPUT_PROFILE_ID
                              ? t('mouse.calibration.matrixAutomatic')
                              : t('mouse.calibration.matrixActive')
                            : t('mouse.calibration.matrixIdentity')}
                        </div>
                      </div>
                    ),
                    children: (
                      <div className="border-t border-neutral-800 pt-3">
                        <div className="mb-2 flex justify-end">
                          <Button
                            ghost
                            size="small"
                            icon={<CopyIcon size={14} />}
                            onClick={() => void copyMatrix()}
                          >
                            {t('mouse.calibration.copyMatrix')}
                          </Button>
                        </div>

                        <div className="overflow-x-auto py-2">
                          <div className="mx-auto flex w-max min-w-full items-center justify-center gap-3 font-mono text-sm">
                            <ColumnVector values={["x'", "y'", '1']} />
                            <span className="text-lg text-neutral-400">∝</span>
                            <Matrix values={matrix} />
                            <ColumnVector values={['x', 'y', '1']} />
                          </div>
                        </div>

                        <div className="mt-4 space-y-1 border-t border-neutral-800 pt-3 font-mono text-xs text-neutral-500">
                          <div>x′ = (h₁₁x + h₁₂y + h₁₃) / (h₃₁x + h₃₂y + 1)</div>
                          <div>y′ = (h₂₁x + h₂₂y + h₂₃) / (h₃₁x + h₃₂y + 1)</div>
                        </div>
                      </div>
                    )
                  }
                ]}
              />
            </div>
          </>
        )}
      </Modal>

      <Modal
        open={renameOpen}
        title={t('mouse.calibration.renameTitle')}
        okText={t('auth.ok')}
        cancelText={t('auth.cancel')}
        okButtonProps={{ disabled: !renameValue.trim() }}
        onOk={renameProfile}
        onCancel={() => {
          setRenameOpen(false);
          setRenameProfileId('');
        }}
      >
        <Input
          autoFocus
          value={renameValue}
          maxLength={64}
          onChange={(event) => setRenameValue(event.target.value)}
          onPressEnter={renameProfile}
        />
      </Modal>
    </>
  );
};

function Matrix({ values }: { values: CalibrationMatrix }) {
  return (
    <div className="grid grid-cols-3 gap-x-4 gap-y-2 border-x-2 border-neutral-500 px-4 py-2 text-right">
      {values.map((value, index) => (
        <span key={index}>{formatMatrixValue(value)}</span>
      ))}
    </div>
  );
}

function ColumnVector({ values }: { values: string[] }) {
  return (
    <div className="grid grid-cols-1 gap-y-2 border-x-2 border-neutral-500 px-3 py-2 text-center">
      {values.map((value) => (
        <span key={value}>{value}</span>
      ))}
    </div>
  );
}

function formatMatrixValue(value: number) {
  const normalized = Math.abs(value) < 5e-7 ? 0 : value;
  return normalized.toFixed(6);
}

function formatMatrixForSharing(matrix: CalibrationMatrix) {
  const rows = [matrix.slice(0, 3), matrix.slice(3, 6), matrix.slice(6, 9)];
  return `H = [\n${rows.map((row) => `  [${row.map(formatMatrixValue).join(', ')}]`).join(',\n')}\n]`;
}
