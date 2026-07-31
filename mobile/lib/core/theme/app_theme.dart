import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'app_colors.dart';

/// Geometry tokens from the design system.
///
/// The deck specifies radii of 26 for cards, 20 for inner surfaces and 99 for
/// pills. Naming them stops "roughly rounded" drifting across screens.
abstract final class AppRadii {
  static const card = 26.0;
  static const inner = 20.0;
  static const pill = 99.0;

  static const cardShape = BorderRadius.all(Radius.circular(card));
  static const innerShape = BorderRadius.all(Radius.circular(inner));
  static const pillShape = BorderRadius.all(Radius.circular(pill));
}

/// Spacing on a 4pt grid.
abstract final class AppSpacing {
  static const xs = 4.0;
  static const sm = 8.0;
  static const md = 12.0;
  static const lg = 16.0;
  static const xl = 20.0;
  static const xxl = 24.0;
  static const section = 32.0;
}

/// Elevation, from the deck: a wide soft shadow for cards, and a stronger
/// tinted one under gradient surfaces so they read as lifted rather than
/// painted on.
abstract final class AppShadows {
  static const card = [
    BoxShadow(
      color: Color(0x0F145078), // rgba(20,80,120,.06)
      blurRadius: 22,
      offset: Offset(0, 8),
    ),
  ];

  static const gradient = [
    BoxShadow(
      color: Color(0x422189C9), // rgba(33,137,201,.26)
      blurRadius: 32,
      offset: Offset(0, 14),
    ),
  ];

  static const cardDark = [
    BoxShadow(color: Color(0x66000000), blurRadius: 22, offset: Offset(0, 8)),
  ];
}

/// Builds the light and dark themes.
///
/// Type is Plus Jakarta Sans throughout, per the deck. The sizes there are
/// given at 2x for the mockups, so they are halved here: Display 48 -> 24,
/// Section 32 -> 16, Card title 26 -> 13... which would be unreadably small on
/// a real device. Instead the *ratios* are preserved against a 16pt body,
/// which is what the mockups actually look like in the hand.
abstract final class AppTheme {
  static ThemeData light() => _build(Brightness.light);
  static ThemeData dark() => _build(Brightness.dark);

