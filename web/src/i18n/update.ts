import type { SupportedLanguage } from './locale.ts';

type UpdateMessages = {
  server?: UpdateServerMessages;
  signatureVerification?: SignatureVerificationMessages;
  region?: RegionMessages;
  offlineTitle: string;
  selectFile: string;
  dropFile: string;
  fileHint: string;
  invalidFileName: string;
  offlineConfirmTitle: string;
  offlineConfirmDesc: string;
  offlineConfirm: string;
  cancelUpload: string;
  powerWarning: string;
  sourceOnline: string;
  sourceOffline: string;
  upload: string;
  verify: string;
  prepare: string;
  statusUnavailable: string;
  updateSucceeded: string;
  continue: string;
  updateFailedTitle: string;
  retry: string;
  phases: {
    idle: string;
    downloading: string;
    uploading: string;
    verifying: string;
    staging: string;
    installing: string;
    restarting: string;
    succeeded: string;
    failed: string;
  };
  errors: {
    busy: string;
    upload_interrupted: string;
    no_space: string;
    invalid_archive: string;
    signature_invalid: string;
    unsupported_package: string;
    same_version: string;
    install_start_failed: string;
    install_failed: string;
    interrupted: string;
    generic: string;
  };
};

type RegionMessages = {
  title: string;
  cn: string;
  global: string;
  confirmTitle: string;
  confirmDesc: string;
  loadFailed: string;
  changeFailed: string;
  cleanupWarning: string;
};

type UpdateServerMessages = {
  title: string;
  description: string;
  save: string;
  loadFailed: string;
  saveFailed: string;
  invalidURL: string;
};

type SignatureVerificationMessages = {
  title: string;
  disabledDesc: string;
  loadFailed: string;
  changeFailed: string;
  disableAuthDesc: string;
  enableAuthDesc: string;
  disableConfirm: string;
  enableConfirm: string;
};

const en: UpdateMessages = {
  offlineTitle: 'Offline update',
  selectFile: 'Select bundle',
  dropFile: 'Drop an update bundle here, or click to select one',
  fileHint: 'Select nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Select a nanokvm-go_<version>.tar.gz update bundle.',
  offlineConfirmTitle: 'Install offline update?',
  offlineConfirmDesc: '{{name}} will be verified and installed. NanoKVM services will restart.',
  offlineConfirm: 'Upload and install',
  cancelUpload: 'Cancel upload',
  powerWarning: 'Do not turn off the device during the update.',
  sourceOnline: 'Online update',
  sourceOffline: 'Offline update',
  upload: 'Upload',
  verify: 'Verify',
  prepare: 'Prepare',
  statusUnavailable:
    'The updater is temporarily unavailable. Reopen this page to check the result.',
  updateSucceeded: 'Update installed successfully',
  continue: 'Continue',
  updateFailedTitle: 'Update not installed',
  retry: 'Check again',
  phases: {
    idle: 'Ready',
    downloading: 'Downloading the update…',
    uploading: 'Uploading the update…',
    verifying: 'Verifying the signed packages…',
    staging: 'Preparing the verified packages…',
    installing: 'Installing the update…',
    restarting: 'Restarting NanoKVM services…',
    succeeded: 'Update installed',
    failed: 'Update failed'
  },
  errors: {
    busy: 'Another maintenance operation is already running.',
    upload_interrupted: 'The upload was interrupted. Select the file and try again.',
    no_space: 'There is not enough free storage for this update.',
    invalid_archive: 'The selected file is not a valid NanoKVM update bundle.',
    signature_invalid: 'The update signature is missing or invalid.',
    unsupported_package: 'The update contains an unsupported package or architecture.',
    same_version: 'This version is already installed.',
    install_start_failed: 'The verified update could not start installing.',
    install_failed: 'Package installation failed.',
    interrupted: 'The update operation was interrupted.',
    generic: 'The update failed. Check the bundle and try again.'
  }
};

const zh: UpdateMessages = {
  offlineTitle: '离线更新',
  selectFile: '选择更新包',
  dropFile: '将更新包拖到此处，或点击选择文件',
  fileHint: '请选择 nanokvm-go_<version>.tar.gz',
  invalidFileName: '请选择 nanokvm-go_<version>.tar.gz 更新包。',
  offlineConfirmTitle: '安装离线更新？',
  offlineConfirmDesc: '将验证并安装 {{name}}，期间 NanoKVM 服务会重启。',
  offlineConfirm: '上传并安装',
  cancelUpload: '取消上传',
  powerWarning: '更新期间请勿断电',
  sourceOnline: '在线更新',
  sourceOffline: '离线更新',
  upload: '上传',
  verify: '验证',
  prepare: '准备',
  statusUnavailable: '更新服务暂时不可用，请重新打开此页面确认结果。',
  updateSucceeded: '更新安装成功',
  continue: '继续',
  updateFailedTitle: '更新未安装',
  retry: '重新检查',
  phases: {
    idle: '就绪',
    downloading: '正在下载更新…',
    uploading: '正在上传更新…',
    verifying: '正在验证签名包…',
    staging: '正在准备已验证的软件包…',
    installing: '正在安装更新…',
    restarting: '正在重启 NanoKVM 服务…',
    succeeded: '更新已安装',
    failed: '更新失败'
  },
  errors: {
    busy: '已有其他维护操作正在进行。',
    upload_interrupted: '上传已中断，请重新选择文件后重试。',
    no_space: '存储空间不足，无法完成更新。',
    invalid_archive: '所选文件不是有效的 NanoKVM 更新包。',
    signature_invalid: '更新签名缺失或无效。',
    unsupported_package: '更新包包含不支持的软件包或架构。',
    same_version: '此版本已经安装。',
    install_start_failed: '已验证的更新无法开始安装。',
    install_failed: '软件包安装失败。',
    interrupted: '更新操作被中断。',
    generic: '更新失败，请检查更新包后重试。'
  }
};

const zh_tw: UpdateMessages = {
  offlineTitle: '離線更新',
  selectFile: '選擇更新包',
  dropFile: '將更新包拖到此處，或按一下選擇檔案',
  fileHint: '請選擇 nanokvm-go_<version>.tar.gz',
  invalidFileName: '請選擇 nanokvm-go_<version>.tar.gz 更新包。',
  offlineConfirmTitle: '安裝離線更新？',
  offlineConfirmDesc: '將驗證並安裝 {{name}}，期間 NanoKVM 服務會重新啟動。',
  offlineConfirm: '上傳並安裝',
  cancelUpload: '取消上傳',
  powerWarning: '更新期間請勿斷電',
  sourceOnline: '線上更新',
  sourceOffline: '離線更新',
  upload: '上傳',
  verify: '驗證',
  prepare: '準備',
  statusUnavailable: '更新服務暫時無法使用，請重新開啟此頁面確認結果。',
  updateSucceeded: '更新安裝成功',
  continue: '繼續',
  updateFailedTitle: '更新未安裝',
  retry: '重新檢查',
  phases: {
    idle: '就緒',
    downloading: '正在下載更新…',
    uploading: '正在上傳更新…',
    verifying: '正在驗證已簽署套件…',
    staging: '正在準備已驗證套件…',
    installing: '正在安裝更新…',
    restarting: '正在重新啟動 NanoKVM 服務…',
    succeeded: '更新已安裝',
    failed: '更新失敗'
  },
  errors: {
    busy: '已有其他維護作業正在進行。',
    upload_interrupted: '上傳已中斷，請重新選擇檔案後再試一次。',
    no_space: '儲存空間不足，無法完成更新。',
    invalid_archive: '所選檔案不是有效的 NanoKVM 更新包。',
    signature_invalid: '更新簽章遺失或無效。',
    unsupported_package: '更新包含不支援的套件或架構。',
    same_version: '此版本已經安裝。',
    install_start_failed: '已驗證的更新無法開始安裝。',
    install_failed: '套件安裝失敗。',
    interrupted: '更新作業已中斷。',
    generic: '更新失敗，請檢查更新包後再試一次。'
  }
};

const de: UpdateMessages = {
  offlineTitle: 'Offline-Aktualisierung',
  selectFile: 'Paket auswählen',
  dropFile: 'Aktualisierungspaket hier ablegen oder zum Auswählen klicken',
  fileHint: 'nanokvm-go_<version>.tar.gz auswählen',
  invalidFileName: 'Wählen Sie ein Aktualisierungspaket nanokvm-go_<version>.tar.gz aus.',
  offlineConfirmTitle: 'Offline-Aktualisierung installieren?',
  offlineConfirmDesc:
    '{{name}} wird geprüft und installiert. Die NanoKVM-Dienste werden neu gestartet.',
  offlineConfirm: 'Hochladen und installieren',
  cancelUpload: 'Hochladen abbrechen',
  powerWarning: 'Schalten Sie das Gerät während der Aktualisierung nicht aus.',
  sourceOnline: 'Online-Aktualisierung',
  sourceOffline: 'Offline-Aktualisierung',
  upload: 'Hochladen',
  verify: 'Prüfen',
  prepare: 'Vorbereiten',
  statusUnavailable:
    'Der Aktualisierungsdienst ist vorübergehend nicht erreichbar. Öffnen Sie diese Seite erneut, um das Ergebnis zu prüfen.',
  updateSucceeded: 'Aktualisierung erfolgreich installiert',
  continue: 'Weiter',
  updateFailedTitle: 'Aktualisierung nicht installiert',
  retry: 'Erneut prüfen',
  phases: {
    idle: 'Bereit',
    downloading: 'Aktualisierung wird heruntergeladen…',
    uploading: 'Aktualisierung wird hochgeladen…',
    verifying: 'Signierte Pakete werden geprüft…',
    staging: 'Geprüfte Pakete werden vorbereitet…',
    installing: 'Aktualisierung wird installiert…',
    restarting: 'NanoKVM-Dienste werden neu gestartet…',
    succeeded: 'Aktualisierung installiert',
    failed: 'Aktualisierung fehlgeschlagen'
  },
  errors: {
    busy: 'Ein anderer Wartungsvorgang läuft bereits.',
    upload_interrupted: 'Das Hochladen wurde unterbrochen. Wählen Sie die Datei erneut aus.',
    no_space: 'Für diese Aktualisierung ist nicht genügend Speicherplatz verfügbar.',
    invalid_archive: 'Die gewählte Datei ist kein gültiges NanoKVM-Aktualisierungspaket.',
    signature_invalid: 'Die Signatur der Aktualisierung fehlt oder ist ungültig.',
    unsupported_package:
      'Die Aktualisierung enthält ein nicht unterstütztes Paket oder eine nicht unterstützte Architektur.',
    same_version: 'Diese Version ist bereits installiert.',
    install_start_failed: 'Die geprüfte Aktualisierung konnte nicht installiert werden.',
    install_failed: 'Die Paketinstallation ist fehlgeschlagen.',
    interrupted: 'Der Aktualisierungsvorgang wurde unterbrochen.',
    generic:
      'Die Aktualisierung ist fehlgeschlagen. Prüfen Sie das Paket und versuchen Sie es erneut.'
  }
};

