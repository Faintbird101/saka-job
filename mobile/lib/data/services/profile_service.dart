import 'package:get/get.dart';

import '../../core/network/api_client.dart';
import '../models/profile.dart';

/// The profile, and the CV upload that feeds it.
class ProfileService extends GetxService {
  ApiClient get _api => Get.find<ApiClient>();

  Future<Profile> get() => _api.get<Profile>(
        '/profile',
        parse: (data) => Profile.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  /// Partial update. The backend treats an absent field as "leave alone", so
  /// only what changed is sent — two screens editing different halves of the
  /// profile can never clobber each other.
  Future<Profile> update(Map<String, dynamic> patch) => _api.patch<Profile>(
        '/profile',
        body: patch,
        parse: (data) => Profile.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  /// Extracts text from a PDF, DOCX or TXT.
  ///
  /// Returns the text WITHOUT saving, deliberately: PDF extraction can mangle a
  /// heavily designed CV, and silently overwriting the profile with soup would
  /// break scoring in a way that is hard to notice. The user reviews, then
  /// saves.
  Future<({String text, int chars, String filename})> extractCv(String filePath) =>
      _api.upload<({String text, int chars, String filename})>(
        '/profile/cv',
        field: 'file',
        filePath: filePath,
        parse: (data) {
          final map = Map<String, dynamic>.from(data as Map);
          return (
            text: (map['text'] ?? '') as String,
            chars: (map['chars'] as num?)?.toInt() ?? 0,
            filename: (map['filename'] ?? '') as String,
          );
        },
      );

  Future<Profile> saveCv(String text) => update({'master_cv': text});
}
