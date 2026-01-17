# wizard101rpc 

[![DE](https://img.shields.io/badge/DE-red)](README.md) [![EN](https://img.shields.io/badge/EN-blue)](README.en.md)

Discord Rich Presence tool for Wizard101.  
Automatically displays your current area, combat status, and playtime.
Currently only available in German. 

## 👀 Preview

| Roaming | Combat |
|--------|--------|
| ![Roaming](public/presence_roaming.png) | ![Battle](public/presence_battle.png) |

## ✨ Features

- Displays current area and world
- Combat detection (⚔️)
- Character selection display  
<br>

- Autostart (optional)
- Reporting of unknown areas (optional)

---

## 📥 Download & Usage

1. Download the latest version from the **Releases** section
2. Start `wizard101rpc.exe`
3. Manage the application via the system tray  
![System Tray](public/presence_tray.png)

### Manual path selection (optional)

If Wizard101 is not installed in the default directory, wizard101rpc may not be able to detect the game automatically.

In this case, select  
**“Select path manually”** from the system tray menu and navigate to the Wizard101 installation directory.  
There, select the file **`Wiz.ico`**.

After setting the path, wizard101rpc will automatically restart.

### App data directory

Depending on usage, a directory will be created automatically under  
**`%APPDATA%\wizard101rpc`**.

The application stores the following files there:
- a configuration file
- a log file if unknown areas are detected

---

## 🛡️ Windows Defender notice

This application is **not signed**.  
On first launch, Windows SmartScreen or Windows Defender may display a warning.

This is a known false positive for applications that:
- run in the background
- offer an autostart option

Select **“More info” → “Run anyway”**.

The full source code is available in this repository.

## 🔒 Privacy

- **No data is collected or transmitted**
- **No accounts** are required
- **No network communication** takes place  
  (except for local communication with Discord)
- All data is processed **locally on your PC**

## 🖥️ Supported platforms

- Windows

---

## 🤝 Contributing

Contributions are welcome — whether bug fixes, feature ideas, or new area translations. 

👉 Details on how to contribute can be found [here](CONTRIBUTING.en.md).

## 📄 License

MIT License

> This project is not affiliated with KingsIsle Entertainment or Wizard101.
