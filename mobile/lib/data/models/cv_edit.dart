/// One change the agent made to the master CV, with its reason.
///
/// The backend verifies every entry against the documents before storing it, so
/// anything reaching the app quotes text that genuinely appears — an edit the
/// candidate cannot find in the CV in front of them would teach them the
/// annotations are decorative.
class CvEdit {
  const CvEdit({
    required this.section,
    required this.before,
    required this.after,
    required this.reason,
  });

  final String section;
  final String before;
  final String after;
  final String reason;

  /// No "before" means text was added; no "after" means it was cut. Both
  /// render differently from a rewrite.
  bool get isAddition => before.trim().isEmpty;
  bool get isDeletion => after.trim().isEmpty;

  factory CvEdit.fromJson(Map<String, dynamic> json) => CvEdit(
        section: (json['section'] ?? '') as String,
        before: (json['before'] ?? '') as String,
        after: (json['after'] ?? '') as String,
        reason: (json['reason'] ?? '') as String,
      );
}
