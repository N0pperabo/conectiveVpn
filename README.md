# Conective

<p align="center">
  <strong>High-Performance, Multi-Engine Anti-Censorship & VPN Desktop Client for Windows</strong>
  <br />
  <strong>کلاینت دسکتاپ پرسرعت، چندموتوره و پیشرفته عبور از فیلترینگ برای ویندوز</strong>
  <br />
  <strong>Высокопроизводительный многомоторный десктоп-клиент для обхода цензуры и VPN для Windows</strong>
  <br />
  <strong>高性能、多引擎 Windows 桌面端抗封锁与 VPN 客户端</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Language-Go%201.22+-00ADD8?style=for-the-badge&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/Platform-Windows%2010%20%2F%2011%20(x64)-0078D6?style=for-the-badge&logo=windows" alt="Windows" />
  <img src="https://img.shields.io/badge/Core-Xray--core%20v1.26+-blue?style=for-the-badge" alt="Xray" />
  <img src="https://img.shields.io/badge/Engine-Psiphon%20%2F%20Aether-orange?style=for-the-badge" alt="Psiphon" />
  <img src="https://img.shields.io/badge/Driver-Wintun%20TUN%20Adapter-darkgreen?style=for-the-badge" alt="Wintun" />
  <img src="https://img.shields.io/badge/UI-WebView2%20Dashboard-purple?style=for-the-badge" alt="WebView2" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License" />
</p>

<p align="center">
  <a href="#english">English</a> •
  <a href="#فارسی-persian">فارسی</a> •
  <a href="#русский-russian">Русский</a> •
  <a href="#中文-chinese">中文</a>
</p>

> **English**: High-Performance, Multi-Engine Anti-Censorship & VPN Desktop Client for Windows with Embedded Xray-core, Psiphon, Clean IP & Wintun.  
> **فارسی**: کلاینت ویندوز چندموتوره و پرسرعت برای عبور از فیلترینگ و دور زدن سانسور با هسته‌های Xray، سایفون، آی‌پی تمیز و تونل Wintun.  
> **Русский**: Высокопроизводительный многомоторный клиент для обхода цензуры и VPN под Windows с интеграцией Xray-core, Psiphon, чистых IP и Wintun.  
> **中文**: 专为 Windows 设计的高性能多引擎抗审查与 VPN 桌面客户端，集成 Xray-core、Psiphon/Aether、Akamai 优选 Clean IP 与 Wintun TUN 虚拟网卡。

---

## English

### Overview

**Conective** is a modern, high-performance Windows desktop circumvention client designed for heavily restricted network environments (such as Iran). Built with Go and a hardware-accelerated Microsoft Edge WebView2 user interface, Conective merges the capabilities of **Xray-core**, **Psiphon / Aether**, **MHR Relay (Google Apps Script Fronting)**, **WinDivert SNI Spoofing**, **Multi-CDN Clean IP Fronting (Akamai, Cloudflare, CloudFront)**, and low-latency **Wintun TUN** virtual networking into a single, cohesive desktop experience.

---

### Key Features

#### 1. Multi-Engine Circumvention
- **Xray Core Integration**: Native support for modern proxy protocols including **VLESS**, **VMess**, **Trojan**, **Shadowsocks**, **VLESS-Reality**, **SOCKS5**, and **HTTP** outbounds.
- **MHR Relay (Google Apps Script & CFW)**: Domain-fronting relay system fronted by Google services (`www.google.com`, `mail.google.com`, `accounts.google.com`), supporting both standard Apps Script and Cloudflare Worker backends with real-time Google Anycast IP scanning and automatic Root CA installation.
- **SNI Spoofing DPI Bypass**: WinDivert-powered packet manipulation engine that transparently injects whitelisted TLS SNI headers at the network layer to bypass strict DPI firewalls.
- **Psiphon & Aether Engine**: Built-in direct Psiphon circumvention engine running in high-performance mode with automated tactic negotiation and native ShiroKhorshid core integration.
- **Egress Region Selection**: Connect directly through 20 curated exit countries or use intelligent **Auto (Fastest)** mode.
- **Unified Tactics Datastore**: Centralized circumvention cache (`psiphon.boltdb` / `psiphon_server_list.dat`) shared across all country selections, preserving learned DPI bypass paths and accelerating handshakes down to 2–5 seconds.
- **Anti-DPI TLS SNI Fragmentation**: RFC-compliant `packets: 1-1` TLS ClientHello SNI record splitting to bypass Deep Packet Inspection firewalls without packet loss.
- **Censorship-Resistant DNS & QUIC Protection**: Built-in DNS-over-HTTPS (DoH) fallback to block DNS poisoning; automatic blocking of UDP port 443 (QUIC) to prevent ISP throttling.

