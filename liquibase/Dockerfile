# Stage 1: Base image with Liquibase and MySQL plugin
FROM liquibase/liquibase:4.31.0 AS base
RUN lpm add mysql --global

# Stage 2: Create non-root user and copy necessary files
FROM debian:bookworm AS build
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Set the Current Working Directory inside the container
WORKDIR /liquibase

# Copy files from the base stage
COPY --from=base /liquibase /liquibase

# Copy the project files
COPY ./liquibase/liquibase.docker.properties /liquibase/changelog/liquibase.docker.properties
COPY ./liquibase/my-project/sql /liquibase/changelog

# Final stage: Run Liquibase
FROM build AS final
# Change ownership of the copied files to the non-root user
RUN chown -R appuser /liquibase

# Switch to the non-root user
USER appuser
ENTRYPOINT [ "sh", "-c", "sleep 10; liquibase --defaults-file=/liquibase/changelog/liquibase.docker.properties update" ]