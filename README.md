# MCP-Mail

MCP (Model Context Protocol) server pro přístup k IMAP e-mailovým schránkám. Umožňuje AI asistentům (Claude, atd.) číst, spravovat a organizovat e-maily.

## Funkce

- **Multi-account** - podpora více IMAP účtů současně
- **Správa složek** - vytváření, mazání, přejmenování, přihlášení/odhlášení složek
- **Správa zpráv** - čtení, mazání, přesouvání, kopírování e-mailů
- **Příznaky** - označování jako přečtené/nepřečtené, hvězdičky, vlastní příznaky
- **Vyhledávání** - plnotextové vyhledávání, filtrování podle data, odesílatele, předmětu
- **Dva transporty** - stdio (pro Claude Desktop) a SSE (HTTP server s autentizací)
- **Cache** - inteligentní cachování pro snížení zátěže IMAP serveru
- **Bezpečné mazání** - mazání e-mailů pouze přesune do koše (nelze vysypat koš přes MCP)

## Instalace

### Požadavky

- Go 1.21 nebo novější

### Kompilace

```bash
git clone https://github.com/spetr/mcp-mail.git
cd mcp-mail
go build -o mcp-mail .
```

## Konfigurace

### 1. Vytvoření konfiguračního souboru

Zkopírujte vzorový konfigurační soubor:

```bash
cp config.example.json config.json
```

### 2. Nastavení IMAP účtu

Upravte `config.json`:

```json
{
  "accounts": [
    {
      "id": "gmail",
      "name": "Můj Gmail",
      "host": "imap.gmail.com",
      "port": 993,
      "tls": true,
      "username": "vas-email@gmail.com",
      "password": "vase-aplikacni-heslo"
    }
  ],
  "settings": {
    "auto_connect": true
  },
  "server": {
    "transport": "stdio"
  },
  "cache": {
    "enabled": true,
    "folder_list_ttl_sec": 300,
    "folder_info_ttl_sec": 120
  }
}
```

**Důležité:** Konfigurační soubor obsahuje citlivé údaje (hesla). Ujistěte se, že:
- Soubor není součástí git repozitáře (přidejte do `.gitignore`)
- Má správná oprávnění (`chmod 600 config.json`)

#### Gmail - App Password

Pro Gmail je nutné použít "App Password":