const fr: UpdateMessages = {
  offlineTitle: 'Mise à jour hors ligne',
  selectFile: 'Choisir le paquet',
  dropFile: 'Déposez le paquet ici ou cliquez pour le sélectionner',
  fileHint: 'Sélectionnez nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Sélectionnez un paquet nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Installer la mise à jour hors ligne ?',
  offlineConfirmDesc: '{{name}} sera vérifié et installé. Les services NanoKVM redémarreront.',
  offlineConfirm: 'Téléverser et installer',
  cancelUpload: 'Annuler le téléversement',
  powerWarning: 'N’éteignez pas l’appareil pendant la mise à jour.',
  sourceOnline: 'Mise à jour en ligne',
  sourceOffline: 'Mise à jour hors ligne',
  upload: 'Téléverser',
  verify: 'Vérifier',
  prepare: 'Préparer',
  statusUnavailable:
    'Le service de mise à jour est temporairement indisponible. Rouvrez cette page pour vérifier le résultat.',
  updateSucceeded: 'Mise à jour installée',
  continue: 'Continuer',
  updateFailedTitle: 'Mise à jour non installée',
  retry: 'Vérifier à nouveau',
  phases: {
    idle: 'Prêt',
    downloading: 'Téléchargement de la mise à jour…',
    uploading: 'Téléversement de la mise à jour…',
    verifying: 'Vérification des paquets signés…',
    staging: 'Préparation des paquets vérifiés…',
    installing: 'Installation de la mise à jour…',
    restarting: 'Redémarrage des services NanoKVM…',
    succeeded: 'Mise à jour installée',
    failed: 'Échec de la mise à jour'
  },
  errors: {
    busy: 'Une autre opération de maintenance est déjà en cours.',
    upload_interrupted: 'Le téléversement a été interrompu. Sélectionnez le fichier et réessayez.',
    no_space: 'L’espace de stockage est insuffisant pour cette mise à jour.',
    invalid_archive: 'Le fichier sélectionné n’est pas un paquet de mise à jour NanoKVM valide.',
    signature_invalid: 'La signature de la mise à jour est absente ou invalide.',
    unsupported_package:
      'La mise à jour contient un paquet ou une architecture non pris en charge.',
    same_version: 'Cette version est déjà installée.',
    install_start_failed: 'L’installation de la mise à jour vérifiée n’a pas pu démarrer.',
    install_failed: 'L’installation du paquet a échoué.',
    interrupted: 'La mise à jour a été interrompue.',
    generic: 'La mise à jour a échoué. Vérifiez le paquet et réessayez.'
  }
};

const es: UpdateMessages = {
  offlineTitle: 'Actualización sin conexión',
  selectFile: 'Seleccionar paquete',
  dropFile: 'Suelta aquí el paquete o haz clic para seleccionarlo',
  fileHint: 'Selecciona nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Selecciona un paquete nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: '¿Instalar la actualización sin conexión?',
  offlineConfirmDesc:
    'Se verificará e instalará {{name}}. Los servicios de NanoKVM se reiniciarán.',
  offlineConfirm: 'Subir e instalar',
  cancelUpload: 'Cancelar carga',
  powerWarning: 'No apague el dispositivo durante la actualización.',
  sourceOnline: 'Actualización en línea',
  sourceOffline: 'Actualización sin conexión',
  upload: 'Subir',
  verify: 'Verificar',
  prepare: 'Preparar',
  statusUnavailable:
    'El servicio de actualización no está disponible temporalmente. Vuelve a abrir esta página para comprobar el resultado.',
  updateSucceeded: 'Actualización instalada correctamente',
  continue: 'Continuar',
  updateFailedTitle: 'Actualización no instalada',
  retry: 'Comprobar de nuevo',
  phases: {
    idle: 'Listo',
    downloading: 'Descargando la actualización…',
    uploading: 'Subiendo la actualización…',
    verifying: 'Verificando los paquetes firmados…',
    staging: 'Preparando los paquetes verificados…',
    installing: 'Instalando la actualización…',
    restarting: 'Reiniciando los servicios de NanoKVM…',
    succeeded: 'Actualización instalada',
    failed: 'Error de actualización'
  },
  errors: {
    busy: 'Ya hay otra operación de mantenimiento en curso.',
    upload_interrupted: 'La carga se interrumpió. Selecciona el archivo e inténtalo de nuevo.',
    no_space: 'No hay espacio libre suficiente para esta actualización.',
    invalid_archive: 'El archivo seleccionado no es un paquete de actualización NanoKVM válido.',
    signature_invalid: 'Falta la firma de la actualización o no es válida.',
    unsupported_package: 'La actualización contiene un paquete o arquitectura no compatible.',
    same_version: 'Esta versión ya está instalada.',
    install_start_failed: 'No se pudo iniciar la instalación de la actualización verificada.',
    install_failed: 'La instalación del paquete falló.',
    interrupted: 'La operación de actualización se interrumpió.',
    generic: 'La actualización falló. Comprueba el paquete e inténtalo de nuevo.'
  }
};

const it: UpdateMessages = {
  offlineTitle: 'Aggiornamento offline',
  selectFile: 'Seleziona pacchetto',
  dropFile: 'Trascina qui il pacchetto o fai clic per selezionarlo',
  fileHint: 'Seleziona nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Seleziona un pacchetto nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Installare l’aggiornamento offline?',
  offlineConfirmDesc:
    '{{name}} verrà verificato e installato. I servizi NanoKVM verranno riavviati.',
  offlineConfirm: 'Carica e installa',
  cancelUpload: 'Annulla caricamento',
  powerWarning: 'Non spegnere il dispositivo durante l’aggiornamento.',
  sourceOnline: 'Aggiornamento online',
  sourceOffline: 'Aggiornamento offline',
  upload: 'Carica',
  verify: 'Verifica',
  prepare: 'Prepara',
  statusUnavailable:
    'Il servizio di aggiornamento è temporaneamente non disponibile. Riapri questa pagina per controllare il risultato.',
  updateSucceeded: 'Aggiornamento installato',
  continue: 'Continua',
  updateFailedTitle: 'Aggiornamento non installato',
  retry: 'Controlla di nuovo',
  phases: {
    idle: 'Pronto',
    downloading: 'Download dell’aggiornamento…',
    uploading: 'Caricamento dell’aggiornamento…',
    verifying: 'Verifica dei pacchetti firmati…',
    staging: 'Preparazione dei pacchetti verificati…',
    installing: 'Installazione dell’aggiornamento…',
    restarting: 'Riavvio dei servizi NanoKVM…',
    succeeded: 'Aggiornamento installato',
    failed: 'Aggiornamento non riuscito'
  },
  errors: {
    busy: 'È già in corso un’altra operazione di manutenzione.',
    upload_interrupted: 'Il caricamento è stato interrotto. Seleziona il file e riprova.',
    no_space: 'Spazio libero insufficiente per questo aggiornamento.',
    invalid_archive: 'Il file selezionato non è un pacchetto di aggiornamento NanoKVM valido.',
    signature_invalid: 'La firma dell’aggiornamento manca o non è valida.',
    unsupported_package: 'L’aggiornamento contiene un pacchetto o un’architettura non supportati.',
    same_version: 'Questa versione è già installata.',
    install_start_failed: 'Impossibile avviare l’installazione dell’aggiornamento verificato.',
    install_failed: 'L’installazione del pacchetto non è riuscita.',
    interrupted: 'L’operazione di aggiornamento è stata interrotta.',
    generic: 'Aggiornamento non riuscito. Controlla il pacchetto e riprova.'
  }
};

const ja: UpdateMessages = {
  offlineTitle: 'オフライン更新',
  selectFile: '更新パッケージを選択',
  dropFile: '更新パッケージをここにドロップするか、クリックして選択',
  fileHint: 'nanokvm-go_<version>.tar.gz を選択してください',
  invalidFileName: 'nanokvm-go_<version>.tar.gz 更新パッケージを選択してください。',
  offlineConfirmTitle: 'オフライン更新をインストールしますか？',
  offlineConfirmDesc: '{{name}} を検証してインストールします。NanoKVM サービスは再起動します。',
  offlineConfirm: 'アップロードしてインストール',
  cancelUpload: 'アップロードをキャンセル',
  powerWarning: '更新中は電源を切らないでください。',
  sourceOnline: 'オンライン更新',
  sourceOffline: 'オフライン更新',
  upload: 'アップロード',
  verify: '検証',
  prepare: '準備',
  statusUnavailable:
    '更新サービスに一時的に接続できません。結果を確認するには、このページを再度開いてください。',
  updateSucceeded: '更新をインストールしました',
  continue: '続行',
  updateFailedTitle: '更新はインストールされませんでした',
  retry: 'もう一度確認',
  phases: {
    idle: '準備完了',
    downloading: '更新をダウンロードしています…',
    uploading: '更新をアップロードしています…',
    verifying: '署名付きパッケージを検証しています…',
    staging: '検証済みパッケージを準備しています…',
    installing: '更新をインストールしています…',
    restarting: 'NanoKVM サービスを再起動しています…',
    succeeded: '更新をインストールしました',
    failed: '更新に失敗しました'
  },
  errors: {
    busy: '別のメンテナンス処理がすでに実行中です。',
    upload_interrupted: 'アップロードが中断されました。ファイルを選択し直してください。',
    no_space: 'この更新に必要な空き容量がありません。',
    invalid_archive: '選択したファイルは有効な NanoKVM 更新パッケージではありません。',
    signature_invalid: '更新の署名がないか、無効です。',
    unsupported_package: '更新に未対応のパッケージまたはアーキテクチャが含まれています。',
    same_version: 'このバージョンはすでにインストールされています。',
    install_start_failed: '検証済み更新のインストールを開始できませんでした。',
    install_failed: 'パッケージのインストールに失敗しました。',
    interrupted: '更新処理が中断されました。',
    generic: '更新に失敗しました。パッケージを確認して再試行してください。'
  }
};

