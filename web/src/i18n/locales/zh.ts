const zh = {
  translation: {
    storage: {
      cards: { mount: '磁盘挂载', share: '文件共享' },
      title: '镜像挂载/文件共享',
      empty: '暂无镜像',
      notAttached: '未装载任何介质',
      attached: '已装载:{{name}}',
      eject: '弹出',
      mount: '装载',
      mountMode: '装载模式',
      mountModeDisk: '磁盘模式',
      mountModeCdrom: '光盘模式',
      openManager: '打开文件管理器',
      refresh: '刷新',
      delete: '删除',
      deleteTitle: '删除介质',
      deleteConfirm: '确定删除 {{name}}?里面的文件会一起丢失。',
      downloadImage: '下载镜像',
      typePartition: '共享文件',
      volumeLabel: '可移动磁盘',
      groups: {
        share: '文件共享',
        images: '镜像'
      },
      failed: '操作失败',
      hidOnly: 'HID-Only 模式:虚拟磁盘不可用',
      disabled: '虚拟磁盘未启用(设置 - 设备)',
      force: {
        title: '强制收回介质?',
        desc: '被控机还挂着这个介质。强制收回会让 USB 设备重新枚举,它还没写盘的数据可能丢失。',
        menuHint: '被控机正在使用该介质,请到文件管理器页面强制收回。'
      },
      tips: {
        title: '如何上传',
        usb1: '将 NanoKVM 通过 USB 连接到你的电脑；',
        usb2: '在本页面以大容量存储模式挂载共享文件；',
        usb3: '在电脑上打开虚拟硬盘，将镜像文件拷贝到虚拟硬盘的根目录下。',
        usb4: '拷贝完成后弹出磁盘，镜像随后会出现在列表中。',
        scp1: '确保 NanoKVM 和你的电脑在同一个局域网内；',
        scp2: '在电脑上打开终端软件，使用 SCP 命令将镜像文件上传到 NanoKVM 的 /data 目录。',
        scp3: '示例：scp your-image-path root@your-nanokvm-ip:/data'
      },
      job: {
        running: '正在处理 {{name}}'
      },
      enableDisk: {
        title: '启用虚拟磁盘?',
        desc: '装载介质需要虚拟磁盘,当前它是关闭的。现在启用?被控机的键鼠会短暂断开。',
        menuHint: '虚拟磁盘未启用,请到 设置 - 设备 打开。'
      },
      add: {
        title: '添加介质',
        upload: '上传 ISO / IMG',
        fromUrl: '从 URL 下载',
        url: '镜像地址',
        urlRequired: '请输入镜像地址',
        onlyImages: '只能挂载 .iso 和 .img 文件',
        dropHint: '把 ISO / IMG 拖到这里，或点击选择文件',
        cdromOnlyIso: '光盘模式只能挂载 ISO / UDF 光盘镜像',
        uploadFailed: '上传失败',
        downloadFailed: '下载失败',
        uploading: '正在上传 {{name}}',
        downloading: '正在下载 {{name}}',
        done: '{{name}} 已就绪',
        checksumFailed: '镜像与 SHA-256 校验值不符。',
        blocked:
          '被控机正在使用这张盘。请先在被控机上安全弹出虚拟 U 盘，再到设置 > 设备关闭虚拟 U 盘开关。'
      },
      files: {
        reasonNotInitialized: '虚拟 U 盘尚未初始化，请先到设置 > 设备初始化虚拟 U 盘。',
        selectMedium: '选择一个介质,查看里面的文件。',
        reasonAttached: '该介质正装载给被控机使用。',
        reasonAttachedHint: '请先在被控机上安全弹出虚拟 U 盘，再到设置 > 设备关闭虚拟 U 盘开关。',
        reasonNotBrowsable: '这个介质里的文件系统无法在网页端打开。',
        reasonNoLoop: '本机不支持挂载磁盘镜像(内核未启用 loop)。',
        reasonISOImage: 'ISO 光盘镜像不支持在网页端浏览目录，镜像本身仍可正常挂载给被控机',
        empty: '这个目录是空的',
        noMatch: '没有匹配的文件',
        free: '剩余 {{size}}',
        search: '搜索当前目录',
        sort: {
          name: '名称',
          size: '大小',
          type: '类型',
          modified: '修改时间',
          asc: '升序',
          desc: '降序'
        },
        view: {
          list: '列表视图',
          grid: '图标视图'
        },
        folder: '文件夹',
        itemCount: '{{count}} 项',
        itemCount_one: '{{count}} 项',
        itemCount_other: '{{count}} 项',
        dropHint: '拖拽文件到此处即可上传到当前目录',
        up: '返回上一级',
        upload: '上传文件到介质',
        uploadButton: '上传文件',
        newFolder: '新建文件夹',
        newFolderPlaceholder: '文件夹名称',
        create: '创建',
        deleteFolderTitle: '删除文件夹',
        deleteFolderConfirm: '删除文件夹 {{name}} 及其中的全部内容？此操作无法撤销。',
        uploading: '正在上传…',
        uploadFailed: '上传失败',
        download: '下载',
        delete: '删除',
        deleteTitle: '删除文件',
        deleteConfirm: '确定从介质中删除 {{name}}?'
      },
      yes: '确定',
      no: '取消'
    },
    head: {
      storage: '镜像挂载/文件共享',
      desktop: '远程桌面',
      login: '登录',
      changePassword: '修改密码',
      terminal: '终端',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed: '无法复制到剪贴板。请允许浏览器访问剪贴板后重试。'
    },
    auth: {
      login: '登录',
      language: '语言',
      placeholderUsername: '请输入用户名',
      placeholderPassword: '请输入密码',
      placeholderPassword2: '请再次输入密码',
      noEmptyUsername: '用户名不能为空',
      noEmptyPassword: '密码不能为空',
      noAccount: '获取用户信息失败，请刷新重试或重置密码',
      invalidUser: '用户名或密码错误',
      error: '未知错误',
      twoFactorTitle: '需要验证',
      twoFactorDesc: '请输入认证码或未使用的恢复代码以继续登录。',
      sensitiveActionDesc: '请输入认证码或未使用的恢复代码以继续。',
      placeholderAuthCode: '认证码',
      noEmptyAuthCode: '认证码不能为空',
      verify: '验证',
      backToLogin: '返回登录',
      verificationFailed: '验证失败',
      twoFactorEnabledLogout: {
        message: '双重验证已启用。出于安全考虑，所有会话均已退出，请重新登录。'
      },
      rateLimited: '连续错误次数过多，请在 {{seconds}} 秒后重试。',
      recoveryCodeUsed: '本次登录使用的恢复代码已失效。',
      changePassword: '修改密码',
      changePasswordDesc: '为了您的设备安全，请修改密码!',
      passwordVerificationRequired: '请返回设置页的帐号页面，重新验证本次密码修改。',
      passwordVerificationExpired: '密码修改验证已过期，请返回设置页的帐号页面重新验证。',
      returnToSettings: '返回设置',
      differentPassword: '两次密码不一致',
      illegalUsername: '用户名中包含非法字符',
      illegalPassword: '密码中包含非法字符',
      forgetPassword: '忘记密码',
      ok: '确定',
      cancel: '取消',
      savePassword: '保存密码',
      loginButtonText: '登录',
      tips: {
        reset1: '网页默认帐号：',
        reset2: 'SSH 默认帐号：',
        reset3: '如需重置密码，请参考 ',
        changeReset: '如需重置网页登录密码，请参考 ',
        resetLink: '重置'
      }
    },
    wifi: {
      errors: {
        connect: '未能连接到 Wi-Fi。请检查网络名称、密码和设备 Wi-Fi 状态后重试。'
      },
      title: 'Wi-Fi',
      description: '配置 NanoKVM Wi-Fi 信息',
      submitted: {
        title: '稍等片刻，NanoKVM 正在努力联网',
        tipsTitle: '接下来可以这样做',
        tipSwitch: '手机现在可以切回原来的 Wi-Fi 了',
        tipCheck: '稍后到 NanoKVM 的屏幕上查看新的 IP 地址，用浏览器访问它',
        tipRetry: '如果等了很久都没有动静，请确认网络名称和密码无误后再试一次'
      },
      invalidMode: '当前模式不支持配置网络。请先前往设备启用 Wi-Fi 配置模式。',
      confirmBtn: '确定',
      finishBtn: '完成',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: '密码',
      steps: {
        settings: '设置',
        wifi: 'Wi-Fi',
        enable: '启用'
      },
      scan: {
        title: '选择 Wi-Fi 网络',
        description: '选择附近的网络以继续。',
        loading: '正在搜索附近的 Wi-Fi 网络…',
        empty: '未找到附近的 Wi-Fi 网络。',
        failed: '无法扫描附近的 Wi-Fi 网络，请改为手动输入网络信息。',
        retryBtn: '重新扫描',
        manualBtn: '手动输入网络',
        backBtn: '返回附近网络',
        connectTitle: '连接到 {{ssid}}',
        connecting: '正在连接…',
        viewSimple: '切换到简约模式',
        viewDetailed: '切换到详细模式'
      },
      ap: {
        authTitle: '身份验证',
        authDescription: '请输入 AP 密码以继续',
        authFailed: '密码错误',
        passPlaceholder: 'AP 密码',
        verifyBtn: '验证'
      },
      qr: {
        cameraBtn: '拍照识别',
        pickBtn: '从相册选择',
        scanHint:
          '用相机拍摄 Wi-Fi 二维码，或从相册选择二维码截图，例如手机「分享 Wi-Fi」时显示的二维码。',
        decoding: '正在识别二维码…',
        notFound: '照片中未找到二维码。请让二维码居中、对焦清晰后重试。',
        invalid: '该二维码不包含可用的 Wi-Fi 信息。',
        pickerUnavailable:
          '相机或相册没有打开？当前登录窗口可能不允许，请在手机浏览器中打开以下地址：',
        confirmTitle: '将 NanoKVM 连接到 {{ssid}}？',
        confirmDescription: '以下网络信息来自二维码，请确认后再连接。',
        openNetwork: '该网络没有密码。',
        connectBtn: '连接',
        otherBtn: '选择其他网络'
      }
    },
    notification: {
      browser: {
        title: '浏览器推荐',
        description:
          '您当前使用的浏览器可能无法完全支持所有高级功能与特性，推荐使用 Chrome 以获得最佳体验。'
      },
      btnOk: '确认'
    },
    screen: {
      title: '屏幕',
      video: '视频模式',
      quality: '图像质量',
      qualityAuto: '自动',
      qualityLossless: '无损',
      qualityHigh: '高',
      qualityMedium: '中',
      qualityLow: '低',
      scale: '缩放',
      noSignal: '无视频信号',
      captureDisabled: '视频采集已关闭，请前往“设置 > 设备 > 视频采集”开启。',
      inconsistentVideoMode: '已暂停，其他视频模式正在播放中',
      bandwidthWarning: '当前网络带宽可能不足以承载视频设置，建议降低码率或帧率。',
      webrtcConnectionFailed: 'WebRTC 连接未建立，请切换到 Direct 视频模式。',
      webrtcUnavailable: {
        title: 'WebRTC 不可用',
        description: '请在浏览器中开启 WebRTC，或切换到 Direct 视频模式。'
      },
      upload: '上传',
      advanced: '高级设置',
      loading: '加载中'
    },
    volume: {
      title: '音量'
    },
    microphone: {
      title: '麦克风',
      notMounted: '麦克风未挂载',
      tip1: '远程主机尚未挂载虚拟麦克风，是否立即挂载并启用？',
      tip2: '您也可以前往 [设置 > 设备 > 虚拟音频] 进行管理。',
      confirm: '启用',
      cancel: '取消',
      noDevice: '当前设备没有可用的麦克风，请连接麦克风后重试',
      permissionDenied: '浏览器未获得麦克风权限，请在浏览器设置中允许访问',
      notReadable: '麦克风被其他程序占用或当前无法读取',
      overconstrained: '当前麦克风不支持所需的音频参数',
      securityBlocked: '当前页面不允许访问麦克风，请检查浏览器安全设置',
      aborted: '麦克风启动已中止，请重试',
      unsupported: '当前浏览器或访问方式不支持麦克风采集',
      enableFailed: '麦克风开启失败，请重试',
      disconnected: '控制端麦克风已断开，麦克风功能已关闭'
    },
    power: {
      title: '电源',
      showConfirm: '显示确认框',
      showConfirmTip: '电源操作需要二次确认',
      powerShort: '电源（短按）',
      powerLong: '电源（长按）',
      powerConfirm: '确认执行电源操作吗？',
      okBtn: '确认',
      cancelBtn: '取消'
    },
    keyboard: {
      errors: {
        paste: '文本未能发送到远程设备。请确认 KVM 连接正常后重试。'
      },
      title: '键盘',
      paste: '粘贴',
      tips: '仅支持标准字母和字符（ASCII)',
      placeholder: '请输入内容',
      submit: '确定',
      virtual: '虚拟键盘',
      swapCommandCtrl: '交换 Cmd 和 Ctrl',
      shortcut: {
        title: '快捷键',
        custom: '自定义',
        capture: '点击此处捕获快捷键',
        clear: '清空',
        save: '保存',
        captureTips: '捕获系统级按键（如 Windows 键）需要全屏权限。',
        enterFullScreen: '切换全屏模式。'
      },
      leaderKey: {
        title: '引导键',
        desc: '绕过浏览器限制，向远程主机发送被系统拦截的快捷键。',
        howToUse: '使用方法',
        simultaneous: {
          title: '组合按键模式',
          desc1: '按住引导键不放，同时按下目标快捷键。',
          desc2: '操作直观，但少数快捷键可能因系统占用而无法生效。'
        },
        sequential: {
          title: '序列输入模式',
          desc1: '点击【引导键】开始 → 依次点击【快捷键】→ 点击【引导键】结束。',
          desc2: '步骤较多，但能完美避开系统键位冲突。'
        },
        enable: '启用引导键',
        tip: '设为引导键后，该按键将仅用于触发快捷键，不再作为普通按键使用。',
        placeholder: '请按下引导键',
        shiftRight: '右 Shift',
        ctrlRight: '右 Ctrl',
        metaRight: '右 Win',
        submit: '确定',
        recorder: {
          rec: '录制中',
          activate: '激活按键',
          input: '请按下快捷键...'
        }
      }
    },
    mouse: {
      title: '鼠标',
      mode: '鼠标模式',
      absolute: '跟随式鼠标',
      absoluteAndroid: '跟随式鼠标（更兼容Android）',
      touchScreen: '多点触摸屏',
      relative: '独占式鼠标',
      direction: '滚轮方向',
      scrollNatural: '自然',
      scrollConventional: '传统',
      speed: '滚轮速度',
      scrollStep: '触控板滚动',
      fast: '快',
      slow: '慢',
      requestPointer: '独占式鼠标已启用，请点击画面锁定并隐藏本地指针。',
      repairIphoneDrag: '修复 iPhone 拖拽',
      resetHid: '重置 HID',
      touchpadGuide: {
        title: '指针输入指南',
        scope: '适用于普通指针输入且鼠标模式为相对模式时。',
        swipeTitle: '滑动移动',
        swipeDesc: '在有效画面区域内滑动，远端鼠标指针会跟随移动。',
        tapTitle: '点按单击',
        tapDesc: '短促点按一次，对应远端鼠标左键单击。',
        holdTitle: '按住触发左键按下',
        holdDesc: '手指按下并保持不动约 1 秒，会进入鼠标左键按住状态。',
        dragTitle: '按住后移动进行拖动',
        dragDesc: '左键按住状态生效后继续移动手指，会变成按住鼠标左键的拖动。'
      },
      hidOnly: {
        errors: {
          load: '未能读取 HID-Only 模式状态。请稍后重试。',
          update: '无法确认 HID-Only 模式是否已更新。请重新打开设置查看当前状态后再试。'
        },
        title: 'HID-Only 模式',
        desc: '若使用过程中遇到鼠标键盘无响应，且重置 HID 无效，可能是 NanoKVM 与您的设备存在兼容性问题。建议尝试启用 HID-Only 模式以提升兼容性。',
        tip1: '启用该模式会禁用虚拟U盘和虚拟网卡',
        tip2: '该模式下无法使用镜像挂载功能',
        enable: '启用 HID-Only 模式',
        disable: '关闭 HID-Only 模式'
      }
    },
    preview: {
      title: '预览'
    },
    image: {
      storageReadOnly: '文件存储当前不可写。请关闭虚拟 U 盘。',
      storageBusy: '存储正在使用中，请完成或取消文件传输后再切换虚拟 U 盘。',
      title: '镜像',
      loading: '加载中',
      empty: '无镜像文件',
      upload: '上传镜像',
      dropFile: '将镜像拖到此处，或点击选择文件',
      uploadAction: '上传',
      cancelAction: '取消',
      mount: '挂载镜像',
      mounted: '已挂载',
      uploadError: '上传失败，请重试',
      uploadInsufficientStorage: '存储空间不足',
      uploadTooLarge: '镜像超出大小限制',
      cancelUploadError: '取消上传失败',
      mountMode: '模式',
      massStorage: '大容量存储',
      cdRom: 'CD ROM',
      readOnly: '只读',
      mountFailed: '挂载失败',
      mountDesc: '在某些系统中，需要在远程主机中弹出虚拟硬盘后再挂载镜像。',
      unmountFailed: '卸载失败',
      unmountDesc: '在某些系统中，需要在远程主机中手动弹出后再卸载镜像。',
      refresh: '刷新镜像列表',
      download: '下载镜像',
      attention: '注意',
      deleteConfirm: '确定要删除该镜像吗？',
      deleteFailed: '删除镜像失败',
      okBtn: '确定',
      cancelBtn: '取消'
    },
    script: {
      title: '脚本',
      upload: '上传',
      run: '运行',
      runBackground: '后台运行',
      runFailed: '运行失败',
      uploadFailed: '上传脚本失败',
      deleteFailed: '删除脚本失败',
      attention: '注意',
      delDesc: '确定要删除该文件吗？',
      confirm: '确定',
      cancel: '取消',
      delete: '删除',
      close: '关闭'
    },
    terminal: {
      title: '终端',
      nanokvm: 'NanoKVM 终端',
      confirm: '确定',
      submit: '提交',
      username: '用户名',
      password: '密码',
      authCode: '认证码',
      authorizeTitle: '授权 Web 终端',
      codeAccountHint: '请输入认证码或未使用的恢复代码。',
      authFailed: '身份验证失败'
    },
    wol: {
      errors: {
        send: '未能发送唤醒指令。请检查 MAC 地址和网络连接后重试。',
        loadFailed: '未能加载网络唤醒设备列表。请确认 NanoKVM 在线后重试。',
        renameFailed: '未能重命名该设备。请检查 MAC 地址后重试。',
        deleteFailed: '未能删除该设备。请稍后重试。'
      },
      title: '网络唤醒',
      sending: '指令发送中...',
      sent: '指令已发送',
      input: '请输入MAC地址',
      ok: '确定'
    },
    download: {
      sha256: 'SHA-256（可选）',
      sha256Placeholder: '请输入 64 位 SHA-256 校验和',
      invalidSHA256: 'SHA-256 必须是 64 位十六进制字符串',
      title: '下载镜像',
      input: '请输入镜像的下载地址',
      ok: '确定',
      disabled: '/data 是只读分区，无法下载镜像',
      downloadingFile: '正在下载：{{file}}',
      downloadingProgress: '正在下载（{{percentage}}）：{{file}}',
      failed: '下载失败',
      insufficientStorage: '存储空间不足',
      remaining: '剩余 {{time}}',
      cancel: '取消',
      cancelFailed: '取消下载失败',
      pause: '暂停',
      resume: '继续',
      pauseFailed: '暂停下载失败',
      resumeFailed: '继续下载失败'
    },
    settings: {
      back: '返回',
      close: '关闭',
      ai: {
        errors: {
          loadServices: '未能加载 AI 服务状态。请确认 NanoKVM 在线后重试。',
          loadMcpConfig: '未能加载 MCP 服务配置。请稍后重试。',
          updateMcpService: 'MCP 服务状态未能更新。请确认当前状态后再试。',
          accessMcpApiKey: '无法获取 MCP API Key。请重试，并按提示重新完成验证。',
          regenerateMcpApiKey: 'MCP API Key 未能重新生成。请稍后重试。',
          regeneratedMcpApiKeyUnavailable:
            'API Key 已重新生成，但暂时无法显示。请重新尝试显示；原有 Key 已失效。',
          loadMemoryFabricState: '未能加载记忆织网服务状态。请稍后重试。',
          updateMemoryFabricService: '记忆织网服务状态未能更新。请确认当前状态后再试。',
          loadScreenTimelapseState: '未能加载延时回放服务状态。请稍后重试。',
          updateScreenTimelapseService: '延时回放服务状态未能更新。请确认当前状态后再试。',
          openScreenTimelapse: '无法打开延时回放。请稍后重试。',
          loadScreenTimelapseRecordings: '未能加载延时回放记录。请稍后重试。',
          playScreenTimelapse: '当前延时回放暂时无法播放。请重新选择日期或稍后重试。',
          screenTimelapseFullscreen: '无法切换全屏状态。请检查浏览器权限后重试。'
        },
        title: 'AI',
        mcpService: 'MCP 服务',
        mcpServiceDesc: '为控制、媒体和已启用的记忆织网工具提供经过鉴权的统一入口',
        memoryFabricService: '记忆织网',
        memoryFabricServiceDesc: '在本机处理 Agent 上下文记忆和 OCR',
        memoryFabricRemoteRequiresMcp: '请启用上方的 MCP 服务，才能远程使用记忆和 OCR 工具。',
        memoryFabricRemoteAvailable:
          '记忆和 OCR 工具已通过上方的 /api/mcp 端点开放，并使用同一个 API Key。',
        memoryFabricConfig: {
          title: '配置',
          description: '仅管理记忆织网的默认模型设置。',
          loadFailed: '无法加载记忆织网配置。',
          saveFailed: '无法保存记忆织网配置。',
          saved: '配置已保存。',
          savedRestarted: '配置已保存，记忆织网已重启。',
          noChanges: '配置没有变化。',
          reload: '重新加载',
          save: '保存',
          unsaved: '有未保存的更改',
          conflictTitle: '配置已发生变化',
          conflictDescription: '配置已在其他位置被修改。请先加载最新版本，再重新保存。',
          conflictReload: '加载最新配置',
          discardTitle: '放弃未保存的更改？',
          discardDescription: '重新加载会用最新保存的配置替换当前表单内容。',
          discardConfirm: '放弃并重新加载',
          sections: {
            models: '模型'
          },
          openAICompatibleHint:
            '支持 OpenAI Compatible 接口（兼容 OpenAI API 格式）。请填写服务商提供的模型名称、Base URL 和 API Key。',
          fields: {
            model: '模型',
            baseUrl: '基础 URL',
            maxTokens: '最大 Token 数',
            timeoutSeconds: '超时时间（秒）'
          },
          apiKey: {
            title: 'API Key',
            configured: '已配置',
            notConfigured: '未配置',
            keep: '保留现有 Key',
            replace: '替换 Key',
            clear: '清除 Key',
            replacement: '新 API Key',
            replacementPlaceholder: '输入新的 API Key',
            clearHint: '已保存的 API Key 将被移除；环境变量仍可能提供 Key。',
            required: '请输入新的 API Key。'
          },
          validation: {
            required: '此项为必填项。',
            invalidUrl: '请输入有效的 HTTP 或 HTTPS URL。'
          }
        },
        screenTimelapseService: '屏幕延时回放',
        screenTimelapseServiceDesc: '回放 Memory Fabric 屏幕历史，并搜索已捕获的屏幕内容',
        screenTimelapseSearchRequiresMemoryFabric: '如需搜索录制内容，请同时开启记忆织网。',
        screenTimelapseOpen: '打开',
        screenTimelapseSearchPlaceholder: '搜索视频内容...',
        screenTimelapseNoVideos: '未找到延时回放视频',
        screenTimelapseNoResults: '未找到匹配画面',
        screenTimelapseMultipleResults: '搜索结果',
        screenTimelapseUnavailable: '不可用',
        screenTimelapsePlayerUnsupported: '当前浏览器无法播放延时回放视频',
        screenTimelapseSearchFailed: '搜索失败',
        screenTimelapseLoading: '正在加载延时回放...',
        screenTimelapseSearchAction: '搜索延时回放',
        screenTimelapsePlay: '播放延时回放',
        screenTimelapsePause: '暂停延时回放',
        screenTimelapseEnterFullscreen: '进入全屏',
        screenTimelapseExitFullscreen: '退出全屏',
        screenTimelapseFullscreenUnsupported: '当前浏览器无法进入全屏',
        screenTimelapsePreviousDay: '前一天',
        screenTimelapseNextDay: '后一天',
        screenTimelapseDeleteDay: '删除 {{date}} 的回放',
        screenTimelapseDelete: '删除',
        screenTimelapseDeleteConfirmTitle: '删除延时回放？',
        screenTimelapseDeleteConfirmDesc:
          '{{date}} 的回放占用 {{size}}。删除后无法恢复，是否继续？',
        screenTimelapseDeleteSuccess: '已删除 {{date}} 的回放',
        screenTimelapseDeleteFailed: '删除 {{date}} 的回放失败',
        endpoint: '端点',
        apiKey: 'API Key',
        regenerate: '重新生成',
        copyApiKey: '复制 API Key',
        copyEndpoint: '复制端点',
        showApiKey: '显示 API Key',
        regenerateConfirmTitle: '重新生成 API key',
        regenerateConfirmDesc:
          '重新生成后，现有的 MCP 客户端必须更新为新的 key 才能继续使用。确定要继续吗？',
        okBtn: '确定',
        cancelBtn: '取消'
      },
      title: '设置',
      about: {
        title: '关于 NanoKVM',
        information: '信息',
        ip: 'IP',
        mdns: 'mDNS',
        application: '应用版本',
        applicationTip: 'NanoKVM 网页应用版本',
        arch: '指令架构',
        device_number: '设备编号',
        image: '镜像版本',
        imageTip: 'NanoKVM 系统镜像版本',
        deviceKey: '设备码',
        community: '社区',
        hostname: '主机名',
        hostnameUpdated: '主机名修改成功，重启后生效',
        ipType: {
          Wired: '有线',
          Wireless: '无线',
          Other: '其他'
        }
      },
      appearance: {
        title: '外观',
        display: '显示',
        language: '语言',
        languageDesc: '选择界面语言',
        languageChangeFailed: '语言切换失败',
        webTitle: '网页标题',
        webTitleDesc: '自定义网站页面标题',
        menuBar: {
          title: '菜单栏',
          mode: '显示方式',
          modeDesc: '菜单栏在屏幕上的显示方式',
          modeOff: '关闭',
          modeAuto: '自动隐藏',
          modeAlways: '始终显示',
          icons: '菜单图标',
          iconsDesc: '是否在菜单栏中显示子菜单图标'
        }
      },
      screen: {
        title: '屏幕',
        videoMode: {
          title: '视频模式',
          description: '选择视频编码和传输方式'
        },
        edid: {
          description: '选择显示器参数',
          default: '默认',
          custom: '自定义',
          document: '文档',
          monitorName: '显示器名称',
          monitorNameDescription: '最多 13 个可打印 ASCII 字符；留空\n则使用 EDID 原名称',
          monitorNamePlaceholder: '使用 EDID 原名称',
          monitorNameApply: '应用',
          monitorNameInvalid: '仅支持最多 13 个可打印 ASCII 字符',
          monitorNameUnsupported: '当前 EDID 不支持显示器名称覆盖',
          monitorNameLoadFailed: '读取显示器名称失败',
          monitorNameUpdateFailed: '更新显示器名称失败',
          powerConfirmTitle: '更新 EDID 前请确认',
          powerConfirmDescription:
            '请先接入 PD 电源。更新 EDID 可能导致显示短暂断开；如果更新后设备无法正确识别显示器，请手动重启设备。',
          powerConfirmOk: '继续更新',
          powerConfirmOkCountdown: '继续更新（{{seconds}}）',
          powerConfirmCancel: '取消',
          errors: {
            badRequest: 'EDID 请求无效',
            invalidFileName: '请选择名称有效的 .bin EDID 文件',
            fileTooLarge: 'EDID 文件过大',
            invalidLength: 'EDID 必须为 128、256、384 或 512 字节',
            invalidHeader: 'EDID 文件头无效',
            unsupportedVersion: '不支持此 EDID 版本',
            extensionCountMismatch: 'EDID 扩展块数量与文件长度不匹配',
            invalidChecksum: 'EDID 校验和无效',
            invalidCta: 'CTA 扩展块结构无效',
            currentCannotOverwrite: '无法覆盖当前正在使用的 EDID，请先切换到其他 EDID',
            saveFailed: '保存 EDID 失败',
            switchFailed: '切换 EDID 失败',
            uploadFailed: '上传 EDID 失败',
            deleteFailed: '删除 EDID 失败',
            networkFailed: '网络错误，请重试'
          }
        },
        rateControlMode: {
          title: '码率控制模式',
          description: '选择控制码率的方式',
          cbr: 'CBR（固定码率）：优先保证连接稳定性。最小化卡顿，但快速变化的画面可能会出现模糊或噪点。',
          vbr: 'VBR（动态码率）：优先保证图像清晰度。快速变化的画面会导致带宽占用增加，可能引起延迟或卡顿。'
        },
        bitrate: {
          title: '码率',
          description: '调节画面清晰度与带宽占用的平衡',
          lower: '较小的码率可以降低网络不佳时的延迟，但会牺牲图像质量，可能会出现花屏的情况。',
          higher: '较大的码率可以提高图像清晰度，但会消耗更多网络流量，如果网络不佳可能会增加延迟。'
        },
        gop: {
          description: '设置完整关键帧之间的间隔',
          lower:
            '数值越小发送关键帧越频繁。这可以加快画面从花屏中恢复的速度，但会损失画面细节，并且消耗更多的网络流量。',
          higher:
            '数值越大发送关键帧越少，会更多的发送增量更新。这可以增加画面细节并节省带宽，但画面花屏后恢复所需的时间会更长。'
        },
        quality: {
          title: '画质',
          description: '设置显示画面的质量'
        },
        fps: {
          auto: '自动',
          description: '设置画面显示帧率',
          tip1: '设为 0 即自动模式，将按照远程主机的帧率来显示。',
          tip2: '降低帧率可显著减少网络带宽占用，建议在网络波动或无需高帧率时手动调低。'
        },
        scale: {
          title: '缩放',
          description: '设置显示画面的缩放比例'
        }
      },
      device: {
        title: '设备',
        virtualAudioConfirm: {
          title: '切换虚拟音频前请确认',
          description:
            '请先接入 PD 电源。切换虚拟音频可能导致显示短暂断开；如果切换后设备无法正确识别显示器，请手动重启设备。',
          ok: '继续切换',
          okCountdown: '继续切换（{{seconds}}）',
          cancel: '取消'
        },
        wifi: {
          title: 'Wi-Fi',
          description: '配置 Wi-Fi 信息',
          apMode: 'AP 模式已启用，该模式下仅支持手动连接',
          others: '其他...',
          connect: '连接 Wi-Fi',
          connectDesc1: '请输入网络名称和密码',
          connectDesc2: '请输入密码以连接此网络',
          disconnect: '是否要断开该网络连接？',
          ssid: '名称',
          password: '密码',
          joinBtn: '加入',
          confirmBtn: '确定',
          cancelBtn: '取消',
          preset: {
            title: 'Wi-Fi 预存',
            description: '提前预存一个目标 Wi-Fi，设备开机后自动连接',
            empty: '无',
            addBtn: '添加预存',
            removeBtn: '删除预存',
            saveTitle: '预存 Wi-Fi',
            saveDesc: '现在保存目标网络，设备开机发现该网络后自动连接',
            saveBtn: '保存',
            passwordPlaceholder: '密码（开放网络可留空）',
            saveFailed: '保存预存 Wi-Fi 失败',
            removeFailed: '删除预存 Wi-Fi 失败',
            removeConfirm: '确定要删除已预存的 Wi-Fi 吗？',
            warning:
              '开启后，设备每次启动时若检测到该 Wi-Fi 将尽力连接，可能干扰连接其他 Wi-Fi；预存将持续生效，直到您修改或删除。'
          }
        },
        capture: {
          title: '视频采集',
          description: '启用视频采集'
        },
        ssh: {
          description: '启用 SSH 远程访问',
          tip: '启用后请在终端中为 root 用户设置强密码。'
        },
        swap: {
          errors: {
            load: '未能读取交换内存状态。请稍后重试。',
            update: '交换内存设置未能更新。请确认当前状态后再试。'
          },
          title: '交换内存',
          description: '启用压缩交换内存'
        },
        advanced: '高级设置',
        staticIp: {
          title: '静态 IP',
          config: '设置',
          tip: '建议在可物理接触设备的情况下进行设置，以防止配置错误导致设备无法远程访问',
          document: '查看文档',
          enable: '启用静态 IP',
          placeholder: '格式：IP/CIDR [Gateway]\n示例：192.168.1.10/22',
          notEmpty: 'IP 地址不能为空',
          notValid: 'IP 地址内容有误',
          failed: '设置失败',
          success: '设置完成，请检查 IP 是否可用',
          okBtn: '确定',
          cancelBtn: '取消'
        },
        usbNetwork: {
          title: 'USB 网卡',
          mode: '模式',
          server: '服务端',
          serverDesc: '通过内置 DHCP 服务为远程主机分配 IP 地址。',
          client: '客户端',
          clientDesc: '通过 DHCP 从远程主机获取 IP 地址。',
          forwarding: 'IP 转发',
          forwardingTip: '如需让远程主机通过 NanoKVM 访问互联网，请开启 IP 转发。',
          manualIp: '手动设置 IP',
          ipv4Placeholder: 'IPv4/CIDR，例如：10.10.10.1/24',
          invalidIpv4: '请输入有效的 IPv4/CIDR 地址。',
          loadFailed: 'USB 网卡设置读取失败。',
          failed: '设置失败。',
          success: '设置完成。',
          applyBtn: '应用',
          cancelBtn: '取消'
        },
        usbId: {
          title: 'USB 设备标识',
          config: '设置',
          tip: '自定义 USB 设备标识会改变主机识别本设备的方式。应用后将重启 USB 设备，键盘、鼠标或网络可能短暂断开。可选字段留空则使用脚本默认值。',
          enable: '启用自定义 USB 设备标识',
          vid: '厂商 ID (VID)',
          pid: '产品 ID (PID)',
          bcdDevice: '设备版本 (bcdDevice)',
          serialnumber: '序列号',
          manufacturer: '厂商名',
          product: '产品名',
          vidPlaceholder: '例如：0x359F',
          pidPlaceholder: '例如：0x3305',
          bcdDevicePlaceholder: '例如：0x0101（可选）',
          serialnumberPlaceholder: '例如：0123456789ABCDEF（可选）',
          manufacturerPlaceholder: '例如：sipeed（可选）',
          productPlaceholder: '例如：NanoKVM-Go（可选）',
          notEmpty: 'VID 与 PID 不能为空',
          notValid: 'VID/PID 格式无效，请使用十六进制，如 0x359F',
          bcdNotValid: 'bcdDevice 格式无效，请使用十六进制如 0x0101，或留空',
          failed: '设置失败',
          success: '设置完成，USB 设备已重启',
          okBtn: '确定',
          cancelBtn: '取消'
        },
        diskInquiry: {
          title: 'U 盘显示名称',
          config: '设置',
          tip: '远程电脑上看到的虚拟 U 盘名称。保存后会重启 USB，键盘、鼠标或网络可能短暂断开。',
          enable: '自定义 U 盘名称',
          placeholder: '例如：sipeed  Flash Drive     1.00',
          notEmpty: '名称不能为空',
          notValid: '名称无效，最多 28 个可打印 ASCII 字符且不能换行',
          failed: '设置失败',
          success: '设置完成，USB 设备已重启',
          okBtn: '确定',
          cancelBtn: '取消'
        },
        diskInquiryCdrom: {
          title: '虚拟光驱名称',
          config: '设置',
          tip: '远程电脑上看到的虚拟光驱 / CD 名称。保存后会重启 USB，键盘、鼠标或网络可能短暂断开。',
          enable: '自定义虚拟光驱名称',
          placeholder: '例如：sipeed  Optical Drive   1.00',
          notEmpty: '名称不能为空',
          notValid: '名称无效，最多 28 个可打印 ASCII 字符且不能换行',
          failed: '设置失败',
          success: '设置完成，USB 设备已重启',
          okBtn: '确定',
          cancelBtn: '取消'
        },
        uac2FunctionName: {
          title: 'USB 音频名称',
          config: '设置',
          tip: '远程电脑上看到的虚拟 USB 麦克风 / 音频设备名称。保存后会重启 USB，键盘、鼠标或网络可能短暂断开。',
          enable: '自定义 USB 音频名称',
          placeholder: '例如：NanoKVM-Go Audio',
          notEmpty: '名称不能为空',
          notValid: '名称无效，最多 64 个可打印字符且不能换行',
          failed: '设置失败',
          success: '设置完成，USB 设备已重启',
          okBtn: '确定',
          cancelBtn: '取消'
        },
        lowPower: {
          title: '低功耗模式',
          description: '减少空闲时的功耗',
          tips: '提示',
          tip1: '开启后，设备将在空闲时暂停视频采集，以降低功耗与发热。',
          tip2: '恢复视频采集可能需要重新初始化，期间可能短暂无画面。',
          okBtn: '确定',
          cancelBtn: '取消'
        },
        mouseJiggler: {
          title: '鼠标抖动',
          description: '防止远程主机休眠',
          disable: '关闭',
          absolute: '绝对模式',
          relative: '相对模式'
        },
        fingerRobot: {
          title: '手指机器人',
          description: '启用手指机器人控制'
        },
        mdns: {
          description: '启用 mDNS 发现服务'
        },
        hidOnly: 'HID-Only 模式',
        imageMounted: '镜像已挂载',
        disk: '虚拟 U 盘',
        diskDesc: '在远程主机中挂载虚拟 U 盘',
        sdCard: {
          warning: '初始化虚拟 U 盘',
          tip1: '首次启用将初始化虚拟 U 盘，并清除设备上已有的镜像文件及虚拟 U 盘数据。此操作无法撤销，请先备份重要文件。',
          tip2: '是否继续初始化？',
          capacity: '容量',
          disable: '禁用',
          disableWarning: '禁用虚拟 U 盘？',
          disableTip:
            '禁用后将清除虚拟 U 盘中的全部文件，并释放其占用的设备存储空间。下次启用时需要重新选择容量并初始化。此操作无法撤销。'
        },
        network: '虚拟网卡',
        networkDesc: '在远程主机中挂载虚拟网卡',
        mic: '虚拟音频',
        micDesc: '选择向远程主机提供的 USB 虚拟音频模式',
        virtualAudioMode: {
          off: '关闭',
          mic: '仅麦克风',
          duplex: '麦克风 + 扬声器'
        },
        speaker: '虚拟扬声器',
        speakerDesc:
          '开启后替换默认DP音频输出，用于支持远程通话，兼容性更强。关闭则只能转发媒体音频，而无法转发通话音频。',
        micTip:
          '受 USB 端点资源限制，启用虚拟麦克风将自动禁用 USB 副屏功能。\n在进行 BIOS 操作时建议关闭虚拟麦克风，以确保最佳兼容性。',
        reboot: '重新启动',
        rebootDesc: '您确定要重新启动 NanoKVM 吗？',
        rebootToFlashing: '重启到烧录模式',
        rebootToFlashingButton: '重启',
        rebootToFlashingTitle: '重启到烧录模式？',
        rebootToFlashingDesc:
          'NanoKVM 将重启并进入 USB 烧录模式，请确保已连接用于烧录的 USB 接口，并准备好烧录工具。',
        datetime: {
          timezone: '时区',
          datetime: '时间',
          dateAndTime: '日期和时间',
          synchronize: '同步时间',
          lastSynchronization: '上次同步：',
          notSynchronized: '系统时间尚未同步',
          syncNow: '立即同步',
          manualSet: '手动设置',
          manualSetTitle: '手动设置时间',
          manualSetDescription: '时间修改后需要重新登录。',
          useBrowserTime: '使用浏览器当前时间',
          setTime: '设置时间',
          invalidTime: '请输入有效的日期和时间。',
          setFailed: '系统时间设置失败，请重试。',
          setSucceeded: '系统时间已更新，请重新登录。'
        },
        dialBackground: {
          errors: {
            load: '未能加载表盘背景图片。请稍后重试。',
            upload: '背景图片未能上传。请确认文件是有效的 BMP 图片后重试。',
            switch: '表盘背景未能切换。请刷新图片列表后重试。'
          },
          title: '表盘背景',
          description: '从 /kvmcomm/ui/bg/ 选择 BMP 图片作为表盘背景',
          upload: '上传',
          loading: '加载中...',
          empty: '暂无 BMP 图片',
          current: '当前',
          onlyBmp: '仅支持 BMP 图片',
          uploaded: '上传完成',
          switched: '背景已切换'
        },
        okBtn: '是',
        cancelBtn: '否'
      },
      tailscale: {
        title: 'Tailscale',
        restart: '确定要重启 Tailscale 吗？',
        stop: '确定要停止 Tailscale 吗？',
        stopDesc: '该操作会退出登录，并停止开机自动启动。',
        loading: '加载中...',
        notInstall: '未检测到 Tailscale，请先安装',
        install: '安装',
        installing: '安装中',
        failed: '安装失败',
        retry: '请刷新后重试，或尝试手动安装',
        download: '下载',
        package: '安装包',
        unzip: '并解压',
        upTailscale: '将 tailscale 文件上传到 /usr/bin/ 目录',
        upTailscaled: '将 tailscaled 文件上传到 /usr/sbin/ 目录',
        refresh: '刷新页面',
        notRunning: 'Tailscale 尚未运行，请先执行启动操作',
        run: '启动',
        notLogin: '该设备尚未绑定，请点击登录并将这台设备绑定到您的账号。',
        urlPeriod: '该链接10分钟内有效',
        login: '登录',
        loginSuccess: '登录完成',
        enable: '启用 Tailscale',
        exitNode: '出口节点',
        exitNodeDesc: '将此设备发布为 tailnet 的互联网出口。',
        subnetRoute: '子网路由',
        subnetRouteDesc: '向 tailnet 发布本机直连局域网网段 {{subnet}}。',
        subnetUnavailable: '未检测到可发布的直连局域网网段。',
        routingUnavailable: '此设备当前无法配置 Tailscale 路由。',
        pendingApproval: '启用后，请在 Tailscale 管理后台批准发布的路由。',
        routingUpdateFailed: '更新 Tailscale 路由设置失败。',
        deviceName: '设备名称',
        deviceIP: '设备地址',
        account: '账号',
        logout: '退出',
        logoutDesc: '确定要退出吗？',
        uninstall: '卸载 Tailscale',
        uninstallDesc: '确定要卸载 Tailscale 吗？',
        okBtn: '确认',
        cancelBtn: '取消'
      },
      update: {
        title: '检查更新',
        queryFailed: '获取版本号失败',
        updateFailed: '更新失败，请重试',
        isLatest: '已经是最新版本。',
        available: '有新的可用版本，确定要更新吗？',
        updating: '更新中，请稍候...',
        confirm: '确定',
        upgrade_tip_nano: 'NanoKVM 新版本现已推出。您确定要立即更新吗？',
        upgrade_tip_firmware: '有新的固件版本可用。您确定要立即更新吗？',
        cancel: '取消',
        preview: '预览更新',
        previewDesc: '率先体验即将推出的新功能和优化',
        previewTip: '预览版更新可能包含一些不稳定因素或未完善的功能！',
        download: '下载',
        install: '安装',
        changelog: '更新日志',
        restart: '重启服务'
      },
      account: {
        title: '帐号',
        webAccount: '网页帐号',
        password: '密码',
        twoFactor: '双重验证',
        enableTwoFactor: '启用',
        enableTwoFactorDesc: '请输入当前密码，以开始设置双重验证。',
        disableTwoFactor: '禁用',
        recoveryCodes: '恢复代码',
        enableRecoveryCodes: '启用',
        updateRecoveryCodes: '更新',
        recoveryCodesTooltipDisabled: '恢复代码是备用双重验证凭证，只有启用双重验证后才能生成。',
        recoveryCodesTooltipEnabled:
          '恢复代码只在双重验证启用时有效，关闭双重验证会使所有恢复代码失效；在双重验证保持启用时只能更新。',
        recoveryCodesRegenerateDesc:
          '请输入当前密码，以及认证码或未使用的恢复代码。更新恢复代码会使所有现有恢复代码失效。',
        enableRecoveryCodesDesc:
          '请输入当前密码和认证码以生成恢复代码。恢复代码只能查看一次，请在离开本页前保存。',
        recoveryCodesVerifyFailed: '无法验证密码，以及认证码或恢复代码。',
        recoveryCodesSavedTitle: '请立即保存这些恢复代码',
        recoveryCodesSavedDesc: '恢复代码只能查看一次。请在离开本页前将其保存到安全位置。',
        copyRecoveryCodes: '复制',
        downloadRecoveryCodes: '下载',
        recoveryCodesCopyFailed: '无法复制恢复代码。',
        done: '完成',
        currentPassword: '当前密码',
        authCode: '认证码',
        authCodeOrRecoveryCode: '认证码或恢复代码',
        authRateLimited: '连续错误次数过多，请在 {{seconds}} 秒后重试。',
        verifyPasswordChange: '验证密码修改',
        verifyPasswordChangeDesc: '修改密码前，请输入当前密码，以及认证码或未使用的恢复代码。',
        verifyPasswordChangeFailed: '无法验证密码，以及认证码或恢复代码。',
        verifyAndEnable: '验证并启用',
        scanTwoFactorQRCode: '使用认证器应用扫描此二维码，然后输入认证码。',
        showSetupKey: '显示设置密钥',
        hideSetupKey: '隐藏设置密钥',
        setupKey: '设置密钥',
        twoFactorSetupFailed: '无法开始双重验证设置。',
        twoFactorVerifyFailed: '无法验证认证码。',
        twoFactorDisableFailed: '无法禁用双重验证。',
        disableTwoFactorDesc:
          '请输入当前密码，以及认证码或未使用的恢复代码。禁用双重验证会使所有恢复代码失效。',
        updateBtn: '修改',
        logoutBtn: '退出',
        logoutDesc: '确定要退出吗？',
        okBtn: '确定',
        cancelBtn: '取消',
        twoFactorProtection: {
          title: '高级双重验证保护',
          description:
            '选择在启用保护强制执行后需要进行双重验证的敏感操作。「直接网页登录」开关相反：开启后仅使用认证码或未使用的恢复代码即可登录。',
          login: '直接网页登录',
          webTerminal: 'Web 终端',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API 密钥',
          targetPower: '目标机电源',
          nanoKVMRestart: 'NanoKVM 重启',
          deleteImage: '删除镜像',
          scripts: '脚本操作',
          screenTimelapse: '屏幕延时录制',
          loginScope:
            '允许网页仅使用认证码或未使用的恢复代码登录，不再输入用户名和密码。这会降低登录安全性。',
          webTerminalScope: '打开 Web 终端。',
          sshScope: '更改 SSH 服务状态。',
          tailscaleScope: '安装、连接或更改 Tailscale。',
          frp: 'FRP',
          frpScope: '安装、连接或更改 FRP。',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: '安装、连接或更改 Cloudflare Tunnel。',
          zerotierScope: '安装、连接或更改 ZeroTier。',
          netbirdScope: '安装、连接或更改 NetBird。',
          mcpScope: '显示、复制或重新生成 MCP API 密钥，以及启用或停用 MCP。',
          targetPowerScope: '通过物理电源按钮关闭或重启目标机。',
          nanoKVMRestartScope: '重启 NanoKVM 本身。',
          deleteImageScope: '永久删除已存储的 ISO 或 IMG 镜像。',
          scriptsScope: '运行、上传或删除脚本。',
          screenTimelapseScope: '启用屏幕延时录制、浏览录制内容及删除录制内容。',
          authorizeTitle: '授权更改保护设置',
          authorizeDesc: '请输入认证码或未使用的恢复代码。授权将在五分钟内有效。',
          authorizeFailed: '无法验证认证码或恢复代码。',
          loadFailed: '无法加载保护设置。',
          updateFailed: '无法保存保护设置。',
          retry: '重试',
          disableScriptsWarningTitle: '关闭脚本保护？',
          disableScriptsWarningDesc:
            '如果不做保护，恶意脚本可能获得此设备的高权限访问，并绕过或移除 2FA 保护。',
          cancelDisabling: '取消',
          continueDisabling: '继续关闭',
          continueDisablingCountdown: '继续关闭（{{seconds}} 秒）'
        },
        twoFactorGuide: {
          title: '双重验证指南',
          scope: '双重验证会为您的帐号增加额外的安全保护。',
          whatTitle: '什么是 2FA？',
          whatDesc:
            '双重验证（2FA）使用认证器应用生成的基于时间的一次性代码保护您的帐号。默认情况下需要在密码之外输入该代码，即使密码泄露，帐号仍能获得保护。如果启用了直接网页登录，仅凭代码即可登录。',
          appsTitle: '推荐应用',
          appsDesc:
            '您可以使用 Google Authenticator 或 Microsoft Authenticator。两者都免费，并支持 iOS 和 Android。',
          setupTitle: '如何设置',
          setupDesc:
            '打开认证器应用，扫描设置时显示的二维码，或手动输入设置密钥。然后输入应用中的 6 位代码以完成启用 2FA。',
          recoveryTitle: '恢复代码',
          recoveryDesc:
            '恢复代码是一次性的备用代码。如果您无法使用认证器应用，可以使用恢复代码登录。生成后请立即保存在安全位置。',
          logoutNotice: '启用 2FA 会退出所有活动会话，包括当前会话。您需要在每台设备上重新登录。'
        }
      }
    },
    error: {
      title: '我们遇到了问题',
      refresh: '刷新'
    },
    pictureInPicture: {
      title: '画中画',
      enter: '进入画中画',
      exit: '退出画中画',
      error: '无法开启画中画，请等待画面加载后重试。'
    },
    fullscreen: {
      toggle: '切换全屏'
    },
    menu: {
      collapse: '收起',
      expand: '展开'
    }
  }
};

export default zh;
