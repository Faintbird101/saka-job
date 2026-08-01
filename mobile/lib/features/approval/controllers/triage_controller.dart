import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/job.dart';
import '../../../data/services/jobs_service.dart';
import '../../../shared/widgets/app_toast.dart';

/// Screen 05 — the triage deck.
///
/// Optimistic on purpose: the card leaves as soon as you swipe, and the request
/// follows. Waiting for a round trip before animating would make the one
/// interaction the whole product is built around feel sluggish on a phone
/// talking to a laptop over Tailscale.
///
/// The cost is that a failure arrives after the card has gone, so a failed
/// decision puts the job back and says so rather than pretending it worked.
class TriageController extends GetxController {
  JobsService get _jobs => Get.find<JobsService>();

  final deck = <Job>[].obs;
  final loading = true.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// Counts for the "3 of 6" header. [_startingCount] is fixed at load so the
  /// denominator does not shrink as you work through the deck.
  int _startingCount = 0;
  int get startingCount => _startingCount;
  int get remaining => deck.length;
  int get done => _startingCount - deck.length;

  /// Roughly a minute per decision, which is what the design's "6 min" claim
  /// assumes. Shown as an estimate, never as a countdown.
  int get minutesLeft => remaining;

  Job? get current => deck.isEmpty ? null : deck.first;

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    try {
      final page = await _jobs.awaitingApproval(limit: 50);
      deck.assignAll(page.jobs);
      _startingCount = page.jobs.length;
    } on Failure catch (f) {
      error.value = f;
    } finally {
      loading.value = false;
    }
  }

  Future<void> approve(Job job) => _decide(job, approved: true);
  Future<void> pass(Job job) => _decide(job, approved: false);

  Future<void> _decide(Job job, {required bool approved}) async {
    // Remove first: the animation should already be running by the time the
    // request leaves.
    final index = deck.indexWhere((j) => j.id == job.id);
    if (index < 0) return;
    deck.removeAt(index);

    try {
      if (approved) {
        await _jobs.approve(job.id);
      } else {
        await _jobs.reject(job.id);
      }
    } on Failure catch (f) {
      // Put it back where it was rather than at the end, so the deck order the
      // user was working through is preserved.
      deck.insert(index.clamp(0, deck.length), job);
      AppToast.failure(f);
    }
  }
}
