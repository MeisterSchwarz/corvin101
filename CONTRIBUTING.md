# Contributing to wizard101rpc

Contributions of any kind are welcome — whether code, feedback, or translations.

## Ways to contribute

### 🐞 Reporting bugs & suggesting features
- Please use GitHub **Issues**
- Try to describe as clearly as possible:
  - What happened?
  - What did you expect to happen?

---

### 🌍 Reporting missing areas

wizard101rpc automatically detects unknown areas when the option  
**“Fehlende Übersetzungen sammeln”** is enabled in the system tray.

In this case, a local file named *missing_zones.log* is created.  
The file is located in *%APPDATA%\wizard101rpc* (you can copy this path and paste it into the Windows file explorer).

Please add your translation suggestion **on the same line**, after the existing entry.

You can submit the entries:
- by opening an Issue
- or by contacting me directly on Discord (**leonuni**)

---

### 🧑‍💻 Local development

If you want to work on the project yourself, feel free to fork the repository and develop locally.  
Once you are happy with your changes, please open a pull request.

Please note:
- no formatting-only commits without functional changes
- keep pull requests small and focused on a single topic

Build from the terminal:
```bash
go build -ldflags="-H=windowsgui -s -w" -o wizard101rpc.exe