const ko: UpdateMessages = {
  offlineTitle: '오프라인 업데이트',
  selectFile: '업데이트 패키지 선택',
  dropFile: '업데이트 패키지를 여기에 놓거나 클릭하여 선택하세요',
  fileHint: 'nanokvm-go_<version>.tar.gz를 선택하세요',
  invalidFileName: 'nanokvm-go_<version>.tar.gz 업데이트 패키지를 선택하세요.',
  offlineConfirmTitle: '오프라인 업데이트를 설치할까요?',
  offlineConfirmDesc: '{{name}}을 검증하고 설치합니다. NanoKVM 서비스가 다시 시작됩니다.',
  offlineConfirm: '업로드 및 설치',
  cancelUpload: '업로드 취소',
  powerWarning: '업데이트 중에는 전원을 끄지 마세요.',
  sourceOnline: '온라인 업데이트',
  sourceOffline: '오프라인 업데이트',
  upload: '업로드',
  verify: '검증',
  prepare: '준비',
  statusUnavailable:
    '업데이트 서비스를 일시적으로 사용할 수 없습니다. 결과를 확인하려면 이 페이지를 다시 여세요.',
  updateSucceeded: '업데이트 설치 완료',
  continue: '계속',
  updateFailedTitle: '업데이트가 설치되지 않음',
  retry: '다시 확인',
  phases: {
    idle: '준비됨',
    downloading: '업데이트 다운로드 중…',
    uploading: '업데이트 업로드 중…',
    verifying: '서명된 패키지 검증 중…',
    staging: '검증된 패키지 준비 중…',
    installing: '업데이트 설치 중…',
    restarting: 'NanoKVM 서비스 다시 시작 중…',
    succeeded: '업데이트 설치됨',
    failed: '업데이트 실패'
  },
  errors: {
    busy: '다른 유지 관리 작업이 이미 실행 중입니다.',
    upload_interrupted: '업로드가 중단되었습니다. 파일을 다시 선택하세요.',
    no_space: '업데이트에 필요한 저장 공간이 부족합니다.',
    invalid_archive: '선택한 파일은 올바른 NanoKVM 업데이트 패키지가 아닙니다.',
    signature_invalid: '업데이트 서명이 없거나 올바르지 않습니다.',
    unsupported_package: '업데이트에 지원되지 않는 패키지 또는 아키텍처가 포함되어 있습니다.',
    same_version: '이 버전은 이미 설치되어 있습니다.',
    install_start_failed: '검증된 업데이트 설치를 시작하지 못했습니다.',
    install_failed: '패키지 설치에 실패했습니다.',
    interrupted: '업데이트 작업이 중단되었습니다.',
    generic: '업데이트에 실패했습니다. 패키지를 확인하고 다시 시도하세요.'
  }
};

const ru: UpdateMessages = {
  offlineTitle: 'Автономное обновление',
  selectFile: 'Выбрать пакет',
  dropFile: 'Перетащите пакет сюда или нажмите для выбора',
  fileHint: 'Выберите nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Выберите пакет обновления nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Установить автономное обновление?',
  offlineConfirmDesc: '{{name}} будет проверен и установлен. Службы NanoKVM перезапустятся.',
  offlineConfirm: 'Загрузить и установить',
  cancelUpload: 'Отменить загрузку',
  powerWarning: 'Не отключайте питание во время обновления.',
  sourceOnline: 'Сетевое обновление',
  sourceOffline: 'Автономное обновление',
  upload: 'Загрузить',
  verify: 'Проверка',
  prepare: 'Подготовка',
  statusUnavailable:
    'Служба обновления временно недоступна. Откройте страницу снова, чтобы проверить результат.',
  updateSucceeded: 'Обновление успешно установлено',
  continue: 'Продолжить',
  updateFailedTitle: 'Обновление не установлено',
  retry: 'Проверить снова',
  phases: {
    idle: 'Готово',
    downloading: 'Загрузка обновления…',
    uploading: 'Отправка обновления…',
    verifying: 'Проверка подписанных пакетов…',
    staging: 'Подготовка проверенных пакетов…',
    installing: 'Установка обновления…',
    restarting: 'Перезапуск служб NanoKVM…',
    succeeded: 'Обновление установлено',
    failed: 'Ошибка обновления'
  },
  errors: {
    busy: 'Уже выполняется другая операция обслуживания.',
    upload_interrupted: 'Отправка была прервана. Выберите файл и повторите попытку.',
    no_space: 'Недостаточно свободного места для обновления.',
    invalid_archive: 'Выбранный файл не является допустимым пакетом обновления NanoKVM.',
    signature_invalid: 'Подпись обновления отсутствует или недействительна.',
    unsupported_package: 'Обновление содержит неподдерживаемый пакет или архитектуру.',
    same_version: 'Эта версия уже установлена.',
    install_start_failed: 'Не удалось начать установку проверенного обновления.',
    install_failed: 'Не удалось установить пакет.',
    interrupted: 'Операция обновления была прервана.',
    generic: 'Не удалось обновить устройство. Проверьте пакет и повторите попытку.'
  }
};

const uk: UpdateMessages = {
  offlineTitle: 'Автономне оновлення',
  selectFile: 'Вибрати пакет',
  dropFile: 'Перетягніть пакет сюди або натисніть для вибору',
  fileHint: 'Виберіть nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Виберіть пакет оновлення nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Установити автономне оновлення?',
  offlineConfirmDesc: '{{name}} буде перевірено й установлено. Служби NanoKVM перезапустяться.',
  offlineConfirm: 'Передати й установити',
  cancelUpload: 'Скасувати передавання',
  powerWarning: 'Не вимикайте живлення під час оновлення.',
  sourceOnline: 'Мережеве оновлення',
  sourceOffline: 'Автономне оновлення',
  upload: 'Завантажити',
  verify: 'Перевірити',
  prepare: 'Підготувати',
  statusUnavailable:
    'Служба оновлення тимчасово недоступна. Відкрийте сторінку знову, щоб перевірити результат.',
  updateSucceeded: 'Оновлення успішно встановлено',
  continue: 'Продовжити',
  updateFailedTitle: 'Оновлення не встановлено',
  retry: 'Перевірити знову',
  phases: {
    idle: 'Готово',
    downloading: 'Завантаження оновлення…',
    uploading: 'Передавання оновлення…',
    verifying: 'Перевірка підписаних пакетів…',
    staging: 'Підготовка перевірених пакетів…',
    installing: 'Установлення оновлення…',
    restarting: 'Перезапуск служб NanoKVM…',
    succeeded: 'Оновлення встановлено',
    failed: 'Помилка оновлення'
  },
  errors: {
    busy: 'Уже виконується інша операція обслуговування.',
    upload_interrupted: 'Передавання перервано. Виберіть файл і повторіть спробу.',
    no_space: 'Недостатньо вільного місця для оновлення.',
    invalid_archive: 'Вибраний файл не є дійсним пакетом оновлення NanoKVM.',
    signature_invalid: 'Підпис оновлення відсутній або недійсний.',
    unsupported_package: 'Оновлення містить непідтримуваний пакет або архітектуру.',
    same_version: 'Цю версію вже встановлено.',
    install_start_failed: 'Не вдалося розпочати встановлення перевіреного оновлення.',
    install_failed: 'Не вдалося встановити пакет.',
    interrupted: 'Операцію оновлення перервано.',
    generic: 'Не вдалося оновити пристрій. Перевірте пакет і повторіть спробу.'
  }
};

const nl: UpdateMessages = {
  offlineTitle: 'Offline-update',
  selectFile: 'Pakket selecteren',
  dropFile: 'Sleep het updatepakket hierheen of klik om het te selecteren',
  fileHint: 'Selecteer nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Selecteer een updatepakket nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Offline-update installeren?',
  offlineConfirmDesc:
    '{{name}} wordt gecontroleerd en geïnstalleerd. De NanoKVM-services worden opnieuw gestart.',
  offlineConfirm: 'Uploaden en installeren',
  cancelUpload: 'Upload annuleren',
  powerWarning: 'Schakel het apparaat niet uit tijdens de update.',
  sourceOnline: 'Online-update',
  sourceOffline: 'Offline-update',
  upload: 'Uploaden',
  verify: 'Controleren',
  prepare: 'Voorbereiden',
  statusUnavailable:
    'De updateservice is tijdelijk niet beschikbaar. Open deze pagina opnieuw om het resultaat te controleren.',
  updateSucceeded: 'Update geïnstalleerd',
  continue: 'Doorgaan',
  updateFailedTitle: 'Update niet geïnstalleerd',
  retry: 'Opnieuw controleren',
  phases: {
    idle: 'Gereed',
    downloading: 'Update downloaden…',
    uploading: 'Update uploaden…',
    verifying: 'Ondertekende pakketten controleren…',
    staging: 'Gecontroleerde pakketten voorbereiden…',
    installing: 'Update installeren…',
    restarting: 'NanoKVM-services opnieuw starten…',
    succeeded: 'Update geïnstalleerd',
    failed: 'Update mislukt'
  },
  errors: {
    busy: 'Er wordt al een andere onderhoudstaak uitgevoerd.',
    upload_interrupted: 'De upload is onderbroken. Selecteer het bestand opnieuw.',
    no_space: 'Er is onvoldoende vrije opslagruimte voor deze update.',
    invalid_archive: 'Het gekozen bestand is geen geldig NanoKVM-updatepakket.',
    signature_invalid: 'De updatehandtekening ontbreekt of is ongeldig.',
    unsupported_package: 'De update bevat een niet-ondersteund pakket of architectuur.',
    same_version: 'Deze versie is al geïnstalleerd.',
    install_start_failed: 'De installatie van de gecontroleerde update kon niet starten.',
    install_failed: 'De pakketinstallatie is mislukt.',
    interrupted: 'De updatebewerking is onderbroken.',
    generic: 'De update is mislukt. Controleer het pakket en probeer opnieuw.'
  }
};

const pl: UpdateMessages = {
  offlineTitle: 'Aktualizacja offline',
  selectFile: 'Wybierz pakiet',
  dropFile: 'Upuść pakiet tutaj lub kliknij, aby go wybrać',
  fileHint: 'Wybierz nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Wybierz pakiet aktualizacji nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Zainstalować aktualizację offline?',
  offlineConfirmDesc:
    '{{name}} zostanie zweryfikowany i zainstalowany. Usługi NanoKVM zostaną uruchomione ponownie.',
  offlineConfirm: 'Prześlij i zainstaluj',
  cancelUpload: 'Anuluj przesyłanie',
  powerWarning: 'Nie wyłączaj urządzenia podczas aktualizacji.',
  sourceOnline: 'Aktualizacja online',
  sourceOffline: 'Aktualizacja offline',
  upload: 'Prześlij',
  verify: 'Weryfikuj',
  prepare: 'Przygotuj',
  statusUnavailable:
    'Usługa aktualizacji jest tymczasowo niedostępna. Otwórz tę stronę ponownie, aby sprawdzić wynik.',
  updateSucceeded: 'Aktualizacja zainstalowana',
  continue: 'Kontynuuj',
  updateFailedTitle: 'Aktualizacja nie została zainstalowana',
  retry: 'Sprawdź ponownie',
  phases: {
    idle: 'Gotowe',
    downloading: 'Pobieranie aktualizacji…',
    uploading: 'Przesyłanie aktualizacji…',
    verifying: 'Weryfikowanie podpisanych pakietów…',
    staging: 'Przygotowywanie zweryfikowanych pakietów…',
    installing: 'Instalowanie aktualizacji…',
    restarting: 'Ponowne uruchamianie usług NanoKVM…',
    succeeded: 'Aktualizacja zainstalowana',
    failed: 'Aktualizacja nie powiodła się'
  },
  errors: {
    busy: 'Inna operacja konserwacji jest już wykonywana.',
    upload_interrupted: 'Przesyłanie zostało przerwane. Wybierz plik ponownie.',
    no_space: 'Brak wystarczającej ilości wolnego miejsca na aktualizację.',
    invalid_archive: 'Wybrany plik nie jest prawidłowym pakietem aktualizacji NanoKVM.',
    signature_invalid: 'Brakuje podpisu aktualizacji lub jest on nieprawidłowy.',
    unsupported_package: 'Aktualizacja zawiera nieobsługiwany pakiet lub architekturę.',
    same_version: 'Ta wersja jest już zainstalowana.',
    install_start_failed: 'Nie udało się rozpocząć instalowania zweryfikowanej aktualizacji.',
    install_failed: 'Instalacja pakietu nie powiodła się.',
    interrupted: 'Operacja aktualizacji została przerwana.',
    generic: 'Aktualizacja nie powiodła się. Sprawdź pakiet i spróbuj ponownie.'
  }
};

