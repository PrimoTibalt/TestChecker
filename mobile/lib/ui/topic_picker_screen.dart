import 'package:flutter/material.dart';

import '../domain/qa_line.dart';
import 'app_state.dart';
import 'quiz_screen.dart';
import 'theme/tui_theme.dart';
import 'widgets/tui_box.dart';

/// The multi-select of topics to be tested on. Tapping a topic's name expands
/// its questions underneath it — what the TUI's lower panel shows for whichever
/// topic the cursor is on.
class TopicPickerScreen extends StatefulWidget {
  const TopicPickerScreen({super.key, required this.state});

  final AppState state;

  @override
  State<TopicPickerScreen> createState() => _TopicPickerScreenState();
}

class _TopicPickerScreenState extends State<TopicPickerScreen> {
  final _selected = <String>{};
  String? _expanded;
  List<String>? _topics;
  final _questions = <String, List<Question>>{};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final topics = await widget.state.store.listTopics();
    if (mounted) {
      setState(() => _topics = topics);
    }
  }

  Future<void> _expand(String topic) async {
    _questions[topic] ??= await widget.state.store.questionsOf(topic);
    if (mounted) {
      setState(() => _expanded = _expanded == topic ? null : topic);
    }
  }

  Future<void> _start() async {
    final questions = <Question>[];
    for (final topic in _topics!.where(_selected.contains)) {
      questions.addAll(_questions[topic] ??= await widget.state.store.questionsOf(topic));
    }

    if (!mounted) {
      return;
    }

    if (questions.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            'выбранный файл не содержит вопросов попробуйте добавить новые',
            style: Tui.body,
          ),
        ),
      );
      return;
    }

    await Navigator.of(context).push(
      MaterialPageRoute<void>(builder: (_) => QuizScreen(questions: questions)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final topics = _topics;

    return Scaffold(
      appBar: AppBar(leading: const BackButton()),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
          child: topics == null
              ? const Center(child: CircularProgressIndicator())
              : TuiBox(
                  fill: true,
                  title: 'Выбери топик(и) для теста',
                  counter: topics.isEmpty ? null : '${_selected.length}/${topics.length}',
                  padding: EdgeInsets.zero,
                  hints: Row(
                    children: [
                      TuiAction(
                        label: 'начать тест',
                        onTap: _selected.isEmpty ? null : _start,
                        emphasised: true,
                      ),
                      const TuiHintSeparator(),
                      Expanded(child: Text('тап по названию — показать вопросы')),
                    ],
                  ),
                  child: topics.isEmpty
                      ? Padding(
                          padding: const EdgeInsets.all(16),
                          child: Text(
                            'Ни одного топика. Синхронизируйся с машиной или добавь топик в терминале.',
                            style: Tui.label,
                          ),
                        )
                      : ListView.builder(
                          padding: const EdgeInsets.symmetric(vertical: 4),
                          itemCount: topics.length,
                          itemBuilder: (context, index) => _TopicRow(
                            topic: topics[index],
                            checked: _selected.contains(topics[index]),
                            expanded: _expanded == topics[index],
                            questions: _questions[topics[index]],
                            onCheck: () => setState(() {
                              _selected.contains(topics[index])
                                  ? _selected.remove(topics[index])
                                  : _selected.add(topics[index]);
                            }),
                            onExpand: () => _expand(topics[index]),
                          ),
                        ),
                ),
        ),
      ),
    );
  }
}

class _TopicRow extends StatelessWidget {
  const _TopicRow({
    required this.topic,
    required this.checked,
    required this.expanded,
    required this.questions,
    required this.onCheck,
    required this.onExpand,
  });

  final String topic;
  final bool checked;
  final bool expanded;
  final List<Question>? questions;
  final VoidCallback onCheck;
  final VoidCallback onExpand;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        InkWell(
          onTap: onExpand,
          child: Row(
            children: [
              // A wide tap target for the checkbox, so checking a topic and
              // peeking into it never get confused for one another.
              InkWell(
                onTap: onCheck,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(12, 11, 8, 11),
                  child: Text(
                    checked ? '[x]' : '[ ]',
                    style: Tui.body.copyWith(color: checked ? Tui.accent : Tui.dim),
                  ),
                ),
              ),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 11),
                  child: Text(
                    topic,
                    style: Tui.body.copyWith(color: checked ? Tui.accent : Tui.text),
                  ),
                ),
              ),
              Padding(
                padding: const EdgeInsets.only(right: 12),
                child: Text(expanded ? '−' : '+', style: Tui.hint),
              ),
            ],
          ),
        ),
        if (expanded)
          Padding(
            padding: const EdgeInsets.fromLTRB(36, 0, 12, 12),
            child: questions == null
                ? Text('...', style: Tui.label)
                : questions!.isEmpty
                    ? Text('в этом топике нет вопросов', style: Tui.label)
                    : Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          for (final question in questions!)
                            Padding(
                              padding: const EdgeInsets.symmetric(vertical: 3),
                              child: Text(question.text, style: Tui.hint),
                            ),
                        ],
                      ),
          ),
      ],
    );
  }
}
