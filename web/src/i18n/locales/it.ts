const it = {
  translation: {
    storage: {
      cards: { mount: 'Monta disco', share: 'Condivisione file' },
      title: 'Immagini e condivisione file',
      empty: 'Nessuna immagine',
      notAttached: 'Niente montato',
      attached: 'Montato: {{name}}',
      eject: 'Espelli',
      mount: 'Monta',
      mountMode: 'Modalità di montaggio',
      mountModeDisk: 'Archiviazione di massa',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Apri il gestore file',
      refresh: 'Aggiorna',
      delete: 'Elimina',
      deleteTitle: 'Elimina supporto',
      deleteConfirm: 'Eliminare {{name}}? I file che contiene andranno persi.',
      downloadImage: "Scarica l'immagine",
      typePartition: 'File condivisi',
      volumeLabel: 'Disco rimovibile',
      groups: {
        share: 'Condivisione file',
        images: 'Immagini'
      },
      failed: 'Operazione non riuscita',
      hidOnly: 'Modalità solo HID: il disco virtuale non è disponibile',
      disabled: 'Il disco virtuale è disattivato (Impostazioni - Dispositivo)',
      force: {
        title: 'Riprendere il supporto con la forza?',
        desc: 'La macchina remota sta ancora usando questo supporto. Riprenderlo rienumera il dispositivo USB e può perdere tutto ciò che non ha ancora scritto sul disco.',
        menuHint:
          'La macchina remota sta usando questo supporto. Apri il gestore file per riprenderlo con la forza.'
      },
      tips: {
        title: 'Come caricare',
        usb1: 'Collega il NanoKVM al tuo computer tramite USB.',
        usb2: 'In questa pagina, monta i file condivisi in modalità archiviazione di massa.',
        usb3: 'Apri il disk virtuale sul tuo computer e copia il file immagine nella directory principale del disk.',
        usb4: "Espelli il disco al termine della copia: l'immagine comparirà nell'elenco.",
        scp1: 'Assicurati che il NanoKVM e il tuo computer siano sulla stessa rete locale.',
        scp2: 'Apri un terminale sul tuo computer e usa il comando SCP per caricare il file immagine nella directory /data del NanoKVM.',
        scp3: 'Esempio: scp il-tuo-percorso-immagine root@il-tuo-ip-nanokvm:/data'
      },
      job: {
        running: 'Elaborazione di {{name}}'
      },
      enableDisk: {
        title: 'Attivare il disco virtuale?',
        desc: 'Per montare un supporto occorre il disco virtuale, che al momento è disattivato. Attivarlo ora? La tastiera e il mouse della macchina remota si disconnetteranno per un istante.',
        menuHint: 'Il disco virtuale è disattivato. Attivalo in Impostazioni - Dispositivo.'
      },
      add: {
        title: 'Aggiungi supporto',
        upload: 'Carica un file ISO o IMG',
        fromUrl: 'Scarica da un URL',
        url: "URL dell'immagine",
        urlRequired: 'Inserisci un URL',
        onlyImages: 'Si possono montare solo file .iso e .img',
        dropHint: 'Trascina qui un file ISO o IMG, oppure fai clic per scegliere un file',
        cdromOnlyIso: 'Un CD-ROM può contenere solo un’immagine ISO 9660 / UDF',
        uploadFailed: 'Caricamento non riuscito',
        downloadFailed: 'Download non riuscito',
        uploading: 'Caricamento di {{name}}',
        downloading: 'Download di {{name}}',
        done: '{{name}} è pronto',
        checksumFailed: "L'immagine non corrisponde al checksum SHA-256.",
        blocked:
          'La macchina remota sta usando il disco. Espelli in modo sicuro l’unità USB virtuale sul computer remoto, poi disattiva l’interruttore Disco virtuale in Impostazioni > Dispositivo.'
      },
      files: {
        reasonNotInitialized:
          'L’unità USB virtuale non è inizializzata. Inizializzala in Impostazioni > Dispositivo prima di condividere file.',
        selectMedium: 'Seleziona un supporto per sfogliare i file che contiene.',
        reasonAttached: 'Questo supporto è collegato alla macchina remota.',
        reasonAttachedHint:
          'Espelli in modo sicuro l’unità USB virtuale sul computer remoto, poi disattiva l’interruttore Disco virtuale in Impostazioni > Dispositivo.',
        reasonNotBrowsable:
          'Questo supporto contiene un filesystem che il gestore file non può aprire.',
        reasonNoLoop: 'Questo dispositivo non può montare immagini disco (nessun supporto loop).',
        reasonISOImage:
          "Un'immagine ISO 9660 non può essere sfogliata qui. Può comunque essere montata sulla macchina remota.",
        empty: 'Questa cartella è vuota',
        noMatch: 'Nessun file corrisponde alla ricerca.',
        free: '{{size}} liberi',
        search: 'Cerca in questa cartella',
        sort: {
          modified: 'Modificato',
          name: 'Nome',
          size: 'Dimensione',
          type: 'Tipo',
          asc: 'Crescente',
          desc: 'Decrescente'
        },
        view: {
          list: 'Vista elenco',
          grid: 'Vista icone'
        },
        folder: 'Cartella',
        itemCount: 'Elementi: {{count}}',
        itemCount_one: '{{count}} elemento',
        itemCount_other: '{{count}} elementi',
        dropHint: 'Trascina qui i file per caricarli in questa cartella',
        up: 'Vai alla cartella superiore',
        uploadButton: 'Carica',
        newFolder: 'Nuova cartella',
        newFolderPlaceholder: 'Nome della cartella',
        create: 'Crea',
        upload: 'Carica un file nel supporto',
        deleteFolderTitle: 'Elimina cartella',
        deleteFolderConfirm:
          "Eliminare la cartella {{name}} e tutto il suo contenuto? L'operazione non può essere annullata.",
        uploading: 'Caricamento…',
        uploadFailed: 'Caricamento non riuscito',
        download: 'Scarica',
        delete: 'Elimina',
        deleteTitle: 'Elimina file',
        deleteConfirm: 'Eliminare {{name}} dal supporto?'
      },
      yes: 'Sì',
      no: 'No'
    },
    head: {
      storage: 'Immagini e condivisione file',
      desktop: 'Desktop Remoto',
      login: 'Accesso',
      changePassword: 'Cambia Password',
      terminal: 'Terminale',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed: 'Impossibile copiare negli appunti. Consenti l’accesso agli appunti e riprova.'
    },
    auth: {
      login: 'Accesso',
      language: 'Lingua',
      placeholderUsername: 'Inserisci il nome utente',
      placeholderPassword: 'Inserisci la password',
      placeholderPassword2: 'Inserisci nuovamente la password',
      noEmptyUsername: 'Il nome utente non può essere vuoto',
      noEmptyPassword: 'La password non può essere vuota',
      noAccount:
        'Impossibile ottenere informazioni utente, aggiorna la pagina o reimposta la password',
      invalidUser: 'Nome utente o password non validi',
      error: 'Errore imprevisto',
      twoFactorTitle: 'Verifica richiesta',
      twoFactorDesc:
        'Inserisci il tuo codice di autenticazione o un codice di recupero non usato per accedere.',
      sensitiveActionDesc:
        'Inserisci un codice di autenticazione o un codice di recupero non usato per continuare.',
      placeholderAuthCode: 'Codice di autenticazione',
      noEmptyAuthCode: 'Codice di autenticazione obbligatorio',
      verify: 'Verifica',
      backToLogin: 'Torna al login',
      verificationFailed: 'Verifica non riuscita',
      twoFactorEnabledLogout: {
        message:
          "L'autenticazione a due fattori è stata attivata. Per sicurezza, tutte le sessioni sono state disconnesse. Accedi di nuovo."
      },
      rateLimited: 'Troppi tentativi non riusciti. Riprova tra {{seconds}} s.',
      recoveryCodeUsed: 'Il codice di recupero usato per questo accesso è stato invalidato.',
      changePassword: 'Cambia Password',
      changePasswordDesc:
        'Per la sicurezza del tuo dispositivo, modifica la password di accesso web.',
      passwordVerificationRequired:
        'Torna alla pagina account nelle impostazioni e verifica di nuovo questa modifica della password.',
      passwordVerificationExpired:
        'La verifica della modifica della password è scaduta. Torna alla pagina account nelle impostazioni e verifica di nuovo.',
      returnToSettings: 'Torna alle impostazioni',
      differentPassword: 'Le password non corrispondono',
      illegalUsername: 'Il nome utente contiene caratteri non validi',
      illegalPassword: 'La password contiene caratteri non validi',
      forgetPassword: 'Hai dimenticato la password',
      ok: 'OK',
      cancel: 'Annulla',
      savePassword: 'Salva password',
      loginButtonText: 'Accedi',
      tips: {
        reset1: 'Account Web predefinito:',
        reset2: 'Account predefinito SSH:',
        reset3: 'Per reimpostare le password, fare riferimento a ',
        changeReset: 'Per reimpostare la password di accesso Web, fare riferimento a ',
        resetLink: 'Reimpostare'
      }
    },
    wifi: {
      errors: {
        connect:
          'Impossibile connettersi al Wi-Fi. Controlla il nome della rete, la password e lo stato Wi-Fi del dispositivo, quindi riprova.'
      },
      title: 'Wi-Fi',
      description: 'Configura Wi-Fi per NanoKVM',
      submitted: {
        title: 'Un attimo: NanoKVM si sta collegando al tuo Wi-Fi',
        tipsTitle: 'E adesso?',
        tipSwitch: 'Ora puoi ricollegare il telefono alla tua rete Wi-Fi abituale.',
        tipCheck:
          'Tra poco controlla lo schermo del NanoKVM per vedere il nuovo indirizzo IP e aprilo nel browser.',
        tipRetry: 'Se dopo un po’ non succede nulla, verifica nome rete e password e riprova.'
      },
      invalidMode:
        'La modalità corrente non supporta la configurazione di rete. Vai al tuo dispositivo e abilita la modalità di configurazione Wi-Fi.',
      confirmBtn: 'OK',
      finishBtn: 'Finito',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: "Parola d'accesso",
      steps: {
        settings: 'Impostazioni',
        wifi: 'Wi-Fi',
        enable: 'Abilita'
      },
      scan: {
        title: 'Scegli una rete Wi-Fi',
        description: 'Seleziona una rete vicina per continuare.',
        loading: 'Ricerca delle reti Wi-Fi vicine…',
        empty: 'Nessuna rete Wi-Fi vicina trovata.',
        failed: 'Impossibile eseguire la scansione. Inserisci i dati manualmente.',
        retryBtn: 'Scansiona di nuovo',
        manualBtn: 'Inserisci rete manualmente',
        backBtn: 'Torna alle reti vicine',
        connectTitle: 'Connetti a {{ssid}}',
        connecting: 'Connessione…',
        viewSimple: 'Passa alla vista semplice',
        viewDetailed: 'Passa alla vista dettagliata'
      },
      ap: {
        authTitle: 'Autenticazione richiesta',
        authDescription: 'Inserisci la password AP per continuare',
        authFailed: 'Password AP non valida',
        passPlaceholder: 'Password AP',
        verifyBtn: 'Verifica'
      },
      qr: {
        cameraBtn: 'Scatta foto',
        pickBtn: 'Scegli screenshot',
        scanHint:
          'Scatta una foto di un codice QR Wi-Fi, oppure scegli uno screenshot, come quello mostrato da un telefono quando condivide il suo Wi-Fi.',
        decoding: 'Lettura del codice QR…',
        notFound:
          'Nessun codice QR trovato in questa foto. Centra il codice e mettilo a fuoco, poi riprova.',
        invalid: 'Questo codice QR non contiene dati Wi-Fi utilizzabili.',
        pickerUnavailable:
          'Non si è aperto nulla? Questa finestra di accesso potrebbe bloccare la fotocamera e le foto. Apri invece questo indirizzo nel browser del telefono:',
        confirmTitle: 'Connettere NanoKVM a {{ssid}}?',
        confirmDescription:
          'Questi dati di rete provengono da un codice QR. Controllali prima di connetterti.',
        openNetwork: 'Questa rete non ha password.',
        connectBtn: 'Connetti',
        otherBtn: "Scegli un'altra rete"
      }
    },
    notification: {
      browser: {
        title: 'Raccomandazione del browser',
        description:
          'Per la migliore esperienza utente, consigliamo di utilizzare il browser Chrome. Anche se ci impegniamo a garantire la compatibilità con tutti i principali browser, alcune funzionalità avanzate potrebbero non funzionare come previsto al di fuori di Chrome.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Schermo',
      video: 'Modalità video',
      quality: 'Qualità',
      qualityAuto: 'Automatico',
      qualityLossless: 'Senza perdita',
      qualityHigh: 'Alto',
      qualityMedium: 'Medio',
      qualityLow: 'Basso',
      scale: 'Scala',
      noSignal: 'Nessun segnale video',
      captureDisabled:
        'L\'acquisizione video è disattivata. Attivala in "Impostazioni > Dispositivo > Acquisizione video".',
      inconsistentVideoMode: "Riproduzione in pausa. È in riproduzione un'altra modalità video.",
      bandwidthWarning:
        'La larghezza di banda della rete potrebbe non essere sufficiente per le impostazioni video attuali. Prova a ridurre il bitrate o la frequenza dei fotogrammi.',
      webrtcConnectionFailed:
        'Impossibile stabilire la connessione WebRTC. Passa a una modalità video Direct.',
      webrtcUnavailable: {
        title: 'WebRTC non è disponibile',
        description: 'Attiva WebRTC nel browser oppure passa alla modalità video Direct.'
      },
      upload: 'Carica',
      advanced: 'Avanzato',
      loading: 'Caricamento'
    },
    volume: {
      title: 'Volume'
    },
    microphone: {
      title: 'Microfono',
      notMounted: 'Microfono non montato',
      tip1: "Il microfono virtuale non è montato sull'host remoto. Desideri montarlo e abilitarlo ora?",
      tip2: 'Puoi anche gestirlo successivamente in "Impostazioni > Dispositivo > Audio virtuale".',
      confirm: 'Abilita',
      cancel: 'Annulla',
      noDevice: 'Nessun microfono rilevato su questo dispositivo. Collegane uno e riprova.',
      permissionDenied:
        "L'accesso al microfono è stato negato. Consentilo nelle impostazioni del browser.",
      notReadable: "Il microfono è utilizzato da un'altra applicazione o non può essere letto.",
      overconstrained: 'Il microfono non supporta i parametri audio richiesti.',
      securityBlocked:
        "L'accesso al microfono è bloccato su questa pagina. Controlla le impostazioni di sicurezza del browser.",
      aborted: "L'avvio del microfono è stato interrotto. Riprova.",
      unsupported:
        "Questo browser o metodo di connessione non supporta l'acquisizione dal microfono.",
      enableFailed: 'Impossibile abilitare il microfono. Riprova.',
      disconnected: 'Il microfono è stato disconnesso e la trasmissione è stata disattivata.'
    },
    power: {
      title: 'Accensione',
      showConfirm: 'Conferma',
      showConfirmTip: 'Le operazioni di accensione richiedono una conferma aggiuntiva',
      powerShort: 'Accensione (clic breve)',
      powerLong: 'Accensione (clic lungo)',
      powerConfirm: 'Eseguire l’operazione di accensione?',
      okBtn: 'Sì',
      cancelBtn: 'Annulla'
    },
    keyboard: {
      errors: {
        paste:
          'Impossibile inviare il testo al dispositivo remoto. Controlla la connessione KVM e riprova.'
      },
      title: 'Tastiera',
      paste: 'Incolla',
      tips: 'Sono supportati solo lettere e simboli standard della tastiera',
      placeholder: 'Inserisci testo',
      submit: 'Invia',
      virtual: 'Tastiera',
      swapCommandCtrl: 'Scambia Cmd e Ctrl',
      shortcut: {
        title: 'Scorciatoie',
        custom: 'Personalizzato',
        capture: 'Fai clic qui per acquisire il collegamento',
        clear: 'Cancella',
        save: 'Salva',
        captureTips:
          'La cattura dei tasti di sistema (come il tasto Windows) richiede l’autorizzazione a schermo intero.',
        enterFullScreen: 'Attiva/disattiva la modalità a schermo intero.'
      },
      leaderKey: {
        title: 'Tasto Leader',
        desc: "Ignora le restrizioni del browser e invia collegamenti di sistema direttamente all'host remoto.",
        howToUse: 'Come usare',
        simultaneous: {
          title: 'Modalità simultanea',
          desc1: 'Tieni premuto il tasto Leader, quindi premi la scorciatoia.',
          desc2: 'Intuitivo, ma potrebbe entrare in conflitto con le scorciatoie di sistema.'
        },
        sequential: {
          title: 'Modalità sequenziale',
          desc1:
            'Premi il tasto Leader → premi la scorciatoia in sequenza → premi di nuovo il tasto Leader.',
          desc2: 'Richiede più passaggi, ma evita completamente i conflitti di sistema.'
        },
        enable: 'Abilita tasto Leader',
        tip: 'Quando assegnato come tasto Leader, questo tasto funziona solo come attivatore di scorciatoie e perde il comportamento predefinito.',
        placeholder: 'Premi il tasto Leader',
        shiftRight: 'Shift destro',
        ctrlRight: 'Ctrl destro',
        metaRight: 'Win destro',
        submit: 'Invia',
        recorder: {
          rec: 'REC',
          activate: 'Attiva i tasti',
          input: 'Premi la scorciatoia...'
        }
      }
    },
    mouse: {
      title: 'Mouse',
      mode: 'Modalità mouse',
      absolute: 'Mouse a inseguimento',
      absoluteAndroid: 'Mouse a inseguimento (maggiore compatibilità Android)',
      touchScreen: 'Schermo multi-touch',
      relative: 'Mouse esclusivo',
      direction: 'Direzione della rotellina',
      scrollUp: "Scorri verso l'alto",
      scrollDown: 'Scorri verso il basso',
      speed: 'Velocità della rotellina',
      fast: 'Veloce',
      slow: 'Lento',
      requestPointer:
        'Usando la modalità relativa. Clicca sul desktop per ottenere il puntatore del mouse.',
      repairIphoneDrag: 'Ripara trascinamento iPhone',
      resetHid: 'Ripristina HID',
      touchpadGuide: {
        title: 'Guida touchpad',
        scope: 'Si applica all’input del puntatore normale quando la modalità mouse è Relativa.',
        swipeTitle: 'Scorri per spostare',
        swipeDesc: 'Scorri nell’area attiva dello schermo per spostare il puntatore remoto.',
        tapTitle: 'Tocca per fare clic',
        tapDesc: 'Un tocco breve invia un clic sinistro.',
        holdTitle: 'Tieni premuto il pulsante sinistro',
        holdDesc:
          'Tocca e mantieni il dito fermo per circa 1 secondo per tenere premuto il pulsante sinistro del mouse.',
        dragTitle: 'Muovi dopo la pressione per trascinare',
        dragDesc:
          'Quando la pressione è attiva, muovi il dito per trascinare con il pulsante sinistro premuto.'
      },
      hidOnly: {
        errors: {
          load: 'Impossibile caricare lo stato della modalità HID-Only. Riprova tra poco.',
          update:
            'Impossibile confermare se la modalità HID-Only è stata aggiornata. Riapri questa impostazione per controllare lo stato attuale, quindi riprova.'
        },
        title: 'Modalità solo HID',
        desc: 'Se il mouse e la tastiera smettono di rispondere e il ripristino di HID non aiuta, potrebbe trattarsi di un problema di compatibilità tra NanoKVM e il dispositivo. Prova ad abilitare la modalità HID-Only per una migliore compatibilità.',
        tip1: "L'abilitazione della modalità HID-Solo smonterà il disco U virtuale e la rete virtuale",
        tip2: "Nella modalità HID-Only, il montaggio dell'immagine è disabilitato",
        enable: 'Abilita la modalità HID-Solo',
        disable: 'Disabilita la modalità HID-Solo'
      }
    },
    preview: {
      title: 'Anteprima'
    },
    image: {
      storageReadOnly: 'Non è possibile scrivere nella memoria. Disattivare il disco virtuale.',
      storageBusy:
        'La memoria è occupata. Completare o annullare il trasferimento prima di cambiare lo stato del disco virtuale.',
      title: 'Immagini',
      loading: 'Caricamento...',
      empty: 'Nessun risultato',
      upload: 'Carica immagine',
      dropFile: "Trascina qui un'immagine o fai clic per selezionare un file",
      uploadAction: 'Carica',
      cancelAction: 'Annulla',
      mount: 'Monta immagine',
      mounted: 'Montato',
      uploadError: 'Caricamento non riuscito, riprova',
      uploadInsufficientStorage: 'Spazio di archiviazione insufficiente',
      uploadTooLarge: 'L’immagine supera il limite di dimensione',
      cancelUploadError: 'Impossibile annullare il caricamento',
      mountMode: 'Modalità di montaggio',
      massStorage: 'Archiviazione di massa',
      cdRom: 'CD ROM',
      readOnly: 'Sola lettura',
      mountFailed: 'Montaggio immagine fallito',
      mountDesc:
        "In alcuni sistemi, è necessario espellere il disco virtuale sull'host remoto prima di montare l'immagine.",
      unmountFailed: 'Smontaggio fallito',
      unmountDesc:
        "In alcuni sistemi, è necessario espellere manualmente sull'host remoto prima di smontare l'immagine.",
      refresh: "Aggiorna l'elenco delle immagini",
      download: 'Scarica immagine',
      attention: 'Attenzione',
      deleteConfirm: 'Sei sicuro di voler eliminare questa immagine?',
      deleteFailed: 'Impossibile eliminare l’immagine',
      okBtn: 'Sì',
      cancelBtn: 'Annulla'
    },
    script: {
      title: 'Script',
      upload: 'Carica',
      run: 'Esegui',
      runBackground: 'Esegui in Background',
      runFailed: 'Esecuzione fallita',
      uploadFailed: 'Impossibile caricare lo script',
      deleteFailed: 'Impossibile eliminare lo script',
      attention: 'Attenzione',
      delDesc: 'Sei sicuro di voler eliminare questo file?',
      confirm: 'Sì',
      cancel: 'Annulla',
      delete: 'Elimina',
      close: 'Chiudi'
    },
    terminal: {
      title: 'Terminale',
      confirm: 'OK',
      submit: 'Invia',
      username: 'Nome utente',
      password: 'Password',
      authCode: 'Codice di autenticazione',
      authorizeTitle: 'Autorizza terminale Web',
      codeAccountHint: 'Inserisci un codice di autenticazione o un codice di recupero non usato.',
      authFailed: 'Autenticazione non riuscita'
    },
    wol: {
      errors: {
        send: 'Impossibile inviare il comando di attivazione. Controlla l’indirizzo MAC e la connessione di rete, quindi riprova.',
        loadFailed:
          'Impossibile caricare l’elenco dei dispositivi Wake-on-LAN. Controlla che NanoKVM sia online, quindi riprova.',
        renameFailed:
          'Impossibile rinominare questo dispositivo. Controlla l’indirizzo MAC, quindi riprova.',
        deleteFailed: 'Impossibile eliminare questo dispositivo. Riprova tra poco.'
      },
      title: 'Wake-on-LAN',
      sending: 'Invio comando...',
      sent: 'Comando inviato',
      input: 'Inserisci il MAC',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (facoltativo)',
      sha256Placeholder: 'Inserisci un checksum SHA-256 di 64 caratteri',
      invalidSHA256: 'SHA-256 deve essere una stringa esadecimale di 64 caratteri',
      title: 'Scaricatore di immagini',
      input: "Inserisci un'immagine remota URL",
      ok: 'OK',
      disabled: "La partizione /data è RO, quindi non possiamo scaricare l'immagine",
      downloadingFile: 'Download in corso: {{file}}',
      downloadingProgress: 'Download in corso ({{percentage}}): {{file}}',
      failed: 'Non riuscito',
      insufficientStorage: 'Spazio di archiviazione insufficiente',
      remaining: 'Tempo rimanente: {{time}}',
      cancel: 'Annulla',
      cancelFailed: 'Impossibile annullare il download',
      pause: 'Pausa',
      resume: 'Riprendi',
      pauseFailed: 'Impossibile mettere in pausa il download',
      resumeFailed: 'Impossibile riprendere il download'
    },
    settings: {
      back: 'Indietro',
      close: 'Chiudi',
      ai: {
        errors: {
          loadServices:
            'Impossibile caricare lo stato dei servizi AI. Verifica che NanoKVM sia online e riprova.',
          loadMcpConfig:
            'Impossibile caricare la configurazione del servizio MCP. Riprova tra poco.',
          updateMcpService:
            'Impossibile aggiornare lo stato del servizio MCP. Controlla lo stato attuale prima di riprovare.',
          accessMcpApiKey:
            'Impossibile recuperare la chiave API MCP. Riprova e completa la verifica se richiesta.',
          regenerateMcpApiKey: 'Impossibile rigenerare la chiave API MCP. Riprova tra poco.',
          regeneratedMcpApiKeyUnavailable:
            'La chiave API è stata rigenerata, ma non può essere visualizzata. Prova a visualizzarla di nuovo. La chiave precedente non è più valida.',
          loadMemoryFabricState:
            'Impossibile caricare lo stato del servizio Memory Fabric. Riprova tra poco.',
          updateMemoryFabricService:
            'Impossibile aggiornare lo stato del servizio Memory Fabric. Controlla lo stato attuale prima di riprovare.',
          loadScreenTimelapseState:
            'Impossibile caricare lo stato del servizio timelapse dello schermo. Riprova tra poco.',
          updateScreenTimelapseService:
            'Impossibile aggiornare lo stato del servizio timelapse dello schermo. Controlla lo stato attuale prima di riprovare.',
          openScreenTimelapse:
            'Impossibile aprire la riproduzione timelapse dello schermo. Riprova tra poco.',
          loadScreenTimelapseRecordings:
            'Impossibile caricare le registrazioni timelapse dello schermo. Riprova tra poco.',
          playScreenTimelapse:
            'Questa registrazione timelapse non può essere riprodotta ora. Seleziona di nuovo la data o riprova più tardi.',
          screenTimelapseFullscreen:
            'Impossibile modificare la modalità a schermo intero. Controlla le autorizzazioni del browser e riprova.'
        },
        title: 'AI',
        mcpService: 'Servizio MCP',
        mcpServiceDesc:
          'Gateway unificato autenticato per controllo, contenuti multimediali e strumenti Memory Fabric abilitati',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Elabora localmente memoria contestuale e OCR degli agenti',
        memoryFabricRemoteRequiresMcp:
          'Abilita il servizio MCP qui sopra per rendere disponibili da remoto gli strumenti di memoria e OCR.',
        memoryFabricRemoteAvailable:
          'Gli strumenti di memoria e OCR sono disponibili tramite l’endpoint /api/mcp qui sopra con la stessa chiave API.',
        memoryFabricConfig: {
          title: 'Configurazione',
          description: 'Gestisci solo le impostazioni del modello predefinito di Memory Fabric.',
          loadFailed: 'Impossibile caricare la configurazione di Memory Fabric.',
          saveFailed: 'Impossibile salvare la configurazione di Memory Fabric.',
          saved: 'Configurazione salvata.',
          savedRestarted: 'Configurazione salvata e Memory Fabric riavviato.',
          noChanges: 'Nessuna modifica alla configurazione.',
          reload: 'Ricarica',
          save: 'Salva',
          unsaved: 'Modifiche non salvate',
          conflictTitle: 'La configurazione è cambiata',
          conflictDescription:
            'La configurazione è stata modificata altrove. Ricarica la versione più recente prima di salvarla nuovamente.',
          conflictReload: 'Carica la versione più recente',
          discardTitle: 'Scartare le modifiche non salvate?',
          discardDescription:
            'Ricaricando, il modulo corrente verrà sostituito con l’ultima configurazione salvata.',
          discardConfirm: 'Scarta e ricarica',
          sections: {
            models: 'Modelli'
          },
          openAICompatibleHint:
            'Supporta i provider di modelli compatibili con OpenAI. Inserisci il nome del modello, l’URL di base e la chiave API forniti dal provider.',
          fields: {
            model: 'Modello',
            baseUrl: 'URL di base',
            maxTokens: 'Numero massimo di token',
            timeoutSeconds: 'Timeout (secondi)'
          },
          apiKey: {
            title: 'Chiave API',
            configured: 'Configurata',
            notConfigured: 'Non configurata',
            keep: 'Mantieni',
            replace: 'Sostituisci',
            clear: 'Rimuovi',
            replacement: 'Nuova chiave API',
            replacementPlaceholder: 'Inserisci la nuova chiave API',
            clearHint:
              'La chiave API salvata verrà rimossa; la variabile di ambiente potrà comunque fornire una chiave.',
            required: 'Inserisci una nuova chiave API.'
          },
          validation: {
            required: 'Questo campo è obbligatorio.',
            invalidUrl: 'Inserisci un URL HTTP o HTTPS valido.'
          }
        },
        screenTimelapseService: 'Timelapse schermo',
        screenTimelapseServiceDesc:
          'Riproduci la cronologia dello schermo di Memory Fabric e cerca nei contenuti acquisiti',
        screenTimelapseSearchRequiresMemoryFabric:
          'Per cercare nei contenuti registrati, attiva anche Memory Fabric.',
        screenTimelapseOpen: 'Apri',
        screenTimelapseSearchPlaceholder: 'Cerca contenuti nel video...',
        screenTimelapseNoVideos: 'Nessun video timelapse trovato',
        screenTimelapseNoResults: 'Nessun fotogramma corrispondente trovato',
        screenTimelapseMultipleResults: 'Risultati della ricerca',
        screenTimelapseUnavailable: 'Non disponibile',
        screenTimelapsePlayerUnsupported: 'Questo browser non può riprodurre video timelapse',
        screenTimelapseSearchFailed: 'Ricerca non riuscita',
        screenTimelapseLoading: 'Caricamento timelapse...',
        screenTimelapseSearchAction: 'Cerca nel timelapse',
        screenTimelapsePlay: 'Riproduci timelapse',
        screenTimelapsePause: 'Metti in pausa timelapse',
        screenTimelapseEnterFullscreen: 'Entra a schermo intero',
        screenTimelapseExitFullscreen: 'Esci da schermo intero',
        screenTimelapseFullscreenUnsupported: 'Questo browser non può entrare a schermo intero',
        screenTimelapsePreviousDay: 'Giorno precedente',
        screenTimelapseNextDay: 'Giorno successivo',
        screenTimelapseDeleteDay: 'Elimina la registrazione del {{date}}',
        screenTimelapseDelete: 'Elimina',
        screenTimelapseDeleteConfirmTitle: 'Eliminare la registrazione timelapse?',
        screenTimelapseDeleteConfirmDesc:
          'La registrazione del {{date}} occupa {{size}}. Questa azione non può essere annullata.',
        screenTimelapseDeleteSuccess: 'Registrazione del {{date}} eliminata',
        screenTimelapseDeleteFailed: 'Impossibile eliminare la registrazione del {{date}}',
        endpoint: 'Punto di accesso',
        apiKey: 'Chiave API',
        regenerate: 'Rigenera',
        copyApiKey: 'Copia chiave API',
        copyEndpoint: 'Copia punto di accesso',
        showApiKey: 'Mostra chiave API',
        regenerateConfirmTitle: 'Rigenera chiave API',
        regenerateConfirmDesc:
          'I client MCP esistenti smetteranno di funzionare finché non saranno aggiornati con la nuova chiave. Vuoi continuare?',
        okBtn: 'Sì',
        cancelBtn: 'Annulla'
      },
      title: 'Impostazioni',
      about: {
        title: 'Informazioni su NanoKVM',
        information: 'Informazioni',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versione Applicazione',
        applicationTip: 'Versione applicazione web NanoKVM',
        arch: 'Architettura',
        device_number: 'Numero dispositivo',
        image: 'Versione Immagine',
        imageTip: 'Versione immagine di sistema NanoKVM',
        deviceKey: 'Chiave Dispositivo',
        community: 'Comunità',
        hostname: 'Nome host',
        hostnameUpdated: 'Nome host aggiornato. Riavviare per applicare.',
        ipType: {
          Wired: 'Cablato',
          Wireless: 'Senza fili',
          Other: 'Altro'
        }
      },
      appearance: {
        title: 'Aspetto',
        display: 'Schermo',
        language: 'Lingua',
        languageDesc: "Seleziona la lingua per l'interfaccia",
        languageChangeFailed: 'Impossibile cambiare lingua',
        webTitle: 'Titolo web',
        webTitleDesc: 'Personalizza il titolo della pagina web',
        menuBar: {
          title: 'Barra dei menu',
          mode: 'Modalità di visualizzazione',
          modeDesc: 'Visualizza la barra dei menu sullo schermo',
          modeOff: 'Spento',
          modeAuto: 'Nascondi automaticamente',
          modeAlways: 'Sempre visibile',
          icons: 'Icone dei sottomenu',
          showIcon: 'Mostra icona',
          activateOnConnect: 'Attiva alla connessione',
          stopSharing: 'Interrompi condivisione',
          iconsDesc: 'Visualizza le icone dei sottomenu nella barra dei menu'
        }
      },
      screen: {
        title: 'Schermo',
        videoMode: {
          title: 'Modalità video',
          description: 'Selezionare la codifica e il metodo di trasmissione'
        },
        edid: {
          description: 'Selezionare i parametri di visualizzazione',
          default: 'Predefinito',
          custom: 'Personalizzato',
          document: 'Documento',
          powerConfirmTitle: 'Conferma prima di aggiornare EDID',
          powerConfirmDescription:
            "Collega prima l'alimentazione PD. L'aggiornamento EDID potrebbe disconnettere brevemente il display; se dopo il dispositivo non riesce a rilevare correttamente il display, riavvialo manualmente.",
          powerConfirmOk: 'Continua',
          powerConfirmOkCountdown: 'Continua ({{seconds}})',
          powerConfirmCancel: 'Annulla',
          errors: {
            badRequest: 'Richiesta EDID non valida',
            invalidFileName: 'Seleziona un file EDID .bin con un nome valido',
            fileTooLarge: 'Il file EDID è troppo grande',
            invalidLength: 'L’EDID deve essere di 128, 256, 384 o 512 byte',
            invalidHeader: 'L’intestazione EDID non è valida',
            unsupportedVersion: 'Questa versione EDID non è supportata',
            extensionCountMismatch:
              'Il numero di blocchi di estensione EDID non corrisponde alla lunghezza del file',
            invalidChecksum: 'Il checksum EDID non è valido',
            invalidCta: 'La struttura dell’estensione CTA non è valida',
            currentCannotOverwrite:
              'L’EDID attualmente in uso non può essere sovrascritto. Seleziona prima un altro EDID',
            saveFailed: 'Impossibile salvare l’EDID',
            switchFailed: 'Impossibile cambiare EDID',
            uploadFailed: 'Impossibile caricare l’EDID',
            deleteFailed: 'Impossibile eliminare l’EDID',
            networkFailed: 'Errore di rete. Riprova'
          }
        },
        rateControlMode: {
          title: 'Modalità di controllo della velocità',
          description: 'Selezionare il metodo per controllare il bitrate',
          cbr: "CBR (bitrate costante): dà priorità alla stabilità della connessione. Riduce al minimo la balbuzie, ma la qualità dell'immagine potrebbe diminuire durante le scene ad alto movimento.",
          vbr: "VBR (Bitrate variabile): dà priorità alla chiarezza dell'immagine. Le scene ad alto movimento possono causare picchi di larghezza di banda, portando potenzialmente a latenza o balbettio."
        },
        bitrate: {
          title: 'Velocità in bit',
          description:
            "Bilancia la chiarezza dell'immagine con l'utilizzo della larghezza di banda",
          lower:
            "Valori più bassi riducono la latenza su reti scadenti ma compromettono la qualità dell'immagine, causando potenzialmente sfocature o artefatti.",
          higher:
            "Valori più alti migliorano la chiarezza dell'immagine ma consumano più larghezza di banda. Potrebbe aumentare la latenza se la connessione è lenta."
        },
        gop: {
          description: "Imposta l'intervallo tra i fotogrammi chiave completi",
          lower:
            "Valori più bassi inviano fotogrammi completi con maggiore frequenza. Ciò accelera il recupero dagli artefatti visivi ma consuma più larghezza di banda e riduce i dettagli dell'immagine.",
          higher:
            "Valori più alti ottimizzano per aggiornamenti incrementali. Ciò migliora i dettagli dell'immagine e consente di risparmiare larghezza di banda, ma il ripristino dagli artefatti visivi richiederà più tempo."
        },
        quality: {
          title: 'Qualità',
          description: "Imposta la qualità dell'immagine visualizzata"
        },
        fps: {
          auto: 'Automatico',
          description: 'Imposta la frequenza dei fotogrammi del display',
          tip1: "Impostando il valore su 0 si attiva la modalità automatica, sincronizzando il display con il frame rate dell'host remoto.",
          tip2: "La riduzione del frame rate riduce significativamente l'utilizzo della larghezza di banda. Diminuirlo manualmente durante l'instabilità della rete o quando non sono richiesti frame rate elevati."
        },
        scale: {
          title: 'Scala',
          description: 'Imposta il rapporto di scala del display'
        }
      },
      device: {
        title: 'Dispositivo',
        virtualAudioConfirm: {
          title: 'Conferma prima di cambiare l’audio virtuale',
          description:
            'Collega prima l’alimentazione PD. Il cambio dell’audio virtuale potrebbe disconnettere brevemente il display; se dopo il dispositivo non riesce a rilevare correttamente il display, riavvialo manualmente.',
          ok: 'Continua con il cambio',
          okCountdown: 'Continua con il cambio ({{seconds}})',
          cancel: 'Annulla'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Configura Wi-Fi',
          apMode: 'La modalità AP è abilitata, connettiti manualmente al Wi-Fi',
          others: 'Altri...',
          connect: 'Accedi al Wi-Fi',
          connectDesc1: "Inserisci l'ssid di rete e la password",
          connectDesc2: 'Inserisci la password per unirti a questa rete',
          disconnect: 'Sei sicuro di disconnettere la rete?',
          ssid: 'Nome',
          password: 'Password',
          joinBtn: 'Iscriviti',
          confirmBtn: 'OK',
          cancelBtn: 'Annulla',
          preset: {
            title: 'Wi-Fi preimpostata',
            description:
              'Preimposta una Wi-Fi di destinazione; il dispositivo vi si connette automaticamente all’avvio',
            empty: 'Nessuna',
            addBtn: 'Aggiungi preimpostazione',
            removeBtn: 'Rimuovi preimpostazione',
            saveTitle: 'Preimposta Wi-Fi',
            saveDesc:
              'Salva ora la rete di destinazione; il dispositivo si connette automaticamente quando la rileva all’avvio',
            saveBtn: 'Salva',
            passwordPlaceholder: 'Password (lascia vuoto per reti aperte)',
            saveFailed: 'Impossibile salvare la Wi-Fi preimpostata',
            removeFailed: 'Impossibile rimuovere la Wi-Fi preimpostata',
            removeConfirm: 'Vuoi davvero rimuovere la Wi-Fi preimpostata?',
            warning:
              'Una volta salvata, il dispositivo proverà a connettersi a questo Wi-Fi a ogni avvio quando lo rileva, il che può interferire con la connessione ad altre reti Wi-Fi. La preimpostazione resta attiva finché non la modifichi o la rimuovi.'
          }
        },
        capture: {
          title: 'Acquisizione video',
          description: "Abilita l'acquisizione video"
        },
        ssh: {
          description: 'Abilita SSH accesso remoto',
          tip: "Dopo aver abilitato SSH, imposta una password complessa per l'utente root nel terminale."
        },
        swap: {
          errors: {
            load: 'Impossibile caricare lo stato della memoria di swap. Riprova tra poco.',
            update:
              'Impossibile aggiornare l’impostazione della memoria di swap. Controlla lo stato attuale prima di riprovare.'
          },
          title: 'Scambia',
          description: 'Abilita la memoria di swap compressa'
        },
        advanced: 'Impostazioni avanzate',
        staticIp: {
          title: 'IP statico',
          config: 'Configurazione',
          tip: 'Per evitare perdite di connessione, si consiglia di eseguire la configurazione mentre si ha accesso fisico al dispositivo.',
          document: 'Controllare il documento.',
          enable: 'Abilita IP statico',
          placeholder: 'Formato: IP/CIDR [Gateway]\nEsempio: 192.168.1.10/22',
          notEmpty: "L'indirizzo IP non può essere vuoto.",
          notValid: "Il formato dell'indirizzo IP non è valido.",
          failed: 'Configurazione non riuscita.',
          success: "Configurazione completata. Controlla se l'indirizzo IP è disponibile.",
          okBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        usbNetwork: {
          title: 'Rete USB',
          mode: 'Modalità',
          server: 'Modalità server',
          serverDesc: "Assegna un indirizzo IP all'host remoto tramite il server DHCP integrato.",
          client: 'Modalità client',
          clientDesc: "Ottiene un indirizzo IP dall'host remoto tramite DHCP.",
          forwarding: 'Inoltro IP',
          forwardingTip:
            "Per consentire all'host remoto di accedere a Internet tramite NanoKVM, abilita l'inoltro IP.",
          manualIp: 'Indirizzo IP manuale',
          ipv4Placeholder: 'IPv4/CIDR, ad es. 10.10.10.1/24',
          invalidIpv4: 'Inserisci un indirizzo IPv4/CIDR valido.',
          loadFailed: 'Impossibile caricare la configurazione della rete USB.',
          failed: 'Configurazione non riuscita.',
          success: 'Configurazione completata.',
          applyBtn: 'Applica',
          cancelBtn: 'Annulla'
        },
        usbId: {
          title: 'Identità dispositivo USB',
          config: 'Configura',
          tip: 'Gli ID USB personalizzati cambiano il modo in cui l’host identifica questo dispositivo. L’applicazione riavvia il gadget USB e può disconnettere brevemente tastiera, mouse o rete.',
          enable: 'Abilita identità USB personalizzata',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Versione dispositivo (bcdDevice)',
          serialnumber: 'Numero di serie',
          manufacturer: 'Produttore',
          product: 'Prodotto',
          configurationName: 'Nome della configurazione USB',
          vidPlaceholder: 'es. 0x046D',
          pidPlaceholder: 'es. 0x0A4E',
          bcdDevicePlaceholder: 'es. 0x0521 (opzionale)',
          serialnumberPlaceholder: 'es. 2214GG01KWF8 (opzionale)',
          manufacturerPlaceholder: 'es. Logitech (opzionale)',
          productPlaceholder: 'es. Logitech P710e (opzionale)',
          notEmpty: 'VID e PID non possono essere vuoti.',
          notValid: 'Formato VID/PID non valido. Usa esadecimale come 0x046D.',
          bcdNotValid: 'Formato bcdDevice non valido. Usa hex come 0x0521 o lascia vuoto.',
          failed: 'Configurazione non riuscita.',
          success: 'Configurazione completata. Gadget USB riavviato.',
          okBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        diskInquiry: {
          title: 'Nome unità USB',
          config: 'Configura',
          tip: 'Nome della chiavetta USB virtuale mostrato sul computer remoto. Il salvataggio riavvia l’USB e può disconnettere brevemente tastiera, mouse o rete.',
          enable: 'Usa un nome unità USB personalizzato',
          placeholder: 'es. LogitechFlash Drive     1.00',
          notEmpty: 'Il nome non può essere vuoto.',
          notValid:
            'Nome non valido. Fino a 28 caratteri ASCII stampabili senza interruzioni di riga.',
          failed: 'Configurazione non riuscita.',
          success: 'Configurazione completata. Gadget USB riavviato.',
          okBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        diskInquiryCdrom: {
          title: 'Nome CD virtuale',
          config: 'Configura',
          tip: 'Nome del lettore CD/DVD virtuale mostrato sul computer remoto. Il salvataggio riavvia l’USB e può disconnettere brevemente tastiera, mouse o rete.',
          enable: 'Usa un nome CD personalizzato',
          placeholder: 'es. LogitechOptical Drive   1.00',
          notEmpty: 'Il nome non può essere vuoto.',
          notValid:
            'Nome non valido. Fino a 28 caratteri ASCII stampabili senza interruzioni di riga.',
          failed: 'Configurazione non riuscita.',
          success: 'Configurazione completata. Gadget USB riavviato.',
          okBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        uac2FunctionName: {
          title: 'Nome audio USB',
          config: 'Configura',
          tip: 'Nome del microfono / dispositivo audio USB virtuale sul computer remoto. Il salvataggio riavvia l’USB e può disconnettere brevemente tastiera, mouse o rete.',
          enable: 'Usa un nome audio USB personalizzato',
          placeholder: 'es. Logitech P710e',
          notEmpty: 'Il nome non può essere vuoto.',
          notValid: 'Nome non valido. Fino a 64 caratteri stampabili senza interruzioni di riga.',
          failed: 'Configurazione non riuscita.',
          success: 'Configurazione completata. Gadget USB riavviato.',
          okBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        lowPower: {
          title: 'Modalità risparmio energetico',
          description: 'Riduce il consumo energetico in modalità inattiva',
          tips: 'Suggerimenti',
          tip1: "Quando è abilitata, l'acquisizione video verrà sospesa nei periodi di inattività per ridurre il consumo energetico e la produzione di calore.",
          tip2: "Il ripristino dell'acquisizione video potrebbe richiedere una nuova inizializzazione, durante la quale il video potrebbe essere temporaneamente non disponibile.",
          okBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        webcamUsbDisplay: 'Disattiva la modalità schermo USB prima di attivare la webcam.',
        webcamEndpoints: 'La webcam richiede due endpoint USB IN liberi.',
        mouseJiggler: {
          noMouse: 'Il simulatore di movimento del mouse non funziona con questa configurazione. Seleziona una modalità mouse o disattiva la modalità a endpoint singolo.',
          title: 'Muovi il mouse',
          description: "Impedisce la sospensione dell'host remoto",
          disable: 'Disabilita',
          absolute: 'Modalità assoluta',
          relative: 'Modalità relativa'
        },
        fingerRobot: {
          title: 'Robot a dito',
          description: 'Abilita il controllo del robot a dito'
        },
        mdns: {
          description: 'Abilita il servizio di rilevamento mDNS'
        },
        hidOnly: 'HID-Solo modalità',
        imageMounted: 'Immagine montata',
        disk: 'Disco virtuale',
        diskDesc: "Monta disco U virtuale sull'host remoto",
        sdCard: {
          warning: 'Inizializza disco virtuale',
          tip1: "Alla prima attivazione, il disco virtuale verrà inizializzato e i file immagine e i dati del disco virtuale presenti sul dispositivo verranno cancellati. L'operazione non può essere annullata. Esegui prima il backup dei file importanti.",
          tip2: "Continuare l'inizializzazione?",
          capacity: 'Capacità',
          disable: 'Disattiva',
          disableWarning: 'Disattivare il disco virtuale?',
          disableTip:
            "La disattivazione cancellerà tutti i file dal disco virtuale e libererà lo spazio di archiviazione del dispositivo utilizzato. Alla successiva attivazione dovrai scegliere nuovamente la capacità e inizializzarlo. L'operazione non può essere annullata."
        },
        network: 'Rete virtuale',
        networkDesc: "Monta scheda di rete virtuale sull'host remoto",
        mic: 'Audio virtuale',
        micDesc: "Scegli la modalità audio virtuale USB da esporre all'host remoto",
        virtualAudioMode: {
          off: 'Disattivato',
          mic: 'Solo microfono',
          duplex: 'Microfono + altoparlante'
        },
        speaker: 'Altoparlante virtuale',
        speakerDesc:
          "Quando è abilitato, sostituisce l'uscita audio DP predefinita per supportare le chiamate remote con maggiore compatibilità. Quando è disabilitato, può essere inoltrato solo l'audio multimediale, non quello delle chiamate.",
        micTip:
          "A causa delle limitazioni dell'endpoint USB, l'abilitazione del microfono virtuale disabiliterà lo schermo secondario USB.\nE si consiglia di disabilitare questa funzionalità durante le operazioni del BIOS per garantire una compatibilità ottimale.",
        reboot: 'Riavvia',
        rebootDesc: 'Sei sicuro di voler riavviare NanoKVM?',
        rebootToFlashing: 'Riavvia in modalità di flashing',
        rebootToFlashingButton: 'Riavvia',
        rebootToFlashingTitle: 'Riavviare in modalità di flashing?',
        rebootToFlashingDesc:
          'NanoKVM si riavvierà ed entrerà in modalità di flashing USB. Assicurati che l’interfaccia USB usata per il flashing sia collegata e che lo strumento di flashing sia pronto.',
        datetime: {
          timezone: 'Fuso orario',
          datetime: 'Data e ora',
          dateAndTime: 'Data e ora',
          synchronize: "Sincronizza l'ora",
          lastSynchronization: 'Ultima sincronizzazione: ',
          notSynchronized: "L'ora del sistema non è ancora sincronizzata",
          syncNow: 'Sincronizza ora',
          manualSet: 'Imposta manualmente',
          manualSetTitle: "Imposta manualmente l'ora",
          manualSetDescription: 'Dopo la modifica dovrai accedere di nuovo.',
          useBrowserTime: "Usa l'ora del browser",
          setTime: "Imposta l'ora",
          invalidTime: "Inserisci una data e un'ora valide.",
          setFailed: "Impossibile impostare l'ora di sistema. Riprova.",
          setSucceeded: "L'ora di sistema è stata aggiornata. Accedi di nuovo."
        },
        dialBackground: {
          errors: {
            load: 'Impossibile caricare le immagini di sfondo del quadrante. Riprova tra poco.',
            upload:
              'Impossibile caricare l’immagine di sfondo. Verifica che sia un file BMP valido e riprova.',
            switch:
              'Impossibile cambiare lo sfondo del quadrante. Aggiorna l’elenco delle immagini e riprova.'
          },
          title: 'Sfondo quadrante',
          description: 'Seleziona un’immagine BMP da /kvmcomm/ui/bg/ come sfondo del quadrante',
          upload: 'Carica',
          loading: 'Caricamento...',
          empty: 'Nessuna immagine BMP trovata',
          current: 'Attuale',
          onlyBmp: 'Sono supportate solo immagini BMP',
          uploaded: 'Caricamento completato',
          switched: 'Sfondo cambiato'
        },
        okBtn: 'Sì',
        cancelBtn: 'Annulla'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Riavviare Tailscale?',
        stop: 'Arrestare Tailscale?',
        stopDesc: 'Esci da Tailscale e disabilita l’avvio automatico al boot.',
        loading: 'Caricamento...',
        notInstall: 'Tailscale non trovato! Per favore, installa.',
        install: 'Installa',
        installing: 'Installazione in corso',
        failed: 'Installazione fallita',
        retry: 'Riprova aggiornando la pagina o installa manualmente',
        download: 'Scarica il',
        package: 'pacchetto di installazione',
        unzip: 'e decomprimilo',
        upTailscale: 'Carica tailscale nella directory /usr/bin/ del NanoKVM',
        upTailscaled: 'Carica tailscaled nella directory /usr/sbin/ del NanoKVM',
        refresh: 'Aggiorna la pagina corrente',
        notRunning: 'Tailscale non è in esecuzione. Per favore avvialo per continuare.',
        run: 'Avvia',
        notLogin:
          'Il dispositivo non è ancora stato associato. Effettua il login e associa questo dispositivo al tuo account.',
        urlPeriod: 'Questo URL è valido per 10 minuti',
        login: 'Accedi',
        loginSuccess: 'Accesso riuscito',
        enable: 'Abilita Tailscale',
        exitNode: 'Nodo di uscita',
        exitNodeDesc: 'Pubblica questo dispositivo come uscita Internet per la tua tailnet.',
        subnetRoute: 'Route di sottorete',
        subnetRouteDesc:
          'Pubblica la sottorete LAN collegata direttamente {{subnet}} nella tua tailnet.',
        subnetUnavailable: 'Non è disponibile alcuna sottorete LAN collegata direttamente.',
        routingUnavailable:
          'Le impostazioni di routing Tailscale non sono disponibili sul dispositivo.',
        pendingApproval:
          'Dopo l’attivazione, approva la route pubblicata nella console di amministrazione Tailscale.',
        routingUpdateFailed: 'Impossibile aggiornare le impostazioni di routing Tailscale.',
        deviceName: 'Nome Dispositivo',
        deviceIP: 'IP Dispositivo',
        account: 'Account',
        logout: 'Disconnetti',
        logoutDesc: 'Sei sicuro di voler uscire?',
        uninstall: 'Disinstalla Tailscale',
        uninstallDesc: 'Sei sicuro di voler disinstallare Tailscale?',
        okBtn: 'Sì',
        cancelBtn: 'Annulla'
      },
      update: {
        title: 'Controlla Aggiornamenti',
        queryFailed: 'Impossibile ottenere la versione',
        updateFailed: 'Aggiornamento fallito. Riprova.',
        isLatest: 'Hai già la versione più recente.',
        available: 'Un aggiornamento è disponibile. Sei sicuro di voler aggiornare?',
        updating: 'Aggiornamento avviato. Attendere prego...',
        confirm: 'Conferma',
        upgrade_tip_nano:
          'È disponibile una nuova versione di NanoKVM. Sei sicuro di voler aggiornare adesso?',
        upgrade_tip_firmware:
          'È disponibile una nuova versione del firmware. Sei sicuro di voler aggiornare adesso?',
        cancel: 'Annulla',
        preview: 'Anteprima aggiornamenti',
        previewDesc: "Ottieni l'accesso anticipato a nuove funzionalità e miglioramenti",
        previewTip:
          'Tieni presente che le versioni di anteprima possono contenere bug o funzionalità incomplete!',
        download: 'Scarica',
        install: 'Installa',
        changelog: 'Registro modifiche',
        restart: 'Riavvia'
      },
      account: {
        title: 'Account',
        webAccount: 'Nome account web',
        password: 'Password',
        twoFactor: 'Autenticazione a due fattori',
        enableTwoFactor: 'Abilita',
        enableTwoFactorDesc:
          "Inserisci la password corrente per iniziare a configurare l'autenticazione a due fattori.",
        disableTwoFactor: 'Disabilita',
        recoveryCodes: 'Codici di recupero',
        enableRecoveryCodes: 'Abilita',
        updateRecoveryCodes: 'Aggiorna',
        recoveryCodesTooltipDisabled:
          "I codici di recupero sono credenziali di backup per l'autenticazione a due fattori. Puoi generarli dopo aver abilitato l'autenticazione a due fattori.",
        recoveryCodesTooltipEnabled:
          "I codici di recupero sono validi solo mentre l'autenticazione a due fattori è abilitata. Disabilitarla invalida tutti i codici di recupero; finché resta abilitata, possono solo essere aggiornati.",
        recoveryCodesRegenerateDesc:
          "Inserisci la password corrente e, a scelta, un codice di autenticazione o un codice di recupero non usato. L'aggiornamento dei codici di recupero invaliderà tutti i codici di recupero esistenti.",
        recoveryCodesVerifyFailed:
          'Impossibile verificare la password e il codice di autenticazione o il codice di recupero.',
        recoveryCodesSavedTitle: 'Salva ora questi codici di recupero',
        recoveryCodesSavedDesc:
          'I codici di recupero possono essere visualizzati una sola volta. Salvali in un luogo sicuro prima di lasciare questa pagina.',
        enableRecoveryCodesDesc:
          'Inserisci la password corrente e un codice di autenticazione per generare i codici di recupero. I codici di recupero possono essere visualizzati una sola volta, quindi salvali prima di lasciare questa pagina.',
        copyRecoveryCodes: 'Copia',
        downloadRecoveryCodes: 'Scarica',
        recoveryCodesCopyFailed: 'Impossibile copiare i codici di recupero.',
        done: 'Fine',
        currentPassword: 'Password corrente',
        authCode: 'Codice di autenticazione',
        authCodeOrRecoveryCode: 'Codice di autenticazione o codice di recupero',
        authRateLimited: 'Troppi tentativi non riusciti. Riprova tra {{seconds}} s.',
        verifyPasswordChange: 'Verifica modifica password',
        verifyPasswordChangeDesc:
          'Inserisci la password corrente e, a scelta, un codice di autenticazione o un codice di recupero non usato prima di cambiare la password.',
        verifyPasswordChangeFailed:
          'Impossibile verificare la password e il codice di autenticazione o il codice di recupero.',
        verifyAndEnable: 'Verifica e abilita',
        scanTwoFactorQRCode:
          'Scansiona questo codice QR con la tua app di autenticazione, poi inserisci il codice.',
        showSetupKey: 'Mostra chiave di configurazione',
        hideSetupKey: 'Nascondi chiave di configurazione',
        setupKey: 'Chiave di configurazione',
        twoFactorSetupFailed:
          "Impossibile avviare la configurazione dell'autenticazione a due fattori.",
        twoFactorVerifyFailed: 'Impossibile verificare il codice di autenticazione.',
        twoFactorDisableFailed: "Impossibile disabilitare l'autenticazione a due fattori.",
        disableTwoFactorDesc:
          "Inserisci la password corrente e, a scelta, un codice di autenticazione o un codice di recupero non usato. Disabilitare l'autenticazione a due fattori invaliderà tutti i codici di recupero.",
        updateBtn: 'Cambia',
        logoutBtn: 'Esci',
        logoutDesc: 'Sei sicuro di voler uscire?',
        okBtn: 'Sì',
        cancelBtn: 'Annulla',
        twoFactorProtection: {
          title: 'Protezione 2FA avanzata',
          description:
            'Scegli le azioni sensibili che richiedono un secondo fattore quando la protezione viene applicata. L\'interruttore "Accesso Web diretto" fa l\'opposto: consente di accedere solo con un codice di autenticazione o un codice di recupero non usato.',
          login: 'Accesso Web diretto',
          webTerminal: 'Terminale Web',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Chiave API MCP',
          targetPower: 'Alimentazione del computer di destinazione',
          nanoKVMRestart: 'Riavvio di NanoKVM',
          deleteImage: 'Elimina immagine',
          scripts: 'Operazioni script',
          screenTimelapse: 'Timelapse dello schermo',
          loginScope:
            'Consente l’accesso Web solo con un codice di autenticazione o un codice di recupero non usato, invece di nome utente e password. Questo riduce la sicurezza di accesso.',
          webTerminalScope: 'Apertura del terminale Web.',
          sshScope: 'Modifica dello stato del servizio SSH.',
          tailscaleScope: 'Installazione, connessione o modifica di Tailscale.',
          frp: 'FRP',
          frpScope: 'Installazione, connessione o modifica di FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Installazione, connessione o modifica di Cloudflare Tunnel.',
          zerotierScope: 'Installazione, connessione o modifica di ZeroTier.',
          netbirdScope: 'Installazione, connessione o modifica di NetBird.',
          mcpScope:
            'Visualizzazione, copia o rigenerazione della chiave API MCP, e abilitazione o disabilitazione di MCP.',
          targetPowerScope:
            'Spegnere o riavviare il computer di destinazione tramite il pulsante di accensione fisico.',
          nanoKVMRestartScope: 'Riavvio di NanoKVM stesso.',
          deleteImageScope: 'Eliminare definitivamente un’immagine ISO o IMG archiviata.',
          scriptsScope: 'Esecuzione, caricamento o eliminazione di script.',
          screenTimelapseScope:
            'Abilitazione della registrazione timelapse dello schermo, consultazione ed eliminazione delle registrazioni.',
          authorizeTitle: 'Autorizza modifiche alla protezione',
          authorizeDesc:
            'Inserisci un codice di autenticazione o un codice di recupero non usato. L’autorizzazione dura cinque minuti.',
          authorizeFailed:
            'Impossibile verificare il codice di autenticazione o il codice di recupero.',
          loadFailed: 'Impossibile caricare le impostazioni di protezione.',
          updateFailed: 'Impossibile salvare le impostazioni di protezione.',
          retry: 'Riprova',
          disableScriptsWarningTitle: 'Disattivare la protezione degli script?',
          disableScriptsWarningDesc:
            'Senza protezione, uno script dannoso potrebbe ottenere accesso con privilegi elevati a questo dispositivo e aggirare o rimuovere la protezione 2FA.',
          cancelDisabling: 'Annulla',
          continueDisabling: 'Continua la disattivazione',
          continueDisablingCountdown: 'Continua la disattivazione ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: "Guida all'autenticazione a due fattori",
          scope:
            "L'autenticazione a due fattori aggiunge un ulteriore livello di sicurezza al tuo account.",
          whatTitle: "Cos'è la 2FA?",
          whatDesc:
            "L'autenticazione a due fattori (2FA) protegge il tuo account con un codice temporaneo generato da un'app di autenticazione. Per impostazione predefinita il codice è richiesto in aggiunta alla password, quindi anche se la password viene compromessa, l'account rimane protetto. Se l'accesso Web diretto è attivo, il solo codice ti consente di accedere.",
          appsTitle: 'App consigliate',
          appsDesc:
            'Puoi usare Google Authenticator o Microsoft Authenticator. Entrambe sono gratuite e funzionano su iOS e Android.',
          setupTitle: 'Come configurarla',
          setupDesc:
            "Apri l'app di autenticazione, scansiona il codice QR mostrato durante la configurazione oppure inserisci manualmente la chiave di configurazione. Poi inserisci il codice a 6 cifre dell'app per completare l'attivazione della 2FA.",
          recoveryTitle: 'Codici di recupero',
          recoveryDesc:
            "I codici di recupero sono codici di backup monouso. Se perdi l'accesso all'app di autenticazione, puoi usare un codice di recupero per accedere. Salvali in un luogo sicuro subito dopo averli generati.",
          logoutNotice:
            "L'attivazione della 2FA disconnetterà tutte le sessioni attive, inclusa questa. Dovrai accedere di nuovo su ogni dispositivo."
        }
      }
    },
    error: {
      title: 'Si è verificato un problema',
      refresh: 'Aggiorna'
    },
    pictureInPicture: {
      title: 'Modalità Picture in Picture',
      enter: 'Attiva Picture in Picture',
      exit: 'Disattiva Picture in Picture',
      error:
        'Impossibile avviare la modalità Picture in Picture. Attendi il caricamento del video e riprova.'
    },
    fullscreen: {
      toggle: 'Attiva/disattiva schermo intero'
    },
    menu: {
      collapse: 'Comprimi menu',
      expand: 'Espandi il menu'
    }
  }
};

export default it;
