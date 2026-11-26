const id = {
  translation: {
    storage: {
      cards: { mount: 'Pasang disk', share: 'Berbagi berkas' },
      title: 'Gambar dan berbagi file',
      empty: 'Belum ada gambar',
      notAttached: 'Tidak ada yang terpasang',
      attached: 'Terpasang: {{name}}',
      eject: 'Keluarkan',
      mount: 'Pasang',
      mountMode: 'Mode pemasangan',
      mountModeDisk: 'Penyimpanan massal',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Buka pengelola file',
      refresh: 'Segarkan',
      delete: 'Hapus',
      deleteTitle: 'Hapus media',
      deleteConfirm: 'Hapus {{name}}? File di dalamnya akan hilang.',
      downloadImage: 'Unduh gambar',
      typePartition: 'File bersama',
      volumeLabel: 'Disk yang dapat dilepas',
      groups: {
        share: 'Berbagi file',
        images: 'Gambar'
      },
      failed: 'Operasi gagal',
      hidOnly: 'Mode hanya HID: disk virtual tidak tersedia',
      disabled: 'Disk virtual dinonaktifkan (Pengaturan - Perangkat)',
      force: {
        title: 'Ambil kembali media dengan paksa?',
        desc: 'Komputer target masih menggunakan media ini. Mengambilnya kembali akan menghitung ulang perangkat USB dan dapat kehilangan apa pun yang belum ditulisnya ke disk.',
        menuHint:
          'Komputer target sedang menggunakan media ini. Buka pengelola file untuk mengambilnya kembali dengan paksa.'
      },
      tips: {
        title: 'Cara mengunggah',
        usb1: 'Hubungkan NanoKVM ke komputer Anda melalui USB.',
        usb2: 'Di halaman ini, pasang berkas bersama dalam mode penyimpanan massal.',
        usb3: 'Buka disk virtual di komputer Anda dan salin file gambar ke direktori root disk virtual.',
        usb4: 'Keluarkan disk setelah penyalinan selesai: image kemudian muncul di daftar.',
        scp1: 'Pastikan NanoKVM dan komputer Anda berada di jaringan lokal yang sama.',
        scp2: 'Buka terminal di komputer Anda dan gunakan perintah SCP untuk mengunggah file gambar ke direktori /data di NanoKVM.',
        scp3: 'Contoh: scp jalur-gambar-anda root@ip-nanokvm-anda:/data'
      },
      job: {
        running: 'Memproses {{name}}'
      },
      enableDisk: {
        title: 'Aktifkan disk virtual?',
        desc: 'Untuk memasang media diperlukan disk virtual, yang saat ini dimatikan. Aktifkan sekarang? Papan ketik dan tetikus komputer target akan terputus sebentar.',
        menuHint: 'Disk virtual dinonaktifkan. Aktifkan di Pengaturan - Perangkat.'
      },
      add: {
        title: 'Tambah media',
        upload: 'Unggah ISO atau IMG',
        fromUrl: 'Unduh dari URL',
        url: 'URL gambar',
        urlRequired: 'Masukkan URL',
        onlyImages: 'Hanya file .iso dan .img yang dapat dipasang',
        dropHint: 'Seret ISO atau IMG ke sini, atau klik untuk memilih file',
        cdromOnlyIso: 'CD-ROM hanya dapat berisi citra ISO 9660 / UDF',
        uploadFailed: 'Unggahan gagal',
        downloadFailed: 'Unduhan gagal',
        uploading: 'Mengunggah {{name}}',
        downloading: 'Mengunduh {{name}}',
        done: '{{name}} siap',
        checksumFailed: 'Image tidak cocok dengan checksum SHA-256.',
        blocked:
          'Komputer target sedang menggunakan disk. Keluarkan drive USB virtual dengan aman di komputer target, lalu matikan tombol Disk Virtual di Pengaturan > Perangkat.'
      },
      files: {
        reasonNotInitialized:
          'Drive USB virtual belum diinisialisasi. Inisialisasi di Pengaturan > Perangkat sebelum menggunakan berbagi file.',
        selectMedium: 'Pilih media untuk melihat file di dalamnya.',
        reasonAttached: 'Media ini terpasang ke komputer target.',
        reasonAttachedHint:
          'Keluarkan drive USB virtual dengan aman di komputer target, lalu matikan tombol Disk Virtual di Pengaturan > Perangkat.',
        reasonNotBrowsable:
          'Media ini berisi sistem berkas yang tidak dapat dibuka pengelola file.',
        reasonNoLoop: 'Perangkat ini tidak dapat memasang image disk (tanpa dukungan loop).',
        reasonISOImage:
          'Citra ISO 9660 tidak dapat dijelajahi di sini. Citra tetap dapat dipasang ke komputer target.',
        empty: 'Folder ini kosong',
        noMatch: 'Tidak ada file yang cocok dengan pencarian.',
        free: '{{size}} kosong',
        search: 'Cari di folder ini',
        sort: {
          modified: 'Diubah',
          name: 'Nama',
          size: 'Ukuran',
          type: 'Jenis',
          asc: 'Naik',
          desc: 'Turun'
        },
        view: {
          list: 'Tampilan daftar',
          grid: 'Tampilan ikon'
        },
        folder: 'Folder',
        itemCount: 'Jumlah item: {{count}}',
        itemCount_one: '{{count}} item',
        itemCount_other: '{{count}} item',
        dropHint: 'Letakkan berkas di sini untuk mengunggahnya ke folder ini',
        up: 'Ke folder induk',
        uploadButton: 'Unggah',
        newFolder: 'Folder baru',
        newFolderPlaceholder: 'Nama folder',
        create: 'Buat',
        upload: 'Unggah file ke media',
        deleteFolderTitle: 'Hapus folder',
        deleteFolderConfirm:
          'Hapus folder {{name}} beserta seluruh isinya? Tindakan ini tidak dapat dibatalkan.',
        uploading: 'Mengunggah…',
        uploadFailed: 'Unggahan gagal',
        download: 'Unduh',
        delete: 'Hapus',
        deleteTitle: 'Hapus file',
        deleteConfirm: 'Hapus {{name}} dari media?'
      },
      yes: 'Ya',
      no: 'Tidak'
    },
    head: {
      storage: 'Gambar dan berbagi file',
      desktop: 'Desktop jarak jauh',
      login: 'Masuk',
      changePassword: 'Ubah Sandi',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Tidak dapat menyalin ke papan klip. Izinkan browser mengakses papan klip, lalu coba lagi.'
    },
    auth: {
      login: 'Masuk',
      language: 'Bahasa',
      placeholderUsername: 'Silahkan masukkan username',
      placeholderPassword: 'Silahkan masukkan password',
      placeholderPassword2: 'Silahkan masukkan password lagi',
      noEmptyUsername: 'nama user tidak boleh kosong',
      noEmptyPassword: 'sandi  tidak boleh kosong',
      noAccount:
        'Gagal mendapatkan informasi user, silahkan segarkan halaman atau atur ulang sandi',
      invalidUser: 'Nama pengguna atau kata sandi tidak valid',
      error: 'terjadi kesalahan tak terduga',
      twoFactorTitle: 'Verifikasi Diperlukan',
      twoFactorDesc:
        'Masukkan kode autentikasi atau kode pemulihan yang belum digunakan untuk login.',
      sensitiveActionDesc:
        'Masukkan kode autentikasi atau kode pemulihan yang belum digunakan untuk melanjutkan.',
      placeholderAuthCode: 'Kode autentikasi',
      noEmptyAuthCode: 'Kode autentikasi wajib diisi',
      verify: 'Verifikasi',
      backToLogin: 'Kembali ke login',
      verificationFailed: 'Verifikasi gagal',
      twoFactorEnabledLogout: {
        message:
          'Autentikasi dua faktor telah diaktifkan. Demi keamanan, semua sesi telah logout. Silakan login kembali.'
      },
      rateLimited: 'Terlalu banyak percobaan gagal. Coba lagi dalam {{seconds}} dtk.',
      recoveryCodeUsed: 'Kode pemulihan yang digunakan untuk login ini telah dibatalkan.',
      changePassword: 'Ganti Sandi',
      changePasswordDesc: 'Untuk keamanan perangkat Anda, silakan ubah kata sandi masuk web.',
      passwordVerificationRequired:
        'Kembali ke halaman pengaturan akun dan verifikasi kembali perubahan kata sandi ini.',
      passwordVerificationExpired:
        'Verifikasi perubahan kata sandi telah kedaluwarsa. Kembali ke halaman pengaturan akun dan verifikasi lagi.',
      returnToSettings: 'Kembali ke pengaturan',
      differentPassword: 'sandi tidak sesuai',
      illegalUsername: 'ada karakter ilegal pada nama user',
      illegalPassword: 'ada karakter ilegal pada sandi',
      forgetPassword: 'Lupa Sandi',
      ok: 'Oke',
      cancel: 'Batalkan',
      savePassword: 'Simpan kata sandi',
      loginButtonText: 'Masuk',
      tips: {
        reset1: 'Akun web bawaan:',
        reset2: 'Akun SSH bawaan:',
        reset3: 'Untuk mengatur ulang kata sandi, lihat ',
        changeReset: 'Untuk mengatur ulang kata sandi login web, lihat ',
        resetLink: 'Atur ulang'
      }
    },
    wifi: {
      errors: {
        connect:
          'Tidak dapat terhubung ke Wi-Fi. Periksa nama jaringan, kata sandi, dan status Wi-Fi perangkat, lalu coba lagi.'
      },
      title: 'Wi-Fi',
      description: 'Konfigurasi Wi-Fi untuk NanoKVM',
      submitted: {
        title: 'Tunggu sebentar — NanoKVM sedang menghubungkan ke Wi-Fi Anda',
        tipsTitle: 'Selanjutnya',
        tipSwitch: 'Sekarang Anda dapat mengembalikan ponsel ke Wi-Fi biasa.',
        tipCheck:
          'Sebentar lagi, lihat alamat IP baru di layar NanoKVM, lalu buka di browser Anda.',
        tipRetry:
          'Jika setelah beberapa saat tidak ada perubahan, periksa nama jaringan dan kata sandi, lalu coba lagi.'
      },
      invalidMode:
        'Mode saat ini tidak mendukung pengaturan jaringan. Silakan buka perangkat Anda dan aktifkan mode konfigurasi Wi-Fi.',
      confirmBtn: 'Oke',
      finishBtn: 'Selesai',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Kata sandi',
      steps: {
        settings: 'Pengaturan',
        wifi: 'Wi-Fi',
        enable: 'Aktifkan'
      },
      scan: {
        title: 'Pilih jaringan Wi-Fi',
        description: 'Pilih jaringan terdekat untuk melanjutkan.',
        loading: 'Mencari jaringan Wi-Fi terdekat…',
        empty: 'Tidak ada jaringan Wi-Fi terdekat yang ditemukan.',
        failed: 'Pemindaian gagal. Masukkan detail jaringan secara manual.',
        retryBtn: 'Pindai lagi',
        manualBtn: 'Input manual',
        backBtn: 'Kembali ke jaringan terdekat',
        connectTitle: 'Hubungkan ke {{ssid}}',
        connecting: 'Menghubungkan…',
        viewSimple: 'Beralih ke tampilan sederhana',
        viewDetailed: 'Beralih ke tampilan detail'
      },
      ap: {
        authTitle: 'Otentikasi Diperlukan',
        authDescription: 'Silakan masukkan kata sandi AP untuk melanjutkan',
        authFailed: 'Kata sandi AP tidak valid',
        passPlaceholder: 'AP kata sandi',
        verifyBtn: 'Verifikasi'
      },
      qr: {
        cameraBtn: 'Ambil foto',
        pickBtn: 'Pilih tangkapan layar',
        scanHint:
          'Ambil foto kode QR Wi-Fi, atau pilih tangkapan layarnya, seperti yang ditampilkan ponsel saat membagikan Wi-Fi-nya.',
        decoding: 'Membaca kode QR…',
        notFound:
          'Kode QR tidak ditemukan pada foto ini. Posisikan kode di tengah dan fokuskan, lalu coba lagi.',
        invalid: 'Kode QR ini tidak berisi detail Wi-Fi yang dapat digunakan.',
        pickerUnavailable:
          'Tidak ada yang terbuka? Jendela masuk ini mungkin memblokir kamera dan foto. Sebagai gantinya, buka alamat ini di browser ponsel Anda:',
        confirmTitle: 'Hubungkan NanoKVM ke {{ssid}}?',
        confirmDescription:
          'Detail jaringan ini berasal dari kode QR. Periksa sebelum menghubungkan.',
        openNetwork: 'Jaringan ini tidak memiliki kata sandi.',
        connectBtn: 'Hubungkan',
        otherBtn: 'Pilih jaringan lain'
      }
    },
    notification: {
      browser: {
        title: 'Rekomendasi Peramban',
        description:
          'Untuk pengalaman pengguna terbaik, sebaiknya gunakan browser Chrome. Meskipun kami mengupayakan kompatibilitas di semua browser utama, beberapa fitur lanjutan mungkin tidak berfungsi sebagaimana mestinya di luar Chrome.'
      },
      btnOk: 'Oke'
    },
    screen: {
      title: 'Layar',
      video: 'Mode Video',
      quality: 'Kualitas',
      qualityAuto: 'Otomatis',
      qualityLossless: 'Tanpa Kehilangan',
      qualityHigh: 'Tinggi',
      qualityMedium: 'Sedang',
      qualityLow: 'Rendah',
      scale: 'Skala',
      noSignal: 'Tidak ada sinyal video',
      captureDisabled:
        'Pengambilan video dinonaktifkan. Aktifkan di "Pengaturan > Perangkat > Pengambilan Video".',
      inconsistentVideoMode: 'Pemutaran dijeda. Mode video lain sedang diputar.',
      bandwidthWarning:
        'Bandwidth jaringan mungkin tidak cukup untuk pengaturan video saat ini. Coba turunkan bitrate atau frame rate.',
      webrtcConnectionFailed:
        'Koneksi WebRTC tidak dapat dibuat. Silakan beralih ke mode video Direct.',
      webrtcUnavailable: {
        title: 'WebRTC tidak tersedia',
        description: 'Aktifkan WebRTC di browser, atau beralih ke mode video Direct.'
      },
      upload: 'Unggah',
      advanced: 'Lanjutan',
      loading: 'Memuat'
    },
    volume: {
      title: 'Volume'
    },
    microphone: {
      title: 'Mikrofon',
      notMounted: 'Mikrofon Tidak Terpasang',
      tip1: 'Mikrofon virtual tidak dipasang pada host jarak jauh. Apakah Anda ingin memasang dan mengaktifkannya sekarang?',
      tip2: 'Anda juga dapat mengaturnya nanti di "Pengaturan > Perangkat > Audio Virtual".',
      confirm: 'Aktifkan',
      cancel: 'Batal',
      noDevice:
        'Tidak ada mikrofon yang terdeteksi pada perangkat ini. Hubungkan mikrofon lalu coba lagi.',
      permissionDenied: 'Akses mikrofon ditolak. Izinkan akses mikrofon di pengaturan browser.',
      notReadable: 'Mikrofon sedang digunakan aplikasi lain atau tidak dapat dibaca.',
      overconstrained: 'Mikrofon tidak mendukung pengaturan audio yang diperlukan.',
      securityBlocked:
        'Akses mikrofon diblokir pada halaman ini. Periksa pengaturan keamanan browser.',
      aborted: 'Proses memulai mikrofon dibatalkan. Silakan coba lagi.',
      unsupported:
        'Browser atau metode koneksi ini tidak mendukung pengambilan audio dari mikrofon.',
      enableFailed: 'Gagal mengaktifkan mikrofon. Silakan coba lagi.',
      disconnected: 'Mikrofon terputus sehingga transmisi mikrofon telah dinonaktifkan.'
    },
    power: {
      title: 'Daya',
      showConfirm: 'Konfirmasi',
      showConfirmTip: 'Operasi daya memerlukan konfirmasi tambahan',
      powerShort: 'Daya (tekan sebentar)',
      powerLong: 'Daya (tekan lama)',
      powerConfirm: 'Jalankan operasi daya?',
      okBtn: 'Ya',
      cancelBtn: 'Tidak'
    },
    keyboard: {
      errors: {
        paste:
          'Tidak dapat mengirim teks ke perangkat jarak jauh. Periksa koneksi KVM, lalu coba lagi.'
      },
      title: 'Papan ketik',
      paste: 'Tempel',
      tips: 'Hanya huruf dan simbol keyboard standar yang didukung',
      placeholder: 'Silahkan isi',
      submit: 'Kirimkan',
      virtual: 'Papan ketik',
      swapCommandCtrl: 'Tukar Cmd dan Ctrl',
      shortcut: {
        title: 'Pintasan',
        custom: 'Kustom',
        capture: 'Klik di sini untuk mengambil pintasan',
        clear: 'Hapus',
        save: 'Simpan',
        captureTips:
          'Menangkap tombol tingkat sistem (seperti tombol Windows) memerlukan izin layar penuh.',
        enterFullScreen: 'Beralih ke mode layar penuh.'
      },
      leaderKey: {
        title: 'Tombol Leader',
        desc: 'Lewati batasan browser dan kirim pintasan sistem langsung ke host jarak jauh.',
        howToUse: 'Cara Menggunakan',
        simultaneous: {
          title: 'Mode Simultan',
          desc1: 'Tekan dan tahan tombol Leader, lalu tekan pintasan.',
          desc2: 'Intuitif, tetapi mungkin bertentangan dengan pintasan sistem.'
        },
        sequential: {
          title: 'Mode Berurutan',
          desc1:
            'Tekan tombol Leader → tekan pintasan secara berurutan → tekan tombol Leader lagi.',
          desc2: 'Memerlukan lebih banyak langkah, namun sepenuhnya menghindari konflik sistem.'
        },
        enable: 'Aktifkan tombol Leader',
        tip: 'Saat ditetapkan sebagai tombol Leader, tombol ini hanya berfungsi sebagai pemicu pintasan dan kehilangan perilaku defaultnya.',
        placeholder: 'Tekan tombol Leader',
        shiftRight: 'Shift kanan',
        ctrlRight: 'Ctrl kanan',
        metaRight: 'Win kanan',
        submit: 'Kirimkan',
        recorder: {
          rec: 'REKAM',
          activate: 'Aktifkan tombol',
          input: 'Silakan tekan pintasan...'
        }
      }
    },
    mouse: {
      title: 'Mouse',
      mode: 'Mode tetikus',
      absolute: 'Mouse mengikuti',
      absoluteAndroid: 'Mouse mengikuti (lebih kompatibel dengan Android)',
      touchScreen: 'Layar sentuh multipoin',
      relative: 'Mouse eksklusif',
      direction: 'Arah roda gulir',
      scrollNatural: 'Alami',
      scrollConventional: 'Konvensional',
      speed: 'Kecepatan roda gulir',
      scrollStep: 'Gulir touchpad',
      fast: 'Cepat',
      slow: 'Lambat',
      requestPointer:
        'Menggunakan mode relatf. Silakan klik desktop untuk mendapatkan penunjuk tetikus.',
      repairIphoneDrag: 'Perbaiki seret iPhone',
      resetHid: 'Setel ulang HID',
      touchpadGuide: {
        title: 'Panduan touchpad',
        scope: 'Berlaku untuk input penunjuk biasa saat Mode Mouse adalah Relatif.',
        swipeTitle: 'Usap untuk menggerakkan',
        swipeDesc: 'Usap di dalam area layar aktif untuk menggerakkan penunjuk jarak jauh.',
        tapTitle: 'Ketuk untuk mengeklik',
        tapDesc: 'Ketukan singkat mengirim satu klik kiri.',
        holdTitle: 'Tahan untuk menekan tombol kiri',
        holdDesc: 'Sentuh dan tahan tetap diam sekitar 1 detik untuk menahan tombol kiri mouse.',
        dragTitle: 'Gerakkan setelah tahan untuk menyeret',
        dragDesc:
          'Setelah tahan aktif, gerakkan jari untuk menyeret dengan tombol kiri tetap ditekan.'
      },
      hidOnly: {
        errors: {
          load: 'Tidak dapat memuat status mode HID-Only. Coba lagi sebentar lagi.',
          update:
            'Tidak dapat memastikan apakah mode HID-Only telah diperbarui. Buka kembali pengaturan ini untuk memeriksa status saat ini, lalu coba lagi.'
        },
        title: 'Mode hanya HID',
        desc: 'Jika mouse dan keyboard Anda berhenti merespons dan menyetel ulang HID tidak membantu, mungkin ada masalah kompatibilitas antara NanoKVM dan perangkat. Coba aktifkan mode HID-Only untuk kompatibilitas yang lebih baik.',
        tip1: 'Mengaktifkan mode HID-Hanya akan melepas U-disk virtual dan jaringan virtual',
        tip2: 'Dalam mode HID-Only, pemasangan gambar dinonaktifkan',
        enable: 'Aktifkan mode HID-Hanya',
        disable: 'Nonaktifkan mode HID-Hanya'
      }
    },
    preview: {
      title: 'Pratinjau'
    },
    image: {
      storageReadOnly: 'Penyimpanan berkas tidak dapat ditulis. Nonaktifkan disk virtual.',
      storageBusy:
        'Penyimpanan sedang digunakan. Selesaikan atau batalkan transfer berkas sebelum beralih disk virtual.',
      title: 'Gambar',
      loading: 'Memuat...',
      empty: 'Tidak ada yang ditemukan',
      upload: 'Unggah Gambar',
      dropFile: 'Letakkan image di sini, atau klik untuk memilih file',
      uploadAction: 'Unggah',
      cancelAction: 'Batalkan',
      mount: 'Pasang gambar',
      mounted: 'Terpasang',
      uploadError: 'Pengunggahan gagal, silakan coba lagi',
      uploadInsufficientStorage: 'Ruang penyimpanan tidak mencukupi',
      uploadTooLarge: 'Image melebihi batas ukuran',
      cancelUploadError: 'Gagal membatalkan pengunggahan',
      mountMode: 'Mode pemasangan',
      massStorage: 'Penyimpanan massal',
      cdRom: 'CD ROM',
      readOnly: 'Hanya baca',
      mountFailed: 'Pemasangan Gagal',
      mountDesc:
        'Di beberapa sistem, perlu mengeluarkan disk virtual pada host jarak jauh sebelum memasang gambar.',
      unmountFailed: 'Pelepasan gagal',
      unmountDesc:
        'Di beberapa sistem, Anda perlu mengeluarkan secara manual di host jarak jauh sebelum melepaskan gambar.',
      refresh: 'Segarkan daftar gambar',
      download: 'Unduh Gambar',
      attention: 'Perhatian',
      deleteConfirm: 'Apakah Anda yakin ingin menghapus gambar ini?',
      deleteFailed: 'Gagal menghapus gambar',
      okBtn: 'Ya',
      cancelBtn: 'Tidak'
    },
    script: {
      title: 'Script',
      upload: 'Unggah',
      run: 'Jalankan',
      runBackground: 'Jalankan di belakang',
      runFailed: 'Gagal menjalankan',
      uploadFailed: 'Gagal mengunggah skrip',
      deleteFailed: 'Gagal menghapus skrip',
      attention: 'Perhatian',
      delDesc: 'Apa kamu yakin menghapus data ini?',
      confirm: 'Ya',
      cancel: 'Tidak',
      delete: 'Hapus',
      close: 'Tutup'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      confirm: 'Oke',
      submit: 'Kirim',
      username: 'Nama pengguna',
      password: 'Kata sandi',
      authCode: 'Kode autentikasi',
      authorizeTitle: 'Otorisasi Terminal Web',
      codeAccountHint: 'Masukkan kode autentikasi atau kode pemulihan yang belum digunakan.',
      authFailed: 'Autentikasi gagal'
    },
    wol: {
      errors: {
        send: 'Tidak dapat mengirim perintah bangunkan. Periksa alamat MAC dan koneksi jaringan, lalu coba lagi.',
        loadFailed:
          'Tidak dapat memuat daftar perangkat Wake-on-LAN. Pastikan NanoKVM online, lalu coba lagi.',
        renameFailed:
          'Tidak dapat mengganti nama perangkat ini. Periksa alamat MAC, lalu coba lagi.',
        deleteFailed: 'Tidak dapat menghapus perangkat ini. Coba lagi sebentar lagi.'
      },
      title: 'Wake-on-LAN',
      sending: 'Kirim perintah...',
      sent: 'Perintah terkirim',
      input: 'Silahkan masukkan MAC',
      ok: 'Oke'
    },
    download: {
      sha256: 'SHA-256 (opsional)',
      sha256Placeholder: 'Masukkan checksum SHA-256 64 karakter',
      invalidSHA256: 'SHA-256 harus berupa string heksadesimal 64 karakter',
      title: 'Pengunduh Gambar',
      input: 'Silakan masukkan gambar jarak jauh URL',
      ok: 'Oke',
      disabled: 'Partisi /data adalah RO, jadi kami tidak dapat mengunduh gambarnya',
      downloadingFile: 'Mengunduh: {{file}}',
      downloadingProgress: 'Mengunduh ({{percentage}}): {{file}}',
      failed: 'Gagal',
      insufficientStorage: 'Ruang penyimpanan tidak mencukupi',
      remaining: 'Tersisa {{time}}',
      cancel: 'Batalkan',
      cancelFailed: 'Gagal membatalkan unduhan',
      pause: 'Jeda',
      resume: 'Lanjutkan',
      pauseFailed: 'Gagal menjeda unduhan',
      resumeFailed: 'Gagal melanjutkan unduhan'
    },
    settings: {
      back: 'Kembali',
      close: 'Tutup',
      ai: {
        errors: {
          loadServices:
            'Tidak dapat memuat status layanan AI. Pastikan NanoKVM online, lalu coba lagi.',
          loadMcpConfig: 'Tidak dapat memuat konfigurasi layanan MCP. Coba lagi sebentar lagi.',
          updateMcpService:
            'Tidak dapat memperbarui status layanan MCP. Periksa status saat ini sebelum mencoba lagi.',
          accessMcpApiKey:
            'Tidak dapat mengambil kunci API MCP. Coba lagi dan selesaikan verifikasi jika diminta.',
          regenerateMcpApiKey: 'Tidak dapat membuat ulang kunci API MCP. Coba lagi sebentar lagi.',
          regeneratedMcpApiKeyUnavailable:
            'Kunci API telah dibuat ulang, tetapi tidak dapat ditampilkan. Coba tampilkan lagi. Kunci sebelumnya tidak lagi valid.',
          loadMemoryFabricState:
            'Tidak dapat memuat status layanan Memory Fabric. Coba lagi sebentar lagi.',
          updateMemoryFabricService:
            'Tidak dapat memperbarui status layanan Memory Fabric. Periksa status saat ini sebelum mencoba lagi.',
          loadScreenTimelapseState:
            'Tidak dapat memuat status layanan timelapse layar. Coba lagi sebentar lagi.',
          updateScreenTimelapseService:
            'Tidak dapat memperbarui status layanan timelapse layar. Periksa status saat ini sebelum mencoba lagi.',
          openScreenTimelapse:
            'Tidak dapat membuka pemutaran timelapse layar. Coba lagi sebentar lagi.',
          loadScreenTimelapseRecordings:
            'Tidak dapat memuat rekaman timelapse layar. Coba lagi sebentar lagi.',
          playScreenTimelapse:
            'Rekaman timelapse ini tidak dapat diputar sekarang. Pilih ulang tanggal atau coba lagi nanti.',
          screenTimelapseFullscreen:
            'Tidak dapat mengubah mode layar penuh. Periksa izin browser, lalu coba lagi.'
        },
        title: 'AI',
        mcpService: 'Layanan MCP',
        mcpServiceDesc:
          'Gateway terpadu terautentikasi untuk kontrol, media, dan alat Memory Fabric yang diaktifkan',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Proses memori konteks agen dan OCR secara lokal',
        memoryFabricRemoteRequiresMcp:
          'Aktifkan Layanan MCP di atas agar alat memori dan OCR tersedia dari jarak jauh.',
        memoryFabricRemoteAvailable:
          'Alat memori dan OCR tersedia melalui endpoint /api/mcp di atas dengan kunci API yang sama.',
        memoryFabricConfig: {
          title: 'Konfigurasi',
          description: 'Kelola hanya pengaturan model default Memory Fabric.',
          loadFailed: 'Gagal memuat konfigurasi Memory Fabric.',
          saveFailed: 'Gagal menyimpan konfigurasi Memory Fabric.',
          saved: 'Konfigurasi telah disimpan.',
          savedRestarted: 'Konfigurasi disimpan dan Memory Fabric dimulai ulang.',
          noChanges: 'Tidak ada perubahan konfigurasi.',
          reload: 'Muat ulang',
          save: 'Simpan',
          unsaved: 'Perubahan belum disimpan',
          conflictTitle: 'Konfigurasi telah berubah',
          conflictDescription:
            'Konfigurasi diubah di tempat lain. Muat versi terbaru sebelum menyimpannya kembali.',
          conflictReload: 'Muat versi terbaru',
          discardTitle: 'Buang perubahan yang belum disimpan?',
          discardDescription:
            'Memuat ulang akan mengganti isi formulir saat ini dengan konfigurasi terbaru yang tersimpan.',
          discardConfirm: 'Buang dan muat ulang',
          sections: {
            models: 'Model'
          },
          openAICompatibleHint:
            'Mendukung penyedia model yang kompatibel dengan OpenAI. Masukkan nama model, URL dasar, dan kunci API yang diberikan penyedia.',
          fields: {
            model: 'Nama model',
            baseUrl: 'URL dasar',
            maxTokens: 'Token maksimum',
            timeoutSeconds: 'Batas waktu (detik)'
          },
          apiKey: {
            title: 'Kunci API',
            configured: 'Sudah dikonfigurasi',
            notConfigured: 'Belum dikonfigurasi',
            keep: 'Pertahankan kunci saat ini',
            replace: 'Ganti kunci',
            clear: 'Hapus kunci',
            replacement: 'Kunci API baru',
            replacementPlaceholder: 'Masukkan kunci API baru',
            clearHint:
              'Kunci API yang tersimpan akan dihapus; variabel lingkungan mungkin masih menyediakan kunci.',
            required: 'Masukkan kunci API baru.'
          },
          validation: {
            required: 'Kolom ini wajib diisi.',
            invalidUrl: 'Masukkan URL HTTP atau HTTPS yang valid.'
          }
        },
        screenTimelapseService: 'Timelapse layar',
        screenTimelapseServiceDesc:
          'Putar ulang riwayat layar Memory Fabric dan cari konten layar yang direkam',
        screenTimelapseSearchRequiresMemoryFabric:
          'Untuk mencari konten rekaman, aktifkan juga Memory Fabric.',
        screenTimelapseOpen: 'Buka',
        screenTimelapseSearchPlaceholder: 'Cari konten dalam video...',
        screenTimelapseNoVideos: 'Tidak ada video timelapse ditemukan',
        screenTimelapseNoResults: 'Tidak ada frame yang cocok ditemukan',
        screenTimelapseMultipleResults: 'Hasil pencarian',
        screenTimelapseUnavailable: 'Tidak tersedia',
        screenTimelapsePlayerUnsupported: 'Browser ini tidak dapat memutar video timelapse',
        screenTimelapseSearchFailed: 'Pencarian gagal',
        screenTimelapseLoading: 'Memuat timelapse...',
        screenTimelapseSearchAction: 'Cari timelapse',
        screenTimelapsePlay: 'Putar timelapse',
        screenTimelapsePause: 'Jeda timelapse',
        screenTimelapseEnterFullscreen: 'Masuk layar penuh',
        screenTimelapseExitFullscreen: 'Keluar layar penuh',
        screenTimelapseFullscreenUnsupported: 'Browser ini tidak dapat masuk layar penuh',
        screenTimelapsePreviousDay: 'Hari sebelumnya',
        screenTimelapseNextDay: 'Hari berikutnya',
        screenTimelapseDeleteDay: 'Hapus rekaman untuk {{date}}',
        screenTimelapseDelete: 'Hapus',
        screenTimelapseDeleteConfirmTitle: 'Hapus rekaman timelapse?',
        screenTimelapseDeleteConfirmDesc:
          'Rekaman untuk {{date}} menggunakan {{size}}. Tindakan ini tidak dapat dibatalkan.',
        screenTimelapseDeleteSuccess: 'Rekaman untuk {{date}} telah dihapus',
        screenTimelapseDeleteFailed: 'Gagal menghapus rekaman untuk {{date}}',
        endpoint: 'Titik akhir',
        apiKey: 'Kunci API',
        regenerate: 'Buat ulang',
        copyApiKey: 'Salin kunci API',
        copyEndpoint: 'Salin titik akhir',
        showApiKey: 'Tampilkan kunci API',
        regenerateConfirmTitle: 'Buat ulang kunci API',
        regenerateConfirmDesc:
          'Klien MCP yang ada akan berhenti berfungsi sampai diperbarui dengan kunci baru. Lanjutkan?',
        okBtn: 'Ya',
        cancelBtn: 'Tidak'
      },
      title: 'Pengaturan',
      about: {
        title: 'Tentang NanoKVM',
        information: 'Informasi',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versi Aplikasi',
        applicationTip: 'Versi aplikasi web NanoKVM',
        arch: 'Arsitektur',
        device_number: 'Nomor Perangkat',
        image: 'Version Gambar',
        imageTip: 'Versi gambar sistem NanoKVM',
        deviceKey: 'Kunci Perangkat',
        community: 'Komunitas',
        hostname: 'Nama Host',
        hostnameUpdated: 'Nama host diperbarui. Nyalakan ulang untuk menerapkan.',
        ipType: {
          Wired: 'Berkabel',
          Wireless: 'Nirkabel',
          Other: 'Lainnya'
        }
      },
      appearance: {
        title: 'Penampilan',
        display: 'Tampilan',
        language: 'Bahasa',
        languageDesc: 'Pilih bahasa untuk antarmuka',
        languageChangeFailed: 'Gagal mengubah bahasa',
        webTitle: 'Judul Web',
        webTitleDesc: 'Menyesuaikan judul halaman web',
        menuBar: {
          title: 'Bilah Menu',
          mode: 'Mode Tampilan',
          modeDesc: 'Menampilkan bilah menu di layar',
          modeOff: 'Mati',
          modeAuto: 'Sembunyikan otomatis',
          modeAlways: 'Selalu terlihat',
          icons: 'Ikon Submenu',
          iconsDesc: 'Menampilkan ikon submenu di bilah menu'
        }
      },
      screen: {
        title: 'Layar',
        videoMode: {
          title: 'Modus Video',
          description: 'Pilih metode pengkodean dan transmisi'
        },
        edid: {
          description: 'Pilih parameter tampilan',
          default: 'Bawaan',
          custom: 'Kustom',
          document: 'Dokumen',
          monitorName: 'Nama Monitor',
          monitorNameDescription:
            'Maksimal 13 karakter ASCII yang dapat dicetak; kosongkan untuk menggunakan nama asli dari EDID',
          monitorNamePlaceholder: 'Gunakan nama EDID asli',
          monitorNameApply: 'Terapkan',
          monitorNameInvalid: 'Masukkan tidak lebih dari 13 karakter ASCII yang dapat dicetak',
          monitorNameUnsupported: 'EDID saat ini tidak mendukung penggantian nama monitor',
          monitorNameLoadFailed: 'Gagal memuat nama monitor',
          monitorNameUpdateFailed: 'Gagal memperbarui nama monitor',
          powerConfirmTitle: 'Konfirmasi sebelum memperbarui EDID',
          powerConfirmDescription:
            'Sambungkan daya PD terlebih dahulu. Memperbarui EDID dapat memutus layar sebentar; jika perangkat tidak dapat mendeteksi layar dengan benar setelahnya, mulai ulang secara manual.',
          powerConfirmOk: 'Lanjutkan',
          powerConfirmOkCountdown: 'Lanjutkan ({{seconds}})',
          powerConfirmCancel: 'Batal',
          errors: {
            badRequest: 'Permintaan EDID tidak valid',
            invalidFileName: 'Pilih file EDID .bin dengan nama file yang valid',
            fileTooLarge: 'File EDID terlalu besar',
            invalidLength: 'EDID harus berukuran 128, 256, 384, atau 512 byte',
            invalidHeader: 'Header EDID tidak valid',
            unsupportedVersion: 'Versi EDID ini tidak didukung',
            extensionCountMismatch: 'Jumlah blok ekstensi EDID tidak cocok dengan panjang file',
            invalidChecksum: 'Checksum EDID tidak valid',
            invalidCta: 'Struktur ekstensi CTA tidak valid',
            currentCannotOverwrite:
              'EDID yang sedang digunakan tidak dapat ditimpa. Beralihlah ke EDID lain terlebih dahulu',
            saveFailed: 'Gagal menyimpan EDID',
            switchFailed: 'Gagal mengganti EDID',
            uploadFailed: 'Gagal mengunggah EDID',
            deleteFailed: 'Gagal menghapus EDID',
            networkFailed: 'Kesalahan jaringan. Coba lagi'
          }
        },
        rateControlMode: {
          title: 'Mode kontrol bitrate',
          description: 'Pilih metode untuk mengontrol bitrate',
          cbr: 'CBR (Bitrate Konstan): Mengutamakan kestabilan koneksi. Meminimalkan kegagapan, namun kualitas gambar mungkin menurun selama adegan gerak tinggi.',
          vbr: 'VBR (Variable Bitrate): Mengutamakan kejernihan gambar. Adegan dengan gerakan tinggi dapat menyebabkan lonjakan bandwidth, yang berpotensi menyebabkan latensi atau stutter.'
        },
        bitrate: {
          title: 'Kecepatan bit',
          description: 'Seimbangkan kejernihan gambar dengan penggunaan bandwidth',
          lower:
            'Nilai yang lebih rendah mengurangi latensi pada jaringan yang buruk namun menurunkan kualitas gambar, sehingga berpotensi menyebabkan keburaman atau artefak.',
          higher:
            'Nilai yang lebih tinggi meningkatkan kejernihan gambar namun mengkonsumsi lebih banyak bandwidth. Dapat meningkatkan latensi jika koneksi lambat.'
        },
        gop: {
          description: 'Atur interval antara bingkai utama penuh',
          lower:
            'Nilai yang lebih rendah mengirim frame penuh lebih sering. Hal ini mempercepat pemulihan artefak visual tetapi menghabiskan lebih banyak bandwidth dan mengurangi detail gambar.',
          higher:
            'Nilai yang lebih tinggi mengoptimalkan pembaruan tambahan. Hal ini meningkatkan detail gambar dan menghemat bandwidth, namun pemulihan dari artefak visual akan memakan waktu lebih lama.'
        },
        quality: {
          title: 'Kualitas',
          description: 'Mengatur kualitas gambar tampilan'
        },
        fps: {
          auto: 'Otomatis',
          description: 'Mengatur kecepatan bingkai tampilan',
          tip1: 'Menetapkan nilai ke 0 mengaktifkan mode Otomatis, menyinkronkan tampilan dengan kecepatan bingkai host jarak jauh.',
          tip2: 'Menurunkan frame rate secara signifikan mengurangi penggunaan bandwidth. Kurangi secara manual saat jaringan tidak stabil atau saat frame rate tinggi tidak diperlukan.'
        },
        scale: {
          title: 'Skala',
          description: 'Mengatur rasio skala tampilan'
        }
      },
      device: {
        title: 'Perangkat',
        virtualAudioConfirm: {
          title: 'Konfirmasi sebelum mengalihkan Audio Virtual',
          description:
            'Sambungkan daya PD terlebih dahulu. Mengalihkan Audio Virtual dapat memutus koneksi layar untuk sementara; jika perangkat tidak dapat mendeteksi layar dengan benar setelah peralihan, mulai ulang perangkat secara manual.',
          ok: 'Lanjutkan peralihan',
          okCountdown: 'Lanjutkan peralihan ({{seconds}})',
          cancel: 'Batal'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurasi Wi-Fi',
          apMode: 'Mode AP diaktifkan, sambungkan ke Wi-Fi secara manual',
          others: 'Lainnya...',
          connect: 'Bergabunglah dengan Wi-Fi',
          connectDesc1: 'Silakan masukkan ssid jaringan dan kata sandi',
          connectDesc2: 'Silakan masukkan kata sandi untuk bergabung dengan jaringan ini',
          disconnect: 'Apakah Anda yakin memutuskan sambungan jaringan?',
          ssid: 'Nama',
          password: 'Kata sandi',
          joinBtn: 'Gabung',
          confirmBtn: 'Oke',
          cancelBtn: 'Batal',
          preset: {
            title: 'Wi-Fi Tersimpan',
            description:
              'Simpan satu Wi-Fi target lebih awal; perangkat akan terhubung otomatis saat dinyalakan',
            empty: 'Tidak ada',
            addBtn: 'Tambah prasetel',
            removeBtn: 'Hapus prasetel',
            saveTitle: 'Simpan Wi-Fi Lebih Awal',
            saveDesc:
              'Simpan jaringan target sekarang; perangkat akan terhubung otomatis saat jaringan ditemukan saat boot',
            saveBtn: 'Simpan',
            passwordPlaceholder: 'Kata sandi (kosongkan untuk jaringan terbuka)',
            saveFailed: 'Gagal menyimpan prasetel Wi-Fi',
            removeFailed: 'Gagal menghapus prasetel Wi-Fi',
            removeConfirm: 'Yakin ingin menghapus Wi-Fi tersimpan?',
            warning:
              'Setelah disimpan, perangkat akan mencoba terhubung ke Wi-Fi ini setiap kali boot saat terdeteksi, yang dapat mengganggu koneksi ke Wi-Fi lainnya. Preset tetap berlaku hingga Anda mengubah atau menghapusnya.'
          }
        },
        capture: {
          title: 'Pengambilan Video',
          description: 'Aktifkan pengambilan video'
        },
        ssh: {
          description: 'Aktifkan akses jarak jauh SSH',
          tip: 'Setelah mengaktifkan SSH, tetapkan kata sandi yang kuat untuk pengguna root melalui terminal.'
        },
        swap: {
          errors: {
            load: 'Tidak dapat memuat status memori swap. Coba lagi sebentar lagi.',
            update:
              'Tidak dapat memperbarui pengaturan memori swap. Periksa status saat ini sebelum mencoba lagi.'
          },
          title: 'Memori swap',
          description: 'Aktifkan memori swap terkompresi'
        },
        advanced: 'Pengaturan Lanjutan',
        staticIp: {
          title: 'IP Statis',
          config: 'Konfigurasi',
          tip: 'Untuk mencegah hilangnya koneksi, disarankan untuk mengonfigurasi saat Anda memiliki akses fisik ke perangkat.',
          document: 'Periksa dokumennya.',
          enable: 'Aktifkan IP statis',
          placeholder: 'Format: IP/CIDR [Gerbang]\nContoh: 192.168.1.10/22',
          notEmpty: 'Alamat IP tidak boleh kosong.',
          notValid: 'Format alamat IP tidak valid.',
          failed: 'Konfigurasi gagal.',
          success: 'Konfigurasi selesai. Silakan periksa apakah alamat IP tersedia.',
          okBtn: 'Oke',
          cancelBtn: 'Batal'
        },
        usbNetwork: {
          title: 'Jaringan USB',
          mode: 'Mode jaringan',
          server: 'Mode server',
          serverDesc: 'Berikan alamat IP kepada host jarak jauh melalui server DHCP bawaan.',
          client: 'Mode klien',
          clientDesc: 'Dapatkan alamat IP dari host jarak jauh melalui DHCP.',
          forwarding: 'Penerusan IP',
          forwardingTip:
            'Agar host jarak jauh dapat mengakses internet melalui NanoKVM, aktifkan penerusan IP.',
          manualIp: 'Alamat IP manual',
          ipv4Placeholder: 'IPv4/CIDR, mis. 10.10.10.1/24',
          invalidIpv4: 'Masukkan alamat IPv4/CIDR yang valid.',
          loadFailed: 'Tidak dapat memuat konfigurasi jaringan USB.',
          failed: 'Konfigurasi gagal.',
          success: 'Konfigurasi selesai.',
          applyBtn: 'Terapkan',
          cancelBtn: 'Batal'
        },
        usbId: {
          title: 'Identitas perangkat USB',
          config: 'Konfigurasi',
          tip: 'ID USB kustom mengubah cara host mengidentifikasi perangkat ini. Menerapkan akan merestart gadget USB dan dapat memutus keyboard, mouse, atau jaringan sebentar.',
          enable: 'Aktifkan identitas perangkat USB kustom',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Versi perangkat (bcdDevice)',
          serialnumber: 'Nomor seri',
          manufacturer: 'Produsen',
          product: 'Produk',
          vidPlaceholder: 'mis. 0x359F',
          pidPlaceholder: 'mis. 0x3305',
          bcdDevicePlaceholder: 'mis. 0x0101 (opsional)',
          serialnumberPlaceholder: 'mis. 0123456789ABCDEF (opsional)',
          manufacturerPlaceholder: 'mis. sipeed (opsional)',
          productPlaceholder: 'mis. NanoKVM-Go (opsional)',
          notEmpty: 'VID dan PID tidak boleh kosong.',
          notValid: 'Format VID/PID tidak valid. Gunakan hex seperti 0x359F.',
          bcdNotValid:
            'Format bcdDevice tidak valid. Gunakan hex seperti 0x0101, atau biarkan kosong.',
          failed: 'Konfigurasi gagal.',
          success: 'Konfigurasi selesai. Gadget USB di-restart.',
          okBtn: 'OK',
          cancelBtn: 'Batal'
        },
        diskInquiry: {
          title: 'Nama drive USB',
          config: 'Konfigurasi',
          tip: 'Nama flash drive USB virtual yang dilihat komputer jarak jauh. Menyimpan akan memulai ulang USB dan dapat memutus keyboard, mouse, atau jaringan sebentar.',
          enable: 'Gunakan nama drive USB kustom',
          placeholder: 'mis. sipeed  Flash Drive     1.00',
          notEmpty: 'Nama tidak boleh kosong.',
          notValid:
            'Nama tidak valid. Maksimal 28 karakter ASCII yang dapat dicetak tanpa baris baru.',
          failed: 'Konfigurasi gagal.',
          success: 'Konfigurasi selesai. Gadget USB dimulai ulang.',
          okBtn: 'OK',
          cancelBtn: 'Batal'
        },
        diskInquiryCdrom: {
          title: 'Nama CD virtual',
          config: 'Konfigurasi',
          tip: 'Nama drive CD/DVD virtual yang dilihat komputer jarak jauh. Menyimpan akan memulai ulang USB dan dapat memutus keyboard, mouse, atau jaringan sebentar.',
          enable: 'Gunakan nama CD virtual kustom',
          placeholder: 'mis. sipeed  Optical Drive   1.00',
          notEmpty: 'Nama tidak boleh kosong.',
          notValid:
            'Nama tidak valid. Maksimal 28 karakter ASCII yang dapat dicetak tanpa baris baru.',
          failed: 'Konfigurasi gagal.',
          success: 'Konfigurasi selesai. Gadget USB dimulai ulang.',
          okBtn: 'OK',
          cancelBtn: 'Batal'
        },
        uac2FunctionName: {
          title: 'Nama audio USB',
          config: 'Konfigurasi',
          tip: 'Nama mikrofon / perangkat audio USB virtual yang dilihat komputer jarak jauh. Menyimpan akan memulai ulang USB dan dapat memutus keyboard, mouse, atau jaringan sebentar.',
          enable: 'Gunakan nama audio USB kustom',
          placeholder: 'mis. NanoKVM-Go Audio',
          notEmpty: 'Nama tidak boleh kosong.',
          notValid: 'Nama tidak valid. Maksimal 64 karakter yang dapat dicetak tanpa baris baru.',
          failed: 'Konfigurasi gagal.',
          success: 'Konfigurasi selesai. Gadget USB dimulai ulang.',
          okBtn: 'OK',
          cancelBtn: 'Batal'
        },
        lowPower: {
          title: 'Mode Daya Rendah',
          description: 'Kurangi penggunaan daya saat idle',
          tips: 'Tip',
          tip1: 'Saat diaktifkan, pengambilan video akan dijeda selama periode tidak aktif untuk mengurangi konsumsi daya dan panas.',
          tip2: 'Melanjutkan pengambilan video mungkin memerlukan inisialisasi ulang, dan selama proses tersebut video mungkin tidak tersedia untuk sementara.',
          okBtn: 'Oke',
          cancelBtn: 'Batal'
        },
        mouseJiggler: {
          title: 'Tikus Jiggler',
          description: 'Mencegah host jarak jauh tertidur',
          disable: 'Nonaktifkan',
          absolute: 'Mode Absolut',
          relative: 'Mode Relatif'
        },
        fingerRobot: {
          title: 'Robot Jari',
          description: 'Aktifkan kontrol robot jari'
        },
        mdns: {
          description: 'Aktifkan layanan penemuan mDNS'
        },
        hidOnly: 'HID-Mode Hanya',
        imageMounted: 'Gambar Terpasang',
        disk: 'Disk Virtual',
        diskDesc: 'Pasang disk U virtual pada host jarak jauh',
        sdCard: {
          warning: 'Inisialisasi Disk Virtual',
          tip1: 'Saat pertama kali diaktifkan, disk virtual akan diinisialisasi dan file image serta data disk virtual yang ada di perangkat akan dihapus. Tindakan ini tidak dapat dibatalkan. Cadangkan file penting terlebih dahulu.',
          tip2: 'Lanjutkan inisialisasi?',
          capacity: 'Kapasitas',
          disable: 'Nonaktifkan',
          disableWarning: 'Nonaktifkan Disk Virtual?',
          disableTip:
            'Menonaktifkan akan menghapus semua file pada disk virtual dan membebaskan penyimpanan perangkat yang digunakannya. Saat diaktifkan kembali, Anda harus memilih kapasitas dan melakukan inisialisasi lagi. Tindakan ini tidak dapat dibatalkan.'
        },
        network: 'Jaringan Virtual',
        networkDesc: 'Pasang kartu jaringan virtual pada host jarak jauh',
        mic: 'Audio Virtual',
        micDesc: 'Pilih mode audio virtual USB untuk host jarak jauh',
        virtualAudioMode: {
          off: 'Nonaktif',
          mic: 'Mikrofon saja',
          duplex: 'Mikrofon + speaker'
        },
        speaker: 'Speaker virtual',
        speakerDesc:
          'Saat diaktifkan, menggantikan keluaran audio DP bawaan untuk mendukung panggilan jarak jauh dengan kompatibilitas yang lebih baik. Saat dinonaktifkan, hanya audio media yang dapat diteruskan, bukan audio panggilan.',
        micTip:
          'Karena keterbatasan titik akhir USB, mengaktifkan Mikrofon Virtual akan menonaktifkan Layar Sekunder USB.\nDan disarankan untuk menonaktifkan fitur ini selama pengoperasian BIOS untuk memastikan kompatibilitas optimal.',
        reboot: 'Mulai ulang',
        rebootDesc: 'Apakah Anda yakin ingin me-reboot NanoKVM?',
        rebootToFlashing: 'Mulai ulang ke mode flashing',
        rebootToFlashingButton: 'Mulai ulang',
        rebootToFlashingTitle: 'Mulai ulang ke mode flashing?',
        rebootToFlashingDesc:
          'NanoKVM akan dimulai ulang dan masuk ke mode flashing USB. Pastikan antarmuka USB untuk flashing telah terhubung dan alat flashing sudah siap.',
        datetime: {
          timezone: 'Zona waktu',
          datetime: 'Tanggal waktu',
          dateAndTime: 'Tanggal dan waktu',
          synchronize: 'Sinkronisasi waktu',
          lastSynchronization: 'Sinkronisasi terakhir: ',
          notSynchronized: 'Waktu sistem belum disinkronkan',
          syncNow: 'Sinkronkan sekarang',
          manualSet: 'Atur manual',
          manualSetTitle: 'Atur waktu secara manual',
          manualSetDescription: 'Anda harus masuk kembali setelah waktu diubah.',
          useBrowserTime: 'Gunakan waktu browser',
          setTime: 'Atur waktu',
          invalidTime: 'Masukkan tanggal dan waktu yang valid.',
          setFailed: 'Tidak dapat mengatur waktu sistem. Coba lagi.',
          setSucceeded: 'Waktu sistem telah diperbarui. Silakan masuk kembali.'
        },
        dialBackground: {
          errors: {
            load: 'Tidak dapat memuat gambar latar belakang dial. Coba lagi sebentar lagi.',
            upload:
              'Tidak dapat mengunggah gambar latar belakang. Pastikan file adalah gambar BMP yang valid, lalu coba lagi.',
            switch:
              'Tidak dapat mengganti latar belakang dial. Segarkan daftar gambar, lalu coba lagi.'
          },
          title: 'Latar dial',
          description: 'Pilih gambar BMP dari /kvmcomm/ui/bg/ sebagai latar dial',
          upload: 'Unggah',
          loading: 'Memuat...',
          empty: 'Tidak ada gambar BMP ditemukan',
          current: 'Saat ini',
          onlyBmp: 'Hanya gambar BMP yang didukung',
          uploaded: 'Unggahan selesai',
          switched: 'Latar diganti'
        },
        okBtn: 'Ya',
        cancelBtn: 'Tidak'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Mulai ulang Tailscale?',
        stop: 'Hentikan Tailscale?',
        stopDesc: 'Keluar dari Tailscale dan nonaktifkan mulai otomatis saat boot.',
        loading: 'Memuat...',
        notInstall: 'Tailscale tidak ditemukan! Silahkan pasang.',
        install: 'Memasang',
        installing: 'Memasangkan',
        failed: 'Gagal memasangkan',
        retry: 'Harap segarkan dan coba lagi. Atau coba instal secara manual',
        download: 'Mengunduh',
        package: 'paket instalasi',
        unzip: 'dan unzip itu',
        upTailscale: 'Unggah tailscale ke direktori NanoKVM /usr/bin/',
        upTailscaled: 'Unggah tailscaled ke direktori NanoKVM /usr/sbin/',
        refresh: 'Segarkan halaman ini',
        notRunning: 'Tailscale tidak berjalan. Silakan mulai untuk melanjutkan.',
        run: 'Mulai',
        notLogin:
          'Perangkat belum ditautkan. Silakan masuk dan tautkan perangkat ini ke akun Anda.',
        urlPeriod: 'Url ini berlaku selama 10 menit',
        login: 'Masuk',
        loginSuccess: 'Berhasil masuk',
        enable: 'Aktifkan Tailscale',
        exitNode: 'Simpul keluar',
        exitNodeDesc: 'Iklankan perangkat ini sebagai jalur keluar internet untuk tailnet Anda.',
        subnetRoute: 'Rute subnet',
        subnetRouteDesc: 'Iklankan subnet LAN yang terhubung langsung {{subnet}} ke tailnet Anda.',
        subnetUnavailable: 'Tidak ada subnet LAN yang terhubung langsung.',
        routingUnavailable: 'Pengaturan perutean Tailscale tidak tersedia di perangkat ini.',
        pendingApproval:
          'Setelah diaktifkan, setujui rute yang diiklankan di konsol admin Tailscale.',
        routingUpdateFailed: 'Gagal memperbarui pengaturan perutean Tailscale.',
        deviceName: 'Nama Perangkat',
        deviceIP: 'IP Perangkat',
        account: 'Akun',
        logout: 'Keluar',
        logoutDesc: 'Apakah Anda yakin ingin logout?',
        uninstall: 'Copot pemasangan Tailscale',
        uninstallDesc: 'Apakah Anda yakin ingin menghapus instalan Tailscale?',
        okBtn: 'Ya',
        cancelBtn: 'Tidak'
      },
      update: {
        title: 'Periksa pembaruan',
        queryFailed: 'Gagal mendapatkan versi',
        updateFailed: 'Gagal memperbarui, tolong coba lagi.',
        isLatest: 'Kamu sudah menggunakan versi terbaru.',
        available: 'Ada pembaruan baru. apa kamu mau memperbarui?',
        updating: 'Pembaruan dimulai. Silahkan tunggu...',
        confirm: 'Konfirmasi',
        upgrade_tip_nano:
          'Versi baru NanoKVM telah tersedia. Apakah Anda yakin ingin memperbarui sekarang?',
        upgrade_tip_firmware:
          'Versi firmware baru telah tersedia. Apakah Anda yakin ingin memperbarui sekarang?',
        cancel: 'Batalkan',
        preview: 'Pratinjau Pembaruan',
        previewDesc: 'Dapatkan akses awal ke fitur dan peningkatan baru',
        previewTip:
          'Perlu diketahui bahwa rilis pratinjau mungkin mengandung bug atau fungsi yang tidak lengkap!',
        download: 'Unduh',
        install: 'Instal',
        changelog: 'Ubahlog',
        restart: 'Mulai ulang'
      },
      account: {
        title: 'Akun',
        webAccount: 'Nama Akun Web',
        password: 'Kata sandi',
        twoFactor: 'Autentikasi dua faktor',
        enableTwoFactor: 'Aktifkan',
        enableTwoFactorDesc:
          'Masukkan kata sandi saat ini untuk mulai menyiapkan autentikasi dua faktor.',
        disableTwoFactor: 'Nonaktifkan',
        recoveryCodes: 'Kode pemulihan',
        enableRecoveryCodes: 'Aktifkan',
        updateRecoveryCodes: 'Perbarui',
        recoveryCodesTooltipDisabled:
          'Kode pemulihan adalah kredensial cadangan untuk autentikasi dua faktor. Anda dapat membuatnya setelah mengaktifkan autentikasi dua faktor.',
        recoveryCodesTooltipEnabled:
          'Kode pemulihan hanya valid saat autentikasi dua faktor aktif. Menonaktifkan autentikasi dua faktor akan membatalkan semua kode pemulihan; selama tetap aktif, kode hanya dapat diperbarui.',
        recoveryCodesRegenerateDesc:
          'Masukkan kata sandi saat ini dan, entah kode autentikasi atau kode pemulihan yang belum digunakan. Memperbarui kode pemulihan akan membatalkan semua kode pemulihan yang ada.',
        recoveryCodesVerifyFailed:
          'Tidak dapat memverifikasi kata sandi dan kode autentikasi atau kode pemulihan.',
        recoveryCodesSavedTitle: 'Simpan kode pemulihan ini sekarang',
        recoveryCodesSavedDesc:
          'Kode pemulihan hanya dapat dilihat sekali. Simpan di tempat aman sebelum meninggalkan halaman ini.',
        enableRecoveryCodesDesc:
          'Masukkan kata sandi saat ini dan kode autentikasi untuk membuat kode pemulihan. Kode pemulihan hanya dapat dilihat sekali, jadi simpan sebelum meninggalkan halaman ini.',
        copyRecoveryCodes: 'Salin',
        downloadRecoveryCodes: 'Unduh',
        recoveryCodesCopyFailed: 'Tidak dapat menyalin kode pemulihan.',
        done: 'Selesai',
        currentPassword: 'Kata sandi saat ini',
        authCode: 'Kode autentikasi',
        authCodeOrRecoveryCode: 'Kode autentikasi atau kode pemulihan',
        authRateLimited: 'Terlalu banyak percobaan gagal. Coba lagi dalam {{seconds}} dtk.',
        verifyPasswordChange: 'Verifikasi perubahan kata sandi',
        verifyPasswordChangeDesc:
          'Masukkan kata sandi saat ini dan, entah kode autentikasi atau kode pemulihan yang belum digunakan, sebelum mengubah kata sandi.',
        verifyPasswordChangeFailed:
          'Tidak dapat memverifikasi kata sandi dan kode autentikasi atau kode pemulihan.',
        verifyAndEnable: 'Verifikasi dan aktifkan',
        scanTwoFactorQRCode:
          'Pindai kode QR ini dengan aplikasi autentikator Anda, lalu masukkan kodenya.',
        showSetupKey: 'Tampilkan kunci pengaturan',
        hideSetupKey: 'Sembunyikan kunci pengaturan',
        setupKey: 'Kunci pengaturan',
        twoFactorSetupFailed: 'Tidak dapat memulai penyiapan autentikasi dua faktor.',
        twoFactorVerifyFailed: 'Tidak dapat memverifikasi kode autentikasi.',
        twoFactorDisableFailed: 'Tidak dapat menonaktifkan autentikasi dua faktor.',
        disableTwoFactorDesc:
          'Masukkan kata sandi saat ini dan, entah kode autentikasi atau kode pemulihan yang belum digunakan. Menonaktifkan autentikasi dua faktor akan membatalkan semua kode pemulihan.',
        updateBtn: 'Ubah',
        logoutBtn: 'Keluar',
        logoutDesc: 'Apakah Anda yakin ingin logout?',
        okBtn: 'Ya',
        cancelBtn: 'Tidak',
        twoFactorProtection: {
          title: 'Perlindungan 2FA Lanjutan',
          description:
            'Pilih tindakan sensitif yang memerlukan faktor kedua saat perlindungan diterapkan. Sakelar "Login web langsung" melakukan sebaliknya: sakelar ini memungkinkan login hanya dengan kode autentikasi atau kode pemulihan yang belum digunakan.',
          login: 'Login web langsung',
          webTerminal: 'Terminal Web',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Kunci API MCP',
          targetPower: 'Daya mesin target',
          nanoKVMRestart: 'Mulai ulang NanoKVM',
          deleteImage: 'Hapus image',
          scripts: 'Operasi skrip',
          screenTimelapse: 'Timelapse layar',
          loginScope:
            'Memungkinkan login web hanya dengan kode autentikasi atau kode pemulihan yang belum digunakan, bukan nama pengguna dan kata sandi. Ini mengurangi keamanan login.',
          webTerminalScope: 'Membuka Terminal Web.',
          sshScope: 'Mengubah status layanan SSH.',
          tailscaleScope: 'Memasang, menghubungkan, atau mengubah Tailscale.',
          frp: 'FRP',
          frpScope: 'Memasang, menghubungkan, atau mengubah FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Memasang, menghubungkan, atau mengubah Cloudflare Tunnel.',
          zerotierScope: 'Memasang, menghubungkan, atau mengubah ZeroTier.',
          netbirdScope: 'Memasang, menghubungkan, atau mengubah NetBird.',
          mcpScope:
            'Menampilkan, menyalin, atau membuat ulang kunci API MCP, serta mengaktifkan atau menonaktifkan MCP.',
          targetPowerScope: 'Mematikan atau memulai ulang mesin target dengan tombol daya fisik.',
          nanoKVMRestartScope: 'Memulai ulang NanoKVM itu sendiri.',
          deleteImageScope: 'Menghapus secara permanen image ISO atau IMG yang disimpan.',
          scriptsScope: 'Menjalankan, mengunggah, atau menghapus skrip.',
          screenTimelapseScope:
            'Mengaktifkan perekaman timelapse layar, menelusuri rekaman, dan menghapus rekaman.',
          authorizeTitle: 'Izinkan perubahan perlindungan',
          authorizeDesc:
            'Masukkan kode autentikasi atau kode pemulihan yang belum digunakan. Izin ini berlaku selama lima menit.',
          authorizeFailed: 'Tidak dapat memverifikasi kode autentikasi atau kode pemulihan.',
          loadFailed: 'Tidak dapat memuat pengaturan perlindungan.',
          updateFailed: 'Tidak dapat menyimpan pengaturan perlindungan.',
          retry: 'Coba lagi',
          disableScriptsWarningTitle: 'Nonaktifkan perlindungan skrip?',
          disableScriptsWarningDesc:
            'Tanpa perlindungan, skrip berbahaya dapat memperoleh akses tingkat tinggi ke perangkat ini dan melewati atau menghapus perlindungan 2FA.',
          cancelDisabling: 'Batal',
          continueDisabling: 'Lanjutkan menonaktifkan',
          continueDisablingCountdown: 'Lanjutkan menonaktifkan ({{seconds}} dtk)'
        },
        twoFactorGuide: {
          title: 'Panduan autentikasi dua faktor',
          scope: 'Autentikasi dua faktor menambahkan lapisan keamanan ekstra ke akun Anda.',
          whatTitle: 'Apa itu 2FA?',
          whatDesc:
            'Autentikasi dua faktor (2FA) melindungi akun Anda dengan kode berbasis waktu dari aplikasi autentikator. Secara bawaan, kode ini diperlukan selain kata sandi Anda, sehingga meskipun kata sandi Anda bocor, akun Anda tetap terlindungi. Jika login web langsung diaktifkan, kode saja sudah cukup untuk masuk.',
          appsTitle: 'Aplikasi yang disarankan',
          appsDesc:
            'Anda dapat menggunakan Google Authenticator atau Microsoft Authenticator. Keduanya gratis dan berfungsi di iOS serta Android.',
          setupTitle: 'Cara menyiapkan',
          setupDesc:
            'Buka aplikasi autentikator, pindai kode QR yang ditampilkan saat penyiapan, atau masukkan kunci penyiapan secara manual. Lalu masukkan kode 6 digit dari aplikasi untuk menyelesaikan aktivasi 2FA.',
          recoveryTitle: 'Kode pemulihan',
          recoveryDesc:
            'Kode pemulihan adalah kode cadangan sekali pakai. Jika Anda kehilangan akses ke aplikasi autentikator, Anda dapat menggunakan kode pemulihan untuk login. Simpan di tempat aman segera setelah dibuat.',
          logoutNotice:
            'Mengaktifkan 2FA akan logout semua sesi aktif, termasuk sesi ini. Anda perlu login kembali di setiap perangkat.'
        }
      }
    },
    error: {
      title: 'Kami mengalami masalah',
      refresh: 'Segarkan'
    },
    pictureInPicture: {
      title: 'Gambar dalam gambar',
      enter: 'Aktifkan gambar dalam gambar',
      exit: 'Nonaktifkan gambar dalam gambar',
      error: 'Tidak dapat memulai gambar dalam gambar. Tunggu video dimuat, lalu coba lagi.'
    },
    fullscreen: {
      toggle: 'Beralih Layar Penuh'
    },
    menu: {
      collapse: 'Tutup Menu',
      expand: 'Perluas Menu'
    }
  }
};

export default id;
