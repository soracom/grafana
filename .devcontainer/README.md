# Using the devcontainer

In order for the devcontainer to be able to bind mount host directories into the devcontainer (containing plugin repos or the db), it is necessary to specify local paths to bind to. devcontainer.json references the following environment variables, which must be set in the shell environment that launches VS-Code. Either this can be done by exporting the variables in a terminal then calling `code.`, or by setting them in your .bashrc file.

Becareful that your folder for grafana plugins does not include this grafana repo in it! Infinite symlink loops will cause the grafana server to fail to start.

Example:
LAGOON_HOST_DATA_DIR=/home/$USER/Documents/grafana
LAGOON_HOST_PLUGINS_DIR=/home/$USER/Documents/git_repo/grafana-plugins

```
cd ${PATH_TO_GRAFANA_REPO}
export LAGOON_HOST_DATA_DIR=/home/$USER/Documents/grafana
export LAGOON_HOST_PLUGINS_DIR=/home/$USER/Documents/git_repo/grafana-plugins
code .
```

After the variables are set, the devcontainer can be launched by executing the `Dev Containers: Rebuild and Reopen in Container` from the VSCode command menu.

# Starting Grafana/Lagoon

Grafana can be started inside the devcontainer environment by using the VSCode launch options, enabling debugging.
The "Run Server" and the "Run Frontend" must both be running simultaneously for Grafana to work.
