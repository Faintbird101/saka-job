import 'package:dio/dio.dart';
import 'package:get/get.dart' hide Response, FormData, MultipartFile;

import '../config/app_config.dart';
import '../error/failures.dart';
import '../storage/secure_store.dart';

/// The only thing in the app that knows an HTTP request exists.
///
/// Services call [get] / [post] / [patch] / [delete] with a path; controllers
/// call services; widgets call controllers. Nothing above this file names an
/// endpoint, carries a token, or handles a status code — which is what makes
/// the base URL, the auth scheme and the error shape changeable in one place.
class ApiClient extends GetxService {
  ApiClient({Dio? dio}) : _dio = dio ?? Dio();

  final Dio _dio;

  /// Called when a request comes back 401, so the app can drop the session and
  /// return to sign-in. Set by AuthService rather than reaching into it from
  /// here, which would make the dependency circular.
  void Function()? onUnauthorized;

  Future<ApiClient> init() async {
    _dio.options = BaseOptions(
      baseUrl: AppConfig.apiBaseUrl,
      connectTimeout: AppConfig.connectTimeout,
      receiveTimeout: AppConfig.receiveTimeout,
      // Never throw on status: failureFrom does the mapping, so every error
      // path is the same shape rather than being split between exceptions and
      // return values.
      validateStatus: (_) => true,
      headers: {'Accept': 'application/json'},
    );

    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          final token = await Get.find<SecureStore>().readToken();
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
        onResponse: (response, handler) {
          final status = response.statusCode ?? 0;
          if (status == 401) {
            // Expired or revoked. Signing out here rather than in every caller
            // means a revoked session cannot leave the app in a half-signed-in
            // state.
            onUnauthorized?.call();
          }
          handler.next(response);
        },
      ),
    );
    return this;
  }

  /// Runs a request and converts anything non-2xx into a [Failure].
  Future<T> _send<T>(
    Future<Response<dynamic>> Function() request,
    T Function(dynamic data) parse,
  ) async {
    try {
      final response = await request();
      final status = response.statusCode ?? 0;
      if (status >= 200 && status < 300) {
        return parse(response.data);
      }
      throw failureFrom(
        DioException(
          requestOptions: response.requestOptions,
          response: response,
          type: DioExceptionType.badResponse,
        ),
      );
    } on Failure {
      rethrow;
    } catch (e) {
      throw failureFrom(e);
    }
  }

  Future<T> get<T>(
    String path, {
    Map<String, dynamic>? query,
    required T Function(dynamic data) parse,
  }) =>
      _send(() => _dio.get(path, queryParameters: query), parse);

  Future<T> post<T>(
    String path, {
    Object? body,
    required T Function(dynamic data) parse,
  }) =>
      _send(() => _dio.post(path, data: body), parse);

  Future<T> patch<T>(
    String path, {
    Object? body,
    required T Function(dynamic data) parse,
  }) =>
      _send(() => _dio.patch(path, data: body), parse);

  Future<T> delete<T>(
    String path, {
    Object? body,
    required T Function(dynamic data) parse,
  }) =>
      _send(() => _dio.delete(path, data: body), parse);

  /// Fetches a document the API serves as `text/markdown` rather than JSON.
  /// The CV and cover letter are prose meant to be read, so they are not
  /// wrapped in a JSON string that the client would have to unescape.
  Future<String> getMarkdown(String path) => _send(
        () => _dio.get<String>(
          path,
          options: Options(responseType: ResponseType.plain),
        ),
        (data) => data?.toString() ?? '',
      );

  /// Multipart upload, for the CV file.
  Future<T> upload<T>(
    String path, {
    required String field,
    required String filePath,
    required T Function(dynamic data) parse,
  }) async {
    final form = FormData.fromMap({
      field: await MultipartFile.fromFile(filePath),
    });
    return _send(() => _dio.post(path, data: form), parse);
  }
}
