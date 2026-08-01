import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../core/error/failures.dart';
import '../../../data/services/auth_service.dart';
import '../../../routes/app_routes.dart';
import '../../../shared/widgets/app_toast.dart';

/// Screen state for sign-in and first-run setup.
///
/// The controller owns validation and the busy flag; the service owns the
/// endpoint. The view owns neither, which is what keeps the widget free of
/// try/catch and status codes.
class AuthController extends GetxController {
  AuthService get _auth => Get.find<AuthService>();

  final emailField = TextEditingController();
  final passwordField = TextEditingController();
  final nameField = TextEditingController();
  final formKey = GlobalKey<FormState>();

  final busy = false.obs;
  final obscurePassword = true.obs;

  /// True when no account exists yet, so the screen offers setup instead of
  /// sign-in. Checked once on open.
  final needsSetup = false.obs;
  final checkingSetup = true.obs;

  /// Set when the reachability check itself failed.
  ///
  /// The screen shows the failure state rather than the form. Showing a working
  /// sign-in form when the server is unreachable makes a network problem look
  /// identical to a working app — you type a password, wait, and get a second
  /// failure that explains nothing new.
  final Rxn<Failure> startupError = Rxn<Failure>();

  @override
  void onInit() {
    super.onInit();
    _checkSetup();
  }

  @override
  void onClose() {
    emailField.dispose();
    passwordField.dispose();
    nameField.dispose();
    super.onClose();
  }

  /// Re-runs the reachability check, for the retry button.
  Future<void> retry() => _checkSetup();

  Future<void> _checkSetup() async {
    checkingSetup.value = true;
    startupError.value = null;
    try {
      needsSetup.value = await _auth.needsSetup();
    } on Failure catch (f) {
      // Surfaced as a full failure state, not a toast: until we know whether an
      // account exists, we cannot honestly show either the sign-in or the setup
      // form, and guessing produces a screen that fails confusingly on submit.
      startupError.value = f;
    } finally {
      checkingSetup.value = false;
    }
  }

  String? validateEmail(String? value) {
    final v = value?.trim() ?? '';
    if (v.isEmpty) return 'empty';
    // Loose on purpose — anything stricter rejects valid addresses, and the
    // server validates properly anyway.
    if (!v.contains('@') || !v.split('@').last.contains('.')) return 'invalid';
    return null;
  }

  String? validatePassword(String? value) {
    final v = value ?? '';
    if (v.isEmpty) return 'empty';
    // Only enforced during setup; signing in must accept whatever was chosen,
    // including a password predating any rule change.
    if (needsSetup.value && v.length < 10) return 'short';
    return null;
  }

  Future<void> submit() async {
    if (!(formKey.currentState?.validate() ?? false)) return;
    if (busy.value) return;

    busy.value = true;
    try {
      if (needsSetup.value) {
        await _auth.createFirstAccount(
          email: emailField.text,
          password: passwordField.text,
          displayName: nameField.text,
        );
      } else {
        await _auth.signIn(
          email: emailField.text,
          password: passwordField.text,
        );
      }
      Get.offAllNamed(Routes.shell);
    } on Failure catch (f) {
      AppToast.failure(f);
    } finally {
      busy.value = false;
    }
  }
}
