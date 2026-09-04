/// Port of `reconcileWith` in `cmd/sync/reconcile.go`, with the phone always
/// playing the initiating side.
///
/// The phone runs no listener of its own — a backgrounded app cannot keep one —
/// so nothing is ever pushed *to* it. Everything happens here, while the app is
/// open: it asks each partner what it has, pulls what is newer there and pushes
/// what is newer here.
///
/// Every configured partner is talked to, not just the first that answers. That
/// is deliberate: the daemon pauses its own watcher while it applies a
/// `/syncTopic`, so a machine that takes a topic from the phone does not pass it
/// on to the others. Reaching all of them is what makes a phone edit land
/// everywhere.
library;

import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'topic_state.dart';
import 'topic_store.dart';

const syncPort = 8081;
const _syncTimeout = Duration(seconds: 10);

/// What one reconcile run did, for the banner the menu shows afterwards.
class SyncReport {
  SyncReport();

  final List<String> pulled = [];
  final List<String> pushed = [];
  final Map<String, String> failures = {};

  bool get changedAnything => pulled.isNotEmpty || pushed.isNotEmpty;
  int get reachedPartners => _reached;
  int _reached = 0;

  void _partnerAnswered() => _reached++;
}

class SyncClient {
  SyncClient(this.store, {http.Client? client}) : _client = client ?? http.Client();

  final TopicStore store;
  final http.Client _client;

  /// Brings the phone and every partner to the same set of topics. Partners are
  /// handled in parallel, so one that is asleep never holds up the others; a
  /// partner that does not answer is simply skipped and picked up next time.
  Future<SyncReport> reconcileAll(List<String> partnerUrls) async {
    final report = SyncReport();
    await Future.wait(partnerUrls.map((url) => _reconcileWith(url, report)));
    return report;
  }

  Future<void> _reconcileWith(String partnerUrl, SyncReport report) async {
    final List<TopicState> partnerTopics;
    try {
      partnerTopics = await _askPartnerForTopics(partnerUrl);
    } catch (error) {
      report.failures[partnerUrl] = _describe(error);
      return;
    }
    report._partnerAnswered();

    final onlyLocal = {for (final topic in await store.localTopics()) topic.name};

    for (final partnerTopic in partnerTopics) {
      onlyLocal.remove(partnerTopic.name);

      // Another partner may have replaced this topic since the listing above,
      // so the comparison uses the state on disk right now.
      final local = await store.stateOf(partnerTopic.name);
      try {
        if (local == null) {
          await _pull(partnerUrl, partnerTopic.name, report);
        } else if (local.hash == partnerTopic.hash) {
          continue;
        } else if (partnerTopic.modified.isAfter(local.modified)) {
          await _pull(partnerUrl, partnerTopic.name, report);
        } else if (local.modified.isAfter(partnerTopic.modified)) {
          await _push(partnerUrl, partnerTopic.name, report);
        }
        // Changed on both sides at the same instant: the local copy is kept,
        // exactly as the daemon resolves it.
      } catch (error) {
        report.failures['$partnerUrl ${partnerTopic.name}'] = _describe(error);
      }
    }

    // Deletions are not reconciled — without tombstones a topic missing on one
    // side cannot be told apart from one newly added on the other — so a topic
    // the partner lacks is offered to it rather than dropped here.
    for (final name in onlyLocal) {
      try {
        await _push(partnerUrl, name, report);
      } catch (error) {
        report.failures['$partnerUrl $name'] = _describe(error);
      }
    }
  }

  Future<List<TopicState>> _askPartnerForTopics(String partnerUrl) async {
    final response = await _get(Uri.parse('$partnerUrl/topics'));
    final listing = jsonDecode(response.body) as List<dynamic>;
    return listing
        .map((topic) => TopicState.fromJson(topic as Map<String, dynamic>))
        .toList();
  }

  Future<void> _pull(String partnerUrl, String name, SyncReport report) async {
    // Topic names hold spaces and Cyrillic, so the query has to be encoded.
    final response = await _get(
      Uri.parse('$partnerUrl/topic').replace(queryParameters: {'name': name}),
    );
    final payload = TopicPayload.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );

    if (await store.applyPayload(payload)) {
      report.pulled.add(name);
    }
  }

  Future<void> _push(String partnerUrl, String name, SyncReport report) async {
    final payload = await store.payloadOf(name);
    final response = await _client
        .post(
          Uri.parse('$partnerUrl/syncTopic'),
          headers: {'Content-Type': 'application/json'},
          body: jsonEncode(payload.toJson()),
        )
        .timeout(_syncTimeout);

    if (response.statusCode != 200) {
      throw HttpStatusException(response.statusCode);
    }

    report.pushed.add(name);
  }

  Future<http.Response> _get(Uri url) async {
    final response = await _client.get(url).timeout(_syncTimeout);
    if (response.statusCode != 200) {
      throw HttpStatusException(response.statusCode);
    }

    return response;
  }

  void close() => _client.close();

  static String _describe(Object error) =>
      error is TimeoutException ? 'нет ответа' : error.toString();
}

class HttpStatusException implements Exception {
  const HttpStatusException(this.statusCode);

  final int statusCode;

  @override
  String toString() => 'партнёр ответил $statusCode';
}
