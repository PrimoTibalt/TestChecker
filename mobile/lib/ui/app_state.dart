/// The one object the screens share: the store on disk, the peer list and the
/// sync client that ties the two to the desktops.
library;

import 'package:flutter/foundation.dart';

import '../data/settings.dart';
import '../data/sync_client.dart';
import '../data/topic_store.dart';
import 'theme/tui_theme.dart';

class AppState extends ChangeNotifier {
  AppState({required this.store, required this.settings}) : sync = SyncClient(store) {
    tuiPalette.value = TuiPalette.of(settings.skin);
  }

  static Future<AppState> load() async => AppState(
        store: await TopicStore.open(),
        settings: await Settings.open(),
      );

  final TopicStore store;
  final Settings settings;
  final SyncClient sync;

  bool syncing = false;

  /// Switching the look repaints the whole tree: everything reads the palette
  /// through [tuiPalette], and the root listens to it.
  Future<void> useSkin(TuiSkin skin) async {
    await settings.setSkin(skin);
    tuiPalette.value = TuiPalette.of(skin);
    notifyListeners();
  }

  /// Reconciles with every configured machine. Never throws: a phone with no
  /// tailnet in reach still has to be able to run a test off its own copy.
  Future<SyncReport?> syncNow() async {
    if (syncing || settings.partnerUrls.isEmpty) {
      return null;
    }

    syncing = true;
    notifyListeners();
    try {
      return await sync.reconcileAll(settings.partnerUrls);
    } finally {
      syncing = false;
      notifyListeners();
    }
  }

  @override
  void dispose() {
    sync.close();
    super.dispose();
  }
}