  static ThemeData _build(Brightness brightness) {
    final dark = brightness == Brightness.dark;

    final scheme = ColorScheme.fromSeed(
      seedColor: AppColors.primary,
      brightness: brightness,
    ).copyWith(
      primary: AppColors.primary,
      surface: dark ? AppColors.darkSurface : AppColors.surface,
      error: AppColors.danger,
    );

    final onSurface = dark ? AppColors.darkInk : AppColors.ink;
    final muted = dark ? AppColors.darkMuted : AppColors.muted;

    final text = TextTheme(
      // Display — the big number on a score card, and the welcome heading.
      displayLarge: GoogleFonts.plusJakartaSans(
        fontSize: 34, fontWeight: FontWeight.w800, color: onSurface, height: 1.1),
      displayMedium: GoogleFonts.plusJakartaSans(
        fontSize: 28, fontWeight: FontWeight.w800, color: onSurface, height: 1.15),
      // Section headings.
      headlineMedium: GoogleFonts.plusJakartaSans(
        fontSize: 22, fontWeight: FontWeight.w800, color: onSurface),
      headlineSmall: GoogleFonts.plusJakartaSans(
        fontSize: 19, fontWeight: FontWeight.w700, color: onSurface),
      // Card titles.
      titleLarge: GoogleFonts.plusJakartaSans(
        fontSize: 17, fontWeight: FontWeight.w700, color: onSurface),
      titleMedium: GoogleFonts.plusJakartaSans(
        fontSize: 15, fontWeight: FontWeight.w700, color: onSurface),
      // Body.
      bodyLarge: GoogleFonts.plusJakartaSans(
        fontSize: 15, fontWeight: FontWeight.w500, color: onSurface, height: 1.45),
      bodyMedium: GoogleFonts.plusJakartaSans(
        fontSize: 14, fontWeight: FontWeight.w500, color: muted, height: 1.45),
      bodySmall: GoogleFonts.plusJakartaSans(
        fontSize: 13, fontWeight: FontWeight.w500, color: muted),
      // Caption — the uppercase labels above sections.
      labelLarge: GoogleFonts.plusJakartaSans(
        fontSize: 14, fontWeight: FontWeight.w700, letterSpacing: 0.2),
      labelMedium: GoogleFonts.plusJakartaSans(
        fontSize: 12, fontWeight: FontWeight.w600, color: muted, letterSpacing: 0.4),
      labelSmall: GoogleFonts.plusJakartaSans(
        fontSize: 11, fontWeight: FontWeight.w600, color: muted, letterSpacing: 0.6),
    );

    return ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: scheme,
      scaffoldBackgroundColor: dark ? AppColors.darkBg : AppColors.bg,
      textTheme: text,
      splashFactory: InkSparkle.splashFactory,

      appBarTheme: AppBarTheme(
        backgroundColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        centerTitle: false,
        titleTextStyle: text.headlineSmall,
        iconTheme: IconThemeData(color: onSurface),
      ),

      cardTheme: CardThemeData(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: const RoundedRectangleBorder(borderRadius: AppRadii.cardShape),
      ),

      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: AppColors.primary,
          foregroundColor: Colors.white,
          minimumSize: const Size.fromHeight(52),
          shape: const RoundedRectangleBorder(borderRadius: AppRadii.pillShape),
          textStyle: text.titleMedium?.copyWith(color: Colors.white),
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: AppColors.primary,
          minimumSize: const Size.fromHeight(52),
          side: BorderSide(
            color: dark ? AppColors.darkSurfaceRaised : AppColors.primaryTint,
          ),
          shape: const RoundedRectangleBorder(borderRadius: AppRadii.pillShape),
          textStyle: text.titleMedium,
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: AppColors.primary,
          textStyle: text.titleMedium,
        ),
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: dark ? AppColors.darkBg : AppColors.bg,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.lg, vertical: AppSpacing.lg),
        border: OutlineInputBorder(
          borderRadius: AppRadii.innerShape,
          borderSide: BorderSide(
            color: dark ? AppColors.darkSurfaceRaised : const Color(0xFFDCE7F0)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: AppRadii.innerShape,
          borderSide: BorderSide(
            color: dark ? AppColors.darkSurfaceRaised : const Color(0xFFDCE7F0)),
        ),
        focusedBorder: const OutlineInputBorder(
          borderRadius: AppRadii.innerShape,
          borderSide: BorderSide(color: AppColors.primary, width: 1.6),
        ),
        errorBorder: const OutlineInputBorder(
          borderRadius: AppRadii.innerShape,
          borderSide: BorderSide(color: AppColors.danger),
        ),
        focusedErrorBorder: const OutlineInputBorder(
          borderRadius: AppRadii.innerShape,
          borderSide: BorderSide(color: AppColors.danger, width: 1.6),
        ),
        hintStyle: text.bodyMedium,
        labelStyle: text.bodyMedium,
      ),

      chipTheme: ChipThemeData(
        backgroundColor: dark ? AppColors.darkSurfaceRaised : AppColors.primaryTint,
        labelStyle: text.labelMedium?.copyWith(
          color: dark ? AppColors.darkInk : AppColors.ink),
        side: BorderSide.none,
        shape: const RoundedRectangleBorder(borderRadius: AppRadii.pillShape),
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.md, vertical: AppSpacing.sm),
      ),

      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: dark ? AppColors.darkSurface : AppColors.surface,
        indicatorColor: AppColors.primaryTint,
        elevation: 0,
        height: 68,
        labelTextStyle: WidgetStatePropertyAll(text.labelSmall),
      ),

      dividerTheme: DividerThemeData(
        color: dark ? AppColors.darkSurfaceRaised : const Color(0xFFE2ECF3),
        thickness: 1,
        space: 1,
      ),

      progressIndicatorTheme: const ProgressIndicatorThemeData(
        color: AppColors.primary,
        linearTrackColor: AppColors.primaryTint,
      ),

      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        backgroundColor: dark ? AppColors.darkSurfaceRaised : AppColors.ink,
        contentTextStyle: text.bodyLarge?.copyWith(color: Colors.white),
        shape: const RoundedRectangleBorder(borderRadius: AppRadii.innerShape),
      ),
    );
  }
}
