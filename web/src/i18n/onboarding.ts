import type { SupportedLanguage } from './locale.ts';

/**
 * Strings owned by the first-run onboarding tour (soft prompt, welcome card,
 * one step per menu-bar group, navigation, finish card, settings replay row). Each step
 * carries one line per rail item so the tour engine can compose a step from
 * only the items actually visible (many items are gated by server config or
 * video mode). Item names mirror the translated menu labels in locales/, the
 * portal module, the screen-timelapse service label, and the finger-robot
 * text module (the rail's "power" key renders the Finger Robot), so the tour
 * names what the user actually sees.
 */
export type OnboardingMessages = {
  prompt: {
    title: string;
    description: string;
    start: string;
    later: string;
    never: string;
    language: string;
  };
  welcome: {
    title: string;
    description: string;
  };
  steps: {
    display: {
      title: string;
      items: { screen: string; portal: string; pictureInPicture: string };
    };
    media: {
      title: string;
      items: { volume: string; microphone: string };
    };
    input: {
      title: string;
      items: { mouseJiggler: string; keyboard: string; mouse: string; preview: string };
    };
    tools: {
      title: string;
      items: { image: string; script: string; terminal: string; wol: string };
    };
    automation: {
      title: string;
      items: { screenTimelapse: string; power: string };
    };
    system: {
      title: string;
      items: { settings: string; fullscreen: string; collapse: string };
    };
  };
  finish: {
    title: string;
    description: string;
  };
  replay: {
    label: string;
    description: string;
    action: string;
  };
  tour: { prev: string; next: string; end: string };
  /** Mobile portrait tour additions. The mobile flow reuses the root-level
   * prompt buttons, group steps (steps.<group>.*), and finish strings; only
   * rail-specific copy and the tour button labels live here. The tour labels
   * are owned (instead of the root sentence-style skip and antd's locale
   * Prev/Next/Finish) because the 268px footer row cannot fit strings like
   * fr "Étape précédente" or da "Spring rundvisningen over". */
  mobile: {
    prompt: { title: string; description: string };
    welcome: { title: string; description: string };
    steps: { rail: { title: string; description: string } };
    tour: { skip: string; prev: string; next: string; end: string };
  };
  skip: string;
};

