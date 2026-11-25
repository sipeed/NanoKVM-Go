const hu = {
  translation: {
    storage: {
      cards: { mount: 'Lemez csatlakoztatása', share: 'Fájlmegosztás' },
      title: 'Képek és fájlmegosztás',
      empty: 'Még nincsenek képek',
      notAttached: 'Nincs csatlakoztatva semmi',
      attached: 'Csatlakoztatva: {{name}}',
      eject: 'Kiadás',
      mount: 'Csatlakoztatás',
      mountMode: 'Felszerelési mód',
      mountModeDisk: 'Háttértár',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Fájlkezelő megnyitása',
      refresh: 'Frissítés',
      delete: 'Törlés',
      deleteTitle: 'Adathordozó törlése',
      deleteConfirm: 'Törli a következőt: {{name}}? A benne lévő fájlok elvesznek.',
      downloadImage: 'A kép letöltése',
      typePartition: 'Megosztott fájlok',
      volumeLabel: 'Cserélhető lemez',
      groups: {
        share: 'Fájlmegosztás',
        images: 'Képek'
      },
      failed: 'A művelet nem sikerült',
      hidOnly: 'Csak HID mód: a virtuális lemez nem érhető el',
      disabled: 'A virtuális lemez ki van kapcsolva (Beállítások - Eszköz)',
      force: {
        title: 'Visszaveszi az adathordozót erővel?',
        desc: 'A célgép még használja ezt az adathordozót. A visszavétel újra felsorolja az USB-eszközt, és elveszhet minden, amit még nem írt a lemezre.',
        menuHint:
          'A célgép használja ezt az adathordozót. Nyissa meg a fájlkezelőt az erővel történő visszavételhez.'
      },
      tips: {
        title: 'Hogyan tölts fel képeket',
        usb1: 'Csatlakoztassa a NanoKVM-t a számítógépéhez USB-n keresztül.',
        usb2: 'Ezen az oldalon csatolja a megosztott fájlokat háttértár módban.',
        usb3: 'Nyissa meg a virtuális lemezt a számítógépén, és másolja a kép fájlt a virtuális lemez gyökérkönyvtárába.',
        usb4: 'A másolás végeztével adja ki a lemezt: a lemezkép ezután megjelenik a listában.',
        scp1: 'Győződjön meg róla, hogy a NanoKVM és a számítógépe ugyanazon a helyi hálózaton van.',
        scp2: 'Nyisson meg egy terminált a számítógépén, és használja az SCP parancsot a kép fájl feltöltésére a /data könyvtárba a NanoKVM-en.',
        scp3: 'Példa: scp your-image-path root@your-nanokvm-ip:/data'
      },
      job: {
        running: '{{name}} feldolgozása'
      },
      enableDisk: {
        title: 'Bekapcsolja a virtuális lemezt?',
        desc: 'Az adathordozó csatlakoztatásához a virtuális lemez kell, amely jelenleg ki van kapcsolva. Bekapcsolja? A célgép billentyűzete és egere egy pillanatra leválik.',
        menuHint: 'A virtuális lemez ki van kapcsolva. Kapcsolja be a Beállítások - Eszköz menüben.'
      },
      add: {
        title: 'Adathordozó hozzáadása',
        upload: 'ISO vagy IMG feltöltése',
        fromUrl: 'Letöltés URL-ről',
        url: 'A kép URL-címe',
        urlRequired: 'Adjon meg egy URL-t',
        onlyImages: 'Csak .iso és .img fájlok csatlakoztathatók',
        dropHint: 'Húzzon ide egy ISO vagy IMG fájlt, vagy kattintson a kiválasztáshoz',
        cdromOnlyIso: 'CD-ROM csak ISO 9660 / UDF képet tartalmazhat',
        uploadFailed: 'A feltöltés nem sikerült',
        downloadFailed: 'A letöltés nem sikerült',
        uploading: '{{name}} feltöltése',
        downloading: '{{name}} letöltése',
        done: '{{name}} elkészült',
        checksumFailed: 'A lemezkép nem egyezik a SHA-256 ellenőrzőösszeggel.',
        blocked:
          'A célgép használja a lemezt. Biztonságosan válassza le a virtuális USB-meghajtót a célgépen, majd kapcsolja ki a Virtuális lemez kapcsolót a Beállítások > Eszköz menüben.'
      },
      files: {
        reasonNotInitialized:
          'A virtuális USB-meghajtó nincs inicializálva. A fájlmegosztás előtt inicializálja a Beállítások > Eszköz menüben.',
        selectMedium: 'Válasszon adathordozót a rajta lévő fájlok megtekintéséhez.',
        reasonAttached: 'Ez az adathordozó a célgéphez van csatlakoztatva.',
        reasonAttachedHint:
          'Biztonságosan válassza le a virtuális USB-meghajtót a célgépen, majd kapcsolja ki a Virtuális lemez kapcsolót a Beállítások > Eszköz menüben.',
        reasonNotBrowsable:
          'Ez az adathordozó olyan fájlrendszert tartalmaz, amelyet a fájlkezelő nem tud megnyitni.',
        reasonNoLoop: 'Ez az eszköz nem tud lemezképeket csatlakoztatni (nincs loop támogatás).',
        reasonISOImage:
          'Az ISO 9660 képek itt nem böngészhetők. A célgépre továbbra is csatlakoztathatók.',
        empty: 'Ez a mappa üres',
        noMatch: 'Nincs a keresésnek megfelelő fájl.',
        free: '{{size}} szabad',
        search: 'Keresés ebben a mappában',
        sort: {
          modified: 'Módosítva',
          name: 'Név',
          size: 'Méret',
          type: 'Típus',
          asc: 'Növekvő',
          desc: 'Csökkenő'
        },
        view: {
          list: 'Listanézet',
          grid: 'Ikonnézet'
        },
        folder: 'Mappa',
        itemCount: 'Elemek száma: {{count}}',
        itemCount_one: '{{count}} elem',
        itemCount_other: '{{count}} elem',
        dropHint: 'Húzza ide a fájlokat a mappába való feltöltéshez',
        up: 'Ugrás a szülőmappába',
        uploadButton: 'Feltöltés',
        newFolder: 'Új mappa',
        newFolderPlaceholder: 'Mappa neve',
        create: 'Létrehozás',
        upload: 'Fájl feltöltése az adathordozóra',
        deleteFolderTitle: 'Mappa törlése',
        deleteFolderConfirm:
          'Törli a(z) {{name}} mappát és a teljes tartalmát? Ez nem vonható vissza.',
        uploading: 'Feltöltés…',
        uploadFailed: 'A feltöltés nem sikerült',
        download: 'Letöltés',
        delete: 'Törlés',
        deleteTitle: 'Fájl törlése',
        deleteConfirm: 'Törli a következőt: {{name}} az adathordozóról?'
      },
      yes: 'Igen',
      no: 'Nem'
    },
    head: {
      storage: 'Képek és fájlmegosztás',
      desktop: 'Távoli Asztal',
      login: 'Bejelentkezés',
      changePassword: 'Jelszó megváltoztatása',
      terminal: 'Terminál',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Nem sikerült a vágólapra másolni. Engedélyezze a böngészőnek a vágólap elérését, majd próbálja újra.'
    },
    auth: {
      login: 'Bejelentkezés',
      language: 'Nyelv',
      placeholderUsername: 'Adja meg a felhasználónevet',
      placeholderPassword: 'Adja meg a jelszót',
      placeholderPassword2: 'Adja meg újra a jelszót',
      noEmptyUsername: 'A felhasználónév nem lehet üres',
      noEmptyPassword: 'A jelszó nem lehet üres',
      noAccount:
        'Nem sikerült megszerezni a felhasználói információkat, frissítse az oldalt vagy állítsa vissza a jelszót',
      invalidUser: 'Érvénytelen felhasználónév vagy jelszó',
      error: 'Váratlan hiba',
      twoFactorTitle: 'Ellenőrzés szükséges',
      twoFactorDesc:
        'A bejelentkezéshez adja meg a hitelesítési kódot vagy egy fel nem használt helyreállítási kódot.',
      sensitiveActionDesc:
        'A folytatáshoz adja meg a hitelesítési kódot vagy egy fel nem használt helyreállítási kódot.',
      placeholderAuthCode: 'Hitelesítési kód',
      noEmptyAuthCode: 'Hitelesítési kód szükséges',
      verify: 'Ellenőrzés',
      backToLogin: 'Vissza a bejelentkezéshez',
      verificationFailed: 'Az ellenőrzés sikertelen',
      twoFactorEnabledLogout: {
        message:
          'A kétfaktoros hitelesítés engedélyezve lett. Biztonsági okokból minden munkamenet kijelentkezett. Jelentkezzen be újra.'
      },
      rateLimited: 'Túl sok sikertelen próbálkozás. Próbálja újra {{seconds}} mp múlva.',
      recoveryCodeUsed:
        'Az ehhez a bejelentkezéshez használt helyreállítási kód érvénytelenné vált.',
      changePassword: 'Jelszó megváltoztatása',
      changePasswordDesc:
        'Az eszköz biztonsága érdekében módosítsa a webes bejelentkezési jelszót.',
      passwordVerificationRequired:
        'Térjen vissza a fiókbeállítások oldalára, és ellenőrizze újra ezt a jelszómódosítást.',
      passwordVerificationExpired:
        'A jelszómódosítás ellenőrzése lejárt. Térjen vissza a fiókbeállítások oldalára, és ellenőrizze újra.',
      returnToSettings: 'Vissza a beállításokhoz',
      differentPassword: 'A jelszavak nem egyeznek',
      illegalUsername: 'A felhasználónév illegális karaktereket tartalmaz',
      illegalPassword: 'A jelszó illegális karaktereket tartalmaz',
      forgetPassword: 'Jelszó-emlékeztető',
      ok: 'OK',
      cancel: 'Mégse',
      savePassword: 'Jelszó mentése',
      loginButtonText: 'Bejelentkezés',
      tips: {
        reset1: 'webes alapértelmezett fiók:',
        reset2: 'alapértelmezett SSH-fiók:',
        reset3: 'A jelszavak visszaállításához lásd: ',
        changeReset: 'A webes bejelentkezési jelszó visszaállításához lásd: ',
        resetLink: 'Visszaállítás'
      }
    },
    wifi: {
      errors: {
        connect:
          'Nem sikerült csatlakozni a Wi-Fi-hez. Ellenőrizze a hálózat nevét, a jelszót és az eszköz Wi-Fi-állapotát, majd próbálja újra.'
      },
      title: 'Wi-Fi',
      description: 'Wi-Fi konfigurálása a NanoKVM számára',
      submitted: {
        title: 'Egy pillanat — a NanoKVM csatlakozik a Wi-Fihez',
        tipsTitle: 'Mi a következő lépés?',
        tipSwitch: 'Most már visszacsatlakoztathatja a telefont a szokásos Wi-Fi hálózatára.',
        tipCheck:
          'Néhány pillanat múlva nézze meg a NanoKVM kijelzőjén az új IP-címet, és nyissa meg a böngészőben.',
        tipRetry:
          'Ha egy idő után sem történik semmi, ellenőrizze a hálózat nevét és jelszavát, majd próbálja újra.'
      },
      invalidMode:
        'Az aktuális mód nem támogatja a hálózat beállítását. Kérjük, lépjen az eszközére, és engedélyezze a Wi-Fi konfigurációs módot.',
      confirmBtn: 'OK',
      finishBtn: 'Befejezve',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Jelszó',
      steps: {
        settings: 'Beállítások',
        wifi: 'Wi-Fi',
        enable: 'Engedélyezés'
      },
      scan: {
        title: 'Wi-Fi-hálózat kiválasztása',
        description: 'A folytatáshoz válasszon egy közeli hálózatot.',
        loading: 'Közeli Wi-Fi-hálózatok keresése…',
        empty: 'Nem található közeli Wi-Fi-hálózat.',
        failed: 'A keresés nem sikerült. Adja meg kézzel az adatokat.',
        retryBtn: 'Újra keresés',
        manualBtn: 'Hálózat kézi megadása',
        backBtn: 'Vissza a közeli hálózatokhoz',
        connectTitle: 'Csatlakozás ehhez: {{ssid}}',
        connecting: 'Csatlakozás…',
        viewSimple: 'Váltás egyszerű nézetre',
        viewDetailed: 'Váltás részletes nézetre'
      },
      ap: {
        authTitle: 'Hitelesítés szükséges',
        authDescription: 'A folytatáshoz adja meg a AP jelszót',
        authFailed: 'Érvénytelen AP jelszó',
        passPlaceholder: 'AP jelszót',
        verifyBtn: 'Ellenőrizze'
      },
      qr: {
        cameraBtn: 'Fotó készítése',
        pickBtn: 'Képernyőkép választása',
        scanHint:
          'Készítsen fotót egy Wi-Fi QR-kódról, vagy válasszon egy képernyőképet róla, például amilyet a telefon a Wi-Fi megosztásakor mutat.',
        decoding: 'QR-kód beolvasása…',
        notFound:
          'Ezen a fotón nem található QR-kód. Igazítsa középre és élesítse a kódot, majd próbálja újra.',
        invalid: 'Ez a QR-kód nem tartalmaz használható Wi-Fi-adatokat.',
        pickerUnavailable:
          'Nem nyílt meg semmi? Ez a bejelentkezési ablak blokkolhatja a kamerát és a fotókat. Ehelyett nyissa meg ezt a címet a telefon böngészőjében:',
        confirmTitle: 'Csatlakoztatja a NanoKVM-et ehhez: {{ssid}}?',
        confirmDescription:
          'Ezek a hálózati adatok egy QR-kódból származnak. Csatlakozás előtt ellenőrizze őket.',
        openNetwork: 'Ennek a hálózatnak nincs jelszava.',
        connectBtn: 'Csatlakozás',
        otherBtn: 'Másik hálózat kiválasztása'
      }
    },
    notification: {
      browser: {
        title: 'Böngészőjavaslat',
        description:
          'A legjobb felhasználói élmény érdekében a Chrome böngésző használatát javasoljuk. Bár törekszünk az összes főbb böngészővel való kompatibilitásra, előfordulhat, hogy egyes speciális funkciók a Chrome-on kívül nem működnek rendeltetésszerűen.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Képernyő',
      video: 'Videó mód',
      quality: 'Minőség',
      qualityAuto: 'Automatikus',
      qualityLossless: 'Veszteségmentes',
      qualityHigh: 'Magas',
      qualityMedium: 'Közepes',
      qualityLow: 'Alacsony',
      scale: 'Skála',
      noSignal: 'Nincs videójel',
      captureDisabled:
        'A videórögzítés ki van kapcsolva. Kapcsolja be a „Beállítások > Eszköz > Videórögzítés” menüpontban.',
      inconsistentVideoMode: 'Lejátszás szünetel. Egy másik videó mód lejátszása folyik.',
      bandwidthWarning:
        'A hálózati sávszélesség elégtelen lehet a jelenlegi videóbeállításokhoz. Próbálja csökkenteni a bitrátát vagy a képkockasebességet.',
      webrtcConnectionFailed:
        'Nem sikerült létrehozni a WebRTC-kapcsolatot. Váltson Direct videómódra.',
      webrtcUnavailable: {
        title: 'A WebRTC nem érhető el',
        description: 'Engedélyezze a WebRTC-t a böngészőben, vagy váltson Direct videómódra.'
      },
      upload: 'Feltöltés',
      advanced: 'Haladó',
      loading: 'Betöltés'
    },
    volume: {
      title: 'Hangerő'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon nincs felszerelve',
      tip1: 'A virtuális mikrofon nincs felszerelve a távoli gazdagépre. Szeretné most felszerelni és engedélyezni?',
      tip2: 'Ezt később a "Beállítások > Eszköz > Virtuális hang" menüpontban is kezelheti.',
      confirm: 'Engedélyezés',
      cancel: 'Mégse',
      noDevice:
        'Az eszközön nem észlelhető mikrofon. Csatlakoztasson egy mikrofont, majd próbálja újra.',
      permissionDenied:
        'A mikrofonhoz való hozzáférés megtagadva. Engedélyezze a hozzáférést a böngésző beállításaiban.',
      notReadable: 'A mikrofont egy másik alkalmazás használja, vagy nem olvasható.',
      overconstrained: 'A mikrofon nem támogatja a szükséges hangbeállításokat.',
      securityBlocked:
        'A mikrofonhoz való hozzáférés ezen az oldalon le van tiltva. Ellenőrizze a böngésző biztonsági beállításait.',
      aborted: 'A mikrofon indítása megszakadt. Próbálja újra.',
      unsupported: 'Ez a böngésző vagy kapcsolódási mód nem támogatja a mikrofon rögzítését.',
      enableFailed: 'A mikrofont nem sikerült engedélyezni. Próbálja újra.',
      disconnected: 'A mikrofont leválasztották, ezért a mikrofon átvitele kikapcsolt.'
    },
    power: {
      title: 'Bekapcsolás',
      showConfirm: 'Megerősítés',
      showConfirmTip: 'A bekapcsolási műveletekhez külön megerősítés szükséges',
      powerShort: 'Bekapcsolás (rövid kattintás)',
      powerLong: 'Bekapcsolás (hosszú kattintás)',
      powerConfirm: 'Végrehajtja a bekapcsolási műveletet?',
      okBtn: 'Igen',
      cancelBtn: 'Nem'
    },
    keyboard: {
      errors: {
        paste:
          'Nem sikerült elküldeni a szöveget a távoli eszközre. Ellenőrizze a KVM-kapcsolatot, majd próbálja újra.'
      },
      title: 'Billentyűzet',
      paste: 'Beillesztés',
      tips: 'Csak a szabványos billentyűzet betűi és szimbólumai támogatottak',
      placeholder: 'Írja be',
      submit: 'Elküldés',
      virtual: 'Billentyűzet',
      swapCommandCtrl: 'Cmd és Ctrl felcserélése',
      shortcut: {
        title: 'Parancsikonok',
        custom: 'Egyedi',
        capture: 'Kattintson ide a parancsikon rögzítéséhez',
        clear: 'Törlés',
        save: 'Mentés',
        captureTips:
          'A rendszerszintű billentyűk (például a Windows billentyű) rögzítéséhez teljes képernyős engedély szükséges.',
        enterFullScreen: 'Teljes képernyős mód váltása.'
      },
      leaderKey: {
        title: 'Leader billentyű',
        desc: 'Kerülje ki a böngésző korlátozásait, és küldje el a rendszer parancsikonjait közvetlenül a távoli gazdagépnek.',
        howToUse: 'Használat',
        simultaneous: {
          title: 'Egyidejű üzemmód',
          desc1: 'Tartsa lenyomva a Leader billentyűt, majd nyomja meg a gyorsbillentyűt.',
          desc2: 'Intuitív, de ütközhet a rendszer parancsikonjaival.'
        },
        sequential: {
          title: 'Szekvenciális mód',
          desc1:
            'Nyomja meg a Leader billentyűt → nyomja meg sorban a gyorsbillentyűt → nyomja meg újra a Leader billentyűt.',
          desc2: 'Több lépést igényel, de teljesen elkerülhető a rendszerütközések.'
        },
        enable: 'Leader billentyű engedélyezése',
        tip: 'Leader billentyűként beállítva ez a billentyű kizárólag gyorsbillentyű-indítóként működik, és elveszíti alapértelmezett viselkedését.',
        placeholder: 'Nyomja meg a Leader billentyűt',
        shiftRight: 'Jobb Shift',
        ctrlRight: 'Jobb Ctrl',
        metaRight: 'Jobb Win',
        submit: 'Elküldés',
        recorder: {
          rec: 'REC',
          activate: 'Billentyűk aktiválása',
          input: 'Kérjük, nyomja meg a parancsikont...'
        }
      }
    },
    mouse: {
      title: 'Egér',
      mode: 'Egér mód',
      absolute: 'Követő egér',
      absoluteAndroid: 'Követő egér (jobb Android-kompatibilitás)',
      touchScreen: 'Többérintéses képernyő',
      relative: 'Exkluzív egér',
      direction: 'Görgő iránya',
      scrollUp: 'Görgessen felfelé',
      scrollDown: 'Görgessen le',
      speed: 'Görgő sebessége',
      fast: 'Gyors',
      slow: 'Lassú',
      requestPointer:
        'Relatív mód használata. Kattintson az asztalra, hogy megjelenjen az egérmutató.',
      repairIphoneDrag: 'iPhone-húzás javítása',
      resetHid: 'HID visszaállítása',
      touchpadGuide: {
        title: 'Érintőpad útmutató',
        scope: 'A normál mutatóbevitelre vonatkozik, ha az egérmód Relatív.',
        swipeTitle: 'Húzza az ujját a mozgatáshoz',
        swipeDesc: 'Húzza az ujját az aktív képernyőterületen a távoli mutató mozgatásához.',
        tapTitle: 'Koppintson a kattintáshoz',
        tapDesc: 'Egy rövid koppintás egy bal kattintást küld.',
        holdTitle: 'Tartsa nyomva a bal gomb lenyomásához',
        holdDesc:
          'Érintse meg és tartsa mozdulatlanul kb. 1 másodpercig, hogy a bal egérgomb lenyomva maradjon.',
        dragTitle: 'Tartás után mozgassa a húzáshoz',
        dragDesc:
          'Amikor a tartás aktív, mozgassa az ujját a bal gomb lenyomva tartásával történő húzáshoz.'
      },
      hidOnly: {
        errors: {
          load: 'Nem sikerült betölteni a HID-Only mód állapotát. Próbálja újra rövidesen.',
          update:
            'Nem sikerült megerősíteni, hogy a HID-Only mód frissült-e. Nyissa meg újra ezt a beállítást, ellenőrizze az aktuális állapotot, majd próbálja újra.'
        },
        title: 'Csak HID mód',
        desc: 'Ha az egér és a billentyűzet nem válaszol, és az HID alaphelyzetbe állítása nem segít, akkor az NanoKVM és az eszköz közötti kompatibilitási probléma lehet. Próbálja engedélyezni az HID-Csak módot a jobb kompatibilitás érdekében.',
        tip1: 'Az HID-Csak mód engedélyezése leválasztja a virtuális U-lemezt és a virtuális hálózatot',
        tip2: 'HID-Csak módban a képrögzítés le van tiltva',
        enable: 'Engedélyezze a HID-Csak módot',
        disable: 'A HID-Csak mód letiltása'
      }
    },
    preview: {
      title: 'Előnézet'
    },
    image: {
      storageReadOnly: 'A fájltároló nem írható. Kapcsolja ki a virtuális lemezt.',
      storageBusy:
        'A tároló foglalt. A virtuális lemez átkapcsolása előtt fejezze be vagy szakítsa meg a fájlátvitelt.',
      title: 'Képek',
      loading: 'Betöltés...',
      empty: 'Nem található semmi',
      upload: 'Kép feltöltése',
      dropFile: 'Húzza ide a lemezképet, vagy kattintson a fájl kiválasztásához',
      uploadAction: 'Feltöltés',
      cancelAction: 'Mégse',
      mount: 'Kép csatolása',
      mounted: 'Felszerelve',
      uploadError: 'A feltöltés sikertelen, próbálja újra',
      uploadInsufficientStorage: 'Nincs elég tárhely',
      uploadTooLarge: 'A lemezkép meghaladja a méretkorlátot',
      cancelUploadError: 'A feltöltést nem sikerült megszakítani',
      mountMode: 'Felszerelési mód',
      massStorage: 'Tömegtároló',
      cdRom: 'CD ROM',
      readOnly: 'Csak olvasható',
      mountFailed: 'Csatlakoztatás sikertelen',
      mountDesc:
        'Egyes rendszerekben szükséges lehet a virtuális lemez eltávolítása a távoli gépen, mielőtt a képet csatlakoztatja.',
      unmountFailed: 'Leválasztás sikertelen',
      unmountDesc:
        'Egyes rendszerekben kézzel ki kell dobni a képet a távoli gazdagépen, mielőtt leválasztanád.',
      refresh: 'Frissítse a képlistát',
      download: 'Kép letöltése',
      attention: 'Figyelem',
      deleteConfirm: 'Biztosan törli ezt a képet?',
      deleteFailed: 'Nem sikerült törölni a képet',
      okBtn: 'Igen',
      cancelBtn: 'Nem'
    },
    script: {
      title: 'Szkriptek',
      upload: 'Feltöltés',
      run: 'Futtatás',
      runBackground: 'Háttérben futtatás',
      runFailed: 'Futtatás sikertelen',
      uploadFailed: 'Nem sikerült feltölteni a szkriptet',
      deleteFailed: 'Nem sikerült törölni a szkriptet',
      attention: 'Figyelem',
      delDesc: 'Biztosan törli ezt a fájlt?',
      confirm: 'Igen',
      cancel: 'Nem',
      delete: 'Törlés',
      close: 'Bezárás'
    },
    terminal: {
      title: 'Terminál',
      nanokvm: 'NanoKVM Terminál',
      confirm: 'OK',
      submit: 'Elküldés',
      username: 'Felhasználónév',
      password: 'Jelszó',
      authCode: 'Hitelesítési kód',
      authorizeTitle: 'Webterminál engedélyezése',
      codeAccountHint:
        'Adjon meg egy hitelesítési kódot vagy fel nem használt helyreállítási kódot.',
      authFailed: 'Hitelesítés sikertelen'
    },
    wol: {
      errors: {
        send: 'Nem sikerült elküldeni az ébresztési parancsot. Ellenőrizze a MAC-címet és a hálózati kapcsolatot, majd próbálja újra.',
        loadFailed:
          'Nem sikerült betölteni a Wake-on-LAN-eszközök listáját. Ellenőrizze, hogy a NanoKVM online-e, majd próbálja újra.',
        renameFailed:
          'Nem sikerült átnevezni ezt az eszközt. Ellenőrizze a MAC-címet, majd próbálja újra.',
        deleteFailed: 'Nem sikerült törölni ezt az eszközt. Próbálja újra kicsit később.'
      },
      title: 'Wake-on-LAN',
      sending: 'Parancs küldése...',
      sent: 'Parancs elküldve',
      input: 'Adja meg a MAC címet',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (opcionális)',
      sha256Placeholder: 'Adjon meg egy 64 karakteres SHA-256 ellenőrzőösszeget',
      invalidSHA256: 'A SHA-256 értékének 64 karakteres hexadecimális karakterláncnak kell lennie',
      title: 'Képletöltő',
      input: 'Adjon meg egy távoli képet URL',
      ok: 'OK',
      disabled: '/data partíció RO, ezért nem tudjuk letölteni a képet',
      downloadingFile: 'Letöltés: {{file}}',
      downloadingProgress: 'Letöltés ({{percentage}}): {{file}}',
      failed: 'Sikertelen',
      insufficientStorage: 'Nincs elég tárhely',
      remaining: 'Hátralévő idő: {{time}}',
      cancel: 'Mégse',
      cancelFailed: 'A letöltést nem sikerült megszakítani',
      pause: 'Szüneteltetés',
      resume: 'Folytatás',
      pauseFailed: 'A letöltést nem sikerült szüneteltetni',
      resumeFailed: 'A letöltést nem sikerült folytatni'
    },
    settings: {
      back: 'Vissza',
      close: 'Bezárás',
      ai: {
        errors: {
          loadServices:
            'Nem sikerült betölteni az AI-szolgáltatások állapotát. Ellenőrizze, hogy a NanoKVM online van-e, majd próbálja újra.',
          loadMcpConfig:
            'Nem sikerült betölteni az MCP-szolgáltatás konfigurációját. Próbálja újra rövidesen.',
          updateMcpService:
            'Nem sikerült frissíteni az MCP-szolgáltatás állapotát. Újrapróbálás előtt ellenőrizze az aktuális állapotot.',
          accessMcpApiKey:
            'Nem sikerült lekérni az MCP API-kulcsot. Próbálja újra, és ha kéri a rendszer, fejezze be az ellenőrzést.',
          regenerateMcpApiKey:
            'Nem sikerült újragenerálni az MCP API-kulcsot. Próbálja újra rövidesen.',
          regeneratedMcpApiKeyUnavailable:
            'Az API-kulcs újragenerálása megtörtént, de nem jeleníthető meg. Próbálja újra megjeleníteni. Az előző kulcs már nem érvényes.',
          loadMemoryFabricState:
            'Nem sikerült betölteni a Memory Fabric szolgáltatás állapotát. Próbálja újra rövidesen.',
          updateMemoryFabricService:
            'Nem sikerült frissíteni a Memory Fabric szolgáltatás állapotát. Újrapróbálás előtt ellenőrizze az aktuális állapotot.',
          loadScreenTimelapseState:
            'Nem sikerült betölteni a képernyő-időzített felvétel szolgáltatás állapotát. Próbálja újra rövidesen.',
          updateScreenTimelapseService:
            'Nem sikerült frissíteni a képernyő-időzített felvétel szolgáltatás állapotát. Újrapróbálás előtt ellenőrizze az aktuális állapotot.',
          openScreenTimelapse:
            'Nem sikerült megnyitni a képernyő-időzített felvétel lejátszását. Próbálja újra rövidesen.',
          loadScreenTimelapseRecordings:
            'Nem sikerült betölteni a képernyő-időzített felvételeit. Próbálja újra rövidesen.',
          playScreenTimelapse:
            'Ez az időzített felvétel jelenleg nem játszható le. Válassza ki újra a dátumot, vagy próbálja később.',
          screenTimelapseFullscreen:
            'Nem sikerült módosítani a teljes képernyős módot. Ellenőrizze a böngésző engedélyeit, majd próbálja újra.'
        },
        title: 'AI',
        mcpService: 'MCP szolgáltatás',
        mcpServiceDesc:
          'Hitelesített közös átjáró a vezérléshez, médiához és az engedélyezett Memory Fabric eszközökhöz',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc:
          'Az ügynökök kontextusmemóriájának és OCR-jének helyi feldolgozása',
        memoryFabricRemoteRequiresMcp:
          'Kapcsolja be fent az MCP szolgáltatást a memória- és OCR-eszközök távoli eléréséhez.',
        memoryFabricRemoteAvailable:
          'A memória- és OCR-eszközök a fenti /api/mcp végponton, ugyanazzal az API-kulccsal érhetők el.',
        memoryFabricConfig: {
          title: 'Konfiguráció',
          description: 'Csak a Memory Fabric alapértelmezett modelljének beállításait kezelheti.',
          loadFailed: 'Nem sikerült betölteni a Memory Fabric konfigurációját.',
          saveFailed: 'Nem sikerült menteni a Memory Fabric konfigurációját.',
          saved: 'A konfiguráció mentve.',
          savedRestarted: 'A konfiguráció mentve, a Memory Fabric pedig újraindítva.',
          noChanges: 'A konfiguráció nem változott.',
          reload: 'Újratöltés',
          save: 'Mentés',
          unsaved: 'Nem mentett módosítások',
          conflictTitle: 'A konfiguráció megváltozott',
          conflictDescription:
            'A konfigurációt máshol módosították. Újabb mentés előtt töltse be a legfrissebb verziót.',
          conflictReload: 'Legfrissebb verzió betöltése',
          discardTitle: 'Elveti a nem mentett módosításokat?',
          discardDescription:
            'Az újratöltés a jelenlegi űrlapot a legutóbb mentett konfigurációra cseréli.',
          discardConfirm: 'Elvetés és újratöltés',
          sections: {
            models: 'Modellek'
          },
          openAICompatibleHint:
            'Támogatja az OpenAI-kompatibilis modellszolgáltatókat. Adja meg a szolgáltató által biztosított modellnevet, alap URL-t és API-kulcsot.',
          fields: {
            model: 'Modell',
            baseUrl: 'Alap URL',
            maxTokens: 'Tokenek maximális száma',
            timeoutSeconds: 'Időtúllépés (másodperc)'
          },
          apiKey: {
            title: 'API-kulcs',
            configured: 'Beállítva',
            notConfigured: 'Nincs beállítva',
            keep: 'Megtartás',
            replace: 'Csere',
            clear: 'Eltávolítás',
            replacement: 'Új API-kulcs',
            replacementPlaceholder: 'Adja meg az új API-kulcsot',
            clearHint:
              'A mentett API-kulcs törlődik; a környezeti változó továbbra is biztosíthat kulcsot.',
            required: 'Adjon meg egy új API-kulcsot.'
          },
          validation: {
            required: 'A mező kitöltése kötelező.',
            invalidUrl: 'Adjon meg érvényes HTTP- vagy HTTPS-URL-t.'
          }
        },
        screenTimelapseService: 'Képernyő-időgyorsítás',
        screenTimelapseServiceDesc:
          'Memory Fabric képernyőelőzmények visszajátszása és rögzített képernyőtartalom keresése',
        screenTimelapseSearchRequiresMemoryFabric:
          'A rögzített tartalom kereséséhez kapcsolja be a Memory Fabric szolgáltatást is.',
        screenTimelapseOpen: 'Megnyitás',
        screenTimelapseSearchPlaceholder: 'Keresés a videó tartalmában...',
        screenTimelapseNoVideos: 'Nem található időgyorsított videó',
        screenTimelapseNoResults: 'Nem található egyező képkocka',
        screenTimelapseMultipleResults: 'Keresési eredmények',
        screenTimelapseUnavailable: 'Nem elérhető',
        screenTimelapsePlayerUnsupported: 'Ez a böngésző nem tud időgyorsított videót lejátszani',
        screenTimelapseSearchFailed: 'Keresés sikertelen',
        screenTimelapseLoading: 'Időgyorsítás betöltése...',
        screenTimelapseSearchAction: 'Keresés az időgyorsításban',
        screenTimelapsePlay: 'Időgyorsítás lejátszása',
        screenTimelapsePause: 'Időgyorsítás szüneteltetése',
        screenTimelapseEnterFullscreen: 'Belépés teljes képernyőre',
        screenTimelapseExitFullscreen: 'Kilépés teljes képernyőből',
        screenTimelapseFullscreenUnsupported: 'Ez a böngésző nem tud teljes képernyőre váltani',
        screenTimelapsePreviousDay: 'Előző nap',
        screenTimelapseNextDay: 'Következő nap',
        screenTimelapseDeleteDay: 'A(z) {{date}} felvételének törlése',
        screenTimelapseDelete: 'Törlés',
        screenTimelapseDeleteConfirmTitle: 'Törli a time-lapse felvételt?',
        screenTimelapseDeleteConfirmDesc:
          'A(z) {{date}} felvétele {{size}} helyet foglal. Ez a művelet nem vonható vissza.',
        screenTimelapseDeleteSuccess: 'A(z) {{date}} felvétele törölve',
        screenTimelapseDeleteFailed: 'A(z) {{date}} felvétele nem törölhető',
        endpoint: 'Végpont',
        apiKey: 'API-kulcs',
        regenerate: 'Újragenerálás',
        copyApiKey: 'API-kulcs másolása',
        copyEndpoint: 'Végpont másolása',
        showApiKey: 'API-kulcs megjelenítése',
        regenerateConfirmTitle: 'API-kulcs újragenerálása',
        regenerateConfirmDesc:
          'A meglévő MCP-kliensek nem fognak működni, amíg nem frissítik őket az új kulccsal. Folytatja?',
        okBtn: 'Igen',
        cancelBtn: 'Nem'
      },
      title: 'Beállítások',
      about: {
        title: 'NanoKVM Névjegy',
        information: 'Információ',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Alkalmazás verzió',
        applicationTip: 'NanoKVM webes alkalmazás verziója',
        arch: 'Architektúra',
        device_number: 'Eszközszám',
        image: 'Képfájl verzió',
        imageTip: 'NanoKVM rendszerkép verziója',
        deviceKey: 'Eszköz kulcs',
        community: 'Közösség',
        hostname: 'Gazdanév',
        hostnameUpdated: 'Gazdanév frissítve. Az alkalmazáshoz indítsa újra.',
        ipType: {
          Wired: 'Vezetékes',
          Wireless: 'Vezeték nélküli',
          Other: 'Egyéb'
        }
      },
      appearance: {
        title: 'Megjelenés',
        display: 'Kijelző',
        language: 'Nyelv',
        languageDesc: 'Válassza ki a felület nyelvét',
        languageChangeFailed: 'Nem sikerült nyelvet váltani',
        webTitle: 'Webcím',
        webTitleDesc: 'A weboldal címének testreszabása',
        menuBar: {
          title: 'Menüsor',
          mode: 'Megjelenítési mód',
          modeDesc: 'Menüsor megjelenítése a képernyőn',
          modeOff: 'Ki',
          modeAuto: 'Automatikus elrejtés',
          modeAlways: 'Mindig látható',
          icons: 'Almenü ikonok',
          showIcon: 'Ikon megjelenítése',
          activateOnConnect: 'Aktiválás csatlakozáskor',
          stopSharing: 'Megosztás leállítása',
          iconsDesc: 'Almenüikonok megjelenítése a menüsorban'
        }
      },
      screen: {
        title: 'Képernyő',
        videoMode: {
          title: 'Videó mód',
          description: 'Válassza ki a kódolási és átviteli módot'
        },
        edid: {
          description: 'Válassza ki a megjelenítési paramétereket',
          default: 'Alapértelmezett',
          custom: 'Egyéni',
          document: 'Dokumentum',
          powerConfirmTitle: 'Megerősítés az EDID frissítése előtt',
          powerConfirmDescription:
            'Először csatlakoztassa a PD-tápellátást. Az EDID frissítése rövid időre leválaszthatja a kijelzőt; ha ezután az eszköz nem érzékeli megfelelően a kijelzőt, indítsa újra kézzel.',
          powerConfirmOk: 'Folytatás',
          powerConfirmOkCountdown: 'Folytatás ({{seconds}})',
          powerConfirmCancel: 'Mégse',
          errors: {
            badRequest: 'Érvénytelen EDID-kérés',
            invalidFileName: 'Válasszon érvényes fájlnevű .bin EDID-fájlt',
            fileTooLarge: 'Az EDID-fájl túl nagy',
            invalidLength: 'Az EDID méretének 128, 256, 384 vagy 512 bájtnak kell lennie',
            invalidHeader: 'Az EDID fejléce érvénytelen',
            unsupportedVersion: 'Ez az EDID-verzió nem támogatott',
            extensionCountMismatch: 'Az EDID bővítőblokkjainak száma nem egyezik a fájl hosszával',
            invalidChecksum: 'Az EDID ellenőrző összege érvénytelen',
            invalidCta: 'A CTA-bővítmény szerkezete érvénytelen',
            currentCannotOverwrite:
              'A jelenleg használt EDID nem írható felül. Először váltson másik EDID-re',
            saveFailed: 'Nem sikerült menteni az EDID-t',
            switchFailed: 'Nem sikerült átváltani az EDID-t',
            uploadFailed: 'Nem sikerült feltölteni az EDID-t',
            deleteFailed: 'Nem sikerült törölni az EDID-t',
            networkFailed: 'Hálózati hiba. Próbálja újra'
          }
        },
        rateControlMode: {
          title: 'Bitráta vezérlési mód',
          description: 'Válassza ki a bitráta szabályozásának módját',
          cbr: 'CBR (állandó bitsebesség): A kapcsolat stabilitását helyezi előtérbe. Minimalizálja a akadozást, de a képminőség romolhat nagy mozgású jelenetek során.',
          vbr: 'VBR (változó bitsebesség): A kép tisztaságát részesíti előnyben. A nagy mozgású jelenetek sávszélesség-ugrásokat okozhatnak, ami késleltetést vagy akadozást okozhat.'
        },
        bitrate: {
          title: 'Bitráta',
          description: 'Egyensúlyozza a kép tisztaságát a sávszélesség-használattal',
          lower:
            'Az alacsonyabb értékek csökkentik a késleltetést gyenge hálózatokon, de rontják a képminőséget, ami elmosódást vagy műtermékeket okozhat.',
          higher:
            'A magasabb értékek javítják a kép tisztaságát, de nagyobb sávszélességet fogyasztanak. Növelheti a késleltetést, ha a kapcsolat lassú.'
        },
        gop: {
          description: 'Állítsa be a teljes kulcskockák közötti intervallumot',
          lower:
            'Az alacsonyabb értékek gyakrabban küldenek teljes képkockákat. Ez felgyorsítja a vizuális műtermékek helyreállítását, de nagyobb sávszélességet fogyaszt, és csökkenti a kép részleteit.',
          higher:
            'A magasabb értékek növekményes frissítésekre optimalizálnak. Ez javítja a kép részleteit és sávszélességet takarít meg, de a vizuális műtermékek helyreállítása tovább tart.'
        },
        quality: {
          title: 'Minőség',
          description: 'Állítsa be a kijelző képminőségét'
        },
        fps: {
          auto: 'Automatikus',
          description: 'Állítsa be a képernyő képkockasebességét',
          tip1: 'Az érték 0-ra állítása engedélyezi az Auto módot, a kijelző szinkronizálását a távoli gazdagép képkockasebességével.',
          tip2: 'A képkockasebesség csökkentése jelentősen csökkenti a sávszélesség-használatot. Csökkentse manuálisan, ha a hálózat instabil, vagy ha nincs szükség nagy képkockasebességre.'
        },
        scale: {
          title: 'Skála',
          description: 'Állítsa be a kijelző méretarányát'
        }
      },
      device: {
        title: 'Eszköz',
        virtualAudioConfirm: {
          title: 'Megerősítés a virtuális hang átváltása előtt',
          description:
            'Először csatlakoztassa a PD-tápellátást. A virtuális hang átváltása rövid időre leválaszthatja a kijelzőt; ha ezután az eszköz nem érzékeli megfelelően a kijelzőt, indítsa újra kézzel.',
          ok: 'Átváltás folytatása',
          okCountdown: 'Átváltás folytatása ({{seconds}})',
          cancel: 'Mégse'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi konfigurálása',
          apMode: 'AP mód engedélyezve van, csatlakozzon manuálisan a Wi-Fi-hez',
          others: 'Mások...',
          connect: 'Csatlakozzon a Wi-Fi-hez',
          connectDesc1: 'Kérjük, adja meg a hálózati azonosítót és a jelszót',
          connectDesc2: 'Kérjük, adja meg a jelszót a hálózathoz való csatlakozáshoz',
          disconnect: 'Biztosan leválasztja a hálózatot?',
          ssid: 'Név',
          password: 'Jelszó',
          joinBtn: 'Csatlakozz',
          confirmBtn: 'Rendben',
          cancelBtn: 'Mégse',
          preset: {
            title: 'Előre mentett Wi-Fi',
            description:
              'Mentsen el előre egy cél Wi-Fi-t; az eszköz induláskor automatikusan csatlakozik hozzá',
            empty: 'Nincs',
            addBtn: 'Előbeállítás hozzáadása',
            removeBtn: 'Előbeállítás eltávolítása',
            saveTitle: 'Wi-Fi előre mentése',
            saveDesc:
              'Mentse el most a célhálózatot; az eszköz induláskor automatikusan csatlakozik, amikor észleli',
            saveBtn: 'Mentés',
            passwordPlaceholder: 'Jelszó (nyílt hálózatnál hagyja üresen)',
            saveFailed: 'Az előre mentett Wi-Fi mentése nem sikerült',
            removeFailed: 'Az előre mentett Wi-Fi eltávolítása nem sikerült',
            removeConfirm: 'Biztosan eltávolítja az előre mentett Wi-Fi-t?',
            warning:
              'Mentés után az eszköz minden indításkor megpróbál csatlakozni ehhez a Wi-Fi-hez, amikor észleli, ami zavarhatja a csatlakozást más Wi-Fi hálózatokhoz. Az előre mentett beállítás addig marad érvényben, amíg meg nem módosítja vagy nem törli.'
          }
        },
        capture: {
          title: 'Videórögzítés',
          description: 'Videórögzítés engedélyezése'
        },
        ssh: {
          description: 'Engedélyezze a SSH távoli hozzáférést',
          tip: 'Az SSH engedélyezése után állítson be erős jelszót a root felhasználónak a terminálban.'
        },
        swap: {
          errors: {
            load: 'Nem sikerült betölteni a cserehely állapotát. Próbálja újra rövidesen.',
            update:
              'Nem sikerült frissíteni a cserehely beállítását. Újrapróbálás előtt ellenőrizze az aktuális állapotot.'
          },
          title: 'Swap memória',
          description: 'Tömörített swap memória engedélyezése'
        },
        advanced: 'Speciális beállítások',
        staticIp: {
          title: 'Statikus IP',
          config: 'Konfig',
          tip: 'A kapcsolat megszakadásának megelőzése érdekében ajánlatos úgy konfigurálni, hogy fizikailag hozzáférjen az eszközhöz.',
          document: 'Ellenőrizze a dokumentumot.',
          enable: 'Statikus IP engedélyezése',
          placeholder: 'Formátum: IP/CIDR [átjáró]\nPélda: 192.168.1.10/22',
          notEmpty: 'IP-cím nem lehet üres.',
          notValid: 'IP-cím formátuma érvénytelen.',
          failed: 'A konfiguráció nem sikerült.',
          success: 'A konfiguráció kész. Kérjük, ellenőrizze, hogy az IP-cím elérhető-e.',
          okBtn: 'Rendben',
          cancelBtn: 'Mégse'
        },
        usbNetwork: {
          title: 'USB-hálózat',
          mode: 'Üzemmód',
          server: 'Kiszolgáló',
          serverDesc: 'IP-cím kiosztása a távoli gazdagépnek a beépített DHCP-kiszolgálóval.',
          client: 'Ügyfél',
          clientDesc: 'IP-cím lekérése a távoli gazdagéptől DHCP-n keresztül.',
          forwarding: 'IP-továbbítás',
          forwardingTip:
            'Ahhoz, hogy a távoli gazdagép a NanoKVM-en keresztül elérje az internetet, engedélyezze az IP-továbbítást.',
          manualIp: 'Kézi IP-cím',
          ipv4Placeholder: 'IPv4/CIDR, például 10.10.10.1/24',
          invalidIpv4: 'Adjon meg érvényes IPv4/CIDR-címet.',
          loadFailed: 'Az USB-hálózat beállításai nem tölthetők be.',
          failed: 'A beállítás sikertelen.',
          success: 'A beállítás elkészült.',
          applyBtn: 'Alkalmaz',
          cancelBtn: 'Mégse'
        },
        usbId: {
          title: 'USB eszközazonosító',
          config: 'Beállítás',
          tip: 'Az egyéni USB azonosítók megváltoztatják, hogyan ismeri fel a gazda ezt az eszközt. Alkalmazáskor az USB gadget újraindul, a billentyűzet, egér vagy hálózat rövid ideig megszakadhat.',
          enable: 'Egyéni USB eszközazonosító engedélyezése',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Eszközverzió (bcdDevice)',
          serialnumber: 'Sorozatszám',
          manufacturer: 'Gyártó',
          product: 'Termék',
          vidPlaceholder: 'pl. 0x359F',
          pidPlaceholder: 'pl. 0x3305',
          bcdDevicePlaceholder: 'pl. 0x0101 (opcionális)',
          serialnumberPlaceholder: 'pl. 0123456789ABCDEF (opcionális)',
          manufacturerPlaceholder: 'pl. sipeed (opcionális)',
          productPlaceholder: 'pl. NanoKVM-Go (opcionális)',
          notEmpty: 'A VID és a PID nem lehet üres.',
          notValid: 'Érvénytelen VID/PID formátum. Használjon hexet, pl. 0x359F.',
          bcdNotValid:
            'Érvénytelen bcdDevice formátum. Használjon hexet, pl. 0x0101, vagy hagyja üresen.',
          failed: 'A konfiguráció sikertelen.',
          success: 'Konfiguráció kész. USB gadget újraindítva.',
          okBtn: 'OK',
          cancelBtn: 'Mégse'
        },
        diskInquiry: {
          title: 'USB-meghajtó neve',
          config: 'Beállítás',
          tip: 'A virtuális USB-pendrive neve, ahogy a távoli számítógép látja. Mentéskor az USB újraindul, a billentyűzet, egér vagy hálózat rövid időre megszakadhat.',
          enable: 'Egyéni USB-meghajtónév használata',
          placeholder: 'pl. sipeed  Flash Drive     1.00',
          notEmpty: 'A név nem lehet üres.',
          notValid: 'Érvénytelen név. Legfeljebb 28 nyomtatható ASCII-karakter sortörés nélkül.',
          failed: 'A konfiguráció sikertelen.',
          success: 'Konfiguráció kész. USB-gadget újraindítva.',
          okBtn: 'OK',
          cancelBtn: 'Mégse'
        },
        diskInquiryCdrom: {
          title: 'Virtuális CD-meghajtó neve',
          config: 'Beállítás',
          tip: 'A virtuális CD/DVD-meghajtó neve a távoli számítógépen. Mentéskor az USB újraindul, a billentyűzet, egér vagy hálózat rövid időre megszakadhat.',
          enable: 'Egyéni CD-meghajtónév használata',
          placeholder: 'pl. sipeed  Optical Drive   1.00',
          notEmpty: 'A név nem lehet üres.',
          notValid: 'Érvénytelen név. Legfeljebb 28 nyomtatható ASCII-karakter sortörés nélkül.',
          failed: 'A konfiguráció sikertelen.',
          success: 'Konfiguráció kész. USB-gadget újraindítva.',
          okBtn: 'OK',
          cancelBtn: 'Mégse'
        },
        uac2FunctionName: {
          title: 'USB hang neve',
          config: 'Beállítás',
          tip: 'A virtuális USB-mikrofon / hangeszköz neve a távoli számítógépen. Mentéskor az USB újraindul, a billentyűzet, egér vagy hálózat rövid időre megszakadhat.',
          enable: 'Egyéni USB hangnév használata',
          placeholder: 'pl. NanoKVM-Go Audio',
          notEmpty: 'A név nem lehet üres.',
          notValid: 'Érvénytelen név. Legfeljebb 64 nyomtatható karakter sortörés nélkül.',
          failed: 'A konfiguráció sikertelen.',
          success: 'Konfiguráció kész. USB-gadget újraindítva.',
          okBtn: 'OK',
          cancelBtn: 'Mégse'
        },
        lowPower: {
          title: 'Alacsony fogyasztású mód',
          description: 'Csökkentse az energiafogyasztást üresjáratban',
          tips: 'Tippek',
          tip1: 'Ha engedélyezve van, a videórögzítés szünetel az üresjárati időszakokban az energiafogyasztás és a hőtermelés csökkentése érdekében.',
          tip2: 'A videórögzítés folytatásához újrainicializálásra lehet szükség, amely során a videó átmenetileg nem érhető el.',
          okBtn: 'Rendben',
          cancelBtn: 'Mégse'
        },
        mouseJiggler: {
          title: 'Egér mozgató',
          description: 'A távoli gazdagép alvó állapotának megakadályozása',
          disable: 'Letiltás',
          absolute: 'Abszolút mód',
          relative: 'Relatív mód'
        },
        fingerRobot: {
          title: 'Ujjrobot',
          description: 'Ujjrobot vezérlés engedélyezése'
        },
        mdns: {
          description: 'Engedélyezze az mDNS felderítési szolgáltatást'
        },
        hidOnly: 'HID-Csak mód',
        imageMounted: 'Kép csatolva',
        disk: 'Virtuális lemez',
        diskDesc: 'Virtuális U-lemez csatolása a távoli gépen',
        sdCard: {
          warning: 'Virtuális lemez inicializálása',
          tip1: 'Az első engedélyezéskor a virtuális lemez inicializálódik, és az eszközön lévő képfájlok, valamint a virtuális lemez adatai törlődnek. Ez a művelet nem vonható vissza. Előbb készítsen biztonsági másolatot a fontos fájlokról.',
          tip2: 'Folytatja az inicializálást?',
          capacity: 'Kapacitás',
          disable: 'Letiltás',
          disableWarning: 'Letiltja a virtuális lemezt?',
          disableTip:
            'A letiltás törli a virtuális lemez összes fájlját, és felszabadítja az általa használt eszköztárhelyet. A következő engedélyezéskor újra ki kell választania a kapacitást és inicializálnia kell a lemezt. Ez a művelet nem vonható vissza.'
        },
        network: 'Virtuális hálózat',
        networkDesc: 'Virtuális hálózati kártya csatolása a távoli gépen',
        mic: 'Virtuális hang',
        micDesc: 'Válassza ki a távoli gép USB virtuális hangmódját',
        virtualAudioMode: {
          off: 'Kikapcsolva',
          mic: 'Csak mikrofon',
          duplex: 'Mikrofon + hangszóró'
        },
        speaker: 'Virtuális hangszóró',
        speakerDesc:
          'Bekapcsolva lecseréli az alapértelmezett DP hangkimenetet, így jobb kompatibilitással támogatja a távoli hívásokat. Kikapcsolva csak a médiahang továbbítható, a híváshang nem.',
        micTip:
          'Az USB-végpont korlátai miatt a Virtuális mikrofon engedélyezése letiltja az USB másodlagos képernyőt.\nJavasoljuk, hogy a BIOS-műveletek során tiltsa le ezt a funkciót az optimális kompatibilitás biztosítása érdekében.',
        reboot: 'Újraindítás',
        rebootDesc: 'Biztos, hogy újra akarja indítani a NanoKVM-t?',
        rebootToFlashing: 'Újraindítás flash módba',
        rebootToFlashingButton: 'Újraindítás',
        rebootToFlashingTitle: 'Újraindítás flash módba?',
        rebootToFlashingDesc:
          'A NanoKVM újraindul, és USB flash módba lép. Győződjön meg arról, hogy a flash művelethez használt USB-csatlakozó csatlakoztatva van, és a flash eszköz készen áll.',
        datetime: {
          timezone: 'Időzóna',
          datetime: 'Dátum és idő',
          dateAndTime: 'Dátum és idő',
          synchronize: 'Idő szinkronizálása',
          lastSynchronization: 'Utolsó szinkronizálás: ',
          notSynchronized: 'A rendszeridő még nincs szinkronizálva',
          syncNow: 'Szinkronizálás most',
          manualSet: 'Kézi beállítás',
          manualSetTitle: 'Idő kézi beállítása',
          manualSetDescription: 'A módosítás után újra be kell jelentkeznie.',
          useBrowserTime: 'Böngésző idejének használata',
          setTime: 'Idő beállítása',
          invalidTime: 'Adjon meg érvényes dátumot és időt.',
          setFailed: 'A rendszeridő nem állítható be. Próbálja újra.',
          setSucceeded: 'A rendszeridő frissült. Jelentkezzen be újra.'
        },
        dialBackground: {
          errors: {
            load: 'Nem sikerült betölteni a számlap háttérképeit. Próbálja újra rövidesen.',
            upload:
              'Nem sikerült feltölteni a háttérképet. Ellenőrizze, hogy a fájl érvényes BMP-kép-e, majd próbálja újra.',
            switch:
              'Nem sikerült megváltoztatni a számlap hátterét. Frissítse a képlistát, majd próbálja újra.'
          },
          title: 'Tárcsa háttere',
          description: 'Válasszon BMP-képet a /kvmcomm/ui/bg/ mappából tárcsaháttérnek',
          upload: 'Feltöltés',
          loading: 'Betöltés...',
          empty: 'Nem található BMP-kép',
          current: 'Jelenlegi',
          onlyBmp: 'Csak BMP-képek támogatottak',
          uploaded: 'Feltöltés befejezve',
          switched: 'Háttér átváltva'
        },
        okBtn: 'Igen',
        cancelBtn: 'Nem'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Újraindítja a Tailscale-t?',
        stop: 'Leállítja a Tailscale-t?',
        stopDesc:
          'Kijelentkezés a Tailscale-ből, és az automatikus indítás letiltása rendszerindításkor.',
        loading: 'Betöltés...',
        notInstall: 'Tailscale nem található! Kérem, telepítse.',
        install: 'Telepítés',
        installing: 'Telepítés folyamatban',
        failed: 'Telepítés sikertelen',
        retry: 'Frissítse az oldalt, majd próbálja újra. Vagy próbálja meg manuálisan telepíteni.',
        download: 'Letöltés a',
        package: 'telepítési csomag',
        unzip: 'és kicsomagolás',
        upTailscale: 'Töltsön fel tailscale-t a NanoKVM /usr/bin/ könyvtárába',
        upTailscaled: 'Töltsön fel tailscaled-t a NanoKVM /usr/sbin/ könyvtárába',
        refresh: 'Frissítse az aktuális oldalt',
        notRunning: 'Tailscale nem fut. Kérjük, indítsa el a folytatáshoz.',
        run: 'Indítás',
        notLogin:
          'Az eszköz még nincs kötve. Kérem, jelentkezzen be és kösse az eszközt a fiókjához.',
        urlPeriod: 'Ez az url 10 percig érvényes',
        login: 'Bejelentkezés',
        loginSuccess: 'Sikeres bejelentkezés',
        enable: 'Tailscale engedélyezése',
        exitNode: 'Kilépési csomópont',
        exitNodeDesc: 'Az eszköz hirdetése internetes kijáratként a tailnet számára.',
        subnetRoute: 'Alhálózati útvonal',
        subnetRouteDesc: 'A közvetlenül csatlakozó {{subnet}} LAN-alhálózat hirdetése a tailneten.',
        subnetUnavailable: 'Nem érhető el közvetlenül csatlakozó LAN-alhálózat.',
        routingUnavailable: 'A Tailscale útválasztási beállításai nem érhetők el ezen az eszközön.',
        pendingApproval:
          'Engedélyezés után hagyja jóvá a hirdetett útvonalat a Tailscale felügyeleti konzolján.',
        routingUpdateFailed: 'Nem sikerült frissíteni a Tailscale útválasztási beállításait.',
        deviceName: 'Eszköz neve',
        deviceIP: 'Eszköz IP',
        account: 'Fiók',
        logout: 'Kijelentkezés',
        logoutDesc: 'Biztos, hogy ki szeretne jelentkezni?',
        uninstall: 'Eltávolítás Tailscale',
        uninstallDesc: 'Biztosan eltávolítja a Tailscale alkalmazást?',
        okBtn: 'Igen',
        cancelBtn: 'Nem'
      },
      update: {
        title: 'Frissítés keresése',
        queryFailed: 'Verzió lekérdezése sikertelen',
        updateFailed: 'Frissítés sikertelen. Kérem, próbálja újra.',
        isLatest: 'Ön már a legfrissebb verziót használja.',
        available: 'Frissítés elérhető. Biztos, hogy frissít?',
        updating: 'Frissítés elkezdődött. Kérem várjon...',
        confirm: 'Megerősítés',
        upgrade_tip_nano: 'Elérhető a NanoKVM új verziója. Biztos, hogy most frissíteni akar?',
        upgrade_tip_firmware: 'Elérhető egy új firmware-verzió. Biztos, hogy most frissíteni akar?',
        cancel: 'Mégse',
        preview: 'Frissítések előnézete',
        previewDesc: 'Korai hozzáférést kap az új funkciókhoz és fejlesztésekhez',
        previewTip:
          'Kérjük, vegye figyelembe, hogy az előzetes verziók hibákat vagy hiányos funkciókat tartalmazhatnak!',
        download: 'Letöltés',
        install: 'Telepítés',
        changelog: 'Változásnapló',
        restart: 'Indítsa újra'
      },
      account: {
        title: 'Fiók',
        webAccount: 'Webes fiók neve',
        password: 'Jelszó',
        twoFactor: 'Kétfaktoros hitelesítés',
        enableTwoFactor: 'Engedélyezés',
        enableTwoFactorDesc:
          'Adja meg jelenlegi jelszavát a kétfaktoros hitelesítés beállításának megkezdéséhez.',
        disableTwoFactor: 'Letiltás',
        recoveryCodes: 'Helyreállítási kódok',
        enableRecoveryCodes: 'Engedélyezés',
        updateRecoveryCodes: 'Frissítés',
        recoveryCodesTooltipDisabled:
          'A helyreállítási kódok tartalék kétfaktoros hitelesítési adatok. A kétfaktoros hitelesítés engedélyezése után generálhatók.',
        recoveryCodesTooltipEnabled:
          'A helyreállítási kódok csak akkor érvényesek, ha a kétfaktoros hitelesítés engedélyezve van. A kétfaktoros hitelesítés letiltása minden helyreállítási kódot érvénytelenít; amíg engedélyezve marad, csak frissíthetők.',
        recoveryCodesRegenerateDesc:
          'Adja meg jelenlegi jelszavát, valamint vagy egy hitelesítési kódot, vagy egy fel nem használt helyreállítási kódot. A helyreállítási kódok frissítése érvényteleníti az összes meglévő helyreállítási kódot.',
        recoveryCodesVerifyFailed:
          'Nem sikerült ellenőrizni a jelszót és a hitelesítési kódot vagy a helyreállítási kódot.',
        recoveryCodesSavedTitle: 'Mentse el ezeket a helyreállítási kódokat most',
        recoveryCodesSavedDesc:
          'A helyreállítási kódok csak egyszer tekinthetők meg. Mentse őket biztonságos helyre, mielőtt elhagyja ezt az oldalt.',
        enableRecoveryCodesDesc:
          'Adja meg jelenlegi jelszavát és egy hitelesítési kódot a helyreállítási kódok generálásához. A helyreállítási kódok csak egyszer tekinthetők meg, ezért mentse őket az oldal elhagyása előtt.',
        copyRecoveryCodes: 'Másolás',
        downloadRecoveryCodes: 'Letöltés',
        recoveryCodesCopyFailed: 'Nem sikerült másolni a helyreállítási kódokat.',
        done: 'Kész',
        currentPassword: 'Jelenlegi jelszó',
        authCode: 'Hitelesítési kód',
        authCodeOrRecoveryCode: 'Hitelesítési kód vagy helyreállítási kód',
        authRateLimited: 'Túl sok sikertelen próbálkozás. Próbálja újra {{seconds}} mp múlva.',
        verifyPasswordChange: 'Jelszómódosítás ellenőrzése',
        verifyPasswordChangeDesc:
          'A jelszó módosítása előtt adja meg jelenlegi jelszavát, valamint vagy egy hitelesítési kódot, vagy egy fel nem használt helyreállítási kódot.',
        verifyPasswordChangeFailed:
          'Nem sikerült ellenőrizni a jelszót és a hitelesítési kódot vagy a helyreállítási kódot.',
        verifyAndEnable: 'Ellenőrzés és engedélyezés',
        scanTwoFactorQRCode:
          'Olvassa be ezt a QR-kódot a hitelesítő alkalmazással, majd adja meg a kódot.',
        showSetupKey: 'Beállítási kulcs megjelenítése',
        hideSetupKey: 'Beállítási kulcs elrejtése',
        setupKey: 'Beállítási kulcs',
        twoFactorSetupFailed: 'Nem sikerült elindítani a kétfaktoros hitelesítés beállítását.',
        twoFactorVerifyFailed: 'Nem sikerült ellenőrizni a hitelesítési kódot.',
        twoFactorDisableFailed: 'Nem sikerült letiltani a kétfaktoros hitelesítést.',
        disableTwoFactorDesc:
          'Adja meg jelenlegi jelszavát, valamint vagy egy hitelesítési kódot, vagy egy fel nem használt helyreállítási kódot. A kétfaktoros hitelesítés letiltása minden helyreállítási kódot érvénytelenít.',
        updateBtn: 'Módosítás',
        logoutBtn: 'Kijelentkezés',
        logoutDesc: 'Biztos, hogy ki szeretne jelentkezni?',
        okBtn: 'Igen',
        cancelBtn: 'Nem',
        twoFactorProtection: {
          title: 'Speciális 2FA-védelem',
          description:
            'Válassza ki azokat az érzékeny műveleteket, amelyekhez második tényező kell a védelem érvényesítésekor. A „Közvetlen webes bejelentkezés” kapcsoló ennek az ellenkezőjét teszi: csak hitelesítési kóddal vagy fel nem használt helyreállítási kóddal enged bejelentkezni.',
          login: 'Közvetlen webes bejelentkezés',
          webTerminal: 'Webterminál',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API-kulcs',
          targetPower: 'Célgép tápellátása',
          nanoKVMRestart: 'NanoKVM újraindítása',
          deleteImage: 'Kép törlése',
          scripts: 'Szkriptműveletek',
          screenTimelapse: 'Képernyő-időzített felvétel',
          loginScope:
            'A webes bejelentkezést csak hitelesítési vagy fel nem használt helyreállítási kóddal engedélyezi felhasználónév és jelszó helyett. Ez csökkenti a bejelentkezés biztonságát.',
          webTerminalScope: 'A webterminál megnyitása.',
          sshScope: 'Az SSH-szolgáltatás állapotának módosítása.',
          tailscaleScope: 'A Tailscale telepítése, csatlakoztatása vagy módosítása.',
          frp: 'FRP',
          frpScope: 'A FRP telepítése, csatlakoztatása vagy módosítása.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'A Cloudflare Tunnel telepítése, csatlakoztatása vagy módosítása.',
          zerotierScope: 'A ZeroTier telepítése, csatlakoztatása vagy módosítása.',
          netbirdScope: 'A NetBird telepítése, csatlakoztatása vagy módosítása.',
          mcpScope:
            'Az MCP API-kulcs megjelenítése, másolása vagy újragenerálása, illetve az MCP engedélyezése vagy letiltása.',
          targetPowerScope: 'A célgép leállítása vagy újraindítása a fizikai bekapcsológombbal.',
          nanoKVMRestartScope: 'Magának a NanoKVM-nek az újraindítása.',
          deleteImageScope: 'Tárolt ISO vagy IMG kép végleges törlése.',
          scriptsScope: 'Szkriptek futtatása, feltöltése vagy törlése.',
          screenTimelapseScope:
            'Képernyő-időzített felvétel rögzítésének engedélyezése, a felvételek böngészése és törlése.',
          authorizeTitle: 'Védelmi módosítások engedélyezése',
          authorizeDesc:
            'Adjon meg egy hitelesítési vagy fel nem használt helyreállítási kódot. Az engedély öt percig érvényes.',
          authorizeFailed:
            'Nem sikerült ellenőrizni a hitelesítési kódot vagy a helyreállítási kódot.',
          loadFailed: 'Nem sikerült betölteni a védelmi beállításokat.',
          updateFailed: 'Nem sikerült menteni a védelmi beállításokat.',
          retry: 'Újrapróbálás',
          disableScriptsWarningTitle: 'Kikapcsolja a szkriptvédelmet?',
          disableScriptsWarningDesc:
            'Védelem nélkül egy rosszindulatú szkript emelt szintű hozzáférést szerezhet ehhez az eszközhöz, és megkerülheti vagy eltávolíthatja a 2FA-védelmet.',
          cancelDisabling: 'Mégse',
          continueDisabling: 'Kikapcsolás folytatása',
          continueDisablingCountdown: 'Kikapcsolás folytatása ({{seconds}} mp)'
        },
        twoFactorGuide: {
          title: 'Kétfaktoros hitelesítési útmutató',
          scope: 'A kétfaktoros hitelesítés egy további biztonsági réteget ad a fiókjához.',
          whatTitle: 'Mi az a 2FA?',
          whatDesc:
            'A kétfaktoros hitelesítés (2FA) egy hitelesítő alkalmazásból származó időalapú kóddal védi fiókját. Alapértelmezés szerint a kód a jelszava mellett szükséges, így ha a jelszava kompromittálódik is, a fiók védett marad. Ha a közvetlen webes bejelentkezés be van kapcsolva, önmagában a kód jelentkezteti be.',
          appsTitle: 'Ajánlott alkalmazások',
          appsDesc:
            'Használhatja a Google Authenticatort vagy a Microsoft Authenticatort. Mindkettő ingyenes, és iOS-en valamint Androidon is működik.',
          setupTitle: 'Beállítás',
          setupDesc:
            'Nyissa meg a hitelesítő alkalmazást, olvassa be a beállításkor megjelenő QR-kódot, vagy írja be kézzel a beállítási kulcsot. Ezután adja meg az alkalmazás 6 jegyű kódját a 2FA engedélyezésének befejezéséhez.',
          recoveryTitle: 'Helyreállítási kódok',
          recoveryDesc:
            'A helyreállítási kódok egyszer használatos tartalék kódok. Ha elveszíti a hozzáférést a hitelesítő alkalmazáshoz, helyreállítási kóddal is bejelentkezhet. Létrehozás után azonnal mentse őket biztonságos helyre.',
          logoutNotice:
            'A 2FA engedélyezése kijelentkezteti az összes aktív munkamenetet, beleértve ezt is. Minden eszközön újra be kell jelentkeznie.'
        }
      }
    },
    error: {
      title: 'Problémába ütköztünk',
      refresh: 'Frissítés'
    },
    pictureInPicture: {
      title: 'Kép a képben',
      enter: 'Kép a képben mód bekapcsolása',
      exit: 'Kép a képben mód kikapcsolása',
      error:
        'Nem sikerült elindítani a kép a képben módot. Várja meg, amíg a videó betöltődik, majd próbálja újra.'
    },
    fullscreen: {
      toggle: 'Teljes képernyő váltás'
    },
    menu: {
      collapse: 'Menü összecsukása',
      expand: 'Bontsa ki a menüt'
    }
  }
};

export default hu;
