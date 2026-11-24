const nb = {
  translation: {
    storage: {
      cards: { mount: 'Diskmontering', share: 'Fildeling' },
      title: 'Diskbilder og fildeling',
      empty: 'Ingen bilder ennå',
      notAttached: 'Ingenting montert',
      attached: 'Montert: {{name}}',
      eject: 'Løs ut',
      mount: 'Monter',
      mountMode: 'Monteringsmodus',
      mountModeDisk: 'Masselagring',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Åpne filbehandler',
      refresh: 'Oppdater',
      delete: 'Slett',
      deleteTitle: 'Slett medium',
      deleteConfirm: 'Vil du slette {{name}}? Filene i den går tapt.',
      downloadImage: 'Last ned bildet',
      typePartition: 'Delte filer',
      volumeLabel: 'Flyttbar disk',
      groups: {
        share: 'Fildeling',
        images: 'Diskbilder'
      },
      failed: 'Handlingen mislyktes',
      hidOnly: 'Kun HID-modus: den virtuelle disken er utilgjengelig',
      disabled: 'Den virtuelle disken er slått av (Innstillinger - Enhet)',
      force: {
        title: 'Ta mediet tilbake med makt?',
        desc: 'Målmaskinen bruker fortsatt dette mediet. Å ta det tilbake registrerer USB-enheten på nytt og kan miste alt den ennå ikke har skrevet til disken.',
        menuHint:
          'Målmaskinen bruker dette mediet. Åpne filbehandleren for å ta det tilbake med makt.'
      },
      tips: {
        title: 'Hvordan laste opp',
        usb1: 'Koble til NanoKVM-enheten til din datamaskin med USB.',
        usb2: 'Monter de delte filene i masselagringsmodus på denne siden.',
        usb3: 'Åpne den virtuelle disken på datamaskinen din og kopier arkivfilen til rot-mappen på den virtuelle disken.',
        usb4: 'Løs ut disken når kopieringen er ferdig: bildet vises da i listen.',
        scp1: 'Sikre at NanoKVM-enheten og datamaskinen din er tilkoblet det samme lokale nettverket.',
        scp2: 'Åpne en terminal på datamaskinen din og bruk SCP-kommandoen til å laste opp arkivfilen til mappen /data på NanoKVM-enheten.',
        scp3: 'Eksempel: scp sti-til-din-arkivfil root@din-nanokvm-ip:/data'
      },
      job: {
        running: 'Arbeider med {{name}}'
      },
      enableDisk: {
        title: 'Slå på den virtuelle disken?',
        desc: 'For å montere et medium trengs den virtuelle disken, som er slått av nå. Slå den på? Tastaturet og musen til målmaskinen kobles fra et øyeblikk.',
        menuHint: 'Den virtuelle disken er slått av. Slå den på under Innstillinger - Enhet.'
      },
      add: {
        title: 'Legg til medium',
        upload: 'Last opp en ISO eller IMG',
        fromUrl: 'Last ned fra en URL',
        url: 'URL til bildet',
        urlRequired: 'Skriv inn en URL',
        onlyImages: 'Bare .iso- og .img-filer kan monteres',
        dropHint: 'Dra en ISO eller IMG hit, eller klikk for å velge en fil',
        cdromOnlyIso: 'En CD-ROM kan bare inneholde et ISO 9660 / UDF-bilde',
        uploadFailed: 'Opplasting mislyktes',
        downloadFailed: 'Nedlasting mislyktes',
        uploading: 'Laster opp {{name}}',
        downloading: 'Laster ned {{name}}',
        done: '{{name}} er klar',
        checksumFailed: 'Bildet samsvarer ikke med SHA-256-kontrollsummen.',
        blocked:
          'Målmaskinen bruker disken. Løs ut den virtuelle USB-disken på målmaskinen på en trygg måte, og slå deretter av Virtuell disk-bryteren under Innstillinger > Enhet.'
      },
      files: {
        reasonNotInitialized:
          'Den virtuelle USB-disken er ikke initialisert. Initialiser den under Innstillinger > Enhet før du bruker fildeling.',
        selectMedium: 'Velg et medium for å se filene på det.',
        reasonAttached: 'Dette mediet er koblet til målmaskinen.',
        reasonAttachedHint:
          'Løs ut den virtuelle USB-disken på målmaskinen på en trygg måte, og slå deretter av Virtuell disk-bryteren under Innstillinger > Enhet.',
        reasonNotBrowsable:
          'Dette mediet inneholder et filsystem som filbehandleren ikke kan åpne.',
        reasonNoLoop: 'Denne enheten kan ikke montere diskbilder (ingen loop-støtte).',
        reasonISOImage:
          'Et ISO 9660-bilde kan ikke utforskes her. Det kan fortsatt monteres på målmaskinen.',
        empty: 'Denne mappen er tom',
        noMatch: 'Ingen filer samsvarer med søket.',
        free: '{{size}} ledige',
        search: 'Søk i denne mappen',
        sort: {
          modified: 'Endret',
          name: 'Navn',
          size: 'Størrelse',
          type: 'Type',
          asc: 'Stigende',
          desc: 'Synkende'
        },
        view: {
          list: 'Listevisning',
          grid: 'Ikonvisning'
        },
        folder: 'Mappe',
        itemCount: 'Antall elementer: {{count}}',
        itemCount_one: '{{count}} element',
        itemCount_other: '{{count}} elementer',
        dropHint: 'Slipp filer her for å laste dem opp til denne mappen',
        up: 'Gå til overordnet mappe',
        uploadButton: 'Last opp',
        newFolder: 'Ny mappe',
        newFolderPlaceholder: 'Mappenavn',
        create: 'Opprett',
        upload: 'Last opp en fil til mediet',
        deleteFolderTitle: 'Slett mappe',
        deleteFolderConfirm: 'Slette mappen {{name}} og alt innholdet? Dette kan ikke angres.',
        uploading: 'Laster opp…',
        uploadFailed: 'Opplasting mislyktes',
        download: 'Last ned',
        delete: 'Slett',
        deleteTitle: 'Slett fil',
        deleteConfirm: 'Vil du slette {{name}} fra mediet?'
      },
      yes: 'Ja',
      no: 'Nei'
    },
    head: {
      storage: 'Diskbilder og fildeling',
      desktop: 'Eksternt skrivebord',
      login: 'Logg inn',
      changePassword: 'Endre passord',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Kunne ikke kopiere til utklippstavlen. Gi nettleseren tilgang til utklippstavlen og prøv igjen.'
    },
    auth: {
      login: 'Logg inn',
      language: 'Språk',
      placeholderUsername: 'Brukernavn',
      placeholderPassword: 'Passord',
      placeholderPassword2: 'Oppgi passord igjen',
      noEmptyUsername: 'Brukernavn påkrevd',
      noEmptyPassword: 'Passord påkrevd',
      noAccount:
        'Kunne ikke hente brukerinformasjon. Vennligst last inn siden på nytt eller gjenopprett passord',
      invalidUser: 'Ugyldig brukernavn eller passord',
      error: 'Uventet feil',
      twoFactorTitle: 'Verifisering kreves',
      twoFactorDesc:
        'Skriv inn autentiseringskoden eller den ubrukte gjenopprettingskoden for å logge inn.',
      sensitiveActionDesc:
        'Skriv inn en autentiseringskode eller ubrukt gjenopprettingskode for å fortsette.',
      placeholderAuthCode: 'Autentiseringskode',
      noEmptyAuthCode: 'Autentiseringskode kreves',
      verify: 'Verifiser',
      backToLogin: 'Tilbake til innlogging',
      verificationFailed: 'Verifisering mislyktes',
      twoFactorEnabledLogout: {
        message:
          'Tofaktorautentisering er aktivert. Av sikkerhetshensyn er alle økter logget ut. Logg inn på nytt.'
      },
      rateLimited: 'For mange mislykkede forsøk. Prøv igjen om {{seconds}} s.',
      recoveryCodeUsed:
        'Gjenopprettingskoden som ble brukt for denne innloggingen, er ugyldiggjort.',
      changePassword: 'Endre passord',
      changePasswordDesc:
        'For sikkerheten til enheten, vennligst endre passordet ditt for web-innlogging.',
      passwordVerificationRequired:
        'Gå tilbake til kontosiden i innstillingene og verifiser denne passordendringen på nytt.',
      passwordVerificationExpired:
        'Verifiseringen av passordendringen er utløpt. Gå tilbake til kontosiden i innstillingene og verifiser på nytt.',
      returnToSettings: 'Tilbake til innstillinger',
      differentPassword: 'Passordene er ikke like',
      illegalUsername: 'Brukernavn inneholder tegn som ikke er tillat',
      illegalPassword: 'Passord inneholder tegn som ikke er tillat',
      forgetPassword: 'Glemt passord',
      ok: 'OK',
      cancel: 'Avbryt',
      savePassword: 'Lagre passord',
      loginButtonText: 'Logg inn',
      tips: {
        reset1: 'Nettstandardkonto:',
        reset2: 'SSH standardkonto:',
        reset3: 'For å tilbakestille passordene, se ',
        changeReset: 'For å tilbakestille passordet for webinnlogging, se ',
        resetLink: 'Tilbakestill'
      }
    },
    wifi: {
      errors: {
        connect:
          'Kunne ikke koble til Wi-Fi. Kontroller nettverksnavn, passord og enhetens Wi-Fi-status, og prøv igjen.'
      },
      title: 'Wi-Fi',
      description: 'Konfigurer Wi-Fi for NanoKVM',
      submitted: {
        title: 'Et øyeblikk — NanoKVM kobler seg til Wi-Fi-nettverket ditt',
        tipsTitle: 'Hva skjer nå?',
        tipSwitch: 'Du kan nå bytte telefonen tilbake til ditt vanlige Wi-Fi.',
        tipCheck:
          'Om et lite øyeblikk finner du den nye IP-adressen på NanoKVM-skjermen. Åpne den i nettleseren din.',
        tipRetry:
          'Hvis ingenting skjer etter en stund, sjekk nettverksnavnet og passordet og prøv igjen.'
      },
      invalidMode:
        'Gjeldende modus støtter ikke nettverksoppsett. Gå til enheten din og aktiver Wi-Fi konfigurasjonsmodus.',
      confirmBtn: 'OK',
      finishBtn: 'Ferdig',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Passord',
      steps: {
        settings: 'Innstillinger',
        wifi: 'Wi-Fi',
        enable: 'Aktiver'
      },
      scan: {
        title: 'Velg et Wi-Fi-nettverk',
        description: 'Velg et nettverk i nærheten for å fortsette.',
        loading: 'Søker etter Wi-Fi-nettverk i nærheten…',
        empty: 'Fant ingen Wi-Fi-nettverk i nærheten.',
        failed: 'Kunne ikke søke. Skriv inn nettverksdetaljene manuelt.',
        retryBtn: 'Søk på nytt',
        manualBtn: 'Skriv inn nettverk manuelt',
        backBtn: 'Tilbake til nettverk i nærheten',
        connectTitle: 'Koble til {{ssid}}',
        connecting: 'Kobler til…',
        viewSimple: 'Bytt til enkel visning',
        viewDetailed: 'Bytt til detaljert visning'
      },
      ap: {
        authTitle: 'Autentisering kreves',
        authDescription: 'Vennligst skriv inn AP passordet for å fortsette',
        authFailed: 'Ugyldig AP passord',
        passPlaceholder: 'AP passord',
        verifyBtn: 'Bekreft'
      },
      qr: {
        cameraBtn: 'Ta bilde',
        pickBtn: 'Velg skjermbilde',
        scanHint:
          'Ta et bilde av en Wi-Fi QR-kode, eller velg et skjermbilde av en, for eksempel den en telefon viser når den deler Wi-Fi-en sin.',
        decoding: 'Leser QR-kode…',
        notFound:
          'Fant ingen QR-kode i dette bildet. Sørg for at koden er sentrert og i fokus, og prøv igjen.',
        invalid: 'Denne QR-koden inneholder ikke brukbare Wi-Fi-detaljer.',
        pickerUnavailable:
          'Åpnet ingenting? Dette innloggingsvinduet kan blokkere kameraet og bilder. Åpne heller denne adressen i telefonens nettleser:',
        confirmTitle: 'Koble NanoKVM til {{ssid}}?',
        confirmDescription:
          'Disse nettverksdetaljene kommer fra en QR-kode. Sjekk dem før du kobler til.',
        openNetwork: 'Dette nettverket har ikke passord.',
        connectBtn: 'Koble til',
        otherBtn: 'Velg et annet nettverk'
      }
    },
    notification: {
      browser: {
        title: 'Nettleseranbefaling',
        description:
          'For den beste brukeropplevelsen anbefaler vi å bruke Chrome-nettleseren. Selv om vi streber etter kompatibilitet på tvers av alle større nettlesere, kan det hende at enkelte avanserte funksjoner ikke fungerer etter hensikten utenfor Chrome.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Skjerm',
      video: 'Videomodus',
      quality: 'Kvalitet',
      qualityAuto: 'Automatisk',
      qualityLossless: 'Tapsfri',
      qualityHigh: 'Høy',
      qualityMedium: 'Middels',
      qualityLow: 'Lav',
      scale: 'Skala',
      noSignal: 'Ingen videosignal',
      captureDisabled:
        'Videoopptak er slått av. Aktiver det under «Innstillinger > Enhet > Videoopptak».',
      inconsistentVideoMode: 'Avspillingen er satt på pause. En annen videomodus spilles av.',
      bandwidthWarning:
        'Nettverksbåndbredden kan være utilstrekkelig for gjeldende videoinnstillinger. Prøv å redusere bithastigheten eller bildefrekvensen.',
      webrtcConnectionFailed:
        'WebRTC-tilkoblingen kunne ikke opprettes. Bytt til Direct-videomodus.',
      webrtcUnavailable: {
        title: 'WebRTC er utilgjengelig',
        description: 'Aktivér WebRTC i nettleseren, eller bytt til Direct-videomodus.'
      },
      upload: 'Last opp',
      advanced: 'Avansert',
      loading: 'Laster'
    },
    volume: {
      title: 'Volum'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon ikke montert',
      tip1: 'Den virtuelle mikrofonen er ikke montert på den eksterne verten. Vil du montere og aktivere det nå?',
      tip2: 'Du kan også administrere dette senere i "Innstillinger > Enhet > Virtuell lyd".',
      confirm: 'Aktiver',
      cancel: 'Avbryt',
      noDevice: 'Ingen mikrofon ble funnet på denne enheten. Koble til en mikrofon og prøv igjen.',
      permissionDenied:
        'Tilgang til mikrofonen ble avvist. Tillat mikrofontilgang i nettleserinnstillingene.',
      notReadable: 'Mikrofonen brukes av et annet program eller kan ikke leses.',
      overconstrained: 'Mikrofonen støtter ikke de nødvendige lydinnstillingene.',
      securityBlocked:
        'Tilgang til mikrofonen er blokkert på denne siden. Kontroller sikkerhetsinnstillingene i nettleseren.',
      aborted: 'Oppstart av mikrofonen ble avbrutt. Prøv igjen.',
      unsupported: 'Denne nettleseren eller tilkoblingsmetoden støtter ikke mikrofonopptak.',
      enableFailed: 'Mikrofonen kunne ikke aktiveres. Prøv igjen.',
      disconnected: 'Mikrofonen ble koblet fra, og mikrofonoverføringen er deaktivert.'
    },
    power: {
      title: 'På-knapp',
      showConfirm: 'Bekreftelse',
      showConfirmTip: 'Strømhandlinger krever en ekstra bekreftelse',
      powerShort: 'På-knapp (kort trykk)',
      powerLong: 'På-knapp (langt trykk)',
      powerConfirm: 'Utfør strømhandling?',
      okBtn: 'Ja',
      cancelBtn: 'Nei'
    },
    keyboard: {
      errors: {
        paste:
          'Kunne ikke sende teksten til den eksterne enheten. Kontroller KVM-tilkoblingen og prøv igjen.'
      },
      title: 'Åpne tastatur',
      paste: 'Lim inn',
      tips: 'Kun vanlige tegn på tastatur er støttet',
      placeholder: 'Vennligst angi teksten du vil lime inn',
      submit: 'Lim inn',
      virtual: 'Åpne tastatur',
      swapCommandCtrl: 'Bytt Cmd og Ctrl',
      shortcut: {
        title: 'Snarveier',
        custom: 'Egendefinert',
        capture: 'Klikk her for å ta en snarvei',
        clear: 'Tøm',
        save: 'Lagre',
        captureTips:
          'Registrering av systemtaster (som Windows-tasten) krever fullskjermtillatelse.',
        enterFullScreen: 'Veksle fullskjermmodus.'
      },
      leaderKey: {
        title: 'Leader-tast',
        desc: 'Omgå nettleserrestriksjoner og send systemsnarveier direkte til den eksterne verten.',
        howToUse: 'Hvordan bruke',
        simultaneous: {
          title: 'Samtidig modus',
          desc1: 'Hold Leader-tasten inne, og trykk deretter på snarveien.',
          desc2: 'Intuitivt, men kan komme i konflikt med systemsnarveier.'
        },
        sequential: {
          title: 'Sekvensiell modus',
          desc1:
            'Trykk på Leader-tasten → trykk på snarveien i rekkefølge → trykk på Leader-tasten igjen.',
          desc2: 'Krever flere trinn, men unngår fullstendig systemkonflikter.'
        },
        enable: 'Aktiver Leader-tast',
        tip: 'Når denne tasten er tilordnet som Leader-tast, fungerer den bare som snarveisutløser og mister standardoppførselen.',
        placeholder: 'Trykk på Leader-tasten',
        shiftRight: 'Høyre Shift',
        ctrlRight: 'Høyre Ctrl',
        metaRight: 'Høyre Win',
        submit: 'Lim inn',
        recorder: {
          rec: 'REC',
          activate: 'Aktiver taster',
          input: 'Trykk snarveien...'
        }
      }
    },
    mouse: {
      title: 'Mus',
      mode: 'Modus',
      absolute: 'Følgemus',
      absoluteAndroid: 'Følgemus (bedre Android-kompatibilitet)',
      touchScreen: 'Flerberøringsskjerm',
      relative: 'Eksklusiv mus',
      direction: 'Rullehjulretning',
      scrollUp: 'Rull opp',
      scrollDown: 'Rull ned',
      speed: 'Rullehjulhastighet',
      fast: 'Rask',
      slow: 'Sakte',
      requestPointer: 'Bruker relativ modus. Vennligsk klikk på skrivebordet for vise musepeker.',
      repairIphoneDrag: 'Reparer iPhone-draing',
      resetHid: 'Tilbakestill HID',
      touchpadGuide: {
        title: 'Touchpad-veiledning',
        scope: 'Gjelder vanlig pekerinndata når musemodus er Relativ.',
        swipeTitle: 'Sveip for å flytte',
        swipeDesc: 'Sveip i det aktive skjermområdet for å flytte fjernpekeren.',
        tapTitle: 'Trykk for å klikke',
        tapDesc: 'Et kort trykk sender ett venstreklikk.',
        holdTitle: 'Hold for å holde venstre knapp nede',
        holdDesc:
          'Trykk og hold fingeren rolig i omtrent 1 sekund for å holde venstre museknapp nede.',
        dragTitle: 'Flytt etter hold for å dra',
        dragDesc: 'Når hold er aktivt, flytter du fingeren for å dra med venstre knapp nede.'
      },
      hidOnly: {
        errors: {
          load: 'Kunne ikke laste inn statusen for HID-Only-modus. Prøv igjen om et øyeblikk.',
          update:
            'Kunne ikke bekrefte om HID-Only-modus ble oppdatert. Åpne denne innstillingen på nytt for å kontrollere gjeldende status, og prøv igjen.'
        },
        title: 'Kun HID-modus',
        desc: 'Hvis musen og tastaturet slutter å svare og tilbakestilling av HID ikke hjelper, kan det være et kompatibilitetsproblem mellom NanoKVM og enheten. Prøv å aktivere HID-Only-modus for bedre kompatibilitet.',
        tip1: 'Aktivering av HID-Only-modus vil demontere den virtuelle U-disken og det virtuelle nettverket',
        tip2: 'I HID-Only-modus er bildemontering deaktivert',
        enable: 'Aktiver HID-Only-modus',
        disable: 'Deaktiver HID-bare-modus'
      }
    },
    preview: {
      title: 'Forhåndsvisning'
    },
    image: {
      storageReadOnly: 'Det kan ikke skrives til fillageret. Slå av den virtuelle disken.',
      storageBusy:
        'Lageret er opptatt. Fullfør eller avbryt filoverføringen før du bytter tilstand for den virtuelle disken.',
      title: 'Bilder',
      loading: 'Laster...',
      empty: 'Ingen funnet',
      upload: 'Last opp bilde',
      dropFile: 'Slipp et diskbilde her, eller klikk for å velge en fil',
      uploadAction: 'Last opp',
      cancelAction: 'Avbryt',
      mount: 'Monter bilde',
      mounted: 'Montert',
      uploadError: 'Opplasting mislyktes, prøv igjen',
      uploadInsufficientStorage: 'Ikke nok lagringsplass',
      uploadTooLarge: 'Imaget overskrider størrelsesgrensen',
      cancelUploadError: 'Kunne ikke avbryte opplastingen',
      mountMode: 'Monteringsmodus',
      massStorage: 'Masselagring',
      cdRom: 'CD ROM',
      readOnly: 'Skrivebeskyttet',
      mountFailed: 'Montering feilet',
      mountDesc:
        'På noen systemer er det nødvendig å koble fra den virtuelle disken på den eksterne verten før man kan montere arkivfilen.',
      unmountFailed: 'Avmontering feilet',
      unmountDesc:
        'I noen systemer må du manuelt kaste ut på den eksterne verten før du avmonterer bildet.',
      refresh: 'Oppdater bildelisten',
      download: 'Last ned bilde',
      attention: 'Merknad',
      deleteConfirm: 'Er du sikker på at du vil slette dette bildet?',
      deleteFailed: 'Kunne ikke slette bildet',
      okBtn: 'Ja',
      cancelBtn: 'Nei'
    },
    script: {
      title: 'Skript',
      upload: 'Last opp',
      run: 'Kjør',
      runBackground: 'Kjør i bakgrunnen',
      runFailed: 'Kjøring feilet',
      uploadFailed: 'Kunne ikke laste opp skriptet',
      deleteFailed: 'Kunne ikke slette skriptet',
      attention: 'Merknad',
      delDesc: 'Er du sikker på at du vil slette denne filen?',
      confirm: 'Ja',
      cancel: 'Nei',
      delete: 'Slett',
      close: 'Lukk'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM',
      confirm: 'OK',
      submit: 'Send',
      username: 'Brukernavn',
      password: 'Passord',
      authCode: 'Autentiseringskode',
      authorizeTitle: 'Autoriser nettterminal',
      codeAccountHint: 'Skriv inn en autentiseringskode eller ubrukt gjenopprettingskode.',
      authFailed: 'Autentisering mislyktes'
    },
    wol: {
      errors: {
        send: 'Kunne ikke sende oppvåkningskommandoen. Kontroller MAC-adressen og nettverkstilkoblingen, og prøv igjen.',
        loadFailed:
          'Kunne ikke laste inn Wake-on-LAN-enhetslisten. Kontroller at NanoKVM er på nett, og prøv igjen.',
        renameFailed:
          'Kunne ikke gi denne enheten nytt navn. Kontroller MAC-adressen, og prøv igjen.',
        deleteFailed: 'Kunne ikke slette denne enheten. Prøv igjen om et øyeblikk.'
      },
      title: 'Wake-on-LAN',
      sending: 'Sender kommando...',
      sent: 'Kommando sendt',
      input: 'Vennligst angi MAC-adressen',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (valgfritt)',
      sha256Placeholder: 'Skriv inn en SHA-256-kontrollsum på 64 tegn',
      invalidSHA256: 'SHA-256 må være en heksadesimal streng på 64 tegn',
      title: 'Bildedaster',
      input: 'Vennligst skriv inn et eksternt bilde URL',
      ok: 'OK',
      disabled: '/data partisjonen er RO, så vi kan ikke laste ned bildet',
      downloadingFile: 'Laster ned: {{file}}',
      downloadingProgress: 'Laster ned ({{percentage}}): {{file}}',
      failed: 'Mislyktes',
      insufficientStorage: 'Ikke nok lagringsplass',
      remaining: '{{time}} gjenstår',
      cancel: 'Avbryt',
      cancelFailed: 'Kunne ikke avbryte nedlastingen',
      pause: 'Sett på pause',
      resume: 'Fortsett',
      pauseFailed: 'Kunne ikke sette nedlastingen på pause',
      resumeFailed: 'Kunne ikke fortsette nedlastingen'
    },
    settings: {
      back: 'Tilbake',
      close: 'Lukk',
      ai: {
        errors: {
          loadServices:
            'Kunne ikke laste inn statusen for AI-tjenestene. Kontroller at NanoKVM er på nett, og prøv igjen.',
          loadMcpConfig:
            'Kunne ikke laste inn konfigurasjonen for MCP-tjenesten. Prøv igjen om et øyeblikk.',
          updateMcpService:
            'Kunne ikke oppdatere statusen for MCP-tjenesten. Kontroller gjeldende status før du prøver igjen.',
          accessMcpApiKey:
            'Kunne ikke hente MCP API-nøkkelen. Prøv igjen og fullfør verifiseringen hvis du blir bedt om det.',
          regenerateMcpApiKey:
            'Kunne ikke generere MCP API-nøkkelen på nytt. Prøv igjen om et øyeblikk.',
          regeneratedMcpApiKeyUnavailable:
            'API-nøkkelen ble generert på nytt, men kunne ikke vises. Prøv å vise den igjen. Den forrige nøkkelen er ikke lenger gyldig.',
          loadMemoryFabricState:
            'Kunne ikke laste inn statusen for Memory Fabric-tjenesten. Prøv igjen om et øyeblikk.',
          updateMemoryFabricService:
            'Kunne ikke oppdatere statusen for Memory Fabric-tjenesten. Kontroller gjeldende status før du prøver igjen.',
          loadScreenTimelapseState:
            'Kunne ikke laste inn statusen for skjerm-timelapse-tjenesten. Prøv igjen om et øyeblikk.',
          updateScreenTimelapseService:
            'Kunne ikke oppdatere statusen for skjerm-timelapse-tjenesten. Kontroller gjeldende status før du prøver igjen.',
          openScreenTimelapse:
            'Kunne ikke åpne skjerm-timelapse-avspilling. Prøv igjen om et øyeblikk.',
          loadScreenTimelapseRecordings:
            'Kunne ikke laste inn skjerm-timelapse-opptak. Prøv igjen om et øyeblikk.',
          playScreenTimelapse:
            'Dette timelapse-opptaket kan ikke spilles av nå. Velg datoen på nytt eller prøv igjen senere.',
          screenTimelapseFullscreen:
            'Kunne ikke endre fullskjermmodus. Kontroller nettlesertillatelsene og prøv igjen.'
        },
        title: 'AI',
        mcpService: 'MCP-tjeneste',
        mcpServiceDesc:
          'Autentisert felles gateway for styring, medier og aktiverte Memory Fabric-verktøy',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Behandle agenters kontekstminne og OCR lokalt',
        memoryFabricRemoteRequiresMcp:
          'Aktiver MCP-tjenesten ovenfor for å gjøre minne- og OCR-verktøy tilgjengelige eksternt.',
        memoryFabricRemoteAvailable:
          'Minne- og OCR-verktøy er tilgjengelige via endepunktet /api/mcp ovenfor med samme API-nøkkel.',
        memoryFabricConfig: {
          title: 'Konfigurasjon',
          description: 'Administrer bare innstillingene for standardmodellen i Memory Fabric.',
          loadFailed: 'Kunne ikke laste konfigurasjonen for Memory Fabric.',
          saveFailed: 'Kunne ikke lagre konfigurasjonen for Memory Fabric.',
          saved: 'Konfigurasjonen er lagret.',
          savedRestarted: 'Konfigurasjonen er lagret, og Memory Fabric er startet på nytt.',
          noChanges: 'Ingen endringer i konfigurasjonen.',
          reload: 'Last inn på nytt',
          save: 'Lagre',
          unsaved: 'Ulagrede endringer',
          conflictTitle: 'Konfigurasjonen er endret',
          conflictDescription:
            'Konfigurasjonen ble endret et annet sted. Last inn den nyeste versjonen før du lagrer igjen.',
          conflictReload: 'Last inn nyeste versjon',
          discardTitle: 'Forkaste ulagrede endringer?',
          discardDescription:
            'Når du laster inn på nytt, erstattes gjeldende skjema med den sist lagrede konfigurasjonen.',
          discardConfirm: 'Forkast og last inn på nytt',
          sections: {
            models: 'Modeller'
          },
          openAICompatibleHint:
            'Støtter OpenAI-kompatible modellleverandører. Skriv inn modellnavnet, basis-URL-en og API-nøkkelen fra leverandøren.',
          fields: {
            model: 'Modell',
            baseUrl: 'Basis-URL',
            maxTokens: 'Maksimalt antall tokens',
            timeoutSeconds: 'Tidsavbrudd (sekunder)'
          },
          apiKey: {
            title: 'API-nøkkel',
            configured: 'Konfigurert',
            notConfigured: 'Ikke konfigurert',
            keep: 'Behold',
            replace: 'Erstatt',
            clear: 'Fjern',
            replacement: 'Ny API-nøkkel',
            replacementPlaceholder: 'Skriv inn den nye API-nøkkelen',
            clearHint:
              'Den lagrede API-nøkkelen fjernes; miljøvariabelen kan fortsatt levere en nøkkel.',
            required: 'Skriv inn en ny API-nøkkel.'
          },
          validation: {
            required: 'Dette feltet er obligatorisk.',
            invalidUrl: 'Skriv inn en gyldig HTTP- eller HTTPS-URL.'
          }
        },
        screenTimelapseService: 'Skjerm-timelapse',
        screenTimelapseServiceDesc:
          'Spill av Memory Fabric-skjermhistorikk og søk i fanget skjerminnhold',
        screenTimelapseSearchRequiresMemoryFabric:
          'For å søke i innspilt innhold må du også aktivere Memory Fabric.',
        screenTimelapseOpen: 'Åpne',
        screenTimelapseSearchPlaceholder: 'Søk i videoinnhold...',
        screenTimelapseNoVideos: 'Fant ingen timelapse-videoer',
        screenTimelapseNoResults: 'Fant ingen samsvarende bilder',
        screenTimelapseMultipleResults: 'Søkeresultater',
        screenTimelapseUnavailable: 'Ikke tilgjengelig',
        screenTimelapsePlayerUnsupported: 'Denne nettleseren kan ikke spille av timelapse-video',
        screenTimelapseSearchFailed: 'Søk mislyktes',
        screenTimelapseLoading: 'Laster timelapse...',
        screenTimelapseSearchAction: 'Søk i timelapse',
        screenTimelapsePlay: 'Spill av timelapse',
        screenTimelapsePause: 'Sett timelapse på pause',
        screenTimelapseEnterFullscreen: 'Gå til fullskjerm',
        screenTimelapseExitFullscreen: 'Avslutt fullskjerm',
        screenTimelapseFullscreenUnsupported: 'Denne nettleseren kan ikke gå til fullskjerm',
        screenTimelapsePreviousDay: 'Forrige dag',
        screenTimelapseNextDay: 'Neste dag',
        screenTimelapseDeleteDay: 'Slett opptaket for {{date}}',
        screenTimelapseDelete: 'Slett',
        screenTimelapseDeleteConfirmTitle: 'Slette tidsforløpopptaket?',
        screenTimelapseDeleteConfirmDesc:
          'Opptaket for {{date}} bruker {{size}}. Handlingen kan ikke angres.',
        screenTimelapseDeleteSuccess: 'Opptaket for {{date}} ble slettet',
        screenTimelapseDeleteFailed: 'Kunne ikke slette opptaket for {{date}}',
        endpoint: 'Endepunkt',
        apiKey: 'API-nøkkel',
        regenerate: 'Generer på nytt',
        copyApiKey: 'Kopier API-nøkkel',
        copyEndpoint: 'Kopier endepunkt',
        showApiKey: 'Vis API-nøkkel',
        regenerateConfirmTitle: 'Generer API-nøkkel på nytt',
        regenerateConfirmDesc:
          'Eksisterende MCP-klienter slutter å fungere til de er oppdatert med den nye nøkkelen. Vil du fortsette?',
        okBtn: 'Ja',
        cancelBtn: 'Nei'
      },
      title: 'Innstillinger',
      about: {
        title: 'Om NanoKVM',
        information: 'Informasjon',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applikasjonsversjon',
        applicationTip: 'NanoKVM webapplikasjonsversjon',
        arch: 'Arkitektur',
        device_number: 'Enhetsnummer',
        image: 'Arkivfil-versjon',
        imageTip: 'NanoKVM systemavbildningsversjon',
        deviceKey: 'Enhetsnøkkel',
        community: 'Fellesskap',
        hostname: 'Vertsnavn',
        hostnameUpdated: 'Vertsnavn oppdatert. Start på nytt for å søke.',
        ipType: {
          Wired: 'Kablet',
          Wireless: 'Trådløs',
          Other: 'Annet'
        }
      },
      appearance: {
        title: 'Utseende',
        display: 'Skjerm',
        language: 'Språk',
        languageDesc: 'Velg språket for grensesnittet',
        languageChangeFailed: 'Kunne ikke endre språk',
        webTitle: 'Netttittel',
        webTitleDesc: 'Tilpass nettsidetittelen',
        menuBar: {
          title: 'Menylinje',
          mode: 'Visningsmodus',
          modeDesc: 'Vis menylinje på skjermen',
          modeOff: 'Av',
          modeAuto: 'Skjul automatisk',
          modeAlways: 'Alltid synlig',
          icons: 'Undermenyikoner',
          showIcon: 'Vis ikon',
          activateOnConnect: 'Aktiver ved tilkobling',
          stopSharing: 'Stopp deling',
          iconsDesc: 'Vis undermenyikoner i menylinjen'
        }
      },
      screen: {
        title: 'Skjerm',
        videoMode: {
          title: 'Videomodus',
          description: 'Velg kodings- og overføringsmetode'
        },
        edid: {
          description: 'Velg skjermparametere',
          default: 'Standard',
          custom: 'Egendefinert',
          document: 'Dokument',
          monitorName: 'Skjermnavn',
          monitorNameDescription:
            'Maksimalt 13 utskrivbare ASCII-tegn; la stå tomt for å bruke det opprinnelige navnet fra EDID',
          monitorNamePlaceholder: 'Bruk opprinnelig EDID-navn',
          monitorNameApply: 'Bruk',
          monitorNameInvalid: 'Skriv inn maksimalt 13 utskrivbare ASCII-tegn',
          monitorNameUnsupported: 'Gjeldende EDID støtter ikke overstyring av skjermnavnet',
          monitorNameLoadFailed: 'Kunne ikke laste inn skjermnavnet',
          monitorNameUpdateFailed: 'Kunne ikke oppdatere skjermnavnet',
          powerConfirmTitle: 'Bekreft før EDID oppdateres',
          powerConfirmDescription:
            'Koble til PD-strøm først. Oppdatering av EDID kan koble fra skjermen kortvarig; hvis enheten ikke kan oppdage skjermen riktig etterpå, start den på nytt manuelt.',
          powerConfirmOk: 'Fortsett',
          powerConfirmOkCountdown: 'Fortsett ({{seconds}})',
          powerConfirmCancel: 'Avbryt',
          errors: {
            badRequest: 'Ugyldig EDID-forespørsel',
            invalidFileName: 'Velg en .bin-EDID-fil med et gyldig filnavn',
            fileTooLarge: 'EDID-filen er for stor',
            invalidLength: 'EDID-filen må være 128, 256, 384 eller 512 byte',
            invalidHeader: 'EDID-hodet er ugyldig',
            unsupportedVersion: 'Denne EDID-versjonen støttes ikke',
            extensionCountMismatch: 'Antallet EDID-utvidelsesblokker samsvarer ikke med fillengden',
            invalidChecksum: 'EDID-kontrollsummen er ugyldig',
            invalidCta: 'Strukturen i CTA-utvidelsen er ugyldig',
            currentCannotOverwrite:
              'EDID-en som er i bruk, kan ikke overskrives. Bytt til en annen EDID først',
            saveFailed: 'Kunne ikke lagre EDID-en',
            switchFailed: 'Kunne ikke bytte EDID',
            uploadFailed: 'Kunne ikke laste opp EDID-en',
            deleteFailed: 'Kunne ikke slette EDID-en',
            networkFailed: 'Nettverksfeil. Prøv på nytt'
          }
        },
        rateControlMode: {
          title: 'Hastighetskontrollmodus',
          description: 'Velg metoden for å kontrollere bithastigheten',
          cbr: 'CBR (Constant Bitrate): Prioriterer tilkoblingsstabilitet. Minimerer stamming, men bildekvaliteten kan falle under høybevegelsesscener.',
          vbr: 'VBR (Variable Bitrate): Prioriterer bildets klarhet. Høybevegelsesscener kan forårsake båndbreddespiker, som potensielt kan føre til latens eller hakking.'
        },
        bitrate: {
          title: 'Bitrate',
          description: 'Balanser bildeklarhet med båndbreddebruk',
          lower:
            'Lavere verdier reduserer ventetiden på dårlige nettverk, men kompromitterer bildekvaliteten, noe som potensielt kan forårsake uskarphet eller artefakter.',
          higher:
            'Høyere verdier forbedrer bildets klarhet, men bruker mer båndbredde. Kan øke ventetiden hvis tilkoblingen er treg.'
        },
        gop: {
          description: 'Angi intervallet mellom hele nøkkelbilder',
          lower:
            'Lavere verdier sender hele bilder oftere. Dette fremskynder gjenoppretting fra visuelle artefakter, men bruker mer båndbredde og reduserer bildedetaljer.',
          higher:
            'Høyere verdier optimaliserer for inkrementelle oppdateringer. Dette forbedrer bildedetaljer og sparer båndbredde, men gjenoppretting fra visuelle artefakter vil ta lengre tid.'
        },
        quality: {
          title: 'Kvalitet',
          description: 'Still inn skjermbildekvaliteten'
        },
        fps: {
          auto: 'Automatisk',
          description: 'Still inn skjermbildefrekvensen',
          tip1: 'Innstilling av verdien til 0 aktiverer automodus, og synkroniserer skjermen med den eksterne vertens bildefrekvens.',
          tip2: 'Redusering av bildefrekvensen reduserer båndbreddebruken betraktelig. Reduser den manuelt under nettverksustabilitet eller når høye bildefrekvenser ikke er nødvendig.'
        },
        scale: {
          title: 'Skala',
          description: 'Still inn displayets skaleringsforhold'
        }
      },
      device: {
        title: 'Enhet',
        virtualAudioConfirm: {
          title: 'Bekreft før du bytter virtuell lyd',
          description:
            'Koble til PD-strøm først. Når du bytter virtuell lyd, kan skjermen bli koblet fra et kort øyeblikk; hvis enheten ikke oppdager skjermen riktig etterpå, starter du den på nytt manuelt.',
          ok: 'Fortsett å bytte',
          okCountdown: 'Fortsett å bytte ({{seconds}})',
          cancel: 'Avbryt'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurer Wi-Fi',
          apMode: 'AP-modus er aktivert, koble til Wi-Fi manuelt',
          others: 'Andre...',
          connect: 'Bli med Wi-Fi',
          connectDesc1: 'Vennligst skriv inn nettverks-ssid og passord',
          connectDesc2: 'Vennligst skriv inn passordet for å bli med i dette nettverket',
          disconnect: 'Er du sikker på at du vil koble fra nettverket?',
          ssid: 'Navn',
          password: 'Passord',
          joinBtn: 'Bli med',
          confirmBtn: 'OK',
          cancelBtn: 'Avbryt',
          preset: {
            title: 'Forhåndslagret Wi-Fi',
            description: 'Forhåndslagre ett mål-Wi-Fi; enheten kobler til automatisk ved oppstart',
            empty: 'Ingen',
            addBtn: 'Legg til forhåndsinnstilling',
            removeBtn: 'Fjern forhåndsinnstilling',
            saveTitle: 'Forhåndslagre Wi-Fi',
            saveDesc:
              'Lagre målnettverket nå; enheten kobler til automatisk når det blir funnet ved oppstart',
            saveBtn: 'Lagre',
            passwordPlaceholder: 'Passord (la stå tomt for åpne nettverk)',
            saveFailed: 'Kunne ikke lagre Wi-Fi-forhåndsinnstillingen',
            removeFailed: 'Kunne ikke fjerne Wi-Fi-forhåndsinnstillingen',
            removeConfirm: 'Er du sikker på at du vil fjerne det forhåndslagrede Wi-Fi-nettverket?',
            warning:
              'Når den er lagret, vil enheten ved hver oppstart prøve å koble seg til dette Wi-Fi-nettverket når det oppdages, noe som kan forstyrre tilkobling til andre Wi-Fi-nettverk. Forhåndsinnstillingen gjelder til du endrer eller fjerner den.'
          }
        },
        capture: {
          title: 'Videoopptak',
          description: 'Aktiver videoopptak'
        },
        ssh: {
          description: 'Aktiver SSH ekstern tilgang',
          tip: 'Etter at du har aktivert SSH, må du angi et sterkt passord for root-brukeren i terminalen.'
        },
        swap: {
          errors: {
            load: 'Kunne ikke laste inn statusen for swap-minne. Prøv igjen om et øyeblikk.',
            update:
              'Kunne ikke oppdatere innstillingen for swap-minne. Kontroller gjeldende status før du prøver igjen.'
          },
          title: 'Swap-minne',
          description: 'Aktiver komprimert swap-minne'
        },
        advanced: 'Avanserte innstillinger',
        staticIp: {
          title: 'Statisk IP',
          config: 'Konfig',
          tip: 'For å forhindre tap av tilkobling, anbefales det å konfigurere mens du har fysisk tilgang til enheten.',
          document: 'Sjekk dokumentet.',
          enable: 'Aktiver statisk IP',
          placeholder: 'Format: IP/CIDR [Gateway]\nEksempel: 192.168.1.10/22',
          notEmpty: 'IP-adressen kan ikke være tom.',
          notValid: 'IP-adresseformatet er ugyldig.',
          failed: 'Konfigurasjonen mislyktes.',
          success: 'Konfigurasjonen er fullført. Vennligst sjekk om IP-adressen er tilgjengelig.',
          okBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        usbNetwork: {
          title: 'USB-nettverk',
          mode: 'Driftsmodus',
          server: 'Servermodus',
          serverDesc: 'Tildel den eksterne verten en IP-adresse via den innebygde DHCP-serveren.',
          client: 'Klientmodus',
          clientDesc: 'Hent en IP-adresse fra den eksterne verten via DHCP.',
          forwarding: 'IP-videresending',
          forwardingTip:
            'For at den eksterne verten skal få tilgang til internett via NanoKVM, må du aktivere IP-videresending.',
          manualIp: 'Manuell IP-adresse',
          ipv4Placeholder: 'IPv4/CIDR, f.eks. 10.10.10.1/24',
          invalidIpv4: 'Skriv inn en gyldig IPv4/CIDR-adresse.',
          loadFailed: 'Kan ikke laste USB-nettverkskonfigurasjonen.',
          failed: 'Konfigurasjonen mislyktes.',
          success: 'Konfigurasjonen er fullført.',
          applyBtn: 'Bruk',
          cancelBtn: 'Avbryt'
        },
        usbId: {
          title: 'USB-enhetsidentitet',
          config: 'Konfig',
          tip: 'Egendefinerte USB-ID-er endrer hvordan verten identifiserer denne enheten. Anvendelse starter USB-gadgeten på nytt og kan midlertidig koble fra tastatur, mus eller nettverk.',
          enable: 'Aktiver egendefinert USB-enhetsidentitet',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Enhetsversjon (bcdDevice)',
          serialnumber: 'Serienummer',
          manufacturer: 'Produsent',
          product: 'Produkt',
          vidPlaceholder: 'f.eks. 0x359F',
          pidPlaceholder: 'f.eks. 0x3305',
          bcdDevicePlaceholder: 'f.eks. 0x0101 (valgfritt)',
          serialnumberPlaceholder: 'f.eks. 0123456789ABCDEF (valgfritt)',
          manufacturerPlaceholder: 'f.eks. sipeed (valgfritt)',
          productPlaceholder: 'f.eks. NanoKVM-Go (valgfritt)',
          notEmpty: 'VID og PID kan ikke være tomme.',
          notValid: 'VID/PID-formatet er ugyldig. Bruk hex som 0x359F.',
          bcdNotValid: 'Ugyldig bcdDevice-format. Bruk hex som 0x0101, eller la stå tomt.',
          failed: 'Konfigurasjonen mislyktes.',
          success: 'Konfigurasjonen er fullført. USB-gadget startet på nytt.',
          okBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        diskInquiry: {
          title: 'USB-stasjonsnavn',
          config: 'Konfig',
          tip: 'Navnet på den virtuelle USB-pennen som den eksterne datamaskinen ser. Lagring starter USB på nytt og kan midlertidig koble fra tastatur, mus eller nettverk.',
          enable: 'Bruk et egendefinert USB-stasjonsnavn',
          placeholder: 'f.eks. sipeed  Flash Drive     1.00',
          notEmpty: 'Navnet kan ikke være tomt.',
          notValid: 'Ugyldig navn. Opptil 28 utskrivbare ASCII-tegn uten linjeskift.',
          failed: 'Konfigurasjonen mislyktes.',
          success: 'Konfigurasjonen er fullført. USB-gadget startet på nytt.',
          okBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        diskInquiryCdrom: {
          title: 'Virtuelt CD-stasjonsnavn',
          config: 'Konfig',
          tip: 'Navnet på den virtuelle CD/DVD-stasjonen på den eksterne datamaskinen. Lagring starter USB på nytt og kan midlertidig koble fra tastatur, mus eller nettverk.',
          enable: 'Bruk et egendefinert CD-stasjonsnavn',
          placeholder: 'f.eks. sipeed  Optical Drive   1.00',
          notEmpty: 'Navnet kan ikke være tomt.',
          notValid: 'Ugyldig navn. Opptil 28 utskrivbare ASCII-tegn uten linjeskift.',
          failed: 'Konfigurasjonen mislyktes.',
          success: 'Konfigurasjonen er fullført. USB-gadget startet på nytt.',
          okBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        uac2FunctionName: {
          title: 'USB-lydnavn',
          config: 'Konfig',
          tip: 'Navnet på den virtuelle USB-mikrofonen / lydenheten på den eksterne datamaskinen. Lagring starter USB på nytt og kan midlertidig koble fra tastatur, mus eller nettverk.',
          enable: 'Bruk et egendefinert USB-lydnavn',
          placeholder: 'f.eks. NanoKVM-Go Audio',
          notEmpty: 'Navnet kan ikke være tomt.',
          notValid: 'Ugyldig navn. Opptil 64 utskrivbare tegn uten linjeskift.',
          failed: 'Konfigurasjonen mislyktes.',
          success: 'Konfigurasjonen er fullført. USB-gadget startet på nytt.',
          okBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        lowPower: {
          title: 'Lavstrømmodus',
          description: 'Reduser strømbruken mens den er inaktiv',
          tips: 'Råd',
          tip1: 'Når dette er aktivert, settes videoopptaket på pause i perioder med inaktivitet for å redusere strømforbruk og varmeutvikling.',
          tip2: 'Gjenopptakelse av videoopptaket kan kreve ny initialisering, og videoen kan derfor være midlertidig utilgjengelig.',
          okBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        mouseJiggler: {
          title: 'Mus Jiggler',
          description: 'Hindre den eksterne verten fra å sove',
          disable: 'Deaktiver',
          absolute: 'Absolutt modus',
          relative: 'Relativ modus'
        },
        fingerRobot: {
          title: 'Fingerrobot',
          description: 'Aktiver styring av fingerroboten'
        },
        mdns: {
          description: 'Aktiver mDNS oppdagelsestjeneste'
        },
        hidOnly: 'HID-Bare modus',
        imageMounted: 'Bildemontert',
        disk: 'Virtuell disk',
        diskDesc: 'Monter virtuell U-disk på den eksterne verten',
        sdCard: {
          warning: 'Initialiser virtuell disk',
          tip1: 'Første gang funksjonen aktiveres, initialiseres den virtuelle disken, og eksisterende bildefiler og data på den virtuelle disken slettes. Handlingen kan ikke angres. Sikkerhetskopier viktige filer først.',
          tip2: 'Fortsette initialiseringen?',
          capacity: 'Kapasitet',
          disable: 'Deaktiver',
          disableWarning: 'Deaktivere virtuell disk?',
          disableTip:
            'Deaktivering sletter alle filer på den virtuelle disken og frigjør lagringsplassen på enheten som den bruker. Neste gang den aktiveres, må du velge kapasitet og initialisere den på nytt. Handlingen kan ikke angres.'
        },
        network: 'Virtuelt nettverk',
        networkDesc: 'Monter virtuelt nettverkskort på den eksterne verten',
        mic: 'Virtuell lyd',
        micDesc: 'Velg virtuell USB-lydmodus for den eksterne verten',
        virtualAudioMode: {
          off: 'Av',
          mic: 'Kun mikrofon',
          duplex: 'Mikrofon + høyttaler'
        },
        speaker: 'Virtuell høyttaler',
        speakerDesc:
          'Når den er aktivert, erstatter den standard DP-lydutgang for å støtte eksterne samtaler med bedre kompatibilitet. Når den er deaktivert, kan bare medielyd videresendes, ikke samtalelyd.',
        micTip:
          'På grunn av USB-endepunktsbegrensninger vil aktivering av den virtuelle mikrofonen deaktivere USB-sekundærskjermen.\nOg det anbefales å deaktivere denne funksjonen under BIOS-operasjoner for å sikre optimal kompatibilitet.',
        reboot: 'Start på nytt',
        rebootDesc: 'Er du sikker på at du vil starte NanoKVM på nytt?',
        rebootToFlashing: 'Start på nytt i flashmodus',
        rebootToFlashingButton: 'Start på nytt',
        rebootToFlashingTitle: 'Starte på nytt i flashmodus?',
        rebootToFlashingDesc:
          'NanoKVM starter på nytt og går inn i USB-flashmodus. Kontroller at USB-grensesnittet som brukes til flashing, er tilkoblet, og at flashverktøyet er klart.',
        datetime: {
          timezone: 'Tidssone',
          datetime: 'Datotid',
          dateAndTime: 'Dato og klokkeslett',
          synchronize: 'Synkroniser tid',
          lastSynchronization: 'Siste synkronisering: ',
          notSynchronized: 'Systemtiden er ikke synkronisert ennå',
          syncNow: 'Synkroniser nå',
          manualSet: 'Angi manuelt',
          manualSetTitle: 'Angi tiden manuelt',
          manualSetDescription: 'Du må logge inn på nytt etter at tiden er endret.',
          useBrowserTime: 'Bruk nettleserens tid',
          setTime: 'Angi tid',
          invalidTime: 'Angi en gyldig dato og et gyldig klokkeslett.',
          setFailed: 'Kan ikke angi systemtiden. Prøv på nytt.',
          setSucceeded: 'Systemtiden er oppdatert. Logg inn på nytt.'
        },
        dialBackground: {
          errors: {
            load: 'Kunne ikke laste inn bakgrunnsbilder for urskiven. Prøv igjen om et øyeblikk.',
            upload:
              'Kunne ikke laste opp bakgrunnsbildet. Kontroller at filen er et gyldig BMP-bilde og prøv igjen.',
            switch: 'Kunne ikke bytte bakgrunn for urskiven. Oppdater bildelisten og prøv igjen.'
          },
          title: 'Urskivebakgrunn',
          description: 'Velg et BMP-bilde fra /kvmcomm/ui/bg/ som urskivebakgrunn',
          upload: 'Last opp',
          loading: 'Laster...',
          empty: 'Fant ingen BMP-bilder',
          current: 'Gjeldende',
          onlyBmp: 'Bare BMP-bilder støttes',
          uploaded: 'Opplasting fullført',
          switched: 'Bakgrunn byttet'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nei'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Starte Tailscale på nytt?',
        stop: 'Stoppe Tailscale?',
        stopDesc: 'Logg ut av Tailscale og deaktiver automatisk oppstart ved boot.',
        loading: 'Laster...',
        notInstall: 'Tailscale er ikke funnet! Vennligst installer.',
        install: 'Installér',
        installing: 'Installerer',
        failed: 'Installering feilet',
        retry: 'Vennligst last inn siden på nytt og forsøk igjen eller installer manuelt',
        download: 'Last ned',
        package: 'installasjonspakken',
        unzip: 'og pakk den ut',
        upTailscale: 'Last opp Tailscale til NanoKVM-enhetens mappe /usr/bin/',
        upTailscaled: 'Last opp tailscaled til NanoKVM-enhetens mappe /usr/sbin/',
        refresh: 'Last inn denne siden på nytt',
        notRunning: 'Tailscale kjører ikke. Start den for å fortsette.',
        run: 'Start',
        notLogin:
          'Denne enheten er ikke knyttet til din konto enda. Vennligst logg inn og knytt den til kontoen din..',
        urlPeriod: 'Denne lenken er gyldig i 10 minutter',
        login: 'Logg inn',
        loginSuccess: 'Logget inn',
        enable: 'Skru på Tailscale',
        exitNode: 'Utgangsnode',
        exitNodeDesc: 'Annonser denne enheten som internettutgang for tailnettet ditt.',
        subnetRoute: 'Delnettrute',
        subnetRouteDesc:
          'Annonser det direkte tilkoblede LAN-delnettet {{subnet}} til tailnettet ditt.',
        subnetUnavailable: 'Ingen direkte tilkoblet LAN-delnett er tilgjengelig.',
        routingUnavailable: 'Tailscale-ruting er ikke tilgjengelig på denne enheten.',
        pendingApproval:
          'Etter aktivering må den annonserte ruten godkjennes i Tailscale-administrasjonskonsollen.',
        routingUpdateFailed: 'Kunne ikke oppdatere innstillingene for Tailscale-ruting.',
        deviceName: 'Enhetens navn',
        deviceIP: 'Enhetens IP',
        account: 'Konto',
        logout: 'Logg ut',
        logoutDesc: 'Er du sikker på at du vil logge ut?',
        uninstall: 'Avinstaller Tailscale',
        uninstallDesc: 'Er du sikker på at du vil avinstallere Tailscale?',
        okBtn: 'Ja',
        cancelBtn: 'Nei'
      },
      update: {
        title: 'Se etter oppdatering',
        queryFailed: 'Kunne ikke hente versjon',
        updateFailed: 'En feil oppstod under oppdatering. Vennligst forsøk igjen.',
        isLatest: 'Du har siste versjon allerede.',
        available: 'En oppdatering er tilgjengelig. Er du sikker på at du ønsker å oppdatere?',
        updating: 'Oppdatering har startet. Vennligst vent...',
        confirm: 'Oppdater',
        upgrade_tip_nano:
          'En ny versjon av NanoKVM er tilgjengelig. Er du sikker på at du vil oppdatere nå?',
        upgrade_tip_firmware:
          'En ny fastvareversjon er tilgjengelig. Er du sikker på at du vil oppdatere nå?',
        cancel: 'Avbryt',
        preview: 'Forhåndsvisningsoppdateringer',
        previewDesc: 'Få tidlig tilgang til nye funksjoner og forbedringer',
        previewTip:
          'Vær oppmerksom på at forhåndsvisningsutgivelser kan inneholde feil eller ufullstendig funksjonalitet!',
        download: 'Last ned',
        install: 'Installer',
        changelog: 'Endringslogg',
        restart: 'Start på nytt'
      },
      account: {
        title: 'Konto',
        webAccount: 'Webkontonavn',
        password: 'Passord',
        twoFactor: 'Tofaktorautentisering',
        enableTwoFactor: 'Aktiver',
        enableTwoFactorDesc:
          'Skriv inn gjeldende passord for å starte oppsett av tofaktorautentisering.',
        disableTwoFactor: 'Deaktiver',
        recoveryCodes: 'Gjenopprettingskoder',
        enableRecoveryCodes: 'Aktiver',
        updateRecoveryCodes: 'Oppdater',
        recoveryCodesTooltipDisabled:
          'Gjenopprettingskoder er reservelegitimasjon for tofaktorautentisering. Du kan generere dem etter at tofaktorautentisering er aktivert.',
        recoveryCodesTooltipEnabled:
          'Gjenopprettingskoder er bare gyldige mens tofaktorautentisering er aktivert. Deaktivering av tofaktorautentisering ugyldiggjør alle gjenopprettingskoder; mens den forblir aktivert, kan de bare oppdateres.',
        recoveryCodesRegenerateDesc:
          'Skriv inn gjeldende passord og enten en autentiseringskode eller en ubrukt gjenopprettingskode. Oppdatering av gjenopprettingskoder ugyldiggjør alle eksisterende gjenopprettingskoder.',
        recoveryCodesVerifyFailed:
          'Kunne ikke verifisere passordet og autentiseringskoden eller gjenopprettingskoden.',
        recoveryCodesSavedTitle: 'Lagre disse gjenopprettingskodene nå',
        recoveryCodesSavedDesc:
          'Gjenopprettingskoder kan bare vises én gang. Lagre dem på et sikkert sted før du forlater denne siden.',
        enableRecoveryCodesDesc:
          'Skriv inn gjeldende passord og en autentiseringskode for å generere gjenopprettingskoder. Gjenopprettingskoder kan bare vises én gang, så lagre dem før du forlater denne siden.',
        copyRecoveryCodes: 'Kopier',
        downloadRecoveryCodes: 'Last ned',
        recoveryCodesCopyFailed: 'Kunne ikke kopiere gjenopprettingskoder.',
        done: 'Ferdig',
        currentPassword: 'Gjeldende passord',
        authCode: 'Autentiseringskode',
        authCodeOrRecoveryCode: 'Autentiseringskode eller gjenopprettingskode',
        authRateLimited: 'For mange mislykkede forsøk. Prøv igjen om {{seconds}} s.',
        verifyPasswordChange: 'Verifiser passordendring',
        verifyPasswordChangeDesc:
          'Skriv inn gjeldende passord og enten en autentiseringskode eller en ubrukt gjenopprettingskode før du endrer passordet.',
        verifyPasswordChangeFailed:
          'Kunne ikke verifisere passordet og autentiseringskoden eller gjenopprettingskoden.',
        verifyAndEnable: 'Verifiser og aktiver',
        scanTwoFactorQRCode:
          'Skann denne QR-koden med autentiseringsappen din, og skriv deretter inn koden.',
        showSetupKey: 'Vis oppsettnøkkel',
        hideSetupKey: 'Skjul oppsettnøkkel',
        setupKey: 'Oppsettnøkkel',
        twoFactorSetupFailed: 'Kunne ikke starte oppsett av tofaktorautentisering.',
        twoFactorVerifyFailed: 'Kunne ikke verifisere autentiseringskoden.',
        twoFactorDisableFailed: 'Kunne ikke deaktivere tofaktorautentisering.',
        disableTwoFactorDesc:
          'Skriv inn gjeldende passord og enten en autentiseringskode eller en ubrukt gjenopprettingskode. Deaktivering av tofaktorautentisering ugyldiggjør alle gjenopprettingskoder.',
        updateBtn: 'Endre',
        logoutBtn: 'Logg ut',
        logoutDesc: 'Er du sikker på at du vil logge ut?',
        okBtn: 'Ja',
        cancelBtn: 'Nei',
        twoFactorProtection: {
          title: 'Avansert 2FA-beskyttelse',
          description:
            'Velg hvilke sensitive handlinger som krever en ekstra faktor når beskyttelsen håndheves. Bryteren «Direkte nettinnlogging» gjør det motsatte: den lar deg logge inn kun med en autentiseringskode eller en ubrukt gjenopprettingskode.',
          login: 'Direkte nettinnlogging',
          webTerminal: 'Nettterminal',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API-nøkkel',
          targetPower: 'Strøm for måldatamaskinen',
          nanoKVMRestart: 'Omstart av NanoKVM',
          deleteImage: 'Slett image',
          scripts: 'Skriptoperasjoner',
          screenTimelapse: 'Skjerm-timelapse',
          loginScope:
            'Tillater nettinnlogging kun med en autentiseringskode eller en ubrukt gjenopprettingskode i stedet for brukernavn og passord. Dette reduserer innloggingssikkerheten.',
          webTerminalScope: 'Åpning av nettterminalen.',
          sshScope: 'Endring av SSH-tjenestens status.',
          tailscaleScope: 'Installering, tilkobling eller endring av Tailscale.',
          frp: 'FRP',
          frpScope: 'Installering, tilkobling eller endring av FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Installering, tilkobling eller endring av Cloudflare Tunnel.',
          zerotierScope: 'Installering, tilkobling eller endring av ZeroTier.',
          netbirdScope: 'Installering, tilkobling eller endring av NetBird.',
          mcpScope:
            'Visning, kopiering eller regenerering av MCP API-nøkkelen, og aktivering eller deaktivering av MCP.',
          targetPowerScope:
            'Slå av eller starte måldatamaskinen på nytt med den fysiske av/på-knappen.',
          nanoKVMRestartScope: 'Omstart av selve NanoKVM.',
          deleteImageScope: 'Slett et lagret ISO- eller IMG-image permanent.',
          scriptsScope: 'Kjøring, opplasting eller sletting av skript.',
          screenTimelapseScope:
            'Aktivering av skjerm-timelapse-opptak, visning av opptak og sletting av opptak.',
          authorizeTitle: 'Godkjenn beskyttelsesendringer',
          authorizeDesc:
            'Skriv inn en autentiseringskode eller ubrukt gjenopprettingskode. Godkjenningen varer i fem minutter.',
          authorizeFailed: 'Kunne ikke bekrefte autentiseringskoden eller gjenopprettingskoden.',
          loadFailed: 'Kunne ikke laste beskyttelsesinnstillinger.',
          updateFailed: 'Kunne ikke lagre beskyttelsesinnstillinger.',
          retry: 'Prøv igjen',
          disableScriptsWarningTitle: 'Deaktivere skriptbeskyttelse?',
          disableScriptsWarningDesc:
            'Uten beskyttelse kan et skadelig skript få utvidet tilgang til denne enheten og omgå eller fjerne 2FA-beskyttelsen.',
          cancelDisabling: 'Avbryt',
          continueDisabling: 'Fortsett deaktivering',
          continueDisablingCountdown: 'Fortsett deaktivering ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Veiledning for tofaktorautentisering',
          scope: 'Tofaktorautentisering legger til et ekstra sikkerhetslag for kontoen din.',
          whatTitle: 'Hva er 2FA?',
          whatDesc:
            'Tofaktorautentisering (2FA) beskytter kontoen din med en tidsbasert kode fra en autentiseringsapp. Som standard kreves koden i tillegg til passordet ditt, så selv om passordet blir kompromittert, er kontoen fortsatt beskyttet. Hvis direkte nettinnlogging er aktivert, logger koden alene deg inn.',
          appsTitle: 'Anbefalte apper',
          appsDesc:
            'Du kan bruke Google Authenticator eller Microsoft Authenticator. Begge er gratis og fungerer på iOS og Android.',
          setupTitle: 'Slik setter du opp',
          setupDesc:
            'Åpne autentiseringsappen, skann QR-koden som vises under oppsettet, eller skriv inn oppsettnøkkelen manuelt. Skriv deretter inn den 6-sifrede koden fra appen for å fullføre aktivering av 2FA.',
          recoveryTitle: 'Gjenopprettingskoder',
          recoveryDesc:
            'Gjenopprettingskoder er engangskoder for reservebruk. Hvis du mister tilgang til autentiseringsappen, kan du bruke en gjenopprettingskode til å logge inn. Lagre dem på et sikkert sted rett etter at de er generert.',
          logoutNotice:
            'Aktivering av 2FA logger ut alle aktive økter, inkludert denne. Du må logge inn på nytt på hver enhet.'
        }
      }
    },
    error: {
      title: 'Vi har hatt et problem',
      refresh: 'Oppdater'
    },
    pictureInPicture: {
      title: 'Bilde i bilde',
      enter: 'Aktiver bilde i bilde',
      exit: 'Avslutt bilde i bilde',
      error: 'Kan ikke starte bilde i bilde. Vent til videoen er lastet inn, og prøv igjen.'
    },
    fullscreen: {
      toggle: 'Veksle fullskjerm'
    },
    menu: {
      collapse: 'Skjul meny',
      expand: 'Utvid menyen'
    }
  }
};

export default nb;
