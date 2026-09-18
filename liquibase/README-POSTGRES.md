# Liquibase: Postgres Migrations

This guide shows how to run Liquibase migrations for Postgres using either the Liquibase CLI or Docker.

Changelog location (Postgres):
- `liquibase/my_project_postgres/sql/my-project-changelog.sql`
- Default properties file: `liquibase/my_project_postgres/sql/liquibase.properties`

## Prerequisites
- Postgres accessible (local or remote)
- Either Liquibase CLI installed locally, or Docker available

## Connection details
By default, `liquibase.properties` points to:
- URL: `jdbc:postgresql://localhost:5432/my_project`
- User: `project`
- Password: `MyPassword_123`

Override at runtime with flags or env vars as needed.

---

## Option A: Liquibase CLI (local install)

Validate changelog:
```sh
liquibase \
  --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties \
  --changeLogFile=my-project-changelog.sql \
  validate
```

Preview SQL (dry run, does not execute):
```sh
liquibase \
  --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties \
  --changeLogFile=my-project-changelog.sql \
  updateSQL > liquibase/my_project_postgres/sql/generated-update.sql
```

Apply migrations:
```sh
liquibase \
  --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties \
  --changeLogFile=my-project-changelog.sql \
  update
```

Status and history:
```sh
liquibase --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties --changeLogFile=my-project-changelog.sql status
liquibase --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties --changeLogFile=my-project-changelog.sql history
```

Rollback examples:
```sh
# Rollback last changeset
liquibase --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties --changeLogFile=my-project-changelog.sql rollbackCount 1

# Tag and rollback to tag
liquibase --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties --changeLogFile=my-project-changelog.sql tag v1.0.0
liquibase --defaults-file=liquibase/my_project_postgres/sql/liquibase.properties --changeLogFile=my-project-changelog.sql rollback v1.0.0
```

---

## Option B: Docker (no local install)

Use the official Liquibase image and mount the changelog folder. On macOS, `host.docker.internal` points to your host machine.

Preview SQL (offline):
```sh
docker run --rm \
  -v "$(pwd)/liquibase/my_project_postgres/sql:/liquibase/changelog" \
  liquibase/liquibase:4.31.0 \
  --classpath=/liquibase/changelog \
  --changeLogFile=my-project-changelog.sql \
  --url=offline:postgresql?outputLiquibaseSql=true \
  updateSQL > liquibase/my_project_postgres/sql/generated-update.sql
```

Apply migrations to a live Postgres:
```sh
# Replace URL/username/password if different
docker run --rm \
  -v "$(pwd)/liquibase/my_project_postgres/sql:/liquibase/changelog" \
  liquibase/liquibase:4.31.0 \
  --classpath=/liquibase/changelog \
  --changeLogFile=my-project-changelog.sql \
  --url=jdbc:postgresql://host.docker.internal:5432/my_project \
  --username=project \
  --password=MyPassword_123 \
  update
```

Validate & status with Docker:
```sh
docker run --rm \
  -v "$(pwd)/liquibase/my_project_postgres/sql:/liquibase/changelog" \
  liquibase/liquibase:4.31.0 \
  --classpath=/liquibase/changelog \
  --changeLogFile=my-project-changelog.sql \
  --url=jdbc:postgresql://host.docker.internal:5432/my_project \
  --username=project \
  --password=MyPassword_123 \
  validate

docker run --rm \
  -v "$(pwd)/liquibase/my_project_postgres/sql:/liquibase/changelog" \
  liquibase/liquibase:4.31.0 \
  --classpath=/liquibase/changelog \
  --changeLogFile=my-project-changelog.sql \
  --url=jdbc:postgresql://host.docker.internal:5432/my_project \
  --username=project \
  --password=MyPassword_123 \
  status
```

---

## Notes
- Postgres soft-delete is supported via `deleted_at` column and a partial unique index that applies only to rows where `deleted_at IS NULL`.
- Recent changesets added:
  - external_ref on `video_share_records` and `share_jobs`
  - next_attempt_at on `share_jobs`
  - error_code on `video_share_records` and `share_jobs`
  - deleted_at on `video_share_records` + partial unique index
- Prefer running `validate` and `updateSQL` before `update` in CI/CD.