#### 2. Advanced Network & TUN Routing
- **True System-Wide TUN Mode**: High-throughput Layer 3 virtual network interface powered by WireGuard's `wintun.dll` driver (`198.18.0.0/16` subnet, prioritized interface metric).
- **Concurrent System Proxy & TUN**: Keeps Windows System Proxy active alongside TUN mode to guarantee zero IPv6 or browser leaks on dual-stack ISP connections.
- **Smart Routing Modes**:
  - **Bypass Iran**: Automatically routes domestic `.ir` domains, Iranian banking IP ranges, and local traffic direct while tunneling blocked international destinations.
  - **Global**: Routes 100% of system traffic through the active tunnel.
- **Low-Latency Gaming Mode**: UDP socket priority, MTU optimization (1400 in gaming mode vs 1500 default), and `TCP_NODELAY` for reduced ping in online games.
- **Routing Loop Prevention**: Explicit process-level exemption rules for circumvention engines (`psiphon-tunnel-core.exe`, `aether.exe`, `sni-spoofing.exe`) ensuring tunnel handshakes never get trapped in virtual routing loops.

#### 3. Multi-CDN Clean IP Scanner & Fronting
- **Multi-CDN Coverage (Akamai, Cloudflare, CloudFront)**: Dedicated scanner targeting high-reputation CDN Anycast CIDRs with automatic CDN provider detection.
- **Country-Targeted Clean IP Scans**: When an egress country is selected, the scanner strictly discovers and tests Clean IPs geolocated in that specific country with accurate country naming and flags.
- **Auto (Fastest) Global Mode**: Scans and ranks the lowest-latency edge IPs worldwide without geographic boundaries.
- **Customizable Scan Parameters**: Adjust worker threads (50 to 300), handshake timeouts (1000ms to 2500ms), and candidate sample sizes.
- **One-Click Deployment**: Directly connect to or add discovered clean IPs to your saved node list.

#### 4. Real-Time Live Logs
- **Server-Sent Events (SSE) Stream**: Live diagnostic terminal streaming real-time stdout/stderr from Psiphon, Aether, Xray, TUN adapter, and REST API.
- **Developer Controls**: Filter logs by source, search log messages, toggle auto-scroll, copy console output to clipboard, or clear with one click.

#### 5. Native Windows Experience
- **WebView2 UI**: Clean, responsive dark dashboard consuming less than 50MB RAM.
- **Clean Aesthetic**: Modern, professional UI design strictly free of emojis.
- **System Tray**: Minimize to Windows Notification Area on close, preserving background connectivity.
- **Persistent State**: Automatic storage of settings, filters, and preferences via SQLite (WAL mode) and local storage.

---

### Architecture

