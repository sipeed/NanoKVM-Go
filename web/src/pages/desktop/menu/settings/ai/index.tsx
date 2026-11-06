import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Divider } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { AIServices } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';

import { MCP } from './mcp';
import { MemoryFabric } from './memory-fabric';
import { ScreenTimelapse } from './screen-timelapse';

const memoryFabricServiceID = 'memoryFabric';
const screenTimelapseServiceID = 'screenTimelapse';

export const AI = () => {
  const { t } = useTranslation();
  const [aiServices, setAIServices] = useState<AIServices>();
  const [mcpEnabled, setMCPEnabled] = useState(false);
  const serviceToggleLockRef = useRef(false);
  const [isServiceToggleLocked, setIsServiceToggleLocked] = useState(false);

  const runServiceToggle = useCallback(async (action: () => Promise<void>) => {
    if (serviceToggleLockRef.current) return;

    serviceToggleLockRef.current = true;
    setIsServiceToggleLocked(true);

    try {
      await action();
    } finally {
      serviceToggleLockRef.current = false;
      setIsServiceToggleLocked(false);
    }
  }, []);

  const getAIServices = useCallback(() => {
    api
      .getAIServices()
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.loadServices'));
          return;
        }

        setAIServices(rsp.data);
      })
      .catch(() => {
        feedback.error(t('settings.ai.errors.loadServices'));
      });
  }, [t]);

  useEffect(() => {
    getAIServices();
  }, [getAIServices]);

  const supportedServiceIDs = useMemo(() => {
    return new Set(
      aiServices?.services.filter((service) => service.supported).map((service) => service.id) ?? []
    );
  }, [aiServices]);

  return (
    <>
      <div className="text-base font-bold">{t('settings.ai.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-8">
        <MCP
          isServiceToggleLocked={isServiceToggleLocked}
          runServiceToggle={runServiceToggle}
          onEnabledChange={setMCPEnabled}
        />
        {supportedServiceIDs.has(memoryFabricServiceID) && (
          <>
            <Divider className="opacity-50" />
            <MemoryFabric
              mcpEnabled={mcpEnabled}
              isServiceToggleLocked={isServiceToggleLocked}
              runServiceToggle={runServiceToggle}
            />
            <Divider className="opacity-50" />
          </>
        )}
        {supportedServiceIDs.has(screenTimelapseServiceID) && (
          <ScreenTimelapse
            isServiceToggleLocked={isServiceToggleLocked}
            runServiceToggle={runServiceToggle}
          />
        )}
      </div>
    </>
  );
};
