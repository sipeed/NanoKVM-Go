import { http } from '@/lib/http.ts';
import { getBaseUrl } from '@/lib/service.ts';

export type AppEnvSpec = {
  label?: string;
  description?: string;
  required?: boolean;
  secret?: boolean;
  default?: string;
  value?: string;
  configured?: boolean;
};

export type AppManifest = {
  app_id: string;
  name: string;
  creator: string;
  create_time: string;
  version: string;
  desc?: string;
  category?: string;
  icon?: string;
  pre_script?: string;
  post_script?: string;
  env?: Record<string, AppEnvSpec>;
};

export type AppInfo = { directory: string; manifest: AppManifest };
export type AppServer = {
  id: string;
  name: string;
  url: string;
  official: boolean;
  builtin?: boolean;
};
export type CatalogApp = AppInfo & { archive: string };
type APIResponse = { code: number; msg: string; data: any };
type InstallStreamEvent = {
  type: 'log' | 'result';
  text?: string;
  code?: number;
  msg?: string;
  data?: any;
};
type StreamRequest = {
  method: 'POST';
  body: FormData | string;
  headers?: Record<string, string>;
};

const storeRequestTimeout = 12_000;

function archiveForm(file: File, id?: string, env?: Record<string, string>) {
  const form = new FormData();
  form.append('file', file);
  form.append('filename', file.name);
  if (id) form.append('id', id);
  if (env) form.append('env', JSON.stringify(env));
  return form;
}

export function listApps() {
  return http.get('/api/apps');
}

export function inspectApp(file: File) {
  return http.post('/api/apps/inspect', archiveForm(file), { timeout: 60_000 });
}

async function streamInstall(
  path: string,
  init: StreamRequest,
  onLog: (text: string) => void,
  signal?: AbortSignal
): Promise<APIResponse> {
  const response = await fetch(`${getBaseUrl('http')}${path}`, {
    ...init,
    signal,
    credentials: 'include',
    headers: { Accept: 'application/x-ndjson', ...init.headers }
  });
  if (!response.ok || !response.body) {
    throw new Error(`Install request failed (HTTP ${response.status})`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffered = '';
  let result: APIResponse | undefined;
  const consume = (line: string) => {
    if (!line.trim()) return;
    const event = JSON.parse(line) as InstallStreamEvent;
    if (event.type === 'log') onLog(event.text ?? '');
    if (event.type === 'result') {
      result = { code: event.code ?? 0, msg: event.msg ?? '', data: event.data };
    }
  };

  while (true) {
    const { done, value } = await reader.read();
    buffered += decoder.decode(value, { stream: !done });
    const lines = buffered.split('\n');
    buffered = lines.pop() ?? '';
    lines.forEach(consume);
    if (done) break;
  }
  consume(buffered);
  if (!result) throw new Error('Install stream ended without a result');
  return result;
}

export function installApp(
  file: File,
  id: string,
  env: Record<string, string>,
  onLog: (text: string) => void,
  signal?: AbortSignal
) {
  return streamInstall(
    '/api/apps/install/stream',
    { method: 'POST', body: archiveForm(file, id, env) },
    onLog,
    signal
  );
}

export function removeApp(id: string) {
  return http.delete(`/api/apps/${encodeURIComponent(id)}`);
}

export function updateAppEnv(id: string, env: Record<string, string>) {
  return http.request({
    method: 'put',
    url: `/api/apps/${encodeURIComponent(id)}/env`,
    data: env
  });
}

export function getServers() {
  return http.request({ method: 'get', url: '/api/apps/servers', timeout: 5_000 });
}

export function putServers(servers: AppServer[]) {
  return http.request({ method: 'put', url: '/api/apps/servers', data: servers });
}

export function getCatalog(serverID: string) {
  return http.request({
    method: 'get',
    url: `/api/apps/servers/${encodeURIComponent(serverID)}/catalog`,
    timeout: storeRequestTimeout
  });
}

export function installRemote(
  serverID: string,
  appID: string,
  env: Record<string, string>,
  onLog: (text: string) => void,
  signal?: AbortSignal
) {
  return streamInstall(
    '/api/apps/install/remote/stream',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ server_id: serverID, app_id: appID, env })
    },
    onLog,
    signal
  );
}

export function downloadURL(id: string) {
  return `/api/apps/${encodeURIComponent(id)}/download`;
}