```
Conective/
├── cmd/
│   ├── conective/        # Main application entry point (GUI window, tray, WebView2)
│   ├── installer/        # Single-file setup builder (Conective-Setup.exe)
│   └── ipdata_tool/      # GeoIP analysis utility
├── pkg/
│   ├── api/              # Local REST API server powering the frontend
│   ├── autostart/        # Windows startup registry integration
│   ├── cleanip/          # Akamai Clean IP scanner, probe, and fronting manager
│   ├── database/         # SQLite storage engine with migrations
│   ├── gaming/           # Gaming mode optimizer
│   ├── geoip/            # GeoLite2 country resolver
│   ├── logger/           # Live streaming log ring buffer & SSE broadcaster
│   ├── mhr/              # MHR Google Apps Script / Cloudflare Worker domain fronting relay
│   ├── models/           # Data models for nodes, settings, and logs
│   ├── proxy/            # Windows WinINet system proxy manager
│   ├── psiphon/          # Psiphon & Aether engine controller
│   ├── scheduler/        # Background health check and auto-scanner
│   ├── snispoof/         # WinDivert SNI Spoofing DPI bypass engine manager
│   ├── subscription/     # Multi-format subscription importer (Base64, Clash, Sing-Box)
│   ├── tray/             # Native Win32 system tray integration
│   ├── tun/              # Wintun Layer 3 virtual adapter manager
│   ├── updater/          # In-app GitHub release updater
│   ├── v2go/             # High-speed node latency tester
│   └── xray/             # Embedded Xray-core manager & routing engine
├── assets/
│   ├── web/              # Dashboard frontend (HTML5, CSS3, ES6 JavaScript)
│   ├── app.ico           # Application icon
│   └── bin/              # Bundled circumvention binaries (aether.exe, psiphon, xray)
└── scripts/
    └── build.ps1         # Automated PowerShell build script
```

---

### Building from Source

#### Prerequisites
1. **Windows 10 / 11** (64-bit)
2. **Go 1.22+** installed and added to `PATH`
3. **Git** for Windows

#### Build Commands
Clone the repository and run in PowerShell:

```powershell
# Build portable executable
go build -ldflags="-H=windowsgui -s -w" -o Conective.exe ./cmd/conective

# Or run the automated script to build Conective.exe and Conective-Setup.exe
.\scripts\build.ps1
```

---

## فارسی (Persian)

### معرفی پروژه

**کانکتیو (Conective)** یک نرم‌افزار دسکتاپ جامع، مدرن و پرسرعت برای سیستم‌عامل ویندوز است که به طور ویژه برای دور زدن فیلترینگ شدید و سانسور اینترنت در ایران توسعه یافته است. این برنامه ترکیبی از هسته پیشرفته **Xray-core**، موتور دور زننده **سایفون و ایثر (Psiphon / Aether)**، رله اختصاصی **MHR Relay (فرانتینگ گوگل اپس اسکریپت)**، تکنولوژی ضد فیلتر **SNI Spoofing با WinDivert**، اسکنر فرانتینگ **آی‌پی‌های تمیز چند CDN (Akamai, Cloudflare, CloudFront)** و درایور پرسرعت شبکه مجازی **Wintun TUN** می‌باشد.

---

### قابلیت‌های کلیدی

