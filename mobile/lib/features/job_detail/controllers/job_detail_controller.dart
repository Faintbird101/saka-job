import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/job.dart';
import '../../../data/services/jobs_service.dart';
import '../../../shared/widgets/app_toast.dart';

/// Screen 06 — one job, with the score broken open.
class JobDetailController extends GetxController {
  JobsService get _jobs => Get.find<JobsService>();

  /// The list already has most of a job, so it is passed in and shown
  /// immediately while the full record — description text, prompt audit —
  /// loads behind it. Opening a detail screen should never start with a blank
  /// page when the data for the header is already in hand.
  final Rxn<Job> job = Rxn<Job>();
  final String jobId;

  JobDetailController({required this.jobId, Job? initial}) {
    job.value = initial;
  }

  final loading = false.obs;
  final acting = false.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// The score threshold, so the header can say "above your 75 threshold"
  /// rather than leaving the number without a reference point.
  final threshold = 0.obs;

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    // Only a full-screen load when there is nothing to show yet.
    loading.value = job.value == null;
    error.value = null;
    try {
      job.value = await _jobs.byId(jobId);
    } on Failure catch (f) {
      if (job.value == null) {
        error.value = f;
      } else {
        AppToast.failure(f);
      }
    } finally {
      loading.value = false;
    }
  }

  Future<bool> approve() => _decide(approved: true);
  Future<bool> pass() => _decide(approved: false);

  Future<bool> _decide({required bool approved}) async {
    final current = job.value;
    if (current == null || acting.value) return false;

    acting.value = true;
    try {
      final updated = approved
          ? await _jobs.approve(current.id)
          : await _jobs.reject(current.id);
      job.value = updated;
      return true;
    } on Failure catch (f) {
      // Conflict wording from the server names the legal moves, which is more
      // useful than a generic failure — the job has usually moved on.
      AppToast.failure(f);
      return false;
    } finally {
      acting.value = false;
    }
  }
}
