import 'package:flutter/material.dart';

import '../domain/qa_line.dart';
import '../domain/quiz_session.dart';
import 'result_screen.dart';
import 'theme/tui_theme.dart';
import 'widgets/tui_box.dart';

/// The quiz itself: the question in its box, the answer underneath, and the
/// mistakes stacked in a third.
///
/// The one place the layout parts company with the TUI is that last box. On a
/// terminal it always sits to the right; on a phone held upright there is no
/// room for it next to the question, so it becomes a collapsible section below
/// the answer and only goes back to the side in landscape.
class QuizScreen extends StatefulWidget {
  const QuizScreen({super.key, required this.questions});

  final List<Question> questions;

  @override
  State<QuizScreen> createState() => _QuizScreenState();
}

class _QuizScreenState extends State<QuizScreen> {
  late QuizSession _session = QuizSession(widget.questions);
  final _answer = TextEditingController();
  bool _mistakesOpen = false;

  @override
  void dispose() {
    _answer.dispose();
    super.dispose();
  }

  void _submit() {
    final passed = _session.submit(_answer.text);
    _answer.clear();
    FocusScope.of(context).unfocus();

    if (_session.isFinished) {
      _finish();
      return;
    }

    setState(() {
      // A wrong answer opens the box it just landed in, so the correct answer is
      // on screen while the next question is being read.
      if (!passed) {
        _mistakesOpen = true;
      }
    });
  }

  Future<void> _finish() async {
    final again = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => ResultScreen(session: _session)),
    );

    if (!mounted) {
      return;
    }

    if (again ?? false) {
      setState(() {
        _session = _session.restart();
        _mistakesOpen = false;
      });
    } else {
      Navigator.of(context).pop();
    }
  }

  @override
  Widget build(BuildContext context) {
    final question = _session.current;
    if (question == null) {
      return const Scaffold(body: SizedBox.shrink());
    }

    final landscape = MediaQuery.of(context).orientation == Orientation.landscape;

    return Scaffold(
      appBar: AppBar(leading: const BackButton()),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
          child: landscape
              ? Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(flex: 4, child: _asking(question)),
                    const SizedBox(width: 12),
                    Expanded(
                      flex: 3,
                      child: SingleChildScrollView(
                        child: _MistakesBox(mistakes: _session.mistakes),
                      ),
                    ),
                  ],
                )
              : _asking(question),
        ),
      ),
    );
  }

  Widget _asking(Question question) {
    final landscape = MediaQuery.of(context).orientation == Orientation.landscape;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TuiBox(
          role: TuiRole.question,
          title: 'вопрос',
          counter: '${_session.answered + 1}/${_session.total}',
          child: TuiText(question.text, style: Tui.termText),
        ),
        const SizedBox(height: 12),
        Expanded(
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                TuiBox(
                  title: 'ответ',
                  hints: Row(
                    children: [
                      TuiAction(label: 'ответить', onTap: _submit, emphasised: true),
                      const TuiHintSeparator(),
                      Expanded(child: Text('enter — новая строка')),
                    ],
                  ),
                  child: TextField(
                    controller: _answer,
                    autofocus: true,
                    maxLines: null,
                    minLines: 3,
                    keyboardType: TextInputType.multiline,
                    textInputAction: TextInputAction.newline,
                    style: Tui.body,
                    decoration: InputDecoration(
                      border: InputBorder.none,
                      isDense: true,
                      contentPadding: EdgeInsets.zero,
                      hintText: 'Напиши ответ на вопрос',
                      hintStyle: Tui.hint,
                    ),
                  ),
                ),
                if (!landscape && _session.mistakes.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  _MistakesBox(
                    mistakes: _session.mistakes,
                    open: _mistakesOpen,
                    onToggle: () => setState(() => _mistakesOpen = !_mistakesOpen),
                  ),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// The pairs failed so far, newest at the top, as `V:` what it should have been
/// and `X:` what was typed. In portrait it collapses; in landscape it is simply
/// always open, the way the terminal's right-hand panel is.
class _MistakesBox extends StatelessWidget {
  const _MistakesBox({required this.mistakes, this.open = true, this.onToggle});

  final List<Mistake> mistakes;
  final bool open;
  final VoidCallback? onToggle;

  @override
  Widget build(BuildContext context) {
    return TuiBox(
      role: TuiRole.mistakes,
      title: 'ошибки',
      counter: '${mistakes.length}',
      hints: onToggle == null
          ? null
          : Row(children: [TuiAction(label: open ? 'свернуть' : 'развернуть', onTap: onToggle)]),
      child: !open
          ? Text('свёрнуто', style: Tui.label)
          : mistakes.isEmpty
              ? Text('пока без ошибок', style: Tui.label)
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final (index, mistake) in mistakes.indexed) ...[
                      if (index > 0)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 10),
                          child: Container(height: 1, color: Tui.rule),
                        ),
                      TuiText('V: ${mistake.question.answer}', style: Tui.termText),
                      TuiText('X: ${mistake.typed}',
                          style: Tui.body.copyWith(color: Tui.dim)),
                    ],
                  ],
                ),
    );
  }
}
