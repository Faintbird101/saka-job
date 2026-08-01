/// One inbound email, matched to an application.
///
/// The backend classifies replies but deliberately does NOT apply the
/// consequential ones — a rejection or an interview invitation is recorded as a
/// suggestion and waits for a human, because acting on a misread would either
/// abandon a live opportunity or claim progress that never happened. This model
/// is what the confirmation screen renders.
class JobEvent {
  const JobEvent({
    required this.id,
    required this.kind,
    required this.confidence,
    this.jobId,
    this.classifier,
    this.sender,
    this.senderDomain,
    this.subject,
    this.excerpt,
    this.receivedAt,
    this.matchScore = 0,
    this.matchReason,
    this.suggestedStatus,
    this.confirmed,
    this.appliedAt,
  });

  final String id;
  final String? jobId;

  /// acknowledgement | rejection | interview | offer | other
  final String kind;
  final int confidence;

  /// `keyword` or `llm:<model>`, so a classification can be traced to whatever
  /// produced it.
  final String? classifier;

  final String? sender;
  final String? senderDomain;
  final String? subject;

  /// A fragment that justifies the classification, not the whole body — the
  /// backend stores only enough to explain itself.
  final String? excerpt;
  final DateTime? receivedAt;

  final int matchScore;
  final String? matchReason;

  /// Empty when the email implies no status change.
  final String? suggestedStatus;

  /// Null while awaiting a decision; true accepted, false dismissed.
  final bool? confirmed;
  final DateTime? appliedAt;

  /// Whether this is sitting in the queue waiting on you.
  bool get isPending => confirmed == null && (suggestedStatus?.isNotEmpty ?? false);

  /// Applied automatically — only ever an acknowledgement, which is not a
  /// decision about the outcome.
  bool get wasAutoApplied => appliedAt != null && confirmed == true;

  /// Whether the classifier was unsure enough that the wording should hedge.
  bool get isLowConfidence => confidence < 70;

  factory JobEvent.fromJson(Map<String, dynamic> json) => JobEvent(
        id: (json['id'] ?? '') as String,
        jobId: json['job_id'] as String?,
        kind: (json['kind'] ?? 'other') as String,
        confidence: (json['confidence'] as num?)?.toInt() ?? 0,
        classifier: _str(json['classifier']),
        sender: _str(json['sender']),
        senderDomain: _str(json['sender_domain']),
        subject: _str(json['subject']),
        excerpt: _str(json['excerpt']),
        receivedAt: _date(json['received_at']),
        matchScore: (json['match_score'] as num?)?.toInt() ?? 0,
        matchReason: _str(json['match_reason']),
        suggestedStatus: _str(json['suggested_status']),
        confirmed: json['confirmed'] as bool?,
        appliedAt: _date(json['applied_at']),
      );

  static String? _str(dynamic v) =>
      v is String && v.trim().isNotEmpty ? v : null;

  static DateTime? _date(dynamic v) =>
      v is String && v.isNotEmpty ? DateTime.tryParse(v) : null;
}
