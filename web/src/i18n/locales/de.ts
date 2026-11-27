const de = {
  translation: {
    storage: {
      cards: { mount: 'Laufwerk einbinden', share: 'Dateifreigabe' },
      title: 'Abbilder & Dateifreigabe',
      empty: 'Noch keine Abbilder',
      notAttached: 'Nichts eingebunden',
      attached: 'Eingebunden: {{name}}',
      eject: 'Auswerfen',
      mount: 'Einbinden',
      mountMode: 'Mount-Modus',
      mountModeDisk: 'Massenspeicher',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Dateimanager öffnen',
      refresh: 'Aktualisieren',
      delete: 'Löschen',
      deleteTitle: 'Medium löschen',
      deleteConfirm: '{{name}} löschen? Die darin enthaltenen Dateien gehen verloren.',
      downloadImage: 'Abbild herunterladen',
      typePartition: 'Freigegebene Dateien',
      volumeLabel: 'Wechseldatenträger',
      groups: {
        share: 'Dateifreigabe',
        images: 'Abbilder'
      },
      failed: 'Vorgang fehlgeschlagen',
      hidOnly: 'HID-Modus: Das virtuelle Laufwerk ist nicht verfügbar',
      disabled: 'Das virtuelle Laufwerk ist deaktiviert (Einstellungen - Gerät)',
      force: {
        title: 'Medium mit Gewalt zurückholen?',
        desc: 'Der Zielrechner verwendet dieses Medium noch. Beim Zurückholen wird das USB-Gerät neu aufgelistet, und alles, was noch nicht auf den Datenträger geschrieben wurde, kann verloren gehen.',
        menuHint:
          'Der Zielrechner verwendet dieses Medium. Öffnen Sie den Dateimanager, um es mit Gewalt zurückzuholen.'
      },
      tips: {
        title: 'So laden Sie Dateien hoch',
        usb1: 'Verbinden Sie den NanoKVM über USB mit Ihrem Computer.',
        usb2: 'Binden Sie auf dieser Seite die freigegebenen Dateien im Massenspeicher-Modus ein.',
        usb3: 'Öffnen Sie die virtuelle Festplatte auf Ihrem Computer und kopieren Sie die Image-Datei in das Stammverzeichnis der virtuellen Festplatte.',
        usb4: 'Werfen Sie das Laufwerk nach dem Kopieren aus: Danach erscheint das Image in der Liste.',
        scp1: 'Stellen Sie sicher, dass sich der NanoKVM und Ihr Computer im selben lokalen Netzwerk befinden.',
        scp2: 'Öffnen Sie ein Terminal auf Ihrem Computer und verwenden Sie den SCP-Befehl, um die Image-Datei in das Verzeichnis /data auf dem NanoKVM hochzuladen.',
        scp3: 'Beispiel: scp your-image-path root@your-nanokvm-ip:/data'
      },
      job: {
        running: '{{name}} wird bearbeitet'
      },
      enableDisk: {
        title: 'Virtuelles Laufwerk aktivieren?',
        desc: 'Zum Einbinden eines Mediums wird das virtuelle Laufwerk benötigt, das derzeit ausgeschaltet ist. Jetzt aktivieren? Tastatur und Maus des Zielrechners werden kurz getrennt.',
        menuHint:
          'Das virtuelle Laufwerk ist deaktiviert. Aktivieren Sie es unter Einstellungen - Gerät.'
      },
      add: {
        title: 'Medium hinzufügen',
        upload: 'ISO oder IMG hochladen',
        fromUrl: 'Von einer URL herunterladen',
        url: 'Abbild-URL',
        urlRequired: 'Bitte geben Sie eine URL ein',
        onlyImages: 'Nur .iso- und .img-Dateien können eingebunden werden',
        dropHint: 'Ziehen Sie eine ISO oder IMG hierher oder klicken Sie, um eine Datei zu wählen',
        cdromOnlyIso: 'Ein CD-ROM kann nur ein ISO / UDF-9660-Abbild enthalten',
        uploadFailed: 'Hochladen fehlgeschlagen',
        downloadFailed: 'Herunterladen fehlgeschlagen',
        uploading: '{{name}} wird hochgeladen',
        downloading: '{{name}} wird heruntergeladen',
        done: '{{name}} ist bereit',
        checksumFailed: 'Das Image stimmt nicht mit der SHA-256-Prüfsumme überein.',
        blocked:
          'Der Zielrechner verwendet die Festplatte. Werfen Sie das virtuelle USB-Laufwerk auf dem Zielrechner sicher aus und schalten Sie anschließend den Schalter „Virtueller Datenträger“ unter Einstellungen > Gerät aus.'
      },
      files: {
        reasonNotInitialized:
          'Das virtuelle USB-Laufwerk ist nicht initialisiert. Initialisieren Sie es unter Einstellungen > Gerät, bevor Sie Dateien freigeben.',
        selectMedium: 'Wählen Sie ein Medium, um die darin enthaltenen Dateien zu durchsuchen.',
        reasonAttached: 'Dieses Medium ist mit dem Zielrechner verbunden.',
        reasonAttachedHint:
          'Werfen Sie das virtuelle USB-Laufwerk auf dem Zielrechner sicher aus und schalten Sie anschließend den Schalter „Virtueller Datenträger“ unter Einstellungen > Gerät aus.',
        reasonNotBrowsable:
          'Dieses Medium enthält ein Dateisystem, das der Dateimanager nicht öffnen kann.',
        reasonNoLoop:
          'Dieses Gerät kann keine Datenträgerabbilder einbinden (keine Loop-Unterstützung).',
        reasonISOImage:
          'Ein ISO-9660-Abbild lässt sich hier nicht durchsuchen. Es kann weiterhin am Zielrechner eingebunden werden.',
        empty: 'Dieser Ordner ist leer',
        noMatch: 'Keine Datei entspricht der Suche.',
        free: '{{size}} frei',
        search: 'Diesen Ordner durchsuchen',
        sort: {
          modified: 'Geändert',
          name: 'Name',
          size: 'Größe',
          type: 'Typ',
          asc: 'Aufsteigend',
          desc: 'Absteigend'
        },
        view: {
          list: 'Listenansicht',
          grid: 'Symbolansicht'
        },
        folder: 'Ordner',
        itemCount: 'Elemente: {{count}}',
        itemCount_one: '{{count}} Element',
        itemCount_other: '{{count}} Elemente',
        dropHint: 'Dateien hier ablegen, um sie in diesen Ordner hochzuladen',
        up: 'Zum übergeordneten Ordner',
        uploadButton: 'Hochladen',
        newFolder: 'Neuer Ordner',
        newFolderPlaceholder: 'Ordnername',
        create: 'Erstellen',
        upload: 'Eine Datei auf das Medium hochladen',
        deleteFolderTitle: 'Ordner löschen',
        deleteFolderConfirm:
          'Den Ordner {{name}} mit seinem gesamten Inhalt löschen? Dies kann nicht rückgängig gemacht werden.',
        uploading: 'Wird hochgeladen…',
        uploadFailed: 'Hochladen fehlgeschlagen',
        download: 'Herunterladen',
        delete: 'Löschen',
        deleteTitle: 'Datei löschen',
        deleteConfirm: '{{name}} vom Medium löschen?'
      },
      yes: 'Ja',
      no: 'Nein'
    },
    head: {
      storage: 'Abbilder & Dateifreigabe',
      desktop: 'Entfernter Desktop',
      login: 'Anmelden',
      changePassword: 'Passwort ändern',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Kopieren in die Zwischenablage nicht möglich. Erlauben Sie den Zugriff auf die Zwischenablage und versuchen Sie es erneut.'
    },
    auth: {
      login: 'Anmelden',
      language: 'Sprache',
      placeholderUsername: 'Benutzername',
      placeholderPassword: 'Passwort',
      placeholderPassword2: 'Bitte Passwort erneut eingeben',
      noEmptyUsername: 'Benutzername benötigt',
      noEmptyPassword: 'Passwort benötigt',
      noAccount:
        'Abfragen der Benutzerdaten fehlgeschlagen, bitte die Seite neu laden oder Passwort zurücksetzen',
      invalidUser: 'Falscher Benutzername oder falsches Passwort',
      error: 'Unerwarteter Fehler',
      twoFactorTitle: 'Verifizierung erforderlich',
      twoFactorDesc:
        'Geben Sie Ihren Authentifizierungscode oder unbenutzten Wiederherstellungscode ein, um sich anzumelden.',
      sensitiveActionDesc:
        'Geben Sie einen Authentifizierungscode oder unbenutzten Wiederherstellungscode ein, um fortzufahren.',
      placeholderAuthCode: 'Authentifizierungscode',
      noEmptyAuthCode: 'Authentifizierungscode erforderlich',
      verify: 'Verifizieren',
      backToLogin: 'Zurück zur Anmeldung',
      verificationFailed: 'Verifizierung fehlgeschlagen',
      twoFactorEnabledLogout: {
        message:
          'Die Zwei-Faktor-Authentifizierung wurde aktiviert. Aus Sicherheitsgründen wurden alle Sitzungen abgemeldet. Bitte melden Sie sich erneut an.'
      },
      rateLimited: 'Zu viele fehlgeschlagene Versuche. Versuchen Sie es in {{seconds}} s erneut.',
      recoveryCodeUsed:
        'Der für diese Anmeldung verwendete Wiederherstellungscode wurde ungültig gemacht.',
      changePassword: 'Passwort ändern',
      changePasswordDesc: 'Für die Sicherheit Ihres Geräts ändern Sie bitte das Passwort!',
      passwordVerificationRequired:
        'Bitte kehren Sie zur Kontoseite in den Einstellungen zurück und verifizieren Sie diese Passwortänderung erneut.',
      passwordVerificationExpired:
        'Die Verifizierung der Passwortänderung ist abgelaufen. Bitte kehren Sie zur Kontoseite in den Einstellungen zurück und verifizieren Sie sie erneut.',
      returnToSettings: 'Zurück zu den Einstellungen',
      differentPassword: 'Passwörter stimmen nicht überein',
      illegalUsername: 'Benutzername enthält ungültige Zeichen',
      illegalPassword: 'Passwort enthält ungültige Zeichen',
      forgetPassword: 'Passwort vergessen',
      ok: 'OK',
      cancel: 'Abbrechen',
      savePassword: 'Passwort speichern',
      loginButtonText: 'Anmelden',
      tips: {
        reset1: 'Web-Standardkonto:',
        reset2: 'SSH-Standardkonto:',
        reset3: 'Informationen zum Zurücksetzen der Passwörter finden Sie unter ',
        changeReset: 'Informationen zum Zurücksetzen des Web-Anmeldepassworts finden Sie unter ',
        resetLink: 'Zurücksetzen'
      }
    },
    wifi: {
      errors: {
        connect:
          'Verbindung mit Wi-Fi nicht möglich. Prüfen Sie Netzwerkname, Passwort und Wi-Fi-Status des Geräts und versuchen Sie es erneut.'
      },
      title: 'Wi-Fi',
      description: 'Wi-Fi Konfiguration für NanoKVM',
      submitted: {
        title: 'Einen Moment bitte – NanoKVM verbindet sich mit deinem WLAN',
        tipsTitle: 'Wie geht es weiter?',
        tipSwitch: 'Du kannst dein Handy jetzt wieder mit deinem normalen WLAN verbinden.',
        tipCheck:
          'Sieh gleich auf dem NanoKVM-Bildschirm nach der neuen IP-Adresse und öffne sie im Browser.',
        tipRetry:
          'Wenn nach einiger Zeit nichts passiert, prüfe Netzwerkname und Passwort und versuche es erneut.'
      },
      invalidMode:
        'Der aktuelle Modus unterstützt keine Netzwerkeinrichtung. Bitte gehen Sie zu Ihrem Gerät und aktivieren Sie den Wi-Fi-Konfigurationsmodus.',
      confirmBtn: 'OK',
      finishBtn: 'Fertig',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Passwort',
      steps: {
        settings: 'Einstellungen',
        wifi: 'Wi-Fi',
        enable: 'Aktivieren'
      },
      scan: {
        title: 'Wi-Fi-Netzwerk auswählen',
        description: 'Wählen Sie ein Netzwerk in der Nähe, um fortzufahren.',
        loading: 'Suche nach Wi-Fi-Netzwerken in der Nähe…',
        empty: 'Keine Wi-Fi-Netzwerke in der Nähe gefunden.',
        failed: 'Netzwerksuche fehlgeschlagen. Geben Sie die Daten manuell ein.',
        retryBtn: 'Erneut suchen',
        manualBtn: 'Netzwerk manuell eingeben',
        backBtn: 'Zurück zu Netzwerken',
        connectTitle: 'Mit {{ssid}} verbinden',
        connecting: 'Verbindung wird hergestellt…',
        viewSimple: 'Zur einfachen Ansicht wechseln',
        viewDetailed: 'Zur detaillierten Ansicht wechseln'
      },
      ap: {
        authTitle: 'Authentifizierung erforderlich',
        authDescription: 'Bitte geben Sie das Passwort AP ein, um fortzufahren',
        authFailed: 'Ungültiges AP Passwort',
        passPlaceholder: 'AP Passwort',
        verifyBtn: 'Überprüfen'
      },
      qr: {
        cameraBtn: 'Foto aufnehmen',
        pickBtn: 'Screenshot wählen',
        scanHint:
          'Machen Sie ein Foto eines Wi-Fi-QR-Codes oder wählen Sie einen Screenshot davon aus, z. B. den Code, den ein Telefon beim Teilen seines Wi-Fi anzeigt.',
        decoding: 'QR-Code wird gelesen…',
        notFound:
          'In diesem Foto wurde kein QR-Code gefunden. Halten Sie den Code zentriert und scharf, und versuchen Sie es erneut.',
        invalid: 'Dieser QR-Code enthält keine verwendbaren Wi-Fi-Daten.',
        pickerUnavailable:
          'Es öffnet sich nichts? Dieses Anmeldefenster blockiert möglicherweise die Kamera und Fotos. Öffnen Sie stattdessen diese Adresse im Browser Ihres Telefons:',
        confirmTitle: 'NanoKVM mit {{ssid}} verbinden?',
        confirmDescription:
          'Diese Netzwerkdaten stammen aus einem QR-Code. Überprüfen Sie sie, bevor Sie sich verbinden.',
        openNetwork: 'Dieses Netzwerk hat kein Passwort.',
        connectBtn: 'Verbinden',
        otherBtn: 'Anderes Netzwerk auswählen'
      }
    },
    notification: {
      browser: {
        title: 'Browser-Empfehlung',
        description:
          'Für die beste Benutzererfahrung empfehlen wir die Verwendung des Chrome-Browsers. Obwohl wir uns um Kompatibilität mit allen gängigen Browsern bemühen, funktionieren einige erweiterte Funktionen außerhalb von Chrome möglicherweise nicht wie vorgesehen.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Bildschirm',
      video: 'Video Modus',
      quality: 'Qualität',
      qualityAuto: 'Automatisch',
      qualityLossless: 'Verlustfrei',
      qualityHigh: 'Hoch',
      qualityMedium: 'Mittel',
      qualityLow: 'Niedrig',
      scale: 'Skala',
      noSignal: 'Kein Videosignal',
      captureDisabled:
        'Die Videoerfassung ist deaktiviert. Aktivieren Sie sie unter „Einstellungen > Gerät > Videoerfassung“.',
      inconsistentVideoMode: 'Wiedergabe pausiert. Ein anderer Videomodus wird abgespielt.',
      bandwidthWarning:
        'Die Netzwerkbandbreite reicht möglicherweise nicht für die aktuellen Videoeinstellungen aus. Reduzieren Sie die Bitrate oder Bildrate.',
      webrtcConnectionFailed:
        'Die WebRTC-Verbindung konnte nicht hergestellt werden. Wechseln Sie zu einem Direct-Videomodus.',
      webrtcUnavailable: {
        title: 'WebRTC ist nicht verfügbar',
        description: 'Aktivieren Sie WebRTC im Browser oder wechseln Sie zum Direct-Videomodus.'
      },
      upload: 'Hochladen',
      advanced: 'Erweitert',
      loading: 'Lädt'
    },
    volume: {
      title: 'Lautstärke'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon nicht montiert',
      tip1: 'Das virtuelle Mikrofon ist nicht auf dem Remote-Host gemountet. Möchten Sie es jetzt mounten und aktivieren?',
      tip2: 'Sie können dies auch später unter „Einstellungen > Gerät > Virtuelles Audio“ verwalten.',
      confirm: 'Aktivieren',
      cancel: 'Abbrechen',
      noDevice:
        'Auf diesem Gerät wurde kein Mikrofon erkannt. Schließen Sie ein Mikrofon an und versuchen Sie es erneut.',
      permissionDenied:
        'Der Mikrofonzugriff wurde verweigert. Erlauben Sie den Zugriff in den Browsereinstellungen.',
      notReadable:
        'Das Mikrofon wird von einer anderen Anwendung verwendet oder kann nicht gelesen werden.',
      overconstrained: 'Das Mikrofon unterstützt die erforderlichen Audioeinstellungen nicht.',
      securityBlocked:
        'Der Mikrofonzugriff ist auf dieser Seite blockiert. Prüfen Sie die Sicherheitseinstellungen des Browsers.',
      aborted: 'Das Starten des Mikrofons wurde abgebrochen. Versuchen Sie es erneut.',
      unsupported: 'Dieser Browser oder diese Verbindungsart unterstützt keine Mikrofonaufnahme.',
      enableFailed: 'Das Mikrofon konnte nicht aktiviert werden. Versuchen Sie es erneut.',
      disconnected: 'Das Mikrofon wurde getrennt. Die Mikrofonübertragung wurde deaktiviert.'
    },
    power: {
      title: 'Stromversorgung',
      showConfirm: 'Bestätigung',
      showConfirmTip: 'Diese Aktionen benötigen eine zusätzliche Bestätigung',
      powerShort: 'Power (Kurzer Klick)',
      powerLong: 'Power (Langer Klick)',
      powerConfirm: 'Power-Aktion durchführen?',
      okBtn: 'Ja',
      cancelBtn: 'Nein'
    },
    keyboard: {
      errors: {
        paste:
          'Text konnte nicht an das Remote-Gerät gesendet werden. Prüfen Sie die KVM-Verbindung und versuchen Sie es erneut.'
      },
      title: 'Tastatur',
      paste: 'Einfügen',
      tips: 'Der Text wird mit der ausgewählten Tastaturbelegung des Zielgeräts eingegeben.',
      placeholder: 'Bitte eingeben',
      submit: 'Senden',
      layouts: 'Tastaturen',
      currentKeyboard: 'Aktuelle Tastatur',
      virtual: 'Virtuelle Tastatur anzeigen',
      canadianMultilingual: 'Kanadisch (mehrsprachiger Standard)',
      layoutError: 'Auf die Tastaturbelegung des Zielgeräts konnte nicht zugegriffen werden.',
      swapCommandCtrl: 'Cmd und Strg vertauschen',
      shortcut: {
        title: 'Verknüpfungen',
        custom: 'Benutzerdefiniert',
        capture: 'Klicken Sie hier, um die Verknüpfung zu erfassen',
        clear: 'Löschen',
        save: 'Speichern',
        captureTips:
          'Das Erfassen systemweiter Tasten (z. B. der Windows-Taste) erfordert die Vollbildberechtigung.',
        enterFullScreen: 'Vollbildmodus umschalten.'
      },
      leaderKey: {
        title: 'Leader-Taste',
        desc: 'Browserbeschränkungen umgehen und Systemverknüpfungen direkt an den Remote-Host senden.',
        howToUse: 'Verwendung',
        simultaneous: {
          title: 'Simultanmodus',
          desc1: 'Halten Sie die Leader-Taste gedrückt und drücken Sie dann die Tastenkombination.',
          desc2: 'Intuitiv, kann jedoch zu Konflikten mit Systemverknüpfungen führen.'
        },
        sequential: {
          title: 'Sequenzieller Modus',
          desc1:
            'Drücken Sie die Leader-Taste → drücken Sie die Tastenkombination nacheinander → drücken Sie erneut die Leader-Taste.',
          desc2: 'Erfordert mehr Schritte, vermeidet jedoch vollständig Systemkonflikte.'
        },
        enable: 'Leader-Taste aktivieren',
        tip: 'Wenn diese Taste als Leader-Taste zugewiesen wird, dient sie ausschließlich als Auslöser für Tastenkombinationen und verliert ihr Standardverhalten.',
        placeholder: 'Bitte drücken Sie die Leader-Taste',
        shiftRight: 'Rechts Shift',
        ctrlRight: 'Rechts Ctrl',
        metaRight: 'Rechts Win',
        submit: 'Senden',
        recorder: {
          rec: 'REC',
          activate: 'Tasten aktivieren',
          input: 'Bitte drücken Sie die Tastenkombination...'
        }
      }
    },
    mouse: {
      title: 'Maus',
      mode: 'Maus Modus',
      absolute: 'Folgende Maus',
      absoluteAndroid: 'Folgende Maus (bessere Android-Kompatibilität)',
      touchScreen: 'Multi-Touchscreen',
      relative: 'Exklusive Maus',
      direction: 'Scrollrichtung',
      scrollUp: 'Nach oben scrollen',
      scrollDown: 'Scrollen Sie nach unten',
      speed: 'Scrollgeschwindigkeit',
      fast: 'Schnell',
      slow: 'Langsam',
      requestPointer:
        'Relativer Modus aktiv. Klicken Sie auf den Desktop um den Mauszeiger zu sehen.',
      repairIphoneDrag: 'iPhone-Ziehen reparieren',
      resetHid: 'HID zurücksetzen',
      touchpadGuide: {
        title: 'Touchpad-Anleitung',
        scope: 'Gilt für normale Zeigereingaben, wenn der Mausmodus relativ ist.',
        swipeTitle: 'Wischen zum Bewegen',
        swipeDesc:
          'Wischen Sie innerhalb des aktiven Bildschirmbereichs, um den Remote-Zeiger zu bewegen.',
        tapTitle: 'Tippen zum Klicken',
        tapDesc: 'Ein kurzes Tippen sendet einen Linksklick.',
        holdTitle: 'Halten für gedrückte linke Taste',
        holdDesc:
          'Berühren und halten Sie den Finger etwa 1 Sekunde still, um die linke Maustaste gedrückt zu halten.',
        dragTitle: 'Nach dem Halten bewegen zum Ziehen',
        dragDesc:
          'Sobald das Halten aktiv ist, bewegen Sie den Finger, um mit gedrückter linker Taste zu ziehen.'
      },
      hidOnly: {
        errors: {
          load: 'Status des HID-Only-Modus konnte nicht geladen werden. Versuchen Sie es in Kürze erneut.',
          update:
            'Es konnte nicht bestätigt werden, ob der HID-Only-Modus aktualisiert wurde. Öffnen Sie diese Einstellung erneut, prüfen Sie den Status und versuchen Sie es erneut.'
        },
        title: 'HID-Only Modus',
        desc: 'Wenn Ihre Maus und Tastatur nicht mehr reagieren und das Zurücksetzen der HID-Verbindung nicht hilft, könnte es sich um ein Kompatibilitätsproblem zwischen dem NanoKVM und dem Gerät handeln. Versuchen Sie, den HID-Only Modus zu aktivieren, um die Kompatibilität zu verbessern.',
        tip1: 'Die Aktivierung des HID-Only Modus entfernt das virtuelle U-Laufwerk und das virtuelle Netzwerk.',
        tip2: 'Im HID-Only Modus ist das Einbinden von Systemabbilder deaktiviert.',
        enable: 'HID-Only Modus aktivieren',
        disable: 'HID-Only Modus deaktivieren'
      }
    },
    preview: {
      title: 'Vorschau'
    },
    image: {
      storageReadOnly:
        'Der Dateispeicher ist nicht beschreibbar. Bitte schalten Sie das virtuelle Laufwerk aus.',
      storageBusy:
        'Der Speicher wird verwendet. Beenden oder stornieren Sie die Dateiübertragung, bevor Sie das virtuelle Laufwerk umschalten.',
      title: 'Abbilder',
      loading: 'Lädt...',
      empty: 'Nichts gefunden',
      upload: 'Bild hochladen',
      dropFile: 'Abbild hier ablegen oder zum Auswählen einer Datei klicken',
      uploadAction: 'Hochladen',
      cancelAction: 'Abbrechen',
      mount: 'Abbild einbinden',
      mounted: 'montiert',
      uploadError: 'Upload fehlgeschlagen, bitte versuchen Sie es erneut',
      uploadInsufficientStorage: 'Nicht genügend Speicherplatz',
      uploadTooLarge: 'Image überschreitet das Größenlimit',
      cancelUploadError: 'Upload konnte nicht abgebrochen werden',
      mountMode: 'Mount-Modus',
      massStorage: 'Massenspeicher',
      cdRom: 'CD ROM',
      readOnly: 'Schreibgeschützt',
      mountFailed: 'Einbinden fehlgeschlagen',
      mountDesc:
        'In einigen Systemen ist es notwendig, die virtuelle Festplatte auf dem entfernten Host auszuwerfen, bevor das Image eingebunden werden kann.',
      unmountFailed: 'Aushängen fehlgeschlagen',
      unmountDesc:
        'In einigen Systemen muss das Abbild auf dem Remote-Host manuell ausgeworfen werden, bevor es ausgehängt wird.',
      refresh: 'Bilder aktualisieren',
      download: 'Bild herunterladen',
      attention: 'Achtung',
      deleteConfirm: 'Sind Sie sicher, dass Sie dieses Bild löschen möchten?',
      deleteFailed: 'Das Bild konnte nicht gelöscht werden',
      okBtn: 'Ja',
      cancelBtn: 'Nein'
    },
    script: {
      title: 'Skripte',
      upload: 'Hochladen',
      run: 'Ausführen',
      runBackground: 'Im Hintergrund ausführen',
      runFailed: 'Ausführung fehlgeschlagen',
      uploadFailed: 'Das Skript konnte nicht hochgeladen werden',
      deleteFailed: 'Das Skript konnte nicht gelöscht werden',
      attention: 'Achtung',
      delDesc: 'Möchten Sie diese Datei wirklich löschen?',
      confirm: 'Ja',
      cancel: 'Nein',
      delete: 'Löschen',
      close: 'Schliessen'
    },
    terminal: {
      title: 'Terminal',
      confirm: 'OK',
      submit: 'Senden',
      username: 'Benutzername',
      password: 'Passwort',
      authCode: 'Authentifizierungscode',
      authorizeTitle: 'Web-Terminal autorisieren',
      codeAccountHint:
        'Geben Sie einen Authentifizierungscode oder unbenutzten Wiederherstellungscode ein.',
      authFailed: 'Authentifizierung fehlgeschlagen'
    },
    wol: {
      errors: {
        send: 'Weckbefehl konnte nicht gesendet werden. Prüfen Sie MAC-Adresse und Netzwerkverbindung und versuchen Sie es erneut.',
        loadFailed:
          'Die Wake-on-LAN-Geräteliste konnte nicht geladen werden. Prüfen Sie, ob NanoKVM online ist, und versuchen Sie es erneut.',
        renameFailed:
          'Dieses Gerät konnte nicht umbenannt werden. Prüfen Sie die MAC-Adresse und versuchen Sie es erneut.',
        deleteFailed: 'Dieses Gerät konnte nicht gelöscht werden. Versuchen Sie es in Kürze erneut.'
      },
      title: 'Wake-on-LAN',
      sending: 'Sende Befehl...',
      sent: 'Befehl gesendet',
      input: 'Bitte MAC Adresse eingeben',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (optional)',
      sha256Placeholder: 'Geben Sie eine 64-stellige SHA-256-Prüfsumme ein',
      invalidSHA256: 'SHA-256 muss eine 64-stellige Hexadezimalzeichenfolge sein',
      title: 'Systemabbild Downloader',
      input: 'Bitte geben Sie die URL für das Remote-Systemabbild ein',
      ok: 'OK',
      disabled:
        '/data Partition ist nur-lesbar, daher kann das Systemabbild nicht heruntergeladen werden',
      downloadingFile: 'Wird heruntergeladen: {{file}}',
      downloadingProgress: 'Wird heruntergeladen ({{percentage}}): {{file}}',
      failed: 'Fehlgeschlagen',
      insufficientStorage: 'Nicht genügend Speicherplatz',
      remaining: 'Noch {{time}}',
      cancel: 'Abbrechen',
      cancelFailed: 'Download konnte nicht abgebrochen werden',
      pause: 'Pausieren',
      resume: 'Fortsetzen',
      pauseFailed: 'Download konnte nicht pausiert werden',
      resumeFailed: 'Download konnte nicht fortgesetzt werden'
    },
    settings: {
      back: 'Zurück',
      close: 'Schliessen',
      ai: {
        errors: {
          loadServices:
            'Status der KI-Dienste konnte nicht geladen werden. Prüfen Sie, ob NanoKVM online ist, und versuchen Sie es erneut.',
          loadMcpConfig:
            'MCP-Dienstkonfiguration konnte nicht geladen werden. Versuchen Sie es in Kürze erneut.',
          updateMcpService:
            'Status des MCP-Dienstes konnte nicht aktualisiert werden. Prüfen Sie den aktuellen Status und versuchen Sie es erneut.',
          accessMcpApiKey:
            'MCP-API-Schlüssel konnte nicht abgerufen werden. Versuchen Sie es erneut und führen Sie bei Aufforderung die Verifizierung durch.',
          regenerateMcpApiKey:
            'MCP-API-Schlüssel konnte nicht neu generiert werden. Versuchen Sie es in Kürze erneut.',
          regeneratedMcpApiKeyUnavailable:
            'Der API-Schlüssel wurde neu generiert, konnte aber nicht angezeigt werden. Versuchen Sie, ihn erneut anzuzeigen. Der vorherige Schlüssel ist nicht mehr gültig.',
          loadMemoryFabricState:
            'Status des Memory-Fabric-Dienstes konnte nicht geladen werden. Versuchen Sie es in Kürze erneut.',
          updateMemoryFabricService:
            'Status des Memory-Fabric-Dienstes konnte nicht aktualisiert werden. Prüfen Sie den aktuellen Status und versuchen Sie es erneut.',
          loadScreenTimelapseState:
            'Status des Bildschirm-Zeitrafferdienstes konnte nicht geladen werden. Versuchen Sie es in Kürze erneut.',
          updateScreenTimelapseService:
            'Status des Bildschirm-Zeitrafferdienstes konnte nicht aktualisiert werden. Prüfen Sie den aktuellen Status und versuchen Sie es erneut.',
          openScreenTimelapse:
            'Bildschirm-Zeitraffer konnte nicht geöffnet werden. Versuchen Sie es in Kürze erneut.',
          loadScreenTimelapseRecordings:
            'Bildschirm-Zeitrafferaufnahmen konnten nicht geladen werden. Versuchen Sie es in Kürze erneut.',
          playScreenTimelapse:
            'Diese Zeitrafferaufnahme kann derzeit nicht abgespielt werden. Wählen Sie das Datum erneut oder versuchen Sie es später erneut.',
          screenTimelapseFullscreen:
            'Vollbildmodus konnte nicht geändert werden. Prüfen Sie die Browserberechtigungen und versuchen Sie es erneut.'
        },
        title: 'AI',
        mcpService: 'MCP-Dienst',
        mcpServiceDesc:
          'Authentifiziertes Gateway für Steuerungs-, Medien- und aktivierte Memory-Fabric-Werkzeuge',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Agent-Kontextspeicher und OCR lokal verarbeiten',
        memoryFabricRemoteRequiresMcp:
          'Aktivieren Sie oben den MCP-Dienst, um Speicher- und OCR-Werkzeuge remote verfügbar zu machen.',
        memoryFabricRemoteAvailable:
          'Speicher- und OCR-Werkzeuge sind über den obigen Endpunkt /api/mcp mit demselben API-Schlüssel verfügbar.',
        memoryFabricConfig: {
          title: 'Konfiguration',
          description:
            'Verwalten Sie ausschließlich die Einstellungen des Memory-Fabric-Standardmodells.',
          loadFailed: 'Die Memory-Fabric-Konfiguration konnte nicht geladen werden.',
          saveFailed: 'Die Memory-Fabric-Konfiguration konnte nicht gespeichert werden.',
          saved: 'Konfiguration gespeichert.',
          savedRestarted: 'Konfiguration gespeichert und Memory Fabric neu gestartet.',
          noChanges: 'Keine Änderungen an der Konfiguration.',
          reload: 'Neu laden',
          save: 'Speichern',
          unsaved: 'Ungespeicherte Änderungen',
          conflictTitle: 'Konfiguration wurde geändert',
          conflictDescription:
            'Die Konfiguration wurde an anderer Stelle geändert. Laden Sie vor dem erneuten Speichern die aktuelle Version.',
          conflictReload: 'Aktuelle Version laden',
          discardTitle: 'Ungespeicherte Änderungen verwerfen?',
          discardDescription:
            'Beim Neuladen wird das aktuelle Formular durch die zuletzt gespeicherte Konfiguration ersetzt.',
          discardConfirm: 'Verwerfen und neu laden',
          sections: {
            models: 'Modelle'
          },
          openAICompatibleHint:
            'Unterstützt OpenAI-kompatible Modellanbieter. Geben Sie den Modellnamen, die Basis-URL und den API-Schlüssel Ihres Anbieters ein.',
          fields: {
            model: 'Modell',
            baseUrl: 'Basis-URL',
            maxTokens: 'Maximale Tokenzahl',
            timeoutSeconds: 'Zeitlimit (Sekunden)'
          },
          apiKey: {
            title: 'API-Schlüssel',
            configured: 'Konfiguriert',
            notConfigured: 'Nicht konfiguriert',
            keep: 'Beibehalten',
            replace: 'Ersetzen',
            clear: 'Entfernen',
            replacement: 'Neuer API-Schlüssel',
            replacementPlaceholder: 'Neuen API-Schlüssel eingeben',
            clearHint:
              'Der gespeicherte API-Schlüssel wird entfernt; über die Umgebungsvariable kann weiterhin ein Schlüssel bereitgestellt werden.',
            required: 'Bitte geben Sie einen neuen API-Schlüssel ein.'
          },
          validation: {
            required: 'Dieses Feld ist erforderlich.',
            invalidUrl: 'Bitte geben Sie eine gültige HTTP- oder HTTPS-URL ein.'
          }
        },
        screenTimelapseService: 'Bildschirm-Zeitraffer',
        screenTimelapseServiceDesc:
          'Memory Fabric-Bildschirmverlauf wiedergeben und erfasste Bildschirminhalte durchsuchen',
        screenTimelapseSearchRequiresMemoryFabric:
          'Um aufgezeichnete Inhalte zu durchsuchen, aktivieren Sie auch Memory Fabric.',
        screenTimelapseOpen: 'Öffnen',
        screenTimelapseSearchPlaceholder: 'In Videoinhalten suchen...',
        screenTimelapseNoVideos: 'Keine Zeitraffervideos gefunden',
        screenTimelapseNoResults: 'Keine passenden Frames gefunden',
        screenTimelapseMultipleResults: 'Suchergebnisse',
        screenTimelapseUnavailable: 'Nicht verfügbar',
        screenTimelapsePlayerUnsupported: 'Dieser Browser kann Zeitraffervideo nicht abspielen',
        screenTimelapseSearchFailed: 'Suche fehlgeschlagen',
        screenTimelapseLoading: 'Zeitraffer wird geladen...',
        screenTimelapseSearchAction: 'Zeitraffer durchsuchen',
        screenTimelapsePlay: 'Zeitraffer abspielen',
        screenTimelapsePause: 'Zeitraffer pausieren',
        screenTimelapseEnterFullscreen: 'Vollbild öffnen',
        screenTimelapseExitFullscreen: 'Vollbild beenden',
        screenTimelapseFullscreenUnsupported:
          'Dieser Browser kann nicht in den Vollbildmodus wechseln',
        screenTimelapsePreviousDay: 'Vorheriger Tag',
        screenTimelapseNextDay: 'Nächster Tag',
        screenTimelapseDeleteDay: 'Aufzeichnung vom {{date}} löschen',
        screenTimelapseDelete: 'Löschen',
        screenTimelapseDeleteConfirmTitle: 'Zeitraffer-Aufzeichnung löschen?',
        screenTimelapseDeleteConfirmDesc:
          'Die Aufzeichnung vom {{date}} belegt {{size}}. Diese Aktion kann nicht rückgängig gemacht werden.',
        screenTimelapseDeleteSuccess: 'Aufzeichnung vom {{date}} wurde gelöscht',
        screenTimelapseDeleteFailed: 'Aufzeichnung vom {{date}} konnte nicht gelöscht werden',
        endpoint: 'Endpunkt',
        apiKey: 'API-Schlüssel',
        regenerate: 'Neu generieren',
        copyApiKey: 'API-Schlüssel kopieren',
        copyEndpoint: 'Endpunkt kopieren',
        showApiKey: 'API-Schlüssel anzeigen',
        regenerateConfirmTitle: 'API-Schlüssel neu generieren',
        regenerateConfirmDesc:
          'Bestehende MCP-Clients funktionieren nicht mehr, bis sie mit dem neuen Schlüssel aktualisiert wurden. Möchten Sie fortfahren?',
        okBtn: 'Ja',
        cancelBtn: 'Nein'
      },
      title: 'Einstellungen',
      about: {
        title: 'Über NanoKVM',
        information: 'Informationen',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applikations-Version',
        applicationTip: 'NanoKVM Web Applikations-Version',
        arch: 'Architektur',
        device_number: 'Gerätenummer',
        image: 'Systemabbild-Version',
        imageTip: 'NanoKVM Systemabbild-Version',
        deviceKey: 'Geräte Schlüssel',
        community: 'Gemeinschaft',
        hostname: 'Host-Name',
        hostnameUpdated: 'Hostname aktualisiert. Neustarten um zu übernehmen.',
        ipType: {
          Wired: 'Kabel',
          Wireless: 'Drahtlos',
          Other: 'Andere'
        }
      },
      appearance: {
        title: 'Erscheinungsbild',
        display: 'Bildschirm',
        language: 'Sprache',
        languageDesc: 'Wählen Sie die Sprache für die Benutzeroberfläche aus',
        languageChangeFailed: 'Sprache konnte nicht geändert werden',
        webTitle: 'Web Titel',
        webTitleDesc: 'Passen Sie den Web-Seite Titel an',
        menuBar: {
          title: 'Menüleiste',
          mode: 'Anzeigemodus',
          modeDesc: 'Menüleiste auf dem Bildschirm anzeigen',
          modeOff: 'Aus',
          modeAuto: 'Automatisch ausblenden',
          modeAlways: 'Immer sichtbar',
          icons: 'Untermenüsymbole',
          showIcon: 'Symbol anzeigen',
          activateOnConnect: 'Beim Verbinden aktivieren',
          stopSharing: 'Freigabe beenden',
          iconsDesc: 'Untermenüsymbole in der Menüleiste anzeigen'
        }
      },
      screen: {
        title: 'Bildschirm',
        videoMode: {
          title: 'Videomodus',
          description: 'Wählen Sie die Kodierungs- und Übertragungsmethode aus'
        },
        edid: {
          description: 'Anzeigeparameter auswählen',
          default: 'Standard',
          custom: 'Benutzerdefiniert',
          document: 'Dokument',
          powerConfirmTitle: 'Vor Aktualisierung der EDID bestätigen',
          powerConfirmDescription:
            'Schließen Sie zuerst die PD-Stromversorgung an. Die Aktualisierung der EDID kann die Anzeige kurz trennen; wenn das Gerät die Anzeige danach nicht korrekt erkennt, starten Sie es manuell neu.',
          powerConfirmOk: 'Fortfahren',
          powerConfirmOkCountdown: 'Fortfahren ({{seconds}})',
          powerConfirmCancel: 'Abbrechen',
          errors: {
            badRequest: 'Ungültige EDID-Anfrage',
            invalidFileName: 'Wählen Sie eine EDID-Datei im .bin-Format mit gültigem Dateinamen',
            fileTooLarge: 'Die EDID-Datei ist zu groß',
            invalidLength: 'Die EDID muss 128, 256, 384 oder 512 Byte groß sein',
            invalidHeader: 'Der EDID-Header ist ungültig',
            unsupportedVersion: 'Diese EDID-Version wird nicht unterstützt',
            extensionCountMismatch:
              'Die Anzahl der EDID-Erweiterungsblöcke stimmt nicht mit der Dateilänge überein',
            invalidChecksum: 'Die EDID-Prüfsumme ist ungültig',
            invalidCta: 'Die Struktur der CTA-Erweiterung ist ungültig',
            currentCannotOverwrite:
              'Die derzeit verwendete EDID kann nicht überschrieben werden. Wechseln Sie zuerst zu einer anderen EDID',
            saveFailed: 'Die EDID konnte nicht gespeichert werden',
            switchFailed: 'Die EDID konnte nicht gewechselt werden',
            uploadFailed: 'Die EDID konnte nicht hochgeladen werden',
            deleteFailed: 'Die EDID konnte nicht gelöscht werden',
            networkFailed: 'Netzwerkfehler. Versuchen Sie es erneut'
          }
        },
        rateControlMode: {
          title: 'Ratenkontrollmodus',
          description: 'Wählen Sie die Methode zur Steuerung der Bitrate aus',
          cbr: 'CBR (Konstante Bitrate): Priorisiert die Verbindungsstabilität. Minimiert Stottern, aber die Bildqualität kann bei Szenen mit viel Bewegung sinken.',
          vbr: 'VBR (Variable Bitrate): Priorisiert die Bildklarheit. Szenen mit vielen Bewegungen können zu Bandbreitenspitzen führen, die möglicherweise zu Latenz oder Stottern führen.'
        },
        bitrate: {
          title: 'Bitrate',
          description: 'Gleichen Sie die Bildschärfe mit der Bandbreitennutzung aus',
          lower:
            'Niedrigere Werte verringern die Latenz bei schlechten Netzwerken, beeinträchtigen jedoch die Bildqualität und können möglicherweise Unschärfe oder Artefakte verursachen.',
          higher:
            'Höhere Werte verbessern die Bildschärfe, verbrauchen jedoch mehr Bandbreite. Kann die Latenz erhöhen, wenn die Verbindung langsam ist.'
        },
        gop: {
          description: 'Legen Sie das Intervall zwischen vollständigen Keyframes fest',
          lower:
            'Niedrigere Werte senden häufiger Vollbilder. Dies beschleunigt die Wiederherstellung nach visuellen Artefakten, verbraucht jedoch mehr Bandbreite und reduziert die Bilddetails.',
          higher:
            'Höhere Werte optimieren inkrementelle Aktualisierungen. Dies verbessert die Bilddetails und spart Bandbreite, die Wiederherstellung von visuellen Artefakten dauert jedoch länger.'
        },
        quality: {
          title: 'Qualität',
          description: 'Stellen Sie die Anzeigebildqualität ein'
        },
        fps: {
          auto: 'Automatisch',
          description: 'Stellen Sie die Anzeigebildrate ein',
          tip1: 'Wenn Sie den Wert auf 0 setzen, wird der Auto-Modus aktiviert und die Anzeige mit der Bildrate des Remote-Hosts synchronisiert.',
          tip2: 'Durch die Verringerung der Bildrate wird die Bandbreitennutzung erheblich reduziert. Verringern Sie den Wert manuell, wenn das Netzwerk instabil ist oder keine hohen Bildraten erforderlich sind.'
        },
        scale: {
          title: 'Skala',
          description: 'Stellen Sie das Skalierungsverhältnis der Anzeige ein'
        }
      },
      device: {
        title: 'Gerät',
        virtualAudioConfirm: {
          title: 'Vor dem Umschalten von virtuellem Audio bestätigen',
          description:
            'Schließen Sie zuerst die PD-Stromversorgung an. Das Umschalten von virtuellem Audio kann die Anzeige kurz trennen; wenn das Gerät die Anzeige danach nicht korrekt erkennt, starten Sie es manuell neu.',
          ok: 'Umschalten fortsetzen',
          okCountdown: 'Umschalten fortsetzen ({{seconds}})',
          cancel: 'Abbrechen'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi konfigurieren',
          apMode: 'AP-Modus ist aktiviert, stellen Sie manuell eine Verbindung zum WLAN her',
          others: 'Andere...',
          connect: 'WLAN beitreten',
          connectDesc1: 'Bitte geben Sie die Netzwerk-SSID und das Passwort ein',
          connectDesc2: 'Bitte geben Sie das Passwort ein, um diesem Netzwerk beizutreten',
          disconnect: 'Möchten Sie das Netzwerk wirklich trennen?',
          ssid: 'Netzwerkname',
          password: 'Passwort',
          joinBtn: 'Beitreten',
          confirmBtn: 'OK',
          cancelBtn: 'Abbrechen',
          preset: {
            title: 'Wi-Fi-Voreinstellung',
            description:
              'Ein Ziel-WLAN vorspeichern; das Gerät verbindet sich beim Start automatisch damit',
            empty: 'Keine',
            addBtn: 'Voreinstellung hinzufügen',
            removeBtn: 'Voreinstellung entfernen',
            saveTitle: 'WLAN vorspeichern',
            saveDesc:
              'Zielnetzwerk jetzt speichern; das Gerät verbindet sich automatisch, sobald es das Netzwerk beim Start findet',
            saveBtn: 'Speichern',
            passwordPlaceholder: 'Passwort (bei offenen Netzwerken leer lassen)',
            saveFailed: 'WLAN-Voreinstellung konnte nicht gespeichert werden',
            removeFailed: 'WLAN-Voreinstellung konnte nicht entfernt werden',
            removeConfirm: 'Möchten Sie das vorgespeicherte WLAN wirklich entfernen?',
            warning:
              'Nach dem Speichern versucht das Gerät bei jedem Start, dieses WLAN zu verbinden, sobald es erkannt wird. Das kann das Verbinden mit anderen WLAN-Netzwerken beeinträchtigen. Die Voreinstellung bleibt aktiv, bis Sie sie ändern oder entfernen.'
          }
        },
        capture: {
          title: 'Videoerfassung',
          description: 'Videoerfassung aktivieren'
        },
        ssh: {
          description: 'Aktiviere entfernten SSH-Zugang',
          tip: 'Legen Sie nach dem Aktivieren im Terminal ein starkes Passwort für den root-Benutzer fest.'
        },
        swap: {
          errors: {
            load: 'Status des Auslagerungsspeichers konnte nicht geladen werden. Versuchen Sie es in Kürze erneut.',
            update:
              'Einstellung für Auslagerungsspeicher konnte nicht aktualisiert werden. Prüfen Sie den aktuellen Status und versuchen Sie es erneut.'
          },
          title: 'Swap-Speicher',
          description: 'Komprimierten Swap-Speicher aktivieren'
        },
        advanced: 'Erweiterte Einstellungen',
        staticIp: {
          title: 'Statische IP',
          config: 'Konfig',
          tip: 'Um einen Verbindungsverlust zu verhindern, wird empfohlen, die Konfiguration durchzuführen, während Sie physischen Zugriff auf das Gerät haben.',
          document: 'Überprüfen Sie das Dokument.',
          enable: 'Statische IP aktivieren',
          placeholder: 'Format: IP/CIDR [Gateway]\nBeispiel: 192.168.1.10/22',
          notEmpty: 'IP-Adresse darf nicht leer sein.',
          notValid: 'IP-Adressformat ist ungültig.',
          failed: 'Konfiguration fehlgeschlagen.',
          success:
            'Konfiguration abgeschlossen. Bitte prüfen Sie, ob die IP-Adresse verfügbar ist.',
          okBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        usbNetwork: {
          title: 'USB-Netzwerk',
          mode: 'Betriebsart',
          server: 'DHCP-Server',
          serverDesc:
            'Dem entfernten Host über den integrierten DHCP-Server eine IP-Adresse zuweisen.',
          client: 'DHCP-Client',
          clientDesc: 'Über DHCP eine IP-Adresse vom entfernten Host beziehen.',
          forwarding: 'IP-Weiterleitung',
          forwardingTip:
            'Damit der entfernte Host über NanoKVM auf das Internet zugreifen kann, aktivieren Sie die IP-Weiterleitung.',
          manualIp: 'IP-Adresse manuell festlegen',
          ipv4Placeholder: 'IPv4/CIDR, z. B. 10.10.10.1/24',
          invalidIpv4: 'Geben Sie eine gültige IPv4/CIDR-Adresse ein.',
          loadFailed: 'Die USB-Netzwerkkonfiguration konnte nicht geladen werden.',
          failed: 'Konfiguration fehlgeschlagen.',
          success: 'Konfiguration abgeschlossen.',
          applyBtn: 'Übernehmen',
          cancelBtn: 'Abbrechen'
        },
        usbId: {
          title: 'USB-Gerätekennung',
          config: 'Konfigurieren',
          tip: 'Benutzerdefinierte USB-IDs ändern, wie der Host dieses Gerät erkennt. Beim Anwenden wird das USB-Gadget neu gestartet; Tastatur, Maus oder Netzwerk können kurz unterbrochen werden.',
          enable: 'Benutzerdefinierte USB-Gerätekennung aktivieren',
          vid: 'Vendor-ID (VID)',
          pid: 'Product-ID (PID)',
          bcdDevice: 'Geräteversion (bcdDevice)',
          serialnumber: 'Seriennummer',
          manufacturer: 'Hersteller',
          product: 'Produkt',
          configurationName: 'Name der USB-Konfiguration',
          vidPlaceholder: 'z. B. 0x046D',
          pidPlaceholder: 'z. B. 0x0A4E',
          bcdDevicePlaceholder: 'z. B. 0x0521 (optional)',
          serialnumberPlaceholder: 'z. B. 2214GG01KWF8 (optional)',
          manufacturerPlaceholder: 'z. B. Logitech (optional)',
          productPlaceholder: 'z. B. Logitech P710e (optional)',
          notEmpty: 'VID und PID dürfen nicht leer sein.',
          notValid: 'VID/PID-Format ist ungültig. Hex wie 0x046D verwenden.',
          bcdNotValid: 'Ungültiges bcdDevice-Format. Hex wie 0x0521 verwenden oder leer lassen.',
          failed: 'Konfiguration fehlgeschlagen.',
          success: 'Konfiguration abgeschlossen. USB-Gadget neu gestartet.',
          okBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        diskInquiry: {
          title: 'USB-Laufwerksname',
          config: 'Konfigurieren',
          tip: 'Name des virtuellen USB-Sticks, wie er auf dem Remote-Computer angezeigt wird. Speichern startet USB neu; Tastatur, Maus oder Netzwerk können kurz unterbrochen werden.',
          enable: 'Eigenen USB-Laufwerksnamen verwenden',
          placeholder: 'z. B. LogitechFlash Drive     1.00',
          notEmpty: 'Name darf nicht leer sein.',
          notValid: 'Name ist ungültig. Bis zu 28 druckbare ASCII-Zeichen ohne Zeilenumbrüche.',
          failed: 'Konfiguration fehlgeschlagen.',
          success: 'Konfiguration abgeschlossen. USB-Gadget neu gestartet.',
          okBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        diskInquiryCdrom: {
          title: 'Virtueller CD-Laufwerksname',
          config: 'Konfigurieren',
          tip: 'Name des virtuellen CD/DVD-Laufwerks auf dem Remote-Computer. Speichern startet USB neu; Tastatur, Maus oder Netzwerk können kurz unterbrochen werden.',
          enable: 'Eigenen CD-Laufwerksnamen verwenden',
          placeholder: 'z. B. LogitechOptical Drive   1.00',
          notEmpty: 'Name darf nicht leer sein.',
          notValid: 'Name ist ungültig. Bis zu 28 druckbare ASCII-Zeichen ohne Zeilenumbrüche.',
          failed: 'Konfiguration fehlgeschlagen.',
          success: 'Konfiguration abgeschlossen. USB-Gadget neu gestartet.',
          okBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        uac2FunctionName: {
          title: 'USB-Audioname',
          config: 'Konfigurieren',
          tip: 'Name des virtuellen USB-Mikrofon-/Audiogeräts auf dem Remote-Computer. Speichern startet USB neu; Tastatur, Maus oder Netzwerk können kurz unterbrochen werden.',
          enable: 'Eigenen USB-Audionamen verwenden',
          placeholder: 'z. B. Logitech P710e',
          notEmpty: 'Name darf nicht leer sein.',
          notValid: 'Name ist ungültig. Bis zu 64 druckbare Zeichen ohne Zeilenumbrüche.',
          failed: 'Konfiguration fehlgeschlagen.',
          success: 'Konfiguration abgeschlossen. USB-Gadget neu gestartet.',
          okBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        lowPower: {
          title: 'Energiesparmodus',
          description: 'Reduzieren Sie den Stromverbrauch im Leerlauf',
          tips: 'Tipps',
          tip1: 'Wenn aktiviert, wird die Videoerfassung während Leerlaufzeiten angehalten, um den Stromverbrauch und die Wärmeentwicklung zu reduzieren.',
          tip2: 'Das Fortsetzen der Videoerfassung kann eine Neuinitialisierung erfordern, während der das Video vorübergehend nicht verfügbar sein kann.',
          okBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        webcamUsbDisplay: 'Deaktiviere den USB-Displaymodus, bevor du die Webcam aktivierst.',
        webcamEndpoints: 'Die Webcam benötigt zwei freie USB-IN-Endpunkte.',
        mouseJiggler: {
          noMouse: 'Der Maus-Jiggler kann mit dieser Konfiguration nicht laufen. Wähle einen Mausmodus oder deaktiviere den Einzelendpunktmodus.',
          title: 'Mausaktivitäts-Simulator',
          description: 'Verhindert, dass der remote Host in den Energiesparmodus wechselt',
          disable: 'Deaktivieren',
          absolute: 'Absoluter Modus',
          relative: 'Relativer Modus'
        },
        fingerRobot: {
          title: 'Fingerroboter',
          description: 'Fingerroboter-Steuerung aktivieren'
        },
        mdns: {
          description: 'mDNS-Erkennungsdienst aktivieren'
        },
        hidOnly: 'Nur-HID-Modus',
        imageMounted: 'Bild montiert',
        disk: 'Virtuelle Festplatte',
        diskDesc: 'Binde das virtuelle U-Laufwerk an den entfernten Host',
        sdCard: {
          warning: 'Virtuelles Laufwerk initialisieren',
          tip1: 'Bei der ersten Aktivierung wird das virtuelle Laufwerk initialisiert. Vorhandene Abbilddateien und Daten des virtuellen Laufwerks auf dem Gerät werden gelöscht. Dies kann nicht rückgängig gemacht werden. Sichern Sie zuerst wichtige Dateien.',
          tip2: 'Initialisierung fortsetzen?',
          capacity: 'Kapazität',
          disable: 'Deaktivieren',
          disableWarning: 'Virtuelles Laufwerk deaktivieren?',
          disableTip:
            'Beim Deaktivieren werden alle Dateien auf dem virtuellen Laufwerk gelöscht und der belegte Gerätespeicher freigegeben. Bei der nächsten Aktivierung müssen Sie die Kapazität erneut auswählen und das Laufwerk initialisieren. Dies kann nicht rückgängig gemacht werden.'
        },
        network: 'Virtuelles Netzwerk',
        networkDesc: 'Binde die virtuelle Netzwerkkarte an den entfernten Host',
        mic: 'Virtuelles Audio',
        micDesc: 'Wählen Sie den USB-Modus für virtuelles Audio auf dem Remote-Host',
        virtualAudioMode: {
          off: 'Aus',
          mic: 'Nur Mikrofon',
          duplex: 'Mikrofon + Lautsprecher'
        },
        speaker: 'Virtueller Lautsprecher',
        speakerDesc:
          'Ersetzt nach der Aktivierung die standardmäßige DP-Audioausgabe, um Remote-Anrufe mit besserer Kompatibilität zu unterstützen. Ohne diese Option kann nur Medienaudio, jedoch kein Anrufaudio übertragen werden.',
        micTip:
          'Aufgrund von USB-Endpunktbeschränkungen wird durch die Aktivierung des virtuellen Mikrofons der USB-Sekundärbildschirm deaktiviert.\nUnd es wird empfohlen, diese Funktion während des BIOS-Vorgangs zu deaktivieren, um eine optimale Kompatibilität sicherzustellen.',
        reboot: 'Neustarten',
        rebootDesc: 'Sind Sie sicher dass Sie NanoKVM neustarten möchten?',
        rebootToFlashing: 'Im Flash-Modus neu starten',
        rebootToFlashingButton: 'Neustarten',
        rebootToFlashingTitle: 'Im Flash-Modus neu starten?',
        rebootToFlashingDesc:
          'NanoKVM wird neu gestartet und wechselt in den USB-Flash-Modus. Stellen Sie sicher, dass die USB-Schnittstelle zum Flashen verbunden und das Flash-Tool bereit ist.',
        datetime: {
          timezone: 'Zeitzone',
          datetime: 'DatumUhrzeit',
          dateAndTime: 'Datum und Uhrzeit',
          synchronize: 'Uhrzeit synchronisieren',
          lastSynchronization: 'Letzte Synchronisierung: ',
          notSynchronized: 'Die Systemzeit ist noch nicht synchronisiert',
          syncNow: 'Jetzt synchronisieren',
          manualSet: 'Manuell einstellen',
          manualSetTitle: 'Uhrzeit manuell einstellen',
          manualSetDescription: 'Nach der Änderung müssen Sie sich erneut anmelden.',
          useBrowserTime: 'Browserzeit verwenden',
          setTime: 'Uhrzeit einstellen',
          invalidTime: 'Geben Sie ein gültiges Datum und eine gültige Uhrzeit ein.',
          setFailed: 'Die Systemzeit konnte nicht eingestellt werden. Versuchen Sie es erneut.',
          setSucceeded: 'Die Systemzeit wurde aktualisiert. Melden Sie sich erneut an.'
        },
        dialBackground: {
          errors: {
            load: 'Zifferblatt-Hintergrundbilder konnten nicht geladen werden. Versuchen Sie es in Kürze erneut.',
            upload:
              'Hintergrundbild konnte nicht hochgeladen werden. Prüfen Sie, ob die Datei ein gültiges BMP-Bild ist, und versuchen Sie es erneut.',
            switch:
              'Zifferblatt-Hintergrund konnte nicht gewechselt werden. Aktualisieren Sie die Bildliste und versuchen Sie es erneut.'
          },
          title: 'Zifferblatt-Hintergrund',
          description: 'BMP-Bild aus /kvmcomm/ui/bg/ als Zifferblatt-Hintergrund auswählen',
          upload: 'Hochladen',
          loading: 'Lädt...',
          empty: 'Keine BMP-Bilder gefunden',
          current: 'Aktuell',
          onlyBmp: 'Nur BMP-Bilder werden unterstützt',
          uploaded: 'Upload abgeschlossen',
          switched: 'Hintergrund gewechselt'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nein'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Tailscale neu starten?',
        stop: 'Tailscale stoppen?',
        stopDesc: 'Von Tailscale abmelden und automatischen Start beim Booten deaktivieren.',
        loading: 'Lädt...',
        notInstall: 'Tailscale nicht gefunden! Bitte installieren.',
        install: 'Installieren',
        installing: 'Installiere',
        failed: 'Installation fehlgeschlagen',
        retry: 'Bitte Seite neu laden und erneut versuchen oder manuelle Installation versuchen.',
        download: 'Laden Sie das',
        package: 'Installations-Paket herunter',
        unzip: 'und entpacken Sie es',
        upTailscale: 'Tailscale nach /usr/bin/ auf NanoKVM hochladen',
        upTailscaled: 'Tailscaled nach /usr/sbin/ auf NanoKVM hochladen',
        refresh: 'Aktuelle Seite neu laden',
        notRunning: 'Tailscale läuft nicht. Bitte starten Sie es, um fortzufahren.',
        run: 'Starten',
        notLogin:
          'Das Gerät konnte noch nicht gefunden werden. Bitte melden Sie sich an und verknüpfen Sie dieses Gerät mit Ihrem Konto.',
        urlPeriod: 'Diese URL ist für 10 Minuten gültig',
        login: 'Anmelden',
        loginSuccess: 'Anmeldung erfolgreich',
        enable: 'Tailscale einschalten',
        exitNode: 'Ausgangsknoten',
        exitNodeDesc: 'Dieses Gerät als Internetausgang für Ihr Tailnet ankündigen.',
        subnetRoute: 'Subnetzroute',
        subnetRouteDesc:
          'Das direkt verbundene LAN-Subnetz {{subnet}} in Ihrem Tailnet ankündigen.',
        subnetUnavailable: 'Es ist kein direkt verbundenes LAN-Subnetz verfügbar.',
        routingUnavailable: 'Tailscale-Routing ist auf diesem Gerät nicht verfügbar.',
        pendingApproval:
          'Genehmigen Sie die angekündigte Route nach dem Aktivieren in der Tailscale-Verwaltungskonsole.',
        routingUpdateFailed:
          'Die Tailscale-Routing-Einstellungen konnten nicht aktualisiert werden.',
        deviceName: 'Geräte Name',
        deviceIP: 'Geräte IP',
        account: 'Konto',
        logout: 'Abmelden',
        logoutDesc: 'Möchten Sie sich wirklich abmelden?',
        uninstall: 'Tailscale deinstallieren',
        uninstallDesc: 'Sind Sie sicher, dass Sie Tailscale deinstallieren möchten?',
        okBtn: 'Ja',
        cancelBtn: 'Nein'
      },
      update: {
        title: 'Nach Aktualisierungen suchen',
        queryFailed: 'Version konnte nicht abgefragt werden',
        updateFailed: 'Aktualisierung fehlgeschlagen. Bitte versuchen Sie es erneut.',
        isLatest: 'Sie haben bereits die aktuellste Version.',
        available: 'Eine Aktualisierung ist verfügbar. Möchten sie diese jetzt durchführen?',
        updating: 'Aktualisierung gestartet. Bitte warten...',
        confirm: 'Bestätigen',
        upgrade_tip_nano:
          'Eine neue Version von NanoKVM ist verfügbar. Sind Sie sicher, dass Sie jetzt aktualisieren möchten?',
        upgrade_tip_firmware:
          'Eine neue Firmware-Version ist verfügbar. Sind Sie sicher, dass Sie jetzt aktualisieren möchten?',
        cancel: 'Abbrechen',
        preview: 'Vorab-Versionen',
        previewDesc: 'Erhalten Sie vorab Zugriff auf neue Funktionen und Verbesserungen',
        previewTip:
          'Bitte beachten Sie, dass Vorab-Versionen womöglich noch Fehler oder unvollständige Funktionen enthalten!',
        download: 'Herunterladen',
        install: 'Installieren',
        changelog: 'Änderungsprotokoll',
        restart: 'Neustart'
      },
      account: {
        title: 'Konto',
        webAccount: 'Web Konto Name',
        password: 'Passwort',
        twoFactor: 'Zwei-Faktor-Authentifizierung',
        enableTwoFactor: 'Aktivieren',
        enableTwoFactorDesc:
          'Geben Sie Ihr aktuelles Passwort ein, um die Zwei-Faktor-Authentifizierung einzurichten.',
        disableTwoFactor: 'Deaktivieren',
        recoveryCodes: 'Wiederherstellungscodes',
        enableRecoveryCodes: 'Aktivieren',
        updateRecoveryCodes: 'Aktualisieren',
        recoveryCodesTooltipDisabled:
          'Wiederherstellungscodes sind Ersatz-Zugangsdaten für die Zwei-Faktor-Authentifizierung. Sie können sie nach dem Aktivieren der Zwei-Faktor-Authentifizierung generieren.',
        recoveryCodesTooltipEnabled:
          'Wiederherstellungscodes sind nur gültig, solange die Zwei-Faktor-Authentifizierung aktiviert ist. Das Deaktivieren der Zwei-Faktor-Authentifizierung macht alle Wiederherstellungscodes ungültig; solange sie aktiviert bleibt, können sie nur aktualisiert werden.',
        recoveryCodesRegenerateDesc:
          'Geben Sie Ihr aktuelles Passwort und entweder einen Authentifizierungscode oder einen unbenutzten Wiederherstellungscode ein. Das Aktualisieren der Wiederherstellungscodes macht alle vorhandenen Wiederherstellungscodes ungültig.',
        recoveryCodesVerifyFailed:
          'Passwort und Authentifizierungscode oder Wiederherstellungscode konnten nicht verifiziert werden.',
        recoveryCodesSavedTitle: 'Speichern Sie diese Wiederherstellungscodes jetzt',
        recoveryCodesSavedDesc:
          'Wiederherstellungscodes können nur einmal angezeigt werden. Speichern Sie sie an einem sicheren Ort, bevor Sie diese Seite verlassen.',
        enableRecoveryCodesDesc:
          'Geben Sie Ihr aktuelles Passwort und einen Authentifizierungscode ein, um Wiederherstellungscodes zu generieren. Wiederherstellungscodes können nur einmal angezeigt werden. Speichern Sie sie daher, bevor Sie diese Seite verlassen.',
        copyRecoveryCodes: 'Kopieren',
        downloadRecoveryCodes: 'Herunterladen',
        recoveryCodesCopyFailed: 'Wiederherstellungscodes konnten nicht kopiert werden.',
        done: 'Fertig',
        currentPassword: 'Aktuelles Passwort',
        authCode: 'Authentifizierungscode',
        authCodeOrRecoveryCode: 'Authentifizierungscode oder Wiederherstellungscode',
        authRateLimited:
          'Zu viele fehlgeschlagene Versuche. Versuchen Sie es in {{seconds}} s erneut.',
        verifyPasswordChange: 'Passwortänderung verifizieren',
        verifyPasswordChangeDesc:
          'Geben Sie vor dem Ändern des Passworts Ihr aktuelles Passwort und entweder einen Authentifizierungscode oder einen unbenutzten Wiederherstellungscode ein.',
        verifyPasswordChangeFailed:
          'Passwort und Authentifizierungscode oder Wiederherstellungscode konnten nicht verifiziert werden.',
        verifyAndEnable: 'Verifizieren und aktivieren',
        scanTwoFactorQRCode:
          'Scannen Sie diesen QR-Code mit Ihrer Authenticator-App und geben Sie anschließend den Code ein.',
        showSetupKey: 'Einrichtungsschlüssel anzeigen',
        hideSetupKey: 'Einrichtungsschlüssel ausblenden',
        setupKey: 'Einrichtungsschlüssel',
        twoFactorSetupFailed:
          'Einrichtung der Zwei-Faktor-Authentifizierung konnte nicht gestartet werden.',
        twoFactorVerifyFailed: 'Authentifizierungscode konnte nicht verifiziert werden.',
        twoFactorDisableFailed: 'Zwei-Faktor-Authentifizierung konnte nicht deaktiviert werden.',
        disableTwoFactorDesc:
          'Geben Sie Ihr aktuelles Passwort und entweder einen Authentifizierungscode oder einen unbenutzten Wiederherstellungscode ein. Das Deaktivieren der Zwei-Faktor-Authentifizierung macht alle Wiederherstellungscodes ungültig.',
        updateBtn: 'Ändern',
        logoutBtn: 'Abmelden',
        logoutDesc: 'Möchten Sie sich wirklich abmelden?',
        okBtn: 'Ja',
        cancelBtn: 'Nein',
        twoFactorProtection: {
          title: 'Erweiterter 2FA-Schutz',
          description:
            'Wählen Sie sensible Aktionen aus, die bei erzwungenem Schutz einen zweiten Faktor erfordern. Der Schalter „Direkte Web-Anmeldung“ bewirkt das Gegenteil: Er erlaubt die Anmeldung nur mit einem Authentifizierungs- oder unbenutzten Wiederherstellungscode.',
          login: 'Direkte Web-Anmeldung',
          webTerminal: 'Web-Terminal',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP-API-Schlüssel',
          targetPower: 'Stromversorgung des Zielrechners',
          nanoKVMRestart: 'NanoKVM-Neustart',
          deleteImage: 'Image löschen',
          scripts: 'Skriptoperationen',
          screenTimelapse: 'Bildschirm-Zeitraffer',
          loginScope:
            'Ermöglicht die Web-Anmeldung nur mit einem Authentifizierungs- oder unbenutzten Wiederherstellungscode statt mit Benutzername und Passwort. Dies verringert die Anmeldesicherheit.',
          webTerminalScope: 'Öffnen des Web-Terminals.',
          sshScope: 'Ändern des SSH-Dienststatus.',
          tailscaleScope: 'Installieren, Verbinden oder Ändern von Tailscale.',
          frp: 'FRP',
          frpScope: 'Installieren, Verbinden oder Ändern von FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Installieren, Verbinden oder Ändern von Cloudflare Tunnel.',
          zerotierScope: 'Installieren, Verbinden oder Ändern von ZeroTier.',
          netbirdScope: 'Installieren, Verbinden oder Ändern von NetBird.',
          mcpScope:
            'Anzeigen, Kopieren oder Neugenerieren des MCP-API-Schlüssels sowie Aktivieren oder Deaktivieren von MCP.',
          targetPowerScope:
            'Den Zielrechner mit dem physischen Ein-/Ausschalter herunterfahren oder neu starten.',
          nanoKVMRestartScope: 'Neustarten von NanoKVM selbst.',
          deleteImageScope: 'Ein gespeichertes ISO- oder IMG-Image dauerhaft löschen.',
          scriptsScope: 'Ausführen, Hochladen oder Löschen von Skripten.',
          screenTimelapseScope:
            'Aktivieren der Bildschirm-Zeitrafferaufnahme, Durchsuchen von Aufnahmen und Löschen von Aufnahmen.',
          authorizeTitle: 'Schutzänderungen autorisieren',
          authorizeDesc:
            'Geben Sie einen Authentifizierungs- oder unbenutzten Wiederherstellungscode ein. Die Autorisierung gilt fünf Minuten.',
          authorizeFailed:
            'Authentifizierungscode oder Wiederherstellungscode konnte nicht überprüft werden.',
          loadFailed: 'Schutzeinstellungen konnten nicht geladen werden.',
          updateFailed: 'Schutzeinstellungen konnten nicht gespeichert werden.',
          retry: 'Erneut versuchen',
          disableScriptsWarningTitle: 'Skriptschutz deaktivieren?',
          disableScriptsWarningDesc:
            'Ohne Schutz kann ein bösartiges Skript erweiterten Zugriff auf dieses Gerät erlangen und den 2FA-Schutz umgehen oder entfernen.',
          cancelDisabling: 'Abbrechen',
          continueDisabling: 'Deaktivierung fortsetzen',
          continueDisablingCountdown: 'Deaktivierung fortsetzen ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Leitfaden zur Zwei-Faktor-Authentifizierung',
          scope:
            'Die Zwei-Faktor-Authentifizierung fügt Ihrem Konto eine zusätzliche Sicherheitsebene hinzu.',
          whatTitle: 'Was ist 2FA?',
          whatDesc:
            'Die Zwei-Faktor-Authentifizierung (2FA) schützt Ihr Konto mit einem zeitbasierten Code aus einer Authenticator-App. Standardmäßig ist der Code zusätzlich zu Ihrem Passwort erforderlich, sodass Ihr Konto selbst bei kompromittiertem Passwort geschützt bleibt. Ist die direkte Web-Anmeldung aktiviert, meldet Sie der Code allein an.',
          appsTitle: 'Empfohlene Apps',
          appsDesc:
            'Sie können Google Authenticator oder Microsoft Authenticator verwenden. Beide sind kostenlos und funktionieren unter iOS und Android.',
          setupTitle: 'Einrichtung',
          setupDesc:
            'Öffnen Sie Ihre Authenticator-App, scannen Sie den während der Einrichtung angezeigten QR-Code oder geben Sie den Einrichtungsschlüssel manuell ein. Geben Sie danach den 6-stelligen Code aus der App ein, um 2FA zu aktivieren.',
          recoveryTitle: 'Wiederherstellungscodes',
          recoveryDesc:
            'Wiederherstellungscodes sind einmalige Ersatzcodes. Wenn Sie keinen Zugriff mehr auf Ihre Authenticator-App haben, können Sie sich mit einem Wiederherstellungscode anmelden. Speichern Sie sie direkt nach dem Erstellen an einem sicheren Ort.',
          logoutNotice:
            'Das Aktivieren von 2FA meldet alle aktiven Sitzungen ab, einschließlich dieser Sitzung. Sie müssen sich auf jedem Gerät erneut anmelden.'
        }
      }
    },
    error: {
      title: 'Wir sind auf ein Problem gestossen',
      refresh: 'Neuladen'
    },
    pictureInPicture: {
      title: 'Bild-im-Bild',
      enter: 'Bild-im-Bild aktivieren',
      exit: 'Bild-im-Bild beenden',
      error:
        'Bild-im-Bild konnte nicht gestartet werden. Warten Sie, bis das Video geladen ist, und versuchen Sie es erneut.'
    },
    fullscreen: {
      toggle: 'Vollbild ein/aus'
    },
    menu: {
      collapse: 'Menü verbergen',
      expand: 'Menü einblenden'
    }
  }
};

export default de;
