This thing reads data from the Awair local API and serves it up in the format that Prometheus understands.

To build the Docker image: `docker build -t awair-local-prometheus .`

To run it, you need to add the `--awair-address` argument. In my case, the command to run the docker image looks like this:

```
docker run -it --rm -p 8080:8080 awair-local-prometheus awair-local-prometheus --awair-address=http://awair-elem-143b7b/
```

Now I get my air data at `http://localhost:8080/air-data/`. This service also exposes metrics about itself at `/metrics`.
