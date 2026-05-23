# sitectl-archivesspace

`sitectl-archivesspace` is the LibOps sitectl plugin for ArchivesSpace.

It registers a first-class create definition for `https://github.com/libops/archivesspace` so the stack can be installed with:

```bash
sitectl create archivesspace
```

It also provides context-aware helpers:

- `sitectl archivesspace build`
- `sitectl archivesspace init`
- `sitectl archivesspace up`
- `sitectl archivesspace down`
- `sitectl archivesspace status`
- `sitectl archivesspace logs [SERVICE...]`
- `sitectl archivesspace rollout`

ArchivesSpace-specific helpers:

- `sitectl archivesspace api login`
- `sitectl archivesspace api request METHOD PATH`
- `sitectl archivesspace version`
- `sitectl archivesspace repositories [ID]`
- `sitectl archivesspace users [ID]`
- `sitectl archivesspace search QUERY`
- `sitectl archivesspace jobs --repo ID [JOB_ID]`
- `sitectl archivesspace schemas [NAME]`
- `sitectl archivesspace diagnostics`
- `sitectl archivesspace script SCRIPT [args...]`
- `sitectl archivesspace setup-database [args...]`
- `sitectl archivesspace backup [args...]`

API helpers accept `--url`, `--session`, and repeated `--query name=value` flags. `api login` accepts `--username` and `--password`, or reads `ARCHIVESSPACE_PASSWORD`.
