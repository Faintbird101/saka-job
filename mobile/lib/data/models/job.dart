import 'dart:convert';

import 'cv_edit.dart';

/// The five axes behind a score.
///
/// Each is nullable and null means "the posting did not say", not zero — the
/// backend is deliberate about that distinction and the UI has to preserve it,
/// or a job with no stated salary would render a red pay bar it never earned.
class ScoreAxes {
  const ScoreAxes({this.skills, this.seniority, this.domain, this.location, this.pay});

  final int? skills;
  final int? seniority;
  final int? domain;
  final int? location;
  final int? pay;

  static ScoreAxes? tryParse(dynamic raw) {
    if (raw == null) return null;
    final map = switch (raw) {
      Map() => Map<String, dynamic>.from(raw),
      String s when s.isNotEmpty => () {
          try {
            final decoded = jsonDecode(s);
            return decoded is Map ? Map<String, dynamic>.from(decoded) : null;
          } catch (_) {
            return null;
          }
        }(),
      _ => null,
    };
    if (map == null) return null;

    int? at(String key) {
      final v = map[key];
      if (v is int) return v;
      if (v is num) return v.round();
      return null;
    }

    return ScoreAxes(
      skills: at('skills'),
      seniority: at('seniority'),
      domain: at('domain'),
      location: at('location'),
      pay: at('pay'),
    );
  }

  /// Every axis in display order, including the unknown ones — the design
  /// shows all five rows and marks the missing ones rather than hiding them.
  List<({String key, int? value})> get all => [
        (key: 'skills', value: skills),
        (key: 'seniority', value: seniority),
        (key: 'domain', value: domain),
        (key: 'location', value: location),
        (key: 'pay', value: pay),
      ];

  /// The lowest KNOWN axis — the one the design calls out as holding a match
  /// back. Unknown axes can never be "weakest"; they are simply not known.
  ({String key, int value})? get weakest {
    ({String key, int value})? worst;
    for (final a in all) {
      final v = a.value;
      if (v == null) continue;
      if (worst == null || v < worst.value) worst = (key: a.key, value: v);
    }
    return worst;
  }

  bool get hasAny => all.any((a) => a.value != null);
}

/// A job, as the API returns it.
///
/// Only the fields the app actually renders are mapped; the rest stay in the
/// response. Parsing is defensive throughout because several fields are
/// genuinely absent in live data — salary is usually null, and jobs scored
/// before the axes existed have no breakdown at all.
class Job {
  const Job({
    required this.id,
    required this.title,
    required this.organization,
    required this.status,
    this.score,
    this.axes,
    this.url,
    this.country,
    this.locationRaw,
    this.workArrangement,
    this.employmentType,
    this.seniority,
    this.orgDomain,
    this.aiSummary,
    this.matchedSkills = const [],
    this.missingSkills = const [],
    this.salaryMin,
    this.salaryMax,
    this.salaryCurrency,
    this.datePosted,
    this.generatedAt,
    this.createdAt,
    this.cvEdits = const [],
  });

  final String id;
  final String title;
  final String organization;
  final String status;
  final int? score;
  final ScoreAxes? axes;
  final String? url;
  final String? country;
  final String? locationRaw;
  final String? workArrangement;
  final String? employmentType;
  final String? seniority;
  final String? orgDomain;
  final String? aiSummary;
  final List<String> matchedSkills;
  final List<String> missingSkills;
  final num? salaryMin;
  final num? salaryMax;
  final String? salaryCurrency;
  final DateTime? datePosted;
  final DateTime? generatedAt;
  final DateTime? createdAt;

  /// What the agent changed in the CV, already verified server-side.
  final List<CvEdit> cvEdits;

