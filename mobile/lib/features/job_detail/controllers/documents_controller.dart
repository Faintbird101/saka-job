import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/services/jobs_service.dart';

/// Screen 07 — the generated CV and cover letter.
///
/// Both are fetched as markdown rather than JSON: they are prose meant to be
/// read, and wrapping them in a JSON string would only mean unescaping every
/// newline before anything could be displayed.
class DocumentsController extends GetxController {
  DocumentsController({required this.jobId, required this.jobTitle});

  final String jobId;
  final String jobTitle;

  JobsService get _jobs => Get.find<JobsService>();

  final cv = ''.obs;
  final coverLetter = ''.obs;
  final loading = true.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// 0 = CV, 1 = cover letter.
  final tab = 0.obs;

  @override
  void onInit() {
    super.onInit();
    load();
  }

  Future<void> load() async {
    loading.value = true;
    error.value = null;
    try {
      // Concurrent: two independent documents, and fetching them in sequence
      // would double the wait for a screen that shows both.
      final results = await Future.wait([
        _jobs.cv(jobId),
        _jobs.coverLetter(jobId),
      ]);
      cv.value = results[0];
      coverLetter.value = results[1];
    } on Failure catch (f) {
      error.value = f;
    } finally {
      loading.value = false;
    }
  }
}