#### ۱. موتورهای چندگانه عبور از فیلترینگ
- **پشتیبانی کامل از Xray**: پشتیبانی داخلی از پروتکل‌های **VLESS**، **VMess**، **Trojan**، **Shadowsocks**، **VLESS-Reality** و خروجی‌های **SOCKS5** و **HTTP**.
- **رله MHR (Google Apps Script & CFW)**: سامانه رله دامین فرانتینگ با دامنه‌های گوگل (`www.google.com`, `mail.google.com`, `accounts.google.com`) با پشتیبانی از اسکریپت گوگل و ورکر کلودفلر، اسکنر پینگ آی‌پی‌های Anycast گوگل و نصب خودکار گواهی Root CA برای اینترسپت ایمن یوتیوب و HTTPS.
- **تکنولوژی فریب هدر SNI (SNI Spoofing)**: دور زدن فیلترینگ بر پایه دستکاری بسته‌ها در لایه ۳ و ۴ با درایور WinDivert جهت تزریق SNIهای مجاز بدون مسدود شدن ترافیک TLS.
- **موتور قدرتمند سایفون (Psiphon / Aether)**: اتصال مستقیم بدون نیاز به کانفیگ با استفاده از تاکتیک‌های مایک، مبهم‌سازی OSSH، شکل‌دهی خودکار ترافیک و هسته محلی شیر و خورشید.
- **انتخاب کشور خروجی (Egress Region)**: امکان انتخاب ۲۰ کشور معتبر جهان یا حالت هوشمند خودکار (سریع‌ترین سرور).
- **دیتابیس تاکتیک‌های یکپارچه**: دیتابیس یادگیرنده مرکزی (`psiphon.boltdb` / `psiphon_server_list.dat`) میان تمام کشورها به اشتراک گذاشته شده است تا تاکتیک‌های موفق دور زدن سانسور حفظ شده و زمان اتصال به ۲ الی ۵ ثانیه کاهش یابد.
- **تکه‌تکه‌سازی هدر TLS (SNI Fragmentation)**: پیاده‌سازی استاندارد قطعه‌بندی `packets: 1-1` در رکورد ClientHello برای فریب فایروال‌های DPI بدون افت پهنای باند و بدون ایجاد تأخیر.
- **محافظت در برابر مسمومیت DNS و QUIC**: مجهز به سامانه امن DNS-over-HTTPS و مسدودسازی ترافیک UDP پورت ۴۴۳ برای جلوگیری از اختلالات عمدی پروتکل QUIC توسط اپراتورها.

#### ۲. شبکه مجازی و حالت سیستم‌عامل (TUN Mode)
- **حالت TUN واقعی**: هدایت کل ترافیک سیستم از طریق درایور لایه ۳ پرسرعت وایرگارد (`wintun.dll`) در ساب‌نت `198.18.0.0/16` با اولویت متریک کارت شبکه.
- **فعالیت همزمان پراکسی سیستمی و تونل**: فعال نگه داشتن پراکسی ویندوز در کنار حالت TUN جهت تضمین عدم نشت ترافیک IPv6 و مرورگرها (کروم، اج، فایرفاکس).
- **حالت‌های مسیریابی هوشمند**:
  - **دور زدن ایران (Bypass Iran)**: عبور مستقیم دامنه‌های ملی (`.ir`)، درگاه‌های بانکی و وب‌سایت‌های داخلی بدون عبور از فیلترشکن.
  - **حالت سراسری (Global)**: عبور ۱۰۰٪ ترافیک کل سیستم از داخل تونل امن.
- **حالت کاهش پینگ مخصوص بازی (Gaming Mode)**: اولویت‌بندی پکت‌های UDP، تنظیم خودکار MTU روی ۱۴۰۰ و فعال‌سازی `TCP_NODELAY`.
- **جلوگیری از لوپ روتینگ**: مستثنی کردن خودکار پروسه‌های سایفون، ایثر و اس‌ان‌آی اسپوفینگ در روت‌های ویندوز برای جلوگیری از افتادن بسته‌ها در چرخه برگشتی شبکه.

#### ۳. اسکنر آی‌پی تمیز چند CDN (Multi-CDN Clean IP)
- **پشتیبانی از چندین CDN (Akamai, Cloudflare, CloudFront)**: اسکن پیشرفته رنج‌های Anycast با تشخیص خودکار نوع ارائه‌دهنده CDN.
- **اسکن هدفمند بر اساس کشور**: در صورت انتخاب یک کشور در نرم‌افزار، اسکنر دقیقاً آی‌پی‌های تمیز متعلق به همان کشور را به همراه نام و پرچم دقیق تست کرده و نمایش می‌دهد.
- **حالت هوشمند سریع‌ترین آی‌پی‌ها (Auto)**: اسکن و مرتب‌سازی سریع‌ترین آی‌پی‌های فعال در جهان بدون محدودیت جغرافیایی.
- **تنظیمات دقیق اسکن**: انتخاب تعداد ورکرها (۵۰ تا ۳۰۰ ترد)، تایم‌اوت پینگ (۱۰۰۰ تا ۲۵۰۰ میلی‌ثانیه) و اندازه نمونه آماری.
- **اتصال با یک کلیک**: اتصال مستقیم به آی‌پی کشف‌شده یا ذخیره آن در لیست کانفیگ‌ها.

