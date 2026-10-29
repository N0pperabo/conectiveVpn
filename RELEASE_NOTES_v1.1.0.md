# Conective v1.1.0 Release Notes

**Release Date:** October 2026  
**Platform:** Windows 10 / 11 (64-bit)  
**Binary Downloads:**
- Portable Client: `Conective.exe`
- Full Installer: `Conective-Setup.exe`

---

## English

### Overview
Conective v1.1.0 is a major feature and stability release designed to maintain connectivity even under extreme network censorship, throttling, and Deep Packet Inspection (DPI) conditions. This release introduces the MHR Relay subsystem, WinDivert SNI Spoofing, multi-CDN clean IP scanning with country targeting, and full Xray SOCKS5/HTTP outbound integration.

---

### New Features (What was Added)

#### 1. MHR Relay Subsystem (Google Apps Script & Cloudflare Worker)
- **Domain-Fronting Relay:** Tunnels traffic through high-reputation Google domains (`www.google.com`, `mail.google.com`, `accounts.google.com`), bypassing standard protocol blocks.
- **Dual Engine Modes:** Supports both standard Google Apps Script backend (`mhr`) and Cloudflare Worker proxy backend (`mhr-cfw`).
- **Real-Time Google Anycast IP Scanner:** Built-in scanner to test and rank reachable, low-latency Google frontend IPs.
- **Automated Root CA Trust Management:** Verifies and installs the internal MITM CA certificate into the Windows user certificate store without UAC elevation, enabling seamless HTTPS and YouTube video decryption.
- **Local Relay Node Integration:** Automatically registers `MHR Local Relay (MHR)` in the saved node list for unified 1-click connection through Xray and TUN mode.

#### 2. WinDivert SNI Spoofing Engine
- **Kernel-Level Packet Manipulation:** Intercepts outgoing TLS ClientHello packets at the Layer 3/4 network boundary via the WinDivert filter driver.
- **SNI Injection:** Transparently replaces DPI-blocked Server Name Indications with permitted hosts (such as `auth.vercel.com`).
- **Flexible Target Routing:** Supports custom IP and port targeting with automated background service recovery.

#### 3. Multi-CDN Clean IP Fronting & Country Filter
- **Expanded Multi-CDN Support:** Automated detection and fronting across Akamai, Cloudflare, and CloudFront edge networks.
- **Strict Country-Targeted Scanning:** When a specific egress country is selected, the scanner tests and outputs clean IPs exclusively located in that country, complete with accurate country names and flags.
- **Auto (Fastest) Global Scan:** Rapidly discovers and ranks top-performing edge IPs worldwide without regional limits.

#### 4. SOCKS5 & HTTP Outbounds in Xray Core
- **Outbound Converter Support:** Added native conversion for `socks` and `http` protocols in `ConvertToXrayOutbound`, enabling seamless chaining between Xray (TUN mode or system proxy) and local or external upstream proxies.

---

### Fixes & Improvements (What was Resolved compared to v1.0.0)

1. **Suppressed Console Windows:**
   - Completely resolved terminal windows popping up during application launch and background engine execution.
   - Recompiled all binaries with `-ldflags="-H=windowsgui -s -w"` and configured child processes with `CREATE_NO_WINDOW`.

2. **Fixed Xray Outbound Converter Errors:**
   - Resolved `protocol socks not supported by Xray outbound converter` when selecting the local MHR relay node.
   - Fixed TUN mode socket option handling by exempting loopback addresses (`127.0.0.1`) from physical network interface binding.

3. **Fixed MHR System Proxy & YouTube Playback:**
   - Enabled Windows System Proxy routing by default when starting MHR relay directly from the modal.
   - Automatically ensured the local Root CA is trusted so browsers no longer block HTTPS or YouTube connections with certificate authority warnings.
   - Fixed syntax and error-handling bugs in `relay_response.py` for Google Apps Script HTML responses.

4. **Improved Single-File Installer:**
   - Updated `cmd/installer` to embed all circumvention tools, runtime drivers (`wintun.dll`), and GeoIP database into `Conective-Setup.exe`.

---

## فارسی (Persian)

### نمای کلی نسخه ۱.۱.۰
نسخه ۱.۱.۰ کانکتیو یک به‌روزرسانی بزرگ و بنیادین برای شرایط اختلالات شدید، فیلترینگ گسترده و اعمال محدودیت‌های سخت‌گیرانه فایروال‌های DPI است. در این نسخه سامانه‌های MHR Relay، سیستم ضد سانسور WinDivert SNI Spoofing، اسکنر چندگانه آی‌پی تمیز با تفکیک کشور و پشتیبانی از پروتکل‌های خروجی SOCKS5/HTTP به هسته برنامه افزوده شده‌اند.

