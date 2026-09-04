import 'package:checktests_mobile/domain/qa_line.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  // The four cases of topicpersistence/qaline_test.go, so both ends of the
  // format are checked against the same expectations.
  test('a pair round-trips through one on-disk line', () {
    const question = 'Как выглядит main?';
    const answer = 'func main() {\n\tfmt.Println("hi")\n}';

    final line = formatQaLine(question, answer);
    expect('\n'.allMatches(line).length, 1, reason: 'a pair occupies one line');
    expect(line.endsWith('\n'), isTrue);

    final parsed = parseQaLine(line.substring(0, line.length - 1));
    expect(parsed, isNotNull);
    expect(parsed!.text, question);
    expect(parsed.answer, answer);
  });

  test('reads lines written before the app existed', () {
    final parsed = parseQaLine('вопрос/!/ответ');
    expect(parsed?.text, 'вопрос');
    expect(parsed?.answer, 'ответ');
  });

  test('rejects lines holding no pair', () {
    for (final line in ['', 'просто текст']) {
      expect(parseQaLine(line), isNull, reason: 'accepted $line');
    }
  });

  test('keeps a delimiter that appears inside the answer', () {
    final line = formatQaLine('q', 'a${delimeterQuestionAnswer}b');
    final parsed = parseQaLine(line.substring(0, line.length - 1));
    expect(parsed?.answer, 'a/!/b');
  });

  test('carriage returns are normalised away before writing', () {
    expect(formatQaLine('q', 'a\r\nb'), 'q${delimeterQuestionAnswer}a${delimeterNewLine}b\n');
  });

  test('parses a topic file copied off disk', () {
    // Byte-for-byte out of the "bash скриптинг" topic, trailing newline and all.
    const content =
        'Вывести через echo \$val все чётные числа от 10 до 0 (ноль включительно)/!/for val in {10..0..2}/!n/do/!n/  echo \$val/!n/done\n'
        'Конвертируй все .html файлы из директории \$dir в .txt/!/for file in \$dir/*.html/!n/do/!n/  mv \$file \$dir/\$(basename -s .html \$file).txt/!n/done\n';

    final questions = parseTopic(content);
    expect(questions, hasLength(2), reason: 'the trailing empty line is not a pair');
    expect(questions.first.answer, 'for val in {10..0..2}\ndo\n  echo \$val\ndone');
    expect(questions.last.text, contains('Конвертируй'));
  });
}
