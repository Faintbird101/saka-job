import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/auth_controller.dart';

/// Screen 01 — sign in, doubling as first-run setup.
///
/// One screen rather than two: the only difference is a name field and the
/// button's wording, and a separate signup route would be dead code the moment
/// the single account exists.
class SignInScreen extends GetView<AuthController> {
  const SignInScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(
              horizontal: AppSpacing.xxl,
              vertical: AppSpacing.section,
            ),
            child: Obx(() {
              // Until the reachability check finishes we do not know whether to
              // offer sign-in or setup, so the form is not shown at all.
              if (controller.checkingSetup.value) {
                return const _CheckingSkeleton();
              }

              final startupError = controller.startupError.value;
              if (startupError != null) {
                return FailureView(
                  failure: startupError,
                  onRetry: controller.retry,
                );
              }

              final isSetup = controller.needsSetup.value;

              return Form(
                key: controller.formKey,
                // Without this, validators only re-run when validate() is
                // called, so an error raised by tapping Sign in stays on screen
                // while you correct the field — a valid address sitting under
                // "that does not look like an email address". onUserInteraction
                // re-checks as you type, so the message clears when it stops
                // being true.
                autovalidateMode: AutovalidateMode.onUserInteraction,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const _Logo(),
                    const SizedBox(height: AppSpacing.xxl),

                    Text(
                      isSetup ? l.createAccount : l.welcomeTitle,
                      textAlign: TextAlign.center,
                      style: text.displayMedium,
                    ),
                    const SizedBox(height: AppSpacing.sm),
                    Text(
                      isSetup ? l.tagline : l.welcomeBody,
                      textAlign: TextAlign.center,
                      style: text.bodyMedium,
                    ),
                    const SizedBox(height: AppSpacing.section),

                    // Only during setup — there is nothing to name when
                    // signing in to an account that already has one.
                    if (isSetup) ...[
                      _Field(
                        controller: controller.nameField,
                        label: l.displayName,
                        icon: MdiIcons.accountOutline,
                        textInputAction: TextInputAction.next,
                      ),
                      const SizedBox(height: AppSpacing.lg),
                    ],

                    _Field(
                      controller: controller.emailField,
                      label: l.email,
                      icon: MdiIcons.emailOutline,
                      keyboardType: TextInputType.emailAddress,
                      textInputAction: TextInputAction.next,
                      autofillHints: const [AutofillHints.username],
                      validator: (v) => switch (controller.validateEmail(v)) {
                        'empty' => l.emailRequired,
                        'invalid' => l.emailInvalid,
                        _ => null,
                      },
                    ),
                    const SizedBox(height: AppSpacing.lg),

                    Obx(
                      () => _Field(
                        controller: controller.passwordField,
                        label: l.password,
                        icon: MdiIcons.lockOutline,
                        obscure: controller.obscurePassword.value,
                        textInputAction: TextInputAction.done,
                        autofillHints: const [AutofillHints.password],
                        onSubmitted: (_) => controller.submit(),
                        suffix: TextButton(
                          onPressed: controller.obscurePassword.toggle,
                          child: Text(
                            controller.obscurePassword.value ? l.show : l.hide,
                          ),
                        ),
                        validator: (v) => switch (controller.validatePassword(v)) {
                          'empty' => l.passwordRequired,
                          'short' => l.passwordTooShort(10),
                          _ => null,
                        },
                      ),
                    ),
                    const SizedBox(height: AppSpacing.xxl),

                    Obx(
                      () => FilledButton(
                        onPressed: controller.busy.value ? null : controller.submit,
                        child: controller.busy.value
                            // Inline rather than a full-screen overlay: the
                            // form stays visible so a typo is still fixable.
                            ? const SizedBox(
                                height: 22,
                                width: 22,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2.4,
                                  color: Colors.white,
                                ),
                              )
                            : Text(isSetup ? l.createAccount : l.signIn),
                      ),
                    ),
                  ],
                ),
              );
            }),
          ),
        ),
      ),
    );
  }
}

/// Shown while the app works out whether an account exists. Shaped like the
/// form that is coming, so nothing jumps when it arrives.
class _CheckingSkeleton extends StatelessWidget {
  const _CheckingSkeleton();

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: const [
        _Logo(),
        SizedBox(height: AppSpacing.xxl),
        Skeleton(width: 220, height: 26, radius: 10),
        SizedBox(height: AppSpacing.md),
        Skeleton(width: 280, height: 14, radius: 8),
        SizedBox(height: AppSpacing.section),
        Skeleton(width: double.infinity, height: 56, radius: AppRadii.inner),
        SizedBox(height: AppSpacing.lg),
        Skeleton(width: double.infinity, height: 56, radius: AppRadii.inner),
        SizedBox(height: AppSpacing.xxl),
        Skeleton(width: double.infinity, height: 52, radius: AppRadii.pill),
      ],
    );
  }
}

class _Logo extends StatelessWidget {
  const _Logo();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Container(
        width: 72,
        height: 72,
        decoration: BoxDecoration(
          gradient: AppColors.primaryGradient,
          borderRadius: BorderRadius.circular(AppRadii.inner),
          boxShadow: AppShadows.gradient,
        ),
        alignment: Alignment.center,
        child: const Text(
          'S',
          style: TextStyle(
            color: Colors.white,
            fontSize: 36,
            fontWeight: FontWeight.w800,
          ),
        ),
      ),
    );
  }
}

/// A labelled field. Extracted because the form has four of them and the
/// decoration is identical each time.
class _Field extends StatelessWidget {
  const _Field({
    required this.controller,
    required this.label,
    required this.icon,
    this.obscure = false,
    this.keyboardType,
    this.textInputAction,
    this.validator,
    this.suffix,
    this.onSubmitted,
    this.autofillHints,
  });

  final TextEditingController controller;
  final String label;
  final IconData icon;
  final bool obscure;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final String? Function(String?)? validator;
  final Widget? suffix;
  final void Function(String)? onSubmitted;
  final Iterable<String>? autofillHints;

  @override
  Widget build(BuildContext context) {
    return TextFormField(
      controller: controller,
      obscureText: obscure,
      keyboardType: keyboardType,
      textInputAction: textInputAction,
      validator: validator,
      onFieldSubmitted: onSubmitted,
      autofillHints: autofillHints,
      autocorrect: false,
      enableSuggestions: !obscure,
      decoration: InputDecoration(
        labelText: label,
        prefixIcon: Icon(icon, size: 20),
        suffixIcon: suffix,
      ),
    );
  }
}