#### ۴. کنسول و لاگ زنده سیستم (Live Logs)
- **استریم بیدرنگ با پروتکل SSE**: نمایش زنده و خط‌به‌خط پیام‌ها و خروجی‌های هسته Xray، سایفون، آداپتور TUN و سرور API.
- **امکانات خط فرمان**: فیلتر بر اساس منبع لاگ، جستجوی زنده، اسکرول خودکار، کپی با یک کلیک در کلیپ‌بورد و پاک‌سازی کنسول.

#### ۵. تجربه کاربری پیشرفته
- **داشبورد بومی و مدرن WebView2**: رابط کاربری شیک، دارک و سبک با مصرف کمتر از ۵۰ مگابایت رم.
- **طراحی حرفه‌ای**: عاری از ایموجی و سازگار با استانداردهای نرم‌افزارهای دسکتاپ مدرن.
- **ناحیه اعلان‌ها (System Tray)**: مینیمایز شدن در کنار ساعت ویندوز در هنگام بستن پنجره بدون قطع شدن اتصال.
- **ذخیره‌سازی پایدار**: حفظ پایدار تمامی تنظیمات، فیلترها و مقادیر در دیتابیس SQLite و حافظه محلی.

---

### ساخت و کامپایل از سورس‌کد

#### پیش‌نیازها
1. سیستم‌عامل **ویندوز ۱۰ یا ۱۱** نسخه ۶۴ بیتی
2. نصب بودن **Go نسخه ۱.۲۲ یا بالاتر** در متغیرهای محیطی سیستم (`PATH`)
3. ابزار **Git**

#### دستورات کامپایل در PowerShell

```powershell
# ساخت فایل اجرایی پرتابل بدون پنجره ترمینال
go build -ldflags="-H=windowsgui -s -w" -o Conective.exe ./cmd/conective

# یا اجرای اسکریپت بیلد کامل برای ساخت فایل نصبی Conective-Setup.exe
.\scripts\build.ps1
```

---

## Русский (Russian)

### Обзор

**Conective** — это современный, высокопроизводительный десктопный клиент для обхода цензуры под Windows, разработанный специально для работы в условиях жестких сетевых ограничений и блокировок (DPI). Написанный на Go с аппаратно-ускоренным интерфейсом на базе Microsoft Edge WebView2, Conective объединяет возможности **Xray-core**, **Psiphon / Aether**, реле **MHR Relay (Google Apps Script Fronting)**, технологии обхода DPI **WinDivert SNI Spoofing**, фронтинга чистых IP **Multi-CDN (Akamai, Cloudflare, CloudFront)** и низколатентного виртуального сетевого адаптера **Wintun TUN** в едином удобном приложении.

---

### Ключевые возможности

#### 1. Мульти-движковый обход блокировок
- **Интеграция Xray Core**: Встроенная поддержка современных прокси-протоколов: **VLESS**, **VMess**, **Trojan**, **Shadowsocks**, **VLESS-Reality**, а также исходящих подключений **SOCKS5** и **HTTP**.
- **MHR Relay (Google Apps Script & CFW)**: Система фронтинга через сервисы Google (`www.google.com`, `mail.google.com`, `accounts.google.com`) с поддержкой скриптов Google и Cloudflare Worker, автоматической установкой сертификата Root CA и сканером Anycast IP Google.
- **Манипуляция SNI (SNI Spoofing)**: Обход DPI на базе WinDivert с подменой заголовков TLS ClientHello на уровне сетевых пакетов (L3/L4).
- **Движок Psiphon & Aether**: Встроенный прямой движок Psiphon, работающий в высокопроизводительном режиме с автоматическим подбором тактик обхода DPI и ядром ShiroKhorshid.
- **Выбор страны выхода (Egress Region)**: Прямое подключение через 20 проверенных стран или интеллектуальный режим **Авто (Самый быстрый)**.
- **Единое хранилище тактик**: Централизованная база данных (`psiphon.boltdb` / `psiphon_server_list.dat`), используемая всеми регионами, сохраняет успешные пути обхода DPI и сокращает время установки соединения до 2–5 секунд.
- **Фрагментация заголовков TLS (SNI Fragmentation)**: Стандартное расщепление записей TLS ClientHello (`packets: 1-1`) для обхода систем глубокого анализа пакетов (DPI) без потери пакетов и задержек.
- **Защита от подмены DNS и блокировки QUIC**: Встроенный механизм DNS-over-HTTPS (DoH) против отравления DNS; автоматическая блокировка UDP-порта 443 (QUIC) для предотвращения замедления трафика провайдерами.

