/// Port of `topicpersistence/retrievequestions.go`.
///
/// The on-disk format is shared with the Go TUI and its sync daemon, so both
/// halves of it have to behave exactly as the Go original: one topic file per
/// topic, one `question/!/answer` line per pair, and a line break inside
/// either half written out as `/!n/` so a code snippet still occupies exactly
/// one line.
library;

const delimeterQuestionAnswer = '/!/';

/// Stands in for a line break inside a question or an answer.
const delimeterNewLine = '/!n/';

/// A single question/answer pair, the Dart counterpart of the TUI's `Question`.
class Question {
  const Question({required this.text, required this.answer});

  final String text;
  final String answer;

  @override
  bool operator ==(Object other) =>
      other is Question && other.text == text && other.answer == answer;

  @override
  int get hashCode => Object.hash(text, answer);
}

/// Renders a pair as the single on-disk line `question/!/answer`, newline
/// included — the same trailing newline `FormatQaLine` writes.
String formatQaLine(String question, String answer) =>
    '${_encodeQaField(question)}$delimeterQuestionAnswer${_encodeQaField(answer)}\n';

/// Splits an on-disk line back into a pair, restoring line breaks. Returns null
/// for lines that hold no pair at all, such as the trailing empty line every
/// topic file ends with.
Question? parseQaLine(String line) {
  final at = line.indexOf(delimeterQuestionAnswer);
  if (at < 0) {
    return null;
  }

  // Split on the first delimiter only, so a `/!/` inside an answer survives.
  return Question(
    text: _decodeQaField(line.substring(0, at)),
    answer: _decodeQaField(line.substring(at + delimeterQuestionAnswer.length)),
  );
}

/// Every pair a topic file holds, skipping the lines that hold none.
List<Question> parseTopic(String content) {
  final questions = <Question>[];
  for (final line in content.split('\n')) {
    final question = parseQaLine(line);
    if (question != null) {
      questions.add(question);
    }
  }

  return questions;
}

String _encodeQaField(String field) =>
    field.replaceAll('\r\n', '\n').replaceAll('\n', delimeterNewLine);

String _decodeQaField(String field) =>
    field.replaceAll(delimeterNewLine, '\n');