const da: UpdateMessages = {
  offlineTitle: 'Offlineopdatering',
  selectFile: 'Vælg pakke',
  dropFile: 'Slip opdateringspakken her, eller klik for at vælge den',
  fileHint: 'Vælg nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Vælg en nanokvm-go_<version>.tar.gz-opdateringspakke.',
  offlineConfirmTitle: 'Installer offlineopdateringen?',
  offlineConfirmDesc:
    '{{name}} bliver kontrolleret og installeret. NanoKVM-tjenesterne genstartes.',
  offlineConfirm: 'Upload og installer',
  cancelUpload: 'Annuller upload',
  powerWarning: 'Sluk ikke enheden under opdateringen.',
  sourceOnline: 'Onlineopdatering',
  sourceOffline: 'Offlineopdatering',
  upload: 'Upload',
  verify: 'Kontrollér',
  prepare: 'Forbered',
  statusUnavailable:
    'Opdateringstjenesten er midlertidigt utilgængelig. Åbn siden igen for at kontrollere resultatet.',
  updateSucceeded: 'Opdateringen er installeret',
  continue: 'Fortsæt',
  updateFailedTitle: 'Opdateringen blev ikke installeret',
  retry: 'Kontrollér igen',
  phases: {
    idle: 'Klar',
    downloading: 'Downloader opdateringen…',
    uploading: 'Uploader opdateringen…',
    verifying: 'Kontrollerer signerede pakker…',
    staging: 'Forbereder kontrollerede pakker…',
    installing: 'Installerer opdateringen…',
    restarting: 'Genstarter NanoKVM-tjenester…',
    succeeded: 'Opdateringen er installeret',
    failed: 'Opdateringen mislykkedes'
  },
  errors: {
    busy: 'En anden vedligeholdelseshandling kører allerede.',
    upload_interrupted: 'Uploaden blev afbrudt. Vælg filen igen.',
    no_space: 'Der er ikke nok ledig lagerplads til opdateringen.',
    invalid_archive: 'Den valgte fil er ikke en gyldig NanoKVM-opdateringspakke.',
    signature_invalid: 'Opdateringens signatur mangler eller er ugyldig.',
    unsupported_package:
      'Opdateringen indeholder en pakke eller arkitektur, der ikke understøttes.',
    same_version: 'Denne version er allerede installeret.',
    install_start_failed: 'Installationen af den kontrollerede opdatering kunne ikke starte.',
    install_failed: 'Pakkeinstallationen mislykkedes.',
    interrupted: 'Opdateringshandlingen blev afbrudt.',
    generic: 'Opdateringen mislykkedes. Kontrollér pakken, og prøv igen.'
  }
};

const nb: UpdateMessages = {
  offlineTitle: 'Frakoblet oppdatering',
  selectFile: 'Velg pakke',
  dropFile: 'Slipp oppdateringspakken her, eller klikk for å velge den',
  fileHint: 'Velg nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Velg en nanokvm-go_<version>.tar.gz-oppdateringspakke.',
  offlineConfirmTitle: 'Installere frakoblet oppdatering?',
  offlineConfirmDesc:
    '{{name}} blir kontrollert og installert. NanoKVM-tjenestene starter på nytt.',
  offlineConfirm: 'Last opp og installer',
  cancelUpload: 'Avbryt opplasting',
  powerWarning: 'Ikke slå av enheten under oppdateringen.',
  sourceOnline: 'Nettoppdatering',
  sourceOffline: 'Frakoblet oppdatering',
  upload: 'Last opp',
  verify: 'Kontroller',
  prepare: 'Forbered',
  statusUnavailable:
    'Oppdateringstjenesten er midlertidig utilgjengelig. Åpne siden på nytt for å kontrollere resultatet.',
  updateSucceeded: 'Oppdateringen er installert',
  continue: 'Fortsett',
  updateFailedTitle: 'Oppdateringen ble ikke installert',
  retry: 'Kontroller på nytt',
  phases: {
    idle: 'Klar',
    downloading: 'Laster ned oppdateringen…',
    uploading: 'Laster opp oppdateringen…',
    verifying: 'Kontrollerer signerte pakker…',
    staging: 'Forbereder kontrollerte pakker…',
    installing: 'Installerer oppdateringen…',
    restarting: 'Starter NanoKVM-tjenestene på nytt…',
    succeeded: 'Oppdateringen er installert',
    failed: 'Oppdateringen mislyktes'
  },
  errors: {
    busy: 'En annen vedlikeholdsoperasjon kjører allerede.',
    upload_interrupted: 'Opplastingen ble avbrutt. Velg filen på nytt.',
    no_space: 'Det er ikke nok ledig lagringsplass til oppdateringen.',
    invalid_archive: 'Den valgte filen er ikke en gyldig NanoKVM-oppdateringspakke.',
    signature_invalid: 'Oppdateringssignaturen mangler eller er ugyldig.',
    unsupported_package: 'Oppdateringen inneholder en pakke eller arkitektur som ikke støttes.',
    same_version: 'Denne versjonen er allerede installert.',
    install_start_failed: 'Installasjonen av den kontrollerte oppdateringen kunne ikke starte.',
    install_failed: 'Pakkeinstallasjonen mislyktes.',
    interrupted: 'Oppdateringsoperasjonen ble avbrutt.',
    generic: 'Oppdateringen mislyktes. Kontroller pakken og prøv igjen.'
  }
};

const id: UpdateMessages = {
  offlineTitle: 'Pembaruan offline',
  selectFile: 'Pilih paket',
  dropFile: 'Letakkan paket pembaruan di sini atau klik untuk memilih',
  fileHint: 'Pilih nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Pilih paket pembaruan nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Pasang pembaruan offline?',
  offlineConfirmDesc:
    '{{name}} akan diverifikasi dan dipasang. Layanan NanoKVM akan dimulai ulang.',
  offlineConfirm: 'Unggah dan pasang',
  cancelUpload: 'Batalkan unggahan',
  powerWarning: 'Jangan matikan perangkat selama pembaruan.',
  sourceOnline: 'Pembaruan daring',
  sourceOffline: 'Pembaruan offline',
  upload: 'Unggah',
  verify: 'Verifikasi',
  prepare: 'Siapkan',
  statusUnavailable:
    'Layanan pembaruan sementara tidak tersedia. Buka kembali halaman ini untuk memeriksa hasilnya.',
  updateSucceeded: 'Pembaruan berhasil dipasang',
  continue: 'Lanjutkan',
  updateFailedTitle: 'Pembaruan tidak dipasang',
  retry: 'Periksa lagi',
  phases: {
    idle: 'Siap',
    downloading: 'Mengunduh pembaruan…',
    uploading: 'Mengunggah pembaruan…',
    verifying: 'Memverifikasi paket bertanda tangan…',
    staging: 'Menyiapkan paket yang telah diverifikasi…',
    installing: 'Memasang pembaruan…',
    restarting: 'Memulai ulang layanan NanoKVM…',
    succeeded: 'Pembaruan terpasang',
    failed: 'Pembaruan gagal'
  },
  errors: {
    busy: 'Operasi pemeliharaan lain sedang berjalan.',
    upload_interrupted: 'Unggahan terputus. Pilih kembali berkas dan coba lagi.',
    no_space: 'Ruang penyimpanan tidak cukup untuk pembaruan ini.',
    invalid_archive: 'Berkas yang dipilih bukan paket pembaruan NanoKVM yang valid.',
    signature_invalid: 'Tanda tangan pembaruan hilang atau tidak valid.',
    unsupported_package: 'Pembaruan berisi paket atau arsitektur yang tidak didukung.',
    same_version: 'Versi ini sudah terpasang.',
    install_start_failed: 'Pemasangan pembaruan terverifikasi tidak dapat dimulai.',
    install_failed: 'Pemasangan paket gagal.',
    interrupted: 'Operasi pembaruan terputus.',
    generic: 'Pembaruan gagal. Periksa paket lalu coba lagi.'
  }
};

const vi: UpdateMessages = {
  offlineTitle: 'Cập nhật ngoại tuyến',
  selectFile: 'Chọn gói cập nhật',
  dropFile: 'Thả gói cập nhật vào đây hoặc nhấp để chọn',
  fileHint: 'Chọn nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Hãy chọn gói cập nhật nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Cài đặt bản cập nhật ngoại tuyến?',
  offlineConfirmDesc: '{{name}} sẽ được xác minh và cài đặt. Các dịch vụ NanoKVM sẽ khởi động lại.',
  offlineConfirm: 'Tải lên và cài đặt',
  cancelUpload: 'Hủy tải lên',
  powerWarning: 'Không tắt nguồn trong khi cập nhật.',
  sourceOnline: 'Cập nhật trực tuyến',
  sourceOffline: 'Cập nhật ngoại tuyến',
  upload: 'Tải lên',
  verify: 'Xác minh',
  prepare: 'Chuẩn bị',
  statusUnavailable:
    'Dịch vụ cập nhật tạm thời không khả dụng. Hãy mở lại trang này để kiểm tra kết quả.',
  updateSucceeded: 'Đã cài đặt bản cập nhật',
  continue: 'Tiếp tục',
  updateFailedTitle: 'Chưa cài đặt bản cập nhật',
  retry: 'Kiểm tra lại',
  phases: {
    idle: 'Sẵn sàng',
    downloading: 'Đang tải bản cập nhật xuống…',
    uploading: 'Đang tải bản cập nhật lên…',
    verifying: 'Đang xác minh các gói đã ký…',
    staging: 'Đang chuẩn bị các gói đã xác minh…',
    installing: 'Đang cài đặt bản cập nhật…',
    restarting: 'Đang khởi động lại dịch vụ NanoKVM…',
    succeeded: 'Đã cài đặt bản cập nhật',
    failed: 'Cập nhật thất bại'
  },
  errors: {
    busy: 'Một tác vụ bảo trì khác đang chạy.',
    upload_interrupted: 'Quá trình tải lên bị gián đoạn. Hãy chọn lại tệp.',
    no_space: 'Không đủ dung lượng trống cho bản cập nhật này.',
    invalid_archive: 'Tệp đã chọn không phải gói cập nhật NanoKVM hợp lệ.',
    signature_invalid: 'Chữ ký cập nhật bị thiếu hoặc không hợp lệ.',
    unsupported_package: 'Bản cập nhật chứa gói hoặc kiến trúc không được hỗ trợ.',
    same_version: 'Phiên bản này đã được cài đặt.',
    install_start_failed: 'Không thể bắt đầu cài đặt bản cập nhật đã xác minh.',
    install_failed: 'Cài đặt gói thất bại.',
    interrupted: 'Tác vụ cập nhật đã bị gián đoạn.',
    generic: 'Cập nhật thất bại. Hãy kiểm tra gói rồi thử lại.'
  }
};

