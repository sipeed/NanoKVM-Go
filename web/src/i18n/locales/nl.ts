const nl = {
  translation: {
    storage: {
      cards: { mount: 'Schijf koppelen', share: 'Bestanden delen' },
      title: 'Afbeeldingen en bestandsdeling',
      empty: 'Nog geen afbeeldingen',
      notAttached: 'Niets gekoppeld',
      attached: 'Gekoppeld: {{name}}',
      eject: 'Uitwerpen',
      mount: 'Koppelen',
      mountMode: 'Montagemodus',
      mountModeDisk: 'Massopslag',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Bestandsbeheer openen',
      refresh: 'Vernieuwen',
      delete: 'Verwijderen',
      deleteTitle: 'Medium verwijderen',
      deleteConfirm: '{{name}} verwijderen? De bestanden erin gaan verloren.',
      downloadImage: 'De afbeelding downloaden',
      typePartition: 'Gedeelde bestanden',
      volumeLabel: 'Verwisselbare schijf',
      groups: {
        share: 'Bestandsdeling',
        images: 'Afbeeldingen'
      },
      failed: 'De bewerking is mislukt',
      hidOnly: 'Alleen HID-modus: de virtuele schijf is niet beschikbaar',
      disabled: 'De virtuele schijf is uitgeschakeld (Instellingen - Apparaat)',
      force: {
        title: 'Het medium met geweld terughalen?',
        desc: 'De doelcomputer gebruikt dit medium nog. Het terughalen laat het USB-apparaat opnieuw opsommen en kan alles verliezen wat nog niet naar de schijf is geschreven.',
        menuHint:
          'De doelcomputer gebruikt dit medium. Open het bestandsbeheer om het met geweld terug te halen.'
      },
      tips: {
        title: 'Hoe te uploaden',
        usb1: 'Verbind de NanoKVM met uw computer via USB.',
        usb2: 'Koppel op deze pagina de gedeelde bestanden in massaopslagmodus.',
        usb3: 'Open de virtuele schijf op uw computer en kopieer het imagebestand naar de hoofdmap van de virtuele schijf.',
        usb4: 'Werp de schijf uit als het kopiëren klaar is: de image verschijnt dan in de lijst.',
        scp1: 'Zorg ervoor dat de NanoKVM en uw computer zich in hetzelfde lokale netwerk bevinden.',
        scp2: 'Open een terminal op uw computer en gebruik het SCP-commando om het imagebestand te uploaden naar de /data directory op de NanoKVM.',
        scp3: 'Voorbeeld: scp uw-image-pad root@uw-nanokvm-ip:/data'
      },
      job: {
        running: 'Bezig met {{name}}'
      },
      enableDisk: {
        title: 'De virtuele schijf inschakelen?',
        desc: 'Voor het koppelen van een medium is de virtuele schijf nodig, die nu uitgeschakeld is. Nu inschakelen? Het toetsenbord en de muis van de doelcomputer worden even losgekoppeld.',
        menuHint: 'De virtuele schijf is uitgeschakeld. Schakel die in bij Instellingen - Apparaat.'
      },
      add: {
        title: 'Medium toevoegen',
        upload: 'Een ISO of IMG uploaden',
        fromUrl: 'Vanaf een URL downloaden',
        url: 'URL van de afbeelding',
        urlRequired: 'Voer een URL in',
        onlyImages: 'Alleen .iso- en .img-bestanden kunnen worden gekoppeld',
        dropHint: 'Sleep een ISO of IMG hierheen, of klik om een bestand te kiezen',
        cdromOnlyIso: 'Een CD-ROM kan alleen een ISO 9660 / UDF-image bevatten',
        uploadFailed: 'Uploaden mislukt',
        downloadFailed: 'Downloaden mislukt',
        uploading: '{{name}} wordt geüpload',
        downloading: '{{name}} wordt gedownload',
        done: '{{name}} is klaar',
        checksumFailed: 'De image komt niet overeen met de SHA-256-controlesom.',
        blocked:
          'De doelcomputer gebruikt de schijf. Werp de virtuele USB-schijf veilig uit op de doelcomputer en schakel daarna de schakelaar Virtuele schijf uit via Instellingen > Apparaat.'
      },
      files: {
        reasonNotInitialized:
          'De virtuele USB-schijf is niet geïnitialiseerd. Initialiseer deze bij Instellingen > Apparaat voordat u bestanden deelt.',
        selectMedium: 'Kies een medium om de bestanden erin te bekijken.',
        reasonAttached: 'Dit medium is aan de doelcomputer gekoppeld.',
        reasonAttachedHint:
          'Werp de virtuele USB-schijf veilig uit op de doelcomputer en schakel daarna de schakelaar Virtuele schijf uit via Instellingen > Apparaat.',
        reasonNotBrowsable:
          'Dit medium bevat een bestandssysteem dat het bestandsbeheer niet kan openen.',
        reasonNoLoop: 'Dit apparaat kan geen schijfimages koppelen (geen loop-ondersteuning).',
        reasonISOImage:
          'Een ISO 9660-image kan hier niet worden doorzocht. Die kan wel gewoon aan de doelcomputer worden gekoppeld.',
        empty: 'Deze map is leeg',
        noMatch: 'Geen bestanden gevonden voor deze zoekopdracht.',
        free: '{{size}} vrij',
        search: 'In deze map zoeken',
        sort: {
          modified: 'Gewijzigd',
          name: 'Naam',
          size: 'Grootte',
          type: 'Type',
          asc: 'Oplopend',
          desc: 'Aflopend'
        },
        view: {
          list: 'Lijstweergave',
          grid: 'Pictogramweergave'
        },
        folder: 'Map',
        itemCount: 'Aantal items: {{count}}',
        itemCount_one: '{{count}} item',
        itemCount_other: '{{count}} items',
        dropHint: 'Sleep bestanden hierheen om ze naar deze map te uploaden',
        up: 'Naar de bovenliggende map',
        uploadButton: 'Uploaden',
        newFolder: 'Nieuwe map',
        newFolderPlaceholder: 'Mapnaam',
        create: 'Aanmaken',
        upload: 'Een bestand naar het medium uploaden',
        deleteFolderTitle: 'Map verwijderen',
        deleteFolderConfirm:
          'De map {{name}} en alles erin verwijderen? Dit kan niet ongedaan worden gemaakt.',
        uploading: 'Uploaden…',
        uploadFailed: 'Uploaden mislukt',
        download: 'Downloaden',
        delete: 'Verwijderen',
        deleteTitle: 'Bestand verwijderen',
        deleteConfirm: '{{name}} van het medium verwijderen?'
      },
      yes: 'Ja',
      no: 'Nee'
    },
    head: {
      storage: 'Afbeeldingen en bestandsdeling',
      desktop: 'Extern bureaublad',
      login: 'Inloggen',
      changePassword: 'Wachtwoord wijzigen',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Kan niet naar het klembord kopiëren. Geef de browser toegang tot het klembord en probeer het opnieuw.'
    },
    auth: {
      login: 'Inloggen',
      language: 'Taal',
      placeholderUsername: 'Voer gebruikersnaam in',
      placeholderPassword: 'Voer wachtwoord in',
      placeholderPassword2: 'Voer wachtwoord nogmaals in',
      noEmptyUsername: 'Gebruikersnaam mag niet leeg zijn',
      noEmptyPassword: 'Wachtwoord mag niet leeg zijn',
      noAccount:
        'Ophalen van gebruikersinformatie mislukt, vernieuw de webpagina of reset het wachtwoord',
      invalidUser: 'Ongeldige gebruikersnaam of wachtwoord',
      error: 'Onverwachte fout',
      twoFactorTitle: 'Verificatie vereist',
      twoFactorDesc: 'Voer je authenticatiecode of ongebruikte herstelcode in om in te loggen.',
      sensitiveActionDesc:
        'Voer een authenticatiecode of ongebruikte herstelcode in om door te gaan.',
      placeholderAuthCode: 'Authenticatiecode',
      noEmptyAuthCode: 'Authenticatiecode is vereist',
      verify: 'Verifiëren',
      backToLogin: 'Terug naar inloggen',
      verificationFailed: 'Verificatie mislukt',
      twoFactorEnabledLogout: {
        message:
          'Tweefactorauthenticatie is ingeschakeld. Voor de veiligheid zijn alle sessies afgemeld. Meld u opnieuw aan.'
      },
      rateLimited: 'Te veel mislukte pogingen. Probeer het opnieuw over {{seconds}} s.',
      recoveryCodeUsed: 'De herstelcode die voor deze aanmelding is gebruikt, is ongeldig gemaakt.',
      changePassword: 'Wachtwoord wijzigen',
      changePasswordDesc:
        'Voor de veiligheid van uw apparaat, wijzig alstublieft het webaanmeldingswachtwoord.',
      passwordVerificationRequired:
        'Ga terug naar de accountpagina in de instellingen en verifieer deze wachtwoordwijziging opnieuw.',
      passwordVerificationExpired:
        'De verificatie voor de wachtwoordwijziging is verlopen. Ga terug naar de accountpagina in de instellingen en verifieer opnieuw.',
      returnToSettings: 'Terug naar instellingen',
      differentPassword: 'Wachtwoorden komen niet overeen',
      illegalUsername: 'Gebruikersnaam bevat ongeldige tekens',
      illegalPassword: 'Wachtwoord bevat ongeldige tekens',
      forgetPassword: 'Wachtwoord vergeten',
      ok: 'OK',
      cancel: 'Annuleren',
      savePassword: 'Wachtwoord opslaan',
      loginButtonText: 'Inloggen',
      tips: {
        reset1: 'Standaard webaccount:',
        reset2: 'SSH-standaardaccount:',
        reset3: 'Om de wachtwoorden opnieuw in te stellen, zie ',
        changeReset: 'Raadpleeg voor het opnieuw instellen van het wachtwoord voor webaanmelding ',
        resetLink: 'Opnieuw instellen'
      }
    },
    wifi: {
      errors: {
        connect:
          'Kan geen verbinding maken met Wi-Fi. Controleer de netwerknaam, het wachtwoord en de Wi-Fi-status van het apparaat en probeer het opnieuw.'
      },
      title: 'Wi-Fi',
      description: 'Wi-Fi configureren voor NanoKVM',
      submitted: {
        title: 'Een ogenblik — NanoKVM maakt verbinding met je wifi',
        tipsTitle: 'Hoe nu verder?',
        tipSwitch: 'Je kunt je telefoon nu weer met je normale wifi verbinden.',
        tipCheck:
          'Kijk zo op het NanoKVM-scherm naar het nieuwe IP-adres en open het in je browser.',
        tipRetry:
          'Als er na een tijdje niets gebeurt, controleer dan de netwerknaam en het wachtwoord en probeer het opnieuw.'
      },
      invalidMode:
        'De huidige modus ondersteunt geen netwerkconfiguratie. Ga naar uw apparaat en schakel de configuratiemodus Wi-Fi in.',
      confirmBtn: 'OK',
      finishBtn: 'Gereed',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Wachtwoord',
      steps: {
        settings: 'Instellingen',
        wifi: 'Wi-Fi',
        enable: 'Inschakelen'
      },
      scan: {
        title: 'Kies een Wi-Fi-netwerk',
        description: 'Selecteer een netwerk in de buurt om door te gaan.',
        loading: 'Zoeken naar Wi-Fi-netwerken in de buurt…',
        empty: 'Geen Wi-Fi-netwerken in de buurt gevonden.',
        failed: 'Scannen mislukt. Voer de netwerkgegevens handmatig in.',
        retryBtn: 'Opnieuw scannen',
        manualBtn: 'Netwerk handmatig invoeren',
        backBtn: 'Terug naar netwerken in de buurt',
        connectTitle: 'Verbinden met {{ssid}}',
        connecting: 'Verbinden…',
        viewSimple: 'Overschakelen naar eenvoudige weergave',
        viewDetailed: 'Overschakelen naar gedetailleerde weergave'
      },
      ap: {
        authTitle: 'Authenticatie vereist',
        authDescription: 'Voer het wachtwoord AP in om door te gaan',
        authFailed: 'Ongeldig AP wachtwoord',
        passPlaceholder: 'AP wachtwoord',
        verifyBtn: 'Verifieer'
      },
      qr: {
        cameraBtn: 'Foto maken',
        pickBtn: 'Schermafbeelding kiezen',
        scanHint:
          'Maak een foto van een Wi-Fi QR-code, of kies een schermafbeelding ervan, zoals de code die een telefoon toont bij het delen van zijn Wi-Fi.',
        decoding: 'QR-code lezen…',
        notFound:
          'Er is geen QR-code gevonden op deze foto. Zorg dat de code gecentreerd en scherp is, en probeer het opnieuw.',
        invalid: 'Deze QR-code bevat geen bruikbare Wi-Fi-gegevens.',
        pickerUnavailable:
          "Er ging niets open? Dit aanmeldvenster kan de camera en foto's blokkeren. Open in plaats daarvan dit adres in de browser van uw telefoon:",
        confirmTitle: 'NanoKVM verbinden met {{ssid}}?',
        confirmDescription:
          'Deze netwerkgegevens komen van een QR-code. Controleer ze voordat u verbinding maakt.',
        openNetwork: 'Dit netwerk heeft geen wachtwoord.',
        connectBtn: 'Verbinden',
        otherBtn: 'Ander netwerk kiezen'
      }
    },
    notification: {
      browser: {
        title: 'Browseraanbeveling',
        description:
          'Voor de beste gebruikerservaring raden wij u aan de Chrome-browser te gebruiken. Hoewel we streven naar compatibiliteit met alle grote browsers, werken sommige geavanceerde functies buiten Chrome mogelijk niet zoals bedoeld.'
      },
      btnOk: 'Oké'
    },
    screen: {
      title: 'Scherm',
      video: 'Videomodus',
      quality: 'Kwaliteit',
      qualityAuto: 'Automatisch',
      qualityLossless: 'Verliesvrij',
      qualityHigh: 'Hoog',
      qualityMedium: 'Gemiddeld',
      qualityLow: 'Laag',
      scale: 'Schaal',
      noSignal: 'Geen videosignaal',
      captureDisabled:
        'Video-opname is uitgeschakeld. Schakel deze in via "Instellingen > Apparaat > Video-opname".',
      inconsistentVideoMode: 'Afspelen gepauzeerd. Er wordt een andere videomodus afgespeeld.',
      bandwidthWarning:
        'De netwerkbandbreedte is mogelijk onvoldoende voor de huidige video-instellingen. Probeer de bitrate of framesnelheid te verlagen.',
      webrtcConnectionFailed:
        'De WebRTC-verbinding kon niet worden gemaakt. Schakel over naar een Direct-videomodus.',
      webrtcUnavailable: {
        title: 'WebRTC is niet beschikbaar',
        description: 'Schakel WebRTC in uw browser in of schakel over naar de Direct-videomodus.'
      },
      upload: 'Uploaden',
      advanced: 'Geavanceerd',
      loading: 'Laden'
    },
    volume: {
      title: 'Volume'
    },
    microphone: {
      title: 'Microfoon',
      notMounted: 'Microfoon niet gemonteerd',
      tip1: 'De virtuele microfoon is niet op de externe host gemonteerd. Wilt u het nu koppelen en inschakelen?',
      tip2: 'U kunt dit ook later beheren in "Instellingen > Apparaat > Virtuele audio".',
      confirm: 'Inschakelen',
      cancel: 'Annuleren',
      noDevice:
        'Er is geen microfoon gedetecteerd op dit apparaat. Sluit een microfoon aan en probeer het opnieuw.',
      permissionDenied:
        'Toegang tot de microfoon is geweigerd. Sta microfoontoegang toe in de browserinstellingen.',
      notReadable:
        'De microfoon wordt door een andere toepassing gebruikt of kan niet worden gelezen.',
      overconstrained: 'De microfoon ondersteunt de vereiste audio-instellingen niet.',
      securityBlocked:
        'Microfoontoegang is op deze pagina geblokkeerd. Controleer de beveiligingsinstellingen van de browser.',
      aborted: 'Het starten van de microfoon is afgebroken. Probeer het opnieuw.',
      unsupported: 'Deze browser of verbindingsmethode ondersteunt geen microfoonopname.',
      enableFailed: 'De microfoon kon niet worden ingeschakeld. Probeer het opnieuw.',
      disconnected:
        'De microfoon is losgekoppeld, waardoor de microfoonoverdracht is uitgeschakeld.'
    },
    power: {
      title: 'Aan/uit',
      showConfirm: 'Bevestiging',
      showConfirmTip: 'Aan/uit-acties vereisen een extra bevestiging',
      powerShort: 'Aan/uit (kort indrukken)',
      powerLong: 'Aan/uit (lang indrukken)',
      powerConfirm: 'Aan/uit-actie uitvoeren?',
      okBtn: 'Ja',
      cancelBtn: 'Nee'
    },
    keyboard: {
      errors: {
        paste:
          'Kan de tekst niet naar het externe apparaat sturen. Controleer de KVM-verbinding en probeer het opnieuw.'
      },
      title: 'Toetsenbord',
      paste: 'Plakken',
      tips: 'Alleen standaard toetsenbordletters en symbolen worden ondersteund',
      placeholder: 'Voer tekst in',
      submit: 'Verzenden',
      virtual: 'Toetsenbord',
      swapCommandCtrl: 'Cmd en Ctrl omwisselen',
      shortcut: {
        title: 'Snelkoppelingen',
        custom: 'Aangepast',
        capture: 'Klik hier om de snelkoppeling vast te leggen',
        clear: 'Wissen',
        save: 'Opslaan',
        captureTips:
          'Voor het vastleggen van systeemtoetsen (zoals de Windows-toets) is toestemming voor volledig scherm vereist.',
        enterFullScreen: 'Schakelen naar volledig scherm.'
      },
      leaderKey: {
        title: 'Leader-toets',
        desc: 'Omzeil browserbeperkingen en stuur systeemsnelkoppelingen rechtstreeks naar de externe host.',
        howToUse: 'Hoe te gebruiken',
        simultaneous: {
          title: 'Gelijktijdige modus',
          desc1: 'Houd de Leader-toets ingedrukt en druk daarna op de sneltoets.',
          desc2: 'Intuïtief, maar kan conflicteren met systeemsnelkoppelingen.'
        },
        sequential: {
          title: 'Sequentiële modus',
          desc1:
            'Druk op de Leader-toets → druk de sneltoets in volgorde in → druk opnieuw op de Leader-toets.',
          desc2: 'Vereist meer stappen, maar vermijdt systeemconflicten volledig.'
        },
        enable: 'Leader-toets inschakelen',
        tip: 'Wanneer deze toets als Leader-toets is toegewezen, werkt hij uitsluitend als sneltoetstrigger en verliest hij zijn standaardgedrag.',
        placeholder: 'Druk op de Leader-toets',
        shiftRight: 'Rechter Shift',
        ctrlRight: 'Rechter Ctrl',
        metaRight: 'Rechter Win',
        submit: 'Verzenden',
        recorder: {
          rec: 'OPN',
          activate: 'Toetsen activeren',
          input: 'Druk op de snelkoppeling...'
        }
      }
    },
    mouse: {
      title: 'Muis',
      mode: 'Muismodus',
      absolute: 'Volgmuis',
      absoluteAndroid: 'Volgmuis (betere Android-compatibiliteit)',
      touchScreen: 'Multitouchscherm',
      relative: 'Exclusieve muis',
      direction: 'Scrollwielrichting',
      scrollUp: 'Scroll naar boven',
      scrollDown: 'Scroll naar beneden',
      speed: 'Scrollwielsnelheid',
      fast: 'Snel',
      slow: 'Langzaam',
      requestPointer:
        'Relatieve modus wordt gebruikt. Klik op het bureaublad om de muisaanwijzer te krijgen.',
      repairIphoneDrag: 'iPhone-slepen herstellen',
      resetHid: 'HID resetten',
      touchpadGuide: {
        title: 'Touchpadgids',
        scope: 'Geldt voor normale aanwijzerinvoer wanneer de muismodus Relatief is.',
        swipeTitle: 'Veeg om te bewegen',
        swipeDesc: 'Veeg binnen het actieve schermgebied om de externe aanwijzer te verplaatsen.',
        tapTitle: 'Tik om te klikken',
        tapDesc: 'Een korte tik stuurt één linkermuisklik.',
        holdTitle: 'Houd vast voor linkermuisknop ingedrukt',
        holdDesc:
          'Raak aan en houd uw vinger ongeveer 1 seconde stil om de linkermuisknop ingedrukt te houden.',
        dragTitle: 'Beweeg na vasthouden om te slepen',
        dragDesc:
          'Wanneer vasthouden actief is, beweeg uw vinger om te slepen met de linkermuisknop ingedrukt.'
      },
      hidOnly: {
        errors: {
          load: 'Kan de status van de HID-Only-modus niet laden. Probeer het zo opnieuw.',
          update:
            'Kan niet bevestigen of de HID-Only-modus is bijgewerkt. Open deze instelling opnieuw om de huidige status te controleren en probeer het daarna opnieuw.'
        },
        title: 'Alleen HID-modus',
        desc: 'Als uw muis en toetsenbord niet meer reageren en het opnieuw instellen van HID niet helpt, kan er sprake zijn van een compatibiliteitsprobleem tussen de NanoKVM en het apparaat. Probeer de modus HID-Only in te schakelen voor betere compatibiliteit.',
        tip1: 'Als u de modus HID-Only inschakelt, worden de virtuele U-schijf en het virtuele netwerk ontkoppeld',
        tip2: 'In de modus HID-Alleen is beeldmontage uitgeschakeld',
        enable: 'Schakel de modus HID-Alleen in',
        disable: 'Schakel de modus HID-Alleen uit'
      }
    },
    preview: {
      title: 'Voorbeeld'
    },
    image: {
      storageReadOnly: 'Bestanden kunnen niet worden geschreven. Schakel de virtuele schijf uit.',
      storageBusy:
        'De opslag is bezet. Voltooi of annuleer de bestandsoverdracht voordat u de virtuele schijf omschakelt.',
      title: 'Afbeeldingen',
      loading: 'Laden...',
      empty: 'Niets gevonden',
      upload: 'Afbeelding uploaden',
      dropFile: 'Sleep hier een imagebestand naartoe of klik om een bestand te kiezen',
      uploadAction: 'Uploaden',
      cancelAction: 'Annuleren',
      mount: 'Afbeelding koppelen',
      mounted: 'Gemonteerd',
      uploadError: 'Uploaden mislukt, probeer het opnieuw',
      uploadInsufficientStorage: 'Onvoldoende opslagruimte',
      uploadTooLarge: 'Image overschrijdt de groottelimiet',
      cancelUploadError: 'Upload annuleren mislukt',
      mountMode: 'Montagemodus',
      massStorage: 'Massaopslag',
      cdRom: 'CD ROM',
      readOnly: 'Alleen-lezen',
      mountFailed: 'Koppelen mislukt',
      mountDesc:
        'In sommige systemen is het noodzakelijk om de virtuele schijf op de externe host uit te werpen voordat het image wordt gekoppeld.',
      unmountFailed: 'Ontkoppelen mislukt',
      unmountDesc:
        'In sommige systemen moet u handmatig uitwerpen op de externe host voordat u de image ontkoppelt.',
      refresh: 'Vernieuw de afbeeldingenlijst',
      download: 'Afbeelding downloaden',
      attention: 'Let op',
      deleteConfirm: 'Weet u zeker dat u deze afbeelding wilt verwijderen?',
      deleteFailed: 'De afbeelding kon niet worden verwijderd',
      okBtn: 'Ja',
      cancelBtn: 'Nee'
    },
    script: {
      title: 'Script',
      upload: 'Uploaden',
      run: 'Uitvoeren',
      runBackground: 'Op achtergrond uitvoeren',
      runFailed: 'Uitvoeren mislukt',
      uploadFailed: 'Het script kon niet worden geüpload',
      deleteFailed: 'Het script kon niet worden verwijderd',
      attention: 'Let op',
      delDesc: 'Weet u zeker dat u dit bestand wilt verwijderen?',
      confirm: 'Ja',
      cancel: 'Nee',
      delete: 'Verwijderen',
      close: 'Sluiten'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM-terminal',
      confirm: 'OK',
      submit: 'Verzenden',
      username: 'Gebruikersnaam',
      password: 'Wachtwoord',
      authCode: 'Authenticatiecode',
      authorizeTitle: 'Webterminal autoriseren',
      codeAccountHint: 'Voer een authenticatiecode of ongebruikte herstelcode in.',
      authFailed: 'Authenticatie mislukt'
    },
    wol: {
      errors: {
        send: 'Kan de wekopdracht niet verzenden. Controleer het MAC-adres en de netwerkverbinding en probeer het opnieuw.',
        loadFailed:
          'Kan de Wake-on-LAN-apparatenlijst niet laden. Controleer of NanoKVM online is en probeer het opnieuw.',
        renameFailed:
          'Kan dit apparaat niet hernoemen. Controleer het MAC-adres en probeer het opnieuw.',
        deleteFailed: 'Kan dit apparaat niet verwijderen. Probeer het zo opnieuw.'
      },
      title: 'Wake-on-LAN',
      sending: 'Commando wordt verzonden...',
      sent: 'Commando verzonden',
      input: 'Voer het MAC-adres in',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (optioneel)',
      sha256Placeholder: 'Voer een SHA-256-controlesom van 64 tekens in',
      invalidSHA256: 'SHA-256 moet een hexadecimale tekenreeks van 64 tekens zijn',
      title: 'Afbeeldingdownloader',
      input: 'Voer een externe afbeelding in URL',
      ok: 'OK',
      disabled: '/data partitie is RO, dus we kunnen de afbeelding niet downloaden',
      downloadingFile: 'Downloaden: {{file}}',
      downloadingProgress: 'Downloaden ({{percentage}}): {{file}}',
      failed: 'Mislukt',
      insufficientStorage: 'Onvoldoende opslagruimte',
      remaining: 'Nog {{time}}',
      cancel: 'Annuleren',
      cancelFailed: 'Download annuleren mislukt',
      pause: 'Pauzeren',
      resume: 'Hervatten',
      pauseFailed: 'Download pauzeren mislukt',
      resumeFailed: 'Download hervatten mislukt'
    },
    settings: {
      back: 'Terug',
      close: 'Sluiten',
      ai: {
        errors: {
          loadServices:
            'Kan de status van AI-services niet laden. Controleer of NanoKVM online is en probeer het opnieuw.',
          loadMcpConfig:
            'Kan de configuratie van de MCP-service niet laden. Probeer het zo opnieuw.',
          updateMcpService:
            'Kan de status van de MCP-service niet bijwerken. Controleer de huidige status voordat u het opnieuw probeert.',
          accessMcpApiKey:
            'Kan de MCP API-sleutel niet ophalen. Probeer het opnieuw en voltooi de verificatie als daarom wordt gevraagd.',
          regenerateMcpApiKey:
            'Kan de MCP API-sleutel niet opnieuw genereren. Probeer het zo opnieuw.',
          regeneratedMcpApiKeyUnavailable:
            'De API-sleutel is opnieuw gegenereerd, maar kan niet worden weergegeven. Probeer hem opnieuw weer te geven. De vorige sleutel is niet meer geldig.',
          loadMemoryFabricState:
            'Kan de status van de Memory Fabric-service niet laden. Probeer het zo opnieuw.',
          updateMemoryFabricService:
            'Kan de status van de Memory Fabric-service niet bijwerken. Controleer de huidige status voordat u het opnieuw probeert.',
          loadScreenTimelapseState:
            'Kan de status van de scherm-timelapse-service niet laden. Probeer het zo opnieuw.',
          updateScreenTimelapseService:
            'Kan de status van de scherm-timelapse-service niet bijwerken. Controleer de huidige status voordat u het opnieuw probeert.',
          openScreenTimelapse:
            'Kan de scherm-timelapseweergave niet openen. Probeer het zo opnieuw.',
          loadScreenTimelapseRecordings:
            'Kan de scherm-timelapseopnamen niet laden. Probeer het zo opnieuw.',
          playScreenTimelapse:
            'Deze timelapseopname kan nu niet worden afgespeeld. Selecteer de datum opnieuw of probeer het later opnieuw.',
          screenTimelapseFullscreen:
            'Kan de volledigeschermmodus niet wijzigen. Controleer de browsermachtigingen en probeer het opnieuw.'
        },
        title: 'AI',
        mcpService: 'MCP-service',
        mcpServiceDesc:
          'Geverifieerde centrale gateway voor bediening, media en ingeschakelde Memory Fabric-tools',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Contextgeheugen en OCR van agents lokaal verwerken',
        memoryFabricRemoteRequiresMcp:
          'Schakel hierboven de MCP-service in om geheugen- en OCR-tools op afstand beschikbaar te maken.',
        memoryFabricRemoteAvailable:
          'Geheugen- en OCR-tools zijn via het endpoint /api/mcp hierboven beschikbaar met dezelfde API-sleutel.',
        memoryFabricConfig: {
          title: 'Configuratie',
          description: 'Beheer alleen de instellingen van het standaardmodel van Memory Fabric.',
          loadFailed: 'De Memory Fabric-configuratie kon niet worden geladen.',
          saveFailed: 'De Memory Fabric-configuratie kon niet worden opgeslagen.',
          saved: 'Configuratie opgeslagen.',
          savedRestarted: 'Configuratie opgeslagen en Memory Fabric opnieuw gestart.',
          noChanges: 'Geen configuratiewijzigingen.',
          reload: 'Opnieuw laden',
          save: 'Opslaan',
          unsaved: 'Niet-opgeslagen wijzigingen',
          conflictTitle: 'Configuratie is gewijzigd',
          conflictDescription:
            'De configuratie is elders gewijzigd. Laad de nieuwste versie voordat je opnieuw opslaat.',
          conflictReload: 'Nieuwste versie laden',
          discardTitle: 'Niet-opgeslagen wijzigingen verwerpen?',
          discardDescription:
            'Opnieuw laden vervangt het huidige formulier door de laatst opgeslagen configuratie.',
          discardConfirm: 'Verwerpen en opnieuw laden',
          sections: {
            models: 'Modellen'
          },
          openAICompatibleHint:
            'Ondersteunt OpenAI-compatibele modelproviders. Voer de modelnaam, basis-URL en API-sleutel van de provider in.',
          fields: {
            model: 'Modelnaam',
            baseUrl: 'Basis-URL',
            maxTokens: 'Maximumaantal tokens',
            timeoutSeconds: 'Time-out (seconden)'
          },
          apiKey: {
            title: 'API-sleutel',
            configured: 'Geconfigureerd',
            notConfigured: 'Niet geconfigureerd',
            keep: 'Behouden',
            replace: 'Vervangen',
            clear: 'Verwijderen',
            replacement: 'Nieuwe API-sleutel',
            replacementPlaceholder: 'Voer de nieuwe API-sleutel in',
            clearHint:
              'De opgeslagen API-sleutel wordt verwijderd; de omgevingsvariabele kan nog steeds een sleutel leveren.',
            required: 'Voer een nieuwe API-sleutel in.'
          },
          validation: {
            required: 'Dit veld is verplicht.',
            invalidUrl: 'Voer een geldige HTTP- of HTTPS-URL in.'
          }
        },
        screenTimelapseService: 'Schermtimelapse',
        screenTimelapseServiceDesc:
          'Speel Memory Fabric-schermgeschiedenis af en zoek in vastgelegde scherminhoud',
        screenTimelapseSearchRequiresMemoryFabric:
          'Schakel ook Memory Fabric in om opgenomen inhoud te doorzoeken.',
        screenTimelapseOpen: 'Openen',
        screenTimelapseSearchPlaceholder: 'Zoeken in video-inhoud...',
        screenTimelapseNoVideos: 'Geen timelapse-video’s gevonden',
        screenTimelapseNoResults: 'Geen overeenkomende frames gevonden',
        screenTimelapseMultipleResults: 'Zoekresultaten',
        screenTimelapseUnavailable: 'Niet beschikbaar',
        screenTimelapsePlayerUnsupported: 'Deze browser kan timelapse-video niet afspelen',
        screenTimelapseSearchFailed: 'Zoeken mislukt',
        screenTimelapseLoading: 'Timelapse laden...',
        screenTimelapseSearchAction: 'Timelapse zoeken',
        screenTimelapsePlay: 'Timelapse afspelen',
        screenTimelapsePause: 'Timelapse pauzeren',
        screenTimelapseEnterFullscreen: 'Volledig scherm openen',
        screenTimelapseExitFullscreen: 'Volledig scherm sluiten',
        screenTimelapseFullscreenUnsupported: 'Deze browser kan niet naar volledig scherm gaan',
        screenTimelapsePreviousDay: 'Vorige dag',
        screenTimelapseNextDay: 'Volgende dag',
        screenTimelapseDeleteDay: 'Opname van {{date}} verwijderen',
        screenTimelapseDelete: 'Verwijderen',
        screenTimelapseDeleteConfirmTitle: 'Timelapse-opname verwijderen?',
        screenTimelapseDeleteConfirmDesc:
          'De opname van {{date}} neemt {{size}} in beslag. Deze actie kan niet ongedaan worden gemaakt.',
        screenTimelapseDeleteSuccess: 'Opname van {{date}} verwijderd',
        screenTimelapseDeleteFailed: 'Kan de opname van {{date}} niet verwijderen',
        endpoint: 'Eindpunt',
        apiKey: 'API-sleutel',
        regenerate: 'Opnieuw genereren',
        copyApiKey: 'API-sleutel kopiëren',
        copyEndpoint: 'Eindpunt kopiëren',
        showApiKey: 'API-sleutel tonen',
        regenerateConfirmTitle: 'API-sleutel opnieuw genereren',
        regenerateConfirmDesc:
          'Bestaande MCP-clients werken niet meer totdat ze met de nieuwe sleutel zijn bijgewerkt. Wil je doorgaan?',
        okBtn: 'Ja',
        cancelBtn: 'Nee'
      },
      title: 'Instellingen',
      about: {
        title: 'Over NanoKVM',
        information: 'Informatie',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applicatie versie',
        applicationTip: 'NanoKVM webapplicatie versie',
        arch: 'Architectuur',
        device_number: 'Apparaatnummer',
        image: 'Image versie',
        imageTip: 'NanoKVM systeem image versie',
        deviceKey: 'Apparaat sleutel',
        community: 'Gemeenschap',
        hostname: 'Hostnaam',
        hostnameUpdated: 'Hostnaam bijgewerkt. Start opnieuw op om toe te passen.',
        ipType: {
          Wired: 'Bedraad',
          Wireless: 'Draadloos',
          Other: 'Anders'
        }
      },
      appearance: {
        title: 'Uiterlijk',
        display: 'Beeldscherm',
        language: 'Taal',
        languageDesc: 'Selecteer de taal voor de interface',
        languageChangeFailed: 'Taal wijzigen mislukt',
        webTitle: 'Webtitel',
        webTitleDesc: 'Pas de titel van de webpagina aan',
        menuBar: {
          title: 'Menubalk',
          mode: 'Weergavemodus',
          modeDesc: 'Geef de menubalk weer op het scherm',
          modeOff: 'Uit',
          modeAuto: 'Automatisch verbergen',
          modeAlways: 'Altijd zichtbaar',
          icons: 'Submenupictogrammen',
          iconsDesc: 'Submenupictogrammen weergeven in de menubalk'
        }
      },
      screen: {
        title: 'Scherm',
        videoMode: {
          title: 'Videomodus',
          description: 'Selecteer de coderings- en transmissiemethode'
        },
        edid: {
          description: 'Weergaveparameters selecteren',
          default: 'Standaard',
          custom: 'Aangepast',
          document: 'Documentatie',
          monitorName: 'Monitornaam',
          monitorNameDescription:
            'Maximaal 13 afdrukbare ASCII-tekens; laat leeg om de oorspronkelijke naam uit de EDID te gebruiken',
          monitorNamePlaceholder: 'Oorspronkelijke EDID-naam gebruiken',
          monitorNameApply: 'Toepassen',
          monitorNameInvalid: 'Voer maximaal 13 afdrukbare ASCII-tekens in',
          monitorNameUnsupported:
            'De huidige EDID ondersteunt geen overschrijving van de monitornaam',
          monitorNameLoadFailed: 'Kan de monitornaam niet laden',
          monitorNameUpdateFailed: 'Kan de monitornaam niet bijwerken',
          powerConfirmTitle: 'Bevestigen voordat EDID wordt bijgewerkt',
          powerConfirmDescription:
            'Sluit eerst PD-voeding aan. Het bijwerken van EDID kan het scherm kort loskoppelen; als het apparaat het scherm daarna niet correct detecteert, start het dan handmatig opnieuw op.',
          powerConfirmOk: 'Doorgaan',
          powerConfirmOkCountdown: 'Doorgaan ({{seconds}})',
          powerConfirmCancel: 'Annuleren',
          errors: {
            badRequest: 'Ongeldig EDID-verzoek',
            invalidFileName: 'Selecteer een .bin-EDID-bestand met een geldige bestandsnaam',
            fileTooLarge: 'Het EDID-bestand is te groot',
            invalidLength: 'De EDID moet 128, 256, 384 of 512 bytes groot zijn',
            invalidHeader: 'De EDID-header is ongeldig',
            unsupportedVersion: 'Deze EDID-versie wordt niet ondersteund',
            extensionCountMismatch:
              'Het aantal EDID-uitbreidingsblokken komt niet overeen met de bestandslengte',
            invalidChecksum: 'De EDID-controlesom is ongeldig',
            invalidCta: 'De structuur van de CTA-uitbreiding is ongeldig',
            currentCannotOverwrite:
              'De EDID die momenteel wordt gebruikt, kan niet worden overschreven. Schakel eerst over naar een andere EDID',
            saveFailed: 'De EDID kon niet worden opgeslagen',
            switchFailed: 'Er kon niet van EDID worden gewisseld',
            uploadFailed: 'De EDID kon niet worden geüpload',
            deleteFailed: 'De EDID kon niet worden verwijderd',
            networkFailed: 'Netwerkfout. Probeer het opnieuw'
          }
        },
        rateControlMode: {
          title: 'Bitrate-regelmodus',
          description: 'Selecteer de methode om de bitsnelheid te regelen',
          cbr: 'CBR (Constant Bitrate): Geeft prioriteit aan verbindingsstabiliteit. Minimaliseert stotteren, maar de beeldkwaliteit kan afnemen tijdens scènes met veel beweging.',
          vbr: 'VBR (variabele bitsnelheid): geeft prioriteit aan de beeldhelderheid. Scènes met veel beweging kunnen bandbreedtepieken veroorzaken, wat mogelijk kan leiden tot latentie of stotteren.'
        },
        bitrate: {
          title: 'Bitsnelheid',
          description: 'Breng de helderheid van het beeld in evenwicht met het bandbreedtegebruik',
          lower:
            'Lagere waarden verminderen de latentie op slechte netwerken, maar brengen de beeldkwaliteit in gevaar, waardoor onscherpte of artefacten kunnen ontstaan.',
          higher:
            'Hogere waarden verbeteren de beeldhelderheid, maar verbruiken meer bandbreedte. Kan de latentie verhogen als de verbinding traag is.'
        },
        gop: {
          description: 'Stel het interval tussen volledige keyframes in',
          lower:
            'Lagere waarden verzenden vaker volledige frames. Dit versnelt het herstel van visuele artefacten, maar verbruikt meer bandbreedte en vermindert beelddetails.',
          higher:
            'Hogere waarden optimaliseren voor incrementele updates. Dit verbetert de beelddetails en bespaart bandbreedte, maar het herstel van visuele artefacten zal langer duren.'
        },
        quality: {
          title: 'Kwaliteit',
          description: 'Stel de beeldkwaliteit van het display in'
        },
        fps: {
          auto: 'Automatisch',
          description: 'Stel de framesnelheid van het beeldscherm in',
          tip1: 'Als u de waarde instelt op 0, wordt de automatische modus ingeschakeld, waarbij de weergave wordt gesynchroniseerd met de framesnelheid van de externe host.',
          tip2: 'Het verlagen van de framesnelheid vermindert het bandbreedtegebruik aanzienlijk. Verlaag deze handmatig tijdens netwerkinstabiliteit of wanneer hoge framesnelheden niet vereist zijn.'
        },
        scale: {
          title: 'Schaal',
          description: 'Stel de schaalverhouding van het scherm in'
        }
      },
      device: {
        title: 'Apparaat',
        virtualAudioConfirm: {
          title: 'Bevestigen voordat virtuele audio wordt omgeschakeld',
          description:
            'Sluit eerst de PD-voeding aan. Het omschakelen van virtuele audio kan het scherm kort loskoppelen; als het apparaat het scherm daarna niet correct detecteert, start het dan handmatig opnieuw op.',
          ok: 'Doorgaan met omschakelen',
          okCountdown: 'Doorgaan met omschakelen ({{seconds}})',
          cancel: 'Annuleren'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Configureer Wi-Fi',
          apMode: 'AP-modus is ingeschakeld, maak handmatig verbinding met Wi-Fi',
          others: 'Anderen...',
          connect: 'Maak verbinding met Wi-Fi',
          connectDesc1: 'Voer de netwerk-ssid en het wachtwoord in',
          connectDesc2: 'Voer het wachtwoord in om lid te worden van dit netwerk',
          disconnect: 'Weet u zeker dat u de verbinding met het netwerk wilt verbreken?',
          ssid: 'Naam',
          password: 'Wachtwoord',
          joinBtn: 'Doe mee',
          confirmBtn: 'Oké',
          cancelBtn: 'Annuleren',
          preset: {
            title: 'Opgeslagen Wi-Fi',
            description:
              'Sla één doel-Wi-Fi vooraf op; het apparaat maakt bij het opstarten automatisch verbinding',
            empty: 'Geen',
            addBtn: 'Voorinstelling toevoegen',
            removeBtn: 'Voorinstelling verwijderen',
            saveTitle: 'Wi-Fi vooraf opslaan',
            saveDesc:
              'Sla het doelnetwerk nu op; het apparaat maakt automatisch verbinding zodra het netwerk bij het opstarten wordt gevonden',
            saveBtn: 'Opslaan',
            passwordPlaceholder: 'Wachtwoord (leeg laten voor open netwerken)',
            saveFailed: 'Opslaan van Wi-Fi-voorinstelling mislukt',
            removeFailed: 'Verwijderen van Wi-Fi-voorinstelling mislukt',
            removeConfirm: 'Weet je zeker dat je de opgeslagen Wi-Fi wilt verwijderen?',
            warning:
              'Eenmaal opgeslagen probeert het apparaat bij elke opstart verbinding te maken met dit Wi-Fi zodra het wordt gedetecteerd, wat het verbinden met andere Wi-Fi-netwerken kan verstoren. De voorinstelling blijft actief totdat u deze wijzigt of verwijdert.'
          }
        },
        capture: {
          title: 'Video-opname',
          description: 'Video-opname inschakelen'
        },
        ssh: {
          description: 'Schakel SSH externe toegang in',
          tip: 'Stel na het inschakelen van SSH in de terminal een sterk wachtwoord in voor de rootgebruiker.'
        },
        swap: {
          errors: {
            load: 'Kan de status van het swapgeheugen niet laden. Probeer het zo opnieuw.',
            update:
              'Kan de instelling voor swapgeheugen niet bijwerken. Controleer de huidige status voordat u het opnieuw probeert.'
          },
          title: 'Swapgeheugen',
          description: 'Schakel gecomprimeerd wisselgeheugen in'
        },
        advanced: 'Geavanceerde instellingen',
        staticIp: {
          title: 'Statisch IP-adres',
          config: 'Configuratie',
          tip: 'Om verbindingsverlies te voorkomen, wordt aanbevolen om te configureren terwijl u fysieke toegang tot het apparaat heeft.',
          document: 'Controleer het document.',
          enable: 'Statisch IP-adres inschakelen',
          placeholder: 'Formaat: IP/CIDR [Gateway]\nVoorbeeld: 192.168.1.10/22',
          notEmpty: 'IP-adres mag niet leeg zijn.',
          notValid: 'IP-adresformaat is ongeldig.',
          failed: 'Configuratie mislukt.',
          success: 'Configuratie voltooid. Controleer of het IP-adres beschikbaar is.',
          okBtn: 'Oké',
          cancelBtn: 'Annuleren'
        },
        usbNetwork: {
          title: 'USB-netwerk',
          mode: 'Werkingsmodus',
          server: 'Servermodus',
          serverDesc: 'Wijs via de ingebouwde DHCP-server een IP-adres toe aan de externe host.',
          client: 'Clientmodus',
          clientDesc: 'Vraag via DHCP een IP-adres op bij de externe host.',
          forwarding: 'IP-doorsturen',
          forwardingTip:
            'Schakel IP-doorsturen in om de externe host via NanoKVM toegang tot internet te geven.',
          manualIp: 'Handmatig IP-adres',
          ipv4Placeholder: 'IPv4/CIDR, bijv. 10.10.10.1/24',
          invalidIpv4: 'Voer een geldig IPv4/CIDR-adres in.',
          loadFailed: 'De USB-netwerkconfiguratie kan niet worden geladen.',
          failed: 'Configuratie mislukt.',
          success: 'Configuratie voltooid.',
          applyBtn: 'Toepassen',
          cancelBtn: 'Annuleren'
        },
        usbId: {
          title: 'USB-apparaatidentiteit',
          config: 'Configureren',
          tip: 'Aangepaste USB-ID’s veranderen hoe de host dit apparaat herkent. Toepassen herstart de USB-gadget en kan toetsenbord, muis of netwerk kort verbreken.',
          enable: 'Aangepaste USB-apparaatidentiteit inschakelen',
          vid: 'Vendor-ID (VID)',
          pid: 'Product-ID (PID)',
          bcdDevice: 'Apparaatversie (bcdDevice)',
          serialnumber: 'Serienummer',
          manufacturer: 'Fabrikant',
          product: 'Productnaam',
          vidPlaceholder: 'bijv. 0x359F',
          pidPlaceholder: 'bijv. 0x3305',
          bcdDevicePlaceholder: 'bijv. 0x0101 (optioneel)',
          serialnumberPlaceholder: 'bijv. 0123456789ABCDEF (optioneel)',
          manufacturerPlaceholder: 'bijv. sipeed (optioneel)',
          productPlaceholder: 'bijv. NanoKVM-Go (optioneel)',
          notEmpty: 'VID en PID mogen niet leeg zijn.',
          notValid: 'VID/PID-indeling is ongeldig. Gebruik hex zoals 0x359F.',
          bcdNotValid: 'Ongeldige bcdDevice-indeling. Gebruik hex zoals 0x0101 of laat leeg.',
          failed: 'Configuratie mislukt.',
          success: 'Configuratie voltooid. USB-gadget herstart.',
          okBtn: 'OK',
          cancelBtn: 'Annuleren'
        },
        diskInquiry: {
          title: 'USB-schijfnaam',
          config: 'Configureren',
          tip: 'Naam van de virtuele USB-stick zoals die op de remote computer verschijnt. Opslaan herstart USB en kan toetsenbord, muis of netwerk kort verbreken.',
          enable: 'Aangepaste USB-schijfnaam gebruiken',
          placeholder: 'bijv. sipeed  Flash Drive     1.00',
          notEmpty: 'Naam mag niet leeg zijn.',
          notValid: 'Naam is ongeldig. Maximaal 28 afdrukbare ASCII-tekens zonder regeleinden.',
          failed: 'Configuratie mislukt.',
          success: 'Configuratie voltooid. USB-gadget herstart.',
          okBtn: 'OK',
          cancelBtn: 'Annuleren'
        },
        diskInquiryCdrom: {
          title: 'Virtuele cd-schijfnaam',
          config: 'Configureren',
          tip: 'Naam van de virtuele cd/dvd-drive op de remote computer. Opslaan herstart USB en kan toetsenbord, muis of netwerk kort verbreken.',
          enable: 'Aangepaste cd-schijfnaam gebruiken',
          placeholder: 'bijv. sipeed  Optical Drive   1.00',
          notEmpty: 'Naam mag niet leeg zijn.',
          notValid: 'Naam is ongeldig. Maximaal 28 afdrukbare ASCII-tekens zonder regeleinden.',
          failed: 'Configuratie mislukt.',
          success: 'Configuratie voltooid. USB-gadget herstart.',
          okBtn: 'OK',
          cancelBtn: 'Annuleren'
        },
        uac2FunctionName: {
          title: 'USB-audionaam',
          config: 'Configureren',
          tip: 'Naam van het virtuele USB-microfoon-/audioapparaat op de remote computer. Opslaan herstart USB en kan toetsenbord, muis of netwerk kort verbreken.',
          enable: 'Aangepaste USB-audionaam gebruiken',
          placeholder: 'bijv. NanoKVM-Go Audio',
          notEmpty: 'Naam mag niet leeg zijn.',
          notValid: 'Naam is ongeldig. Maximaal 64 afdrukbare tekens zonder regeleinden.',
          failed: 'Configuratie mislukt.',
          success: 'Configuratie voltooid. USB-gadget herstart.',
          okBtn: 'OK',
          cancelBtn: 'Annuleren'
        },
        lowPower: {
          title: 'Energiezuinige modus',
          description: 'Verminder het stroomverbruik tijdens inactiviteit',
          tips: 'Aanwijzingen',
          tip1: 'Indien ingeschakeld, wordt de video-opname tijdens perioden van inactiviteit gepauzeerd om het energieverbruik en de warmteontwikkeling te verminderen.',
          tip2: 'Het hervatten van de video-opname kan herinitialisatie vereisen, waarbij video tijdelijk niet beschikbaar kan zijn.',
          okBtn: 'Oké',
          cancelBtn: 'Annuleren'
        },
        mouseJiggler: {
          title: 'Muisschommel',
          description: 'Voorkom dat de externe host in slaap valt',
          disable: 'Uitschakelen',
          absolute: 'Absolute modus',
          relative: 'Relatieve modus'
        },
        fingerRobot: {
          title: 'Vingerrobot',
          description: 'Vingerrobotbediening inschakelen'
        },
        mdns: {
          description: 'Schakel de mDNS detectieservice in'
        },
        hidOnly: 'HID-Alleen modus',
        imageMounted: 'Afbeelding gemonteerd',
        disk: 'Virtuele schijf',
        diskDesc: 'Koppel virtuele U-schijf aan de externe host',
        sdCard: {
          warning: 'Virtuele schijf initialiseren',
          tip1: 'Bij de eerste activering wordt de virtuele schijf geïnitialiseerd en worden bestaande imagebestanden en gegevens van de virtuele schijf op het apparaat gewist. Dit kan niet ongedaan worden gemaakt. Maak eerst een back-up van belangrijke bestanden.',
          tip2: 'Doorgaan met initialiseren?',
          capacity: 'Capaciteit',
          disable: 'Uitschakelen',
          disableWarning: 'Virtuele schijf uitschakelen?',
          disableTip:
            'Bij uitschakelen worden alle bestanden op de virtuele schijf gewist en wordt de gebruikte apparaatopslag vrijgegeven. De volgende keer dat u de schijf inschakelt, moet u opnieuw een capaciteit kiezen en initialiseren. Dit kan niet ongedaan worden gemaakt.'
        },
        network: 'Virtueel Netwerk',
        networkDesc: 'Koppel virtueel netwerk kaart aan de externe host',
        mic: 'Virtuele audio',
        micDesc: 'Kies de virtuele USB-audiomodus voor de externe host',
        virtualAudioMode: {
          off: 'Uit',
          mic: 'Alleen microfoon',
          duplex: 'Microfoon + luidspreker'
        },
        speaker: 'Virtuele luidspreker',
        speakerDesc:
          'Indien ingeschakeld vervangt dit de standaard DP-audio-uitgang om externe gesprekken met betere compatibiliteit te ondersteunen. Indien uitgeschakeld kan alleen media-audio worden doorgestuurd, geen gespreksaudio.',
        micTip:
          'Vanwege USB-eindpuntbeperkingen wordt bij het inschakelen van de virtuele microfoon het secundaire USB-scherm uitgeschakeld.\nEn het wordt aanbevolen om deze functie uit te schakelen tijdens BIOS-bewerkingen om optimale compatibiliteit te garanderen.',
        reboot: 'Opnieuw opstarten',
        rebootDesc: 'Weet u zeker dat u NanoKVM opnieuw wilt opstarten?',
        rebootToFlashing: 'Opnieuw opstarten in flashmodus',
        rebootToFlashingButton: 'Opnieuw opstarten',
        rebootToFlashingTitle: 'Opnieuw opstarten in flashmodus?',
        rebootToFlashingDesc:
          'NanoKVM wordt opnieuw opgestart en gaat naar de USB-flashmodus. Zorg dat de USB-interface voor het flashen is aangesloten en dat het flashprogramma gereed is.',
        datetime: {
          timezone: 'Tijdzone',
          datetime: 'Datumtijd',
          dateAndTime: 'Datum en tijd',
          synchronize: 'Tijd synchroniseren',
          lastSynchronization: 'Laatste synchronisatie: ',
          notSynchronized: 'De systeemtijd is nog niet gesynchroniseerd',
          syncNow: 'Nu synchroniseren',
          manualSet: 'Handmatig instellen',
          manualSetTitle: 'Tijd handmatig instellen',
          manualSetDescription: 'Na de wijziging moet u zich opnieuw aanmelden.',
          useBrowserTime: 'Browsertijd gebruiken',
          setTime: 'Tijd instellen',
          invalidTime: 'Voer een geldige datum en tijd in.',
          setFailed: 'De systeemtijd kan niet worden ingesteld. Probeer het opnieuw.',
          setSucceeded: 'De systeemtijd is bijgewerkt. Meld u opnieuw aan.'
        },
        dialBackground: {
          errors: {
            load: 'Kan de achtergrondafbeeldingen van de wijzerplaat niet laden. Probeer het zo opnieuw.',
            upload:
              'Kan de achtergrondafbeelding niet uploaden. Controleer of het een geldig BMP-bestand is en probeer het opnieuw.',
            switch:
              'Kan de achtergrond van de wijzerplaat niet wijzigen. Vernieuw de afbeeldingslijst en probeer het opnieuw.'
          },
          title: 'Wijzerplaatachtergrond',
          description:
            'Selecteer een BMP-afbeelding uit /kvmcomm/ui/bg/ als wijzerplaatachtergrond',
          upload: 'Uploaden',
          loading: 'Laden...',
          empty: 'Geen BMP-afbeeldingen gevonden',
          current: 'Huidig',
          onlyBmp: 'Alleen BMP-afbeeldingen worden ondersteund',
          uploaded: 'Upload voltooid',
          switched: 'Achtergrond gewijzigd'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nee'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Weet u zeker dat u Tailscale opnieuw wilt opstarten?',
        stop: 'Weet u zeker dat u Tailscale wilt stoppen?',
        stopDesc: 'Meld Tailscale af en schakel het automatisch opstarten bij het opstarten uit.',
        loading: 'Laden...',
        notInstall: 'Tailscale niet gevonden! Installeer a.u.b.',
        install: 'Installeren',
        installing: 'Installeren bezig',
        failed: 'Installatie mislukt',
        retry: 'Vernieuw en probeer opnieuw. Of probeer handmatig te installeren',
        download: 'Download het',
        package: 'installatiepakket',
        unzip: 'en pak het uit',
        upTailscale: 'Upload tailscale naar NanoKVM directory /usr/bin/',
        upTailscaled: 'Upload tailscaled naar NanoKVM directory /usr/sbin/',
        refresh: 'Vernieuw huidige pagina',
        notRunning: 'Tailscale is niet actief. Start het programma om door te gaan.',
        run: 'Starten',
        notLogin:
          'Het apparaat is nog niet gekoppeld. Log in en koppel dit apparaat aan uw account.',
        urlPeriod: 'Deze url is 10 minuten geldig',
        login: 'Inloggen',
        loginSuccess: 'Inloggen gelukt',
        enable: 'Tailscale inschakelen',
        exitNode: 'Uitgangsknooppunt',
        exitNodeDesc: 'Kondig dit apparaat aan als internetuitgang voor uw tailnet.',
        subnetRoute: 'Subnetroute',
        subnetRouteDesc: 'Kondig het direct verbonden LAN-subnet {{subnet}} aan in uw tailnet.',
        subnetUnavailable: 'Er is geen direct verbonden LAN-subnet beschikbaar.',
        routingUnavailable:
          'Tailscale-routeringsinstellingen zijn niet beschikbaar op dit apparaat.',
        pendingApproval:
          'Keur na inschakeling de aangekondigde route goed in de Tailscale-beheerconsole.',
        routingUpdateFailed: 'De Tailscale-routeringsinstellingen konden niet worden bijgewerkt.',
        deviceName: 'Apparaatnaam',
        deviceIP: 'Apparaat IP',
        account: 'Account',
        logout: 'Uitloggen',
        logoutDesc: 'Weet u zeker dat u wilt uitloggen?',
        uninstall: 'Verwijderen Tailscale',
        uninstallDesc: 'Weet u zeker dat u Tailscale wilt verwijderen?',
        okBtn: 'Ja',
        cancelBtn: 'Nee'
      },
      update: {
        title: 'Controleren op updates',
        queryFailed: 'Ophalen versie mislukt',
        updateFailed: 'Update mislukt. Probeer het opnieuw.',
        isLatest: 'U heeft al de nieuwste versie.',
        available: 'Er is een update beschikbaar. Weet u zeker dat u wilt updaten?',
        updating: 'Update gestart. Even geduld a.u.b...',
        confirm: 'Bevestigen',
        upgrade_tip_nano:
          'Er is een nieuwe versie van NanoKVM beschikbaar. Weet je zeker dat je nu wilt updaten?',
        upgrade_tip_firmware:
          'Er is een nieuwe firmwareversie beschikbaar. Weet je zeker dat je nu wilt updaten?',
        cancel: 'Annuleren',
        preview: 'Preview-updates',
        previewDesc: 'Krijg vroegtijdig toegang tot nieuwe functies en verbeteringen',
        previewTip:
          'Houd er rekening mee dat preview-releases bugs of onvolledige functionaliteit kunnen bevatten!',
        download: 'Downloaden',
        install: 'Installeren',
        changelog: 'Wijzigingslog',
        restart: 'Opnieuw opstarten'
      },
      account: {
        title: 'Account',
        webAccount: 'Naam webaccount',
        password: 'Wachtwoord',
        twoFactor: 'Tweefactorauthenticatie',
        enableTwoFactor: 'Inschakelen',
        enableTwoFactorDesc:
          'Voer je huidige wachtwoord in om tweefactorauthenticatie in te stellen.',
        disableTwoFactor: 'Uitschakelen',
        recoveryCodes: 'Herstelcodes',
        enableRecoveryCodes: 'Inschakelen',
        updateRecoveryCodes: 'Bijwerken',
        recoveryCodesTooltipDisabled:
          'Herstelcodes zijn reservegegevens voor tweefactorauthenticatie. Je kunt ze genereren nadat tweefactorauthenticatie is ingeschakeld.',
        recoveryCodesTooltipEnabled:
          'Herstelcodes zijn alleen geldig zolang tweefactorauthenticatie is ingeschakeld. Het uitschakelen van tweefactorauthenticatie maakt alle herstelcodes ongeldig; zolang het ingeschakeld blijft, kunnen ze alleen worden bijgewerkt.',
        recoveryCodesRegenerateDesc:
          'Voer je huidige wachtwoord en hetzij een authenticatiecode of een ongebruikte herstelcode in. Het bijwerken van herstelcodes maakt alle bestaande herstelcodes ongeldig.',
        recoveryCodesVerifyFailed:
          'Kan het wachtwoord en de authenticatiecode of herstelcode niet verifiëren.',
        recoveryCodesSavedTitle: 'Sla deze herstelcodes nu op',
        recoveryCodesSavedDesc:
          'Herstelcodes kunnen maar één keer worden bekeken. Sla ze op een veilige locatie op voordat je deze pagina verlaat.',
        enableRecoveryCodesDesc:
          'Voer je huidige wachtwoord en een authenticatiecode in om herstelcodes te genereren. Herstelcodes kunnen maar één keer worden bekeken, dus sla ze op voordat je deze pagina verlaat.',
        copyRecoveryCodes: 'Kopiëren',
        downloadRecoveryCodes: 'Downloaden',
        recoveryCodesCopyFailed: 'Kan herstelcodes niet kopiëren.',
        done: 'Gereed',
        currentPassword: 'Huidig wachtwoord',
        authCode: 'Authenticatiecode',
        authCodeOrRecoveryCode: 'Authenticatiecode of herstelcode',
        authRateLimited: 'Te veel mislukte pogingen. Probeer het opnieuw over {{seconds}} s.',
        verifyPasswordChange: 'Wachtwoordwijziging verifiëren',
        verifyPasswordChangeDesc:
          'Voer je huidige wachtwoord en hetzij een authenticatiecode of een ongebruikte herstelcode in voordat je het wachtwoord wijzigt.',
        verifyPasswordChangeFailed:
          'Kan het wachtwoord en de authenticatiecode of herstelcode niet verifiëren.',
        verifyAndEnable: 'Verifiëren en inschakelen',
        scanTwoFactorQRCode:
          'Scan deze QR-code met je authenticator-app en voer daarna de code in.',
        showSetupKey: 'Instelsleutel tonen',
        hideSetupKey: 'Instelsleutel verbergen',
        setupKey: 'Instelsleutel',
        twoFactorSetupFailed: 'Kan het instellen van tweefactorauthenticatie niet starten.',
        twoFactorVerifyFailed: 'Kan de authenticatiecode niet verifiëren.',
        twoFactorDisableFailed: 'Kan tweefactorauthenticatie niet uitschakelen.',
        disableTwoFactorDesc:
          'Voer je huidige wachtwoord en hetzij een authenticatiecode of een ongebruikte herstelcode in. Het uitschakelen van tweefactorauthenticatie maakt alle herstelcodes ongeldig.',
        updateBtn: 'Wijzigen',
        logoutBtn: 'Afmelden',
        logoutDesc: 'Weet u zeker dat u wilt uitloggen?',
        okBtn: 'Ja',
        cancelBtn: 'Nee',
        twoFactorProtection: {
          title: 'Geavanceerde 2FA-bescherming',
          description:
            'Kies welke gevoelige acties een tweede factor vereisen wanneer bescherming wordt afgedwongen. De schakelaar "Directe webaanmelding" doet het tegenovergestelde: je kunt dan alleen aanmelden met een authenticatiecode of ongebruikte herstelcode.',
          login: 'Directe webaanmelding',
          webTerminal: 'Webterminal',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API-sleutel',
          targetPower: 'Voeding van doelcomputer',
          nanoKVMRestart: 'NanoKVM opnieuw starten',
          deleteImage: 'Image verwijderen',
          scripts: 'Scriptbewerkingen',
          screenTimelapse: 'Schermtimelapse',
          loginScope:
            'Hiermee kunt u zich op het web aanmelden met alleen een authenticatiecode of ongebruikte herstelcode in plaats van een gebruikersnaam en wachtwoord. Dit vermindert de aanmeldbeveiliging.',
          webTerminalScope: 'De webterminal openen.',
          sshScope: 'De status van de SSH-service wijzigen.',
          tailscaleScope: 'Tailscale installeren, verbinden of wijzigen.',
          frp: 'FRP',
          frpScope: 'FRP installeren, verbinden of wijzigen.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Cloudflare Tunnel installeren, verbinden of wijzigen.',
          zerotierScope: 'ZeroTier installeren, verbinden of wijzigen.',
          netbirdScope: 'NetBird installeren, verbinden of wijzigen.',
          mcpScope:
            'De MCP API-sleutel weergeven, kopiëren of opnieuw genereren, en MCP in- of uitschakelen.',
          targetPowerScope:
            'De doelcomputer afsluiten of opnieuw starten met de fysieke aan/uitknop.',
          nanoKVMRestartScope: 'NanoKVM zelf opnieuw starten.',
          deleteImageScope: 'Een opgeslagen ISO- of IMG-image permanent verwijderen.',
          scriptsScope: 'Scripts uitvoeren, uploaden of verwijderen.',
          screenTimelapseScope:
            'Schermtimelapse-opname inschakelen, opnamen bekijken en opnamen verwijderen.',
          authorizeTitle: 'Beschermingswijzigingen autoriseren',
          authorizeDesc:
            'Voer een authenticatiecode of ongebruikte herstelcode in. Deze autorisatie duurt vijf minuten.',
          authorizeFailed: 'Kan de authenticatiecode of herstelcode niet verifiëren.',
          loadFailed: 'Kan beschermingsinstellingen niet laden.',
          updateFailed: 'Kan beschermingsinstellingen niet opslaan.',
          retry: 'Opnieuw proberen',
          disableScriptsWarningTitle: 'Scriptbescherming uitschakelen?',
          disableScriptsWarningDesc:
            'Zonder bescherming kan een kwaadwillend script verhoogde toegang tot dit apparaat verkrijgen en de 2FA-bescherming omzeilen of verwijderen.',
          cancelDisabling: 'Annuleren',
          continueDisabling: 'Doorgaan met uitschakelen',
          continueDisablingCountdown: 'Doorgaan met uitschakelen ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Gids voor tweefactorauthenticatie',
          scope: 'Tweefactorauthenticatie voegt een extra beveiligingslaag toe aan uw account.',
          whatTitle: 'Wat is 2FA?',
          whatDesc:
            'Tweefactorauthenticatie (2FA) beschermt je account met een tijdgebonden code uit een authenticator-app. Standaard is de code naast je wachtwoord vereist, dus zelfs als je wachtwoord is gecompromitteerd, blijft je account beschermd. Als directe webaanmelding is ingeschakeld, meldt de code je alleen aan.',
          appsTitle: 'Aanbevolen apps',
          appsDesc:
            'U kunt Google Authenticator of Microsoft Authenticator gebruiken. Beide zijn gratis en werken op iOS en Android.',
          setupTitle: 'Instellen',
          setupDesc:
            'Open uw authenticator-app, scan de QR-code die tijdens het instellen wordt weergegeven of voer de instelsleutel handmatig in. Voer daarna de 6-cijferige code uit de app in om 2FA te activeren.',
          recoveryTitle: 'Herstelcodes',
          recoveryDesc:
            'Herstelcodes zijn eenmalige back-upcodes. Als u geen toegang meer hebt tot uw authenticator-app, kunt u een herstelcode gebruiken om in te loggen. Bewaar ze direct na het genereren op een veilige plaats.',
          logoutNotice:
            'Als u 2FA inschakelt, worden alle actieve sessies afgemeld, inclusief deze sessie. U moet zich op elk apparaat opnieuw aanmelden.'
        }
      }
    },
    error: {
      title: 'Er is een probleem opgetreden',
      refresh: 'Vernieuwen'
    },
    pictureInPicture: {
      title: 'Beeld-in-beeld',
      enter: 'Beeld-in-beeld inschakelen',
      exit: 'Beeld-in-beeld uitschakelen',
      error:
        'Kan beeld-in-beeld niet starten. Wacht tot de video is geladen en probeer het opnieuw.'
    },
    fullscreen: {
      toggle: 'Volledig scherm schakelen'
    },
    menu: {
      collapse: 'Menu samenvouwen',
      expand: 'Menu uitvouwen'
    }
  }
};

export default nl;
