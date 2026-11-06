import type { ApplicationUpdatePhase, ApplicationUpdateSource } from '@/api/application.ts';

import type { MockState } from './state.ts';

type MockWebSocketClient = { send: (data: string) => void };
type MockSSEClient = {
  send: (payload: { event: 'snapshot' | 'service_changed'; data: unknown }) => void;
};

function aiServicesSnapshot(state: MockState) {
  return {
    type: 'snapshot' as const,
    revision: state.ai.serviceRevision,
    services: {
      mcp: { enabled: state.ai.mcpEnabled, source: 'mocked' },
      memoryFabric: { enabled: state.ai.memoryFabric.enabled, source: 'mocked' },
      screenTimelapse: { enabled: state.ai.screenTimelapseEnabled, source: 'mocked' }
    }
  };
}

function safelySend(client: { send: (payload: never) => void }, payload: unknown): void {
  try {
    client.send(payload as never);
  } catch {
    // A browser may close a mocked transport before the next broadcast.
  }
}

export type MockHubs = {
  addControlClient: (client: MockWebSocketClient) => void;
  removeControlClient: (client: MockWebSocketClient) => void;
  addAIClient: (client: MockSSEClient) => void;
  removeAIClient: (client: MockSSEClient) => void;
  publishAIService: (state: MockState, service: string, enabled: boolean) => void;
  publishAISnapshot: (state: MockState, client?: MockSSEClient) => void;
  publishUpdate: (state: MockState, source?: ApplicationUpdateSource) => void;
  dispose: () => void;
};

export function createMockHubs(): MockHubs {
  const controlClients = new Set<MockWebSocketClient>();
  const aiClients = new Set<MockSSEClient>();
  const timers = new Set<ReturnType<typeof setTimeout>>();

  function schedule(delay: number, callback: () => void) {
    const timer = setTimeout(() => {
      timers.delete(timer);
      callback();
    }, delay);
    timers.add(timer);
  }

  function broadcastLegacyUpdate(phase: ApplicationUpdatePhase, progress: number) {
    const step = phase === 'downloading' || phase === 'uploading' ? 'download' : 'install';
    const payload = JSON.stringify({ type: 'update', data: JSON.stringify({ step, progress }) });
    controlClients.forEach((client) => safelySend(client, payload));
  }

  return {
    addControlClient: (client) => controlClients.add(client),
    removeControlClient: (client) => controlClients.delete(client),
    addAIClient: (client) => aiClients.add(client),
    removeAIClient: (client) => aiClients.delete(client),
    publishAIService: (state, service, enabled) => {
      state.ai.serviceRevision += 1;
      const payload = {
        type: 'service_changed' as const,
        revision: state.ai.serviceRevision,
        service,
        enabled,
        source: 'mocked'
      };
      aiClients.forEach((client) =>
        safelySend(client, { event: 'service_changed', data: payload })
      );
    },
    publishAISnapshot: (state, client) => {
      const payload = { event: 'snapshot' as const, data: aiServicesSnapshot(state) };
      if (client) {
        safelySend(client, payload);
        return;
      }
      aiClients.forEach((currentClient) => safelySend(currentClient, payload));
    },
    publishUpdate: (state, source = 'online') => {
      const now = new Date(state.clock.epochMs).toISOString();
      const operationId = `mock-update-${state.application.nextOperation++}`;
      state.application.update = {
        operationId,
        source,
        version: state.application.latest,
        phase: source === 'offline' ? 'uploading' : 'downloading',
        progress: 0,
        errorCode: '',
        startedAt: now,
        updatedAt: now
      };

      const steps: Array<{ delay: number; phase: ApplicationUpdatePhase; progress: number }> =
        source === 'offline'
          ? [
              { delay: 0, phase: 'uploading', progress: 100 },
              { delay: 120, phase: 'verifying', progress: 100 },
              { delay: 240, phase: 'staging', progress: 100 },
              { delay: 360, phase: 'installing', progress: 100 },
              { delay: 480, phase: 'restarting', progress: 100 },
              { delay: 600, phase: 'succeeded', progress: 100 }
            ]
          : [
              { delay: 0, phase: 'downloading', progress: 25 },
              { delay: 120, phase: 'downloading', progress: 75 },
              { delay: 240, phase: 'verifying', progress: 100 },
              { delay: 360, phase: 'staging', progress: 100 },
              { delay: 480, phase: 'installing', progress: 100 },
              { delay: 600, phase: 'restarting', progress: 100 },
              { delay: 720, phase: 'succeeded', progress: 100 }
            ];

      steps.forEach(({ delay, phase, progress }) => {
        schedule(delay, () => {
          if (state.application.update.operationId !== operationId) return;
          state.application.update = {
            ...state.application.update,
            phase,
            progress,
            updatedAt: new Date(state.clock.epochMs + delay).toISOString()
          };
          broadcastLegacyUpdate(phase, progress);
          if (phase === 'succeeded') state.application.current = state.application.update.version;
        });
      });
    },
    dispose: () => {
      timers.forEach((timer) => clearTimeout(timer));
      timers.clear();
      controlClients.clear();
      aiClients.clear();
    }
  };
}
