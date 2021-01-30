This thing reads data from the Awair local API and serves it up in the format that Prometheus understands.

There's a [multi-arch image on Docker hub](https://hub.docker.com/r/bjackman/awair-local-prometheus)

Building 'im for arm64
======================

AFAICT Docker Hub's autobuild doesn't support multi-arch images. You can do it with Github Actions but it appears to require storing your Docker Hub password as a Github secret? That's dumb, but maybe I misunderstood. Anyaway, just build and push manually:

 - Get `docker buildx`. I had to stupid shit to get this installed, hopefully it's packaged properly by the time you're reading this.
 - `docker buildx create --use --name build --node build`
 - `docker login`
 - `docker buildx build --platform linux/amd64,linux/arm64 -t bjackman/awair-local-prometheus:latest --push .`

Running 'im
============

To run it, you need to add the `--awair-address` argument to the commandline (it isn't in the Dockerfile). In my case, the command to run the docker image looks like this:

```
docker run -it --rm -p 8080:8080 awair-local-prometheus awair-local-prometheus --awair-address=http://awair-elem-143b7b/
```

Now I get my air data at `http://localhost:8080/air-data/`. This service also exposes metrics about itself at `/metrics`.
