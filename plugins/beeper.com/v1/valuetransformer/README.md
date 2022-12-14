# Beeper value transformer for Kustomize
A backwards compatible KRM Exec Function for transforming templated Kubernetes resources from any sensible variable or secrets source.

To be able to execute this function Kustomize needs to be run with the following flags and the binary of this project needs to be available in your `$PATH`:
```sh
kustomize build --enable-alpha-plugins --enable-exec
```

## Configuration
Add transformers block to your `kustomization.yaml`:
```yaml
...
transformers;
  - valuetransformer.yaml
```

Create `valuetransformer.yaml` next to it:
```yaml
apiVersion: beeper.com/v1
kind: ValueTransformer
metadata:
  name: valuetransformer
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: valuetransformer
includes:
  - <path>
sources:
  <alias>:
    type: <type>
    args:
      [source specific arguments]
      awsRegion: [AWS region]
      awsRoleArn: [AWS role ARN]
merges:
  <alias>:
    ...
transforms:
 - source: <alias>
   regex: [regex]
   target:
     kind: <kind>
     name: [name]
     namespace: [namespace]
excludes:
 - kind: <kind>
   name: [name]
   namespace: [namespace]
```

If you are running this with older Kustomize (or `kubectl kustomize`) you need to drop the annotations and make sure the executable is in the correct plugin directory.

All sources support environment variable expansion before evaluating:
```yaml
sources:
  common:
    type: File
    args:
      path: ${HOME}/foo/bar.yml
```

## Includes
Allows including another ValueTransformer files using absolute paths.
Supports environment variable expansion.

```yaml
includes:
  - ${SOME_VAR}/valuetransformer.yaml
```

## Sources
Sources are variable data that are grouped to a source alias and flattened to Terraform-like dot notation.
Nesting and lists are supported.

For example the following YAML input with the file source:
```yaml
foo: bar
some:
  depth: here
lists:
  - are
  - supported
```

Will be flattened for the transformer to:
```
foo = bar
some.depth = here
lists.0 = are
lists.1 = supported
```

### Variable
Inlined variables, useful for overlays.
Vars must be defined.

```yaml
sources:
  <alias>:
    type: Variable
    vars:
      foo: bar
```

### Environment
Vars must be defined with the variables that are to be expanded.
Use the same value for `ALIAS` if you don't want to rename it.

```yaml
sources:
  <alias>:
    type: Environment
    vars:
      NAME: ALIAS
```

### Exec
Executes a command with and expects the output to be valid YAML.
Support may be expanded in the future.

Direct exec, first argument is looked up from current `$PATH`.
```yaml
sources:
  <alias>:
    type: Exec
    args:
      command: ['sops', '-d', '/path/to/secrets.enc.yaml']
```

Shell exec through `/bin/sh -c <command>`:
```yaml
sources:
  <alias>:
    type: Exec
    args:
      command: sops -d /path/to/secrets.enc.yaml
```

### File

Variable files, YAML or JSON. Vars is optional, default is to expand all. Remote files over `s3://` are supported.

```yaml
sources:
  <alias>:
    type: File
    args:
      path: /path/to/some.json
    vars:
      original: alias
```

```yaml
sources:
  <alias>:
    type: File
    args:
      path: s3://<bucket>/path/to/some.yaml
      # AWS keys are optional, default env context is used as the base
      awsRegion: eu-central-1
      awsRoleArn: arn:...
```

### AWS Secrets Manager

Use AWS Secrets Manager value as JSON input.
Non-JSON secrets are not supported in v1.

```yaml
sources:
  <alias>:
    type: SecretsManager
    args:
      name: some/secret
      # AWS keys are optional, default env context is used as the base
      awsRegion: eu-central-1
      awsRoleArn: arn:...
```

### Terraform state

Local or remote Terraform state output variables.
Setting `output` will root the secret tree to given output key.

If `output` is not given the whole TF output will be built into a tree.
If `path` contains an S3 URL it is fetched with AWS credentials from there before processing.
Local paths should also work.

```yaml
sources:
  <alias>:
    type: TerraformState
    args:
      path: s3://<bucket>/state/<name>.tfstate
      output: [name]
      # AWS keys are optional, default env context is used as the base
      awsRegion: eu-central-1
      awsRoleArn: arn:...
```

## Merges

Merges allow you to take in multiple sources and build a new combined source for transformation.
The resulting alias works like any source in transformations.

```yaml
merges:
  <alias>:
    key: old_source.key
    nested:
      key: second_source.nested.key
```

All keys are flattened as for any source and all values are expected to be prefixed with the source alias and then the fully qualified flattened path of the original key.

## Transforms

Select Kubernetes objects for transforming.
Maps sources with optional transformation regex to resources.
Sources and targets can be repeated to combine multiple sources.

The default regex is to replace envsubst style `${some.nested.source}` variables with nesting support.
Unmatched variables are left untouched.

Transform _all_ ConfigMaps with the source called `vars` with the default regex.
```yaml
transforms:
  - source: vars
    target:
      kind: ConfigMap
```

Transform all ingresses called `foo` with `__FOO__` syntax with the source called `ingress`:
```yaml
transforms:
  - source: ingress
    regex: __([A-Z_]+)__
    target:
      kind: Ingress
      name: foo
```

Process Jinja style templating:
```yaml
transforms:
  - source: something
    regex: '{{\s*([^\s}]+)\s*}}'
    target:
      kind: ConfigMap
      name: foo
```

## Excludes

Exclude Kubernetes objects for transforming.
All keys are optional.

```yaml
excludes:
  - kind: ConfigMap
    name: init-script
    namespace: somewhere
```

## TODO
- local cache for remote sources to speed up multiple executions within build
- clean up and expand AWS configuration
- reverse annotation based transformer/source selection?
- multiple targets per transform to prevent repetition?