import 'package:flutter/material.dart';

import '../domain/quiz_session.dart';
import 'theme/tui_theme.dart';
import 'widgets/tui_box.dart';

/// The end of a run: the score and every question that was failed, with the
/// answer that should have been given — the TUI's summary, word for word.
class ResultScreen extends StatelessWidget {
  const ResultScreen({super.key, required this.session});

  final QuizSession session;

  @override
  Widget build(BuildContext context) {
    final failed = session.failed;
    final total = failed.length + session.succeeded.length;

    return Scaffold(
      appBar: AppBar(leading: const BackButton()),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 16),
          children: [
            TuiBox(
              role: failed.isEmpty ? TuiRole.question : TuiRole.mistakes,
              title: 'результат',
              counter: '${total - failed.length}/$total',
              hints: Row(
                children: [
                  TuiAction(
                    label: 'пройти ещё раз',
                    onTap: () => Navigator.of(context).pop(true),
                    emphasised: true,
                  ),
                  const TuiHintSeparator(),
                  TuiAction(
                    label: 'к топикам',
                    onTap: () => Navigator.of(context).pop(false),
                  ),
                ],
              ),
              child: Text(
                failed.isEmpty
                    ? 'Вы не завалили ни одного вопроса. Молодец!'
                    : 'Неправильно ответил на ${failed.length} вопросов из $total.',
                style: Tui.termText,
              ),
            ),
            if (failed.isNotEmpty) ...[
              const SizedBox(height: 12),
              TuiBox(
                title: 'заваленные вопросы',
                counter: '${failed.length}',
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final (index, question) in failed.indexed) ...[
                      if (index > 0)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 10),
                          child: Container(height: 1, color: Tui.rule),
                        ),
                      Text('вопрос', style: Tui.label),
                      TuiText(question.text),
                      const SizedBox(height: 4),
                      Text('ответ', style: Tui.label),
                      TuiText(question.answer, style: Tui.termText),
                    ],
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
