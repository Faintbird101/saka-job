# Mobile (Flutter)

Feature-first architecture. Each feature owns its UI + state; shared plumbing
lives in `core/` (network, theme, config) and `data/` (models, repositories).

- `lib/main.dart`            app entry, provider/router setup
- `lib/core/network/`        Dio/http client, adds auth token, base URL
- `lib/core/theme/`          colors, typography, ThemeData
- `lib/core/config/`         env/base-url config
- `lib/core/error/`          Failure types, error mapping
- `lib/data/models/`         Job, Profile (mirror backend JSON)
- `lib/data/datasources/`    remote API calls
- `lib/data/repositories/`   expose data to features
- `lib/features/jobs/`       job list (filter by status, sort by score)
- `lib/features/job_detail/` full job + AI score + CV preview
- `lib/features/approval/`   approve / reject / request-edit actions
- `lib/features/profile/`    master CV, keywords, threshold
- `lib/features/dashboard/`  counts per pipeline stage
- `lib/shared/widgets/`      reusable widgets (status chip, score badge)

State management: pick ONE (Riverpod recommended) and keep it consistent.
