const cz = {
  translation: {
    storage: {
      cards: { mount: 'Připojení disku', share: 'Sdílení souborů' },
      title: 'Obrázky a sdílení souborů',
      empty: 'Zatím žádné obrazy',
      notAttached: 'Nic není připojeno',
      attached: 'Připojeno: {{name}}',
      eject: 'Vysunout',
      mount: 'Připojit',
      mountMode: 'Režim připojení',
      mountModeDisk: 'Hromadné úložiště',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Otevřít správce souborů',
      refresh: 'Obnovit',
      delete: 'Smazat',
      deleteTitle: 'Smazat médium',
      deleteConfirm: 'Smazat {{name}}? Soubory v něm budou ztraceny.',
      downloadImage: 'Stáhnout obraz',
      typePartition: 'Sdílené soubory',
      volumeLabel: 'Vyměnitelný disk',
      groups: {
        share: 'Sdílení souborů',
        images: 'Obrázky'
      },
      failed: 'Operace se nezdařila',
      hidOnly: 'Režim pouze HID: virtuální disk není k dispozici',
      disabled: 'Virtuální disk je vypnutý (Nastavení - Zařízení)',
      force: {
        title: 'Vzít médium zpět násilím?',
        desc: 'Vzdálený počítač toto médium stále používá. Vzetí zpět znovu vyjmenuje zařízení USB a může ztratit vše, co ještě nezapsal na disk.',
        menuHint:
          'Vzdálený počítač toto médium používá. Otevřete správce souborů a vezměte ho zpět násilím.'
      },
      tips: {
        title: 'Jak nahrát',
        usb1: 'Připojte NanoKVM k vašemu počítači přes USB.',
        usb2: 'Na této stránce připojte sdílené soubory v režimu velkokapacitního úložiště.',
        usb3: 'Otevřete virtuální disk na vašem počítači a zkopírujte soubor s obrazem do kořenového adresáře virtuálního disku.',
        usb4: 'Po dokončení kopírování disk vysuňte: obraz se pak objeví v seznamu.',
        scp1: 'Ujistěte se, že jsou NanoKVM a váš počítač ve stejné místní síti.',
        scp2: 'Otevřete terminál na vašem počítači a použijte příkaz SCP pro nahrání souboru s obrazem do adresáře /data na zařízení NanoKVM.',
        scp3: 'Příklad: scp cesta-k-vašemu-obrazu root@ip-nanokvm:/data'
      },
      job: {
        running: 'Zpracovává se {{name}}'
      },
      enableDisk: {
        title: 'Zapnout virtuální disk?',
        desc: 'K připojení média je potřeba virtuální disk, který je nyní vypnutý. Zapnout ho? Klávesnice a myš vzdáleného počítače se na okamžik odpojí.',
        menuHint: 'Virtuální disk je vypnutý. Zapněte ho v Nastavení - Zařízení.'
      },
      add: {
        title: 'Přidat médium',
        upload: 'Nahrát ISO nebo IMG',
        fromUrl: 'Stáhnout z adresy URL',
        url: 'Adresa URL obrazu',
        urlRequired: 'Zadejte adresu URL',
        onlyImages: 'Připojit lze pouze soubory .iso a .img',
        dropHint: 'Přetáhněte sem ISO nebo IMG, nebo klikněte pro výběr souboru',
        cdromOnlyIso: 'CD-ROM může obsahovat pouze obraz ISO 9660 / UDF',
        uploadFailed: 'Nahrání se nezdařilo',
        downloadFailed: 'Stažení se nezdařilo',
        uploading: 'Nahrávání {{name}}',
        downloading: 'Stahování {{name}}',
        done: '{{name}} je připraven',
        checksumFailed: 'Obraz neodpovídá kontrolnímu součtu SHA-256.',
        blocked:
          'Vzdálený počítač disk používá. Bezpečně vysuňte virtuální USB disk na vzdáleném počítači a potom vypněte přepínač Virtuální disk v Nastavení > Zařízení.'
      },
      files: {
        reasonNotInitialized:
          'Virtuální USB disk není inicializován. Před sdílením souborů jej inicializujte v Nastavení > Zařízení.',
        selectMedium: 'Vyberte médium a prohlédněte si soubory v něm.',
        reasonAttached: 'Toto médium je připojeno ke vzdálenému počítači.',
        reasonAttachedHint:
          'Bezpečně vysuňte virtuální USB disk na vzdáleném počítači a potom vypněte přepínač Virtuální disk v Nastavení > Zařízení.',
        reasonNotBrowsable:
          'Toto médium obsahuje souborový systém, který správce nedokáže otevřít.',
        reasonNoLoop: 'Toto zařízení nedokáže připojit obrazy disků (chybí podpora loop).',
        reasonISOImage:
          'Obraz ISO 9660 zde nelze procházet. Lze ho ale připojit ke vzdálenému počítači.',
        empty: 'Tato složka je prázdná',
        noMatch: 'Vyhledávání neodpovídá žádný soubor.',
        free: '{{size}} volných',
        search: 'Hledat v této složce',
        sort: {
          modified: 'Změněno',
          name: 'Název',
          size: 'Velikost',
          type: 'Typ',
          asc: 'Vzestupně',
          desc: 'Sestupně'
        },
        view: {
          list: 'Seznam',
          grid: 'Zobrazení ikon'
        },
        folder: 'Složka',
        itemCount: 'Počet položek: {{count}}',
        itemCount_one: '{{count}} položka',
        itemCount_other: 'Počet položek: {{count}}',
        dropHint: 'Přetažením sem nahrajete soubory do této složky',
        up: 'O úroveň výš',
        uploadButton: 'Nahrát',
        newFolder: 'Nová složka',
        newFolderPlaceholder: 'Název složky',
        create: 'Vytvořit',
        upload: 'Nahrát soubor do média',
        deleteFolderTitle: 'Smazat složku',
        deleteFolderConfirm: 'Smazat složku {{name}} a celý její obsah? Tuto akci nelze vrátit.',
        uploading: 'Nahrávání…',
        uploadFailed: 'Nahrání se nezdařilo',
        download: 'Stáhnout',
        delete: 'Smazat',
        deleteTitle: 'Smazat soubor',
        deleteConfirm: 'Smazat {{name}} z média?'
      },
      yes: 'Ano',
      no: 'Ne'
    },
    head: {
      storage: 'Obrázky a sdílení souborů',
      desktop: 'Vzdálená plocha',
      login: 'Přihlášení',
      changePassword: 'Změna hesla',
      terminal: 'Terminál',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Nelze kopírovat do schránky. Povolte prohlížeči přístup ke schránce a zkuste to znovu.'
    },
    auth: {
      login: 'Přihlášení',
      language: 'Jazyk',
      placeholderUsername: 'Zadejte prosím uživatelské jméno',
      placeholderPassword: 'Zadejte prosím heslo',
      placeholderPassword2: 'Zadejte prosím heslo znovu',
      noEmptyUsername: 'Uživatelské jméno nesmí být prázdné',
      noEmptyPassword: 'Heslo nesmí být prázdné',
      noAccount:
        'Nepodařilo se získat informace o uživateli, prosím obnovte stránku nebo resetujte heslo',
      invalidUser: 'Neplatné uživatelské jméno nebo heslo',
      error: 'Neočekávaná chyba',
      twoFactorTitle: 'Vyžadováno ověření',
      twoFactorDesc: 'Pro přihlášení zadejte ověřovací kód nebo nepoužitý obnovovací kód.',
      sensitiveActionDesc: 'Zadejte ověřovací kód nebo nepoužitý obnovovací kód pro pokračování.',
      placeholderAuthCode: 'Ověřovací kód',
      noEmptyAuthCode: 'Ověřovací kód je povinný',
      verify: 'Ověřit',
      backToLogin: 'Zpět na přihlášení',
      verificationFailed: 'Ověření selhalo',
      twoFactorEnabledLogout: {
        message:
          'Dvoufaktorové ověřování bylo povoleno. Z bezpečnostních důvodů byly všechny relace odhlášeny. Přihlaste se prosím znovu.'
      },
      rateLimited: 'Příliš mnoho neúspěšných pokusů. Zkuste to znovu za {{seconds}} s.',
      recoveryCodeUsed: 'Obnovovací kód použitý pro toto přihlášení byl zneplatněn.',
      changePassword: 'Změnit heslo',
      changePasswordDesc:
        'Pro bezpečnost vašeho zařízení prosím změňte heslo pro přihlášení na webu.',
      passwordVerificationRequired:
        'Vraťte se na stránku nastavení účtu a tuto změnu hesla znovu ověřte.',
      passwordVerificationExpired:
        'Ověření změny hesla vypršelo. Vraťte se na stránku nastavení účtu a ověřte ji znovu.',
      returnToSettings: 'Zpět do nastavení',
      differentPassword: 'Hesla se neshodují',
      illegalUsername: 'Uživatelské jméno obsahuje nepovolené znaky',
      illegalPassword: 'Heslo obsahuje nepovolené znaky',
      forgetPassword: 'Zapomenuté heslo',
      ok: 'OK',
      cancel: 'Zrušit',
      savePassword: 'Uložit heslo',
      loginButtonText: 'Přihlášení',
      tips: {
        reset1: 'výchozí webový účet:',
        reset2: 'výchozí účet SSH:',
        reset3: 'Chcete-li resetovat hesla, viz ',
        changeReset: 'Chcete-li obnovit heslo pro webové přihlášení, viz ',
        resetLink: 'Resetovat'
      }
    },
    wifi: {
      errors: {
        connect:
          'Nelze se připojit k Wi-Fi. Zkontrolujte název sítě, heslo a stav Wi-Fi zařízení a zkuste to znovu.'
      },
      title: 'Wi-Fi',
      description: 'Konfigurace Wi-Fi pro NanoKVM',
      submitted: {
        title: 'Chvíli strpení — NanoKVM se připojuje k vaší Wi-Fi',
        tipsTitle: 'Co dál?',
        tipSwitch: 'Telefon teď můžete přepnout zpět na svou běžnou Wi-Fi.',
        tipCheck:
          'Za okamžik zkontrolujte na obrazovce NanoKVM novou IP adresu a otevřete ji v prohlížeči.',
        tipRetry:
          'Pokud se po chvíli nic nestane, zkontrolujte název sítě a heslo a zkuste to znovu.'
      },
      invalidMode:
        'Aktuální režim nepodporuje nastavení sítě. Přejděte do svého zařízení a povolte konfigurační režim Wi-Fi.',
      confirmBtn: 'Dobře',
      finishBtn: 'Dokončeno',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Heslo',
      steps: {
        settings: 'Nastavení',
        wifi: 'Wi-Fi',
        enable: 'Povolit'
      },
      scan: {
        title: 'Vyberte Wi-Fi síť',
        description: 'Pokračujte výběrem sítě v okolí.',
        loading: 'Hledání okolních Wi-Fi sítí…',
        empty: 'Nebyly nalezeny žádné okolní Wi-Fi sítě.',
        failed: 'Skenování se nezdařilo. Zadejte údaje ručně.',
        retryBtn: 'Skenovat znovu',
        manualBtn: 'Zadat síť ručně',
        backBtn: 'Zpět na okolní sítě',
        connectTitle: 'Připojit k {{ssid}}',
        connecting: 'Připojování…',
        viewSimple: 'Přepnout na jednoduché zobrazení',
        viewDetailed: 'Přepnout na podrobné zobrazení'
      },
      ap: {
        authTitle: 'Vyžaduje se ověření',
        authDescription: 'Pokračujte zadáním hesla AP',
        authFailed: 'Neplatné heslo AP',
        passPlaceholder: 'AP heslo',
        verifyBtn: 'Ověřte'
      },
      qr: {
        cameraBtn: 'Vyfotit',
        pickBtn: 'Vybrat snímek',
        scanHint:
          'Vyfoťte QR kód Wi-Fi, nebo vyberte jeho snímek obrazovky, například ten, který telefon zobrazí při sdílení Wi-Fi.',
        decoding: 'Čtení QR kódu…',
        notFound:
          'Na fotografii nebyl nalezen žádný QR kód. Umístěte kód doprostřed, zaostřete a zkuste to znovu.',
        invalid: 'Tento QR kód neobsahuje použitelné údaje o Wi-Fi.',
        pickerUnavailable:
          'Nic se neotevřelo? Toto přihlašovací okno může blokovat fotoaparát i fotografie. Otevřete místo toho tuto adresu v prohlížeči telefonu:',
        confirmTitle: 'Připojit NanoKVM k {{ssid}}?',
        confirmDescription:
          'Tyto síťové údaje pocházejí z QR kódu. Před připojením je zkontrolujte.',
        openNetwork: 'Tato síť nemá heslo.',
        connectBtn: 'Připojit',
        otherBtn: 'Vybrat jinou síť'
      }
    },
    notification: {
      browser: {
        title: 'Doporučení prohlížeče',
        description:
          'Pro nejlepší uživatelský zážitek doporučujeme používat prohlížeč Chrome. I když se snažíme o kompatibilitu ve všech hlavních prohlížečích, některé pokročilé funkce nemusí mimo Chrome fungovat tak, jak bylo zamýšleno.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Obrazovka',
      video: 'Režim videa',
      quality: 'Kvalita',
      qualityAuto: 'Automaticky',
      qualityLossless: 'Bezeztrátový',
      qualityHigh: 'Vysoký',
      qualityMedium: 'Střední',
      qualityLow: 'Nízký',
      scale: 'Měřítko',
      noSignal: 'Žádný videosignál',
      captureDisabled:
        'Zachytávání videa je vypnuté. Zapněte ho v „Nastavení > Zařízení > Zachytávání videa“.',
      inconsistentVideoMode: 'Přehrávání pozastaveno. Přehrává se jiný režim videa.',
      bandwidthWarning:
        'Síťová šířka pásma nemusí pro aktuální nastavení videa stačit. Zkuste snížit datový tok nebo snímkovou frekvenci.',
      webrtcConnectionFailed:
        'Připojení WebRTC se nepodařilo navázat. Přepněte na režim videa Direct.',
      webrtcUnavailable: {
        title: 'WebRTC není k dispozici',
        description: 'Povolte WebRTC v prohlížeči nebo přepněte na režim videa Direct.'
      },
      upload: 'Nahrát',
      advanced: 'Pokročilé',
      loading: 'Načítání'
    },
    volume: {
      title: 'Hlasitost'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon není namontován',
      tip1: 'Virtuální mikrofon není namontován na vzdáleném hostiteli. Chcete jej nyní připojit a povolit?',
      tip2: 'Můžete to také spravovat později v "Nastavení > Zařízení > Virtuální zvuk".',
      confirm: 'Povolit',
      cancel: 'Zrušit',
      noDevice:
        'Na tomto zařízení nebyl zjištěn žádný mikrofon. Připojte mikrofon a zkuste to znovu.',
      permissionDenied: 'Přístup k mikrofonu byl zamítnut. Povolte jej v nastavení prohlížeče.',
      notReadable: 'Mikrofon používá jiná aplikace nebo jej nelze načíst.',
      overconstrained: 'Mikrofon nepodporuje požadované parametry zvuku.',
      securityBlocked:
        'Přístup k mikrofonu je na této stránce blokován. Zkontrolujte nastavení zabezpečení prohlížeče.',
      aborted: 'Spuštění mikrofonu bylo přerušeno. Zkuste to znovu.',
      unsupported: 'Tento prohlížeč nebo způsob připojení nepodporuje snímání zvuku z mikrofonu.',
      enableFailed: 'Mikrofon se nepodařilo zapnout. Zkuste to znovu.',
      disconnected: 'Mikrofon byl odpojen, proto byl jeho přenos vypnut.'
    },
    power: {
      title: 'Napájení',
      showConfirm: 'Potvrzení',
      showConfirmTip: 'Operace napájení vyžadují další potvrzení',
      powerShort: 'Napájení (krátký stisk)',
      powerLong: 'Napájení (dlouhý stisk)',
      powerConfirm: 'Provést operaci napájení?',
      okBtn: 'Ano',
      cancelBtn: 'Ne'
    },
    keyboard: {
      errors: {
        paste:
          'Text nelze odeslat do vzdáleného zařízení. Zkontrolujte připojení KVM a zkuste to znovu.'
      },
      title: 'Klávesnice',
      paste: 'Vložit',
      tips: 'Text se zadává pomocí vybraného rozložení klávesnice cílového zařízení.',
      placeholder: 'Zadejte text',
      submit: 'Odeslat',
      layouts: 'Klávesnice',
      currentKeyboard: 'Aktuální klávesnice',
      virtual: 'Zobrazit virtuální klávesnici',
      canadianMultilingual: 'Kanadské vícejazyčné standardní',
      layoutError: 'Nelze získat přístup k rozložení klávesnice cílového zařízení.',
      swapCommandCtrl: 'Prohodit Cmd a Ctrl',
      shortcut: {
        title: 'Zkratky',
        custom: 'Vlastní',
        capture: 'Kliknutím sem zachytíte zástupce',
        clear: 'Vymazat',
        save: 'Uložit',
        captureTips:
          'Zachycení systémových kláves (například klávesy Windows) vyžaduje oprávnění pro celou obrazovku.',
        enterFullScreen: 'Přepnout režim celé obrazovky.'
      },
      leaderKey: {
        title: 'Klávesa Leader',
        desc: 'Obejít omezení prohlížeče a odeslat systémové zkratky přímo vzdálenému hostiteli.',
        howToUse: 'Jak používat',
        simultaneous: {
          title: 'Simultánní režim',
          desc1: 'Stiskněte a podržte klávesu Leader a poté stiskněte zkratku.',
          desc2: 'Intuitivní, ale může být v rozporu se systémovými zkratkami.'
        },
        sequential: {
          title: 'Sekvenční režim',
          desc1:
            'Stiskněte klávesu Leader → postupně stiskněte zkratku → znovu stiskněte klávesu Leader.',
          desc2: 'Vyžaduje více kroků, ale zcela se vyhne systémovým konfliktům.'
        },
        enable: 'Povolit klávesu Leader',
        tip: 'Když je tato klávesa nastavena jako klávesa Leader, slouží pouze jako spouštěč zkratek a ztrácí své výchozí chování.',
        placeholder: 'Stiskněte klávesu Leader',
        shiftRight: 'Pravý Shift',
        ctrlRight: 'Pravý Ctrl',
        metaRight: 'Pravý Win',
        submit: 'Odeslat',
        recorder: {
          rec: 'REC',
          activate: 'Aktivovat klávesy',
          input: 'Stiskněte prosím zkratku...'
        }
      }
    },
    mouse: {
      title: 'Myš',
      mode: 'Režim myši',
      absolute: 'Sledovací myš',
      absoluteAndroid: 'Sledovací myš (lepší kompatibilita s Androidem)',
      touchScreen: 'Vícedotyková obrazovka',
      relative: 'Exkluzivní myš',
      direction: 'Směr kolečka',
      scrollUp: 'Přejděte nahoru',
      scrollDown: 'Přejděte dolů',
      speed: 'Rychlost kolečka',
      fast: 'Rychle',
      slow: 'Pomalu',
      requestPointer:
        'Používá se relativní režim. Klikněte prosím na plochu pro získání kurzoru myši.',
      repairIphoneDrag: 'Opravit tažení na iPhonu',
      resetHid: 'Resetovat HID',
      touchpadGuide: {
        title: 'Průvodce touchpadem',
        scope: 'Platí pro běžné ovládání ukazatele, když je režim myši relativní.',
        swipeTitle: 'Přejetím posunete',
        swipeDesc: 'Přejeďte v aktivní oblasti obrazovky, aby se vzdálený ukazatel myši posunul.',
        tapTitle: 'Klepnutím kliknete',
        tapDesc: 'Krátké klepnutí odešle jedno kliknutí levým tlačítkem.',
        holdTitle: 'Podržením stisknete levé tlačítko',
        holdDesc:
          'Dotkněte se a držte prst v klidu asi 1 sekundu, aby levé tlačítko myši zůstalo stisknuté.',
        dragTitle: 'Po podržení pohybem přetáhnete',
        dragDesc:
          'Jakmile je podržení aktivní, pohybem prstu přetáhnete se stisknutým levým tlačítkem.'
      },
      hidOnly: {
        errors: {
          load: 'Nelze načíst stav režimu HID-Only. Zkuste to za chvíli znovu.',
          update:
            'Nelze potvrdit, zda byl režim HID-Only aktualizován. Znovu otevřete toto nastavení, zkontrolujte aktuální stav a zkuste to znovu.'
        },
        title: 'Režim pouze HID',
        desc: 'Pokud vaše myš a klávesnice přestanou reagovat a resetování HID nepomůže, může jít o problém s kompatibilitou mezi NanoKVM a zařízením. Zkuste povolit režim HID-Only pro lepší kompatibilitu.',
        tip1: 'Povolení režimu HID-Only odpojí virtuální U-disk a virtuální síť',
        tip2: 'V režimu HID-Only je připojení obrazu zakázáno',
        enable: 'Povolit režim HID-Only',
        disable: 'Zakázat režim HID-Only'
      }
    },
    preview: {
      title: 'Náhled'
    },
    image: {
      storageReadOnly: 'Do úložiště nelze zapisovat. Vypněte virtuální disk.',
      storageBusy:
        'Úložiště je zaneprázdněné. Před přepnutím virtuálního disku dokončete nebo zrušte přenos souborů.',
      title: 'Obrazy',
      loading: 'Načítání...',
      empty: 'Nic nenalezeno',
      upload: 'Nahrajte obrázek',
      dropFile: 'Přetáhněte obraz sem nebo kliknutím vyberte soubor',
      uploadAction: 'Nahrát',
      cancelAction: 'Zrušit',
      mount: 'Připojit obraz',
      mounted: 'Namontováno',
      uploadError: 'Nahrávání se nezdařilo, zkuste to prosím znovu',
      uploadInsufficientStorage: 'Nedostatek úložného prostoru',
      uploadTooLarge: 'Obrázek překračuje limit velikosti',
      cancelUploadError: 'Nahrávání se nepodařilo zrušit',
      mountMode: 'Režim připojení',
      massStorage: 'Velkokapacitní úložiště',
      cdRom: 'CD ROM',
      readOnly: 'Pouze pro čtení',
      mountFailed: 'Připojení se nezdařilo',
      mountDesc:
        'V některých systémech je nutné před připojením obrazu vysunout virtuální disk na vzdáleném hostiteli.',
      unmountFailed: 'Odpojení selhalo',
      unmountDesc:
        'V některých systémech je nutné před odpojením ručně vysunout obraz na vzdáleném hostiteli.',
      refresh: 'Obnovte seznam obrázků',
      download: 'Stáhnout obrázek',
      attention: 'Pozor',
      deleteConfirm: 'Opravdu chcete smazat tento obrázek?',
      deleteFailed: 'Obrázek se nepodařilo smazat',
      okBtn: 'Ano',
      cancelBtn: 'Ne'
    },
    script: {
      title: 'Skript',
      upload: 'Nahrát',
      run: 'Spustit',
      runBackground: 'Spustit na pozadí',
      runFailed: 'Spuštění se nezdařilo',
      uploadFailed: 'Skript se nepodařilo nahrát',
      deleteFailed: 'Skript se nepodařilo odstranit',
      attention: 'Pozor',
      delDesc: 'Opravdu chcete tento soubor smazat?',
      confirm: 'Ano',
      cancel: 'Ne',
      delete: 'Smazat',
      close: 'Zavřít'
    },
    terminal: {
      title: 'Terminál',
      confirm: 'OK',
      submit: 'Odeslat',
      username: 'Uživatelské jméno',
      password: 'Heslo',
      authCode: 'Ověřovací kód',
      authorizeTitle: 'Autorizovat webový terminál',
      codeAccountHint: 'Zadejte ověřovací kód nebo nepoužitý obnovovací kód.',
      authFailed: 'Ověření selhalo'
    },
    wol: {
      errors: {
        send: 'Nelze odeslat příkaz probuzení. Zkontrolujte adresu MAC a síťové připojení a zkuste to znovu.',
        loadFailed:
          'Nelze načíst seznam zařízení Wake-on-LAN. Zkontrolujte, zda je NanoKVM online, a zkuste to znovu.',
        renameFailed: 'Toto zařízení nelze přejmenovat. Zkontrolujte adresu MAC a zkuste to znovu.',
        deleteFailed: 'Toto zařízení nelze odstranit. Zkuste to za chvíli znovu.'
      },
      title: 'Wake-on-LAN',
      sending: 'Odesílání příkazu...',
      sent: 'Příkaz odeslán',
      input: 'Zadejte prosím MAC adresu',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (volitelné)',
      sha256Placeholder: 'Zadejte 64znakový kontrolní součet SHA-256',
      invalidSHA256: 'SHA-256 musí být 64znakový hexadecimální řetězec',
      title: 'Stahovač obrazů',
      input: 'Zadejte prosím vzdálený obrázek URL',
      ok: 'OK',
      disabled: 'Oddíl /data je RO, takže obrázek nelze stáhnout',
      downloadingFile: 'Stahování: {{file}}',
      downloadingProgress: 'Stahování ({{percentage}}): {{file}}',
      failed: 'Selhalo',
      insufficientStorage: 'Nedostatek úložného prostoru',
      remaining: 'Zbývá {{time}}',
      cancel: 'Zrušit',
      cancelFailed: 'Stahování se nepodařilo zrušit',
      pause: 'Pozastavit',
      resume: 'Pokračovat',
      pauseFailed: 'Stahování se nepodařilo pozastavit',
      resumeFailed: 'Ve stahování se nepodařilo pokračovat'
    },
    settings: {
      back: 'Zpět',
      close: 'Zavřít',
      ai: {
        errors: {
          loadServices:
            'Nelze načíst stav služeb AI. Ověřte, že je NanoKVM online, a zkuste to znovu.',
          loadMcpConfig: 'Nelze načíst konfiguraci služby MCP. Zkuste to za chvíli znovu.',
          updateMcpService:
            'Nelze aktualizovat stav služby MCP. Před dalším pokusem zkontrolujte aktuální stav.',
          accessMcpApiKey:
            'Nelze získat klíč MCP API. Zkuste to znovu a na vyžádání dokončete ověření.',
          regenerateMcpApiKey: 'Nelze znovu vygenerovat klíč MCP API. Zkuste to za chvíli znovu.',
          regeneratedMcpApiKeyUnavailable:
            'Klíč API byl znovu vygenerován, ale nelze jej zobrazit. Zkuste jej zobrazit znovu. Předchozí klíč již není platný.',
          loadMemoryFabricState:
            'Nelze načíst stav služby Memory Fabric. Zkuste to za chvíli znovu.',
          updateMemoryFabricService:
            'Nelze aktualizovat stav služby Memory Fabric. Před dalším pokusem zkontrolujte aktuální stav.',
          loadScreenTimelapseState:
            'Nelze načíst stav služby časosběru obrazovky. Zkuste to za chvíli znovu.',
          updateScreenTimelapseService:
            'Nelze aktualizovat stav služby časosběru obrazovky. Před dalším pokusem zkontrolujte aktuální stav.',
          openScreenTimelapse:
            'Nelze otevřít přehrávání časosběru obrazovky. Zkuste to za chvíli znovu.',
          loadScreenTimelapseRecordings:
            'Nelze načíst záznamy časosběru obrazovky. Zkuste to za chvíli znovu.',
          playScreenTimelapse:
            'Tento časosběrný záznam nyní nelze přehrát. Znovu vyberte datum nebo to zkuste později.',
          screenTimelapseFullscreen:
            'Nelze změnit režim celé obrazovky. Zkontrolujte oprávnění prohlížeče a zkuste to znovu.'
        },
        title: 'AI',
        mcpService: 'Služba MCP',
        mcpServiceDesc:
          'Ověřená jednotná brána pro ovládání, média a povolené nástroje Memory Fabric',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Místní zpracování kontextové paměti agentů a OCR',
        memoryFabricRemoteRequiresMcp:
          'Zapněte výše službu MCP, aby byly nástroje paměti a OCR dostupné vzdáleně.',
        memoryFabricRemoteAvailable:
          'Nástroje paměti a OCR jsou dostupné přes výše uvedený endpoint /api/mcp se stejným API klíčem.',
        memoryFabricConfig: {
          title: 'Konfigurace',
          description: 'Spravujte pouze nastavení výchozího modelu Memory Fabric.',
          loadFailed: 'Konfiguraci Memory Fabric se nepodařilo načíst.',
          saveFailed: 'Konfiguraci Memory Fabric se nepodařilo uložit.',
          saved: 'Konfigurace byla uložena.',
          savedRestarted: 'Konfigurace byla uložena a Memory Fabric byl restartován.',
          noChanges: 'Konfigurace neobsahuje žádné změny.',
          reload: 'Načíst znovu',
          save: 'Uložit',
          unsaved: 'Neuložené změny',
          conflictTitle: 'Konfigurace se změnila',
          conflictDescription:
            'Konfigurace byla změněna jinde. Před dalším uložením načtěte nejnovější verzi.',
          conflictReload: 'Načíst nejnovější verzi',
          discardTitle: 'Zahodit neuložené změny?',
          discardDescription:
            'Opětovné načtení nahradí aktuální formulář naposledy uloženou konfigurací.',
          discardConfirm: 'Zahodit a načíst znovu',
          sections: {
            models: 'Modely'
          },
          openAICompatibleHint:
            'Podporuje poskytovatele modelů kompatibilní s OpenAI. Zadejte název modelu, základní URL a API klíč poskytnutý poskytovatelem.',
          fields: {
            model: 'Model služby',
            baseUrl: 'Základní URL',
            maxTokens: 'Maximální počet tokenů',
            timeoutSeconds: 'Časový limit (sekundy)'
          },
          apiKey: {
            title: 'Klíč API',
            configured: 'Nakonfigurován',
            notConfigured: 'Nenakonfigurován',
            keep: 'Ponechat',
            replace: 'Nahradit',
            clear: 'Odstranit',
            replacement: 'Nový klíč API',
            replacementPlaceholder: 'Zadejte nový klíč API',
            clearHint:
              'Uložený klíč API bude odstraněn; klíč může být nadále poskytnut proměnnou prostředí.',
            required: 'Zadejte nový klíč API.'
          },
          validation: {
            required: 'Toto pole je povinné.',
            invalidUrl: 'Zadejte platnou adresu HTTP nebo HTTPS.'
          }
        },
        screenTimelapseService: 'Časosběr obrazovky',
        screenTimelapseServiceDesc:
          'Přehrávejte historii obrazovky Memory Fabric a vyhledávejte zachycený obsah obrazovky',
        screenTimelapseSearchRequiresMemoryFabric:
          'Chcete-li prohledávat nahraný obsah, zapněte také Memory Fabric.',
        screenTimelapseOpen: 'Otevřít',
        screenTimelapseSearchPlaceholder: 'Hledat obsah ve videu...',
        screenTimelapseNoVideos: 'Nenalezena žádná časosběrná videa',
        screenTimelapseNoResults: 'Nenalezeny žádné odpovídající snímky',
        screenTimelapseMultipleResults: 'Výsledky hledání',
        screenTimelapseUnavailable: 'Nedostupné',
        screenTimelapsePlayerUnsupported: 'Tento prohlížeč neumí přehrát časosběrné video',
        screenTimelapseSearchFailed: 'Vyhledávání selhalo',
        screenTimelapseLoading: 'Načítání časosběru...',
        screenTimelapseSearchAction: 'Hledat v časosběru',
        screenTimelapsePlay: 'Přehrát časosběr',
        screenTimelapsePause: 'Pozastavit časosběr',
        screenTimelapseEnterFullscreen: 'Přejít na celou obrazovku',
        screenTimelapseExitFullscreen: 'Ukončit celou obrazovku',
        screenTimelapseFullscreenUnsupported: 'Tento prohlížeč neumí přejít na celou obrazovku',
        screenTimelapsePreviousDay: 'Předchozí den',
        screenTimelapseNextDay: 'Další den',
        screenTimelapseDeleteDay: 'Odstranit záznam pro {{date}}',
        screenTimelapseDelete: 'Odstranit',
        screenTimelapseDeleteConfirmTitle: 'Odstranit časosběrný záznam?',
        screenTimelapseDeleteConfirmDesc:
          'Záznam pro {{date}} zabírá {{size}}. Tuto akci nelze vrátit zpět.',
        screenTimelapseDeleteSuccess: 'Záznam pro {{date}} byl odstraněn',
        screenTimelapseDeleteFailed: 'Záznam pro {{date}} se nepodařilo odstranit',
        endpoint: 'Koncový bod',
        apiKey: 'Klíč API',
        regenerate: 'Vygenerovat znovu',
        copyApiKey: 'Kopírovat klíč API',
        copyEndpoint: 'Kopírovat koncový bod',
        showApiKey: 'Zobrazit klíč API',
        regenerateConfirmTitle: 'Vygenerovat klíč API znovu',
        regenerateConfirmDesc:
          'Stávající klienti MCP přestanou fungovat, dokud nebudou aktualizováni novým klíčem. Chcete pokračovat?',
        okBtn: 'Ano',
        cancelBtn: 'Ne'
      },
      title: 'Nastavení',
      about: {
        title: 'O NanoKVM',
        information: 'Informace',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Verze aplikace',
        applicationTip: 'Verze webové aplikace NanoKVM',
        arch: 'Architektura',
        device_number: 'Číslo zařízení',
        image: 'Verze obrazu',
        imageTip: 'Verze obrazu systému NanoKVM',
        deviceKey: 'Klíč zařízení',
        community: 'Komunita',
        hostname: 'Název hostitele',
        hostnameUpdated: 'Název hostitele byl aktualizován. Pro použití restartujte.',
        ipType: {
          Wired: 'Kabelové',
          Wireless: 'Bezdrátové',
          Other: 'Jiné'
        }
      },
      appearance: {
        title: 'Vzhled',
        display: 'Zobrazení',
        language: 'Jazyk',
        languageDesc: 'Vyberte jazyk rozhraní',
        languageChangeFailed: 'Změna jazyka selhala',
        webTitle: 'Název webu',
        webTitleDesc: 'Přizpůsobte název webové stránky',
        menuBar: {
          title: 'Panel nabídek',
          mode: 'Režim zobrazení',
          modeDesc: 'Zobrazení panelu nabídek na obrazovce',
          modeOff: 'Vypnuto',
          modeAuto: 'Automatické skrytí',
          modeAlways: 'Vždy viditelné',
          icons: 'Ikony podnabídky',
          showIcon: 'Zobrazit ikonu',
          activateOnConnect: 'Aktivovat při připojení',
          stopSharing: 'Ukončit sdílení',
          iconsDesc: 'Zobrazení ikon podnabídky na liště nabídek'
        }
      },
      screen: {
        title: 'Obrazovka',
        videoMode: {
          title: 'Režim videa',
          description: 'Vyberte způsob kódování a přenosu'
        },
        edid: {
          description: 'Vyberte parametry zobrazení',
          default: 'Výchozí',
          custom: 'Vlastní',
          document: 'Dokument',
          powerConfirmTitle: 'Potvrdit před aktualizací EDID',
          powerConfirmDescription:
            'Nejprve připojte napájení PD. Aktualizace EDID může krátce odpojit displej; pokud zařízení poté nedokáže displej správně detekovat, ručně jej restartujte.',
          powerConfirmOk: 'Pokračovat',
          powerConfirmOkCountdown: 'Pokračovat ({{seconds}})',
          powerConfirmCancel: 'Zrušit',
          errors: {
            badRequest: 'Neplatný požadavek EDID',
            invalidFileName: 'Vyberte soubor EDID .bin s platným názvem',
            fileTooLarge: 'Soubor EDID je příliš velký',
            invalidLength: 'Soubor EDID musí mít 128, 256, 384 nebo 512 bajtů',
            invalidHeader: 'Hlavička EDID je neplatná',
            unsupportedVersion: 'Tato verze EDID není podporována',
            extensionCountMismatch: 'Počet rozšiřujících bloků EDID neodpovídá délce souboru',
            invalidChecksum: 'Kontrolní součet EDID je neplatný',
            invalidCta: 'Struktura rozšíření CTA je neplatná',
            currentCannotOverwrite:
              'Aktuálně používaný EDID nelze přepsat. Nejprve přepněte na jiný EDID',
            saveFailed: 'EDID se nepodařilo uložit',
            switchFailed: 'EDID se nepodařilo přepnout',
            uploadFailed: 'EDID se nepodařilo nahrát',
            deleteFailed: 'EDID se nepodařilo odstranit',
            networkFailed: 'Chyba sítě. Zkuste to znovu'
          }
        },
        rateControlMode: {
          title: 'Režim řízení rychlosti',
          description: 'Vyberte způsob ovládání datového toku',
          cbr: 'CBR (Constant Bitrate): Upřednostňuje stabilitu připojení. Minimalizuje zadrhávání, ale kvalita obrazu může během scén s vysokým pohybem klesnout.',
          vbr: 'VBR (Variable Bitrate): Upřednostňuje čistotu obrazu. Scény s vysokým pohybem mohou způsobit špičky šířky pásma, což může vést k latenci nebo zadrhávání.'
        },
        bitrate: {
          title: 'Přenosová rychlost',
          description: 'Vyvažte čistotu obrazu s využitím šířky pásma',
          lower:
            'Nižší hodnoty snižují latenci ve špatných sítích, ale snižují kvalitu obrazu, což může způsobit rozmazání nebo artefakty.',
          higher:
            'Vyšší hodnoty zlepšují čistotu obrazu, ale spotřebovávají větší šířku pásma. Může zvýšit latenci, pokud je připojení pomalé.'
        },
        gop: {
          description: 'Nastavte interval mezi úplnými klíčovými snímky',
          lower:
            'Nižší hodnoty odesílají celé snímky častěji. To urychluje obnovu z vizuálních artefaktů, ale spotřebovává větší šířku pásma a snižuje detaily obrazu.',
          higher:
            'Vyšší hodnoty se optimalizují pro přírůstkové aktualizace. To zlepšuje detaily obrazu a šetří šířku pásma, ale obnova z vizuálních artefaktů bude trvat déle.'
        },
        quality: {
          title: 'Kvalita',
          description: 'Nastavte kvalitu zobrazení'
        },
        fps: {
          auto: 'Automaticky',
          description: 'Nastavte snímkovou frekvenci zobrazení',
          tip1: 'Nastavením hodnoty na 0 se aktivuje automatický režim, který synchronizuje zobrazení se snímkovou frekvencí vzdáleného hostitele.',
          tip2: 'Snížení snímkové frekvence výrazně snižuje využití šířky pásma. Ručně jej snižte během nestability sítě nebo když není vyžadována vysoká snímková frekvence.'
        },
        scale: {
          title: 'Měřítko',
          description: 'Nastavte poměr měřítka zobrazení'
        }
      },
      device: {
        title: 'Zařízení',
        virtualAudioConfirm: {
          title: 'Potvrdit před přepnutím virtuálního zvuku',
          description:
            'Nejprve připojte napájení PD. Přepnutí virtuálního zvuku může krátce odpojit displej; pokud zařízení poté nedokáže displej správně detekovat, ručně jej restartujte.',
          ok: 'Pokračovat v přepínání',
          okCountdown: 'Pokračovat v přepínání ({{seconds}})',
          cancel: 'Zrušit'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurace Wi-Fi',
          apMode: 'Režim AP je povolen, připojte se k Wi-Fi ručně',
          others: 'Ostatní...',
          connect: 'Připojte se k Wi-Fi',
          connectDesc1: 'Zadejte síťové SSID a heslo',
          connectDesc2: 'Pro připojení k této síti zadejte prosím heslo',
          disconnect: 'Opravdu chcete odpojit síť?',
          ssid: 'Jméno',
          password: 'Heslo',
          joinBtn: 'Připojte se',
          confirmBtn: 'Dobře',
          cancelBtn: 'Zrušit',
          preset: {
            title: 'Přednastavená Wi-Fi',
            description:
              'Předuložte jednu cílovou Wi-Fi; zařízení se k ní při spuštění připojí automaticky',
            empty: 'Žádná',
            addBtn: 'Přidat předvolbu',
            removeBtn: 'Odebrat předvolbu',
            saveTitle: 'Předuložit Wi-Fi',
            saveDesc:
              'Uložte cílovou síť nyní; zařízení se k ní připojí automaticky, když ji při spuštění najde',
            saveBtn: 'Uložit',
            passwordPlaceholder: 'Heslo (u otevřené sítě ponechte prázdné)',
            saveFailed: 'Předvolbu Wi-Fi se nepodařilo uložit',
            removeFailed: 'Předvolbu Wi-Fi se nepodařilo odebrat',
            removeConfirm: 'Opravdu chcete odebrat předuloženou Wi-Fi?',
            warning:
              'Po uložení se zařízení při každém spuštění pokusí připojit k této Wi-Fi, kdykoli ji detekuje, což může narušit připojování k jiným sítím Wi-Fi. Přednastavená síť zůstane aktivní, dokud ji neupravíte nebo neodeberete.'
          }
        },
        capture: {
          title: 'Zachytávání videa',
          description: 'Povolit zachytávání videa'
        },
        ssh: {
          description: 'Povolit vzdálený přístup SSH',
          tip: 'Po povolení SSH nastavte uživateli root silné heslo v terminálu.'
        },
        swap: {
          errors: {
            load: 'Nelze načíst stav odkládací paměti. Zkuste to za chvíli znovu.',
            update:
              'Nelze aktualizovat nastavení odkládací paměti. Před dalším pokusem zkontrolujte aktuální stav.'
          },
          title: 'Odkládací paměť',
          description: 'Povolit komprimovanou odkládací paměť'
        },
        advanced: 'Pokročilá nastavení',
        staticIp: {
          title: 'Statická IP',
          config: 'Konfigurace',
          tip: 'Chcete-li zabránit ztrátě připojení, doporučujeme provést konfiguraci, když máte k zařízení fyzický přístup.',
          document: 'Zkontrolujte dokument.',
          enable: 'Povolit statickou IP',
          placeholder: 'Formát: IP/CIDR [Brána]\nPříklad: 192.168.1.10/22',
          notEmpty: 'IP adresa nemůže být prázdná.',
          notValid: 'Formát IP adresy je neplatný.',
          failed: 'Konfigurace se nezdařila.',
          success: 'Konfigurace dokončena. Zkontrolujte, zda je IP adresa dostupná.',
          okBtn: 'Dobře',
          cancelBtn: 'Zrušit'
        },
        usbNetwork: {
          title: 'Síť USB',
          mode: 'Režim',
          server: 'Serverový režim',
          serverDesc: 'Přiřadí vzdálenému hostiteli IP adresu pomocí vestavěného serveru DHCP.',
          client: 'Klientský režim',
          clientDesc: 'Získá IP adresu od vzdáleného hostitele pomocí DHCP.',
          forwarding: 'Předávání IP',
          forwardingTip:
            'Chcete-li vzdálenému hostiteli umožnit přístup k internetu přes NanoKVM, zapněte předávání IP.',
          manualIp: 'Ruční IP adresa',
          ipv4Placeholder: 'IPv4/CIDR, např. 10.10.10.1/24',
          invalidIpv4: 'Zadejte platnou adresu IPv4/CIDR.',
          loadFailed: 'Konfiguraci sítě USB nelze načíst.',
          failed: 'Konfigurace se nezdařila.',
          success: 'Konfigurace byla dokončena.',
          applyBtn: 'Použít',
          cancelBtn: 'Zrušit'
        },
        usbId: {
          title: 'Identita USB zařízení',
          config: 'Konfigurace',
          tip: 'Vlastní USB ID mění způsob, jakým hostitel identifikuje toto zařízení. Aplikace restartuje USB gadget a může krátce odpojit klávesnici, myš nebo síť.',
          enable: 'Povolit vlastní identitu USB zařízení',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Verze zařízení (bcdDevice)',
          serialnumber: 'Sériové číslo',
          manufacturer: 'Výrobce',
          product: 'Produkt',
          configurationName: 'Název konfigurace USB',
          vidPlaceholder: 'např. 0x046D',
          pidPlaceholder: 'např. 0x0A4E',
          bcdDevicePlaceholder: 'např. 0x0521 (volitelné)',
          serialnumberPlaceholder: 'např. 2214GG01KWF8 (volitelné)',
          manufacturerPlaceholder: 'např. Logitech (volitelné)',
          productPlaceholder: 'např. Logitech P710e (volitelné)',
          notEmpty: 'VID a PID nesmí být prázdné.',
          notValid: 'Neplatný formát VID/PID. Použijte hex, např. 0x046D.',
          bcdNotValid:
            'Neplatný formát bcdDevice. Použijte hex, např. 0x0521, nebo nechte prázdné.',
          failed: 'Konfigurace selhala.',
          success: 'Konfigurace dokončena. USB gadget restartován.',
          okBtn: 'OK',
          cancelBtn: 'Zrušit'
        },
        diskInquiry: {
          title: 'Název USB disku',
          config: 'Konfigurace',
          tip: 'Název virtuálního flash disku, který vidí vzdálený počítač. Uložení restartuje USB a může krátce odpojit klávesnici, myš nebo síť.',
          enable: 'Použít vlastní název USB disku',
          placeholder: 'např. LogitechFlash Drive     1.00',
          notEmpty: 'Název nesmí být prázdný.',
          notValid: 'Neplatný název. Nejvýše 28 tisknutelných znaků ASCII bez zalomení řádků.',
          failed: 'Konfigurace selhala.',
          success: 'Konfigurace dokončena. USB gadget restartován.',
          okBtn: 'OK',
          cancelBtn: 'Zrušit'
        },
        diskInquiryCdrom: {
          title: 'Název virtuálního CD',
          config: 'Konfigurace',
          tip: 'Název virtuální CD/DVD mechaniky na vzdáleném počítači. Uložení restartuje USB a může krátce odpojit klávesnici, myš nebo síť.',
          enable: 'Použít vlastní název virtuálního CD',
          placeholder: 'např. LogitechOptical Drive   1.00',
          notEmpty: 'Název nesmí být prázdný.',
          notValid: 'Neplatný název. Nejvýše 28 tisknutelných znaků ASCII bez zalomení řádků.',
          failed: 'Konfigurace selhala.',
          success: 'Konfigurace dokončena. USB gadget restartován.',
          okBtn: 'OK',
          cancelBtn: 'Zrušit'
        },
        uac2FunctionName: {
          title: 'Název USB audia',
          config: 'Konfigurace',
          tip: 'Název virtuálního USB mikrofonu / audio zařízení na vzdáleném počítači. Uložení restartuje USB a může krátce odpojit klávesnici, myš nebo síť.',
          enable: 'Použít vlastní název USB audia',
          placeholder: 'např. Logitech P710e',
          notEmpty: 'Název nesmí být prázdný.',
          notValid: 'Neplatný název. Nejvýše 64 tisknutelných znaků bez zalomení řádků.',
          failed: 'Konfigurace selhala.',
          success: 'Konfigurace dokončena. USB gadget restartován.',
          okBtn: 'OK',
          cancelBtn: 'Zrušit'
        },
        lowPower: {
          title: 'Režim nízké spotřeby',
          description: 'Snižte spotřebu energie při nečinnosti',
          tips: 'Tipy',
          tip1: 'Pokud je tato možnost povolena, zachytávání videa se během nečinnosti pozastaví, aby se snížila spotřeba energie a zahřívání.',
          tip2: 'Obnovení zachytávání videa může vyžadovat opětovnou inicializaci, během níž může být video dočasně nedostupné.',
          okBtn: 'Dobře',
          cancelBtn: 'Zrušit'
        },
        webcamUsbDisplay: 'Před zapnutím webkamery vypněte režim USB displeje.',
        webcamEndpoints: 'Webkamera vyžaduje dva volné vstupní koncové body USB.',
        mouseJiggler: {
          noMouse: 'V tomto režimu nelze simulovat pohyb myši. Vyberte režim myši nebo vypněte režim jednoho koncového bodu.',
          title: 'Pohyb myši (Jiggler)',
          description: 'Zabraňte spánku vzdáleného hostitele',
          disable: 'Zakázat',
          absolute: 'Absolutní režim',
          relative: 'Relativní režim'
        },
        fingerRobot: {
          title: 'Prstový robot',
          description: 'Povolit ovládání prstového robota'
        },
        mdns: {
          description: 'Povolit službu zjišťování mDNS'
        },
        hidOnly: 'HID-Pouze režim',
        imageMounted: 'Obrázek připojen',
        disk: 'Virtuální disk',
        diskDesc: 'Připojit virtuální U-disk ke vzdálenému hostiteli',
        sdCard: {
          warning: 'Inicializovat virtuální disk',
          tip1: 'Při prvním povolení se virtuální disk inicializuje a stávající soubory obrazů i data virtuálního disku v zařízení budou smazány. Tuto akci nelze vrátit zpět. Nejprve zazálohujte důležité soubory.',
          tip2: 'Pokračovat v inicializaci?',
          capacity: 'Kapacita',
          disable: 'Zakázat',
          disableWarning: 'Zakázat virtuální disk?',
          disableTip:
            'Zakázáním se smažou všechny soubory na virtuálním disku a uvolní se úložiště zařízení, které používá. Při příštím povolení bude nutné znovu vybrat kapacitu a disk inicializovat. Tuto akci nelze vrátit zpět.'
        },
        network: 'Virtuální síť',
        networkDesc: 'Připojit virtuální síťovou kartu ke vzdálenému hostiteli',
        mic: 'Virtuální zvuk',
        micDesc: 'Vyberte režim virtuálního zvuku USB pro vzdáleného hostitele',
        virtualAudioMode: {
          off: 'Vypnuto',
          mic: 'Pouze mikrofon',
          duplex: 'Mikrofon + reproduktor'
        },
        speaker: 'Virtuální reproduktor',
        speakerDesc:
          'Po zapnutí nahradí výchozí zvukový výstup DP pro podporu vzdálených hovorů s lepší kompatibilitou. Po vypnutí lze přenášet pouze zvuk médií, nikoli zvuk hovorů.',
        micTip:
          'Kvůli omezením koncových bodů USB deaktivuje povolení virtuálního mikrofonu sekundární obrazovku USB.\nA je doporučeno tuto funkci zakázat během operací BIOS, aby byla zajištěna optimální kompatibilita.',
        reboot: 'Restartujte',
        rebootDesc: 'Opravdu chcete restartovat NanoKVM?',
        rebootToFlashing: 'Restartovat do režimu flashování',
        rebootToFlashingButton: 'Restartovat',
        rebootToFlashingTitle: 'Restartovat do režimu flashování?',
        rebootToFlashingDesc:
          'NanoKVM se restartuje a přejde do režimu flashování přes USB. Ujistěte se, že je připojeno USB rozhraní pro flashování a že je připraven flashovací nástroj.',
        datetime: {
          timezone: 'Časové pásmo',
          datetime: 'Datum a čas',
          dateAndTime: 'Datum a čas',
          synchronize: 'Synchronizovat čas',
          lastSynchronization: 'Poslední synchronizace: ',
          notSynchronized: 'Systémový čas ještě není synchronizován',
          syncNow: 'Synchronizovat nyní',
          manualSet: 'Nastavit ručně',
          manualSetTitle: 'Ruční nastavení času',
          manualSetDescription: 'Po změně času se budete muset znovu přihlásit.',
          useBrowserTime: 'Použít čas prohlížeče',
          setTime: 'Nastavit čas',
          invalidTime: 'Zadejte platné datum a čas.',
          setFailed: 'Systémový čas nelze nastavit. Zkuste to znovu.',
          setSucceeded: 'Systémový čas byl aktualizován. Přihlaste se znovu.'
        },
        dialBackground: {
          errors: {
            load: 'Nelze načíst obrázky pozadí ciferníku. Zkuste to za chvíli znovu.',
            upload:
              'Nelze nahrát obrázek pozadí. Zkontrolujte, zda je soubor platný obrázek BMP, a zkuste to znovu.',
            switch: 'Nelze změnit pozadí ciferníku. Obnovte seznam obrázků a zkuste to znovu.'
          },
          title: 'Pozadí ciferníku',
          description: 'Vyberte obrázek BMP z /kvmcomm/ui/bg/ jako pozadí ciferníku',
          upload: 'Nahrát',
          loading: 'Načítání...',
          empty: 'Nenalezeny žádné obrázky BMP',
          current: 'Aktuální',
          onlyBmp: 'Podporovány jsou pouze obrázky BMP',
          uploaded: 'Nahrávání dokončeno',
          switched: 'Pozadí změněno'
        },
        okBtn: 'Ano',
        cancelBtn: 'Ne'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Restartovat Tailscale?',
        stop: 'Zastavit Tailscale?',
        stopDesc: 'Odhlásit Tailscale a zakázat automatické spouštění při startu.',
        loading: 'Načítání...',
        notInstall: 'Tailscale nebyl nalezen! Prosím nainstalujte.',
        install: 'Nainstalovat',
        installing: 'Instalace probíhá',
        failed: 'Instalace se nezdařila',
        retry: 'Obnovte stránku a zkuste to znovu. Nebo zkuste instalaci manuálně',
        download: 'Stáhnout',
        package: 'instalační balíček',
        unzip: 'a rozbalit ho',
        upTailscale: 'Nahrajte Tailscale do adresáře NanoKVM /usr/bin/',
        upTailscaled: 'Nahrajte Tailscaled do adresáře NanoKVM /usr/sbin/',
        refresh: 'Obnovit stránku',
        notRunning: 'Tailscale neběží. Chcete-li pokračovat, spusťte jej.',
        run: 'Spustit',
        notLogin:
          'Zařízení nebylo dosud spárováno. Přihlaste se prosím a spárujte toto zařízení s vaším účtem.',
        urlPeriod: 'Tento odkaz je platný po dobu 10 minut',
        login: 'Přihlášení',
        loginSuccess: 'Přihlášení úspěšné',
        enable: 'Povolit Tailscale',
        exitNode: 'Výstupní uzel',
        exitNodeDesc: 'Inzerovat toto zařízení jako internetový výstup pro váš tailnet.',
        subnetRoute: 'Trasa podsítě',
        subnetRouteDesc: 'Inzerovat přímo připojenou síť LAN {{subnet}} ve vašem tailnetu.',
        subnetUnavailable: 'Není k dispozici žádná přímo připojená síť LAN.',
        routingUnavailable: 'Nastavení směrování Tailscale nejsou na tomto zařízení dostupná.',
        pendingApproval: 'Po povolení schvalte inzerovanou trasu v konzoli pro správu Tailscale.',
        routingUpdateFailed: 'Nastavení směrování Tailscale se nepodařilo aktualizovat.',
        deviceName: 'Název zařízení',
        deviceIP: 'IP zařízení',
        account: 'Účet',
        logout: 'Odhlásit se',
        logoutDesc: 'Opravdu se chcete odhlásit?',
        uninstall: 'Odinstalovat Tailscale',
        uninstallDesc: 'Opravdu chcete odinstalovat Tailscale?',
        okBtn: 'Ano',
        cancelBtn: 'Ne'
      },
      update: {
        title: 'Zkontrolovat aktualizaci',
        queryFailed: 'Nepodařilo se získat verzi',
        updateFailed: 'Aktualizace se nezdařila. Zkuste to prosím znovu.',
        isLatest: 'Máte nejnovější verzi.',
        available: 'Je dostupná aktualizace. Opravdu chcete aktualizovat?',
        updating: 'Aktualizace zahájena. Prosím čekejte...',
        confirm: 'Potvrdit',
        upgrade_tip_nano:
          'K dispozici je nová verze NanoKVM. Opravdu chcete provést aktualizaci nyní?',
        upgrade_tip_firmware:
          'K dispozici je nová verze firmwaru. Opravdu chcete provést aktualizaci nyní?',
        cancel: 'Zrušit',
        preview: 'Náhled aktualizací',
        previewDesc: 'Získejte včasný přístup k novým funkcím a vylepšením',
        previewTip:
          'Uvědomte si prosím, že předběžné verze mohou obsahovat chyby nebo neúplné funkce!',
        download: 'Stáhnout',
        install: 'Instalovat',
        changelog: 'Seznam změn',
        restart: 'Restartujte'
      },
      account: {
        title: 'Účet',
        webAccount: 'Název webového účtu',
        password: 'Heslo',
        twoFactor: 'Dvoufaktorové ověřování',
        enableTwoFactor: 'Povolit',
        enableTwoFactorDesc: 'Zadejte aktuální heslo a začněte nastavovat dvoufaktorové ověřování.',
        disableTwoFactor: 'Zakázat',
        recoveryCodes: 'Obnovovací kódy',
        enableRecoveryCodes: 'Povolit',
        updateRecoveryCodes: 'Aktualizovat',
        recoveryCodesTooltipDisabled:
          'Obnovovací kódy jsou záložní přihlašovací údaje pro dvoufaktorové ověřování. Můžete je vygenerovat po zapnutí dvoufaktorového ověřování.',
        recoveryCodesTooltipEnabled:
          'Obnovovací kódy platí pouze při zapnutém dvoufaktorovém ověřování. Vypnutí dvoufaktorového ověřování zneplatní všechny obnovovací kódy; dokud zůstane zapnuté, lze je pouze aktualizovat.',
        recoveryCodesRegenerateDesc:
          'Zadejte aktuální heslo a buď ověřovací kód, nebo nepoužitý obnovovací kód. Aktualizace obnovovacích kódů zneplatní všechny stávající obnovovací kódy.',
        recoveryCodesVerifyFailed:
          'Nepodařilo se ověřit heslo a ověřovací kód nebo obnovovací kód.',
        recoveryCodesSavedTitle: 'Uložte si tyto obnovovací kódy nyní',
        recoveryCodesSavedDesc:
          'Obnovovací kódy lze zobrazit pouze jednou. Před opuštěním této stránky je uložte na bezpečné místo.',
        enableRecoveryCodesDesc:
          'Zadejte aktuální heslo a ověřovací kód pro vygenerování obnovovacích kódů. Obnovovací kódy lze zobrazit pouze jednou, proto je před opuštěním této stránky uložte.',
        copyRecoveryCodes: 'Kopírovat',
        downloadRecoveryCodes: 'Stáhnout',
        recoveryCodesCopyFailed: 'Nepodařilo se zkopírovat obnovovací kódy.',
        done: 'Hotovo',
        currentPassword: 'Aktuální heslo',
        authCode: 'Ověřovací kód',
        authCodeOrRecoveryCode: 'Ověřovací kód nebo obnovovací kód',
        authRateLimited: 'Příliš mnoho neúspěšných pokusů. Zkuste to znovu za {{seconds}} s.',
        verifyPasswordChange: 'Ověřit změnu hesla',
        verifyPasswordChangeDesc:
          'Před změnou hesla zadejte aktuální heslo a buď ověřovací kód, nebo nepoužitý obnovovací kód.',
        verifyPasswordChangeFailed:
          'Nepodařilo se ověřit heslo a ověřovací kód nebo obnovovací kód.',
        verifyAndEnable: 'Ověřit a povolit',
        scanTwoFactorQRCode: 'Naskenujte tento QR kód v ověřovací aplikaci a poté zadejte kód.',
        showSetupKey: 'Zobrazit nastavovací klíč',
        hideSetupKey: 'Skrýt nastavovací klíč',
        setupKey: 'Nastavovací klíč',
        twoFactorSetupFailed: 'Nepodařilo se zahájit nastavení dvoufaktorového ověřování.',
        twoFactorVerifyFailed: 'Nepodařilo se ověřit ověřovací kód.',
        twoFactorDisableFailed: 'Nepodařilo se zakázat dvoufaktorové ověřování.',
        disableTwoFactorDesc:
          'Zadejte aktuální heslo a buď ověřovací kód, nebo nepoužitý obnovovací kód. Vypnutí dvoufaktorového ověřování zneplatní všechny obnovovací kódy.',
        updateBtn: 'Změnit',
        logoutBtn: 'Odhlásit se',
        logoutDesc: 'Opravdu se chcete odhlásit?',
        okBtn: 'Ano',
        cancelBtn: 'Ne',
        twoFactorProtection: {
          title: 'Pokročilá ochrana 2FA',
          description:
            'Vyberte citlivé akce, které budou při vynucení ochrany vyžadovat druhý faktor. Přepínač „Přímé webové přihlášení“ funguje opačně: umožní přihlášení pouze pomocí ověřovacího kódu nebo nepoužitého obnovovacího kódu.',
          login: 'Přímé webové přihlášení',
          webTerminal: 'Webový terminál',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Klíč MCP API',
          targetPower: 'Napájení cílového počítače',
          nanoKVMRestart: 'Restart NanoKVM',
          deleteImage: 'Smazat image',
          scripts: 'Operace se skripty',
          screenTimelapse: 'Časosběr obrazovky',
          loginScope:
            'Umožňuje přihlášení do webového rozhraní pouze pomocí ověřovacího nebo nepoužitého obnovovacího kódu místo uživatelského jména a hesla. Snižuje bezpečnost přihlášení.',
          webTerminalScope: 'Otevření webového terminálu.',
          sshScope: 'Změna stavu služby SSH.',
          tailscaleScope: 'Instalace, připojení nebo změna Tailscale.',
          frp: 'FRP',
          frpScope: 'Instalace, připojení nebo změna FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Instalace, připojení nebo změna Cloudflare Tunnel.',
          zerotierScope: 'Instalace, připojení nebo změna ZeroTier.',
          netbirdScope: 'Instalace, připojení nebo změna NetBird.',
          mcpScope:
            'Zobrazení, kopírování či opětovné vygenerování klíče MCP API a zapnutí či vypnutí MCP.',
          targetPowerScope: 'Vypnutí nebo restart cílového počítače fyzickým tlačítkem napájení.',
          nanoKVMRestartScope: 'Restart samotného NanoKVM.',
          deleteImageScope: 'Trvale smazat uložený ISO nebo IMG image.',
          scriptsScope: 'Spuštění, nahrání nebo odstranění skriptů.',
          screenTimelapseScope: 'Zapnutí záznamu časosběru obrazovky, procházení a mazání záznamů.',
          authorizeTitle: 'Autorizovat změny ochrany',
          authorizeDesc:
            'Zadejte ověřovací kód nebo nepoužitý obnovovací kód. Autorizace trvá pět minut.',
          authorizeFailed: 'Ověřovací kód nebo obnovovací kód se nepodařilo ověřit.',
          loadFailed: 'Nelze načíst nastavení ochrany.',
          updateFailed: 'Nelze uložit nastavení ochrany.',
          retry: 'Zkusit znovu',
          disableScriptsWarningTitle: 'Vypnout ochranu skriptů?',
          disableScriptsWarningDesc:
            'Bez ochrany může škodlivý skript získat zvýšený přístup k tomuto zařízení a obejít nebo odstranit ochranu 2FA.',
          cancelDisabling: 'Zrušit',
          continueDisabling: 'Pokračovat ve vypnutí',
          continueDisablingCountdown: 'Pokračovat ve vypnutí ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Průvodce dvoufaktorovým ověřováním',
          scope: 'Dvoufaktorové ověřování přidává k vašemu účtu další vrstvu zabezpečení.',
          whatTitle: 'Co je 2FA?',
          whatDesc:
            'Dvoufaktorové ověřování (2FA) chrání váš účet pomocí časově omezeného kódu z ověřovací aplikace. Ve výchozím nastavení je tento kód vyžadován kromě hesla, takže i když je vaše heslo kompromitováno, účet zůstává chráněný. Pokud je zapnuté přímé webové přihlášení, přihlásíte se pouze pomocí kódu.',
          appsTitle: 'Doporučené aplikace',
          appsDesc:
            'Můžete použít Google Authenticator nebo Microsoft Authenticator. Obě aplikace jsou zdarma a fungují na iOS i Androidu.',
          setupTitle: 'Jak nastavit',
          setupDesc:
            'Otevřete ověřovací aplikaci, naskenujte QR kód zobrazený během nastavení nebo ručně zadejte nastavovací klíč. Poté zadejte 6místný kód z aplikace a dokončete povolení 2FA.',
          recoveryTitle: 'Obnovovací kódy',
          recoveryDesc:
            'Obnovovací kódy jsou jednorázové záložní kódy. Pokud ztratíte přístup k ověřovací aplikaci, můžete se přihlásit pomocí obnovovacího kódu. Ihned po vytvoření je uložte na bezpečné místo.',
          logoutNotice:
            'Povolení 2FA odhlásí všechny aktivní relace, včetně této. Na každém zařízení se budete muset znovu přihlásit.'
        }
      }
    },
    error: {
      title: 'Narazili jsme na problém',
      refresh: 'Obnovit'
    },
    pictureInPicture: {
      title: 'Obraz v obraze',
      enter: 'Zapnout obraz v obraze',
      exit: 'Vypnout obraz v obraze',
      error: 'Nelze spustit obraz v obraze. Počkejte, až se video načte, a zkuste to znovu.'
    },
    fullscreen: {
      toggle: 'Přepnout na celou obrazovku'
    },
    menu: {
      collapse: 'Sbalit nabídku',
      expand: 'Rozbalte nabídku'
    }
  }
};

export default cz;
