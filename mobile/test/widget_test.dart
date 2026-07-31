import 'package:flutter_test/flutter_test.dart';
import 'package:saka_job/core/theme/app_colors.dart';
import 'package:saka_job/data/models/user.dart';

void main() {
  group('AppUser', () {
    test('reads the shape /auth/login returns', () {
      final u = AppUser.fromJson({
        'id': 'abc',
        'email': 'kinyuaviktar@gmail.com',
        'display_name': 'Victor Kinyua',
      });
      expect(u.id, 'abc');
      expect(u.firstName, 'Victor');
      expect(u.initials, 'VK');
    });

    // /auth/me returns the session, which names the field user_id rather than
    // id. Both shapes have to parse, or the app signs itself out on restart.
    test('reads the shape /auth/me returns', () {
      final u = AppUser.fromJson({
        'user_id': 'xyz',
        'email': 'someone@example.com',
        'expires_at': '2026-08-29T20:36:26Z',
      });
      expect(u.id, 'xyz');
      expect(u.expiresAt, isNotNull);
    });

    test('falls back to the email when there is no display name', () {
      final u = AppUser.fromJson({'id': '1', 'email': 'victor.kinyua@mail.com'});
      expect(u.firstName, 'victor.kinyua');
      expect(u.initials, 'VK');
    });

    test('does not crash on an empty or odd payload', () {
      expect(AppUser.fromJson(const {}).initials, '?');
      expect(AppUser.fromJson(const {'email': 'a@b.co'}).initials, 'AB');
    });
  });

  group('score colour', () {
    // The badge colour and the wording in the backend's own summary must not
    // disagree: a green badge next to "weak match" reads as a bug.
    test('matches the language the backend uses', () {
      expect(AppColors.forScore(92), AppColors.success); // strong
      expect(AppColors.forScore(75), AppColors.primary); // good
      expect(AppColors.forScore(60), AppColors.warn); // partial
      expect(AppColors.forScore(30), AppColors.danger); // weak
    });

    test('covers the boundaries', () {
      expect(AppColors.forScore(85), AppColors.success);
      expect(AppColors.forScore(84), AppColors.primary);
      expect(AppColors.forScore(70), AppColors.primary);
      expect(AppColors.forScore(69), AppColors.warn);
      expect(AppColors.forScore(50), AppColors.warn);
      expect(AppColors.forScore(49), AppColors.danger);
      expect(AppColors.forScore(0), AppColors.danger);
      expect(AppColors.forScore(100), AppColors.success);
    });
  });
}
