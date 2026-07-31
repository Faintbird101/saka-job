import 'package:flutter/material.dart';

/// The palette from the design system, verbatim.
///
/// Every value here is a token from the deck rather than a choice made in a
/// widget, so a colour change happens in one place instead of forty.
abstract final class AppColors {
  // ---- brand ----
  /// The gradient runs #62C4EC -> #2189C9. Both ends are named because the
  /// lighter one is also used flat, for tints and highlights.
  static const primaryLight = Color(0xFF62C4EC);
  static const primary = Color(0xFF2189C9);
  static const primaryTint = Color(0xFFE7F4FB);

  // ---- surfaces and text ----
  static const ink = Color(0xFF12293B);
  static const bg = Color(0xFFEEF5FA);
  static const muted = Color(0xFF7B93A7);
  static const surface = Color(0xFFFFFFFF);

  // ---- semantic ----
  static const success = Color(0xFF2FB98A);
  static const warn = Color(0xFFF2A63B);
  static const danger = Color(0xFFE5484D);

  // ---- dark mode ----
  //
  // The deck notes the ink palette already exists and dark mode "needs the
  // glass surfaces inverted". These are derived from ink rather than being a
  // second, unrelated palette, so the two themes stay recognisably the same
  // product.
  static const darkBg = Color(0xFF0B1926);
  static const darkSurface = Color(0xFF12293B);
  static const darkSurfaceRaised = Color(0xFF1B3549);
  static const darkInk = Color(0xFFE8F0F6);
  static const darkMuted = Color(0xFF8FA6B8);

  /// The primary gradient, used on the approve button and score cards.
  static const primaryGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [primaryLight, primary],
  );

  /// Score colours. Thresholds match the language the backend already uses in
  /// its summaries — "strong", "good", "partial", "weak" — so a green badge in
  /// the app and the word "strong" in the summary never disagree.
  static Color forScore(int score) {
    if (score >= 85) return success;
    if (score >= 70) return primary;
    if (score >= 50) return warn;
    return danger;
  }

  /// The soft fill behind a score badge.
  static Color forScoreTint(int score, {required bool dark}) {
    final base = forScore(score);
    return dark ? base.withValues(alpha: 0.18) : base.withValues(alpha: 0.12);
  }
}
