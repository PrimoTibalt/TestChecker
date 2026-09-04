/// The quiz state machine, lifted out of the TUI's `TestCheck` model in
/// `testcheck.go` and `testknowledge.go` with no UI in it, so it can be tested
/// on its own and driven by whatever widget tree sits on top.
library;

import 'dart:math';

import 'grading.dart';
import 'qa_line.dart';

/// One question that was answered wrongly, together with what was typed —
/// the `V:`/`X:` pair the TUI stacks into its right-hand panel.
class Mistake {
  const Mistake({required this.question, required this.typed});

  final Question question;
  final String typed;
}

class QuizSession {
  QuizSession(List<Question> questions, {Random? random})
      : assert(questions.isNotEmpty, 'a quiz needs at least one question'),
        _all = List.unmodifiable(questions),
        _remaining = List.of(questions),
        _random = random ?? Random() {
    _current = _remaining[_random.nextInt(_remaining.length)];
  }

  final List<Question> _all;
  final List<Question> _remaining;
  final Random _random;

  final List<Question> _succeeded = [];
  final List<Question> _failed = [];
  /// Newest first, the way the panel stacks them.
  final List<Mistake> _mistakes = [];

  late Question _current;

  /// The question on screen, or null once the run is over.
  Question? get current => _finished ? null : _current;

  bool get _finished => _remaining.isEmpty;
  bool get isFinished => _finished;

  int get answered => _succeeded.length + _failed.length;
  int get total => _all.length;
  List<Question> get succeeded => List.unmodifiable(_succeeded);
  List<Question> get failed => List.unmodifiable(_failed);
  List<Mistake> get mistakes => List.unmodifiable(_mistakes);

  /// Grades what was typed, files the pair and moves on to a random one of the
  /// questions left. Reports whether the answer passed.
  bool submit(String typed) {
    if (_finished) {
      return false;
    }

    final question = _current;
    final passed = grade(typed, question.answer);
    if (passed) {
      _succeeded.add(question);
    } else {
      _failed.add(question);
      _mistakes.insert(0, Mistake(question: question, typed: typed.trim()));
    }

    _remaining.remove(question);
    if (!_finished) {
      _current = _remaining[_random.nextInt(_remaining.length)];
    }

    return passed;
  }

  /// Starts the same set over, the way pressing `r` on the TUI's result screen
  /// does.
  QuizSession restart() => QuizSession(_all, random: _random);
}
