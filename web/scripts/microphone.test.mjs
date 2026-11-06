import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

function loadTypeScriptModule(file, globals = {}) {
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, module, ...globals }, { filename: file });
  return module.exports;
}

const microphoneFile = path.resolve(import.meta.dirname, '../src/lib/microphone.ts');
const microphone = loadTypeScriptModule(microphoneFile);
const virtualAudioSettingSource = fs.readFileSync(
  path.resolve(
    import.meta.dirname,
    '../src/pages/desktop/menu/settings/device/virtual-devices/microphone.tsx'
  ),
  'utf8'
);
const microphoneContextSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/pages/desktop/microphone-context.tsx'),
  'utf8'
);
const desktopMenuSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/pages/desktop/menu/index.tsx'),
  'utf8'
);
const virtualDeviceApiSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/api/virtual-device.ts'),
  'utf8'
);
const countdownConfirmationSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/hooks/useCountdownConfirmation.tsx'),
  'utf8'
);
const zhLocale = loadTypeScriptModule(
  path.resolve(import.meta.dirname, '../src/i18n/locales/zh.ts')
).default;

test('top volume and microphone controls are rendered only in WebRTC modes', () => {
  assert.match(
    desktopMenuSource,
    /const isWebRTCVideoMode = videoMode === 'h264-webrtc' \|\| videoMode === 'h265-webrtc';/
  );
  assert.match(desktopMenuSource, /const showVolume = isEnabled\('volume'\) && isWebRTCVideoMode;/);
  assert.match(
    desktopMenuSource,
    /const showMicrophone = isEnabled\('microphone'\) && isWebRTCVideoMode;/
  );
});

test('virtual audio changes require the shared five-second confirmation', () => {
  assert.match(virtualAudioSettingSource, /useCountdownConfirmation/);
  assert.match(
    virtualAudioSettingSource,
    /confirmVirtualAudioUpdate\(\(\)\s*=>\s*update\(nextMode\)\)/
  );
  assert.match(virtualAudioSettingSource, /onChange=/);
  assert.doesNotMatch(virtualAudioSettingSource, /onChange=\{update\}/);
  assert.match(virtualAudioSettingSource, /\{contextHolder\}/);
  assert.doesNotMatch(virtualAudioSettingSource, /settings\.screen\.edid/);
  assert.match(countdownConfirmationSource, /const UPDATE_COUNTDOWN_SECONDS = 5;/);
});

test('virtual audio uses an explicit typed three-mode request', () => {
  assert.match(
    virtualDeviceApiSource,
    /export type VirtualAudioMode = 'off' \| 'mic' \| 'duplex';/
  );
  assert.match(virtualDeviceApiSource, /updateVirtualAudioMode\(mode: VirtualAudioMode\)/);
  assert.match(virtualDeviceApiSource, /device: 'audio',\s*mode/);
  assert.match(
    microphoneContextSource,
    /current\.data\.audioMode === 'off' \? 'mic' : current\.data\.audioMode/
  );
  assert.match(microphoneContextSource, /isVirtualAudioRuntimeReady/);
  assert.match(virtualAudioSettingSource, /isVirtualAudioRuntimeReady/);
  assert.match(microphoneContextSource, /updateVirtualAudioMode\(targetMode\)/);
  assert.doesNotMatch(microphoneContextSource, /updateVirtualDevice\('mic'\)/);
  assert.match(virtualDeviceApiSource, /VIRTUAL_AUDIO_MODE_CHANGED_EVENT/);
  assert.match(virtualDeviceApiSource, /window\.dispatchEvent/);
});

