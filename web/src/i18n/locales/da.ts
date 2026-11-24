const da = {
  translation: {
    storage: {
      cards: { mount: 'Diskmontering', share: 'Fildeling' },
      title: 'Diskbilleder og fildeling',
      empty: 'Ingen billeder endnu',
      notAttached: 'Intet monteret',
      attached: 'Monteret: {{name}}',
      eject: 'Skub ud',
      mount: 'Montér',
      mountMode: 'Monteringstilstand',
      mountModeDisk: 'Masselager',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Åbn filhåndtering',
      refresh: 'Opdater',
      delete: 'Slet',
      deleteTitle: 'Slet medie',
      deleteConfirm: 'Vil du slette {{name}}? Filerne i den går tabt.',
      downloadImage: 'Download diskbilledet',
      typePartition: 'Delte filer',
      volumeLabel: 'Flytbar disk',
      groups: {
        share: 'Fildeling',
        images: 'Diskbilleder'
      },
      failed: 'Handlingen mislykkedes',
      hidOnly: 'Kun HID-tilstand: den virtuelle disk er ikke tilgængelig',
      disabled: 'Den virtuelle disk er slået fra (Indstillinger - Enhed)',
      force: {
        title: 'Tag mediet tilbage med magt?',
        desc: 'Målcomputeren bruger stadig dette medie. At tage det tilbage opregner USB-enheden igen og kan miste alt, hvad den endnu ikke har skrevet til disken.',
        menuHint:
          'Målcomputeren bruger dette medie. Åbn filhåndteringen for at tage det tilbage med magt.'
      },
      tips: {
        title: 'Sådan uploader du',
        usb1: 'Forbind din NanoKVM til din computer via USB.',
        usb2: 'Montér de delte filer i masselagringstilstand på denne side.',
        usb3: 'Åben den virtuelle disk på din computer og kopier diskbilledet til roden af den virtuelle disk.',
        usb4: 'Skub disken ud, når kopieringen er færdig: billedet vises derefter på listen.',
        scp1: 'Kontroller at din NanoKVM og din computer er på samme lokale netværk.',
        scp2: 'Åben en terminal på din computer og brug SCP-kommandoen for at uploade diskbilledet til /data mappen på din NanoKVM.',
        scp3: 'Eksempel: scp sti-til-dit-diskbillede root@din-nanokvm-ip:/data'
      },
      job: {
        running: 'Arbejder på {{name}}'
      },
      enableDisk: {
        title: 'Slå den virtuelle disk til?',
        desc: 'For at montere et medie kræves den virtuelle disk, som er slået fra lige nu. Slå den til? Målcomputerens tastatur og mus afbrydes et øjeblik.',
        menuHint: 'Den virtuelle disk er slået fra. Slå den til under Indstillinger - Enhed.'
      },
      add: {
        title: 'Tilføj medie',
        upload: 'Upload en ISO eller IMG',
        fromUrl: 'Download fra en URL',
        url: 'URL til diskbilledet',
        urlRequired: 'Indtast en URL',
        onlyImages: 'Kun .iso- og .img-filer kan monteres',
        dropHint: 'Træk en ISO eller IMG hertil, eller klik for at vælge en fil',
        cdromOnlyIso: 'En CD-ROM kan kun indeholde et ISO 9660 / UDF-diskbillede',
        uploadFailed: 'Upload mislykkedes',
        downloadFailed: 'Download mislykkedes',
        uploading: 'Uploader {{name}}',
        downloading: 'Downloader {{name}}',
        done: '{{name}} er klar',
        checksumFailed: 'Billedet stemmer ikke overens med SHA-256-kontrolsummen.',
        blocked:
          'Målcomputeren bruger disken. Skub det virtuelle USB-drev sikkert ud på målcomputeren, og slå derefter kontakten Virtuel disk fra under Indstillinger > Enhed.'
      },
      files: {
        reasonNotInitialized:
          'Det virtuelle USB-drev er ikke initialiseret. Initialiser det under Indstillinger > Enhed, før du bruger fildeling.',
        selectMedium: 'Vælg et medie for at se filerne på det.',
        reasonAttached: 'Dette medie er koblet til målcomputeren.',
        reasonAttachedHint:
          'Skub det virtuelle USB-drev sikkert ud på målcomputeren, og slå derefter kontakten Virtuel disk fra under Indstillinger > Enhed.',
        reasonNotBrowsable:
          'Dette medie indeholder et filsystem, som filhåndteringen ikke kan åbne.',
        reasonNoLoop: 'Denne enhed kan ikke montere diskbilleder (ingen loop-understøttelse).',
        reasonISOImage:
          'Et ISO 9660-diskbillede kan ikke gennemses her. Det kan stadig monteres på målcomputeren.',
        empty: 'Denne mappe er tom',
        noMatch: 'Ingen filer matcher søgningen.',
        free: '{{size}} ledige',
        search: 'Søg i denne mappe',
        sort: {
          modified: 'Ændret',
          name: 'Navn',
          size: 'Størrelse',
          type: 'Type',
          asc: 'Stigende',
          desc: 'Faldende'
        },
        view: {
          list: 'Listevisning',
          grid: 'Ikonvisning'
        },
        folder: 'Mappe',
        itemCount: 'Antal elementer: {{count}}',
        itemCount_one: '{{count}} element',
        itemCount_other: '{{count}} elementer',
        dropHint: 'Slip filer her for at uploade dem til denne mappe',
        up: 'Gå til den overordnede mappe',
        uploadButton: 'Upload filer',
        newFolder: 'Ny mappe',
        newFolderPlaceholder: 'Mappenavn',
        create: 'Opret',
        upload: 'Upload en fil til mediet',
        deleteFolderTitle: 'Slet mappe',
        deleteFolderConfirm: 'Slet mappen {{name}} og alt dens indhold? Dette kan ikke fortrydes.',
        uploading: 'Uploader…',
        uploadFailed: 'Upload mislykkedes',
        download: 'Download',
        delete: 'Slet',
        deleteTitle: 'Slet fil',
        deleteConfirm: 'Vil du slette {{name}} fra mediet?'
      },
      yes: 'Ja',
      no: 'Nej'
    },
    head: {
      storage: 'Diskbilleder og fildeling',
      desktop: 'Fjernskrivebord',
      login: 'Log ind',
      changePassword: 'Skift adgangskode',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Kunne ikke kopiere til udklipsholderen. Giv browseren adgang til udklipsholderen, og prøv igen.'
    },
    auth: {
      login: 'Log ind',
      language: 'Sprog',
      placeholderUsername: 'Indtast brugernavn',
      placeholderPassword: 'indtast adgangskode',
      placeholderPassword2: 'indtast adgangskode igen',
      noEmptyUsername: 'brugernavn kan ikke være tom',
      noEmptyPassword: 'adgangskode kan ikke være tom',
      noAccount:
        'Kunne ikke hente brugeroplysninger. Prøv at opdater siden eller nulstil adgangskoden',
      invalidUser: 'ugyldigt brugernavn eller adgangskode',
      error: 'uventet fejl',
      twoFactorTitle: 'Verifikation kræves',
      twoFactorDesc:
        'Indtast din godkendelseskode eller ubrugte gendannelseskode for at logge ind.',
      sensitiveActionDesc:
        'Indtast en godkendelseskode eller ubrugt gendannelseskode for at fortsætte.',
      placeholderAuthCode: 'Godkendelseskode',
      noEmptyAuthCode: 'Godkendelseskode er påkrævet',
      verify: 'Verificer',
      backToLogin: 'Tilbage til login',
      verificationFailed: 'Verifikation mislykkedes',
      twoFactorEnabledLogout: {
        message:
          'Tofaktorgodkendelse er blevet aktiveret. Af sikkerhedshensyn er alle sessioner logget ud. Log ind igen.'
      },
      rateLimited: 'For mange mislykkede forsøg. Prøv igen om {{seconds}} s.',
      recoveryCodeUsed:
        'Gendannelseskoden, der blev brugt til dette login, er blevet ugyldiggjort.',
      changePassword: 'Skift adgangskode',
      changePasswordDesc: 'For sikkerheden af din enhed, bedes du ændre web-login adgangskoden.',
      passwordVerificationRequired:
        'Gå tilbage til kontosiden i indstillingerne, og verificer denne adgangskodeændring igen.',
      passwordVerificationExpired:
        'Verifikationen af adgangskodeændringen er udløbet. Gå tilbage til kontosiden i indstillingerne, og verificer igen.',
      returnToSettings: 'Tilbage til indstillinger',
      differentPassword: 'Adgangskoder er ikke ens',
      illegalUsername: 'brugernavn indeholder ugyldige tegn',
      illegalPassword: 'adgangskode indeholder ugyldige tegn',
      forgetPassword: 'Glem adgangskode',
      ok: 'OK',
      cancel: 'Annuller',
      savePassword: 'Gem adgangskode',
      loginButtonText: 'Log ind',
      tips: {
        reset1: 'Webstandardkonto:',
        reset2: 'SSH standardkonto:',
        reset3: 'For at nulstille adgangskoden, se venligst ',
        changeReset: 'For at nulstille adgangskoden til weblogin, se ',
        resetLink: 'Nulstil'
      }
    },
    wifi: {
      errors: {
        connect:
          'Kunne ikke oprette forbindelse til Wi-Fi. Kontrollér netværksnavn, adgangskode og enhedens Wi-Fi-status, og prøv igen.'
      },
      title: 'Wi-Fi',
      description: 'Konfigurer Wi-Fi for NanoKVM',
      submitted: {
        title: 'Et øjeblik — NanoKVM opretter forbindelse til dit Wi-Fi',
        tipsTitle: 'Hvad nu?',
        tipSwitch: 'Du kan nu skifte telefonen tilbage til dit sædvanlige Wi-Fi.',
        tipCheck:
          'Se om lidt efter den nye IP-adresse på NanoKVM-skærmen, og åbn den i din browser.',
        tipRetry:
          'Hvis der ikke sker noget efter et stykke tid, så tjek netværksnavn og adgangskode, og prøv igen.'
      },
      invalidMode:
        'Den aktuelle tilstand understøtter ikke netværksopsætning. Gå til din enhed og aktiver Wi-Fi konfigurationstilstand.',
      confirmBtn: 'OK',
      finishBtn: 'Udført',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Adgangskode',
      steps: {
        settings: 'Indstillinger',
        wifi: 'Wi-Fi',
        enable: 'Aktiver'
      },
      scan: {
        title: 'Vælg et Wi-Fi-netværk',
        description: 'Vælg et netværk i nærheden for at fortsætte.',
        loading: 'Søger efter Wi-Fi-netværk i nærheden…',
        empty: 'Ingen Wi-Fi-netværk i nærheden blev fundet.',
        failed: 'Kunne ikke scanne. Indtast netværksoplysninger manuelt.',
        retryBtn: 'Scan igen',
        manualBtn: 'Indtast netværk manuelt',
        backBtn: 'Tilbage til netværk i nærheden',
        connectTitle: 'Opret forbindelse til {{ssid}}',
        connecting: 'Opretter forbindelse…',
        viewSimple: 'Skift til enkel visning',
        viewDetailed: 'Skift til detaljeret visning'
      },
      ap: {
        authTitle: 'Godkendelse påkrævet',
        authDescription: 'Indtast venligst AP adgangskoden for at fortsætte',
        authFailed: 'Ugyldig AP adgangskode',
        passPlaceholder: 'AP adgangskode',
        verifyBtn: 'Bekræft'
      },
      qr: {
        cameraBtn: 'Tag foto',
        pickBtn: 'Vælg skærmbillede',
        scanHint:
          'Tag et foto af en Wi-Fi QR-kode, eller vælg et skærmbillede af en, f.eks. den din telefon viser, når den deler sin Wi-Fi.',
        decoding: 'Læser QR-kode…',
        notFound:
          'Der blev ikke fundet nogen QR-kode i billedet. Sørg for, at koden er centreret og i fokus, og prøv igen.',
        invalid: 'Denne QR-kode indeholder ikke brugbare Wi-Fi-oplysninger.',
        pickerUnavailable:
          'Åbnede intet? Dette login-vindue kan blokere kameraet og billeder. Åbn i stedet denne adresse i telefonens browser:',
        confirmTitle: 'Opret forbindelse mellem NanoKVM og {{ssid}}?',
        confirmDescription:
          'Disse netværksoplysninger stammer fra en QR-kode. Kontrollér dem, før du opretter forbindelse.',
        openNetwork: 'Dette netværk har ingen adgangskode.',
        connectBtn: 'Opret forbindelse',
        otherBtn: 'Vælg et andet netværk'
      }
    },
    notification: {
      browser: {
        title: 'Browseranbefaling',
        description:
          'For den bedste brugeroplevelse anbefaler vi at bruge Chrome-browseren. Selvom vi stræber efter kompatibilitet på tværs af alle større browsere, fungerer nogle avancerede funktioner muligvis ikke efter hensigten uden for Chrome.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Skærm',
      video: 'Videotilstand',
      quality: 'Kvalitet',
      qualityAuto: 'Automatisk',
      qualityLossless: 'Tabsfri',
      qualityHigh: 'Høj',
      qualityMedium: 'Mellem',
      qualityLow: 'Lav',
      scale: 'Skala',
      noSignal: 'Intet videosignal',
      captureDisabled:
        'Videooptagelse er deaktiveret. Aktivér den under "Indstillinger > Enhed > Videooptagelse".',
      inconsistentVideoMode: 'Afspilning sat på pause. En anden videotilstand afspilles.',
      bandwidthWarning:
        'Netværksbåndbredden er muligvis utilstrækkelig til de aktuelle videoindstillinger. Prøv at sænke bithastigheden eller billedhastigheden.',
      webrtcConnectionFailed:
        'WebRTC-forbindelsen kunne ikke oprettes. Skift til Direct-videotilstand.',
      webrtcUnavailable: {
        title: 'WebRTC er ikke tilgængeligt',
        description: 'Aktivér WebRTC i browseren, eller skift til Direct-videotilstand.'
      },
      upload: 'Upload',
      advanced: 'Avanceret',
      loading: 'Indlæser'
    },
    volume: {
      title: 'Lydstyrke'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon ikke monteret',
      tip1: 'Den virtuelle mikrofon er ikke monteret på fjernværten. Vil du montere og aktivere det nu?',
      tip2: 'Du kan også administrere dette senere i "Indstillinger > Enhed > Virtuel lyd".',
      confirm: 'Aktiver',
      cancel: 'Annuller',
      noDevice:
        'Der blev ikke fundet nogen mikrofon på denne enhed. Tilslut en mikrofon, og prøv igen.',
      permissionDenied:
        'Adgang til mikrofonen blev afvist. Tillad mikrofonadgang i browserens indstillinger.',
      notReadable: 'Mikrofonen bruges af et andet program eller kan ikke læses.',
      overconstrained: 'Mikrofonen understøtter ikke de krævede lydindstillinger.',
      securityBlocked:
        'Adgang til mikrofonen er blokeret på denne side. Kontrollér browserens sikkerhedsindstillinger.',
      aborted: 'Start af mikrofonen blev afbrudt. Prøv igen.',
      unsupported: 'Denne browser eller forbindelsesmetode understøtter ikke mikrofonoptagelse.',
      enableFailed: 'Mikrofonen kunne ikke aktiveres. Prøv igen.',
      disconnected: 'Mikrofonen blev afbrudt, så mikrofonoverførslen er blevet deaktiveret.'
    },
    power: {
      title: 'Tænd/sluk-knap',
      showConfirm: 'Bekræftelse',
      showConfirmTip: 'Strømhandlinger kræver en ekstra bekræftelse',
      powerShort: 'Tænd/sluk-knap (kort tryk)',
      powerLong: 'Tænd/sluk-knap (langt tryk)',
      powerConfirm: 'Udfør strømhandling?',
      okBtn: 'Ja',
      cancelBtn: 'Nej'
    },
    keyboard: {
      errors: {
        paste:
          'Kunne ikke sende teksten til fjernenheden. Kontrollér KVM-forbindelsen, og prøv igen.'
      },
      title: 'Tastatur',
      paste: 'Indsæt',
      tips: 'Kun standard bogstaver og symboler er understøttet',
      placeholder: 'Indtast tekst',
      submit: 'Send',
      virtual: 'Tastatur',
      swapCommandCtrl: 'Byt Cmd og Ctrl',
      shortcut: {
        title: 'Genveje',
        custom: 'Brugerdefineret',
        capture: 'Klik her for at fange genvej',
        clear: 'Ryd',
        save: 'Gem',
        captureTips:
          'Optagelse af systemtaster (såsom Windows-tasten) kræver fuldskærmstilladelse.',
        enterFullScreen: 'Skift fuldskærmstilstand.'
      },
      leaderKey: {
        title: 'Leader-tast',
        desc: 'Omgå browserbegrænsninger og send systemgenveje direkte til fjernværten.',
        howToUse: 'Sådan bruges',
        simultaneous: {
          title: 'Samtidig tilstand',
          desc1: 'Hold Leader-tasten nede, og tryk derefter på genvejen.',
          desc2: 'Intuitivt, men kan være i konflikt med systemgenveje.'
        },
        sequential: {
          title: 'Sekventiel tilstand',
          desc1:
            'Tryk på Leader-tasten → tryk på genvejen i rækkefølge → tryk på Leader-tasten igen.',
          desc2: 'Kræver flere trin, men undgår fuldstændig systemkonflikter.'
        },
        enable: 'Aktiver Leader-tast',
        tip: 'Når denne tast tildeles som Leader-tast, fungerer den kun som genvejsudløser og mister sin standardfunktion.',
        placeholder: 'Tryk på Leader-tasten',
        shiftRight: 'Højre Shift',
        ctrlRight: 'Højre Ctrl',
        metaRight: 'Højre Win',
        submit: 'Send',
        recorder: {
          rec: 'REC',
          activate: 'Aktiver taster',
          input: 'Tryk på genvejen...'
        }
      }
    },
    mouse: {
      title: 'Mus',
      mode: 'Tilstand for mus',
      absolute: 'Følgende mus',
      absoluteAndroid: 'Følgende mus (bedre Android-kompatibilitet)',
      touchScreen: 'Multitouchskærm',
      relative: 'Eksklusiv mus',
      direction: 'Rullehjulsretning',
      scrollUp: 'Rul op',
      scrollDown: 'Rul ned',
      speed: 'Rullehjulshastighed',
      fast: 'Hurtigt',
      slow: 'Langsomt',
      requestPointer: 'Bruger relativ-tilstand. Klik på skrivebordet for at få musemarkør.',
      repairIphoneDrag: 'Reparer iPhone-træk',
      resetHid: 'Nulstil HID',
      touchpadGuide: {
        title: 'Touchpad-guide',
        scope: 'Gælder for almindelig markørinput, når musetilstanden er Relativ.',
        swipeTitle: 'Stryg for at flytte',
        swipeDesc: 'Stryg i det aktive skærmområde for at flytte den eksterne markør.',
        tapTitle: 'Tryk for at klikke',
        tapDesc: 'Et kort tryk sender ét venstreklik.',
        holdTitle: 'Hold for venstre knap nede',
        holdDesc: 'Tryk og hold fingeren stille i ca. 1 sekund for at holde venstre museknap nede.',
        dragTitle: 'Flyt efter hold for at trække',
        dragDesc: 'Når hold er aktivt, kan du flytte fingeren for at trække med venstre knap nede.'
      },
      hidOnly: {
        errors: {
          load: 'Kunne ikke indlæse status for HID-Only-tilstand. Prøv igen om et øjeblik.',
          update:
            'Kunne ikke bekræfte, om HID-Only-tilstand blev opdateret. Åbn denne indstilling igen for at kontrollere den aktuelle status, og prøv derefter igen.'
        },
        title: 'Kun HID-tilstand',
        desc: 'Hvis din mus og tastatur holder op med at reagere, og nulstilling af HID ikke hjælper, kan det være et kompatibilitetsproblem mellem NanoKVM og enheden. Prøv at aktivere HID-Only-tilstand for bedre kompatibilitet.',
        tip1: 'Aktivering af HID-Only-tilstand vil afmontere den virtuelle U-disk og det virtuelle netværk',
        tip2: 'I HID-Only-tilstand er billedmontering deaktiveret',
        enable: 'Aktiver HID-kun tilstand',
        disable: 'Deaktiver HID-kun tilstand'
      }
    },
    preview: {
      title: 'Forhåndsvisning'
    },
    image: {
      storageReadOnly: 'Der kan ikke skrives til fillageret. Slå den virtuelle disk fra.',
      storageBusy:
        'Lageret er optaget. Afslut eller annuller filoverførslen, før du skifter den virtuelle disk.',
      title: 'Diskbilleder',
      loading: 'Kontrollerer...',
      empty: 'Ingen fundet',
      upload: 'Upload billede',
      dropFile: 'Slip et diskbillede her, eller klik for at vælge en fil',
      uploadAction: 'Overfør',
      cancelAction: 'Annuller',
      mount: 'Monter billede',
      mounted: 'Monteret',
      uploadError: 'Upload mislykkedes, prøv venligst igen',
      uploadInsufficientStorage: 'Ikke nok lagerplads',
      uploadTooLarge: 'Imaget overskrider størrelsesgrænsen',
      cancelUploadError: 'Uploaden kunne ikke annulleres',
      mountMode: 'Monteringstilstand',
      massStorage: 'Masselager',
      cdRom: 'CD ROM',
      readOnly: 'Skrivebeskyttet',
      mountFailed: 'Montering af diskbillede mislykkedes',
      mountDesc:
        'På nogle systemer kan det være nødvendigt at skubbe den virtuelle disk ud på fjerncomputeren før du kan montere diskbilledet.',
      unmountFailed: 'Afmontering mislykkedes',
      unmountDesc:
        'I nogle systemer skal du manuelt eje afbryde på fjernværten, før du afmonterer billedet.',
      refresh: 'Opdater billedlisten',
      download: 'Download billede',
      attention: 'Opmærksomhed påkrævet',
      deleteConfirm: 'Er du sikker på, at du vil slette dette billede?',
      deleteFailed: 'Billedet kunne ikke slettes',
      okBtn: 'Ja',
      cancelBtn: 'Annuller'
    },
    script: {
      title: 'Script',
      upload: 'Upload',
      run: 'Kør',
      runBackground: 'Kør i baggrunden',
      runFailed: 'Kørsel mislykkedes',
      uploadFailed: 'Scriptet kunne ikke uploades',
      deleteFailed: 'Scriptet kunne ikke slettes',
      attention: 'Opmærksomhed påkrævet',
      delDesc: 'Er du sikker på at du vil slette denne fil?',
      confirm: 'Ja',
      cancel: 'Annuller',
      delete: 'Slet',
      close: 'Luk'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal til NanoKVM',
      confirm: 'OK',
      submit: 'Send',
      username: 'Brugernavn',
      password: 'Adgangskode',
      authCode: 'Godkendelseskode',
      authorizeTitle: 'Autoriser webterminal',
      codeAccountHint: 'Indtast en godkendelseskode eller en ubrugt gendannelseskode.',
      authFailed: 'Godkendelse mislykkedes'
    },
    wol: {
      errors: {
        send: 'Kunne ikke sende vækkekommandoen. Kontrollér MAC-adressen og netværksforbindelsen, og prøv igen.',
        loadFailed:
          'Kunne ikke indlæse Wake-on-LAN-enhedslisten. Kontrollér, at NanoKVM er online, og prøv igen.',
        renameFailed: 'Kunne ikke omdøbe denne enhed. Kontrollér MAC-adressen, og prøv igen.',
        deleteFailed: 'Kunne ikke slette denne enhed. Prøv igen om et øjeblik.'
      },
      title: 'Wake-on-LAN',
      sending: 'Sender Wake-on-LAN magic packet',
      sent: 'Wake-on-LAN magic packet sendt',
      input: 'Angiv MAC-adresse',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (valgfri)',
      sha256Placeholder: 'Indtast en SHA-256-kontrolsum på 64 tegn',
      invalidSHA256: 'SHA-256 skal være en hexadecimal streng på 64 tegn',
      title: 'Billedhenter',
      input: 'Indtast venligst et fjernbillede URL',
      ok: 'OK',
      disabled: '/data partitionen er RO, så vi kan ikke downloade billedet',
      downloadingFile: 'Downloader: {{file}}',
      downloadingProgress: 'Downloader ({{percentage}}): {{file}}',
      failed: 'Mislykkedes',
      insufficientStorage: 'Ikke nok lagerplads',
      remaining: '{{time}} tilbage',
      cancel: 'Annuller',
      cancelFailed: 'Downloaden kunne ikke annulleres',
      pause: 'Sæt på pause',
      resume: 'Fortsæt',
      pauseFailed: 'Downloaden kunne ikke sættes på pause',
      resumeFailed: 'Downloaden kunne ikke fortsættes'
    },
    settings: {
      back: 'Tilbage',
      close: 'Luk',
      ai: {
        errors: {
          loadServices:
            'Kunne ikke indlæse status for AI-tjenester. Kontrollér, at NanoKVM er online, og prøv igen.',
          loadMcpConfig:
            'Kunne ikke indlæse konfigurationen for MCP-tjenesten. Prøv igen om et øjeblik.',
          updateMcpService:
            'Kunne ikke opdatere status for MCP-tjenesten. Kontrollér den aktuelle status, før du prøver igen.',
          accessMcpApiKey:
            'Kunne ikke hente MCP API-nøglen. Prøv igen, og gennemfør bekræftelsen, hvis du bliver bedt om det.',
          regenerateMcpApiKey: 'Kunne ikke gendanne MCP API-nøglen. Prøv igen om et øjeblik.',
          regeneratedMcpApiKeyUnavailable:
            'API-nøglen blev gendannet, men kunne ikke vises. Prøv at vise den igen. Den tidligere nøgle er ikke længere gyldig.',
          loadMemoryFabricState:
            'Kunne ikke indlæse status for Memory Fabric-tjenesten. Prøv igen om et øjeblik.',
          updateMemoryFabricService:
            'Kunne ikke opdatere status for Memory Fabric-tjenesten. Kontrollér den aktuelle status, før du prøver igen.',
          loadScreenTimelapseState:
            'Kunne ikke indlæse status for skærm-timelapse-tjenesten. Prøv igen om et øjeblik.',
          updateScreenTimelapseService:
            'Kunne ikke opdatere status for skærm-timelapse-tjenesten. Kontrollér den aktuelle status, før du prøver igen.',
          openScreenTimelapse:
            'Kunne ikke åbne skærm-timelapse-afspilning. Prøv igen om et øjeblik.',
          loadScreenTimelapseRecordings:
            'Kunne ikke indlæse skærm-timelapse-optagelser. Prøv igen om et øjeblik.',
          playScreenTimelapse:
            'Denne timelapse-optagelse kan ikke afspilles lige nu. Vælg datoen igen, eller prøv senere.',
          screenTimelapseFullscreen:
            'Kunne ikke ændre fuldskærmstilstand. Kontrollér browserens tilladelser, og prøv igen.'
        },
        title: 'AI',
        mcpService: 'MCP-tjeneste',
        mcpServiceDesc:
          'Godkendt samlet gateway til styring, medier og aktiverede Memory Fabric-værktøjer',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Behandl agenters konteksthukommelse og OCR lokalt',
        memoryFabricRemoteRequiresMcp:
          'Aktivér MCP-tjenesten ovenfor for at gøre hukommelses- og OCR-værktøjer tilgængelige eksternt.',
        memoryFabricRemoteAvailable:
          'Hukommelses- og OCR-værktøjer er tilgængelige via endpointet /api/mcp ovenfor med den samme API-nøgle.',
        memoryFabricConfig: {
          title: 'Konfiguration',
          description: 'Administrer kun indstillingerne for standardmodellen i Memory Fabric.',
          loadFailed: 'Memory Fabric-konfigurationen kunne ikke indlæses.',
          saveFailed: 'Memory Fabric-konfigurationen kunne ikke gemmes.',
          saved: 'Konfigurationen er gemt.',
          savedRestarted: 'Konfigurationen er gemt, og Memory Fabric er genstartet.',
          noChanges: 'Ingen ændringer i konfigurationen.',
          reload: 'Genindlæs',
          save: 'Gem',
          unsaved: 'Ikke-gemte ændringer',
          conflictTitle: 'Konfigurationen er ændret',
          conflictDescription:
            'Konfigurationen er blevet ændret et andet sted. Genindlæs den nyeste version, før du gemmer igen.',
          conflictReload: 'Indlæs nyeste version',
          discardTitle: 'Kassér ikke-gemte ændringer?',
          discardDescription:
            'Ved genindlæsning erstattes den aktuelle formular med den senest gemte konfiguration.',
          discardConfirm: 'Kassér og genindlæs',
          sections: {
            models: 'Modeller'
          },
          openAICompatibleHint:
            'Understøtter OpenAI-kompatible modeludbydere. Angiv modelnavnet, basis-URL’en og API-nøglen fra din udbyder.',
          fields: {
            model: 'Modelnavn',
            baseUrl: 'Basis-URL',
            maxTokens: 'Maksimalt antal tokens',
            timeoutSeconds: 'Timeout (sekunder)'
          },
          apiKey: {
            title: 'API-nøgle',
            configured: 'Konfigureret',
            notConfigured: 'Ikke konfigureret',
            keep: 'Behold',
            replace: 'Erstat',
            clear: 'Fjern',
            replacement: 'Ny API-nøgle',
            replacementPlaceholder: 'Indtast den nye API-nøgle',
            clearHint: 'Den gemte API-nøgle fjernes; miljøvariablen kan stadig levere en nøgle.',
            required: 'Indtast en ny API-nøgle.'
          },
          validation: {
            required: 'Dette felt er påkrævet.',
            invalidUrl: 'Indtast en gyldig HTTP- eller HTTPS-URL.'
          }
        },
        screenTimelapseService: 'Skærm-timelapse',
        screenTimelapseServiceDesc:
          'Afspil Memory Fabric-skærmhistorik, og søg i optaget skærmindhold',
        screenTimelapseSearchRequiresMemoryFabric:
          'Hvis du vil søge i optaget indhold, skal du også aktivere Memory Fabric.',
        screenTimelapseOpen: 'Åbn',
        screenTimelapseSearchPlaceholder: 'Søg i videoindhold...',
        screenTimelapseNoVideos: 'Ingen timelapse-videoer fundet',
        screenTimelapseNoResults: 'Ingen matchende billeder fundet',
        screenTimelapseMultipleResults: 'Søgeresultater',
        screenTimelapseUnavailable: 'Ikke tilgængelig',
        screenTimelapsePlayerUnsupported: 'Denne browser kan ikke afspille timelapse-video',
        screenTimelapseSearchFailed: 'Søgning mislykkedes',
        screenTimelapseLoading: 'Indlæser timelapse...',
        screenTimelapseSearchAction: 'Søg i timelapse',
        screenTimelapsePlay: 'Afspil timelapse',
        screenTimelapsePause: 'Sæt timelapse på pause',
        screenTimelapseEnterFullscreen: 'Gå til fuld skærm',
        screenTimelapseExitFullscreen: 'Afslut fuld skærm',
        screenTimelapseFullscreenUnsupported: 'Denne browser kan ikke gå til fuld skærm',
        screenTimelapsePreviousDay: 'Forrige dag',
        screenTimelapseNextDay: 'Næste dag',
        screenTimelapseDeleteDay: 'Slet optagelsen for {{date}}',
        screenTimelapseDelete: 'Slet',
        screenTimelapseDeleteConfirmTitle: 'Slet timelapse-optagelsen?',
        screenTimelapseDeleteConfirmDesc:
          'Optagelsen for {{date}} bruger {{size}}. Handlingen kan ikke fortrydes.',
        screenTimelapseDeleteSuccess: 'Optagelsen for {{date}} blev slettet',
        screenTimelapseDeleteFailed: 'Optagelsen for {{date}} kunne ikke slettes',
        endpoint: 'Slutpunkt',
        apiKey: 'API-nøgle',
        regenerate: 'Generér igen',
        copyApiKey: 'Kopiér API-nøgle',
        copyEndpoint: 'Kopiér slutpunkt',
        showApiKey: 'Vis API-nøgle',
        regenerateConfirmTitle: 'Generér API-nøgle igen',
        regenerateConfirmDesc:
          'Eksisterende MCP-klienter holder op med at fungere, indtil de opdateres med den nye nøgle. Vil du fortsætte?',
        okBtn: 'Ja',
        cancelBtn: 'Nej'
      },
      title: 'Indstillinger',
      about: {
        title: 'Om NanoKVM',
        information: 'Information',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Program version',
        applicationTip: 'NanoKVM webapplikationsversion',
        arch: 'Arkitektur',
        device_number: 'Enhedsnummer',
        image: 'Firmware version',
        imageTip: 'NanoKVM systemafbildningsversion',
        deviceKey: 'Enhedsnøgle',
        community: 'Fællesskab',
        hostname: 'Værtsnavn',
        hostnameUpdated: 'Værtsnavn opdateret. Genstart for at anvende.',
        ipType: {
          Wired: 'Kablet',
          Wireless: 'Trådløs',
          Other: 'Andet'
        }
      },
      appearance: {
        title: 'Udseende',
        display: 'Skærm',
        language: 'Sprog',
        languageDesc: 'Vælg sproget til grænsefladen',
        languageChangeFailed: 'Kunne ikke skifte sprog',
        webTitle: 'Webtitel',
        webTitleDesc: 'Tilpas websidens titel',
        menuBar: {
          title: 'Menulinje',
          mode: 'Visningstilstand',
          modeDesc: 'Vis menulinje på skærmen',
          modeOff: 'Fra',
          modeAuto: 'Skjul automatisk',
          modeAlways: 'Altid synlig',
          icons: 'Undermenuikoner',
          showIcon: 'Vis ikon',
          activateOnConnect: 'Aktivér ved tilslutning',
          stopSharing: 'Stop deling',
          iconsDesc: 'Vis undermenuikoner i menulinjen'
        }
      },
      screen: {
        title: 'Skærm',
        videoMode: {
          title: 'Videotilstand',
          description: 'Vælg kodnings- og transmissionsmetoden'
        },
        edid: {
          description: 'Vælg displayparametre',
          default: 'Standard',
          custom: 'Brugerdefineret',
          document: 'Dokument',
          monitorName: 'Skærmnavn',
          monitorNameDescription:
            'Maks. 13 ASCII-tegn, der kan udskrives; lad feltet stå tomt for at bruge det oprindelige navn fra EDID',
          monitorNamePlaceholder: 'Brug oprindeligt EDID-navn',
          monitorNameApply: 'Anvend',
          monitorNameInvalid: 'Indtast højst 13 ASCII-tegn, der kan udskrives',
          monitorNameUnsupported:
            'Den aktuelle EDID understøtter ikke tilsidesættelse af skærmnavnet',
          monitorNameLoadFailed: 'Skærmnavnet kunne ikke indlæses',
          monitorNameUpdateFailed: 'Skærmnavnet kunne ikke opdateres',
          powerConfirmTitle: 'Bekræft før opdatering af EDID',
          powerConfirmDescription:
            'Tilslut først PD-strøm. Opdatering af EDID kan kortvarigt afbryde skærmen; hvis enheden derefter ikke kan registrere skærmen korrekt, skal du genstarte den manuelt.',
          powerConfirmOk: 'Fortsæt',
          powerConfirmOkCountdown: 'Fortsæt ({{seconds}})',
          powerConfirmCancel: 'Annuller',
          errors: {
            badRequest: 'Ugyldig EDID-anmodning',
            invalidFileName: 'Vælg en .bin-EDID-fil med et gyldigt filnavn',
            fileTooLarge: 'EDID-filen er for stor',
            invalidLength: 'EDID-filen skal være på 128, 256, 384 eller 512 byte',
            invalidHeader: 'EDID-headeren er ugyldig',
            unsupportedVersion: 'Denne EDID-version understøttes ikke',
            extensionCountMismatch:
              'Antallet af EDID-udvidelsesblokke svarer ikke til filens længde',
            invalidChecksum: 'EDID-kontrolsummen er ugyldig',
            invalidCta: 'CTA-udvidelsens struktur er ugyldig',
            currentCannotOverwrite:
              'Den EDID, der bruges i øjeblikket, kan ikke overskrives. Skift først til en anden EDID',
            saveFailed: 'EDID-filen kunne ikke gemmes',
            switchFailed: 'Der kunne ikke skiftes EDID',
            uploadFailed: 'EDID-filen kunne ikke uploades',
            deleteFailed: 'EDID-filen kunne ikke slettes',
            networkFailed: 'Netværksfejl. Prøv igen'
          }
        },
        rateControlMode: {
          title: 'Hastighedskontroltilstand',
          description: 'Vælg metoden til at styre bithastigheden',
          cbr: 'CBR (Konstant Bitrate): Prioriterer forbindelsesstabilitet. Minimerer hakken, men billedkvaliteten kan falde under højbevægelsesscener.',
          vbr: 'VBR (Variabel Bitrate): Prioriterer billedets klarhed. Højbevægelsesscener kan forårsage båndbreddespidser, hvilket potentielt kan føre til latency eller hakken.'
        },
        bitrate: {
          title: 'Bitrate',
          description: 'Balancer billedets klarhed med båndbreddeforbrug',
          lower:
            'Lavere værdier reducerer latenstiden på dårlige netværk, men kompromitterer billedkvaliteten, hvilket potentielt kan forårsage sløring eller artefakter.',
          higher:
            'Højere værdier forbedrer billedets klarhed, men bruger mere båndbredde. Kan øge latensen, hvis forbindelsen er langsom.'
        },
        gop: {
          description: 'Indstil intervallet mellem fulde keyframes',
          lower:
            'Lavere værdier sender hele billeder oftere. Dette fremskynder gendannelse fra visuelle artefakter, men bruger mere båndbredde og reducerer billeddetaljer.',
          higher:
            'Højere værdier optimerer til trinvise opdateringer. Dette forbedrer billeddetaljerne og sparer båndbredde, men gendannelse fra visuelle artefakter vil tage længere tid.'
        },
        quality: {
          title: 'Kvalitet',
          description: 'Indstil skærmens billedkvalitet'
        },
        fps: {
          auto: 'Automatisk',
          description: 'Indstil skærmens billedhastighed',
          tip1: 'Indstilling af værdien til 0 aktiverer automatisk tilstand, og synkroniserer skærmen med fjernværtens billedhastighed.',
          tip2: 'Sænkning af billedhastigheden reducerer båndbreddeforbruget betydeligt. Sænk den manuelt under netværksustabilitet, eller når høje billedhastigheder ikke er påkrævet.'
        },
        scale: {
          title: 'Skala',
          description: 'Indstil displayets skaleringsforhold'
        }
      },
      device: {
        title: 'Enhed',
        virtualAudioConfirm: {
          title: 'Bekræft før du skifter virtuel lyd',
          description:
            'Tilslut først PD-strøm. Når du skifter virtuel lyd, kan skærmen kortvarigt blive afbrudt; hvis enheden derefter ikke kan registrere skærmen korrekt, skal du genstarte den manuelt.',
          ok: 'Fortsæt med at skifte',
          okCountdown: 'Fortsæt med at skifte ({{seconds}})',
          cancel: 'Annuller'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurer Wi-Fi',
          apMode: 'AP-tilstand er aktiveret, opret forbindelse til Wi-Fi manuelt',
          others: 'Andre...',
          connect: 'Tilslut Wi-Fi',
          connectDesc1: 'Indtast venligst netværkets ssid og adgangskode',
          connectDesc2: 'Indtast venligst adgangskoden for at deltage i dette netværk',
          disconnect: 'Er du sikker på at afbryde netværket?',
          ssid: 'Navn',
          password: 'Adgangskode',
          joinBtn: 'Deltag',
          confirmBtn: 'Okay',
          cancelBtn: 'Annuller',
          preset: {
            title: 'Wi-Fi-forudindstilling',
            description:
              'Forudgem ét mål-Wi-Fi; enheden opretter automatisk forbindelse til det ved opstart',
            empty: 'Ingen',
            addBtn: 'Tilføj forudindstilling',
            removeBtn: 'Fjern forudindstilling',
            saveTitle: 'Forudgem Wi-Fi',
            saveDesc:
              'Gem målnetværket nu; enheden opretter automatisk forbindelse, når det findes ved opstart',
            saveBtn: 'Gem',
            passwordPlaceholder: 'Adgangskode (lad stå tom for åbne netværk)',
            saveFailed: 'Kunne ikke gemme Wi-Fi-forudindstillingen',
            removeFailed: 'Kunne ikke fjerne Wi-Fi-forudindstillingen',
            removeConfirm: 'Vil du virkelig fjerne det forudgemte Wi-Fi?',
            warning:
              'Når den er gemt, vil enheden ved hver opstart prøve at tilslutte sig dette Wi-Fi, når det registreres, hvilket kan forstyrre tilslutning til andre Wi-Fi-netværk. Forudindstillingen forbliver aktiv, indtil du redigerer eller fjerner den.'
          }
        },
        capture: {
          title: 'Videooptagelse',
          description: 'Aktiver videooptagelse'
        },
        ssh: {
          description: 'Aktiver SSH fjernadgang',
          tip: 'Når du har aktiveret SSH, skal du angive en stærk adgangskode til root-brugeren i terminalen.'
        },
        swap: {
          errors: {
            load: 'Kunne ikke indlæse status for swap-hukommelse. Prøv igen om et øjeblik.',
            update:
              'Kunne ikke opdatere indstillingen for swap-hukommelse. Kontrollér den aktuelle status, før du prøver igen.'
          },
          title: 'Swap-hukommelse',
          description: 'Aktiver komprimeret swap-hukommelse'
        },
        advanced: 'Avancerede indstillinger',
        staticIp: {
          title: 'Statisk IP',
          config: 'Konfig',
          tip: 'For at forhindre tab af forbindelse anbefales det at konfigurere, mens du har fysisk adgang til enheden.',
          document: 'Tjek dokumentet.',
          enable: 'Aktiver statisk IP',
          placeholder: 'Format: IP/CIDR [Gateway]\nEksempel: 192.168.1.10/22',
          notEmpty: 'IP-adressen må ikke være tom.',
          notValid: 'IP-adresseformatet er ugyldigt.',
          failed: 'Konfiguration mislykkedes.',
          success: 'Konfigurationen er fuldført. Tjek venligst om IP-adressen er tilgængelig.',
          okBtn: 'Okay',
          cancelBtn: 'Annuller'
        },
        usbNetwork: {
          title: 'USB-netværk',
          mode: 'Driftstilstand',
          server: 'Servertilstand',
          serverDesc: 'Tildel fjernværten en IP-adresse via den indbyggede DHCP-server.',
          client: 'Klienttilstand',
          clientDesc: 'Hent en IP-adresse fra fjernværten via DHCP.',
          forwarding: 'IP-videresendelse',
          forwardingTip:
            'Hvis fjernværten skal have adgang til internettet via NanoKVM, skal du aktivere IP-videresendelse.',
          manualIp: 'Manuel IP-adresse',
          ipv4Placeholder: 'IPv4/CIDR, f.eks. 10.10.10.1/24',
          invalidIpv4: 'Indtast en gyldig IPv4/CIDR-adresse.',
          loadFailed: 'USB-netværkskonfigurationen kunne ikke indlæses.',
          failed: 'Konfigurationen mislykkedes.',
          success: 'Konfigurationen er fuldført.',
          applyBtn: 'Anvend',
          cancelBtn: 'Annuller'
        },
        usbId: {
          title: 'USB-enhedsidentitet',
          config: 'Konfig',
          tip: 'Tilpassede USB-ID’er ændrer, hvordan værten identificerer denne enhed. Anvendelse genstarter USB-gadgeten og kan kortvarigt afbryde tastatur, mus eller netværk.',
          enable: 'Aktiver tilpasset USB-enhedsidentitet',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Enhedsversion (bcdDevice)',
          serialnumber: 'Serienummer',
          manufacturer: 'Producent',
          product: 'Produkt',
          vidPlaceholder: 'f.eks. 0x359F',
          pidPlaceholder: 'f.eks. 0x3305',
          bcdDevicePlaceholder: 'f.eks. 0x0101 (valgfrit)',
          serialnumberPlaceholder: 'f.eks. 0123456789ABCDEF (valgfrit)',
          manufacturerPlaceholder: 'f.eks. sipeed (valgfrit)',
          productPlaceholder: 'f.eks. NanoKVM-Go (valgfrit)',
          notEmpty: 'VID og PID må ikke være tomme.',
          notValid: 'VID/PID-format er ugyldigt. Brug hex som 0x359F.',
          bcdNotValid: 'Ugyldigt bcdDevice-format. Brug hex som 0x0101, eller lad stå tomt.',
          failed: 'Konfiguration mislykkedes.',
          success: 'Konfiguration fuldført. USB-gadget genstartet.',
          okBtn: 'Okay',
          cancelBtn: 'Annuller'
        },
        diskInquiry: {
          title: 'USB-drevnavn',
          config: 'Konfiguration',
          tip: 'Navnet på den virtuelle USB-nøgle, som den fjerne computer ser. Gem genstarter USB og kan midlertidigt afbryde tastatur, mus eller netværk.',
          enable: 'Brug et tilpasset USB-drevnavn',
          placeholder: 'f.eks. sipeed  Flash Drive     1.00',
          notEmpty: 'Navnet må ikke være tomt.',
          notValid: 'Ugyldigt navn. Op til 28 udskrivbare ASCII-tegn uden linjeskift.',
          failed: 'Konfiguration mislykkedes.',
          success: 'Konfiguration fuldført. USB-gadget genstartet.',
          okBtn: 'OK',
          cancelBtn: 'Annuller'
        },
        diskInquiryCdrom: {
          title: 'Virtuelt CD-drevnavn',
          config: 'Konfiguration',
          tip: 'Navnet på det virtuelle CD/DVD-drev på den fjerne computer. Gem genstarter USB og kan midlertidigt afbryde tastatur, mus eller netværk.',
          enable: 'Brug et tilpasset CD-drevnavn',
          placeholder: 'f.eks. sipeed  Optical Drive   1.00',
          notEmpty: 'Navnet må ikke være tomt.',
          notValid: 'Ugyldigt navn. Op til 28 udskrivbare ASCII-tegn uden linjeskift.',
          failed: 'Konfiguration mislykkedes.',
          success: 'Konfiguration fuldført. USB-gadget genstartet.',
          okBtn: 'OK',
          cancelBtn: 'Annuller'
        },
        uac2FunctionName: {
          title: 'USB-lydnavn',
          config: 'Konfiguration',
          tip: 'Navnet på den virtuelle USB-mikrofon/lydenhed på den fjerne computer. Gem genstarter USB og kan midlertidigt afbryde tastatur, mus eller netværk.',
          enable: 'Brug et tilpasset USB-lydnavn',
          placeholder: 'f.eks. NanoKVM-Go Audio',
          notEmpty: 'Navnet må ikke være tomt.',
          notValid: 'Ugyldigt navn. Op til 64 udskrivbare tegn uden linjeskift.',
          failed: 'Konfiguration mislykkedes.',
          success: 'Konfiguration fuldført. USB-gadget genstartet.',
          okBtn: 'OK',
          cancelBtn: 'Annuller'
        },
        lowPower: {
          title: 'Laveffekttilstand',
          description: 'Reducer strømforbruget i inaktiv tilstand',
          tips: 'Råd',
          tip1: 'Når funktionen er aktiveret, sættes videooptagelsen på pause i inaktive perioder for at reducere strømforbrug og varmeudvikling.',
          tip2: 'Genoptagelse af videooptagelsen kan kræve geninitialisering, hvor videoen midlertidigt kan være utilgængelig.',
          okBtn: 'Okay',
          cancelBtn: 'Annuller'
        },
        mouseJiggler: {
          title: 'Mus Jiggler',
          description: 'Forhindrer fjernværten i at sove',
          disable: 'Deaktiver',
          absolute: 'Absolut tilstand',
          relative: 'Relativ tilstand'
        },
        fingerRobot: {
          title: 'Fingerrobot',
          description: 'Aktivér styring af fingerrobotten'
        },
        mdns: {
          description: 'Aktiver mDNS opdagelsestjeneste'
        },
        hidOnly: 'HID-Kun tilstand',
        imageMounted: 'Billedmonteret',
        disk: 'Virtuel disk',
        diskDesc: 'Monter virtuel U-disk på den fjerntliggende vært',
        sdCard: {
          warning: 'Initialiser virtuel disk',
          tip1: 'Første gang funktionen aktiveres, initialiseres den virtuelle disk, og eksisterende billedfiler samt data på den virtuelle disk slettes. Handlingen kan ikke fortrydes. Sikkerhedskopiér vigtige filer først.',
          tip2: 'Fortsæt initialiseringen?',
          capacity: 'Kapacitet',
          disable: 'Deaktiver',
          disableWarning: 'Deaktiver virtuel disk?',
          disableTip:
            'Deaktivering sletter alle filer på den virtuelle disk og frigiver den lagerplads på enheden, som den bruger. Næste gang den aktiveres, skal du vælge kapacitet og initialisere den igen. Handlingen kan ikke fortrydes.'
        },
        network: 'Virtuelt netværk',
        networkDesc: 'Monter virtuelt netværkskort på den fjerntliggende vært',
        mic: 'Virtuel lyd',
        micDesc: 'Vælg den virtuelle USB-lydtilstand til fjernværten',
        virtualAudioMode: {
          off: 'Fra',
          mic: 'Kun mikrofon',
          duplex: 'Mikrofon + højttaler'
        },
        speaker: 'Virtuel højttaler',
        speakerDesc:
          'Når den er aktiveret, erstatter den standard-DP-lydudgangen for at understøtte fjernopkald med bedre kompatibilitet. Når den er deaktiveret, kan kun medielyd videresendes, ikke opkaldslyd.',
        micTip:
          'På grund af USB-endepunktsbegrænsninger vil aktivering af den virtuelle mikrofon deaktivere den sekundære USB-skærm.\nOg det anbefales at deaktivere denne funktion under BIOS-operationer for at sikre optimal kompatibilitet.',
        reboot: 'Genstart',
        rebootDesc: 'Er du sikker på, at du vil genstarte NanoKVM?',
        rebootToFlashing: 'Genstart til flashingtilstand',
        rebootToFlashingButton: 'Genstart',
        rebootToFlashingTitle: 'Genstart til flashingtilstand?',
        rebootToFlashingDesc:
          'NanoKVM genstarter og går i USB-flashingtilstand. Sørg for, at USB-forbindelsen til flashing er tilsluttet, og at flashværktøjet er klar.',
        datetime: {
          timezone: 'Tidszone',
          datetime: 'Datotid',
          dateAndTime: 'Dato og klokkeslæt',
          synchronize: 'Synkroniser tid',
          lastSynchronization: 'Sidste synkronisering: ',
          notSynchronized: 'Systemtiden er ikke synkroniseret endnu',
          syncNow: 'Synkroniser nu',
          manualSet: 'Indstil manuelt',
          manualSetTitle: 'Indstil tiden manuelt',
          manualSetDescription: 'Du skal logge ind igen, når tiden er ændret.',
          useBrowserTime: 'Brug browserens tid',
          setTime: 'Indstil tid',
          invalidTime: 'Indtast en gyldig dato og et gyldigt klokkeslæt.',
          setFailed: 'Systemtiden kunne ikke indstilles. Prøv igen.',
          setSucceeded: 'Systemtiden er opdateret. Log ind igen.'
        },
        dialBackground: {
          errors: {
            load: 'Kunne ikke indlæse baggrundsbilleder til urskiven. Prøv igen om et øjeblik.',
            upload:
              'Kunne ikke uploade baggrundsbilledet. Kontrollér, at filen er et gyldigt BMP-billede, og prøv igen.',
            switch: 'Kunne ikke skifte baggrund for urskiven. Opdatér billedlisten, og prøv igen.'
          },
          title: 'Urskivebaggrund',
          description: 'Vælg et BMP-billede fra /kvmcomm/ui/bg/ som urskivebaggrund',
          upload: 'Upload',
          loading: 'Indlæser...',
          empty: 'Ingen BMP-billeder fundet',
          current: 'Aktuel',
          onlyBmp: 'Kun BMP-billeder understøttes',
          uploaded: 'Upload fuldført',
          switched: 'Baggrund skiftet'
        },
        okBtn: 'Ja',
        cancelBtn: 'Annuller'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Genstart Tailscale?',
        stop: 'Stoppe Tailscale?',
        stopDesc: 'Log Tailscale ud, og deaktiver automatisk start ved opstart.',
        loading: 'Indlæser...',
        notInstall: 'Tailscale ikke fundet! Installer det for at fuldføre opsætningen.',
        install: 'Installer',
        installing: 'Installerer',
        failed: 'Installation mislykkedes',
        retry: 'Opdater siden og prøv igen. Ellers prøv at installere manuelt.',
        download: 'Download',
        package: 'installationspakken',
        unzip: 'og udpak den',
        upTailscale: 'Upload tailscale til følgende mappe på enheden: /usr/bin/',
        upTailscaled: 'Upload tailscaled til følgende mappe på enheden: /usr/sbin/',
        refresh: 'Opdater sides',
        notRunning: 'Tailscale kører ikke. Start det for at fortsætte.',
        run: 'Start',
        notLogin:
          'Enheden er ikke tilknyttet en Tailscale-konto endnu. Log ind for at fuldføre tilknytningen til din konto.',
        urlPeriod: 'Denne URL er gyldig i 10 minutter',
        login: 'Log ind',
        loginSuccess: 'Log ind lykkedes',
        enable: 'Aktiver Tailscale',
        exitNode: 'Udgangsnode',
        exitNodeDesc: 'Annoncér denne enhed som internetudgang for dit tailnet.',
        subnetRoute: 'Undernet-rute',
        subnetRouteDesc:
          'Annoncér det direkte tilsluttede LAN-undernet {{subnet}} til dit tailnet.',
        subnetUnavailable: 'Der er intet direkte tilsluttet LAN-undernet tilgængeligt.',
        routingUnavailable: 'Tailscale-routingindstillinger er ikke tilgængelige på denne enhed.',
        pendingApproval:
          'Efter aktivering skal den annoncerede rute godkendes i Tailscale-administrationskonsollen.',
        routingUpdateFailed: 'Tailscale-routingindstillingerne kunne ikke opdateres.',
        deviceName: 'Enhedens navn',
        deviceIP: 'Enhedens IP',
        account: 'Konto',
        logout: 'Log ud',
        logoutDesc: 'Er du sikker på, at du vil logge ud?',
        uninstall: 'Afinstaller Tailscale',
        uninstallDesc: 'Er du sikker på, at du vil afinstallere Tailscale?',
        okBtn: 'Ja',
        cancelBtn: 'Nej'
      },
      update: {
        title: 'Kontroller for opdatering',
        queryFailed: 'Opdateringskontrol mislykkedes',
        updateFailed: 'Opdatering fejlede. Prøv igen.',
        isLatest: 'Du har allerede den nyeste version.',
        available: 'En opdatering er tilgængelig. Vil du installere den?',
        updating: 'Opdatering i gang. Vent venligst...',
        confirm: 'Bekræft',
        upgrade_tip_nano:
          'En ny version af NanoKVM er tilgængelig. Er du sikker på, at du vil opdatere nu?',
        upgrade_tip_firmware:
          'En ny firmwareversion er tilgængelig. Er du sikker på, at du vil opdatere nu?',
        cancel: 'Annuller',
        preview: 'Forhåndsvisning af opdateringer',
        previewDesc: 'Få tidlig adgang til nye funktioner og forbedringer',
        previewTip:
          'Vær opmærksom på, at forhåndsvisningsudgivelser kan indeholde fejl eller ufuldstændig funktionalitet!',
        download: 'Download',
        install: 'Installer',
        changelog: 'Ændringslog',
        restart: 'Genstart'
      },
      account: {
        title: 'Konto',
        webAccount: 'Webkontonavn',
        password: 'Adgangskode',
        twoFactor: 'Tofaktorgodkendelse',
        enableTwoFactor: 'Aktiver',
        enableTwoFactorDesc:
          'Indtast din nuværende adgangskode for at starte opsætningen af tofaktorgodkendelse.',
        disableTwoFactor: 'Deaktiver',
        recoveryCodes: 'Gendannelseskoder',
        enableRecoveryCodes: 'Aktiver',
        updateRecoveryCodes: 'Opdater',
        recoveryCodesTooltipDisabled:
          'Gendannelseskoder er reserveoplysninger til tofaktorgodkendelse. Du kan generere dem efter aktivering af tofaktorgodkendelse.',
        recoveryCodesTooltipEnabled:
          'Gendannelseskoder er kun gyldige, mens tofaktorgodkendelse er aktiveret. Deaktivering af tofaktorgodkendelse ugyldiggør alle gendannelseskoder; mens den forbliver aktiveret, kan de kun opdateres.',
        recoveryCodesRegenerateDesc:
          'Indtast din nuværende adgangskode og enten en godkendelseskode eller en ubrugt gendannelseskode. Opdatering af gendannelseskoder ugyldiggør alle eksisterende gendannelseskoder.',
        recoveryCodesVerifyFailed:
          'Kunne ikke verificere adgangskoden og godkendelseskoden eller gendannelseskoden.',
        recoveryCodesSavedTitle: 'Gem disse gendannelseskoder nu',
        recoveryCodesSavedDesc:
          'Gendannelseskoder kan kun vises én gang. Gem dem et sikkert sted, før du forlader denne side.',
        enableRecoveryCodesDesc:
          'Indtast din nuværende adgangskode og en godkendelseskode for at generere gendannelseskoder. Gendannelseskoder kan kun vises én gang, så gem dem, før du forlader denne side.',
        copyRecoveryCodes: 'Kopiér',
        downloadRecoveryCodes: 'Download',
        recoveryCodesCopyFailed: 'Kunne ikke kopiere gendannelseskoder.',
        done: 'Færdig',
        currentPassword: 'Nuværende adgangskode',
        authCode: 'Godkendelseskode',
        authCodeOrRecoveryCode: 'Godkendelseskode eller gendannelseskode',
        authRateLimited: 'For mange mislykkede forsøg. Prøv igen om {{seconds}} s.',
        verifyPasswordChange: 'Verificer adgangskodeændring',
        verifyPasswordChangeDesc:
          'Indtast din nuværende adgangskode og enten en godkendelseskode eller en ubrugt gendannelseskode, før du ændrer adgangskoden.',
        verifyPasswordChangeFailed:
          'Kunne ikke verificere adgangskoden og godkendelseskoden eller gendannelseskoden.',
        verifyAndEnable: 'Verificer og aktiver',
        scanTwoFactorQRCode:
          'Scan denne QR-kode med din godkendelsesapp, og indtast derefter koden.',
        showSetupKey: 'Vis opsætningsnøgle',
        hideSetupKey: 'Skjul opsætningsnøgle',
        setupKey: 'Opsætningsnøgle',
        twoFactorSetupFailed: 'Kunne ikke starte opsætning af tofaktorgodkendelse.',
        twoFactorVerifyFailed: 'Kunne ikke verificere godkendelseskoden.',
        twoFactorDisableFailed: 'Kunne ikke deaktivere tofaktorgodkendelse.',
        disableTwoFactorDesc:
          'Indtast din nuværende adgangskode og enten en godkendelseskode eller en ubrugt gendannelseskode. Deaktivering af tofaktorgodkendelse ugyldiggør alle gendannelseskoder.',
        updateBtn: 'Skift',
        logoutBtn: 'Log ud',
        logoutDesc: 'Er du sikker på, at du vil logge ud?',
        okBtn: 'Ja',
        cancelBtn: 'Annuller',
        twoFactorProtection: {
          title: 'Avanceret 2FA-beskyttelse',
          description:
            'Vælg de følsomme handlinger, der kræver en ekstra faktor, når beskyttelsen håndhæves. Kontakten "Direkte web-login" gør det modsatte: den tillader login med kun en godkendelseskode eller ubrugt gendannelseskode.',
          login: 'Direkte web-login',
          webTerminal: 'Webterminal',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API-nøgle',
          targetPower: 'Strøm til målcomputeren',
          nanoKVMRestart: 'Genstart af NanoKVM',
          deleteImage: 'Slet image',
          scripts: 'Scripthandlinger',
          screenTimelapse: 'Skærm-timelapse',
          loginScope:
            'Gør det muligt at logge ind på webgrænsefladen kun med en godkendelseskode eller en ubrugt gendannelseskode i stedet for brugernavn og adgangskode. Dette reducerer loginsikkerheden.',
          webTerminalScope: 'Åbning af Webterminal.',
          sshScope: 'Ændring af SSH-tjenestens tilstand.',
          tailscaleScope: 'Installation, tilslutning eller ændring af Tailscale.',
          frp: 'FRP',
          frpScope: 'Installation, tilslutning eller ændring af FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Installation, tilslutning eller ændring af Cloudflare Tunnel.',
          zerotierScope: 'Installation, tilslutning eller ændring af ZeroTier.',
          netbirdScope: 'Installation, tilslutning eller ændring af NetBird.',
          mcpScope:
            'Visning, kopiering eller gendannelse af MCP API-nøglen samt aktivering eller deaktivering af MCP.',
          targetPowerScope:
            'Slukning eller genstart af målcomputeren med den fysiske tænd/sluk-knap.',
          nanoKVMRestartScope: 'Genstart af selve NanoKVM.',
          deleteImageScope: 'Slet et gemt ISO- eller IMG-image permanent.',
          scriptsScope: 'Kørsel, upload eller sletning af scripts.',
          screenTimelapseScope:
            'Aktivering af skærm-timelapseoptagelse, gennemgang af optagelser og sletning af optagelser.',
          authorizeTitle: 'Godkend beskyttelsesændringer',
          authorizeDesc:
            'Indtast en godkendelseskode eller ubrugt gendannelseskode. Godkendelsen varer fem minutter.',
          authorizeFailed: 'Godkendelseskoden eller gendannelseskoden kunne ikke bekræftes.',
          loadFailed: 'Kunne ikke indlæse beskyttelsesindstillinger.',
          updateFailed: 'Kunne ikke gemme beskyttelsesindstillinger.',
          retry: 'Prøv igen',
          disableScriptsWarningTitle: 'Slå scriptbeskyttelse fra?',
          disableScriptsWarningDesc:
            'Uden beskyttelse kan et ondsindet script få udvidet adgang til denne enhed og omgå eller fjerne 2FA-beskyttelse.',
          cancelDisabling: 'Annuller',
          continueDisabling: 'Fortsæt med at slå fra',
          continueDisablingCountdown: 'Fortsæt med at slå fra ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Vejledning til tofaktorgodkendelse',
          scope: 'Tofaktorgodkendelse tilføjer et ekstra sikkerhedslag til din konto.',
          whatTitle: 'Hvad er 2FA?',
          whatDesc:
            'Tofaktorgodkendelse (2FA) beskytter din konto med en tidsbaseret kode fra en godkendelsesapp. Som standard er koden påkrævet ud over din adgangskode, så selv hvis din adgangskode kompromitteres, er din konto stadig beskyttet. Hvis direkte web-login er aktiveret, logger koden alene dig ind.',
          appsTitle: 'Anbefalede apps',
          appsDesc:
            'Du kan bruge Google Authenticator eller Microsoft Authenticator. Begge er gratis og fungerer på iOS og Android.',
          setupTitle: 'Sådan konfigureres det',
          setupDesc:
            'Åbn din godkendelsesapp, scan QR-koden, der vises under opsætningen, eller indtast opsætningsnøglen manuelt. Indtast derefter den 6-cifrede kode fra appen for at aktivere 2FA.',
          recoveryTitle: 'Gendannelseskoder',
          recoveryDesc:
            'Gendannelseskoder er engangskoder til backup. Hvis du mister adgang til din godkendelsesapp, kan du bruge en gendannelseskode til at logge ind. Gem dem et sikkert sted straks efter oprettelsen.',
          logoutNotice:
            'Aktivering af 2FA logger alle aktive sessioner ud, inklusive denne. Du skal logge ind igen på hver enhed.'
        }
      }
    },
    error: {
      title: 'Vi er stødt på et problem',
      refresh: 'Opdater'
    },
    pictureInPicture: {
      title: 'Billede-i-billede',
      enter: 'Aktivér billede-i-billede',
      exit: 'Luk billede-i-billede',
      error: 'Kunne ikke starte billede-i-billede. Vent til videoen er indlæst, og prøv igen.'
    },
    fullscreen: {
      toggle: 'Skift fuldskærm'
    },
    menu: {
      collapse: 'Skjul menu',
      expand: 'Udvid menu'
    }
  }
};

export default da;