const th: UpdateMessages = {
  offlineTitle: 'อัปเดตแบบออฟไลน์',
  selectFile: 'เลือกแพ็กเกจ',
  dropFile: 'วางแพ็กเกจอัปเดตที่นี่ หรือคลิกเพื่อเลือก',
  fileHint: 'เลือก nanokvm-go_<version>.tar.gz',
  invalidFileName: 'โปรดเลือกแพ็กเกจอัปเดต nanokvm-go_<version>.tar.gz',
  offlineConfirmTitle: 'ติดตั้งอัปเดตแบบออฟไลน์หรือไม่',
  offlineConfirmDesc: 'ระบบจะตรวจสอบและติดตั้ง {{name}} แล้วเริ่มบริการ NanoKVM ใหม่',
  offlineConfirm: 'อัปโหลดและติดตั้ง',
  cancelUpload: 'ยกเลิกการอัปโหลด',
  powerWarning: 'อย่าปิดเครื่องระหว่างการอัปเดต',
  sourceOnline: 'อัปเดตออนไลน์',
  sourceOffline: 'อัปเดตออฟไลน์',
  upload: 'อัปโหลด',
  verify: 'ตรวจสอบ',
  prepare: 'เตรียม',
  statusUnavailable: 'บริการอัปเดตไม่พร้อมใช้งานชั่วคราว โปรดเปิดหน้านี้อีกครั้งเพื่อตรวจสอบผล',
  updateSucceeded: 'ติดตั้งอัปเดตสำเร็จ',
  continue: 'ดำเนินการต่อ',
  updateFailedTitle: 'ไม่ได้ติดตั้งอัปเดต',
  retry: 'ตรวจสอบอีกครั้ง',
  phases: {
    idle: 'พร้อม',
    downloading: 'กำลังดาวน์โหลดอัปเดต…',
    uploading: 'กำลังอัปโหลดอัปเดต…',
    verifying: 'กำลังตรวจสอบแพ็กเกจที่ลงนาม…',
    staging: 'กำลังเตรียมแพ็กเกจที่ตรวจสอบแล้ว…',
    installing: 'กำลังติดตั้งอัปเดต…',
    restarting: 'กำลังเริ่มบริการ NanoKVM ใหม่…',
    succeeded: 'ติดตั้งอัปเดตแล้ว',
    failed: 'อัปเดตล้มเหลว'
  },
  errors: {
    busy: 'มีงานบำรุงรักษาอื่นกำลังทำงานอยู่',
    upload_interrupted: 'การอัปโหลดถูกขัดจังหวะ โปรดเลือกไฟล์อีกครั้ง',
    no_space: 'พื้นที่จัดเก็บไม่เพียงพอสำหรับอัปเดตนี้',
    invalid_archive: 'ไฟล์ที่เลือกไม่ใช่แพ็กเกจอัปเดต NanoKVM ที่ถูกต้อง',
    signature_invalid: 'ลายเซ็นอัปเดตหายไปหรือไม่ถูกต้อง',
    unsupported_package: 'อัปเดตมีแพ็กเกจหรือสถาปัตยกรรมที่ไม่รองรับ',
    same_version: 'ติดตั้งเวอร์ชันนี้แล้ว',
    install_start_failed: 'ไม่สามารถเริ่มติดตั้งอัปเดตที่ตรวจสอบแล้ว',
    install_failed: 'การติดตั้งแพ็กเกจล้มเหลว',
    interrupted: 'งานอัปเดตถูกขัดจังหวะ',
    generic: 'อัปเดตล้มเหลว โปรดตรวจสอบแพ็กเกจแล้วลองอีกครั้ง'
  }
};

const hu: UpdateMessages = {
  offlineTitle: 'Offline frissítés',
  selectFile: 'Csomag kiválasztása',
  dropFile: 'Húzza ide a frissítési csomagot, vagy kattintson a kiválasztáshoz',
  fileHint: 'Válassza a nanokvm-go_<version>.tar.gz fájlt',
  invalidFileName: 'Válasszon nanokvm-go_<version>.tar.gz frissítési csomagot.',
  offlineConfirmTitle: 'Telepíti az offline frissítést?',
  offlineConfirmDesc:
    'A(z) {{name}} ellenőrzése és telepítése után a NanoKVM szolgáltatásai újraindulnak.',
  offlineConfirm: 'Feltöltés és telepítés',
  cancelUpload: 'Feltöltés megszakítása',
  powerWarning: 'Frissítés közben ne kapcsolja ki az eszközt.',
  sourceOnline: 'Online frissítés',
  sourceOffline: 'Offline frissítés',
  upload: 'Feltöltés',
  verify: 'Ellenőrzés',
  prepare: 'Előkészítés',
  statusUnavailable:
    'A frissítési szolgáltatás átmenetileg nem érhető el. Nyissa meg újra az oldalt az eredmény ellenőrzéséhez.',
  updateSucceeded: 'A frissítés sikeresen települt',
  continue: 'Folytatás',
  updateFailedTitle: 'A frissítés nem települt',
  retry: 'Újraellenőrzés',
  phases: {
    idle: 'Kész',
    downloading: 'Frissítés letöltése…',
    uploading: 'Frissítés feltöltése…',
    verifying: 'Aláírt csomagok ellenőrzése…',
    staging: 'Ellenőrzött csomagok előkészítése…',
    installing: 'Frissítés telepítése…',
    restarting: 'NanoKVM-szolgáltatások újraindítása…',
    succeeded: 'A frissítés települt',
    failed: 'A frissítés sikertelen'
  },
  errors: {
    busy: 'Már fut egy másik karbantartási művelet.',
    upload_interrupted: 'A feltöltés megszakadt. Válassza ki újra a fájlt.',
    no_space: 'Nincs elegendő szabad tárhely a frissítéshez.',
    invalid_archive: 'A kiválasztott fájl nem érvényes NanoKVM frissítési csomag.',
    signature_invalid: 'A frissítés aláírása hiányzik vagy érvénytelen.',
    unsupported_package: 'A frissítés nem támogatott csomagot vagy architektúrát tartalmaz.',
    same_version: 'Ez a verzió már telepítve van.',
    install_start_failed: 'Az ellenőrzött frissítés telepítése nem indítható el.',
    install_failed: 'A csomag telepítése sikertelen.',
    interrupted: 'A frissítési művelet megszakadt.',
    generic: 'A frissítés sikertelen. Ellenőrizze a csomagot, majd próbálja újra.'
  }
};

const cz: UpdateMessages = {
  offlineTitle: 'Offline aktualizace',
  selectFile: 'Vybrat balíček',
  dropFile: 'Přetáhněte balíček sem nebo jej vyberte kliknutím',
  fileHint: 'Vyberte nanokvm-go_<version>.tar.gz',
  invalidFileName: 'Vyberte aktualizační balíček nanokvm-go_<version>.tar.gz.',
  offlineConfirmTitle: 'Nainstalovat offline aktualizaci?',
  offlineConfirmDesc: '{{name}} bude ověřen a nainstalován. Služby NanoKVM se restartují.',
  offlineConfirm: 'Nahrát a nainstalovat',
  cancelUpload: 'Zrušit nahrávání',
  powerWarning: 'Během aktualizace zařízení nevypínejte.',
  sourceOnline: 'Online aktualizace',
  sourceOffline: 'Offline aktualizace',
  upload: 'Nahrát',
  verify: 'Ověřit',
  prepare: 'Připravit',
  statusUnavailable:
    'Aktualizační služba je dočasně nedostupná. Výsledek ověřte opětovným otevřením stránky.',
  updateSucceeded: 'Aktualizace byla nainstalována',
  continue: 'Pokračovat',
  updateFailedTitle: 'Aktualizace nebyla nainstalována',
  retry: 'Zkontrolovat znovu',
  phases: {
    idle: 'Připraveno',
    downloading: 'Stahování aktualizace…',
    uploading: 'Nahrávání aktualizace…',
    verifying: 'Ověřování podepsaných balíčků…',
    staging: 'Příprava ověřených balíčků…',
    installing: 'Instalace aktualizace…',
    restarting: 'Restartování služeb NanoKVM…',
    succeeded: 'Aktualizace nainstalována',
    failed: 'Aktualizace selhala'
  },
  errors: {
    busy: 'Již probíhá jiná operace údržby.',
    upload_interrupted: 'Nahrávání bylo přerušeno. Vyberte soubor znovu.',
    no_space: 'Pro tuto aktualizaci není dostatek volného místa.',
    invalid_archive: 'Vybraný soubor není platný aktualizační balíček NanoKVM.',
    signature_invalid: 'Podpis aktualizace chybí nebo je neplatný.',
    unsupported_package: 'Aktualizace obsahuje nepodporovaný balíček nebo architekturu.',
    same_version: 'Tato verze je již nainstalována.',
    install_start_failed: 'Instalaci ověřené aktualizace se nepodařilo spustit.',
    install_failed: 'Instalace balíčku selhala.',
    interrupted: 'Operace aktualizace byla přerušena.',
    generic: 'Aktualizace selhala. Zkontrolujte balíček a zkuste to znovu.'
  }
};

const locales: Record<SupportedLanguage, UpdateMessages> = {
  en,
  zh,
  zh_tw,
  de,
  fr,
  es,
  it,
  ja,
  ko,
  ru,
  uk,
  nl,
  pl,
  da,
  nb,
  id,
  vi,
  th,
  hu,
  cz
};