#### 2. Продвинутая маршрутизация и режим TUN
- **Полноценный системный режим TUN**: Высокоскоростной виртуальный сетевой интерфейс 3-го уровня (L3) на базе драйвера WireGuard `wintun.dll` (подсеть `198.18.0.0/16`, приоритетная метрика интерфейса).
- **Одновременная работа системного прокси и TUN**: Поддержание системного прокси Windows активным вместе с режимом TUN для гарантированного предотвращения утечек IPv6 и браузерного трафика.
- **Интеллектуальные режимы маршрутизации**:
  - **Обход локального трафика (Bypass Iran / Local)**: Прямая маршрутизация национальных доменов, банковских шлюзов и локальных IP-диапазонов в обход прокси.
  - **Глобальный режим (Global)**: Перенаправление 100% системного трафика через активный туннель.
- **Игровой режим с низким пингом (Gaming Mode)**: Приоритет UDP-сокетов, оптимизация MTU (1400 в игровом режиме вместо 1500 по умолчанию) и `TCP_NODELAY` для минимального пинга в онлайн-играх.
- **Предотвращение петель маршрутизации**: Исключение процессов движков обхода (`psiphon-tunnel-core.exe`, `aether.exe`, `sni-spoofing.exe`) из маршрутизации туннеля для предотвращения сетевых петель.

#### 3. Сканер и фронтинг чистых IP Multi-CDN (Clean IP)
- **Поддержка нескольких CDN (Akamai, Cloudflare, CloudFront)**: Специализированный сканер Anycast CIDR с автоматическим определением провайдера CDN.
- **Сканирование с фильтром по стране**: При выборе страны сканер находит и тестирует чистые IP исключительно для указанной страны с точными названиями и флагами.
- **Авто-режим (Самый быстрый)**: Поиск быстрейших глобальных IP-адресов без географических ограничений.
- **Настраиваемые параметры сканирования**: Выбор количества потоков (от 50 до 300), таймаута проверки (от 1000 до 2500 мс) и размера выборки.
- **Подключение в один клик**: Мгновенное подключение к найденному чистому IP или добавление его в сохраненный список узлов.

#### 4. Журнал событий в реальном времени (Live Logs)
- **Потоковая передача через SSE**: Диагностический терминал, отображающий stdout/stderr движков Psiphon, Aether, Xray, адаптера TUN и REST API в реальном времени.
- **Инструменты управления**: Фильтрация по источнику, поиск по тексту, автопрокрутка, копирование в буфер обмена в один клик и очистка консоли.

#### 5. Нативный интерфейс для Windows
- **Интерфейс WebView2**: Стильный темный дашборд с потреблением оперативной памяти менее 50 МБ.
- **Лаконичный дизайн**: Строгий профессиональный стиль без эмодзи.
- **Системный трей**: Сворачивание в область уведомлений Windows при закрытии с сохранением фонового соединения.
- **Сохранение состояния**: Надежное сохранение настроек, фильтров и параметров в SQLite (режим WAL) и локальном хранилище.

---

