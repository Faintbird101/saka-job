import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';
import 'package:toastification/toastification.dart';

import '../../core/error/failures.dart';
import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';
import '../../l10n/app_localizations.dart';

/// Every toast in the app.
///
/// Centralised so failures are phrased consistently and, more importantly, so
/// a [Failure] can be handed straight to [failure] — the mapping from error
/// type to human wording lives here rather than being re-decided at each call
/// site.
abstract final class AppToast {
  static void success(String message) => _show(
        message,
        icon: MdiIcons.checkCircleOutline,
        color: AppColors.success,
      );

  static void info(String message) => _show(
        message,
        icon: MdiIcons.informationOutline,
        color: AppColors.primary,
      );

  static void warn(String message) => _show(
        message,
        icon: MdiIcons.alertOutline,
        color: AppColors.warn,
      );

  static void error(String message, {String? detail}) => _show(
        message,
        detail: detail,
        icon: MdiIcons.alertCircleOutline,
        color: AppColors.danger,
      );

  /// Renders a [Failure] in the user's language.
  ///
  /// Validation and conflict failures carry the server's own wording, which is
  /// written to be read — "profile.master_cv is empty", "cannot move a job
  /// from New to Applied". Showing that verbatim is more useful than replacing
  /// it with a generic sentence. Everything else maps to a localised string,
  /// because "HTTP 500" is not a message for a person.
  static void failure(Failure f) {
    final l = _l10n;
    final message = switch (f) {
      NetworkFailure() => l?.couldNotReach ?? 'Could not reach the agent',
      UnauthorizedFailure() => l?.signIn ?? 'Please sign in again',
      ValidationFailure() || ConflictFailure() => f.message,
      _ => l?.somethingWentWrong ?? 'Something went wrong',
    };
    error(message, detail: f.technical);
  }

  static L10n? get _l10n {
    final context = Get.context;
    if (context == null) return null;
    return L10n.of(context);
  }

  static void _show(
    String message, {
    required IconData icon,
    required Color color,
    String? detail,
  }) {
    toastification.show(
      type: ToastificationType.custom(message, color, icon),
      style: ToastificationStyle.flatColored,
      alignment: Alignment.topCenter,
      autoCloseDuration: const Duration(seconds: 4),
      // Dismissible on tap and on drag: a toast that blocks the thing you were
      // reaching for is worse than no toast.
      dragToClose: true,
      showProgressBar: false,
      borderRadius: AppRadii.innerShape,
      primaryColor: color,
      icon: Icon(icon, color: color),
      title: Text(
        message,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(fontWeight: FontWeight.w700),
      ),
      description: detail == null
          ? null
          // The technical line is deliberately small and quiet: it is there
          // for when you are reporting a problem, and ignorable otherwise.
          : Text(
              detail,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 11.5, height: 1.3),
            ),
    );
  }
}
