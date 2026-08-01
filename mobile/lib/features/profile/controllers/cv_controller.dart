import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/services/profile_service.dart';
import '../../../shared/widgets/app_toast.dart';
import 'profile_controller.dart';

/// The master CV editor.
///
/// Upload is deliberately two steps — extract, then save. PDF extraction can
/// mangle a heavily designed CV into unusable text, and silently overwriting
/// the profile with that would break scoring in a way nobody would notice until
/// every job started scoring low. The backend returns the text without saving;
/// this screen shows it and waits.
class CvController extends GetxController {
  ProfileService get _profiles => Get.find<ProfileService>();

  final text = TextEditingController();

  final busy = false.obs;
  final extracting = false.obs;

  /// The filename just extracted, so the screen can say what it read.
  final sourceFile = ''.obs;

  /// Tracks whether the buffer differs from what was loaded, so Save can be
  /// disabled when there is nothing to save.
  final dirty = false.obs;
  String _original = '';

  int get charCount => text.text.length;

  @override
  void onInit() {
    super.onInit();
    _original = Get.find<ProfileController>().profile.value?.masterCv ?? '';
    text.text = _original;
    text.addListener(() => dirty.value = text.text != _original);
  }

  @override
  void onClose() {
    text.dispose();
    super.onClose();
  }

  /// Picks a file and asks the backend to extract its text.
  Future<void> pickFile() async {
    if (extracting.value) return;

    final picked = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      // Exactly what the backend parses. Offering more would mean the picker
      // hands back something the server rejects, which reads as a bug.
      allowedExtensions: const ['pdf', 'docx', 'txt', 'md'],
      withData: false,
    );

    final path = picked?.files.single.path;
    if (path == null) return;

    extracting.value = true;
    try {
      final result = await _profiles.extractCv(path);
      text.text = result.text;
      sourceFile.value = result.filename;
      dirty.value = text.text != _original;

      if (result.chars == 0) {
        AppToast.warn('That file produced no text');
      }
    } on Failure catch (f) {
      // The backend's own wording is specific and worth passing through — it
      // distinguishes a scanned PDF with no text layer from an unsupported
      // format, and those need different fixes.
      AppToast.failure(f);
    } finally {
      extracting.value = false;
    }
  }

  Future<bool> save() async {
    if (busy.value) return false;

    final body = text.text.trim();
    if (body.isEmpty) {
      AppToast.warn('The CV cannot be empty');
      return false;
    }

    busy.value = true;
    try {
      final updated = await _profiles.saveCv(body);
      // Keep the profile screen in step without a second request.
      Get.find<ProfileController>().profile.value = updated;
      _original = body;
      dirty.value = false;
      return true;
    } on Failure catch (f) {
      AppToast.failure(f);
      return false;
    } finally {
      busy.value = false;
    }
  }
}
