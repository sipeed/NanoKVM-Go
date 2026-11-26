const fr = {
  translation: {
    storage: {
      cards: { mount: 'Montage du disque', share: 'Partage de fichiers' },
      title: 'Images et partage de fichiers',
      empty: 'Aucune image',
      notAttached: 'Rien de monté',
      attached: 'Monté : {{name}}',
      eject: 'Éjecter',
      mount: 'Monter',
      mountMode: 'Mode de montage',
      mountModeDisk: 'Stockage de masse',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Ouvrir le gestionnaire de fichiers',
      refresh: 'Actualiser',
      delete: 'Supprimer',
      deleteTitle: 'Supprimer le support',
      deleteConfirm: "Supprimer {{name}} ? Les fichiers qu'il contient seront perdus.",
      downloadImage: "Télécharger l'image",
      typePartition: 'Fichiers partagés',
      volumeLabel: 'Disque amovible',
      groups: {
        share: 'Partage de fichiers',
        images: 'Images'
      },
      failed: "L'opération a échoué",
      hidOnly: 'Mode HID uniquement : le disque virtuel est indisponible',
      disabled: 'Le disque virtuel est désactivé (Paramètres - Appareil)',
      force: {
        title: 'Reprendre le support de force ?',
        desc: "La machine distante utilise encore ce support. Le reprendre réénumère le périphérique USB et peut perdre tout ce qu'elle n'a pas encore écrit sur le disque.",
        menuHint:
          'La machine distante utilise ce support. Ouvrez le gestionnaire de fichiers pour le reprendre de force.'
      },
      tips: {
        title: 'Comment télécharger',
        usb1: 'Connectez le NanoKVM à votre ordinateur via USB.',
        usb2: 'Sur cette page, montez les fichiers partagés en mode stockage de masse.',
        usb3: 'Ouvrez le disque virtuel sur votre ordinateur et copiez le fichier image dans le répertoire racine du disque virtuel.',
        usb4: "Éjectez le disque une fois la copie terminée : l'image apparaît alors dans la liste.",
        scp1: 'Assurez-vous que le NanoKVM et votre ordinateur sont sur le même réseau local.',
        scp2: 'Ouvrez un terminal sur votre ordinateur et utilisez la commande SCP pour copier le fichier image dans le répertoire /data du NanoKVM.',
        scp3: 'Exemple : scp chemin-de-votre-image root@ip-de-votre-nanokvm:/data'
      },
      job: {
        running: 'Traitement de {{name}}'
      },
      enableDisk: {
        title: 'Activer le disque virtuel ?',
        desc: "Monter un support nécessite le disque virtuel, actuellement désactivé. L'activer maintenant ? Le clavier et la souris de la machine distante seront brièvement déconnectés.",
        menuHint: 'Le disque virtuel est désactivé. Activez-le dans Paramètres - Appareil.'
      },
      add: {
        title: 'Ajouter un support',
        upload: 'Envoyer une ISO ou une IMG',
        fromUrl: 'Télécharger depuis une URL',
        url: "URL de l'image",
        urlRequired: 'Veuillez saisir une URL',
        onlyImages: 'Seuls les fichiers .iso et .img peuvent être montés',
        dropHint: 'Glissez une ISO ou IMG ici, ou cliquez pour choisir un fichier',
        cdromOnlyIso: 'Un CD-ROM ne peut contenir qu’une image ISO 9660 / UDF',
        uploadFailed: "Échec de l'envoi",
        downloadFailed: 'Échec du téléchargement',
        uploading: 'Envoi de {{name}}',
        downloading: 'Téléchargement de {{name}}',
        done: '{{name}} est prêt',
        checksumFailed: "L'image ne correspond pas à la somme de contrôle SHA-256.",
        blocked:
          'La machine distante utilise le disque. Éjectez en toute sécurité le lecteur USB virtuel sur cette machine, puis désactivez le commutateur Disque virtuel dans Paramètres > Appareil.'
      },
      files: {
        reasonNotInitialized:
          'Le lecteur USB virtuel n’est pas initialisé. Initialisez-le dans Paramètres > Périphérique avant de partager des fichiers.',
        selectMedium: "Sélectionnez un support pour parcourir les fichiers qu'il contient.",
        reasonAttached: 'Ce support est connecté à la machine distante.',
        reasonAttachedHint:
          'Éjectez en toute sécurité le lecteur USB virtuel sur la machine distante, puis désactivez le commutateur Disque virtuel dans Paramètres > Appareil.',
        reasonNotBrowsable:
          'Ce support contient un système de fichiers que le gestionnaire ne peut pas ouvrir.',
        reasonNoLoop:
          "Cet appareil ne peut pas monter d'images disque (pas de prise en charge de loop).",
        empty: 'Ce dossier est vide',
        reasonISOImage:
          'Une image ISO 9660 ne peut pas être parcourue ici. Elle peut toujours être montée sur la machine distante.',
        noMatch: 'Aucun fichier ne correspond à la recherche.',
        free: '{{size}} libres',
        search: 'Rechercher dans ce dossier',
        sort: {
          modified: 'Modifié',
          name: 'Nom',
          size: 'Taille',
          type: 'Type',
          asc: 'Croissant',
          desc: 'Décroissant'
        },
        view: {
          list: 'Vue en liste',
          grid: 'Vue en icônes'
        },
        folder: 'Dossier',
        itemCount: 'Éléments : {{count}}',
        itemCount_one: '{{count}} élément',
        itemCount_other: '{{count}} éléments',
        dropHint: 'Déposez des fichiers ici pour les importer dans ce dossier',
        up: 'Accéder au dossier parent',
        uploadButton: 'Importer',
        newFolder: 'Nouveau dossier',
        newFolderPlaceholder: 'Nom du dossier',
        create: 'Créer',
        upload: 'Envoyer un fichier sur le support',
        deleteFolderTitle: 'Supprimer le dossier',
        deleteFolderConfirm:
          'Supprimer le dossier {{name}} et tout son contenu ? Cette action est irréversible.',
        uploading: 'Envoi…',
        uploadFailed: "Échec de l'envoi",
        download: 'Télécharger',
        delete: 'Supprimer',
        deleteTitle: 'Supprimer le fichier',
        deleteConfirm: 'Supprimer {{name}} du support ?'
      },
      yes: 'Oui',
      no: 'Non'
    },
    head: {
      storage: 'Images et partage de fichiers',
      desktop: 'Bureau à distance',
      login: 'Connexion',
      changePassword: 'Changer le mot de passe',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Impossible de copier dans le presse-papiers. Autorisez l’accès au presse-papiers, puis réessayez.'
    },
    auth: {
      login: 'Connexion',
      language: 'Langue',
      placeholderUsername: "Veuillez entrer votre nom d'utilisateur",
      placeholderPassword: 'Veuillez entrer votre mot de passe',
      placeholderPassword2: 'Veuillez entrer votre mot de passe à nouveau',
      noEmptyUsername: "Le nom d'utilisateur ne peut pas être vide",
      noEmptyPassword: 'Le mot de passe ne peut pas être vide',
      noAccount:
        "Impossible de récupérer les informations de l'utilisateur, veuillez rafraîchir la page ou réinitialiser le mot de passe",
      invalidUser: "Nom d'utilisateur ou mot de passe invalide",
      error: 'Erreur inattendue',
      twoFactorTitle: 'Vérification requise',
      twoFactorDesc:
        "Saisissez votre code d'authentification ou votre code de récupération non utilisé pour vous connecter.",
      sensitiveActionDesc:
        "Saisissez un code d'authentification ou un code de récupération non utilisé pour continuer.",
      placeholderAuthCode: "Code d'authentification",
      noEmptyAuthCode: "Le code d'authentification est requis",
      verify: 'Vérifier',
      backToLogin: 'Retour à la connexion',
      verificationFailed: 'Échec de la vérification',
      twoFactorEnabledLogout: {
        message:
          "L'authentification à deux facteurs a été activée. Par sécurité, toutes les sessions ont été déconnectées. Veuillez vous reconnecter."
      },
      rateLimited: 'Trop de tentatives échouées. Réessayez dans {{seconds}} s.',
      recoveryCodeUsed: 'Le code de récupération utilisé pour cette connexion a été invalidé.',
      changePassword: 'Changer le mot de passe',
      changePasswordDesc:
        'Pour la sécurité de votre appareil, veuillez modifier le mot de passe de connexion Web.',
      passwordVerificationRequired:
        'Retournez à la page du compte dans les paramètres et vérifiez à nouveau ce changement de mot de passe.',
      passwordVerificationExpired:
        'La vérification du changement de mot de passe a expiré. Retournez à la page du compte dans les paramètres et vérifiez à nouveau.',
      returnToSettings: 'Retour aux paramètres',
      differentPassword: 'Les mots de passe ne correspondent pas',
      illegalUsername: "Le nom d'utilisateur contient des caractères illégaux",
      illegalPassword: 'Le mot de passe contient des caractères illégaux',
      forgetPassword: 'Mot de passe oublié',
      ok: 'Se connecter',
      cancel: 'Annuler',
      savePassword: 'Enregistrer le mot de passe',
      loginButtonText: 'Connexion',
      tips: {
        reset1: 'Compte Web par défaut :',
        reset2: 'Compte SSH par défaut :',
        reset3: 'Pour réinitialiser les mots de passe, veuillez vous référer à ',
        changeReset: 'Pour réinitialiser le mot de passe de connexion Web, consultez ',
        resetLink: 'Réinitialiser'
      }
    },
    wifi: {
      errors: {
        connect:
          'Impossible de se connecter au Wi-Fi. Vérifiez le nom du réseau, le mot de passe et l’état Wi-Fi de l’appareil, puis réessayez.'
      },
      title: 'Wi-Fi',
      description: 'Configurez le Wi-Fi pour le NanoKVM',
      submitted: {
        title: 'Un instant — NanoKVM se connecte à votre Wi-Fi',
        tipsTitle: 'Et ensuite ?',
        tipSwitch: 'Vous pouvez reconnecter votre téléphone à votre Wi-Fi habituel.',
        tipCheck:
          'Dans un instant, regardez l’écran du NanoKVM pour trouver sa nouvelle adresse IP, puis ouvrez-la dans votre navigateur.',
        tipRetry:
          'Si rien n’apparaît au bout d’un moment, vérifiez le nom du réseau et le mot de passe, puis réessayez.'
      },
      invalidMode:
        'Le mode actuel ne prend pas en charge la configuration réseau. Veuillez accéder à votre appareil et activer le mode de configuration Wi-Fi.',
      confirmBtn: 'OK',
      finishBtn: 'Terminé',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Mot de passe',
      steps: {
        settings: 'Paramètres',
        wifi: 'Wi-Fi',
        enable: 'Activer'
      },
      scan: {
        title: 'Choisir un réseau Wi-Fi',
        description: 'Sélectionnez un réseau proche pour continuer.',
        loading: 'Recherche des réseaux Wi-Fi proches…',
        empty: 'Aucun réseau Wi-Fi proche trouvé.',
        failed: 'Impossible de rechercher les réseaux. Saisissez les informations manuellement.',
        retryBtn: 'Rechercher à nouveau',
        manualBtn: 'Saisie manuelle',
        backBtn: 'Retour aux réseaux proches',
        connectTitle: 'Se connecter à {{ssid}}',
        connecting: 'Connexion…',
        viewSimple: 'Passer en vue simple',
        viewDetailed: 'Passer en vue détaillée'
      },
      ap: {
        authTitle: 'Authentification requise',
        authDescription: 'Veuillez saisir le mot de passe AP pour continuer',
        authFailed: 'Mot de passe AP invalide',
        passPlaceholder: 'AP mot de passe',
        verifyBtn: 'Vérifier'
      },
      qr: {
        cameraBtn: 'Prendre une photo',
        pickBtn: 'Choisir une capture',
        scanHint:
          "Prenez une photo d'un code QR Wi-Fi, ou choisissez une capture d'écran, comme celui qu'un téléphone affiche lors du partage de son Wi-Fi.",
        decoding: 'Lecture du code QR…',
        notFound:
          "Aucun code QR n'a été trouvé sur cette photo. Centrez le code et assurez-vous qu'il soit net, puis réessayez.",
        invalid: "Ce code QR ne contient pas d'informations Wi-Fi utilisables.",
        pickerUnavailable:
          "Rien ne s'est ouvert ? Cette fenêtre de connexion peut bloquer la caméra et les photos. Ouvrez plutôt cette adresse dans le navigateur de votre téléphone :",
        confirmTitle: 'Connecter NanoKVM à {{ssid}} ?',
        confirmDescription:
          "Ces informations réseau proviennent d'un code QR. Vérifiez-les avant de vous connecter.",
        openNetwork: "Ce réseau n'a pas de mot de passe.",
        connectBtn: 'Se connecter',
        otherBtn: 'Choisir un autre réseau'
      }
    },
    notification: {
      browser: {
        title: 'Recommandation du navigateur',
        description:
          "Pour une expérience utilisateur optimale, nous vous recommandons d'utiliser le navigateur Chrome. Bien que nous nous efforcions d'assurer la compatibilité entre tous les principaux navigateurs, certaines fonctionnalités avancées peuvent ne pas fonctionner comme prévu en dehors de Chrome."
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Écran',
      video: 'Mode vidéo',
      quality: 'Qualité',
      qualityAuto: 'Automatique',
      qualityLossless: 'Sans perte',
      qualityHigh: 'Élevé',
      qualityMedium: 'Moyen',
      qualityLow: 'Bas',
      scale: 'Échelle',
      noSignal: 'Aucun signal vidéo',
      captureDisabled:
        'La capture vidéo est désactivée. Activez-la dans « Paramètres > Appareil > Capture vidéo ».',
      inconsistentVideoMode: 'Lecture interrompue. Un autre mode vidéo est en cours de lecture.',
      bandwidthWarning:
        'La bande passante réseau est peut-être insuffisante pour les paramètres vidéo actuels. Essayez de réduire le débit ou la fréquence d’images.',
      webrtcConnectionFailed:
        'La connexion WebRTC n’a pas pu être établie. Passez à un mode vidéo Direct.',
      webrtcUnavailable: {
        title: 'WebRTC n’est pas disponible',
        description: 'Activez WebRTC dans votre navigateur ou passez au mode vidéo Direct.'
      },
      upload: 'Téléverser',
      advanced: 'Avancé',
      loading: 'Chargement'
    },
    volume: {
      title: 'Volume'
    },
    microphone: {
      title: 'Micro',
      notMounted: 'Microphone non monté',
      tip1: "Le microphone virtuel n'est pas monté sur l'hôte distant. Souhaitez-vous le monter et l’activer maintenant ?",
      tip2: 'Vous pourrez également gérer cela plus tard dans "Paramètres > Appareil > Audio virtuel".',
      confirm: 'Activer',
      cancel: 'Annuler',
      noDevice: 'Aucun microphone n’a été détecté sur cet appareil. Branchez-en un et réessayez.',
      permissionDenied:
        'L’accès au microphone a été refusé. Autorisez-le dans les paramètres du navigateur.',
      notReadable: 'Le microphone est utilisé par une autre application ou ne peut pas être lu.',
      overconstrained: 'Le microphone ne prend pas en charge les paramètres audio requis.',
      securityBlocked:
        'L’accès au microphone est bloqué sur cette page. Vérifiez les paramètres de sécurité du navigateur.',
      aborted: 'Le démarrage du microphone a été interrompu. Réessayez.',
      unsupported:
        'Ce navigateur ou ce mode de connexion ne prend pas en charge la capture du microphone.',
      enableFailed: 'Impossible d’activer le microphone. Réessayez.',
      disconnected: 'Le microphone a été déconnecté et sa transmission a été désactivée.'
    },
    power: {
      title: 'Alimentation',
      showConfirm: 'Demande de confirmation',
      showConfirmTip: 'Les opérations d’alimentation nécessitent une confirmation supplémentaire',
      powerShort: 'Power (appui court)',
      powerLong: 'Power (appui long)',
      powerConfirm: 'Exécuter l’opération d’alimentation ?',
      okBtn: 'Oui',
      cancelBtn: 'Non'
    },
    keyboard: {
      errors: {
        paste:
          'Impossible d’envoyer le texte à l’appareil distant. Vérifiez la connexion KVM, puis réessayez.'
      },
      title: 'Clavier',
      paste: 'Coller',
      tips: 'Seuls les caractères et symboles standard du clavier sont pris en charge',
      placeholder: 'Veuillez saisir',
      submit: 'Soumettre',
      virtual: 'Clavier',
      swapCommandCtrl: 'Intervertir Cmd et Ctrl',
      shortcut: {
        title: 'Raccourcis',
        custom: 'Personnalisé',
        capture: 'Cliquez ici pour capturer le raccourci',
        clear: 'Effacer',
        save: 'Enregistrer',
        captureTips:
          'La capture de touches système (comme la touche Windows) nécessite l’autorisation du plein écran.',
        enterFullScreen: 'Basculer en mode plein écran.'
      },
      leaderKey: {
        title: 'Touche Leader',
        desc: "Contournez les restrictions du navigateur et envoyez les raccourcis système directement à l'hôte distant.",
        howToUse: 'Comment utiliser',
        simultaneous: {
          title: 'Mode simultané',
          desc1: 'Maintenez la touche Leader enfoncée, puis appuyez sur le raccourci.',
          desc2: 'Intuitif, mais peut entrer en conflit avec les raccourcis système.'
        },
        sequential: {
          title: 'Mode séquentiel',
          desc1:
            'Appuyez sur la touche Leader → appuyez sur le raccourci dans l’ordre → appuyez à nouveau sur la touche Leader.',
          desc2: "Nécessite plus d'étapes, mais évite complètement les conflits système."
        },
        enable: 'Activer la touche Leader',
        tip: 'Lorsqu’elle est définie comme touche Leader, cette touche sert uniquement de déclencheur de raccourci et perd son comportement par défaut.',
        placeholder: 'Appuyez sur la touche Leader',
        shiftRight: 'Shift droit',
        ctrlRight: 'Ctrl droit',
        metaRight: 'Win droit',
        submit: 'Soumettre',
        recorder: {
          rec: 'REC',
          activate: 'Activer les touches',
          input: 'Veuillez appuyer sur le raccourci...'
        }
      }
    },
    mouse: {
      title: 'Souris',
      mode: 'Mode de la souris',
      absolute: 'Souris suiveuse',
      absoluteAndroid: 'Souris suiveuse (meilleure compatibilité Android)',
      touchScreen: 'Écran tactile multipoint',
      relative: 'Souris exclusive',
      direction: 'Sens de la molette',
      scrollUp: 'Faire défiler vers le haut',
      scrollDown: 'Faites défiler vers le bas',
      speed: 'Vitesse de la molette',
      fast: 'Rapide',
      slow: 'Lent',
      requestPointer:
        'Pour utiliser le mode relatif, cliquez sur le bureau pour capturer le pointeur de la souris.',
      repairIphoneDrag: 'Réparer le glisser sur iPhone',
      resetHid: 'Réinitialiser HID',
      touchpadGuide: {
        title: 'Guide du pavé tactile',
        scope: 'S’applique à l’entrée normale du pointeur lorsque le mode souris est Relatif.',
        swipeTitle: 'Balayer pour déplacer',
        swipeDesc: 'Balayez dans la zone active de l’écran pour déplacer le pointeur distant.',
        tapTitle: 'Toucher pour cliquer',
        tapDesc: 'Un appui bref envoie un clic gauche.',
        holdTitle: 'Maintenir pour garder le bouton gauche enfoncé',
        holdDesc:
          'Touchez et gardez le doigt immobile pendant environ 1 seconde pour maintenir le bouton gauche de la souris enfoncé.',
        dragTitle: 'Déplacer après maintien pour faire glisser',
        dragDesc:
          'Une fois le maintien actif, déplacez le doigt pour faire glisser avec le bouton gauche enfoncé.'
      },
      hidOnly: {
        errors: {
          load: 'Impossible de charger l’état du mode HID-Only. Réessayez dans un instant.',
          update:
            'Impossible de confirmer si le mode HID-Only a été mis à jour. Rouvrez ce réglage pour vérifier son état actuel, puis réessayez.'
        },
        title: 'Mode HID uniquement',
        desc: "Si votre souris et votre clavier ne répondent plus et que la réinitialisation de HID ne vous aide pas, il peut s'agir d'un problème de compatibilité entre le NanoKVM et l'appareil. Essayez d'activer le mode HID-Only pour une meilleure compatibilité.",
        tip1: "L'activation du mode HID-Only démontera le disque U virtuel et le réseau virtuel",
        tip2: "En mode HID-Only, le montage d'image est désactivé",
        enable: 'Activer le mode HID uniquement',
        disable: 'Désactiver le mode HID uniquement'
      }
    },
    preview: {
      title: 'Aperçu'
    },
    image: {
      storageReadOnly:
        'Le stockage de fichiers n’est pas accessible en écriture. Veuillez désactiver le disque virtuel.',
      storageBusy:
        'Le stockage est occupé. Terminez ou annulez le transfert avant de changer l’état du disque virtuel.',
      title: 'Images disque',
      loading: 'Chargement',
      empty: 'Vide',
      upload: "Télécharger l'image",
      dropFile: 'Déposez une image ici ou cliquez pour sélectionner un fichier',
      uploadAction: 'Téléverser',
      cancelAction: 'Annuler',
      mount: "Monter l'image",
      mounted: 'Monté',
      uploadError: 'Échec du téléchargement, veuillez réessayer',
      uploadInsufficientStorage: 'Espace de stockage insuffisant',
      uploadTooLarge: 'L’image dépasse la limite de taille',
      cancelUploadError: "Échec de l'annulation du téléversement",
      mountMode: 'Mode de montage',
      massStorage: 'Stockage de masse',
      cdRom: 'CD ROM',
      readOnly: 'Lecture seule',
      mountFailed: "Échec du montage de l'image.",
      mountDesc:
        "Dans certains systèmes, il est nécessaire de déséjecter le disque virtuel sur l'hôte distant avant de monter l'image.",
      unmountFailed: 'Démontage échoué',
      unmountDesc:
        "Dans certains systèmes, vous devez éjecter manuellement sur l'hôte distant avant de démonter l'image.",
      refresh: 'Actualiser la liste des images',
      download: "Télécharger l'image",
      attention: 'À confirmer',
      deleteConfirm: 'Etes-vous sûr de vouloir supprimer cette image?',
      deleteFailed: 'Impossible de supprimer l’image',
      okBtn: 'Oui',
      cancelBtn: 'Non'
    },
    script: {
      title: 'Script',
      upload: 'Téléverser',
      run: 'Exécuter',
      runBackground: 'Exécuter en arrière-plan',
      runFailed: "Échec de l'exécution",
      uploadFailed: 'Impossible de téléverser le script',
      deleteFailed: 'Impossible de supprimer le script',
      attention: 'À confirmer',
      delDesc: 'Êtes-vous sûr de vouloir supprimer ce fichier ?',
      confirm: 'Oui',
      cancel: 'Non',
      delete: 'Supprimer',
      close: 'Fermer'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      confirm: 'OK',
      submit: 'Soumettre',
      username: 'Nom d’utilisateur',
      password: 'Mot de passe',
      authCode: "Code d'authentification",
      authorizeTitle: 'Autoriser le terminal Web',
      codeAccountHint:
        "Saisissez un code d'authentification ou un code de récupération non utilisé.",
      authFailed: "Échec de l'authentification"
    },
    wol: {
      errors: {
        send: 'Impossible d’envoyer la commande de réveil. Vérifiez l’adresse MAC et la connexion réseau, puis réessayez.',
        loadFailed:
          'Impossible de charger la liste des appareils Wake-on-LAN. Vérifiez que NanoKVM est en ligne, puis réessayez.',
        renameFailed:
          'Impossible de renommer cet appareil. Vérifiez l’adresse MAC, puis réessayez.',
        deleteFailed: 'Impossible de supprimer cet appareil. Réessayez dans un instant.'
      },
      title: 'Wake-on-LAN',
      sending: 'Envoi de la commande...',
      sent: 'Commande envoyée',
      input: "Veuillez entrer l'adresse MAC",
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (facultatif)',
      sha256Placeholder: 'Saisissez une somme de contrôle SHA-256 de 64 caractères',
      invalidSHA256: 'SHA-256 doit être une chaîne hexadécimale de 64 caractères',
      title: 'Télécharger l’image',
      input: 'Veuillez entrer l’URL d’une image distante',
      ok: 'OK',
      disabled: 'La partition /data est en lecture seule, impossible de télécharger l’image',
      downloadingFile: 'Téléchargement : {{file}}',
      downloadingProgress: 'Téléchargement ({{percentage}}) : {{file}}',
      failed: 'Échec',
      insufficientStorage: 'Espace de stockage insuffisant',
      remaining: 'Temps restant : {{time}}',
      cancel: 'Annuler',
      cancelFailed: 'Impossible d’annuler le téléchargement',
      pause: 'Suspendre',
      resume: 'Reprendre',
      pauseFailed: 'Impossible de suspendre le téléchargement',
      resumeFailed: 'Impossible de reprendre le téléchargement'
    },
    settings: {
      back: 'Retour',
      close: 'Fermer',
      ai: {
        errors: {
          loadServices:
            'Impossible de charger l’état des services IA. Vérifiez que NanoKVM est en ligne, puis réessayez.',
          loadMcpConfig:
            'Impossible de charger la configuration du service MCP. Réessayez dans un instant.',
          updateMcpService:
            'Impossible de mettre à jour l’état du service MCP. Vérifiez son état actuel avant de réessayer.',
          accessMcpApiKey:
            'Impossible de récupérer la clé API MCP. Réessayez et effectuez la vérification si elle est demandée.',
          regenerateMcpApiKey: 'Impossible de régénérer la clé API MCP. Réessayez dans un instant.',
          regeneratedMcpApiKeyUnavailable:
            'La clé API a été régénérée, mais elle ne peut pas être affichée. Essayez de l’afficher à nouveau. L’ancienne clé n’est plus valide.',
          loadMemoryFabricState:
            'Impossible de charger l’état du service Memory Fabric. Réessayez dans un instant.',
          updateMemoryFabricService:
            'Impossible de mettre à jour l’état du service Memory Fabric. Vérifiez son état actuel avant de réessayer.',
          loadScreenTimelapseState:
            'Impossible de charger l’état du service de timelapse d’écran. Réessayez dans un instant.',
          updateScreenTimelapseService:
            'Impossible de mettre à jour l’état du service de timelapse d’écran. Vérifiez son état actuel avant de réessayer.',
          openScreenTimelapse:
            'Impossible d’ouvrir la lecture du timelapse d’écran. Réessayez dans un instant.',
          loadScreenTimelapseRecordings:
            'Impossible de charger les enregistrements du timelapse d’écran. Réessayez dans un instant.',
          playScreenTimelapse:
            'Cet enregistrement timelapse ne peut pas être lu pour le moment. Sélectionnez de nouveau la date ou réessayez plus tard.',
          screenTimelapseFullscreen:
            'Impossible de modifier le mode plein écran. Vérifiez les autorisations du navigateur, puis réessayez.'
        },
        title: 'AI',
        mcpService: 'Service MCP',
        mcpServiceDesc:
          'Passerelle authentifiée pour le contrôle, les médias et les outils Memory Fabric activés',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Traiter localement la mémoire contextuelle et l’OCR des agents',
        memoryFabricRemoteRequiresMcp:
          'Activez le service MCP ci-dessus pour rendre les outils de mémoire et d’OCR accessibles à distance.',
        memoryFabricRemoteAvailable:
          'Les outils de mémoire et d’OCR sont disponibles via le point de terminaison /api/mcp ci-dessus avec la même clé API.',
        memoryFabricConfig: {
          title: 'Paramètres',
          description: 'Gérez uniquement les paramètres du modèle par défaut de Memory Fabric.',
          loadFailed: 'Impossible de charger la configuration de Memory Fabric.',
          saveFailed: 'Impossible d’enregistrer la configuration de Memory Fabric.',
          saved: 'Configuration enregistrée.',
          savedRestarted: 'Configuration enregistrée et Memory Fabric redémarré.',
          noChanges: 'Aucune modification de la configuration.',
          reload: 'Recharger',
          save: 'Enregistrer',
          unsaved: 'Modifications non enregistrées',
          conflictTitle: 'La configuration a changé',
          conflictDescription:
            'La configuration a été modifiée ailleurs. Rechargez la dernière version avant de l’enregistrer à nouveau.',
          conflictReload: 'Charger la dernière version',
          discardTitle: 'Ignorer les modifications non enregistrées ?',
          discardDescription:
            'Le rechargement remplacera le formulaire actuel par la dernière configuration enregistrée.',
          discardConfirm: 'Ignorer et recharger',
          sections: {
            models: 'Modèles'
          },
          openAICompatibleHint:
            'Prend en charge les fournisseurs de modèles compatibles avec OpenAI. Saisissez le nom du modèle, l’URL de base et la clé API fournis par votre fournisseur.',
          fields: {
            model: 'Modèle',
            baseUrl: 'URL de base',
            maxTokens: 'Nombre maximal de jetons',
            timeoutSeconds: 'Délai d’attente (secondes)'
          },
          apiKey: {
            title: 'Clé API',
            configured: 'Configurée',
            notConfigured: 'Non configurée',
            keep: 'Conserver',
            replace: 'Remplacer',
            clear: 'Supprimer',
            replacement: 'Nouvelle clé API',
            replacementPlaceholder: 'Saisissez la nouvelle clé API',
            clearHint:
              'La clé API enregistrée sera supprimée ; la variable d’environnement pourra toujours fournir une clé.',
            required: 'Veuillez saisir une nouvelle clé API.'
          },
          validation: {
            required: 'Ce champ est obligatoire.',
            invalidUrl: 'Saisissez une URL HTTP ou HTTPS valide.'
          }
        },
        screenTimelapseService: 'Timelapse de l’écran',
        screenTimelapseServiceDesc:
          'Relire l’historique d’écran Memory Fabric et rechercher dans le contenu d’écran capturé',
        screenTimelapseSearchRequiresMemoryFabric:
          'Pour rechercher dans le contenu enregistré, activez également Memory Fabric.',
        screenTimelapseOpen: 'Ouvrir',
        screenTimelapseSearchPlaceholder: 'Rechercher dans le contenu vidéo...',
        screenTimelapseNoVideos: 'Aucune vidéo timelapse trouvée',
        screenTimelapseNoResults: 'Aucune image correspondante trouvée',
        screenTimelapseMultipleResults: 'Résultats de recherche',
        screenTimelapseUnavailable: 'Indisponible',
        screenTimelapsePlayerUnsupported: 'Ce navigateur ne peut pas lire la vidéo timelapse',
        screenTimelapseSearchFailed: 'Échec de la recherche',
        screenTimelapseLoading: 'Chargement du timelapse...',
        screenTimelapseSearchAction: 'Rechercher dans le timelapse',
        screenTimelapsePlay: 'Lire le timelapse',
        screenTimelapsePause: 'Mettre le timelapse en pause',
        screenTimelapseEnterFullscreen: 'Passer en plein écran',
        screenTimelapseExitFullscreen: 'Quitter le plein écran',
        screenTimelapseFullscreenUnsupported: 'Ce navigateur ne peut pas passer en plein écran',
        screenTimelapsePreviousDay: 'Jour précédent',
        screenTimelapseNextDay: 'Jour suivant',
        screenTimelapseDeleteDay: 'Supprimer l’enregistrement du {{date}}',
        screenTimelapseDelete: 'Supprimer',
        screenTimelapseDeleteConfirmTitle: 'Supprimer l’enregistrement timelapse ?',
        screenTimelapseDeleteConfirmDesc:
          'L’enregistrement du {{date}} occupe {{size}}. Cette action est irréversible.',
        screenTimelapseDeleteSuccess: 'Enregistrement du {{date}} supprimé',
        screenTimelapseDeleteFailed: 'Impossible de supprimer l’enregistrement du {{date}}',
        endpoint: 'Point de terminaison',
        apiKey: 'Clé API',
        regenerate: 'Régénérer',
        copyApiKey: 'Copier la clé API',
        copyEndpoint: 'Copier le point de terminaison',
        showApiKey: 'Afficher la clé API',
        regenerateConfirmTitle: 'Régénérer la clé API',
        regenerateConfirmDesc:
          'Les clients MCP existants cesseront de fonctionner tant qu’ils n’auront pas été mis à jour avec la nouvelle clé. Voulez-vous continuer ?',
        okBtn: 'Oui',
        cancelBtn: 'Non'
      },
      title: 'Paramètres',
      about: {
        title: 'A propos de NanoKVM',
        information: 'Informations',
        ip: 'IP',
        mdns: 'mDNS',
        application: "Version de l'application",
        applicationTip: "Version de l'application Web NanoKVM",
        arch: 'Architecture',
        device_number: 'Numéro de périphérique',
        image: "Version de l'image",
        imageTip: "Version de l'image système NanoKVM",
        deviceKey: "Clé de l'appareil",
        community: 'Communauté',
        hostname: "Nom d'hôte",
        hostnameUpdated: "Nom d'hôte mis à jour. Redémarrez pour appliquer.",
        ipType: {
          Wired: 'Filaire',
          Wireless: 'Sans fil',
          Other: 'Autre'
        }
      },
      appearance: {
        title: 'Apparence',
        display: 'Affichage',
        language: 'Langue',
        languageDesc: "Sélectionnez la langue de l'interface",
        languageChangeFailed: 'Échec du changement de langue',
        webTitle: 'Titre Web',
        webTitleDesc: 'Personnaliser le titre de la page Web',
        menuBar: {
          title: 'Barre de menus',
          mode: "Mode d'affichage",
          modeDesc: "Afficher la barre de menu sur l'écran",
          modeOff: 'Désactivé',
          modeAuto: 'Masquer automatiquement',
          modeAlways: 'Toujours visible',
          icons: 'Icônes du sous-menu',
          showIcon: 'Afficher l’icône',
          activateOnConnect: 'Activer à la connexion',
          stopSharing: 'Arrêter le partage',
          iconsDesc: 'Afficher les icônes des sous-menus dans la barre de menus'
        }
      },
      screen: {
        title: 'Écran',
        videoMode: {
          title: 'Mode vidéo',
          description: "Sélectionnez la méthode d'encodage et de transmission"
        },
        edid: {
          description: "Sélectionner les paramètres d'affichage",
          default: 'Par défaut',
          custom: 'Personnalisé',
          document: 'Documentation',
          powerConfirmTitle: "Confirmer avant de mettre à jour l'EDID",
          powerConfirmDescription:
            "Branchez d'abord l'alimentation PD. La mise à jour de l'EDID peut déconnecter brièvement l'écran ; si l'appareil ne détecte pas correctement l'écran ensuite, redémarrez-le manuellement.",
          powerConfirmOk: 'Continuer',
          powerConfirmOkCountdown: 'Continuer ({{seconds}})',
          powerConfirmCancel: 'Annuler',
          errors: {
            badRequest: 'Requête EDID non valide',
            invalidFileName: 'Sélectionnez un fichier EDID .bin avec un nom valide',
            fileTooLarge: 'Le fichier EDID est trop volumineux',
            invalidLength: 'L’EDID doit contenir 128, 256, 384 ou 512 octets',
            invalidHeader: 'L’en-tête de l’EDID n’est pas valide',
            unsupportedVersion: 'Cette version de l’EDID n’est pas prise en charge',
            extensionCountMismatch:
              'Le nombre de blocs d’extension de l’EDID ne correspond pas à la longueur du fichier',
            invalidChecksum: 'La somme de contrôle de l’EDID n’est pas valide',
            invalidCta: 'La structure de l’extension CTA n’est pas valide',
            currentCannotOverwrite:
              'L’EDID actuellement utilisé ne peut pas être remplacé. Sélectionnez d’abord un autre EDID',
            saveFailed: 'Impossible d’enregistrer l’EDID',
            switchFailed: 'Impossible de changer d’EDID',
            uploadFailed: 'Impossible de téléverser l’EDID',
            deleteFailed: 'Impossible de supprimer l’EDID',
            networkFailed: 'Erreur réseau. Réessayez'
          }
        },
        rateControlMode: {
          title: 'Mode de contrôle du débit',
          description: 'Sélectionnez la méthode pour contrôler le débit binaire',
          cbr: "CBR (Constant Bitrate) : donne la priorité à la stabilité de la connexion. Réduit le bégaiement, mais la qualité de l'image peut chuter lors des scènes à mouvement élevé.",
          vbr: "VBR (débit binaire variable) : donne la priorité à la clarté de l'image. Les scènes à forte intensité de mouvement peuvent provoquer des pics de bande passante, pouvant entraîner une latence ou un bégaiement."
        },
        bitrate: {
          title: 'Débit binaire',
          description: "Équilibrez la clarté de l'image avec l'utilisation de la bande passante",
          lower:
            "Des valeurs inférieures réduisent la latence sur les réseaux médiocres mais compromettent la qualité de l'image, provoquant potentiellement un flou ou des artefacts.",
          higher:
            "Des valeurs plus élevées améliorent la clarté de l'image mais consomment plus de bande passante. Peut augmenter la latence si la connexion est lente."
        },
        gop: {
          description: "Définir l'intervalle entre les images clés complètes",
          lower:
            "Les valeurs inférieures envoient des images complètes plus fréquemment. Cela accélère la récupération des artefacts visuels, mais consomme plus de bande passante et réduit les détails de l'image.",
          higher:
            "Des valeurs plus élevées sont optimisées pour les mises à jour incrémentielles. Cela améliore les détails de l'image et économise la bande passante, mais la récupération des artefacts visuels prendra plus de temps."
        },
        quality: {
          title: 'Qualité',
          description: "Définir la qualité de l'image d'affichage"
        },
        fps: {
          auto: 'Automatique',
          description: "Définir la fréquence d'images de l'affichage",
          tip1: "Définir la valeur sur 0 active le mode Auto, synchronisant l'affichage avec la fréquence d'images de l'hôte distant.",
          tip2: "La réduction de la fréquence d'images réduit considérablement l'utilisation de la bande passante. Réduisez-le manuellement en cas d'instabilité du réseau ou lorsque des fréquences d'images élevées ne sont pas requises."
        },
        scale: {
          title: 'Échelle',
          description: "Définir le rapport de mise à l'échelle de l'affichage"
        }
      },
      device: {
        title: 'Appareil',
        virtualAudioConfirm: {
          title: 'Confirmer avant de basculer l’audio virtuel',
          description:
            'Branchez d’abord l’alimentation PD. Le basculement de l’audio virtuel peut déconnecter brièvement l’écran ; si l’appareil ne détecte pas correctement l’écran ensuite, redémarrez-le manuellement.',
          ok: 'Continuer le basculement',
          okCountdown: 'Continuer le basculement ({{seconds}})',
          cancel: 'Annuler'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Configurez le Wi-Fi',
          apMode: 'Le mode AP est activé, connectez-vous au Wi-Fi manuellement',
          others: 'Autres...',
          connect: 'Rejoindre le Wi-Fi',
          connectDesc1: 'Veuillez saisir le SSID et le mot de passe du réseau.',
          connectDesc2: 'Veuillez saisir le mot de passe pour rejoindre ce réseau',
          disconnect: 'Êtes-vous sûr de déconnecter le réseau ?',
          ssid: 'Nom',
          password: 'Mot de passe',
          joinBtn: 'Rejoindre',
          confirmBtn: 'OK',
          cancelBtn: 'Annuler',
          preset: {
            title: 'Préréglage Wi-Fi',
            description:
              'Préenregistrez un réseau Wi-Fi cible ; l’appareil s’y connecte automatiquement au démarrage',
            empty: 'Aucun',
            addBtn: 'Ajouter un préréglage',
            removeBtn: 'Supprimer le préréglage',
            saveTitle: 'Préenregistrer un Wi-Fi',
            saveDesc:
              'Enregistrez le réseau cible maintenant ; l’appareil s’y connecte automatiquement lorsqu’il est détecté au démarrage',
            saveBtn: 'Enregistrer',
            passwordPlaceholder: 'Mot de passe (laisser vide pour un réseau ouvert)',
            saveFailed: 'Échec de l’enregistrement du préréglage Wi-Fi',
            removeFailed: 'Échec de la suppression du préréglage Wi-Fi',
            removeConfirm: 'Voulez-vous vraiment supprimer le Wi-Fi préenregistré ?',
            warning:
              "Une fois enregistré, l'appareil essaiera de se connecter à ce Wi-Fi à chaque démarrage lorsqu'il le détecte, ce qui peut perturber la connexion à d'autres réseaux Wi-Fi. Le préréglage reste en vigueur jusqu'à ce que vous le modifiiez ou le supprimiez."
          }
        },
        capture: {
          title: 'Capture vidéo',
          description: 'Activer la capture vidéo'
        },
        ssh: {
          description: "Activer l'accès à distance SSH",
          tip: "Après avoir activé SSH, définissez un mot de passe fort pour l'utilisateur root dans le terminal."
        },
        swap: {
          errors: {
            load: 'Impossible de charger l’état de la mémoire d’échange. Réessayez dans un instant.',
            update:
              'Impossible de mettre à jour le réglage de la mémoire d’échange. Vérifiez son état actuel avant de réessayer.'
          },
          title: 'Mémoire swap',
          description: "Activer la mémoire d'échange compressée"
        },
        advanced: 'Paramètres avancés',
        staticIp: {
          title: 'IP statique',
          config: 'Configuration',
          tip: "Pour éviter toute perte de connexion, il est recommandé d'effectuer la configuration lorsque vous avez un accès physique à l'appareil.",
          document: 'Vérifiez le document.',
          enable: "Activer l'adresse IP statique",
          placeholder: 'Format : IP/CIDR [Passerelle]\nExemple : 192.168.1.10/22',
          notEmpty: "L'adresse IP ne peut pas être vide.",
          notValid: "Le format de l'adresse IP n'est pas valide.",
          failed: 'Échec de la configuration.',
          success: "Configuration terminée. Veuillez vérifier si l'adresse IP est disponible.",
          okBtn: 'OK',
          cancelBtn: 'Annuler'
        },
        usbNetwork: {
          title: 'Réseau USB',
          mode: 'Mode réseau',
          server: 'Serveur',
          serverDesc: "Attribuer une adresse IP à l'hôte distant avec le serveur DHCP intégré.",
          client: 'Mode client',
          clientDesc: "Obtenir une adresse IP de l'hôte distant via DHCP.",
          forwarding: 'Transfert IP',
          forwardingTip:
            "Pour permettre à l'hôte distant d'accéder à Internet via NanoKVM, activez le transfert IP.",
          manualIp: 'Adresse IP manuelle',
          ipv4Placeholder: 'IPv4/CIDR, par ex. 10.10.10.1/24',
          invalidIpv4: 'Saisissez une adresse IPv4/CIDR valide.',
          loadFailed: 'Impossible de charger la configuration du réseau USB.',
          failed: 'Échec de la configuration.',
          success: 'Configuration terminée.',
          applyBtn: 'Appliquer',
          cancelBtn: 'Annuler'
        },
        usbId: {
          title: "Identité de l'appareil USB",
          config: 'Configuration',
          tip: "Les identifiants USB personnalisés modifient la façon dont l'hôte identifie cet appareil. L'application redémarre le gadget USB et peut brièvement déconnecter le clavier, la souris ou le réseau. Laissez les champs optionnels vides pour conserver les valeurs par défaut du script.",
          enable: "Activer l'identité USB personnalisée",
          vid: 'ID fournisseur (VID)',
          pid: 'ID produit (PID)',
          bcdDevice: "Version de l'appareil (bcdDevice)",
          serialnumber: 'Numéro de série',
          manufacturer: 'Fabricant',
          product: 'Produit',
          configurationName: 'Nom de la configuration USB',
          vidPlaceholder: 'ex. 0x046D',
          pidPlaceholder: 'ex. 0x0A4E',
          bcdDevicePlaceholder: 'ex. 0x0521 (optionnel)',
          serialnumberPlaceholder: 'ex. 2214GG01KWF8 (optionnel)',
          manufacturerPlaceholder: 'ex. Logitech (optionnel)',
          productPlaceholder: 'ex. Logitech P710e (optionnel)',
          notEmpty: 'Le VID et le PID ne peuvent pas être vides.',
          notValid: "Le format VID/PID n'est pas valide. Utilisez un hexadécimal comme 0x046D.",
          bcdNotValid:
            'Format bcdDevice invalide. Utilisez un hexadécimal comme 0x0521, ou laissez vide.',
          failed: 'Échec de la configuration.',
          success: 'Configuration terminée. Gadget USB redémarré.',
          okBtn: 'OK',
          cancelBtn: 'Annuler'
        },
        diskInquiry: {
          title: 'Nom du lecteur USB',
          config: 'Configuration',
          tip: "Nom de la clé USB virtuelle affiché sur l'ordinateur distant. L'enregistrement redémarre l'USB et peut brièvement déconnecter le clavier, la souris ou le réseau.",
          enable: 'Utiliser un nom de lecteur USB personnalisé',
          placeholder: 'ex. LogitechFlash Drive     1.00',
          notEmpty: 'Le nom ne peut pas être vide.',
          notValid: "Nom invalide. Jusqu'à 28 caractères ASCII imprimables sans sauts de ligne.",
          failed: 'Échec de la configuration.',
          success: 'Configuration terminée. Gadget USB redémarré.',
          okBtn: 'OK',
          cancelBtn: 'Annuler'
        },
        diskInquiryCdrom: {
          title: 'Nom du lecteur CD virtuel',
          config: 'Configuration',
          tip: "Nom du lecteur CD/DVD virtuel affiché sur l'ordinateur distant. L'enregistrement redémarre l'USB et peut brièvement déconnecter le clavier, la souris ou le réseau.",
          enable: 'Utiliser un nom de lecteur CD personnalisé',
          placeholder: 'ex. LogitechOptical Drive   1.00',
          notEmpty: 'Le nom ne peut pas être vide.',
          notValid: "Nom invalide. Jusqu'à 28 caractères ASCII imprimables sans sauts de ligne.",
          failed: 'Échec de la configuration.',
          success: 'Configuration terminée. Gadget USB redémarré.',
          okBtn: 'OK',
          cancelBtn: 'Annuler'
        },
        uac2FunctionName: {
          title: 'Nom audio USB',
          config: 'Configuration',
          tip: "Nom du micro / périphérique audio USB virtuel affiché sur l'ordinateur distant. L'enregistrement redémarre l'USB et peut brièvement déconnecter le clavier, la souris ou le réseau.",
          enable: 'Utiliser un nom audio USB personnalisé',
          placeholder: 'ex. Logitech P710e',
          notEmpty: 'Le nom ne peut pas être vide.',
          notValid: "Nom invalide. Jusqu'à 64 caractères imprimables sans sauts de ligne.",
          failed: 'Échec de la configuration.',
          success: 'Configuration terminée. Gadget USB redémarré.',
          okBtn: 'OK',
          cancelBtn: 'Annuler'
        },
        lowPower: {
          title: 'Mode faible consommation',
          description: "Réduisez la consommation d'énergie en cas d'inactivité",
          tips: 'Conseils',
          tip1: "Lorsqu'elle est activée, la capture vidéo est mise en pause pendant les périodes d'inactivité afin de réduire la consommation d'énergie et le dégagement de chaleur.",
          tip2: 'La reprise de la capture vidéo peut nécessiter une nouvelle initialisation, pendant laquelle la vidéo peut être temporairement indisponible.',
          okBtn: 'OK',
          cancelBtn: 'Annuler'
        },
        mouseJiggler: {
          title: 'Souris Jiggler',
          description: "Empêcher l'hôte distant de dormir",
          disable: 'Désactiver',
          absolute: 'Mode absolu',
          relative: 'Mode relatif'
        },
        fingerRobot: {
          title: 'Robot-doigt',
          description: 'Activer le contrôle du robot-doigt'
        },
        mdns: {
          description: 'Activer le service de découverte mDNS'
        },
        hidOnly: 'HID-Mode uniquement',
        imageMounted: 'Image montée',
        disk: 'Disque virtuel',
        diskDesc: "Monter le disque virtuel U sur l'hôte distant",
        sdCard: {
          warning: 'Initialiser le disque virtuel',
          tip1: "Lors de la première activation, le disque virtuel sera initialisé et les fichiers image ainsi que les données du disque virtuel présents sur l'appareil seront effacés. Cette opération est irréversible. Sauvegardez d'abord les fichiers importants.",
          tip2: "Continuer l'initialisation ?",
          capacity: 'Capacité',
          disable: 'Désactiver',
          disableWarning: 'Désactiver le disque virtuel ?',
          disableTip:
            "La désactivation effacera tous les fichiers du disque virtuel et libérera l'espace de stockage utilisé sur l'appareil. Lors de la prochaine activation, vous devrez choisir à nouveau une capacité et initialiser le disque. Cette opération est irréversible."
        },
        network: 'Réseau virtuel',
        networkDesc: "Monter la carte réseau virtuelle sur l'hôte distant",
        mic: 'Audio virtuel',
        micDesc: 'Choisissez le mode audio virtuel USB présenté à l’hôte distant',
        virtualAudioMode: {
          off: 'Désactivé',
          mic: 'Microphone uniquement',
          duplex: 'Microphone + haut-parleur'
        },
        speaker: 'Haut-parleur virtuel',
        speakerDesc:
          'Une fois activé, remplace la sortie audio DP par défaut afin de prendre en charge les appels à distance avec une meilleure compatibilité. Désactivé, seul le son multimédia peut être transmis, pas celui des appels.',
        micTip:
          "En raison des limitations du point de terminaison USB, l'activation du micro virtuel désactivera l'écran secondaire USB.\nEt il est recommandé de désactiver cette fonctionnalité lors des opérations du BIOS pour garantir une compatibilité optimale.",
        reboot: 'Redémarrer',
        rebootDesc: 'Êtes-vous sûr de vouloir redémarrer NanoKVM?',
        rebootToFlashing: 'Redémarrer en mode flash',
        rebootToFlashingButton: 'Redémarrer',
        rebootToFlashingTitle: 'Redémarrer en mode flash ?',
        rebootToFlashingDesc:
          'NanoKVM redémarrera et passera en mode flash USB. Assurez-vous que l’interface USB destinée au flash est connectée et que l’outil de flash est prêt.',
        datetime: {
          timezone: 'Fuseau horaire',
          datetime: 'Dateheure',
          dateAndTime: 'Date et heure',
          synchronize: "Synchroniser l'heure",
          lastSynchronization: 'Dernière synchronisation : ',
          notSynchronized: "L'heure système n'est pas encore synchronisée",
          syncNow: 'Synchroniser maintenant',
          manualSet: 'Régler manuellement',
          manualSetTitle: "Régler l'heure manuellement",
          manualSetDescription: 'Vous devrez vous reconnecter après la modification.',
          useBrowserTime: "Utiliser l'heure du navigateur",
          setTime: "Régler l'heure",
          invalidTime: 'Saisissez une date et une heure valides.',
          setFailed: "Impossible de régler l'heure système. Réessayez.",
          setSucceeded: "L'heure système a été mise à jour. Reconnectez-vous."
        },
        dialBackground: {
          errors: {
            load: 'Impossible de charger les images d’arrière-plan du cadran. Réessayez dans un instant.',
            upload:
              'Impossible d’envoyer l’image d’arrière-plan. Vérifiez qu’il s’agit d’un fichier BMP valide, puis réessayez.',
            switch:
              'Impossible de changer l’arrière-plan du cadran. Actualisez la liste des images, puis réessayez.'
          },
          title: 'Fond du cadran',
          description: 'Sélectionnez une image BMP depuis /kvmcomm/ui/bg/ comme fond du cadran',
          upload: 'Téléverser',
          loading: 'Chargement...',
          empty: 'Aucune image BMP trouvée',
          current: 'Actuel',
          onlyBmp: 'Seules les images BMP sont prises en charge',
          uploaded: 'Téléversement terminé',
          switched: 'Fond changé'
        },
        okBtn: 'Oui',
        cancelBtn: 'Non'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Êtes-vous sûr de vouloir redémarrer Tailscale ?',
        stop: 'Êtes-vous sûr de vouloir arrêter Tailscale ?',
        stopDesc: 'Arrêtez Tailscale et désactivez son démarrage automatique.',
        loading: 'Chargement...',
        notInstall: "Tailscale non trouvé ! Veuillez l'installer.",
        install: 'Installer',
        installing: 'Installation',
        failed: 'Installation échouée',
        retry: "Veuillez rafraîchir et réessayer. Ou essayez d'installer manuellement",
        download: 'Télécharger le',
        package: "paquet d'installation",
        unzip: 'et décompressez-le',
        upTailscale: 'Téléverser tailscale dans le répertoire NanoKVM /usr/bin/',
        upTailscaled: 'Téléverser tailscaled dans le répertoire NanoKVM /usr/sbin/',
        refresh: 'Rafraîchir la page courante',
        notRunning:
          "Tailscale n'est pas en cours d'exécution. Veuillez le démarrer pour continuer.",
        run: 'Démarrer',
        notLogin: "L'appareil n'est pas relié. Connectez-vous et liez cet appareil à votre compte.",
        urlPeriod: "L'URL est valide pendant 10 minutes",
        login: 'Connexion',
        loginSuccess: 'Connexion réussie',
        enable: 'Démarrer Tailscale',
        exitNode: 'Nœud de sortie',
        exitNodeDesc: 'Publier cet appareil comme sortie Internet pour votre tailnet.',
        subnetRoute: 'Route de sous-réseau',
        subnetRouteDesc:
          'Publier le sous-réseau LAN directement connecté {{subnet}} dans votre tailnet.',
        subnetUnavailable: "Aucun sous-réseau LAN directement connecté n'est disponible.",
        routingUnavailable:
          'Les paramètres de routage Tailscale ne sont pas disponibles sur cet appareil.',
        pendingApproval:
          "Après l'activation, approuvez la route publiée dans la console d'administration Tailscale.",
        routingUpdateFailed: 'Impossible de mettre à jour les paramètres de routage Tailscale.',
        deviceName: "Nom de l'appareil",
        deviceIP: "IP de l'appareil",
        account: 'Compte',
        logout: 'Déconnexion',
        logoutDesc: 'Êtes-vous sûr de vouloir vous déconnecter?',
        uninstall: 'Désinstaller Tailscale',
        uninstallDesc: 'Êtes-vous sûr de vouloir désinstaller Tailscale?',
        okBtn: 'Oui',
        cancelBtn: 'Non'
      },
      update: {
        title: 'Vérifier les mises à jour',
        queryFailed: 'Impossible de vérifier les mises à jour. Veuillez réessayer.',
        updateFailed: 'Mise à jour échouée. Veuillez réessayer.',
        isLatest: 'Vous avez déjà la dernière version.',
        available: 'Une mise à jour est disponible. Voulez-vous vraiment mettre à jour?',
        updating: 'Mise à jour en cours. Veuillez patienter...',
        confirm: 'Confirmer',
        upgrade_tip_nano:
          'Une nouvelle version de NanoKVM est disponible. Êtes-vous sûr de vouloir mettre à jour maintenant ?',
        upgrade_tip_firmware:
          'Une nouvelle version du firmware est disponible. Êtes-vous sûr de vouloir mettre à jour maintenant ?',
        cancel: 'Annuler',
        preview: 'Aperçu des mises à jour',
        previewDesc:
          "Bénéficiez d'un accès anticipé aux nouvelles fonctionnalités et améliorations",
        previewTip:
          'Veuillez noter que les versions préliminaires peuvent contenir des bugs ou des fonctionnalités incomplètes!',
        download: 'Télécharger',
        install: 'Installer',
        changelog: 'Journal des modifications',
        restart: 'Redémarrer'
      },
      account: {
        title: 'Compte',
        webAccount: 'Nom du compte Web',
        password: 'Mot de passe',
        twoFactor: 'Authentification à deux facteurs',
        enableTwoFactor: 'Activer',
        enableTwoFactorDesc:
          "Saisissez votre mot de passe actuel pour commencer à configurer l'authentification à deux facteurs.",
        disableTwoFactor: 'Désactiver',
        recoveryCodes: 'Codes de récupération',
        enableRecoveryCodes: 'Activer',
        updateRecoveryCodes: 'Mettre à jour',
        recoveryCodesTooltipDisabled:
          "Les codes de récupération sont des identifiants de secours pour l'authentification à deux facteurs. Vous pouvez les générer après avoir activé l'authentification à deux facteurs.",
        recoveryCodesTooltipEnabled:
          "Les codes de récupération ne sont valides que lorsque l'authentification à deux facteurs est activée. La désactivation de l'authentification à deux facteurs invalide tous les codes de récupération ; tant qu'elle reste activée, ils ne peuvent qu'être mis à jour.",
        recoveryCodesRegenerateDesc:
          "Saisissez votre mot de passe actuel et, soit un code d'authentification, soit un code de récupération non utilisé. La mise à jour des codes de récupération invalidera tous les codes de récupération existants.",
        recoveryCodesVerifyFailed:
          "Impossible de vérifier le mot de passe et le code d'authentification ou le code de récupération.",
        recoveryCodesSavedTitle: 'Enregistrez ces codes de récupération maintenant',
        recoveryCodesSavedDesc:
          "Les codes de récupération ne peuvent être consultés qu'une seule fois. Enregistrez-les dans un emplacement sûr avant de quitter cette page.",
        enableRecoveryCodesDesc:
          "Saisissez votre mot de passe actuel et un code d'authentification pour générer des codes de récupération. Les codes de récupération ne peuvent être consultés qu'une seule fois ; enregistrez-les avant de quitter cette page.",
        copyRecoveryCodes: 'Copier',
        downloadRecoveryCodes: 'Télécharger',
        recoveryCodesCopyFailed: 'Impossible de copier les codes de récupération.',
        done: 'Terminé',
        currentPassword: 'Mot de passe actuel',
        authCode: "Code d'authentification",
        authCodeOrRecoveryCode: "Code d'authentification ou code de récupération",
        authRateLimited: 'Trop de tentatives échouées. Réessayez dans {{seconds}} s.',
        verifyPasswordChange: 'Vérifier le changement de mot de passe',
        verifyPasswordChangeDesc:
          "Saisissez votre mot de passe actuel et, soit un code d'authentification, soit un code de récupération non utilisé, avant de changer le mot de passe.",
        verifyPasswordChangeFailed:
          "Impossible de vérifier le mot de passe et le code d'authentification ou le code de récupération.",
        verifyAndEnable: 'Vérifier et activer',
        scanTwoFactorQRCode:
          "Scannez ce QR code avec votre application d'authentification, puis saisissez le code.",
        showSetupKey: 'Afficher la clé de configuration',
        hideSetupKey: 'Masquer la clé de configuration',
        setupKey: 'Clé de configuration',
        twoFactorSetupFailed:
          "Impossible de démarrer la configuration de l'authentification à deux facteurs.",
        twoFactorVerifyFailed: "Impossible de vérifier le code d'authentification.",
        twoFactorDisableFailed: "Impossible de désactiver l'authentification à deux facteurs.",
        disableTwoFactorDesc:
          "Saisissez votre mot de passe actuel et, soit un code d'authentification, soit un code de récupération non utilisé. La désactivation de l'authentification à deux facteurs invalidera tous les codes de récupération.",
        updateBtn: 'Mettre à jour',
        logoutBtn: 'Déconnexion',
        logoutDesc: 'Êtes-vous sûr de vouloir vous déconnecter?',
        okBtn: 'Oui',
        cancelBtn: 'Non',
        twoFactorProtection: {
          title: 'Protection 2FA avancée',
          description:
            "Choisissez les actions sensibles qui exigent un second facteur lorsque la protection est appliquée. Le commutateur « Connexion Web directe » fait l'inverse : il permet de se connecter uniquement avec un code d'authentification ou un code de récupération non utilisé.",
          login: 'Connexion Web directe',
          webTerminal: 'Terminal Web',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Clé API MCP',
          targetPower: 'Alimentation de la machine cible',
          nanoKVMRestart: 'Redémarrage de NanoKVM',
          deleteImage: 'Suppression d’image',
          scripts: 'Opérations sur les scripts',
          screenTimelapse: 'Timelapse d’écran',
          loginScope:
            "Permet la connexion Web uniquement avec un code d'authentification ou un code de récupération non utilisé, au lieu d'un nom d'utilisateur et d'un mot de passe. Cela réduit la sécurité de connexion.",
          webTerminalScope: 'Ouverture du terminal Web.',
          sshScope: 'Modification de l’état du service SSH.',
          tailscaleScope: 'Installation, connexion ou modification de Tailscale.',
          frp: 'FRP',
          frpScope: 'Installation, connexion ou modification de FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Installation, connexion ou modification de Cloudflare Tunnel.',
          zerotierScope: 'Installation, connexion ou modification de ZeroTier.',
          netbirdScope: 'Installation, connexion ou modification de NetBird.',
          mcpScope:
            'Affichage, copie ou régénération de la clé API MCP, et activation ou désactivation de MCP.',
          targetPowerScope:
            'Arrêter ou redémarrer la machine cible avec son bouton d’alimentation physique.',
          nanoKVMRestartScope: 'Redémarrer NanoKVM lui-même.',
          deleteImageScope: 'Supprimer définitivement une image ISO ou IMG stockée.',
          scriptsScope: 'Exécution, téléversement ou suppression de scripts.',
          screenTimelapseScope:
            'Activer l’enregistrement timelapse de l’écran, consulter les enregistrements et les supprimer.',
          authorizeTitle: 'Autoriser les modifications de protection',
          authorizeDesc:
            "Saisissez un code d'authentification ou un code de récupération non utilisé. Cette autorisation dure cinq minutes.",
          authorizeFailed:
            "Impossible de vérifier le code d'authentification ou le code de récupération.",
          loadFailed: 'Impossible de charger les paramètres de protection.',
          updateFailed: 'Impossible d’enregistrer les paramètres de protection.',
          retry: 'Réessayer',
          disableScriptsWarningTitle: 'Désactiver la protection des scripts ?',
          disableScriptsWarningDesc:
            'Sans protection, un script malveillant pourrait obtenir un accès privilégié à cet appareil et contourner ou supprimer la protection 2FA.',
          cancelDisabling: 'Annuler',
          continueDisabling: 'Continuer la désactivation',
          continueDisablingCountdown: 'Continuer la désactivation ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: "Guide de l'authentification à deux facteurs",
          scope:
            "L'authentification à deux facteurs ajoute une couche de sécurité supplémentaire à votre compte.",
          whatTitle: "Qu'est-ce que la 2FA ?",
          whatDesc:
            "L'authentification à deux facteurs (2FA) protège votre compte avec un code temporaire généré par une application d'authentification. Par défaut, ce code est requis en plus de votre mot de passe, de sorte que même si votre mot de passe est compromis, votre compte reste protégé. Si la connexion Web directe est activée, le code suffit à vous connecter.",
          appsTitle: 'Applications recommandées',
          appsDesc:
            'Vous pouvez utiliser Google Authenticator ou Microsoft Authenticator. Les deux sont gratuites et fonctionnent sur iOS et Android.',
          setupTitle: 'Configuration',
          setupDesc:
            "Ouvrez votre application d'authentification, scannez le QR code affiché pendant la configuration ou saisissez manuellement la clé de configuration. Entrez ensuite le code à 6 chiffres de l'application pour terminer l'activation de la 2FA.",
          recoveryTitle: 'Codes de récupération',
          recoveryDesc:
            "Les codes de récupération sont des codes de secours à usage unique. Si vous perdez l'accès à votre application d'authentification, vous pouvez utiliser un code de récupération pour vous connecter. Enregistrez-les dans un endroit sûr dès leur génération.",
          logoutNotice:
            "L'activation de la 2FA déconnectera toutes les sessions actives, y compris celle-ci. Vous devrez vous reconnecter sur chaque appareil."
        }
      }
    },
    error: {
      title: 'Nous avons rencontré un problème',
      refresh: 'Actualiser'
    },
    pictureInPicture: {
      title: "Image dans l'image",
      enter: "Activer l'image dans l'image",
      exit: "Quitter l'image dans l'image",
      error:
        "Impossible de démarrer l'image dans l'image. Attendez que la vidéo soit chargée et réessayez."
    },
    fullscreen: {
      toggle: 'Basculer vers le plein écran'
    },
    menu: {
      collapse: 'Réduire le menu',
      expand: 'Développer le menu'
    }
  }
};

export default fr;