### Сборка из исходного кода

#### Требования
1. **Windows 10 / 11** (64-бит)
2. **Go 1.22+** (добавленный в системный `PATH`)
3. **Git** для Windows

#### Команды сборки в PowerShell

```powershell
# Сборка портативного исполняемого файла
go build -ldflags="-H=windowsgui -s -w" -o Conective.exe ./cmd/conective

# Или автоматическая сборка Conective.exe и установщика Conective-Setup.exe
.\scripts\build.ps1
```

---

## 中文 (Chinese)

### 概述

**Conective** 是一款专为 Windows 设计的高性能、多引擎桌面端抗审查与网络穿透客户端，特别针对高度网络封锁环境（如深度包检测 DPI 防火墙）进行了深度优化。基于 Go 语言开发，并采用硬件加速的 Microsoft Edge WebView2 构建现代化仪表盘，Conective 将 **Xray-core**、**Psiphon / Aether**、**MHR Relay（Google Apps Script 域前置中继）**、**WinDivert SNI 伪装**、**多 CDN 优选 IP 前置（Akamai, Cloudflare, CloudFront）** 以及低延迟 **Wintun TUN** 虚拟网卡技术融为一体，提供稳定、高速且无缝的桌面翻墙与穿透体验。

---

### 核心特性

#### 1. 多引擎抗审查架构
- **Xray Core 原生集成**: 完整支持现代主流代理协议，包括 **VLESS**、**VMess**、**Trojan**、**Shadowsocks**、**VLESS-Reality** 以及 **SOCKS5** 和 **HTTP** 出站代理。
- **MHR Relay (Google Apps Script & CFW)**: 基于谷歌服务前置（`www.google.com`, `mail.google.com`, `accounts.google.com`）的高级域前置中继系统，支持 Google Apps Script 与 Cloudflare Worker 双后端，集成 Google Anycast IP 快速扫描和 Root CA 根证书自动信任安装。
- **WinDivert SNI 伪装 (SNI Spoofing)**: 基于 WinDivert 的底层 L3/L4 数据包操控引擎，在数据链路层实时替换 TLS ClientHello 中的真实域名为白名单域名，绕过深度包检测。
- **Psiphon & Aether 引擎**: 内置直连 Psiphon 穿透引擎，以高性能模式运行并支持智能自动化战术协商与 ShiroKhorshid 原生内核。
- **出口地区自由选择 (Egress Region)**: 支持直接选择 20 个主流国家出口节点，亦可使用 **Auto（最快节点）** 智能路由。
- **统一战术数据存储**: 所有地区共享集中式绕过策略数据库 (`psiphon.boltdb` / `psiphon_server_list.dat`)，自动记忆成功突破 DPI 的握手路径，将连接建立时间缩短至 2–5 秒。
- **抗 DPI 的 TLS SNI 分片 (SNI Fragmentation)**: 遵循 RFC 标准的 `packets: 1-1` TLS ClientHello SNI 分片技术，有效绕过深度包检测（DPI）防火墙，零丢包、低延迟。
- **防 DNS 污染与 QUIC 保护**: 内置 DNS-over-HTTPS (DoH) 备用机制抵御 DNS 投毒；自动阻断 UDP 443 (QUIC) 端口，防止运营商 QoS 限速干扰。

#### 2. 高级网络与系统级 TUN 路由
- **真·全局 TUN 模式**: 基于 WireGuard 高吞吐 L3 虚拟网卡驱动 `wintun.dll`（`198.18.0.0/16` 子网，高优先级接口跃点数）。
- **系统代理与 TUN 双重保障**: TUN 模式开启的同时保持 Windows 系统代理同步激活，彻底杜绝双栈网络下的 IPv6 或浏览器泄漏。
- **智能分流路由**:
  - **本地白名单分流 (Bypass Local / Iran)**: 自动直连本地域名、银行网银与本地 IP 段，仅对海外被墙流量进行隧道加速。
  - **全局代理 (Global)**: 100% 接管系统所有流量并通过安全隧道转发。
