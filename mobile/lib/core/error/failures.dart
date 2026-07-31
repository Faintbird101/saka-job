import 'package:dio/dio.dart';

/// A failure the UI can actually render.
///
/// Every network error is converted into one of these at the service boundary,
/// so no widget ever sees a DioException and no screen has to guess what a
/// status code means. The design's failure state renders [message] and shows
/// [technical] in small type underneath.
sealed class Failure implements Exception {
  const Failure(this.message, {this.technical});

  /// What to show the user. Plain language, never a status code.
  final String message;

  /// The detail for the small grey line — "NetworkFailure · timeout after 15s".
  /// Useful when reporting a problem, ignorable otherwise.
  final String? technical;

  @override
  String toString() => technical == null ? message : '$message ($technical)';
}

/// No usable connection, or the server could not be reached at all.
///
/// Distinct from every other failure because it is the one the user can often
/// fix themselves, and the one where cached data is still worth showing.
final class NetworkFailure extends Failure {
  const NetworkFailure([String? technical])
      : super('network', technical: technical);
}

/// Credentials are missing, expired, or were revoked. The app signs out.
final class UnauthorizedFailure extends Failure {
  const UnauthorizedFailure([String? technical])
      : super('unauthorized', technical: technical);
}

/// The caller is authenticated but not allowed. In this app that means a
/// session token reached an n8n-only endpoint, which is a bug rather than
/// something the user can act on.
final class ForbiddenFailure extends Failure {
  const ForbiddenFailure([String? technical])
      : super('forbidden', technical: technical);
}

/// The thing being asked for does not exist.
final class NotFoundFailure extends Failure {
  const NotFoundFailure([String? technical])
      : super('not_found', technical: technical);
}

/// The server rejected the request and explained why. [message] carries the
/// backend's own wording, which is written to be shown — "score must be
/// between 0 and 100", "profile.master_cv is empty".
final class ValidationFailure extends Failure {
  const ValidationFailure(super.message, {super.technical});
}

/// A state-machine conflict: the job moved on, or the suggestion was already
/// decided. Worth surfacing verbatim because the backend names the legal moves.
final class ConflictFailure extends Failure {
  const ConflictFailure(super.message, {super.technical});
}

/// Anything else, including a 500.
final class ServerFailure extends Failure {
  const ServerFailure([String? technical])
      : super('server', technical: technical);
}

/// Converts a Dio error into a [Failure].
///
/// Centralised so the mapping exists once. The backend uses one error envelope
/// — `{"error":{"message":"...","request_id":"..."}}` — so the message can be
/// lifted out reliably, and the request id is carried into [Failure.technical]
/// where it is the thing that makes a report actionable.
Failure failureFrom(Object error) {
  if (error is Failure) return error;
  if (error is! DioException) return ServerFailure(error.toString());

  switch (error.type) {
    case DioExceptionType.connectionTimeout:
    case DioExceptionType.sendTimeout:
    case DioExceptionType.receiveTimeout:
    case DioExceptionType.transformTimeout:
      return NetworkFailure('timeout after ${error.requestOptions.connectTimeout?.inSeconds}s');
    case DioExceptionType.connectionError:
    case DioExceptionType.unknown:
      return const NetworkFailure('could not reach the server');
    case DioExceptionType.badCertificate:
      // Worth its own wording: this almost always means the app is pointed at
      // a host serving Caddy's internal CA rather than the Tailscale name.
      return const NetworkFailure('certificate not trusted');
    case DioExceptionType.cancel:
      return const NetworkFailure('request cancelled');
    case DioExceptionType.badResponse:
      break;
  }

  final response = error.response;
  final status = response?.statusCode ?? 0;
  final body = response?.data;

  String? serverMessage;
  String? requestId;
  if (body is Map) {
    final err = body['error'];
    if (err is Map) {
      serverMessage = err['message'] as String?;
      requestId = err['request_id'] as String?;
    }
  }
  final technical = [
    'HTTP $status',
    if (requestId != null) 'req $requestId',
  ].join(' · ');

  return switch (status) {
    401 => UnauthorizedFailure(technical),
    403 => ForbiddenFailure(technical),
    404 => NotFoundFailure(technical),
    409 => ConflictFailure(serverMessage ?? 'conflict', technical: technical),
    >= 400 && < 500 => ValidationFailure(
        serverMessage ?? 'invalid request', technical: technical),
    _ => ServerFailure(technical),
  };
}
