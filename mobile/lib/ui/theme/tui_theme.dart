/// Two looks, both of them TUI-flavoured, picked on the settings screen.
///
/// * [TuiSkin.terminal] is the Go app's own palette, sampled out of `docs/*.png`
///   — the cyan question panel, the tan question text, the dark red panel for
///   the mistakes.
/// * [TuiSkin.reader] is the definition-box look from the bilingual reader:
///   near black, one light border, a single warm accent, thin rules between the
///   sections and a hint bar along the bottom. Easier on a phone, where the
///   terminal's colour coding has no terminal around it to belong to.
///
/// Only the palette and the font change. The widgets are the same either way,
/// and a box asks for a *role* — question, mistakes, plain — rather than a
/// colour, so the palette is the one place that decides how a role looks.
library;

import 'package:flutter/material.dart';

enum TuiSkin {
  terminal('Терминал', 'как в терминальном приложении'),
  reader('Читалка', 'спокойнее, для телефона');

  const TuiSkin(this.label, this.description);

  final String label;
  final String description;

  static TuiSkin byName(String? name) =>
      values.firstWhere((skin) => skin.name == name, orElse: () => TuiSkin.reader);
}

/// What a bordered box is for. The palette turns it into a colour, so no screen
/// has to know which skin is on.
enum TuiRole { plain, question, mistakes }

@immutable
class TuiPalette {
  const TuiPalette({
    required this.skin,
    required this.background,
    required this.border,
    required this.rule,
    required this.text,
    required this.dim,
    required this.accent,
    required this.term,
    required this.questionBorder,
    required this.mistakesBorder,
    required this.fontFamily,
    required this.selection,
  });

  /// The Go app's colours, straight out of the screenshots in `docs/`.
  static const terminal = TuiPalette(
    skin: TuiSkin.terminal,
    background: Color(0xFF1E1E2E),
    border: Color(0xFF313244),
    rule: Color(0xFF313244),
    text: Color(0xFFCDD6F4),
    dim: Color(0xFF6C7086),
    accent: Color(0xFF94E2D5),
    term: Color(0xFFD7AF87),
    questionBorder: Color(0xFF94E2D5),
    mistakesBorder: Color(0xFF5F0000),
    fontFamily: 'NotoSansMono',
    selection: Color(0x2694E2D5),
  );

  /// The reader's definition box: `tui.css` and `definition-box.css`.
  static const reader = TuiPalette(
    skin: TuiSkin.reader,
    background: Color(0xFF0E0E0E),
    border: Color(0xFFB2B2B2),
    rule: Color(0xFF333333),
    text: Color(0xFFD8D8D8),
    dim: Color(0xFF8A8A8A),
    accent: Color(0xFFC8B98F),
    term: Color(0xFFC8B98F),
    // The definition box carries one border colour and separates meaning with
    // rules and labels instead, so a box's role does not change its outline.
    questionBorder: Color(0xFFB2B2B2),
    mistakesBorder: Color(0xFFB2B2B2),
    fontFamily: 'JetBrainsMono',
    selection: Color(0x24C8B98F),
  );

  static const all = [TuiPalette.terminal, TuiPalette.reader];

  static TuiPalette of(TuiSkin skin) =>
      skin == TuiSkin.terminal ? TuiPalette.terminal : TuiPalette.reader;

  final TuiSkin skin;
  final Color background;
  final Color border;
  final Color rule;
  final Color text;
  final Color dim;
  final Color accent;

  /// What a question, an expected answer, or a looked-up term is printed in.
  final Color term;
  final Color questionBorder;
  final Color mistakesBorder;
  final Color selection;
  final String fontFamily;

  Color borderFor(TuiRole role) => switch (role) {
        TuiRole.plain => skin == TuiSkin.terminal ? rule : border,
        TuiRole.question => questionBorder,
        TuiRole.mistakes => mistakesBorder,
      };

  /// The reader's box is outlined in one clean stroke; the terminal's panels are
  /// drawn with heavier box-drawing characters.
  double get borderWidth => 2;
}

/// The palette in force. Everything reads it through [Tui], and the root of the
/// app rebuilds on it, so switching skins repaints the whole tree at once.
final ValueNotifier<TuiPalette> tuiPalette = ValueNotifier(TuiPalette.reader);

/// The current palette's colours and text styles.
///
/// These are getters rather than constants because the palette is chosen at
/// runtime — which is also why nothing built out of them can be `const`.
abstract final class Tui {
  static TuiPalette get palette => tuiPalette.value;

  static Color get background => palette.background;
  static Color get border => palette.border;
  static Color get rule => palette.rule;
  static Color get text => palette.text;
  static Color get dim => palette.dim;
  static Color get accent => palette.accent;
  static Color get term => palette.term;
  static String get fontFamily => palette.fontFamily;

  static TextStyle get body =>
      TextStyle(fontFamily: fontFamily, fontSize: 15, color: text, height: 1.45);

  /// A question, or the answer that should have been given.
  static TextStyle get termText =>
      TextStyle(fontFamily: fontFamily, fontSize: 15, color: term, height: 1.45);

  /// A box's header: the term on the left of the definition box.
  static TextStyle get heading => TextStyle(
        fontFamily: fontFamily,
        fontSize: 14,
        color: accent,
        fontWeight: FontWeight.w700,
      );

  /// The counter on the right of a header, and the hint bar along the bottom.
  static TextStyle get hint =>
      TextStyle(fontFamily: fontFamily, fontSize: 13, color: dim, height: 1.4);

  /// The italic dim label the definition box puts above a sense.
  static TextStyle get label => TextStyle(
        fontFamily: fontFamily,
        fontSize: 13,
        color: dim,
        fontStyle: FontStyle.italic,
      );

  static ThemeData theme(TuiPalette palette) {
    final scheme = ColorScheme.dark(
      surface: palette.background,
      primary: palette.accent,
      onPrimary: palette.background,
      secondary: palette.accent,
      error: palette.term,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: palette.background,
      fontFamily: palette.fontFamily,
      splashFactory: NoSplash.splashFactory,
      highlightColor: palette.selection,
      textSelectionTheme: TextSelectionThemeData(
        cursorColor: palette.accent,
        selectionColor: palette.selection,
        selectionHandleColor: palette.accent,
      ),
      textTheme: TextTheme(bodyMedium: body, bodyLarge: body, titleMedium: heading),
      appBarTheme: AppBarTheme(
        backgroundColor: palette.background,
        foregroundColor: palette.dim,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: heading,
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: palette.background,
        contentTextStyle: body,
        shape: RoundedRectangleBorder(
          side: BorderSide(color: palette.border, width: palette.borderWidth),
        ),
        behavior: SnackBarBehavior.floating,
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(color: palette.accent),
    );
  }
}
