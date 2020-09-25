# Paketo NPM Start Cloud Native Buildpack

## `gcr.io/paketo-buildpacks/npm-start`

The NPM Start CNB sets the start command for the given application. The start
command is generated from the contents of `package.json`. For example, given a
`package.json` with the following content:

```json
{
  "scripts": {
    "prestart": "<prestart-command>",
    "poststart": "<poststart-command>",
    "start": "<start-command>"
  }
}
```

The start command will be `<prestart-command> && <start-command> && <poststart-command>`.

## Integration

This CNB sets a start command, so there's currently no scenario we can
imagine that you would need to require it as dependency.

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
```
pack build <app-name> -p <path-to-app> -b <path/to/node-engine.cnb> -b \
<path/to/npm-install.cnb> -b build/buildpackage.cnb
```

## `buildpack.yml` Configurations

There are no extra configurations for this buildpack based on `buildpack.yml`.

## Run Tests

To run all unit tests, run:
```
./scripts/unit.sh
```

To run all integration tests, run:
```
/scripts/integration.sh
```

## Graceful shutdown and signal handling

You can add signal handlers in your app to support graceful shutdown and
program interrupts. If running a node server is the start command, the
buildpack runs the node server as the init process, and thus it ignores any
signal with the default action. As a result, the process will not terminate on
`SIGINT` or `SIGTERM` unless it is coded to do so. You can also use docker's
`--init` flag to wrap your node process with an init system that will properly
handle signals.