1. Přejděte na [Google Account Security](https://myaccount.google.com/security)
2. Zapněte 2-faktorovou autentizaci (pokud není)
3. Vytvořte App Password: Security → 2-Step Verification → App passwords
4. Vyberte "Mail" a "Other (Custom name)"
5. Zkopírujte vygenerované heslo do konfigurace

#### Outlook/Microsoft 365

```json
{
  "id": "outlook",
  "name": "Outlook",
  "host": "outlook.office365.com",
  "port": 993,
  "tls": true,
  "username": "vas-email@outlook.com",
  "password": "vase-heslo"
}
```

#### Vlastní IMAP server

```json
{
  "id": "vlastni",
  "name": "Firemní mail",
  "host": "mail.example.com",
  "port": 993,
  "tls": true,
  "username": "user@example.com",
  "password": "vase-heslo"
}
```

## Integrace s Claude Desktop

### 1. Najděte konfigurační soubor Claude Desktop

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`

### 2. Přidejte MCP server do konfigurace

Otevřete `claude_desktop_config.json` a přidejte:

```json
{
  "mcpServers": {
    "mail": {
      "command": "/absolutni/cesta/k/mcp-mail",
      "args": ["-config", "/absolutni/cesta/k/config.json"]
    }
  }
}
```

**Příklad pro macOS:**

```json
{
  "mcpServers": {
    "mail": {
      "command": "/Users/username/mcp-mail/mcp-mail",
      "args": ["-config", "/Users/username/mcp-mail/config.json"]
    }
  }
}
```

**Příklad pro Windows:**

```json
{
  "mcpServers": {
    "mail": {
      "command": "C:\\Users\\username\\mcp-mail\\mcp-mail.exe",
      "args": ["-config", "C:\\Users\\username\\mcp-mail\\config.json"]
    }
  }
}
```

### 3. Restartujte Claude Desktop

Po uložení konfigurace restartujte Claude Desktop aplikaci. MCP-Mail server se automaticky připojí.

### 4. Ověření

V Claude Desktop byste měli vidět dostupné IMAP nástroje. Zkuste:

> "Připoj se k mému Gmail účtu a ukaž mi nepřečtené e-maily v INBOXu"

## Dostupné nástroje

### Správa účtů

| Nástroj | Popis |
|---------|-------|
| `account_list` | Seznam všech nakonfigurovaných účtů a jejich stav |
| `account_add` | Přidání nového IMAP účtu |
| `account_remove` | Odebrání účtu |
| `account_connect` | Připojení k účtu |
| `account_disconnect` | Odpojení od účtu |
| `account_status` | Stav připojení účtu |

### Správa složek

| Nástroj | Popis |
|---------|-------|
| `folder_list` | Seznam všech složek v účtu |
| `folder_info` | Detailní informace o složce (počet zpráv, nepřečtené, atd.) |
| `folder_create` | Vytvoření nové složky |
| `folder_rename` | Přejmenování složky |
| `folder_delete` | Smazání složky |
| `folder_subscribe` | Přihlášení ke složce |
| `folder_unsubscribe` | Odhlášení od složky |

### Správa zpráv

| Nástroj | Popis |
|---------|-------|
| `message_list` | Seznam zpráv ve složce (s stránkováním) |
| `message_get` | Načtení kompletní zprávy včetně těla a příloh |
| `message_get_headers` | Načtení pouze hlaviček (rychlejší) |
| `message_delete` | Přesunutí zprávy do koše |
| `message_move` | Přesunutí zprávy do jiné složky |
| `message_copy` | Kopírování zprávy do jiné složky |

### Příznaky zpráv

| Nástroj | Popis |
|---------|-------|
| `message_set_flags` | Nastavení příznaků (nahradí existující) |
| `message_add_flags` | Přidání příznaků |
| `message_remove_flags` | Odebrání příznaků |
| `message_mark_read` | Označení jako přečtené |
| `message_mark_unread` | Označení jako nepřečtené |
| `message_flag` | Přidání hvězdičky |
| `message_unflag` | Odebrání hvězdičky |
| `message_get_flags` | Získání aktuálních příznaků |

### Vyhledávání

| Nástroj | Popis |
|---------|-------|
| `search` | Komplexní vyhledávání s více kritérii |
| `search_unread` | Rychlé vyhledání nepřečtených zpráv |
| `search_flagged` | Rychlé vyhledání označených zpráv |
| `search_from` | Vyhledání podle odesílatele |
| `search_subject` | Vyhledání podle předmětu |
| `search_today` | Vyhledání dnešních zpráv |
| `search_date_range` | Vyhledání v rozsahu dat |

## Příklady použití

### Základní práce s e-maily

```
"Připoj se k účtu gmail a zobraz posledních 10 zpráv z INBOXu"

"Najdi všechny nepřečtené e-maily"

"Přečti e-mail s UID 12345"

"Označ zprávu 12345 jako přečtenou"

"Přesuň zprávu 12345 do složky Archive"
```

### Vyhledávání

```
"Najdi všechny e-maily od john@example.com"

"Vyhledej e-maily s předmětem 'faktura'"

"Najdi e-maily z posledního týdne"

"Vyhledej e-maily větší než 5MB"
```

### Organizace

```
"Vytvoř složku 'Projekty/2024'"

"Přesuň všechny e-maily od newsletter@example.com do složky Newsletters"

"Smaž všechny přečtené e-maily starší než 30 dní ve složce Trash"
```

## SSE Transport (volitelné)

Pro použití přes HTTP s autentizací:

### Spuštění serveru

```bash
./mcp-mail -transport sse -port 8080 -auth-token "tajny-token"
```

### Konfigurace v config.json

```json
{
  "server": {
    "transport": "sse",
    "host": "localhost",
    "port": 8080,
    "auth": {
      "type": "bearer",
      "token": "tajny-token"
    }
  }
}
```

### Podporované metody autentizace

- **Bearer Token:** `Authorization: Bearer <token>`
- **Basic Auth:** `Authorization: Basic <base64(user:pass)>`
- **API Key:** `X-API-Key: <key>`

## Řešení problémů

### "failed to connect: ..."

- Zkontrolujte správnost IMAP serveru a portu
- Ověřte, že používáte správné přihlašovací údaje
- Pro Gmail použijte App Password, ne běžné heslo

### "account not found"

- Nejprve se připojte k účtu pomocí `account_connect`
- Zkontrolujte, že ID účtu odpovídá konfiguraci

### "folder not found"

- Použijte `folder_list` pro zobrazení dostupných složek
- Názvy složek jsou case-sensitive
- Některé servery používají odlišné názvy (např. "[Gmail]/Trash")

### Claude Desktop nevidí MCP server

- Zkontrolujte absolutní cesty v konfiguraci
- Ověřte, že binárka má práva ke spuštění (`chmod +x mcp-mail`)
- Zkontrolujte logy Claude Desktop pro chybové hlášky

## Bezpečnost

- **Nikdy** neukládejte konfigurační soubor s hesly do git repozitáře
- Přidejte `config.json` do `.gitignore`
- Nastavte správná oprávnění: `chmod 600 config.json`
- Pro Gmail a Microsoft účty používejte App Passwords
- **Mazání e-mailů** - MCP server pouze přesouvá e-maily do koše, nikdy je trvale nesmaže
- **Nelze vysypat koš** - z bezpečnostních důvodů není možné přes MCP vysypat koš

## Cache

MCP-Mail používá in-memory cache pro snížení zátěže IMAP serveru. Cache je automaticky invalidována při změnách (vytvoření/mazání složek, přesouvání zpráv, atd.).

### Konfigurace cache

```json
{
  "cache": {
    "enabled": true,
    "folder_list_ttl_sec": 300,
    "folder_info_ttl_sec": 120,
    "message_list_ttl_sec": 120,
    "search_results_ttl_sec": 60
  }
}
```

- `folder_list_ttl_sec` - TTL pro seznam složek (výchozí: 300s = 5 minut)
- `folder_info_ttl_sec` - TTL pro informace o složce (výchozí: 120s = 2 minuty)
- `message_list_ttl_sec` - TTL pro seznam zpráv (výchozí: 120s)
- `search_results_ttl_sec` - TTL pro výsledky vyhledávání (výchozí: 60s)

Cache respektuje IMAP UIDVALIDITY - při změně UIDVALIDITY složky je cache automaticky invalidována.

## Licence

MIT License

## Přispívání

Pull requesty jsou vítány. Pro větší změny nejprve otevřete issue k diskuzi.
