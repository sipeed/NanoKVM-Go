const en = {
  translation: {
    webcam: { title: 'Webcam' },
    storage: {
      cards: { mount: 'Disk mount', share: 'File sharing' },
      title: 'Images & File sharing',
      empty: 'No images',
      notAttached: 'Nothing mounted',
      attached: 'Mounted: {{name}}',
      eject: 'Eject',
      mount: 'Mount',
      mountMode: 'Mount mode',
      mountModeDisk: 'Mass storage',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Open file manager',
      refresh: 'Refresh',
      delete: 'Delete',
      deleteTitle: 'Delete medium',
      deleteConfirm: 'Delete {{name}}? The files inside it are lost.',
      downloadImage: 'Download the image',
      typePartition: 'Shared files',
      volumeLabel: 'Removable disk',
      groups: {
        share: 'File sharing',
        images: 'Images'
      },
      failed: 'Operation failed',
      hidOnly: 'HID-only mode: the virtual disk is unavailable',
      disabled: 'The virtual disk is disabled (Settings - Device)',
      force: {
        title: 'Take the medium back by force?',
        desc: 'The target machine is still using this medium. Taking it back re-enumerates the USB device and can lose whatever it has not written to disk yet.',
        menuHint:
          'The target machine is using this medium. Open the file manager to take it back by force.'
      },
      tips: {
        title: 'How to upload',
        usb1: 'Connect the NanoKVM to your computer via USB.',
        usb2: 'On this page, mount the shared files in mass storage mode.',
        usb3: 'Open the virtual disk on your computer and copy the image file to the root directory of the virtual disk.',
        usb4: 'Eject the disk when the copy is done: the image then shows up in the list.',
        scp1: 'Make sure the NanoKVM and your computer are on the same local network.',
        scp2: 'Open a terminal on your computer and use the SCP command to upload the image file to the /data directory on the NanoKVM.',
        scp3: 'Example: scp your-image-path root@your-nanokvm-ip:/data'
      },
      job: {
        running: 'Working on {{name}}'
      },
      enableDisk: {
        title: 'Enable the virtual disk?',
        desc: 'Mounting a medium needs the virtual disk, which is currently switched off. Enable it now? The keyboard and mouse of the target machine will disconnect for a moment.',
        menuHint: 'The virtual disk is disabled. Enable it in Settings - Device.'
      },
      add: {
        title: 'Add media',
        upload: 'Upload an ISO or IMG',
        fromUrl: 'Download from a URL',
        url: 'Image URL',
        urlRequired: 'Please enter a URL',
        onlyImages: 'Only .iso and .img files can be mounted',
        dropHint: 'Drag an ISO or IMG here, or click to choose a file',
        cdromOnlyIso: 'A CD-ROM can only carry an ISO 9660 / UDF image',
        uploadFailed: 'Upload failed',
        downloadFailed: 'Download failed',
        uploading: 'Uploading {{name}}',
        downloading: 'Downloading {{name}}',
        done: '{{name}} is ready',
        checksumFailed: 'The image does not match the SHA-256 checksum.',
        blocked:
          'The target machine is using the disk. Safely eject the virtual USB drive on the target machine, then turn off the Virtual Disk switch in Settings > Device.'
      },
      files: {
        reasonNotInitialized:
          'The virtual USB drive is not initialized. Initialize it in Settings > Device before using file sharing.',
        selectMedium: 'Select a medium to browse the files inside it.',
        reasonAttached: 'This medium is attached to the target machine.',
        reasonAttachedHint:
          'Safely eject the virtual USB drive on the target machine, then turn off the Virtual Disk switch in Settings > Device.',
        reasonNotBrowsable: 'This medium holds a filesystem the file manager cannot open.',
        reasonNoLoop: 'This device cannot mount disk images (no loop support).',
        reasonISOImage:
          'An ISO 9660 image cannot be browsed here. It can still be mounted to the target machine.',
        empty: 'This folder is empty',
        noMatch: 'No files match the search.',
        free: '{{size}} free',
        search: 'Search this folder',
        sort: {
          name: 'Name',
          size: 'Size',
          type: 'Type',
          modified: 'Modified',
          asc: 'Ascending',
          desc: 'Descending'
        },
        view: {
          list: 'List view',
          grid: 'Icon view'
        },
        folder: 'Folder',
        itemCount: 'Items: {{count}}',
        itemCount_one: '{{count}} item',
        itemCount_other: '{{count}} items',
        dropHint: 'Drop files here to upload them into this folder',
        up: 'Up to the parent folder',
        upload: 'Upload a file into the medium',
        uploadButton: 'Upload',
        newFolder: 'New folder',
        newFolderPlaceholder: 'Folder name',
        create: 'Create',
        deleteFolderTitle: 'Delete folder',
        deleteFolderConfirm:
          'Delete the folder {{name}} and everything inside it? This cannot be undone.',
        uploading: 'Uploading…',
        uploadFailed: 'Upload failed',
        download: 'Download',
        delete: 'Delete',
        deleteTitle: 'Delete file',
        deleteConfirm: 'Delete {{name}} from the medium?'
      },
      yes: 'Yes',
      no: 'No'
    },
    head: {
      storage: 'Images & File sharing',
      desktop: 'Remote Desktop',
      login: 'Login',
      changePassword: 'Change Password',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed: 'Unable to copy to the clipboard. Allow clipboard access and try again.'
    },
    auth: {
      login: 'Login',
      language: 'Language',
      placeholderUsername: 'Username',
      placeholderPassword: 'Password',
      placeholderPassword2: 'Please enter password again',
      noEmptyUsername: 'Username required',
      noEmptyPassword: 'Password required',
      noAccount: 'Failed to get user information, please refresh web page or reset password',
      invalidUser: 'Invalid username or password',
      error: 'Unexpected error',
      twoFactorTitle: 'Verification Required',
      twoFactorDesc: 'Enter your authentication code or unused recovery code to sign in.',
      sensitiveActionDesc: 'Enter an authentication code or unused recovery code to continue.',
      placeholderAuthCode: 'Authentication code',
      noEmptyAuthCode: 'Authentication code required',
      verify: 'Verify',
      backToLogin: 'Back to login',
      verificationFailed: 'Verification failed',
      twoFactorEnabledLogout: {
        message:
          'Two-factor authentication was enabled. For security, all sessions have been logged out. Please sign in again.'
      },
      rateLimited: 'Too many failed attempts. Try again in {{seconds}}s.',
      recoveryCodeUsed: 'The recovery code used for this login has been invalidated.',
      changePassword: 'Change Password',
      changePasswordDesc: 'For the security of your device, please change the password!',
      passwordVerificationRequired:
        'Please return to the account settings page and verify this password change again.',
      passwordVerificationExpired:
        'Password change verification has expired. Please return to the account settings page and verify again.',
      returnToSettings: 'Return to settings',
      differentPassword: 'Passwords do not match',
      illegalUsername: 'Username contains illegal characters',
      illegalPassword: 'Password contains illegal characters',
      forgetPassword: 'Forgot Password',
      ok: 'Ok',
      cancel: 'Cancel',
      savePassword: 'Save password',
      loginButtonText: 'Login',
      tips: {
        reset1: 'Web default account:',
        reset2: 'SSH default account:',
        reset3: 'To reset the passwords, please refer to ',
        changeReset: 'To reset the Web login password, please refer to ',
        resetLink: 'Reset'
      }
    },
    wifi: {
      errors: {
        connect:
          'Unable to connect to Wi-Fi. Check the network name, password, and device Wi-Fi status, then try again.'
      },
      title: 'Wi-Fi',
      description: 'Configure Wi-Fi for NanoKVM',
      submitted: {
        title: 'Hang tight — NanoKVM is joining your Wi-Fi',
        tipsTitle: 'What’s next',
        tipSwitch: 'You can switch your phone back to your usual Wi-Fi now.',
        tipCheck:
          'In a moment, check the NanoKVM screen for its new IP address and open it in your browser.',
        tipRetry:
          'If nothing shows up after a while, double-check the network name and password, then try again.'
      },
      invalidMode:
        'The current mode does not support network setup. Please go to your device and enable Wi-Fi configuration mode.',
      confirmBtn: 'Ok',
      finishBtn: 'Finished',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Password',
      steps: {
        settings: 'Settings',
        wifi: 'Wi-Fi',
        enable: 'Enable'
      },
      scan: {
        title: 'Choose a Wi-Fi network',
        description: 'Select a nearby network to continue.',
        loading: 'Searching for nearby Wi-Fi networks…',
        empty: 'No nearby Wi-Fi networks were found.',
        failed: "Couldn't scan nearby Wi-Fi networks. Enter the details manually instead.",
        retryBtn: 'Scan again',
        manualBtn: 'Enter network manually',
        backBtn: 'Back to nearby networks',
        connectTitle: 'Connect to {{ssid}}',
        connecting: 'Connecting…',
        viewSimple: 'Switch to simple view',
        viewDetailed: 'Switch to detailed view'
      },
      ap: {
        authTitle: 'Authentication Required',
        authDescription: 'Please enter the AP password to continue',
        authFailed: 'Invalid AP password',
        passPlaceholder: 'AP password',
        verifyBtn: 'Verify'
      },
      qr: {
        cameraBtn: 'Take photo',
        pickBtn: 'Choose screenshot',
        scanHint:
          'Take a photo of a Wi-Fi QR code, or choose a screenshot of one, such as the code a phone shows when sharing its Wi-Fi.',
        decoding: 'Reading QR code…',
        notFound:
          'No QR code was found in this photo. Keep the code centered and in focus, then try again.',
        invalid: 'This QR code does not contain usable Wi-Fi details.',
        pickerUnavailable:
          "Nothing opened? This sign-in window may block the camera and photos. Open this address in your phone's browser instead:",
        confirmTitle: 'Connect NanoKVM to {{ssid}}?',
        confirmDescription:
          'These network details came from a QR code. Check them before connecting.',
        openNetwork: 'This network has no password.',
        connectBtn: 'Connect',
        otherBtn: 'Choose another network'
      }
    },
    notification: {
      browser: {
        title: 'Browser Recommendation',
        description:
          'For the best user experience, we recommend using the Chrome browser. While we strive for compatibility across all major browsers, some advanced features may not work as intended outside of Chrome.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Screen',
      video: 'Video Mode',
      quality: 'Quality',
      qualityAuto: 'Automatic',
      qualityLossless: 'Lossless',
      qualityHigh: 'High',
      qualityMedium: 'Medium',
      qualityLow: 'Low',
      scale: 'Scale',
      noSignal: 'No video signal',
      captureDisabled:
        'Video capture is turned off. Enable it in "Settings > Device > Video Capture".',
      inconsistentVideoMode: 'Play paused. Another video mode is playing.',
      bandwidthWarning:
        'Network bandwidth may be insufficient for the current video settings. Try lowering the bitrate or frame rate.',
      webrtcConnectionFailed: 'WebRTC could not connect. Please switch to a Direct video mode.',
      webrtcUnavailable: {
        title: 'WebRTC is unavailable',
        description: 'Enable WebRTC in your browser, or switch to a Direct video mode.'
      },
      upload: 'Upload',
      advanced: 'Advanced',
      loading: 'Loading'
    },
    volume: {
      title: 'Volume'
    },
    microphone: {
      title: 'Microphone',
      notMounted: 'Microphone Not Mounted',
      tip1: 'The virtual microphone is not mounted on the remote host. Would you like to mount and enable it now?',
      tip2: 'You can also manage this later in "Settings > Device > Virtual Audio".',
      confirm: 'Enable',
      cancel: 'Cancel',
      noDevice: 'No available microphone on this device. Connect a microphone and try again.',
      permissionDenied:
        'Microphone access was denied. Allow microphone access in your browser settings.',
      notReadable: 'The microphone is in use by another application or cannot be read.',
      overconstrained: 'The microphone does not support the required audio settings.',
      securityBlocked:
        'Microphone access is blocked on this page. Check your browser security settings.',
      aborted: 'Microphone startup was aborted. Please try again.',
      unsupported: 'This browser or connection does not support microphone capture.',
      enableFailed: 'Failed to enable the microphone. Please try again.',
      disconnected:
        'The microphone was disconnected, so microphone transmission has been turned off.'
    },
    power: {
      title: 'Power',
      showConfirm: 'Confirmation',
      showConfirmTip: 'Power operations require an extra confirmation',
      powerShort: 'Power (short click)',
      powerLong: 'Power (long click)',
      powerConfirm: 'Proceed power operation?',
      okBtn: 'Yes',
      cancelBtn: 'No'
    },
    keyboard: {
      errors: {
        paste:
          'Unable to send the text to the remote device. Check the KVM connection and try again.'
      },
      title: 'Keyboard',
      paste: 'Paste',
      tips: 'Only standard keyboard letters and symbols are supported',
      placeholder: 'Please input',
      submit: 'Submit',
      virtual: 'Keyboard',
      swapCommandCtrl: 'Swap Cmd and Ctrl',
      shortcut: {
        title: 'Shortcuts',
        custom: 'Custom',
        capture: 'Click here to capture shortcut',
        clear: 'Clear',
        save: 'Save',
        captureTips:
          'Capturing system-level keys (such as the Windows key) requires full-screen permission.',
        enterFullScreen: 'Toggle full-screen mode.'
      },
      leaderKey: {
        title: 'Leader Key',
        desc: 'Bypass browser restrictions and send system shortcuts directly to the remote host.',
        howToUse: 'How to Use',
        simultaneous: {
          title: 'Simultaneous Mode',
          desc1: 'Press and hold the Leader Key, then press the shortcut.',
          desc2: 'Intuitive, but may conflict with system shortcuts.'
        },
        sequential: {
          title: 'Sequential Mode',
          desc1:
            'Press the Leader Key → press the shortcut in sequence → press the Leader Key again.',
          desc2: 'Requires more steps, but completely avoids system conflicts.'
        },
        enable: 'Enable Leader Key',
        tip: 'When assigned as a Leader Key, this key functions exclusively as a shortcut trigger and loses its default behavior.',
        placeholder: 'Please press the Leader Key',
        shiftRight: 'Right Shift',
        ctrlRight: 'Right Ctrl',
        metaRight: 'Right Win',
        submit: 'Submit',
        recorder: {
          rec: 'REC',
          activate: 'Activate keys',
          input: 'Please press the shortcut...'
        }
      }
    },
    mouse: {
      title: 'Mouse',
      mode: 'Mouse mode',
      absolute: 'Follow Mouse (absolute mode)',
      absoluteAndroid: 'Follow Mouse (Better Android Compatibility)',
      touchScreen: 'Multi-touch Screen',
      relative: 'Exclusive Mouse (relative mode)',
      direction: 'Wheel direction',
      scrollUp: 'Scroll up',
      scrollDown: 'Scroll down',
      speed: 'Wheel speed',
      fast: 'Fast',
      slow: 'Slow',
      requestPointer: 'Exclusive Mouse is active. Click the screen to lock and hide the pointer.',
      repairIphoneDrag: 'Repair iPhone drag',
      resetHid: 'Reset HID',
      touchpadGuide: {
        title: 'Pointer input guide',
        scope: 'Applies to normal pointer input when Mouse Mode is Relative.',
        swipeTitle: 'Swipe to move',
        swipeDesc: 'Swipe inside the active screen area to move the remote pointer.',
        tapTitle: 'Tap to click',
        tapDesc: 'A short tap sends one left click.',
        holdTitle: 'Hold for left button down',
        holdDesc: 'Touch and keep still for about 1 second to hold the left mouse button down.',
        dragTitle: 'Move after hold to drag',
        dragDesc: 'After the hold is active, move your finger to drag with the left button held.'
      },
      hidOnly: {
        errors: {
          load: 'Unable to load the HID-Only mode status. Try again in a moment.',
          update:
            'Unable to confirm whether HID-Only mode was updated. Reopen this setting to check its current status, then try again.'
        },
        title: 'HID-Only mode',
        desc: "If your mouse and keyboard stop responding and resetting HID doesn't help, it could be a compatibility issue between the NanoKVM and the device. Try to enable HID-Only mode for better compatibility.",
        tip1: 'Enabling HID-Only mode will unmount the virtual U-disk and virtual network',
        tip2: 'In HID-Only mode, image mounting is disabled',
        enable: 'Enable HID-Only mode',
        disable: 'Disable HID-Only mode'
      }
    },
    preview: {
      title: 'Preview'
    },
    image: {
      storageReadOnly: 'File storage is not writable. Please turn off the virtual disk.',
      storageBusy:
        'Storage is busy. Finish or cancel the file transfer before switching the virtual disk.',
      title: 'Images',
      loading: 'Loading...',
      empty: 'Nothing Found',
      upload: 'Upload Image',
      dropFile: 'Drop an image here, or click to select a file',
      uploadAction: 'Upload',
      cancelAction: 'Cancel',
      mount: 'Mount Image',
      mounted: 'Mounted',
      uploadError: 'Upload failed, please try again',
      uploadInsufficientStorage: 'Insufficient storage space',
      uploadTooLarge: 'Image exceeds the size limit',
      cancelUploadError: 'Failed to cancel the upload',
      mountMode: 'Mount mode',
      massStorage: 'Mass Storage',
      cdRom: 'CD ROM',
      readOnly: 'Read only',
      mountFailed: 'Mount failed',
      mountDesc:
        'On some systems, you need to eject the virtual hard drive from the remote host before mounting the image.',
      unmountFailed: 'Unmount failed',
      unmountDesc:
        'On some systems, you need to manually eject from the remote host before unmounting the image.',
      refresh: 'Refresh the image list',
      download: 'Download Image',
      attention: 'Attention',
      deleteConfirm: 'Are you sure you want to delete this image?',
      deleteFailed: 'Failed to delete the image',
      okBtn: 'Yes',
      cancelBtn: 'No'
    },
    script: {
      title: 'Scripts',
      upload: 'Upload',
      run: 'Run',
      runBackground: 'Run Background',
      runFailed: 'Run failed',
      uploadFailed: 'Failed to upload the script',
      deleteFailed: 'Failed to delete the script',
      attention: 'Attention',
      delDesc: 'Are you sure you want to delete this file?',
      confirm: 'Yes',
      cancel: 'No',
      delete: 'Delete',
      close: 'Close'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM Terminal',
      confirm: 'Ok',
      submit: 'Submit',
      username: 'Username',
      password: 'Password',
      authCode: 'Authentication code',
      authorizeTitle: 'Authorize Web Terminal',
      codeAccountHint: 'Enter an authentication code or unused recovery code.',
      authFailed: 'Authentication failed'
    },
    wol: {
      errors: {
        send: 'Unable to send the wake command. Check the MAC address and network connection, then try again.',
        loadFailed:
          'Unable to load the Wake-on-LAN device list. Check that NanoKVM is online and try again.',
        renameFailed: 'Unable to rename this device. Check the MAC address and try again.',
        deleteFailed: 'Unable to delete this device. Try again in a moment.'
      },
      title: 'Wake-on-LAN',
      sending: 'Sending command...',
      sent: 'Command sent',
      input: 'Please enter the MAC',
      ok: 'Ok'
    },
    download: {
      sha256: 'SHA-256 (optional)',
      sha256Placeholder: 'Enter a 64-character SHA-256 checksum',
      invalidSHA256: 'SHA-256 must be a 64-character hexadecimal string',
      title: 'Image Downloader',
      input: 'Please enter a remote image URL',
      ok: 'Ok',
      disabled: '/data partition is RO, so we cannot download the image',
      downloadingFile: 'Downloading: {{file}}',
      downloadingProgress: 'Downloading ({{percentage}}): {{file}}',
      failed: 'Failed',
      insufficientStorage: 'Insufficient storage space',
      remaining: '{{time}} remaining',
      cancel: 'Cancel',
      cancelFailed: 'Failed to cancel the download',
      pause: 'Pause',
      resume: 'Resume',
      pauseFailed: 'Failed to pause the download',
      resumeFailed: 'Failed to resume the download'
    },
    settings: {
      back: 'Back',
      close: 'Close',
      ai: {
        errors: {
          loadServices:
            'Unable to load AI service status. Check that NanoKVM is online and try again.',
          loadMcpConfig: 'Unable to load the MCP service configuration. Try again in a moment.',
          updateMcpService:
            'Unable to update the MCP service status. Check its current status before trying again.',
          accessMcpApiKey:
            'Unable to retrieve the MCP API key. Try again and complete verification if prompted.',
          regenerateMcpApiKey: 'Unable to regenerate the MCP API key. Try again in a moment.',
          regeneratedMcpApiKeyUnavailable:
            'The API key was regenerated, but it could not be displayed. Try displaying it again. The previous key is no longer valid.',
          loadMemoryFabricState:
            'Unable to load the Memory Fabric service status. Try again in a moment.',
          updateMemoryFabricService:
            'Unable to update the Memory Fabric service status. Check its current status before trying again.',
          loadScreenTimelapseState:
            'Unable to load the screen timelapse service status. Try again in a moment.',
          updateScreenTimelapseService:
            'Unable to update the screen timelapse service status. Check its current status before trying again.',
          openScreenTimelapse: 'Unable to open screen timelapse playback. Try again in a moment.',
          loadScreenTimelapseRecordings:
            'Unable to load screen timelapse recordings. Try again in a moment.',
          playScreenTimelapse:
            'This timelapse recording cannot be played right now. Select the date again or try again later.',
          screenTimelapseFullscreen:
            'Unable to change fullscreen mode. Check browser permissions and try again.'
        },
        title: 'AI',
        mcpService: 'MCP Service',
        mcpServiceDesc: 'Authenticated gateway for control, media, and enabled Memory Fabric tools',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Process agent context memory and OCR locally',
        memoryFabricRemoteRequiresMcp:
          'Enable MCP Service above to make memory and OCR tools available remotely.',
        memoryFabricRemoteAvailable:
          'Memory and OCR tools are available through the /api/mcp endpoint above using the same API key.',
        memoryFabricConfig: {
          title: 'Configuration',
          description: 'Manage only the default Memory Fabric model settings.',
          loadFailed: 'Failed to load Memory Fabric configuration.',
          saveFailed: 'Failed to save Memory Fabric configuration.',
          saved: 'Configuration saved.',
          savedRestarted: 'Configuration saved and Memory Fabric restarted.',
          noChanges: 'No configuration changes.',
          reload: 'Reload',
          save: 'Save',
          unsaved: 'Unsaved changes',
          conflictTitle: 'Configuration changed',
          conflictDescription:
            'The configuration was modified elsewhere. Reload the latest version before saving again.',
          conflictReload: 'Reload latest',
          discardTitle: 'Discard unsaved changes?',
          discardDescription:
            'Reloading will replace the current form with the latest saved configuration.',
          discardConfirm: 'Discard and reload',
          sections: {
            models: 'Models'
          },
          openAICompatibleHint:
            'Supports OpenAI-compatible model providers. Enter the model name, Base URL, and API key provided by your provider.',
          fields: {
            model: 'Model',
            baseUrl: 'Base URL',
            maxTokens: 'Maximum tokens',
            timeoutSeconds: 'Timeout (seconds)'
          },
          apiKey: {
            title: 'API key',
            configured: 'Configured',
            notConfigured: 'Not configured',
            keep: 'Keep existing key',
            replace: 'Replace key',
            clear: 'Clear key',
            replacement: 'New API key',
            replacementPlaceholder: 'Enter a new API key',
            clearHint:
              'The saved API key will be removed; the environment variable may still provide a key.',
            required: 'Enter a new API key.'
          },
          validation: {
            required: 'This field is required.',
            invalidUrl: 'Enter a valid HTTP or HTTPS URL.'
          }
        },
        screenTimelapseService: 'Screen Timelapse',
        screenTimelapseServiceDesc:
          'Replay Memory Fabric screen history and search captured screen content',
        screenTimelapseSearchRequiresMemoryFabric:
          'To search recorded content, enable Memory Fabric as well.',
        screenTimelapseOpen: 'Open',
        screenTimelapseSearchPlaceholder: 'Search contents in video...',
        screenTimelapseNoVideos: 'No timelapse videos found',
        screenTimelapseNoResults: 'No matching frames found',
        screenTimelapseMultipleResults: 'Search Results',
        screenTimelapseUnavailable: 'Unavailable',
        screenTimelapsePlayerUnsupported: 'This browser cannot play timelapse video',
        screenTimelapseSearchFailed: 'Search failed',
        screenTimelapseLoading: 'Loading timelapse...',
        screenTimelapseSearchAction: 'Search timelapse',
        screenTimelapsePlay: 'Play timelapse',
        screenTimelapsePause: 'Pause timelapse',
        screenTimelapseEnterFullscreen: 'Enter fullscreen',
        screenTimelapseExitFullscreen: 'Exit fullscreen',
        screenTimelapseFullscreenUnsupported: 'This browser cannot enter fullscreen',
        screenTimelapsePreviousDay: 'Previous day',
        screenTimelapseNextDay: 'Next day',
        screenTimelapseDeleteDay: 'Delete recording for {{date}}',
        screenTimelapseDelete: 'Delete',
        screenTimelapseDeleteConfirmTitle: 'Delete timelapse recording?',
        screenTimelapseDeleteConfirmDesc:
          'The recording for {{date}} occupies {{size}}. This action cannot be undone.',
        screenTimelapseDeleteSuccess: 'Deleted the recording for {{date}}',
        screenTimelapseDeleteFailed: 'Failed to delete the recording for {{date}}',
        endpoint: 'Endpoint',
        apiKey: 'API Key',
        regenerate: 'Regenerate',
        copyApiKey: 'Copy API key',
        copyEndpoint: 'Copy endpoint',
        showApiKey: 'Show API key',
        regenerateConfirmTitle: 'Regenerate API key',
        regenerateConfirmDesc:
          'Existing MCP clients will stop working until updated with the new key. Do you want to continue?',
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      title: 'Settings',
      about: {
        title: 'About NanoKVM',
        information: 'Information',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Application Version',
        applicationTip: 'NanoKVM web application version',
        arch: 'Arch',
        device_number: 'Device Number',
        image: 'Image Version',
        imageTip: 'NanoKVM system image version',
        deviceKey: 'Device Key',
        community: 'Community',
        hostname: 'Hostname',
        hostnameUpdated: 'Hostname updated. Reboot to apply.',
        ipType: {
          Wired: 'Wired',
          Wireless: 'Wireless',
          Other: 'Other'
        }
      },
      appearance: {
        title: 'Appearance',
        display: 'Display',
        language: 'Language',
        languageDesc: 'Select the language for the interface',
        languageChangeFailed: 'Failed to change language',
        webTitle: 'Web Title',
        webTitleDesc: 'Customize the web page title',
        menuBar: {
          title: 'Menu Bar',
          mode: 'Display Mode',
          modeDesc: 'Display menu bar on the screen',
          modeOff: 'Off',
          modeAuto: 'Auto hide',
          modeAlways: 'Always visible',
          icons: 'Submenu Icons',
          showIcon: 'Show icon',
          activateOnConnect: 'Activate on Connect',
          stopSharing: 'Stop sharing',
          iconsDesc: 'Display submenu icons in the menu bar'
        }
      },
      screen: {
        title: 'Screen',
        videoMode: {
          title: 'Video Mode',
          description: 'Select the encoding and transmission method'
        },
        edid: {
          description: 'Select display parameters',
          default: 'Default',
          custom: 'Custom',
          document: 'Document',
          powerConfirmTitle: 'Confirm before updating EDID',
          powerConfirmDescription:
            'Connect PD power first. Updating EDID may briefly disconnect the display; if the device cannot detect the display correctly afterward, restart it manually.',
          powerConfirmOk: 'Continue',
          powerConfirmOkCountdown: 'Continue ({{seconds}})',
          powerConfirmCancel: 'Cancel',
          errors: {
            badRequest: 'Invalid EDID request',
            invalidFileName: 'Select a .bin EDID file with a valid file name',
            fileTooLarge: 'The EDID file is too large',
            invalidLength: 'The EDID must be 128, 256, 384, or 512 bytes',
            invalidHeader: 'The EDID header is invalid',
            unsupportedVersion: 'This EDID version is not supported',
            extensionCountMismatch: 'The EDID extension count does not match the file length',
            invalidChecksum: 'The EDID checksum is invalid',
            invalidCta: 'The CTA extension structure is invalid',
            currentCannotOverwrite:
              'The EDID currently in use cannot be overwritten. Switch to another EDID first',
            saveFailed: 'Failed to save the EDID',
            switchFailed: 'Failed to switch the EDID',
            uploadFailed: 'Failed to upload the EDID',
            deleteFailed: 'Failed to delete the EDID',
            networkFailed: 'Network error. Try again'
          }
        },
        rateControlMode: {
          title: 'Rate Control Mode',
          description: 'Select the method to control the bitrate',
          cbr: 'CBR (Constant Bitrate): Prioritizes connection stability. Minimizes stuttering, but image quality may drop during high-motion scenes.',
          vbr: 'VBR (Variable Bitrate): Prioritizes image clarity. High-motion scenes may cause bandwidth spikes, potentially leading to latency or stutter.'
        },
        bitrate: {
          title: 'Bitrate',
          description: 'Balance image clarity with bandwidth usage',
          lower:
            'Lower values reduce latency on poor networks but compromise image quality, potentially causing blur or artifacts.',
          higher:
            'Higher values improve image clarity but consume more bandwidth. May increase latency if the connection is slow.'
        },
        gop: {
          description: 'Set the interval between full keyframes',
          lower:
            'Lower values send full frames more frequently. This speeds up recovery from visual artifacts but consumes more bandwidth and reduces image detail.',
          higher:
            'Higher values optimize for incremental updates. This improves image detail and saves bandwidth, but recovery from visual artifacts will take longer.'
        },
        quality: {
          title: 'Quality',
          description: 'Set the display image quality'
        },
        fps: {
          auto: 'Auto',
          description: 'Set the display frame rate',
          tip1: "Setting the value to 0 enables Auto mode, synchronizing the display with the remote host's frame rate.",
          tip2: 'Lowering the frame rate significantly reduces bandwidth usage. Manually decrease it during network instability or when high frame rates are not required.'
        },
        scale: {
          title: 'Scale',
          description: 'Set the display scaling ratio'
        }
      },
      device: {
        title: 'Device',
        virtualAudioConfirm: {
          title: 'Confirm before switching Virtual Audio',
          description:
            'Connect PD power first. Switching Virtual Audio may briefly disconnect the display; if the device cannot detect the display correctly afterward, restart it manually.',
          ok: 'Continue switching',
          okCountdown: 'Continue switching ({{seconds}})',
          cancel: 'Cancel'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Configure Wi-Fi',
          apMode: 'AP mode is enabled, connect to Wi-Fi manually',
          others: 'Others...',
          connect: 'Join Wi-Fi',
          connectDesc1: 'Please enter the network ssid and password',
          connectDesc2: 'Please enter the password to join this network',
          disconnect: 'Are you sure to disconnect the network?',
          ssid: 'Name',
          password: 'Password',
          joinBtn: 'Join',
          confirmBtn: 'Ok',
          cancelBtn: 'Cancel',
          preset: {
            title: 'Wi-Fi Preset',
            description: 'Pre-store one target Wi-Fi; the device joins it automatically on boot',
            empty: 'None',
            addBtn: 'Add preset',
            removeBtn: 'Remove preset',
            saveTitle: 'Pre-store Wi-Fi',
            saveDesc:
              'Save the target network now; the device joins it automatically when the network is found at boot',
            saveBtn: 'Save',
            passwordPlaceholder: 'Password (leave empty for open networks)',
            saveFailed: 'Failed to save the Wi-Fi preset',
            removeFailed: 'Failed to remove the Wi-Fi preset',
            removeConfirm: 'Are you sure you want to remove the pre-stored Wi-Fi?',
            warning:
              'Once saved, the device will try its best to join this Wi-Fi whenever it is detected at boot, which may interfere with joining other Wi-Fi networks. The preset stays in effect until you edit or remove it.'
          }
        },
        capture: {
          title: 'Video Capture',
          description: 'Enable video capture'
        },
        ssh: {
          description: 'Enable SSH remote access',
          tip: 'After enabling SSH, set a strong password for the root user in Terminal.'
        },
        swap: {
          errors: {
            load: 'Unable to load the swap memory status. Try again in a moment.',
            update:
              'Unable to update the swap memory setting. Check its current status before trying again.'
          },
          title: 'Swap Memory',
          description: 'Enable compressed swap memory'
        },
        advanced: 'Advanced Settings',
        staticIp: {
          title: 'Static IP',
          config: 'Config',
          tip: 'To prevent connection loss, it is recommended to configure while you have physical access to the device.',
          document: 'Check the document.',
          enable: 'Enable static IP',
          placeholder: 'Format: IP/CIDR [Gateway]\nExample: 192.168.1.10/22',
          notEmpty: 'IP address cannot be empty.',
          notValid: 'IP address format is invalid.',
          failed: 'Configuration failed.',
          success: 'Configuration complete. Please check if the IP address is available.',
          okBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        usbNetwork: {
          title: 'USB Network',
          mode: 'Mode',
          server: 'Server',
          serverDesc: 'Assign an IP address to the remote host through the built-in DHCP server.',
          client: 'Client',
          clientDesc: 'Obtain an IP address from the remote host through DHCP.',
          forwarding: 'IP forwarding',
          forwardingTip:
            'To allow the remote host to access the internet through NanoKVM, enable IP forwarding.',
          manualIp: 'Manual IP address',
          ipv4Placeholder: 'IPv4/CIDR, e.g. 10.10.10.1/24',
          invalidIpv4: 'Enter a valid IPv4/CIDR address.',
          loadFailed: 'Unable to load the USB network configuration.',
          failed: 'Configuration failed.',
          success: 'Configuration complete.',
          applyBtn: 'Apply',
          cancelBtn: 'Cancel'
        },
        usbId: {
          title: 'USB Device Identity',
          config: 'Config',
          tip: 'Custom USB identity changes how the host identifies this device. Applying restarts the USB gadget and may briefly disconnect keyboard, mouse, or network. Leave optional fields empty to keep script defaults.',
          enable: 'Enable custom USB device identity',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Device version (bcdDevice)',
          serialnumber: 'Serial number',
          manufacturer: 'Manufacturer',
          product: 'Product',
          vidPlaceholder: 'e.g. 0x359F',
          pidPlaceholder: 'e.g. 0x3305',
          bcdDevicePlaceholder: 'e.g. 0x0101 (optional)',
          serialnumberPlaceholder: 'e.g. 0123456789ABCDEF (optional)',
          manufacturerPlaceholder: 'e.g. sipeed (optional)',
          productPlaceholder: 'e.g. NanoKVM-Go (optional)',
          notEmpty: 'VID and PID cannot be empty.',
          notValid: 'VID/PID format is invalid. Use hex like 0x359F.',
          bcdNotValid: 'bcdDevice format is invalid. Use hex like 0x0101, or leave empty.',
          failed: 'Configuration failed.',
          success: 'Configuration complete. USB gadget restarted.',
          okBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        diskInquiry: {
          title: 'USB Drive Name',
          config: 'Config',
          tip: 'The name of this device’s virtual USB flash drive as shown on the remote computer. Saving restarts USB and may briefly disconnect keyboard, mouse, or network.',
          enable: 'Use a custom USB drive name',
          placeholder: 'e.g. sipeed  Flash Drive     1.00',
          notEmpty: 'Name cannot be empty.',
          notValid: 'Name is invalid. Use up to 28 printable ASCII characters without line breaks.',
          failed: 'Configuration failed.',
          success: 'Configuration complete. USB gadget restarted.',
          okBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        diskInquiryCdrom: {
          title: 'Virtual CD Drive Name',
          config: 'Config',
          tip: 'The name of this device’s virtual CD/DVD drive as shown on the remote computer. Saving restarts USB and may briefly disconnect keyboard, mouse, or network.',
          enable: 'Use a custom CD drive name',
          placeholder: 'e.g. sipeed  Optical Drive   1.00',
          notEmpty: 'Name cannot be empty.',
          notValid: 'Name is invalid. Use up to 28 printable ASCII characters without line breaks.',
          failed: 'Configuration failed.',
          success: 'Configuration complete. USB gadget restarted.',
          okBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        webcamFunctionName: {
          title: 'USB Webcam Name',
          config: 'Config',
          tip: 'Set the USB camera name reported to the attached computer. A new name takes effect the next time Webcam is enabled or the Go+ restarts.',
          enable: 'Use a custom webcam name',
          placeholder: 'UVC Camera',
          notEmpty: 'Enter a webcam name.',
          notValid: 'Use a name of up to 63 characters without line breaks or control characters.',
          failed: 'Could not save the webcam name.',
          success: 'Webcam name saved.',
          okBtn: 'Save',
          cancelBtn: 'Cancel'
        },
        uac2FunctionName: {
          title: 'USB Audio Name',
          config: 'Config',
          tip: 'The name of the virtual USB microphone/audio device as shown on the remote computer. Saving restarts USB and may briefly disconnect keyboard, mouse, or network.',
          enable: 'Use a custom USB audio name',
          placeholder: 'e.g. NanoKVM-Go Audio',
          notEmpty: 'Name cannot be empty.',
          notValid: 'Name is invalid. Use up to 64 printable characters without line breaks.',
          failed: 'Configuration failed.',
          success: 'Configuration complete. USB gadget restarted.',
          okBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        lowPower: {
          title: 'Low Power Mode',
          description: 'Reduce power use while idle',
          tips: 'Tips',
          tip1: 'When enabled, video capture will be paused during idle periods to reduce power consumption and heat.',
          tip2: 'Resuming video capture may require re-initialization, during which video may be temporarily unavailable.',
          okBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        mouseJiggler: {
          title: 'Mouse Jiggler',
          description: 'Prevent the remote host from sleeping',
          disable: 'Disable',
          absolute: 'Absolute Mode',
          relative: 'Relative Mode'
        },
        fingerRobot: {
          title: 'Finger Robot',
          description: 'Enable finger robot control'
        },
        mdns: {
          description: 'Enable mDNS discovery service'
        },
        hidOnly: 'HID-Only Mode',
        imageMounted: 'Image Mounted',
        disk: 'Virtual Disk',
        diskDesc: 'Mount virtual U-disk on the remote host',
        sdCard: {
          warning: 'Initialize Virtual Disk',
          tip1: 'The first time you enable this feature, the virtual disk will be initialized and existing image files and virtual disk data on the device will be erased. This cannot be undone. Back up important files first.',
          tip2: 'Continue initialization?',
          capacity: 'Capacity',
          disable: 'Disable',
          disableWarning: 'Disable Virtual Disk?',
          disableTip:
            'Disabling will erase all files on the virtual disk and release the device storage it uses. You will need to select a capacity and initialize it again the next time you enable it. This cannot be undone.'
        },
        network: 'Virtual Network',
        networkDesc: 'Mount virtual network card on the remote host',
        mic: 'Virtual Audio',
        micDesc: 'Choose the USB virtual audio mode exposed to the remote host',
        virtualAudioMode: {
          off: 'Off',
          mic: 'Microphone only',
          duplex: 'Microphone + speaker'
        },
        speaker: 'Virtual Speaker',
        speakerDesc:
          'When enabled, replaces the default DP audio output to support remote calls with broader compatibility. When disabled, only media audio can be forwarded; call audio cannot.',
        micTip:
          'Due to USB endpoint limitations, enabling the Virtual Mic will disable the USB Secondary Screen.\n' +
          'And it is recommended to disable this feature during BIOS operations to ensure optimal compatibility.',
        reboot: 'Reboot',
        rebootDesc: 'Are you sure you want to reboot NanoKVM?',
        rebootToFlashing: 'Reboot to flashing mode',
        rebootToFlashingButton: 'Reboot',
        rebootToFlashingTitle: 'Reboot to flashing mode?',
        rebootToFlashingDesc:
          'NanoKVM will reboot and enter USB flashing mode. Make sure the USB interface used for flashing is connected and the flashing tool is ready.',
        datetime: {
          timezone: 'Time zone',
          datetime: 'Datetime',
          dateAndTime: 'Date and time',
          synchronize: 'Synchronize time',
          lastSynchronization: 'Last synchronization: ',
          notSynchronized: 'The system time is not synchronized yet',
          syncNow: 'Sync now',
          manualSet: 'Manual',
          manualSetTitle: 'Set time manually',
          manualSetDescription: 'You will need to sign in again after the time is changed.',
          useBrowserTime: 'Use browser time',
          setTime: 'Set time',
          invalidTime: 'Enter a valid date and time.',
          setFailed: 'Unable to set the system time. Try again.',
          setSucceeded: 'System time updated. Please sign in again.'
        },
        dialBackground: {
          errors: {
            load: 'Unable to load dial background images. Try again in a moment.',
            upload:
              'Unable to upload the background image. Check that it is a valid BMP file and try again.',
            switch: 'Unable to switch the dial background. Refresh the image list and try again.'
          },
          title: 'Dial Background',
          description: 'Select a BMP image from /kvmcomm/ui/bg/ as the dial background',
          upload: 'Upload',
          loading: 'Loading...',
          empty: 'No BMP images found',
          current: 'Current',
          onlyBmp: 'Only BMP images are supported',
          uploaded: 'Upload complete',
          switched: 'Background switched'
        },
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Restart Tailscale?',
        stop: 'Stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable automatic startup on boot.',
        loading: 'Loading...',
        notInstall: 'Tailscale not found! Please install.',
        install: 'Install',
        installing: 'Installing',
        failed: 'Install failed',
        retry: 'Please refresh and try again. Or try to install manually',
        download: 'Download the',
        package: 'installation package',
        unzip: 'and unzip it',
        upTailscale: 'Upload tailscale to NanoKVM directory /usr/bin/',
        upTailscaled: 'Upload tailscaled to NanoKVM directory /usr/sbin/',
        refresh: 'Refresh current page',
        notRunning: 'Tailscale is not running. Please start it to continue.',
        run: 'Start',
        notLogin:
          'The device has not been bound yet. Please login and bind this device to your account.',
        urlPeriod: 'This url is valid for 10 minutes',
        login: 'Login',
        loginSuccess: 'Login Success',
        enable: 'Enable Tailscale',
        exitNode: 'Exit node',
        exitNodeDesc: 'Advertise this device as an internet exit for your tailnet.',
        subnetRoute: 'Subnet route',
        subnetRouteDesc: 'Advertise the directly connected LAN subnet {{subnet}} to your tailnet.',
        subnetUnavailable: 'No directly connected LAN subnet is available.',
        routingUnavailable: 'Tailscale routing settings are unavailable on this device.',
        pendingApproval:
          'After enabling, approve the advertised route in the Tailscale admin console.',
        routingUpdateFailed: 'Failed to update Tailscale routing settings.',
        deviceName: 'Device Name',
        deviceIP: 'Device IP',
        account: 'Account',
        logout: 'Logout',
        logoutDesc: 'Are you sure you want to logout?',
        uninstall: 'Uninstall Tailscale',
        uninstallDesc: 'Are you sure you want to uninstall Tailscale?',
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      update: {
        title: 'Check for Updates',
        queryFailed: 'Get version failed',
        updateFailed: 'Update failed. Please retry.',
        isLatest: 'You already have the latest version.',
        available: 'An update is available. Are you sure you want to update now?',
        updating: 'Update started. Please wait...',
        confirm: 'Confirm',
        upgrade_tip_nano:
          'A new version of NanoKVM is available. Are you sure you want to update now?',
        upgrade_tip_firmware:
          'A new firmware version is available. Are you sure you want to update now?',
        cancel: 'Cancel',
        preview: 'Preview Updates',
        previewDesc: 'Get early access to new features and improvements',
        previewTip:
          'Please be aware that preview releases may contain bugs or incomplete functionality!',
        download: 'Download',
        install: 'Install',
        changelog: 'Changelog',
        restart: 'Restart'
      },
      account: {
        title: 'Account',
        webAccount: 'Web Account Name',
        password: 'Password',
        twoFactor: 'Two-factor authentication',
        enableTwoFactor: 'Enable',
        enableTwoFactorDesc:
          'Enter your current password to start setting up two-factor authentication.',
        disableTwoFactor: 'Disable',
        recoveryCodes: 'Recovery codes',
        enableRecoveryCodes: 'Enable',
        updateRecoveryCodes: 'Update',
        recoveryCodesTooltipDisabled:
          'Recovery codes are backup two-factor credentials. You can generate them after enabling two-factor authentication.',
        recoveryCodesTooltipEnabled:
          'Recovery codes are valid only while two-factor authentication is enabled. Disabling two-factor authentication invalidates all recovery codes; while two-factor authentication stays enabled, they can only be updated.',
        recoveryCodesRegenerateDesc:
          'Enter your current password and either an authentication code or an unused recovery code. Updating recovery codes will invalidate all existing recovery codes.',
        enableRecoveryCodesDesc:
          'Enter your current password and an authentication code to generate recovery codes. Recovery codes can only be viewed once, so save them before leaving this page.',
        recoveryCodesVerifyFailed:
          'Unable to verify the password and the authentication code or recovery code.',
        recoveryCodesSavedTitle: 'Save these recovery codes now',
        recoveryCodesSavedDesc:
          'Recovery codes can only be viewed once. Save them in a secure location before leaving this page.',
        copyRecoveryCodes: 'Copy',
        downloadRecoveryCodes: 'Download',
        recoveryCodesCopyFailed: 'Unable to copy recovery codes.',
        done: 'Done',
        currentPassword: 'Current password',
        authCode: 'Authentication code',
        authCodeOrRecoveryCode: 'Authentication code or recovery code',
        authRateLimited: 'Too many failed attempts. Try again in {{seconds}}s.',
        verifyPasswordChange: 'Verify password change',
        verifyPasswordChangeDesc:
          'Enter your current password and either an authentication code or an unused recovery code before changing the password.',
        verifyPasswordChangeFailed:
          'Unable to verify the password and the authentication code or recovery code.',
        verifyAndEnable: 'Verify and enable',
        scanTwoFactorQRCode: 'Scan this QR code with your authenticator app, then enter the code.',
        showSetupKey: 'Show setup key',
        hideSetupKey: 'Hide setup key',
        setupKey: 'Setup key',
        twoFactorSetupFailed: 'Unable to start two-factor setup.',
        twoFactorVerifyFailed: 'Unable to verify the authentication code.',
        twoFactorDisableFailed: 'Unable to disable two-factor authentication.',
        disableTwoFactorDesc:
          'Enter your current password and either an authentication code or an unused recovery code. Disabling two-factor authentication will invalidate all recovery codes.',
        updateBtn: 'Change',
        logoutBtn: 'Logout',
        logoutDesc: 'Are you sure you want to logout?',
        okBtn: 'Yes',
        cancelBtn: 'No',
        twoFactorProtection: {
          title: 'Advanced 2FA Protection',
          description:
            'Select which sensitive actions require a second factor when protection enforcement is enabled. The "Direct web login" switch does the opposite: it allows signing in with only an authentication code or unused recovery code.',
          login: 'Direct web login',
          webTerminal: 'Web Terminal',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP',
          targetPower: 'Target machine power',
          nanoKVMRestart: 'NanoKVM restart',
          deleteImage: 'Delete image',
          scripts: 'Script operations',
          screenTimelapse: 'Screen Timelapse',
          loginScope:
            'Lets web login use only an authentication code or unused recovery code instead of a username and password. This reduces login security.',
          webTerminalScope: 'Opening Web Terminal.',
          sshScope: 'Changing the SSH service state.',
          tailscaleScope: 'Installing, connecting, or changing Tailscale.',
          frp: 'FRP',
          frpScope: 'Installing, connecting, or changing FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Installing, connecting, or changing Cloudflare Tunnel.',
          zerotierScope: 'Installing, connecting, or changing ZeroTier.',
          netbirdScope: 'Installing, connecting, or changing NetBird.',
          mcpScope:
            'Showing, copying, or regenerating the MCP API key, and enabling or disabling MCP.',
          targetPowerScope:
            'Shutting down or restarting the target machine with its physical power button.',
          nanoKVMRestartScope: 'Restarting NanoKVM itself.',
          deleteImageScope: 'Permanently deleting a stored ISO or IMG image.',
          scriptsScope: 'Running, uploading, or deleting scripts.',
          screenTimelapseScope:
            'Enabling Screen Timelapse recording, browsing recordings, and deleting recordings.',
          authorizeTitle: 'Authorize protection changes',
          authorizeDesc:
            'Enter an authentication code or unused recovery code. This authorization lasts for five minutes.',
          authorizeFailed: 'Unable to verify the authentication code or recovery code.',
          loadFailed: 'Unable to load protection settings.',
          updateFailed: 'Unable to save protection settings.',
          retry: 'Retry',
          disableScriptsWarningTitle: 'Turn off script protection?',
          disableScriptsWarningDesc:
            'Without protection, a malicious script could gain elevated access to this device and bypass or remove 2FA protection.',
          cancelDisabling: 'Cancel',
          continueDisabling: 'Continue disabling',
          continueDisablingCountdown: 'Continue disabling ({{seconds}}s)'
        },
        twoFactorGuide: {
          title: 'Two-factor authentication guide',
          scope: 'Two-factor authentication adds an extra layer of security to your account.',
          whatTitle: 'What is 2FA?',
          whatDesc:
            'Two-factor authentication (2FA) protects your account with a time-based code from an authenticator app. By default the code is required in addition to your password, so even if your password is compromised, your account stays protected. If direct web login is enabled, the code alone signs you in.',
          appsTitle: 'Recommended apps',
          appsDesc:
            'You can use Google Authenticator or Microsoft Authenticator. Both are free and work on iOS and Android.',
          setupTitle: 'How to set up',
          setupDesc:
            'Open your authenticator app, scan the QR code shown during setup, or enter the setup key manually. Then enter the 6-digit code from the app to finish enabling 2FA.',
          recoveryTitle: 'Recovery codes',
          recoveryDesc:
            'Recovery codes are one-time backup codes. If you lose access to your authenticator app, you can use a recovery code to log in. Save them in a secure place immediately after generating them.',
          logoutNotice:
            'Enabling 2FA will log out all active sessions, including this one. You will need to sign in again on every device.'
        }
      }
    },
    error: {
      title: "We've ran into an issue",
      refresh: 'Refresh'
    },
    pictureInPicture: {
      title: 'Picture-in-Picture',
      enter: 'Enter Picture-in-Picture',
      exit: 'Exit Picture-in-Picture',
      error: 'Unable to start Picture-in-Picture. Wait for the video to load and try again.'
    },
    fullscreen: {
      toggle: 'Toggle Fullscreen'
    },
    menu: {
      collapse: 'Collapse Menu',
      expand: 'Expand Menu'
    }
  }
};

export default en;
