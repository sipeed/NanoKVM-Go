import { useEffect, useRef, useState } from 'react';
import { AppleOutlined, WindowsOutlined } from '@ant-design/icons';
import clsx from 'clsx';
import { useAtom, useAtomValue } from 'jotai';
import { XIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { KeyboardReact as Keyboard, type KeyboardButtonTheme } from 'react-simple-keyboard';
import { Drawer } from 'vaul';

import 'react-simple-keyboard/build/css/index.css';
import '@/assets/styles/keyboard.css';

import { ConfigProvider, Segmented, Select, theme } from 'antd';
import { useMediaQuery } from 'react-responsive';

import languageDisplayNames from '@/i18n/languages.ts';
import { getAntdLocale } from '@/i18n/locale.ts';
import { inputRouter, InputSource } from '@/lib/input-router.ts';
import { getKeycode, getModifierBit } from '@/lib/keymap.ts';
import { LAYER_VIRTUAL_KEYBOARD } from '@/lib/layers.ts';
import * as storage from '@/lib/localstorage.ts';
import { isKeyboardOpenAtom } from '@/jotai/keyboard.ts';
import { isPreviewModeAtom } from '@/jotai/preview.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import {
  doubleKeys,
  keyboardArrowsOptions,
  keyboardControlPadOptions,
  keyboardOptions,
  modifierKeys,
  numericKeyboardOptions,
  specialKeyMap
} from './virtual-keys.ts';

type KeyboardSystem = 'win' | 'mac';
type KeyboardChoice = 'numeric' | KeyboardSystem;

