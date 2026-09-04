import 'package:flutter/material.dart';

import '../data/sync_client.dart';
import 'add_question_screen.dart';
import 'app_state.dart';
import 'settings_screen.dart';
import 'theme/tui_theme.dart';
import 'topic_picker_screen.dart';
import 'widgets/tui_box.dart';

/// The TUI's top-level menu: the same five actions in the same order, with the
/// three that v1 does not implement left visible but inert, so the screen does
/// not have to be redesigned when they land.
class MenuScreen extends StatefulWidget {
  const MenuScreen({super.key, required this.state});

  final AppState state;

  @override
  State<MenuScreen> createState() => _MenuScreenState();
}

class _MenuScreenState extends State<MenuScreen> {
  @override
  void initState() {
    super.initState();
    // Catch up with the machines before anything is read off disk, the way the
    // daemon reconciles once at startup.
    WidgetsBinding.instance.addPostFrameCallback((_) => _sync(quiet: true));
  }

  Future<void> _sync({bool quiet = false}) async {
    final report = await widget.state.syncNow();
    if (!mounted) {
      return;
    }

    if (report == null) {
      if (!quiet) {
        _say('Не добавлено ни одной машины для синхронизации.');
      }
      return;
    }

    if (report.failures.isNotEmpty && !report.changedAnything) {
      if (!quiet) {
        _say('Синхронизация не удалась: ${report.failures.values.first}');
      }
      return;
    }

    if (report.changedAnything) {
      setState(() {});
      _say(_describe(report));
    } else if (!quiet) {
      _say('Всё уже синхронизировано.');
    }
  }

  String _describe(SyncReport report) {
    final parts = <String>[];
    if (report.pulled.isNotEmpty) {
      parts.add('получено ${report.pulled.length}');
    }
    if (report.pushed.isNotEmpty) {
      parts.add('отправлено ${report.pushed.length}');
    }

    return 'синхронизация: ${parts.join(', ')}';
  }

  void _say(String message) => ScaffoldMessenger.of(context)
    ..clearSnackBars()
    ..showSnackBar(SnackBar(content: Text(message, style: Tui.body)));

  Future<void> _open(Widget screen) async {
    await Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => screen));
    if (mounted) {
      setState(() {});
    }
  }

  @override
  Widget build(BuildContext context) {
    final actions = <_MenuAction>[
      _MenuAction('Добавить новый вопрос в топик',
          () => _open(AddQuestionScreen(state: widget.state))),
      _MenuAction('Проверить знания по топику',
          () => _open(TopicPickerScreen(state: widget.state))),
      const _MenuAction('Добавить новый топик', null),
      const _MenuAction('Удалить топик', null),
      const _MenuAction('Редактировать топик', null),
    ];

    return Scaffold(
      body: SafeArea(
        child: RefreshIndicator(
          color: Tui.accent,
          backgroundColor: Tui.background,
          onRefresh: _sync,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              AnimatedBuilder(
                animation: widget.state,
                builder: (context, _) => TuiBox(
                  title: 'Знания - сила. Что делать будем?',
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  hints: Row(
                    children: [
                      Expanded(
                        child: Text(
                          widget.state.syncing
                              ? 'синхронизация...'
                              : 'потяни вниз · ${_machines(widget.state.settings.partners.length)}',
                        ),
                      ),
                      TuiAction(
                        label: 'настройки',
                        onTap: () => _open(SettingsScreen(state: widget.state)),
                      ),
                    ],
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [for (final action in actions) _MenuRow(action: action)],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// "1 машиной", but "2 машинами" — the instrumental case the sentence needs
/// only changes for counts ending in a single 1, and not for 11 itself. Zero
/// gets a phrase of its own rather than a count nothing agrees with.
String _machines(int count) {
  if (count == 0) {
    return 'нет машин';
  }

  final singular = count % 10 == 1 && count % 100 != 11;
  return '$count ${singular ? 'машиной' : 'машинами'}';
}

class _MenuAction {
  const _MenuAction(this.label, this.onTap);

  final String label;
  final VoidCallback? onTap;
}

/// One row of the menu, marked with the `>` the terminal draws on the line the
/// cursor is on.
class _MenuRow extends StatelessWidget {
  const _MenuRow({required this.action});

  final _MenuAction action;

  @override
  Widget build(BuildContext context) {
    final enabled = action.onTap != null;
    return InkWell(
      onTap: action.onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 11, horizontal: 12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(enabled ? '> ' : '  ',
                style: Tui.body.copyWith(color: enabled ? Tui.accent : Tui.rule)),
            Expanded(
              child: Text(
                action.label,
                style: Tui.body.copyWith(color: enabled ? Tui.text : Tui.dim),
              ),
            ),
            if (!enabled) Text('только в терминале', style: Tui.label),
          ],
        ),
      ),
    );
  }
}
