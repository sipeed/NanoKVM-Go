import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import path from 'node:path';
import test from 'node:test';
import { promisify } from 'node:util';

const run = promisify(execFile);
const probe = path.resolve(
  import.meta.dirname,
  '../../support/tools/input-router-linux-uhid-probe.py'
);

test('Linux UHID probe extracts all independent HID2 profiles from usbdev.sh', async () => {
  const { stdout } = await run('python3', [probe, '--verify-descriptors-only']);
  const evidence = JSON.parse(stdout);

  assert.equal(evidence.schema, 'nanokvm-input-router-linux-uhid/v2');
  assert.equal(evidence.event_probe_skipped, true);
  assert.equal(evidence.pass, true);
  assert.deepEqual(evidence.descriptor_profiles, {
    absolute: { descriptor_length: 65, report_length: 7 },
    'digitizer-stylus': { descriptor_length: 69, report_length: 7 },
    'touch-screen': { descriptor_length: 589, report_length: 62 }
  });
});
