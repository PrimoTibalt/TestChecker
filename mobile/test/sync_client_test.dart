import 'dart:convert';
import 'dart:io';

import 'package:checktests_mobile/data/sync_client.dart';
import 'package:checktests_mobile/data/topic_state.dart';
import 'package:checktests_mobile/data/topic_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;

/// A stand-in for a desktop running the sync daemon: it serves the three
/// endpoints the client uses and records what was pushed to it, the way
/// `reconcile_test.go` does with httptest.
class FakePartner {
  FakePartner(this._server) : url = 'http://127.0.0.1:${_server.port}' {
    _server.listen(_handle);
  }

  static Future<FakePartner> start() async =>
      FakePartner(await HttpServer.bind(InternetAddress.loopbackIPv4, 0));

  final HttpServer _server;

  /// Read at construction: the port is gone once the server is stopped, and the
  /// tests below stop one on purpose to play an unreachable machine.
  final String url;
  final Map<String, TopicPayload> topics = {};
  final List<TopicPayload> received = [];
  int listingRequests = 0;

  void hold(String name, String content, DateTime modified) =>
      topics[name] = TopicPayload(name: name, content: content, modified: modified);

  Future<void> _handle(HttpRequest request) async {
    final response = request.response..headers.contentType = ContentType.json;
    switch ('${request.method} ${request.uri.path}') {
      case 'GET /topics':
        listingRequests++;
        response.write(jsonEncode([
          for (final topic in topics.values)
            {
              'name': topic.name,
              'modified': topic.modified.toUtc().toIso8601String(),
              'hash': hashContent(topic.content),
            }
        ]));
      case 'GET /topic':
        final name = request.uri.queryParameters['name'];
        final topic = topics[name];
        if (topic == null) {
          response.statusCode = HttpStatus.notFound;
        } else {
          response.write(jsonEncode(topic.toJson()));
        }
      case 'POST /syncTopic':
        final body = jsonDecode(await utf8.decodeStream(request)) as Map<String, dynamic>;
        final payload = TopicPayload.fromJson(body);
        received.add(payload);
        topics[payload.name] = payload;
      default:
        response.statusCode = HttpStatus.notFound;
    }

    await response.close();
  }

  Future<void> stop() => _server.close(force: true);
}

void main() {
  late Directory dir;
  late TopicStore store;
  late SyncClient client;
  late FakePartner partner;

  final earlier = DateTime.utc(2026, 9, 1, 12);
  final later = DateTime.utc(2026, 9, 2, 12);

  setUp(() async {
    dir = await Directory.systemTemp.createTemp('checktests_sync');
    store = TopicStore(Directory(p.join(dir.path, 'Questions')))
      ..questionsDir.createSync(recursive: true);
    client = SyncClient(store);
    partner = await FakePartner.start();
  });

  tearDown(() async {
    client.close();
    await partner.stop();
    await dir.delete(recursive: true);
  });

  Future<void> writeLocal(String name, String content, DateTime modified) async {
    await store.writeTopic(name, content);
    await store.topicFile(name).setLastModified(modified);
  }

  test('pulls a topic the phone does not have', () async {
    partner.hold('Go printf', 'Вывести тип аргумента/!/%T\n', later);

    final report = await client.reconcileAll([partner.url]);

    expect(report.pulled, ['Go printf']);
    expect(await store.readTopic('Go printf'), 'Вывести тип аргумента/!/%T\n');
    // The sender's mtime comes over with it, so the copies compare equal next
    // time rather than looking newer here.
    expect((await store.stateOf('Go printf'))!.modified, later);
  });

  test('pushes a topic the partner does not have', () async {
    await writeLocal('bash скриптинг', 'q/!/a\n', later);

    final report = await client.reconcileAll([partner.url]);

    expect(report.pushed, ['bash скриптинг']);
    expect(partner.received.single.name, 'bash скриптинг');
    expect(partner.received.single.content, 'q/!/a\n');
  });

  test('a topic with the same content is left alone', () async {
    const content = 'вопрос/!/ответ\n';
    partner.hold('Git комманды', content, later);
    await writeLocal('Git комманды', content, earlier);

    final report = await client.reconcileAll([partner.url]);

    expect(report.pulled, isEmpty);
    expect(report.pushed, isEmpty, reason: 'equal hashes settle it before mtimes');
  });

  test('the newer side wins in each direction', () async {
    partner.hold('newer there', 'partner/!/copy\n', later);
    await writeLocal('newer there', 'phone/!/copy\n', earlier);
    partner.hold('newer here', 'partner/!/copy\n', earlier);
    await writeLocal('newer here', 'phone/!/copy\n', later);

    await client.reconcileAll([partner.url]);

    expect(await store.readTopic('newer there'), 'partner/!/copy\n');
    expect(partner.topics['newer here']!.content, 'phone/!/copy\n');
  });

  test('an older payload never overwrites a newer local copy', () async {
    await writeLocal('Go printf', 'phone/!/copy\n', later);

    final applied = await store.applyPayload(
      TopicPayload(name: 'Go printf', content: 'stale/!/copy\n', modified: earlier),
    );

    expect(applied, isFalse);
    expect(await store.readTopic('Go printf'), 'phone/!/copy\n');
  });

  test('an unreachable partner is reported, not thrown', () async {
    await writeLocal('Go printf', 'q/!/a\n', later);
    await partner.stop();

    final report = await client.reconcileAll([partner.url]);

    expect(report.failures, hasLength(1));
    expect(report.reachedPartners, 0);
    expect(await store.readTopic('Go printf'), 'q/!/a\n', reason: 'the phone keeps working offline');
  });

  test('one dead partner does not stop the others', () async {
    final alive = await FakePartner.start();
    alive.hold('Go printf', 'q/!/a\n', later);
    await partner.stop();

    final report = await client.reconcileAll([partner.url, alive.url]);

    expect(report.pulled, ['Go printf']);
    expect(report.failures, hasLength(1));
    expect(report.reachedPartners, 1);
    await alive.stop();
  });

  test('every partner is talked to, since none of them relays a push', () async {
    final second = await FakePartner.start();
    await writeLocal('Go printf', 'q/!/a\n', later);

    await client.reconcileAll([partner.url, second.url]);

    expect(partner.received, hasLength(1));
    expect(second.received, hasLength(1));
    await second.stop();
  });
}
