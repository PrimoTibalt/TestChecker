import 'dart:math';

import 'package:checktests_mobile/domain/qa_line.dart';
import 'package:checktests_mobile/domain/quiz_session.dart';
import 'package:flutter_test/flutter_test.dart';

const _pairs = [
  Question(text: 'Вывести тип аргумента', answer: '%T'),
  Question(text: 'Бинарная форма', answer: '%b'),
  Question(text: 'Как задать пароль группе sales?', answer: 'gpasswd sales'),
];

void main() {
  test('asks every question exactly once', () {
    final session = QuizSession(_pairs, random: Random(1));
    final asked = <Question>[];

    while (!session.isFinished) {
      asked.add(session.current!);
      session.submit('');
    }

    expect(asked.toSet(), _pairs.toSet());
    expect(asked, hasLength(_pairs.length));
    expect(session.current, isNull);
  });

  test('files right and wrong answers apart', () {
    final session = QuizSession(_pairs, random: Random(2));

    while (!session.isFinished) {
      final question = session.current!;
      // Answer the short ones correctly and get the long one wrong.
      final passed = session.submit(question.answer == 'gpasswd sales' ? 'нет' : question.answer);
      expect(passed, question.answer != 'gpasswd sales');
    }

    expect(session.succeeded, hasLength(2));
    expect(session.failed.single.answer, 'gpasswd sales');
    expect(session.answered, 3);
  });

  test('mistakes stack newest first, with what was typed', () {
    final session = QuizSession(_pairs, random: Random(3));
    final wrong = <String>[];

    while (!session.isFinished) {
      final typed = 'ответ ${session.answered}  ';
      wrong.add(typed.trim());
      session.submit(typed);
    }

    expect(session.mistakes.map((m) => m.typed).toList(), wrong.reversed.toList());
  });

  test('restart brings back the whole set', () {
    final session = QuizSession(_pairs, random: Random(4));
    while (!session.isFinished) {
      session.submit('');
    }

    final again = session.restart();
    expect(again.isFinished, isFalse);
    expect(again.total, _pairs.length);
    expect(again.failed, isEmpty);
  });
}
