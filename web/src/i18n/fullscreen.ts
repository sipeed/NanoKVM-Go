import type { SupportedLanguage } from './locale.ts';

export type FullscreenMessages = {
  error: string;
  guide: {
    title: string;
    shareMenu: string;
    addToHomeScreen: string;
    add: string;
    webAppNote: string;
    loginNotice: string;
  };
};

const messages: Record<SupportedLanguage, FullscreenMessages> = {
  en: {
    error: 'Unable to change fullscreen mode. Check browser permissions and try again.',
    guide: {
      title: 'Add NanoKVM to the Home Screen',
      shareMenu: 'Open the Share menu.',
      addToHomeScreen: 'Choose “Add to Home Screen”.',
      add: 'Tap “Add” to finish.',
      webAppNote: 'This opens NanoKVM as a Web App without browser toolbars.',
      loginNotice: 'The first launch from the Home Screen may require you to sign in again.'
    }
  },
  zh: {
    error: '无法切换全屏状态。请检查浏览器权限后重试。',
    guide: {
      title: '将 NanoKVM 添加到主屏幕',
      shareMenu: '打开分享菜单。',
      addToHomeScreen: '选择“添加到主屏幕”。',
      add: '轻点“添加”完成操作。',
      webAppNote: '以不显示浏览器工具栏的 Web App 方式打开 NanoKVM。',
      loginNotice: '首次从主屏幕打开时，可能需要重新登录。'
    }
  },
  zh_tw: {
    error: '無法切換全螢幕狀態。請檢查瀏覽器權限後再試一次。',
    guide: {
      title: '將 NanoKVM 加入主畫面',
      shareMenu: '開啟分享選單。',
      addToHomeScreen: '選擇「加入主畫面」。',
      add: '點一下「新增」以完成操作。',
      webAppNote: '這會以不顯示瀏覽器工具列的 Web App 方式開啟 NanoKVM。',
      loginNotice: '第一次從主畫面開啟時，可能需要重新登入。'
    }
  },
  de: {
    error:
      'Der Vollbildmodus konnte nicht geändert werden. Prüfen Sie die Browserberechtigungen und versuchen Sie es erneut.',
    guide: {
      title: 'NanoKVM zum Home-Bildschirm hinzufügen',
      shareMenu: 'Öffnen Sie das Teilen-Menü.',
      addToHomeScreen: 'Wählen Sie „Zum Home-Bildschirm“.',
      add: 'Tippen Sie zum Abschluss auf „Hinzufügen“.',
      webAppNote: 'NanoKVM wird damit als Web-App ohne Browser-Symbolleisten geöffnet.',
      loginNotice:
        'Beim ersten Start vom Home-Bildschirm müssen Sie sich möglicherweise erneut anmelden.'
    }
  },
  es: {
    error:
      'No se pudo cambiar el modo de pantalla completa. Comprueba los permisos del navegador e inténtalo de nuevo.',
    guide: {
      title: 'Añadir NanoKVM a la pantalla de inicio',
      shareMenu: 'Abre el menú Compartir.',
      addToHomeScreen: 'Elige «Añadir a pantalla de inicio».',
      add: 'Toca «Añadir» para terminar.',
      webAppNote: 'NanoKVM se abrirá como una app web sin barras del navegador.',
      loginNotice:
        'Es posible que debas iniciar sesión de nuevo la primera vez que abras la app desde el inicio.'
    }
  },
  fr: {
    error:
      'Impossible de changer le mode plein écran. Vérifiez les autorisations du navigateur et réessayez.',
    guide: {
      title: 'Ajouter NanoKVM à l’écran d’accueil',
      shareMenu: 'Ouvrez le menu Partager.',
      addToHomeScreen: 'Choisissez « Sur l’écran d’accueil ».',
      add: 'Touchez « Ajouter » pour terminer.',
      webAppNote: 'NanoKVM s’ouvrira comme une app web sans barres du navigateur.',
      loginNotice:
        'Une nouvelle connexion peut être nécessaire au premier lancement depuis l’écran d’accueil.'
    }
  },
  it: {
    error:
      'Impossibile cambiare la modalità a schermo intero. Controlla i permessi del browser e riprova.',
    guide: {
      title: 'Aggiungi NanoKVM alla schermata Home',
      shareMenu: 'Apri il menu Condividi.',
      addToHomeScreen: 'Scegli “Aggiungi alla schermata Home”.',
      add: 'Tocca “Aggiungi” per terminare.',
      webAppNote: 'NanoKVM verrà aperto come app web senza barre del browser.',
      loginNotice:
        'Al primo avvio dalla schermata Home potrebbe essere necessario accedere di nuovo.'
    }
  },
  nl: {
    error:
      'De volledig-schermmodus kon niet worden gewijzigd. Controleer de browsermachtigingen en probeer het opnieuw.',
    guide: {
      title: 'NanoKVM aan het beginscherm toevoegen',
      shareMenu: 'Open het deelmenu.',
      addToHomeScreen: 'Kies ‘Zet op beginscherm’.',
      add: 'Tik op ‘Voeg toe’ om af te ronden.',
      webAppNote: 'NanoKVM wordt als webapp zonder browserwerkbalken geopend.',
      loginNotice: 'Bij de eerste start vanaf het beginscherm moet u mogelijk opnieuw inloggen.'
    }
  },
  da: {
    error:
      'Fuldskærmstilstanden kunne ikke ændres. Kontrollér browserens tilladelser, og prøv igen.',
    guide: {
      title: 'Føj NanoKVM til hjemmeskærmen',
      shareMenu: 'Åbn menuen Del.',
      addToHomeScreen: 'Vælg “Føj til hjemmeskærm”.',
      add: 'Tryk på “Tilføj” for at afslutte.',
      webAppNote: 'NanoKVM åbnes som en webapp uden browserens værktøjslinjer.',
      loginNotice: 'Du skal muligvis logge ind igen første gang, appen åbnes fra hjemmeskærmen.'
    }
  },
  nb: {
    error: 'Kunne ikke endre fullskjermmodus. Kontroller nettlesertillatelsene og prøv på nytt.',
    guide: {
      title: 'Legg NanoKVM til på Hjem-skjermen',
      shareMenu: 'Åpne Del-menyen.',
      addToHomeScreen: 'Velg «Legg til på Hjem-skjermen».',
      add: 'Trykk på «Legg til» for å fullføre.',
      webAppNote: 'NanoKVM åpnes som en nettapp uten nettleserens verktøylinjer.',
      loginNotice: 'Du må kanskje logge inn på nytt første gang du starter fra Hjem-skjermen.'
    }
  },
  pl: {
    error:
      'Nie udało się zmienić trybu pełnoekranowego. Sprawdź uprawnienia przeglądarki i spróbuj ponownie.',
    guide: {
      title: 'Dodaj NanoKVM do ekranu początkowego',
      shareMenu: 'Otwórz menu Udostępnij.',
      addToHomeScreen: 'Wybierz „Dodaj do ekranu początkowego”.',
      add: 'Stuknij „Dodaj”, aby zakończyć.',
      webAppNote: 'NanoKVM otworzy się jako aplikacja webowa bez pasków przeglądarki.',
      loginNotice:
        'Przy pierwszym uruchomieniu z ekranu początkowego może być wymagane ponowne logowanie.'
    }
  },
  ru: {
    error:
      'Не удалось изменить полноэкранный режим. Проверьте разрешения браузера и повторите попытку.',
    guide: {
      title: 'Добавление NanoKVM на экран «Домой»',
      shareMenu: 'Откройте меню «Поделиться».',
      addToHomeScreen: 'Выберите «На экран Домой».',
      add: 'Нажмите «Добавить», чтобы завершить.',
      webAppNote: 'NanoKVM откроется как веб-приложение без панелей браузера.',
      loginNotice: 'При первом запуске с экрана «Домой» может потребоваться повторный вход.'
    }
  },
  uk: {
    error:
      'Не вдалося змінити повноекранний режим. Перевірте дозволи браузера та повторіть спробу.',
    guide: {
      title: 'Додавання NanoKVM на початковий екран',
      shareMenu: 'Відкрийте меню «Поділитися».',
      addToHomeScreen: 'Виберіть «На початковий екран».',
      add: 'Торкніться «Додати», щоб завершити.',
      webAppNote: 'NanoKVM відкриється як вебпрограма без панелей браузера.',
      loginNotice: 'Під час першого запуску з початкового екрана може знадобитися повторний вхід.'
    }
  },
  cz: {
    error:
      'Režim celé obrazovky se nepodařilo změnit. Zkontrolujte oprávnění prohlížeče a zkuste to znovu.',
    guide: {
      title: 'Přidání NanoKVM na plochu',
      shareMenu: 'Otevřete nabídku Sdílet.',
      addToHomeScreen: 'Vyberte „Přidat na plochu“.',
      add: 'Dokončete klepnutím na „Přidat“.',
      webAppNote: 'NanoKVM se otevře jako webová aplikace bez panelů prohlížeče.',
      loginNotice: 'Při prvním spuštění z plochy se možná budete muset znovu přihlásit.'
    }
  },
  hu: {
    error:
      'Nem sikerült módosítani a teljes képernyős módot. Ellenőrizze a böngésző engedélyeit, majd próbálja újra.',
    guide: {
      title: 'A NanoKVM hozzáadása a Főképernyőhöz',
      shareMenu: 'Nyissa meg a Megosztás menüt.',
      addToHomeScreen: 'Válassza a „Főképernyőhöz adás” lehetőséget.',
      add: 'A befejezéshez koppintson a „Hozzáadás” gombra.',
      webAppNote: 'A NanoKVM böngészősávok nélküli webalkalmazásként nyílik meg.',
      loginNotice: 'A Főképernyőről történő első indításkor lehet, hogy újra be kell jelentkeznie.'
    }
  },
  id: {
    error: 'Tidak dapat mengubah mode layar penuh. Periksa izin browser lalu coba lagi.',
    guide: {
      title: 'Tambahkan NanoKVM ke Layar Utama',
      shareMenu: 'Buka menu Bagikan.',
      addToHomeScreen: 'Pilih “Tambahkan ke Layar Utama”.',
      add: 'Ketuk “Tambah” untuk menyelesaikan.',
      webAppNote: 'NanoKVM akan dibuka sebagai App Web tanpa bar alat browser.',
      loginNotice: 'Anda mungkin perlu masuk lagi saat pertama kali membuka dari Layar Utama.'
    }
  },
  vi: {
    error:
      'Không thể thay đổi chế độ toàn màn hình. Hãy kiểm tra quyền của trình duyệt rồi thử lại.',
    guide: {
      title: 'Thêm NanoKVM vào Màn hình chính',
      shareMenu: 'Mở menu Chia sẻ.',
      addToHomeScreen: 'Chọn “Thêm vào Màn hình chính”.',
      add: 'Chạm “Thêm” để hoàn tất.',
      webAppNote: 'NanoKVM sẽ mở dưới dạng ứng dụng web không có thanh công cụ trình duyệt.',
      loginNotice: 'Bạn có thể phải đăng nhập lại trong lần đầu mở từ Màn hình chính.'
    }
  },
  ja: {
    error:
      '全画面表示を切り替えられませんでした。ブラウザの権限を確認して、もう一度お試しください。',
    guide: {
      title: 'NanoKVMをホーム画面に追加',
      shareMenu: '共有メニューを開きます。',
      addToHomeScreen: '「ホーム画面に追加」を選択します。',
      add: '「追加」をタップして完了します。',
      webAppNote: 'NanoKVMがブラウザのツールバーなしのWebアプリとして開きます。',
      loginNotice: 'ホーム画面から初めて起動するときは、再ログインが必要になる場合があります。'
    }
  },
  ko: {
    error: '전체 화면 모드를 변경할 수 없습니다. 브라우저 권한을 확인한 후 다시 시도하세요.',
    guide: {
      title: '홈 화면에 NanoKVM 추가',
      shareMenu: '공유 메뉴를 여세요.',
      addToHomeScreen: '“홈 화면에 추가”를 선택하세요.',
      add: '“추가”를 눌러 완료하세요.',
      webAppNote: 'NanoKVM이 브라우저 도구 막대가 없는 웹 앱으로 열립니다.',
      loginNotice: '홈 화면에서 처음 실행할 때 다시 로그인해야 할 수 있습니다.'
    }
  },
  th: {
    error: 'ไม่สามารถเปลี่ยนโหมดเต็มหน้าจอได้ โปรดตรวจสอบสิทธิ์ของเบราว์เซอร์แล้วลองอีกครั้ง',
    guide: {
      title: 'เพิ่ม NanoKVM ไปยังหน้าจอโฮม',
      shareMenu: 'เปิดเมนูแชร์',
      addToHomeScreen: 'เลือก “เพิ่มไปยังหน้าจอโฮม”',
      add: 'แตะ “เพิ่ม” เพื่อเสร็จสิ้น',
      webAppNote: 'NanoKVM จะเปิดเป็นเว็บแอปที่ไม่มีแถบเครื่องมือเบราว์เซอร์',
      loginNotice: 'การเปิดจากหน้าจอโฮมครั้งแรกอาจต้องเข้าสู่ระบบอีกครั้ง'
    }
  }
};

export default function fullscreenMessages(language: string): FullscreenMessages {
  return messages[language as SupportedLanguage] ?? messages.en;
}
