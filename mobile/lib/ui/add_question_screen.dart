import 'package:flutter/material.dart';

import '../domain/qa_line.dart';
import 'app_state.dart';
import 'theme/tui_theme.dart';
import 'widgets/tui_box.dart';

/// Adding a pair to an existing topic: pick the topic, then type the question
/// and the answer. The pairs the topic already holds stay on screen so the same
/// one is not added twice — the TUI's left panel.
class AddQuestionScreen extends StatefulWidget {
  const AddQuestionScreen({super.key, required this.state});

  final AppState state;

  @override
  State<AddQuestionScreen> createState() => _AddQuestionScreenState();
}

class _AddQuestionScreenState extends State<AddQuestionScreen> {
  final _question = TextEditingController();
  final _answer = TextEditingController();

  List<String>? _topics;
  String? _topic;
  List<Question> _existing = [];

  @override
  void initState() {
    super.initState();
    widget.state.store.listTopics().then((topics) {
      if (mounted) {
        setState(() => _topics = topics);
      }
    });
  }

  @override
  void dispose() {
    _question.dispose();
    _answer.dispose();
    super.dispose();
  }

  Future<void> _choose(String topic) async {
    final existing = await widget.state.store.questionsOf(topic);
    if (mounted) {
      setState(() {
        _topic = topic;
        _existing = existing;
      });
    }
  }

  Future<void> _add() async {
    final question = _question.text.trim();
    final answer = _answer.text.trim();
    if (question.isEmpty || answer.isEmpty) {
      return;
    }

    await widget.state.store.appendPair(_topic!, question, answer);
    _question.clear();
    _answer.clear();
    final existing = await widget.state.store.questionsOf(_topic!);
    if (mounted) {
      setState(() => _existing = existing);
      FocusScope.of(context).unfocus();
    }

    // Hand the new pair to the machines straight away, rather than leaving it
    // sitting on the phone until the next start.
    final report = await widget.state.syncNow();
    if (mounted && report != null && report.pushed.contains(_topic)) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('отправлено на машины', style: Tui.body)),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final topics = _topics;
    if (topics == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    return Scaffold(
      appBar: AppBar(leading: const BackButton()),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
          child: _topic == null ? _pickingTopic(topics) : _fillingIn(),
        ),
      ),
    );
  }

  Widget _pickingTopic(List<String> topics) {
    return TuiBox(
      fill: true,
      title: 'Выбери топик для добавления вопросов',
      counter: '${topics.length}',
      padding: EdgeInsets.zero,
      child: topics.isEmpty
          ? Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                'Ни одного топика. Топики пока создаются только в терминале.',
                style: Tui.label,
              ),
            )
          : ListView.builder(
              padding: const EdgeInsets.symmetric(vertical: 4),
              itemCount: topics.length,
              itemBuilder: (context, index) => InkWell(
                onTap: () => _choose(topics[index]),
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 11, horizontal: 12),
                  child: Row(
                    children: [
                      Text('> ', style: Tui.body.copyWith(color: Tui.accent)),
                      Expanded(child: Text(topics[index], style: Tui.body)),
                    ],
                  ),
                ),
              ),
            ),
    );
  }

  Widget _fillingIn() {
    return ListView(
      children: [
        TuiBox(
          title: _topic,
          hints: Row(
            children: [
              TuiAction(label: 'добавить', onTap: _add, emphasised: true),
              const TuiHintSeparator(),
              TuiAction(label: 'другой топик', onTap: () => setState(() => _topic = null)),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('вопрос', style: Tui.label),
              _Field(controller: _question, hint: 'Введите вопрос'),
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 10),
                child: Container(height: 1, color: Tui.rule),
              ),
              Text('ответ', style: Tui.label),
              _Field(controller: _answer, hint: 'Введите ответ на вопрос'),
            ],
          ),
        ),
        const SizedBox(height: 12),
        TuiBox(
          title: 'вопросы в топике',
          counter: '${_existing.length}',
          child: _existing.isEmpty
              ? Text('пока ни одного', style: Tui.label)
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Newest first, so a pair just added is the one on screen.
                    for (final (index, pair) in _existing.reversed.indexed) ...[
                      if (index > 0)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 10),
                          child: Container(height: 1, color: Tui.rule),
                        ),
                      TuiText('Q: ${pair.text}'),
                      TuiText('A: ${pair.answer}', style: Tui.termText),
                    ],
                  ],
                ),
        ),
      ],
    );
  }
}

class _Field extends StatelessWidget {
  const _Field({required this.controller, required this.hint});

  final TextEditingController controller;
  final String hint;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      maxLines: null,
      minLines: 2,
      keyboardType: TextInputType.multiline,
      style: Tui.body,
      decoration: InputDecoration(
        border: InputBorder.none,
        isDense: true,
        contentPadding: EdgeInsets.zero,
        hintText: hint,
        hintStyle: Tui.hint,
      ),
    );
  }
}
