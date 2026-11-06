import type { SupportedLanguage } from './locale.ts';

export type ScreenTimelapseRecordingMessages = {
  title: string;
  low: string;
  default: string;
  high: string;
  activeHint: string;
  loadFailed: string;
  saveFailed: string;
  saved: string;
  savedAndRestarted: string;
  noChanges: string;
  save: string;
};

const messages: Record<SupportedLanguage, ScreenTimelapseRecordingMessages> = {
  en: {
    title: 'Recording quality',
    low: 'Low',
    default: 'Default',
    high: 'High',
    activeHint:
      'Higher recording quality uses more storage. Saving changes briefly restarts the video service.',
    loadFailed: 'Failed to load recording settings.',
    saveFailed: 'Failed to save recording settings.',
    saved: 'Recording quality saved.',
    savedAndRestarted: 'Recording quality saved and the video service restarted.',
    noChanges: 'No changes to save.',
    save: 'Save'
  },
  zh: {
    title: '录制质量',
    low: '低',
    default: '默认',
    high: '高',
    activeHint: '录制质量越高，占用空间越大。保存后视频服务会短暂重启。',
    loadFailed: '加载录制设置失败。',
    saveFailed: '保存录制设置失败。',
    saved: '录制质量已保存。',
    savedAndRestarted: '录制质量已保存，视频服务已重启。',
    noChanges: '没有需要保存的更改。',
    save: '保存'
  },
  zh_tw: {
    title: '錄製品質',
    low: '低',
    default: '預設',
    high: '高',
    activeHint: '錄製品質越高，佔用空間越大。儲存後影片服務會短暫重新啟動。',
    loadFailed: '載入錄製設定失敗。',
    saveFailed: '儲存錄製設定失敗。',
    saved: '錄製品質已儲存。',
    savedAndRestarted: '錄製品質已儲存，影片服務已重新啟動。',
    noChanges: '沒有需要儲存的變更。',
    save: '儲存'
  },
  nl: {
    title: 'Opnamekwaliteit',
    low: 'Laag',
    default: 'Standaard',
    high: 'Hoog',
    activeHint:
      'Een hogere opnamekwaliteit gebruikt meer opslag. Als u wijzigingen opslaat, wordt de videoservice kort opnieuw gestart.',
    loadFailed: 'Laden van opname-instellingen mislukt.',
    saveFailed: 'Opslaan van opname-instellingen mislukt.',
    saved: 'Opnamekwaliteit opgeslagen.',
    savedAndRestarted: 'Opnamekwaliteit opgeslagen en de videoservice opnieuw gestart.',
    noChanges: 'Geen wijzigingen om op te slaan.',
    save: 'Opslaan'
  },
  da: {
    title: 'Optagelseskvalitet',
    low: 'Lav',
    default: 'Standard',
    high: 'Høj',
    activeHint:
      'Højere optagelseskvalitet bruger mere lagerplads. Når du gemmer ændringer, genstartes videotjenesten kortvarigt.',
    loadFailed: 'Kunne ikke indlæse optagelsesindstillinger.',
    saveFailed: 'Kunne ikke gemme optagelsesindstillinger.',
    saved: 'Optagelseskvalitet gemt.',
    savedAndRestarted: 'Optagelseskvalitet gemt, og videotjenesten er genstartet.',
    noChanges: 'Ingen ændringer at gemme.',
    save: 'Gem'
  },
  de: {
    title: 'Aufnahmequalität',
    low: 'Niedrig',
    default: 'Standard',
    high: 'Hoch',
    activeHint:
      'Eine höhere Aufnahmequalität belegt mehr Speicher. Beim Speichern von Änderungen wird der Videodienst kurz neu gestartet.',
    loadFailed: 'Aufnahmeeinstellungen konnten nicht geladen werden.',
    saveFailed: 'Aufnahmeeinstellungen konnten nicht gespeichert werden.',
    saved: 'Aufnahmequalität gespeichert.',
    savedAndRestarted: 'Aufnahmequalität gespeichert und der Videodienst wurde neu gestartet.',
    noChanges: 'Keine Änderungen zum Speichern.',
    save: 'Speichern'
  },
  es: {
    title: 'Calidad de grabación',
    low: 'Bajo',
    default: 'Predeterminado',
    high: 'Alto',
    activeHint:
      'Una mayor calidad de grabación usa más almacenamiento. Al guardar los cambios, el servicio de vídeo se reinicia brevemente.',
    loadFailed: 'No se pudo cargar la configuración de grabación.',
    saveFailed: 'No se pudo guardar la configuración de grabación.',
    saved: 'Calidad de grabación guardada.',
    savedAndRestarted: 'Calidad de grabación guardada y el servicio de vídeo reiniciado.',
    noChanges: 'No hay cambios que guardar.',
    save: 'Guardar'
  },
  fr: {
    title: 'Qualité d’enregistrement',
    low: 'Bas',
    default: 'Par défaut',
    high: 'Élevé',
    activeHint:
      'Une qualité d’enregistrement plus élevée utilise plus d’espace de stockage. Enregistrer les modifications redémarre brièvement le service vidéo.',
    loadFailed: 'Impossible de charger les paramètres d’enregistrement.',
    saveFailed: 'Impossible d’enregistrer les paramètres d’enregistrement.',
    saved: 'Qualité d’enregistrement enregistrée.',
    savedAndRestarted: 'Qualité d’enregistrement enregistrée et le service vidéo redémarré.',
    noChanges: 'Aucune modification à enregistrer.',
    save: 'Enregistrer'
  },
  id: {
    title: 'Kualitas rekaman',
    low: 'Rendah',
    default: 'Bawaan',
    high: 'Tinggi',
    activeHint:
      'Kualitas rekaman yang lebih tinggi menggunakan lebih banyak penyimpanan. Menyimpan perubahan akan memulai ulang layanan video sebentar.',
    loadFailed: 'Tidak dapat memuat pengaturan rekaman.',
    saveFailed: 'Gagal menyimpan pengaturan rekaman.',
    saved: 'Kualitas rekaman disimpan.',
    savedAndRestarted: 'Kualitas rekaman disimpan dan layanan video dimulai ulang.',
    noChanges: 'Tidak ada perubahan untuk disimpan.',
    save: 'Simpan'
  },
  it: {
    title: 'Qualità di registrazione',
    low: 'Basso',
    default: 'Predefinito',
    high: 'Alto',
    activeHint:
      'Una qualità di registrazione più alta usa più spazio di archiviazione. Il salvataggio delle modifiche riavvia brevemente il servizio video.',
    loadFailed: 'Impossibile caricare le impostazioni di registrazione.',
    saveFailed: 'Impossibile salvare le impostazioni di registrazione.',
    saved: 'Qualità di registrazione salvata.',
    savedAndRestarted: 'Qualità di registrazione salvata e il servizio video riavviato.',
    noChanges: 'Nessuna modifica da salvare.',
    save: 'Salva'
  },
  pl: {
    title: 'Jakość nagrywania',
    low: 'Niski',
    default: 'Domyślne',
    high: 'Wysoki',
    activeHint:
      'Wyższa jakość nagrywania zajmuje więcej miejsca. Zapisanie zmian na krótko ponownie uruchomi usługę wideo.',
    loadFailed: 'Nie udało się wczytać ustawień nagrywania.',
    saveFailed: 'Nie udało się zapisać ustawień nagrywania.',
    saved: 'Jakość nagrywania zapisano.',
    savedAndRestarted: 'Jakość nagrywania zapisano i ponownie uruchomiono usługę wideo.',
    noChanges: 'Brak zmian do zapisania.',
    save: 'Zapisz'
  },
  ru: {
    title: 'Качество записи',
    low: 'Низкий',
    default: 'По умолчанию',
    high: 'Высокий',
    activeHint:
      'Чем выше качество записи, тем больше места требуется. При сохранении изменений видеослужба ненадолго перезапускается.',
    loadFailed: 'Не удалось загрузить настройки записи.',
    saveFailed: 'Не удалось сохранить настройки записи.',
    saved: 'Качество записи сохранено.',
    savedAndRestarted: 'Качество записи сохранено, видеослужба перезапущена.',
    noChanges: 'Нет изменений для сохранения.',
    save: 'Сохранить'
  },
  ko: {
    title: '녹화 품질',
    low: '낮음',
    default: '기본값',
    high: '높음',
    activeHint:
      '녹화 품질이 높을수록 더 많은 저장 공간을 사용합니다. 변경 사항을 저장하면 비디오 서비스가 잠시 다시 시작됩니다.',
    loadFailed: '녹화 설정을 불러올 수 없습니다.',
    saveFailed: '녹화 설정을 저장할 수 없습니다.',
    saved: '녹화 품질을 저장했습니다.',
    savedAndRestarted: '녹화 품질을 저장했고 비디오 서비스가 다시 시작되었습니다.',
    noChanges: '저장할 변경 사항이 없습니다.',
    save: '저장'
  },
  hu: {
    title: 'Felvétel minősége',
    low: 'Alacsony',
    default: 'Alapértelmezett',
    high: 'Magas',
    activeHint:
      'A magasabb felvételi minőség több tárhelyet használ. A módosítások mentése rövid időre újraindítja a videoszolgáltatást.',
    loadFailed: 'A felvételi beállítások betöltése sikertelen.',
    saveFailed: 'A felvételi beállítások mentése sikertelen.',
    saved: 'A felvétel minősége mentve.',
    savedAndRestarted: 'A felvétel minősége mentve, a videoszolgáltatás újraindult.',
    noChanges: 'Nincs mentendő módosítás.',
    save: 'Mentés'
  },
  vi: {
    title: 'Chất lượng bản ghi',
    low: 'Thấp',
    default: 'Mặc định',
    high: 'Cao',
    activeHint:
      'Chất lượng bản ghi cao hơn sẽ dùng nhiều dung lượng hơn. Lưu thay đổi sẽ khởi động lại dịch vụ video trong giây lát.',
    loadFailed: 'Không thể tải cài đặt ghi.',
    saveFailed: 'Không thể lưu cài đặt ghi.',
    saved: 'Đã lưu chất lượng bản ghi.',
    savedAndRestarted: 'Đã lưu chất lượng bản ghi và dịch vụ video đã khởi động lại.',
    noChanges: 'Không có thay đổi để lưu.',
    save: 'Lưu'
  },
  ja: {
    title: '録画品質',
    low: '低い',
    default: 'デフォルト',
    high: '高い',
    activeHint:
      '録画品質を高くするとストレージを多く使用します。変更を保存すると映像サービスが一時的に再起動します。',
    loadFailed: '録画設定の読み込みに失敗しました。',
    saveFailed: '録画設定の保存に失敗しました。',
    saved: '録画品質を保存しました。',
    savedAndRestarted: '録画品質を保存し、映像サービスが再起動しました。',
    noChanges: '保存する変更はありません。',
    save: '保存'
  },
  cz: {
    title: 'Kvalita záznamu',
    low: 'Nízký',
    default: 'Výchozí',
    high: 'Vysoký',
    activeHint:
      'Vyšší kvalita záznamu zabírá více úložiště. Uložení změn krátce restartuje videoslužbu.',
    loadFailed: 'Nastavení záznamu se nepodařilo načíst.',
    saveFailed: 'Nastavení záznamu se nepodařilo uložit.',
    saved: 'Kvalita záznamu byla uložena.',
    savedAndRestarted: 'Kvalita záznamu byla uložena a videoslužba byla restartována.',
    noChanges: 'Žádné změny k uložení.',
    save: 'Uložit'
  },
  uk: {
    title: 'Якість запису',
    low: 'Низький',
    default: 'За замовчуванням',
    high: 'Високий',
    activeHint:
      'Вища якість запису використовує більше сховища. Збереження змін ненадовго перезапустить відеослужбу.',
    loadFailed: 'Не вдалося завантажити налаштування запису.',
    saveFailed: 'Не вдалося зберегти налаштування запису.',
    saved: 'Якість запису збережено.',
    savedAndRestarted: 'Якість запису збережено, а відеослужбу перезапущено.',
    noChanges: 'Немає змін для збереження.',
    save: 'Зберегти'
  },
  nb: {
    title: 'Opptakskvalitet',
    low: 'Lav',
    default: 'Standard',
    high: 'Høy',
    activeHint:
      'Høyere opptakskvalitet bruker mer lagringsplass. Lagring av endringer starter videotjenesten på nytt et øyeblikk.',
    loadFailed: 'Kunne ikke laste inn opptaksinnstillinger.',
    saveFailed: 'Kunne ikke lagre opptaksinnstillinger.',
    saved: 'Opptakskvaliteten er lagret.',
    savedAndRestarted: 'Opptakskvaliteten er lagret, og videotjenesten er startet på nytt.',
    noChanges: 'Ingen endringer å lagre.',
    save: 'Lagre'
  },
  th: {
    title: 'คุณภาพการบันทึก',
    low: 'ต่ำ',
    default: 'ค่าเริ่มต้น',
    high: 'สูง',
    activeHint:
      'คุณภาพการบันทึกที่สูงขึ้นใช้พื้นที่จัดเก็บมากขึ้น การบันทึกการเปลี่ยนแปลงจะรีสตาร์ทบริการวิดีโอชั่วคราว',
    loadFailed: 'ไม่สามารถโหลดการตั้งค่าการบันทึกได้',
    saveFailed: 'ไม่สามารถบันทึกการตั้งค่าการบันทึกได้',
    saved: 'บันทึกคุณภาพการบันทึกแล้ว',
    savedAndRestarted: 'บันทึกคุณภาพการบันทึกและรีสตาร์ทบริการวิดีโอแล้ว',
    noChanges: 'ไม่มีการเปลี่ยนแปลงให้บันทึก',
    save: 'บันทึก'
  }
};

export default function screenTimelapseRecordingMessages(
  language: string
): ScreenTimelapseRecordingMessages {
  return messages[language as SupportedLanguage] ?? messages.en;
}