const messages: Record<SupportedLanguage, OnboardingMessages> = {
  en: {
    prompt: {
      title: 'New to NanoKVM?',
      description: 'Take a one-minute tour of the menu bar.',
      start: 'Start tour',
      later: 'Later',
      never: "Don't show again",
      language: 'Language'
    },
    welcome: {
      title: 'Meet the menu bar',
      description:
        'This bar holds every control for the remote machine. Drag it by the grip to move it; after a few idle seconds it hides, and it returns when you move the pointer to the top edge. While the tour runs, your keyboard and mouse are not forwarded to the remote machine.'
    },
    steps: {
      display: {
        title: 'Display',
        items: {
          screen: '"Screen" — click to toggle view-only mode. Open its menu for display settings.',
          portal: '"Portal" — crop the view to the area that matters.',
          pictureInPicture:
            '"Picture-in-Picture" — keep the desktop in a floating window while you use other apps.'
        }
      },
      media: {
        title: 'Audio',
        items: {
          volume: '"Volume" — play the remote machine\'s audio and control the level.',
          microphone: '"Microphone" — send your voice to the remote machine.'
        }
      },
      input: {
        title: 'Keyboard and mouse',
        items: {
          mouseJiggler: '"Mouse Jiggler" — Prevent the remote host from sleeping',
          keyboard: '"Keyboard" — on-screen keys, shortcuts, and paste-as-typing.',
          mouse: '"Mouse" — switch pointer modes and calibrate.',
          preview: '"Preview" — watch the screen without sending any input.'
        }
      },
      tools: {
        title: 'Remote tools',
        items: {
          image: '"Images" — mount a virtual disk on the remote machine.',
          script: '"Scripts" — run your own code on this KVM.',
          terminal: '"Terminal" — open a console on the KVM itself.',
          wol: '"Wake-on-LAN" — power up other machines on the network.'
        }
      },
      automation: {
        title: 'Automation',
        items: {
          screenTimelapse: '"Screen Timelapse" — record the screen and review what happened later.',
          power: '"Finger Robot" — press the remote machine\'s physical power button.'
        }
      },
      system: {
        title: 'Settings and window',
        items: {
          settings: '"Settings" — device, network, security, and updates.',
          fullscreen: '"Toggle Fullscreen" — fill your display with the remote screen.',
          collapse: '"Collapse Menu" — fold the bar into a small handle.'
        }
      }
    },
    finish: {
      title: "You're all set",
      description:
        "That's the whole menu bar. Replay it anytime from Settings, on the Appearance page."
    },
    replay: {
      label: 'Menu tour',
      description: 'Replay the introduction to the menu bar.',
      action: 'Replay'
    },
    tour: {
      prev: 'Previous',
      next: 'Next',
      end: 'End tour'
    },
    mobile: {
      prompt: {
        title: 'New to NanoKVM?',
        description: 'Take a one-minute tour of the menu rail.'
      },
      welcome: {
        title: 'Meet the menu rail',
        description: 'This floating bar holds every control for the remote machine. Drag it by the grip to park it anywhere on the edge, and collapse it when you need the full screen. While the tour runs, your touches are not sent to the remote machine.'
      },
      steps: {
        rail: {
          title: 'The menu rail',
          description: 'Tap the grip to open tools and settings. The tour spotlights each group next.'
        }
      },
      tour: {
        skip: 'Skip',
        prev: 'Previous',
        next: 'Next',
        end: 'End tour'
      }
    },
    skip: 'Skip tour'
  },
  zh: {
    prompt: {
      title: '第一次使用 NanoKVM？',
      description: '花一分钟了解顶部菜单栏。',
      start: '开始导览',
      later: '稍后',
      never: '不再显示',
      language: '语言'
    },
    welcome: {
      title: '认识菜单栏',
      description:
        '这是远程机器的全部控制入口。拖动手柄即可移动；空闲几秒后自动隐藏，把指针移到屏幕顶边即可唤回。导览期间，你的键盘和鼠标不会发送到远程机器。'
    },
    steps: {
      display: {
        title: '画面',
        items: {
          screen: '「屏幕」调整分辨率、画质和帧率。',
          portal: '「去黑边」把画面裁剪到需要的区域。',
          pictureInPicture: '「画中画」把远程桌面浮成小窗，可同时使用其他应用。'
        }
      },
      media: {
        title: '声音',
        items: {
          volume: '「音量」播放远程机器的声音并调节大小。',
          microphone: '「麦克风」把你的声音传输到远程机器。'
        }
      },
      input: {
        title: '键盘与鼠标',
        items: {
          mouseJiggler: '「鼠标抖动」 — 防止远程主机休眠',
          keyboard: '「键盘」提供屏幕键盘、快捷键和粘贴输入。',
          mouse: '「鼠标」切换指针模式并校准。',
          preview: '「预览」只看画面，不发送任何输入。'
        }
      },
      tools: {
        title: '远程工具',
        items: {
          image: '「镜像」为远程机器挂载虚拟磁盘。',
          script: '「脚本」在这台 KVM 上运行你的代码。',
          terminal: '「终端」打开 KVM 控制台。',
          wol: '「网络唤醒」开启网络中的其他机器。'
        }
      },
      automation: {
        title: '自动化',
        items: {
          screenTimelapse: '「屏幕延时回放」录制屏幕，便于事后查看。',
          power: '「手指机器人」替你按远程机器的实体电源键。'
        }
      },
      system: {
        title: '设置与窗口',
        items: {
          settings: '「设置」包含设备、网络、安全与更新。',
          fullscreen: '「切换全屏」让远程画面铺满显示器。',
          collapse: '「收起」把菜单栏收成小手柄。'
        }
      }
    },
    finish: {
      title: '导览完成',
      description: '以上是菜单栏的全部内容。随时可以在「设置」的「外观」页面重新观看导览。'
    },
    replay: {
      label: '菜单导览',
      description: '重新播放菜单栏导览。',
      action: '重新观看'
    },
    tour: {
      prev: '上一步',
      next: '下一步',
      end: '结束导览'
    },
    mobile: {
      prompt: {
        title: '第一次使用 NanoKVM？',
        description: '花一分钟了解侧边菜单栏。'
      },
      welcome: {
        title: '认识菜单栏',
        description: '这个悬浮栏是远程机器的全部控制入口。拖动手柄可把它停在屏幕边缘任意位置，需要完整画面时可将其收起。导览期间，你的触摸操作不会发送到远程机器。'
      },
      steps: {
        rail: {
          title: '菜单栏',
          description: '点击手柄即可打开工具与设置，接下来会逐个介绍各功能分组。'
        }
      },
      tour: {
        skip: '跳过',
        prev: '上一步',
        next: '下一步',
        end: '结束导览'
      }
    },
    skip: '跳过导览'
  },
  zh_tw: {
    prompt: {
      title: '第一次使用 NanoKVM？',
      description: '花一分鐘認識頂端選單列。',
      start: '開始導覽',
      later: '稍後',
      never: '不再顯示',
      language: '語言'
    },
    welcome: {
      title: '認識選單列',
      description:
        '這是遠端機器的全部控制入口。拖曳握把即可移動；閒置幾秒後自動隱藏，把指標移到螢幕頂緣即可喚回。導覽期間，你的鍵盤和滑鼠不會傳送到遠端機器。'
    },
    steps: {
      display: {
        title: '畫面',
        items: {
          screen: '「螢幕」調整解析度、畫質與幀率。',
          portal: '「去黑邊」把畫面裁剪到需要的區域。',
          pictureInPicture: '「子母畫面」讓遠端桌面浮成小窗，可同時使用其他應用程式。'
        }
      },
      media: {
        title: '聲音',
        items: {
          volume: '「音量」播放遠端機器的聲音並調整大小。',
          microphone: '「麥克風」把你的聲音傳輸到遠端機器。'
        }
      },
      input: {
        title: '鍵盤與滑鼠',
        items: {
          mouseJiggler: '「滑鼠抖動模式 (Mouse Jiggler)」 — 避免遠端主機進入休眠狀態',
          keyboard: '「鍵盤」提供螢幕鍵盤、快速鍵與貼上輸入。',
          mouse: '「滑鼠」切換指標模式並校準。',
          preview: '「預覽」只看畫面，不傳送任何輸入。'
        }
      },
      tools: {
        title: '遠端工具',
        items: {
          image: '「映像」為遠端機器掛載虛擬磁碟。',
          script: '「腳本」在這台 KVM 上執行你的程式碼。',
          terminal: '「終端機」開啟 KVM 控制台。',
          wol: '「網路喚醒」開啟網路中的其他機器。'
        }
      },
      automation: {
        title: '自動化',
        items: {
          screenTimelapse: '「螢幕延時回放」錄製螢幕，方便事後查看。',
          power: '「手指機器人」替你按遠端機器的實體電源鍵。'
        }
      },
      system: {
        title: '設定與視窗',
        items: {
          settings: '「設定」包含裝置、網路、安全與更新。',
          fullscreen: '「進入全螢幕模式」讓遠端畫面鋪滿顯示器。',
          collapse: '「收起選單」把選單列收成小握把。'
        }
      }
    },
    finish: {
      title: '導覽完成',
      description: '以上是選單列的全部內容。之後隨時可在「設定」的「外觀」頁面重新觀看導覽。'
    },
    replay: {
      label: '選單導覽',
      description: '重新播放選單列導覽。',
      action: '重新觀看'
    },
    tour: {
      prev: '上一步',
      next: '下一步',
      end: '結束導覽'
    },
    mobile: {
      prompt: {
        title: '第一次使用 NanoKVM？',
        description: '花一分鐘認識側邊選單列。'
      },
      welcome: {
        title: '認識選單列',
        description: '這個懸浮列是遠端機器的全部控制入口。拖曳握把可停靠在螢幕邊緣任意位置，需要完整畫面時可將其收起。導覽期間，你的觸控操作不會傳送到遠端機器。'
      },
      steps: {
        rail: {
          title: '選單列',
          description: '點擊握把即可開啟工具與設定，接下來會逐一介紹各功能分組。'
        }
      },
      tour: {
        skip: '跳過',
        prev: '上一步',
        next: '下一步',
        end: '結束導覽'
      }
    },
    skip: '略過導覽'
  },
  ja: {
    prompt: {
      title: 'NanoKVM は初めてですか？',
      description: '画面上部のメニューバーを 1 分で紹介します。',
      start: 'ツアーを開始',
      later: 'あとで',
      never: '今後表示しない',
      language: '言語'
    },
    welcome: {
      title: 'メニューバーの紹介',
      description:
        '遠隔マシンのすべての操作はこのバーから行えます。グリップをドラッグして移動でき、数秒操作がないと自動で隠れ、画面上端にポインターを移動すると再表示されます。ツアー中、キーボードとマウスは遠隔マシンに転送されません。'
    },
    steps: {
      display: {
        title: '画面',
        items: {
          screen: '「画面」— 解像度・品質・フレームレートを調整します。',
          portal: '「黒枠の除去」— 表示範囲を必要な領域に切り抜きます。',
          pictureInPicture:
            '「ピクチャー イン ピクチャー」— デスクトップを小窓に浮かべたまま他の作業ができます。'
        }
      },
      media: {
        title: '音声',
        items: {
          volume: '「音量」— 遠隔マシンの音声を再生し、音量を調整します。',
          microphone: '「マイク」— あなたの声を遠隔マシンへ送ります。'
        }
      },
      input: {
        title: 'キーボードとマウス',
        items: {
          mouseJiggler: '「マウスジグラー」 — リモートホストの休止を防ぐ',
          keyboard: '「キーボード」— 画面上のキー、ショートカット、貼り付け入力。',
          mouse: '「マウス」— ポインターモードを切り替え、キャリブレーションを行います。',
          preview: '「プレビュー」— 入力を送らずに画面だけを表示します。'
        }
      },
      tools: {
        title: 'リモートツール',
        items: {
          image: '「イメージ」— 遠隔マシンに仮想ディスクをマウントします。',
          script: '「スクリプト」— この KVM 上で独自のコードを実行します。',
          terminal: '「ターミナル」— KVM のコンソールを開きます。',
          wol: '「Wake-on-LAN」— ネットワーク上の他のマシンを起動します。'
        }
      },
      automation: {
        title: '自動化',
        items: {
          screenTimelapse: '「画面タイムラプス」— 画面を録画して後から確認できます。',
          power: '「指ロボット」— 遠隔マシンの物理電源ボタンを押します。'
        }
      },
      system: {
        title: '設定とウィンドウ',
        items: {
          settings: '「設定」— デバイス、ネットワーク、セキュリティ、更新。',
          fullscreen: '「全画面表示切り替え」— 遠隔画面をディスプレイいっぱいに広げます。',
          collapse: '「メニューを折りたたむ」— バーを小さなハンドルに収めます。'
        }
      }
    },
    finish: {
      title: '準備完了',
      description: 'メニューバーは以上です。「設定」の「外観」ページからいつでも再生できます。'
    },
    replay: {
      label: 'メニューツアー',
      description: 'メニューバーの紹介を再生します。',
      action: '再生'
    },
    tour: {
      prev: '戻る',
      next: '次へ',
      end: 'ツアーを終了'
    },
    mobile: {
      prompt: {
        title: 'NanoKVM は初めてですか？',
        description: '画面端のメニューレールを 1 分で紹介します。'
      },
      welcome: {
        title: 'メニューレールの紹介',
        description: 'このフローティングバーにはリモートマシンのすべてのコントロールが集約されています。グリップをドラッグして画面端の好きな位置に固定でき、全画面が必要なときは折りたためます。ツアー中、タッチ操作はリモートマシンに送信されません。'
      },
      steps: {
        rail: {
          title: 'メニューレール',
          description: 'グリップをタップしてツールと設定を開きます。次に各グループを順番に紹介します。'
        }
      },
      tour: {
        skip: 'スキップ',
        prev: '戻る',
        next: '次へ',
        end: 'ツアーを終了'
      }
    },
    skip: 'ツアーをスキップ'
  },
  ko: {
    prompt: {
      title: 'NanoKVM이 처음이신가요?',
      description: '화면 상단의 메뉴 바를 1분 만에 둘러보세요.',
      start: '투어 시작',
      later: '나중에',
      never: '다시 보지 않기',
      language: '언어'
    },
    welcome: {
      title: '메뉴 바 소개',
      description:
        '원격 기기의 모든 제어는 이 막대에 있습니다. 손잡이를 끌어 위치를 옮길 수 있고, 몇 초 동안 사용하지 않으면 자동으로 숨으며 포인터를 화면 상단 가장자리로 옮기면 다시 나타납니다. 투어 중에는 키보드와 마우스가 원격 기기로 전달되지 않습니다.'
    },
    steps: {
      display: {
        title: '화면',
        items: {
          screen: '"화면" — 해상도, 화질, 프레임 레이트를 조정합니다.',
          portal: '"검은 테두리 제거" — 보이는 영역을 필요한 부분으로 자릅니다.',
          pictureInPicture:
            '"그림 속 그림" — 원격 바탕화면을 작은 창으로 띄워 다른 작업과 병행합니다.'
        }
      },
      media: {
        title: '사운드',
        items: {
          volume: '"볼륨" — 원격 기기의 소리를 재생하고 크기를 조절합니다.',
          microphone: '"마이크" — 내 목소리를 원격 기기로 보냅니다.'
        }
      },
      input: {
        title: '키보드와 마우스',
        items: {
          mouseJiggler: '"마우스 흔들기" — 원격 호스트가 절전 모드로 진입하는 것을 방지',
          keyboard: '"키보드" — 화면에 표시되는 키, 단축키, 붙여넣기 입력을 제공합니다.',
          mouse: '"마우스" — 포인터 모드를 전환하고 보정합니다.',
          preview: '"미리보기" — 입력을 보내지 않고 화면만 봅니다.'
        }
      },
      tools: {
        title: '원격 도구',
        items: {
          image: '"이미지" — 원격 기기에 가상 디스크를 마운트합니다.',
          script: '"스크립트" — 이 KVM에서 직접 작성한 코드를 실행합니다.',
          terminal: '"터미널" — KVM 콘솔을 엽니다.',
          wol: '"Wake-on-LAN" — 네트워크의 다른 기기를 켭니다.'
        }
      },
      automation: {
        title: '자동화',
        items: {
          screenTimelapse: '"화면 타임랩스" — 화면을 녹화해 나중에 확인합니다.',
          power: '"손가락 로봇" — 원격 기기의 실제 전원 버튼을 눌러줍니다.'
        }
      },
      system: {
        title: '설정과 창',
        items: {
          settings: '"설정" — 기기, 네트워크, 보안, 업데이트.',
          fullscreen: '"전체 화면 전환" — 원격 화면을 디스플레이에 가득 채웁니다.',
          collapse: '"메뉴 접기" — 막대를 작은 손잡이로 접습니다.'
        }
      }
    },
    finish: {
      title: '준비 완료',
      description:
        '메뉴 바는 여기까지입니다. "설정"의 "디자인" 페이지에서 언제든 다시 볼 수 있습니다.'
    },
    replay: {
      label: '메뉴 투어',
      description: '메뉴 바 소개를 다시 재생합니다.',
      action: '다시 보기'
    },
    tour: {
      prev: '이전',
      next: '다음',
      end: '투어 종료'
    },
    mobile: {
      prompt: {
        title: 'NanoKVM이 처음이신가요?',
        description: '화면 가장자리의 메뉴 레일을 1분 만에 둘러보세요.'
      },
      welcome: {
        title: '메뉴 레일 살펴보기',
        description: '이 플로팅 바에는 원격 기기의 모든 제어 기능이 들어 있습니다. 그립을 드래그해 화면 가장자리 어디든 고정할 수 있고, 전체 화면이 필요하면 접을 수 있습니다. 투어가 진행되는 동안 터치 입력은 원격 기기로 전송되지 않습니다.'
      },
      steps: {
        rail: {
          title: '메뉴 레일',
          description: '그립을 탭하면 도구와 설정이 열립니다. 이어서 각 그룹을 차례로 소개합니다.'
        }
      },
      tour: {
        skip: '건너뛰기',
        prev: '이전',
        next: '다음',
        end: '투어 종료'
      }
    },
    skip: '투어 건너뛰기'
  },
  de: {
    prompt: {
      title: 'Neu bei NanoKVM?',
      description: 'Machen Sie in einer Minute eine Tour durch die Menüleiste.',
      start: 'Tour starten',
      later: 'Später',
      never: 'Nicht mehr anzeigen',
      language: 'Sprache'
    },
    welcome: {
      title: 'Die Menüleiste im Überblick',
      description:
        'Auf dieser Leiste liegen alle Steuerungen für die ferngesteuerte Maschine. Ziehen Sie sie am Griff an eine andere Position; nach wenigen Sekunden ohne Eingabe blendet sie sich aus und kehrt zurück, sobald der Zeiger den oberen Rand berührt. Während der Tour werden Tastatur und Maus nicht an die ferngesteuerte Maschine weitergeleitet.'
    },
    steps: {
      display: {
        title: 'Anzeige',
        items: {
          screen: '„Bildschirm“ — Auflösung, Qualität und Bildrate einstellen.',
          portal:
            '„Schwarze Ränder entfernen“ — die Ansicht auf den relevanten Bereich zuschneiden.',
          pictureInPicture:
            '„Bild-im-Bild“ — den entfernten Desktop in einem kleinen schwebenden Fenster behalten, während Sie andere Anwendungen nutzen.'
        }
      },
      media: {
        title: 'Ton',
        items: {
          volume: '„Lautstärke“ — den Ton der ferngesteuerten Maschine abspielen und regeln.',
          microphone: '„Mikrofon“ — Ihre Stimme an die ferngesteuerte Maschine übertragen.'
        }
      },
      input: {
        title: 'Tastatur und Maus',
        items: {
          mouseJiggler: '„Mausaktivitäts-Simulator“ — Verhindert, dass der remote Host in den Energiesparmodus wechselt',
          keyboard: '„Tastatur“ — Bildschirmtastatur, Kürzel und Einfügen als Tastatureingabe.',
          mouse: '„Maus“ — Zeigermodi wechseln und kalibrieren.',
          preview: '„Vorschau“ — das Bild anzeigen, ohne Eingaben zu senden.'
        }
      },
      tools: {
        title: 'Remote-Werkzeuge',
        items: {
          image: '„Abbilder“ — ein virtuelles Laufwerk auf der ferngesteuerten Maschine einbinden.',
          script: '„Skripte“ — eigenen Code auf diesem KVM ausführen.',
          terminal: '„Terminal“ — eine Konsole auf dem KVM öffnen.',
          wol: '„Wake-on-LAN“ — weitere Maschinen im Netzwerk starten.'
        }
      },
      automation: {
        title: 'Automatisierung',
        items: {
          screenTimelapse:
            '„Bildschirm-Zeitraffer“ — den Bildschirm aufzeichnen, um später nachsehen zu können.',
          power:
            '„Fingerroboter“ — den physischen Netzschalter der ferngesteuerten Maschine drücken.'
        }
      },
      system: {
        title: 'Einstellungen und Fenster',
        items: {
          settings: '„Einstellungen“ — Gerät, Netzwerk, Sicherheit und Updates.',
          fullscreen: '„Vollbild ein/aus“ — das entfernte Bild bildschirmfüllend darstellen.',
          collapse: '„Menü verbergen“ — die Leiste zu einem kleinen Griff zusammenklappen.'
        }
      }
    },
    finish: {
      title: 'Fertig',
      description:
        'Das ist die ganze Menüleiste. Die Tour lässt sich jederzeit in den Einstellungen auf der Seite „Erscheinungsbild“ erneut abspielen.'
    },
    replay: {
      label: 'Menü-Tour',
      description: 'Die Einführung in die Menüleiste erneut abspielen.',
      action: 'Erneut abspielen'
    },
    tour: {
      prev: 'Zurück',
      next: 'Weiter',
      end: 'Tour beenden'
    },
    mobile: {
      prompt: {
        title: 'Neu bei NanoKVM?',
        description: 'Machen Sie in einer Minute eine Tour durch die Menüleiste am Bildschirmrand.'
      },
      welcome: {
        title: 'Die Menüleiste kennenlernen',
        description: 'Diese schwebende Leiste enthält alle Steuerelemente für den entfernten Rechner. Ziehen Sie sie am Griff an eine beliebige Stelle des Bildschirmrands, und klappen Sie sie ein, wenn Sie den Vollbildmodus benötigen. Während der Tour werden Ihre Berührungen nicht an den entfernten Rechner übertragen.'
      },
      steps: {
        rail: {
          title: 'Die Menüleiste',
          description: 'Tippen Sie auf den Griff, um Werkzeuge und Einstellungen zu öffnen. Die Tour zeigt Ihnen nun jede Gruppe einzeln.'
        }
      },
      tour: {
        skip: 'Überspringen',
        prev: 'Zurück',
        next: 'Weiter',
        end: 'Tour beenden'
      }
    },
    skip: 'Tour überspringen'
  },
  fr: {
    prompt: {
      title: 'Nouveau sur NanoKVM ?',
      description: "Faites le tour d'une minute de la barre de menu.",
      start: 'Commencer',
      later: 'Plus tard',
      never: 'Ne plus afficher',
      language: 'Langue'
    },
    welcome: {
      title: 'Découvrez la barre de menu',
      description:
        "Cette barre regroupe toutes les commandes de la machine distante. Faites-la glisser par sa poignée pour la déplacer ; après quelques secondes d'inactivité elle se masque et revient dès que le pointeur atteint le bord supérieur. Pendant la visite, votre clavier et votre souris ne sont pas transmis à la machine distante."
    },
    steps: {
      display: {
        title: 'Écran',
        items: {
          screen: "« Écran » — règle la résolution, la qualité et la fréquence d'images.",
          portal: "« Supprimer les bordures noires » — recadre l'affichage sur la zone utile.",
          pictureInPicture:
            "« Image dans l'image » — garde le bureau distant dans une petite fenêtre flottante pendant que vous utilisez d'autres applications."
        }
      },
      media: {
        title: 'Son',
        items: {
          volume: '« Volume » — diffuse le son de la machine distante et en règle le niveau.',
          microphone: '« Micro » — transmet votre voix à la machine distante.'
        }
      },
      input: {
        title: 'Clavier et souris',
        items: {
          mouseJiggler: '«Souris Jiggler» — Empêcher l\'hôte distant de dormir',
          keyboard: "« Clavier » — touches à l'écran, raccourcis et saisie par collage.",
          mouse: '« Souris » — change de mode de pointeur et calibre.',
          preview: "« Aperçu » — affiche l'écran sans envoyer aucune entrée."
        }
      },
      tools: {
        title: 'Outils distants',
        items: {
          image: '« Images disque » — monte un disque virtuel sur la machine distante.',
          script: '« Script » — exécute votre code sur ce KVM.',
          terminal: '« Terminal » — ouvre une console sur le KVM.',
          wol: "« Wake-on-LAN » — démarre d'autres machines du réseau."
        }
      },
      automation: {
        title: 'Automatisation',
        items: {
          screenTimelapse:
            '« Timelapse de l’écran » — enregistre l’écran pour revoir ce qui s’est passé.',
          power:
            "« Robot-doigt » — appuie physiquement sur le bouton d'alimentation de la machine distante."
        }
      },
      system: {
        title: 'Paramètres et fenêtre',
        items: {
          settings: '« Paramètres » — appareil, réseau, sécurité et mises à jour.',
          fullscreen:
            "« Basculer vers le plein écran » — étend l'image distante à tout votre écran.",
          collapse: '« Réduire le menu » — replie la barre en une petite poignée.'
        }
      }
    },
    finish: {
      title: "C'est prêt",
      description:
        'Voilà toute la barre de menu. Rejouez cette visite à tout moment dans Paramètres, à la page Apparence.'
    },
    replay: {
      label: 'Visite du menu',
      description: 'Rejouez la présentation de la barre de menu.',
      action: 'Rejouer'
    },
    tour: {
      prev: 'Précédent',
      next: 'Suivant',
      end: 'Terminer la visite'
    },
    mobile: {
      prompt: {
        title: 'Nouveau sur NanoKVM ?',
        description: "Faites le tour d'une minute de la barre de menu latérale."
      },
      welcome: {
        title: 'Découvrez la barre de menu',
        description: "Cette barre flottante regroupe toutes les commandes de la machine distante. Faites-la glisser par sa poignée pour la placer où vous voulez sur le bord de l'écran, et repliez-la quand vous avez besoin de tout l'écran. Pendant la visite, vos touchers ne sont pas transmis à la machine distante."
      },
      steps: {
        rail: {
          title: 'La barre de menu',
          description: 'Touchez la poignée pour ouvrir les outils et les réglages. La visite mettra ensuite en évidence chaque groupe.'
        }
      },
      tour: {
        skip: 'Ignorer',
        prev: 'Précédent',
        next: 'Suivant',
        end: 'Terminer la visite'
      }
    },
    skip: 'Ignorer la visite'
  },
  es: {
    prompt: {
      title: '¿Nuevo en NanoKVM?',
      description: 'Haz un recorrido de un minuto por la barra de menú.',
      start: 'Empezar',
      later: 'Más tarde',
      never: 'No volver a mostrar',
      language: 'Idioma'
    },
    welcome: {
      title: 'Conoce la barra de menú',
      description:
        'Esta barra reúne todos los controles de la máquina remota. Arrástrala por el asa para moverla; tras unos segundos sin uso se oculta y vuelve al acercar el puntero al borde superior. Durante el recorrido, tu teclado y ratón no se envían a la máquina remota.'
    },
    steps: {
      display: {
        title: 'Pantalla',
        items: {
          screen: '«Pantalla» — ajusta resolución, calidad y frecuencia de imagen.',
          portal: '«Recortar bordes negros» — recorta la vista al área importante.',
          pictureInPicture:
            '«Imagen en imagen» — mantiene el escritorio remoto en una ventana flotante mientras usas otras aplicaciones.'
        }
      },
      media: {
        title: 'Sonido',
        items: {
          volume: '«Volumen» — reproduce el audio de la máquina remota y ajusta su nivel.',
          microphone: '«Micrófono» — envía tu voz a la máquina remota.'
        }
      },
      input: {
        title: 'Teclado y ratón',
        items: {
          mouseJiggler: '«Agitador de ratón» — Evitar que el host remoto entre en reposo',
          keyboard: '«Teclado» — teclas en pantalla, atajos y pegado como tecleo.',
          mouse: '«Ratón» — cambia los modos de puntero y calibra.',
          preview: '«Vista previa» — muestra la imagen sin enviar ninguna entrada.'
        }
      },
      tools: {
        title: 'Herramientas remotas',
        items: {
          image: '«Imágenes» — monta un disco virtual en la máquina remota.',
          script: '«Script» — ejecuta tu código en este KVM.',
          terminal: '«Consola» — abre la línea de comandos del KVM.',
          wol: '«Wake-on-LAN» — enciende otras máquinas de la red.'
        }
      },
      automation: {
        title: 'Automatización',
        items: {
          screenTimelapse: '«Timelapse de pantalla» — graba la pantalla para revisarla después.',
          power: '«Robot de dedo» — pulsa físicamente el botón de encendido de la máquina remota.'
        }
      },
      system: {
        title: 'Ajustes y ventana',
        items: {
          settings: '«Ajustes» — dispositivo, red, seguridad y actualizaciones.',
          fullscreen:
            '«Activar/Desactivar pantalla completa» — amplía la imagen remota a toda tu pantalla.',
          collapse: '«Colapsar menú» — pliega la barra en un pequeño asa.'
        }
      }
    },
    finish: {
      title: 'Listo',
      description:
        'Esa es toda la barra de menú. Repite este recorrido cuando quieras desde Ajustes, en la página Apariencia.'
    },
    replay: {
      label: 'Recorrido del menú',
      description: 'Vuelve a mostrar la presentación de la barra de menú.',
      action: 'Repetir'
    },
    tour: {
      prev: 'Anterior',
      next: 'Siguiente',
      end: 'Finalizar el recorrido'
    },
    mobile: {
      prompt: {
        title: '¿Nuevo en NanoKVM?',
        description: 'Haz un recorrido de un minuto por la barra de menú lateral.'
      },
      welcome: {
        title: 'Conoce la barra de menú',
        description: 'Esta barra flotante reúne todos los controles de la máquina remota. Arrástrala por el asa para fijarla en cualquier punto del borde y pliégala cuando necesites la pantalla completa. Durante el recorrido, tus toques no se envían a la máquina remota.'
      },
      steps: {
        rail: {
          title: 'La barra de menú',
          description: 'Toca el asa para abrir herramientas y ajustes. A continuación, el recorrido destacará cada grupo.'
        }
      },
      tour: {
        skip: 'Omitir',
        prev: 'Anterior',
        next: 'Siguiente',
        end: 'Finalizar el recorrido'
      }
    },
    skip: 'Saltar recorrido'
  },
  it: {
    prompt: {
      title: 'Prima volta su NanoKVM?',
      description: 'Fai un giro di un minuto della barra dei menu.',
      start: 'Avvia il tour',
      later: 'Più tardi',
      never: 'Non mostrare più',
      language: 'Lingua'
    },
    welcome: {
      title: 'Scopri la barra dei menu',
      description:
        "Questa barra raccoglie tutti i controlli della macchina remota. Trascinala dall'impugnatura per spostarla; dopo pochi secondi di inattività si nasconde e riappare portando il puntatore sul bordo superiore. Durante il tour, tastiera e mouse non vengono inoltrati alla macchina remota."
    },
    steps: {
      display: {
        title: 'Schermo',
        items: {
          screen: '"Schermo" — regola risoluzione, qualità e frequenza dei fotogrammi.',
          portal: '"Rimuovi bordi neri" — ritaglia la vista sull\'area che conta.',
          pictureInPicture:
            '"Modalità Picture in Picture" — mantiene il desktop remoto in una piccola finestra fluttuante mentre usi altre app.'
        }
      },
      media: {
        title: 'Audio',
        items: {
          volume: '"Volume" — riproduce l\'audio della macchina remota e ne regola il livello.',
          microphone: '"Microfono" — trasmette la tua voce alla macchina remota.'
        }
      },
      input: {
        title: 'Tastiera e mouse',
        items: {
          mouseJiggler: '"Muovi il mouse" — Impedisce la sospensione dell\'host remoto',
          keyboard: '"Tastiera" — tasti su schermo, scorciatoie e incolla come digitazione.',
          mouse: '"Mouse" — cambia le modalità del puntatore e calibra.',
          preview: '"Anteprima" — mostra lo schermo senza inviare alcun input.'
        }
      },
      tools: {
        title: 'Strumenti remoti',
        items: {
          image: '"Immagini" — monta un disco virtuale sulla macchina remota.',
          script: '"Script" — esegue il tuo codice su questo KVM.',
          terminal: '"Terminale" — apre una console sul KVM.',
          wol: '"Wake-on-LAN" — accende altre macchine in rete.'
        }
      },
      automation: {
        title: 'Automazione',
        items: {
          screenTimelapse:
            '"Timelapse schermo" — registra lo schermo per rivedere cosa è successo.',
          power:
            '"Robot a dito" — preme fisicamente il pulsante di accensione della macchina remota.'
        }
      },
      system: {
        title: 'Impostazioni e finestra',
        items: {
          settings: '"Impostazioni" — dispositivo, rete, sicurezza e aggiornamenti.',
          fullscreen:
            '"Attiva/disattiva schermo intero" — espande lo schermo remoto a tutto il display.',
          collapse: '"Comprimi menu" — riduce la barra a una piccola impugnatura.'
        }
      }
    },
    finish: {
      title: 'Tutto pronto',
      description:
        'Questa è tutta la barra dei menu. Rivedi questo tour quando vuoi da Impostazioni, nella pagina Aspetto.'
    },
    replay: {
      label: 'Tour del menu',
      description: 'Riproduci di nuovo la presentazione della barra dei menu.',
      action: 'Riproduci'
    },
    tour: {
      prev: 'Indietro',
      next: 'Avanti',
      end: 'Termina il tour'
    },
    mobile: {
      prompt: {
        title: 'Prima volta su NanoKVM?',
        description: 'Fai un giro di un minuto della barra dei menu laterale.'
      },
      welcome: {
        title: 'Scopri la barra dei menu',
        description: 'Questa barra fluttuante raccoglie tutti i controlli della macchina remota. Trascinala dall’impugnatura per posizionarla in qualsiasi punto del bordo e comprimila quando ti serve lo schermo intero. Durante il tour, i tuoi tocchi non vengono inviati alla macchina remota.'
      },
      steps: {
        rail: {
          title: 'La barra dei menu',
          description: 'Tocca l’impugnatura per aprire strumenti e impostazioni. Il tour ora evidenzierà ogni gruppo.'
        }
      },
      tour: {
        skip: 'Salta',
        prev: 'Indietro',
        next: 'Avanti',
        end: 'Termina il tour'
      }
    },
    skip: 'Salta il tour'
  },
  nl: {
    prompt: {
      title: 'Nieuw bij NanoKVM?',
      description: 'Maak een rondleiding van één minuut over de menubalk.',
      start: 'Start de tour',
      later: 'Later',
      never: 'Niet meer tonen',
      language: 'Taal'
    },
    welcome: {
      title: 'Maak kennis met de menubalk',
      description:
        'Op deze balk staan alle bedieningen voor de externe machine. Sleep hem aan de greep om te verplaatsen; na een paar seconden verdwijnt hij vanzelf en komt terug zodra de aanwijzer de bovenrand raakt. Tijdens de tour worden je toetsenbord en muis niet doorgestuurd naar de externe machine.'
    },
    steps: {
      display: {
        title: 'Scherm',
        items: {
          screen: '"Scherm" — pas resolutie, kwaliteit en beeldfrequentie aan.',
          portal: '"Zwarte balken verwijderen" — snijd de weergave bij tot het relevante gebied.',
          pictureInPicture:
            '"Beeld-in-beeld" — houd het externe bureaublad in een klein zwevend venster terwijl je andere apps gebruikt.'
        }
      },
      media: {
        title: 'Geluid',
        items: {
          volume: '"Volume" — speel het geluid van de externe machine af en regel het niveau.',
          microphone: '"Microfoon" — stuur je stem naar de externe machine.'
        }
      },
      input: {
        title: 'Toetsenbord en muis',
        items: {
          mouseJiggler: '"Muisschommel" — Voorkom dat de externe host in slaap valt',
          keyboard: '"Toetsenbord" — schermtoetsen, sneltoetsen en plakken als typen.',
          mouse: '"Muis" — schakel tussen aanwijzermodi en kalibreer.',
          preview: '"Voorbeeld" — bekijk het scherm zonder invoer te sturen.'
        }
      },
      tools: {
        title: 'Externe gereedschappen',
        items: {
          image: '"Afbeeldingen" — koppel een virtuele schijf aan de externe machine.',
          script: '"Script" — voer je eigen code uit op deze KVM.',
          terminal: '"Terminal" — open een console op de KVM.',
          wol: '"Wake-on-LAN" — start andere machines op het netwerk.'
        }
      },
      automation: {
        title: 'Automatisering',
        items: {
          screenTimelapse: '"Schermtimelapse" — neem het scherm op om later terug te kijken.',
          power: '"Vingerrobot" — druk op de fysieke aan/uit-knop van de externe machine.'
        }
      },
      system: {
        title: 'Instellingen en venster',
        items: {
          settings: '"Instellingen" — apparaat, netwerk, beveiliging en updates.',
          fullscreen: '"Volledig scherm schakelen" — vul je beeldscherm met het externe beeld.',
          collapse: '"Menu samenvouwen" — vouw de balk op tot een kleine greep.'
        }
      }
    },
    finish: {
      title: 'Klaar',
      description:
        'Dat is de hele menubalk. Speel de tour op elk moment opnieuw af via Instellingen, op de pagina Uiterlijk.'
    },
    replay: {
      label: 'Menutour',
      description: 'Speel de introductie van de menubalk opnieuw af.',
      action: 'Opnieuw afspelen'
    },
    tour: {
      prev: 'Vorige',
      next: 'Volgende',
      end: 'Tour beëindigen'
    },
    mobile: {
      prompt: {
        title: 'Nieuw bij NanoKVM?',
        description: 'Maak een rondleiding van één minuut over de menubalk aan de zijkant.'
      },
      welcome: {
        title: 'Maak kennis met de menubalk',
        description: 'Deze zwevende balk bevat alle bediening voor de externe machine. Sleep hem aan de greep naar een willekeurige plek aan de rand en klap hem in als je het volledige scherm nodig hebt. Tijdens de rondleiding worden je aanrakingen niet naar de externe machine doorgestuurd.'
      },
      steps: {
        rail: {
          title: 'De menubalk',
          description: 'Tik op de greep om tools en instellingen te openen. De rondleiding licht nu elke groep uit.'
        }
      },
      tour: {
        skip: 'Overslaan',
        prev: 'Vorige',
        next: 'Volgende',
        end: 'Tour beëindigen'
      }
    },
    skip: 'Tour overslaan'
  },
  pl: {
    prompt: {
      title: 'Pierwszy raz z NanoKVM?',
      description: 'Zrób minutową wycieczkę po pasku menu.',
      start: 'Rozpocznij',
      later: 'Później',
      never: 'Nie pokazuj ponownie',
      language: 'Język'
    },
    welcome: {
      title: 'Poznaj pasek menu',
      description:
        'Ten pasek zawiera wszystkie elementy sterowania zdalną maszyną. Przeciągnij go za uchwyt, aby przenieść; po kilku sekundach bezczynności chowa się i wraca, gdy przesuniesz wskaźnik do górnej krawędzi. Podczas wycieczki klawiatura i mysz nie są przekazywane do zdalnej maszyny.'
    },
    steps: {
      display: {
        title: 'Ekran',
        items: {
          screen: '„Ekran” — ustawia rozdzielczość, jakość i liczbę klatek.',
          portal: '„Przytnij czarne ramki” — kadruje obraz do istotnego obszaru.',
          pictureInPicture:
            '„Obraz w obrazie” — utrzymuje zdalny pulpit w małym pływającym oknie, gdy korzystasz z innych aplikacji.'
        }
      },
      media: {
        title: 'Dźwięk',
        items: {
          volume: '„Głośność” — odtwarza dźwięk zdalnej maszyny i reguluje jego poziom.',
          microphone: '„Mikrofon” — przesyła twój głos do zdalnej maszyny.'
        }
      },
      input: {
        title: 'Klawiatura i mysz',
        items: {
          mouseJiggler: '„Jiggler myszy” — Uniemożliwia uśpienie zdalnego hosta',
          keyboard: '„Klawiatura” — klawisze na ekranie, skróty i wklejanie jako pisanie.',
          mouse: '„Mysz” — przełącza tryby wskaźnika i wykonuje jego kalibrację.',
          preview: '„Podgląd” — pokazuje obraz bez wysyłania jakichkolwiek danych wejściowych.'
        }
      },
      tools: {
        title: 'Narzędzia zdalne',
        items: {
          image: '„Obrazy” — montuje wirtualny dysk na zdalnej maszynie.',
          script: '„Skrypty” — uruchamia twój kod na tym KVM.',
          terminal: '„Terminal” — otwiera konsolę KVM.',
          wol: '„Wake-on-LAN” — włącza inne maszyny w sieci.'
        }
      },
      automation: {
        title: 'Automatyzacja',
        items: {
          screenTimelapse:
            '„Timelapse ekranu” — nagrywa ekran, aby później sprawdzić, co się stało.',
          power: '„Robot palcowy” — naciska fizyczny przycisk zasilania zdalnej maszyny.'
        }
      },
      system: {
        title: 'Ustawienia i okno',
        items: {
          settings: '„Ustawienia” — urządzenie, sieć, bezpieczeństwo i aktualizacje.',
          fullscreen: '„Przełącz tryb pełnoekranowy” — rozciąga zdalny obraz na cały ekran.',
          collapse: '„Zwiń menu” — zwija pasek do małego uchwytu.'
        }
      }
    },
    finish: {
      title: 'Gotowe',
      description:
        'To cały pasek menu. Wycieczkę odtworzysz w dowolnym momencie w Ustawieniach, na stronie Wygląd.'
    },
    replay: {
      label: 'Wycieczka po menu',
      description: 'Odtwórz ponownie prezentację paska menu.',
      action: 'Odtwórz'
    },
    tour: {
      prev: 'Wróć',
      next: 'Dalej',
      end: 'Zakończ wycieczkę'
    },
    mobile: {
      prompt: {
        title: 'Pierwszy raz z NanoKVM?',
        description: 'Zrób minutową wycieczkę po bocznym pasku menu.'
      },
      welcome: {
        title: 'Poznaj pasek menu',
        description: 'Ten pływający pasek zawiera wszystkie elementy sterowania zdalnej maszyny. Przeciągnij go za uchwyt, aby przypiąć w dowolnym miejscu krawędzi, i zwiń, gdy potrzebujesz pełnego ekranu. Podczas wycieczki twoje dotknięcia nie są wysyłane do zdalnej maszyny.'
      },
      steps: {
        rail: {
          title: 'Pasek menu',
          description: 'Dotknij uchwytu, aby otworzyć narzędzia i ustawienia. Następnie wycieczka podświetli każdą grupę.'
        }
      },
      tour: {
        skip: 'Pomiń',
        prev: 'Wróć',
        next: 'Dalej',
        end: 'Zakończ wycieczkę'
      }
    },
    skip: 'Pomiń wycieczkę'
  },
  ru: {
    prompt: {
      title: 'Впервые в NanoKVM?',
      description: 'Минутный тур по панели меню.',
      start: 'Начать тур',
      later: 'Позже',
      never: 'Больше не показывать',
      language: 'Язык'
    },
    welcome: {
      title: 'Знакомство с панелью меню',
      description:
        'На этой панели собраны все элементы управления удалённой машиной. Перетащите её за ручку, чтобы переместить; через несколько секунд простоя она скрывается и возвращается, когда курсор подходит к верхнему краю. Во время тура клавиатура и мышь не передаются удалённой машине.'
    },
    steps: {
      display: {
        title: 'Экран',
        items: {
          screen: '«Экран» — настраивает разрешение, качество и частоту кадров.',
          portal: '«Удаление чёрных полос» — обрезает изображение до нужной области.',
          pictureInPicture:
            '«Картинка в картинке» — держит удалённый рабочий стол в небольшом плавающем окне, пока вы пользуетесь другими приложениями.'
        }
      },
      media: {
        title: 'Звук',
        items: {
          volume: '«Громкость» — воспроизводит звук удалённой машины и регулирует его уровень.',
          microphone: '«Микрофон» — передаёт ваш голос удалённой машине.'
        }
      },
      input: {
        title: 'Клавиатура и мышь',
        items: {
          mouseJiggler: '«Дрожалка для мыши» — Включите данную опцию, если хост засыпает',
          keyboard: '«Клавиатура» — экранные клавиши, сочетания клавиш и ввод вставкой.',
          mouse: '«Мышь» — переключает режимы указателя и калибрует его.',
          preview: '«Предпросмотр» — показывает экран, не отправляя никакого ввода.'
        }
      },
      tools: {
        title: 'Удалённые инструменты',
        items: {
          image: '«Образы» — монтирует виртуальный диск на удалённой машине.',
          script: '«Скрипты» — запускает ваш код на этом KVM.',
          terminal: '«Терминал» — открывает консоль KVM.',
          wol: '«Wake-on-LAN» — включает другие машины в сети.'
        }
      },
      automation: {
        title: 'Автоматизация',
        items: {
          screenTimelapse:
            '«Таймлапс экрана» — записывает экран, чтобы позже посмотреть, что происходило.',
          power: '«Пальцевый робот» — физически нажимает кнопку питания удалённой машины.'
        }
      },
      system: {
        title: 'Настройки и окно',
        items: {
          settings: '«Настройки» — устройство, сеть, безопасность и обновления.',
          fullscreen: '«Полноэкранный режим» — разворачивает удалённое изображение на весь экран.',
          collapse: '«Свернуть меню» — убирает панель в маленькую ручку.'
        }
      }
    },
    finish: {
      title: 'Готово',
      description:
        'Это вся панель меню. Тур можно повторить в любое время в Настройках на странице «Внешний вид».'
    },
    replay: {
      label: 'Тур по меню',
      description: 'Повторить знакомство с панелью меню.',
      action: 'Повторить'
    },
    tour: {
      prev: 'Назад',
      next: 'Далее',
      end: 'Завершить тур'
    },
    mobile: {
      prompt: {
        title: 'Впервые в NanoKVM?',
        description: 'Минутный тур по боковой панели меню.'
      },
      welcome: {
        title: 'Знакомство с панелью меню',
        description: 'Эта плавающая панель собирает все элементы управления удалённой машиной. Перетащите её за рукоятку, чтобы закрепить в любом месте края экрана, и сложите, когда нужен полный экран. Во время тура ваши касания не передаются на удалённую машину.'
      },
      steps: {
        rail: {
          title: 'Панель меню',
          description: 'Нажмите на рукоятку, чтобы открыть инструменты и настройки. Далее тур подсветит каждую группу.'
        }
      },
      tour: {
        skip: 'Пропустить',
        prev: 'Назад',
        next: 'Далее',
        end: 'Завершить тур'
      }
    },
    skip: 'Пропустить тур'
  },
  uk: {
    prompt: {
      title: 'Вперше в NanoKVM?',
      description: 'Хвилинний тур рядком меню.',
      start: 'Почати тур',
      later: 'Пізніше',
      never: 'Більше не показувати',
      language: 'Мова'
    },
    welcome: {
      title: 'Знайомство з рядком меню',
      description:
        'На цьому рядку зібрано всі елементи керування віддаленою машиною. Перетягніть його за ручку, щоб перемістити; після кількох секунд простою він ховається й повертається, коли курсор наближається до верхнього краю. Під час туру клавіатура та миша не передаються віддаленій машині.'
    },
    steps: {
      display: {
        title: 'Екран',
        items: {
          screen: '«Екран» — налаштовує роздільну здатність, якість і частоту кадрів.',
          portal: '«Видалення чорних смуг» — обрізає зображення до потрібної області.',
          pictureInPicture:
            '«Картинка в картинці» — тримає віддалений робочий стіл у маленькому плаваючому вікні, поки ви користуєтесь іншими програмами.'
        }
      },
      media: {
        title: 'Звук',
        items: {
          volume: '«Гучність» — відтворює звук віддаленої машини та регулює його рівень.',
          microphone: '«Мікрофон» — передає ваш голос віддаленій машині.'
        }
      },
      input: {
        title: 'Клавіатура і миша',
        items: {
          mouseJiggler: '«Налаштування пацюка» — Не дозволяйте віддаленому хосту спати',
          keyboard: '«Клавіатура» — екранні клавіші, сполучення клавіш і ввід вставленням.',
          mouse: '«Миша» — перемикає режими вказівника та калібрує його.',
          preview: '«Попередній перегляд» — показує екран без надсилання будь-якого вводу.'
        }
      },
      tools: {
        title: 'Віддалені інструменти',
        items: {
          image: '«Образи» — монтує віртуальний диск на віддаленій машині.',
          script: '«Скрипти» — запускає ваш код на цьому KVM.',
          terminal: '«Термінал» — відкриває консоль KVM.',
          wol: '«Wake-on-LAN» — вмикає інші машини в мережі.'
        }
      },
      automation: {
        title: 'Автоматизація',
        items: {
          screenTimelapse:
            '«Таймлапс екрана» — записує екран, щоб пізніше переглянути, що відбувалося.',
          power: '«Пальцевий робот» — фізично натискає кнопку живлення віддаленої машини.'
        }
      },
      system: {
        title: 'Налаштування та вікно',
        items: {
          settings: '«Налаштування» — пристрій, мережа, безпека та оновлення.',
          fullscreen:
            '«Перемикатися на повному екрані» — розгортає віддалене зображення на весь екран.',
          collapse: '«Згорнути меню» — згортає рядок у маленьку ручку.'
        }
      }
    },
    finish: {
      title: 'Готово',
      description:
        'Це весь рядок меню. Тур можна переглянути знову будь-коли в Налаштуваннях на сторінці «Зовнішній вигляд».'
    },
    replay: {
      label: 'Тур меню',
      description: 'Повторити знайомство з рядком меню.',
      action: 'Повторити'
    },
    tour: {
      prev: 'Назад',
      next: 'Далі',
      end: 'Завершити тур'
    },
    mobile: {
      prompt: {
        title: 'Вперше в NanoKVM?',
        description: 'Хвилинний тур боковою панеллю меню.'
      },
      welcome: {
        title: 'Знайомство з панеллю меню',
        description: 'Ця плаваюча панель зібрала всі елементи керування віддаленою машиною. Перетягніть її за ручку, щоб закріпити в будь-якому місці краю екрана, і складіть, коли потрібен повний екран. Під час туру ваші дотики не передаються на віддалену машину.'
      },
      steps: {
        rail: {
          title: 'Панель меню',
          description: 'Торкніться ручки, щоб відкрити інструменти та налаштування. Далі тур підсвітить кожну групу.'
        }
      },
      tour: {
        skip: 'Пропустити',
        prev: 'Назад',
        next: 'Далі',
        end: 'Завершити тур'
      }
    },
    skip: 'Пропустити тур'
  },
  cz: {
    prompt: {
      title: 'Poprvé v NanoKVM?',
      description: 'Projděte si minutovou prohlídku lišty nabídek.',
      start: 'Spustit prohlídku',
      later: 'Později',
      never: 'Už nezobrazovat',
      language: 'Jazyk'
    },
    welcome: {
      title: 'Seznamte se s lištou nabídek',
      description:
        'Na této liště najdete veškeré ovládání vzdáleného počítače. Přetáhněte ji za úchyt, kam potřebujete; po několika sekundách nečinnosti se schová a vrátí se, když najedete ukazatelem k hornímu okraji. Během prohlídky se klávesnice a myš nepředávají vzdálenému počítači.'
    },
    steps: {
      display: {
        title: 'Obrazovka',
        items: {
          screen: '„Obrazovka“ — nastavuje rozlišení, kvalitu a snímkovou frekvenci.',
          portal: '„Oříznout černé okraje“ — ořezává zobrazení na důležitou oblast.',
          pictureInPicture:
            '„Obraz v obraze“ — drží vzdálenou plochu v malém plovoucím okně, zatímco používáte jiné aplikace.'
        }
      },
      media: {
        title: 'Zvuk',
        items: {
          volume: '„Hlasitost“ — přehrává zvuk vzdáleného počítače a nastavuje jeho úroveň.',
          microphone: '„Mikrofon“ — přenáší váš hlas do vzdáleného počítače.'
        }
      },
      input: {
        title: 'Klávesnice a myš',
        items: {
          mouseJiggler: '„Pohyb myši (Jiggler)“ — Zabraňte spánku vzdáleného hostitele',
          keyboard: '„Klávesnice“ — klávesy na obrazovce, zkratky a vkládání textu jako psaní.',
          mouse: '„Myš“ — přepíná režimy ukazatele a provádí jeho kalibraci.',
          preview: '„Náhled“ — ukazuje obraz bez odesílání jakéhokoli vstupu.'
        }
      },
      tools: {
        title: 'Vzdálené nástroje',
        items: {
          image: '„Obrazy“ — připojí virtuální disk ke vzdálenému počítači.',
          script: '„Skript“ — spustí váš kód na tomto KVM.',
          terminal: '„Terminál“ — otevře konzoli KVM.',
          wol: '„Wake-on-LAN“ — zapne další počítače v síti.'
        }
      },
      automation: {
        title: 'Automatizace',
        items: {
          screenTimelapse:
            '„Časosběr obrazovky“ — nahrává obrazovku, abyste později viděli, co se stalo.',
          power: '„Prstový robot“ — fyzicky stiskne tlačítko napájení vzdáleného počítače.'
        }
      },
      system: {
        title: 'Nastavení a okno',
        items: {
          settings: '„Nastavení“ — zařízení, síť, zabezpečení a aktualizace.',
          fullscreen: '„Přepnout na celou obrazovku“ — roztáhne vzdálený obraz na celý displej.',
          collapse: '„Sbalit nabídku“ — složí lištu do malého úchytu.'
        }
      }
    },
    finish: {
      title: 'Hotovo',
      description:
        'To je celá lišta nabídek. Prohlídku kdykoli znovu přehrajete v Nastavení na stránce Vzhled.'
    },
    replay: {
      label: 'Prohlídka nabídek',
      description: 'Znovu přehrát seznámení s lištou nabídek.',
      action: 'Přehrát znovu'
    },
    tour: {
      prev: 'Předchozí',
      next: 'Další',
      end: 'Ukončit prohlídku'
    },
    mobile: {
      prompt: {
        title: 'Poprvé v NanoKVM?',
        description: 'Projděte si minutovou prohlídku boční lišty nabídek.'
      },
      welcome: {
        title: 'Seznamte se s lištou nabídek',
        description: 'Tento plovoucí panel obsahuje všechny ovládací prvky vzdáleného počítače. Přetáhněte jej za úchyt kamkoli na okraj obrazovky a složte jej, když potřebujete celou obrazovku. Během prohlídky se vaše dotyky nepřenášejí do vzdáleného počítače.'
      },
      steps: {
        rail: {
          title: 'Lišta nabídek',
          description: 'Klepnutím na úchyt otevřete nástroje a nastavení. Prohlídka nyní postupně zvýrazní každou skupinu.'
        }
      },
      tour: {
        skip: 'Přeskočit',
        prev: 'Předchozí',
        next: 'Další',
        end: 'Ukončit prohlídku'
      }
    },
    skip: 'Přeskočit prohlídku'
  },
  da: {
    prompt: {
      title: 'Ny til NanoKVM?',
      description: 'Tag et minuts rundvisning på menulinjen.',
      start: 'Start rundvisning',
      later: 'Senere',
      never: 'Vis ikke igen',
      language: 'Sprog'
    },
    welcome: {
      title: 'Mød menulinjen',
      description:
        'På denne bjælke finder du alle betjeninger til fjernmaskinen. Træk den i håndtaget for at flytte den; efter få sekunders pause gemmer den sig og kommer tilbage, når markøren rører den øverste kant. Under rundvisningen sendes tastatur og mus ikke videre til fjernmaskinen.'
    },
    steps: {
      display: {
        title: 'Skærm',
        items: {
          screen: '"Skærm" — juster opløsning, kvalitet og billedhastighed.',
          portal: '"Fjern sorte kanter" — beskær visningen til det relevante område.',
          pictureInPicture:
            '"Billede-i-billede" — hold fjernskrivebordet i et lille flydende vindue, mens du bruger andre apps.'
        }
      },
      media: {
        title: 'Lyd',
        items: {
          volume: '"Lydstyrke" — afspil fjernmaskinens lyd og indstil niveauet.',
          microphone: '"Mikrofon" — send din stemme til fjernmaskinen.'
        }
      },
      input: {
        title: 'Tastatur og mus',
        items: {
          mouseJiggler: '"Mus Jiggler" — Forhindrer fjernværten i at sove',
          keyboard: '"Tastatur" — skærmtaster, genveje og indsæt som tastning.',
          mouse: '"Mus" — skift mellem markørtilstande og kalibrer.',
          preview: '"Forhåndsvisning" — se skærmen uden at sende input.'
        }
      },
      tools: {
        title: 'Fjernværktøjer',
        items: {
          image: '"Diskbilleder" — montér en virtuel disk på fjernmaskinen.',
          script: '"Script" — kør din egen kode på denne KVM.',
          terminal: '"Terminal" — åbn en konsol på KVM\'en.',
          wol: '"Wake-on-LAN" — tænd andre maskiner på netværket.'
        }
      },
      automation: {
        title: 'Automatisering',
        items: {
          screenTimelapse:
            '"Skærm-timelapse" — optag skærmen, så du senere kan se, hvad der skete.',
          power: '"Fingerrobot" — tryk fysisk på fjernmaskinens tænd/sluk-knap.'
        }
      },
      system: {
        title: 'Indstillinger og vindue',
        items: {
          settings: '"Indstillinger" — enhed, netværk, sikkerhed og opdateringer.',
          fullscreen: '"Skift fuldskærm" — udvid fjernbilledet til hele skærmen.',
          collapse: '"Skjul menu" — fold bjælken sammen til et lille håndtag.'
        }
      }
    },
    finish: {
      title: 'Færdig',
      description:
        'Det var hele menulinjen. Afspil rundvisningen igen når som helst under Indstillinger på siden Udseende.'
    },
    replay: {
      label: 'Menurundvisning',
      description: 'Afspil introduktionen til menulinjen igen.',
      action: 'Afspil igen'
    },
    tour: {
      prev: 'Forrige',
      next: 'Næste',
      end: 'Afslut rundvisning'
    },
    mobile: {
      prompt: {
        title: 'Ny til NanoKVM?',
        description: 'Tag et minuts rundvisning på menulinjen i siden.'
      },
      welcome: {
        title: 'Mød menulinjen',
        description: 'Denne flydende linje samler alle kontrolelementer til fjernmaskinen. Træk den i grebet for at placere den hvor som helst på kanten, og fold den sammen, når du har brug for hele skærmen. Mens rundvisningen kører, sendes dine berøringer ikke til fjernmaskinen.'
      },
      steps: {
        rail: {
          title: 'Menulinjen',
          description: 'Tryk på grebet for at åbne værktøjer og indstillinger. Rundvisningen fremhæver nu hver gruppe.'
        }
      },
      tour: {
        skip: 'Spring over',
        prev: 'Forrige',
        next: 'Næste',
        end: 'Afslut rundvisning'
      }
    },
    skip: 'Spring rundvisningen over'
  },
  hu: {
    prompt: {
      title: 'Először használja a NanoKVM-et?',
      description: 'Tekintsen meg egy egyperces bemutatót a menüsorról.',
      start: 'Bemutató indítása',
      later: 'Később',
      never: 'Ne jelenjen meg többet',
      language: 'Nyelv'
    },
    welcome: {
      title: 'Ismerkedjen meg a menüsorral',
      description:
        'Ezen a soron található a távoli gép összes vezérlése. A fogópontnál húzva áthelyezheti; néhány másodpercnyi tétlenség után elrejtőzik, és visszatér, ha a mutatót a felső szélhez viszi. A bemutató alatt a billentyűzet és az egér nem kerül továbbításra a távoli gépre.'
    },
    steps: {
      display: {
        title: 'Képernyő',
        items: {
          screen: '„Képernyő” — beállítja a felbontást, a minőséget és a képkockasebességet.',
          portal: '„Fekete sávok levágása” — a képet a lényeges területre vágja.',
          pictureInPicture:
            '„Kép a képben” — egy kis lebegő ablakban tartja a távoli asztalt, miközben más alkalmazásokat használ.'
        }
      },
      media: {
        title: 'Hang',
        items: {
          volume: '„Hangerő” — lejátssza a távoli gép hangját és állítja annak szintjét.',
          microphone: '„Mikrofon” — az Ön hangját továbbítja a távoli gépre.'
        }
      },
      input: {
        title: 'Billentyűzet és egér',
        items: {
          mouseJiggler: '„Egér mozgató” — A távoli gazdagép alvó állapotának megakadályozása',
          keyboard:
            '„Billentyűzet” — képernyőn megjelenő billentyűk, gyorsbillentyűk és beillesztés gépelésként.',
          mouse: '„Egér” — váltja a mutató módjait és elvégzi a kalibrációt.',
          preview: '„Előnézet” — bemenet küldése nélkül mutatja a képet.'
        }
      },
      tools: {
        title: 'Távoli eszközök',
        items: {
          image: '„Képek” — virtuális lemezt csatol a távoli géphez.',
          script: '„Szkriptek” — az Ön kódját futtatja ezen a KVM-en.',
          terminal: '„Terminál” — konzolt nyit a KVM-en.',
          wol: '„Wake-on-LAN” — más gépeket kapcsol be a hálózaton.'
        }
      },
      automation: {
        title: 'Automatizálás',
        items: {
          screenTimelapse:
            '„Képernyő-időgyorsítás” — rögzíti a képernyőt, hogy később visszanézhesse.',
          power: '„Ujjrobot” — fizikailag megnyomja a távoli gép bekapcsológombját.'
        }
      },
      system: {
        title: 'Beállítások és ablak',
        items: {
          settings: '„Beállítások” — eszköz, hálózat, biztonság és frissítések.',
          fullscreen: '„Teljes képernyő váltás” — kiterjeszti a távoli képet a teljes kijelzőre.',
          collapse: '„Menü összecsukása” — egy kis fogóponttá csukja össze a sort.'
        }
      }
    },
    finish: {
      title: 'Kész',
      description:
        'Ez az egész menüsor. A bemutatót bármikor újra lejátssza a Beállítások Megjelenés oldalán.'
    },
    replay: {
      label: 'Menübemutató',
      description: 'Játssza le újra a menüsor bemutatását.',
      action: 'Újra'
    },
    tour: {
      prev: 'Előző',
      next: 'Következő',
      end: 'Bemutató befejezése'
    },
    mobile: {
      prompt: {
        title: 'Először használja a NanoKVM-et?',
        description: 'Tekintsen meg egy egyperces bemutatót az oldalsó menüsorról.'
      },
      welcome: {
        title: 'A menüsor bemutatása',
        description: 'Ez a lebegő sáv a távoli gép összes vezérlőjét tartalmazza. Húzza a fogantyúnál az él tetszőleges pontjára, és csukja össze, ha teljes képernyőre van szüksége. A bemutató alatt érintései nem kerülnek továbbításra a távoli gépre.'
      },
      steps: {
        rail: {
          title: 'A menüsor',
          description: 'Érintse meg a fogantyút az eszközök és beállítások megnyitásához. A bemutató most sorban kiemeli az egyes csoportokat.'
        }
      },
      tour: {
        skip: 'Kihagyás',
        prev: 'Előző',
        next: 'Következő',
        end: 'Bemutató befejezése'
      }
    },
    skip: 'Bemutató kihagyása'
  },
  id: {
    prompt: {
      title: 'Baru di NanoKVM?',
      description: 'Ikuti tur satu menit bilah menu.',
      start: 'Mulai tur',
      later: 'Nanti',
      never: 'Jangan tampilkan lagi',
      language: 'Bahasa'
    },
    welcome: {
      title: 'Kenali bilah menu',
      description:
        'Bilah ini memuat semua kendali untuk mesin jarak jauh. Seret lewat gagangnya untuk memindahkan; setelah beberapa detik tidak aktif, bilah bersembunyi dan kembali saat penunjuk menyentuh tepi atas. Selama tur, keyboard dan mouse Anda tidak diteruskan ke mesin jarak jauh.'
    },
    steps: {
      display: {
        title: 'Layar',
        items: {
          screen: '"Layar" — mengatur resolusi, kualitas, dan kecepatan bingkai.',
          portal: '"Hapus batas hitam" — memangkas tampilan ke area yang penting.',
          pictureInPicture:
            '"Gambar dalam gambar" — menjaga desktop jarak jauh tetap mengambang di jendela kecil saat Anda memakai aplikasi lain.'
        }
      },
      media: {
        title: 'Suara',
        items: {
          volume: '"Volume" — memutar audio mesin jarak jauh dan mengatur levelnya.',
          microphone: '"Mikrofon" — mengalirkan suara Anda ke mesin jarak jauh.'
        }
      },
      input: {
        title: 'Keyboard dan mouse',
        items: {
          mouseJiggler: '"Tikus Jiggler" — Mencegah host jarak jauh tertidur',
          keyboard: '"Papan ketik" — tombol di layar, pintasan, dan tempel-sebagai-ketikan.',
          mouse: '"Mouse" — mengganti mode penunjuk dan mengalibrasi.',
          preview: '"Pratinjau" — menampilkan layar tanpa mengirim input apa pun.'
        }
      },
      tools: {
        title: 'Alat jarak jauh',
        items: {
          image: '"Gambar" — memasang disk virtual di mesin jarak jauh.',
          script: '"Script" — menjalankan kode Anda di KVM ini.',
          terminal: '"Terminal" — membuka konsol di KVM.',
          wol: '"Wake-on-LAN" — menyalakan mesin lain di jaringan.'
        }
      },
      automation: {
        title: 'Otomatisasi',
        items: {
          screenTimelapse: '"Timelapse layar" — merekam layar untuk ditinjau nanti.',
          power: '"Robot Jari" — menekan tombol daya fisik mesin jarak jauh.'
        }
      },
      system: {
        title: 'Pengaturan dan jendela',
        items: {
          settings: '"Pengaturan" — perangkat, jaringan, keamanan, dan pembaruan.',
          fullscreen:
            '"Beralih Layar Penuh" — memperluas layar jarak jauh ke seluruh tampilan Anda.',
          collapse: '"Tutup Menu" — melipat bilah menjadi gagang kecil.'
        }
      }
    },
    finish: {
      title: 'Semua siap',
      description:
        'Itulah seluruh bilah menu. Putar ulang tur ini kapan saja dari Pengaturan, di halaman Penampilan.'
    },
    replay: {
      label: 'Tur menu',
      description: 'Putar ulang pengenalan bilah menu.',
      action: 'Putar ulang'
    },
    tour: {
      prev: 'Sebelumnya',
      next: 'Lanjut',
      end: 'Akhiri tur'
    },
    mobile: {
      prompt: {
        title: 'Baru di NanoKVM?',
        description: 'Ikuti tur satu menit bilah menu di tepi layar.'
      },
      welcome: {
        title: 'Kenali bilah menu',
        description: 'Bilah mengambang ini memuat semua kendali mesin jarak jauh. Seret lewat pegangan untuk menempatkannya di mana saja pada tepi layar, dan lipat saat Anda membutuhkan layar penuh. Selama tur berjalan, sentuhan Anda tidak dikirim ke mesin jarak jauh.'
      },
      steps: {
        rail: {
          title: 'Bilah menu',
          description: 'Ketuk pegangan untuk membuka alat dan pengaturan. Tur selanjutnya akan menyorot setiap grup.'
        }
      },
      tour: {
        skip: 'Lewati',
        prev: 'Sebelumnya',
        next: 'Lanjut',
        end: 'Akhiri tur'
      }
    },
    skip: 'Lewati tur'
  },
  nb: {
    prompt: {
      title: 'Ny til NanoKVM?',
      description: 'Ta en minutts omvisning på menylinjen.',
      start: 'Start omvisning',
      later: 'Senere',
      never: 'Ikke vis igjen',
      language: 'Språk'
    },
    welcome: {
      title: 'Bli kjent med menylinjen',
      description:
        'På denne linjen finner du alle kontrollene for fjernmaskinen. Dra den i håndtaket for å flytte den; etter noen sekunder uten bruk skjuler den seg og kommer tilbake når pekeren når den øverste kanten. Mens omvisningen pågår, sendes ikke tastaturet og musen videre til fjernmaskinen.'
    },
    steps: {
      display: {
        title: 'Skjerm',
        items: {
          screen: '«Skjerm» — juster oppløsning, kvalitet og bildefrekvens.',
          portal: '«Fjern svarte kanter» — beskjær visningen til området som betyr noe.',
          pictureInPicture:
            '«Bilde i bilde» — hold fjernskrivebordet i et lite flytende vindu mens du bruker andre apper.'
        }
      },
      media: {
        title: 'Lyd',
        items: {
          volume: '«Volum» — spill av lyden fra fjernmaskinen og styr nivået.',
          microphone: '«Mikrofon» — send stemmen din til fjernmaskinen.'
        }
      },
      input: {
        title: 'Tastatur og mus',
        items: {
          mouseJiggler: '«Mus Jiggler» — Hindre den eksterne verten fra å sove',
          keyboard: '«Åpne tastatur» — skjermtaster, snarveier og innliming som tasting.',
          mouse: '«Mus» — bytt mellom pekermoduser og kalibrer.',
          preview: '«Forhåndsvisning» — se skjermen uten å sende inndata.'
        }
      },
      tools: {
        title: 'Fjernverktøy',
        items: {
          image: '«Bilder» — monter en virtuell disk på fjernmaskinen.',
          script: '«Skript» — kjør din egen kode på denne KVM-en.',
          terminal: '«Terminal» — åpne en konsoll på KVM-en.',
          wol: '«Wake-on-LAN» — slå på andre maskiner på nettverket.'
        }
      },
      automation: {
        title: 'Automatisering',
        items: {
          screenTimelapse:
            '«Skjerm-timelapse» — ta opp skjermen slik at du kan se hva som skjedde senere.',
          power: '«Fingerrobot» — trykk fysisk på fjernmaskinens strømknapp.'
        }
      },
      system: {
        title: 'Innstillinger og vindu',
        items: {
          settings: '«Innstillinger» — enhet, nettverk, sikkerhet og oppdateringer.',
          fullscreen: '«Veksle fullskjerm» — utvid fjernbildet til hele skjermen din.',
          collapse: '«Skjul meny» — legg linjen ned i et lite håndtak.'
        }
      }
    },
    finish: {
      title: 'Ferdig',
      description:
        'Det er hele menylinjen. Spill av omvisningen igjen når som helst fra Innstillinger, på siden Utseende.'
    },
    replay: {
      label: 'Menyomvisning',
      description: 'Spill av introduksjonen til menylinjen på nytt.',
      action: 'Spill av igjen'
    },
    tour: {
      prev: 'Forrige',
      next: 'Neste',
      end: 'Avslutt omvisningen'
    },
    mobile: {
      prompt: {
        title: 'Ny til NanoKVM?',
        description: 'Ta en minutts omvisning på menylinjen i kanten.'
      },
      welcome: {
        title: 'Bli kjent med menylinjen',
        description: 'Denne flytende linjen samler alle kontrollene for fjernmaskinen. Dra den i håndtaket for å plassere den hvor som helst langs kanten, og fold den sammen når du trenger hele skjermen. Mens omvisningen pågår, sendes ikke berøringene dine til fjernmaskinen.'
      },
      steps: {
        rail: {
          title: 'Menylinjen',
          description: 'Trykk på håndtaket for å åpne verktøy og innstillinger. Omvisningen fremhever nå hver gruppe.'
        }
      },
      tour: {
        skip: 'Hopp over',
        prev: 'Forrige',
        next: 'Neste',
        end: 'Avslutt omvisningen'
      }
    },
    skip: 'Hopp over omvisningen'
  },
  th: {
    prompt: {
      title: 'ใช้ NanoKVM ครั้งแรกไหม',
      description: 'ใช้เวลาหนึ่งนาทีทำความรู้จักแถบเมนู',
      start: 'เริ่มทัวร์',
      later: 'ภายหลัง',
      never: 'ไม่ต้องแสดงอีก',
      language: 'ภาษา'
    },
    welcome: {
      title: 'รู้จักแถบเมนู',
      description:
        'แถบนี้รวมทุกการควบคุมสำหรับเครื่องระยะไกล ลากที่ตัวจับเพื่อย้ายตำแหน่ง เมื่อไม่ได้ใช้งานสักครู่มันจะซ่อนตัว และจะกลับมาเมื่อเลื่อนตัวชี้ไปที่ขอบบน ระหว่างทัวร์ คีย์บอร์ดและเมาส์ของคุณจะไม่ถูกส่งไปยังเครื่องระยะไกล'
    },
    steps: {
      display: {
        title: 'หน้าจอ',
        items: {
          screen: '"หน้าจอ" — ปรับความละเอียด คุณภาพ และอัตราเฟรม',
          portal: '"ตัดขอบดำ" — ครอบตัดมุมมองให้เหลือเฉพาะพื้นที่ที่สำคัญ',
          pictureInPicture:
            '"ภาพซ้อนภาพ" — เก็บเดสก์ท็อประยะไกลไว้ในหน้าต่างลอยขนาดเล็กขณะใช้งานแอปอื่น'
        }
      },
      media: {
        title: 'เสียง',
        items: {
          volume: '"ระดับเสียง" — เล่นเสียงของเครื่องระยะไกลและปรับความดัง',
          microphone: '"ไมโครโฟน" — ส่งเสียงของคุณไปยังเครื่องระยะไกล'
        }
      },
      input: {
        title: 'คีย์บอร์ดและเมาส์',
        items: {
          mouseJiggler: '"เมาส์กระตุก" — ป้องกันไม่ให้โฮสต์ระยะไกลเข้าสู่โหมดสลีป',
          keyboard: '"คีบอร์ด" — แป้นพิมพ์บนหน้าจอ คีย์ลัด และการวางแบบพิมพ์',
          mouse: '"เมาส์" — สลับโหมดตัวชี้และปรับเทียบ',
          preview: '"ตัวอย่าง" — แสดงหน้าจอโดยไม่ส่งอินพุตใด ๆ'
        }
      },
      tools: {
        title: 'เครื่องมือระยะไกล',
        items: {
          image: '"อิมเมจ" — เมาต์ดิสก์เสมือนให้เครื่องระยะไกล',
          script: '"สคริปต์" — รันโค้ดของคุณบน KVM เครื่องนี้',
          terminal: '"เทอร์มินอล" — เปิดคอนโซลของ KVM',
          wol: '"Wake-on-LAN" — ปลุกเครื่องอื่นในเครือข่ายให้ตื่น'
        }
      },
      automation: {
        title: 'ระบบอัตโนมัติ',
        items: {
          screenTimelapse: '"ไทม์แลปส์หน้าจอ" — บันทึกหน้าจอเพื่อย้อนดูภายหลัง',
          power: '"หุ่นยนต์นิ้ว" — กดปุ่มพลังงานจริงของเครื่องระยะไกล'
        }
      },
      system: {
        title: 'การตั้งค่าและหน้าต่าง',
        items: {
          settings: '"การตั้งค่า" — อุปกรณ์ เครือข่าย ความปลอดภัย และการอัปเดต',
          fullscreen: '"โหมดเต็มหน้าจอ" — ขยายภาพระยะไกลให้เต็มจอ',
          collapse: '"ย่อเมนู" — เก็บแถบเมนูเป็นตัวจับเล็ก ๆ'
        }
      }
    },
    finish: {
      title: 'พร้อมใช้งาน',
      description:
        'นั่นคือแถบเมนูทั้งหมด เล่นทัวร์นี้ซ้ำได้ทุกเมื่อจาก การตั้งค่า ที่หน้า ลักษณะการแสดงผล'
    },
    replay: {
      label: 'ทัวร์เมนู',
      description: 'เล่นซ้ำการแนะนำแถบเมนู',
      action: 'เล่นซ้ำ'
    },
    tour: {
      prev: 'ก่อนหน้า',
      next: 'ถัดไป',
      end: 'จบทัวร์'
    },
    mobile: {
      prompt: {
        title: 'ใช้ NanoKVM ครั้งแรกไหม',
        description: 'ใช้เวลาหนึ่งนาทีทำความรู้จักแถบเมนูด้านข้าง'
      },
      welcome: {
        title: 'ทำความรู้จักแถบเมนู',
        description: 'แถบลอยนี้รวมตัวควบคุมทั้งหมดของเครื่องระยะไกล ลากที่ตัวจับเพื่อวางไว้ที่ขอบหน้าจอตำแหน่งใดก็ได้ และพับเก็บเมื่อคุณต้องการเต็มหน้าจอ ระหว่างการแนะนำ การแตะของคุณจะไม่ถูกส่งไปยังเครื่องระยะไกล'
      },
      steps: {
        rail: {
          title: 'แถบเมนู',
          description: 'แตะที่ตัวจับเพื่อเปิดเครื่องมือและการตั้งค่า จากนั้นการแนะนำจะไฮไลต์แต่ละกลุ่มทีละกลุ่ม'
        }
      },
      tour: {
        skip: 'ข้าม',
        prev: 'ก่อนหน้า',
        next: 'ถัดไป',
        end: 'จบทัวร์'
      }
    },
    skip: 'ข้ามทัวร์'
  },
  vi: {
    prompt: {
      title: 'Lần đầu dùng NanoKVM?',
      description: 'Dành một phút khám phá thanh trình đơn.',
      start: 'Bắt đầu',
      later: 'Để sau',
      never: 'Không hiện lại',
      language: 'Ngôn ngữ'
    },
    welcome: {
      title: 'Làm quen thanh trình đơn',
      description:
        'Thanh này chứa mọi điều khiển cho máy từ xa. Kéo tay cầm để di chuyển; sau vài giây không dùng, thanh tự ẩn và trở lại khi bạn đưa con trỏ lên mép trên. Trong lúc xem hướng dẫn, bàn phím và chuột của bạn không được chuyển tới máy từ xa.'
    },
    steps: {
      display: {
        title: 'Màn hình',
        items: {
          screen: '"Màn hình" — chỉnh độ phân giải, chất lượng và tốc độ khung hình.',
          portal: '"Cắt viền đen" — cắt khung hình về vùng bạn quan tâm.',
          pictureInPicture:
            '"Ảnh trong ảnh" — giữ màn hình từ xa trong một cửa sổ nhỏ nổi trong khi bạn dùng ứng dụng khác.'
        }
      },
      media: {
        title: 'Âm thanh',
        items: {
          volume: '"Âm lượng" — phát âm thanh của máy từ xa và chỉnh mức.',
          microphone: '"Micrô" — truyền giọng của bạn tới máy từ xa.'
        }
      },
      input: {
        title: 'Bàn phím và chuột',
        items: {
          mouseJiggler: '"Máy lắc lư chuột" — Ngăn máy chủ từ xa ngủ',
          keyboard: '"Bàn phím" — phím trên màn hình, phím tắt và dán dưới dạng gõ phím.',
          mouse: '"Chuột" — đổi chế độ con trỏ và hiệu chỉnh.',
          preview: '"Xem trước" — chỉ xem màn hình mà không gửi bất kỳ đầu vào nào.'
        }
      },
      tools: {
        title: 'Công cụ từ xa',
        items: {
          image: '"Hình ảnh" — gắn đĩa ảo vào máy từ xa.',
          script: '"Tập lệnh" — chạy mã của bạn trên KVM này.',
          terminal: '"Thiết bị đầu cuối" — mở bảng điều khiển của KVM.',
          wol: '"Wake-on-LAN" — bật các máy khác trong mạng.'
        }
      },
      automation: {
        title: 'Tự động hóa',
        items: {
          screenTimelapse: '"Timelapse màn hình" — ghi lại màn hình để xem lại sau.',
          power: '"Robot ngón tay" — nhấn nút nguồn vật lý của máy từ xa.'
        }
      },
      system: {
        title: 'Cài đặt và cửa sổ',
        items: {
          settings: '"Cài đặt" — thiết bị, mạng, bảo mật và cập nhật.',
          fullscreen: '"Chuyển đổi toàn màn hình" — mở rộng hình ảnh từ xa ra toàn màn hình.',
          collapse: '"Thu gọn Menu" — gập thanh thành một tay cầm nhỏ.'
        }
      }
    },
    finish: {
      title: 'Hoàn tất',
      description:
        'Đó là toàn bộ thanh trình đơn. Xem lại hướng dẫn này bất cứ lúc nào trong Cài đặt, tại trang Giao diện.'
    },
    replay: {
      label: 'Hướng dẫn trình đơn',
      description: 'Xem lại phần giới thiệu về thanh trình đơn.',
      action: 'Xem lại'
    },
    tour: {
      prev: 'Trước',
      next: 'Tiếp',
      end: 'Kết thúc'
    },
    mobile: {
      prompt: {
        title: 'Lần đầu dùng NanoKVM?',
        description: 'Dành một phút khám phá thanh trình đơn cạnh màn hình.'
      },
      welcome: {
        title: 'Làm quen với thanh trình đơn',
        description: 'Thanh nổi này chứa mọi điều khiển của máy từ xa. Kéo bằng tay cầm để đặt ở bất kỳ vị trí nào trên cạnh màn hình, và thu gọn khi bạn cần toàn màn hình. Trong khi hướng dẫn chạy, thao tác chạm của bạn sẽ không được gửi tới máy từ xa.'
      },
      steps: {
        rail: {
          title: 'Thanh trình đơn',
          description: 'Chạm vào tay cầm để mở công cụ và cài đặt. Tiếp theo, hướng dẫn sẽ làm nổi bật từng nhóm.'
        }
      },
      tour: {
        skip: 'Bỏ qua',
        prev: 'Trước',
        next: 'Tiếp',
        end: 'Kết thúc'
      }
    },
    skip: 'Bỏ qua hướng dẫn'
  }
};

export default function onboardingMessages(language: string): OnboardingMessages {
  return messages[language as SupportedLanguage] ?? messages.en;
}
