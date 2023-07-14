FROM ubuntu

COPY manager /app/manager

ENTRYPOINT ["/app/manager"]