  factory Job.fromJson(Map<String, dynamic> json) => Job(
        id: (json['id'] ?? '') as String,
        title: (json['title'] ?? '') as String,
        organization: (json['organization'] ?? '') as String,
        status: (json['status'] ?? '') as String,
        score: json['score'] is num ? (json['score'] as num).round() : null,
        axes: ScoreAxes.tryParse(json['score_axes']),
        url: _str(json['url']),
        country: _str(json['country']),
        locationRaw: _str(json['location_raw']),
        workArrangement: _str(json['work_arrangement']),
        employmentType: _str(json['employment_type']),
        seniority: _str(json['seniority']),
        orgDomain: _str(json['org_domain']),
        aiSummary: _str(json['ai_summary']),
        matchedSkills: _stringList(json['matched_skills']),
        missingSkills: _stringList(json['missing_skills']),
        salaryMin: json['salary_min'] as num?,
        salaryMax: json['salary_max'] as num?,
        salaryCurrency: _str(json['salary_currency']),
        datePosted: _date(json['date_posted']),
        generatedAt: _date(json['generated_at']),
        createdAt: _date(json['created_at']),
        cvEdits: _editList(json['cv_edits']),
      );

  /// JSONB arrives as a List, but tolerate a raw string in case the column is
  /// ever returned unparsed — a missing diff is a worse failure than a
  /// defensive parser.
  static List<CvEdit> _editList(dynamic v) {
    final raw = switch (v) {
      List() => v,
      String s when s.isNotEmpty => () {
          try {
            final d = jsonDecode(s);
            return d is List ? d : const [];
          } catch (_) {
            return const [];
          }
        }(),
      _ => const [],
    };
    return raw
        .whereType<Map>()
        .map((e) => CvEdit.fromJson(Map<String, dynamic>.from(e)))
        .toList();
  }

  /// "Nairobi, Kenya · Hybrid" — whichever parts exist.
  String get locationLabel {
    final parts = [
      if (locationRaw != null && locationRaw!.isNotEmpty) locationRaw,
      if (country != null && country!.isNotEmpty && country != locationRaw) country,
    ].whereType<String>();
    return parts.join(', ');
  }

  /// Null rather than an empty range: most postings state no salary, and a
  /// blank "KES —" line is worse than no line at all.
  String? get salaryLabel {
    if (salaryMin == null && salaryMax == null) return null;
    final currency = salaryCurrency ?? '';
    String fmt(num v) {
      if (v >= 1000000) return '${(v / 1000000).toStringAsFixed(1)}M';
      if (v >= 1000) return '${(v / 1000).round()}k';
      return v.round().toString();
    }

    if (salaryMin != null && salaryMax != null) {
      return '$currency ${fmt(salaryMin!)} – ${fmt(salaryMax!)}'.trim();
    }
    return '$currency ${fmt((salaryMin ?? salaryMax)!)}'.trim();
  }

  bool get isAwaitingApproval => status == 'AwaitingApproval';
  bool get hasDocuments => generatedAt != null;

  static String? _str(dynamic v) {
    if (v is String && v.trim().isNotEmpty) return v;
    return null;
  }

  static DateTime? _date(dynamic v) =>
      v is String && v.isNotEmpty ? DateTime.tryParse(v) : null;

  /// JSONB arrays arrive as a List from Dio, but as a String if the column was
  /// ever returned unparsed. Handling both keeps a chip row from vanishing.
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

/// One page of jobs plus the total, so the list can show "50 of 312".
class JobPage {
  const JobPage({required this.jobs, required this.total, this.limit = 0, this.offset = 0});

  final List<Job> jobs;
  final int total;
  final int limit;
  final int offset;

  factory JobPage.fromJson(Map<String, dynamic> json) => JobPage(
        jobs: ((json['jobs'] as List?) ?? const [])
            .map((e) => Job.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        total: (json['total'] as num?)?.toInt() ?? 0,
        limit: (json['limit'] as num?)?.toInt() ?? 0,
        offset: (json['offset'] as num?)?.toInt() ?? 0,
      );

  static const empty = JobPage(jobs: [], total: 0);
}
