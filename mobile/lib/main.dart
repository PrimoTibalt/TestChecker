import 'package:flutter/material.dart';

import 'ui/app_state.dart';
import 'ui/menu_screen.dart';
import 'ui/theme/tui_theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(CheckTestsApp(state: await AppState.load()));
}

class CheckTestsApp extends StatelessWidget {
  const CheckTestsApp({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) => ValueListenableBuilder<TuiPalette>(
        valueListenable: tuiPalette,
        builder: (context, palette, _) => MaterialApp(
          title: 'checkTests',
          debugShowCheckedModeBanner: false,
          theme: Tui.theme(palette),
          home: MenuScreen(state: state),
        ),
      );
}
