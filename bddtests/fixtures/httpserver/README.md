This is a test HTTP server used to check external connectivity for the HTTP Snap.

To build the server, run `make http-server` or `make all` at the root of this project. This will produce a binary at `build/test`.

You can configure the path to the config file by setting the environment variable EXT_SERVER_CFG_PATH
Alternatively you can place config file in `/etc/external-http-server/`

The config file can be used to configure paths to the TLS certs used for mutual authentication.
