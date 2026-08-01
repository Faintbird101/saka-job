import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/job.dart';
import '../../../data/services/jobs_service.dart';
import '../../../shared/widgets/app_toast.dart';

/// Screen 04 — every job, filtered by status.
///
/// Paged, because the list grows without bound: the pipeline keeps ingesting
/// and `LowMatch` alone will run to hundreds. Loading all of them to show
/// twenty would waste the connection this app is usually on.
class JobsController extends GetxController {
  JobsService get _jobs => Get.find<JobsService>();

  /// Empty means "all". The chips map onto backend statuses exactly, so no
  /// translation table can drift out of step with the state machine.
  final status = ''.obs;
  final search = ''.obs;

  final jobs = <Job>[].obs;
  final total = 0.obs;
  final loading = true.obs;
  final loadingMore = false.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  static const _pageSize = 20;
  int _offset = 0;
  bool _exhausted = false;

  bool get hasMore => !_exhausted && jobs.length < total.value;

  @override
  void onInit() {
    super.onInit();
    load();

    // Debounced so typing does not fire a request per keystroke.
    debounce<String>(
      search,
      (_) => load(),
      time: const Duration(milliseconds: 400),
    );
  }

  /// Switches the status filter. Resets paging: the old offset belongs to a
  /// different result set and reusing it would skip rows.
  void setStatus(String value) {
    if (status.value == value) return;
    status.value = value;
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    _offset = 0;
    _exhausted = false;

    try {
      final page = await _jobs.list(
        status: status.value,
        search: search.value,
        limit: _pageSize,
        offset: 0,
      );
      jobs.assignAll(page.jobs);
      total.value = page.total;
      _offset = page.jobs.length;
      _exhausted = page.jobs.length < _pageSize;
    } on Failure catch (f) {
      jobs.clear();
      error.value = f;
    } finally {
      loading.value = false;
    }
  }

  /// Appends the next page. Guarded against re-entry so a fast scroll cannot
  /// fire several overlapping requests for the same offset.
  Future<void> loadMore() async {
    if (loadingMore.value || loading.value || !hasMore) return;

    loadingMore.value = true;
    try {
      final page = await _jobs.list(
        status: status.value,
        search: search.value,
        limit: _pageSize,
        offset: _offset,
      );
      // De-duplicate on id: a job whose status changed between pages can
      // otherwise appear twice, and a duplicate key crashes the list.
      final seen = jobs.map((j) => j.id).toSet();
      jobs.addAll(page.jobs.where((j) => !seen.contains(j.id)));
      total.value = page.total;
      _offset += page.jobs.length;
      _exhausted = page.jobs.length < _pageSize;
    } on Failure catch (f) {
      AppToast.failure(f);
    } finally {
      loadingMore.value = false;
    }
  }

  /// Overrides GetxController's own refresh(), which exists for rebuilding
  /// GetBuilder consumers. This screen uses Obx, so reloading the data is the
  /// only thing "refresh" can usefully mean here.
  @override
  Future<void> refresh() => load();
}