const signatureVerificationLocales: Record<SupportedLanguage, SignatureVerificationMessages> = {
  en: {
    title: 'Update signature verification',
    disabledDesc: 'Signature verification is off. Updates cannot be trusted.',
    loadFailed: 'Could not read this setting. Verification remains enabled for safety.',
    changeFailed: 'Could not change signature verification.',
    disableAuthDesc:
      'Danger: online and offline updates will no longer be authenticated and may run untrusted code as root.',
    enableAuthDesc: 'Authenticate to require signature verification for all application updates.',
    disableConfirm: 'Disable verification',
    enableConfirm: 'Enable verification'
  },
  zh: {
    title: '更新签名校验',
    disabledDesc: '签名校验已关闭，无法确认更新包是否可信。',
    loadFailed: '无法读取此设置，为安全起见保持开启显示。',
    changeFailed: '无法修改签名校验设置。',
    disableAuthDesc: '在线和离线更新将不再验证来源，可能运行不受信任的代码。',
    enableAuthDesc: '请验证身份，以便对所有应用更新启用签名校验。',
    disableConfirm: '关闭校验',
    enableConfirm: '开启校验'
  },
  zh_tw: {
    title: '更新簽章驗證',
    disabledDesc: '簽章驗證已關閉，無法確認更新套件是否可信。',
    loadFailed: '無法讀取此設定，為安全起見維持開啟顯示。',
    changeFailed: '無法變更簽章驗證設定。',
    disableAuthDesc: '危險：線上和離線更新將不再驗證來源，可能以 root 權限執行不受信任的程式碼。',
    enableAuthDesc: '請驗證身分，以便為所有應用程式更新啟用簽章驗證。',
    disableConfirm: '關閉驗證',
    enableConfirm: '開啟驗證'
  },
  de: {
    title: 'Signaturprüfung für Aktualisierungen',
    disabledDesc: 'Die Signaturprüfung ist aus. Aktualisierungen sind nicht vertrauenswürdig.',
    loadFailed:
      'Die Einstellung konnte nicht gelesen werden. Die Prüfung bleibt sicherheitshalber an.',
    changeFailed: 'Die Signaturprüfung konnte nicht geändert werden.',
    disableAuthDesc:
      'Gefahr: Online- und Offline-Aktualisierungen werden nicht mehr authentifiziert und können nicht vertrauenswürdigen Code als root ausführen.',
    enableAuthDesc: 'Authentifizieren Sie sich, um alle Aktualisierungssignaturen zu prüfen.',
    disableConfirm: 'Prüfung deaktivieren',
    enableConfirm: 'Prüfung aktivieren'
  },
  fr: {
    title: 'Vérification de signature des mises à jour',
    disabledDesc: 'La vérification est désactivée. Les mises à jour ne sont pas fiables.',
    loadFailed: 'Impossible de lire ce réglage. La vérification reste activée par sécurité.',
    changeFailed: 'Impossible de modifier la vérification de signature.',
    disableAuthDesc:
      'Danger : les mises à jour ne seront plus authentifiées et pourront exécuter du code non fiable en tant que root.',
    enableAuthDesc: 'Authentifiez-vous pour vérifier la signature de toutes les mises à jour.',
    disableConfirm: 'Désactiver la vérification',
    enableConfirm: 'Activer la vérification'
  },
  es: {
    title: 'Verificación de firma de actualizaciones',
    disabledDesc: 'La verificación está desactivada. No se puede confiar en las actualizaciones.',
    loadFailed: 'No se pudo leer este ajuste. La verificación sigue activada por seguridad.',
    changeFailed: 'No se pudo cambiar la verificación de firma.',
    disableAuthDesc:
      'Peligro: las actualizaciones dejarán de autenticarse y podrán ejecutar código no confiable como root.',
    enableAuthDesc: 'Autentícate para verificar la firma de todas las actualizaciones.',
    disableConfirm: 'Desactivar verificación',
    enableConfirm: 'Activar verificación'
  },
  it: {
    title: 'Verifica firma degli aggiornamenti',
    disabledDesc: 'La verifica è disattivata. Gli aggiornamenti non sono attendibili.',
    loadFailed: 'Impossibile leggere questa impostazione. La verifica resta attiva per sicurezza.',
    changeFailed: 'Impossibile modificare la verifica della firma.',
    disableAuthDesc:
      'Pericolo: gli aggiornamenti non saranno più autenticati e potranno eseguire codice non attendibile come root.',
    enableAuthDesc: 'Autenticati per verificare la firma di tutti gli aggiornamenti.',
    disableConfirm: 'Disattiva verifica',
    enableConfirm: 'Attiva verifica'
  },
  ja: {
    title: '更新の署名検証',
    disabledDesc: '署名検証は無効です。更新を信頼できません。',
    loadFailed: '設定を読み込めません。安全のため検証は有効として扱われます。',
    changeFailed: '署名検証の設定を変更できません。',
    disableAuthDesc:
      '危険：更新元が検証されなくなり、信頼できないコードが root 権限で実行される可能性があります。',
    enableAuthDesc: 'すべてのアプリ更新で署名検証を有効にするには認証してください。',
    disableConfirm: '検証を無効化',
    enableConfirm: '検証を有効化'
  },
  ko: {
    title: '업데이트 서명 검증',
    disabledDesc: '서명 검증이 꺼져 있어 업데이트를 신뢰할 수 없습니다.',
    loadFailed: '설정을 읽을 수 없습니다. 안전을 위해 검증을 켠 상태로 표시합니다.',
    changeFailed: '서명 검증 설정을 변경할 수 없습니다.',
    disableAuthDesc:
      '위험: 업데이트 출처가 더 이상 검증되지 않으며 신뢰할 수 없는 코드가 root 권한으로 실행될 수 있습니다.',
    enableAuthDesc: '모든 앱 업데이트의 서명을 검증하려면 인증하세요.',
    disableConfirm: '검증 끄기',
    enableConfirm: '검증 켜기'
  },
  ru: {
    title: 'Проверка подписи обновлений',
    disabledDesc: 'Проверка отключена. Обновлениям нельзя доверять.',
    loadFailed: 'Не удалось прочитать настройку. Для безопасности проверка считается включённой.',
    changeFailed: 'Не удалось изменить проверку подписи.',
    disableAuthDesc:
      'Опасно: обновления больше не будут проверяться и смогут выполнять недоверенный код от root.',
    enableAuthDesc: 'Подтвердите личность, чтобы проверять подписи всех обновлений.',
    disableConfirm: 'Отключить проверку',
    enableConfirm: 'Включить проверку'
  },
  uk: {
    title: 'Перевірка підпису оновлень',
    disabledDesc: 'Перевірку вимкнено. Оновленням не можна довіряти.',
    loadFailed:
      'Не вдалося прочитати налаштування. З міркувань безпеки перевірка вважається ввімкненою.',
    changeFailed: 'Не вдалося змінити перевірку підпису.',
    disableAuthDesc:
      'Небезпечно: оновлення більше не перевірятимуться та зможуть виконувати ненадійний код від root.',
    enableAuthDesc: 'Підтвердьте особу, щоб перевіряти підписи всіх оновлень.',
    disableConfirm: 'Вимкнути перевірку',
    enableConfirm: 'Увімкнути перевірку'
  },
  nl: {
    title: 'Handtekeningcontrole voor updates',
    disabledDesc: 'De controle staat uit. Updates kunnen niet worden vertrouwd.',
    loadFailed:
      'Deze instelling kon niet worden gelezen. De controle blijft voor de veiligheid aan.',
    changeFailed: 'De handtekeningcontrole kon niet worden gewijzigd.',
    disableAuthDesc:
      'Gevaar: updates worden niet meer geverifieerd en kunnen niet-vertrouwde code als root uitvoeren.',
    enableAuthDesc: 'Verifieer uw identiteit om alle updatehandtekeningen te controleren.',
    disableConfirm: 'Controle uitschakelen',
    enableConfirm: 'Controle inschakelen'
  },
  pl: {
    title: 'Weryfikacja podpisu aktualizacji',
    disabledDesc: 'Weryfikacja jest wyłączona. Aktualizacjom nie można ufać.',
    loadFailed: 'Nie można odczytać ustawienia. Dla bezpieczeństwa weryfikacja pozostaje włączona.',
    changeFailed: 'Nie można zmienić weryfikacji podpisu.',
    disableAuthDesc:
      'Niebezpieczeństwo: aktualizacje nie będą uwierzytelniane i mogą uruchomić niezaufany kod jako root.',
    enableAuthDesc: 'Uwierzytelnij się, aby weryfikować podpisy wszystkich aktualizacji.',
    disableConfirm: 'Wyłącz weryfikację',
    enableConfirm: 'Włącz weryfikację'
  },
  da: {
    title: 'Signaturkontrol af opdateringer',
    disabledDesc: 'Signaturkontrol er slået fra. Opdateringer kan ikke betragtes som sikre.',
    loadFailed:
      'Indstillingen kunne ikke læses. Kontrollen forbliver slået til af sikkerhedshensyn.',
    changeFailed: 'Signaturkontrollen kunne ikke ændres.',
    disableAuthDesc:
      'Fare: opdateringer godkendes ikke længere og kan køre kode, der ikke er tillid til, som root.',
    enableAuthDesc: 'Godkend dig for at kontrollere signaturen på alle opdateringer.',
    disableConfirm: 'Deaktivér kontrol',
    enableConfirm: 'Aktivér kontrol'
  },
  nb: {
    title: 'Signaturkontroll for oppdateringer',
    disabledDesc: 'Signaturkontroll er av. Oppdateringer kan ikke anses som pålitelige.',
    loadFailed: 'Innstillingen kunne ikke leses. Kontrollen forblir på av sikkerhetshensyn.',
    changeFailed: 'Signaturkontrollen kunne ikke endres.',
    disableAuthDesc:
      'Fare: oppdateringer godkjennes ikke lenger og kan kjøre uklarert kode som root.',
    enableAuthDesc: 'Bekreft identiteten din for å kontrollere alle oppdateringssignaturer.',
    disableConfirm: 'Deaktiver kontroll',
    enableConfirm: 'Aktiver kontroll'
  },
  id: {
    title: 'Verifikasi tanda tangan pembaruan',
    disabledDesc: 'Verifikasi dimatikan. Pembaruan tidak dapat dipercaya.',
    loadFailed: 'Pengaturan tidak dapat dibaca. Verifikasi tetap aktif demi keamanan.',
    changeFailed: 'Verifikasi tanda tangan tidak dapat diubah.',
    disableAuthDesc:
      'Bahaya: pembaruan tidak lagi diautentikasi dan dapat menjalankan kode tidak tepercaya sebagai root.',
    enableAuthDesc: 'Autentikasi untuk memverifikasi tanda tangan semua pembaruan.',
    disableConfirm: 'Nonaktifkan verifikasi',
    enableConfirm: 'Aktifkan verifikasi'
  },
  vi: {
    title: 'Xác minh chữ ký bản cập nhật',
    disabledDesc: 'Đã tắt xác minh. Không thể tin cậy các bản cập nhật.',
    loadFailed: 'Không thể đọc cài đặt. Xác minh vẫn được bật để bảo đảm an toàn.',
    changeFailed: 'Không thể thay đổi xác minh chữ ký.',
    disableAuthDesc:
      'Nguy hiểm: bản cập nhật sẽ không còn được xác thực và có thể chạy mã không đáng tin cậy với quyền root.',
    enableAuthDesc: 'Xác thực để kiểm tra chữ ký của mọi bản cập nhật.',
    disableConfirm: 'Tắt xác minh',
    enableConfirm: 'Bật xác minh'
  },
  th: {
    title: 'ตรวจสอบลายเซ็นการอัปเดต',
    disabledDesc: 'ปิดการตรวจสอบแล้ว จึงไม่สามารถเชื่อถือการอัปเดตได้',
    loadFailed: 'อ่านการตั้งค่าไม่ได้ ระบบจะแสดงว่าเปิดการตรวจสอบไว้เพื่อความปลอดภัย',
    changeFailed: 'เปลี่ยนการตรวจสอบลายเซ็นไม่ได้',
    disableAuthDesc:
      'อันตราย: ระบบจะไม่ยืนยันแหล่งที่มาของการอัปเดต และอาจเรียกใช้โค้ดที่ไม่น่าเชื่อถือด้วยสิทธิ์ root',
    enableAuthDesc: 'ยืนยันตัวตนเพื่อเปิดการตรวจสอบลายเซ็นสำหรับทุกการอัปเดต',
    disableConfirm: 'ปิดการตรวจสอบ',
    enableConfirm: 'เปิดการตรวจสอบ'
  },
  hu: {
    title: 'Frissítési aláírás ellenőrzése',
    disabledDesc: 'Az ellenőrzés ki van kapcsolva. A frissítések nem tekinthetők megbízhatónak.',
    loadFailed: 'A beállítás nem olvasható. Biztonsági okból az ellenőrzés bekapcsolva marad.',
    changeFailed: 'Az aláírás-ellenőrzés nem módosítható.',
    disableAuthDesc:
      'Veszély: a frissítések hitelesítése megszűnik, és nem megbízható kódot futtathatnak root jogosultsággal.',
    enableAuthDesc: 'Hitelesítse magát minden frissítés aláírásának ellenőrzéséhez.',
    disableConfirm: 'Ellenőrzés kikapcsolása',
    enableConfirm: 'Ellenőrzés bekapcsolása'
  },
  cz: {
    title: 'Ověření podpisu aktualizací',
    disabledDesc: 'Ověření je vypnuté. Aktualizacím nelze důvěřovat.',
    loadFailed: 'Nastavení nelze načíst. Z bezpečnostních důvodů zůstává ověření zapnuté.',
    changeFailed: 'Ověření podpisu nelze změnit.',
    disableAuthDesc:
      'Nebezpečí: aktualizace již nebudou ověřovány a mohou spouštět nedůvěryhodný kód jako root.',
    enableAuthDesc: 'Ověřte svou totožnost, aby se kontrolovaly podpisy všech aktualizací.',
    disableConfirm: 'Vypnout ověření',
    enableConfirm: 'Zapnout ověření'
  }
};

