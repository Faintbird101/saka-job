/// The signed-in account.
///
/// Named AppUser rather than User because `User` collides with Firebase Auth's
/// type once firebase_core is in the tree, and an ambiguous import is a
/// tedious thing to debug later.
class AppUser {
  const AppUser({
    required this.id,
    required this.email,
    this.displayName,
    this.expiresAt,
  });

  final String id;
  final String email;
  final String? displayName;

  /// When the session expires. The server enforces it; this is only so the
  /// profile screen can say when a re-login is due.
  final DateTime? expiresAt;

  /// Tolerates both shapes the API returns: `/auth/login` nests a `user`
  /// object with `id`, while `/auth/me` returns the session with `user_id`.
  factory AppUser.fromJson(Map<String, dynamic> json) => AppUser(
        id: (json['id'] ?? json['user_id'] ?? '') as String,
        email: (json['email'] ?? '') as String,
        displayName: json['display_name'] as String?,
        expiresAt: json['expires_at'] == null
            ? null
            : DateTime.tryParse(json['expires_at'] as String),
      );

  /// The first name, for "Good morning, Victor".
  String get firstName {
    final name = displayName?.trim();
    if (name == null || name.isEmpty) {
      // Fall back to the local part of the email rather than showing nothing.
      final local = email.split('@').first;
      return local.isEmpty ? '' : local;
    }
    return name.split(RegExp(r'\s+')).first;
  }

  /// Up to two letters for the avatar.
  String get initials {
    final source = (displayName?.trim().isNotEmpty ?? false)
        ? displayName!.trim()
        : email;
    final parts = source.split(RegExp(r'[\s@._-]+')).where((p) => p.isNotEmpty).toList();
    if (parts.isEmpty) return '?';
    if (parts.length == 1) return parts.first.substring(0, 1).toUpperCase();
    return (parts[0].substring(0, 1) + parts[1].substring(0, 1)).toUpperCase();
  }
}
