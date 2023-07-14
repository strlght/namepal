FROM ubuntu

COPY watcher /app/watcher

ENTRYPOINT ["/app/watcher"]