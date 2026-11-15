# ftm voice

How ftm talks: in the app, the terminal, the README and the store listing.
Every string in `internal/i18n/locales/{en,es,pt}.yaml` follows this page.
`go test ./internal/i18n/` checks the mechanical rules (marked **[test]**).

## Who we write for

A Gamemaster getting a session ready, often ten minutes before the players arrive. They
know Foundry well. They may not know what a tunnel, a port or a provider is, and they
shouldn't have to learn to get the table open. Copy should read like a friend at the
table who knows the tech: calm, short, on your side.

## The voice in one line

**Warm, plain, tabletop-aware.** "Share your world with players everywhere," not
"Expose your local service via secure tunnels."

## Rules

### 1. Say what is true, then what to do

Every message answers "what happened?" and, when something is wrong, "what do I do now?".
Lead with the part the reader acts on.

| Do | Don't |
|---|---|
| Nothing on port 30000. Launch Foundry, then check again. | Connection refused |
| No link yet. Start the connection first. | No URL available - start tunnel first |
| The link for Ravenloft closes in 5 minutes. | Tunnel expiring soon |

### 2. Plain words, short sentences

Everyday verbs (open, send, start, stop). One idea per sentence. Labels are nouns,
buttons are verb + object. Use contractions in English ("can't", "isn't").

| Do | Don't |
|---|---|
| Copy link | Copy URL to clipboard |
| Couldn't save the changes: {error} | Failed to update connection: {error} |
| Pick a connection to see its details | Select a tunnel to view details |

### 3. Calm, even when things break **[test]**

No exclamation marks, in any language, including success messages. No "Oops", no
"Successfully", no "Awesome". A success message states the result; an error states the
problem without blame and without drama. Put the raw provider error after a colon, at
the end, where people who need it can find it.

| Do | Don't |
|---|---|
| Connection added | Tunnel added! |
| Download finished | Download complete! |
| Couldn't update bore: 404 | Error: Update failed! |

### 4. No hype, no filler **[test]**

Never "seamless", "powerful", "magic", "simply", "just", "unleash", "supercharge". Never
"secure" as a selling point: a public link is reachable by anyone who has it, and copy
must not suggest otherwise. If a sentence still works with a word removed, remove it.

| Do | Don't |
|---|---|
| Give Foundry a public link | Create a secure tunnel to your local service |
| Anyone with the link can reach your world while the connection is open. | Your world is now securely exposed! |

### 5. Hide the plumbing **[test]**

The reader cares about their world, their players and the link. Don't show binaries,
endpoints, PATH, stdout, TUI, process states or HTTP codes unless they are the only thing
that helps (the CLI may name `PATH` or a command; the app never does).

| Concept in code | Say |
|---|---|
| tunnel (a saved config) | connection |
| public URL | link |
| provider binary | tunnel program |
| need_installing | Not downloaded |
| dashboard / web UI | web view |
| TUI | ftm (in the terminal) |

"Provider" stays: it is a choice the user makes (Cloudflare, Pinggy, bore) and the CLI
command is `ftm providers`.

### 6. Tabletop-aware, not in costume

Use the words Gamemasters already use when they are literally true: *world*,
*players*, *table*, *session*, *invite*, *Gamemaster*. "Open the table" works because
starting the connection is what lets the table sit down. Don't dress plumbing up as
fantasy ("summon a portal", "your quest awaits", "brave adventurer"); it is cute once and
grating by the third session.

Foundry itself calls the address players use an *invitation link* and the host a
*Gamemaster*, so ftm uses *link*, *invite* and *Gamemaster* too.

| Do | Don't |
|---|---|
| The table is open | Portal summoned! |
| Send the link to your players. On a phone they can scan the code. | Share this URL with your players |
| Welcome, Gamemaster | Welcome, Dungeon Master! |

### 7. Numbers get a label, not a telegram

A count reads as a label and a number, or as a full sentence. Never a fragment glued
with a middle dot or "ya son".

| Do | Don't |
|---|---|
| Ravenloft: 3 visits in the last 5 minutes | Entró un jugador · ya son 3 |
| Players connected  3 | 3 players connected 🎉 |

### 8. Mechanics **[test]**

- Sentence case for everything: titles, buttons, menu items, settings. Proper names keep
  their case (Foundry VTT, Cloudflare, Microsoft Store, bore).
- `…` (one character) only for work in progress: "Connecting…". Never `...`.
- No emoji. Icons come from the icon library.
- Placeholders: `{0}` for positional values from Go `TF`, `{name}` for named values from
  the web `t()`. Never printf verbs (`%s`, `%d`). Every locale has the same placeholders
  as English.
- The product is "ftm" in running text, "Foundry Tunnel Manager" only as the full title.
- Keys in the terminal UI are written bare: "Press a to add one", "c to copy".
- Status words in the terminal list are short uppercase tags (ONLINE, FAILED) because
  they sit in a fixed-width column. Everywhere else, sentence case.

## Spanish and Portuguese

