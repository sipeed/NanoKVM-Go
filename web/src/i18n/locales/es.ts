const es = {
  translation: {
    storage: {
      cards: { mount: 'Montar disco', share: 'Compartir archivos' },
      title: 'Imágenes y uso compartido de archivos',
      empty: 'Aún no hay imágenes',
      notAttached: 'Nada montado',
      attached: 'Montado: {{name}}',
      eject: 'Expulsar',
      mount: 'Montar',
      mountMode: 'Modo de montaje',
      mountModeDisk: 'Almacenamiento masivo',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Abrir el gestor de archivos',
      refresh: 'Actualizar',
      delete: 'Eliminar',
      deleteTitle: 'Eliminar el medio',
      deleteConfirm: '¿Eliminar {{name}}? Los archivos que contiene se perderán.',
      downloadImage: 'Descargar la imagen',
      typePartition: 'Archivos compartidos',
      volumeLabel: 'Disco extraíble',
      groups: {
        share: 'Uso compartido de archivos',
        images: 'Imágenes'
      },
      failed: 'La operación ha fallado',
      hidOnly: 'Modo solo HID: el disco virtual no está disponible',
      disabled: 'El disco virtual está desactivado (Ajustes - Dispositivo)',
      force: {
        title: '¿Recuperar el medio por la fuerza?',
        desc: 'El equipo remoto todavía está usando este medio. Recuperarlo vuelve a enumerar el dispositivo USB y puede perder todo lo que aún no ha escrito en el disco.',
        menuHint:
          'El equipo remoto está usando este medio. Abre el gestor de archivos para recuperarlo por la fuerza.'
      },
      tips: {
        title: 'Cómo subir imágenes',
        usb1: 'Conecta el NanoKVM a tu computadora mediante USB.',
        usb2: 'En esta página, monta los archivos compartidos en modo de almacenamiento masivo.',
        usb3: 'Abre el disco virtual en tu computadora y copia el archivo de imagen en el directorio raíz del disco virtual.',
        usb4: 'Expulsa el disco cuando termine la copia: la imagen aparecerá entonces en la lista.',
        scp1: 'Asegúrate de que el NanoKVM y tu computadora estén en la misma red local.',
        scp2: 'Abre una terminal en tu computadora y usa el comando SCP para subir el archivo de imagen al directorio /data en el NanoKVM.',
        scp3: 'Ejemplo: scp tu-ruta-de-imagen root@tu-ip-del-nanokvm:/data'
      },
      job: {
        running: 'Procesando {{name}}'
      },
      enableDisk: {
        title: '¿Activar el disco virtual?',
        desc: 'Para montar un medio se necesita el disco virtual, que ahora está desactivado. ¿Activarlo ahora? El teclado y el ratón del equipo remoto se desconectarán un momento.',
        menuHint: 'El disco virtual está desactivado. Actívalo en Ajustes - Dispositivo.'
      },
      add: {
        title: 'Añadir medio',
        upload: 'Subir una ISO o IMG',
        fromUrl: 'Descargar desde una URL',
        url: 'URL de la imagen',
        urlRequired: 'Introduce una URL',
        onlyImages: 'Solo se pueden montar archivos .iso y .img',
        dropHint: 'Arrastra una ISO o IMG aquí, o haz clic para elegir un archivo',
        cdromOnlyIso: 'Un CD-ROM solo puede contener una imagen ISO 9660 / UDF',
        uploadFailed: 'La subida ha fallado',
        downloadFailed: 'La descarga ha fallado',
        uploading: 'Subiendo {{name}}',
        downloading: 'Descargando {{name}}',
        done: '{{name}} está listo',
        checksumFailed: 'La imagen no coincide con la suma de comprobación SHA-256.',
        blocked:
          'El equipo remoto está usando el disco. Expulsa de forma segura la unidad USB virtual en el equipo remoto y después apaga el interruptor Disco virtual en Ajustes > Dispositivo.'
      },
      files: {
        reasonNotInitialized:
          'La unidad USB virtual no está inicializada. Inicialízala en Ajustes > Dispositivo antes de compartir archivos.',
        selectMedium: 'Selecciona un medio para ver los archivos que contiene.',
        reasonAttached: 'Este medio está conectado al equipo remoto.',
        reasonAttachedHint:
          'Expulsa de forma segura la unidad USB virtual en el equipo remoto y después apaga el interruptor Disco virtual en Ajustes > Dispositivo.',
        reasonNotBrowsable:
          'Este medio contiene un sistema de archivos que el gestor no puede abrir.',
        reasonNoLoop: 'Este dispositivo no puede montar imágenes de disco (sin soporte de loop).',
        reasonISOImage:
          'Una imagen ISO 9660 no se puede explorar aquí. Se puede montar igualmente en el equipo remoto.',
        empty: 'Esta carpeta está vacía',
        noMatch: 'Ningún archivo coincide con la búsqueda.',
        free: '{{size}} libres',
        search: 'Buscar en esta carpeta',
        sort: {
          modified: 'Modificado',
          name: 'Nombre',
          size: 'Tamaño',
          type: 'Tipo',
          asc: 'Ascendente',
          desc: 'Descendente'
        },
        view: {
          list: 'Vista de lista',
          grid: 'Vista de iconos'
        },
        folder: 'Carpeta',
        itemCount: 'Elementos: {{count}}',
        itemCount_one: '{{count}} elemento',
        itemCount_other: '{{count}} elementos',
        dropHint: 'Suelta archivos aquí para subirlos a esta carpeta',
        up: 'Ir a la carpeta superior',
        uploadButton: 'Subir',
        newFolder: 'Nueva carpeta',
        newFolderPlaceholder: 'Nombre de la carpeta',
        create: 'Crear',
        upload: 'Subir un archivo al medio',
        deleteFolderTitle: 'Eliminar carpeta',
        deleteFolderConfirm:
          '¿Eliminar la carpeta {{name}} y todo su contenido? Esta acción no se puede deshacer.',
        uploading: 'Subiendo…',
        uploadFailed: 'La subida ha fallado',
        download: 'Descargar',
        delete: 'Eliminar',
        deleteTitle: 'Eliminar el archivo',
        deleteConfirm: '¿Eliminar {{name}} del medio?'
      },
      yes: 'Sí',
      no: 'No'
    },
    head: {
      storage: 'Imágenes y uso compartido de archivos',
      desktop: 'Escritorio remoto',
      login: 'Inicio de sesión',
      changePassword: 'Cambiar contraseña',
      terminal: 'Consola',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'No se pudo copiar al portapapeles. Permita el acceso al portapapeles e inténtelo de nuevo.'
    },
    auth: {
      login: 'Iniciar sesión',
      language: 'Idioma',
      placeholderUsername: 'Introduce tu usuario',
      placeholderPassword: 'Introduce tu contraseña',
      placeholderPassword2: 'Introduce tu contraseña de nuevo',
      noEmptyUsername: 'El usuario no puede estar vacío',
      noEmptyPassword: 'La contraseña no puede estar vacía',
      noAccount:
        'No se ha encontrado la cuenta. Por favor, recarga la página o recupera tu contraseña.',
      invalidUser: 'Usuario o contraseña incorrectos',
      error: 'Error inesperado',
      twoFactorTitle: 'Verificación requerida',
      twoFactorDesc:
        'Introduce tu código de autenticación o código de recuperación sin usar para iniciar sesión.',
      sensitiveActionDesc:
        'Introduce un código de autenticación o un código de recuperación sin usar para continuar.',
      placeholderAuthCode: 'Código de autenticación',
      noEmptyAuthCode: 'El código de autenticación es obligatorio',
      verify: 'Verificar',
      backToLogin: 'Volver al inicio de sesión',
      verificationFailed: 'Verificación fallida',
      twoFactorEnabledLogout: {
        message:
          'La autenticación de dos factores se ha activado. Por seguridad, se han cerrado todas las sesiones. Inicia sesión de nuevo.'
      },
      rateLimited: 'Demasiados intentos fallidos. Inténtalo de nuevo en {{seconds}} s.',
      recoveryCodeUsed:
        'El código de recuperación usado para este inicio de sesión ha sido invalidado.',
      changePassword: 'Cambiar contraseña',
      changePasswordDesc:
        'Para la seguridad de su dispositivo, por favor modifique la contraseña de inicio de sesión en la web.',
      passwordVerificationRequired:
        'Vuelve a la página de cuenta en ajustes y verifica de nuevo este cambio de contraseña.',
      passwordVerificationExpired:
        'La verificación del cambio de contraseña ha caducado. Vuelve a la página de cuenta en ajustes y verifica de nuevo.',
      returnToSettings: 'Volver a ajustes',
      differentPassword: 'Las contraseñas no coinciden',
      illegalUsername: 'El usuario contiene caracteres no permitidos',
      illegalPassword: 'La contraseña contiene caracteres no permitidos',
      forgetPassword: 'Contraseña olvidada',
      ok: 'Aceptar',
      cancel: 'Cancelar',
      savePassword: 'Guardar contraseña',
      loginButtonText: 'Iniciar sesión',
      tips: {
        reset1: 'Cuenta web predeterminada:',
        reset2: 'Cuenta predeterminada SSH:',
        reset3: 'Para restablecer las contraseñas, consulte ',
        changeReset: 'Para restablecer la contraseña de acceso web, consulta ',
        resetLink: 'Restablecer'
      }
    },
    wifi: {
      errors: {
        connect:
          'No se pudo conectar al Wi-Fi. Revise el nombre de la red, la contraseña y el estado del Wi-Fi del dispositivo, y vuelva a intentarlo.'
      },
      title: 'Wi-Fi',
      description: 'Configurar Wi-Fi para NanoKVM',
      submitted: {
        title: 'Un momento: NanoKVM se está conectando a tu Wi-Fi',
        tipsTitle: '¿Qué sigue?',
        tipSwitch: 'Ya puedes volver a conectar tu teléfono a tu Wi-Fi habitual.',
        tipCheck:
          'En unos instantes, mira la pantalla del NanoKVM para ver su nueva dirección IP y ábrela en tu navegador.',
        tipRetry:
          'Si pasado un rato no ocurre nada, comprueba el nombre de la red y la contraseña e inténtalo de nuevo.'
      },
      invalidMode:
        'El modo actual no admite la configuración de red. Vaya a su dispositivo y habilite el modo de configuración Wi-Fi.',
      confirmBtn: 'Aceptar',
      finishBtn: 'Finalizado',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Contraseña',
      steps: {
        settings: 'Ajustes',
        wifi: 'Wi-Fi',
        enable: 'Habilitar'
      },
      scan: {
        title: 'Elegir una red Wi-Fi',
        description: 'Selecciona una red cercana para continuar.',
        loading: 'Buscando redes Wi-Fi cercanas…',
        empty: 'No se encontraron redes Wi-Fi cercanas.',
        failed: 'No se pudo escanear. Introduce los datos manualmente.',
        retryBtn: 'Escanear de nuevo',
        manualBtn: 'Introducir red manualmente',
        backBtn: 'Volver a redes cercanas',
        connectTitle: 'Conectar a {{ssid}}',
        connecting: 'Conectando…',
        viewSimple: 'Cambiar a vista simple',
        viewDetailed: 'Cambiar a vista detallada'
      },
      ap: {
        authTitle: 'Autenticación requerida',
        authDescription: 'Por favor ingrese la contraseña AP para continuar',
        authFailed: 'Contraseña AP no válida',
        passPlaceholder: 'AP contraseña',
        verifyBtn: 'Verificar'
      },
      qr: {
        cameraBtn: 'Tomar foto',
        pickBtn: 'Elegir captura',
        scanHint:
          'Toma una foto de un código QR de Wi-Fi, o elige una captura de pantalla de uno, como el que muestra un teléfono al compartir su Wi-Fi.',
        decoding: 'Leyendo código QR…',
        notFound:
          'No se encontró ningún código QR en esta foto. Centra el código y enfócalo bien, luego inténtalo de nuevo.',
        invalid: 'Este código QR no contiene datos de Wi-Fi utilizables.',
        pickerUnavailable:
          '¿No se abrió nada? Esta ventana de inicio de sesión puede bloquear la cámara y las fotos. Abre esta dirección en el navegador de tu teléfono en su lugar:',
        confirmTitle: '¿Conectar NanoKVM a {{ssid}}?',
        confirmDescription:
          'Estos datos de red provienen de un código QR. Revísalos antes de conectar.',
        openNetwork: 'Esta red no tiene contraseña.',
        connectBtn: 'Conectar',
        otherBtn: 'Elegir otra red'
      }
    },
    notification: {
      browser: {
        title: 'Recomendación del navegador',
        description:
          'Para obtener la mejor experiencia de usuario, recomendamos utilizar el navegador Chrome. Si bien nos esforzamos por lograr la compatibilidad entre los principales navegadores, es posible que algunas funciones avanzadas no funcionen según lo previsto fuera de Chrome.'
      },
      btnOk: 'Aceptar'
    },
    screen: {
      title: 'Pantalla',
      video: 'Modo de vídeo',
      quality: 'Calidad',
      qualityAuto: 'Automático',
      qualityLossless: 'Sin pérdida',
      qualityHigh: 'Alto',
      qualityMedium: 'Medio',
      qualityLow: 'Bajo',
      scale: 'Escala',
      noSignal: 'No hay señal de vídeo',
      captureDisabled:
        'La captura de vídeo está desactivada. Actívela en «Ajustes > Dispositivo > Captura de vídeo».',
      inconsistentVideoMode: 'Reproducción en pausa. Se está reproduciendo otro modo de vídeo.',
      bandwidthWarning:
        'Es posible que el ancho de banda de la red no sea suficiente para la configuración de vídeo actual. Prueba a reducir la tasa de bits o de fotogramas.',
      webrtcConnectionFailed:
        'No se pudo establecer la conexión WebRTC. Cambia a un modo de vídeo Direct.',
      webrtcUnavailable: {
        title: 'WebRTC no está disponible',
        description: 'Activa WebRTC en el navegador o cambia al modo de vídeo Direct.'
      },
      upload: 'Subir',
      advanced: 'Avanzado',
      loading: 'Cargando'
    },
    volume: {
      title: 'Volumen'
    },
    microphone: {
      title: 'Micrófono',
      notMounted: 'Micrófono no montado',
      tip1: 'El micrófono virtual no está montado en el host remoto. ¿Le gustaría montarlo y habilitarlo ahora?',
      tip2: 'También puedes administrar esto más tarde en "Configuración > Dispositivo > Audio virtual".',
      confirm: 'Habilitar',
      cancel: 'Cancelar',
      noDevice:
        'No se ha detectado ningún micrófono en este dispositivo. Conecta uno e inténtalo de nuevo.',
      permissionDenied:
        'Se ha denegado el acceso al micrófono. Permítelo en la configuración del navegador.',
      notReadable: 'Otra aplicación está usando el micrófono o no se puede leer.',
      overconstrained: 'El micrófono no admite los parámetros de audio requeridos.',
      securityBlocked:
        'El acceso al micrófono está bloqueado en esta página. Revisa la configuración de seguridad del navegador.',
      aborted: 'Se canceló el inicio del micrófono. Inténtalo de nuevo.',
      unsupported: 'Este navegador o método de conexión no admite la captura del micrófono.',
      enableFailed: 'No se pudo activar el micrófono. Inténtalo de nuevo.',
      disconnected: 'El micrófono se desconectó y se desactivó su transmisión.'
    },
    power: {
      title: 'Encender / Apagar',
      showConfirm: 'Confirmación',
      showConfirmTip: 'Las operaciones de encendido requieren una confirmación adicional',
      powerShort: 'Encender / Apagar (pulsación corta)',
      powerLong: 'Encender / Apagar (pulsación larga)',
      powerConfirm: '¿Ejecutar operación de encendido?',
      okBtn: 'Sí',
      cancelBtn: 'Cancelar'
    },
    keyboard: {
      errors: {
        paste:
          'No se pudo enviar el texto al dispositivo remoto. Revise la conexión KVM y vuelva a intentarlo.'
      },
      title: 'Teclado',
      paste: 'Pegar',
      tips: 'Solo están soportadas las letras y símbolos estándar del teclado',
      placeholder: 'Por favor, introduce el texto',
      submit: 'Enviar',
      virtual: 'Teclado virtual',
      swapCommandCtrl: 'Intercambiar Cmd y Ctrl',
      shortcut: {
        title: 'Atajos',
        custom: 'Personalizado',
        capture: 'Haga clic aquí para capturar el acceso directo',
        clear: 'Borrar',
        save: 'Guardar',
        captureTips:
          'Capturar teclas del sistema (como la tecla Windows) requiere permiso de pantalla completa.',
        enterFullScreen: 'Alternar el modo de pantalla completa.'
      },
      leaderKey: {
        title: 'Tecla líder',
        desc: 'Omite las restricciones del navegador y envía accesos directos al sistema directamente al host remoto.',
        howToUse: 'Cómo utilizar',
        simultaneous: {
          title: 'Modo Simultáneo',
          desc1: 'Mantenga pulsada la tecla líder y luego pulse el atajo.',
          desc2: 'Intuitivo, pero puede entrar en conflicto con los atajos del sistema.'
        },
        sequential: {
          title: 'Modo Secuencial',
          desc1:
            'Pulse la tecla líder → pulse el atajo en secuencia → vuelva a pulsar la tecla líder.',
          desc2: 'Requiere más pasos, pero evita por completo conflictos del sistema.'
        },
        enable: 'Habilitar tecla líder',
        tip: 'Al asignarse como tecla líder, esta tecla funciona únicamente como activador de atajos y pierde su comportamiento predeterminado.',
        placeholder: 'Pulse la tecla líder',
        shiftRight: 'Shift derecho',
        ctrlRight: 'Ctrl derecho',
        metaRight: 'Win derecho',
        submit: 'Enviar',
        recorder: {
          rec: 'REC',
          activate: 'Activar teclas',
          input: 'Por favor presione el atajo...'
        }
      }
    },
    mouse: {
      title: 'Ratón',
      mode: 'Modo de ratón',
      absolute: 'Ratón de seguimiento',
      absoluteAndroid: 'Ratón de seguimiento (mayor compatibilidad con Android)',
      touchScreen: 'Pantalla multitáctil',
      relative: 'Ratón exclusivo',
      direction: 'Dirección de la rueda de desplazamiento',
      scrollUp: 'Desplazarse hacia arriba',
      scrollDown: 'Desplácese hacia abajo',
      speed: 'Velocidad de la rueda de desplazamiento',
      fast: 'Rápida',
      slow: 'Lenta',
      requestPointer:
        'Usando modo relativo. Por favor, haz clic en el escritorio para obtener el cursor del ratón.',
      repairIphoneDrag: 'Reparar arrastre en iPhone',
      resetHid: 'Restablecer HID',
      touchpadGuide: {
        title: 'Guía del panel táctil',
        scope: 'Se aplica a la entrada normal del puntero cuando el modo de ratón es Relativo.',
        swipeTitle: 'Desliza para mover',
        swipeDesc: 'Desliza dentro del área activa de la pantalla para mover el puntero remoto.',
        tapTitle: 'Toca para hacer clic',
        tapDesc: 'Un toque breve envía un clic izquierdo.',
        holdTitle: 'Mantén pulsado para presionar el botón izquierdo',
        holdDesc:
          'Toca y mantén el dedo quieto durante aproximadamente 1 segundo para mantener pulsado el botón izquierdo del ratón.',
        dragTitle: 'Mueve después de mantener para arrastrar',
        dragDesc:
          'Cuando la pulsación larga esté activa, mueve el dedo para arrastrar con el botón izquierdo pulsado.'
      },
      hidOnly: {
        errors: {
          load: 'No se pudo cargar el estado del modo HID-Only. Vuelva a intentarlo en unos instantes.',
          update:
            'No se pudo confirmar si el modo HID-Only se actualizó. Vuelva a abrir esta configuración para comprobar el estado actual y vuelva a intentarlo.'
        },
        title: 'Modo solo HID',
        desc: 'Si tu ratón y teclado dejan de responder y restablecer el HID no ayuda, podría ser un problema de compatibilidad entre el NanoKVM y el dispositivo. Prueba a habilitar el modo sólo HID para mejorar la compatibilidad.',
        tip1: 'Habilitar el modo sólo HID desmontará el disco virtual y la red virtual',
        tip2: 'En modo sólo HID, el montaje de imágenes está deshabilitado',
        enable: 'Habilitar modo sólo HID',
        disable: 'Desactivar modo sólo HID'
      }
    },
    preview: {
      title: 'Vista previa'
    },
    image: {
      storageReadOnly: 'No se puede escribir en el almacenamiento. Desactive el disco virtual.',
      storageBusy:
        'El almacenamiento está ocupado. Finalice o cancele la transferencia antes de cambiar el estado del disco virtual.',
      title: 'Imágenes',
      loading: 'Cargando...',
      empty: 'No se ha encontrado nada',
      upload: 'Subir imagen',
      dropFile: 'Suelta una imagen aquí o haz clic para seleccionar un archivo',
      uploadAction: 'Subir',
      cancelAction: 'Cancelar',
      mount: 'Montar imagen',
      mounted: 'Montado',
      uploadError: 'La carga falló, inténtelo de nuevo',
      uploadInsufficientStorage: 'Espacio de almacenamiento insuficiente',
      uploadTooLarge: 'La imagen supera el límite de tamaño',
      cancelUploadError: 'No se pudo cancelar la carga',
      mountMode: 'Modo de montaje',
      massStorage: 'Almacenamiento masivo',
      cdRom: 'CD ROM',
      readOnly: 'Sólo lectura',
      mountFailed: 'Fallo al montar',
      mountDesc:
        'En algunos sistemas, es necesario expulsar el disco virtual en el host remoto antes de montar una imagen.',
      unmountFailed: 'Desmontaje fallido',
      unmountDesc:
        'En algunos sistemas, es necesario expulsar manualmente en el host remoto antes de desmontar la imagen.',
      refresh: 'Actualizar la lista de imágenes',
      download: 'Descargar imagen',
      attention: 'Atención',
      deleteConfirm: '¿Estás seguro de que deseas eliminar esta imagen?',
      deleteFailed: 'No se pudo eliminar la imagen',
      okBtn: 'Sí',
      cancelBtn: 'Cancelar'
    },
    script: {
      title: 'Script',
      upload: 'Subir',
      run: 'Ejecutar',
      runBackground: 'Ejecutar en segundo plano',
      runFailed: 'Ejecución fallida',
      uploadFailed: 'No se pudo subir el script',
      deleteFailed: 'No se pudo eliminar el script',
      attention: 'Atención',
      delDesc: '¿Estás seguro que deseas eliminar este archivo?',
      confirm: 'Sí',
      cancel: 'Cancelar',
      delete: 'Eliminar',
      close: 'Cerrar'
    },
    terminal: {
      title: 'Consola',
      confirm: 'Confirmar',
      submit: 'Enviar',
      username: 'Usuario',
      password: 'Contraseña',
      authCode: 'Código de autenticación',
      authorizeTitle: 'Autorizar terminal web',
      codeAccountHint: 'Introduce un código de autenticación o un código de recuperación sin usar.',
      authFailed: 'Error de autenticación'
    },
    wol: {
      errors: {
        send: 'No se pudo enviar el comando de activación. Revise la dirección MAC y la conexión de red, y vuelva a intentarlo.',
        loadFailed:
          'No se pudo cargar la lista de dispositivos Wake-on-LAN. Compruebe que NanoKVM esté en línea y vuelva a intentarlo.',
        renameFailed:
          'No se pudo renombrar este dispositivo. Revise la dirección MAC y vuelva a intentarlo.',
        deleteFailed: 'No se pudo eliminar este dispositivo. Vuelva a intentarlo en unos instantes.'
      },
      title: 'Wake-on-LAN',
      sending: 'Enviando comando...',
      sent: 'Comando enviado',
      input: 'Por favor, introduce la dirección MAC',
      ok: 'Vale'
    },
    download: {
      sha256: 'SHA-256 (opcional)',
      sha256Placeholder: 'Introduzca una suma de comprobación SHA-256 de 64 caracteres',
      invalidSHA256: 'SHA-256 debe ser una cadena hexadecimal de 64 caracteres',
      title: 'Descargador de imágenes',
      input: 'Por favor, introduce la URL de una imagen remota',
      ok: 'Aceptar',
      disabled: 'La partición /data es de sólo lectura, no se puede descargar la imagen',
      downloadingFile: 'Descargando: {{file}}',
      downloadingProgress: 'Descargando ({{percentage}}): {{file}}',
      failed: 'Error',
      insufficientStorage: 'Espacio de almacenamiento insuficiente',
      remaining: 'Tiempo restante: {{time}}',
      cancel: 'Cancelar',
      cancelFailed: 'No se pudo cancelar la descarga',
      pause: 'Pausar',
      resume: 'Reanudar',
      pauseFailed: 'No se pudo pausar la descarga',
      resumeFailed: 'No se pudo reanudar la descarga'
    },
    settings: {
      back: 'Atrás',
      close: 'Cerrar',
      ai: {
        errors: {
          loadServices:
            'No se pudo cargar el estado de los servicios de IA. Confirme que NanoKVM esté en línea y vuelva a intentarlo.',
          loadMcpConfig:
            'No se pudo cargar la configuración del servicio MCP. Vuelva a intentarlo en unos instantes.',
          updateMcpService:
            'No se pudo actualizar el estado del servicio MCP. Compruebe el estado actual antes de volver a intentarlo.',
          accessMcpApiKey:
            'No se pudo obtener la clave API de MCP. Vuelva a intentarlo y complete la verificación si se le solicita.',
          regenerateMcpApiKey:
            'No se pudo regenerar la clave API de MCP. Vuelva a intentarlo en unos instantes.',
          regeneratedMcpApiKeyUnavailable:
            'La clave API se regeneró, pero no se pudo mostrar. Intente mostrarla de nuevo. La clave anterior ya no es válida.',
          loadMemoryFabricState:
            'No se pudo cargar el estado del servicio Memory Fabric. Vuelva a intentarlo en unos instantes.',
          updateMemoryFabricService:
            'No se pudo actualizar el estado del servicio Memory Fabric. Compruebe el estado actual antes de volver a intentarlo.',
          loadScreenTimelapseState:
            'No se pudo cargar el estado del servicio de timelapse de pantalla. Vuelva a intentarlo en unos instantes.',
          updateScreenTimelapseService:
            'No se pudo actualizar el estado del servicio de timelapse de pantalla. Compruebe el estado actual antes de volver a intentarlo.',
          openScreenTimelapse:
            'No se pudo abrir la reproducción del timelapse de pantalla. Vuelva a intentarlo en unos instantes.',
          loadScreenTimelapseRecordings:
            'No se pudieron cargar las grabaciones de timelapse de pantalla. Vuelva a intentarlo en unos instantes.',
          playScreenTimelapse:
            'Esta grabación de timelapse no se puede reproducir ahora. Vuelva a seleccionar la fecha o inténtelo más tarde.',
          screenTimelapseFullscreen:
            'No se pudo cambiar el modo de pantalla completa. Revise los permisos del navegador y vuelva a intentarlo.'
        },
        title: 'AI',
        mcpService: 'Servicio MCP',
        mcpServiceDesc:
          'Puerta de enlace autenticada para control, medios y herramientas habilitadas de Memory Fabric',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Procesar localmente la memoria de contexto y el OCR del agente',
        memoryFabricRemoteRequiresMcp:
          'Active arriba el servicio MCP para que las herramientas de memoria y OCR estén disponibles de forma remota.',
        memoryFabricRemoteAvailable:
          'Las herramientas de memoria y OCR están disponibles en el endpoint /api/mcp de arriba con la misma clave API.',
        memoryFabricConfig: {
          title: 'Configuración',
          description:
            'Gestiona únicamente la configuración del modelo predeterminado de Memory Fabric.',
          loadFailed: 'No se pudo cargar la configuración de Memory Fabric.',
          saveFailed: 'No se pudo guardar la configuración de Memory Fabric.',
          saved: 'Configuración guardada.',
          savedRestarted: 'Configuración guardada y Memory Fabric reiniciado.',
          noChanges: 'No hay cambios en la configuración.',
          reload: 'Volver a cargar',
          save: 'Guardar',
          unsaved: 'Cambios sin guardar',
          conflictTitle: 'La configuración ha cambiado',
          conflictDescription:
            'La configuración se modificó en otro lugar. Carga la versión más reciente antes de volver a guardar.',
          conflictReload: 'Cargar la última versión',
          discardTitle: '¿Descartar los cambios sin guardar?',
          discardDescription:
            'Al volver a cargar, el formulario actual se sustituirá por la última configuración guardada.',
          discardConfirm: 'Descartar y volver a cargar',
          sections: {
            models: 'Modelos'
          },
          openAICompatibleHint:
            'Admite proveedores de modelos compatibles con OpenAI. Introduce el nombre del modelo, la URL base y la clave API proporcionados por el proveedor.',
          fields: {
            model: 'Modelo',
            baseUrl: 'URL base',
            maxTokens: 'Máximo de tokens',
            timeoutSeconds: 'Tiempo de espera (segundos)'
          },
          apiKey: {
            title: 'Clave API',
            configured: 'Configurada',
            notConfigured: 'Sin configurar',
            keep: 'Conservar',
            replace: 'Reemplazar',
            clear: 'Eliminar',
            replacement: 'Nueva clave API',
            replacementPlaceholder: 'Introduce la nueva clave API',
            clearHint:
              'Se eliminará la clave API guardada; la variable de entorno todavía puede proporcionar una clave.',
            required: 'Introduce una nueva clave API.'
          },
          validation: {
            required: 'Este campo es obligatorio.',
            invalidUrl: 'Introduce una URL HTTP o HTTPS válida.'
          }
        },
        screenTimelapseService: 'Timelapse de pantalla',
        screenTimelapseServiceDesc:
          'Reproduce el historial de pantalla de Memory Fabric y busca contenido de pantalla capturado',
        screenTimelapseSearchRequiresMemoryFabric:
          'Para buscar contenido grabado, activa también Memory Fabric.',
        screenTimelapseOpen: 'Abrir',
        screenTimelapseSearchPlaceholder: 'Buscar contenido en el vídeo...',
        screenTimelapseNoVideos: 'No se encontraron vídeos timelapse',
        screenTimelapseNoResults: 'No se encontraron fotogramas coincidentes',
        screenTimelapseMultipleResults: 'Resultados de búsqueda',
        screenTimelapseUnavailable: 'No disponible',
        screenTimelapsePlayerUnsupported: 'Este navegador no puede reproducir vídeo timelapse',
        screenTimelapseSearchFailed: 'La búsqueda falló',
        screenTimelapseLoading: 'Cargando timelapse...',
        screenTimelapseSearchAction: 'Buscar timelapse',
        screenTimelapsePlay: 'Reproducir timelapse',
        screenTimelapsePause: 'Pausar timelapse',
        screenTimelapseEnterFullscreen: 'Entrar en pantalla completa',
        screenTimelapseExitFullscreen: 'Salir de pantalla completa',
        screenTimelapseFullscreenUnsupported: 'Este navegador no puede entrar en pantalla completa',
        screenTimelapsePreviousDay: 'Día anterior',
        screenTimelapseNextDay: 'Día siguiente',
        screenTimelapseDeleteDay: 'Eliminar la grabación del {{date}}',
        screenTimelapseDelete: 'Eliminar',
        screenTimelapseDeleteConfirmTitle: '¿Eliminar la grabación timelapse?',
        screenTimelapseDeleteConfirmDesc:
          'La grabación del {{date}} ocupa {{size}}. Esta acción no se puede deshacer.',
        screenTimelapseDeleteSuccess: 'Se eliminó la grabación del {{date}}',
        screenTimelapseDeleteFailed: 'No se pudo eliminar la grabación del {{date}}',
        endpoint: 'Punto de conexión',
        apiKey: 'Clave de API',
        regenerate: 'Regenerar',
        copyApiKey: 'Copiar clave de API',
        copyEndpoint: 'Copiar punto de conexión',
        showApiKey: 'Mostrar clave de API',
        regenerateConfirmTitle: 'Regenerar clave de API',
        regenerateConfirmDesc:
          'Los clientes MCP existentes dejarán de funcionar hasta que se actualicen con la nueva clave. ¿Quieres continuar?',
        okBtn: 'Sí',
        cancelBtn: 'Cancelar'
      },
      title: 'Ajustes',
      about: {
        title: 'Sobre NanoKVM',
        information: 'Información',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versión de la aplicación',
        applicationTip: 'Versión de la aplicación web de NanoKVM',
        arch: 'Arquitectura',
        device_number: 'Número de dispositivo',
        image: 'Versión de la imagen',
        imageTip: 'Versión de la imagen del sistema NanoKVM',
        deviceKey: 'Clave del dispositivo',
        community: 'Comunidad',
        hostname: 'Nombre del host',
        hostnameUpdated: 'Nombre del host actualizado. Reinicia para aplicar.',
        ipType: {
          Wired: 'Cableada',
          Wireless: 'Inalámbrica',
          Other: 'Otra'
        }
      },
      appearance: {
        title: 'Apariencia',
        display: 'Pantalla',
        language: 'Idioma',
        languageDesc: 'Seleccionar el idioma de la interfaz',
        languageChangeFailed: 'No se pudo cambiar el idioma',
        webTitle: 'Título web',
        webTitleDesc: 'Personaliza el título de la página web',
        menuBar: {
          title: 'Barra de menú',
          mode: 'Modo de visualización',
          modeDesc: 'Mostrar barra de menú en la pantalla',
          modeOff: 'Apagado',
          modeAuto: 'Ocultar automáticamente',
          modeAlways: 'Siempre visible',
          icons: 'Iconos del submenú',
          showIcon: 'Mostrar icono',
          activateOnConnect: 'Activar al conectar',
          stopSharing: 'Dejar de compartir',
          iconsDesc: 'Mostrar iconos de submenú en la barra de menú'
        }
      },
      screen: {
        title: 'Pantalla',
        videoMode: {
          title: 'Modo de vídeo',
          description: 'Seleccione el método de codificación y transmisión'
        },
        edid: {
          description: 'Seleccionar parámetros de visualización',
          default: 'Predeterminado',
          custom: 'Personalizado',
          document: 'Documento',
          powerConfirmTitle: 'Confirmar antes de actualizar EDID',
          powerConfirmDescription:
            'Conecta primero la alimentación PD. Actualizar EDID puede desconectar brevemente la pantalla; si después el dispositivo no detecta la pantalla correctamente, reinícialo manualmente.',
          powerConfirmOk: 'Continuar',
          powerConfirmOkCountdown: 'Continuar ({{seconds}})',
          powerConfirmCancel: 'Cancelar',
          errors: {
            badRequest: 'Solicitud de EDID no válida',
            invalidFileName: 'Selecciona un archivo EDID .bin con un nombre válido',
            fileTooLarge: 'El archivo EDID es demasiado grande',
            invalidLength: 'El EDID debe tener 128, 256, 384 o 512 bytes',
            invalidHeader: 'La cabecera del EDID no es válida',
            unsupportedVersion: 'Esta versión de EDID no es compatible',
            extensionCountMismatch:
              'El número de bloques de extensión del EDID no coincide con la longitud del archivo',
            invalidChecksum: 'La suma de comprobación del EDID no es válida',
            invalidCta: 'La estructura de la extensión CTA no es válida',
            currentCannotOverwrite:
              'No se puede sobrescribir el EDID que está en uso. Cambia primero a otro EDID',
            saveFailed: 'No se pudo guardar el EDID',
            switchFailed: 'No se pudo cambiar el EDID',
            uploadFailed: 'No se pudo subir el EDID',
            deleteFailed: 'No se pudo eliminar el EDID',
            networkFailed: 'Error de red. Inténtalo de nuevo'
          }
        },
        rateControlMode: {
          title: 'Modo de control de velocidad',
          description: 'Seleccione el método para controlar la tasa de bits',
          cbr: 'CBR (tasa de bits constante): prioriza la estabilidad de la conexión. Minimiza el tartamudeo, pero la calidad de la imagen puede disminuir durante escenas de mucho movimiento.',
          vbr: 'VBR (tasa de bits variable): prioriza la claridad de la imagen. Las escenas de mucho movimiento pueden provocar picos de ancho de banda, lo que podría provocar latencia o tartamudeo.'
        },
        bitrate: {
          title: 'Tasa de bits',
          description: 'Equilibre la claridad de la imagen con el uso del ancho de banda',
          lower:
            'Los valores más bajos reducen la latencia en redes deficientes, pero comprometen la calidad de la imagen, lo que puede causar desenfoque o artefactos.',
          higher:
            'Los valores más altos mejoran la claridad de la imagen pero consumen más ancho de banda. Puede aumentar la latencia si la conexión es lenta.'
        },
        gop: {
          description: 'Establece el intervalo entre fotogramas clave completos',
          lower:
            'Los valores más bajos envían fotogramas completos con mayor frecuencia. Esto acelera la recuperación de artefactos visuales pero consume más ancho de banda y reduce los detalles de la imagen.',
          higher:
            'Los valores más altos optimizan las actualizaciones incrementales. Esto mejora los detalles de la imagen y ahorra ancho de banda, pero la recuperación de artefactos visuales llevará más tiempo.'
        },
        quality: {
          title: 'Calidad',
          description: 'Establece la calidad de la imagen de visualización'
        },
        fps: {
          auto: 'Automático',
          description: 'Establecer la velocidad de fotogramas de visualización',
          tip1: 'Establecer el valor en 0 habilita el modo automático, sincronizando la pantalla con la velocidad de fotogramas del host remoto.',
          tip2: 'Reducir la velocidad de fotogramas reduce significativamente el uso de ancho de banda. Disminúyalo manualmente durante la inestabilidad de la red o cuando no se requieran altas velocidades de fotogramas.'
        },
        scale: {
          title: 'Escala',
          description: 'Establecer la relación de escala de visualización'
        }
      },
      device: {
        title: 'Dispositivo',
        virtualAudioConfirm: {
          title: 'Confirmar antes de cambiar el audio virtual',
          description:
            'Conecta primero la alimentación PD. Cambiar el audio virtual puede desconectar brevemente la pantalla; si después el dispositivo no detecta la pantalla correctamente, reinícialo manualmente.',
          ok: 'Continuar con el cambio',
          okCountdown: 'Continuar con el cambio ({{seconds}})',
          cancel: 'Cancelar'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Configurar Wi-Fi',
          apMode: 'El modo AP está habilitado, conéctese a Wi-Fi manualmente',
          others: 'Otros...',
          connect: 'Unirse a Wi-Fi',
          connectDesc1: 'Ingrese el ssid y la contraseña de la red',
          connectDesc2: 'Ingrese la contraseña para unirse a esta red',
          disconnect: '¿Estás seguro de desconectar la red?',
          ssid: 'Nombre',
          password: 'Contraseña',
          joinBtn: 'Unirse',
          confirmBtn: 'Aceptar',
          cancelBtn: 'Cancelar',
          preset: {
            title: 'Wi-Fi preguardada',
            description:
              'Guarda previamente una Wi-Fi de destino; el dispositivo se conectará a ella automáticamente al arrancar',
            empty: 'Ninguna',
            addBtn: 'Añadir preguardado',
            removeBtn: 'Eliminar preguardado',
            saveTitle: 'Preguardar Wi-Fi',
            saveDesc:
              'Guarda ahora la red de destino; el dispositivo se conectará automáticamente cuando la detecte al arrancar',
            saveBtn: 'Guardar',
            passwordPlaceholder: 'Contraseña (déjala vacía para redes abiertas)',
            saveFailed: 'No se pudo guardar la Wi-Fi preguardada',
            removeFailed: 'No se pudo eliminar la Wi-Fi preguardada',
            removeConfirm: '¿Seguro que quieres eliminar la Wi-Fi preguardada?',
            warning:
              'Una vez guardada, el dispositivo intentará conectarse a esta Wi-Fi en cada arranque cuando la detecte, lo que puede interferir con la conexión a otras redes Wi-Fi. El ajuste preguardado sigue vigente hasta que lo modifiques o elimines.'
          }
        },
        capture: {
          title: 'Captura de vídeo',
          description: 'Habilitar la captura de vídeo'
        },
        ssh: {
          description: 'Habilitar acceso remoto SSH',
          tip: 'Después de habilitar SSH, establece una contraseña segura para el usuario root en la terminal.'
        },
        swap: {
          errors: {
            load: 'No se pudo cargar el estado de la memoria de intercambio. Vuelva a intentarlo en unos instantes.',
            update:
              'No se pudo actualizar la configuración de la memoria de intercambio. Compruebe el estado actual antes de volver a intentarlo.'
          },
          title: 'Memoria Swap',
          description: 'Habilitar memoria swap comprimida'
        },
        advanced: 'Ajustes avanzados',
        staticIp: {
          title: 'IP estática',
          config: 'Configuración',
          tip: 'Para evitar la pérdida de conexión, se recomienda configurar mientras tiene acceso físico al dispositivo.',
          document: 'Consulta el documento.',
          enable: 'Habilitar IP estática',
          placeholder: 'Formato: IP/CIDR [Puerta de enlace]\nEjemplo: 192.168.1.10/22',
          notEmpty: 'La dirección IP no puede estar vacía.',
          notValid: 'El formato de la dirección IP no es válido.',
          failed: 'La configuración falló.',
          success:
            'Configuración completa. Por favor verifique si la dirección IP está disponible.',
          okBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        usbNetwork: {
          title: 'Red USB',
          mode: 'Modo',
          server: 'Servidor',
          serverDesc: 'Asigna una dirección IP al host remoto mediante el servidor DHCP integrado.',
          client: 'Cliente',
          clientDesc: 'Obtiene una dirección IP del host remoto mediante DHCP.',
          forwarding: 'Reenvío de IP',
          forwardingTip:
            'Para que el host remoto acceda a Internet a través de NanoKVM, activa el reenvío de IP.',
          manualIp: 'Dirección IP manual',
          ipv4Placeholder: 'IPv4/CIDR, p. ej. 10.10.10.1/24',
          invalidIpv4: 'Introduce una dirección IPv4/CIDR válida.',
          loadFailed: 'No se pudo cargar la configuración de la red USB.',
          failed: 'Error de configuración.',
          success: 'Configuración completada.',
          applyBtn: 'Aplicar',
          cancelBtn: 'Cancelar'
        },
        usbId: {
          title: 'Identidad del dispositivo USB',
          config: 'Configuración',
          tip: 'Los ID USB personalizados cambian cómo el host identifica este dispositivo. Al aplicar se reinicia el gadget USB y puede desconectar brevemente teclado, ratón o red.',
          enable: 'Habilitar identidad USB personalizada',
          vid: 'ID de proveedor (VID)',
          pid: 'ID de producto (PID)',
          bcdDevice: 'Versión del dispositivo (bcdDevice)',
          serialnumber: 'Número de serie',
          manufacturer: 'Fabricante',
          product: 'Producto',
          configurationName: 'Nombre de la configuración USB',
          vidPlaceholder: 'p. ej. 0x046D',
          pidPlaceholder: 'p. ej. 0x0A4E',
          bcdDevicePlaceholder: 'p. ej. 0x0521 (opcional)',
          serialnumberPlaceholder: 'p. ej. 2214GG01KWF8 (opcional)',
          manufacturerPlaceholder: 'p. ej. Logitech (opcional)',
          productPlaceholder: 'p. ej. Logitech P710e (opcional)',
          notEmpty: 'VID y PID no pueden estar vacíos.',
          notValid: 'El formato de VID/PID no es válido. Use hex como 0x046D.',
          bcdNotValid: 'Formato de bcdDevice no válido. Use hex como 0x0521 o déjelo vacío.',
          failed: 'La configuración falló.',
          success: 'Configuración completa. Gadget USB reiniciado.',
          okBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        diskInquiry: {
          title: 'Nombre de la unidad USB',
          config: 'Configuración',
          tip: 'Nombre del pen drive USB virtual que ve el ordenador remoto. Guardar reinicia el USB y puede desconectar brevemente teclado, ratón o red.',
          enable: 'Usar un nombre de unidad USB personalizado',
          placeholder: 'p. ej. LogitechFlash Drive     1.00',
          notEmpty: 'El nombre no puede estar vacío.',
          notValid: 'Nombre no válido. Hasta 28 caracteres ASCII imprimibles sin saltos de línea.',
          failed: 'La configuración falló.',
          success: 'Configuración completa. Gadget USB reiniciado.',
          okBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        diskInquiryCdrom: {
          title: 'Nombre del CD virtual',
          config: 'Configuración',
          tip: 'Nombre de la unidad de CD/DVD virtual que ve el ordenador remoto. Guardar reinicia el USB y puede desconectar brevemente teclado, ratón o red.',
          enable: 'Usar un nombre de CD personalizado',
          placeholder: 'p. ej. LogitechOptical Drive   1.00',
          notEmpty: 'El nombre no puede estar vacío.',
          notValid: 'Nombre no válido. Hasta 28 caracteres ASCII imprimibles sin saltos de línea.',
          failed: 'La configuración falló.',
          success: 'Configuración completa. Gadget USB reiniciado.',
          okBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        uac2FunctionName: {
          title: 'Nombre de audio USB',
          config: 'Configuración',
          tip: 'Nombre del micrófono / dispositivo de audio USB virtual que ve el ordenador remoto. Guardar reinicia el USB y puede desconectar brevemente teclado, ratón o red.',
          enable: 'Usar un nombre de audio USB personalizado',
          placeholder: 'p. ej. Logitech P710e',
          notEmpty: 'El nombre no puede estar vacío.',
          notValid: 'Nombre no válido. Hasta 64 caracteres imprimibles sin saltos de línea.',
          failed: 'La configuración falló.',
          success: 'Configuración completa. Gadget USB reiniciado.',
          okBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        lowPower: {
          title: 'Modo de bajo consumo',
          description: 'Reducir el consumo de energía mientras está inactivo',
          tips: 'Consejos',
          tip1: 'Cuando está habilitada, la captura de vídeo se pausará durante los períodos de inactividad para reducir el consumo de energía y la generación de calor.',
          tip2: 'Reanudar la captura de vídeo puede requerir una reinicialización, durante la cual el vídeo podría no estar disponible temporalmente.',
          okBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        webcamUsbDisplay: 'Desactiva el modo de pantalla USB antes de activar la cámara web.',
        webcamEndpoints: 'La cámara web necesita dos puntos finales USB IN libres.',
        mouseJiggler: {
          noMouse: 'El simulador de movimiento no funciona con esta configuración. Selecciona un modo de ratón o desactiva el modo de punto final único.',
          title: 'Agitador de ratón',
          description: 'Evitar que el host remoto entre en reposo',
          disable: 'Desactivar',
          absolute: 'Modo absoluto',
          relative: 'Modo relativo'
        },
        fingerRobot: {
          title: 'Robot de dedo',
          description: 'Habilitar el control del robot de dedo'
        },
        mdns: {
          description: 'Habilitar servicio de descubrimiento mDNS'
        },
        hidOnly: 'Modo sólo HID',
        imageMounted: 'Imagen montada',
        disk: 'Disco virtual',
        diskDesc: 'Montar disco U virtual en el host remoto',
        sdCard: {
          warning: 'Inicializar disco virtual',
          tip1: 'La primera vez que se active, el disco virtual se inicializará y se borrarán los archivos de imagen y los datos del disco virtual existentes en el dispositivo. Esta acción no se puede deshacer. Haga antes una copia de seguridad de los archivos importantes.',
          tip2: '¿Continuar con la inicialización?',
          capacity: 'Capacidad',
          disable: 'Desactivar',
          disableWarning: '¿Desactivar el disco virtual?',
          disableTip:
            'Al desactivarlo se borrarán todos los archivos del disco virtual y se liberará el almacenamiento del dispositivo que utiliza. La próxima vez que lo active deberá volver a elegir una capacidad e inicializarlo. Esta acción no se puede deshacer.'
        },
        network: 'Red virtual',
        networkDesc: 'Montar tarjeta de red virtual en el host remoto',
        mic: 'Audio virtual',
        micDesc: 'Elige el modo de audio virtual USB que se presenta al host remoto',
        virtualAudioMode: {
          off: 'Desactivado',
          mic: 'Solo micrófono',
          duplex: 'Micrófono + altavoz'
        },
        speaker: 'Altavoz virtual',
        speakerDesc:
          'Al activarlo, reemplaza la salida de audio DP predeterminada para admitir llamadas remotas con mayor compatibilidad. Al desactivarlo, solo se puede reenviar audio multimedia, no audio de llamadas.',
        micTip:
          'Debido a las limitaciones del terminal USB, al habilitar el micrófono virtual se desactivará la pantalla secundaria USB.\nY se recomienda desactivar esta función durante las operaciones del BIOS para garantizar una compatibilidad óptima.',
        reboot: 'Reiniciar',
        rebootDesc: '¿Estás seguro de que deseas reiniciar el NanoKVM?',
        rebootToFlashing: 'Reiniciar en modo de flasheo',
        rebootToFlashingButton: 'Reiniciar',
        rebootToFlashingTitle: '¿Reiniciar en modo de flasheo?',
        rebootToFlashingDesc:
          'NanoKVM se reiniciará y entrará en el modo de flasheo USB. Asegúrate de que la interfaz USB para el flasheo esté conectada y que la herramienta de flasheo esté preparada.',
        datetime: {
          timezone: 'Zona horaria',
          datetime: 'Fecha y hora',
          dateAndTime: 'Fecha y hora',
          synchronize: 'Sincronizar hora',
          lastSynchronization: 'Última sincronización: ',
          notSynchronized: 'La hora del sistema aún no está sincronizada',
          syncNow: 'Sincronizar ahora',
          manualSet: 'Ajustar manualmente',
          manualSetTitle: 'Ajustar la hora manualmente',
          manualSetDescription: 'Deberá iniciar sesión de nuevo después del cambio.',
          useBrowserTime: 'Usar la hora del navegador',
          setTime: 'Ajustar hora',
          invalidTime: 'Introduzca una fecha y hora válidas.',
          setFailed: 'No se pudo ajustar la hora del sistema. Inténtelo de nuevo.',
          setSucceeded: 'La hora del sistema se ha actualizado. Inicie sesión de nuevo.'
        },
        dialBackground: {
          errors: {
            load: 'No se pudieron cargar las imágenes de fondo de la esfera. Vuelva a intentarlo en unos instantes.',
            upload:
              'No se pudo cargar la imagen de fondo. Compruebe que sea un archivo BMP válido y vuelva a intentarlo.',
            switch:
              'No se pudo cambiar el fondo de la esfera. Actualice la lista de imágenes y vuelva a intentarlo.'
          },
          title: 'Fondo del dial',
          description: 'Selecciona una imagen BMP de /kvmcomm/ui/bg/ como fondo del dial',
          upload: 'Subir',
          loading: 'Cargando...',
          empty: 'No se encontraron imágenes BMP',
          current: 'Actual',
          onlyBmp: 'Solo se admiten imágenes BMP',
          uploaded: 'Carga completada',
          switched: 'Fondo cambiado'
        },
        okBtn: 'Sí',
        cancelBtn: 'Cancelar'
      },
      tailscale: {
        title: 'Tailscale',
        restart: '¿Reiniciar Tailscale?',
        stop: '¿Detener Tailscale?',
        stopDesc: 'Cerrar sesión en Tailscale y deshabilitar el inicio automático al arrancar.',
        loading: 'Cargando...',
        notInstall: '¡Tailscale no encontrado! Por favor, instálalo.',
        install: 'Instalar',
        installing: 'Instalando',
        failed: 'La instalación falló',
        retry:
          'Por favor, actualiza la página e inténtalo de nuevo. O intenta instalarlo manualmente',
        download: 'Descargar el',
        package: 'paquete de instalación',
        unzip: 'y descomprimirlo',
        upTailscale: 'Sube tailscale al directorio /usr/bin/ del NanoKVM',
        upTailscaled: 'Sube tailscaled al directorio /usr/sbin/ del NanoKVM',
        refresh: 'Actualizar la página actual',
        notRunning: 'Tailscale no se está ejecutando. Por favor, inícialo para continuar.',
        run: 'Iniciar',
        notLogin:
          'El dispositivo aún no ha sido vinculado. Por favor, inicia sesión y vincula este dispositivo a tu cuenta.',
        urlPeriod: 'Esta URL es válida por 10 minutos',
        login: 'Iniciar sesión',
        loginSuccess: 'Inicio de sesión exitoso',
        enable: 'Habilitar Tailscale',
        exitNode: 'Nodo de salida',
        exitNodeDesc: 'Anuncia este dispositivo como salida a Internet para tu tailnet.',
        subnetRoute: 'Ruta de subred',
        subnetRouteDesc: 'Anuncia la subred LAN conectada directamente {{subnet}} en tu tailnet.',
        subnetUnavailable: 'No hay ninguna subred LAN conectada directamente disponible.',
        routingUnavailable: 'La configuración de enrutamiento de Tailscale no está disponible.',
        pendingApproval:
          'Después de habilitarla, aprueba la ruta anunciada en la consola de administración de Tailscale.',
        routingUpdateFailed: 'No se pudo actualizar la configuración de enrutamiento de Tailscale.',
        deviceName: 'Nombre del dispositivo',
        deviceIP: 'IP del dispositivo',
        account: 'Cuenta',
        logout: 'Cerrar sesión',
        logoutDesc: '¿Estás seguro de que deseas cerrar sesión?',
        uninstall: 'Desinstalar Tailscale',
        uninstallDesc: '¿Estás seguro de que deseas desinstalar Tailscale?',
        okBtn: 'Sí',
        cancelBtn: 'Cancelar'
      },
      update: {
        title: 'Buscar actualizaciones',
        queryFailed: 'Error al obtener la versión',
        updateFailed: 'La actualización falló. Por favor, inténtalo de nuevo.',
        isLatest: 'Ya tienes la última versión.',
        available: 'Hay una actualización disponible. ¿Estás seguro de que quieres actualizar?',
        updating: 'Actualización iniciada. Por favor, espera...',
        confirm: 'Confirmar',
        upgrade_tip_nano:
          'Hay disponible una nueva versión de NanoKVM. ¿Estás seguro de que quieres actualizar ahora?',
        upgrade_tip_firmware:
          'Hay una nueva versión de firmware disponible. ¿Estás seguro de que quieres actualizar ahora?',
        cancel: 'Cancelar',
        preview: 'Vista previa de actualizaciones',
        previewDesc: 'Accede anticipadamente a nuevas funciones y mejoras',
        previewTip:
          'Ten en cuenta que las versiones de vista previa pueden contener errores o funcionalidades incompletas',
        download: 'Descargar',
        install: 'Instalar',
        changelog: 'Registro de cambios',
        restart: 'Reiniciar'
      },
      account: {
        title: 'Cuenta',
        webAccount: 'Nombre de la cuenta web',
        password: 'Contraseña',
        twoFactor: 'Autenticación de dos factores',
        enableTwoFactor: 'Activar',
        enableTwoFactorDesc:
          'Introduce tu contraseña actual para empezar a configurar la autenticación de dos factores.',
        disableTwoFactor: 'Desactivar',
        recoveryCodes: 'Códigos de recuperación',
        enableRecoveryCodes: 'Activar',
        updateRecoveryCodes: 'Actualizar',
        recoveryCodesTooltipDisabled:
          'Los códigos de recuperación son credenciales de respaldo para la autenticación de dos factores. Puedes generarlos después de activar la autenticación de dos factores.',
        recoveryCodesTooltipEnabled:
          'Los códigos de recuperación solo son válidos mientras la autenticación de dos factores esté activada. Desactivarla invalida todos los códigos de recuperación; mientras siga activada, solo pueden actualizarse.',
        recoveryCodesRegenerateDesc:
          'Introduce tu contraseña actual y, ya sea un código de autenticación o un código de recuperación sin usar. Actualizar los códigos de recuperación invalidará todos los códigos existentes.',
        recoveryCodesVerifyFailed:
          'No se pudo verificar la contraseña y el código de autenticación o el código de recuperación.',
        recoveryCodesSavedTitle: 'Guarda estos códigos de recuperación ahora',
        recoveryCodesSavedDesc:
          'Los códigos de recuperación solo pueden verse una vez. Guárdalos en un lugar seguro antes de salir de esta página.',
        enableRecoveryCodesDesc:
          'Introduce tu contraseña actual y un código de autenticación para generar códigos de recuperación. Los códigos de recuperación solo pueden verse una vez, así que guárdalos antes de salir de esta página.',
        copyRecoveryCodes: 'Copiar',
        downloadRecoveryCodes: 'Descargar',
        recoveryCodesCopyFailed: 'No se pudieron copiar los códigos de recuperación.',
        done: 'Listo',
        currentPassword: 'Contraseña actual',
        authCode: 'Código de autenticación',
        authCodeOrRecoveryCode: 'Código de autenticación o código de recuperación',
        authRateLimited: 'Demasiados intentos fallidos. Inténtalo de nuevo en {{seconds}} s.',
        verifyPasswordChange: 'Verificar cambio de contraseña',
        verifyPasswordChangeDesc:
          'Introduce tu contraseña actual y, ya sea un código de autenticación o un código de recuperación sin usar, antes de cambiar la contraseña.',
        verifyPasswordChangeFailed:
          'No se pudo verificar la contraseña y el código de autenticación o el código de recuperación.',
        verifyAndEnable: 'Verificar y activar',
        scanTwoFactorQRCode:
          'Escanea este código QR con tu aplicación de autenticación y luego introduce el código.',
        showSetupKey: 'Mostrar clave de configuración',
        hideSetupKey: 'Ocultar clave de configuración',
        setupKey: 'Clave de configuración',
        twoFactorSetupFailed:
          'No se pudo iniciar la configuración de la autenticación de dos factores.',
        twoFactorVerifyFailed: 'No se pudo verificar el código de autenticación.',
        twoFactorDisableFailed: 'No se pudo desactivar la autenticación de dos factores.',
        disableTwoFactorDesc:
          'Introduce tu contraseña actual y, ya sea un código de autenticación o un código de recuperación sin usar. Desactivar la autenticación de dos factores invalidará todos los códigos de recuperación.',
        updateBtn: 'Cambiar',
        logoutBtn: 'Cerrar sesión',
        logoutDesc: '¿Estás seguro de que deseas cerrar sesión?',
        okBtn: 'Sí',
        cancelBtn: 'Cancelar',
        twoFactorProtection: {
          title: 'Protección avanzada con 2FA',
          description:
            'Selecciona qué acciones sensibles requieren un segundo factor cuando se aplique la protección. El interruptor «Inicio de sesión web directo» hace lo contrario: permite iniciar sesión solo con un código de autenticación o un código de recuperación sin usar.',
          login: 'Inicio de sesión web directo',
          webTerminal: 'Terminal web',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Clave de API MCP',
          targetPower: 'Alimentación del equipo objetivo',
          nanoKVMRestart: 'Reinicio de NanoKVM',
          deleteImage: 'Eliminar imagen',
          scripts: 'Operaciones de scripts',
          screenTimelapse: 'Timelapse de pantalla',
          loginScope:
            'Permite iniciar sesión en la web solo con un código de autenticación o un código de recuperación sin usar, en lugar de un nombre de usuario y contraseña. Esto reduce la seguridad del inicio de sesión.',
          webTerminalScope: 'Abrir la terminal web.',
          sshScope: 'Cambiar el estado del servicio SSH.',
          tailscaleScope: 'Instalar, conectar o cambiar Tailscale.',
          frp: 'FRP',
          frpScope: 'Instalar, conectar o cambiar FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Instalar, conectar o cambiar Cloudflare Tunnel.',
          zerotierScope: 'Instalar, conectar o cambiar ZeroTier.',
          netbirdScope: 'Instalar, conectar o cambiar NetBird.',
          mcpScope: 'Mostrar, copiar o regenerar la clave de API MCP, y activar o desactivar MCP.',
          targetPowerScope:
            'Apagar o reiniciar el equipo objetivo mediante su botón de encendido físico.',
          nanoKVMRestartScope: 'Reiniciar NanoKVM.',
          deleteImageScope: 'Eliminar permanentemente una imagen ISO o IMG almacenada.',
          scriptsScope: 'Ejecutar, subir o eliminar scripts.',
          screenTimelapseScope:
            'Activar la grabación de timelapse de pantalla, explorar las grabaciones y eliminarlas.',
          authorizeTitle: 'Autorizar cambios de protección',
          authorizeDesc:
            'Introduce un código de autenticación o de recuperación sin usar. La autorización dura cinco minutos.',
          authorizeFailed:
            'No se pudo verificar el código de autenticación o el código de recuperación.',
          loadFailed: 'No se pudieron cargar los ajustes de protección.',
          updateFailed: 'No se pudieron guardar los ajustes de protección.',
          retry: 'Reintentar',
          disableScriptsWarningTitle: '¿Desactivar la protección de scripts?',
          disableScriptsWarningDesc:
            'Sin protección, un script malicioso podría obtener acceso elevado a este dispositivo y eludir o eliminar la protección 2FA.',
          cancelDisabling: 'Cancelar',
          continueDisabling: 'Continuar desactivando',
          continueDisablingCountdown: 'Continuar desactivando ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Guía de autenticación de dos factores',
          scope:
            'La autenticación de dos factores añade una capa adicional de seguridad a tu cuenta.',
          whatTitle: '¿Qué es 2FA?',
          whatDesc:
            'La autenticación de dos factores (2FA) protege tu cuenta con un código temporal de una aplicación de autenticación. De forma predeterminada, el código es obligatorio además de tu contraseña, así que aunque tu contraseña se vea comprometida, tu cuenta permanece protegida. Si el inicio de sesión web directo está activado, el código por sí solo te da acceso.',
          appsTitle: 'Aplicaciones recomendadas',
          appsDesc:
            'Puedes usar Google Authenticator o Microsoft Authenticator. Ambas son gratuitas y funcionan en iOS y Android.',
          setupTitle: 'Cómo configurarla',
          setupDesc:
            'Abre tu aplicación de autenticación, escanea el código QR que aparece durante la configuración o introduce manualmente la clave de configuración. Luego introduce el código de 6 dígitos de la aplicación para terminar de activar 2FA.',
          recoveryTitle: 'Códigos de recuperación',
          recoveryDesc:
            'Los códigos de recuperación son códigos de respaldo de un solo uso. Si pierdes acceso a tu aplicación de autenticación, puedes usar un código de recuperación para iniciar sesión. Guárdalos en un lugar seguro inmediatamente después de generarlos.',
          logoutNotice:
            'Activar 2FA cerrará todas las sesiones activas, incluida esta. Tendrás que iniciar sesión de nuevo en cada dispositivo.'
        }
      }
    },
    error: {
      title: 'Hemos encontrado un problema',
      refresh: 'Actualizar'
    },
    pictureInPicture: {
      title: 'Imagen en imagen',
      enter: 'Activar imagen en imagen',
      exit: 'Desactivar imagen en imagen',
      error:
        'No se pudo iniciar imagen en imagen. Espera a que el vídeo se cargue e inténtalo de nuevo.'
    },
    fullscreen: {
      toggle: 'Activar/Desactivar pantalla completa'
    },
    menu: {
      collapse: 'Colapsar menú',
      expand: 'Expandir menú'
    }
  }
};

export default es;
