import 'package:get/get.dart';

import '../../core/network/api_client.dart';
import '../models/job_event.dart';

/// Inbound-reply events.
class EventsService extends GetxService {
  ApiClient get _api => Get.find<ApiClient>();

  List<JobEvent> _parseList(dynamic data) =>
      (((data as Map)['events'] as List?) ?? const [])
          .map((e) => JobEvent.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();

  /// Classifications awaiting a decision — the queue that exists because a
  /// rejection or an interview invitation is never applied on a guess.
  Future<List<JobEvent>> pending({int limit = 50}) => _api.get<List<JobEvent>>(
        '/events/pending',
        query: {'limit': limit},
        parse: _parseList,
      );

  /// Mail the matcher could not attribute. Worth reviewing: it is the evidence
  /// that the matching rules need work.
  Future<List<JobEvent>> unmatched({int limit = 30}) => _api.get<List<JobEvent>>(
        '/events/unmatched',
        query: {'limit': limit},
        parse: _parseList,
      );

  /// The reply timeline for one job.
  Future<List<JobEvent>> forJob(String jobId, {int limit = 20}) =>
      _api.get<List<JobEvent>>(
        '/jobs/$jobId/events',
        query: {'limit': limit},
        parse: _parseList,
      );

  /// Accepts or dismisses a suggestion.
  ///
  /// `accept` is required by the server rather than defaulted — dismissing a
  /// suggestion by accident is not recoverable.
  Future<JobEvent> confirm(String eventId, {required bool accept}) =>
      _api.post<JobEvent>(
        '/events/$eventId/confirm',
        body: {'accept': accept},
        parse: (data) => JobEvent.fromJson(Map<String, dynamic>.from(data as Map)),
      );
}
