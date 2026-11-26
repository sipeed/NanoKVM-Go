const ja = {
  translation: {
    storage: {
      cards: { mount: 'ディスクマウント', share: 'ファイル共有' },
      title: 'イメージとファイル共有',
      empty: 'イメージはまだありません',
      notAttached: '何もマウントされていません',
      attached: 'マウント中: {{name}}',
      eject: '取り出す',
      mount: 'マウント',
      mountMode: 'マウントモード',
      mountModeDisk: 'マスストレージ',
      mountModeCdrom: 'CD-ROM',
      openManager: 'ファイルマネージャーを開く',
      refresh: '更新',
      delete: '削除',
      deleteTitle: 'メディアを削除',
      deleteConfirm: '{{name}} を削除しますか？中のファイルは失われます。',
      downloadImage: 'イメージをダウンロード',
      typePartition: '共有ファイル',
      volumeLabel: 'リムーバブルディスク',
      groups: {
        share: 'ファイル共有',
        images: 'イメージ'
      },
      failed: '操作に失敗しました',
      hidOnly: 'HID のみのモード: 仮想ディスクは利用できません',
      disabled: '仮想ディスクは無効です (設定 - デバイス)',
      force: {
        title: 'メディアを強制的に取り戻しますか？',
        desc: 'ターゲットマシンがまだこのメディアを使用しています。取り戻すと USB デバイスが再列挙され、まだディスクに書き込まれていない内容が失われる可能性があります。',
        menuHint:
          'ターゲットマシンがこのメディアを使用しています。ファイルマネージャーを開いて強制的に取り戻してください。'
      },
      tips: {
        title: 'アップロード方法',
        usb1: 'NanoKVM を USB 経由でコンピュータに接続します；',
        usb2: 'このページで共有ファイルをマスストレージモードでマウントします。',
        usb3: 'コンピュータ上で仮想ディスクを開き、イメージファイルを仮想ディスクのルートディレクトリにコピーします。',
        usb4: 'コピーが終わったらディスクを取り出します。イメージがリストに表示されます。',
        scp1: 'NanoKVM とコンピュータが同じローカルエリアネットワークに接続されていることを確認します；',
        scp2: 'コンピュータでターミナルを開き、SCP コマンドを使用してイメージファイルを NanoKVM の /data ディレクトリにアップロードします。',
        scp3: '例：scp your-image-path root@your-nanokvm-ip:/data'
      },
      job: {
        running: '{{name}} を処理中'
      },
      enableDisk: {
        title: '仮想ディスクを有効にしますか？',
        desc: 'メディアをマウントするには仮想ディスクが必要ですが、現在は無効です。今すぐ有効にしますか？ターゲットマシンのキーボードとマウスが一時的に切断されます。',
        menuHint: '仮想ディスクは無効です。設定 - デバイスで有効にしてください。'
      },
      add: {
        title: 'メディアを追加',
        upload: 'ISO または IMG をアップロード',
        fromUrl: 'URL からダウンロード',
        url: 'イメージの URL',
        urlRequired: 'URL を入力してください',
        onlyImages: 'マウントできるのは .iso と .img ファイルのみです',
        dropHint: 'ISO または IMG をここにドロップするか、クリックしてファイルを選択',
        cdromOnlyIso: 'CD-ROM には ISO 9660 / UDF イメージしか載せられません',
        uploadFailed: 'アップロードに失敗しました',
        downloadFailed: 'ダウンロードに失敗しました',
        uploading: '{{name}} をアップロード中',
        downloading: '{{name}} をダウンロード中',
        done: '{{name}} の準備ができました',
        checksumFailed: 'イメージが SHA-256 チェックサムと一致しません。',
        blocked:
          'ターゲットマシンがディスクを使用しています。ターゲットマシンで仮想 USB ドライブを安全に取り出してから、設定 > デバイスで仮想ディスクのスイッチをオフにしてください。'
      },
      files: {
        reasonNotInitialized:
          '仮想 USB ドライブは未初期化です。ファイル共有を使用する前に、設定 > デバイスで初期化してください。',
        selectMedium: 'メディアを選択すると中のファイルを閲覧できます。',
        reasonAttached: 'このメディアはターゲットマシンに接続されています。',
        reasonAttachedHint:
          'ターゲットマシンで仮想 USB ドライブを安全に取り出してから、設定 > デバイスで仮想ディスクのスイッチをオフにしてください。',
        reasonNotBrowsable:
          'このメディアにはファイルマネージャーで開けないファイルシステムが含まれています。',
        reasonNoLoop: 'このデバイスはディスクイメージをマウントできません (loop サポートなし)。',
        reasonISOImage:
          'ISO 9660 イメージの中身はここでは閲覧できません。ターゲットマシンへのマウントは可能です。',
        empty: 'このフォルダは空です',
        noMatch: '検索に一致するファイルはありません。',
        free: '空き {{size}}',
        search: 'このフォルダを検索',
        sort: {
          modified: '更新日時',
          name: '名前',
          size: 'サイズ',
          type: '種類',
          asc: '昇順',
          desc: '降順'
        },
        view: {
          list: 'リスト表示',
          grid: 'アイコン表示'
        },
        folder: 'フォルダー',
        itemCount: '{{count}} 項目',
        itemCount_one: '{{count}} 項目',
        itemCount_other: '{{count}} 項目',
        dropHint: 'ここにファイルをドロップして、このフォルダーにアップロード',
        up: '親フォルダーへ移動',
        uploadButton: 'アップロード',
        newFolder: '新しいフォルダー',
        newFolderPlaceholder: 'フォルダー名',
        create: '作成',
        upload: 'メディアにファイルをアップロード',
        deleteFolderTitle: 'フォルダーを削除',
        deleteFolderConfirm:
          'フォルダー {{name}} とその中身をすべて削除しますか？この操作は元に戻せません。',
        uploading: 'アップロード中…',
        uploadFailed: 'アップロードに失敗しました',
        download: 'ダウンロード',
        delete: '削除',
        deleteTitle: 'ファイルを削除',
        deleteConfirm: '{{name}} をメディアから削除しますか？'
      },
      yes: 'はい',
      no: 'いいえ'
    },
    head: {
      storage: 'イメージとファイル共有',
      desktop: 'リモートデスクトップ',
      login: 'ログイン',
      changePassword: 'パスワード変更',
      terminal: 'ターミナル',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'クリップボードにコピーできません。ブラウザーにクリップボードへのアクセスを許可してから、もう一度お試しください。'
    },
    auth: {
      login: 'ログイン',
      language: '言語',
      placeholderUsername: 'ユーザー名を入力してください',
      placeholderPassword: 'パスワードを入力してください',
      placeholderPassword2: '再度パスワードを入力してください',
      noEmptyUsername: 'ユーザー名は空にできません',
      noEmptyPassword: 'パスワードは空にできません',
      noAccount:
        'ユーザー情報の取得に失敗しました。ウェブページをリフレッシュするか、パスワードをリセットしてください。',
      invalidUser: '無効なユーザー名またはパスワード',
      error: '予期しないエラー',
      twoFactorTitle: '確認が必要です',
      twoFactorDesc: 'ログインするには認証コードまたは未使用のリカバリーコードを入力してください。',
      sensitiveActionDesc:
        '続行するには認証コードまたは未使用のリカバリーコードを入力してください。',
      placeholderAuthCode: '認証コード',
      noEmptyAuthCode: '認証コードは必須です',
      verify: '確認',
      backToLogin: 'ログインに戻る',
      verificationFailed: '確認に失敗しました',
      twoFactorEnabledLogout: {
        message:
          '2要素認証が有効になりました。セキュリティのため、すべてのセッションがログアウトされました。もう一度サインインしてください。'
      },
      rateLimited: '失敗回数が多すぎます。{{seconds}} 秒後に再試行してください。',
      recoveryCodeUsed: 'このログインに使用したリカバリーコードは無効化されました。',
      changePassword: 'パスワード変更',
      changePasswordDesc:
        'デバイスのセキュリティのために、ウェブログインのパスワードを変更してください。',
      passwordVerificationRequired:
        '設定のアカウントページに戻り、このパスワード変更をもう一度確認してください。',
      passwordVerificationExpired:
        'パスワード変更の確認期限が切れました。設定のアカウントページに戻って再度確認してください。',
      returnToSettings: '設定に戻る',
      differentPassword: 'パスワードが一致しません',
      illegalUsername: 'ユーザー名に不正な文字が含まれています',
      illegalPassword: 'パスワードに不正な文字が含まれています',
      forgetPassword: 'パスワードを忘れた',
      ok: 'OK',
      cancel: 'キャンセル',
      savePassword: 'パスワードを保存',
      loginButtonText: 'ログイン',
      tips: {
        reset1: 'ウェブのデフォルトアカウント:',
        reset2: 'SSH デフォルト アカウント:',
        reset3: 'パスワードをリセットするには、以下を参照してください。 ',
        changeReset: 'Web ログインパスワードをリセットするには、以下を参照してください。 ',
        resetLink: 'リセット'
      }
    },
    wifi: {
      errors: {
        connect:
          'Wi-Fi に接続できません。ネットワーク名、パスワード、デバイスの Wi-Fi 状態を確認してから、もう一度お試しください。'
      },
      title: 'Wi-Fi',
      description: 'NanoKVM の Wi-Fi を構成する',
      submitted: {
        title: '少々お待ちください。NanoKVM が Wi-Fi に接続しています',
        tipsTitle: '次にすること',
        tipSwitch: 'スマートフォンは元の Wi-Fi に戻して大丈夫です。',
        tipCheck:
          'しばらくしたら NanoKVM の画面で新しい IP アドレスを確認し、ブラウザで開いてください。',
        tipRetry:
          'いつまでも変化がない場合は、ネットワーク名とパスワードを確認してもう一度お試しください。'
      },
      invalidMode:
        '現在のモードではネットワーク設定はサポートされていません。デバイスで Wi-Fi 設定モードを有効にしてください。',
      confirmBtn: 'OK',
      finishBtn: '完了',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'パスワード',
      steps: {
        settings: '設定',
        wifi: 'Wi-Fi',
        enable: '有効化'
      },
      scan: {
        title: 'Wi-Fi ネットワークを選択',
        description: '続行するには近くのネットワークを選択してください。',
        loading: '近くの Wi-Fi ネットワークを検索中…',
        empty: '近くに Wi-Fi ネットワークが見つかりません。',
        failed: 'スキャンできませんでした。ネットワーク情報を手動で入力してください。',
        retryBtn: '再スキャン',
        manualBtn: 'ネットワークを手動入力',
        backBtn: '近くのネットワークに戻る',
        connectTitle: '{{ssid}} に接続',
        connecting: '接続中…',
        viewSimple: 'シンプル表示に切り替え',
        viewDetailed: '詳細表示に切り替え'
      },
      ap: {
        authTitle: '認証が必要です',
        authDescription: '続行するには、AP パスワードを入力してください',
        authFailed: '無効な AP パスワード',
        passPlaceholder: 'AP パスワード',
        verifyBtn: '確認する'
      },
      qr: {
        cameraBtn: '写真を撮る',
        pickBtn: 'スクリーンショットを選ぶ',
        scanHint:
          'Wi-Fi QR コードの写真を撮影するか、スマートフォンが Wi-Fi を共有する際に表示するようなスクリーンショットを選択してください。',
        decoding: 'QR コードを読み取り中…',
        notFound:
          'この写真から QR コードが見つかりませんでした。コードを中央に配置し、ピントを合わせてから再度お試しください。',
        invalid: 'この QR コードには利用可能な Wi-Fi 情報が含まれていません。',
        pickerUnavailable:
          '何も開きませんか?このサインイン画面ではカメラや写真がブロックされている可能性があります。代わりにスマートフォンのブラウザで次のアドレスを開いてください:',
        confirmTitle: 'NanoKVM を {{ssid}} に接続しますか?',
        confirmDescription:
          'このネットワーク情報は QR コードから取得されました。接続する前に確認してください。',
        openNetwork: 'このネットワークにはパスワードがありません。',
        connectBtn: '接続',
        otherBtn: '別のネットワークを選択'
      }
    },
    notification: {
      browser: {
        title: '推奨ブラウザ',
        description:
          '最高のユーザー エクスペリエンスを実現するには、Chrome ブラウザの使用をお勧めします。すべての主要なブラウザ間での互換性を確保するよう努めていますが、一部の高度な機能は Chrome 以外では意図したとおりに動作しない場合があります。'
      },
      btnOk: 'OK'
    },
    screen: {
      title: '画面',
      video: 'ビデオモード',
      quality: '品質',
      qualityAuto: '自動',
      qualityLossless: 'ロスレス',
      qualityHigh: '高い',
      qualityMedium: '中くらい',
      qualityLow: '低い',
      scale: '倍率',
      noSignal: '映像信号がありません',
      captureDisabled:
        '映像キャプチャはオフになっています。「設定 > デバイス > 映像キャプチャ」で有効にしてください。',
      inconsistentVideoMode: '再生を一時停止しました。別のビデオモードが再生中です。',
      bandwidthWarning:
        '現在の映像設定に対してネットワーク帯域幅が不足している可能性があります。ビットレートまたはフレームレートを下げてください。',
      webrtcConnectionFailed:
        'WebRTC 接続を確立できませんでした。Direct 映像モードに切り替えてください。',
      webrtcUnavailable: {
        title: 'WebRTC を利用できません',
        description: 'ブラウザで WebRTC を有効にするか、Direct ビデオモードに切り替えてください。'
      },
      upload: 'アップロード',
      advanced: '詳細',
      loading: '読み込み中'
    },
    volume: {
      title: '音量'
    },
    microphone: {
      title: 'マイク',
      notMounted: 'マイクが取り付けられていません',
      tip1: '仮想マイクはリモート ホストにマウントされていません。今すぐマウントして有効にしますか?',
      tip2: 'これは、後から「設定 > デバイス > 仮想オーディオ」で管理することもできます。',
      confirm: '有効にする',
      cancel: 'キャンセル',
      noDevice:
        'このデバイスでマイクが検出されませんでした。マイクを接続してからもう一度お試しください。',
      permissionDenied:
        'マイクへのアクセスが拒否されました。ブラウザーの設定でマイクへのアクセスを許可してください。',
      notReadable: 'マイクが別のアプリケーションで使用中か、読み取ることができません。',
      overconstrained: 'このマイクは必要なオーディオ設定に対応していません。',
      securityBlocked:
        'このページではマイクへのアクセスがブロックされています。ブラウザーのセキュリティ設定を確認してください。',
      aborted: 'マイクの起動が中止されました。もう一度お試しください。',
      unsupported: 'このブラウザーまたは接続方法ではマイクを使用できません。',
      enableFailed: 'マイクを有効にできませんでした。もう一度お試しください。',
      disconnected: 'マイクが切断されたため、マイクの送信を停止しました。'
    },
    power: {
      title: '電源',
      showConfirm: '確認',
      showConfirmTip: '電源操作には追加の確認が必要です',
      powerShort: '電源（短いクリック）',
      powerLong: '電源（長いクリック）',
      powerConfirm: '電源操作を実行しますか？',
      okBtn: 'はい',
      cancelBtn: 'いいえ'
    },
    keyboard: {
      errors: {
        paste:
          'テキストをリモートデバイスに送信できません。KVM 接続を確認してから、もう一度お試しください。'
      },
      title: 'キーボード',
      paste: '貼り付け',
      tips: '標準的なキーボードの文字と記号のみがサポートされています',
      placeholder: '入力してください',
      submit: '送信',
      virtual: 'キーボード',
      swapCommandCtrl: 'Cmd と Ctrl を入れ替える',
      shortcut: {
        title: 'ショートカット',
        custom: 'カスタム',
        capture: 'ショートカットをキャプチャするにはここをクリックしてください',
        clear: 'クリア',
        save: '保存',
        captureTips:
          'Windows キーなどのシステムレベルのキーを取得するには、全画面表示の許可が必要です。',
        enterFullScreen: '全画面モードに切り替えます。'
      },
      leaderKey: {
        title: 'リーダーキー',
        desc: 'ブラウザの制限を回避して、システムによってブロックされているショートカットキーをリモートホストに送信します。',
        howToUse: '使用方法',
        simultaneous: {
          title: '同時モード',
          desc1: 'リーダーキーを押したまま、ショートカットを押します。',
          desc2:
            '操作は直感的ですが、システムの使用状況により一部のショートカットキーが機能しない場合があります。'
        },
        sequential: {
          title: 'シーケンシャルモード',
          desc1: 'リーダーキーを押す → ショートカットを順番に押す → リーダーキーをもう一度押す。',
          desc2: 'いくつかの手順が必要ですが、システムキーの競合を完全に回避します。'
        },
        enable: 'リーダーキーを有効化',
        tip: 'リーダーキーに設定すると、このキーはショートカットのトリガー専用になり、通常の動作は失われます。',
        placeholder: 'リーダーキーを押してください',
        shiftRight: '右 Shift',
        ctrlRight: '右 Ctrl',
        metaRight: '右 Win',
        submit: '送信',
        recorder: {
          rec: 'REC',
          activate: 'キーを有効化',
          input: 'ショートカットキーを押してください...'
        }
      }
    },
    mouse: {
      title: 'マウス',
      mode: 'マウスモード',
      absolute: '追従式マウス',
      absoluteAndroid: '追従式マウス（Android 互換性向上）',
      touchScreen: 'マルチタッチスクリーン',
      relative: '排他式マウス',
      direction: 'ホイール方向',
      scrollUp: '上',
      scrollDown: '下',
      speed: 'ホイール速度',
      fast: '速い',
      slow: '遅い',
      requestPointer:
        '相対モードを使用中です。デスクトップをクリックしてマウスポインタを取得してください。',
      repairIphoneDrag: 'iPhone のドラッグを修復',
      resetHid: 'HID をリセット',
      touchpadGuide: {
        title: 'タッチパッドガイド',
        scope: 'マウスモードが相対モードの場合の通常のポインター入力に適用されます。',
        swipeTitle: 'スワイプして移動',
        swipeDesc: 'アクティブな画面領域内でスワイプすると、リモートポインターが移動します。',
        tapTitle: 'タップしてクリック',
        tapDesc: '短くタップすると左クリックを 1 回送信します。',
        holdTitle: '長押しで左ボタンを押下',
        holdDesc: '約 1 秒間触れたまま動かさないと、マウスの左ボタンを押した状態になります。',
        dragTitle: '長押し後に動かしてドラッグ',
        dragDesc: '長押しが有効になった後、指を動かすと左ボタンを押したままドラッグします。'
      },
      hidOnly: {
        errors: {
          load: 'HID-Only モードの状態を読み込めません。しばらくしてからもう一度お試しください。',
          update:
            'HID-Only モードが更新されたか確認できません。設定を開き直して現在の状態を確認してから、もう一度お試しください。'
        },
        title: 'HID-Only モード',
        desc: '使用中にマウスとキーボードが反応しなくなり、HID をリセットしても効果がない場合は、NanoKVM とデバイス間の互換性に問題がある可能性があります。互換性を向上させるために、HID-Only モードを有効にすることをお勧めします。',
        tip1: 'HID-Only モードを有効にすると、仮想 U ディスクと仮想ネットワークがアンマウントされます',
        tip2: 'HID-Only モードでは、イメージのマウントは無効になります',
        enable: 'HID-Only モードを有効化',
        disable: 'HID-Only モードを無効化'
      }
    },
    preview: {
      title: 'プレビュー'
    },
    image: {
      storageReadOnly: 'ファイルストレージに書き込めません。仮想ディスクをオフにしてください。',
      storageBusy:
        'ストレージは使用中です。ファイル転送を完了またはキャンセルしてから、仮想ディスクを切り替えてください。',
      title: 'イメージ',
      loading: '読み込み中...',
      empty: '見つかりませんでした',
      upload: '画像をアップロード',
      dropFile: 'イメージをここにドロップするか、クリックして選択',
      uploadAction: 'アップロード',
      cancelAction: 'キャンセル',
      mount: 'イメージをマウント',
      mounted: '実装済み',
      uploadError: 'アップロードに失敗しました。もう一度お試しください。',
      uploadInsufficientStorage: 'ストレージ容量が不足しています',
      uploadTooLarge: 'イメージがサイズ制限を超えています',
      cancelUploadError: 'アップロードをキャンセルできませんでした。',
      mountMode: 'マウントモード',
      massStorage: 'マスストレージ',
      cdRom: 'CD ROM',
      readOnly: '読み取り専用',
      mountFailed: 'マウントに失敗しました',
      mountDesc:
        '一部のシステムでは、イメージをマウントする前にリモートホストで仮想ディスクをアンマウントする必要があります。',
      unmountFailed: 'アンマウント失敗',
      unmountDesc:
        '一部のシステムでは、イメージをアンマウントする前にリモートホストで手動で取り出す必要があります。',
      refresh: 'イメージリストを更新',
      download: '画像をダウンロード',
      attention: '注意',
      deleteConfirm: 'このイメージを削除してもよろしいですか？',
      deleteFailed: 'イメージを削除できませんでした',
      okBtn: 'はい',
      cancelBtn: 'いいえ'
    },
    script: {
      title: 'スクリプト',
      upload: 'アップロード',
      run: '実行',
      runBackground: 'バックグラウンドで実行',
      runFailed: '実行に失敗しました',
      uploadFailed: 'スクリプトをアップロードできませんでした',
      deleteFailed: 'スクリプトを削除できませんでした',
      attention: '注意',
      delDesc: 'このファイルを削除してもよろしいですか？',
      confirm: 'はい',
      cancel: 'いいえ',
      delete: '削除',
      close: '閉じる'
    },
    terminal: {
      title: 'ターミナル',
      nanokvm: 'NanoKVMターミナル',
      confirm: 'OK',
      submit: '送信',
      username: 'ユーザー名',
      password: 'パスワード',
      authCode: '認証コード',
      authorizeTitle: 'Web ターミナルを承認',
      codeAccountHint: '認証コードまたは未使用のリカバリーコードを入力してください。',
      authFailed: '認証に失敗しました'
    },
    wol: {
      errors: {
        send: 'Wake-on-LAN コマンドを送信できません。MAC アドレスとネットワーク接続を確認してから、もう一度お試しください。',
        loadFailed:
          'Wake-on-LAN デバイス一覧を読み込めません。NanoKVM がオンラインであることを確認してから、もう一度お試しください。',
        renameFailed:
          'このデバイスの名前を変更できません。MAC アドレスを確認してから、もう一度お試しください。',
        deleteFailed: 'このデバイスを削除できません。しばらくしてからもう一度お試しください。'
      },
      title: 'Wake-on-LAN',
      sending: 'コマンドを送信中...',
      sent: 'コマンドを送信しました',
      input: 'MACを入力してください',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256（任意）',
      sha256Placeholder: '64 文字の SHA-256 チェックサムを入力してください',
      invalidSHA256: 'SHA-256 は 64 文字の 16 進数文字列である必要があります',
      title: 'イメージダウンローダー',
      input: 'リモートイメージの URL を入力してください',
      ok: 'OK',
      disabled:
        '/data パーティションは読み取り専用であり、イメージのダウンロードには使用できません',
      downloadingFile: 'ダウンロード中: {{file}}',
      downloadingProgress: 'ダウンロード中 ({{percentage}}): {{file}}',
      failed: '失敗しました',
      insufficientStorage: 'ストレージ容量が不足しています',
      remaining: '残り {{time}}',
      cancel: 'キャンセル',
      cancelFailed: 'ダウンロードをキャンセルできませんでした',
      pause: '一時停止',
      resume: '再開',
      pauseFailed: 'ダウンロードを一時停止できませんでした',
      resumeFailed: 'ダウンロードを再開できませんでした'
    },
    settings: {
      back: '戻る',
      close: '閉じる',
      ai: {
        errors: {
          loadServices:
            'AI サービスの状態を読み込めません。NanoKVM がオンラインであることを確認してから、もう一度お試しください。',
          loadMcpConfig:
            'MCP サービスの設定を読み込めません。しばらくしてからもう一度お試しください。',
          updateMcpService:
            'MCP サービスの状態を更新できません。現在の状態を確認してから、もう一度お試しください。',
          accessMcpApiKey:
            'MCP API Key を取得できません。もう一度試し、求められた場合は認証を完了してください。',
          regenerateMcpApiKey:
            'MCP API Key を再生成できません。しばらくしてからもう一度お試しください。',
          regeneratedMcpApiKeyUnavailable:
            'API Key は再生成されましたが、表示できません。もう一度表示をお試しください。以前の Key は無効になりました。',
          loadMemoryFabricState:
            'Memory Fabric サービスの状態を読み込めません。しばらくしてからもう一度お試しください。',
          updateMemoryFabricService:
            'Memory Fabric サービスの状態を更新できません。現在の状態を確認してから、もう一度お試しください。',
          loadScreenTimelapseState:
            '画面タイムラプスサービスの状態を読み込めません。しばらくしてからもう一度お試しください。',
          updateScreenTimelapseService:
            '画面タイムラプスサービスの状態を更新できません。現在の状態を確認してから、もう一度お試しください。',
          openScreenTimelapse:
            '画面タイムラプス再生を開けません。しばらくしてからもう一度お試しください。',
          loadScreenTimelapseRecordings:
            '画面タイムラプス録画を読み込めません。しばらくしてからもう一度お試しください。',
          playScreenTimelapse:
            'このタイムラプス録画は現在再生できません。日付を選び直すか、しばらくしてからもう一度お試しください。',
          screenTimelapseFullscreen:
            '全画面モードを切り替えられません。ブラウザーの権限を確認してから、もう一度お試しください。'
        },
        title: 'AI',
        mcpService: 'MCP サービス',
        mcpServiceDesc: '制御、メディア、有効な Memory Fabric ツールの認証済み統合ゲートウェイ',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Agent のコンテキスト記憶と OCR をローカルで処理します',
        memoryFabricRemoteRequiresMcp:
          '記憶および OCR ツールをリモートで利用するには、上の MCP サービスを有効にしてください。',
        memoryFabricRemoteAvailable:
          '記憶および OCR ツールは、同じ API キーを使って上の /api/mcp エンドポイントから利用できます。',
        memoryFabricConfig: {
          title: '設定',
          description: 'Memory Fabric のデフォルトモデル設定のみを管理します。',
          loadFailed: 'Memory Fabric の設定を読み込めませんでした。',
          saveFailed: 'Memory Fabric の設定を保存できませんでした。',
          saved: '設定を保存しました。',
          savedRestarted: '設定を保存し、Memory Fabric を再起動しました。',
          noChanges: '設定に変更はありません。',
          reload: '再読み込み',
          save: '保存',
          unsaved: '未保存の変更があります',
          conflictTitle: '設定が変更されています',
          conflictDescription:
            '設定が別の場所で変更されました。もう一度保存する前に、最新版を読み込んでください。',
          conflictReload: '最新版を読み込む',
          discardTitle: '未保存の変更を破棄しますか？',
          discardDescription:
            '再読み込みすると、現在の入力内容が最新の保存済み設定に置き換わります。',
          discardConfirm: '破棄して再読み込み',
          sections: {
            models: 'モデル'
          },
          openAICompatibleHint:
            'OpenAI 互換のモデルプロバイダーに対応しています。プロバイダーから提供されたモデル名、ベース URL、API キーを入力してください。',
          fields: {
            model: 'モデル',
            baseUrl: 'ベース URL',
            maxTokens: '最大トークン数',
            timeoutSeconds: 'タイムアウト（秒）'
          },
          apiKey: {
            title: 'API キー',
            configured: '設定済み',
            notConfigured: '未設定',
            keep: '現在のキーを維持',
            replace: 'キーを置き換える',
            clear: 'キーを削除',
            replacement: '新しい API キー',
            replacementPlaceholder: '新しい API キーを入力',
            clearHint:
              '保存済みの API キーは削除されますが、環境変数からキーが提供される場合があります。',
            required: '新しい API キーを入力してください。'
          },
          validation: {
            required: 'この項目は必須です。',
            invalidUrl: '有効な HTTP または HTTPS URL を入力してください。'
          }
        },
        screenTimelapseService: '画面タイムラプス',
        screenTimelapseServiceDesc:
          'Memory Fabric の画面履歴を再生し、キャプチャ済み画面コンテンツを検索します',
        screenTimelapseSearchRequiresMemoryFabric:
          '録画内容を検索するには、Memory Fabric も有効にしてください。',
        screenTimelapseOpen: '開く',
        screenTimelapseSearchPlaceholder: '動画内の内容を検索...',
        screenTimelapseNoVideos: 'タイムラプス動画が見つかりません',
        screenTimelapseNoResults: '一致するフレームが見つかりません',
        screenTimelapseMultipleResults: '検索結果',
        screenTimelapseUnavailable: '利用不可',
        screenTimelapsePlayerUnsupported: 'このブラウザではタイムラプス動画を再生できません',
        screenTimelapseSearchFailed: '検索に失敗しました',
        screenTimelapseLoading: 'タイムラプスを読み込み中...',
        screenTimelapseSearchAction: 'タイムラプスを検索',
        screenTimelapsePlay: 'タイムラプスを再生',
        screenTimelapsePause: 'タイムラプスを一時停止',
        screenTimelapseEnterFullscreen: '全画面にする',
        screenTimelapseExitFullscreen: '全画面を終了',
        screenTimelapseFullscreenUnsupported: 'このブラウザでは全画面表示にできません',
        screenTimelapsePreviousDay: '前日',
        screenTimelapseNextDay: '翌日',
        screenTimelapseDeleteDay: '{{date}} の録画を削除',
        screenTimelapseDelete: '削除',
        screenTimelapseDeleteConfirmTitle: 'タイムラプス録画を削除しますか？',
        screenTimelapseDeleteConfirmDesc:
          '{{date}} の録画は {{size}} を使用しています。この操作は元に戻せません。',
        screenTimelapseDeleteSuccess: '{{date}} の録画を削除しました',
        screenTimelapseDeleteFailed: '{{date}} の録画を削除できませんでした',
        endpoint: 'エンドポイント',
        apiKey: 'API キー',
        regenerate: '再生成',
        copyApiKey: 'API キーをコピー',
        copyEndpoint: 'エンドポイントをコピー',
        showApiKey: 'API キーを表示',
        regenerateConfirmTitle: 'API キーを再生成',
        regenerateConfirmDesc:
          '既存の MCP クライアントは、新しいキーで更新されるまで動作しなくなります。続行しますか？',
        okBtn: 'はい',
        cancelBtn: 'いいえ'
      },
      title: '設定',
      about: {
        title: 'NanoKVMについて',
        information: '情報',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'アプリケーションバージョン',
        applicationTip: 'NanoKVM Web アプリケーションのバージョン',
        arch: 'アーキテクチャ',
        device_number: 'デバイス番号',
        image: 'イメージバージョン',
        imageTip: 'NanoKVM システムイメージのバージョン',
        deviceKey: 'デバイスキー',
        community: 'コミュニティ',
        hostname: 'ホスト名',
        hostnameUpdated: 'ホスト名は正常に変更され、再起動後に有効になります',
        ipType: {
          Wired: '有線',
          Wireless: 'ワイヤレス',
          Other: 'その他'
        }
      },
      appearance: {
        title: '外観',
        display: 'ディスプレイ',
        language: '言語',
        languageDesc: 'インターフェース言語の選択',
        languageChangeFailed: '言語の変更に失敗しました',
        webTitle: 'ウェブページタイトル',
        webTitleDesc: 'ウェブページタイトルのカスタマイズ',
        menuBar: {
          title: 'メニューバー',
          mode: '表示モード',
          modeDesc: 'メニューバーの画面表示方法',
          modeOff: '閉じる',
          modeAuto: '自動非表示',
          modeAlways: '常に表示',
          icons: 'メニューアイコン',
          showIcon: 'アイコンを表示',
          activateOnConnect: '接続時に有効化',
          stopSharing: '共有を停止',
          iconsDesc: 'メニューバーでのサブメニューアイコンの表示'
        }
      },
      screen: {
        title: '画面',
        videoMode: {
          title: 'ビデオモード',
          description: 'エンコードと送信方法を選択します'
        },
        edid: {
          description: '表示パラメータの選択',
          default: 'デフォルト',
          custom: 'カスタム',
          document: 'ドキュメント',
          powerConfirmTitle: 'EDID を更新する前に確認',
          powerConfirmDescription:
            '先に PD 電源を接続してください。EDID の更新中にディスプレイが一時的に切断されることがあります。更新後にデバイスがディスプレイを正しく検出できない場合は、手動で再起動してください。',
          powerConfirmOk: '続行',
          powerConfirmOkCountdown: '続行 ({{seconds}})',
          powerConfirmCancel: 'キャンセル',
          errors: {
            badRequest: 'EDID リクエストが無効です',
            invalidFileName: '有効なファイル名の .bin EDID ファイルを選択してください',
            fileTooLarge: 'EDID ファイルが大きすぎます',
            invalidLength: 'EDID は 128、256、384、または 512 バイトである必要があります',
            invalidHeader: 'EDID ヘッダーが無効です',
            unsupportedVersion: 'この EDID バージョンはサポートされていません',
            extensionCountMismatch: 'EDID 拡張ブロック数がファイルの長さと一致しません',
            invalidChecksum: 'EDID チェックサムが無効です',
            invalidCta: 'CTA 拡張の構造が無効です',
            currentCannotOverwrite:
              '現在使用中の EDID は上書きできません。先に別の EDID に切り替えてください',
            saveFailed: 'EDID を保存できませんでした',
            switchFailed: 'EDID を切り替えられませんでした',
            uploadFailed: 'EDID をアップロードできませんでした',
            deleteFailed: 'EDID を削除できませんでした',
            networkFailed: 'ネットワークエラーです。もう一度お試しください'
          }
        },
        rateControlMode: {
          title: 'レート制御モード',
          description: 'ビットレートを制御する方法を選択します',
          cbr: 'CBR (固定ビットレート): 接続の安定性を優先します。途切れを最小限に抑えますが、動きの激しいシーンでは画質が低下する可能性があります。',
          vbr: 'VBR（可変ビットレート）：画像の鮮明さを優先します。動きの多いシーンでは帯域幅のスパイクが発生し、遅延や途切れが発生する可能性があります。'
        },
        bitrate: {
          title: 'ビットレート',
          description: '画像の鮮明さと帯域幅の使用量のバランスをとる',
          lower:
            '値を低くすると、貧弱なネットワークでの遅延は減少しますが、画質が低下し、ぼやけやアーティファクトが発生する可能性があります。',
          higher:
            '値を大きくすると画像の鮮明さが向上しますが、より多くの帯域幅を消費します。接続が遅い場合、遅延が増加する可能性があります。'
        },
        gop: {
          description: '完全なキーフレーム間の間隔を設定します',
          lower:
            '値を小さくすると、完全なフレームがより頻繁に送信されます。これにより、視覚的なアーティファクトからの回復が高速化されますが、より多くの帯域幅が消費され、画像の詳細が低下します。',
          higher:
            '値を大きくすると、増分更新が最適化されます。これにより、画像の詳細が改善され、帯域幅が節約されますが、視覚的なアーティファクトからの回復には時間がかかります。'
        },
        quality: {
          title: '品質',
          description: '表示画質を設定する'
        },
        fps: {
          auto: '自動',
          description: '表示フレームレートを設定します',
          tip1: '値を 0 に設定すると自動モードが有効になり、表示がリモート ホストのフレーム レートと同期します。',
          tip2: 'フレーム レートを下げると、帯域幅の使用量が大幅に削減されます。ネットワークが不安定なとき、または高いフレーム レートが必要ないときは、手動で値を下げます。'
        },
        scale: {
          title: 'スケール',
          description: '表示倍率を設定します'
        }
      },
      device: {
        title: 'デバイス',
        virtualAudioConfirm: {
          title: '仮想オーディオを切り替える前に確認',
          description:
            '先に PD 電源を接続してください。仮想オーディオの切り替え中にディスプレイが一時的に切断されることがあります。切り替え後にデバイスがディスプレイを正しく検出できない場合は、手動で再起動してください。',
          ok: '切り替えを続行',
          okCountdown: '切り替えを続行 ({{seconds}})',
          cancel: 'キャンセル'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi の構成',
          apMode: 'AP モードが有効になっており、Wi-Fi に手動で接続します',
          others: 'その他...',
          connect: 'Wi-Fi に接続',
          connectDesc1: 'SSID とパスワードを入力してください',
          connectDesc2: 'このネットワークに接続するためのパスワードを入力してください',
          disconnect: 'このネットワーク接続を切断しますか？',
          ssid: 'SSID',
          password: 'パスワード',
          joinBtn: '接続',
          confirmBtn: 'OK',
          cancelBtn: 'キャンセル',
          preset: {
            title: 'Wi-Fi プリセット',
            description: '接続先の Wi-Fi を 1 件事前登録すると、起動時に自動で接続します',
            empty: 'なし',
            addBtn: 'プリセットを追加',
            removeBtn: 'プリセットを削除',
            saveTitle: 'Wi-Fi を事前登録',
            saveDesc:
              '対象ネットワークを今すぐ保存すると、起動時に検出された場合に自動で接続します',
            saveBtn: '保存',
            passwordPlaceholder: 'パスワード（オープンネットワークは空欄）',
            saveFailed: 'Wi-Fi プリセットの保存に失敗しました',
            removeFailed: 'Wi-Fi プリセットの削除に失敗しました',
            removeConfirm: '事前登録した Wi-Fi を削除しますか？',
            warning:
              '保存すると、端末は起動時にこの Wi-Fi を検出した際に常に接続を試みます。これにより他の Wi-Fi への接続を妨げる場合があります。プリセットは変更または削除するまで有効です。'
          }
        },
        capture: {
          title: '映像キャプチャ',
          description: '映像キャプチャを有効にする'
        },
        ssh: {
          description: 'SSH リモートアクセスを有効にする',
          tip: 'SSH を有効にした後、ターミナルで root ユーザーに強力なパスワードを設定してください。'
        },
        swap: {
          errors: {
            load: 'スワップメモリの状態を読み込めません。しばらくしてからもう一度お試しください。',
            update:
              'スワップメモリの設定を更新できません。現在の状態を確認してから、もう一度お試しください。'
          },
          title: 'スワップ',
          description: '圧縮スワップメモリを有効にする'
        },
        advanced: '詳細設定',
        staticIp: {
          title: '静的 IP',
          config: '設定',
          tip: '接続の切断を防ぐため、デバイスに物理的にアクセスできるときに設定することをお勧めします。',
          document: '文書を確認してください。',
          enable: '静的 IP を有効にする',
          placeholder: '形式: IP/CIDR [ゲートウェイ]\n例: 192.168.1.10/22',
          notEmpty: 'IP アドレスを空にすることはできません。',
          notValid: 'IP アドレスの形式が無効です。',
          failed: '設定に失敗しました。',
          success: '設定が完了しました。 IPアドレスが使用可能かどうかを確認してください。',
          okBtn: 'わかりました',
          cancelBtn: 'キャンセル'
        },
        usbNetwork: {
          title: 'USB ネットワーク',
          mode: 'モード',
          server: 'サーバー',
          serverDesc: '内蔵 DHCP サーバーからリモートホストに IP アドレスを割り当てます。',
          client: 'クライアント',
          clientDesc: 'DHCP を使用してリモートホストから IP アドレスを取得します。',
          forwarding: 'IP 転送',
          forwardingTip:
            'リモートホストから NanoKVM 経由でインターネットにアクセスするには、IP 転送を有効にしてください。',
          manualIp: 'IP アドレスを手動設定',
          ipv4Placeholder: 'IPv4/CIDR（例：10.10.10.1/24）',
          invalidIpv4: '有効な IPv4/CIDR アドレスを入力してください。',
          loadFailed: 'USB ネットワーク設定を読み込めません。',
          failed: '設定に失敗しました。',
          success: '設定が完了しました。',
          applyBtn: '適用',
          cancelBtn: 'キャンセル'
        },
        usbId: {
          title: 'USB デバイス識別情報',
          config: '設定',
          tip: 'カスタム USB ID はホストでのデバイス識別方法を変更します。適用すると USB ガジェットが再起動し、キーボード・マウス・ネットワークが一時的に切断されることがあります。',
          enable: 'カスタム USB デバイス識別情報を有効化',
          vid: 'ベンダー ID (VID)',
          pid: 'プロダクト ID (PID)',
          bcdDevice: 'デバイスバージョン (bcdDevice)',
          serialnumber: 'シリアル番号',
          manufacturer: 'メーカー',
          product: '製品名',
          configurationName: 'USB 構成名',
          vidPlaceholder: '例: 0x046D',
          pidPlaceholder: '例: 0x0A4E',
          bcdDevicePlaceholder: '例: 0x0521（任意）',
          serialnumberPlaceholder: '例: 2214GG01KWF8（任意）',
          manufacturerPlaceholder: '例: Logitech（任意）',
          productPlaceholder: '例: Logitech P710e（任意）',
          notEmpty: 'VID と PID は空にできません。',
          notValid: 'VID/PID の形式が無効です。0x046D のような 16 進数を使用してください。',
          bcdNotValid:
            'bcdDevice の形式が無効です。0x0521 のような 16 進数を使うか、空のままにしてください。',
          failed: '設定に失敗しました。',
          success: '設定が完了しました。USB ガジェットを再起動しました。',
          okBtn: 'OK',
          cancelBtn: 'キャンセル'
        },
        diskInquiry: {
          title: 'USB ドライブ名',
          config: '設定',
          tip: 'リモート PC に表示される仮想 USB メモリの名前です。保存すると USB が再起動し、キーボード・マウス・ネットワークが一時的に切断される場合があります。',
          enable: 'USB ドライブ名をカスタマイズ',
          placeholder: '例: LogitechFlash Drive     1.00',
          notEmpty: '名前は空にできません。',
          notValid: '名前が無効です。改行なしの印刷可能な ASCII 文字で最大 28 文字です。',
          failed: '設定に失敗しました。',
          success: '設定が完了しました。USB ガジェットを再起動しました。',
          okBtn: 'OK',
          cancelBtn: 'キャンセル'
        },
        diskInquiryCdrom: {
          title: '仮想 CD ドライブ名',
          config: '設定',
          tip: 'リモート PC に表示される仮想 CD/DVD ドライブの名前です。保存すると USB が再起動し、キーボード・マウス・ネットワークが一時的に切断される場合があります。',
          enable: '仮想 CD ドライブ名をカスタマイズ',
          placeholder: '例: LogitechOptical Drive   1.00',
          notEmpty: '名前は空にできません。',
          notValid: '名前が無効です。改行なしの印刷可能な ASCII 文字で最大 28 文字です。',
          failed: '設定に失敗しました。',
          success: '設定が完了しました。USB ガジェットを再起動しました。',
          okBtn: 'OK',
          cancelBtn: 'キャンセル'
        },
        uac2FunctionName: {
          title: 'USB オーディオ名',
          config: '設定',
          tip: 'リモート PC に表示される仮想 USB マイク / オーディオ機器の名前です。保存すると USB が再起動し、キーボード・マウス・ネットワークが一時的に切断される場合があります。',
          enable: 'USB オーディオ名をカスタマイズ',
          placeholder: '例: Logitech P710e',
          notEmpty: '名前は空にできません。',
          notValid: '名前が無効です。改行なしの印刷可能文字で最大 64 文字です。',
          failed: '設定に失敗しました。',
          success: '設定が完了しました。USB ガジェットを再起動しました。',
          okBtn: 'OK',
          cancelBtn: 'キャンセル'
        },
        lowPower: {
          title: '低電力モード',
          description: 'アイドル時の電力使用量を削減します',
          tips: 'ヒント',
          tip1: '有効にすると、消費電力と発熱を抑えるため、アイドル時に映像キャプチャが一時停止します。',
          tip2: '映像キャプチャの再開には再初期化が必要になる場合があり、その間は映像が一時的に利用できなくなることがあります。',
          okBtn: 'わかりました',
          cancelBtn: 'キャンセル'
        },
        webcamUsbDisplay: 'ウェブカメラを有効にする前に USB ディスプレイモードをオフにしてください。',
        webcamEndpoints: 'ウェブカメラには空き USB IN エンドポイントが 2 つ必要です。',
        mouseJiggler: {
          noMouse: 'この構成ではマウスジグラーを使用できません。マウスモードを選択するか、単一エンドポイントモードをオフにしてください。',
          title: 'マウスジグラー',
          description: 'リモートホストの休止を防ぐ',
          disable: '閉じる',
          absolute: '絶対モード',
          relative: '相対モード'
        },
        fingerRobot: {
          title: '指ロボット',
          description: '指ロボット制御を有効にする'
        },
        mdns: {
          description: 'mDNS 検出サービスを有効にする'
        },
        hidOnly: 'HID-Only モード',
        imageMounted: 'イメージがマウントされました',
        disk: '仮想ディスク',
        diskDesc: 'リモートホストに仮想 U ディスクをマウントする',
        sdCard: {
          warning: '仮想ディスクを初期化',
          tip1: '初回の有効化では仮想ディスクを初期化し、デバイス上の既存のイメージファイルと仮想ディスクのデータを消去します。この操作は元に戻せません。重要なファイルを先にバックアップしてください。',
          tip2: '初期化を続行しますか？',
          capacity: '容量',
          disable: '無効化',
          disableWarning: '仮想ディスクを無効化しますか？',
          disableTip:
            '無効化すると仮想ディスク内のすべてのファイルが消去され、使用しているデバイスのストレージ容量が解放されます。次回有効化するときは、容量を再度選択して初期化する必要があります。この操作は元に戻せません。'
        },
        network: '仮想ネットワーク',
        networkDesc: 'リモートホストに仮想ネットワークカードをマウントする',
        mic: '仮想オーディオ',
        micDesc: 'リモートホストに提供する USB 仮想オーディオモードを選択します',
        virtualAudioMode: {
          off: 'オフ',
          mic: 'マイクのみ',
          duplex: 'マイク + スピーカー'
        },
        speaker: '仮想スピーカー',
        speakerDesc:
          '有効にすると既定の DP 音声出力を置き換え、互換性の高いリモート通話に対応します。無効の場合はメディア音声のみ転送でき、通話音声は転送できません。',
        micTip:
          'USB エンドポイントの制限により、仮想マイクを有効にすると USB セカンダリ スクリーンが無効になります。\nまた、最適な互換性を確保するために、BIOS 操作中はこの機能を無効にすることをお勧めします。',
        reboot: '再起動',
        rebootDesc: 'NanoKVM を再起動してもよろしいですか?',
        rebootToFlashing: '書き込みモードで再起動',
        rebootToFlashingButton: '再起動',
        rebootToFlashingTitle: '書き込みモードで再起動しますか？',
        rebootToFlashingDesc:
          'NanoKVM は再起動して USB 書き込みモードに入ります。書き込みに使用する USB インターフェースが接続され、書き込みツールの準備ができていることを確認してください。',
        datetime: {
          timezone: 'タイムゾーン',
          datetime: '日時',
          dateAndTime: '日付と時刻',
          synchronize: '時刻を同期する',
          lastSynchronization: '前回の同期: ',
          notSynchronized: 'システム時刻はまだ同期されていません',
          syncNow: '今すぐ同期',
          manualSet: '手動設定',
          manualSetTitle: '時刻を手動で設定',
          manualSetDescription: '変更後は再度ログインする必要があります。',
          useBrowserTime: 'ブラウザーの時刻を使用',
          setTime: '時刻を設定',
          invalidTime: '有効な日付と時刻を入力してください。',
          setFailed: 'システム時刻を設定できません。もう一度お試しください。',
          setSucceeded: 'システム時刻を更新しました。もう一度ログインしてください。'
        },
        dialBackground: {
          errors: {
            load: 'ダイヤル背景画像を読み込めません。しばらくしてからもう一度お試しください。',
            upload:
              '背景画像をアップロードできません。有効な BMP 画像か確認してから、もう一度お試しください。',
            switch:
              'ダイヤル背景を切り替えられません。画像一覧を更新してから、もう一度お試しください。'
          },
          title: 'ダイヤル背景',
          description: '/kvmcomm/ui/bg/ の BMP 画像をダイヤル背景として選択します',
          upload: 'アップロード',
          loading: '読み込み中...',
          empty: 'BMP 画像が見つかりません',
          current: '現在',
          onlyBmp: 'BMP 画像のみサポートされています',
          uploaded: 'アップロード完了',
          switched: '背景を切り替えました'
        },
        okBtn: 'はい',
        cancelBtn: 'いいえ'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Tailscale を再起動しますか？',
        stop: 'Tailscale を停止しますか？',
        stopDesc: 'Tailscale からログアウトし、起動時の自動開始を無効にします。',
        loading: '読み込み中...',
        notInstall: 'Tailscaleが見つかりません！インストールしてください。',
        install: 'インストール',
        installing: 'インストール中',
        failed: 'インストールに失敗しました',
        retry: 'ページをリフレッシュして再試行してください。または手動でインストールしてください',
        download: 'インストールパッケージをダウンロードして',
        package: '解凍してください',
        unzip: '解凍してください',
        upTailscale: 'tailscaleをNanoKVMのディレクトリ/usr/bin/にアップロードしてください',
        upTailscaled: 'tailscaledをNanoKVMのディレクトリ/usr/sbin/にアップロードしてください',
        refresh: '現在のページをリフレッシュします',
        notRunning: 'Tailscale はまだ実行されていません。起動操作を実行してください',
        run: '開始',
        notLogin:
          'デバイスはまだバインドされていません。ログインしてこのデバイスをアカウントにバインドしてください。',
        urlPeriod: 'このURLは10分間有効です',
        login: 'ログイン',
        loginSuccess: 'ログイン成功',
        enable: 'Tailscaleを有効化',
        exitNode: '出口ノード',
        exitNodeDesc: 'このデバイスを tailnet のインターネット出口として公開します。',
        subnetRoute: 'サブネットルート',
        subnetRouteDesc: '直接接続された LAN サブネット {{subnet}} を tailnet に公開します。',
        subnetUnavailable: '公開できる直接接続 LAN サブネットがありません。',
        routingUnavailable: 'このデバイスでは Tailscale のルーティング設定を利用できません。',
        pendingApproval: '有効化後、Tailscale 管理コンソールで公開されたルートを承認してください。',
        routingUpdateFailed: 'Tailscale のルーティング設定を更新できませんでした。',
        deviceName: 'デバイス名',
        deviceIP: 'デバイスIP',
        account: 'アカウント',
        logout: 'ログアウト',
        logoutDesc: 'ログアウトしてもよろしいですか？',
        uninstall: 'Tailscale をアンインストール',
        uninstallDesc: 'Tailscale をアンインストールしてもよろしいですか？',
        okBtn: 'はい',
        cancelBtn: 'いいえ'
      },
      update: {
        title: 'アップデートの確認',
        queryFailed: 'バージョンの取得に失敗しました',
        updateFailed: '更新に失敗しました。再試行してください。',
        isLatest: '最新のバージョンを既に持っています。',
        available: 'アップデートが利用可能です。更新してもよろしいですか？',
        updating: '更新を開始しました。お待ちください...',
        confirm: '確認',
        upgrade_tip_nano:
          'NanoKVM の新しいバージョンが利用可能です。今すぐ更新してもよろしいですか?',
        upgrade_tip_firmware:
          '新しいファームウェア バージョンが利用可能です。今すぐ更新してもよろしいですか?',
        cancel: 'キャンセル',
        preview: 'プレビューアップデート',
        previewDesc: '新機能や改善をいち早く体験する',
        previewTip:
          'プレビューアップデートには不安定な部分や不完全な機能が含まれる場合があります！',
        download: 'ダウンロード',
        install: 'インストール',
        changelog: '変更履歴',
        restart: '再起動'
      },
      account: {
        title: 'アカウント',
        webAccount: 'Web アカウント名',
        password: 'パスワード',
        twoFactor: '2要素認証',
        enableTwoFactor: '有効化',
        enableTwoFactorDesc: '2要素認証の設定を開始するには、現在のパスワードを入力してください。',
        disableTwoFactor: '無効化',
        recoveryCodes: 'リカバリーコード',
        enableRecoveryCodes: '有効化',
        updateRecoveryCodes: '更新',
        recoveryCodesTooltipDisabled:
          'リカバリーコードは2要素認証のバックアップ認証情報です。2要素認証を有効化した後に生成できます。',
        recoveryCodesTooltipEnabled:
          'リカバリーコードは2要素認証が有効な間だけ有効です。2要素認証を無効化するとすべてのリカバリーコードが無効になります。有効なままの場合は更新のみ可能です。',
        recoveryCodesRegenerateDesc:
          '現在のパスワードと、認証コードまたは未使用のリカバリーコードのいずれかを入力してください。リカバリーコードを更新すると、既存のすべてのリカバリーコードが無効になります。',
        recoveryCodesVerifyFailed:
          'パスワードと認証コードまたはリカバリーコードを確認できませんでした。',
        recoveryCodesSavedTitle: '今すぐリカバリーコードを保存してください',
        recoveryCodesSavedDesc:
          'リカバリーコードは一度しか表示できません。このページを離れる前に安全な場所に保存してください。',
        enableRecoveryCodesDesc:
          'リカバリーコードを生成するには、現在のパスワードと認証コードを入力してください。リカバリーコードは一度しか表示できないため、このページを離れる前に保存してください。',
        copyRecoveryCodes: 'コピー',
        downloadRecoveryCodes: 'ダウンロード',
        recoveryCodesCopyFailed: 'リカバリーコードをコピーできませんでした。',
        done: '完了',
        currentPassword: '現在のパスワード',
        authCode: '認証コード',
        authCodeOrRecoveryCode: '認証コードまたはリカバリーコード',
        authRateLimited: '失敗回数が多すぎます。{{seconds}} 秒後に再試行してください。',
        verifyPasswordChange: 'パスワード変更を確認',
        verifyPasswordChangeDesc:
          'パスワードを変更する前に、現在のパスワードと、認証コードまたは未使用のリカバリーコードのいずれかを入力してください。',
        verifyPasswordChangeFailed:
          'パスワードと認証コードまたはリカバリーコードを確認できませんでした。',
        verifyAndEnable: '確認して有効化',
        scanTwoFactorQRCode: '認証アプリでこの QR コードをスキャンし、コードを入力してください。',
        showSetupKey: 'セットアップキーを表示',
        hideSetupKey: 'セットアップキーを非表示',
        setupKey: 'セットアップキー',
        twoFactorSetupFailed: '2要素認証の設定を開始できませんでした。',
        twoFactorVerifyFailed: '認証コードを確認できませんでした。',
        twoFactorDisableFailed: '2要素認証を無効化できませんでした。',
        disableTwoFactorDesc:
          '現在のパスワードと、認証コードまたは未使用のリカバリーコードのいずれかを入力してください。2要素認証を無効化すると、すべてのリカバリーコードが無効になります。',
        updateBtn: '変更',
        logoutBtn: 'ログアウト',
        logoutDesc: 'ログアウトしてもよろしいですか？',
        okBtn: 'はい',
        cancelBtn: 'いいえ',
        twoFactorProtection: {
          title: '高度な 2FA 保護',
          description:
            '保護の強制が有効なときに、二要素目を必要とする機密操作を選択します。「直接 Web ログイン」スイッチは逆の動作で、認証コードまたは未使用のリカバリーコードだけで Web にログインできるようにします。',
          login: '直接 Web ログイン',
          webTerminal: 'Web ターミナル',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API キー',
          targetPower: '対象マシンの電源',
          nanoKVMRestart: 'NanoKVM の再起動',
          deleteImage: 'イメージの削除',
          scripts: 'スクリプト操作',
          screenTimelapse: '画面タイムラプス',
          loginScope:
            'ユーザー名とパスワードの代わりに、認証コードまたは未使用のリカバリーコードだけで Web にログインできます。ログインの安全性は低下します。',
          webTerminalScope: 'Web ターミナルを開く。',
          sshScope: 'SSH サービスの状態を変更する。',
          tailscaleScope: 'Tailscale をインストール、接続、または変更する。',
          frp: 'FRP',
          frpScope: 'FRP をインストール、接続、または変更する。',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Cloudflare Tunnel をインストール、接続、または変更する。',
          zerotierScope: 'ZeroTier をインストール、接続、または変更する。',
          netbirdScope: 'NetBird をインストール、接続、または変更する。',
          mcpScope: 'MCP API キーの表示、コピー、再生成、または MCP の有効化・無効化。',
          targetPowerScope: '物理電源ボタンで対象マシンをシャットダウンまたは再起動します。',
          nanoKVMRestartScope: 'NanoKVM 自体を再起動します。',
          deleteImageScope: '保存済みの ISO または IMG イメージを完全に削除します。',
          scriptsScope: 'スクリプトの実行、アップロード、または削除。',
          screenTimelapseScope: '画面タイムラプス録画の有効化、録画の閲覧と削除。',
          authorizeTitle: '保護設定の変更を承認',
          authorizeDesc:
            '認証コード、または未使用のリカバリーコードを入力してください。この承認は 5 分間有効です。',
          authorizeFailed: '認証コードまたはリカバリーコードを確認できませんでした。',
          loadFailed: '保護設定を読み込めません。',
          updateFailed: '保護設定を保存できません。',
          retry: '再試行',
          disableScriptsWarningTitle: 'スクリプト保護を無効にしますか？',
          disableScriptsWarningDesc:
            '保護がない場合、悪意のあるスクリプトがこのデバイスへの昇格されたアクセスを取得し、2FA 保護を回避または削除する可能性があります。',
          cancelDisabling: 'キャンセル',
          continueDisabling: '無効化を続行',
          continueDisablingCountdown: '無効化を続行 ({{seconds}} 秒)'
        },
        twoFactorGuide: {
          title: '2要素認証ガイド',
          scope: '2要素認証はアカウントに追加のセキュリティ層を加えます。',
          whatTitle: '2FAとは？',
          whatDesc:
            '2要素認証（2FA）は、認証アプリが生成する時間ベースのコードでアカウントを保護します。既定ではこのコードはパスワードに加えて必要となるため、パスワードが漏えいしてもアカウントは保護されます。直接 Web ログインが有効な場合は、コードだけでログインできます。',
          appsTitle: '推奨アプリ',
          appsDesc:
            'Google Authenticator または Microsoft Authenticator を使用できます。どちらも無料で、iOS と Android で利用できます。',
          setupTitle: '設定方法',
          setupDesc:
            '認証アプリを開き、設定中に表示されるQRコードをスキャンするか、セットアップキーを手動で入力します。その後、アプリの6桁コードを入力して2FAの有効化を完了します。',
          recoveryTitle: 'リカバリーコード',
          recoveryDesc:
            'リカバリーコードは1回限りのバックアップコードです。認証アプリにアクセスできなくなった場合、リカバリーコードでログインできます。生成後すぐに安全な場所へ保存してください。',
          logoutNotice:
            '2FAを有効にすると、このセッションを含むすべてのアクティブなセッションがログアウトされます。各デバイスで再度サインインする必要があります。'
        }
      }
    },
    error: {
      title: 'エラーが発生しました',
      refresh: '更新'
    },
    pictureInPicture: {
      title: 'ピクチャー イン ピクチャー',
      enter: 'ピクチャー イン ピクチャーを開始',
      exit: 'ピクチャー イン ピクチャーを終了',
      error:
        'ピクチャー イン ピクチャーを開始できませんでした。動画の読み込みを待ってから再試行してください。'
    },
    fullscreen: {
      toggle: '全画面表示切り替え'
    },
    menu: {
      collapse: 'メニューを折りたたむ',
      expand: 'メニューを展開する'
    }
  }
};

export default ja;