export const VirtualKeyboard = () => {
  const { i18n } = useTranslation();
  const isBigScreen = useMediaQuery({ minWidth: 850 });

  const [isKeyboardOpen, setIsKeyboardOpen] = useAtom(isKeyboardOpenAtom);
  const isPreviewMode = useAtomValue(isPreviewModeAtom);

  const [keyboardLayout, setKeyboardLayout] = useState('default');
  const [keyboardSystem, setKeyboardSystem] = useState<KeyboardSystem>('win');
  const [keyboardLanguage, setKeyboardLanguage] = useState('en');
  const [activeModifierKeys, setActiveModifierKeys] = useState<string[]>([]);
  const [keyboardChoice, setKeyboardChoice] = useState<KeyboardChoice>(
    isBigScreen ? 'win' : 'numeric'
  );

  const keyboardRef = useRef<any>(null);
  const wasKeyboardOpen = useRef(false);

  const systems = [
    { value: 'win', icon: <WindowsOutlined /> },
    { value: 'mac', icon: <AppleOutlined /> }
  ];

  const mobileKeyboardChoices = [{ value: 'numeric', label: '123' }, ...systems];

  const languages = (['en', 'fr', 'ru'] as const).map((value) => ({
    value,
    label: languageDisplayNames.find((language) => language.key === value)?.name ?? value
  }));

  useEffect(() => {
    const system = storage.getKeyboardSystem();
    if (system === 'win' || system === 'mac') {
      setKeyboardSystem(system);
    }

    const language = storage.getKeyboardLanguage();
    if (language && languages.some((lng) => lng.value === language)) {
      setKeyboardLanguage(language);
    }
  }, []);

  useEffect(() => {
    const layoutMap = new Map([
      ['en', 'default'],
      ['ru', 'rus'],
      ['fr', 'azerty']
    ]);

    if (keyboardLanguage === 'en' && keyboardSystem === 'mac') {
      setKeyboardLayout('mac');
      return;
    }

    if (layoutMap.has(keyboardLanguage)) {
      setKeyboardLayout(layoutMap.get(keyboardLanguage)!);
      return;
    }

    setKeyboardLayout('default');
  }, [keyboardSystem, keyboardLanguage]);

  useEffect(() => {
    if (isKeyboardOpen && !wasKeyboardOpen.current) {
      setKeyboardChoice(isBigScreen ? keyboardSystem : 'numeric');
    } else if (!isKeyboardOpen && wasKeyboardOpen.current) {
      releaseKeyboardState();
    }

    wasKeyboardOpen.current = isKeyboardOpen;
  }, [isBigScreen, isKeyboardOpen, keyboardSystem]);

  useEffect(() => {
    if (isPreviewMode && isKeyboardOpen) setIsKeyboardOpen(false);
  }, [isKeyboardOpen, isPreviewMode, setIsKeyboardOpen]);

  useEffect(() => {
    return () => {
      if (wasKeyboardOpen.current) {
        inputRouter.sendKeyboard(InputSource.VirtualKeyboard, new Uint8Array(8));
      }
    };
  }, []);

  const isNumericKeyboard = keyboardChoice === 'numeric';

  // Press key
  function onKeyPress(key: string) {
    if (isPreviewMode) {
      return;
    }

    if (modifierKeys.includes(key)) {
      if (activeModifierKeys.includes(key)) {
        sendModifierKeyDown();
        sendModifierKeyUp();
      } else {
        setActiveModifierKeys([...activeModifierKeys, key]);
      }
      return;
    }

    sendKeydown(key);
  }

  // Release key
  function onKeyReleased(key: string) {
    if (isPreviewMode) {
      return;
    }

    if (modifierKeys.includes(key)) {
      return;
    }

    sendKeyup();
  }

  // Send all keys
  function sendKeydown(key: string) {
    if (isPreviewMode) {
      return;
    }

    const code = getKeyboardCode(key);
    if (!code) {
      console.log('unknown code: ', key);
      return;
    }

    const modifier = sendModifierKeyDown();

    send(modifier, code);
  }

  function getKeyboardCode(key: string) {
    // AZERTY: swap A↔Q and Z↔W on French physical positions
    if (keyboardLanguage === 'fr' && key.endsWith('_azerty')) {
      const base = key.replace('_azerty', '');
      if (base === 'KeyA') return getKeycode('KeyQ');
      if (base === 'KeyQ') return getKeycode('KeyA');
      if (base === 'KeyZ') return getKeycode('KeyW');
      if (base === 'KeyW') return getKeycode('KeyZ');
      // all other labels use their own code
      return getKeycode(base);
    }

    const specialKey = specialKeyMap.get(key);
    if (specialKey) {
      return getKeycode(specialKey);
    }

    return getKeycode(key);
  }

  // Release all keys
  function sendKeyup() {
    if (isPreviewMode) {
      return;
    }

    sendModifierKeyUp();
    send(0, 0);
  }

  function releaseKeyboardState() {
    inputRouter.sendKeyboard(InputSource.VirtualKeyboard, new Uint8Array(8));
    setActiveModifierKeys([]);
  }

  // Send modifier keys
  function sendModifierKeyDown() {
    if (isPreviewMode) {
      return 0;
    }

    let modifier = 0;

    activeModifierKeys.forEach((modifierKey) => {
      const key = specialKeyMap.get(modifierKey)!;

      modifier |= getModifierBit(key)!;
      const code = getKeycode(key)!;

      send(modifier, code);
    });

    return modifier;
  }

  // Release modifier keys
  function sendModifierKeyUp() {
    if (isPreviewMode) {
      return;
    }

    if (activeModifierKeys.length === 0) return;

    activeModifierKeys.forEach(() => {
      send(0, 0);
    });

    setActiveModifierKeys([]);
  }

  function send(modifier: number, code: number) {
    if (isPreviewMode) {
      return;
    }

    inputRouter.sendKeyboard(
      InputSource.VirtualKeyboard,
      new Uint8Array([modifier, 0, code, 0, 0, 0, 0, 0])
    );
  }

  function selectSystem(system: string | number) {
    if (system !== 'win' && system !== 'mac') {
      return;
    }

    setKeyboardSystem(system);
    storage.setKeyboardSystem(system);
  }

  function selectMobileKeyboard(choice: string | number) {
    switch (choice) {
      case 'numeric':
        setKeyboardChoice(choice);
        return;
      case 'win':
      case 'mac':
        setKeyboardChoice(choice);
        selectSystem(choice);
        return;
    }
  }

  function selectLanguage(language: string) {
    setKeyboardLanguage(language);
    storage.setKeyboardLanguage(language);
  }

  function getButtonTheme(): KeyboardButtonTheme[] {
    const theme = [{ class: 'hg-double', buttons: doubleKeys.join(' ') }];

    if (activeModifierKeys.length > 0) {
      const buttons = activeModifierKeys.join(' ');
      theme.push({ class: 'hg-highlight', buttons });
    }

    return theme;
  }

  return (
    <Drawer.Root open={isKeyboardOpen} onOpenChange={setIsKeyboardOpen} modal={false}>
      <Drawer.Portal>
        <Drawer.Content
          className={clsx(
            'fixed bottom-0 left-0 right-0 mx-auto overflow-hidden rounded bg-white outline-none',
            isBigScreen ? 'w-[820px]' : 'w-full max-w-[650px]'
          )}
          style={{ zIndex: LAYER_VIRTUAL_KEYBOARD }}
        >
          <Drawer.Title className="m-0">
            <div className="flex items-center justify-between px-3 py-2">
              <ConfigProvider
                locale={getAntdLocale(i18n.resolvedLanguage || i18n.language)}
                theme={{
                  algorithm: theme.defaultAlgorithm
                }}
              >
                <div className="flex items-center space-x-5">
                  <Select
                    size="small"
                    style={{ minWidth: 90 }}
                    defaultValue={keyboardLanguage}
                    options={languages}
                    onChange={selectLanguage}
                  />

                  {keyboardLanguage === 'en' && (
                    <Segmented
                      size="small"
                      options={isBigScreen ? systems : mobileKeyboardChoices}
                      value={isBigScreen ? keyboardSystem : keyboardChoice}
                      onChange={isBigScreen ? selectSystem : selectMobileKeyboard}
                    />
                  )}
                </div>
              </ConfigProvider>

              <div className="flex items-center justify-end space-x-2">
                {isNumericKeyboard && keyboardLanguage !== 'en' && (
                  <ButtonBase
                    className="flex h-[24px] min-w-10 items-center justify-center rounded px-2 text-sm font-medium text-neutral-600 hover:bg-neutral-300 hover:text-white"
                    onClick={() => setKeyboardChoice(keyboardSystem)}
                  >
                    ABC
                  </ButtonBase>
                )}
                <div
                  className="flex h-[20px] w-[20px] cursor-pointer items-center justify-center rounded text-neutral-600 hover:bg-neutral-300 hover:text-white"
                  onClick={() => setIsKeyboardOpen(false)}
                >
                  <XIcon size={18} />
                </div>
              </div>
            </div>
          </Drawer.Title>

          <div className="h-px flex-shrink-0 border-b bg-neutral-300" />

          <div data-vaul-no-drag className="keyboardContainer w-full">
            {isNumericKeyboard ? (
              <Keyboard
                key="numeric"
                layoutName="numeric"
                onKeyPress={onKeyPress}
                onKeyReleased={onKeyReleased}
                {...numericKeyboardOptions}
              />
            ) : (
              <>
                {/* main keyboard */}
                <Keyboard
                  key="full"
                  buttonTheme={getButtonTheme()}
                  keyboardRef={(r) => (keyboardRef.current = r)}
                  onKeyPress={onKeyPress}
                  onKeyReleased={onKeyReleased}
                  layoutName={keyboardLayout}
                  {...keyboardOptions}
                />

                {/* control keyboard */}
                {isBigScreen && (
                  <div className="controlArrows">
                    <Keyboard
                      onKeyPress={onKeyPress}
                      onKeyReleased={onKeyReleased}
                      {...keyboardControlPadOptions}
                    />

                    <Keyboard
                      onKeyPress={onKeyPress}
                      onKeyReleased={onKeyReleased}
                      {...keyboardArrowsOptions}
                    />
                  </div>
                )}
              </>
            )}
          </div>
        </Drawer.Content>
        <Drawer.Overlay />
      </Drawer.Portal>
    </Drawer.Root>
  );
};
