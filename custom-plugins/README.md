# custom-plugins

Staging area for plugin dists baked into the Grafana image (see the
`COPY custom-plugins/` step in the Dockerfile). Populate it with
`./scripts/build-custom-plugins.sh` before `docker build`:

```
custom-plugins/
  quickwit-quickwit-datasource/    # quickwit datasource dist (incl. gpx_* binaries)
  grafana-lokiexplore-app/         # Logs Drilldown, ported to Quickwit
```

The plugins land in `/usr/share/grafana/plugins-custom/<plugin-id>/` in the
image, deliberately **not** `/var/lib/grafana/plugins`, which is shadowed by
the Kubernetes data volume. Grafana loads them via `[plugin.<id>] path = ...`
config sections:

```ini
[plugins]
allow_loading_unsigned_plugins = quickwit-quickwit-datasource,grafana-lokiexplore-app

[plugin.quickwit-quickwit-datasource]
path = /usr/share/grafana/plugins-custom/quickwit-quickwit-datasource

[plugin.grafana-lokiexplore-app]
path = /usr/share/grafana/plugins-custom/grafana-lokiexplore-app
```

Don't also preinstall these plugins from the catalog. This fork already drops
`grafana-lokiexplore-app` from Grafana's default preinstall list, and loads
`[plugin.<id>] path` sources before the plugins directory, so a stale catalog
copy left on the data volume is ignored in favour of the baked-in one.

The dists must be unsigned (no `MANIFEST.txt`): Grafana always rejects a
*modified* signed plugin regardless of `allow_loading_unsigned_plugins`, so the
script strips the manifest.
