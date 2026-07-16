# sitectl-archivesspace

`sitectl-archivesspace` simplifies the creation and operation of repositories created using the [LibOps ArchivesSpace template](https://github.com/libops/archivesspace). It provides sitectl commands for the ArchivesSpace API, resource shortcuts, container scripts, validation, and health checks.

Documentation: https://sitectl.libops.io/plugins/archivesspace

## Requirements

- Stable [`sitectl`](https://sitectl.libops.io/install) v1.0.0 or newer; this plugin uses RPC protocol 1.
- Docker with the Compose v2 plugin for local ArchivesSpace sites.
- No additional app-plugin dependency beyond core `sitectl`.

## Quick Start

Create a local ArchivesSpace site from the matching template:

```bash
sitectl create archivesspace/default \
  --template-repo https://github.com/libops/archivesspace \
  --path ./my-archivesspace-site \
  --type local \
  --checkout-source template \
  --default-context
```

The template README is at https://github.com/libops/archivesspace.

## Basic Operations

Use [`sitectl compose`](https://sitectl.libops.io/commands/compose) to start or inspect the stack:

```bash
sitectl compose up --remove-orphans -d
```

Use [`sitectl healthcheck`](https://sitectl.libops.io/commands/healthcheck) and [`sitectl validate`](https://sitectl.libops.io/commands/validate) to check the site:

```bash
sitectl healthcheck
sitectl validate
```

Use [`sitectl image`](https://sitectl.libops.io/commands/image) for local image or build-arg overrides:

```bash
sitectl image set --tag archivesspace=4.2.0 --tag solr=4.2.0
```

The plugin intentionally does not register broad development bind mounts because they can hide application content bundled in the base image. Add customizations through the downstream build or a narrowly targeted override.

Use [`sitectl set`](https://sitectl.libops.io/commands/set) for component changes; it updates component-owned files immediately:

```bash
sitectl set ingress enabled --mode https-custom --domain archivesspace.localhost
sitectl set ingress enabled --trusted-ip 203.0.113.10/32
```

See the [ArchivesSpace plugin docs](https://sitectl.libops.io/plugins/archivesspace) for API helpers, resource shortcuts, container scripts, lifecycle operations, and rollout details.

## License

`sitectl-archivesspace` is licensed under the MIT License.
