const vi = {
  translation: {
    storage: {
      cards: { mount: 'Gắn đĩa', share: 'Chia sẻ tệp' },
      title: 'Hình ảnh và chia sẻ tệp',
      empty: 'Chưa có hình ảnh nào',
      notAttached: 'Chưa gắn kết gì',
      attached: 'Đã gắn kết: {{name}}',
      eject: 'Đẩy ra',
      mount: 'Gắn kết',
      mountMode: 'Chế độ gắn kết',
      mountModeDisk: 'Lưu trữ hàng loạt',
      mountModeCdrom: 'CD-ROM',
      openManager: 'Mở trình quản lý tệp',
      refresh: 'Làm mới',
      delete: 'Xóa',
      deleteTitle: 'Xóa phương tiện',
      deleteConfirm: 'Xóa {{name}}? Các tệp bên trong sẽ bị mất.',
      downloadImage: 'Tải hình ảnh xuống',
      typePartition: 'Tệp chia sẻ',
      volumeLabel: 'Đĩa di động',
      groups: {
        share: 'Chia sẻ tệp',
        images: 'Hình ảnh'
      },
      failed: 'Thao tác không thành công',
      hidOnly: 'Chế độ chỉ HID: đĩa ảo không khả dụng',
      disabled: 'Đĩa ảo đang tắt (Cài đặt - Thiết bị)',
      force: {
        title: 'Cưỡng chế lấy lại phương tiện?',
        desc: 'Máy đích vẫn đang dùng phương tiện này. Lấy lại sẽ liệt kê lại thiết bị USB và có thể mất những gì nó chưa ghi vào đĩa.',
        menuHint: 'Máy đích đang dùng phương tiện này. Mở trình quản lý tệp để cưỡng chế lấy lại.'
      },
      tips: {
        title: 'Cách tải lên',
        usb1: 'Kết nối NanoKVM với máy tính của bạn qua USB.',
        usb2: 'Trên trang này, gắn các tệp chia sẻ ở chế độ lưu trữ dung lượng lớn.',
        usb3: 'Mở đĩa ảo trên máy tính của bạn và sao chép vào thư mục gốc của đĩa ảo.',
        usb4: 'Đẩy đĩa ra khi sao chép xong: ảnh đĩa sẽ xuất hiện trong danh sách.',
        scp1: 'Đảm bảo NanoKVM và máy tính của bạn đang trên cùng một mạng nội bộ.',
        scp2: 'Mở terminal trên máy tính và sử dụng lệnh SCP để tải đĩa ảo lên thư mục /data trên NanoKVM.',
        scp3: 'Ví dụ: scp đường-dẫn-image root@ip-của-nanokvm:/data'
      },
      job: {
        running: 'Đang xử lý {{name}}'
      },
      enableDisk: {
        title: 'Bật đĩa ảo?',
        desc: 'Việc gắn kết phương tiện cần đĩa ảo, hiện đang tắt. Bật ngay bây giờ? Bàn phím và chuột của máy đích sẽ ngắt kết nối trong giây lát.',
        menuHint: 'Đĩa ảo đang tắt. Hãy bật trong Cài đặt - Thiết bị.'
      },
      add: {
        title: 'Thêm phương tiện',
        upload: 'Tải lên tệp ISO hoặc IMG',
        fromUrl: 'Tải xuống từ URL',
        url: 'URL hình ảnh',
        urlRequired: 'Vui lòng nhập URL',
        onlyImages: 'Chỉ có thể gắn kết tệp .iso và .img',
        dropHint: 'Kéo tệp ISO hoặc IMG vào đây, hoặc bấm để chọn tệp',
        cdromOnlyIso: 'CD-ROM chỉ có thể chứa ảnh ISO 9660 / UDF',
        uploadFailed: 'Tải lên không thành công',
        downloadFailed: 'Tải xuống không thành công',
        uploading: 'Đang tải lên {{name}}',
        downloading: 'Đang tải xuống {{name}}',
        done: '{{name}} đã sẵn sàng',
        checksumFailed: 'Ảnh đĩa không khớp với mã kiểm tra SHA-256.',
        blocked:
          'Máy đích đang dùng đĩa. Hãy tháo an toàn ổ USB ảo trên máy đích, sau đó tắt công tắc Đĩa ảo trong Cài đặt > Thiết bị.'
      },
      files: {
        reasonNotInitialized:
          'Ổ USB ảo chưa được khởi tạo. Hãy khởi tạo trong Cài đặt > Thiết bị trước khi chia sẻ tệp.',
        selectMedium: 'Chọn một phương tiện để xem các tệp bên trong.',
        reasonAttached: 'Phương tiện này đang gắn với máy đích.',
        reasonAttachedHint:
          'Hãy tháo an toàn ổ USB ảo trên máy đích, sau đó tắt công tắc Đĩa ảo trong Cài đặt > Thiết bị.',
        reasonNotBrowsable: 'Phương tiện này chứa hệ thống tệp mà trình quản lý không mở được.',
        reasonNoLoop: 'Thiết bị này không gắn kết được ảnh đĩa (không có hỗ trợ loop).',
        reasonISOImage:
          'Không thể duyệt nội dung ảnh ISO 9660 tại đây. Ảnh vẫn có thể được gắn kết vào máy đích.',
        empty: 'Thư mục này trống',
        noMatch: 'Không có tệp nào khớp với tìm kiếm.',
        free: '{{size}} trống',
        search: 'Tìm trong thư mục này',
        sort: {
          modified: 'Ngày sửa đổi',
          name: 'Tên',
          size: 'Kích thước',
          type: 'Loại',
          asc: 'Tăng dần',
          desc: 'Giảm dần'
        },
        view: {
          list: 'Dạng danh sách',
          grid: 'Dạng biểu tượng'
        },
        folder: 'Thư mục',
        itemCount: 'Số mục: {{count}}',
        itemCount_one: '{{count}} mục',
        itemCount_other: '{{count}} mục',
        dropHint: 'Thả tệp vào đây để tải lên thư mục này',
        up: 'Đi đến thư mục cha',
        uploadButton: 'Tải lên',
        newFolder: 'Thư mục mới',
        newFolderPlaceholder: 'Tên thư mục',
        create: 'Tạo',
        upload: 'Tải một tệp lên phương tiện',
        deleteFolderTitle: 'Xóa thư mục',
        deleteFolderConfirm:
          'Xóa thư mục {{name}} và toàn bộ nội dung bên trong? Không thể hoàn tác thao tác này.',
        uploading: 'Đang tải lên…',
        uploadFailed: 'Tải lên không thành công',
        download: 'Tải xuống',
        delete: 'Xóa',
        deleteTitle: 'Xóa tệp',
        deleteConfirm: 'Xóa {{name}} khỏi phương tiện?'
      },
      yes: 'Có',
      no: 'Không'
    },
    head: {
      storage: 'Hình ảnh và chia sẻ tệp',
      desktop: 'Màn hình từ xa',
      login: 'Đăng nhập',
      changePassword: 'Đổi mật khẩu',
      terminal: 'Thiết bị đầu cuối',
      wifi: 'Wi-Fi'
    },
    clipboard: {
      copyFailed:
        'Không thể sao chép vào bộ nhớ tạm. Hãy cho phép trình duyệt truy cập bộ nhớ tạm rồi thử lại.'
    },
    auth: {
      login: 'Đăng nhập',
      language: 'Ngôn ngữ',
      placeholderUsername: 'Vui lòng nhập tên người dùng',
      placeholderPassword: 'vui lòng nhập mật khẩu',
      placeholderPassword2: 'vui lòng nhập lại mật khẩu',
      noEmptyUsername: 'tên người dùng không được để trống',
      noEmptyPassword: 'mật khẩu không được để trống',
      noAccount:
        'Không thể lấy thông tin người dùng, vui lòng làm mới trang web hoặc đặt lại mật khẩu',
      invalidUser: 'tên người dùng hoặc mật khẩu không hợp lệ',
      error: 'lỗi không mong đợi',
      twoFactorTitle: 'Cần xác minh',
      twoFactorDesc: 'Nhập mã xác thực hoặc mã khôi phục chưa dùng để đăng nhập.',
      sensitiveActionDesc: 'Nhập mã xác thực hoặc mã khôi phục chưa dùng để tiếp tục.',
      placeholderAuthCode: 'Mã xác thực',
      noEmptyAuthCode: 'Cần nhập mã xác thực',
      verify: 'Xác minh',
      backToLogin: 'Quay lại đăng nhập',
      verificationFailed: 'Xác minh thất bại',
      twoFactorEnabledLogout: {
        message:
          'Xác thực hai yếu tố đã được bật. Vì lý do bảo mật, tất cả phiên đăng nhập đã bị đăng xuất. Vui lòng đăng nhập lại.'
      },
      rateLimited: 'Quá nhiều lần thử thất bại. Thử lại sau {{seconds}} giây.',
      recoveryCodeUsed: 'Mã khôi phục dùng cho lần đăng nhập này đã bị vô hiệu hóa.',
      changePassword: 'Đổi mật khẩu',
      changePasswordDesc: 'Để bảo mật thiết bị của bạn, vui lòng thay đổi mật khẩu đăng nhập web.',
      passwordVerificationRequired:
        'Vui lòng quay lại trang tài khoản trong cài đặt và xác minh lại lần đổi mật khẩu này.',
      passwordVerificationExpired:
        'Xác minh đổi mật khẩu đã hết hạn. Vui lòng quay lại trang tài khoản trong cài đặt và xác minh lại.',
      returnToSettings: 'Quay lại cài đặt',
      differentPassword: 'mật khẩu không khớp',
      illegalUsername: 'tên người dùng chứa ký tự không hợp lệ',
      illegalPassword: 'mật khẩu chứa ký tự không hợp lệ',
      forgetPassword: 'Quên mật khẩu',
      ok: 'OK',
      cancel: 'Hủy',
      savePassword: 'Lưu mật khẩu',
      loginButtonText: 'Đăng nhập',
      tips: {
        reset1: 'Tài khoản mặc định trên web:',
        reset2: 'Tài khoản SSH mặc định:',
        reset3: 'Để đặt lại mật khẩu, vui lòng tham khảo ',
        changeReset: 'Để đặt lại mật khẩu đăng nhập web, vui lòng tham khảo ',
        resetLink: 'Đặt lại'
      }
    },
    wifi: {
      errors: {
        connect:
          'Không thể kết nối Wi-Fi. Hãy kiểm tra tên mạng, mật khẩu và trạng thái Wi-Fi của thiết bị rồi thử lại.'
      },
      title: 'Wi-Fi',
      description: 'Cấu hình Wi-Fi cho NanoKVM',
      submitted: {
        title: 'Vui lòng đợi — NanoKVM đang kết nối vào Wi-Fi của bạn',
        tipsTitle: 'Tiếp theo',
        tipSwitch: 'Bây giờ bạn có thể chuyển điện thoại về mạng Wi-Fi thường dùng.',
        tipCheck:
          'Sau ít phút, hãy xem địa chỉ IP mới trên màn hình NanoKVM rồi mở bằng trình duyệt.',
        tipRetry:
          'Nếu đợi lâu vẫn không thấy gì, hãy kiểm tra lại tên mạng và mật khẩu rồi thử lại.'
      },
      invalidMode:
        'Chế độ hiện tại không hỗ trợ thiết lập mạng. Vui lòng truy cập thiết bị của bạn và bật chế độ cấu hình Wi-Fi.',
      confirmBtn: 'OK',
      finishBtn: 'Hoàn tất',
      ssidPlaceholder: 'SSID',
      passwordPlaceholder: 'Mật khẩu',
      steps: {
        settings: 'Cài đặt',
        wifi: 'Wi-Fi',
        enable: 'Bật'
      },
      scan: {
        title: 'Chọn mạng Wi-Fi',
        description: 'Chọn mạng lân cận để tiếp tục.',
        loading: 'Đang tìm mạng Wi-Fi lân cận…',
        empty: 'Không tìm thấy mạng Wi-Fi lân cận.',
        failed: 'Không thể quét mạng. Hãy nhập thông tin thủ công.',
        retryBtn: 'Quét lại',
        manualBtn: 'Nhập mạng thủ công',
        backBtn: 'Quay lại các mạng lân cận',
        connectTitle: 'Kết nối với {{ssid}}',
        connecting: 'Đang kết nối…',
        viewSimple: 'Chuyển sang chế độ xem đơn giản',
        viewDetailed: 'Chuyển sang chế độ xem chi tiết'
      },
      ap: {
        authTitle: 'Yêu cầu xác thực',
        authDescription: 'Vui lòng nhập mật khẩu AP để tiếp tục',
        authFailed: 'Mật khẩu AP không hợp lệ',
        passPlaceholder: 'AP mật khẩu',
        verifyBtn: 'Xác minh'
      },
      qr: {
        cameraBtn: 'Chụp ảnh',
        pickBtn: 'Chọn ảnh chụp màn hình',
        scanHint:
          'Chụp ảnh mã QR Wi-Fi, hoặc chọn ảnh chụp màn hình của mã đó, chẳng hạn như mã mà điện thoại hiển thị khi chia sẻ Wi-Fi của nó.',
        decoding: 'Đang đọc mã QR…',
        notFound:
          'Không tìm thấy mã QR trong ảnh này. Hãy căn giữa mã và lấy nét rõ ràng rồi thử lại.',
        invalid: 'Mã QR này không chứa thông tin Wi-Fi có thể sử dụng được.',
        pickerUnavailable:
          'Không có gì mở ra? Cửa sổ đăng nhập này có thể chặn máy ảnh và ảnh. Thay vào đó, hãy mở địa chỉ sau trong trình duyệt điện thoại của bạn:',
        confirmTitle: 'Kết nối NanoKVM với {{ssid}}?',
        confirmDescription: 'Thông tin mạng này lấy từ mã QR. Hãy kiểm tra trước khi kết nối.',
        openNetwork: 'Mạng này không có mật khẩu.',
        connectBtn: 'Kết nối',
        otherBtn: 'Chọn mạng khác'
      }
    },
    notification: {
      browser: {
        title: 'Đề xuất trình duyệt',
        description:
          'Để có trải nghiệm người dùng tốt nhất, chúng tôi khuyên bạn nên sử dụng trình duyệt Chrome. Mặc dù chúng tôi cố gắng đạt được khả năng tương thích trên tất cả các trình duyệt chính nhưng một số tính năng nâng cao có thể không hoạt động như dự định bên ngoài Chrome.'
      },
      btnOk: 'Được rồi'
    },
    screen: {
      title: 'Màn hình',
      video: 'Chế độ video',
      quality: 'Chất lượng',
      qualityAuto: 'Tự động',
      qualityLossless: 'Không mất dữ liệu',
      qualityHigh: 'Cao',
      qualityMedium: 'Trung bình',
      qualityLow: 'Thấp',
      scale: 'Quy mô',
      noSignal: 'Không có tín hiệu video',
      captureDisabled: 'Tính năng thu video đã tắt. Hãy bật tại "Cài đặt > Thiết bị > Thu video".',
      inconsistentVideoMode: 'Chơi đã tạm dừng. Một chế độ video khác đang phát.',
      bandwidthWarning:
        'Băng thông mạng có thể không đủ cho cài đặt video hiện tại. Hãy thử giảm tốc độ bit hoặc tốc độ khung hình.',
      webrtcConnectionFailed:
        'Không thể thiết lập kết nối WebRTC. Vui lòng chuyển sang chế độ video Direct.',
      webrtcUnavailable: {
        title: 'WebRTC không khả dụng',
        description: 'Hãy bật WebRTC trong trình duyệt hoặc chuyển sang chế độ video Direct.'
      },
      upload: 'Tải lên',
      advanced: 'Nâng cao',
      loading: 'Đang tải'
    },
    volume: {
      title: 'Âm lượng'
    },
    microphone: {
      title: 'Micrô',
      notMounted: 'Chưa gắn micrô',
      tip1: 'Micrô ảo chưa được gắn trên máy chủ từ xa. Bạn có muốn gắn kết và kích hoạt nó ngay bây giờ không?',
      tip2: 'Bạn cũng có thể quản lý việc này sau trong "Cài đặt > Thiết bị > Âm thanh ảo".',
      confirm: 'Kích hoạt',
      cancel: 'Hủy',
      noDevice: 'Không phát hiện thấy micrô trên thiết bị này. Hãy kết nối micrô rồi thử lại.',
      permissionDenied:
        'Quyền truy cập micrô đã bị từ chối. Hãy cho phép truy cập trong phần cài đặt trình duyệt.',
      notReadable: 'Micrô đang được ứng dụng khác sử dụng hoặc không thể đọc được.',
      overconstrained: 'Micrô không hỗ trợ các thông số âm thanh cần thiết.',
      securityBlocked:
        'Quyền truy cập micrô bị chặn trên trang này. Hãy kiểm tra cài đặt bảo mật của trình duyệt.',
      aborted: 'Quá trình khởi động micrô đã bị hủy. Vui lòng thử lại.',
      unsupported: 'Trình duyệt hoặc phương thức kết nối này không hỗ trợ thu âm từ micrô.',
      enableFailed: 'Không thể bật micrô. Vui lòng thử lại.',
      disconnected: 'Micrô đã bị ngắt kết nối nên việc truyền âm thanh từ micrô đã bị tắt.'
    },
    power: {
      title: 'Nguồn',
      showConfirm: 'Xác nhận',
      showConfirmTip: 'Thao tác nguồn cần xác nhận thêm',
      powerShort: 'Nguồn (nhấp ngắn)',
      powerLong: 'Nguồn (nhấp dài)',
      powerConfirm: 'Thực hiện thao tác nguồn?',
      okBtn: 'Có',
      cancelBtn: 'Không'
    },
    keyboard: {
      errors: {
        paste: 'Không thể gửi văn bản đến thiết bị từ xa. Hãy kiểm tra kết nối KVM rồi thử lại.'
      },
      title: 'Bàn phím',
      paste: 'Dán',
      tips: 'Chỉ hỗ trợ các chữ cái và ký hiệu bàn phím tiêu chuẩn',
      placeholder: 'Vui lòng nhập',
      submit: 'Gửi',
      virtual: 'Bàn phím',
      swapCommandCtrl: 'Hoán đổi Cmd và Ctrl',
      shortcut: {
        title: 'Phím tắt',
        custom: 'Tùy chỉnh',
        capture: 'Bấm vào đây để chụp phím tắt',
        clear: 'Xóa',
        save: 'Lưu',
        captureTips:
          'Việc ghi nhận các phím cấp hệ thống (như phím Windows) yêu cầu quyền toàn màn hình.',
        enterFullScreen: 'Chuyển sang chế độ toàn màn hình.'
      },
      leaderKey: {
        title: 'Phím Leader',
        desc: 'Bỏ qua các hạn chế của trình duyệt và gửi các phím tắt hệ thống trực tiếp đến máy chủ từ xa.',
        howToUse: 'Cách sử dụng',
        simultaneous: {
          title: 'Chế độ đồng thời',
          desc1: 'Nhấn giữ phím Leader, rồi nhấn phím tắt.',
          desc2: 'Trực quan nhưng có thể xung đột với các phím tắt hệ thống.'
        },
        sequential: {
          title: 'Chế độ tuần tự',
          desc1: 'Nhấn phím Leader → nhấn phím tắt theo thứ tự → nhấn lại phím Leader.',
          desc2: 'Yêu cầu nhiều bước hơn nhưng hoàn toàn tránh được xung đột hệ thống.'
        },
        enable: 'Bật phím Leader',
        tip: 'Khi được gán làm phím Leader, phím này chỉ hoạt động như bộ kích hoạt phím tắt và mất hành vi mặc định.',
        placeholder: 'Vui lòng nhấn phím Leader',
        shiftRight: 'Shift phải',
        ctrlRight: 'Ctrl phải',
        metaRight: 'Win phải',
        submit: 'Gửi',
        recorder: {
          rec: 'REC',
          activate: 'Kích hoạt phím',
          input: 'Hãy nhấn phím tắt...'
        }
      }
    },
    mouse: {
      title: 'Chuột',
      mode: 'Chế độ chuột',
      absolute: 'Chuột bám theo',
      absoluteAndroid: 'Chuột bám theo (tương thích Android tốt hơn)',
      touchScreen: 'Màn hình cảm ứng đa điểm',
      relative: 'Chuột độc quyền',
      direction: 'Hướng bánh xe cuộn',
      scrollUp: 'Cuộn lên',
      scrollDown: 'Cuộn xuống',
      speed: 'Tốc độ bánh xe cuộn',
      fast: 'Nhanh lên',
      slow: 'Chậm',
      requestPointer:
        'Đang sử dụng chế độ tương đối. Vui lòng nhấp vào màn hình để lấy con trỏ chuột.',
      repairIphoneDrag: 'Sửa kéo trên iPhone',
      resetHid: 'Đặt lại HID',
      touchpadGuide: {
        title: 'Hướng dẫn bàn di chuột',
        scope: 'Áp dụng cho thao tác con trỏ thông thường khi Chế độ chuột là Tương đối.',
        swipeTitle: 'Vuốt để di chuyển',
        swipeDesc: 'Vuốt trong vùng màn hình hoạt động để di chuyển con trỏ từ xa.',
        tapTitle: 'Chạm để nhấp',
        tapDesc: 'Một lần chạm ngắn sẽ gửi một cú nhấp trái.',
        holdTitle: 'Giữ để nhấn nút trái',
        holdDesc: 'Chạm và giữ yên khoảng 1 giây để giữ nút trái chuột ở trạng thái nhấn.',
        dragTitle: 'Di chuyển sau khi giữ để kéo',
        dragDesc: 'Sau khi giữ có hiệu lực, di chuyển ngón tay để kéo với nút trái đang được giữ.'
      },
      hidOnly: {
        errors: {
          load: 'Không thể tải trạng thái chế độ HID-Only. Hãy thử lại sau ít phút.',
          update:
            'Không thể xác nhận chế độ HID-Only đã được cập nhật hay chưa. Hãy mở lại cài đặt này để kiểm tra trạng thái hiện tại rồi thử lại.'
        },
        title: 'Chế độ chỉ HID',
        desc: 'Nếu chuột và bàn phím của bạn ngừng phản hồi và việc đặt lại HID không có tác dụng thì đó có thể là sự cố tương thích giữa NanoKVM và thiết bị. Hãy thử bật chế độ HID-Only để tương thích tốt hơn.',
        tip1: 'Kích hoạt HID-Chế độ chỉ sẽ ngắt kết nối đĩa U ảo và mạng ảo',
        tip2: 'Ở chế độ HID-Chỉ, tính năng gắn hình ảnh bị tắt',
        enable: 'Bật chế độ HID-Chỉ',
        disable: 'Tắt chế độ HID-Chỉ'
      }
    },
    preview: {
      title: 'Xem trước'
    },
    image: {
      storageReadOnly: 'Hiện không thể ghi vào bộ nhớ tệp. Vui lòng tắt ổ đĩa ảo.',
      storageBusy:
        'Bộ nhớ đang được sử dụng. Hãy hoàn tất hoặc hủy truyền tệp trước khi chuyển trạng thái ổ đĩa ảo.',
      title: 'Hình ảnh',
      loading: 'Đang tải...',
      empty: 'Không tìm thấy',
      upload: 'Tải hình ảnh lên',
      dropFile: 'Thả ảnh đĩa vào đây hoặc nhấp để chọn tệp',
      uploadAction: 'Tải lên',
      cancelAction: 'Hủy',
      mount: 'Gắn hình ảnh',
      mounted: 'Đã gắn kết',
      uploadError: 'Tải lên không thành công, vui lòng thử lại',
      uploadInsufficientStorage: 'Không đủ dung lượng lưu trữ',
      uploadTooLarge: 'Image vượt quá giới hạn kích thước',
      cancelUploadError: 'Không thể hủy tải lên',
      mountMode: 'Chế độ gắn kết',
      massStorage: 'Lưu trữ khối',
      cdRom: 'CD ROM',
      readOnly: 'Chỉ đọc',
      mountFailed: 'Mount thất bại',
      mountDesc: 'Trong một số hệ thống, cần phải eject đĩa ảo trên máy remote trước khi mount.',
      unmountFailed: 'Gỡ bỏ thất bại',
      unmountDesc:
        'Trong một số hệ thống, bạn cần phải tháo gỡ thủ công trên máy chủ từ xa trước khi gỡ bỏ ảnh.',
      refresh: 'Làm mới danh sách hình ảnh',
      download: 'Tải hình ảnh xuống',
      attention: 'Chú ý',
      deleteConfirm: 'Bạn có chắc chắn muốn xóa hình ảnh này không?',
      deleteFailed: 'Không thể xóa hình ảnh',
      okBtn: 'Có',
      cancelBtn: 'Không'
    },
    script: {
      title: 'Tập lệnh',
      upload: 'Tải lên',
      run: 'Chạy',
      runBackground: 'Chạy nền',
      runFailed: 'Chạy thất bại',
      uploadFailed: 'Không thể tải tập lệnh lên',
      deleteFailed: 'Không thể xóa tập lệnh',
      attention: 'Chú ý',
      delDesc: 'Bạn có chắc chắn muốn xóa tệp này không?',
      confirm: 'Có',
      cancel: 'Không',
      delete: 'Xóa',
      close: 'Đóng'
    },
    terminal: {
      title: 'Thiết bị đầu cuối',
      nanokvm: 'Thiết bị đầu cuối NanoKVM',
      confirm: 'OK',
      submit: 'Gửi',
      username: 'Tên người dùng',
      password: 'Mật khẩu',
      authCode: 'Mã xác thực',
      authorizeTitle: 'Ủy quyền Web Terminal',
      codeAccountHint: 'Nhập mã xác thực hoặc mã khôi phục chưa dùng.',
      authFailed: 'Xác thực thất bại'
    },
    wol: {
      errors: {
        send: 'Không thể gửi lệnh đánh thức. Hãy kiểm tra địa chỉ MAC và kết nối mạng rồi thử lại.',
        loadFailed:
          'Không thể tải danh sách thiết bị Wake-on-LAN. Hãy kiểm tra NanoKVM đang trực tuyến rồi thử lại.',
        renameFailed: 'Không thể đổi tên thiết bị này. Hãy kiểm tra địa chỉ MAC rồi thử lại.',
        deleteFailed: 'Không thể xóa thiết bị này. Vui lòng thử lại sau.'
      },
      title: 'Wake-on-LAN',
      sending: 'Đang gửi lệnh...',
      sent: 'Đã gửi lệnh',
      input: 'Vui lòng nhập địa chỉ MAC',
      ok: 'OK'
    },
    download: {
      sha256: 'SHA-256 (tùy chọn)',
      sha256Placeholder: 'Nhập mã kiểm tra SHA-256 gồm 64 ký tự',
      invalidSHA256: 'SHA-256 phải là chuỗi thập lục phân gồm 64 ký tự',
      title: 'Trình tải xuống hình ảnh',
      input: 'Vui lòng nhập hình ảnh từ xa URL',
      ok: 'OK',
      disabled: '/data phân vùng là RO nên không tải được image',
      downloadingFile: 'Đang tải xuống: {{file}}',
      downloadingProgress: 'Đang tải xuống ({{percentage}}): {{file}}',
      failed: 'Thất bại',
      insufficientStorage: 'Không đủ dung lượng lưu trữ',
      remaining: 'Còn lại {{time}}',
      cancel: 'Hủy',
      cancelFailed: 'Không thể hủy tải xuống',
      pause: 'Tạm dừng',
      resume: 'Tiếp tục',
      pauseFailed: 'Không thể tạm dừng tải xuống',
      resumeFailed: 'Không thể tiếp tục tải xuống'
    },
    settings: {
      back: 'Quay lại',
      close: 'Đóng',
      ai: {
        errors: {
          loadServices:
            'Không thể tải trạng thái dịch vụ AI. Hãy đảm bảo NanoKVM đang trực tuyến rồi thử lại.',
          loadMcpConfig: 'Không thể tải cấu hình dịch vụ MCP. Hãy thử lại sau ít phút.',
          updateMcpService:
            'Không thể cập nhật trạng thái dịch vụ MCP. Hãy kiểm tra trạng thái hiện tại trước khi thử lại.',
          accessMcpApiKey:
            'Không thể lấy MCP API Key. Hãy thử lại và hoàn tất xác minh nếu được yêu cầu.',
          regenerateMcpApiKey: 'Không thể tạo lại MCP API Key. Hãy thử lại sau ít phút.',
          regeneratedMcpApiKeyUnavailable:
            'API Key đã được tạo lại nhưng không thể hiển thị. Hãy thử hiển thị lại. Key trước đó không còn hợp lệ.',
          loadMemoryFabricState:
            'Không thể tải trạng thái dịch vụ Memory Fabric. Hãy thử lại sau ít phút.',
          updateMemoryFabricService:
            'Không thể cập nhật trạng thái dịch vụ Memory Fabric. Hãy kiểm tra trạng thái hiện tại trước khi thử lại.',
          loadScreenTimelapseState:
            'Không thể tải trạng thái dịch vụ timelapse màn hình. Hãy thử lại sau ít phút.',
          updateScreenTimelapseService:
            'Không thể cập nhật trạng thái dịch vụ timelapse màn hình. Hãy kiểm tra trạng thái hiện tại trước khi thử lại.',
          openScreenTimelapse: 'Không thể mở phát timelapse màn hình. Hãy thử lại sau ít phút.',
          loadScreenTimelapseRecordings:
            'Không thể tải bản ghi timelapse màn hình. Hãy thử lại sau ít phút.',
          playScreenTimelapse:
            'Không thể phát bản ghi timelapse này ngay bây giờ. Hãy chọn lại ngày hoặc thử lại sau.',
          screenTimelapseFullscreen:
            'Không thể thay đổi chế độ toàn màn hình. Hãy kiểm tra quyền của trình duyệt rồi thử lại.'
        },
        title: 'AI',
        mcpService: 'Dịch vụ MCP',
        mcpServiceDesc:
          'Cổng hợp nhất có xác thực cho điều khiển, phương tiện và các công cụ Memory Fabric đã bật',
        memoryFabricService: 'Memory Fabric',
        memoryFabricServiceDesc: 'Xử lý cục bộ bộ nhớ ngữ cảnh của agent và OCR',
        memoryFabricRemoteRequiresMcp:
          'Bật Dịch vụ MCP ở trên để cung cấp công cụ bộ nhớ và OCR từ xa.',
        memoryFabricRemoteAvailable:
          'Công cụ bộ nhớ và OCR có sẵn qua endpoint /api/mcp ở trên bằng cùng khóa API.',
        memoryFabricConfig: {
          title: 'Cấu hình',
          description: 'Chỉ quản lý cài đặt mô hình mặc định của Memory Fabric.',
          loadFailed: 'Không thể tải cấu hình Memory Fabric.',
          saveFailed: 'Không thể lưu cấu hình Memory Fabric.',
          saved: 'Đã lưu cấu hình.',
          savedRestarted: 'Đã lưu cấu hình và khởi động lại Memory Fabric.',
          noChanges: 'Cấu hình không có thay đổi.',
          reload: 'Tải lại',
          save: 'Lưu',
          unsaved: 'Có thay đổi chưa lưu',
          conflictTitle: 'Cấu hình đã thay đổi',
          conflictDescription:
            'Cấu hình đã được sửa ở nơi khác. Hãy tải phiên bản mới nhất trước khi lưu lại.',
          conflictReload: 'Tải bản mới nhất',
          discardTitle: 'Bỏ các thay đổi chưa lưu?',
          discardDescription:
            'Tải lại sẽ thay nội dung biểu mẫu hiện tại bằng cấu hình đã lưu mới nhất.',
          discardConfirm: 'Bỏ thay đổi và tải lại',
          sections: {
            models: 'Mô hình'
          },
          openAICompatibleHint:
            'Hỗ trợ các nhà cung cấp mô hình tương thích với OpenAI. Nhập tên mô hình, URL cơ sở và khóa API do nhà cung cấp cung cấp.',
          fields: {
            model: 'Mô hình',
            baseUrl: 'URL cơ sở',
            maxTokens: 'Số token tối đa',
            timeoutSeconds: 'Thời gian chờ (giây)'
          },
          apiKey: {
            title: 'Khóa API',
            configured: 'Đã cấu hình',
            notConfigured: 'Chưa cấu hình',
            keep: 'Giữ khóa hiện tại',
            replace: 'Thay khóa',
            clear: 'Xóa khóa',
            replacement: 'Khóa API mới',
            replacementPlaceholder: 'Nhập khóa API mới',
            clearHint:
              'Khóa API đã lưu sẽ bị xóa; biến môi trường vẫn có thể cung cấp một khóa khác.',
            required: 'Hãy nhập khóa API mới.'
          },
          validation: {
            required: 'Trường này là bắt buộc.',
            invalidUrl: 'Hãy nhập URL HTTP hoặc HTTPS hợp lệ.'
          }
        },
        screenTimelapseService: 'Timelapse màn hình',
        screenTimelapseServiceDesc:
          'Phát lại lịch sử màn hình Memory Fabric và tìm kiếm nội dung màn hình đã ghi',
        screenTimelapseSearchRequiresMemoryFabric:
          'Để tìm kiếm nội dung đã ghi, hãy bật cả Memory Fabric.',
        screenTimelapseOpen: 'Mở',
        screenTimelapseSearchPlaceholder: 'Tìm nội dung trong video...',
        screenTimelapseNoVideos: 'Không tìm thấy video timelapse',
        screenTimelapseNoResults: 'Không tìm thấy khung hình khớp',
        screenTimelapseMultipleResults: 'Kết quả tìm kiếm',
        screenTimelapseUnavailable: 'Không khả dụng',
        screenTimelapsePlayerUnsupported: 'Trình duyệt này không thể phát video timelapse',
        screenTimelapseSearchFailed: 'Tìm kiếm thất bại',
        screenTimelapseLoading: 'Đang tải timelapse...',
        screenTimelapseSearchAction: 'Tìm kiếm timelapse',
        screenTimelapsePlay: 'Phát timelapse',
        screenTimelapsePause: 'Tạm dừng timelapse',
        screenTimelapseEnterFullscreen: 'Vào toàn màn hình',
        screenTimelapseExitFullscreen: 'Thoát toàn màn hình',
        screenTimelapseFullscreenUnsupported: 'Trình duyệt này không thể vào toàn màn hình',
        screenTimelapsePreviousDay: 'Ngày trước',
        screenTimelapseNextDay: 'Ngày sau',
        screenTimelapseDeleteDay: 'Xóa bản ghi ngày {{date}}',
        screenTimelapseDelete: 'Xóa',
        screenTimelapseDeleteConfirmTitle: 'Xóa bản ghi tua nhanh thời gian?',
        screenTimelapseDeleteConfirmDesc:
          'Bản ghi ngày {{date}} chiếm {{size}}. Không thể hoàn tác thao tác này.',
        screenTimelapseDeleteSuccess: 'Đã xóa bản ghi ngày {{date}}',
        screenTimelapseDeleteFailed: 'Không thể xóa bản ghi ngày {{date}}',
        endpoint: 'Điểm cuối',
        apiKey: 'Khóa API',
        regenerate: 'Tạo lại',
        copyApiKey: 'Sao chép khóa API',
        copyEndpoint: 'Sao chép điểm cuối',
        showApiKey: 'Hiển thị khóa API',
        regenerateConfirmTitle: 'Tạo lại khóa API',
        regenerateConfirmDesc:
          'Các ứng dụng khách MCP hiện có sẽ ngừng hoạt động cho đến khi được cập nhật bằng khóa mới. Bạn có muốn tiếp tục không?',
        okBtn: 'Có',
        cancelBtn: 'Không'
      },
      title: 'Cài đặt',
      about: {
        title: 'Giới thiệu về NanoKVM',
        information: 'Thông tin',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Phiên bản Ứng dụng',
        applicationTip: 'Phiên bản ứng dụng web NanoKVM',
        arch: 'Kiến trúc',
        device_number: 'Số thiết bị',
        image: 'Phiên bản Hình ảnh',
        imageTip: 'Phiên bản hình ảnh hệ thống NanoKVM',
        deviceKey: 'Khóa Thiết bị',
        community: 'Cộng đồng',
        hostname: 'Tên máy chủ',
        hostnameUpdated: 'Đã cập nhật tên máy chủ. Khởi động lại để áp dụng.',
        ipType: {
          Wired: 'Có dây',
          Wireless: 'Không dây',
          Other: 'Khác'
        }
      },
      appearance: {
        title: 'Giao diện',
        display: 'Hiển thị',
        language: 'Ngôn ngữ',
        languageDesc: 'Chọn ngôn ngữ cho giao diện',
        languageChangeFailed: 'Đổi ngôn ngữ thất bại',
        webTitle: 'Tiêu đề trang web',
        webTitleDesc: 'Tùy chỉnh tiêu đề trang web',
        menuBar: {
          title: 'Thanh menu',
          mode: 'Chế độ hiển thị',
          modeDesc: 'Hiển thị thanh menu trên màn hình',
          modeOff: 'Tắt',
          modeAuto: 'Tự động ẩn',
          modeAlways: 'Luôn hiển thị',
          icons: 'Biểu tượng menu con',
          showIcon: 'Hiện biểu tượng',
          activateOnConnect: 'Bật khi kết nối',
          stopSharing: 'Dừng chia sẻ',
          iconsDesc: 'Hiển thị biểu tượng menu con trên thanh menu'
        }
      },
      screen: {
        title: 'Màn hình',
        videoMode: {
          title: 'Chế độ video',
          description: 'Chọn phương thức mã hóa và truyền'
        },
        edid: {
          description: 'Chọn thông số hiển thị',
          default: 'Mặc định',
          custom: 'Tùy chỉnh',
          document: 'Tài liệu',
          powerConfirmTitle: 'Xác nhận trước khi cập nhật EDID',
          powerConfirmDescription:
            'Hãy kết nối nguồn PD trước. Cập nhật EDID có thể làm ngắt màn hình trong thời gian ngắn; nếu sau đó thiết bị không phát hiện đúng màn hình, hãy khởi động lại thủ công.',
          powerConfirmOk: 'Tiếp tục',
          powerConfirmOkCountdown: 'Tiếp tục ({{seconds}})',
          powerConfirmCancel: 'Hủy',
          errors: {
            badRequest: 'Yêu cầu EDID không hợp lệ',
            invalidFileName: 'Chọn tệp EDID .bin có tên tệp hợp lệ',
            fileTooLarge: 'Tệp EDID quá lớn',
            invalidLength: 'EDID phải có kích thước 128, 256, 384 hoặc 512 byte',
            invalidHeader: 'Phần đầu EDID không hợp lệ',
            unsupportedVersion: 'Phiên bản EDID này không được hỗ trợ',
            extensionCountMismatch: 'Số khối mở rộng EDID không khớp với độ dài tệp',
            invalidChecksum: 'Tổng kiểm tra EDID không hợp lệ',
            invalidCta: 'Cấu trúc phần mở rộng CTA không hợp lệ',
            currentCannotOverwrite:
              'Không thể ghi đè EDID đang được sử dụng. Hãy chuyển sang EDID khác trước',
            saveFailed: 'Không thể lưu EDID',
            switchFailed: 'Không thể chuyển EDID',
            uploadFailed: 'Không thể tải EDID lên',
            deleteFailed: 'Không thể xóa EDID',
            networkFailed: 'Lỗi mạng. Hãy thử lại'
          }
        },
        rateControlMode: {
          title: 'Chế độ kiểm soát tốc độ',
          description: 'Chọn phương thức kiểm soát tốc độ bit',
          cbr: 'CBR (Constant Bitrate): Ưu tiên độ ổn định của kết nối. Giảm thiểu tình trạng giật hình nhưng chất lượng hình ảnh có thể giảm trong các cảnh chuyển động cao.',
          vbr: 'VBR (Tốc độ bit thay đổi): Ưu tiên độ rõ nét của hình ảnh. Cảnh chuyển động cao có thể khiến băng thông tăng đột biến, có khả năng dẫn đến độ trễ hoặc giật hình.'
        },
        bitrate: {
          title: 'Tốc độ bit',
          description: 'Cân bằng độ rõ nét của hình ảnh với mức sử dụng băng thông',
          lower:
            'Giá trị thấp hơn giúp giảm độ trễ trên các mạng kém nhưng làm ảnh hưởng đến chất lượng hình ảnh, có khả năng gây mờ hoặc tạo hình giả.',
          higher:
            'Giá trị cao hơn sẽ cải thiện độ rõ nét của hình ảnh nhưng tiêu tốn nhiều băng thông hơn. Có thể tăng độ trễ nếu kết nối chậm.'
        },
        gop: {
          description: 'Đặt khoảng thời gian giữa các khung hình chính đầy đủ',
          lower:
            'Giá trị thấp hơn sẽ gửi khung hình đầy đủ thường xuyên hơn. Điều này tăng tốc độ phục hồi từ các tạo phẩm trực quan nhưng tiêu tốn nhiều băng thông hơn và làm giảm chi tiết hình ảnh.',
          higher:
            'Giá trị cao hơn sẽ tối ưu hóa cho các bản cập nhật gia tăng. Điều này cải thiện chi tiết hình ảnh và tiết kiệm băng thông, nhưng việc khôi phục từ các thành phần hình ảnh sẽ mất nhiều thời gian hơn.'
        },
        quality: {
          title: 'Chất lượng',
          description: 'Đặt chất lượng hình ảnh hiển thị'
        },
        fps: {
          auto: 'Tự động',
          description: 'Đặt tốc độ khung hình hiển thị',
          tip1: 'Việc đặt giá trị thành 0 sẽ bật chế độ Tự động, đồng bộ hóa màn hình với tốc độ khung hình của máy chủ từ xa.',
          tip2: 'Việc giảm tốc độ khung hình sẽ làm giảm đáng kể mức sử dụng băng thông. Giảm nó theo cách thủ công khi mạng không ổn định hoặc khi không cần tốc độ khung hình cao.'
        },
        scale: {
          title: 'Tỷ lệ',
          description: 'Đặt tỷ lệ chia tỷ lệ hiển thị'
        }
      },
      device: {
        title: 'Thiết bị',
        virtualAudioConfirm: {
          title: 'Xác nhận trước khi chuyển đổi âm thanh ảo',
          description:
            'Hãy kết nối nguồn PD trước. Việc chuyển đổi âm thanh ảo có thể làm ngắt màn hình trong thời gian ngắn; nếu sau đó thiết bị không phát hiện đúng màn hình, hãy khởi động lại thủ công.',
          ok: 'Tiếp tục chuyển đổi',
          okCountdown: 'Tiếp tục chuyển đổi ({{seconds}})',
          cancel: 'Hủy'
        },
        wifi: {
          title: 'Wi-Fi',
          description: 'Cấu hình Wi-Fi',
          apMode: 'Chế độ AP được bật, kết nối Wi-Fi theo cách thủ công',
          others: 'Những người khác...',
          connect: 'Kết nối Wi-Fi',
          connectDesc1: 'Vui lòng nhập ssid mạng và mật khẩu',
          connectDesc2: 'Vui lòng nhập mật khẩu để tham gia mạng này',
          disconnect: 'Bạn có chắc chắn ngắt kết nối mạng không?',
          ssid: 'Tên',
          password: 'Mật khẩu',
          joinBtn: 'Kết nối',
          confirmBtn: 'Được rồi',
          cancelBtn: 'Hủy',
          preset: {
            title: 'Wi-Fi lưu trước',
            description:
              'Lưu trước một Wi-Fi đích; thiết bị sẽ tự động kết nối với nó khi khởi động',
            empty: 'Không có',
            addBtn: 'Thêm cài đặt trước',
            removeBtn: 'Xóa cài đặt trước',
            saveTitle: 'Lưu Wi-Fi trước',
            saveDesc:
              'Lưu mạng đích ngay bây giờ; thiết bị sẽ tự động kết nối khi phát hiện mạng lúc khởi động',
            saveBtn: 'Lưu',
            passwordPlaceholder: 'Mật khẩu (để trống với mạng mở)',
            saveFailed: 'Không thể lưu Wi-Fi cài đặt trước',
            removeFailed: 'Không thể xóa Wi-Fi cài đặt trước',
            removeConfirm: 'Bạn có chắc muốn xóa Wi-Fi đã lưu trước không?',
            warning:
              'Sau khi lưu, mỗi lần khởi động thiết bị sẽ cố gắng kết nối với Wi-Fi này khi phát hiện được, điều này có thể cản trở việc kết nối với các Wi-Fi khác. Cài đặt sẵn sẽ có hiệu lực cho đến khi bạn chỉnh sửa hoặc xóa nó.'
          }
        },
        capture: {
          title: 'Thu video',
          description: 'Bật thu video'
        },
        ssh: {
          description: 'Kích hoạt SSH truy cập từ xa',
          tip: 'Sau khi bật SSH, hãy đặt mật khẩu mạnh cho người dùng root trong thiết bị đầu cuối.'
        },
        swap: {
          errors: {
            load: 'Không thể tải trạng thái bộ nhớ hoán đổi. Hãy thử lại sau ít phút.',
            update:
              'Không thể cập nhật cài đặt bộ nhớ hoán đổi. Hãy kiểm tra trạng thái hiện tại trước khi thử lại.'
          },
          title: 'Bộ nhớ swap',
          description: 'Bật bộ nhớ hoán đổi nén'
        },
        advanced: 'Cài đặt nâng cao',
        staticIp: {
          title: 'IP tĩnh',
          config: 'Cấu hình',
          tip: 'Để tránh mất kết nối, bạn nên định cấu hình khi có quyền truy cập vật lý vào thiết bị.',
          document: 'Kiểm tra tài liệu.',
          enable: 'Kích hoạt IP tĩnh',
          placeholder: 'Định dạng: IP/CIDR [Cổng]\nVí dụ: 192.168.1.10/22',
          notEmpty: 'Địa chỉ IP không được để trống.',
          notValid: 'Định dạng địa chỉ IP không hợp lệ.',
          failed: 'Cấu hình không thành công.',
          success: 'Cấu hình hoàn tất. Vui lòng kiểm tra xem địa chỉ IP có sẵn không.',
          okBtn: 'Được rồi',
          cancelBtn: 'Hủy'
        },
        usbNetwork: {
          title: 'Mạng USB',
          mode: 'Chế độ',
          server: 'Máy chủ',
          serverDesc: 'Cấp địa chỉ IP cho máy từ xa bằng máy chủ DHCP tích hợp.',
          client: 'Máy khách',
          clientDesc: 'Nhận địa chỉ IP từ máy từ xa qua DHCP.',
          forwarding: 'Chuyển tiếp IP',
          forwardingTip:
            'Để máy từ xa truy cập Internet thông qua NanoKVM, hãy bật chuyển tiếp IP.',
          manualIp: 'Địa chỉ IP thủ công',
          ipv4Placeholder: 'IPv4/CIDR, ví dụ 10.10.10.1/24',
          invalidIpv4: 'Nhập địa chỉ IPv4/CIDR hợp lệ.',
          loadFailed: 'Không thể tải cấu hình mạng USB.',
          failed: 'Cấu hình thất bại.',
          success: 'Đã hoàn tất cấu hình.',
          applyBtn: 'Áp dụng',
          cancelBtn: 'Hủy'
        },
        usbId: {
          title: 'Nhận dạng thiết bị USB',
          config: 'Cấu hình',
          tip: 'ID USB tùy chỉnh thay đổi cách máy chủ nhận diện thiết bị này. Áp dụng sẽ khởi động lại USB gadget và có thể ngắt bàn phím, chuột hoặc mạng trong giây lát.',
          enable: 'Bật nhận dạng thiết bị USB tùy chỉnh',
          vid: 'Vendor ID (VID)',
          pid: 'Product ID (PID)',
          bcdDevice: 'Phiên bản thiết bị (bcdDevice)',
          serialnumber: 'Số sê-ri',
          manufacturer: 'Nhà sản xuất',
          product: 'Sản phẩm',
          vidPlaceholder: 'vd. 0x359F',
          pidPlaceholder: 'vd. 0x3305',
          bcdDevicePlaceholder: 'vd. 0x0101 (tùy chọn)',
          serialnumberPlaceholder: 'vd. 0123456789ABCDEF (tùy chọn)',
          manufacturerPlaceholder: 'vd. sipeed (tùy chọn)',
          productPlaceholder: 'vd. NanoKVM-Go (tùy chọn)',
          notEmpty: 'VID và PID không được để trống.',
          notValid: 'Định dạng VID/PID không hợp lệ. Dùng hex như 0x359F.',
          bcdNotValid: 'Định dạng bcdDevice không hợp lệ. Dùng hex như 0x0101 hoặc để trống.',
          failed: 'Cấu hình thất bại.',
          success: 'Cấu hình hoàn tất. Đã khởi động lại USB gadget.',
          okBtn: 'OK',
          cancelBtn: 'Hủy'
        },
        diskInquiry: {
          title: 'Tên ổ USB',
          config: 'Cấu hình',
          tip: 'Tên USB flash ảo mà máy tính từ xa nhìn thấy. Lưu sẽ khởi động lại USB và có thể ngắt bàn phím, chuột hoặc mạng trong giây lát.',
          enable: 'Dùng tên ổ USB tùy chỉnh',
          placeholder: 'vd. sipeed  Flash Drive     1.00',
          notEmpty: 'Tên không được để trống.',
          notValid: 'Tên không hợp lệ. Tối đa 28 ký tự ASCII in được, không có xuống dòng.',
          failed: 'Cấu hình thất bại.',
          success: 'Cấu hình hoàn tất. Đã khởi động lại USB gadget.',
          okBtn: 'OK',
          cancelBtn: 'Hủy'
        },
        diskInquiryCdrom: {
          title: 'Tên ổ CD ảo',
          config: 'Cấu hình',
          tip: 'Tên ổ CD/DVD ảo mà máy tính từ xa nhìn thấy. Lưu sẽ khởi động lại USB và có thể ngắt bàn phím, chuột hoặc mạng trong giây lát.',
          enable: 'Dùng tên ổ CD ảo tùy chỉnh',
          placeholder: 'vd. sipeed  Optical Drive   1.00',
          notEmpty: 'Tên không được để trống.',
          notValid: 'Tên không hợp lệ. Tối đa 28 ký tự ASCII in được, không có xuống dòng.',
          failed: 'Cấu hình thất bại.',
          success: 'Cấu hình hoàn tất. Đã khởi động lại USB gadget.',
          okBtn: 'OK',
          cancelBtn: 'Hủy'
        },
        uac2FunctionName: {
          title: 'Tên âm thanh USB',
          config: 'Cấu hình',
          tip: 'Tên micro / thiết bị âm thanh USB ảo mà máy tính từ xa nhìn thấy. Lưu sẽ khởi động lại USB và có thể ngắt bàn phím, chuột hoặc mạng trong giây lát.',
          enable: 'Dùng tên âm thanh USB tùy chỉnh',
          placeholder: 'vd. NanoKVM-Go Audio',
          notEmpty: 'Tên không được để trống.',
          notValid: 'Tên không hợp lệ. Tối đa 64 ký tự in được, không có xuống dòng.',
          failed: 'Cấu hình thất bại.',
          success: 'Cấu hình hoàn tất. Đã khởi động lại USB gadget.',
          okBtn: 'OK',
          cancelBtn: 'Hủy'
        },
        lowPower: {
          title: 'Chế độ năng lượng thấp',
          description: 'Giảm mức sử dụng điện năng khi không hoạt động',
          tips: 'Lời khuyên',
          tip1: 'Khi được bật, quá trình thu video sẽ tạm dừng trong thời gian không hoạt động để giảm mức tiêu thụ điện năng và nhiệt.',
          tip2: 'Việc tiếp tục thu video có thể cần khởi tạo lại, trong thời gian đó video có thể tạm thời không khả dụng.',
          okBtn: 'Được rồi',
          cancelBtn: 'Hủy'
        },
        mouseJiggler: {
          title: 'Máy lắc lư chuột',
          description: 'Ngăn máy chủ từ xa ngủ',
          disable: 'Tắt',
          absolute: 'Chế độ tuyệt đối',
          relative: 'Chế độ tương đối'
        },
        fingerRobot: {
          title: 'Robot ngón tay',
          description: 'Bật điều khiển robot ngón tay'
        },
        mdns: {
          description: 'Kích hoạt dịch vụ khám phá mDNS'
        },
        hidOnly: 'HID-Chế độ chỉ',
        imageMounted: 'Đã gắn hình ảnh',
        disk: 'Ổ đĩa ảo',
        diskDesc: 'Gắn ổ U ảo trên máy chủ từ xa',
        sdCard: {
          warning: 'Khởi tạo ổ đĩa ảo',
          tip1: 'Trong lần bật đầu tiên, ổ đĩa ảo sẽ được khởi tạo và các tệp ảnh cùng dữ liệu ổ đĩa ảo hiện có trên thiết bị sẽ bị xóa. Thao tác này không thể hoàn tác. Hãy sao lưu các tệp quan trọng trước.',
          tip2: 'Tiếp tục khởi tạo?',
          capacity: 'Dung lượng',
          disable: 'Tắt',
          disableWarning: 'Tắt ổ đĩa ảo?',
          disableTip:
            'Việc tắt sẽ xóa tất cả tệp trên ổ đĩa ảo và giải phóng dung lượng lưu trữ thiết bị mà ổ đĩa đang sử dụng. Lần bật tiếp theo, bạn sẽ phải chọn lại dung lượng và khởi tạo. Thao tác này không thể hoàn tác.'
        },
        network: 'Mạng ảo',
        networkDesc: 'Gắn thẻ mạng ảo trên máy chủ từ xa',
        mic: 'Âm thanh ảo',
        micDesc: 'Chọn chế độ âm thanh ảo USB cho máy chủ từ xa',
        virtualAudioMode: {
          off: 'Tắt',
          mic: 'Chỉ micrô',
          duplex: 'Micrô + loa'
        },
        speaker: 'Loa ảo',
        speakerDesc:
          'Khi bật, tính năng này thay thế đầu ra âm thanh DP mặc định để hỗ trợ cuộc gọi từ xa với khả năng tương thích tốt hơn. Khi tắt, chỉ âm thanh đa phương tiện được chuyển tiếp, không thể chuyển tiếp âm thanh cuộc gọi.',
        micTip:
          'Do các hạn chế của điểm cuối USB, việc bật Mic ảo sẽ tắt Màn hình phụ USB.\nVà nên tắt tính năng này trong quá trình thao tác BIOS để đảm bảo khả năng tương thích tối ưu.',
        reboot: 'Khởi động lại',
        rebootDesc: 'Bạn có chắc chắn muốn khởi động lại NanoKVM không?',
        rebootToFlashing: 'Khởi động lại vào chế độ nạp firmware',
        rebootToFlashingButton: 'Khởi động lại',
        rebootToFlashingTitle: 'Khởi động lại vào chế độ nạp firmware?',
        rebootToFlashingDesc:
          'NanoKVM sẽ khởi động lại và chuyển sang chế độ nạp firmware qua USB. Hãy đảm bảo giao diện USB dùng để nạp firmware đã được kết nối và công cụ nạp firmware đã sẵn sàng.',
        datetime: {
          timezone: 'Múi giờ',
          datetime: 'Ngày giờ',
          dateAndTime: 'Ngày và giờ',
          synchronize: 'Đồng bộ hóa thời gian',
          lastSynchronization: 'Đồng bộ hóa lần cuối: ',
          notSynchronized: 'Thời gian hệ thống chưa được đồng bộ',
          syncNow: 'Đồng bộ hóa ngay',
          manualSet: 'Đặt thủ công',
          manualSetTitle: 'Đặt thời gian thủ công',
          manualSetDescription: 'Bạn sẽ cần đăng nhập lại sau khi thay đổi thời gian.',
          useBrowserTime: 'Dùng thời gian trình duyệt',
          setTime: 'Đặt thời gian',
          invalidTime: 'Nhập ngày và giờ hợp lệ.',
          setFailed: 'Không thể đặt thời gian hệ thống. Hãy thử lại.',
          setSucceeded: 'Đã cập nhật thời gian hệ thống. Vui lòng đăng nhập lại.'
        },
        dialBackground: {
          errors: {
            load: 'Không thể tải ảnh nền mặt số. Hãy thử lại sau ít phút.',
            upload: 'Không thể tải ảnh nền lên. Hãy đảm bảo tệp là ảnh BMP hợp lệ rồi thử lại.',
            switch: 'Không thể đổi nền mặt số. Hãy làm mới danh sách ảnh rồi thử lại.'
          },
          title: 'Nền mặt đồng hồ',
          description: 'Chọn ảnh BMP từ /kvmcomm/ui/bg/ làm nền mặt đồng hồ',
          upload: 'Tải lên',
          loading: 'Đang tải...',
          empty: 'Không tìm thấy ảnh BMP',
          current: 'Hiện tại',
          onlyBmp: 'Chỉ hỗ trợ ảnh BMP',
          uploaded: 'Tải lên hoàn tất',
          switched: 'Đã đổi nền'
        },
        okBtn: 'Có',
        cancelBtn: 'Không'
      },
      tailscale: {
        title: 'Tailscale',
        restart: 'Khởi động lại Tailscale?',
        stop: 'Dừng Tailscale?',
        stopDesc: 'Đăng xuất khỏi Tailscale và tắt tự động khởi động khi boot.',
        loading: 'Đang tải...',
        notInstall: 'Không tìm thấy Tailscale! Vui lòng cài đặt.',
        install: 'Cài đặt',
        installing: 'Đang cài đặt',
        failed: 'Cài đặt thất bại',
        retry: 'Vui lòng làm mới và thử lại. Hoặc thử cài đặt thủ công',
        download: 'Tải xuống',
        package: 'gói cài đặt',
        unzip: 'và giải nén nó',
        upTailscale: 'Tải tailscale lên thư mục /usr/bin/ của NanoKVM',
        upTailscaled: 'Tải tailscaled lên thư mục /usr/sbin/ của NanoKVM',
        refresh: 'Làm mới trang hiện tại',
        notRunning: 'Tailscale không chạy. Hãy bắt đầu nó để tiếp tục.',
        run: 'Khởi động',
        notLogin:
          'Thiết bị chưa được liên kết. Vui lòng đăng nhập và liên kết thiết bị này với tài khoản của bạn.',
        urlPeriod: 'URL này có hiệu lực trong 10 phút',
        login: 'Đăng nhập',
        loginSuccess: 'Đăng nhập thành công',
        enable: 'Kích hoạt Tailscale',
        exitNode: 'Nút thoát',
        exitNodeDesc: 'Quảng bá thiết bị này làm lối ra Internet cho tailnet của bạn.',
        subnetRoute: 'Tuyến mạng con',
        subnetRouteDesc: 'Quảng bá mạng LAN kết nối trực tiếp {{subnet}} vào tailnet của bạn.',
        subnetUnavailable: 'Không có mạng LAN kết nối trực tiếp nào khả dụng.',
        routingUnavailable: 'Thiết bị này không hỗ trợ cài đặt định tuyến Tailscale.',
        pendingApproval:
          'Sau khi bật, hãy phê duyệt tuyến được quảng bá trong bảng điều khiển quản trị Tailscale.',
        routingUpdateFailed: 'Không thể cập nhật cài đặt định tuyến Tailscale.',
        deviceName: 'Tên Thiết bị',
        deviceIP: 'IP Thiết bị',
        account: 'Tài khoản',
        logout: 'Đăng xuất',
        logoutDesc: 'Bạn có chắc chắn muốn đăng xuất không?',
        uninstall: 'Gỡ cài đặt Tailscale',
        uninstallDesc: 'Bạn có chắc chắn muốn gỡ cài đặt Tailscale không?',
        okBtn: 'Có',
        cancelBtn: 'Không'
      },
      update: {
        title: 'Kiểm tra cập nhật',
        queryFailed: 'Lấy phiên bản thất bại',
        updateFailed: 'Cập nhật thất bại. Vui lòng thử lại.',
        isLatest: 'Bạn đã có phiên bản mới nhất.',
        available: 'Có bản cập nhật mới. Bạn có chắc chắn muốn cập nhật không?',
        updating: 'Bắt đầu cập nhật. Vui lòng chờ...',
        confirm: 'Xác nhận',
        upgrade_tip_nano:
          'Đã có phiên bản mới của NanoKVM. Bạn có chắc chắn muốn cập nhật ngay bây giờ không?',
        upgrade_tip_firmware:
          'Đã có phiên bản phần sụn mới. Bạn có chắc chắn muốn cập nhật ngay bây giờ không?',
        cancel: 'Hủy',
        preview: 'Bản cập nhật xem trước',
        previewDesc: 'Nhận quyền truy cập sớm vào các tính năng và cải tiến mới',
        previewTip:
          'Xin lưu ý rằng các bản phát hành xem trước có thể có lỗi hoặc chức năng chưa hoàn chỉnh!',
        download: 'Tải xuống',
        install: 'Cài đặt',
        changelog: 'Nhật ký thay đổi',
        restart: 'Khởi động lại'
      },
      account: {
        title: 'Tài khoản',
        webAccount: 'Tên tài khoản web',
        password: 'Mật khẩu',
        twoFactor: 'Xác thực hai yếu tố',
        enableTwoFactor: 'Bật',
        enableTwoFactorDesc: 'Nhập mật khẩu hiện tại để bắt đầu thiết lập xác thực hai yếu tố.',
        disableTwoFactor: 'Tắt',
        recoveryCodes: 'Mã khôi phục',
        enableRecoveryCodes: 'Bật',
        updateRecoveryCodes: 'Cập nhật',
        recoveryCodesTooltipDisabled:
          'Mã khôi phục là thông tin dự phòng cho xác thực hai yếu tố. Bạn có thể tạo chúng sau khi bật xác thực hai yếu tố.',
        recoveryCodesTooltipEnabled:
          'Mã khôi phục chỉ hợp lệ khi xác thực hai yếu tố đang bật. Tắt xác thực hai yếu tố sẽ vô hiệu hóa tất cả mã khôi phục; khi vẫn bật, chúng chỉ có thể được cập nhật.',
        recoveryCodesRegenerateDesc:
          'Nhập mật khẩu hiện tại và một trong hai lựa chọn: mã xác thực hoặc mã khôi phục chưa dùng. Cập nhật mã khôi phục sẽ vô hiệu hóa tất cả mã khôi phục hiện có.',
        recoveryCodesVerifyFailed: 'Không thể xác minh mật khẩu và mã xác thực hoặc mã khôi phục.',
        recoveryCodesSavedTitle: 'Lưu các mã khôi phục này ngay',
        recoveryCodesSavedDesc:
          'Mã khôi phục chỉ có thể xem một lần. Hãy lưu chúng ở nơi an toàn trước khi rời trang này.',
        enableRecoveryCodesDesc:
          'Nhập mật khẩu hiện tại và mã xác thực để tạo mã khôi phục. Mã khôi phục chỉ có thể xem một lần, vì vậy hãy lưu chúng trước khi rời trang này.',
        copyRecoveryCodes: 'Sao chép',
        downloadRecoveryCodes: 'Tải xuống',
        recoveryCodesCopyFailed: 'Không thể sao chép mã khôi phục.',
        done: 'Xong',
        currentPassword: 'Mật khẩu hiện tại',
        authCode: 'Mã xác thực',
        authCodeOrRecoveryCode: 'Mã xác thực hoặc mã khôi phục',
        authRateLimited: 'Quá nhiều lần thử thất bại. Thử lại sau {{seconds}} giây.',
        verifyPasswordChange: 'Xác minh đổi mật khẩu',
        verifyPasswordChangeDesc:
          'Nhập mật khẩu hiện tại và một trong hai lựa chọn: mã xác thực hoặc mã khôi phục chưa dùng trước khi đổi mật khẩu.',
        verifyPasswordChangeFailed: 'Không thể xác minh mật khẩu và mã xác thực hoặc mã khôi phục.',
        verifyAndEnable: 'Xác minh và bật',
        scanTwoFactorQRCode: 'Quét mã QR này bằng ứng dụng xác thực, sau đó nhập mã.',
        showSetupKey: 'Hiển thị khóa thiết lập',
        hideSetupKey: 'Ẩn khóa thiết lập',
        setupKey: 'Khóa thiết lập',
        twoFactorSetupFailed: 'Không thể bắt đầu thiết lập xác thực hai yếu tố.',
        twoFactorVerifyFailed: 'Không thể xác minh mã xác thực.',
        twoFactorDisableFailed: 'Không thể tắt xác thực hai yếu tố.',
        disableTwoFactorDesc:
          'Nhập mật khẩu hiện tại và một trong hai lựa chọn: mã xác thực hoặc mã khôi phục chưa dùng. Tắt xác thực hai yếu tố sẽ vô hiệu hóa tất cả mã khôi phục.',
        updateBtn: 'Thay đổi',
        logoutBtn: 'Đăng xuất',
        logoutDesc: 'Bạn có chắc chắn muốn đăng xuất không?',
        okBtn: 'Có',
        cancelBtn: 'Không',
        twoFactorProtection: {
          title: 'Bảo vệ 2FA nâng cao',
          description:
            'Chọn các thao tác nhạy cảm cần yếu tố thứ hai khi việc bảo vệ được áp dụng. Công tắc "Đăng nhập web trực tiếp" làm điều ngược lại: cho phép đăng nhập chỉ bằng mã xác thực hoặc mã khôi phục chưa dùng.',
          login: 'Đăng nhập web trực tiếp',
          webTerminal: 'Terminal web',
          ssh: 'SSH',
          tailscale: 'Tailscale',
          mcp: 'Khóa API MCP',
          targetPower: 'Nguồn máy đích',
          nanoKVMRestart: 'Khởi động lại NanoKVM',
          deleteImage: 'Xóa image',
          scripts: 'Thao tác tập lệnh',
          screenTimelapse: 'Tua nhanh màn hình',
          loginScope:
            'Cho phép đăng nhập web chỉ bằng mã xác thực hoặc mã khôi phục chưa dùng thay vì tên người dùng và mật khẩu. Điều này làm giảm bảo mật đăng nhập.',
          webTerminalScope: 'Mở Terminal web.',
          sshScope: 'Thay đổi trạng thái dịch vụ SSH.',
          tailscaleScope: 'Cài đặt, kết nối hoặc thay đổi Tailscale.',
          frp: 'FRP',
          frpScope: 'Cài đặt, kết nối hoặc thay đổi FRP.',
          cloudflare: 'Cloudflare Tunnel',
          zerotier: 'ZeroTier',
          netbird: 'NetBird',
          cloudflareScope: 'Cài đặt, kết nối hoặc thay đổi Cloudflare Tunnel.',
          zerotierScope: 'Cài đặt, kết nối hoặc thay đổi ZeroTier.',
          netbirdScope: 'Cài đặt, kết nối hoặc thay đổi NetBird.',
          mcpScope: 'Hiển thị, sao chép hoặc tạo lại khóa API MCP, và bật hoặc tắt MCP.',
          targetPowerScope: 'Tắt hoặc khởi động lại máy đích bằng nút nguồn vật lý.',
          nanoKVMRestartScope: 'Khởi động lại chính NanoKVM.',
          deleteImageScope: 'Xóa vĩnh viễn image ISO hoặc IMG đã lưu.',
          scriptsScope: 'Chạy, tải lên hoặc xóa tập lệnh.',
          screenTimelapseScope: 'Bật ghi tua nhanh màn hình, xem bản ghi và xóa bản ghi.',
          authorizeTitle: 'Cho phép thay đổi bảo vệ',
          authorizeDesc:
            'Nhập mã xác thực hoặc mã khôi phục chưa dùng. Quyền này có hiệu lực trong năm phút.',
          authorizeFailed: 'Không thể xác minh mã xác thực hoặc mã khôi phục.',
          loadFailed: 'Không thể tải cài đặt bảo vệ.',
          updateFailed: 'Không thể lưu cài đặt bảo vệ.',
          retry: 'Thử lại',
          disableScriptsWarningTitle: 'Tắt bảo vệ tập lệnh?',
          disableScriptsWarningDesc:
            'Không có bảo vệ, tập lệnh độc hại có thể có được quyền truy cập nâng cao vào thiết bị này và bỏ qua hoặc xóa bỏ bảo vệ 2FA.',
          cancelDisabling: 'Hủy',
          continueDisabling: 'Tiếp tục tắt',
          continueDisablingCountdown: 'Tiếp tục tắt ({{seconds}} giây)'
        },
        twoFactorGuide: {
          title: 'Hướng dẫn xác thực hai yếu tố',
          scope: 'Xác thực hai yếu tố thêm một lớp bảo mật bổ sung cho tài khoản của bạn.',
          whatTitle: '2FA là gì?',
          whatDesc:
            'Xác thực hai yếu tố (2FA) bảo vệ tài khoản của bạn bằng mã theo thời gian từ ứng dụng xác thực. Theo mặc định, mã này được yêu cầu ngoài mật khẩu, vì vậy ngay cả khi mật khẩu bị lộ, tài khoản vẫn được bảo vệ. Nếu bật đăng nhập web trực tiếp, chỉ cần mã là đủ để đăng nhập.',
          appsTitle: 'Ứng dụng được đề xuất',
          appsDesc:
            'Bạn có thể dùng Google Authenticator hoặc Microsoft Authenticator. Cả hai đều miễn phí và hoạt động trên iOS và Android.',
          setupTitle: 'Cách thiết lập',
          setupDesc:
            'Mở ứng dụng xác thực, quét mã QR hiển thị trong quá trình thiết lập hoặc nhập khóa thiết lập thủ công. Sau đó nhập mã 6 chữ số từ ứng dụng để hoàn tất bật 2FA.',
          recoveryTitle: 'Mã khôi phục',
          recoveryDesc:
            'Mã khôi phục là mã dự phòng dùng một lần. Nếu mất quyền truy cập vào ứng dụng xác thực, bạn có thể dùng mã khôi phục để đăng nhập. Hãy lưu chúng ở nơi an toàn ngay sau khi tạo.',
          logoutNotice:
            'Bật 2FA sẽ đăng xuất tất cả phiên đang hoạt động, bao gồm phiên này. Bạn sẽ cần đăng nhập lại trên mọi thiết bị.'
        }
      }
    },
    error: {
      title: 'Chúng tôi đã gặp sự cố',
      refresh: 'Làm mới'
    },
    pictureInPicture: {
      title: 'Ảnh trong ảnh',
      enter: 'Bật ảnh trong ảnh',
      exit: 'Tắt ảnh trong ảnh',
      error: 'Không thể bắt đầu ảnh trong ảnh. Vui lòng đợi video tải xong rồi thử lại.'
    },
    fullscreen: {
      toggle: 'Chuyển đổi toàn màn hình'
    },
    menu: {
      collapse: 'Thu gọn Menu',
      expand: 'Mở rộng Menu'
    }
  }
};

export default vi;
