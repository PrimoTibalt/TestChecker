import 'package:flutter/material.dart';

import '../data/sync_client.dart';
import 'app_state.dart';
import 'theme/tui_theme.dart';
import 'widgets/tui_box.dart';

/// Two settings: which machines to sync with — the phone's stand-in for
/// `TAILSCALE_PARTNER_IP`, since it has no environment to read it from — and
/// which of the two looks the app wears.
class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key, required this.state});

  final AppState state;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  late List<String> _partners = List.of(widget.state.settings.partners);
  final _newPartner = TextEditingController();

  @override
  void dispose() {
    _newPartner.dispose();
    super.dispose();
  }

  Future<void> _savePartners() async {
    await widget.state.settings.setPartners(_partners);
    setState(() => _partners = List.of(widget.state.settings.partners));
  }

  Future<void> _add() async {
    final address = _newPartner.text.trim();
    if (address.isEmpty) {
      return;
    }

    _partners.add(address);
    _newPartner.clear();
    await _savePartners();
  }

  @override
  Widget build(BuildContext context) {
    final skin = widget.state.settings.skin;

    return Scaffold(
      appBar: AppBar(leading: const BackButton()),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 16),
          children: [
            TuiBox(
              title: 'Внешний вид',
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  for (final option in TuiSkin.values)
                    InkWell(
                      // Repaints the whole app, this screen included, so the
                      // choice is visible the moment it is made.
                      onTap: () async {
                        await widget.state.useSkin(option);
                        if (mounted) {
                          setState(() {});
                        }
                      },
                      child: Padding(
                        padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              option == skin ? '(*) ' : '( ) ',
                              style: Tui.body.copyWith(
                                color: option == skin ? Tui.accent : Tui.dim,
                              ),
                            ),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    option.label,
                                    style: Tui.body.copyWith(
                                      color: option == skin ? Tui.accent : Tui.text,
                                    ),
                                  ),
                                  Text(option.description, style: Tui.label),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 12),
            TuiBox(
              title: 'Машины для синхронизации',
              counter: '${_partners.length}',
              hints: Row(
                children: [
                  TuiAction(label: 'добавить', onTap: _add, emphasised: true),
                  const TuiHintSeparator(),
                  Expanded(child: Text('порт $syncPort у всех')),
                ],
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Tailscale-адреса машин с демоном.', style: Tui.label),
                  const SizedBox(height: 8),
                  for (final partner in _partners)
                    Row(
                      children: [
                        Expanded(child: Text('$partner:$syncPort', style: Tui.body)),
                        TuiAction(
                          label: 'убрать',
                          onTap: () async {
                            _partners.remove(partner);
                            await _savePartners();
                          },
                        ),
                      ],
                    ),
                  const SizedBox(height: 4),
                  TextField(
                    controller: _newPartner,
                    style: Tui.body,
                    onSubmitted: (_) => _add(),
                    decoration: InputDecoration(
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(vertical: 8),
                      hintText: '100.x.y.z',
                      hintStyle: Tui.hint,
                      enabledBorder: UnderlineInputBorder(
                        borderSide: BorderSide(color: Tui.rule),
                      ),
                      focusedBorder: UnderlineInputBorder(
                        borderSide: BorderSide(color: Tui.accent),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
