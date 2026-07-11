# wizlink

Discord Rich Presence tool for Wizard101.
Displays your current location, combat status, playtime, and recent battle information.

> **Note:** Currently only available in German.

## 👀 Preview

| Roaming                                 | Combat                                |
| --------------------------------------- | ------------------------------------- |
| ![Roaming](public/presence_roaming.png) | ![Battle](public/presence_battle.png) |

## ✨ Features

* Displays current area and world

* Combat detection

* Character selection display

* Playtime tracking

* Web UI with live status information

* Displays the most recently defeated enemy

* Automatic translation data updates from a separate repository <br>

* Autostart (optional)

* Reporting of unknown areas (optional)

---

## 🖥️ Supported platforms

* Windows

## 📥 Download & Usage

1. Download the latest version from the **Releases** section.
2. Start `wizlink.exe`.
3. Manage the application via the system tray.

![System Tray](public/presence_tray.png)

### Web UI

wizlink starts a local web interface, available at:

**http://localhost:8081**

The Web UI provides live information about the current game session, including:

* Current character
* Current world and area
* Combat status
* Session playtime
* Most recently defeated enemy

### Manual path selection (optional)

If Wizard101 is not installed in the default directory, wizlink may not be able to detect the game automatically.

Select **"Select path manually"** from the system tray menu and navigate to your Wizard101 installation directory. Then select **`Wiz.ico`**.

After setting the path, wizlink will restart automatically.

### Translation data

Translation data is no longer bundled with the application.

Instead, wizlink automatically downloads and updates the latest translation data from the dedicated [wizlink-data](https://github.com/MeisterSchwarz/wizlink-data) repository.

### App data directory

Depending on usage, a directory is created automatically under:

**`%APPDATA%\wizlink`**

The application stores data such as:

- `config.json`      – application configuration
- `enemy_names.json` – local name translations (not yet synchronized with the repository)
- `enemy_zones.json` – local zone translations (not yet synchronized with the repository)
---

## 🛡️ Windows Defender notice

This application is **not code signed**.

On first launch, Windows SmartScreen or Windows Defender may display a warning.

This is expected because the application:

* runs in the background
* supports autostart
* accesses local Wizard101 files
* hosts a local web interface
* monitors a scheduled task as part of its watchdog functionality

Select **"More info" → "Run anyway"** to continue.

The complete source code is available in this repository.

---

## 🔒 Privacy

* **No personal data is collected**
* **No user accounts** are required
* **No gameplay information is sent to external services**
* Network communication is limited to:

  * Discord Rich Presence
  * Downloading translation data
  * The local Web UI (`localhost`)
* All processing is performed **locally on your PC**

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## 📄 License

MIT License

> This project is not affiliated with KingsIsle Entertainment or Wizard101.
