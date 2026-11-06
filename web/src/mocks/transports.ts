import { sse, ws, type RequestHandler, type WebSocketHandler } from 'msw';

import type { MockHubs } from './hubs.ts';
import type { MockState } from './state.ts';

function toWebSocketOrigin(origin: string): string {
  const url = new URL(origin);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.origin;
}

function isTerminalResizeMessage(value: string): boolean {
  try {
    const message: unknown = JSON.parse(value);
    if (!message || typeof message !== 'object' || Array.isArray(message)) return false;
    const record = message as Record<string, unknown>;
    return typeof record.rows === 'number' && typeof record.cols === 'number';
  } catch {
    return false;
  }
}

export function createTransportHandlers(
  origin: string,
  state: MockState,
  hubs: MockHubs
): Array<RequestHandler | WebSocketHandler> {
  const websocketOrigin = toWebSocketOrigin(origin);
  const control = ws.link(`${websocketOrigin}/api/ws`);
  const terminal = ws.link(`${websocketOrigin}/api/vm/terminal`);

  const controlHandler = control.addEventListener('connection', ({ client }) => {
    hubs.addControlClient(client);
    client.addEventListener('close', () => hubs.removeControlClient(client));
  });

  const terminalHandler = terminal.addEventListener('connection', ({ client }) => {
    client.send('NanoKVM mocked terminal\r\nmock-admin@mock-nanokvm:~$ ');
    client.addEventListener('message', (event) => {
      if (typeof event.data !== 'string' || isTerminalResizeMessage(event.data)) return;
      client.send(event.data);
    });
  });

  return [
    controlHandler,
    terminalHandler,
    sse<{ snapshot: unknown; service_changed: unknown }>(
      `${origin}/api/ai/services/events`,
      ({ client, request }) => {
        hubs.addAIClient(client);
        hubs.publishAISnapshot(state, client);
        request.signal.addEventListener('abort', () => hubs.removeAIClient(client), { once: true });
      }
    ),
    sse<{ message: string }>(`${origin}/api/vm/time`, ({ client, request }) => {
      let closed = false;
      let timer: ReturnType<typeof setInterval> | undefined;

      function sendTime() {
        if (closed) return;
        client.send({ data: String(state.clock.epochMs) });
        state.clock.epochMs += 1000;
      }

      sendTime();
      timer = setInterval(sendTime, 1000);
      request.signal.addEventListener(
        'abort',
        () => {
          closed = true;
          if (timer) clearInterval(timer);
        },
        { once: true }
      );
    })
  ];
}
