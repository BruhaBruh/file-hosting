# File Hosting Service

Features

- File hosting service with REST and gRPC APIs
- Useful configuration file
- Generative file name for Public API
- Supports custom metadata for files
- Supports file expiration and permanent storage
- Supports S3 or local storage.
- Supports authentication for file upload with non-generative name and permanent storage

## REST

`GET /file/:file`

Retrieve a file by its ID. If has custom metadata, it will be returned in the response headers with a prefix `X-Meta-`.

`GET /file/:file/metadata`

Retrieve metadata for a file by its ID.

`POST /upload`

Upload a file by `file` in multipart/form-data.

Can set own metadata by using headers starts with `X-Meta-`.

Can set duration by using `d` query parameter. Default duration is 1 hour.
Possible durations:
- `5m` - 5 minutes
- `60m` - 1 hour
- `1h` - 1 hour
- `24h` - 1 day
- `1d` - 1 day
- `7d` - 1 week
- `1w` - 1 week

`POST /upload/:file`

Upload a file by `file` in multipart/form-data.

Requires authentication by `Authorization` header with secret key.

Can set own metadata by using headers starts with `X-Meta-`.

Can set duration by using `d` query parameter. Default duration is 1 hour.
Possible durations:
- `-1` - Permanent
- `5m` - 5 minutes
- `60m` - 1 hour
- `1h` - 1 hour
- `24h` - 1 day
- `1d` - 1 day
- `7d` - 1 week
- `1w` - 1 week

## gRPC

You can find proto file in `proto` directory.

## TODO

- [ ] Add traces, metrics and logging. Also add collectors and exporters
