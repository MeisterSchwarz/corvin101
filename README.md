# wizard101rpc 

Discord Rich Presence tool for Wizard101.  
Displays current area, combat status, and playtime.
Currently only available in German. 

## 👀 Preview

| Roaming | Combat |
|--------|--------|
| ![Roaming](public/presence_roaming.png) | ![Battle](public/presence_battle.png) |

## ✨ Features

- Displays current area and world
- Combat detection
- Character selection display  
<br>

- Autostart (optional)
- Reporting of unknown areas (optional)

---

## 🖥️ Supported platforms

- Windows

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
- a configuration file (config.json)
- a log file if unknown areas are detected (missing_zones.log)

---

## 🛡️ Windows Defender notice

This application is **not signed**.  
On first launch, Windows SmartScreen or Windows Defender may display a warning.

This may occur because the application runs in the background, supports autostart, accesses local files, and monitors a scheduled task as part of its watchdog functionality.

Select **“More info” → “Run anyway”**.

The full source code is available in this repository.

## 🔒 Privacy

- **No data is collected or transmitted**
- **No accounts** are required
- **No network communication** takes place  
  (except for local communication with Discord)
- All data is processed **locally on your PC**


---

## 🤝 Contributing

👉 Details on how to contribute can be found [here](CONTRIBUTING.md).

## 📄 License

MIT License

> This project is not affiliated with KingsIsle Entertainment or Wizard101.
