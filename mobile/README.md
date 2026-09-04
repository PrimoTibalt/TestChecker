# checkTests — Android app

A touch front-end over the same question set as the [terminal app](../README.md):
the same `Questions` directory, the same `question/!/answer` line format, and the
same edit-distance grading.

## What is a port, and of what

| Dart | Go it was ported from |
| --- | --- |
| `lib/domain/qa_line.dart` | `topicpersistence/retrievequestions.go` |
| `lib/domain/grading.dart` | `ld.go`, and `IsInputAndAnswerEqual` in `testcheck.go` |
| `lib/domain/quiz_session.dart` | the state machine inside `TestCheck` |
| `lib/data/topic_store.dart` | `topicpersistence/`, plus the file handling of `cmd/sync/topicstate.go` |
| `lib/data/sync_client.dart` | `reconcileWith` in `cmd/sync/reconcile.go` |

Two things to keep in mind when touching any of them:

- **Runes, not code units.** The questions are in Russian, and the Go original
  counts runes — both the distance and the length that picks the tolerance band.
- **The bytes on disk are a contract.** A topic file written here is read by the
  TUI and by the daemon, so `formatQaLine` has to produce exactly what
  `FormatQaLine` does, trailing newline included.

## Two looks

The app ships with both, switchable under *настройки → Внешний вид*, and the
choice is remembered:

- **Терминал** — the Go app's own palette, sampled out of `../docs/*.png`: the
  cyan question panel, the tan question text, the dark red panel for the
  mistakes. Noto Sans Mono.
- **Читалка** (default) — the definition-box look from the bilingual reader
  (`WebApi/wwwroot/definition-box.css` and `tui.css` over there): near black, one
  light border, a single warm accent, thin rules between the sections and a hint
  bar along the bottom. JetBrains Mono. Easier on a phone, where the terminal's
  colour coding has no terminal around it to belong to.

Only the palette and the font differ — the widget tree is the same either way.
`TuiBox` is the one bordered box every screen is built from, and it asks for a
`TuiRole` (question, mistakes, plain) rather than a colour, so `TuiPalette` is
the single place that decides how a role looks. Adding a third look means adding
a palette, nothing else.

Both faces are bundled rather than fetched, and both are subset to Latin,
Cyrillic, punctuation and box drawing — 8.8 MB of full faces come to about
500 KB that way (`pyftsubset`, see the comment in `pubspec.yaml`).

## Syncing

The phone is a client-only peer. It never listens, so nothing is pushed to it; it
reconciles when it is opened, when the menu is pulled down, and after a question
is added. Everything else works offline off its own copy.

It talks to **every** machine on its list, not just the first that answers: the
daemon pauses its watcher while applying a `/syncTopic`, so a machine that takes
a topic from the phone does not relay it to the others.

Deletions do not travel, in either direction — the same no-tombstones limitation
the daemon has.

## Running it

```sh
flutter test        # ported logic, and sync against fake partners on loopback
flutter analyze
flutter run         # onto a connected phone
```

### If a sync just times out

The phone reaches the daemon over the tailnet, so the machine running it has to
actually accept the connection: `ufw` and the like will drop it silently. On a
box with ufw active:

```sh
sudo ufw allow in on tailscale0 to any port 8081 proto tcp
```

`tailscale ping <machine>` succeeding proves the tailnet is fine and points the
finger at the firewall — the app itself just reports the machine as unreachable
and carries on off its own copy.

To try the sync without Tailscale at all, point the phone at a daemon on this
machine over USB:

```sh
adb reverse tcp:8081 tcp:8081
TAILSCALE_IP=127.0.0.1 TAILSCALE_PARTNER_IP=<anything> go run ../cmd/sync
```

then add `127.0.0.1` on the app's settings screen.
