# Paketo NPM Start Cloud Native Buildpack

## `gcr.io/paketo-buildpacks/npm-start`

The NPM Start CNB sets the start command for the given application. The start
command uses [tini](https://github.com/krallin/tini) as the init process to run
`npm start`

## Integration

This CNB sets a start command, so there's currently no scenario we can
imagine that you would need to require it as dependency.

To package this buildpack for consumption:
```
$ ./scripts/package.sh
```

## `buildpack.yml` Configurations

There are no extra configurations for this buildpack based on `buildpack.yml`.