const regionLocales: Record<SupportedLanguage, RegionMessages> = {
  en: {
    title: 'Region',
    cn: 'China Mainland',
    global: 'Global',
    confirmTitle: 'Switch service region?',
    confirmDesc: 'Switch to {{region}}? The kvmcomm service will restart.',
    loadFailed: 'The current service region could not be loaded.',
    changeFailed: 'The service region could not be changed.',
    cleanupWarning: 'Switched to Global, but the Sipeed cloud network could not be deleted.'
  },
  zh: {
    title: '区域',
    cn: '中国大陆',
    global: '全球',
    confirmTitle: '切换区域？',
    confirmDesc: '确定切换到“{{region}}”吗？kvmcomm 服务将会重启。',
    loadFailed: '无法读取当前区域。',
    changeFailed: '无法切换区域。',
    cleanupWarning: '已切换到全球，但无法删除 Sipeed 云端网络。'
  },
  zh_tw: {
    title: '區域',
    cn: '中國大陸',
    global: '全球',
    confirmTitle: '切換區域？',
    confirmDesc: '確定切換到「{{region}}」嗎？kvmcomm 服務將會重新啟動。',
    loadFailed: '無法讀取目前的區域。',
    changeFailed: '無法切換區域。',
    cleanupWarning: '已切換到全球，但無法刪除 Sipeed 雲端網路。'
  },
  nl: {
    title: 'Regio',
    cn: 'Vasteland van China',
    global: 'Wereldwijd',
    confirmTitle: 'Serviceregio wijzigen?',
    confirmDesc: 'Overschakelen naar {{region}}?',
    loadFailed: 'De huidige serviceregio kon niet worden geladen.',
    changeFailed: 'De serviceregio kon niet worden gewijzigd.',
    cleanupWarning:
      'Er is overgeschakeld naar Wereldwijd, maar het Sipeed-cloudnetwerk kon niet worden verwijderd.'
  },
  da: {
    title: 'Område',
    cn: 'Fastlandskina',
    global: 'Globalt',
    confirmTitle: 'Skift tjenesteområde?',
    confirmDesc: 'Skift til {{region}}?',
    loadFailed: 'Det aktuelle tjenesteområde kunne ikke indlæses.',
    changeFailed: 'Tjenesteområdet kunne ikke ændres.',
    cleanupWarning: 'Der blev skiftet til Globalt, men Sipeed-netværket i skyen kunne ikke slettes.'
  },
  de: {
    title: 'Serviceregion',
    cn: 'Festlandchina',
    global: 'Weltweit',
    confirmTitle: 'Serviceregion wechseln?',
    confirmDesc: 'Zu {{region}} wechseln?',
    loadFailed: 'Die aktuelle Serviceregion konnte nicht geladen werden.',
    changeFailed: 'Die Serviceregion konnte nicht geändert werden.',
    cleanupWarning:
      'Es wurde zu Weltweit gewechselt, aber das Sipeed-Cloud-Netzwerk konnte nicht gelöscht werden.'
  },
  es: {
    title: 'Región',
    cn: 'China continental',
    global: 'Mundial',
    confirmTitle: '¿Cambiar la región del servicio?',
    confirmDesc: '¿Cambiar a {{region}}?',
    loadFailed: 'No se pudo cargar la región actual del servicio.',
    changeFailed: 'No se pudo cambiar la región del servicio.',
    cleanupWarning: 'Se cambió a Mundial, pero no se pudo eliminar la red de Sipeed en la nube.'
  },
  fr: {
    title: 'Région',
    cn: 'Chine continentale',
    global: 'Monde',
    confirmTitle: 'Changer de région de service ?',
    confirmDesc: 'Passer à {{region}} ?',
    loadFailed: 'Impossible de charger la région de service actuelle.',
    changeFailed: 'Impossible de changer de région de service.',
    cleanupWarning:
      'La région Monde est active, mais le réseau cloud Sipeed n’a pas pu être supprimé.'
  },
  id: {
    title: 'Wilayah',
    cn: 'Tiongkok Daratan',
    global: 'Seluruh dunia',
    confirmTitle: 'Ganti wilayah layanan?',
    confirmDesc: 'Beralih ke {{region}}?',
    loadFailed: 'Wilayah layanan saat ini tidak dapat dimuat.',
    changeFailed: 'Wilayah layanan tidak dapat diubah.',
    cleanupWarning:
      'Wilayah telah dialihkan ke Seluruh dunia, tetapi jaringan cloud Sipeed tidak dapat dihapus.'
  },
  it: {
    title: 'Regione',
    cn: 'Cina continentale',
    global: 'Globale',
    confirmTitle: 'Cambiare la regione del servizio?',
    confirmDesc: 'Passare a {{region}}?',
    loadFailed: 'Impossibile caricare la regione del servizio corrente.',
    changeFailed: 'Impossibile cambiare la regione del servizio.',
    cleanupWarning:
      'La regione è stata impostata su Globale, ma non è stato possibile eliminare la rete cloud Sipeed.'
  },
  pl: {
    title: 'Obszar',
    cn: 'Chiny kontynentalne',
    global: 'Globalny',
    confirmTitle: 'Zmienić region usługi?',
    confirmDesc: 'Przełączyć na {{region}}?',
    loadFailed: 'Nie można wczytać bieżącego regionu usługi.',
    changeFailed: 'Nie można zmienić regionu usługi.',
    cleanupWarning:
      'Przełączono na region Globalny, ale nie udało się usunąć sieci Sipeed w chmurze.'
  },
  ru: {
    title: 'Регион',
    cn: 'Материковый Китай',
    global: 'Весь мир',
    confirmTitle: 'Изменить регион сервиса?',
    confirmDesc: 'Переключиться на {{region}}?',
    loadFailed: 'Не удалось загрузить текущий регион сервиса.',
    changeFailed: 'Не удалось изменить регион сервиса.',
    cleanupWarning: 'Регион переключён на «Весь мир», но удалить облачную сеть Sipeed не удалось.'
  },
  ko: {
    title: '지역',
    cn: '중국 본토',
    global: '전 세계',
    confirmTitle: '서비스 지역을 변경할까요?',
    confirmDesc: '{{region}}(으)로 변경할까요?',
    loadFailed: '현재 서비스 지역을 불러오지 못했습니다.',
    changeFailed: '서비스 지역을 변경하지 못했습니다.',
    cleanupWarning: '전 세계 지역으로 전환했지만 Sipeed 클라우드 네트워크를 삭제하지 못했습니다.'
  },
  hu: {
    title: 'Régió',
    cn: 'Kontinentális Kína',
    global: 'Globális',
    confirmTitle: 'Módosítja a szolgáltatási régiót?',
    confirmDesc: 'Átvált erre: {{region}}?',
    loadFailed: 'A jelenlegi szolgáltatási régió nem tölthető be.',
    changeFailed: 'A szolgáltatási régió nem módosítható.',
    cleanupWarning:
      'A régió Globális értékre váltott, de a Sipeed felhőhálózatot nem sikerült törölni.'
  },
  vi: {
    title: 'Khu vực',
    cn: 'Trung Quốc đại lục',
    global: 'Toàn cầu',
    confirmTitle: 'Chuyển khu vực dịch vụ?',
    confirmDesc: 'Chuyển sang {{region}}?',
    loadFailed: 'Không thể tải khu vực dịch vụ hiện tại.',
    changeFailed: 'Không thể thay đổi khu vực dịch vụ.',
    cleanupWarning: 'Đã chuyển sang Toàn cầu nhưng không thể xóa mạng đám mây Sipeed.'
  },
  ja: {
    title: 'リージョン',
    cn: '中国大陸',
    global: 'グローバル',
    confirmTitle: 'サービスリージョンを切り替えますか？',
    confirmDesc: '{{region}}に切り替えますか？',
    loadFailed: '現在のサービスリージョンを読み込めませんでした。',
    changeFailed: 'サービスリージョンを変更できませんでした。',
    cleanupWarning:
      'グローバルへ切り替えましたが、Sipeed クラウドネットワークを削除できませんでした。'
  },
  cz: {
    title: 'Oblast',
    cn: 'Pevninská Čína',
    global: 'Celý svět',
    confirmTitle: 'Změnit oblast služby?',
    confirmDesc: 'Přepnout na {{region}}?',
    loadFailed: 'Aktuální oblast služby nelze načíst.',
    changeFailed: 'Oblast služby nelze změnit.',
    cleanupWarning:
      'Oblast byla přepnuta na Celý svět, ale cloudovou síť Sipeed se nepodařilo odstranit.'
  },
  uk: {
    title: 'Регіон',
    cn: 'Материковий Китай',
    global: 'Увесь світ',
    confirmTitle: 'Змінити регіон сервісу?',
    confirmDesc: 'Перемкнутися на {{region}}?',
    loadFailed: 'Не вдалося завантажити поточний регіон сервісу.',
    changeFailed: 'Не вдалося змінити регіон сервісу.',
    cleanupWarning:
      'Регіон перемкнено на «Увесь світ», але видалити хмарну мережу Sipeed не вдалося.'
  },
  nb: {
    title: 'Område',
    cn: 'Fastlands-Kina',
    global: 'Globalt',
    confirmTitle: 'Bytte tjenesteregion?',
    confirmDesc: 'Bytte til {{region}}?',
    loadFailed: 'Den gjeldende tjenesteregionen kunne ikke lastes inn.',
    changeFailed: 'Tjenesteregionen kunne ikke endres.',
    cleanupWarning:
      'Området ble byttet til Globalt, men Sipeed-nettverket i skyen kunne ikke slettes.'
  },
  th: {
    title: 'ภูมิภาค',
    cn: 'จีนแผ่นดินใหญ่',
    global: 'ทั่วโลก',
    confirmTitle: 'เปลี่ยนภูมิภาคบริการหรือไม่',
    confirmDesc: 'เปลี่ยนเป็น {{region}} หรือไม่',
    loadFailed: 'โหลดภูมิภาคบริการปัจจุบันไม่ได้',
    changeFailed: 'เปลี่ยนภูมิภาคบริการไม่ได้',
    cleanupWarning: 'เปลี่ยนเป็นทั่วโลกแล้ว แต่ไม่สามารถลบเครือข่าย Sipeed บนคลาวด์ได้'
  }
};

