# Distroless-style image: scratch + the static Go binary. ~5 MB total.
# goreleaser places the prebuilt binary in the build context for us.
FROM scratch

COPY cron-doctor /usr/local/bin/cron-doctor

# Mount the crontab you want to audit, e.g.
#   docker run --rm -v /etc/crontab:/etc/crontab:ro ghcr.io/heytalepazguato/cron-doctor /etc/crontab
ENTRYPOINT ["/usr/local/bin/cron-doctor"]
