/// Build-time configuration.
///
/// The base URL is a `--dart-define` rather than a constant in source, so the
/// same build can point at the Tailscale host, a LAN address while developing,
/// or a real deployment later, without editing code:
///
///   flutter run --dart-define=API_BASE_URL=https://ronnie.tail35a8a7.ts.net
abstract final class AppConfig {
  /// Defaults to the Tailscale hostname, which is the only address that works
  /// from a phone off the home network AND presents a certificate the platform
  /// HTTP client already trusts. The local `.home` hostnames use Caddy's
  /// internal CA, which Flutter rejects outright.
  static const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'https://ronnie.tail35a8a7.ts.net',
  );

  /// Connect/receive timeouts. Generous on receive because some endpoints are
  /// genuinely slow — the backend scores one job per model call — but a phone
  /// should never sit on a spinner for minutes, so the app polls rather than
  /// holding a request open for those.
  static const connectTimeout = Duration(seconds: 15);
  static const receiveTimeout = Duration(seconds: 30);

  /// How many jobs a list page pulls.
  static const pageSize = 20;

  static bool get isConfigured => apiBaseUrl.isNotEmpty;
}
