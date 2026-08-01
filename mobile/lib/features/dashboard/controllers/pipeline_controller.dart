import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/services/jobs_service.dart';

/// One stage of the funnel.
class Stage {
  const Stage({
    required this.key,
    required this.count,
    required this.statuses,
  });

  final String key;
  final int count;

  /// The backend statuses rolled into this row, so tapping through can filter
  /// the job list by exactly what the number counted.
  final List<String> statuses;
}

/// Screen 08 — the pipeline.
///
/// The deck's framing is "proof, not vanity metrics": one bar for the whole
/// funnel, five stage rows, and a single honest headline number.
class PipelineController extends GetxController {
  JobsService get _jobs => Get.find<JobsService>();

  final counts = <String, int>{}.obs;
  final total = 0.obs;
  final loading = true.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// Stages, grouped by what they mean rather than one row per status —
  /// sixteen rows would be a database dump, not a funnel.
  static const _grouping = <String, List<String>>{
    'queue': ['AwaitingApproval', 'CVGenerated', 'Approved'],
    'applied': ['Applied', 'ManualApply', 'FollowUpSent'],
    'screening': ['Acknowledged'],
    'interview': ['Interviewing'],
    'offer': ['OfferReceived'],
  };

  List<Stage> get stages => [
        for (final entry in _grouping.entries)
          Stage(
            key: entry.key,
            count: entry.value.fold(0, (sum, s) => sum + (counts[s] ?? 0)),
            statuses: entry.value,
          ),
      ];

  /// Live applications — everything sent that has not concluded. The headline
  /// number, and deliberately not "total jobs seen", which would flatter.
  int get live => stages
      .where((s) => s.key != 'queue')
      .fold(0, (sum, s) => sum + s.count);

  /// Replies as a share of what was actually sent.
  ///
  /// Null rather than 0% when nothing has been sent yet: "0% reply rate" on an
  /// empty pipeline reads as failure when it only means "not started".
  double? get replyRate {
    final sent = live;
    if (sent == 0) return null;
    final replied = stages
        .where((s) => s.key == 'screening' || s.key == 'interview' || s.key == 'offer')
        .fold(0, (sum, s) => sum + s.count);
    return replied / sent;
  }

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    try {
      final stats = await _jobs.stats();
      final byStatus = (stats['by_status'] as List?) ?? const [];
      counts.assignAll({
        for (final row in byStatus)
          (row as Map)['status'] as String: (row['count'] as num).toInt(),
      });
      total.value = (stats['total'] as num?)?.toInt() ?? 0;
    } on Failure catch (f) {
      error.value = f;
    } finally {
      loading.value = false;
    }
  }

  @override
  Future<void> refresh() => load();
}
