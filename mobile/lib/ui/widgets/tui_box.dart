/// The one bordered box every screen is built out of, shaped after the reader's
/// definition box: a header carrying a label on the left and a counter on the
/// right, a body, and an optional hint bar along the bottom, with a thin rule
/// between each pair.
///
/// It asks for a [TuiRole] rather than a colour, so the same screens come out in
/// either skin — the palette is what decides whether a question is outlined in
/// cyan or in the same neutral stroke as everything else.
library;

import 'package:flutter/material.dart';

import '../theme/tui_theme.dart';

class TuiBox extends StatelessWidget {
  const TuiBox({
    super.key,
    required this.child,
    this.role = TuiRole.plain,
    this.title,
    this.counter,
    this.hints,
    this.padding = const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
    this.fill = false,
  });

  final Widget child;
  final TuiRole role;

  /// The label on the left of the header. Without it there is no header.
  final String? title;

  /// The `1/1` on the right of the header.
  final String? counter;

  /// The bar along the bottom. Give it tappable text and it reads as the hint
  /// line of a terminal app rather than as a button.
  final Widget? hints;

  final EdgeInsets padding;

  /// Whether the body takes all the height the box is given, so a list inside
  /// it scrolls rather than pushing the hint bar off the screen. The box then
  /// has to be handed a bounded height — put it in an [Expanded].
  final bool fill;

  @override
  Widget build(BuildContext context) {
    final palette = Tui.palette;

    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        color: palette.background,
        border: Border.all(
          color: palette.borderFor(role),
          width: palette.borderWidth,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: fill ? MainAxisSize.max : MainAxisSize.min,
        children: [
          if (title != null) ...[
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.baseline,
                textBaseline: TextBaseline.alphabetic,
                children: [
                  Expanded(child: Text(title!, style: Tui.heading)),
                  if (counter != null) ...[
                    const SizedBox(width: 12),
                    Text(
                      counter!,
                      style: Tui.hint.copyWith(fontFeatures: const [FontFeature.tabularFigures()]),
                    ),
                  ],
                ],
              ),
            ),
            _Rule(),
          ],
          if (fill)
            Expanded(child: Padding(padding: padding, child: child))
          else
            Padding(padding: padding, child: child),
          if (hints != null) ...[
            _Rule(),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 7, 12, 7),
              child: DefaultTextStyle(style: Tui.hint, child: hints!),
            ),
          ],
        ],
      ),
    );
  }
}

class _Rule extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Container(height: 1, color: Tui.rule);
}

/// A hint-bar action: the terminal's `esc close` turned into something a thumb
/// can hit, without turning it into a Material button.
class TuiAction extends StatelessWidget {
  const TuiAction({super.key, required this.label, this.onTap, this.emphasised = false});

  final String label;
  final VoidCallback? onTap;
  final bool emphasised;

  @override
  Widget build(BuildContext context) {
    final enabled = onTap != null;
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
        child: Text(
          label,
          style: Tui.hint.copyWith(
            color: enabled ? (emphasised ? Tui.accent : Tui.dim) : Tui.rule,
            fontWeight: emphasised ? FontWeight.w700 : FontWeight.w400,
          ),
        ),
      ),
    );
  }
}

/// The separator the hint bars use between actions.
class TuiHintSeparator extends StatelessWidget {
  const TuiHintSeparator({super.key});

  @override
  Widget build(BuildContext context) =>
      Padding(padding: const EdgeInsets.symmetric(horizontal: 2), child: Text('·', style: Tui.hint));
}

/// Text as the panels print it: monospace, line breaks kept, long lines wrapped.
class TuiText extends StatelessWidget {
  const TuiText(this.text, {super.key, this.style});

  final String text;
  final TextStyle? style;

  @override
  Widget build(BuildContext context) => SelectableText(text, style: style ?? Tui.body);
}
