import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/profile.dart';
import '../../../data/services/profile_service.dart';
import '../../../shared/widgets/app_toast.dart';

/// Screens 09 and 10 — the profile, and the agent settings that steer it.
class ProfileController extends GetxController {
  ProfileService get _profiles => Get.find<ProfileService>();

  final Rxn<Profile> profile = Rxn<Profile>();
  final loading = true.obs;
  final saving = false.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// The threshold slider's live value, separate from the saved one so the
  /// dial can move freely and only commit when released.
  final threshold = 75.obs;

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    try {
      final p = await _profiles.get();
      profile.value = p;
      threshold.value = p.minScoreThreshold;
    } on Failure catch (f) {
      error.value = f;
    } finally {
      loading.value = false;
    }
  }

  /// Roughly how many jobs a day this threshold would surface.
  ///
  /// The design's dial "predicts its own consequence". This is an estimate from
  /// the profile's own cap, not a measurement — a threshold you have not run
  /// yet has no history to count.
  int get estimatedPerDay {
    final cap = profile.value?.maxJobsPerRun ?? 10;
    // Two ingest runs a day, and the share that clears the bar falls away
    // steeply as the threshold climbs.
    final share = switch (threshold.value) {
      >= 90 => 0.1,
      >= 80 => 0.25,
      >= 70 => 0.45,
      >= 60 => 0.65,
      _ => 0.85,
    };
    return (cap * 2 * share).round().clamp(0, cap * 2);
  }

  Future<void> saveThreshold() async {
    final current = profile.value;
    if (current == null || current.minScoreThreshold == threshold.value) return;
    await _patch({'min_score_threshold': threshold.value});
  }

  Future<void> setModels({String? scoring, String? generation}) => _patch({
        'scoring_model': ?scoring,
        'generation_model': ?generation,
      });

  Future<void> setNotification(String key, bool value) => _patch({key: value});

  Future<void> _patch(Map<String, dynamic> body) async {
    if (body.isEmpty) return;
    saving.value = true;
    try {
      profile.value = await _profiles.update(body);
      AppToast.success('saved');
    } on Failure catch (f) {
      // Reset the dial to whatever the server still holds, so the UI never
      // shows a value that was not actually stored.
      final p = profile.value;
      if (p != null) threshold.value = p.minScoreThreshold;
      AppToast.failure(f);
    } finally {
      saving.value = false;
    }
  }
}
