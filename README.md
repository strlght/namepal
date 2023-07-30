# Namepal

A tool to automate DNS updates.

## Quick start

Launch manager on a single machine:

```bash
$ docker run --detach \
    namepal/manager:dev
```

Launch agent on every machine:

```bash
$ docker run --detach \
    --volume /var/run/docker.sock:/var/run/docker.sock:ro \
    namepal/agent:dev
```