Write each language from the meaning, not from the English sentence. If the natural
sentence has a different shape, use that shape.

**Spanish (`es`)**: neutral Spanish that reads well in Latin America and Spain.

- Tú, never usted, never vos/vosotros. Imperatives: *Inicia*, *Abre*, *Pulsa*.
- Buttons in the infinitive: *Copiar enlace*, *Abrir la mesa*.
- Avoid words that split the regions: *teléfono* (not *celular*/*móvil*), *equipo* (not
  *computadora*/*ordenador*), *Ajustes* for settings.
- No anglicisms when Spanish has the word: *eliminar/quitar* (not *remover*),
  *enlace* (not *link*), *registro* (not *logs*), *novedades* (not *release notes*).
- GM is *máster*, the word Spanish-speaking tables use.
- *La conexión* is feminine; agree with it (*Detenida*, *Iníciala*).

**Portuguese (`pt`)**: Brazilian Portuguese.

- Você, never tu/teu. Imperatives: *Inicie*, *Abra*, *Aperte*.
- Buttons in the infinitive: *Copiar link*, *Abrir a mesa*.
- Brazilian words: *celular*, *excluir* (delete), *baixar* (download), *tela*, *link*
  (Brazilian UI uses *link*), *registros* for logs.
- "Não deu para…" for failures reads like a person; "Falha ao…" reads like a log.
- GM is *mestre*.

| en | es: do | es: don't (word for word) |
|---|---|---|
| Couldn't copy. Select the link and copy it by hand. | No se pudo copiar. Selecciona el enlace y cópialo a mano. | Error al copiar. Seleccione el URL manualmente. |
| Start it, copy the URL, send it to your table. | Inicia la conexión, copia el enlace y pásaselo a tus jugadores. | Arráncala, copia la URL, envíala a tu mesa. |

| en | pt: do | pt: don't |
|---|---|---|
| Couldn't save your settings | Não deu para salvar suas configurações | Erro ao salvar configuração |
| Anyone with the link can reach your world… | Qualquer pessoa com o link consegue entrar no seu mundo… | Qualquer pessoa com a ligação pode abrir o teu mundo… (pt-PT) |

## Keys

- A key exists only while code uses it. `TestEveryLocaleKeyIsUsed` fails on dead keys;
  `TestEveryUsedKeyExistsInEveryLocale` fails when `web/src` or Go uses a key a locale
  lacks. Keys built at runtime (`foundry_${check}`, `lang_${code}`) are listed in
  `dynamicKeyFamilies` in `internal/i18n/usage_test.go`.
- Add the English, Spanish and Portuguese text in the same change. Don't ship English
  in the es or pt file as a placeholder.

## Sources

What each one taught us, and where it shows up above.

- Mailchimp Content Style Guide, [Voice and tone](https://styleguide.mailchimp.com/voice-and-tone/)
  and [Grammar and mechanics](https://styleguide.mailchimp.com/grammar-and-mechanics/):
  one constant voice with a tone that shifts by situation; clarity over cleverness; no
  exclamation marks in failure messages. Rules 1, 3.
- [GOV.UK style guide](https://www.gov.uk/guidance/style-guide/a-to-z-of-gov-uk-style)
  and [Writing for GOV.UK](https://www.gov.uk/guidance/content-design/writing-for-gov-uk):
  plain English, short sentences, front-load the useful part, drop jargon and buzzwords.
  Rules 1, 2, 4.
- [Microsoft Writing Style Guide](https://learn.microsoft.com/en-us/style-guide/welcome/):
  sentence case, contractions, everyday words, error messages that say what happened and
  what to do. Rules 2, 3, 8.
- Apple Human Interface Guidelines, [Writing](https://developer.apple.com/design/human-interface-guidelines/writing):
  buttons as verbs, consistent terms, write for the person in front of the screen.
  Rules 2, 5.
- Microsoft localization style guides for
  [neutral Spanish](https://download.microsoft.com/download/4/3/1/431d9836-40dc-4f2b-aafc-def5c7b5d0f3/spa-neu-StyleGuide.pdf)
  and [Brazilian Portuguese](https://download.microsoft.com/download/8/e/3/8e349e32-9eb9-4b63-9909-7586b94a24dd/por-bra-StyleGuide.pdf):
  neutral vocabulary across regions, informal address, translate meaning not words.
- Zulip, [Spanish translation style guide](https://zulip.readthedocs.io/en/8.2/translating/spanish.html):
  tú is the norm for consumer software in Spanish.
- Foundry VTT, [InvitationLinks](https://foundryvtt.com/api/v12/classes/client.InvitationLinks.html):
  Foundry's own name for the address players join with, so ftm's words line up with what
  the Gamemaster already sees in Foundry. Rule 6.
- Owlbear Rodeo, [Dev log 7](https://blog.owlbear.rodeo/owlbear-rodeo-dev-log-7/): a
  small tabletop tool that frames joining as "send them the link to the room", with no
  networking words in sight. Rules 5, 6.
