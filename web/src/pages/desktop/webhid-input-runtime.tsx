import { useEffect } from 'react';

import { webHIDInput } from '@/lib/webhid-input.ts';

export const WebHIDInputRuntime = () => {
  useEffect(() => {
    let disposed = false;
    let connecting = false;
    let suspended = false;
    let profileKey = registeredProfileKey();

    const connectGranted = async () => {
      if (disposed || suspended || connecting || !webHIDInput.available) return;
      connecting = true;
      try {
        await webHIDInput.connectGrantedDevices();
      } catch (error) {
        console.warn('[WebHID] Unable to reconnect granted input devices', error);
      } finally {
        connecting = false;
      }
    };
    const handlePageHide = () => {
      suspended = true;
      void webHIDInput.disconnectAll();
    };
    const handlePageShow = () => {
      suspended = false;
      void connectGranted();
    };
    const unsubscribe = webHIDInput.subscribe(() => {
      const nextProfileKey = registeredProfileKey();
      if (nextProfileKey === profileKey) return;
      profileKey = nextProfileKey;
      void connectGranted();
    });

    void connectGranted();
    window.addEventListener('pagehide', handlePageHide);
    window.addEventListener('pageshow', handlePageShow);
    return () => {
      disposed = true;
      unsubscribe();
      window.removeEventListener('pagehide', handlePageHide);
      window.removeEventListener('pageshow', handlePageShow);
      void webHIDInput.disconnectAll();
    };
  }, []);

  return null;
};

function registeredProfileKey() {
  return webHIDInput.registeredProfiles.map((profile) => profile.id).join('\u0000');
}
