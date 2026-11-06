import { useEffect, useState } from 'react';
import { Switch } from 'antd';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import { Status } from './types.ts';

type DeviceProps = {
  status: Status;
  onStatusRefresh: () => void | Promise<void>;
};

export const Device = ({ status, onStatusRefresh }: DeviceProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();

  const [isRunning, setIsRunning] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [isExitNodeUpdating, setIsExitNodeUpdating] = useState(false);
  const [isSubnetRouteUpdating, setIsSubnetRouteUpdating] = useState(false);
  const [gateTarget, setGateTarget] = useState<'connection' | 'exit-node' | 'subnet-route' | null>(
    null
  );
  const [errMsg, setErrMsg] = useState('');
  const [exitNodeErrMsg, setExitNodeErrMsg] = useState('');
  const [subnetRouteErrMsg, setSubnetRouteErrMsg] = useState('');

  const isRoutingUpdating = isExitNodeUpdating || isSubnetRouteUpdating;
  const isMutationBusy = isUpdating || isRoutingUpdating || isGateBusy;

  useEffect(() => {
    setIsRunning(status.state === 'running');
  }, [status]);

  useEffect(() => {
    if (!isGateBusy) {
      setGateTarget(null);
    }
  }, [isGateBusy]);

  async function update() {
    if (isMutationBusy) return;
    setGateTarget('connection');

    if (isRunning) {
      await runWithChallenge('tailscale_down', 'tailscale', async (challenge) => {
        setIsUpdating(true);
        setErrMsg('');
        try {
          const rsp = await api.down(challenge);
          if (rsp.code === -3) {
            return rsp;
          }
          if (rsp.code !== 0) {
            console.log(rsp.msg);
            setErrMsg(t('settings.tailscale.failed'));
            return rsp;
          }
          setIsRunning(false);
          await onStatusRefresh();
          return rsp;
        } finally {
          setIsUpdating(false);
        }
      });
      return;
    }

    await runWithChallenge('tailscale_up', 'tailscale', async (challenge) => {
      setIsUpdating(true);
      setErrMsg('');
      try {
        const rsp = await api.up(challenge);
        if (rsp.code === -3) {
          return rsp;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setErrMsg(t('settings.tailscale.failed'));
          return rsp;
        }
        setIsRunning(true);
        await onStatusRefresh();
        return rsp;
      } finally {
        setIsUpdating(false);
      }
    });
  }

  async function updateExitNode(enabled: boolean) {
    if (isMutationBusy || !status.routingAvailable || enabled === status.advertiseExitNode) {
      return;
    }

    setExitNodeErrMsg('');
    setGateTarget('exit-node');
    await runWithChallenge('tailscale_set_exit_node', 'tailscale', async (challenge) => {
      setIsExitNodeUpdating(true);
      try {
        const rsp = await api.setExitNode(enabled, challenge);
        if (rsp.code === -3 || rsp.code === -4) {
          return rsp;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setExitNodeErrMsg(t('settings.tailscale.routingUpdateFailed'));
          return rsp;
        }

        await onStatusRefresh();
        return rsp;
      } catch {
        setExitNodeErrMsg(t('settings.tailscale.routingUpdateFailed'));
        return { code: -1 };
      } finally {
        setIsExitNodeUpdating(false);
      }
    });
  }

  async function updateSubnetRoute(enabled: boolean) {
    if (
      isMutationBusy ||
      !status.routingAvailable ||
      (!status.lanSubnet && !status.advertiseSubnetRoute) ||
      enabled === status.advertiseSubnetRoute
    ) {
      return;
    }

    setSubnetRouteErrMsg('');
    setGateTarget('subnet-route');
    await runWithChallenge('tailscale_set_subnet_route', 'tailscale', async (challenge) => {
      setIsSubnetRouteUpdating(true);
      try {
        const rsp = await api.setSubnetRoute(enabled, challenge);
        if (rsp.code === -3 || rsp.code === -4) {
          return rsp;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setSubnetRouteErrMsg(t('settings.tailscale.routingUpdateFailed'));
          return rsp;
        }

        await onStatusRefresh();
        return rsp;
      } catch {
        setSubnetRouteErrMsg(t('settings.tailscale.routingUpdateFailed'));
        return { code: -1 };
      } finally {
        setIsSubnetRouteUpdating(false);
      }
    });
  }

  return (
    <>
      <div className="flex flex-col space-y-7 rounded-2xl border border-solid border-neutral-700/60 bg-neutral-900/40 p-5">
        <div className="flex justify-between">
          <span>{t('settings.tailscale.enable')}</span>
          <Switch
            checked={isRunning}
            loading={isUpdating || (isGateBusy && gateTarget === 'connection')}
            disabled={isMutationBusy}
            onClick={update}
          />
        </div>

        <div className="flex flex-col space-y-2">
          <div className="flex items-center justify-between gap-4">
            <div className="flex min-w-0 flex-col space-y-1">
              <span>{t('settings.tailscale.exitNode')}</span>
              <span className="text-xs text-neutral-500">
                {t('settings.tailscale.exitNodeDesc')}
              </span>
            </div>
            <Switch
              aria-label={t('settings.tailscale.exitNode')}
              checked={status.advertiseExitNode}
              loading={isExitNodeUpdating || (isGateBusy && gateTarget === 'exit-node')}
              disabled={isMutationBusy || !status.routingAvailable}
              onChange={updateExitNode}
            />
          </div>
          {exitNodeErrMsg && (
            <span role="alert" className="text-xs text-red-500">
              {exitNodeErrMsg}
            </span>
          )}
        </div>

        <div className="flex flex-col space-y-2">
          <div className="flex items-center justify-between gap-4">
            <div className="flex min-w-0 flex-col space-y-1">
              <span>{t('settings.tailscale.subnetRoute')}</span>
              <span className="text-xs text-neutral-500">
                {status.lanSubnet
                  ? t('settings.tailscale.subnetRouteDesc', { subnet: status.lanSubnet })
                  : t('settings.tailscale.subnetUnavailable')}
              </span>
            </div>
            <Switch
              aria-label={t('settings.tailscale.subnetRoute')}
              checked={status.advertiseSubnetRoute}
              loading={isSubnetRouteUpdating || (isGateBusy && gateTarget === 'subnet-route')}
              disabled={
                isMutationBusy ||
                !status.routingAvailable ||
                (!status.lanSubnet && !status.advertiseSubnetRoute)
              }
              onChange={updateSubnetRoute}
            />
          </div>
          {subnetRouteErrMsg && (
            <span role="alert" className="text-xs text-red-500">
              {subnetRouteErrMsg}
            </span>
          )}
        </div>

        {!status.routingAvailable && (
          <div className="text-xs text-amber-500">{t('settings.tailscale.routingUnavailable')}</div>
        )}

        {status.routingAvailable && (status.advertiseExitNode || status.advertiseSubnetRoute) && (
          <div className="rounded border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-300">
            {t('settings.tailscale.pendingApproval')}
          </div>
        )}

        <div className="flex justify-between">
          <span>{t('settings.tailscale.deviceName')}</span>
          <span>{status.name}</span>
        </div>

        <div className="flex justify-between">
          <span>{t('settings.tailscale.deviceIP')}</span>
          <span>{status.ip}</span>
        </div>

        <div className="flex justify-between">
          <span>{t('settings.tailscale.account')}</span>
          <span>{status.account}</span>
        </div>

        {errMsg && <span className="text-red-500">{errMsg}</span>}
      </div>
      {modal}
    </>
  );
};
