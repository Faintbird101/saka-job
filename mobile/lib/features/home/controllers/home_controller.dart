import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/job.dart';
import '../../../data/models/job_event.dart';
import '../../../data/services/events_service.dart';
import '../../../data/services/jobs_service.dart';
import '../../../shared/widgets/app_toast.dart';

/// State for the home feed.
///
/// Two lists, because the design shows two: what is blocked on you, and what
/// arrived recently. They are separate requests rather than one filtered
/// client-side, so the queue stays correct even when the recent list is paged.
class HomeController extends GetxController {
  JobsService get _jobs => Get.find<JobsService>();
  EventsService get _events => Get.find<EventsService>();

  final queue = <Job>[].obs;
  final recent = <Job>[].obs;

  /// Employer replies awaiting a decision. Surfaced on the feed because they
  /// are time-sensitive in a way an unapproved job is not — an interview
  /// invitation sitting unread for a week is a real cost.
  final replies = <JobEvent>[].obs;

  final loading = true.obs;
  final refreshing = false.obs;

  /// Set when a load fails with nothing cached to fall back on — the screen
  /// then shows the failure state rather than a permanently empty list.
  final Rxn<Failure> error = Rxn<Failure>();

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    await _fetch();
    loading.value = false;
  }

  /// Pull to refresh. Keeps whatever is on screen while it runs, so the list
  /// does not blank out under the user's thumb.
  ///
  /// GetxController declares a `refresh()` of its own for rebuilding
  /// GetBuilder consumers; this overrides it because the screen uses Obx and
  /// the only thing worth refreshing here is the data.
  @override
  Future<void> refresh() async {
    refreshing.value = true;
    await _fetch();
    refreshing.value = false;
  }

  Future<void> _fetch() async {
    try {
      // Concurrent: they are independent, and doing them in sequence would
      // double the time the user waits on a slow connection.
      final results = await Future.wait([
        _jobs.awaitingApproval(limit: 10),
        _jobs.list(limit: 10),
      ]);

      // Replies are fetched separately and tolerated failing: an older backend
      // without the events endpoints should still render a working feed.
      try {
        replies.assignAll(await _events.pending(limit: 20));
      } on Failure {
        replies.clear();
      }

      queue.assignAll((results[0]).jobs);
      // The queue is already shown above; repeating those cards in "latest"
      // would make the feed look like it is stuttering.
      final queueIds = queue.map((j) => j.id).toSet();
      recent.assignAll(results[1].jobs.where((j) => !queueIds.contains(j.id)));
      error.value = null;
    } on Failure catch (f) {
      // Only surface the failure state if there is nothing to show. With
      // cached data on screen, a toast is enough — blanking the feed because a
      // refresh failed loses more than it explains.
      if (queue.isEmpty && recent.isEmpty) {
        error.value = f;
      } else {
        AppToast.failure(f);
      }
    }
  }

  /// Drops a job from both lists after it has been acted on, so the card
  /// disappears immediately rather than after a round trip.
  void removeLocally(String id) {
    queue.removeWhere((j) => j.id == id);
    recent.removeWhere((j) => j.id == id);
  }
}
