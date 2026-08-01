import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/models/cv_edit.dart';
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

  /// Verified server-side before storage, so every quotation here can be found
  /// in the CV beside it.
  final edits = <CvEdit>[].obs;
  final loading = true.obs;
  final Rxn<Failure> error = Rxn<Failure>();

  /// 0 = changes, 1 = full CV, 2 = cover letter.
  ///
  /// Changes leads, because the deck's whole argument is that the diff is what
  /// makes the tailoring checkable — the finished CV is what you get anywhere.
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

      // The edit list rides on the job row rather than its own endpoint. It is
      // allowed to fail on its own: losing the annotations must not cost the
      // documents.
      try {
        edits.assignAll((await _jobs.byId(jobId)).cvEdits);
      } on Failure {
        edits.clear();
      }
    } on Failure catch (f) {
      error.value = f;
    } finally {
      loading.value = false;
    }
  }
}
