/// The phone's side of `topicpersistence`, plus the file handling of
/// `cmd/sync/topicstate.go`.
///
/// The layout is the one the desktop uses — a `Questions` directory holding one
/// plain file per topic, the file name being the topic name — so a file copied
/// either way needs no conversion.
library;

import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

import '../domain/qa_line.dart';
import 'topic_state.dart';

class TopicStore {
  TopicStore(this.questionsDir);

  /// The store the app itself uses. The directory is app-private, so topic
  /// names holding spaces and Cyrillic are safe in it.
  static Future<TopicStore> open() async {
    final documents = await getApplicationDocumentsDirectory();
    final store = TopicStore(Directory(p.join(documents.path, 'Questions')));
    await store.questionsDir.create(recursive: true);
    return store;
  }

  final Directory questionsDir;

  /// Resolves a topic name to a file inside the Questions dir. Names arrive
  /// over the network, so anything that is not a plain file name is rejected
  /// rather than allowed to escape the directory — `TopicPath`'s job in Go.
  File topicFile(String name) {
    if (name.isEmpty || name == '.' || name == '..' || p.basename(name) != name) {
      throw ArgumentError.value(name, 'name', 'invalid topic name');
    }

    return File(p.join(questionsDir.path, name));
  }

  /// Every topic on the phone, in the order the picker shows them. Dot-prefixed
  /// entries are the half-written temporary files of [writeTopic], never topics.
  Future<List<String>> listTopics() async {
    if (!await questionsDir.exists()) {
      return [];
    }

    final names = <String>[];
    await for (final entry in questionsDir.list()) {
      final name = p.basename(entry.path);
      if (entry is! File || name.startsWith('.')) {
        continue;
      }

      names.add(name);
    }

    names.sort();
    return names;
  }

  Future<String> readTopic(String name) async {
    final file = topicFile(name);
    if (!await file.exists()) {
      return '';
    }

    return file.readAsString();
  }

  Future<List<Question>> questionsOf(String name) async => parseTopic(await readTopic(name));

  /// Appends one pair, the way the TUI's "add question" does. A topic file is
  /// written a line at a time, so a copy that lost its trailing newline gets one
  /// back before the new pair goes on the end.
  Future<void> appendPair(String name, String question, String answer) async {
    var content = await readTopic(name);
    if (content.isNotEmpty && !content.endsWith('\n')) {
      content += '\n';
    }

    await writeTopic(name, content + formatQaLine(question, answer));
  }

  /// Replaces a topic in one step: the content goes to a temporary file in the
  /// same directory and is renamed into place, so a reader — the sync client on
  /// another isolate, or the next screen — never sees a torn file. The temp name
  /// is dot-prefixed, which is what the daemon's watcher already ignores.
  Future<void> writeTopic(String name, String content) async {
    final destination = topicFile(name);
    final temp = File(p.join(questionsDir.path, '.$name.tmp'));
    await temp.writeAsString(content, flush: true);
    await temp.rename(destination.path);
  }

  Future<void> removeTopic(String name) async {
    final file = topicFile(name);
    if (await file.exists()) {
      await file.delete();
    }
  }

  /// The state of one topic as the daemon would report it, or null when there
  /// is no such topic here.
  Future<TopicState?> stateOf(String name) async {
    final file = topicFile(name);
    if (!await file.exists()) {
      return null;
    }

    return TopicState(
      name: name,
      modified: (await file.lastModified()).toUtc(),
      hash: hashContent(await file.readAsString()),
    );
  }

  Future<List<TopicState>> localTopics() async {
    final states = <TopicState>[];
    for (final name in await listTopics()) {
      final state = await stateOf(name);
      if (state != null) {
        states.add(state);
      }
    }

    return states;
  }

  /// Stores a topic that came from a partner and reports whether it actually
  /// replaced anything — the compare-and-set of `WriteTopic`. A copy that is
  /// already newer than the one being offered is kept, so the newest version
  /// wins no matter what order the partners are talked to in.
  Future<bool> applyPayload(TopicPayload payload) async {
    final current = await stateOf(payload.name);
    if (current != null) {
      if (current.hash == hashContent(payload.content)) {
        return false;
      }
      if (current.modified.isAfter(payload.modified)) {
        return false;
      }
    }

    await writeTopic(payload.name, payload.content);
    // Carry the sender's mtime over, so the two copies compare equal next time
    // — the daemon's `Chtimes` after a write.
    await topicFile(payload.name).setLastModified(payload.modified);
    return true;
  }

  Future<TopicPayload> payloadOf(String name) async {
    final file = topicFile(name);
    return TopicPayload(
      name: name,
      content: await file.readAsString(),
      modified: (await file.lastModified()).toUtc(),
    );
  }
}
