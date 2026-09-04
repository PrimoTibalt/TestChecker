/// Where the app remembers which machines to sync with.
///
/// The daemon takes its partners from `TAILSCALE_PARTNER_IP`; the phone has no
/// environment to read, so the same list is kept here and edited on the settings
/// screen. Only the address is stored — everyone listens on [syncPort], the way
/// `cmd/sync/config.go` assumes.
library;

import 'package:shared_preferences/shared_preferences.dart';

import '../ui/theme/tui_theme.dart';
import 'sync_client.dart';

class Settings {
  Settings(this._prefs);

  static Future<Settings> open() async => Settings(await SharedPreferences.getInstance());

  static const _partnersKey = 'partners';
  static const _skinKey = 'skin';

  final SharedPreferences _prefs;

  /// The tailnet addresses of the machines running the daemon.
  List<String> get partners => _prefs.getStringList(_partnersKey) ?? const [];

  List<String> get partnerUrls => [for (final ip in partners) 'http://$ip:$syncPort'];

  /// Which of the two looks the app wears.
  TuiSkin get skin => TuiSkin.byName(_prefs.getString(_skinKey));

  Future<void> setSkin(TuiSkin skin) async => _prefs.setString(_skinKey, skin.name);

  Future<void> setPartners(List<String> partners) async {
    final cleaned = <String>[];
    for (final partner in partners) {
      final trimmed = partner.trim();
      if (trimmed.isNotEmpty && !cleaned.contains(trimmed)) {
        cleaned.add(trimmed);
      }
    }

    await _prefs.setStringList(_partnersKey, cleaned);
  }
}