test('virtual audio setting rechecks the marker-backed mode and stops browser capture', () => {
  assert.match(virtualAudioSettingSource, /audioMode: VirtualAudioMode/);
  assert.match(virtualAudioSettingSource, /<Segmented<VirtualAudioMode>/);
  assert.match(virtualAudioSettingSource, /value=\{audioMode\}/);
  assert.match(virtualAudioSettingSource, /repairCurrentMode/);
  assert.match(virtualAudioSettingSource, /await api\.getVirtualDevice\(\)/);
  assert.match(virtualAudioSettingSource, /current\.data\.audioMode === nextMode/);
  assert.match(virtualAudioSettingSource, /statusRef\.current !== 'disabled'/);
  assert.match(virtualAudioSettingSource, /await disable\(\)/);
  assert.match(virtualAudioSettingSource, /setTimeout\(resolve, 1000\)/);
  assert.match(virtualAudioSettingSource, /finally \{\s*try \{\s*await getVirtualDevices\(\)/);
  assert.doesNotMatch(virtualAudioSettingSource, /\benable\(/);
});

test('virtual audio confirmation is limited to speaker-state changes', () => {
  assert.match(
    virtualAudioSettingSource,
    /const speakerStateChanges = \(audioMode === 'duplex'\) !== \(nextMode === 'duplex'\)/
  );
  assert.match(virtualAudioSettingSource, /if \(speakerStateChanges\)/);
  assert.match(virtualAudioSettingSource, /onChange=\{requestUpdate\}/);
  assert.match(virtualAudioSettingSource, /void update\(nextMode\)/);
});

test('microphone disable stops capture without detaching the reusable sender track', () => {
  assert.match(microphoneContextSource, /clearCapture\(stream, track\);/);
  assert.match(microphoneContextSource, /transport\.sender\.replaceTrack\(track\)/);
  assert.doesNotMatch(microphoneContextSource, /replaceTrack\(null\)/);
  assert.doesNotMatch(microphoneContextSource, /detachTrack/);
});

test('virtual audio confirmation uses the approved Chinese wording', () => {
  const labels = zhLocale.translation.settings.device.virtualAudioConfirm;

  assert.equal(labels.title, '切换虚拟音频前请确认');
  assert.equal(
    labels.description,
    '请先接入 PD 电源。切换虚拟音频可能导致显示短暂断开；如果切换后设备无法正确识别显示器，请手动重启设备。'
  );
  assert.equal(labels.ok, '继续切换');
  assert.equal(labels.okCountdown, '继续切换（{{seconds}}）');
  assert.equal(labels.cancel, '取消');
});

test('every locale keeps virtual audio wording and countdown interpolation', () => {
  const localesDir = path.resolve(import.meta.dirname, '../src/i18n/locales');
  const localeFiles = fs.readdirSync(localesDir).filter((file) => file.endsWith('.ts'));

  for (const localeFile of localeFiles) {
    const locale = loadTypeScriptModule(path.join(localesDir, localeFile)).default;
    const labels = locale.translation.settings.device.virtualAudioConfirm;

    assert.ok(labels, `${localeFile} is missing virtualAudioConfirm`);
    assert.match(
      labels.okCountdown,
      /\{\{seconds\}\}/,
      `${localeFile} lost countdown interpolation`
    );
    for (const value of Object.values(labels)) {
      assert.doesNotMatch(value, /EDID/i, `${localeFile} exposes EDID in virtual audio copy`);
    }
  }
});

test('maps browser microphone error names to translation keys', () => {
  const cases = [
    ['NotFoundError', 'microphone.noDevice'],
    ['DevicesNotFoundError', 'microphone.noDevice'],
    ['NotAllowedError', 'microphone.permissionDenied'],
    ['PermissionDeniedError', 'microphone.permissionDenied'],
    ['NotReadableError', 'microphone.notReadable'],
    ['TrackStartError', 'microphone.notReadable'],
    ['OverconstrainedError', 'microphone.overconstrained'],
    ['ConstraintNotSatisfiedError', 'microphone.overconstrained'],
    ['SecurityError', 'microphone.securityBlocked'],
    ['AbortError', 'microphone.aborted'],
    ['NotSupportedError', 'microphone.unsupported']
  ];

  for (const [name, expected] of cases) {
    assert.equal(
      microphone.getMicrophoneErrorTranslationKey({ name }),
      expected,
      `unexpected translation key for ${name}`
    );
  }
});

test('preserves explicit microphone errors and falls back for unknown errors', () => {
  const explicit = new microphone.MicrophoneError('microphone.securityBlocked');
  assert.equal(microphone.getMicrophoneErrorTranslationKey(explicit), 'microphone.securityBlocked');

  for (const error of [null, 'failed', {}, { name: 123 }, { name: 'UnknownError' }]) {
    assert.equal(microphone.getMicrophoneErrorTranslationKey(error), 'microphone.enableFailed');
  }
});

test('returns null when an audio stream has no usable track', () => {
  assert.equal(
    microphone.getUsableAudioTrack({
      getAudioTracks: () => []
    }),
    null
  );

  assert.equal(
    microphone.getUsableAudioTrack({
      getAudioTracks: () => [{ readyState: 'ended' }, { readyState: 'ended' }]
    }),
    null
  );
});

test('returns the first audio track that has not ended', () => {
  const usableTrack = { readyState: 'live' };
  const laterTrack = { readyState: 'live' };

  assert.equal(
    microphone.getUsableAudioTrack({
      getAudioTracks: () => [{ readyState: 'ended' }, usableTrack, laterTrack]
    }),
    usableTrack
  );
});

test('stopMediaStream clears handlers and stops every track', () => {
  const stoppedWithHandlers = [];
  const tracks = [
    {
      onended: () => {},
      stop() {
        stoppedWithHandlers.push(this.onended);
      }
    },
    {
      onended: () => {},
      stop() {
        stoppedWithHandlers.push(this.onended);
      }
    }
  ];

  assert.doesNotThrow(() => microphone.stopMediaStream(null));
  microphone.stopMediaStream({ getTracks: () => tracks });

  assert.deepEqual(stoppedWithHandlers, [null, null]);
  assert.equal(tracks[0].onended, null);
  assert.equal(tracks[1].onended, null);
});

test('requestControlMicrophone rejects unsupported or insecure environments', async () => {
  const insecureMicrophone = loadTypeScriptModule(microphoneFile, {
    navigator: {},
    window: { isSecureContext: false }
  });
  await assert.rejects(
    insecureMicrophone.requestControlMicrophone(),
    (error) => error.translationKey === 'microphone.securityBlocked'
  );

  const unsupportedMicrophone = loadTypeScriptModule(microphoneFile, {
    navigator: {},
    window: { isSecureContext: true }
  });
  await assert.rejects(
    unsupportedMicrophone.requestControlMicrophone(),
    (error) => error.translationKey === 'microphone.unsupported'
  );
});

test('requestControlMicrophone passes constraints through and returns a live track', async () => {
  let receivedConstraints;
  const track = { readyState: 'live' };
  const stream = {
    getAudioTracks: () => [track],
    getTracks: () => [track]
  };
  const requestMicrophone = loadTypeScriptModule(microphoneFile, {
    navigator: {
      mediaDevices: {
        async getUserMedia(constraints) {
          receivedConstraints = constraints;
          return stream;
        }
      }
    },
    window: { isSecureContext: true }
  });

  const capture = await requestMicrophone.requestControlMicrophone();

  assert.equal(capture.stream, stream);
  assert.equal(capture.track, track);
  assert.equal(receivedConstraints.audio, requestMicrophone.controlMicrophoneConstraints);
});

test('requestControlMicrophone stops an empty stream before reporting no device', async () => {
  let stopped = false;
  const emptyStream = {
    getAudioTracks: () => [],
    getTracks: () => [
      {
        onended: () => {},
        stop() {
          stopped = true;
          assert.equal(this.onended, null);
        }
      }
    ]
  };
  const requestMicrophone = loadTypeScriptModule(microphoneFile, {
    navigator: {
      mediaDevices: {
        async getUserMedia() {
          return emptyStream;
        }
      }
    },
    window: { isSecureContext: true }
  });

  await assert.rejects(
    requestMicrophone.requestControlMicrophone(),
    (error) => error.translationKey === 'microphone.noDevice'
  );
  assert.equal(stopped, true);
});
