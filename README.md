# RavenDex

A companion application for **Wizard101** featuring Discord Rich Presence, a local knowledge base, and community-driven game data collection.

RavenDex displays your current in-game status while helping build an open database of Wizard101 information.

> **Note:** Currently only available in German.

---

## 👀 Preview

| Roaming | Combat |
| ------- | ------ |
| ![Roaming](public/presence_roaming.png) | ![Battle](public/presence_battle.png) |

---

## ✨ Features

### 🎮 Discord Rich Presence

- Displays your current world and area
- Detects combat automatically
- Shows your most recently defeated enemy

### 🌐 Local Web Interface

The integrated Web UI provides live information about your current game session and allows you to contribute to the shared RavenDex dataset.

Current functionality:

- Live game status
- Current world and area
- Combat status
- Recently defeated enemy
- Rename unknown enemies
- Rename unknown areas

### 📚 Shared Community Data

RavenDex uses the separate **[ravendex-data](https://github.com/MeisterSchwarz/ravendex-data)** repository for all game data.

This includes:

- Game IDs
- Localizations
- Community translations

Newly identified entities can be prepared for submission to the shared repository directly from the Web UI.

### ⚙️ Additional Features

- Optional autostart
- Automatic game detection
- Local configuration

---

## 🚧 Planned Features

The following features are planned for future releases:

- 🔎 Item search
- 📦 Community-driven drop reporting
- 🌍 Improved area naming workflow
- 🌐 Additional language support
- 📊 Community statistics

---

## 🖥️ Supported Platforms

- Windows

---

## 📥 Download & Usage

1. Download the latest version from the **Releases** page.
2. Start `RavenDex.exe`.
3. Manage the application through the system tray.

![System Tray](public/presence_tray.png)

---

## 🌐 Web Interface

RavenDex hosts a local web interface available at:

**http://localhost:8081**

The Web UI is designed to become the central place for interacting with community data.

Current pages include:

- Live session overview
- Unknown enemy identification
- Unknown area identification

Future versions will expand the Web UI with additional tools such as item search and drop reporting.

---

## 📂 Game Data

Game data is **not bundled** with RavenDex.

Instead, RavenDex automatically downloads the latest dataset from the dedicated **[ravendex-data](https://github.com/MeisterSchwarz/ravendex-data)** repository.

Separating the application from the data allows both projects to evolve independently while making the data available for other community projects as well.

---

## 📁 App Data Directory

Depending on usage, RavenDex creates the following directory:

**`%APPDATA%\ravendex`**

Typical files include:

- `config.json` – application configuration
- `enemy_names____.json` – local enemy names awaiting submission

---

## 🛡️ Windows Defender Notice

This application is **not code signed**.

On first launch, Windows SmartScreen or Windows Defender may display a warning.

This is expected because the application:

- runs in the background
- supports autostart
- accesses local Wizard101 files
- hosts a local web interface

Select **More info → Run anyway** to continue.

The complete source code is available in this repository.

---

## 🔒 Privacy

RavenDex is designed to process everything locally.

- No personal data is collected.
- No user accounts are required.
- No gameplay information is sent to external services.

Network communication is limited to:

- Discord Rich Presence
- Downloading community data from `ravendex-data`
- Accessing the local Web UI (`localhost`)

---

## 🤝 Contributing

Contributions are always welcome.

Game data contributions are handled through the **ravendex-data** repository, while application improvements can be submitted here.

Please see **CONTRIBUTING.md** for more information.

---

## 📄 License

MIT License

> RavenDex is an independent community project and is not affiliated with KingsIsle Entertainment or Wizard101.
