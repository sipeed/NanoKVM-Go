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

  vm.runInNewContext(
    output,
    { console, ...globals, exports: module.exports, module },
    { filename: file }
  );
  return module.exports;
}

function createDeferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });

  return { promise, reject, resolve };
}

function createError(name, message) {
  const error = new Error(message);
  error.name = name;
  return error;
}

const audioVolumeFile = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/screen/audio-volume.ts'
);

function createAudioVolumeModule(moduleConsole = console) {
  return loadTypeScriptModule(audioVolumeFile, { console: moduleConsole });
}

test('returns not-ready when there is no audio element', async () => {
  const audioVolume = createAudioVolumeModule();

  assert.equal(await audioVolume.synchronizeAudioVolume(null, 37), 'not-ready');
});

test('returns not-ready without playing until a stream is available', async () => {
  const audioVolume = createAudioVolumeModule();
  let playCalls = 0;
  const audio = {
    muted: true,
    srcObject: null,
    volume: 1,
    play() {
      playCalls += 1;
      return Promise.resolve();
    }
  };

  const result = await audioVolume.synchronizeAudioVolume(audio, 37);

  assert.equal(result, 'not-ready');
  assert.equal(audio.volume, 0.37);
  assert.equal(audio.muted, true);
  assert.equal(playCalls, 0);
});

test('keeps zero volume muted without starting playback', async () => {
  const audioVolume = createAudioVolumeModule();
  let playCalls = 0;
  const audio = {
    muted: false,
    srcObject: {},
    volume: 0.37,
    play() {
      playCalls += 1;
      return Promise.resolve();
    }
  };

  const result = await audioVolume.synchronizeAudioVolume(audio, 0);

  assert.equal(result, 'muted');
  assert.equal(audio.volume, 0);
  assert.equal(audio.muted, true);
  assert.equal(playCalls, 0);
});

test('plays an available stream at the requested volume', async () => {
  const audioVolume = createAudioVolumeModule();
  let playCalls = 0;
  const audio = {
    muted: true,
    srcObject: {},
    volume: 1,
    play() {
      playCalls += 1;
      return Promise.resolve();
    }
  };

  const result = await audioVolume.synchronizeAudioVolume(audio, 37);

  assert.equal(result, 'playing');
  assert.equal(audio.volume, 0.37);
  assert.equal(audio.muted, false);
  assert.equal(playCalls, 1);
});

test('reports blocked autoplay, mutes audio, and does not install a retry', async () => {
  const audioVolume = createAudioVolumeModule();
  let playCalls = 0;
  const audio = {
    muted: true,
    srcObject: {},
    volume: 1,
    play() {
      playCalls += 1;
      return Promise.reject(createError('NotAllowedError', 'autoplay blocked'));
    }
  };

  const result = await audioVolume.synchronizeAudioVolume(audio, 37);

  assert.equal(result, 'blocked');
  assert.equal(audio.volume, 0.37);
  assert.equal(audio.muted, true);
  assert.equal(playCalls, 1);
});

test('reports and logs playback errors other than autoplay blocking', async () => {
  const errors = [];
  const audioVolume = createAudioVolumeModule({ error: (error) => errors.push(error) });
  const playbackError = createError('NotSupportedError', 'unsupported stream');
  const audio = {
    muted: true,
    srcObject: {},
    volume: 1,
    play() {
      return Promise.reject(playbackError);
    }
  };

  const result = await audioVolume.synchronizeAudioVolume(audio, 37);

  assert.equal(result, 'error');
  assert.equal(audio.muted, false);
  assert.deepEqual(errors, [playbackError]);
});

test('attaches a stream before synchronizing playback', async () => {
  const audioVolume = createAudioVolumeModule();
  let playCalls = 0;
  const audio = {
    muted: true,
    srcObject: null,
    volume: 1,
    play() {
      playCalls += 1;
      return Promise.resolve();
    }
  };
  const stream = {};

  assert.equal(await audioVolume.attachAudioStream(null, stream, 37), 'not-ready');
  assert.equal(await audioVolume.attachAudioStream(audio, stream, 37), 'playing');
  assert.equal(audio.srcObject, stream);
  assert.equal(audio.volume, 0.37);
  assert.equal(audio.muted, false);
  assert.equal(playCalls, 1);
});

test('a stale autoplay rejection cannot mute newer playback', async () => {
  const audioVolume = createAudioVolumeModule();
  const firstPlay = createDeferred();
  let playCalls = 0;
  const audio = {
    muted: true,
    srcObject: {},
    volume: 1,
    play() {
      playCalls += 1;
      return playCalls === 1 ? firstPlay.promise : Promise.resolve();
    }
  };

  const firstSynchronization = audioVolume.synchronizeAudioVolume(audio, 37);
  const latestResult = await audioVolume.synchronizeAudioVolume(audio, 50);
  firstPlay.reject(createError('NotAllowedError', 'old autoplay attempt blocked'));

  assert.equal(latestResult, 'playing');
  assert.equal(await firstSynchronization, 'stale');
  assert.equal(audio.volume, 0.5);
  assert.equal(audio.muted, false);
  assert.equal(playCalls, 2);
});

test('a playback rejection arriving after disposal is stale', async () => {
  const audioVolume = createAudioVolumeModule();
  const pendingPlay = createDeferred();
  const audio = {
    muted: true,
    srcObject: {},
    volume: 1,
    play() {
      return pendingPlay.promise;
    }
  };

  const synchronization = audioVolume.synchronizeAudioVolume(audio, 37);
  audioVolume.disposeAudioVolumeSynchronization(audio);
  pendingPlay.reject(createError('NotAllowedError', 'late autoplay rejection'));

  assert.equal(await synchronization, 'stale');
});

test('a stale successful play cannot override a newer muted state', async () => {
  const audioVolume = createAudioVolumeModule();
  const firstPlay = createDeferred();
  const audio = {
    muted: true,
    srcObject: {},
    volume: 1,
    play() {
      return firstPlay.promise;
    }
  };

  const firstSynchronization = audioVolume.synchronizeAudioVolume(audio, 37);
  assert.equal(await audioVolume.synchronizeAudioVolume(audio, 0), 'muted');
  firstPlay.resolve();

  assert.equal(await firstSynchronization, 'stale');
  assert.equal(audio.volume, 0);
  assert.equal(audio.muted, true);
});

test('autoplay blocking only clears the volume that was actually attempted', () => {
  const audioVolume = createAudioVolumeModule();

  assert.equal(audioVolume.resolveVolumeAfterAutoplayBlock(37, 37), 0);
  assert.equal(audioVolume.resolveVolumeAfterAutoplayBlock(50, 37), 50);
  assert.equal(audioVolume.resolveVolumeAfterAutoplayBlock(0, 37), 0);
});
