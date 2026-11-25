const zh_tw = {
  translation: {
    storage: {
      cards: { mount: '磁碟掛載', share: '檔案分享' },
      title: '映像檔掛載 / 檔案共享',
      empty: '暫無映像檔',
      notAttached: '尚未裝載任何媒體',
      attached: '已裝載：{{name}}',
      eject: '彈出',
      mount: '裝載',
      mountMode: '裝載模式',
      mountModeDisk: '磁碟模式',
      mountModeCdrom: '光碟模式',
      openManager: '開啟檔案管理員',
      refresh: '重新整理',
      delete: '刪除',
      deleteTitle: '刪除媒體',
      deleteConfirm: '確定刪除 {{name}}？裡面的檔案會一起遺失。',
      downloadImage: '下載映像檔',
      typePartition: '共享檔案',
      volumeLabel: '可移除磁碟',
      groups: {
        share: '檔案共享',
        images: '映像檔'
      },
      failed: '操作失敗',
      hidOnly: 'HID-Only 模式：虛擬隨身碟無法使用',
      disabled: '虛擬隨身碟未啟用（設定 - 設備）',
      force: {
        title: '強制收回媒體？',
        desc: '受控裝置還掛著這個媒體。強制收回會讓 USB 裝置重新列舉，它還沒寫入磁碟的資料可能會遺失。',
        menuHint: '受控裝置正在使用該媒體，請到檔案管理員頁面強制收回。'
      },
      tips: {
        title: '如何上傳',
        usb1: '透過 USB 將 NanoKVM 連接到您的電腦。',
        usb2: '在本頁面以大容量儲存模式掛載共享檔案；',
        usb3: '開啟電腦上的虛擬磁碟，將映象檔案複製到虛擬磁碟的根目錄下。',
        usb4: '複製完成後退出磁碟，映像檔隨後會出現在清單中。',
        scp1: '確保 NanoKVM 和您的電腦位於同一區域網路。',
        scp2: '開啟電腦上的終端機，使用 SCP 指令將映像檔案上傳到 NanoKVM 的 /data 目錄下。',
        scp3: '範例：scp your-image-path root@your-nanokvm-ip:/data'
      },
      job: {
        running: '正在處理 {{name}}'
      },
      enableDisk: {
        title: '啟用虛擬隨身碟？',
        desc: '裝載媒體需要虛擬隨身碟，目前它是關閉的。現在啟用？受控裝置的鍵盤與滑鼠會短暫中斷。',
        menuHint: '虛擬隨身碟未啟用，請到 設定 - 設備 開啟。'
      },
      add: {
        title: '新增媒體',
        upload: '上傳 ISO / IMG',
        fromUrl: '從 URL 下載',
        url: '映像檔網址',
        urlRequired: '請輸入映像檔網址',
        onlyImages: '只能掛載 .iso 和 .img 檔案',
        dropHint: '把 ISO / IMG 拖到這裡，或點擊選擇檔案',
        cdromOnlyIso: '光碟模式只能掛載 ISO / UDF 光碟映像檔',
        uploadFailed: '上傳失敗',
        downloadFailed: '下載失敗',
        uploading: '正在上傳 {{name}}',
        downloading: '正在下載 {{name}}',
        done: '{{name}} 已就緒',
        checksumFailed: '映像檔與 SHA-256 校驗值不符。',
        blocked:
          '受控裝置正在使用這張磁碟。請先在受控裝置上安全退出虛擬隨身碟，再到設定 > 設備關閉虛擬隨身碟開關。'
      },
      files: {
        reasonNotInitialized: '虛擬隨身碟尚未初始化，請先到設定 > 裝置初始化虛擬隨身碟。',
        selectMedium: '選擇一個媒體，查看裡面的檔案。',
        reasonAttached: '該媒體正裝載給受控裝置使用。',
        reasonAttachedHint:
          '請先在受控裝置上安全退出虛擬隨身碟，再到設定 > 設備關閉虛擬隨身碟開關。',
        reasonNotBrowsable: '這個媒體裡的檔案系統無法在網頁端開啟。',
        reasonNoLoop: '本機不支援掛載磁碟映像檔（核心未啟用 loop）。',
        reasonISOImage: 'ISO 光碟映像檔不支援在網頁端瀏覽目錄，映像檔本身仍可正常掛載給受控裝置',
        empty: '這個資料夾是空的',
        noMatch: '沒有符合的檔案',
        free: '剩餘 {{size}}',
        search: '搜尋目前資料夾',
        sort: {
          name: '名稱',
          size: '大小',
          type: '類型',
          modified: '修改時間',
          asc: '升冪',
          desc: '降冪'
        },
        view: {
          list: '清單檢視',
          grid: '圖示檢視'
        },
        folder: '資料夾',
        itemCount: '{{count}} 個項目',
        itemCount_one: '{{count}} 個項目',
        itemCount_other: '{{count}} 個項目',
        dropHint: '拖曳檔案到此處即可上傳到目前資料夾',
        up: '回到上一層',
        upload: '上傳檔案到媒體',
        uploadButton: '上傳檔案',
        newFolder: '新增資料夾',
        newFolderPlaceholder: '資料夾名稱',
        create: '建立',
        deleteFolderTitle: '刪除資料夾',
        deleteFolderConfirm: '刪除資料夾 {{name}} 及其中的所有內容？此操作無法復原。',
        uploading: '正在上傳…',
        uploadFailed: '上傳失敗',
        download: '下載',
        delete: '刪除',
        deleteTitle: '刪除檔案',
        deleteConfirm: '確定從媒體中刪除 {{name}}？'
      },
      yes: '確定',
      no: '取消'
    },
    head: {
      storage: '映像檔掛載 / 檔案共享',
      desktop: '遠端桌面',
      login: '登入',
      changePassword: '更改密碼',
      terminal: '終端機',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed: '無法複製到剪貼簿。請允許瀏覽器存取剪貼簿後再試。'
    },
    auth: {
      login: '登入',
      language: '語言',
      placeholderUsername: '使用者名稱',
      placeholderPassword: '密碼',
      placeholderPassword2: '請再次輸入密碼',
      noEmptyUsername: '使用者名稱不能為空',
      noEmptyPassword: '密碼不能為空',
      noAccount: '取得使用者資訊失敗，請重新整理網頁或重設密碼',
      invalidUser: '使用者名稱或密碼錯誤',
      error: '非預期性錯誤',
      twoFactorTitle: '需要驗證',
      twoFactorDesc: '請輸入驗證碼或未使用的復原代碼以繼續登入。',
      sensitiveActionDesc: '請輸入驗證碼或未使用的復原代碼以繼續。',
      placeholderAuthCode: '驗證碼',
      noEmptyAuthCode: '驗證碼不能為空',
      verify: '驗證',
      backToLogin: '返回登入',
      verificationFailed: '驗證失敗',
      twoFactorEnabledLogout: {
        message: '雙重驗證已啟用。基於安全考量，所有工作階段均已登出，請重新登入。'
      },
      rateLimited: '連續錯誤次數過多，請在 {{seconds}} 秒後重試。',
      recoveryCodeUsed: '本次登入使用的復原代碼已失效。',
      changePassword: '更改密碼',
      changePasswordDesc: '為了您的裝置安全，請修改登入密碼。',
      passwordVerificationRequired: '請返回設定頁的帳號頁面，重新驗證本次密碼修改。',
      passwordVerificationExpired: '密碼修改驗證已過期，請返回設定頁的帳號頁面重新驗證。',
      returnToSettings: '返回設定',
      differentPassword: '密碼不一致',
      illegalUsername: '使用者名稱包含非法字元',
      illegalPassword: '密碼包含非法字元',
      forgetPassword: '忘記密碼',
      ok: '確定',
      cancel: '取消',
      savePassword: '儲存密碼',
      loginButtonText: '登入',
      tips: {
        reset1: 'Web預設帳戶：',
        reset2: 'SSH預設帳戶：',
        reset3: '重設密碼請參考',
        changeReset: '如需重設網頁登入密碼，請參考 ',
        resetLink: '重設'
      }
    },
    wifi: {
      errors: {
        connect: '無法連線至 Wi-Fi。請檢查網路名稱、密碼和裝置 Wi-Fi 狀態後再試。'
      },
      title: 'Wi-Fi',
      description: '設定 NanoKVM WiFi',
      submitted: {
        title: '稍等片刻，NanoKVM 正在努力連網',
        tipsTitle: '接下來可以這樣做',
        tipSwitch: '手機現在可以切回原來的 Wi-Fi 了',
        tipCheck: '稍後到 NanoKVM 的螢幕上查看新的 IP 位址，用瀏覽器存取它',
        tipRetry: '如果等了很久都沒有動靜，請確認網路名稱和密碼無誤後再試一次'
      },
      invalidMode: '目前模式不支援設定網路。請先前往裝置啟用 Wi-Fi 配置模式。',
      confirmBtn: '確定',
      finishBtn: '完成',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: '密碼',
      steps: {
        settings: '設定',
        wifi: 'Wi-Fi',
        enable: '啟用'
      },
      scan: {
        title: '選擇 Wi-Fi 網路',
        description: '選擇附近的網路以繼續。',
        loading: '正在搜尋附近的 Wi-Fi 網路…',
        empty: '找不到附近的 Wi-Fi 網路。',
        failed: '無法掃描附近的網路，請改為手動輸入。',
        retryBtn: '重新掃描',
        manualBtn: '手動輸入網路',
        backBtn: '返回附近網路',
        connectTitle: '連線至 {{ssid}}',
        connecting: '正在連線…',
        viewSimple: '切換到簡約模式',
        viewDetailed: '切換到詳細模式'
      },
      ap: {
        authTitle: '需要身份驗證',
        authDescription: '請輸入 AP 密碼繼續',
        authFailed: 'AP 密碼無效',
        passPlaceholder: 'AP 密碼',
        verifyBtn: '驗證'
      },
      qr: {
        cameraBtn: '拍照辨識',
        pickBtn: '從相簿選擇',
        scanHint:
          '用相機拍攝 Wi-Fi QR 碼，或從相簿選擇 QR 碼截圖，例如手機「分享 Wi-Fi」時顯示的 QR 碼。',
        decoding: '正在辨識 QR 碼…',
        notFound: '照片中找不到 QR 碼。請讓 QR 碼置中、對焦清晰後再試一次。',
        invalid: '此 QR 碼不包含可用的 Wi-Fi 資訊。',
        pickerUnavailable:
          '相機或相簿沒有開啟？目前的登入視窗可能不允許，請改用手機瀏覽器開啟以下網址：',
        confirmTitle: '要將 NanoKVM 連線到 {{ssid}} 嗎？',
        confirmDescription: '以下網路資訊來自 QR 碼，請確認後再連線。',
        openNetwork: '此網路沒有密碼。',
        connectBtn: '連線',
        otherBtn: '選擇其他網路'
      }
    },
    notification: {
      browser: {
        title: '瀏覽器推薦',
        description:
          '為了獲得最佳的使用者體驗，我們建議使用Chrome瀏覽器。雖然我們努力實現所有主要瀏覽器的兼容性，但某些高級功能可能無法在 Chrome 之外按預期運行。'
      },
      btnOk: '好的'
    },
    screen: {
      title: '螢幕',
      video: '視訊模式',
      quality: '品質',
      qualityAuto: '自動',
      qualityLossless: '無損',
      qualityHigh: '高',
      qualityMedium: '中',
      qualityLow: '低',
      scale: '縮放',
      noSignal: '無視訊訊號',
      captureDisabled: '視訊擷取已關閉，請前往「設定 > 設備 > 視訊擷取」開啟。',
      inconsistentVideoMode: '播放已暫停。另一種視訊模式正在播放。',
      bandwidthWarning: '目前網路頻寬可能不足以承載視訊設定，建議降低位元率或幀率。',
      webrtcConnectionFailed: 'WebRTC 連線未建立，請切換到 Direct 視訊模式。',
      webrtcUnavailable: {
        title: 'WebRTC 無法使用',
        description: '請在瀏覽器中開啟 WebRTC，或切換至 Direct 視訊模式。'
      },
      upload: '上傳',
      advanced: '高級',
      loading: '載入中'
    },
    volume: {
      title: '音量'
    },
    microphone: {
      title: '麥克風',
      notMounted: '麥克風未安裝',
      tip1: '遠端主機上未安裝虛擬麥克風。您想立即安裝並啟用它嗎？',
      tip2: '您也可以稍後在「設定 > 裝置 > 虛擬音訊」中進行管理。',
      confirm: '啟用',
      cancel: '取消',
      noDevice: '目前裝置沒有可用的麥克風，請連接麥克風後再試',
      permissionDenied: '瀏覽器未取得麥克風權限，請在瀏覽器設定中允許存取',
      notReadable: '麥克風正由其他程式使用或目前無法讀取',
      overconstrained: '目前的麥克風不支援所需的音訊參數',
      securityBlocked: '目前頁面不允許存取麥克風，請檢查瀏覽器安全性設定',
      aborted: '麥克風啟動已中止，請再試一次',
      unsupported: '目前的瀏覽器或存取方式不支援麥克風擷取',
      enableFailed: '麥克風啟用失敗，請再試一次',
      disconnected: '控制端麥克風已中斷連線，麥克風功能已關閉'
    },
    power: {
      title: '電源',
      showConfirm: '顯示確認框',
      showConfirmTip: '電源操作需要二次確認',
      powerShort: '電源 (短按)',
      powerLong: '電源 (長按)',
      powerConfirm: '確認執行電源操作嗎？',
      okBtn: '確認',
      cancelBtn: '取消'
    },
    keyboard: {
      errors: {
        paste: '無法將文字傳送至遠端裝置。請確認 KVM 連線正常後再試。'
      },
      title: '鍵盤',
      paste: '貼上',
      tips: '僅支援標準鍵盤字母和符號',
      placeholder: '請輸入內容',
      submit: '送出',
      virtual: '虛擬鍵盤',
      swapCommandCtrl: '交換 Cmd 和 Ctrl',
      shortcut: {
        title: '快捷鍵',
        custom: '自定義',
        capture: '點選此處錄製快捷鍵',
        clear: '清空',
        save: '儲存',
        captureTips: '擷取系統級按鍵（如 Windows 鍵）需要全螢幕權限。',
        enterFullScreen: '切換全螢幕模式。'
      },
      leaderKey: {
        title: '引導鍵',
        desc: '繞過瀏覽器限制並將系統捷徑直接傳送到遠端主機。',
        howToUse: '如何使用',
        simultaneous: {
          title: '同步模式',
          desc1: '按住引導鍵不放，同時按下目標快捷鍵。',
          desc2: '直觀，但可能與系統快速鍵衝突。'
        },
        sequential: {
          title: '順序模式',
          desc1: '點擊引導鍵開始 → 依序點擊快捷鍵 → 再次點擊引導鍵結束。',
          desc2: '需要更多步驟，但完全避免了系統衝突。'
        },
        enable: '啟用引導鍵',
        tip: '設為引導鍵後，該按鍵將僅用於觸發快捷鍵，不再作為普通按鍵使用。',
        placeholder: '請按下引導鍵',
        shiftRight: '右 Shift',
        ctrlRight: '右 Ctrl',
        metaRight: '右 Win',
        submit: '送出',
        recorder: {
          rec: '記錄',
          activate: '啟用按鍵',
          input: '請按快捷鍵...'
        }
      }
    },
    mouse: {
      title: '滑鼠',
      mode: '滑鼠模式',
      absolute: '跟隨式滑鼠',
      absoluteAndroid: '跟隨式滑鼠（更相容 Android）',
      touchScreen: '多點觸控螢幕',
      relative: '獨占式滑鼠',
      direction: '滾輪方向',
      scrollUp: '向上',
      scrollDown: '向下',
      speed: '滾輪速度',
      fast: '快',
      slow: '慢',
      requestPointer: '獨占式滑鼠已啟用，請點擊畫面鎖定並隱藏本機指標。',
      repairIphoneDrag: '修復 iPhone 拖曳',
      resetHid: '重設 HID',
      touchpadGuide: {
        title: '指標輸入指南',
        scope: '適用於一般指標輸入且滑鼠模式為相對模式時。',
        swipeTitle: '滑動移動',
        swipeDesc: '在有效畫面區域內滑動，遠端滑鼠游標會跟隨移動。',
        tapTitle: '點按單擊',
        tapDesc: '短促點按一次，對應遠端滑鼠左鍵單擊。',
        holdTitle: '按住觸發左鍵按下',
        holdDesc: '手指按下並保持不動約 1 秒，會進入滑鼠左鍵按住狀態。',
        dragTitle: '按住後移動進行拖曳',
        dragDesc: '左鍵按住狀態生效後繼續移動手指，會變成按住滑鼠左鍵的拖曳。'
      },
      hidOnly: {
        errors: {
          load: '無法讀取 HID-Only 模式狀態。請稍後再試。',
          update: '無法確認 HID-Only 模式是否已更新。請重新開啟設定檢查目前狀態後再試。'
        },
        title: 'HID-Only 模式',
        desc: '如果您的滑鼠和鍵盤沒有反應，且重設 HID 無效，可能是 NanoKVM 與您的裝置間有相容性問題。請嘗試啟用 HID-Only 模式以獲得更好的相容性。',
        tip1: '啟用 HID-Only 模式將會停用虛擬隨身碟和虛擬網卡的功能',
        tip2: '在 HID-Only 模式下，映像檔掛載功能將被停用',
        enable: '啟用 HID-Only 模式',
        disable: '停用 HID-Only 模式'
      }
    },
    preview: {
      title: '預覽'
    },
    image: {
      storageReadOnly: '檔案儲存空間目前無法寫入。請關閉虛擬隨身碟。',
      storageBusy: '儲存空間使用中，請完成或取消檔案傳輸後再切換虛擬隨身碟。',
      title: '映像',
      loading: '載入中...',
      empty: '未找到任何內容',
      upload: '上傳圖片',
      dropFile: '將映像拖到此處，或按一下選擇檔案',
      uploadAction: '上傳',
      cancelAction: '取消',
      mount: '掛載映像',
      mounted: '已安裝',
      uploadError: '上傳失敗，請重試',
      uploadInsufficientStorage: '儲存空間不足',
      uploadTooLarge: '映像檔超出大小限制',
      cancelUploadError: '取消上傳失敗',
      mountMode: '掛載模式',
      massStorage: '大量儲存',
      cdRom: 'CD ROM',
      readOnly: '唯讀',
      mountFailed: '掛載失敗',
      mountDesc: '在某些系統中，在掛載映像之前需要中斷遠端主機上的虛擬磁碟。',
      unmountFailed: '卸載失敗',
      unmountDesc: '在某些系統中，需要在遠端主機中手動彈出後再卸載映像。',
      refresh: '重新整理映像檔列表',
      download: '下載圖片',
      attention: '注意',
      deleteConfirm: '確定要刪除該映像檔嗎？',
      deleteFailed: '刪除映像檔失敗',
      okBtn: '確定',
      cancelBtn: '取消'
    },
    script: {
      title: '腳本',
      upload: '上傳',
      run: '執行',
      runBackground: '背景執行',
      runFailed: '執行失敗',
      uploadFailed: '上傳腳本失敗',
      deleteFailed: '刪除腳本失敗',
      attention: '注意',
      delDesc: '確定刪除該檔案？',
      confirm: '確定',
      cancel: '取消',
      delete: '刪除',
      close: '關閉'
    },
    terminal: {
      title: '終端機',
      nanokvm: 'NanoKVM 終端機',
      confirm: '確定',
      submit: '提交',
      username: '使用者名稱',
      password: '密碼',
      authCode: '驗證碼',
      authorizeTitle: '授權 Web 終端',
      codeAccountHint: '請輸入驗證碼或未使用的復原代碼。',
      authFailed: '身分驗證失敗'
    },
    wol: {
      errors: {
        send: '無法傳送喚醒指令。請檢查 MAC 位址和網路連線後再試。',
        loadFailed: '無法載入網路喚醒裝置清單。請確認 NanoKVM 在線後再試。',
        renameFailed: '無法重新命名該裝置。請檢查 MAC 位址後再試。',
        deleteFailed: '無法刪除該裝置。請稍後再試。'
      },
      title: '網路喚醒',
      sending: '發送指令中...',
      sent: '指令已發送',
      input: '請輸入 MAC 位址',
      ok: '確定'
    },
    download: {
      sha256: 'SHA-256（可選）',
      sha256Placeholder: '請輸入 64 位元 SHA-256 校驗和',
      invalidSHA256: 'SHA-256 必須是 64 位元十六進位字串',
      title: '下載映像',
      input: '請輸入映像檔的下載URL',
      ok: '確定',
      disabled: '/data 為唯讀目錄，無法下載映像檔',
      downloadingFile: '正在下載：{{file}}',
      downloadingProgress: '正在下載（{{percentage}}）：{{file}}',
      failed: '下載失敗',
      insufficientStorage: '儲存空間不足',
      remaining: '剩餘 {{time}}',
      cancel: '取消',
      cancelFailed: '取消下載失敗',
      pause: '暫停',
      resume: '繼續',
      pauseFailed: '暫停下載失敗',
      resumeFailed: '繼續下載失敗'
    },
    settings: {
      back: '返回',
      close: '關閉',
      ai: {
        errors: {
          loadServices: '無法載入 AI 服務狀態。請確認 NanoKVM 在線後再試。',
          loadMcpConfig: '無法載入 MCP 服務設定。請稍後再試。',
          updateMcpService: 'MCP 服務狀態未能更新。請確認目前狀態後再試。',
          accessMcpApiKey: '無法取得 MCP API Key。請重試，並依提示重新完成驗證。',
          regenerateMcpApiKey: 'MCP API Key 未能重新產生。請稍後再試。',
          regeneratedMcpApiKeyUnavailable:
            'API Key 已重新產生，但暫時無法顯示。請重新嘗試顯示；原有 Key 已失效。',
          loadMemoryFabricState: '無法載入記憶織網服務狀態。請稍後再試。',
          updateMemoryFabricService: '記憶織網服務狀態未能更新。請確認目前狀態後再試。',
          loadScreenTimelapseState: '無法載入螢幕延時回放服務狀態。請稍後再試。',
          updateScreenTimelapseService: '延時回放服務狀態未能更新。請確認目前狀態後再試。',
          openScreenTimelapse: '無法開啟延時回放。請稍後再試。',
          loadScreenTimelapseRecordings: '無法載入延時回放記錄。請稍後再試。',
          playScreenTimelapse: '目前延時回放暫時無法播放。請重新選擇日期或稍後再試。',
          screenTimelapseFullscreen: '無法切換全螢幕狀態。請檢查瀏覽器權限後再試。'
        },
        title: 'AI',
        mcpService: 'MCP 服務',
        mcpServiceDesc: '為控制、媒體及已啟用的 Memory Fabric 工具提供已驗證的統一入口',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: '在本機處理 Agent 上下文記憶與 OCR',
        memoryFabricRemoteRequiresMcp: '請啟用上方的 MCP 服務，才能從遠端使用記憶與 OCR 工具。',
        memoryFabricRemoteAvailable:
          '記憶與 OCR 工具已透過上方的 /api/mcp 端點提供，並使用相同的 API Key。',
        memoryFabricConfig: {
          title: '設定',
          description: '僅管理 Memory Fabric 的預設模型設定。',
          loadFailed: '無法載入 Memory Fabric 設定。',
          saveFailed: '無法儲存 Memory Fabric 設定。',
          saved: '設定已儲存。',
          savedRestarted: '設定已儲存，Memory Fabric 已重新啟動。',
          noChanges: '設定沒有變更。',
          reload: '重新載入',
          save: '儲存',
          unsaved: '有尚未儲存的變更',
          conflictTitle: '設定已變更',
          conflictDescription: '設定已在其他位置被修改。請先載入最新版本，再重新儲存。',
          conflictReload: '載入最新設定',
          discardTitle: '放棄尚未儲存的變更？',
          discardDescription: '重新載入會以最新儲存的設定取代目前表單內容。',
          discardConfirm: '放棄並重新載入',
          sections: {
            models: '模型'
          },
          openAICompatibleHint:
            '支援 OpenAI Compatible 介面（相容 OpenAI API 格式）。請填寫服務商提供的模型名稱、Base URL 和 API Key。',
          fields: {
            model: '模型',
            baseUrl: '基礎 URL',
            maxTokens: '最大 Token 數',
            timeoutSeconds: '逾時時間（秒）'
          },
          apiKey: {
            title: 'API 金鑰',
            configured: '已設定',
            notConfigured: '尚未設定',
            keep: '保留現有金鑰',
            replace: '更換金鑰',
            clear: '清除金鑰',
            replacement: '新 API 金鑰',
            replacementPlaceholder: '輸入新的 API 金鑰',
            clearHint: '已儲存的 API 金鑰將被移除；環境變數仍可能提供金鑰。',
            required: '請輸入新的 API 金鑰。'
          },
          validation: {
            required: '此欄位為必填。',
            invalidUrl: '請輸入有效的 HTTP 或 HTTPS URL。'
          }
        },
        screenTimelapseService: '螢幕延時回放',
        screenTimelapseServiceDesc: '回放 Memory Fabric 螢幕歷史，並搜尋已擷取的螢幕內容',
        screenTimelapseSearchRequiresMemoryFabric: '如需搜尋錄製內容，請同時啟用 Memory Fabric。',
        screenTimelapseOpen: '開啟',
        screenTimelapseSearchPlaceholder: '搜尋影片內容...',
        screenTimelapseNoVideos: '未找到延時回放影片',
        screenTimelapseNoResults: '未找到符合畫面',
        screenTimelapseMultipleResults: '搜尋結果',
        screenTimelapseUnavailable: '不可用',
        screenTimelapsePlayerUnsupported: '目前瀏覽器無法播放延時回放影片',
        screenTimelapseSearchFailed: '搜尋失敗',
        screenTimelapseLoading: '正在載入延時回放...',
        screenTimelapseSearchAction: '搜尋延時回放',
        screenTimelapsePlay: '播放延時回放',
        screenTimelapsePause: '暫停延時回放',
        screenTimelapseEnterFullscreen: '進入全螢幕',
        screenTimelapseExitFullscreen: '退出全螢幕',
        screenTimelapseFullscreenUnsupported: '目前瀏覽器無法進入全螢幕',
        screenTimelapsePreviousDay: '前一天',
        screenTimelapseNextDay: '後一天',
        screenTimelapseDeleteDay: '刪除 {{date}} 的回放',
        screenTimelapseDelete: '刪除',
        screenTimelapseDeleteConfirmTitle: '刪除延時回放？',
        screenTimelapseDeleteConfirmDesc:
          '{{date}} 的回放佔用 {{size}}。刪除後無法復原，是否繼續？',
        screenTimelapseDeleteSuccess: '已刪除 {{date}} 的回放',
        screenTimelapseDeleteFailed: '刪除 {{date}} 的回放失敗',
        endpoint: '端點',
        apiKey: 'API 金鑰',
        regenerate: '重新產生',
        copyApiKey: '複製 API 金鑰',
        copyEndpoint: '複製端點',
        showApiKey: '顯示 API 金鑰',
        regenerateConfirmTitle: '重新產生 API 金鑰',
        regenerateConfirmDesc:
          '重新產生後，現有的 MCP 用戶端必須更新為新的金鑰才能繼續使用。確定要繼續嗎？',
        okBtn: '確定',
        cancelBtn: '取消'
      },
      title: '設定',
      about: {
        title: '關於 NanoKVM',
        information: '資訊',
        ip: 'IP',
        mdns: 'mDNS',
        application: '應用程式版本',
        applicationTip: 'NanoKVM 網頁程式版本',
        arch: '架構',
        device_number: '設備編號',
        image: '映像版本',
        imageTip: 'NanoKVM 系統映像版本',
        deviceKey: '設備序號',
        community: '社群',
        hostname: '主機名稱',
        hostnameUpdated: '已更新主機名稱. 請重新啟動以生效',
        ipType: {
          Wired: '有線',
          Wireless: '無線',
          Other: '其他'
        }
      },
      appearance: {
        title: '外觀',
        display: '顯示',
        language: '語言',
        languageDesc: '選擇介面語言',
        languageChangeFailed: '語言切換失敗',
        webTitle: '網頁標題',
        webTitleDesc: '自訂網頁標題',
        menuBar: {
          title: '選單列',
          mode: '顯示方式',
          modeDesc: '選單欄在螢幕上的顯示方式',
          modeOff: '關閉',
          modeAuto: '自動隱藏',
          modeAlways: '始終顯示',
          icons: '選單圖示',
          showIcon: '顯示圖示',
          activateOnConnect: '連線時啟用',
          stopSharing: '停止分享',
          iconsDesc: '是否在選單欄中顯示子選單圖示'
        }
      },
      screen: {
        title: '螢幕',
        videoMode: {
          title: '視訊模式',
          description: '選擇編碼與傳輸方式'
        },
        edid: {
          description: '選擇顯示參數',
          default: '預設',
          custom: '自訂',
          document: '文檔',
          powerConfirmTitle: '更新 EDID 前請確認',
          powerConfirmDescription:
            '請先接入 PD 電源。更新 EDID 可能導致顯示短暫斷開；如果更新後設備無法正確識別顯示器，請手動重啟設備。',
          powerConfirmOk: '繼續更新',
          powerConfirmOkCountdown: '繼續更新（{{seconds}}）',
          powerConfirmCancel: '取消',
          errors: {
            badRequest: 'EDID 請求無效',
            invalidFileName: '請選擇名稱有效的 .bin EDID 檔案',
            fileTooLarge: 'EDID 檔案過大',
            invalidLength: 'EDID 必須為 128、256、384 或 512 位元組',
            invalidHeader: 'EDID 檔頭無效',
            unsupportedVersion: '不支援此 EDID 版本',
            extensionCountMismatch: 'EDID 擴充區塊數量與檔案長度不符',
            invalidChecksum: 'EDID 總和檢查碼無效',
            invalidCta: 'CTA 擴充區塊結構無效',
            currentCannotOverwrite: '無法覆寫目前使用中的 EDID，請先切換至其他 EDID',
            saveFailed: '儲存 EDID 失敗',
            switchFailed: '切換 EDID 失敗',
            uploadFailed: '上傳 EDID 失敗',
            deleteFailed: '刪除 EDID 失敗',
            networkFailed: '網路錯誤，請再試一次'
          }
        },
        rateControlMode: {
          title: '速率控制模式',
          description: '選擇控制碼率的方法',
          cbr: 'CBR（恆定位元速率）：優先考慮連接穩定性。最大限度地減少卡頓，但在高速運動場景中影像品質可能會下降。',
          vbr: 'VBR（可變位元速率）：優先考慮影像清晰度。高速運動場景可能會導致頻寬峰值，從而可能導致延遲或卡頓。'
        },
        bitrate: {
          title: '比特率',
          description: '平衡影像清晰度與頻寬使用',
          lower: '較低的值可以減少較差網路上的延遲，但會影響影像質量，可能導致模糊或偽影。',
          higher: '較高的值可提高影像清晰度，但會消耗更多頻寬。如果連線速度慢，可能會增加延遲。'
        },
        gop: {
          description: '設定完整關鍵影格之間的間隔',
          lower:
            '較低的值更頻繁地發送全幀。這可以加快視覺偽影的恢復速度，但會消耗更多頻寬並減少影像細節。',
          higher:
            '較高的值可最佳化增量更新。這可以改善影像細節並節省頻寬，但從視覺偽影中恢復將需要更長的時間。'
        },
        quality: {
          title: '質量',
          description: '設定顯示影像質量'
        },
        fps: {
          auto: '自動',
          description: '設定顯示幀率',
          tip1: '將值設為 0 可啟用自動模式，使顯示與遠端主機的幀速率同步。',
          tip2: '降低幀率可大幅減少頻寬使用。在網路不穩定或不需要高幀速率時手動降低它。'
        },
        scale: {
          title: '縮放',
          description: '設定顯示縮放比例'
        }
      },
      device: {
        title: '設備',
        virtualAudioConfirm: {
          title: '切換虛擬音訊前請確認',
          description:
            '請先連接 PD 電源。切換虛擬音訊可能導致顯示短暫中斷；如果切換後裝置無法正確辨識顯示器，請手動重新啟動裝置。',
          ok: '繼續切換',
          okCountdown: '繼續切換（{{seconds}}）',
          cancel: '取消'
        },
        wifi: {
          title: 'Wi-Fi',
          description: '設定 Wi-Fi',
          apMode: 'AP模式已啟用，手動連線Wi-Fi',
          others: '其他...',
          connect: '連線 Wi-Fi',
          connectDesc1: '請輸入網路名稱和密碼',
          connectDesc2: '請輸入密碼以連線此網路',
          disconnect: '是否要中斷該網路連線？',
          ssid: 'SSID 名稱',
          password: '密碼',
          joinBtn: '加入',
          confirmBtn: '確定',
          cancelBtn: '取消',
          preset: {
            title: 'Wi-Fi 預存',
            description: '提前預存一個目標 Wi-Fi，裝置開機後自動連線',
            empty: '無',
            addBtn: '新增預存',
            removeBtn: '刪除預存',
            saveTitle: '預存 Wi-Fi',
            saveDesc: '現在儲存目標網路，裝置開機偵測到該網路後自動連線',
            saveBtn: '儲存',
            passwordPlaceholder: '密碼（開放網路可留空）',
            saveFailed: '儲存預存 Wi-Fi 失敗',
            removeFailed: '刪除預存 Wi-Fi 失敗',
            removeConfirm: '確定要刪除已預存的 Wi-Fi 嗎？',
            warning:
              '開啟後，裝置每次啟動時若偵測到該 Wi-Fi 將盡力連線，可能干擾連線其他 Wi-Fi；預存將持續生效，直到您修改或刪除。'
          }
        },
        capture: {
          title: '視訊擷取',
          description: '啟用視訊擷取'
        },
        ssh: {
          description: '啟用SSH',
          tip: '啟用後請在終端機中為 root 使用者設定強密碼。'
        },
        swap: {
          errors: {
            load: '無法讀取交換記憶體狀態。請稍後再試。',
            update: '交換記憶體設定未能更新。請確認目前狀態後再試。'
          },
          title: '交換記憶體',
          description: '啟用壓縮交換記憶體'
        },
        advanced: '進階設定',
        staticIp: {
          title: '靜態IP',
          config: '配置',
          tip: '為防止連線遺失，建議您在實體存取裝置時進行設定。',
          document: '檢查文件。',
          enable: '啟用靜態IP',
          placeholder: '格式：IP/CIDR [網關]\n範例：192.168.1.10/22',
          notEmpty: 'IP位址不能為空。',
          notValid: 'IP 位址格式無效。',
          failed: '配置失敗。',
          success: '配置完成。請檢查IP位址是否可用。',
          okBtn: '好的',
          cancelBtn: '取消'
        },
        usbNetwork: {
          title: 'USB 網卡',
          mode: '模式',
          server: '伺服器端',
          serverDesc: '透過內建 DHCP 服務為遠端主機分配 IP 位址。',
          client: '用戶端',
          clientDesc: '透過 DHCP 從遠端主機取得 IP 位址。',
          forwarding: 'IP 轉送',
          forwardingTip: '若要讓遠端主機透過 NanoKVM 存取網際網路，請開啟 IP 轉送。',
          manualIp: '手動設定 IP',
          ipv4Placeholder: 'IPv4/CIDR，例如：10.10.10.1/24',
          invalidIpv4: '請輸入有效的 IPv4/CIDR 位址。',
          loadFailed: '無法讀取 USB 網卡設定。',
          failed: '設定失敗。',
          success: '設定完成。',
          applyBtn: '套用',
          cancelBtn: '取消'
        },
        usbId: {
          title: 'USB 裝置標識',
          config: '設定',
          tip: '自訂 USB 裝置標識會改變主機辨識本裝置的方式。套用後將重啟 USB 裝置，鍵盤、滑鼠或網路可能短暫中斷。可選欄位留空則使用腳本預設值。',
          enable: '啟用自訂 USB 裝置標識',
          vid: '廠商 ID (VID)',
          pid: '產品 ID (PID)',
          bcdDevice: '裝置版本 (bcdDevice)',
          serialnumber: '序號',
          manufacturer: '廠商名',
          product: '產品名',
          vidPlaceholder: '例如：0x359F',
          pidPlaceholder: '例如：0x3305',
          bcdDevicePlaceholder: '例如：0x0101（可選）',
          serialnumberPlaceholder: '例如：0123456789ABCDEF（可選）',
          manufacturerPlaceholder: '例如：sipeed（可選）',
          productPlaceholder: '例如：NanoKVM-Go（可選）',
          notEmpty: 'VID 與 PID 不能為空',
          notValid: 'VID/PID 格式無效，請使用十六進位，如 0x359F',
          bcdNotValid: 'bcdDevice 格式無效，請使用十六進位如 0x0101，或留空',
          failed: '設定失敗',
          success: '設定完成，USB 裝置已重啟',
          okBtn: '確定',
          cancelBtn: '取消'
        },
        diskInquiry: {
          title: '隨身碟顯示名稱',
          config: '設定',
          tip: '遠端電腦上看到的虛擬隨身碟名稱。儲存後會重啟 USB，鍵盤、滑鼠或網路可能短暫中斷。',
          enable: '自訂隨身碟名稱',
          placeholder: '例如：sipeed  Flash Drive     1.00',
          notEmpty: '名稱不能為空',
          notValid: '名稱無效，最多 28 個可列印 ASCII 字元且不能換行',
          failed: '設定失敗',
          success: '設定完成，USB 裝置已重啟',
          okBtn: '確定',
          cancelBtn: '取消'
        },
        diskInquiryCdrom: {
          title: '虛擬光碟機名稱',
          config: '設定',
          tip: '遠端電腦上看到的虛擬光碟機 / CD 名稱。儲存後會重啟 USB，鍵盤、滑鼠或網路可能短暫中斷。',
          enable: '自訂虛擬光碟機名稱',
          placeholder: '例如：sipeed  Optical Drive   1.00',
          notEmpty: '名稱不能為空',
          notValid: '名稱無效，最多 28 個可列印 ASCII 字元且不能換行',
          failed: '設定失敗',
          success: '設定完成，USB 裝置已重啟',
          okBtn: '確定',
          cancelBtn: '取消'
        },
        uac2FunctionName: {
          title: 'USB 音訊名稱',
          config: '設定',
          tip: '遠端電腦上看到的虛擬 USB 麥克風 / 音訊裝置名稱。儲存後會重啟 USB，鍵盤、滑鼠或網路可能短暫中斷。',
          enable: '自訂 USB 音訊名稱',
          placeholder: '例如：NanoKVM-Go Audio',
          notEmpty: '名稱不能為空',
          notValid: '名稱無效，最多 64 個可列印字元且不能換行',
          failed: '設定失敗',
          success: '設定完成，USB 裝置已重啟',
          okBtn: '確定',
          cancelBtn: '取消'
        },
        lowPower: {
          title: '低功耗模式',
          description: '減少空閒時的功耗',
          tips: '小貼士',
          tip1: '啟用後，裝置會在閒置時暫停視訊擷取，以降低功耗與發熱。',
          tip2: '恢復視訊擷取可能需要重新初始化，期間可能暫時沒有畫面。',
          okBtn: '好的',
          cancelBtn: '取消'
        },
        mouseJiggler: {
          title: '滑鼠抖動模式 (Mouse Jiggler)',
          description: '避免遠端主機進入休眠狀態',
          disable: '停用',
          absolute: '絕對模式',
          relative: '相對模式'
        },
        fingerRobot: {
          title: '手指機器人',
          description: '啟用手指機器人控制'
        },
        mdns: {
          description: '啟用 mDNS 發現服務'
        },
        hidOnly: 'HID-Only 模式',
        imageMounted: '已安裝映像',
        disk: '虛擬隨身碟',
        diskDesc: '在遠端主機上連接虛擬隨身碟',
        sdCard: {
          warning: '初始化虛擬隨身碟',
          tip1: '首次啟用將初始化虛擬隨身碟，並清除裝置上現有的映像檔及虛擬隨身碟資料。此操作無法復原，請先備份重要檔案。',
          tip2: '是否繼續初始化？',
          capacity: '容量',
          disable: '停用',
          disableWarning: '停用虛擬隨身碟？',
          disableTip:
            '停用後將清除虛擬隨身碟中的所有檔案，並釋放其占用的裝置儲存空間。下次啟用時需要重新選擇容量並初始化。此操作無法復原。'
        },
        network: '虛擬網卡',
        networkDesc: '在遠端主機上新增虛擬網卡',
        mic: '虛擬音訊',
        micDesc: '選擇提供給遠端主機的 USB 虛擬音訊模式',
        virtualAudioMode: {
          off: '關閉',
          mic: '僅麥克風',
          duplex: '麥克風 + 揚聲器'
        },
        speaker: '虛擬揚聲器',
        speakerDesc:
          '開啟後替換預設 DP 音訊輸出，用於支援遠端通話，相容性更強。關閉後只能轉發媒體音訊，無法轉發通話音訊。',
        micTip:
          '由於 USB 端點限制，啟用虛擬麥克風將停用 USB 輔助螢幕。\n建議在 BIOS 操作期間停用此功能，以確保最佳相容性。',
        reboot: '重新啟動',
        rebootDesc: '您確定要重新啟動 NanoKVM?',
        rebootToFlashing: '重新啟動至燒錄模式',
        rebootToFlashingButton: '重啟',
        rebootToFlashingTitle: '重新啟動至燒錄模式？',
        rebootToFlashingDesc:
          'NanoKVM 將重新啟動並進入 USB 燒錄模式，請確認已連接用於燒錄的 USB 介面，並準備好燒錄工具。',
        datetime: {
          timezone: '時區',
          datetime: '日期時間',
          dateAndTime: '日期與時間',
          synchronize: '同步時間',
          lastSynchronization: '上次同步：',
          notSynchronized: '系統時間尚未同步',
          syncNow: '立即同步',
          manualSet: '手動設定',
          manualSetTitle: '手動設定時間',
          manualSetDescription: '時間變更後需要重新登入。',
          useBrowserTime: '使用瀏覽器目前時間',
          setTime: '設定時間',
          invalidTime: '請輸入有效的日期與時間。',
          setFailed: '無法設定系統時間，請再試一次。',
          setSucceeded: '系統時間已更新，請重新登入。'
        },
        dialBackground: {
          errors: {
            load: '無法載入錶盤背景圖片。請稍後再試。',
            upload: '背景圖片未能上傳。請確認檔案是有效的 BMP 圖片後再試。',
            switch: '錶盤背景未能切換。請重新整理圖片清單後再試。'
          },
          title: '錶盤背景',
          description: '從 /kvmcomm/ui/bg/ 選擇 BMP 圖片作為錶盤背景',
          upload: '上傳',
          loading: '載入中...',
          empty: '未找到 BMP 圖片',
          current: '目前',
          onlyBmp: '僅支援 BMP 圖片',
          uploaded: '上傳完成',
          switched: '背景已切換'
        },
        okBtn: '確定',
        cancelBtn: '取消'
      },
      tailscale: {
        title: 'Tailscale',
        restart: '確定要重啟 Tailscale 嗎？',
        stop: '確定要停止 Tailscale 嗎？',
        stopDesc: '此操作將會登出帳號，並停止開機自動啟動。',
        loading: '載入中...',
        notInstall: '未找到 Tailscale ！請先安裝。',
        install: '安裝',
        installing: '安裝中',
        failed: '安裝失敗',
        retry: '請重新整理並重試。或嘗試手動安裝',
        download: '下載',
        package: '安裝包',
        unzip: '並解壓縮它',
        upTailscale: '將 Tailscale 上傳到 NanoKVM 目錄 /usr/bin/',
        upTailscaled: '將 tailscaled 上傳到 NanoKVM 目錄 /usr/sbin/',
        refresh: '重新整理頁面',
        notRunning: 'Tailscale 尚未執行',
        run: '啟動',
        notLogin: '設備尚未綁定。請登入並將該裝置綁定到您的帳戶。',
        urlPeriod: '此網址有效期限為 10 分鐘',
        login: '登入',
        loginSuccess: '登入成功',
        enable: '啟用 Tailscale',
        exitNode: '出口節點',
        exitNodeDesc: '將此裝置發佈為 tailnet 的網際網路出口。',
        subnetRoute: '子網路由',
        subnetRouteDesc: '向 tailnet 發佈本機直接連線的區域網路網段 {{subnet}}。',
        subnetUnavailable: '未偵測到可發佈的直接連線區域網路網段。',
        routingUnavailable: '此裝置目前無法設定 Tailscale 路由。',
        pendingApproval: '啟用後，請在 Tailscale 管理後台核准發佈的路由。',
        routingUpdateFailed: '更新 Tailscale 路由設定失敗。',
        deviceName: '裝置名稱',
        deviceIP: '裝置 IP',
        account: '帳號',
        logout: '登出',
        logoutDesc: '確認要登出嗎？',
        uninstall: '移除 Tailscale',
        uninstallDesc: '確定要解除安裝 Tailscale 嗎？',
        okBtn: '確認',
        cancelBtn: '取消'
      },
      update: {
        title: '檢查更新',
        queryFailed: '取得版本失敗',
        updateFailed: '更新失敗。請重試。',
        isLatest: '您已經擁有最新版本。',
        available: '有可用更新。確定要更新嗎？',
        updating: '正在更新。請稍等...',
        confirm: '確定',
        upgrade_tip_nano: 'NanoKVM 新版本現已推出。您確定要立即更新嗎？',
        upgrade_tip_firmware: '有新的韌體版本可用。您確定要立即更新嗎？',
        cancel: '取消',
        preview: '預覽更新',
        previewDesc: '預覽版本，搶先體驗新功能和改進',
        previewTip: '請注意，預覽版本可能包含一些不穩定因素或未完善的功能！',
        download: '下載',
        install: '安裝',
        changelog: '變更日誌',
        restart: '重啟'
      },
      account: {
        title: '帳號',
        webAccount: '網頁帳號',
        password: '密碼',
        twoFactor: '雙重驗證',
        enableTwoFactor: '啟用',
        enableTwoFactorDesc: '請輸入目前密碼，以開始設定雙重驗證。',
        disableTwoFactor: '停用',
        recoveryCodes: '復原代碼',
        enableRecoveryCodes: '啟用',
        updateRecoveryCodes: '更新',
        recoveryCodesTooltipDisabled: '復原代碼是備用雙重驗證憑證，只有啟用雙重驗證後才能產生。',
        recoveryCodesTooltipEnabled:
          '復原代碼只在雙重驗證啟用時有效，停用雙重驗證會讓所有復原代碼失效；在雙重驗證保持啟用時只能更新。',
        recoveryCodesRegenerateDesc:
          '請輸入目前密碼，以及驗證碼或未使用的復原代碼。更新復原代碼會讓所有現有復原代碼失效。',
        enableRecoveryCodesDesc:
          '請輸入目前密碼和驗證碼以產生復原代碼。復原代碼只能檢視一次，請在離開本頁前儲存。',
        recoveryCodesVerifyFailed: '無法驗證密碼，以及驗證碼或復原代碼。',
        recoveryCodesSavedTitle: '請立即儲存這些復原代碼',
        recoveryCodesSavedDesc: '復原代碼只能檢視一次。請在離開本頁前將其儲存在安全位置。',
        copyRecoveryCodes: '複製',
        downloadRecoveryCodes: '下載',
        recoveryCodesCopyFailed: '無法複製復原代碼。',
        done: '完成',
        currentPassword: '目前密碼',
        authCode: '驗證碼',
        authCodeOrRecoveryCode: '驗證碼或復原代碼',
        authRateLimited: '連續錯誤次數過多，請在 {{seconds}} 秒後重試。',
        verifyPasswordChange: '驗證密碼修改',
        verifyPasswordChangeDesc: '修改密碼前，請輸入目前密碼，以及驗證碼或未使用的復原代碼。',
        verifyPasswordChangeFailed: '無法驗證密碼，以及驗證碼或復原代碼。',
        verifyAndEnable: '驗證並啟用',
        scanTwoFactorQRCode: '使用驗證器應用程式掃描此 QR Code，然後輸入驗證碼。',
        showSetupKey: '顯示設定密鑰',
        hideSetupKey: '隱藏設定密鑰',
        setupKey: '設定密鑰',
        twoFactorSetupFailed: '無法開始雙重驗證設定。',
        twoFactorVerifyFailed: '無法驗證驗證碼。',
        twoFactorDisableFailed: '無法停用雙重驗證。',
        disableTwoFactorDesc:
          '請輸入目前密碼，以及驗證碼或未使用的復原代碼。停用雙重驗證會讓所有復原代碼失效。',
        updateBtn: '修改',
        logoutBtn: '登出',
        logoutDesc: '您確定要登出嗎?',
        okBtn: '確定',
        cancelBtn: '取消',
        twoFactorProtection: {
          title: '進階雙重驗證保護',
          description:
            '選擇在啟用保護強制執行後需要進行雙重驗證的敏感操作。「直接網頁登入」開關相反：開啟後僅使用驗證碼或未使用的復原代碼即可登入。',
          login: '直接網頁登入',
          webTerminal: 'Web 終端',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API 金鑰',
          targetPower: '目標機電源',
          nanoKVMRestart: 'NanoKVM 重新啟動',
          deleteImage: '刪除映像',
          scripts: '指令碼操作',
          screenTimelapse: '螢幕延時錄製',
          loginScope:
            '允許網頁僅使用驗證碼或未使用的復原代碼登入，不必輸入使用者名稱和密碼。這會降低登入安全性。',
          webTerminalScope: '開啟 Web 終端。',
          sshScope: '變更 SSH 服務狀態。',
          tailscaleScope: '安裝、連線或變更 Tailscale。',
          frp: 'FRP',
          frpScope: '安裝、連線或變更 FRP。',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: '安裝、連線或變更 Cloudflare Tunnel。',
          zerotierScope: '安裝、連線或變更 ZeroTier。',
          netbirdScope: '安裝、連線或變更 NetBird。',
          mcpScope: '顯示、複製或重新產生 MCP API 金鑰，以及啟用或停用 MCP。',
          targetPowerScope: '透過實體電源按鈕關閉或重新啟動目標機。',
          nanoKVMRestartScope: '重新啟動 NanoKVM 本身。',
          deleteImageScope: '永久刪除已儲存的 ISO 或 IMG 映像。',
          scriptsScope: '執行、上傳或刪除指令碼。',
          screenTimelapseScope: '啟用螢幕延時錄製、瀏覽錄製內容及刪除錄製內容。',
          authorizeTitle: '授權變更保護設定',
          authorizeDesc: '請輸入驗證碼或未使用的復原代碼。授權將在五分鐘內有效。',
          authorizeFailed: '無法驗證驗證碼或復原代碼。',
          loadFailed: '無法載入保護設定。',
          updateFailed: '無法儲存保護設定。',
          retry: '重試',
          disableScriptsWarningTitle: '關閉指令碼保護？',
          disableScriptsWarningDesc:
            '若不啟用保護，惡意指令碼可能取得此裝置的高權限存取，並繞過或移除 2FA 保護。',
          cancelDisabling: '取消',
          continueDisabling: '繼續關閉',
          continueDisablingCountdown: '繼續關閉（{{seconds}} 秒）'
        },
        twoFactorGuide: {
          title: '雙重驗證指南',
          scope: '雙重驗證會為您的帳號增加額外的安全保護。',
          whatTitle: '什麼是 2FA？',
          whatDesc:
            '雙重驗證（2FA）使用驗證器應用程式產生的基於時間的一次性代碼保護您的帳號。預設情況下需要在密碼之外輸入該代碼，即使密碼外洩，帳號仍能受到保護。如果啟用了直接網頁登入，僅憑代碼即可登入。',
          appsTitle: '建議應用程式',
          appsDesc:
            '您可以使用 Google Authenticator 或 Microsoft Authenticator。兩者皆免費，並支援 iOS 和 Android。',
          setupTitle: '如何設定',
          setupDesc:
            '打開驗證器應用程式，掃描設定時顯示的 QR Code，或手動輸入設定密鑰。然後輸入應用程式中的 6 位數代碼以完成啟用 2FA。',
          recoveryTitle: '復原代碼',
          recoveryDesc:
            '復原代碼是一次性的備用代碼。如果您無法使用驗證器應用程式，可以使用復原代碼登入。產生後請立即儲存在安全位置。',
          logoutNotice:
            '啟用 2FA 會登出所有作用中的工作階段，包括目前工作階段。您需要在每台裝置上重新登入。'
        }
      }
    },
    error: {
      title: '我們遇到了一些問題',
      refresh: '重新整理'
    },
    pictureInPicture: {
      title: '子母畫面',
      enter: '開啟子母畫面',
      exit: '關閉子母畫面',
      error: '無法開啟子母畫面，請等待畫面載入後再試一次。'
    },
    fullscreen: {
      toggle: '進入全螢幕模式'
    },
    menu: {
      collapse: '收起選單',
      expand: '展開選單'
    }
  }
};

export default zh_tw;