---

### قابلیت‌های جدید (موارد اضافه شده)

#### ۱. سامانه رله MHR (Google Apps Script & CFW)
- **دامین فرانتینگ بر بستر گوگل:** عبور ترافیک از بستر سرویس‌های گوگل (`www.google.com`, `mail.google.com`, `accounts.google.com`) جهت دور زدن کامل اختلالات و فیلترینگ پروتکل‌ها.
- **دو حالت کاری متنوع:** پشتیبانی از اسکریپت گوگل (`mhr`) و حالت هیبریدی با ورکر کلودفلر (`mhr-cfw`).
- **اسکنر اختصاصی آی‌پی‌های Anycast گوگل:** اسکن و ارزیابی پینگ سریع‌ترین آی‌پی‌های سالم فرانتینگ گوگل در لحظه.
- **نصب خودکار گواهی Root CA:** بررسی و نصب خودکار گواهی امنیتی در مخزن ویندوز برای باز شدن روان و بدون خطای یوتیوب و سایت‌های تحت HTTPS.
- **یکپارچه‌سازی نود در لیست سرورها:** افزوده شدن خودکار نود `MHR Local Relay` به منوی سرورها برای اتصال سریع با یک کلیک در حالت TUN و سیستم پروکسی.

#### ۲. تکنولوژی فریب هدر SNI با درایور WinDivert
- **تغییر پکت‌ها در لایه ۳ و ۴ شبکه:** رهگیری و تغییر بسته‌های ClientHello پروتکل TLS بدون افت سرعت.
- **تزریق SNI مجاز:** تعویض آدرس سایت‌های مسدود شده با دامنه‌های سفید (مانند `auth.vercel.com`) در سطح شبکه.
- **پشتیبانی از پورت و آی‌پی سفارشی:** امکان هدایت ترافیک مستقیم به سمت سرور مقصد دلخواه.

#### ۳. اسکنر آی‌پی تمیز چند CDN با فیلتر کشورها
- **پشتیبانی از ۳ شبکه CDN بزرگ:** اسکن خودکار شبکه‌های Akamai، Cloudflare و CloudFront به همراه تشخیص خودکار نوع شبکه.
- **اسکن هدفمند بر اساس کشور انتخابی:** اسکن و یافتن دقیق آی‌پی‌های تمیز متعلق به همان کشور انتخابی کاربر همراه با پرچم و عنوان صحیح کشور.
- **حالت خودکار سریع‌ترین‌ها (Auto):** جستجو و رتبه‌بندی سریع‌ترین آی‌پی‌های بدون محدودیت جغرافیایی در سراسر جهان.

#### ۴. پشتیبانی از خروجی‌های SOCKS و HTTP در Xray
- **مبدل خروجی Xray:** پشتیبانی کامل از پروتکل‌های `socks` و `http` جهت اتصال ترافیک TUN یا سیستم پروکسی به رله‌های محلی یا پروکسی‌های خارجی.

---

### بهبودها و رفع اشکالات نسبت به نسخه ۱.۰.۰

۱. **حذف کامل باز شدن پنجره‌های ترمینال:**
   - رفع باز شدن ناخواسته پنجره‌های مشکی کنسول و ترمینال موقع اجرای برنامه و بیلد تمیز با پارامترهای گرافیکی ویندوز (`-ldflags="-H=windowsgui -s -w"`).

۲. **برطرف شدن خطای اتصال نود رله لوکال:**
   - حل مشکل `protocol socks not supported by Xray outbound converter`.
   - مستثنی کردن آدرس‌های لوکال (`127.0.0.1`) از بایند شدن به کارت شبکه فیزیکی در حالت TUN.

۳. **اصلاح پروکسی سیستم و رفع مشکل پخش یوتیوب در MHR:**
   - فعال‌سازی خودکار تیک پروکسی سیستم ویندوز هنگام روشن شدن رله.
   - ثبت خودکار Root CA در ویندوز برای رفع خطای امنیتی مرورگرها در پخش یوتیوب.
   - برطرف شدن خطاهای نحوی در پردازش پاسخ‌های HTML گوگل اپس اسکریپت.

۴. **بهینه‌سازی ستاپ نصبی:**
   - ارتقای فایل ستاپ تک‌فایلی [Conective-Setup.exe](file:///d:/conective/Conective/Conective-Setup.exe) جهت استخراج کامل درایورها، پایگاه داده GeoIP و باینری‌های دور زدن سانسور.