const updateServerLocales: Record<SupportedLanguage, UpdateServerMessages> = {
  en: {
    title: 'Custom Update Server',
    description: 'Check for and download online updates from a specified server',
    save: 'Save',
    loadFailed: 'Could not load the update server setting. Try again.',
    saveFailed: 'Could not save the update server. Try again.',
    invalidURL:
      'Enter an HTTP/HTTPS directory URL without credentials, query parameters or a fragment.'
  },
  zh: {
    title: '自定义更新服务器',
    description: '从指定服务器检查并下载在线更新',
    save: '保存',
    loadFailed: '无法读取更新服务器设置，请重试。',
    saveFailed: '无法保存更新服务器，请重试。',
    invalidURL: '请输入有效的 HTTP/HTTPS 目录地址，不包含用户名、密码、查询参数或片段。'
  },
  zh_tw: {
    title: '自訂更新伺服器',
    description: '從指定伺服器檢查並下載線上更新',
    save: '儲存',
    loadFailed: '無法讀取更新伺服器設定，請重試。',
    saveFailed: '無法儲存更新伺服器，請重試。',
    invalidURL: '請輸入有效的 HTTP/HTTPS 目錄位址，不包含使用者名稱、密碼、查詢參數或片段。'
  },
  de: {
    title: 'Benutzerdefinierter Update-Server',
    description: 'Online-Updates von einem angegebenen Server suchen und herunterladen',
    save: 'Speichern',
    loadFailed: 'Die Servereinstellung konnte nicht geladen werden. Bitte erneut versuchen.',
    saveFailed: 'Der Update-Server konnte nicht gespeichert werden. Bitte erneut versuchen.',
    invalidURL:
      'Geben Sie eine HTTP/HTTPS-Verzeichnisadresse ohne Zugangsdaten, Abfrageparameter oder Fragment ein.'
  },
  fr: {
    title: 'Serveur de mise à jour personnalisé',
    description: 'Rechercher et télécharger les mises à jour en ligne depuis un serveur spécifié',
    save: 'Enregistrer',
    loadFailed: 'Impossible de charger le serveur configuré. Réessayez.',
    saveFailed: 'Impossible d’enregistrer le serveur de mise à jour. Réessayez.',
    invalidURL:
      'Saisissez une adresse HTTP/HTTPS de répertoire sans identifiants, paramètres de requête ni fragment.'
  },
  es: {
    title: 'Servidor de actualizaciones personalizado',
    description: 'Buscar y descargar actualizaciones en línea desde un servidor especificado',
    save: 'Guardar',
    loadFailed: 'No se pudo cargar el servidor configurado. Inténtelo de nuevo.',
    saveFailed: 'No se pudo guardar el servidor de actualizaciones. Inténtelo de nuevo.',
    invalidURL:
      'Introduzca una URL HTTP/HTTPS de directorio sin credenciales, parámetros de consulta ni fragmento.'
  },
  it: {
    title: 'Server di aggiornamento personalizzato',
    description: 'Cerca e scarica gli aggiornamenti online da un server specificato',
    save: 'Salva',
    loadFailed: 'Impossibile caricare il server configurato. Riprova.',
    saveFailed: 'Impossibile salvare il server degli aggiornamenti. Riprova.',
    invalidURL:
      'Inserisci un URL HTTP/HTTPS di cartella senza credenziali, parametri di ricerca o frammenti.'
  },
  nl: {
    title: 'Aangepaste updateserver',
    description: 'Online-updates zoeken en downloaden vanaf een opgegeven server',
    save: 'Opslaan',
    loadFailed: 'De ingestelde updateserver kon niet worden geladen. Probeer opnieuw.',
    saveFailed: 'De updateserver kon niet worden opgeslagen. Probeer opnieuw.',
    invalidURL: 'Voer een HTTP/HTTPS-mapadres in zonder inloggegevens, queryparameters of fragment.'
  },
  da: {
    title: 'Brugerdefineret opdateringsserver',
    description: 'Søg efter og download onlineopdateringer fra en angivet server',
    save: 'Gem',
    loadFailed: 'Serverindstillingen kunne ikke indlæses. Prøv igen.',
    saveFailed: 'Opdateringsserveren kunne ikke gemmes. Prøv igen.',
    invalidURL:
      'Indtast en HTTP/HTTPS-mappeadresse uden loginoplysninger, forespørgselsparametre eller fragment.'
  },
  nb: {
    title: 'Egendefinert oppdateringsserver',
    description: 'Se etter og last ned nettbaserte oppdateringer fra en angitt server',
    save: 'Lagre',
    loadFailed: 'Serverinnstillingen kunne ikke lastes inn. Prøv igjen.',
    saveFailed: 'Oppdateringsserveren kunne ikke lagres. Prøv igjen.',
    invalidURL:
      'Skriv inn en HTTP/HTTPS-mappeadresse uten påloggingsinformasjon, spørringsparametere eller fragment.'
  },
  pl: {
    title: 'Niestandardowy serwer aktualizacji',
    description: 'Sprawdzaj dostępność aktualizacji online i pobieraj je ze wskazanego serwera',
    save: 'Zapisz',
    loadFailed: 'Nie można wczytać ustawienia serwera. Spróbuj ponownie.',
    saveFailed: 'Nie można zapisać serwera aktualizacji. Spróbuj ponownie.',
    invalidURL:
      'Podaj adres katalogu HTTP/HTTPS bez danych logowania, parametrów zapytania ani fragmentu.'
  },
  cz: {
    title: 'Vlastní aktualizační server',
    description: 'Vyhledávejte a stahujte online aktualizace ze zadaného serveru',
    save: 'Uložit',
    loadFailed: 'Nastavení serveru nelze načíst. Zkuste to znovu.',
    saveFailed: 'Server aktualizací nelze uložit. Zkuste to znovu.',
    invalidURL:
      'Zadejte adresu adresáře HTTP/HTTPS bez přihlašovacích údajů, parametrů dotazu nebo fragmentu.'
  },
  hu: {
    title: 'Egyéni frissítési kiszolgáló',
    description: 'Online frissítések keresése és letöltése a megadott kiszolgálóról',
    save: 'Mentés',
    loadFailed: 'A kiszolgáló beállítása nem tölthető be. Próbálja újra.',
    saveFailed: 'A frissítési kiszolgáló nem menthető. Próbálja újra.',
    invalidURL:
      'Adjon meg egy HTTP/HTTPS könyvtárcímet hitelesítő adatok, lekérdezési paraméterek és töredék nélkül.'
  },
  ru: {
    title: 'Пользовательский сервер обновлений',
    description: 'Проверяйте наличие обновлений и загружайте их с указанного сервера',
    save: 'Сохранить',
    loadFailed: 'Не удалось загрузить настройку сервера. Повторите попытку.',
    saveFailed: 'Не удалось сохранить сервер обновлений. Повторите попытку.',
    invalidURL:
      'Введите HTTP/HTTPS-адрес каталога без учётных данных, параметров запроса и фрагмента.'
  },
  uk: {
    title: 'Користувацький сервер оновлень',
    description: 'Перевіряйте наявність оновлень і завантажуйте їх із зазначеного сервера',
    save: 'Зберегти',
    loadFailed: 'Не вдалося завантажити налаштування сервера. Спробуйте ще раз.',
    saveFailed: 'Не вдалося зберегти сервер оновлень. Спробуйте ще раз.',
    invalidURL:
      'Введіть HTTP/HTTPS-адресу каталогу без облікових даних, параметрів запиту та фрагмента.'
  },
  ja: {
    title: 'カスタム更新サーバー',
    description: '指定したサーバーでオンラインアップデートを確認し、ダウンロードします',
    save: '保存',
    loadFailed: 'サーバー設定を読み込めませんでした。再試行してください。',
    saveFailed: '更新サーバーを保存できませんでした。再試行してください。',
    invalidURL:
      '認証情報、クエリパラメータ、フラグメントを含まない HTTP/HTTPS のディレクトリ URL を入力してください。'
  },
  ko: {
    title: '사용자 지정 업데이트 서버',
    description: '지정한 서버에서 온라인 업데이트를 확인하고 다운로드합니다',
    save: '저장',
    loadFailed: '서버 설정을 불러오지 못했습니다. 다시 시도하세요.',
    saveFailed: '업데이트 서버를 저장하지 못했습니다. 다시 시도하세요.',
    invalidURL:
      '인증 정보, 쿼리 매개변수 또는 프래그먼트가 없는 HTTP/HTTPS 디렉터리 주소를 입력하세요.'
  },
  id: {
    title: 'Server Pembaruan Kustom',
    description: 'Periksa dan unduh pembaruan daring dari server yang ditentukan',
    save: 'Simpan',
    loadFailed: 'Tidak dapat memuat pengaturan server. Coba lagi.',
    saveFailed: 'Tidak dapat menyimpan server pembaruan. Coba lagi.',
    invalidURL: 'Masukkan URL direktori HTTP/HTTPS tanpa kredensial, parameter kueri, atau fragmen.'
  },
  vi: {
    title: 'Máy chủ cập nhật tùy chỉnh',
    description: 'Kiểm tra và tải xuống các bản cập nhật trực tuyến từ máy chủ được chỉ định',
    save: 'Lưu',
    loadFailed: 'Không thể tải cài đặt máy chủ. Hãy thử lại.',
    saveFailed: 'Không thể lưu máy chủ cập nhật. Hãy thử lại.',
    invalidURL:
      'Nhập URL thư mục HTTP/HTTPS không chứa thông tin đăng nhập, tham số truy vấn hoặc đoạn định danh.'
  },
  th: {
    title: 'เซิร์ฟเวอร์อัปเดตแบบกำหนดเอง',
    description: 'ตรวจสอบและดาวน์โหลดการอัปเดตออนไลน์จากเซิร์ฟเวอร์ที่ระบุ',
    save: 'บันทึก',
    loadFailed: 'โหลดการตั้งค่าเซิร์ฟเวอร์ไม่ได้ โปรดลองอีกครั้ง',
    saveFailed: 'บันทึกเซิร์ฟเวอร์อัปเดตไม่ได้ โปรดลองอีกครั้ง',
    invalidURL:
      'ป้อน URL ไดเรกทอรี HTTP/HTTPS ที่ไม่มีข้อมูลเข้าสู่ระบบ พารามิเตอร์คำค้น หรือส่วนย่อย'
  }
};

export default function updateMessages(language: string): UpdateMessages {
  const normalized = language as SupportedLanguage;
  return {
    ...(locales[normalized] ?? en),
    server: updateServerLocales[normalized] ?? updateServerLocales.en,
    region: regionLocales[normalized] ?? regionLocales.en,
    signatureVerification:
      signatureVerificationLocales[normalized] ?? signatureVerificationLocales.en
  };
}