- **低延迟游戏模式 (Gaming Mode)**: UDP 套接字优先处理、自动优化 MTU（游戏模式 1400，默认 1500）并开启 `TCP_NODELAY`，显著降低联机游戏延迟。
- **防止路由环路 (Routing Loop Prevention)**: 自动将穿透进程（`psiphon-tunnel-core.exe`, `aether.exe`, `sni-spoofing.exe`）加入路由豁免，确保握手包绝不陷入虚拟网卡循环。

#### 3. 多 CDN 优选 Clean IP 扫描与前置
- **多 CDN 支持 (Akamai, Cloudflare, CloudFront)**: 针对高信誉 CDN Anycast CIDR 节点池的专用多线程扫描器，具备 CDN 厂商智能识别功能。
- **国家定向精准扫描**: 当用户在界面指定国家时，扫描器精准发现并测试归属该国家的优选 IP，展示正确的国家名称与国旗。
- **全局极速模式 (Auto)**: 全网范围内实时测速并优选延迟最低的节点，不受单一地区限制。
- **深度可定制扫描参数**: 自由调节并发工作线程（50 到 300）、握手超时（1000ms 到 2500ms）及候选样本数量。
- **一键直连与部署**: 扫描发现的高质量节点可一键直接连接，或快速保存至节点列表。

#### 4. 实时控制台日志 (Live Logs)
- **SSE 实时流式传输**: 诊断级控制台，实时流式输出来自 Psiphon、Aether、Xray、TUN 虚拟网卡及 REST API 的 stdout/stderr。
- **开发者调试工具**: 支持按来源过滤日志、全局文本搜索、自动滚屏开关、一键复制到剪贴板及控制台即时清屏。

#### 5. 原生 Windows 桌面体验
- **WebView2 现代仪表盘**: 典雅深色界面，内存占用通常低于 50MB。
- **专业克制美学**: 严格去除所有 Emoji 表情，符合工业级软件设计规范。
- **系统托盘 (System Tray)**: 关闭窗口时自动最小化至 Windows 任务栏通知区域，保持后台持续连接。
- **持久化配置管理**: 所有设置、过滤状态与偏好均通过 SQLite (WAL 模式) 和本地存储自动持久化。

---

### 从源码编译

#### 环境要求
1. **Windows 10 / 11** (64 位)
2. **Go 1.22+**（已配置到系统环境变量 `PATH`）
3. **Git** for Windows

#### PowerShell 编译命令

```powershell
# 编译便携版独立程序（无命令行黑框）
go build -ldflags="-H=windowsgui -s -w" -o Conective.exe ./cmd/conective

# 或运行自动化构建脚本，同时打包 Conective.exe 与 Conective-Setup.exe 安装包
.\scripts\build.ps1
```

---

## Security & Privacy | امنیت و حریم خصوصی | Безопасность и конфиденциальность | 安全与隐私

- **Open Source & Privacy**: 100% open-source, no telemetry, no tracking, and zero remote data collection.
- **حفظ حریم خصوصی**: نرم‌افزار ۱۰۰٪ متن‌باز بوده و فاقد هرگونه تله‌متری، جمع‌آوری داده یا گزارش‌دهی به سرورهای خارجی است.
- **Конфиденциальность**: Полностью открытый исходный код без какой-либо телеметрии, слежки или сбора данных.
- **开源与隐私**: 100% 纯开源代码，绝无任何遥测、跟踪或远程数据收集行为。
- **Crash Recovery**: Automatically restores Windows proxy settings and cleans up routing tables upon sudden exit.
- **Admin Privileges**: Administrator rights are only required for managing the Wintun virtual network adapter in TUN mode.

---

## License | مجوز | Лицензия | 开源协议

This project is licensed under the [MIT License](LICENSE).  
Xray-core, Psiphon, and Wintun are trademarks of their respective owners and distributed under their open-source licenses.
