# Mitwirken an wizard101rpc

Vielen Dank für dein Interesse an wizard101rpc!  
Beiträge in jeder Form sind willkommen – egal ob Code, Feedback oder Übersetzungen.

## Möglichkeiten zur Mitarbeit

### 🐞 Fehler melden & Feature vorschlagen
- Bitte nutze die **Issues** auf GitHub
- Beschreibe möglichst genau:
  - Was passiert?
  - Was hättest du erwartet?

---

### 🌍 Fehlende Gebiete melden

wizard101rpc erkennt unbekannte Gebiete automatisch, wenn die Option  
**„Fehlende Übersetzungen sammeln“** im Tray aktiviert ist.

Dabei wird lokal eine Datei *missing_zones.log* erzeugt. Diese befindet sich im Verzeichnis *%APPDATA%\wizard101rpc* (diesen Pfad kopieren und im Dateiexplorer einfügen). Bitte schreibe zeilenweise deinen Übersetzungsvorschlag dahinter

Du kannst die Einträge:
- als Issue posten
- oder direkt per Discord (**leonuni**) weitergeben

---

### 🧑‍💻 Selbst entwickeln

Wenn du selbst an dem Projekt arbeiten möchtest, kannst du das Repository forken und lokal weiterentwickeln. Wenn du zufrieden bist erstelle gerne einen Pull Request

Bitte:
- keine Formatierungs-Commits ohne funktionale Änderung
- möglichst kleine, thematisch fokussierte PRs

Build im Terminal:
```bash
go build -ldflags="-H=windowsgui -s -w" -o wizard101rpc.exe
```
Damit wird eine Windows-EXE ohne sichtbares Terminal erzeugt
