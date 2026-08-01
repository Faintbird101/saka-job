import 'package:get/get.dart';

import '../../core/network/api_client.dart';
import '../models/job.dart';

/// Everything the app reads or writes about jobs.
///
/// Paths live here and nowhere else. A controller asks for "the jobs waiting on
/// me" rather than for `/jobs?status=AwaitingApproval`, which is what lets the
/// API change shape without touching a screen.
class JobsService extends GetxService {
  ApiClient get _api => Get.find<ApiClient>();

  /// The statuses the app treats as "still live", for the pipeline view.
  static const liveStatuses = [
    'Applied', 'FollowUpSent', 'Acknowledged', 'Interviewing', 'ManualApply',
  ];

  Future<JobPage> list({
    String? status,
    int? minScore,
    String? search,
    int limit = 20,
    int offset = 0,
  }) =>
      _api.get<JobPage>(
        '/jobs',
        query: {
          if (status != null && status.isNotEmpty) 'status': status,
          'min_score': ?minScore,
          if (search != null && search.trim().isNotEmpty) 'q': search.trim(),
          'limit': limit,
          'offset': offset,
        },
        parse: (data) => JobPage.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  /// The queue — what the pipeline is actually blocked on.
  Future<JobPage> awaitingApproval({int limit = 20, int offset = 0}) =>
      list(status: 'AwaitingApproval', limit: limit, offset: offset);

  Future<Job> byId(String id) => _api.get<Job>(
        '/jobs/$id',
        parse: (data) => Job.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  /// Moves a job through the state machine.
  ///
  /// The server validates the transition, so an illegal move comes back as a
  /// ConflictFailure naming the legal ones rather than being silently allowed.
  Future<Job> setStatus(String id, String status) => _api.patch<Job>(
        '/jobs/$id',
        body: {'status': status},
        parse: (data) => Job.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  Future<Job> approve(String id) => setStatus(id, 'Approved');
  Future<Job> reject(String id) => setStatus(id, 'Rejected');

  /// Re-runs scoring for one job, after the CV or the threshold changed.
  Future<Job> rescore(String id) => _api.post<Job>(
        '/jobs/$id/rescore',
        parse: (data) => Job.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  /// The generated documents. Served as markdown rather than JSON because they
  /// are prose meant to be read.
  Future<String> cv(String id) => _api.getMarkdown('/jobs/$id/cv');
  Future<String> coverLetter(String id) => _api.getMarkdown('/jobs/$id/cover-letter');

  /// Counts per pipeline stage, for the dashboard.
  Future<Map<String, dynamic>> stats() => _api.get<Map<String, dynamic>>(
        '/stats',
        parse: (data) => Map<String, dynamic>.from(data as Map),
      );
}
