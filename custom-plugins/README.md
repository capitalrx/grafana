# custom-plugins

Staging area for plugin dists baked into the Grafana image (see the
`COPY custom-plugins/` step in the Dockerfile). Populate it with
`./scripts/build-custom-plugins.sh` before `docker build`:

```
custom-plugins/
  quickwit-quickwit-datasource/    # quickwit datasource dist (incl. gpx_* binaries)
```

The plugins land in `/usr/share/grafana/plugins-custom/<plugin-id>/` in the
image, deliberately **not** `/var/lib/grafana/plugins`, which is shadowed by
the Kubernetes data volume. Grafana loads them via `[plugin.<id>] path = ...`
config sections:

```ini
[plugins]
allow_loading_unsigned_plugins = quickwit-quickwit-datasource

[plugin.quickwit-quickwit-datasource]
path = /usr/share/grafana/plugins-custom/quickwit-quickwit-datasource
```

Don't also preinstall `quickwit-quickwit-datasource` from the catalog, or it
will compete with the baked-in copy.

The dist must be unsigned (no `MANIFEST.txt`): Grafana always rejects a
*modified* signed plugin regardless of `allow_loading_unsigned_plugins`, so the
script strips the manifest.
