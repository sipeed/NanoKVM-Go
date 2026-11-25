const pl = {
  translation: {
    storage: {
      cards: { mount: 'Montowanie dysku', share: 'Udostępnianie plików' },
      title: 'Obrazy i udostępnianie plików',
      empty: 'Brak obrazów',
      notAttached: 'Nic nie jest zamontowane',
      attached: 'Zamontowano: {{name}}',
      eject: 'Wysuń',
      mount: 'Zamontuj',
      mountMode: 'Tryb montowania',
      mountModeDisk: 'Pamięć masowa',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Otwórz menedżer plików',
      refresh: 'Odśwież',
      delete: 'Usuń',
      deleteTitle: 'Usuń nośnik',
      deleteConfirm: 'Usunąć {{name}}? Znajdujące się w nim pliki zostaną utracone.',
      downloadImage: 'Pobierz obraz',
      typePartition: 'Udostępnione pliki',
      volumeLabel: 'Dysk wymienny',
      groups: {
        share: 'Udostępnianie plików',
        images: 'Obrazy'
      },
      failed: 'Operacja nie powiodła się',
      hidOnly: 'Tryb tylko HID: dysk wirtualny jest niedostępny',
      disabled: 'Dysk wirtualny jest wyłączony (Ustawienia - Urządzenie)',
      force: {
        title: 'Odebrać nośnik siłą?',
        desc: 'Zdalny komputer nadal używa tego nośnika. Odebranie go ponownie wyliczy urządzenie USB i może utracić wszystko, czego jeszcze nie zapisało na dysku.',
        menuHint: 'Zdalny komputer używa tego nośnika. Otwórz menedżer plików, aby odebrać go siłą.'
      },
      tips: {
        title: 'Jak przesłać obrazy',
        usb1: 'Podłącz urządzenie NanoKVM do komputera przez USB.',
        usb2: 'Na tej stronie zamontuj udostępnione pliki w trybie pamięci masowej.',
        usb3: 'Otwórz dysk wirtualny na swoim komputerze i skopiuj plik obrazu do katalogu głównego dysku wirtualnego.',
        usb4: 'Po zakończeniu kopiowania wysuń dysk: obraz pojawi się wtedy na liście.',
        scp1: 'Upewnij się że NanoKVM i twój komputer są na tej samej sieci lokalnej.',
        scp2: 'Otwórz terminal na komputerze i użyj komendę SCP aby przesłać obraz do katalogu /data na NanoKVM.',
        scp3: 'Przykład: scp lokalizacja-zrodlowego-obrazu root@ip-twojego-nanokvm:/data'
      },
      job: {
        running: 'Przetwarzanie {{name}}'
      },
      enableDisk: {
        title: 'Włączyć dysk wirtualny?',
        desc: 'Do zamontowania nośnika potrzebny jest dysk wirtualny, który jest teraz wyłączony. Włączyć go? Klawiatura i mysz zdalnego komputera rozłączą się na chwilę.',
        menuHint: 'Dysk wirtualny jest wyłączony. Włącz go w Ustawienia - Urządzenie.'
      },
      add: {
        title: 'Dodaj nośnik',
        upload: 'Prześlij plik ISO lub IMG',
        fromUrl: 'Pobierz z adresu URL',
        url: 'Adres URL obrazu',
        urlRequired: 'Podaj adres URL',
        onlyImages: 'Można montować tylko pliki .iso i .img',
        dropHint: 'Przeciągnij tutaj plik ISO lub IMG albo kliknij, aby go wybrać',
        cdromOnlyIso: 'CD-ROM może zawierać wyłącznie obraz ISO 9660 / UDF',
        uploadFailed: 'Przesyłanie nie powiodło się',
        downloadFailed: 'Pobieranie nie powiodło się',
        uploading: 'Przesyłanie {{name}}',
        downloading: 'Pobieranie {{name}}',
        done: '{{name}} jest gotowy',
        checksumFailed: 'Obraz nie zgadza się z sumą kontrolną SHA-256.',
        blocked:
          'Zdalny komputer używa dysku. Bezpiecznie wysuń wirtualny dysk USB na zdalnym komputerze, a następnie wyłącz przełącznik Dysk wirtualny w Ustawienia > Urządzenie.'
      },
      files: {
        reasonNotInitialized:
          'Wirtualny dysk USB nie został zainicjowany. Przed udostępnianiem plików zainicjuj go w Ustawienia > Urządzenie.',
        selectMedium: 'Wybierz nośnik, aby przeglądać znajdujące się na nim pliki.',
        reasonAttached: 'Ten nośnik jest podłączony do zdalnego komputera.',
        reasonAttachedHint:
          'Bezpiecznie wysuń wirtualny dysk USB na zdalnym komputerze, a następnie wyłącz przełącznik Dysk wirtualny w Ustawienia > Urządzenie.',
        reasonNotBrowsable: 'Ten nośnik zawiera system plików, którego menedżer nie może otworzyć.',
        reasonNoLoop: 'To urządzenie nie może montować obrazów dysków (brak obsługi loop).',
        reasonISOImage:
          'Obrazu ISO 9660 nie można tu przeglądać. Nadal można go zamontować na zdalnym komputerze.',
        empty: 'Ten folder jest pusty',
        noMatch: 'Żaden plik nie pasuje do wyszukiwania.',
        free: '{{size}} wolne',
        search: 'Szukaj w tym folderze',
        sort: {
          modified: 'Zmodyfikowano',
          name: 'Nazwa',
          size: 'Rozmiar',
          type: 'Typ',
          asc: 'Rosnąco',
          desc: 'Malejąco'
        },
        view: {
          list: 'Widok listy',
          grid: 'Widok ikon'
        },
        folder: 'Folder',
        itemCount: 'Liczba elementów: {{count}}',
        itemCount_one: '{{count}} element',
        itemCount_other: 'Liczba elementów: {{count}}',
        dropHint: 'Upuść tutaj pliki, aby przesłać je do tego folderu',
        up: 'Przejdź do folderu nadrzędnego',
        uploadButton: 'Prześlij',
        newFolder: 'Nowy folder',
        newFolderPlaceholder: 'Nazwa folderu',
        create: 'Utwórz',
        upload: 'Prześlij plik na nośnik',
        deleteFolderTitle: 'Usuń folder',
        deleteFolderConfirm:
          'Usunąć folder {{name}} wraz z całą zawartością? Tej operacji nie można cofnąć.',
        uploading: 'Przesyłanie…',
        uploadFailed: 'Przesyłanie nie powiodło się',
        download: 'Pobierz',
        delete: 'Usuń',
        deleteTitle: 'Usuń plik',
        deleteConfirm: 'Usunąć {{name}} z nośnika?'
      },
      yes: 'Tak',
      no: 'Nie'
    },
    head: {
      storage: 'Obrazy i udostępnianie plików',
      desktop: 'Zdalny pulpit',
      login: 'Zaloguj się',
      changePassword: 'Zmień Hasło',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Nie można skopiować do schowka. Zezwól przeglądarce na dostęp do schowka i spróbuj ponownie.'
    },
    auth: {
      login: 'Zaloguj się',
      language: 'Język',
      placeholderUsername: 'Wprowadź nazwę użykownika',
      placeholderPassword: 'wprowadź hasło',
      placeholderPassword2: 'wprowadź hasło ponownie',
      noEmptyUsername: 'nazwa użykownika nie może być pusta',
      noEmptyPassword: 'hasło nie może być puste',
      noAccount:
        'Nie udało się uzyskać informacji o użytkowniku, odśwież stronę lub zresetuj hasło',
      invalidUser: 'Błędne hasło lub nazwa użykownika',
      error: 'niespodziewany błąd',
      twoFactorTitle: 'Wymagana weryfikacja',
      twoFactorDesc:
        'Aby kontynuować logowanie, wpisz kod uwierzytelniający lub nieużyty kod odzyskiwania.',
      sensitiveActionDesc:
        'Wpisz kod uwierzytelniający lub nieużyty kod odzyskiwania, aby kontynuować.',
      placeholderAuthCode: 'Kod uwierzytelniający',
      noEmptyAuthCode: 'Kod uwierzytelniający jest wymagany',
      verify: 'Zweryfikuj',
      backToLogin: 'Wróć do logowania',
      verificationFailed: 'Weryfikacja nie powiodła się',
      twoFactorEnabledLogout: {
        message:
          'Uwierzytelnianie dwuskładnikowe zostało włączone. Ze względów bezpieczeństwa wszystkie sesje zostały wylogowane. Zaloguj się ponownie.'
      },
      rateLimited: 'Zbyt wiele nieudanych prób. Spróbuj ponownie za {{seconds}} s.',
      recoveryCodeUsed: 'Kod odzyskiwania użyty do tego logowania został unieważniony.',
      changePassword: 'Zmień Hasło',
      changePasswordDesc:
        'Dla bezpieczeństwa Twojego urządzenia, proszę zmień hasło do logowania w sieci.',
      passwordVerificationRequired:
        'Wróć do strony konta w ustawieniach i ponownie zweryfikuj tę zmianę hasła.',
      passwordVerificationExpired:
        'Weryfikacja zmiany hasła wygasła. Wróć do strony konta w ustawieniach i zweryfikuj ponownie.',
      returnToSettings: 'Wróć do ustawień',
      differentPassword: 'hasła nie zgadzają się',
      illegalUsername: 'nazwa użytkownika zawiera niedozwolone znaki',
      illegalPassword: 'hasło zawiera niedozwolone znaki',
      forgetPassword: 'Zapomiałeś hasła?',
      ok: 'OK',
      cancel: 'Anuluj',
      savePassword: 'Zapisz hasło',
      loginButtonText: 'Zaloguj się',
      tips: {
        reset1: 'Domyślne konto internetowe:',
        reset2: 'Domyślne konto SSH:',
        reset3: 'Aby zresetować hasła, zapoznaj się z ',
        changeReset: 'Aby zresetować hasło logowania do interfejsu webowego, zobacz ',
        resetLink: 'Zresetuj'
      }
    },
    wifi: {
      errors: {
        connect:
          'Nie można połączyć się z Wi-Fi. Sprawdź nazwę sieci, hasło i stan Wi-Fi urządzenia, a następnie spróbuj ponownie.'
      },
      title: 'Wi-Fi',
      description: 'Skonfiguruj Wi-Fi dla NanoKVM',
      submitted: {
        title: 'Chwilę cierpliwości — NanoKVM łączy się z Twoją siecią Wi-Fi',
        tipsTitle: 'Co dalej?',
        tipSwitch: 'Możesz teraz przełączyć telefon z powrotem na swoją zwykłą sieć Wi-Fi.',
        tipCheck: 'Za chwilę sprawdź na ekranie NanoKVM nowy adres IP i otwórz go w przeglądarce.',
        tipRetry:
          'Jeśli po dłuższej chwili nic się nie dzieje, sprawdź nazwę sieci i hasło, a następnie spróbuj ponownie.'
      },
      invalidMode:
        'Bieżący tryb nie obsługuje konfiguracji sieci. Przejdź do swojego urządzenia i włącz tryb konfiguracji Wi-Fi.',
      confirmBtn: 'OK',
      finishBtn: 'Zakończono',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Hasło',
      steps: {
        settings: 'Ustawienia',
        wifi: 'Wi-Fi',
        enable: 'Włącz'
      },
      scan: {
        title: 'Wybierz sieć Wi-Fi',
        description: 'Wybierz pobliską sieć, aby kontynuować.',
        loading: 'Wyszukiwanie pobliskich sieci Wi-Fi…',
        empty: 'Nie znaleziono pobliskich sieci Wi-Fi.',
        failed: 'Nie można przeskanować sieci. Wprowadź dane ręcznie.',
        retryBtn: 'Skanuj ponownie',
        manualBtn: 'Wprowadź sieć ręcznie',
        backBtn: 'Wróć do pobliskich sieci',
        connectTitle: 'Połącz z {{ssid}}',
        connecting: 'Łączenie…',
        viewSimple: 'Przełącz na widok prosty',
        viewDetailed: 'Przełącz na widok szczegółowy'
      },
      ap: {
        authTitle: 'Wymagane uwierzytelnienie',
        authDescription: 'Aby kontynuować, wprowadź AP hasło',
        authFailed: 'Nieprawidłowe hasło AP',
        passPlaceholder: 'AP hasło',
        verifyBtn: 'Sprawdź'
      },
      qr: {
        cameraBtn: 'Zrób zdjęcie',
        pickBtn: 'Wybierz zrzut ekranu',
        scanHint:
          'Zrób zdjęcie kodu QR Wi-Fi lub wybierz jego zrzut ekranu, na przykład taki, jaki telefon pokazuje przy udostępnianiu swojej sieci Wi-Fi.',
        decoding: 'Odczytywanie kodu QR…',
        notFound:
          'Na tym zdjęciu nie znaleziono kodu QR. Wyśrodkuj kod i ustaw ostrość, a następnie spróbuj ponownie.',
        invalid: 'Ten kod QR nie zawiera użytecznych danych Wi-Fi.',
        pickerUnavailable:
          'Nic się nie otworzyło? To okno logowania może blokować aparat i zdjęcia. Zamiast tego otwórz ten adres w przeglądarce telefonu:',
        confirmTitle: 'Połączyć NanoKVM z {{ssid}}?',
        confirmDescription: 'Te dane sieciowe pochodzą z kodu QR. Sprawdź je przed połączeniem.',
        openNetwork: 'Ta sieć nie ma hasła.',
        connectBtn: 'Połącz',
        otherBtn: 'Wybierz inną sieć'
      }
    },
    notification: {
      browser: {
        title: 'Zalecenia dotyczące przeglądarki',
        description:
          'Aby zapewnić najlepszą wygodę użytkowania, zalecamy korzystanie z przeglądarki Chrome. Chociaż staramy się zapewnić kompatybilność ze wszystkimi głównymi przeglądarkami, niektóre zaawansowane funkcje mogą nie działać zgodnie z oczekiwaniami poza przeglądarką Chrome.'
      },
      btnOk: 'OK'
    },
    screen: {
      title: 'Ekran',
      video: 'Tryb wideo',
      quality: 'Jakość',
      qualityAuto: 'Automatyczny',
      qualityLossless: 'Bezstratny',
      qualityHigh: 'Wysoki',
      qualityMedium: 'Średni',
      qualityLow: 'Niski',
      scale: 'Skala',
      noSignal: 'Brak sygnału wideo',
      captureDisabled:
        'Przechwytywanie wideo jest wyłączone. Włącz je w „Ustawienia > Urządzenie > Przechwytywanie wideo”.',
      inconsistentVideoMode: 'Odtwarzanie wstrzymane. Odtwarzany jest inny tryb wideo.',
      bandwidthWarning:
        'Przepustowość sieci może być niewystarczająca dla bieżących ustawień wideo. Spróbuj obniżyć szybkość transmisji lub liczbę klatek na sekundę.',
      webrtcConnectionFailed:
        'Nie udało się nawiązać połączenia WebRTC. Przełącz na tryb wideo Direct.',
      webrtcUnavailable: {
        title: 'WebRTC jest niedostępne',
        description: 'Włącz WebRTC w przeglądarce albo przełącz na tryb wideo Direct.'
      },
      upload: 'Prześlij',
      advanced: 'Zaawansowane',
      loading: 'Ładowanie'
    },
    volume: {
      title: 'Głośność'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon nie jest zamontowany',
      tip1: 'Mikrofon wirtualny nie jest zamontowany na hoście zdalnym. Czy chcesz go teraz zamontować i włączyć?',
      tip2: 'Możesz także zarządzać tym później w „Ustawienia > Urządzenie > Wirtualny dźwięk”.',
      confirm: 'Włącz',
      cancel: 'Anuluj',
      noDevice: 'Na tym urządzeniu nie wykryto mikrofonu. Podłącz mikrofon i spróbuj ponownie.',
      permissionDenied:
        'Odmówiono dostępu do mikrofonu. Zezwól na dostęp w ustawieniach przeglądarki.',
      notReadable: 'Mikrofon jest używany przez inną aplikację lub nie można go odczytać.',
      overconstrained: 'Mikrofon nie obsługuje wymaganych ustawień dźwięku.',
      securityBlocked:
        'Dostęp do mikrofonu jest zablokowany na tej stronie. Sprawdź ustawienia zabezpieczeń przeglądarki.',
      aborted: 'Uruchamianie mikrofonu zostało przerwane. Spróbuj ponownie.',
      unsupported:
        'Ta przeglądarka lub metoda połączenia nie obsługuje przechwytywania dźwięku z mikrofonu.',
      enableFailed: 'Nie udało się włączyć mikrofonu. Spróbuj ponownie.',
      disconnected: 'Mikrofon został odłączony, więc transmisja z mikrofonu została wyłączona.'
    },
    power: {
      title: 'Zasilanie',
      showConfirm: 'Potwierdzenie',
      showConfirmTip: 'Operacje zasilania wymagają dodatkowego potwierdzenia',
      powerShort: 'Zasilanie (krótkie kliknięcie)',
      powerLong: 'Zasilanie (długie kliknięcie)',
      powerConfirm: 'Wykonać operację zasilania?',
      okBtn: 'Tak',
      cancelBtn: 'Nie'
    },
    keyboard: {
      errors: {
        paste:
          'Nie można wysłać tekstu do urządzenia zdalnego. Sprawdź połączenie KVM i spróbuj ponownie.'
      },
      title: 'Klawiatura',
      paste: 'Wklej',
      tips: 'Tylko standardowe klawiaturowe znaki i symbole są obsługiwane.',
      placeholder: 'Proszę wprowadzić coś',
      submit: 'Prześlij',
      virtual: 'Klawiatura',
      swapCommandCtrl: 'Zamień Cmd i Ctrl',
      shortcut: {
        title: 'Skróty',
        custom: 'Niestandardowe',
        capture: 'Kliknij tutaj, aby przechwycić skrót',
        clear: 'Wyczyść',
        save: 'Zapisz',
        captureTips:
          'Przechwytywanie klawiszy systemowych (takich jak klawisz Windows) wymaga uprawnienia do pełnego ekranu.',
        enterFullScreen: 'Przełącz tryb pełnoekranowy.'
      },
      leaderKey: {
        title: 'Klawisz Leader',
        desc: 'Omiń ograniczenia przeglądarki i wyślij skróty systemowe bezpośrednio do zdalnego hosta.',
        howToUse: 'Jak używać',
        simultaneous: {
          title: 'Tryb symultaniczny',
          desc1: 'Naciśnij i przytrzymaj klawisz Leader, a następnie naciśnij skrót.',
          desc2: 'Intuicyjne, ale może kolidować ze skrótami systemowymi.'
        },
        sequential: {
          title: 'Tryb sekwencyjny',
          desc1:
            'Naciśnij klawisz Leader → naciśnij skrót w sekwencji → ponownie naciśnij klawisz Leader.',
          desc2:
            'Wymaga większej liczby kroków, ale całkowicie pozwala uniknąć konfliktów systemowych.'
        },
        enable: 'Włącz klawisz Leader',
        tip: 'Po przypisaniu jako klawisz Leader ten klawisz działa wyłącznie jako wyzwalacz skrótów i traci swoje domyślne zachowanie.',
        placeholder: 'Naciśnij klawisz Leader',
        shiftRight: 'Prawy Shift',
        ctrlRight: 'Prawy Ctrl',
        metaRight: 'Prawy Win',
        submit: 'Prześlij',
        recorder: {
          rec: 'NAGR',
          activate: 'Aktywuj klawisze',
          input: 'Proszę nacisnąć skrót...'
        }
      }
    },
    mouse: {
      title: 'Mysz',
      mode: 'Tryb myszki',
      absolute: 'Mysz śledząca',
      absoluteAndroid: 'Mysz śledząca (lepsza zgodność z Androidem)',
      touchScreen: 'Ekran wielodotykowy',
      relative: 'Mysz wyłączna',
      direction: 'Kierunek kółka przewijania',
      scrollUp: 'Przewiń w górę',
      scrollDown: 'Przewiń w dół',
      speed: 'Szybkość kółka przewijania',
      fast: 'Szybko',
      slow: 'Powoli',
      requestPointer: 'Korzystanie z trybu względnego. Kliknij pulpit, aby uzyskać wskaźnik myszy.',
      repairIphoneDrag: 'Napraw przeciąganie na iPhonie',
      resetHid: 'Resetuj HID',
      touchpadGuide: {
        title: 'Przewodnik po touchpadzie',
        scope: 'Dotyczy zwykłego sterowania wskaźnikiem, gdy tryb myszy jest względny.',
        swipeTitle: 'Przesuń, aby poruszyć',
        swipeDesc: 'Przesuń palcem w aktywnym obszarze ekranu, aby poruszyć zdalnym wskaźnikiem.',
        tapTitle: 'Stuknij, aby kliknąć',
        tapDesc: 'Krótkie stuknięcie wysyła jedno kliknięcie lewym przyciskiem.',
        holdTitle: 'Przytrzymaj, aby nacisnąć lewy przycisk',
        holdDesc:
          'Dotknij i trzymaj palec nieruchomo przez około 1 sekundę, aby przytrzymać lewy przycisk myszy.',
        dragTitle: 'Po przytrzymaniu przesuń, aby przeciągnąć',
        dragDesc:
          'Gdy przytrzymanie jest aktywne, przesuń palec, aby przeciągać z wciśniętym lewym przyciskiem.'
      },
      hidOnly: {
        errors: {
          load: 'Nie można wczytać stanu trybu HID-Only. Spróbuj ponownie za chwilę.',
          update:
            'Nie można potwierdzić, czy tryb HID-Only został zaktualizowany. Otwórz ponownie to ustawienie, aby sprawdzić bieżący stan, a następnie spróbuj ponownie.'
        },
        title: 'Tryb tylko HID',
        desc: 'Jeśli mysz i klawiatura przestaną odpowiadać, a resetowanie HID nie pomoże, może to oznaczać problem ze zgodnością między NanoKVM a urządzeniem. Spróbuj włączyć tryb HID-Only, aby uzyskać lepszą kompatybilność.',
        tip1: 'Włączenie trybu HID-Only spowoduje odmontowanie wirtualnego dysku U i sieci wirtualnej',
        tip2: 'W trybie HID-Only montowanie obrazu jest wyłączone',
        enable: 'Włącz tryb HID-Only',
        disable: 'Wyłącz tryb HID-Tylko'
      }
    },
    preview: {
      title: 'Podgląd'
    },
    image: {
      storageReadOnly: 'Nie można zapisywać plików. Wyłącz dysk wirtualny.',
      storageBusy:
        'Pamięć jest zajęta. Zakończ lub anuluj przesyłanie plików przed przełączeniem dysku wirtualnego.',
      title: 'Obrazy',
      loading: 'Ładowanie...',
      empty: 'Nic nie znaleziono',
      upload: 'Prześlij obraz',
      dropFile: 'Upuść obraz tutaj lub kliknij, aby wybrać plik',
      uploadAction: 'Prześlij',
      cancelAction: 'Anuluj',
      mount: 'Zamontuj obraz',
      mounted: 'Zamontowany',
      uploadError: 'Przesyłanie nie powiodło się, spróbuj ponownie',
      uploadInsufficientStorage: 'Niewystarczająca ilość miejsca na dysku',
      uploadTooLarge: 'Obraz przekracza limit rozmiaru',
      cancelUploadError: 'Nie udało się anulować przesyłania',
      mountMode: 'Tryb montowania',
      massStorage: 'Pamięć masowa',
      cdRom: 'CD ROM',
      readOnly: 'Tylko do odczytu',
      mountFailed: 'Nie udało się zamontować obrazu',
      mountDesc:
        'W niektórych systemach wymagane jest wyjęcie dysku wirtualnego na zdalnym hoście przed zamontowaniem obrazu.',
      unmountFailed: 'Odmontowanie nie powiodło się',
      unmountDesc:
        'W niektórych systemach należy ręcznie wysunąć obraz na hoście zdalnym przed odmontowaniem.',
      refresh: 'Odśwież listę obrazów',
      download: 'Pobierz obraz',
      attention: 'Uwaga',
      deleteConfirm: 'Czy na pewno chcesz usunąć to zdjęcie?',
      deleteFailed: 'Nie udało się usunąć zdjęcia',
      okBtn: 'Tak',
      cancelBtn: 'Nie'
    },
    script: {
      title: 'Skrypty',
      upload: 'Prześlij',
      run: 'Uruchom',
      runBackground: 'Uruchomiony w tle',
      runFailed: 'Uruchomienie nie powiodło się',
      uploadFailed: 'Nie udało się przesłać skryptu',
      deleteFailed: 'Nie udało się usunąć skryptu',
      attention: 'Uwaga',
      delDesc: 'Czy na pewno chcesz usunąć ten plik?',
      confirm: 'Tak',
      cancel: 'Nie',
      delete: 'Usuń',
      close: 'Zamknij'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      confirm: 'OK',
      submit: 'Prześlij',
      username: 'Nazwa użytkownika',
      password: 'Hasło',
      authCode: 'Kod uwierzytelniający',
      authorizeTitle: 'Autoryzuj terminal WWW',
      codeAccountHint: 'Wprowadź kod uwierzytelniający lub nieużyty kod odzyskiwania.',
      authFailed: 'Uwierzytelnianie nie powiodło się'
    },
    wol: {
      errors: {
        send: 'Nie można wysłać polecenia wybudzenia. Sprawdź adres MAC i połączenie sieciowe, a następnie spróbuj ponownie.',
        loadFailed:
          'Nie można załadować listy urządzeń Wake-on-LAN. Sprawdź, czy NanoKVM jest online, i spróbuj ponownie.',
        renameFailed:
          'Nie można zmienić nazwy tego urządzenia. Sprawdź adres MAC i spróbuj ponownie.',
        deleteFailed: 'Nie można usunąć tego urządzenia. Spróbuj ponownie za chwilę.'
      },
      title: 'Wake-on-LAN',
      sending: 'Wysyłanie komendy...',
      sent: 'Komenda wysłana',
      input: 'Wprowadź numer adresu MAC',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (opcjonalnie)',
      sha256Placeholder: 'Wprowadź 64-znakową sumę kontrolną SHA-256',
      invalidSHA256: 'SHA-256 musi być 64-znakowym ciągiem szesnastkowym',
      title: 'Narzędzie do pobierania obrazów',
      input: 'Proszę wprowadzić zdalny obraz URL',
      ok: 'OK',
      disabled: '/data partycja to RO, więc nie możemy pobrać obrazu',
      downloadingFile: 'Pobieranie: {{file}}',
      downloadingProgress: 'Pobieranie ({{percentage}}): {{file}}',
      failed: 'Niepowodzenie',
      insufficientStorage: 'Niewystarczająca ilość miejsca na dysku',
      remaining: 'Pozostało: {{time}}',
      cancel: 'Anuluj',
      cancelFailed: 'Nie udało się anulować pobierania',
      pause: 'Wstrzymaj',
      resume: 'Wznów',
      pauseFailed: 'Nie udało się wstrzymać pobierania',
      resumeFailed: 'Nie udało się wznowić pobierania'
    },
    settings: {
      back: 'Wstecz',
      close: 'Zamknij',
      ai: {
        errors: {
          loadServices:
            'Nie można wczytać stanu usług AI. Sprawdź, czy NanoKVM jest online, a następnie spróbuj ponownie.',
          loadMcpConfig: 'Nie można wczytać konfiguracji usługi MCP. Spróbuj ponownie za chwilę.',
          updateMcpService:
            'Nie można zaktualizować stanu usługi MCP. Sprawdź bieżący stan przed ponowną próbą.',
          accessMcpApiKey:
            'Nie można pobrać klucza API MCP. Spróbuj ponownie i ukończ weryfikację, jeśli zostanie wyświetlona prośba.',
          regenerateMcpApiKey:
            'Nie można ponownie wygenerować klucza API MCP. Spróbuj ponownie za chwilę.',
          regeneratedMcpApiKeyUnavailable:
            'Klucz API został ponownie wygenerowany, ale nie można go wyświetlić. Spróbuj wyświetlić go ponownie. Poprzedni klucz nie jest już ważny.',
          loadMemoryFabricState:
            'Nie można wczytać stanu usługi Memory Fabric. Spróbuj ponownie za chwilę.',
          updateMemoryFabricService:
            'Nie można zaktualizować stanu usługi Memory Fabric. Sprawdź bieżący stan przed ponowną próbą.',
          loadScreenTimelapseState:
            'Nie można wczytać stanu usługi timelapse ekranu. Spróbuj ponownie za chwilę.',
          updateScreenTimelapseService:
            'Nie można zaktualizować stanu usługi timelapse ekranu. Sprawdź bieżący stan przed ponowną próbą.',
          openScreenTimelapse:
            'Nie można otworzyć odtwarzania timelapse ekranu. Spróbuj ponownie za chwilę.',
          loadScreenTimelapseRecordings:
            'Nie można wczytać nagrań timelapse ekranu. Spróbuj ponownie za chwilę.',
          playScreenTimelapse:
            'Tego nagrania timelapse nie można teraz odtworzyć. Wybierz datę ponownie lub spróbuj później.',
          screenTimelapseFullscreen:
            'Nie można zmienić trybu pełnoekranowego. Sprawdź uprawnienia przeglądarki i spróbuj ponownie.'
        },
        title: 'AI',
        mcpService: 'Usługa MCP',
        mcpServiceDesc:
          'Uwierzytelniona wspólna brama do sterowania, multimediów i włączonych narzędzi Memory Fabric',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Lokalne przetwarzanie pamięci kontekstu agentów i OCR',
        memoryFabricRemoteRequiresMcp:
          'Włącz powyżej usługę MCP, aby narzędzia pamięci i OCR były dostępne zdalnie.',
        memoryFabricRemoteAvailable:
          'Narzędzia pamięci i OCR są dostępne przez powyższy endpoint /api/mcp przy użyciu tego samego klucza API.',
        memoryFabricConfig: {
          title: 'Konfiguracja',
          description: 'Zarządzaj wyłącznie ustawieniami domyślnego modelu Memory Fabric.',
          loadFailed: 'Nie udało się wczytać konfiguracji Memory Fabric.',
          saveFailed: 'Nie udało się zapisać konfiguracji Memory Fabric.',
          saved: 'Konfiguracja została zapisana.',
          savedRestarted: 'Konfiguracja została zapisana, a Memory Fabric uruchomiono ponownie.',
          noChanges: 'Brak zmian w konfiguracji.',
          reload: 'Wczytaj ponownie',
          save: 'Zapisz',
          unsaved: 'Niezapisane zmiany',
          conflictTitle: 'Konfiguracja uległa zmianie',
          conflictDescription:
            'Konfiguracja została zmieniona w innym miejscu. Przed ponownym zapisaniem wczytaj najnowszą wersję.',
          conflictReload: 'Wczytaj najnowszą wersję',
          discardTitle: 'Odrzucić niezapisane zmiany?',
          discardDescription:
            'Ponowne wczytanie zastąpi bieżący formularz ostatnio zapisaną konfiguracją.',
          discardConfirm: 'Odrzuć i wczytaj ponownie',
          sections: {
            models: 'Modele'
          },
          openAICompatibleHint:
            'Obsługuje dostawców modeli zgodnych z OpenAI. Wprowadź nazwę modelu, bazowy adres URL i klucz API otrzymane od dostawcy.',
          fields: {
            model: 'Nazwa modelu',
            baseUrl: 'Bazowy adres URL',
            maxTokens: 'Maksymalna liczba tokenów',
            timeoutSeconds: 'Limit czasu (sekundy)'
          },
          apiKey: {
            title: 'Klucz API',
            configured: 'Skonfigurowany',
            notConfigured: 'Nieskonfigurowany',
            keep: 'Zachowaj',
            replace: 'Zastąp',
            clear: 'Usuń',
            replacement: 'Nowy klucz API',
            replacementPlaceholder: 'Wpisz nowy klucz API',
            clearHint:
              'Zapisany klucz API zostanie usunięty; zmienna środowiskowa nadal może dostarczać klucz.',
            required: 'Wpisz nowy klucz API.'
          },
          validation: {
            required: 'To pole jest wymagane.',
            invalidUrl: 'Wpisz prawidłowy adres URL HTTP lub HTTPS.'
          }
        },
        screenTimelapseService: 'Timelapse ekranu',
        screenTimelapseServiceDesc:
          'Odtwarzaj historię ekranu Memory Fabric i wyszukuj przechwyconą zawartość ekranu',
        screenTimelapseSearchRequiresMemoryFabric:
          'Aby przeszukiwać nagrane treści, włącz także Memory Fabric.',
        screenTimelapseOpen: 'Otwórz',
        screenTimelapseSearchPlaceholder: 'Szukaj zawartości wideo...',
        screenTimelapseNoVideos: 'Nie znaleziono filmów timelapse',
        screenTimelapseNoResults: 'Nie znaleziono pasujących klatek',
        screenTimelapseMultipleResults: 'Wyniki wyszukiwania',
        screenTimelapseUnavailable: 'Niedostępne',
        screenTimelapsePlayerUnsupported: 'Ta przeglądarka nie może odtworzyć filmu timelapse',
        screenTimelapseSearchFailed: 'Wyszukiwanie nie powiodło się',
        screenTimelapseLoading: 'Ładowanie timelapse...',
        screenTimelapseSearchAction: 'Szukaj w timelapse',
        screenTimelapsePlay: 'Odtwórz timelapse',
        screenTimelapsePause: 'Wstrzymaj timelapse',
        screenTimelapseEnterFullscreen: 'Przejdź do pełnego ekranu',
        screenTimelapseExitFullscreen: 'Opuść pełny ekran',
        screenTimelapseFullscreenUnsupported: 'Ta przeglądarka nie może przejść do pełnego ekranu',
        screenTimelapsePreviousDay: 'Poprzedni dzień',
        screenTimelapseNextDay: 'Następny dzień',
        screenTimelapseDeleteDay: 'Usuń nagranie z {{date}}',
        screenTimelapseDelete: 'Usuń',
        screenTimelapseDeleteConfirmTitle: 'Usunąć nagranie poklatkowe?',
        screenTimelapseDeleteConfirmDesc:
          'Nagranie z {{date}} zajmuje {{size}}. Tej operacji nie można cofnąć.',
        screenTimelapseDeleteSuccess: 'Usunięto nagranie z {{date}}',
        screenTimelapseDeleteFailed: 'Nie udało się usunąć nagrania z {{date}}',
        endpoint: 'Punkt końcowy',
        apiKey: 'Klucz API',
        regenerate: 'Wygeneruj ponownie',
        copyApiKey: 'Kopiuj klucz API',
        copyEndpoint: 'Kopiuj punkt końcowy',
        showApiKey: 'Pokaż klucz API',
        regenerateConfirmTitle: 'Wygeneruj ponownie klucz API',
        regenerateConfirmDesc:
          'Istniejący klienci MCP przestaną działać, dopóki nie zostaną zaktualizowani nowym kluczem. Czy chcesz kontynuować?',
        okBtn: 'Tak',
        cancelBtn: 'Nie'
      },
      title: 'Ustawienia',
      about: {
        title: 'NanoKVM - informacje',
        information: 'Informacje',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Wersja oprogramowania',
        applicationTip: 'Wersja aplikacji webowej NanoKVM',
        arch: 'Architektura',
        device_number: 'Numer urządzenia',
        image: 'Wersja obrazu',
        imageTip: 'Wersja obrazu systemu NanoKVM',
        deviceKey: 'Klucz urządzenia',
        community: 'Społeczność',
        hostname: 'Nazwa hosta',
        hostnameUpdated: 'Zaktualizowano nazwę hosta. Uruchom ponownie, aby zastosować.',
        ipType: {
          Wired: 'Przewodowy',
          Wireless: 'Bezprzewodowe',
          Other: 'Inne'
        }
      },
      appearance: {
        title: 'Wygląd',
        display: 'Wyświetlacz',
        language: 'Język',
        languageDesc: 'Wybierz język interfejsu',
        languageChangeFailed: 'Nie udało się zmienić języka',
        webTitle: 'Tytuł strony internetowej',
        webTitleDesc: 'Dostosuj tytuł strony internetowej',
        menuBar: {
          title: 'Pasek menu',
          mode: 'Tryb wyświetlania',
          modeDesc: 'Wyświetl pasek menu na ekranie',
          modeOff: 'Wyłączone',
          modeAuto: 'Automatyczne ukrywanie',
          modeAlways: 'Zawsze widoczny',
          icons: 'Ikony podmenu',
          showIcon: 'Pokaż ikonę',
          activateOnConnect: 'Włącz przy połączeniu',
          stopSharing: 'Zakończ udostępnianie',
          iconsDesc: 'Wyświetla ikony podmenu na pasku menu'
        }
      },
      screen: {
        title: 'Ekran',
        videoMode: {
          title: 'Tryb wideo',
          description: 'Wybierz metodę kodowania i transmisji'
        },
        edid: {
          description: 'Wybierz parametry wyświetlania',
          default: 'Domyślne',
          custom: 'Niestandardowe',
          document: 'Dokument',
          powerConfirmTitle: 'Potwierdź przed aktualizacją EDID',
          powerConfirmDescription:
            'Najpierw podłącz zasilanie PD. Aktualizacja EDID może na chwilę odłączyć ekran; jeśli później urządzenie nie wykryje ekranu poprawnie, uruchom je ręcznie ponownie.',
          powerConfirmOk: 'Kontynuuj',
          powerConfirmOkCountdown: 'Kontynuuj ({{seconds}})',
          powerConfirmCancel: 'Anuluj',
          errors: {
            badRequest: 'Nieprawidłowe żądanie EDID',
            invalidFileName: 'Wybierz plik EDID .bin z prawidłową nazwą',
            fileTooLarge: 'Plik EDID jest zbyt duży',
            invalidLength: 'Plik EDID musi mieć 128, 256, 384 lub 512 bajtów',
            invalidHeader: 'Nagłówek EDID jest nieprawidłowy',
            unsupportedVersion: 'Ta wersja EDID nie jest obsługiwana',
            extensionCountMismatch: 'Liczba bloków rozszerzeń EDID nie odpowiada długości pliku',
            invalidChecksum: 'Suma kontrolna EDID jest nieprawidłowa',
            invalidCta: 'Struktura rozszerzenia CTA jest nieprawidłowa',
            currentCannotOverwrite:
              'Nie można nadpisać aktualnie używanego EDID. Najpierw przełącz na inny EDID',
            saveFailed: 'Nie udało się zapisać EDID',
            switchFailed: 'Nie udało się przełączyć EDID',
            uploadFailed: 'Nie udało się przesłać EDID',
            deleteFailed: 'Nie udało się usunąć EDID',
            networkFailed: 'Błąd sieci. Spróbuj ponownie'
          }
        },
        rateControlMode: {
          title: 'Tryb kontroli bitrate',
          description: 'Wybierz metodę kontrolowania szybkości transmisji',
          cbr: 'CBR (stała szybkość transmisji): nadaje priorytet stabilności połączenia. Minimalizuje zacinanie się, ale jakość obrazu może spaść w scenach o dużym ruchu.',
          vbr: 'VBR (zmienna szybkość transmisji): nadaje priorytet klarowności obrazu. Sceny o dużym ruchu mogą powodować skoki przepustowości, co może prowadzić do opóźnień lub zacięć.'
        },
        bitrate: {
          title: 'Szybkość transmisji',
          description: 'Zrównoważ klarowność obrazu z wykorzystaniem przepustowości',
          lower:
            'Niższe wartości zmniejszają opóźnienia w słabych sieciach, ale pogarszają jakość obrazu, potencjalnie powodując rozmycie lub artefakty.',
          higher:
            'Wyższe wartości poprawiają klarowność obrazu, ale zużywają większą przepustowość. Może zwiększyć opóźnienie, jeśli połączenie jest wolne.'
        },
        gop: {
          description: 'Ustawia odstęp pomiędzy pełnymi klatkami kluczowymi',
          lower:
            'Niższe wartości powodują częstsze przesyłanie pełnych klatek. Przyspiesza to usuwanie artefaktów wizualnych, ale zużywa większą przepustowość i zmniejsza szczegółowość obrazu.',
          higher:
            'Wyższe wartości optymalizują aktualizacje przyrostowe. Poprawia to szczegółowość obrazu i oszczędza przepustowość, ale przywracanie artefaktów wizualnych będzie trwało dłużej.'
        },
        quality: {
          title: 'Jakość',
          description: 'Ustaw jakość wyświetlanego obrazu'
        },
        fps: {
          auto: 'Automatycznie',
          description: 'Ustaw częstotliwość wyświetlania klatek',
          tip1: 'Ustawienie wartości 0 włącza tryb automatyczny, synchronizując wyświetlacz z szybkością klatek zdalnego hosta.',
          tip2: 'Obniżenie liczby klatek na sekundę znacznie zmniejsza wykorzystanie przepustowości. Zmniejsz go ręcznie w przypadku niestabilności sieci lub gdy nie jest wymagana duża liczba klatek na sekundę.'
        },
        scale: {
          title: 'Skala',
          description: 'Ustaw współczynnik skalowania wyświetlacza'
        }
      },
      device: {
        title: 'Urządzenie',
        virtualAudioConfirm: {
          title: 'Potwierdź przed przełączeniem wirtualnego dźwięku',
          description:
            'Najpierw podłącz zasilanie PD. Przełączenie wirtualnego dźwięku może na chwilę odłączyć ekran; jeśli później urządzenie nie wykryje ekranu poprawnie, uruchom je ręcznie ponownie.',
          ok: 'Kontynuuj przełączanie',
          okCountdown: 'Kontynuuj przełączanie ({{seconds}})',
          cancel: 'Anuluj'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Skonfiguruj Wi-Fi',
          apMode: 'Tryb AP jest włączony, połącz się z Wi-Fi ręcznie',
          others: 'Inne...',
          connect: 'Dołącz do Wi-Fi',
          connectDesc1: 'Proszę wprowadzić identyfikator sieciowy i hasło',
          connectDesc2: 'Wprowadź hasło, aby dołączyć do tej sieci',
          disconnect: 'Czy na pewno chcesz odłączyć sieć?',
          ssid: 'Imię',
          password: 'Hasło',
          joinBtn: 'Dołącz',
          confirmBtn: 'OK',
          cancelBtn: 'Anuluj',
          preset: {
            title: 'Zapisana sieć Wi-Fi',
            description:
              'Zapisz wcześniej jedną docelową sieć Wi-Fi; urządzenie połączy się z nią automatycznie przy starcie',
            empty: 'Brak',
            addBtn: 'Dodaj ustawienie wstępne',
            removeBtn: 'Usuń ustawienie wstępne',
            saveTitle: 'Zapisz Wi-Fi wcześniej',
            saveDesc:
              'Zapisz docelową sieć teraz; urządzenie połączy się automatycznie, gdy wykryje ją przy starcie',
            saveBtn: 'Zapisz',
            passwordPlaceholder: 'Hasło (pozostaw puste dla sieci otwartych)',
            saveFailed: 'Nie udało się zapisać ustawienia wstępnego Wi-Fi',
            removeFailed: 'Nie udało się usunąć ustawienia wstępnego Wi-Fi',
            removeConfirm: 'Czy na pewno chcesz usunąć zapisaną sieć Wi-Fi?',
            warning:
              'Po zapisaniu urządzenie przy każdym uruchomieniu spróbuje połączyć się z tą siecią Wi-Fi, gdy ją wykryje, co może zakłócać łączenie z innymi sieciami Wi-Fi. Ustawienie wstępne pozostaje aktywne, dopóki go nie zmienisz lub nie usuniesz.'
          }
        },
        capture: {
          title: 'Przechwytywanie wideo',
          description: 'Włącz przechwytywanie wideo'
        },
        ssh: {
          description: 'Włącz SSH zdalny dostęp',
          tip: 'Po włączeniu SSH ustaw silne hasło użytkownika root w terminalu.'
        },
        swap: {
          errors: {
            load: 'Nie można wczytać stanu pamięci wymiany. Spróbuj ponownie za chwilę.',
            update:
              'Nie można zaktualizować ustawienia pamięci wymiany. Sprawdź bieżący stan przed ponowną próbą.'
          },
          title: 'Pamięć swap',
          description: 'Włącz skompresowaną pamięć swap'
        },
        advanced: 'Ustawienia zaawansowane',
        staticIp: {
          title: 'Statyczny adres IP',
          config: 'Konfiguracja',
          tip: 'Aby zapobiec utracie połączenia, zaleca się przeprowadzanie konfiguracji przy fizycznym dostępie do urządzenia.',
          document: 'Sprawdź dokument.',
          enable: 'Włącz statyczny adres IP',
          placeholder: 'Format: IP/CIDR [brama]\nPrzykład: 192.168.1.10/22',
          notEmpty: 'Adres IP nie może być pusty.',
          notValid: 'Format adresu IP jest nieprawidłowy.',
          failed: 'Konfiguracja nie powiodła się.',
          success: 'Konfiguracja zakończona. Sprawdź, czy adres IP jest dostępny.',
          okBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        usbNetwork: {
          title: 'Sieć USB',
          mode: 'Tryb',
          server: 'Serwer',
          serverDesc: 'Przydziel adres IP hostowi zdalnemu za pomocą wbudowanego serwera DHCP.',
          client: 'Klient',
          clientDesc: 'Pobierz adres IP z hosta zdalnego przez DHCP.',
          forwarding: 'Przekazywanie IP',
          forwardingTip:
            'Aby umożliwić hostowi zdalnemu dostęp do Internetu przez NanoKVM, włącz przekazywanie IP.',
          manualIp: 'Ręczny adres IP',
          ipv4Placeholder: 'IPv4/CIDR, np. 10.10.10.1/24',
          invalidIpv4: 'Wprowadź prawidłowy adres IPv4/CIDR.',
          loadFailed: 'Nie można wczytać konfiguracji sieci USB.',
          failed: 'Konfiguracja nie powiodła się.',
          success: 'Konfiguracja zakończona.',
          applyBtn: 'Zastosuj',
          cancelBtn: 'Anuluj'
        },
        usbId: {
          title: 'Tożsamość urządzenia USB',
          config: 'Konfiguracja',
          tip: 'Niestandardowe ID USB zmieniają sposób identyfikacji urządzenia przez hosta. Zastosowanie restartuje gadżet USB i może krótko rozłączyć klawiaturę, mysz lub sieć.',
          enable: 'Włącz niestandardową tożsamość USB',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Wersja urządzenia (bcdDevice)',
          serialnumber: 'Numer seryjny',
          manufacturer: 'Producent',
          product: 'Produkt',
          vidPlaceholder: 'np. 0x359F',
          pidPlaceholder: 'np. 0x3305',
          bcdDevicePlaceholder: 'np. 0x0101 (opcjonalne)',
          serialnumberPlaceholder: 'np. 0123456789ABCDEF (opcjonalne)',
          manufacturerPlaceholder: 'np. sipeed (opcjonalne)',
          productPlaceholder: 'np. NanoKVM-Go (opcjonalne)',
          notEmpty: 'VID i PID nie mogą być puste.',
          notValid: 'Nieprawidłowy format VID/PID. Użyj hex, np. 0x359F.',
          bcdNotValid: 'Nieprawidłowy format bcdDevice. Użyj hex jak 0x0101 lub zostaw puste.',
          failed: 'Konfiguracja nie powiodła się.',
          success: 'Konfiguracja zakończona. Gadżet USB zrestartowany.',
          okBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        diskInquiry: {
          title: 'Nazwa dysku USB',
          config: 'Konfiguruj',
          tip: 'Nazwa wirtualnego pendrive’a widoczna na zdalnym komputerze. Zapis restartuje USB i może krótko odłączyć klawiaturę, mysz lub sieć.',
          enable: 'Użyj własnej nazwy dysku USB',
          placeholder: 'np. sipeed  Flash Drive     1.00',
          notEmpty: 'Nazwa nie może być pusta.',
          notValid: 'Nieprawidłowa nazwa. Do 28 drukowalnych znaków ASCII bez łamania wierszy.',
          failed: 'Konfiguracja nie powiodła się.',
          success: 'Konfiguracja zakończona. Gadżet USB zrestartowany.',
          okBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        diskInquiryCdrom: {
          title: 'Nazwa wirtualnego CD',
          config: 'Konfiguruj',
          tip: 'Nazwa wirtualnego napędu CD/DVD widoczna na zdalnym komputerze. Zapis restartuje USB i może krótko odłączyć klawiaturę, mysz lub sieć.',
          enable: 'Użyj własnej nazwy wirtualnego CD',
          placeholder: 'np. sipeed  Optical Drive   1.00',
          notEmpty: 'Nazwa nie może być pusta.',
          notValid: 'Nieprawidłowa nazwa. Do 28 drukowalnych znaków ASCII bez łamania wierszy.',
          failed: 'Konfiguracja nie powiodła się.',
          success: 'Konfiguracja zakończona. Gadżet USB zrestartowany.',
          okBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        uac2FunctionName: {
          title: 'Nazwa audio USB',
          config: 'Konfiguruj',
          tip: 'Nazwa wirtualnego mikrofonu / urządzenia audio USB widoczna na zdalnym komputerze. Zapis restartuje USB i może krótko odłączyć klawiaturę, mysz lub sieć.',
          enable: 'Użyj własnej nazwy audio USB',
          placeholder: 'np. NanoKVM-Go Audio',
          notEmpty: 'Nazwa nie może być pusta.',
          notValid: 'Nieprawidłowa nazwa. Do 64 znaków drukowalnych bez łamania wierszy.',
          failed: 'Konfiguracja nie powiodła się.',
          success: 'Konfiguracja zakończona. Gadżet USB zrestartowany.',
          okBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        lowPower: {
          title: 'Tryb niskiego zużycia energii',
          description: 'Zmniejsz zużycie energii w stanie bezczynności',
          tips: 'Wskazówki',
          tip1: 'Po włączeniu przechwytywanie wideo zostanie wstrzymane w okresach bezczynności, aby zmniejszyć zużycie energii i wydzielanie ciepła.',
          tip2: 'Wznowienie przechwytywania wideo może wymagać ponownej inicjalizacji, podczas której obraz może być tymczasowo niedostępny.',
          okBtn: 'OK',
          cancelBtn: 'Anuluj'
        },
        mouseJiggler: {
          title: 'Jiggler myszy',
          description: 'Uniemożliwia uśpienie zdalnego hosta',
          disable: 'Wyłącz',
          absolute: 'Tryb absolutny',
          relative: 'Tryb względny'
        },
        fingerRobot: {
          title: 'Robot palcowy',
          description: 'Włącz sterowanie robotem palcowym'
        },
        mdns: {
          description: 'Włącz usługę wykrywania mDNS'
        },
        hidOnly: 'HID – tylko tryb',
        imageMounted: 'Obraz zamontowany',
        disk: 'Dysk wirtualny',
        diskDesc: 'Zamontuj wirtualny dysk U na zdalnym hoście',
        sdCard: {
          warning: 'Inicjalizuj dysk wirtualny',
          tip1: 'Przy pierwszym włączeniu dysk wirtualny zostanie zainicjalizowany, a istniejące na urządzeniu pliki obrazów i dane dysku wirtualnego zostaną usunięte. Tej operacji nie można cofnąć. Najpierw wykonaj kopię zapasową ważnych plików.',
          tip2: 'Kontynuować inicjalizację?',
          capacity: 'Pojemność',
          disable: 'Wyłącz',
          disableWarning: 'Wyłączyć dysk wirtualny?',
          disableTip:
            'Wyłączenie usunie wszystkie pliki z dysku wirtualnego i zwolni używaną przez niego pamięć urządzenia. Przy następnym włączeniu trzeba ponownie wybrać pojemność i zainicjalizować dysk. Tej operacji nie można cofnąć.'
        },
        network: 'Sieć wirtualna',
        networkDesc: 'Zamontuj wirtualną kartę sieciową na zdalnym hoście',
        mic: 'Wirtualny dźwięk',
        micDesc: 'Wybierz tryb wirtualnego dźwięku USB dla zdalnego hosta',
        virtualAudioMode: {
          off: 'Wyłączone',
          mic: 'Tylko mikrofon',
          duplex: 'Mikrofon + głośnik'
        },
        speaker: 'Wirtualny głośnik',
        speakerDesc:
          'Po włączeniu zastępuje domyślne wyjście audio DP, zapewniając obsługę połączeń zdalnych z większą zgodnością. Po wyłączeniu można przekazywać tylko dźwięk multimediów, bez dźwięku połączeń.',
        micTip:
          'Ze względu na ograniczenia punktów końcowych USB włączenie wirtualnego mikrofonu spowoduje wyłączenie dodatkowego ekranu USB.\nZaleca się wyłączenie tej funkcji podczas operacji BIOS-u, aby zapewnić optymalną kompatybilność.',
        reboot: 'Uruchom ponownie',
        rebootDesc: 'Czy na pewno chcesz ponownie uruchomić NanoKVM?',
        rebootToFlashing: 'Uruchom ponownie w trybie flashowania',
        rebootToFlashingButton: 'Uruchom ponownie',
        rebootToFlashingTitle: 'Uruchomić ponownie w trybie flashowania?',
        rebootToFlashingDesc:
          'NanoKVM uruchomi się ponownie i przejdzie w tryb flashowania przez USB. Upewnij się, że interfejs USB używany do flashowania jest podłączony, a narzędzie do flashowania jest gotowe.',
        datetime: {
          timezone: 'Strefa czasowa',
          datetime: 'Data i godzina',
          dateAndTime: 'Data i godzina',
          synchronize: 'Synchronizuj czas',
          lastSynchronization: 'Ostatnia synchronizacja: ',
          notSynchronized: 'Czas systemowy nie jest jeszcze zsynchronizowany',
          syncNow: 'Zsynchronizuj teraz',
          manualSet: 'Ustaw ręcznie',
          manualSetTitle: 'Ręczne ustawianie czasu',
          manualSetDescription: 'Po zmianie czasu musisz zalogować się ponownie.',
          useBrowserTime: 'Użyj czasu przeglądarki',
          setTime: 'Ustaw czas',
          invalidTime: 'Wpisz prawidłową datę i godzinę.',
          setFailed: 'Nie można ustawić czasu systemowego. Spróbuj ponownie.',
          setSucceeded: 'Czas systemowy został zaktualizowany. Zaloguj się ponownie.'
        },
        dialBackground: {
          errors: {
            load: 'Nie można wczytać obrazów tła tarczy. Spróbuj ponownie za chwilę.',
            upload:
              'Nie można przesłać obrazu tła. Sprawdź, czy plik jest prawidłowym obrazem BMP, i spróbuj ponownie.',
            switch: 'Nie można zmienić tła tarczy. Odśwież listę obrazów i spróbuj ponownie.'
          },
          title: 'Tło pokrętła',
          description: 'Wybierz obraz BMP z /kvmcomm/ui/bg/ jako tło pokrętła',
          upload: 'Prześlij',
          loading: 'Ładowanie...',
          empty: 'Nie znaleziono obrazów BMP',
          current: 'Bieżące',
          onlyBmp: 'Obsługiwane są tylko obrazy BMP',
          uploaded: 'Przesyłanie zakończone',
          switched: 'Tło przełączone'
        },
        okBtn: 'Tak',
        cancelBtn: 'Nie'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Uruchomić ponownie Tailscale?',
        stop: 'Zatrzymać Tailscale?',
        stopDesc: 'Wyloguj Tailscale i wyłącz automatyczne uruchamianie przy starcie systemu.',
        loading: 'Ładowanie...',
        notInstall: 'Nie znaleziono Tailscale! Proszę zainstalować.',
        install: 'Instaluj',
        installing: 'Instalowanie',
        failed: 'Instalowanie nie powiodło się',
        retry: 'Odśwież stronę i spróbuj ponownie, albo spróbuj zainstalować manualnie.',
        download: 'Pobierz',
        package: 'pakiet instalacyjny',
        unzip: 'i wypakuj pliki',
        upTailscale: 'Prześlij tailscale do NanoKVM w katalogu /usr/bin/',
        upTailscaled: 'Prześlij tailscaled do NanoKVM w katalogu /usr/sbin/',
        refresh: 'Odśwież obecną stronę',
        notRunning: 'Tailscale nie działa. Rozpocznij, aby kontynuować.',
        run: 'Uruchom',
        notLogin:
          'Urządzenie nie zostało jeszcze powiązane. Zaloguj się i powiąż to urządzenie ze swoim kontem.',
        urlPeriod: 'Ten URL jest ważny przez 10 minut',
        login: 'Zaloguj',
        loginSuccess: 'Zalogowanie pomyślne',
        enable: 'Włącz Tailscale',
        exitNode: 'Węzeł wyjściowy',
        exitNodeDesc: 'Ogłaszaj to urządzenie jako wyjście internetowe dla sieci tailnet.',
        subnetRoute: 'Trasa podsieci',
        subnetRouteDesc: 'Ogłaszaj bezpośrednio połączoną podsieć LAN {{subnet}} w sieci tailnet.',
        subnetUnavailable: 'Brak dostępnej bezpośrednio połączonej podsieci LAN.',
        routingUnavailable: 'Ustawienia routingu Tailscale są niedostępne na tym urządzeniu.',
        pendingApproval:
          'Po włączeniu zatwierdź ogłaszaną trasę w konsoli administracyjnej Tailscale.',
        routingUpdateFailed: 'Nie udało się zaktualizować ustawień routingu Tailscale.',
        deviceName: 'Nazwa urządzenia',
        deviceIP: 'Adres IP urządzenia',
        account: 'Konto',
        logout: 'Wyloguj',
        logoutDesc: 'Czy na pewno chcesz się wylogować?',
        uninstall: 'Odinstaluj Tailscale',
        uninstallDesc: 'Czy na pewno chcesz odinstalować Tailscale?',
        okBtn: 'Tak',
        cancelBtn: 'Nie'
      },
      update: {
        title: 'Sprawdź aktualizacje',
        queryFailed: 'Uzyskanie wersji nie powiodło się',
        updateFailed: 'Aktualizacja nie powiodła się. Spróbuj ponownie.',
        isLatest: 'Oprogramowanie jest aktualne.',
        available: 'Aktualizacja jest dostępna. Czy na pewno chcesz dokonać aktualizacji?',
        updating: 'Aktualizacja rozpoczęta. Proszę czekać...',
        confirm: 'Potwierdź',
        upgrade_tip_nano:
          'Dostępna jest nowa wersja NanoKVM. Czy na pewno chcesz teraz dokonać aktualizacji?',
        upgrade_tip_firmware:
          'Dostępna jest nowa wersja oprogramowania sprzętowego. Czy na pewno chcesz teraz dokonać aktualizacji?',
        cancel: 'Anuluj',
        preview: 'Podgląd aktualizacji',
        previewDesc: 'Uzyskaj wcześniejszy dostęp do nowych funkcji i ulepszeń',
        previewTip:
          'Należy pamiętać, że wersje poglądowe mogą zawierać błędy lub niekompletną funkcjonalność!',
        download: 'Pobierz',
        install: 'Zainstaluj',
        changelog: 'Dziennik zmian',
        restart: 'Uruchom ponownie'
      },
      account: {
        title: 'Konto',
        webAccount: 'Nazwa konta webowego',
        password: 'Hasło',
        twoFactor: 'Uwierzytelnianie dwuskładnikowe',
        enableTwoFactor: 'Włącz',
        enableTwoFactorDesc:
          'Wpisz bieżące hasło, aby rozpocząć konfigurację uwierzytelniania dwuskładnikowego.',
        disableTwoFactor: 'Wyłącz',
        recoveryCodes: 'Kody odzyskiwania',
        enableRecoveryCodes: 'Włącz',
        updateRecoveryCodes: 'Zaktualizuj',
        recoveryCodesTooltipDisabled:
          'Kody odzyskiwania są zapasowymi danymi uwierzytelniania dwuskładnikowego. Możesz je wygenerować po włączeniu uwierzytelniania dwuskładnikowego.',
        recoveryCodesTooltipEnabled:
          'Kody odzyskiwania są ważne tylko wtedy, gdy uwierzytelnianie dwuskładnikowe jest włączone. Wyłączenie go unieważnia wszystkie kody odzyskiwania; dopóki pozostaje włączone, można je tylko aktualizować.',
        recoveryCodesRegenerateDesc:
          'Wpisz bieżące hasło oraz albo kod uwierzytelniający, albo nieużyty kod odzyskiwania. Aktualizacja kodów odzyskiwania unieważni wszystkie istniejące kody odzyskiwania.',
        recoveryCodesVerifyFailed:
          'Nie udało się zweryfikować hasła i kodu uwierzytelniającego lub kodu odzyskiwania.',
        recoveryCodesSavedTitle: 'Zapisz teraz te kody odzyskiwania',
        recoveryCodesSavedDesc:
          'Kody odzyskiwania można wyświetlić tylko raz. Zapisz je w bezpiecznym miejscu przed opuszczeniem tej strony.',
        enableRecoveryCodesDesc:
          'Wpisz bieżące hasło i kod uwierzytelniający, aby wygenerować kody odzyskiwania. Kody odzyskiwania można wyświetlić tylko raz, więc zapisz je przed opuszczeniem tej strony.',
        copyRecoveryCodes: 'Kopiuj',
        downloadRecoveryCodes: 'Pobierz',
        recoveryCodesCopyFailed: 'Nie udało się skopiować kodów odzyskiwania.',
        done: 'Gotowe',
        currentPassword: 'Bieżące hasło',
        authCode: 'Kod uwierzytelniający',
        authCodeOrRecoveryCode: 'Kod uwierzytelniający lub kod odzyskiwania',
        authRateLimited: 'Zbyt wiele nieudanych prób. Spróbuj ponownie za {{seconds}} s.',
        verifyPasswordChange: 'Zweryfikuj zmianę hasła',
        verifyPasswordChangeDesc:
          'Przed zmianą hasła wpisz bieżące hasło oraz albo kod uwierzytelniający, albo nieużyty kod odzyskiwania.',
        verifyPasswordChangeFailed:
          'Nie udało się zweryfikować hasła i kodu uwierzytelniającego lub kodu odzyskiwania.',
        verifyAndEnable: 'Zweryfikuj i włącz',
        scanTwoFactorQRCode:
          'Zeskanuj ten kod QR aplikacją uwierzytelniającą, a następnie wpisz kod.',
        showSetupKey: 'Pokaż klucz konfiguracji',
        hideSetupKey: 'Ukryj klucz konfiguracji',
        setupKey: 'Klucz konfiguracji',
        twoFactorSetupFailed:
          'Nie udało się rozpocząć konfiguracji uwierzytelniania dwuskładnikowego.',
        twoFactorVerifyFailed: 'Nie udało się zweryfikować kodu uwierzytelniającego.',
        twoFactorDisableFailed: 'Nie udało się wyłączyć uwierzytelniania dwuskładnikowego.',
        disableTwoFactorDesc:
          'Wpisz bieżące hasło oraz albo kod uwierzytelniający, albo nieużyty kod odzyskiwania. Wyłączenie uwierzytelniania dwuskładnikowego unieważni wszystkie kody odzyskiwania.',
        updateBtn: 'Zmień',
        logoutBtn: 'Wyloguj się',
        logoutDesc: 'Czy na pewno chcesz się wylogować?',
        okBtn: 'Tak',
        cancelBtn: 'Nie',
        twoFactorProtection: {
          title: 'Zaawansowana ochrona 2FA',
          description:
            'Wybierz poufne działania wymagające drugiego składnika, gdy ochrona będzie egzekwowana. Przełącznik „Bezpośrednie logowanie przez WWW” działa odwrotnie: umożliwia zalogowanie się wyłącznie za pomocą kodu uwierzytelniającego lub nieużytego kodu odzyskiwania.',
          login: 'Bezpośrednie logowanie przez WWW',
          webTerminal: 'Terminal WWW',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Klucz API MCP',
          targetPower: 'Zasilanie komputera docelowego',
          nanoKVMRestart: 'Restart NanoKVM',
          deleteImage: 'Usuń obraz',
          scripts: 'Operacje na skryptach',
          screenTimelapse: 'Timelapse ekranu',
          loginScope:
            'Umożliwia logowanie przez WWW wyłącznie za pomocą kodu uwierzytelniającego lub niewykorzystanego kodu odzyskiwania zamiast nazwy użytkownika i hasła. Zmniejsza to bezpieczeństwo logowania.',
          webTerminalScope: 'Otwarcie terminala WWW.',
          sshScope: 'Zmiana stanu usługi SSH.',
          tailscaleScope: 'Instalacja, połączenie lub zmiana Tailscale.',
          frp: 'FRP',
          frpScope: 'Instalacja, połączenie lub zmiana FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Instalacja, połączenie lub zmiana Cloudflare Tunnel.',
          zerotierScope: 'Instalacja, połączenie lub zmiana ZeroTier.',
          netbirdScope: 'Instalacja, połączenie lub zmiana NetBird.',
          mcpScope:
            'Wyświetlenie, skopiowanie lub ponowne wygenerowanie klucza API MCP oraz włączenie lub wyłączenie MCP.',
          targetPowerScope:
            'Wyłączanie lub ponowne uruchamianie komputera docelowego fizycznym przyciskiem zasilania.',
          nanoKVMRestartScope: 'Ponowne uruchamianie samego NanoKVM.',
          deleteImageScope: 'Trwałe usunięcie zapisanego obrazu ISO lub IMG.',
          scriptsScope: 'Uruchomienie, przesłanie lub usunięcie skryptów.',
          screenTimelapseScope:
            'Włączanie nagrywania timelapse ekranu, przeglądanie i usuwanie nagrań.',
          authorizeTitle: 'Autoryzuj zmiany ochrony',
          authorizeDesc:
            'Wprowadź kod uwierzytelniający lub nieużyty kod odzyskiwania. Autoryzacja trwa pięć minut.',
          authorizeFailed: 'Nie można zweryfikować kodu uwierzytelniającego lub kodu odzyskiwania.',
          loadFailed: 'Nie można wczytać ustawień ochrony.',
          updateFailed: 'Nie można zapisać ustawień ochrony.',
          retry: 'Spróbuj ponownie',
          disableScriptsWarningTitle: 'Wyłączyć ochronę skryptów?',
          disableScriptsWarningDesc:
            'Bez ochrony złośliwy skrypt może uzyskać podwyższony dostęp do tego urządzenia i ominąć lub usunąć ochronę 2FA.',
          cancelDisabling: 'Anuluj',
          continueDisabling: 'Kontynuuj wyłączanie',
          continueDisablingCountdown: 'Kontynuuj wyłączanie ({{seconds}} s)'
        },
        twoFactorGuide: {
          title: 'Przewodnik po uwierzytelnianiu dwuskładnikowym',
          scope: 'Uwierzytelnianie dwuskładnikowe dodaje dodatkową warstwę zabezpieczeń do konta.',
          whatTitle: 'Czym jest 2FA?',
          whatDesc:
            'Uwierzytelnianie dwuskładnikowe (2FA) chroni konto za pomocą kodu czasowego z aplikacji uwierzytelniającej. Domyślnie kod jest wymagany oprócz hasła, więc nawet jeśli hasło zostanie przejęte, konto pozostaje chronione. Jeśli włączono bezpośrednie logowanie przez WWW, wystarczy sam kod, aby się zalogować.',
          appsTitle: 'Zalecane aplikacje',
          appsDesc:
            'Możesz użyć Google Authenticator lub Microsoft Authenticator. Obie aplikacje są bezpłatne i działają na iOS oraz Androidzie.',
          setupTitle: 'Jak skonfigurować',
          setupDesc:
            'Otwórz aplikację uwierzytelniającą, zeskanuj kod QR wyświetlony podczas konfiguracji albo ręcznie wpisz klucz konfiguracji. Następnie wpisz 6-cyfrowy kod z aplikacji, aby zakończyć włączanie 2FA.',
          recoveryTitle: 'Kody odzyskiwania',
          recoveryDesc:
            'Kody odzyskiwania to jednorazowe kody zapasowe. Jeśli utracisz dostęp do aplikacji uwierzytelniającej, możesz użyć kodu odzyskiwania do logowania. Zapisz je w bezpiecznym miejscu od razu po wygenerowaniu.',
          logoutNotice:
            'Włączenie 2FA wyloguje wszystkie aktywne sesje, w tym tę. Na każdym urządzeniu trzeba będzie zalogować się ponownie.'
        }
      }
    },
    error: {
      title: 'Wystąpił problem',
      refresh: 'Odśwież'
    },
    pictureInPicture: {
      title: 'Obraz w obrazie',
      enter: 'Włącz obraz w obrazie',
      exit: 'Wyłącz obraz w obrazie',
      error:
        'Nie można uruchomić trybu obraz w obrazie. Poczekaj, aż wideo się załaduje, i spróbuj ponownie.'
    },
    fullscreen: {
      toggle: 'Przełącz tryb pełnoekranowy'
    },
    menu: {
      collapse: 'Zwiń menu',
      expand: 'Rozwiń Menu'
    }
  }
};

export default pl;
