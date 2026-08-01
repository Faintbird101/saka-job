import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/job_event.dart';
import '../../../data/services/events_service.dart';
import '../../../shared/widgets/app_toast.dart';

/// Screen 11 — replies that need a decision.
///
/// The backend applies acknowledgements on its own ("they got it" is not a
/// decision) but parks rejections, interview invitations and offers. This is
/// where those get confirmed, and it is the one place in the app where getting
/// it wrong has a real cost: dismissing a genuine interview invitation loses an
/// opportunity silently.
class RepliesController extends GetxController {
  EventsService get _events => Get.find<EventsService>();

  final pending = <JobEvent>[].obs;
  final loading = true.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// Ids currently being confirmed, so a card can show progress without
  /// locking the whole list — several can be decided in quick succession.
  final acting = <String>{}.obs;

  int get count => pending.length;

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    try {
      pending.assignAll(await _events.pending());
    } on Failure catch (f) {
      error.value = f;
    } finally {
      loading.value = false;
    }
  }

  Future<void> decide(JobEvent event, {required bool accept}) async {
    if (acting.contains(event.id)) return;
    acting.add(event.id);

    try {
      await _events.confirm(event.id, accept: accept);
      // Removed rather than re-fetched: the list is short and the card should
      // leave immediately once the decision is recorded.
      pending.removeWhere((e) => e.id == event.id);
    } on Failure catch (f) {
      // Includes the 409 the server returns when the suggestion was already
      // decided elsewhere — worth surfacing verbatim rather than as a generic
      // failure, since it explains why nothing changed.
      AppToast.failure(f);
    } finally {
      acting.remove(event.id);
    }
  }

  @override
  Future<void> refresh() => load();
}
