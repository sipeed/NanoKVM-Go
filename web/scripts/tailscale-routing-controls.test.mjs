import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

const typesSource = source('src/pages/desktop/menu/settings/remote-network/components/tailscale/types.ts');
const apiSource = source('src/api/extensions/tailscale.ts');
const deviceSource = source(
  'src/pages/desktop/menu/settings/remote-network/components/tailscale/device.tsx'
);
const pageSource = source(
  'src/pages/desktop/menu/settings/remote-network/components/tailscale/view.tsx'
);
const authSource = source('src/api/auth.ts');
const protectionSource = source('src/lib/protection-challenge.ts');

function functionSlice(name, nextMarker) {
  const start = deviceSource.indexOf(`async function ${name}`);
  const end = deviceSource.indexOf(nextMarker, start);
  assert.notEqual(start, -1, `${name} must exist`);
  assert.notEqual(end, -1, `${name} must have a stable end marker`);
  return deviceSource.slice(start, end);
}

test('Tailscale status exposes the authoritative routing state', () => {
  for (const field of [
    'routingAvailable: boolean',
    'advertiseExitNode: boolean',
    'advertiseSubnetRoute: boolean',
    'lanSubnet: string'
  ]) {
    assert.match(typesSource, new RegExp(field));
  }
});

test('routing API helpers send enabled state and optional challenges to fixed endpoints', () => {
  assert.match(apiSource, /challenge \? \{ enabled, challenge \} : \{ enabled \}/);
  assert.match(apiSource, /'\/api\/extensions\/tailscale\/exit-node'/);
  assert.match(apiSource, /'\/api\/extensions\/tailscale\/subnet-route'/);
});

test('both routing switches use Tailscale protection and refresh authoritative status', () => {
  const exitNodeFunction = functionSlice('updateExitNode', 'async function updateSubnetRoute');
  const subnetRouteFunction = functionSlice('updateSubnetRoute', '\n\n  return (');

  assert.match(
    deviceSource,
    /runWithChallenge\('tailscale_set_exit_node', 'tailscale', async \(challenge\) =>/
  );
  assert.match(
    deviceSource,
    /runWithChallenge\('tailscale_set_subnet_route', 'tailscale', async \(challenge\) =>/
  );
  assert.match(deviceSource, /checked=\{status\.advertiseExitNode\}/);
  assert.match(deviceSource, /checked=\{status\.advertiseSubnetRoute\}/);
  assert.match(deviceSource, /disabled=\{isMutationBusy \|\| !status\.routingAvailable\}/);
  assert.match(exitNodeFunction, /await onStatusRefresh\(\);/);
  assert.match(subnetRouteFunction, /await onStatusRefresh\(\);/);
  assert.match(
    deviceSource,
    /setExitNodeErrMsg\(t\('settings\.tailscale\.routingUpdateFailed'\)\)/
  );
  assert.match(
    deviceSource,
    /setSubnetRouteErrMsg\(t\('settings\.tailscale\.routingUpdateFailed'\)\)/
  );
});

test('overlapping status requests queue an authoritative follow-up refresh', () => {
  assert.match(pageSource, /const statusRefreshQueuedRef = useRef\(false\)/);
  assert.match(
    pageSource,
    /if \(statusRequestInFlightRef\.current\) \{\s*statusRefreshQueuedRef\.current = true;\s*return;/
  );
  assert.match(pageSource, /while \(isMountedRef\.current && statusRefreshQueuedRef\.current\)/);
});

test('routing challenge activity is shown on the initiating switch', () => {
  assert.match(deviceSource, /setGateTarget\('exit-node'\)/);
  assert.match(deviceSource, /setGateTarget\('subnet-route'\)/);
  assert.match(
    deviceSource,
    /isExitNodeUpdating \|\| \(isGateBusy && gateTarget === 'exit-node'\)/
  );
  assert.match(
    deviceSource,
    /isSubnetRouteUpdating \|\| \(isGateBusy && gateTarget === 'subnet-route'\)/
  );
});

test('routing actions are valid frontend sensitive-action names', () => {
  for (const action of ['tailscale_set_exit_node', 'tailscale_set_subnet_route']) {
    assert.match(authSource, new RegExp(`\\| '${action}'`));
    assert.match(protectionSource, new RegExp(`\\| '${action}'`));
  }
});
