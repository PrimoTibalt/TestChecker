/// The wire types the sync daemon speaks, ported from `cmd/sync/topicstate.go`.
/// The field names are the JSON ones the Go structs are tagged with, so both
/// ends parse each other without a translation layer.
library;

import 'dart:convert';

import 'package:crypto/crypto.dart';

/// One topic file as it exists on a single machine — what the two ends exchange
/// to work out who has the fresher copy.
class TopicState {
  const TopicState({
    required this.name,
    required this.modified,
    required this.hash,
  });

  factory TopicState.fromJson(Map<String, dynamic> json) => TopicState(
        name: json['name'] as String,
        modified: DateTime.parse(json['modified'] as String).toUtc(),
        hash: json['hash'] as String,
      );

  final String name;
  final DateTime modified;
  final String hash;
}

/// The whole content of a topic, together with the modification time that
/// should be preserved on the receiving side.
class TopicPayload {
  const TopicPayload({
    required this.name,
    required this.content,
    required this.modified,
  });

  factory TopicPayload.fromJson(Map<String, dynamic> json) => TopicPayload(
        name: json['name'] as String,
        content: json['content'] as String,
        modified: DateTime.parse(json['modified'] as String).toUtc(),
      );

  final String name;
  final String content;
  final DateTime modified;

  Map<String, dynamic> toJson() => {
        'name': name,
        'content': content,
        'modified': modified.toUtc().toIso8601String(),
      };
}

/// The same sha256-of-the-bytes the daemon puts in a listing, so a hash from
/// either side can be compared with one from the other.
String hashContent(String content) => sha256.convert(utf8.encode(content)).toString();
