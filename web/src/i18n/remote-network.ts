import type { SupportedLanguage } from './locale.ts';

/**
 * Strings owned by the remote network section itself (the component hub).
 * Individual components (Tailscale, FRP, ...) keep their own keys in
 * their own folder, so a new component only adds a registry entry.
 */
export type RemoteNetworkMessages = {
  title: string;
  status: {
    off: string;
    busy: string;
    on: string;
    bad: string;
  };
  mode: {
    heading: string;
    enable: string;
    experimental: string;
  };
};

const messages: Record<SupportedLanguage, RemoteNetworkMessages> = {
  nl: {
    title: 'Netwerk op afstand',
    status: {
      off: 'Niet ingeschakeld',
      busy: 'Wordt ingeschakeld…',
      on: 'Ingeschakeld',
      bad: 'Fout'
    },
    mode: {
      heading: 'Kies een verbindingsnetwerk',
      enable: 'Netwerk op afstand inschakelen',
      experimental: 'Experimenteel'
    }
  },
  da: {
    title: 'Fjernnetværk',
    status: {
      off: 'Ikke aktiveret',
      busy: 'Aktiverer…',
      on: 'Aktiveret',
      bad: 'Fejl'
    },
    mode: {
      heading: 'Vælg et forbindelsesnetværk',
      enable: 'Aktivér fjernnetværk',
      experimental: 'Eksperimentel'
    }
  },
  de: {
    title: 'Remote-Netzwerk',
    status: {
      off: 'Nicht aktiviert',
      busy: 'Wird aktiviert…',
      on: 'Aktiviert',
      bad: 'Fehler'
    },
    mode: {
      heading: 'Verbindungsnetzwerk wählen',
      enable: 'Remote-Netzwerk aktivieren',
      experimental: 'Experimentell'
    }
  },
  en: {
    title: 'Remote Network',
    status: {
      off: 'Not enabled',
      busy: 'Enabling…',
      on: 'Enabled',
      bad: 'Error'
    },
    mode: {
      heading: 'Choose a connection network',
      enable: 'Enable remote network',
      experimental: 'Experimental'
    }
  },
  es: {
    title: 'Red remota',
    status: {
      off: 'Sin activar',
      busy: 'Activando…',
      on: 'Activado',
      bad: 'Error'
    },
    mode: {
      heading: 'Elige una red de conexión',
      enable: 'Activar red remota',
      experimental: 'Experimental'
    }
  },
  fr: {
    title: 'Réseau distant',
    status: {
      off: 'Non activé',
      busy: 'Activation…',
      on: 'Activé',
      bad: 'Erreur'
    },
    mode: {
      heading: 'Choisir un réseau de connexion',
      enable: 'Activer le réseau distant',
      experimental: 'Expérimental'
    }
  },
  id: {
    title: 'Jaringan Jarak Jauh',
    status: {
      off: 'Belum diaktifkan',
      busy: 'Mengaktifkan…',
      on: 'Diaktifkan',
      bad: 'Galat'
    },
    mode: {
      heading: 'Pilih jaringan koneksi',
      enable: 'Aktifkan jaringan jarak jauh',
      experimental: 'Eksperimental'
    }
  },
  it: {
    title: 'Rete remota',
    status: {
      off: 'Non attivato',
      busy: 'Attivazione…',
      on: 'Attivato',
      bad: 'Errore'
    },
    mode: {
      heading: 'Scegli una rete di connessione',
      enable: 'Attiva rete remota',
      experimental: 'Sperimentale'
    }
  },
  pl: {
    title: 'Sieć zdalna',
    status: {
      off: 'Nie włączono',
      busy: 'Włączanie…',
      on: 'Włączone',
      bad: 'Błąd'
    },
    mode: {
      heading: 'Wybierz sieć połączenia',
      enable: 'Włącz zdalną sieć',
      experimental: 'Eksperymentalne'
    }
  },
  ru: {
    title: 'Удалённая сеть',
    status: {
      off: 'Не включено',
      busy: 'Включение…',
      on: 'Включено',
      bad: 'Ошибка'
    },
    mode: {
      heading: 'Выберите сеть подключения',
      enable: 'Включить удалённую сеть',
      experimental: 'Экспериментально'
    }
  },
  ko: {
    title: '원격 네트워크',
    status: {
      off: '사용 안 함',
      busy: '활성화 중…',
      on: '활성화됨',
      bad: '오류'
    },
    mode: {
      heading: '연결 네트워크 선택',
      enable: '원격 네트워크 활성화',
      experimental: '실험적'
    }
  },
  zh: {
    title: '远程网络',
    status: {
      off: '未启用',
      busy: '启用中…',
      on: '已启用',
      bad: '异常'
    },
    mode: {
      heading: '选择连接网络',
      enable: '启动',
      experimental: '实验性'
    }
  },
  zh_tw: {
    title: '遠端網路',
    status: {
      off: '未啟用',
      busy: '啟用中…',
      on: '已啟用',
      bad: '異常'
    },
    mode: {
      heading: '選擇連線網路',
      enable: '啟動',
      experimental: '實驗性'
    }
  },
  hu: {
    title: 'Távoli hálózat',
    status: {
      off: 'Nincs bekapcsolva',
      busy: 'Bekapcsolás…',
      on: 'Bekapcsolva',
      bad: 'Hiba'
    },
    mode: {
      heading: 'Válasszon kapcsolódási hálózatot',
      enable: 'Távoli hálózat engedélyezése',
      experimental: 'Kísérleti'
    }
  },
  vi: {
    title: 'Mạng từ xa',
    status: {
      off: 'Chưa bật',
      busy: 'Đang bật…',
      on: 'Đã bật',
      bad: 'Lỗi'
    },
    mode: {
      heading: 'Chọn mạng kết nối',
      enable: 'Bật mạng từ xa',
      experimental: 'Thử nghiệm'
    }
  },
  ja: {
    title: 'リモートネットワーク',
    status: {
      off: '未有効',
      busy: '有効化中…',
      on: '有効',
      bad: 'エラー'
    },
    mode: {
      heading: '接続ネットワークを選択',
      enable: 'リモートネットワークを有効化',
      experimental: '試験的'
    }
  },
  cz: {
    title: 'Vzdálená síť',
    status: {
      off: 'Není zapnuto',
      busy: 'Zapínání…',
      on: 'Zapnuto',
      bad: 'Chyba'
    },
    mode: {
      heading: 'Vyberte připojovací síť',
      enable: 'Zapnout vzdálenou síť',
      experimental: 'Experimentální'
    }
  },
  uk: {
    title: 'Віддалена мережа',
    status: {
      off: 'Не увімкнено',
      busy: 'Увімкнення…',
      on: 'Увімкнено',
      bad: 'Помилка'
    },
    mode: {
      heading: 'Виберіть мережу підключення',
      enable: 'Увімкнути віддалену мережу',
      experimental: 'Експериментально'
    }
  },
  nb: {
    title: 'Fjernnettverk',
    status: {
      off: 'Ikke aktivert',
      busy: 'Aktiverer…',
      on: 'Aktivert',
      bad: 'Feil'
    },
    mode: {
      heading: 'Velg et tilkoblingsnettverk',
      enable: 'Aktiver fjernnettverk',
      experimental: 'Eksperimentell'
    }
  },
  th: {
    title: 'เครือข่ายระยะไกล',
    status: {
      off: 'ยังไม่เปิดใช้งาน',
      busy: 'กำลังเปิดใช้งาน…',
      on: 'เปิดใช้งานแล้ว',
      bad: 'ข้อผิดพลาด'
    },
    mode: {
      heading: 'เลือกเครือข่ายการเชื่อมต่อ',
      enable: 'เปิดใช้งานเครือข่ายระยะไกล',
      experimental: 'ทดลอง'
    }
  }
};

export default function remoteNetworkMessages(language: string): RemoteNetworkMessages {
  return messages[language as SupportedLanguage] ?? messages.en;
}
