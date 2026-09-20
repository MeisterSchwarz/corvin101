# Contributing to corvin101

Contributions of any kind are welcome — whether code, feedback, or translations.

## Ways to contribute

### 🐞 Reporting bugs & suggesting features

* Please use GitHub **Issues**.
* Try to describe as clearly as possible:

  * What happened?
  * What did you expect to happen?

---

### 🌍 Translation data

Area and enemy translations are maintained in the dedicated **ravendex-data** repository:

https://github.com/MeisterSchwarz/ravendex-data
---

### 🧑‍💻 Local development

If you want to work on the project yourself, feel free to fork the repository and develop locally.
Once you are happy with your changes, please open a pull request.

Please note:

* no formatting-only commits without functional changes
* keep pull requests small and focused on a single topic

Build from the terminal:

```bash
go build -ldflags="-H=windowsgui" -o corvin.exe ./cmd/corvin101
```
