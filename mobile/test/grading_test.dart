import 'package:checktests_mobile/domain/grading.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  // Ported from ld_test.go.
  test('a snippet indented differently is still at distance zero', () {
    const expected = 'func main() {\n\tfmt.Println("hi")\n}';
    const typedWithSpaces = 'func main() {\n    fmt.Println("hi")\n}';
    const typedFlat = 'func main() {\nfmt.Println("hi")\n}\n\n';

    for (final actual in [expected, typedWithSpaces, typedFlat]) {
      expect(ld(ignoreIndentation(actual), ignoreIndentation(expected)), 0,
          reason: 'ld($actual)');
    }
  });

  test('a real difference survives the normaliser', () {
    expect(ld(ignoreIndentation('a\n\tc'), ignoreIndentation('a\n\tb')), 1);
  });

  test('line breaks are kept, surrounding blanks are not', () {
    expect(ignoreIndentation('  a  \n\n\t b\t'), 'a\nb');
  });

  test('distance counts runes, not bytes', () {
    // Both strings are two-byte-per-character Cyrillic: a byte-wise distance
    // would come out as 2 here instead of 1.
    expect(ld('кот', 'код'), 1);
  });

  group('the tolerance ladder', () {
    // (answer, allowed distance, first rejected distance)
    const ladder = [
      ('%T', 0, 1),
      ('gpasswd sales', 2, 3),
      ('setfacl -m u:regular:rwx', 4, 5),
    ];

    for (final (answer, allowed, rejected) in ladder) {
      test('"$answer" allows $allowed but not $rejected', () {
        expect(isInputAndAnswerEqual(allowed, answer), isTrue);
        expect(isInputAndAnswerEqual(rejected, answer), isFalse);
      });
    }

    test('length is measured in runes, so Cyrillic is not over-tolerated', () {
      // 9 runes but 18 bytes: measured in bytes it would land in the ≤25 band
      // and forgive four edits instead of demanding an exact answer.
      const answer = 'разрешить';
      expect(answer.runes.length, 9);
      expect(isInputAndAnswerEqual(0, answer), isTrue);
      expect(isInputAndAnswerEqual(1, answer), isFalse);
    });

    test('indentation is excluded from the measured length', () {
      // 11 runes of text laid out over three indented lines: the tolerance must
      // come from the text, not from the whitespace around it.
      const answer = '  do\n\n\techo x\n';
      expect(isInputAndAnswerEqual(0, answer), isTrue);
      expect(isInputAndAnswerEqual(1, answer), isFalse);
    });
  });

  test('grade puts the two halves together', () {
    expect(grade('for i\n  do', 'for i\n\tdo'), isTrue);
    expect(grade('%d', '%T'), isFalse);
  });
}
