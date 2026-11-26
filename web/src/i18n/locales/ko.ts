const ko = {
  translation: {
    storage: {
      cards: { mount: '디스크 마운트', share: '파일 공유' },
      title: '이미지 및 파일 공유',
      empty: '아직 이미지가 없습니다',
      notAttached: '마운트된 항목 없음',
      attached: '마운트됨: {{name}}',
      eject: '꺼내기',
      mount: '마운트',
      mountMode: '마운트 모드',
      mountModeDisk: '대용량 저장소',
      mountModeCdrom: 'CD-ROM',
      openManager: '파일 관리자 열기',
      refresh: '새로 고침',
      delete: '삭제',
      deleteTitle: '미디어 삭제',
      deleteConfirm: '{{name}}을(를) 삭제할까요? 안에 있는 파일이 사라집니다.',
      downloadImage: '이미지 다운로드',
      typePartition: '공유 파일',
      volumeLabel: '이동식 디스크',
      groups: {
        share: '파일 공유',
        images: '이미지'
      },
      failed: '작업 실패',
      hidOnly: 'HID 전용 모드: 가상 디스크를 사용할 수 없습니다',
      disabled: '가상 디스크가 꺼져 있습니다 (설정 - 장치)',
      force: {
        title: '미디어를 강제로 되찾을까요?',
        desc: '대상 컴퓨터가 아직 이 미디어를 사용하고 있습니다. 되찾으면 USB 장치가 다시 열거되고 아직 디스크에 기록하지 않은 내용이 사라질 수 있습니다.',
        menuHint:
          '대상 컴퓨터가 이 미디어를 사용하고 있습니다. 파일 관리자를 열어 강제로 되찾으세요.'
      },
      tips: {
        title: '업로드 방법',
        usb1: 'USB를 통해 NanoKVM을 컴퓨터에 연결하세요.',
        usb2: '이 페이지에서 공유 파일을 대용량 저장 장치 모드로 마운트합니다.',
        usb3: '컴퓨터에서 가상 디스크를 열고 이미지 파일을 가상 디스크의 루트 디렉토리로 복사하세요.',
        usb4: '복사가 끝나면 디스크를 꺼냅니다. 그러면 이미지가 목록에 나타납니다.',
        scp1: 'NanoKVM과 컴퓨터가 동일한 로컬 네트워크에 있는지 확인하세요.',
        scp2: '컴퓨터에서 터미널을 열고 SCP 명령을 사용하여 이미지 파일을 NanoKVM의 /data 디렉터리에 업로드하세요.',
        scp3: '예시: scp [이미지 파일 경로] root@[NanoKVM IP 주소]:/data'
      },
      job: {
        running: '{{name}} 처리 중'
      },
      enableDisk: {
        title: '가상 디스크를 켤까요?',
        desc: '미디어를 마운트하려면 현재 꺼져 있는 가상 디스크가 필요합니다. 지금 켤까요? 대상 컴퓨터의 키보드와 마우스가 잠시 연결 해제됩니다.',
        menuHint: '가상 디스크가 꺼져 있습니다. 설정 - 장치에서 켜세요.'
      },
      add: {
        title: '미디어 추가',
        upload: 'ISO 또는 IMG 업로드',
        fromUrl: 'URL에서 다운로드',
        url: '이미지 URL',
        urlRequired: 'URL을 입력하세요',
        onlyImages: '.iso 및 .img 파일만 마운트할 수 있습니다',
        dropHint: 'ISO 또는 IMG를 여기로 끌어오거나 클릭하여 파일을 선택하세요',
        cdromOnlyIso: 'CD-ROM에는 ISO 9660 / UDF 이미지만 넣을 수 있습니다',
        uploadFailed: '업로드 실패',
        downloadFailed: '다운로드 실패',
        uploading: '{{name}} 업로드 중',
        downloading: '{{name}} 다운로드 중',
        done: '{{name}} 준비 완료',
        checksumFailed: '이미지가 SHA-256 체크섬과 일치하지 않습니다.',
        blocked:
          '대상 컴퓨터가 디스크를 사용 중입니다. 대상 컴퓨터에서 가상 USB 드라이브를 안전하게 꺼낸 다음 설정 > 장치에서 가상 디스크 스위치를 끄세요.'
      },
      files: {
        reasonNotInitialized:
          '가상 USB 드라이브가 초기화되지 않았습니다. 파일 공유를 사용하기 전에 설정 > 장치에서 초기화하세요.',
        selectMedium: '미디어를 선택하면 안에 있는 파일을 볼 수 있습니다.',
        reasonAttached: '이 미디어는 대상 컴퓨터에 연결되어 있습니다.',
        reasonAttachedHint:
          '대상 컴퓨터에서 가상 USB 드라이브를 안전하게 꺼낸 다음 설정 > 장치에서 가상 디스크 스위치를 끄세요.',
        reasonNotBrowsable: '이 미디어에는 파일 관리자가 열 수 없는 파일 시스템이 있습니다.',
        reasonNoLoop: '이 장치는 디스크 이미지를 마운트할 수 없습니다(loop 미지원).',
        reasonISOImage:
          'ISO 9660 이미지의 내용은 여기에서 볼 수 없습니다. 대상 컴퓨터에는 그대로 마운트할 수 있습니다.',
        empty: '이 폴더는 비어 있습니다',
        noMatch: '검색과 일치하는 파일이 없습니다.',
        free: '{{size}} 남음',
        search: '이 폴더에서 검색',
        sort: {
          modified: '수정한 날짜',
          name: '이름',
          size: '크기',
          type: '종류',
          asc: '오름차순',
          desc: '내림차순'
        },
        view: {
          list: '목록 보기',
          grid: '아이콘 보기'
        },
        folder: '폴더',
        itemCount: '{{count}}개 항목',
        itemCount_one: '{{count}}개 항목',
        itemCount_other: '{{count}}개 항목',
        dropHint: '여기에 파일을 놓으면 이 폴더에 업로드됩니다',
        up: '상위 폴더로 이동',
        uploadButton: '업로드',
        newFolder: '새 폴더',
        newFolderPlaceholder: '폴더 이름',
        create: '만들기',
        upload: '미디어에 파일 업로드',
        deleteFolderTitle: '폴더 삭제',
        deleteFolderConfirm:
          '폴더 {{name}}와(과) 그 안의 모든 내용을 삭제할까요? 되돌릴 수 없습니다.',
        uploading: '업로드 중…',
        uploadFailed: '업로드 실패',
        download: '다운로드',
        delete: '삭제',
        deleteTitle: '파일 삭제',
        deleteConfirm: '미디어에서 {{name}}을(를) 삭제할까요?'
      },
      yes: '예',
      no: '아니요'
    },
    head: {
      storage: '이미지 및 파일 공유',
      desktop: '원격 데스크톱',
      login: '로그인',
      changePassword: '비밀번호 변경',
      terminal: '터미널',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        '클립보드에 복사할 수 없습니다. 브라우저에 클립보드 접근 권한을 허용한 후 다시 시도하세요.'
    },
    auth: {
      login: '로그인',
      language: '언어',
      placeholderUsername: '사용자 이름을 입력하세요.',
      placeholderPassword: '비밀번호를 입력하세요.',
      placeholderPassword2: '비밀번호를 다시 입력하세요.',
      noEmptyUsername: '사용자 이름은 비어있을 수 없습니다.',
      noEmptyPassword: '비밀번호는 비어있을 수 없습니다.',
      noAccount:
        '사용자 정보를 불러오는 데 실패했습니다. 페이지를 새로고침하거나 비밀번호를 초기화하세요.',
      invalidUser: '사용자 이름이나 비밀번호가 틀렸습니다.',
      error: '알 수 없는 오류',
      twoFactorTitle: '인증 필요',
      twoFactorDesc: '로그인하려면 인증 코드 또는 사용하지 않은 복구 코드를 입력하세요.',
      sensitiveActionDesc: '계속하려면 인증 코드 또는 사용하지 않은 복구 코드를 입력하세요.',
      placeholderAuthCode: '인증 코드',
      noEmptyAuthCode: '인증 코드가 필요합니다',
      verify: '인증',
      backToLogin: '로그인으로 돌아가기',
      verificationFailed: '인증 실패',
      twoFactorEnabledLogout: {
        message:
          '2단계 인증이 활성화되었습니다. 보안을 위해 모든 세션이 로그아웃되었습니다. 다시 로그인하세요.'
      },
      rateLimited: '실패한 시도가 너무 많습니다. {{seconds}}초 후 다시 시도하세요.',
      recoveryCodeUsed: '이 로그인에 사용된 복구 코드는 무효화되었습니다.',
      changePassword: '비밀번호 변경',
      changePasswordDesc: '보안을 위해 웹 로그인 비밀번호를 변경하세요.',
      passwordVerificationRequired: '계정 설정 페이지로 돌아가 이 비밀번호 변경을 다시 인증하세요.',
      passwordVerificationExpired:
        '비밀번호 변경 인증이 만료되었습니다. 계정 설정 페이지로 돌아가 다시 인증하세요.',
      returnToSettings: '설정으로 돌아가기',
      differentPassword: '비밀번호가 서로 일치하지 않습니다.',
      illegalUsername: '사용자 이름에 사용할 수 없는 문자가 있습니다.',
      illegalPassword: '비밀번호에 사용할 수 없는 문자가 있습니다.',
      forgetPassword: '비밀번호 분실',
      ok: '확인',
      cancel: '취소',
      savePassword: '비밀번호 저장',
      loginButtonText: '로그인',
      tips: {
        reset1: '웹 기본 계정:',
        reset2: 'SSH 기본 계정:',
        reset3: '비밀번호를 재설정하려면 다음을 참조하세요. ',
        changeReset: '웹 로그인 비밀번호를 재설정하려면 다음을 참조하세요. ',
        resetLink: '재설정'
      }
    },
    wifi: {
      errors: {
        connect:
          'Wi-Fi에 연결할 수 없습니다. 네트워크 이름, 비밀번호 및 기기의 Wi-Fi 상태를 확인한 후 다시 시도하세요.'
      },
      title: 'Wi-Fi',
      description: 'NanoKVM Wi-Fi 설정',
      submitted: {
        title: '잠시만 기다려 주세요. NanoKVM이 Wi-Fi에 연결하고 있습니다',
        tipsTitle: '다음으로 할 일',
        tipSwitch: '이제 휴대폰을 원래 Wi-Fi로 다시 연결해도 됩니다.',
        tipCheck: '잠시 후 NanoKVM 화면에서 새 IP 주소를 확인하고 브라우저로 접속하세요.',
        tipRetry: '한참 지나도 변화가 없다면 네트워크 이름과 비밀번호를 확인한 후 다시 시도하세요.'
      },
      invalidMode:
        '현재 모드는 네트워크 설정을 지원하지 않습니다. 장치로 이동하여 Wi-Fi 구성 모드를 활성화하십시오.',
      confirmBtn: '확인',
      finishBtn: '완료',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: '비밀번호',
      steps: {
        settings: '설정',
        wifi: 'Wi-Fi',
        enable: '활성화'
      },
      scan: {
        title: 'Wi-Fi 네트워크 선택',
        description: '계속하려면 주변 네트워크를 선택하세요.',
        loading: '주변 Wi-Fi 네트워크 검색 중…',
        empty: '주변에서 Wi-Fi 네트워크를 찾지 못했습니다.',
        failed: '검색하지 못했습니다. 네트워크 정보를 직접 입력하세요.',
        retryBtn: '다시 검색',
        manualBtn: '네트워크 직접 입력',
        backBtn: '주변 네트워크로 돌아가기',
        connectTitle: '{{ssid}}에 연결',
        connecting: '연결 중…',
        viewSimple: '간단 보기로 전환',
        viewDetailed: '자세히 보기로 전환'
      },
      ap: {
        authTitle: '인증 필요',
        authDescription: '계속하려면 AP 비밀번호를 입력하세요',
        authFailed: '잘못된 AP 비밀번호',
        passPlaceholder: 'AP 비밀번호',
        verifyBtn: '확인'
      },
      qr: {
        cameraBtn: '사진 촬영',
        pickBtn: '스크린샷 선택',
        scanHint:
          '휴대폰이 Wi-Fi를 공유할 때 표시하는 것과 같은 Wi-Fi QR 코드 사진을 촬영하거나, 스크린샷을 선택하세요.',
        decoding: 'QR 코드 읽는 중…',
        notFound:
          '이 사진에서 QR 코드를 찾을 수 없습니다. 코드를 중앙에 맞추고 초점을 맞춘 후 다시 시도하세요.',
        invalid: '이 QR 코드에는 사용할 수 있는 Wi-Fi 정보가 없습니다.',
        pickerUnavailable:
          '아무것도 열리지 않나요? 이 로그인 창에서 카메라와 사진을 차단할 수 있습니다. 대신 휴대폰 브라우저에서 다음 주소를 여세요:',
        confirmTitle: 'NanoKVM을 {{ssid}}에 연결할까요?',
        confirmDescription:
          '이 네트워크 정보는 QR 코드에서 가져온 것입니다. 연결하기 전에 확인하세요.',
        openNetwork: '이 네트워크에는 비밀번호가 없습니다.',
        connectBtn: '연결',
        otherBtn: '다른 네트워크 선택'
      }
    },
    notification: {
      browser: {
        title: '브라우저 추천',
        description:
          '최상의 사용자 환경을 위해 Chrome 브라우저 사용을 권장합니다. 우리는 모든 주요 브라우저 간의 호환성을 위해 노력하고 있지만 일부 고급 기능은 Chrome 외부에서 의도한 대로 작동하지 않을 수 있습니다.'
      },
      btnOk: '알았어'
    },
    screen: {
      title: '화면',
      video: '비디오 모드',
      quality: '품질',
      qualityAuto: '자동',
      qualityLossless: '무손실',
      qualityHigh: '높음',
      qualityMedium: '중간',
      qualityLow: '낮음',
      scale: '규모',
      noSignal: '영상 신호 없음',
      captureDisabled: '영상 캡처가 꺼져 있습니다. "설정 > 장치 > 영상 캡처"에서 활성화하세요.',
      inconsistentVideoMode: '재생이 일시중지되었습니다. 다른 비디오 모드가 재생 중입니다.',
      bandwidthWarning:
        '현재 비디오 설정에 비해 네트워크 대역폭이 부족할 수 있습니다. 비트레이트 또는 프레임 레이트를 낮춰 보세요.',
      webrtcConnectionFailed: 'WebRTC 연결을 설정하지 못했습니다. Direct 영상 모드로 전환하세요.',
      webrtcUnavailable: {
        title: 'WebRTC를 사용할 수 없음',
        description: '브라우저에서 WebRTC를 활성화하거나 Direct 비디오 모드로 전환하세요.'
      },
      upload: '업로드',
      advanced: '고급',
      loading: '불러오는 중'
    },
    volume: {
      title: '볼륨'
    },
    microphone: {
      title: '마이크',
      notMounted: '마이크가 장착되지 않음',
      tip1: '가상 마이크가 원격 호스트에 마운트되어 있지 않습니다. 지금 마운트하고 활성화하시겠습니까?',
      tip2: '나중에 "설정 > 장치 > 가상 오디오"에서 관리할 수도 있습니다.',
      confirm: '활성화',
      cancel: '취소',
      noDevice: '이 장치에서 마이크를 찾을 수 없습니다. 마이크를 연결한 후 다시 시도하세요.',
      permissionDenied:
        '마이크 접근 권한이 거부되었습니다. 브라우저 설정에서 마이크 접근을 허용하세요.',
      notReadable: '다른 프로그램에서 마이크를 사용 중이거나 마이크를 읽을 수 없습니다.',
      overconstrained: '이 마이크는 필요한 오디오 설정을 지원하지 않습니다.',
      securityBlocked:
        '이 페이지에서 마이크 접근이 차단되었습니다. 브라우저 보안 설정을 확인하세요.',
      aborted: '마이크 시작이 중단되었습니다. 다시 시도하세요.',
      unsupported: '이 브라우저 또는 연결 방식은 마이크 캡처를 지원하지 않습니다.',
      enableFailed: '마이크를 활성화하지 못했습니다. 다시 시도하세요.',
      disconnected: '마이크 연결이 해제되어 마이크 전송이 꺼졌습니다.'
    },
    power: {
      title: '전원',
      showConfirm: '확인',
      showConfirmTip: '전원 작업에는 추가 확인이 필요합니다',
      powerShort: '전원 (짧게 누르기)',
      powerLong: '전원 (길게 누르기)',
      powerConfirm: '전원 작업을 실행하시겠습니까?',
      okBtn: '예',
      cancelBtn: '아니요'
    },
    keyboard: {
      errors: {
        paste: '원격 장치로 텍스트를 전송할 수 없습니다. KVM 연결을 확인한 후 다시 시도하세요.'
      },
      title: '키보드',
      paste: '붙여넣기',
      tips: '표준 키보드 문자 및 기호만 지원됩니다',
      placeholder: '입력하세요',
      submit: '전송',
      virtual: '키보드',
      swapCommandCtrl: 'Cmd와 Ctrl 바꾸기',
      shortcut: {
        title: '바로가기',
        custom: '사용자 지정',
        capture: '바로가기를 캡처하려면 여기를 클릭하세요',
        clear: '클리어',
        save: '저장',
        captureTips: 'Windows 키와 같은 시스템 수준 키를 캡처하려면 전체 화면 권한이 필요합니다.',
        enterFullScreen: '전체 화면 모드를 전환합니다.'
      },
      leaderKey: {
        title: '리더 키',
        desc: '브라우저 제한을 우회하고 시스템 바로가기를 원격 호스트에 직접 보냅니다.',
        howToUse: '사용방법',
        simultaneous: {
          title: '동시 모드',
          desc1: '리더 키를 누른 상태에서 단축키를 누르세요.',
          desc2: '직관적이지만 시스템 단축키와 충돌할 수 있습니다.'
        },
        sequential: {
          title: '순차 모드',
          desc1: '리더 키를 누름 → 단축키를 순서대로 누름 → 리더 키를 다시 누름.',
          desc2: '더 많은 단계가 필요하지만 시스템 충돌을 완전히 방지합니다.'
        },
        enable: '리더 키 활성화',
        tip: '리더 키로 지정하면 이 키는 단축키 트리거로만 동작하며 기본 동작을 잃습니다.',
        placeholder: '리더 키를 누르세요',
        shiftRight: '오른쪽 Shift',
        ctrlRight: '오른쪽 Ctrl',
        metaRight: '오른쪽 Win',
        submit: '전송',
        recorder: {
          rec: 'REC',
          activate: '키 활성화',
          input: '단축키를 눌러주세요...'
        }
      }
    },
    mouse: {
      title: '마우스',
      mode: '마우스 모드',
      absolute: '추적식 마우스',
      absoluteAndroid: '추적식 마우스 (Android 호환성 향상)',
      touchScreen: '멀티터치 스크린',
      relative: '독점식 마우스',
      direction: '스크롤 휠 방향',
      scrollNatural: '자연스러운 스크롤',
      scrollConventional: '기존 방식',
      speed: '스크롤 휠 속도',
      scrollStep: '터치패드 스크롤',
      fast: '빠름',
      slow: '느림',
      requestPointer: '상대값 모드를 사용 중입니다. 커서를 찾으려면 데스크톱을 클릭하세요.',
      repairIphoneDrag: 'iPhone 드래그 복구',
      resetHid: 'HID 초기화',
      touchpadGuide: {
        title: '터치패드 가이드',
        scope: '마우스 모드가 상대 모드일 때 일반 포인터 입력에 적용됩니다.',
        swipeTitle: '스와이프하여 이동',
        swipeDesc: '활성 화면 영역 안에서 스와이프하면 원격 포인터가 이동합니다.',
        tapTitle: '탭하여 클릭',
        tapDesc: '짧게 탭하면 왼쪽 클릭 한 번을 보냅니다.',
        holdTitle: '길게 눌러 왼쪽 버튼 누르기',
        holdDesc:
          '손가락을 대고 약 1초 동안 움직이지 않으면 왼쪽 마우스 버튼을 누른 상태가 됩니다.',
        dragTitle: '길게 누른 뒤 움직여 드래그',
        dragDesc:
          '길게 누르기가 활성화된 뒤 손가락을 움직이면 왼쪽 버튼을 누른 상태로 드래그합니다.'
      },
      hidOnly: {
        errors: {
          load: 'HID-Only 모드 상태를 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
          update:
            'HID-Only 모드가 업데이트되었는지 확인할 수 없습니다. 이 설정을 다시 열어 현재 상태를 확인한 후 다시 시도하세요.'
        },
        title: 'HID 전용 모드',
        desc: '마우스와 키보드가 응답하지 않고 HID 초기화도 도움이 되지 않는다면, NanoKVM과 장치 간의 호환성 문제일 수 있습니다. 더 나은 호환성을 위해 HID 전용 모드를 활성화해 보세요.',
        tip1: 'HID 전용 모드를 활성화하면 가상 USB와 가상 네트워크가 언마운트됩니다',
        tip2: 'HID 전용 모드에서는 이미지 마운트가 비활성화됩니다',
        enable: 'HID 전용 모드 활성화',
        disable: 'HID 전용 모드 비활성화'
      }
    },
    preview: {
      title: '미리보기'
    },
    image: {
      storageReadOnly: '파일 저장소에 쓸 수 없습니다. 가상 디스크를 끄세요.',
      storageBusy:
        '저장소가 사용 중입니다. 파일 전송을 완료하거나 취소한 후 가상 디스크를 전환하세요.',
      title: '이미지',
      loading: '불러오는 중...',
      empty: '아무것도 없습니다.',
      upload: '이미지 업로드',
      dropFile: '이미지를 여기에 놓거나 클릭하여 파일을 선택하세요',
      uploadAction: '업로드',
      cancelAction: '취소',
      mount: '이미지 마운트',
      mounted: '장착됨',
      uploadError: '업로드에 실패했습니다. 다시 시도해 주세요.',
      uploadInsufficientStorage: '저장 공간이 부족합니다',
      uploadTooLarge: '이미지가 크기 제한을 초과합니다',
      cancelUploadError: '업로드를 취소하지 못했습니다.',
      mountMode: '마운트 모드',
      massStorage: '대용량 저장소',
      cdRom: 'CD ROM',
      readOnly: '읽기 전용',
      mountFailed: '이미지 마운트 실패',
      mountDesc:
        '일부 시스템에서는 이미지를 마운트하기 전에 원격 호스트에서 가상 디스크를 제거해야 합니다.',
      unmountFailed: '마운트 해제 실패',
      unmountDesc:
        '일부 시스템에서는 이미지를 마운트 해제하기 전에 원격 호스트에서 수동으로 꺼내야 합니다.',
      refresh: '이미지 목록 새로고침',
      download: '이미지 다운로드',
      attention: '주의',
      deleteConfirm: '이 이미지를 제거하시겠습니까?',
      deleteFailed: '이미지를 삭제하지 못했습니다',
      okBtn: '네',
      cancelBtn: '아니오'
    },
    script: {
      title: '스크립트',
      upload: '업로드',
      run: '실행',
      runBackground: '백그라운드에서 실행',
      runFailed: '실행 실패',
      uploadFailed: '스크립트를 업로드하지 못했습니다',
      deleteFailed: '스크립트를 삭제하지 못했습니다',
      attention: '주의',
      delDesc: '이 파일을 정말로 삭제합니까?',
      confirm: '네',
      cancel: '아니오',
      delete: '삭제',
      close: '닫기'
    },
    terminal: {
      title: '터미널',
      nanokvm: 'NanoKVM 터미널',
      confirm: '확인',
      submit: '제출',
      username: '사용자 이름',
      password: '비밀번호',
      authCode: '인증 코드',
      authorizeTitle: 'Web 터미널 승인',
      codeAccountHint: '인증 코드 또는 사용하지 않은 복구 코드를 입력하세요.',
      authFailed: '인증에 실패했습니다'
    },
    wol: {
      errors: {
        send: '절전 모드 해제 명령을 보낼 수 없습니다. MAC 주소와 네트워크 연결을 확인한 후 다시 시도하세요.',
        loadFailed:
          'Wake-on-LAN 장치 목록을 불러올 수 없습니다. NanoKVM이 온라인인지 확인한 후 다시 시도하세요.',
        renameFailed: '이 장치의 이름을 변경할 수 없습니다. MAC 주소를 확인한 후 다시 시도하세요.',
        deleteFailed: '이 장치를 삭제할 수 없습니다. 잠시 후 다시 시도하세요.'
      },
      title: 'Wake-on-LAN',
      sending: '패킷 전송 중...',
      sent: '패킷 전송 완료',
      input: 'MAC주소를 입력하세요.',
      ok: '확인'
    },
    download: {
      sha256: 'SHA-256 (선택 사항)',
      sha256Placeholder: '64자 SHA-256 체크섬을 입력하세요',
      invalidSHA256: 'SHA-256은 64자의 16진수 문자열이어야 합니다',
      title: '이미지 다운로드',
      input: '원격 이미지 URL을 입력하세요.',
      ok: '확인',
      disabled: '/data 파티션이 읽기 전용(RO) 상태이므로 이미지를 다운로드할 수 없습니다.',
      downloadingFile: '다운로드 중: {{file}}',
      downloadingProgress: '다운로드 중 ({{percentage}}): {{file}}',
      failed: '실패',
      insufficientStorage: '저장 공간이 부족합니다',
      remaining: '남은 시간 {{time}}',
      cancel: '취소',
      cancelFailed: '다운로드를 취소하지 못했습니다',
      pause: '일시 중지',
      resume: '계속',
      pauseFailed: '다운로드를 일시 중지하지 못했습니다',
      resumeFailed: '다운로드를 계속하지 못했습니다'
    },
    settings: {
      back: '뒤로',
      close: '닫기',
      ai: {
        errors: {
          loadServices:
            'AI 서비스 상태를 불러올 수 없습니다. NanoKVM이 온라인인지 확인한 후 다시 시도하세요.',
          loadMcpConfig: 'MCP 서비스 구성을 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
          updateMcpService:
            'MCP 서비스 상태를 업데이트할 수 없습니다. 현재 상태를 확인한 후 다시 시도하세요.',
          accessMcpApiKey:
            'MCP API Key를 가져올 수 없습니다. 다시 시도하고 요청되면 인증을 완료하세요.',
          regenerateMcpApiKey: 'MCP API Key를 다시 생성할 수 없습니다. 잠시 후 다시 시도하세요.',
          regeneratedMcpApiKeyUnavailable:
            'API Key가 다시 생성되었지만 표시할 수 없습니다. 다시 표시해 보세요. 이전 Key는 더 이상 유효하지 않습니다.',
          loadMemoryFabricState:
            'Memory Fabric 서비스 상태를 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
          updateMemoryFabricService:
            'Memory Fabric 서비스 상태를 업데이트할 수 없습니다. 현재 상태를 확인한 후 다시 시도하세요.',
          loadScreenTimelapseState:
            '화면 타임랩스 서비스 상태를 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
          updateScreenTimelapseService:
            '화면 타임랩스 서비스 상태를 업데이트할 수 없습니다. 현재 상태를 확인한 후 다시 시도하세요.',
          openScreenTimelapse: '화면 타임랩스 재생을 열 수 없습니다. 잠시 후 다시 시도하세요.',
          loadScreenTimelapseRecordings:
            '화면 타임랩스 녹화본을 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
          playScreenTimelapse:
            '이 타임랩스 녹화본은 지금 재생할 수 없습니다. 날짜를 다시 선택하거나 나중에 다시 시도하세요.',
          screenTimelapseFullscreen:
            '전체 화면 모드를 변경할 수 없습니다. 브라우저 권한을 확인한 후 다시 시도하세요.'
        },
        title: 'AI',
        mcpService: 'MCP 서비스',
        mcpServiceDesc: '제어, 미디어 및 활성화된 Memory Fabric 도구를 위한 인증된 통합 게이트웨이',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Agent 컨텍스트 메모리와 OCR을 로컬에서 처리',
        memoryFabricRemoteRequiresMcp:
          '메모리 및 OCR 도구를 원격으로 사용하려면 위의 MCP 서비스를 활성화하세요.',
        memoryFabricRemoteAvailable:
          '메모리 및 OCR 도구는 동일한 API 키로 위의 /api/mcp 엔드포인트에서 사용할 수 있습니다.',
        memoryFabricConfig: {
          title: '구성',
          description: 'Memory Fabric 기본 모델 설정만 관리합니다.',
          loadFailed: 'Memory Fabric 구성을 불러오지 못했습니다.',
          saveFailed: 'Memory Fabric 구성을 저장하지 못했습니다.',
          saved: '구성을 저장했습니다.',
          savedRestarted: '구성을 저장하고 Memory Fabric을 다시 시작했습니다.',
          noChanges: '변경된 구성이 없습니다.',
          reload: '다시 불러오기',
          save: '저장',
          unsaved: '저장하지 않은 변경 사항',
          conflictTitle: '구성이 변경됨',
          conflictDescription:
            '다른 곳에서 구성이 변경되었습니다. 다시 저장하기 전에 최신 버전을 불러오세요.',
          conflictReload: '최신 구성 불러오기',
          discardTitle: '저장하지 않은 변경 사항을 버릴까요?',
          discardDescription: '다시 불러오면 현재 양식이 최근에 저장한 구성으로 대체됩니다.',
          discardConfirm: '버리고 다시 불러오기',
          sections: {
            models: '모델'
          },
          openAICompatibleHint:
            'OpenAI 호환 모델 공급자를 지원합니다. 공급자가 제공한 모델 이름, 기본 URL 및 API 키를 입력하세요.',
          fields: {
            model: '모델',
            baseUrl: '기본 URL',
            maxTokens: '최대 토큰 수',
            timeoutSeconds: '시간 제한(초)'
          },
          apiKey: {
            title: 'API 키',
            configured: '설정됨',
            notConfigured: '설정되지 않음',
            keep: '현재 키 유지',
            replace: '키 교체',
            clear: '키 지우기',
            replacement: '새 API 키',
            replacementPlaceholder: '새 API 키 입력',
            clearHint: '저장된 API 키가 제거되며, 환경 변수에서 키가 계속 제공될 수 있습니다.',
            required: '새 API 키를 입력하세요.'
          },
          validation: {
            required: '필수 항목입니다.',
            invalidUrl: '올바른 HTTP 또는 HTTPS URL을 입력하세요.'
          }
        },
        screenTimelapseService: '화면 타임랩스',
        screenTimelapseServiceDesc:
          'Memory Fabric 화면 기록을 재생하고 캡처된 화면 내용을 검색합니다',
        screenTimelapseSearchRequiresMemoryFabric:
          '녹화된 콘텐츠를 검색하려면 Memory Fabric도 활성화하세요.',
        screenTimelapseOpen: '열기',
        screenTimelapseSearchPlaceholder: '동영상 내용 검색...',
        screenTimelapseNoVideos: '타임랩스 동영상을 찾을 수 없습니다',
        screenTimelapseNoResults: '일치하는 프레임을 찾을 수 없습니다',
        screenTimelapseMultipleResults: '검색 결과',
        screenTimelapseUnavailable: '사용할 수 없음',
        screenTimelapsePlayerUnsupported: '이 브라우저는 타임랩스 동영상을 재생할 수 없습니다',
        screenTimelapseSearchFailed: '검색 실패',
        screenTimelapseLoading: '타임랩스 불러오는 중...',
        screenTimelapseSearchAction: '타임랩스 검색',
        screenTimelapsePlay: '타임랩스 재생',
        screenTimelapsePause: '타임랩스 일시 중지',
        screenTimelapseEnterFullscreen: '전체 화면 시작',
        screenTimelapseExitFullscreen: '전체 화면 종료',
        screenTimelapseFullscreenUnsupported: '이 브라우저는 전체 화면을 지원하지 않습니다',
        screenTimelapsePreviousDay: '이전 날',
        screenTimelapseNextDay: '다음 날',
        screenTimelapseDeleteDay: '{{date}} 녹화 삭제',
        screenTimelapseDelete: '삭제',
        screenTimelapseDeleteConfirmTitle: '타임랩스 녹화를 삭제할까요?',
        screenTimelapseDeleteConfirmDesc:
          '{{date}} 녹화는 {{size}}의 공간을 차지합니다. 삭제하면 복구할 수 없습니다.',
        screenTimelapseDeleteSuccess: '{{date}} 녹화를 삭제했습니다',
        screenTimelapseDeleteFailed: '{{date}} 녹화를 삭제하지 못했습니다',
        endpoint: '엔드포인트',
        apiKey: 'API 키',
        regenerate: '다시 생성',
        copyApiKey: 'API 키 복사',
        copyEndpoint: '엔드포인트 복사',
        showApiKey: 'API 키 표시',
        regenerateConfirmTitle: 'API 키 다시 생성',
        regenerateConfirmDesc:
          '기존 MCP 클라이언트는 새 키로 업데이트될 때까지 작동하지 않습니다. 계속하시겠습니까?',
        okBtn: '예',
        cancelBtn: '아니요'
      },
      title: '설정',
      about: {
        title: 'NanoKVM 정보',
        information: '정보',
        ip: 'IP',
        mdns: 'mDNS',
        application: '애플리케이션 버전',
        applicationTip: 'NanoKVM 웹 애플리케이션 버전',
        arch: '아키텍처',
        device_number: '기기번호',
        image: '이미지 버전',
        imageTip: 'NanoKVM 시스템 이미지 버전',
        deviceKey: '장치 키',
        community: '커뮤니티',
        hostname: '호스트 이름',
        hostnameUpdated: '호스트 이름이 업데이트되었습니다. 적용하려면 재부팅하세요.',
        ipType: {
          Wired: '유선',
          Wireless: '무선',
          Other: '기타'
        }
      },
      appearance: {
        title: '디자인',
        display: '표시',
        language: '언어',
        languageDesc: '인터페이스 언어를 선택하세요',
        languageChangeFailed: '언어 변경에 실패했습니다',
        webTitle: '웹 제목',
        webTitleDesc: '웹 페이지 제목 사용자 지정',
        menuBar: {
          title: '메뉴 바',
          mode: '표시 모드',
          modeDesc: '메뉴 바를 화면에 표시합니다',
          modeOff: '꺼짐',
          modeAuto: '자동 숨기기',
          modeAlways: '항상 보이기',
          icons: '하위 메뉴 아이콘',
          iconsDesc: '메뉴 바에 하위 메뉴 아이콘을 표시합니다'
        }
      },
      screen: {
        title: '화면',
        videoMode: {
          title: '비디오 모드',
          description: '인코딩 및 전송 방식을 선택하세요.'
        },
        edid: {
          description: '표시 매개변수 선택',
          default: '기본값',
          custom: '사용자 지정',
          document: '문서',
          monitorName: '모니터 이름',
          monitorNameDescription:
            '인쇄 가능한 ASCII 문자 최대 13자. 비워 두면 EDID의 원래 이름을 사용합니다',
          monitorNamePlaceholder: '원래 EDID 이름 사용',
          monitorNameApply: '적용',
          monitorNameInvalid: '인쇄 가능한 ASCII 문자를 최대 13자까지 입력하세요',
          monitorNameUnsupported: '현재 EDID는 모니터 이름 재정의를 지원하지 않습니다',
          monitorNameLoadFailed: '모니터 이름을 불러오지 못했습니다',
          monitorNameUpdateFailed: '모니터 이름을 업데이트하지 못했습니다',
          powerConfirmTitle: 'EDID 업데이트 전 확인',
          powerConfirmDescription:
            '먼저 PD 전원을 연결하세요. EDID 업데이트 중 디스플레이가 잠시 연결 해제될 수 있습니다. 이후 장치가 디스플레이를 올바르게 감지하지 못하면 수동으로 다시 시작하세요.',
          powerConfirmOk: '계속',
          powerConfirmOkCountdown: '계속 ({{seconds}})',
          powerConfirmCancel: '취소',
          errors: {
            badRequest: '잘못된 EDID 요청입니다',
            invalidFileName: '파일 이름이 올바른 .bin EDID 파일을 선택하세요',
            fileTooLarge: 'EDID 파일이 너무 큽니다',
            invalidLength: 'EDID 크기는 128, 256, 384 또는 512바이트여야 합니다',
            invalidHeader: 'EDID 헤더가 잘못되었습니다',
            unsupportedVersion: '이 EDID 버전은 지원되지 않습니다',
            extensionCountMismatch: 'EDID 확장 블록 수가 파일 길이와 일치하지 않습니다',
            invalidChecksum: 'EDID 체크섬이 잘못되었습니다',
            invalidCta: 'CTA 확장 구조가 잘못되었습니다',
            currentCannotOverwrite:
              '현재 사용 중인 EDID는 덮어쓸 수 없습니다. 먼저 다른 EDID로 전환하세요',
            saveFailed: 'EDID를 저장하지 못했습니다',
            switchFailed: 'EDID를 전환하지 못했습니다',
            uploadFailed: 'EDID를 업로드하지 못했습니다',
            deleteFailed: 'EDID를 삭제하지 못했습니다',
            networkFailed: '네트워크 오류입니다. 다시 시도하세요'
          }
        },
        rateControlMode: {
          title: '속도 제어 모드',
          description: '비트레이트 조절 방법을 선택하세요.',
          cbr: 'CBR(Constant Bitrate): 연결 안정성을 우선시합니다. 끊김 현상을 최소화하지만 움직임이 많은 장면에서는 화질이 저하될 수 있습니다.',
          vbr: 'VBR(가변 비트 전송률): 이미지 선명도를 우선시합니다. 움직임이 많은 장면에서는 대역폭 스파이크가 발생하여 대기 시간이나 끊김 현상이 발생할 수 있습니다.'
        },
        bitrate: {
          title: '비트레이트',
          description: '대역폭 사용량과 이미지 선명도의 균형',
          lower:
            '값이 낮을수록 열악한 네트워크에서 대기 시간이 줄어들지만 이미지 품질이 저하되어 잠재적으로 흐릿하거나 아티팩트가 발생할 수 있습니다.',
          higher:
            '값이 높을수록 이미지 선명도가 향상되지만 더 많은 대역폭을 소비합니다. 연결이 느린 경우 대기 시간이 늘어날 수 있습니다.'
        },
        gop: {
          description: '전체 키프레임 사이의 간격을 설정합니다.',
          lower:
            '값이 낮을수록 전체 프레임을 더 자주 보냅니다. 이렇게 하면 시각적 아티팩트 복구 속도가 빨라지지만 더 많은 대역폭을 소비하고 이미지 세부 정보가 줄어듭니다.',
          higher:
            '값이 높을수록 증분 업데이트에 최적화됩니다. 이렇게 하면 이미지 세부 정보가 향상되고 대역폭이 절약되지만 시각적 아티팩트 복구에는 시간이 더 오래 걸립니다.'
        },
        quality: {
          title: '품질',
          description: '디스플레이 이미지 품질을 설정합니다.'
        },
        fps: {
          auto: '자동',
          description: '디스플레이 프레임 속도를 설정합니다.',
          tip1: '값을 0으로 설정하면 자동 모드가 활성화되어 디스플레이가 원격 호스트의 프레임 속도와 동기화됩니다.',
          tip2: '프레임 속도를 낮추면 대역폭 사용량이 크게 줄어듭니다. 네트워크가 불안정하거나 높은 프레임 속도가 필요하지 않은 경우 수동으로 줄입니다.'
        },
        scale: {
          title: '배율',
          description: '디스플레이 배율 비율을 설정합니다.'
        }
      },
      device: {
        title: '장치',
        virtualAudioConfirm: {
          title: '가상 오디오 전환 전 확인',
          description:
            '먼저 PD 전원을 연결하세요. 가상 오디오를 전환하는 동안 디스플레이 연결이 잠시 끊길 수 있습니다. 전환 후 장치가 디스플레이를 올바르게 감지하지 못하면 수동으로 다시 시작하세요.',
          ok: '전환 계속',
          okCountdown: '전환 계속 ({{seconds}})',
          cancel: '취소'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi 설정',
          apMode: 'AP 모드가 활성화되었습니다. Wi-Fi에 수동으로 연결하세요.',
          others: '기타...',
          connect: 'Wi-Fi 연결',
          connectDesc1: '네트워크 ssid와 비밀번호를 입력해주세요',
          connectDesc2: '이 네트워크에 연결하려면 비밀번호를 입력하세요.',
          disconnect: '네트워크 연결을 끊으시겠습니까?',
          ssid: '이름',
          password: '비밀번호',
          joinBtn: '가입',
          confirmBtn: '알았어',
          cancelBtn: '취소',
          preset: {
            title: 'Wi-Fi 사전 저장',
            description: '대상 Wi-Fi를 하나 미리 저장하면 부팅 시 자동으로 연결됩니다',
            empty: '없음',
            addBtn: '사전 저장 추가',
            removeBtn: '사전 저장 삭제',
            saveTitle: 'Wi-Fi 사전 저장',
            saveDesc:
              '대상 네트워크를 지금 저장하면 부팅 시 해당 네트워크를 찾아 자동으로 연결합니다',
            saveBtn: '저장',
            passwordPlaceholder: '비밀번호(개방형 네트워크는 비워 두기)',
            saveFailed: 'Wi-Fi 사전 저장에 실패했습니다',
            removeFailed: 'Wi-Fi 사전 저장 삭제에 실패했습니다',
            removeConfirm: '사전 저장된 Wi-Fi를 삭제하시겠습니까?',
            warning:
              '저장하면 부팅 시 이 Wi-Fi가 감지될 때마다 기기가 이에 연결을 시도하여 다른 Wi-Fi 연결을 방해할 수 있습니다. 사전 설정은 수정하거나 삭제할 때까지 계속 유지됩니다.'
          }
        },
        capture: {
          title: '영상 캡처',
          description: '영상 캡처 활성화'
        },
        ssh: {
          description: 'SSH 원격 접속 활성화',
          tip: 'SSH를 활성화한 후 터미널에서 root 사용자의 강력한 비밀번호를 설정하세요.'
        },
        swap: {
          errors: {
            load: '스왑 메모리 상태를 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
            update:
              '스왑 메모리 설정을 업데이트할 수 없습니다. 현재 상태를 확인한 후 다시 시도하세요.'
          },
          title: '스왑',
          description: '압축 스왑 메모리 활성화'
        },
        advanced: '고급 설정',
        staticIp: {
          title: '고정IP',
          config: '구성',
          tip: '연결 손실을 방지하려면 장치에 물리적으로 접근할 수 있는 상태에서 구성하는 것이 좋습니다.',
          document: '문서를 확인해보세요.',
          enable: '고정 IP 활성화',
          placeholder: '형식: IP/CIDR [게이트웨이]\n예: 192.168.1.10/22',
          notEmpty: 'IP 주소는 비워둘 수 없습니다.',
          notValid: 'IP 주소 형식이 잘못되었습니다.',
          failed: '구성에 실패했습니다.',
          success: '구성이 완료되었습니다. IP 주소를 사용할 수 있는지 확인하세요.',
          okBtn: '알았어',
          cancelBtn: '취소'
        },
        usbNetwork: {
          title: 'USB 네트워크',
          mode: '모드',
          server: '서버',
          serverDesc: '내장 DHCP 서버를 통해 원격 호스트에 IP 주소를 할당합니다.',
          client: '클라이언트',
          clientDesc: 'DHCP를 통해 원격 호스트에서 IP 주소를 받습니다.',
          forwarding: 'IP 포워딩',
          forwardingTip:
            '원격 호스트가 NanoKVM을 통해 인터넷에 접속하도록 하려면 IP 포워딩을 활성화하세요.',
          manualIp: 'IP 주소 수동 설정',
          ipv4Placeholder: 'IPv4/CIDR, 예: 10.10.10.1/24',
          invalidIpv4: '올바른 IPv4/CIDR 주소를 입력하세요.',
          loadFailed: 'USB 네트워크 설정을 불러올 수 없습니다.',
          failed: '설정에 실패했습니다.',
          success: '설정이 완료되었습니다.',
          applyBtn: '적용',
          cancelBtn: '취소'
        },
        usbId: {
          title: 'USB 장치 식별 정보',
          config: '설정',
          tip: '사용자 지정 USB ID는 호스트가 이 장치를 식별하는 방식을 변경합니다. 적용하면 USB 가젯이 재시작되며 키보드, 마우스 또는 네트워크가 잠시 끊길 수 있습니다.',
          enable: '사용자 지정 USB 장치 식별 정보 사용',
          vid: '벤더 ID (VID)',
          pid: '제품 ID (PID)',
          bcdDevice: '장치 버전 (bcdDevice)',
          serialnumber: '일련번호',
          manufacturer: '제조사',
          product: '제품명',
          vidPlaceholder: '예: 0x359F',
          pidPlaceholder: '예: 0x3305',
          bcdDevicePlaceholder: '예: 0x0101 (선택)',
          serialnumberPlaceholder: '예: 0123456789ABCDEF (선택)',
          manufacturerPlaceholder: '예: sipeed (선택)',
          productPlaceholder: '예: NanoKVM-Go (선택)',
          notEmpty: 'VID와 PID는 비워 둘 수 없습니다.',
          notValid: 'VID/PID 형식이 잘못되었습니다. 0x359F와 같은 16진수를 사용하세요.',
          bcdNotValid:
            'bcdDevice 형식이 잘못되었습니다. 0x0101 같은 16진수를 사용하거나 비워 두세요.',
          failed: '구성에 실패했습니다.',
          success: '구성이 완료되었습니다. USB 가젯이 재시작되었습니다.',
          okBtn: '확인',
          cancelBtn: '취소'
        },
        diskInquiry: {
          title: 'USB 드라이브 이름',
          config: '설정',
          tip: '원격 컴퓨터에 표시되는 가상 USB 플래시 드라이브 이름입니다. 저장하면 USB가 재시작되며 키보드, 마우스 또는 네트워크가 잠시 끊길 수 있습니다.',
          enable: '사용자 지정 USB 드라이브 이름 사용',
          placeholder: '예: sipeed  Flash Drive     1.00',
          notEmpty: '이름은 비워 둘 수 없습니다.',
          notValid: '이름이 잘못되었습니다. 줄 바꿈 없이 인쇄 가능한 ASCII 문자 최대 28자입니다.',
          failed: '구성에 실패했습니다.',
          success: '구성이 완료되었습니다. USB 가젯이 재시작되었습니다.',
          okBtn: '확인',
          cancelBtn: '취소'
        },
        diskInquiryCdrom: {
          title: '가상 CD 드라이브 이름',
          config: '설정',
          tip: '원격 컴퓨터에 표시되는 가상 CD/DVD 드라이브 이름입니다. 저장하면 USB가 재시작되며 키보드, 마우스 또는 네트워크가 잠시 끊길 수 있습니다.',
          enable: '사용자 지정 CD 드라이브 이름 사용',
          placeholder: '예: sipeed  Optical Drive   1.00',
          notEmpty: '이름은 비워 둘 수 없습니다.',
          notValid: '이름이 잘못되었습니다. 줄 바꿈 없이 인쇄 가능한 ASCII 문자 최대 28자입니다.',
          failed: '구성에 실패했습니다.',
          success: '구성이 완료되었습니다. USB 가젯이 재시작되었습니다.',
          okBtn: '확인',
          cancelBtn: '취소'
        },
        uac2FunctionName: {
          title: 'USB 오디오 이름',
          config: '설정',
          tip: '원격 컴퓨터에 표시되는 가상 USB 마이크/오디오 장치 이름입니다. 저장하면 USB가 재시작되며 키보드, 마우스 또는 네트워크가 잠시 끊길 수 있습니다.',
          enable: '사용자 지정 USB 오디오 이름 사용',
          placeholder: '예: NanoKVM-Go Audio',
          notEmpty: '이름은 비워 둘 수 없습니다.',
          notValid: '이름이 잘못되었습니다. 줄 바꿈 없이 인쇄 가능한 문자 최대 64자입니다.',
          failed: '구성에 실패했습니다.',
          success: '구성이 완료되었습니다. USB 가젯이 재시작되었습니다.',
          okBtn: '확인',
          cancelBtn: '취소'
        },
        lowPower: {
          title: '저전력 모드',
          description: '유휴 상태에서 전력 사용량을 줄입니다.',
          tips: '팁',
          tip1: '활성화하면 전력 소비와 발열을 줄이기 위해 유휴 시간 동안 영상 캡처가 일시 중지됩니다.',
          tip2: '영상 캡처를 재개하려면 다시 초기화해야 할 수 있으며, 이 과정에서 영상이 일시적으로 표시되지 않을 수 있습니다.',
          okBtn: '알았어',
          cancelBtn: '취소'
        },
        mouseJiggler: {
          title: '마우스 흔들기',
          description: '원격 호스트가 절전 모드로 진입하는 것을 방지',
          disable: '비활성화',
          absolute: '절대값 모드',
          relative: '상대값 모드'
        },
        fingerRobot: {
          title: '손가락 로봇',
          description: '손가락 로봇 제어 활성화'
        },
        mdns: {
          description: 'mDNS 검색 서비스 활성화'
        },
        hidOnly: 'HID 전용 모드',
        imageMounted: '이미지 마운트됨',
        disk: '가상 디스크',
        diskDesc: '원격 호스트에서 가상 U-디스크를 마운트합니다.',
        sdCard: {
          warning: '가상 디스크 초기화',
          tip1: '처음 활성화하면 가상 디스크가 초기화되고 장치에 있는 기존 이미지 파일과 가상 디스크 데이터가 삭제됩니다. 이 작업은 되돌릴 수 없습니다. 중요한 파일을 먼저 백업하십시오.',
          tip2: '초기화를 계속하시겠습니까?',
          capacity: '용량',
          disable: '비활성화',
          disableWarning: '가상 디스크를 비활성화하시겠습니까?',
          disableTip:
            '비활성화하면 가상 디스크의 모든 파일이 삭제되고 사용 중인 장치 저장 공간이 해제됩니다. 다음에 활성화할 때 용량을 다시 선택하고 초기화해야 합니다. 이 작업은 되돌릴 수 없습니다.'
        },
        network: '가상 네트워크',
        networkDesc: '원격 호스트에서 가상 네트워크 카드를 마운트합니다.',
        mic: '가상 오디오',
        micDesc: '원격 호스트에 제공할 USB 가상 오디오 모드를 선택합니다.',
        virtualAudioMode: {
          off: '끔',
          mic: '마이크만',
          duplex: '마이크 + 스피커'
        },
        speaker: '가상 스피커',
        speakerDesc:
          '활성화하면 기본 DP 오디오 출력을 대체하여 호환성이 더 높은 원격 통화를 지원합니다. 비활성화하면 미디어 오디오만 전달할 수 있고 통화 오디오는 전달할 수 없습니다.',
        micTip:
          'USB 끝점 제한으로 인해 가상 마이크를 활성화하면 USB 보조 화면이 비활성화됩니다.\n그리고 최적의 호환성을 보장하려면 BIOS 작업 중에 이 기능을 비활성화하는 것이 좋습니다.',
        reboot: '재부팅',
        rebootDesc: 'NanoKVM을 재부팅하시겠습니까?',
        rebootToFlashing: '플래싱 모드로 재부팅',
        rebootToFlashingButton: '재부팅',
        rebootToFlashingTitle: '플래싱 모드로 재부팅하시겠습니까?',
        rebootToFlashingDesc:
          'NanoKVM이 재부팅되어 USB 플래싱 모드로 진입합니다. 플래싱에 사용할 USB 인터페이스가 연결되어 있고 플래싱 도구가 준비되어 있는지 확인하세요.',
        datetime: {
          timezone: '시간대',
          datetime: '날짜시간',
          dateAndTime: '날짜 및 시간',
          synchronize: '시간 동기화',
          lastSynchronization: '마지막 동기화: ',
          notSynchronized: '시스템 시간이 아직 동기화되지 않았습니다.',
          syncNow: '지금 동기화',
          manualSet: '수동 설정',
          manualSetTitle: '시간 수동 설정',
          manualSetDescription: '시간을 변경한 후 다시 로그인해야 합니다.',
          useBrowserTime: '브라우저 시간 사용',
          setTime: '시간 설정',
          invalidTime: '올바른 날짜와 시간을 입력하세요.',
          setFailed: '시스템 시간을 설정할 수 없습니다. 다시 시도하세요.',
          setSucceeded: '시스템 시간이 업데이트되었습니다. 다시 로그인하세요.'
        },
        dialBackground: {
          errors: {
            load: '다이얼 배경 이미지를 불러올 수 없습니다. 잠시 후 다시 시도하세요.',
            upload:
              '배경 이미지를 업로드할 수 없습니다. 유효한 BMP 이미지인지 확인한 후 다시 시도하세요.',
            switch: '다이얼 배경을 전환할 수 없습니다. 이미지 목록을 새로고침한 후 다시 시도하세요.'
          },
          title: '다이얼 배경',
          description: '/kvmcomm/ui/bg/에서 BMP 이미지를 다이얼 배경으로 선택합니다',
          upload: '업로드',
          loading: '불러오는 중...',
          empty: 'BMP 이미지를 찾을 수 없습니다',
          current: '현재',
          onlyBmp: 'BMP 이미지만 지원됩니다',
          uploaded: '업로드 완료',
          switched: '배경이 변경되었습니다'
        },
        okBtn: '네',
        cancelBtn: '아니오'
      },
      tailscale: {
        title: 'Tailscale',
        restart: '정말로 Tailscale을 다시 시작하시겠습니까?',
        stop: '정말로 Tailscale을 중지하시겠습니까?',
        stopDesc: 'Tailscale에서 로그아웃하고 자동 시작을 비활성화합니다.',
        loading: '불러오는 중...',
        notInstall: 'Tailscale이 없습니다. 설치해주세요.',
        install: '설치',
        installing: '설치중',
        failed: '설치 실패',
        retry: '새로고침하고 다시 시도하거나, 수동으로 설치하세요',
        download: '다운로드 중 :',
        package: '패키지 설치',
        unzip: '압축 해제',
        upTailscale: 'tailscale을 NanoKVM 의 다음 경로에 업로드 했습니다. : /usr/bin/',
        upTailscaled: 'tailscaled을 NanoKVM 의 다음 경로에 업로드 했습니다. :  /usr/sbin/',
        refresh: '현재 페이지 새로고침',
        notRunning: 'Tailscale이 실행되고 있지 않습니다. 계속하려면 시작해 주세요.',
        run: '시작',
        notLogin: '이 기기는 현재 연동 되지 않았습니다. 로그인해서 계정에 이 장치를 연동하세요.',
        urlPeriod: '이 주소는 10분간 유효합니다.',
        login: '로그인',
        loginSuccess: '로그인 성공',
        enable: 'Tailscale 활성화',
        exitNode: '출구 노드',
        exitNodeDesc: '이 장치를 tailnet의 인터넷 출구로 알립니다.',
        subnetRoute: '서브넷 경로',
        subnetRouteDesc: '직접 연결된 LAN 서브넷 {{subnet}}을 tailnet에 알립니다.',
        subnetUnavailable: '사용 가능한 직접 연결 LAN 서브넷이 없습니다.',
        routingUnavailable: '이 장치에서는 Tailscale 라우팅 설정을 사용할 수 없습니다.',
        pendingApproval: '활성화한 후 Tailscale 관리 콘솔에서 알린 경로를 승인하세요.',
        routingUpdateFailed: 'Tailscale 라우팅 설정을 업데이트하지 못했습니다.',
        deviceName: '장치 이름',
        deviceIP: '장치 IP',
        account: '계정',
        logout: '로그아웃',
        logoutDesc: '정말로 로그아웃 하시겠습니까?',
        uninstall: 'Tailscale 제거',
        uninstallDesc: '정말로 Tailscale을 제거할까요?',
        okBtn: '네',
        cancelBtn: '아니오'
      },
      update: {
        title: '업데이트 확인',
        queryFailed: '버전 확인 실패',
        updateFailed: '업데이트 실패, 재시도하세요.',
        isLatest: '이미 최신 버전입니다.',
        available: '업데이트가 가능합니다. 정말로 업데이트 할까요?',
        updating: '업데이트 시작. 잠시 기다려주세요...',
        confirm: '확인',
        upgrade_tip_nano: 'NanoKVM의 새 버전을 사용할 수 있습니다. 지금 업데이트하시겠습니까?',
        upgrade_tip_firmware: '새로운 펌웨어 버전이 출시되었습니다. 지금 업데이트하시겠습니까?',
        cancel: '취소',
        preview: '미리보기 업데이트',
        previewDesc: '새로운 기능과 개선 사항에 미리 접근하세요',
        previewTip: '미리보기 버전에는 버그나 완성되지 않은 기능이 포함될 수 있으니 주의하세요!',
        download: '다운로드',
        install: '설치',
        changelog: '변경 내역',
        restart: '재시작'
      },
      account: {
        title: '계정',
        webAccount: 'Web 계정',
        password: '비밀번호',
        twoFactor: '2단계 인증',
        enableTwoFactor: '활성화',
        enableTwoFactorDesc: '2단계 인증 설정을 시작하려면 현재 비밀번호를 입력하세요.',
        disableTwoFactor: '비활성화',
        recoveryCodes: '복구 코드',
        enableRecoveryCodes: '활성화',
        updateRecoveryCodes: '업데이트',
        recoveryCodesTooltipDisabled:
          '복구 코드는 2단계 인증을 위한 백업 인증 정보입니다. 2단계 인증을 활성화한 후 생성할 수 있습니다.',
        recoveryCodesTooltipEnabled:
          '복구 코드는 2단계 인증이 활성화된 동안에만 유효합니다. 2단계 인증을 비활성화하면 모든 복구 코드가 무효화되며, 활성화 상태에서는 업데이트만 가능합니다.',
        recoveryCodesRegenerateDesc:
          '현재 비밀번호와 인증 코드 또는 사용하지 않은 복구 코드 중 하나를 입력하세요. 복구 코드를 업데이트하면 기존의 모든 복구 코드가 무효화됩니다.',
        recoveryCodesVerifyFailed: '비밀번호와 인증 코드 또는 복구 코드를 확인할 수 없습니다.',
        recoveryCodesSavedTitle: '지금 이 복구 코드를 저장하세요',
        recoveryCodesSavedDesc:
          '복구 코드는 한 번만 볼 수 있습니다. 이 페이지를 떠나기 전에 안전한 위치에 저장하세요.',
        enableRecoveryCodesDesc:
          '복구 코드를 생성하려면 현재 비밀번호와 인증 코드를 입력하세요. 복구 코드는 한 번만 볼 수 있으므로 이 페이지를 떠나기 전에 저장하세요.',
        copyRecoveryCodes: '복사',
        downloadRecoveryCodes: '다운로드',
        recoveryCodesCopyFailed: '복구 코드를 복사할 수 없습니다.',
        done: '완료',
        currentPassword: '현재 비밀번호',
        authCode: '인증 코드',
        authCodeOrRecoveryCode: '인증 코드 또는 복구 코드',
        authRateLimited: '실패한 시도가 너무 많습니다. {{seconds}}초 후 다시 시도하세요.',
        verifyPasswordChange: '비밀번호 변경 인증',
        verifyPasswordChangeDesc:
          '비밀번호를 변경하기 전에 현재 비밀번호와 인증 코드 또는 사용하지 않은 복구 코드 중 하나를 입력하세요.',
        verifyPasswordChangeFailed: '비밀번호와 인증 코드 또는 복구 코드를 확인할 수 없습니다.',
        verifyAndEnable: '인증 후 활성화',
        scanTwoFactorQRCode: '인증 앱으로 이 QR 코드를 스캔한 다음 코드를 입력하세요.',
        showSetupKey: '설정 키 표시',
        hideSetupKey: '설정 키 숨기기',
        setupKey: '설정 키',
        twoFactorSetupFailed: '2단계 인증 설정을 시작할 수 없습니다.',
        twoFactorVerifyFailed: '인증 코드를 확인할 수 없습니다.',
        twoFactorDisableFailed: '2단계 인증을 비활성화할 수 없습니다.',
        disableTwoFactorDesc:
          '현재 비밀번호와 인증 코드 또는 사용하지 않은 복구 코드 중 하나를 입력하세요. 2단계 인증을 비활성화하면 모든 복구 코드가 무효화됩니다.',
        updateBtn: '업데이트',
        logoutBtn: '로그아웃',
        logoutDesc: '정말로 로그아웃 하시겠습니까?',
        okBtn: '네',
        cancelBtn: '아니오',
        twoFactorProtection: {
          title: '고급 2FA 보호',
          description:
            '보호 정책이 적용될 때 두 번째 인증 요소가 필요한 민감한 작업을 선택합니다. "직접 웹 로그인" 스위치는 반대로 작동하여 인증 코드 또는 사용하지 않은 복구 코드만으로 로그인할 수 있게 합니다.',
          login: '직접 웹 로그인',
          webTerminal: 'Web 터미널',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'MCP API 키',
          targetPower: '대상 장치 전원',
          nanoKVMRestart: 'NanoKVM 재시작',
          deleteImage: '이미지 삭제',
          scripts: '스크립트 작업',
          screenTimelapse: '화면 타임랩스',
          loginScope:
            '사용자 이름과 비밀번호 대신 인증 코드 또는 사용하지 않은 복구 코드만으로 웹에 로그인할 수 있습니다. 로그인 보안이 낮아집니다.',
          webTerminalScope: 'Web 터미널 열기.',
          sshScope: 'SSH 서비스 상태 변경.',
          tailscaleScope: 'Tailscale 설치, 연결 또는 변경.',
          frp: 'FRP',
          frpScope: 'FRP 설치, 연결 또는 변경.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Cloudflare Tunnel 설치, 연결 또는 변경.',
          zerotierScope: 'ZeroTier 설치, 연결 또는 변경.',
          netbirdScope: 'NetBird 설치, 연결 또는 변경.',
          mcpScope: 'MCP API 키 표시, 복사 또는 재생성 및 MCP 활성화 또는 비활성화.',
          targetPowerScope: '물리적 전원 버튼으로 대상 장치를 종료하거나 다시 시작합니다.',
          nanoKVMRestartScope: 'NanoKVM 자체를 다시 시작합니다.',
          deleteImageScope: '저장된 ISO 또는 IMG 이미지를 영구적으로 삭제합니다.',
          scriptsScope: '스크립트 실행, 업로드 또는 삭제.',
          screenTimelapseScope: '화면 타임랩스 녹화 활성화, 녹화물 열람 및 삭제.',
          authorizeTitle: '보호 설정 변경 승인',
          authorizeDesc:
            '인증 코드 또는 사용하지 않은 복구 코드를 입력하세요. 승인은 5분 동안 유효합니다.',
          authorizeFailed: '인증 코드 또는 복구 코드를 확인할 수 없습니다.',
          loadFailed: '보호 설정을 불러올 수 없습니다.',
          updateFailed: '보호 설정을 저장할 수 없습니다.',
          retry: '다시 시도',
          disableScriptsWarningTitle: '스크립트 보호를 해제하시겠습니까?',
          disableScriptsWarningDesc:
            '보호하지 않으면 악성 스크립트가 이 장치에 대한 상승된 액세스 권한을 얻고 2FA 보호를 우회하거나 제거할 수 있습니다.',
          cancelDisabling: '취소',
          continueDisabling: '해제 계속',
          continueDisablingCountdown: '해제 계속 ({{seconds}}초)'
        },
        twoFactorGuide: {
          title: '2단계 인증 안내',
          scope: '2단계 인증은 계정에 추가 보안 계층을 더합니다.',
          whatTitle: '2FA란?',
          whatDesc:
            '2단계 인증(2FA)은 인증 앱에서 생성되는 시간 기반 코드로 계정을 보호합니다. 기본적으로 이 코드는 비밀번호와 함께 필요하므로 비밀번호가 유출되더라도 계정을 보호할 수 있습니다. 직접 웹 로그인이 활성화된 경우 코드만으로 로그인됩니다.',
          appsTitle: '권장 앱',
          appsDesc:
            'Google Authenticator 또는 Microsoft Authenticator를 사용할 수 있습니다. 두 앱 모두 무료이며 iOS와 Android에서 작동합니다.',
          setupTitle: '설정 방법',
          setupDesc:
            '인증 앱을 열고 설정 중 표시되는 QR 코드를 스캔하거나 설정 키를 수동으로 입력하세요. 그런 다음 앱의 6자리 코드를 입력해 2FA 활성화를 완료하세요.',
          recoveryTitle: '복구 코드',
          recoveryDesc:
            '복구 코드는 일회용 백업 코드입니다. 인증 앱에 접근할 수 없을 때 복구 코드로 로그인할 수 있습니다. 생성한 즉시 안전한 위치에 저장하세요.',
          logoutNotice:
            '2FA를 활성화하면 현재 세션을 포함한 모든 활성 세션이 로그아웃됩니다. 모든 기기에서 다시 로그인해야 합니다.'
        }
      }
    },
    error: {
      title: '문제가 발생했습니다.',
      refresh: '새로고침'
    },
    pictureInPicture: {
      title: '그림 속 그림',
      enter: '그림 속 그림 시작',
      exit: '그림 속 그림 종료',
      error: '그림 속 그림을 시작할 수 없습니다. 동영상이 로드될 때까지 기다린 후 다시 시도하세요.'
    },
    fullscreen: {
      toggle: '전체 화면 전환'
    },
    menu: {
      collapse: '메뉴 접기',
      expand: '메뉴 펼치기'
    }
  }
};

export default ko;
