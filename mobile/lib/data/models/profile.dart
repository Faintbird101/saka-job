import 'dart:convert';

/// The single settings row that steers the agent.
///
/// Mirrors the backend's profile: search terms, the score threshold, per-stage
/// models, and the notification switches. Everything here changes what the
/// pipeline does on its next run, which is why the design calls this screen
/// "steering the agent" rather than "settings".
class Profile {
  const Profile({
    required this.masterCv,
    required this.searchTitles,
    required this.preferredSkills,
    required this.minScoreThreshold,
    required this.maxJobsPerRun,
    this.scoringModel,
    this.generationModel,
    this.coverLetterNotes,
    this.notifyEmail,
    this.preferredLocations = const [],
    this.remotePreference,
    this.salaryFloor,
    this.salaryCurrency,
    this.pushOnApproval = true,
    this.pushOnReply = true,
    this.pushOnFollowUp = true,
    this.pushOnFailure = true,
    this.updatedAt,
  });

  final String masterCv;
  final List<String> searchTitles;
  final List<String> preferredSkills;
  final int minScoreThreshold;
  final int maxJobsPerRun;

  final String? scoringModel;
  final String? generationModel;
  final String? coverLetterNotes;
  final String? notifyEmail;

  final List<String> preferredLocations;
  final String? remotePreference;
  final num? salaryFloor;
  final String? salaryCurrency;

  final bool pushOnApproval;
  final bool pushOnReply;
  final bool pushOnFollowUp;
  final bool pushOnFailure;

  final DateTime? updatedAt;

  /// Whether the CV is set at all. Scoring refuses to run without one, so the
  /// profile screen leads with this rather than burying it.
  bool get hasCv => masterCv.trim().isNotEmpty;

  /// A crude readability signal for the CV strength meter.
  ///
  /// Deliberately not presented as a score out of 100 by any objective
  /// standard — it measures length and whether the preferred skills actually
  /// appear, which is what the keyword scorer will look for. Anything more
  /// confident would be inventing a judgement.
  int get cvStrength {
    if (!hasCv) return 0;
    final length = masterCv.trim().length;
    // Below ~600 characters there is not enough to tailor from; past ~3000
    // extra length stops helping.
    final lengthScore = (length / 3000).clamp(0.0, 1.0) * 60;
    final mentioned = preferredSkills
        .where((s) => masterCv.toLowerCase().contains(s.toLowerCase()))
        .length;
    final skillScore = preferredSkills.isEmpty
        ? 20.0
        : (mentioned / preferredSkills.length) * 40;
    return (lengthScore + skillScore).round().clamp(0, 100);
  }

  factory Profile.fromJson(Map<String, dynamic> json) => Profile(
        masterCv: (json['master_cv'] ?? '') as String,
        searchTitles: _stringList(json['search_titles']),
        preferredSkills: _stringList(json['preferred_skills']),
        minScoreThreshold: (json['min_score_threshold'] as num?)?.toInt() ?? 75,
        maxJobsPerRun: (json['max_jobs_per_run'] as num?)?.toInt() ?? 10,
        scoringModel: _str(json['scoring_model']),
        generationModel: _str(json['generation_model']),
        coverLetterNotes: _str(json['cover_letter_notes']),
        notifyEmail: _str(json['notify_email']),
        preferredLocations: _stringList(json['preferred_locations']),
        remotePreference: _str(json['remote_preference']),
        salaryFloor: json['salary_floor'] as num?,
        salaryCurrency: _str(json['salary_currency']),
        pushOnApproval: json['push_on_approval'] as bool? ?? true,
        pushOnReply: json['push_on_reply'] as bool? ?? true,
        pushOnFollowUp: json['push_on_followup'] as bool? ?? true,
        pushOnFailure: json['push_on_failure'] as bool? ?? true,
        updatedAt: json['updated_at'] is String
            ? DateTime.tryParse(json['updated_at'] as String)
            : null,
      );

  static String? _str(dynamic v) =>
      v is String && v.trim().isNotEmpty ? v : null;

  /// JSONB arrays arrive as a List, but tolerate a raw string in case a column
  /// is ever returned unparsed — a missing chip row is a worse failure than a
  /// slightly defensive parser.
  static List<String> _stringList(dynamic v) {
    if (v is List) return v.map((e) => e.toString()).toList();
    if (v is String && v.isNotEmpty) {
      try {
        final decoded = jsonDecode(v);
        if (decoded is List) return decoded.map((e) => e.toString()).toList();
      } catch (_) {}
    }
    return const [];
  }
}
