import type { RemoteNetworkComponentMessages } from '../../types.ts';

export const cloudflareMessages: RemoteNetworkComponentMessages = {
  en: {
    'cloudflare.modeHeading': 'Choose a tunnel mode',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Use a Tunnel Token and your domain for a stable address.',
    'cloudflare.quickModeDesc':
      'No account or Token required. A temporary address is generated on each start.',
    'cloudflare.continue': 'Continue',
    'cloudflare.quickConnectionFailed':
      'Could not connect. Check outbound network access. Retrying automatically.',
    'cloudflare.tokenStored': 'Saved',
    'cloudflare.clearToken': 'Clear',
    'cloudflare.clearTokenConfirm':
      'Stop the service, disable autostart and clear the saved Token? Other settings will be kept.',
    'cloudflare.statusNotInstalled': 'Not installed',
    'cloudflare.statusNotConfigured': 'Not configured',
    'cloudflare.notInstalled': 'Cloudflare Tunnel not found! Please install.',
    'cloudflare.install': 'Install',
    'cloudflare.installing': 'Installing',
    'cloudflare.uninstall': 'Uninstall Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Stop Cloudflare Tunnel and remove the client. Your configuration will be kept for reinstallation.',
    'cloudflare.tokenSaved': 'Saved; enter a new token to replace it',
    'cloudflare.save': 'Save and apply',
    'cloudflare.start': 'Enable',
    'cloudflare.stop': 'Disable',
    'cloudflare.restart': 'Restart',
    'cloudflare.disconnect': 'This interrupts sessions using Cloudflare Tunnel.',
    'cloudflare.confirm': 'Confirm',
    'cloudflare.cancel': 'Cancel',
    'cloudflare.accessUrl': 'Access address',
    'cloudflare.stopped': 'Disabled',
    'cloudflare.connecting': 'Connecting',
    'cloudflare.error': 'Connection error',
    'cloudflare.invalidConfig': 'The configuration could not be read or validated.',
    'cloudflare.installFailed':
      'Installation failed. Check the network and available storage, then retry.',
    'cloudflare.rollbackFailed':
      'The previous configuration could not be restored. Check the settings before enabling Cloudflare Tunnel.',
    'cloudflare.failed': 'The operation failed. Refresh and try again.',
    'cloudflare.connected': 'Tunnel connected',
    'cloudflare.invalidToken':
      'Paste the Tunnel Token, not an account API token or installation command.',
    'cloudflare.invalidURL':
      'Enter an HTTPS address without a path, credentials, query or fragment.',
    'cloudflare.connectionFailed':
      'Could not connect. Check the Tunnel Token and outbound network access. Retrying automatically.',
    'cloudflare.httpsRequired': 'Enable HTTPS on NanoKVM before starting this tunnel.'
  },
  zh: {
    'cloudflare.modeHeading': '选择隧道模式',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': '使用 Tunnel Token 和自己的域名，适合长期访问。',
    'cloudflare.quickModeDesc': '无需账户或 Token，每次启动自动生成临时访问地址。',
    'cloudflare.continue': '继续',
    'cloudflare.quickConnectionFailed': '连接失败，请检查出站网络。服务正在自动重试。',
    'cloudflare.tokenStored': '已保存',
    'cloudflare.clearToken': '清除',
    'cloudflare.clearTokenConfirm': '停用服务和自启，并清除已保存的 Token？其他配置会保留。',
    'cloudflare.statusNotInstalled': '未安装',
    'cloudflare.statusNotConfigured': '未配置',
    'cloudflare.notInstalled': '未检测到 Cloudflare Tunnel，请先安装',
    'cloudflare.install': '安装',
    'cloudflare.installing': '正在安装',
    'cloudflare.uninstall': '卸载 Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      '停止 Cloudflare Tunnel 并卸载客户端。配置会保留，重新安装后可继续使用。',
    'cloudflare.tokenSaved': '已保存；输入新 Token 可替换',
    'cloudflare.save': '保存并应用',
    'cloudflare.start': '启用',
    'cloudflare.stop': '停用',
    'cloudflare.restart': '重启',
    'cloudflare.disconnect': '这会中断通过 Cloudflare Tunnel 连接的会话。',
    'cloudflare.confirm': '确认',
    'cloudflare.cancel': '取消',
    'cloudflare.accessUrl': '访问地址',
    'cloudflare.stopped': '已停用',
    'cloudflare.connecting': '正在连接',
    'cloudflare.error': '连接异常',
    'cloudflare.invalidConfig': '无法读取或验证配置。',
    'cloudflare.installFailed': '安装失败，请检查网络和可用存储空间后重试。',
    'cloudflare.rollbackFailed': '无法恢复之前的配置，请检查设置后再启用 Cloudflare Tunnel。',
    'cloudflare.failed': '操作失败，请刷新后重试。',
    'cloudflare.connected': '隧道已连接',
    'cloudflare.invalidToken': '请粘贴 Tunnel Token，而非账户 API Token 或安装命令。',
    'cloudflare.invalidURL': '请输入 HTTPS 地址，不包含路径、账户信息、查询参数或片段。',
    'cloudflare.connectionFailed': '连接失败，请检查 Tunnel Token 和出站网络。服务正在自动重试。',
    'cloudflare.httpsRequired': '请先在 NanoKVM 上启用 HTTPS，再启动隧道。'
  },
  zh_tw: {
    'cloudflare.modeHeading': '選擇隧道模式',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': '使用 Tunnel Token 與自己的網域，適合長期存取。',
    'cloudflare.quickModeDesc': '無需帳號或 Token，每次啟動自動產生臨時存取位址。',
    'cloudflare.continue': '繼續',
    'cloudflare.quickConnectionFailed': '連線失敗，請檢查對外網路。服務正在自動重試。',
    'cloudflare.tokenStored': '已儲存',
    'cloudflare.clearToken': '清除',
    'cloudflare.clearTokenConfirm': '停用服務與自動啟動，並清除已儲存的 Token？其他設定會保留。',
    'cloudflare.statusNotInstalled': '未安裝',
    'cloudflare.statusNotConfigured': '未設定',
    'cloudflare.notInstalled': '未找到 Cloudflare Tunnel ！請先安裝。',
    'cloudflare.install': '安裝',
    'cloudflare.installing': '正在安裝',
    'cloudflare.uninstall': '解除安裝 Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      '停止 Cloudflare Tunnel 並移除用戶端。設定會保留，重新安裝後可繼續使用。',
    'cloudflare.tokenSaved': '已儲存；輸入新的 Token 可替換',
    'cloudflare.save': '儲存並套用',
    'cloudflare.start': '啟用',
    'cloudflare.stop': '停用',
    'cloudflare.restart': '重新啟動',
    'cloudflare.disconnect': '這會中斷透過 Cloudflare Tunnel 連線的工作階段。',
    'cloudflare.confirm': '確認',
    'cloudflare.cancel': '取消',
    'cloudflare.accessUrl': '存取位址',
    'cloudflare.stopped': '已停用',
    'cloudflare.connecting': '正在連線',
    'cloudflare.error': '連線異常',
    'cloudflare.invalidConfig': '無法讀取或驗證設定。',
    'cloudflare.installFailed': '安裝失敗，請檢查網路及可用儲存空間後重試。',
    'cloudflare.rollbackFailed': '無法還原先前的設定，請檢查設定後再啟用 Cloudflare Tunnel。',
    'cloudflare.failed': '操作失敗，請重新整理後重試。',
    'cloudflare.connected': '通道已連線',
    'cloudflare.invalidToken': '請貼上 Tunnel Token，而非帳戶 API Token 或安裝指令。',
    'cloudflare.invalidURL': '請輸入 HTTPS 位址，不包含路徑、帳戶資訊、查詢參數或片段。',
    'cloudflare.connectionFailed': '連線失敗，請檢查 Tunnel Token 與對外網路。服務正在自動重試。',
    'cloudflare.httpsRequired': '請先在 NanoKVM 上啟用 HTTPS，再啟動通道。'
  },
  ja: {
    'cloudflare.modeHeading': 'トンネル方式を選択',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc':
      'Tunnel Token と独自ドメインを使用し、固定アドレスでアクセスします。',
    'cloudflare.quickModeDesc':
      'アカウントや Token は不要です。起動のたびに一時アドレスを生成します。',
    'cloudflare.continue': '続行',
    'cloudflare.quickConnectionFailed':
      '接続できません。外部ネットワークへの接続を確認してください。自動的に再試行します。',
    'cloudflare.tokenStored': '保存済み',
    'cloudflare.clearToken': 'クリア',
    'cloudflare.clearTokenConfirm':
      'サービスと自動起動を停止し、保存済みトークンを削除しますか？他の設定は保持されます。',
    'cloudflare.statusNotInstalled': '未インストール',
    'cloudflare.statusNotConfigured': '未設定',
    'cloudflare.notInstalled': 'Cloudflare Tunnelが見つかりません！インストールしてください。',
    'cloudflare.install': 'インストール',
    'cloudflare.installing': 'インストール中',
    'cloudflare.uninstall': 'Cloudflare Tunnel をアンインストール',
    'cloudflare.uninstallDesc':
      'Cloudflare Tunnel を停止し、クライアントを削除します。設定は再インストール時のために保持されます。',
    'cloudflare.tokenSaved': '保存済み。新しいトークンを入力すると置き換えます',
    'cloudflare.save': '保存して適用',
    'cloudflare.start': '有効にする',
    'cloudflare.stop': '無効にする',
    'cloudflare.restart': '再起動',
    'cloudflare.disconnect': 'Cloudflare Tunnel 経由のセッションが切断されます。',
    'cloudflare.confirm': '確認',
    'cloudflare.cancel': 'キャンセル',
    'cloudflare.accessUrl': 'アクセス先',
    'cloudflare.stopped': '無効',
    'cloudflare.connecting': '接続中',
    'cloudflare.error': '接続エラー',
    'cloudflare.invalidConfig': '設定を読み込むか検証できませんでした。',
    'cloudflare.installFailed':
      'インストールに失敗しました。ネットワークと空き容量を確認して再試行してください。',
    'cloudflare.rollbackFailed':
      '以前の設定を復元できませんでした。設定を確認してから Cloudflare Tunnel を有効にしてください。',
    'cloudflare.failed': '操作に失敗しました。再読み込みして再試行してください。',
    'cloudflare.connected': 'トンネル接続済み',
    'cloudflare.invalidToken':
      'アカウントの API トークンやインストールコマンドではなく、Tunnel Token を貼り付けてください。',
    'cloudflare.invalidURL':
      'パス、認証情報、クエリ、フラグメントを含まない HTTPS アドレスを入力してください。',
    'cloudflare.connectionFailed':
      '接続できません。Tunnel Token と外向きの通信を確認してください。自動的に再試行します。',
    'cloudflare.httpsRequired': 'トンネルを起動する前に NanoKVM の HTTPS を有効にしてください。'
  },
  ko: {
    'cloudflare.modeHeading': '터널 모드 선택',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Tunnel Token과 자체 도메인으로 고정 주소를 사용합니다.',
    'cloudflare.quickModeDesc': '계정이나 Token 없이 시작할 때마다 임시 주소를 생성합니다.',
    'cloudflare.continue': '계속',
    'cloudflare.quickConnectionFailed':
      '연결하지 못했습니다. 외부 네트워크 연결을 확인하세요. 자동으로 다시 시도합니다.',
    'cloudflare.tokenStored': '저장됨',
    'cloudflare.clearToken': '지우기',
    'cloudflare.clearTokenConfirm':
      '서비스와 자동 시작을 중지하고 저장된 토큰을 삭제할까요? 다른 설정은 유지됩니다.',
    'cloudflare.statusNotInstalled': '설치되지 않음',
    'cloudflare.statusNotConfigured': '설정되지 않음',
    'cloudflare.notInstalled': 'Cloudflare Tunnel이 없습니다. 설치해주세요.',
    'cloudflare.install': '설치',
    'cloudflare.installing': '설치 중',
    'cloudflare.uninstall': 'Cloudflare Tunnel 제거',
    'cloudflare.uninstallDesc':
      'Cloudflare Tunnel를 중지하고 클라이언트를 제거합니다. 재설치를 위해 설정은 유지됩니다.',
    'cloudflare.tokenSaved': '저장됨. 새 토큰을 입력하면 교체됩니다',
    'cloudflare.save': '저장 및 적용',
    'cloudflare.start': '활성화',
    'cloudflare.stop': '비활성화',
    'cloudflare.restart': '다시 시작',
    'cloudflare.disconnect': 'Cloudflare Tunnel를 사용하는 세션이 끊어집니다.',
    'cloudflare.confirm': '확인',
    'cloudflare.cancel': '취소',
    'cloudflare.accessUrl': '접속 주소',
    'cloudflare.stopped': '비활성화됨',
    'cloudflare.connecting': '연결 중',
    'cloudflare.error': '연결 오류',
    'cloudflare.invalidConfig': '설정을 읽거나 검증할 수 없습니다.',
    'cloudflare.installFailed':
      '설치에 실패했습니다. 네트워크와 저장 공간을 확인하고 다시 시도하세요.',
    'cloudflare.rollbackFailed':
      '이전 설정을 복원할 수 없습니다. 설정을 확인한 후 Cloudflare Tunnel를 활성화하세요.',
    'cloudflare.failed': '작업에 실패했습니다. 새로고침 후 다시 시도하세요.',
    'cloudflare.connected': '터널 연결됨',
    'cloudflare.invalidToken': '계정 API 토큰이나 설치 명령이 아닌 Tunnel Token을 붙여넣으세요.',
    'cloudflare.invalidURL':
      '경로, 인증 정보, 쿼리 또는 프래그먼트가 없는 HTTPS 주소를 입력하세요.',
    'cloudflare.connectionFailed':
      '연결할 수 없습니다. Tunnel Token과 외부 네트워크 연결을 확인하세요. 자동으로 재시도합니다.',
    'cloudflare.httpsRequired': '터널을 시작하기 전에 NanoKVM에서 HTTPS를 켜세요.'
  },
  de: {
    'cloudflare.modeHeading': 'Tunnelmodus wählen',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Mit Tunnel Token und eigener Domain eine feste Adresse verwenden.',
    'cloudflare.quickModeDesc':
      'Kein Konto oder Token nötig. Bei jedem Start wird eine temporäre Adresse erzeugt.',
    'cloudflare.continue': 'Weiter',
    'cloudflare.quickConnectionFailed':
      'Verbindung fehlgeschlagen. Ausgehende Netzwerkverbindung prüfen. Automatischer Neuversuch.',
    'cloudflare.tokenStored': 'Gespeichert',
    'cloudflare.clearToken': 'Löschen',
    'cloudflare.clearTokenConfirm':
      'Dienst und Autostart deaktivieren und das gespeicherte Token löschen? Andere Einstellungen bleiben erhalten.',
    'cloudflare.statusNotInstalled': 'Nicht installiert',
    'cloudflare.statusNotConfigured': 'Nicht konfiguriert',
    'cloudflare.notInstalled': 'Cloudflare Tunnel nicht gefunden! Bitte installieren.',
    'cloudflare.install': 'Installieren',
    'cloudflare.installing': 'Wird installiert',
    'cloudflare.uninstall': 'Cloudflare Tunnel deinstallieren',
    'cloudflare.uninstallDesc':
      'Cloudflare Tunnel stoppen und den Client entfernen. Die Konfiguration bleibt für eine erneute Installation erhalten.',
    'cloudflare.tokenSaved': 'Gespeichert; zum Ersetzen neuen Token eingeben',
    'cloudflare.save': 'Speichern und anwenden',
    'cloudflare.start': 'Aktivieren',
    'cloudflare.stop': 'Deaktivieren',
    'cloudflare.restart': 'Neu starten',
    'cloudflare.disconnect': 'Dies unterbricht Sitzungen über Cloudflare Tunnel.',
    'cloudflare.confirm': 'Bestätigen',
    'cloudflare.cancel': 'Abbrechen',
    'cloudflare.accessUrl': 'Zugriffsadresse',
    'cloudflare.stopped': 'Deaktiviert',
    'cloudflare.connecting': 'Verbindung wird hergestellt',
    'cloudflare.error': 'Verbindungsfehler',
    'cloudflare.invalidConfig': 'Die Konfiguration konnte nicht gelesen oder geprüft werden.',
    'cloudflare.installFailed':
      'Installation fehlgeschlagen. Netzwerk und freien Speicher prüfen und erneut versuchen.',
    'cloudflare.rollbackFailed':
      'Vorherige Konfiguration konnte nicht wiederhergestellt werden. Einstellungen vor dem Aktivieren prüfen.',
    'cloudflare.failed': 'Vorgang fehlgeschlagen. Aktualisieren und erneut versuchen.',
    'cloudflare.connected': 'Tunnel verbunden',
    'cloudflare.invalidToken':
      'Fügen Sie das Tunnel Token ein, keinen Konto-API-Token oder Installationsbefehl.',
    'cloudflare.invalidURL':
      'Geben Sie eine HTTPS-Adresse ohne Pfad, Zugangsdaten, Abfrage oder Fragment ein.',
    'cloudflare.connectionFailed':
      'Verbindung fehlgeschlagen. Prüfen Sie Tunnel Token und ausgehenden Netzwerkzugriff. Automatische Wiederholung läuft.',
    'cloudflare.httpsRequired': 'Aktivieren Sie HTTPS auf NanoKVM, bevor Sie den Tunnel starten.'
  },
  fr: {
    'cloudflare.modeHeading': 'Choisir un mode de tunnel',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Utilisez un Tunnel Token et votre domaine pour une adresse fixe.',
    'cloudflare.quickModeDesc':
      'Aucun compte ni jeton requis. Une adresse temporaire est créée à chaque démarrage.',
    'cloudflare.continue': 'Continuer',
    'cloudflare.quickConnectionFailed':
      'Connexion impossible. Vérifiez l’accès au réseau externe. Nouvelle tentative automatique.',
    'cloudflare.tokenStored': 'Enregistré',
    'cloudflare.clearToken': 'Effacer',
    'cloudflare.clearTokenConfirm':
      'Arrêter le service, désactiver le démarrage automatique et effacer le jeton enregistré ? Les autres réglages seront conservés.',
    'cloudflare.statusNotInstalled': 'Non installé',
    'cloudflare.statusNotConfigured': 'Non configuré',
    'cloudflare.notInstalled': "Cloudflare Tunnel non trouvé ! Veuillez l'installer.",
    'cloudflare.install': 'Installer',
    'cloudflare.installing': 'Installation en cours',
    'cloudflare.uninstall': 'Désinstaller Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Arrêter Cloudflare Tunnel et supprimer le client. La configuration sera conservée pour une réinstallation.',
    'cloudflare.tokenSaved': 'Enregistré ; saisissez un nouveau jeton pour le remplacer',
    'cloudflare.save': 'Enregistrer et appliquer',
    'cloudflare.start': 'Activer',
    'cloudflare.stop': 'Désactiver',
    'cloudflare.restart': 'Redémarrer',
    'cloudflare.disconnect': 'Les sessions utilisant Cloudflare Tunnel seront interrompues.',
    'cloudflare.confirm': 'Confirmer',
    'cloudflare.cancel': 'Annuler',
    'cloudflare.accessUrl': 'Adresse d’accès',
    'cloudflare.stopped': 'Désactivé',
    'cloudflare.connecting': 'Connexion en cours',
    'cloudflare.error': 'Erreur de connexion',
    'cloudflare.invalidConfig': 'Impossible de lire ou de valider la configuration.',
    'cloudflare.installFailed':
      'Échec de l’installation. Vérifiez le réseau et l’espace disponible, puis réessayez.',
    'cloudflare.rollbackFailed':
      'Impossible de restaurer la configuration précédente. Vérifiez les paramètres avant d’activer Cloudflare Tunnel.',
    'cloudflare.failed': 'L’opération a échoué. Actualisez et réessayez.',
    'cloudflare.connected': 'Tunnel connecté',
    'cloudflare.invalidToken':
      'Collez le Tunnel Token, pas un jeton API de compte ni une commande d’installation.',
    'cloudflare.invalidURL':
      'Saisissez une adresse HTTPS sans chemin, identifiants, paramètres ni fragment.',
    'cloudflare.connectionFailed':
      'Connexion impossible. Vérifiez le Tunnel Token et l’accès réseau sortant. Nouvelle tentative automatique.',
    'cloudflare.httpsRequired': 'Activez HTTPS sur NanoKVM avant de démarrer le tunnel.'
  },
  es: {
    'cloudflare.modeHeading': 'Elegir modo de túnel',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Usa un Tunnel Token y tu dominio para tener una dirección fija.',
    'cloudflare.quickModeDesc':
      'No requiere cuenta ni token. Se genera una dirección temporal en cada inicio.',
    'cloudflare.continue': 'Continuar',
    'cloudflare.quickConnectionFailed':
      'No se pudo conectar. Comprueba la conexión de salida. Se reintentará automáticamente.',
    'cloudflare.tokenStored': 'Guardado',
    'cloudflare.clearToken': 'Borrar',
    'cloudflare.clearTokenConfirm':
      '¿Detener el servicio, desactivar el inicio automático y borrar el token guardado? Los demás ajustes se conservarán.',
    'cloudflare.statusNotInstalled': 'Sin instalar',
    'cloudflare.statusNotConfigured': 'Sin configurar',
    'cloudflare.notInstalled': '¡Cloudflare Tunnel no encontrado! Por favor, instálalo.',
    'cloudflare.install': 'Instalar',
    'cloudflare.installing': 'Instalando',
    'cloudflare.uninstall': 'Desinstalar Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Detener Cloudflare Tunnel y eliminar el cliente. Se conservará la configuración para reinstalarlo.',
    'cloudflare.tokenSaved': 'Guardado; introduce otro token para sustituirlo',
    'cloudflare.save': 'Guardar y aplicar',
    'cloudflare.start': 'Activar',
    'cloudflare.stop': 'Desactivar',
    'cloudflare.restart': 'Reiniciar',
    'cloudflare.disconnect': 'Esto interrumpe las sesiones que usan Cloudflare Tunnel.',
    'cloudflare.confirm': 'Confirmar',
    'cloudflare.cancel': 'Cancelar',
    'cloudflare.accessUrl': 'Dirección de acceso',
    'cloudflare.stopped': 'Desactivado',
    'cloudflare.connecting': 'Conectando',
    'cloudflare.error': 'Error de conexión',
    'cloudflare.invalidConfig': 'No se pudo leer o validar la configuración.',
    'cloudflare.installFailed':
      'Falló la instalación. Comprueba la red y el espacio disponible y vuelve a intentarlo.',
    'cloudflare.rollbackFailed':
      'No se pudo restaurar la configuración anterior. Revísala antes de activar Cloudflare Tunnel.',
    'cloudflare.failed': 'La operación falló. Actualiza y vuelve a intentarlo.',
    'cloudflare.connected': 'Túnel conectado',
    'cloudflare.invalidToken':
      'Pegue el Tunnel Token, no un token API de cuenta ni un comando de instalación.',
    'cloudflare.invalidURL':
      'Introduzca una dirección HTTPS sin ruta, credenciales, consulta ni fragmento.',
    'cloudflare.connectionFailed':
      'No se pudo conectar. Compruebe el Tunnel Token y el acceso de red saliente. Se reintentará automáticamente.',
    'cloudflare.httpsRequired': 'Active HTTPS en NanoKVM antes de iniciar el túnel.'
  },
  it: {
    'cloudflare.modeHeading': 'Scegli la modalità del tunnel',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc':
      'Usa un Tunnel Token e il tuo dominio per avere un indirizzo fisso.',
    'cloudflare.quickModeDesc':
      'Non servono account o token. A ogni avvio viene generato un indirizzo temporaneo.',
    'cloudflare.continue': 'Continua',
    'cloudflare.quickConnectionFailed':
      'Connessione non riuscita. Verifica l’accesso alla rete esterna. Nuovo tentativo automatico.',
    'cloudflare.tokenStored': 'Salvato',
    'cloudflare.clearToken': 'Cancella',
    'cloudflare.clearTokenConfirm':
      'Arrestare il servizio, disattivare l’avvio automatico e cancellare il token salvato? Le altre impostazioni verranno mantenute.',
    'cloudflare.statusNotInstalled': 'Non installato',
    'cloudflare.statusNotConfigured': 'Non configurato',
    'cloudflare.notInstalled': 'Cloudflare Tunnel non trovato! Per favore, installa.',
    'cloudflare.install': 'Installa',
    'cloudflare.installing': 'Installazione in corso',
    'cloudflare.uninstall': 'Disinstalla Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Arresta Cloudflare Tunnel e rimuovi il client. La configurazione viene conservata per una reinstallazione.',
    'cloudflare.tokenSaved': 'Salvato; inserisci un nuovo token per sostituirlo',
    'cloudflare.save': 'Salva e applica',
    'cloudflare.start': 'Abilita',
    'cloudflare.stop': 'Disabilita',
    'cloudflare.restart': 'Riavvia',
    'cloudflare.disconnect': 'Le sessioni che usano Cloudflare Tunnel verranno interrotte.',
    'cloudflare.confirm': 'Conferma',
    'cloudflare.cancel': 'Annulla',
    'cloudflare.accessUrl': 'Indirizzo di accesso',
    'cloudflare.stopped': 'Disabilitato',
    'cloudflare.connecting': 'Connessione in corso',
    'cloudflare.error': 'Errore di connessione',
    'cloudflare.invalidConfig': 'Impossibile leggere o convalidare la configurazione.',
    'cloudflare.installFailed':
      'Installazione non riuscita. Controlla rete e spazio disponibile, poi riprova.',
    'cloudflare.rollbackFailed':
      'Impossibile ripristinare la configurazione precedente. Controlla le impostazioni prima di abilitare Cloudflare Tunnel.',
    'cloudflare.failed': 'Operazione non riuscita. Aggiorna e riprova.',
    'cloudflare.connected': 'Tunnel connesso',
    'cloudflare.invalidToken':
      'Incollare il Tunnel Token, non un token API dell’account o un comando di installazione.',
    'cloudflare.invalidURL':
      'Inserire un indirizzo HTTPS senza percorso, credenziali, query o frammento.',
    'cloudflare.connectionFailed':
      'Connessione non riuscita. Controllare il Tunnel Token e l’accesso alla rete in uscita. Nuovo tentativo automatico.',
    'cloudflare.httpsRequired': 'Attivare HTTPS su NanoKVM prima di avviare il tunnel.'
  },
  nl: {
    'cloudflare.modeHeading': 'Kies een tunnelmodus',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Gebruik een Tunnel Token en je domein voor een vast adres.',
    'cloudflare.quickModeDesc':
      'Geen account of token nodig. Elke start genereert een tijdelijk adres.',
    'cloudflare.continue': 'Doorgaan',
    'cloudflare.quickConnectionFailed':
      'Verbinding mislukt. Controleer de uitgaande netwerkverbinding. Automatisch opnieuw proberen.',
    'cloudflare.tokenStored': 'Opgeslagen',
    'cloudflare.clearToken': 'Wissen',
    'cloudflare.clearTokenConfirm':
      'De service en automatisch starten uitschakelen en het opgeslagen token wissen? Andere instellingen blijven behouden.',
    'cloudflare.statusNotInstalled': 'Niet geïnstalleerd',
    'cloudflare.statusNotConfigured': 'Niet ingesteld',
    'cloudflare.notInstalled': 'Cloudflare Tunnel niet gevonden! Installeer a.u.b.',
    'cloudflare.install': 'Installeren',
    'cloudflare.installing': 'Installeren bezig',
    'cloudflare.uninstall': 'Cloudflare Tunnel verwijderen',
    'cloudflare.uninstallDesc':
      'Stop Cloudflare Tunnel en verwijder de client. De configuratie blijft bewaard voor herinstallatie.',
    'cloudflare.tokenSaved': 'Opgeslagen; voer een nieuw token in om het te vervangen',
    'cloudflare.save': 'Opslaan en toepassen',
    'cloudflare.start': 'Inschakelen',
    'cloudflare.stop': 'Uitschakelen',
    'cloudflare.restart': 'Herstarten',
    'cloudflare.disconnect': 'Dit onderbreekt sessies via Cloudflare Tunnel.',
    'cloudflare.confirm': 'Bevestigen',
    'cloudflare.cancel': 'Annuleren',
    'cloudflare.accessUrl': 'Toegangsadres',
    'cloudflare.stopped': 'Uitgeschakeld',
    'cloudflare.connecting': 'Verbinden',
    'cloudflare.error': 'Verbindingsfout',
    'cloudflare.invalidConfig': 'De configuratie kon niet worden gelezen of gevalideerd.',
    'cloudflare.installFailed':
      'Installatie mislukt. Controleer netwerk en vrije opslagruimte en probeer opnieuw.',
    'cloudflare.rollbackFailed':
      'De vorige configuratie kon niet worden hersteld. Controleer de instellingen voordat u Cloudflare Tunnel inschakelt.',
    'cloudflare.failed': 'De bewerking is mislukt. Vernieuw en probeer opnieuw.',
    'cloudflare.connected': 'Tunnel verbonden',
    'cloudflare.invalidToken':
      'Plak het Tunnel Token, geen account-API-token of installatieopdracht.',
    'cloudflare.invalidURL':
      'Voer een HTTPS-adres in zonder pad, inloggegevens, query of fragment.',
    'cloudflare.connectionFailed':
      'Verbinding mislukt. Controleer het Tunnel Token en uitgaand netwerkverkeer. Er wordt automatisch opnieuw geprobeerd.',
    'cloudflare.httpsRequired': 'Schakel HTTPS op NanoKVM in voordat u de tunnel start.'
  },
  da: {
    'cloudflare.modeHeading': 'Vælg tunneltilstand',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Brug et Tunnel Token og dit domæne til en fast adresse.',
    'cloudflare.quickModeDesc':
      'Ingen konto eller token kræves. En midlertidig adresse oprettes ved hver start.',
    'cloudflare.continue': 'Fortsæt',
    'cloudflare.quickConnectionFailed':
      'Kunne ikke oprette forbindelse. Kontrollér udgående netværksadgang. Prøver automatisk igen.',
    'cloudflare.tokenStored': 'Gemt',
    'cloudflare.clearToken': 'Ryd',
    'cloudflare.clearTokenConfirm':
      'Stop tjenesten, slå automatisk start fra, og ryd den gemte token? Andre indstillinger bevares.',
    'cloudflare.statusNotInstalled': 'Ikke installeret',
    'cloudflare.statusNotConfigured': 'Ikke konfigureret',
    'cloudflare.notInstalled':
      'Cloudflare Tunnel ikke fundet! Installer det for at fuldføre opsætningen.',
    'cloudflare.install': 'Installer',
    'cloudflare.installing': 'Installerer',
    'cloudflare.uninstall': 'Afinstaller Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Stop Cloudflare Tunnel og fjern klienten. Konfigurationen gemmes til geninstallation.',
    'cloudflare.tokenSaved': 'Gemt; indtast et nyt token for at erstatte det',
    'cloudflare.save': 'Gem og anvend',
    'cloudflare.start': 'Aktiver',
    'cloudflare.stop': 'Deaktiver',
    'cloudflare.restart': 'Genstart',
    'cloudflare.disconnect': 'Dette afbryder sessioner via Cloudflare Tunnel.',
    'cloudflare.confirm': 'Bekræft',
    'cloudflare.cancel': 'Annuller',
    'cloudflare.accessUrl': 'Adgangsadresse',
    'cloudflare.stopped': 'Deaktiveret',
    'cloudflare.connecting': 'Opretter forbindelse',
    'cloudflare.error': 'Forbindelsesfejl',
    'cloudflare.invalidConfig': 'Konfigurationen kunne ikke læses eller valideres.',
    'cloudflare.installFailed':
      'Installationen mislykkedes. Kontroller netværk og ledig plads, og prøv igen.',
    'cloudflare.rollbackFailed':
      'Den tidligere konfiguration kunne ikke gendannes. Kontroller indstillingerne før aktivering af Cloudflare Tunnel.',
    'cloudflare.failed': 'Handlingen mislykkedes. Opdater og prøv igen.',
    'cloudflare.connected': 'Tunnel forbundet',
    'cloudflare.invalidToken':
      'Indsæt Tunnel Token, ikke en API-token til kontoen eller en installationskommando.',
    'cloudflare.invalidURL':
      'Indtast en HTTPS-adresse uden sti, loginoplysninger, forespørgsel eller fragment.',
    'cloudflare.connectionFailed':
      'Forbindelsen mislykkedes. Kontrollér Tunnel Token og udgående netværksadgang. Der forsøges automatisk igen.',
    'cloudflare.httpsRequired': 'Aktivér HTTPS på NanoKVM, før tunnelen startes.'
  },
  nb: {
    'cloudflare.modeHeading': 'Velg tunnelmodus',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Bruk et Tunnel Token og ditt domene for en fast adresse.',
    'cloudflare.quickModeDesc':
      'Ingen konto eller token kreves. En midlertidig adresse opprettes ved hver start.',
    'cloudflare.continue': 'Fortsett',
    'cloudflare.quickConnectionFailed':
      'Kunne ikke koble til. Kontroller utgående nettverkstilgang. Prøver automatisk på nytt.',
    'cloudflare.tokenStored': 'Lagret',
    'cloudflare.clearToken': 'Slett',
    'cloudflare.clearTokenConfirm':
      'Stopp tjenesten, slå av automatisk oppstart og slett lagret token? Andre innstillinger beholdes.',
    'cloudflare.statusNotInstalled': 'Ikke installert',
    'cloudflare.statusNotConfigured': 'Ikke konfigurert',
    'cloudflare.notInstalled': 'Cloudflare Tunnel er ikke funnet! Vennligst installer.',
    'cloudflare.install': 'Installer',
    'cloudflare.installing': 'Installerer',
    'cloudflare.uninstall': 'Avinstaller Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Stopp Cloudflare Tunnel og fjern klienten. Konfigurasjonen beholdes for ny installasjon.',
    'cloudflare.tokenSaved': 'Lagret; skriv inn et nytt token for å erstatte det',
    'cloudflare.save': 'Lagre og bruk',
    'cloudflare.start': 'Aktiver',
    'cloudflare.stop': 'Deaktiver',
    'cloudflare.restart': 'Start på nytt',
    'cloudflare.disconnect': 'Dette avbryter økter som bruker Cloudflare Tunnel.',
    'cloudflare.confirm': 'Bekreft',
    'cloudflare.cancel': 'Avbryt',
    'cloudflare.accessUrl': 'Tilgangsadresse',
    'cloudflare.stopped': 'Deaktivert',
    'cloudflare.connecting': 'Kobler til',
    'cloudflare.error': 'Tilkoblingsfeil',
    'cloudflare.invalidConfig': 'Konfigurasjonen kunne ikke leses eller valideres.',
    'cloudflare.installFailed':
      'Installasjonen mislyktes. Kontroller nettverket og ledig lagringsplass, og prøv igjen.',
    'cloudflare.rollbackFailed':
      'Forrige konfigurasjon kunne ikke gjenopprettes. Kontroller innstillingene før du aktiverer Cloudflare Tunnel.',
    'cloudflare.failed': 'Handlingen mislyktes. Oppdater og prøv igjen.',
    'cloudflare.connected': 'Tunnel tilkoblet',
    'cloudflare.invalidToken':
      'Lim inn Tunnel Token, ikke en API-token for kontoen eller en installasjonskommando.',
    'cloudflare.invalidURL':
      'Skriv inn en HTTPS-adresse uten sti, påloggingsdata, spørring eller fragment.',
    'cloudflare.connectionFailed':
      'Kunne ikke koble til. Kontroller Tunnel Token og utgående nettverkstilgang. Prøver automatisk igjen.',
    'cloudflare.httpsRequired': 'Aktiver HTTPS på NanoKVM før du starter tunnelen.'
  },
  pl: {
    'cloudflare.modeHeading': 'Wybierz tryb tunelu',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Użyj Tunnel Token i własnej domeny, aby uzyskać stały adres.',
    'cloudflare.quickModeDesc':
      'Konto ani token nie są wymagane. Każde uruchomienie tworzy adres tymczasowy.',
    'cloudflare.continue': 'Dalej',
    'cloudflare.quickConnectionFailed':
      'Nie można połączyć. Sprawdź dostęp do sieci zewnętrznej. Próba zostanie ponowiona automatycznie.',
    'cloudflare.tokenStored': 'Zapisano',
    'cloudflare.clearToken': 'Wyczyść',
    'cloudflare.clearTokenConfirm':
      'Zatrzymać usługę, wyłączyć automatyczny start i usunąć zapisany token? Pozostałe ustawienia zostaną zachowane.',
    'cloudflare.statusNotInstalled': 'Niezainstalowane',
    'cloudflare.statusNotConfigured': 'Nieskonfigurowane',
    'cloudflare.notInstalled': 'Nie znaleziono Cloudflare Tunnel! Proszę zainstalować.',
    'cloudflare.install': 'Zainstaluj',
    'cloudflare.installing': 'Instalowanie',
    'cloudflare.uninstall': 'Odinstaluj Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Zatrzymaj Cloudflare Tunnel i usuń klienta. Konfiguracja zostanie zachowana do ponownej instalacji.',
    'cloudflare.tokenSaved': 'Zapisany; wprowadź nowy token, aby go zastąpić',
    'cloudflare.save': 'Zapisz i zastosuj',
    'cloudflare.start': 'Włącz',
    'cloudflare.stop': 'Wyłącz',
    'cloudflare.restart': 'Uruchom ponownie',
    'cloudflare.disconnect': 'Spowoduje to przerwanie sesji korzystających z Cloudflare Tunnel.',
    'cloudflare.confirm': 'Potwierdź',
    'cloudflare.cancel': 'Anuluj',
    'cloudflare.accessUrl': 'Adres dostępu',
    'cloudflare.stopped': 'Wyłączono',
    'cloudflare.connecting': 'Łączenie',
    'cloudflare.error': 'Błąd połączenia',
    'cloudflare.invalidConfig': 'Nie można odczytać ani zweryfikować konfiguracji.',
    'cloudflare.installFailed':
      'Instalacja nie powiodła się. Sprawdź sieć i wolne miejsce, a następnie spróbuj ponownie.',
    'cloudflare.rollbackFailed':
      'Nie można przywrócić poprzedniej konfiguracji. Sprawdź ustawienia przed włączeniem Cloudflare Tunnel.',
    'cloudflare.failed': 'Operacja nie powiodła się. Odśwież i spróbuj ponownie.',
    'cloudflare.connected': 'Tunel połączony',
    'cloudflare.invalidToken':
      'Wklej Tunnel Token, a nie token API konta ani polecenie instalacji.',
    'cloudflare.invalidURL':
      'Wpisz adres HTTPS bez ścieżki, danych logowania, zapytania i fragmentu.',
    'cloudflare.connectionFailed':
      'Nie można się połączyć. Sprawdź Tunnel Token i dostęp sieciowy na zewnątrz. Próba zostanie ponowiona automatycznie.',
    'cloudflare.httpsRequired': 'Włącz HTTPS na NanoKVM przed uruchomieniem tunelu.'
  },
  cz: {
    'cloudflare.modeHeading': 'Vyberte režim tunelu',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Použijte Tunnel Token a vlastní doménu pro stálou adresu.',
    'cloudflare.quickModeDesc':
      'Účet ani token nejsou potřeba. Při každém spuštění se vytvoří dočasná adresa.',
    'cloudflare.continue': 'Pokračovat',
    'cloudflare.quickConnectionFailed':
      'Připojení se nezdařilo. Zkontrolujte přístup k vnější síti. Pokus se automaticky opakuje.',
    'cloudflare.tokenStored': 'Uloženo',
    'cloudflare.clearToken': 'Vymazat',
    'cloudflare.clearTokenConfirm':
      'Zastavit službu, vypnout automatické spouštění a smazat uložený token? Ostatní nastavení zůstanou zachována.',
    'cloudflare.statusNotInstalled': 'Nenainstalováno',
    'cloudflare.statusNotConfigured': 'Nenastaveno',
    'cloudflare.notInstalled': 'Cloudflare Tunnel nebyl nalezen! Prosím nainstalujte.',
    'cloudflare.install': 'Nainstalovat',
    'cloudflare.installing': 'Instalace probíhá',
    'cloudflare.uninstall': 'Odinstalovat Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Zastavit Cloudflare Tunnel a odebrat klienta. Konfigurace zůstane zachována pro další instalaci.',
    'cloudflare.tokenSaved': 'Uloženo; nový token nahradí stávající',
    'cloudflare.save': 'Uložit a použít',
    'cloudflare.start': 'Povolit',
    'cloudflare.stop': 'Zakázat',
    'cloudflare.restart': 'Restartovat',
    'cloudflare.disconnect': 'Tím se přeruší relace využívající Cloudflare Tunnel.',
    'cloudflare.confirm': 'Potvrdit',
    'cloudflare.cancel': 'Zrušit',
    'cloudflare.accessUrl': 'Přístupová adresa',
    'cloudflare.stopped': 'Zakázáno',
    'cloudflare.connecting': 'Připojování',
    'cloudflare.error': 'Chyba připojení',
    'cloudflare.invalidConfig': 'Konfiguraci nelze načíst nebo ověřit.',
    'cloudflare.installFailed':
      'Instalace se nezdařila. Zkontrolujte síť a volné místo a zkuste to znovu.',
    'cloudflare.rollbackFailed':
      'Předchozí konfiguraci nelze obnovit. Před povolením Cloudflare Tunnel zkontrolujte nastavení.',
    'cloudflare.failed': 'Operace se nezdařila. Obnovte stránku a zkuste to znovu.',
    'cloudflare.connected': 'Tunel připojen',
    'cloudflare.invalidToken': 'Vložte Tunnel Token, nikoli token API účtu nebo instalační příkaz.',
    'cloudflare.invalidURL':
      'Zadejte adresu HTTPS bez cesty, přihlašovacích údajů, dotazu a fragmentu.',
    'cloudflare.connectionFailed':
      'Připojení se nezdařilo. Zkontrolujte Tunnel Token a odchozí síťové připojení. Opakování proběhne automaticky.',
    'cloudflare.httpsRequired': 'Před spuštěním tunelu zapněte HTTPS na NanoKVM.'
  },
  hu: {
    'cloudflare.modeHeading': 'Alagútmód kiválasztása',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Használjon Tunnel Tokent és saját domaint az állandó címhez.',
    'cloudflare.quickModeDesc':
      'Nem kell fiók vagy token. Minden indításkor új ideiglenes cím jön létre.',
    'cloudflare.continue': 'Folytatás',
    'cloudflare.quickConnectionFailed':
      'Nem sikerült csatlakozni. Ellenőrizze a kimenő hálózati kapcsolatot. Automatikus újrapróbálkozás.',
    'cloudflare.tokenStored': 'Mentve',
    'cloudflare.clearToken': 'Törlés',
    'cloudflare.clearTokenConfirm':
      'Leállítja a szolgáltatást, kikapcsolja az automatikus indítást és törli a mentett tokent? A többi beállítás megmarad.',
    'cloudflare.statusNotInstalled': 'Nincs telepítve',
    'cloudflare.statusNotConfigured': 'Nincs beállítva',
    'cloudflare.notInstalled': 'Cloudflare Tunnel nem található! Kérem, telepítse.',
    'cloudflare.install': 'Telepítés',
    'cloudflare.installing': 'Telepítés folyamatban',
    'cloudflare.uninstall': 'Cloudflare Tunnel eltávolítása',
    'cloudflare.uninstallDesc':
      'Az Cloudflare Tunnel leállítása és a kliens eltávolítása. A beállítások megmaradnak az újratelepítéshez.',
    'cloudflare.tokenSaved': 'Mentve; új token megadásával lecserélhető',
    'cloudflare.save': 'Mentés és alkalmazás',
    'cloudflare.start': 'Engedélyezés',
    'cloudflare.stop': 'Letiltás',
    'cloudflare.restart': 'Újraindítás',
    'cloudflare.disconnect': 'Ez megszakítja az Cloudflare Tunnel-t használó munkameneteket.',
    'cloudflare.confirm': 'Megerősítés',
    'cloudflare.cancel': 'Mégse',
    'cloudflare.accessUrl': 'Elérési cím',
    'cloudflare.stopped': 'Letiltva',
    'cloudflare.connecting': 'Csatlakozás',
    'cloudflare.error': 'Kapcsolati hiba',
    'cloudflare.invalidConfig': 'A beállítások nem olvashatók vagy nem ellenőrizhetők.',
    'cloudflare.installFailed':
      'A telepítés sikertelen. Ellenőrizze a hálózatot és a szabad tárhelyet, majd próbálja újra.',
    'cloudflare.rollbackFailed':
      'A korábbi beállítások nem állíthatók vissza. Ellenőrizze őket az Cloudflare Tunnel engedélyezése előtt.',
    'cloudflare.failed': 'A művelet sikertelen. Frissítsen, és próbálja újra.',
    'cloudflare.connected': 'Alagút csatlakoztatva',
    'cloudflare.invalidToken':
      'A Tunnel Token értékét illessze be, ne a fiók API-tokenjét vagy telepítési parancsát.',
    'cloudflare.invalidURL':
      'Útvonal, hitelesítő adatok, lekérdezés és töredék nélküli HTTPS-címet adjon meg.',
    'cloudflare.connectionFailed':
      'Sikertelen csatlakozás. Ellenőrizze a Tunnel Token értékét és a kimenő hálózati hozzáférést. Automatikus újrapróbálkozás.',
    'cloudflare.httpsRequired': 'Az alagút indítása előtt engedélyezze a HTTPS-t a NanoKVM-en.'
  },
  ru: {
    'cloudflare.modeHeading': 'Выберите режим туннеля',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Используйте Tunnel Token и свой домен для постоянного адреса.',
    'cloudflare.quickModeDesc':
      'Аккаунт и токен не нужны. При каждом запуске создаётся временный адрес.',
    'cloudflare.continue': 'Продолжить',
    'cloudflare.quickConnectionFailed':
      'Не удалось подключиться. Проверьте доступ к внешней сети. Повторная попытка выполняется автоматически.',
    'cloudflare.tokenStored': 'Сохранён',
    'cloudflare.clearToken': 'Очистить',
    'cloudflare.clearTokenConfirm':
      'Остановить службу, отключить автозапуск и удалить сохранённый токен? Остальные настройки сохранятся.',
    'cloudflare.statusNotInstalled': 'Не установлено',
    'cloudflare.statusNotConfigured': 'Не настроено',
    'cloudflare.notInstalled': 'Cloudflare Tunnel не найден! Пожалуйста, установите.',
    'cloudflare.install': 'Установить',
    'cloudflare.installing': 'Установка',
    'cloudflare.uninstall': 'Удалить Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Остановить Cloudflare Tunnel и удалить клиент. Настройки сохранятся для повторной установки.',
    'cloudflare.tokenSaved': 'Сохранён; введите новый токен для замены',
    'cloudflare.save': 'Сохранить и применить',
    'cloudflare.start': 'Включить',
    'cloudflare.stop': 'Отключить',
    'cloudflare.restart': 'Перезапустить',
    'cloudflare.disconnect': 'Это прервёт сеансы, использующие Cloudflare Tunnel.',
    'cloudflare.confirm': 'Подтвердить',
    'cloudflare.cancel': 'Отмена',
    'cloudflare.accessUrl': 'Адрес доступа',
    'cloudflare.stopped': 'Отключено',
    'cloudflare.connecting': 'Подключение',
    'cloudflare.error': 'Ошибка подключения',
    'cloudflare.invalidConfig': 'Не удалось прочитать или проверить настройки.',
    'cloudflare.installFailed':
      'Ошибка установки. Проверьте сеть и свободное место, затем повторите попытку.',
    'cloudflare.rollbackFailed':
      'Не удалось восстановить предыдущие настройки. Проверьте их перед включением Cloudflare Tunnel.',
    'cloudflare.failed': 'Операция не выполнена. Обновите страницу и повторите попытку.',
    'cloudflare.connected': 'Туннель подключён',
    'cloudflare.invalidToken':
      'Вставьте Tunnel Token, а не API-токен учётной записи или команду установки.',
    'cloudflare.invalidURL':
      'Введите адрес HTTPS без пути, учётных данных, параметров запроса и фрагмента.',
    'cloudflare.connectionFailed':
      'Не удалось подключиться. Проверьте Tunnel Token и исходящее сетевое соединение. Повторная попытка выполняется автоматически.',
    'cloudflare.httpsRequired': 'Включите HTTPS на NanoKVM перед запуском туннеля.'
  },
  uk: {
    'cloudflare.modeHeading': 'Виберіть режим тунелю',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Використовуйте Tunnel Token і власний домен для постійної адреси.',
    'cloudflare.quickModeDesc':
      'Обліковий запис і токен не потрібні. Кожен запуск створює тимчасову адресу.',
    'cloudflare.continue': 'Продовжити',
    'cloudflare.quickConnectionFailed':
      'Не вдалося підключитися. Перевірте доступ до зовнішньої мережі. Спробу буде повторено автоматично.',
    'cloudflare.tokenStored': 'Збережено',
    'cloudflare.clearToken': 'Очистити',
    'cloudflare.clearTokenConfirm':
      'Зупинити службу, вимкнути автозапуск і видалити збережений токен? Інші налаштування буде збережено.',
    'cloudflare.statusNotInstalled': 'Не встановлено',
    'cloudflare.statusNotConfigured': 'Не налаштовано',
    'cloudflare.notInstalled': 'Cloudflare Tunnel не знайдено! Будь ласка, встановіть його.',
    'cloudflare.install': 'Встановити',
    'cloudflare.installing': 'Встановлення',
    'cloudflare.uninstall': 'Видалити Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Зупинити Cloudflare Tunnel і видалити клієнт. Налаштування збережуться для повторного встановлення.',
    'cloudflare.tokenSaved': 'Збережено; введіть новий токен для заміни',
    'cloudflare.save': 'Зберегти й застосувати',
    'cloudflare.start': 'Увімкнути',
    'cloudflare.stop': 'Вимкнути',
    'cloudflare.restart': 'Перезапустити',
    'cloudflare.disconnect': 'Це перерве сеанси через Cloudflare Tunnel.',
    'cloudflare.confirm': 'Підтвердити',
    'cloudflare.cancel': 'Скасувати',
    'cloudflare.accessUrl': 'Адреса доступу',
    'cloudflare.stopped': 'Вимкнено',
    'cloudflare.connecting': 'Підключення',
    'cloudflare.error': 'Помилка підключення',
    'cloudflare.invalidConfig': 'Не вдалося прочитати або перевірити налаштування.',
    'cloudflare.installFailed':
      'Не вдалося встановити. Перевірте мережу й вільне місце та повторіть спробу.',
    'cloudflare.rollbackFailed':
      'Не вдалося відновити попередні налаштування. Перевірте їх перед увімкненням Cloudflare Tunnel.',
    'cloudflare.failed': 'Операція не вдалася. Оновіть сторінку й повторіть спробу.',
    'cloudflare.connected': 'Тунель підключено',
    'cloudflare.invalidToken':
      'Вставте Tunnel Token, а не API-токен облікового запису чи команду встановлення.',
    'cloudflare.invalidURL':
      'Введіть адресу HTTPS без шляху, облікових даних, параметрів запиту та фрагмента.',
    'cloudflare.connectionFailed':
      'Не вдалося підключитися. Перевірте Tunnel Token і вихідне мережеве з’єднання. Повторна спроба виконується автоматично.',
    'cloudflare.httpsRequired': 'Увімкніть HTTPS на NanoKVM перед запуском тунелю.'
  },
  id: {
    'cloudflare.modeHeading': 'Pilih mode tunnel',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Gunakan Tunnel Token dan domain Anda untuk alamat tetap.',
    'cloudflare.quickModeDesc':
      'Tidak perlu akun atau token. Alamat sementara dibuat setiap kali dimulai.',
    'cloudflare.continue': 'Lanjutkan',
    'cloudflare.quickConnectionFailed':
      'Tidak dapat terhubung. Periksa akses jaringan keluar. Mencoba lagi secara otomatis.',
    'cloudflare.tokenStored': 'Tersimpan',
    'cloudflare.clearToken': 'Hapus',
    'cloudflare.clearTokenConfirm':
      'Hentikan layanan, nonaktifkan mulai otomatis, dan hapus token tersimpan? Pengaturan lainnya tetap disimpan.',
    'cloudflare.statusNotInstalled': 'Belum terpasang',
    'cloudflare.statusNotConfigured': 'Belum dikonfigurasi',
    'cloudflare.notInstalled': 'Cloudflare Tunnel tidak ditemukan! Silahkan pasang.',
    'cloudflare.install': 'Pasang',
    'cloudflare.installing': 'Memasang',
    'cloudflare.uninstall': 'Hapus Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Hentikan Cloudflare Tunnel dan hapus klien. Konfigurasi disimpan untuk pemasangan ulang.',
    'cloudflare.tokenSaved': 'Tersimpan; masukkan token baru untuk menggantinya',
    'cloudflare.save': 'Simpan dan terapkan',
    'cloudflare.start': 'Aktifkan',
    'cloudflare.stop': 'Nonaktifkan',
    'cloudflare.restart': 'Mulai ulang',
    'cloudflare.disconnect': 'Ini akan memutus sesi yang menggunakan Cloudflare Tunnel.',
    'cloudflare.confirm': 'Konfirmasi',
    'cloudflare.cancel': 'Batal',
    'cloudflare.accessUrl': 'Alamat akses',
    'cloudflare.stopped': 'Dinonaktifkan',
    'cloudflare.connecting': 'Menghubungkan',
    'cloudflare.error': 'Kesalahan koneksi',
    'cloudflare.invalidConfig': 'Konfigurasi tidak dapat dibaca atau divalidasi.',
    'cloudflare.installFailed':
      'Pemasangan gagal. Periksa jaringan dan ruang penyimpanan, lalu coba lagi.',
    'cloudflare.rollbackFailed':
      'Konfigurasi sebelumnya tidak dapat dipulihkan. Periksa pengaturan sebelum mengaktifkan Cloudflare Tunnel.',
    'cloudflare.failed': 'Operasi gagal. Muat ulang dan coba lagi.',
    'cloudflare.connected': 'Tunnel terhubung',
    'cloudflare.invalidToken':
      'Tempel Tunnel Token, bukan token API akun atau perintah pemasangan.',
    'cloudflare.invalidURL': 'Masukkan alamat HTTPS tanpa jalur, kredensial, kueri, atau fragmen.',
    'cloudflare.connectionFailed':
      'Tidak dapat terhubung. Periksa Tunnel Token dan akses jaringan keluar. Mencoba lagi secara otomatis.',
    'cloudflare.httpsRequired': 'Aktifkan HTTPS di NanoKVM sebelum memulai tunnel.'
  },
  vi: {
    'cloudflare.modeHeading': 'Chọn chế độ đường hầm',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'Dùng Tunnel Token và tên miền của bạn để có địa chỉ cố định.',
    'cloudflare.quickModeDesc':
      'Không cần tài khoản hay token. Mỗi lần khởi động sẽ tạo một địa chỉ tạm thời.',
    'cloudflare.continue': 'Tiếp tục',
    'cloudflare.quickConnectionFailed':
      'Không thể kết nối. Hãy kiểm tra mạng ra ngoài. Hệ thống đang tự động thử lại.',
    'cloudflare.tokenStored': 'Đã lưu',
    'cloudflare.clearToken': 'Xóa',
    'cloudflare.clearTokenConfirm':
      'Dừng dịch vụ, tắt tự khởi động và xóa token đã lưu? Các cài đặt khác sẽ được giữ lại.',
    'cloudflare.statusNotInstalled': 'Chưa cài đặt',
    'cloudflare.statusNotConfigured': 'Chưa cấu hình',
    'cloudflare.notInstalled': 'Không tìm thấy Cloudflare Tunnel! Vui lòng cài đặt.',
    'cloudflare.install': 'Cài đặt',
    'cloudflare.installing': 'Đang cài đặt',
    'cloudflare.uninstall': 'Gỡ cài đặt Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'Dừng Cloudflare Tunnel và gỡ trình khách. Cấu hình sẽ được giữ để cài đặt lại.',
    'cloudflare.tokenSaved': 'Đã lưu; nhập token mới để thay thế',
    'cloudflare.save': 'Lưu và áp dụng',
    'cloudflare.start': 'Bật',
    'cloudflare.stop': 'Tắt',
    'cloudflare.restart': 'Khởi động lại',
    'cloudflare.disconnect': 'Thao tác này sẽ ngắt các phiên dùng Cloudflare Tunnel.',
    'cloudflare.confirm': 'Xác nhận',
    'cloudflare.cancel': 'Hủy',
    'cloudflare.accessUrl': 'Địa chỉ truy cập',
    'cloudflare.stopped': 'Đã tắt',
    'cloudflare.connecting': 'Đang kết nối',
    'cloudflare.error': 'Lỗi kết nối',
    'cloudflare.invalidConfig': 'Không thể đọc hoặc xác thực cấu hình.',
    'cloudflare.installFailed': 'Cài đặt thất bại. Kiểm tra mạng và dung lượng trống rồi thử lại.',
    'cloudflare.rollbackFailed':
      'Không thể khôi phục cấu hình trước. Kiểm tra cài đặt trước khi bật Cloudflare Tunnel.',
    'cloudflare.failed': 'Thao tác thất bại. Tải lại và thử lại.',
    'cloudflare.connected': 'Đường hầm đã kết nối',
    'cloudflare.invalidToken': 'Dán Tunnel Token, không phải mã API tài khoản hay lệnh cài đặt.',
    'cloudflare.invalidURL':
      'Nhập địa chỉ HTTPS không chứa đường dẫn, thông tin đăng nhập, truy vấn hay đoạn neo.',
    'cloudflare.connectionFailed':
      'Không thể kết nối. Kiểm tra Tunnel Token và kết nối mạng ra ngoài. Hệ thống sẽ tự động thử lại.',
    'cloudflare.httpsRequired': 'Bật HTTPS trên NanoKVM trước khi khởi động đường hầm.'
  },
  th: {
    'cloudflare.modeHeading': 'เลือกโหมดอุโมงค์',
    'cloudflare.namedMode': 'Named Tunnel',
    'cloudflare.namedModeDesc': 'ใช้ Tunnel Token และโดเมนของคุณเพื่อรับที่อยู่ถาวร',
    'cloudflare.quickModeDesc':
      'ไม่ต้องใช้บัญชีหรือ Token ระบบจะสร้างที่อยู่ชั่วคราวใหม่ทุกครั้งที่เริ่ม',
    'cloudflare.continue': 'ดำเนินการต่อ',
    'cloudflare.quickConnectionFailed':
      'เชื่อมต่อไม่สำเร็จ โปรดตรวจสอบเครือข่ายขาออก ระบบกำลังลองใหม่อัตโนมัติ',
    'cloudflare.tokenStored': 'บันทึกแล้ว',
    'cloudflare.clearToken': 'ล้าง',
    'cloudflare.clearTokenConfirm':
      'หยุดบริการ ปิดการเริ่มอัตโนมัติ และลบโทเค็นที่บันทึกไว้หรือไม่ การตั้งค่าอื่นจะยังคงอยู่',
    'cloudflare.statusNotInstalled': 'ยังไม่ได้ติดตั้ง',
    'cloudflare.statusNotConfigured': 'ยังไม่ได้ตั้งค่า',
    'cloudflare.notInstalled': 'ไม่พบ Cloudflare Tunnel! กรุณาติดตั้ง',
    'cloudflare.install': 'ติดตั้ง',
    'cloudflare.installing': 'กำลังติดตั้ง',
    'cloudflare.uninstall': 'ถอนการติดตั้ง Cloudflare Tunnel',
    'cloudflare.uninstallDesc':
      'หยุด Cloudflare Tunnel และลบไคลเอนต์ การตั้งค่าจะถูกเก็บไว้สำหรับการติดตั้งใหม่',
    'cloudflare.tokenSaved': 'บันทึกแล้ว ป้อนโทเค็นใหม่เพื่อแทนที่',
    'cloudflare.save': 'บันทึกและใช้งาน',
    'cloudflare.start': 'เปิดใช้งาน',
    'cloudflare.stop': 'ปิดใช้งาน',
    'cloudflare.restart': 'เริ่มใหม่',
    'cloudflare.disconnect': 'การดำเนินการนี้จะตัดการเชื่อมต่อเซสชันที่ใช้ Cloudflare Tunnel',
    'cloudflare.confirm': 'ยืนยัน',
    'cloudflare.cancel': 'ยกเลิก',
    'cloudflare.accessUrl': 'ที่อยู่สำหรับเข้าถึง',
    'cloudflare.stopped': 'ปิดใช้งานแล้ว',
    'cloudflare.connecting': 'กำลังเชื่อมต่อ',
    'cloudflare.error': 'การเชื่อมต่อผิดพลาด',
    'cloudflare.invalidConfig': 'ไม่สามารถอ่านหรือตรวจสอบการตั้งค่าได้',
    'cloudflare.installFailed': 'ติดตั้งไม่สำเร็จ ตรวจสอบเครือข่ายและพื้นที่ว่างแล้วลองอีกครั้ง',
    'cloudflare.rollbackFailed':
      'ไม่สามารถคืนค่าการตั้งค่าก่อนหน้าได้ โปรดตรวจสอบการตั้งค่าก่อนเปิดใช้งาน Cloudflare Tunnel',
    'cloudflare.failed': 'ดำเนินการไม่สำเร็จ รีเฟรชแล้วลองอีกครั้ง',
    'cloudflare.connected': 'เชื่อมต่อทันเนลแล้ว',
    'cloudflare.invalidToken': 'วาง Tunnel Token ไม่ใช่โทเค็น API ของบัญชีหรือคำสั่งติดตั้ง',
    'cloudflare.invalidURL':
      'ป้อนที่อยู่ HTTPS ที่ไม่มีพาธ ข้อมูลเข้าสู่ระบบ คิวรี หรือส่วนอ้างอิง',
    'cloudflare.connectionFailed':
      'เชื่อมต่อไม่ได้ โปรดตรวจสอบ Tunnel Token และการเข้าถึงเครือข่ายขาออก ระบบจะลองใหม่โดยอัตโนมัติ',
    'cloudflare.httpsRequired': 'เปิด HTTPS บน NanoKVM ก่อนเริ่มทันเนล'
  }
};
