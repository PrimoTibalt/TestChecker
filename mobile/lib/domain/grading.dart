/// Port of `ld.go` and of `IsInputAndAnswerEqual` in `testcheck.go`.
///
/// This is the part that decides right from wrong, so it has to agree with the
/// TUI answer for answer. Everything works on runes rather than UTF-16 code
/// units, because the questions are written in Russian and Go counts runes.
library;

/// Drops the whitespace that only lays a snippet out — the spaces and tabs
/// around every line, and blank lines altogether — so that a code answer typed
/// at a different indentation still comes out at distance 0.
String ignoreIndentation(String text) {
  final lines = <String>[];
  for (final line in text.split('\n')) {
    final trimmed = line.replaceAll(_surroundingBlanks, '');
    if (trimmed.isEmpty) {
      continue;
    }

    lines.add(trimmed);
  }

  return lines.join('\n');
}

final _surroundingBlanks = RegExp(r'^[ \t\r]+|[ \t\r]+$');

/// The Levenshtein distance between what was typed and what was expected.
///
/// The Go original re-slices the strings into runes inside the inner loop; here
/// both are converted once up front. Same distances, but it stays usable on the
/// long answers the ladder below still accepts.
int ld(String actual, String expected) {
  final expectedRunes = expected.runes.toList();
  final actualRunes = actual.runes.toList();
  final rows = expectedRunes.length + 1;
  final columns = actualRunes.length + 1;

  // Only the previous row is ever read, so one row of state is enough.
  var previous = List<int>.generate(columns, (column) => column);
  var current = List<int>.filled(columns, 0);

  for (var row = 1; row < rows; row++) {
    current[0] = row;
    for (var column = 1; column < columns; column++) {
      if (expectedRunes[row - 1] == actualRunes[column - 1]) {
        current[column] = previous[column - 1];
        continue;
      }

      current[column] = 1 +
          _min3(current[column - 1], previous[column - 1], previous[column]);
    }

    final swap = previous;
    previous = current;
    current = swap;
  }

  return previous[columns - 1];
}

int _min3(int a, int b, int c) => a < b ? (a < c ? a : c) : (b < c ? b : c);

const _smallAnswerLen = 10;
const _mediumAnswerLen = 15;
const _bigAnswerLen = 25;
const _paragraphAnswerLen = 100;
const _poemAnswerLength = 250;
const _dissertationAnswerLength = 1000;

/// Whether an answer passes, with a tolerance that grows with the length of the
/// expected answer: a five-character command has to be exact while a paragraph
/// does not.
bool isInputAndAnswerEqual(int distance, String answer) {
  final answerLen = ignoreIndentation(answer).runes.length;
  if (answerLen <= _smallAnswerLen) {
    return distance <= 0;
  } else if (answerLen <= _mediumAnswerLen) {
    return distance <= 2;
  } else if (answerLen <= _bigAnswerLen) {
    return distance <= 4;
  } else if (answerLen <= _paragraphAnswerLen) {
    return distance <= 8;
  } else if (answerLen <= _poemAnswerLength) {
    return distance <= 16;
  } else if (answerLen <= _dissertationAnswerLength) {
    return distance <= 32;
  }

  return distance < 100;
}

/// Grades one typed answer against the expected one, indentation ignored on
/// both sides.
bool grade(String input, String answer) =>
    isInputAndAnswerEqual(ld(ignoreIndentation(input), ignoreIndentation(answer)), answer);
