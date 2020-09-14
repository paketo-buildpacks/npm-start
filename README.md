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

To package this buildpack for consumption:
```
$ ./scripts/package.sh
```

## `buildpack.yml` Configurations

There are no extra configurations for this buildpack based on `buildpack.yml`.
