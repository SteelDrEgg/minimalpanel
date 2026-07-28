# HTTP management API

All responses use the common JSON response envelope. Except for login, logout,
and authentication checks, management endpoints require an authenticated
session. `[API]` capabilities decide whether a domain is exposed, and
`Route.Allow` can further restrict each HTTP method and path by group.

## Session and kernel

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/login` | Create a session |
| `POST` | `/api/logout` | Delete the current session |
| `GET` | `/api/check-auth` | Inspect the current session |
| `GET` | `/api/kernel/version` | Read the kernel version |
| `POST` | `/api/kernel/reload` | Reload `config.toml` |

Kernel reload is always registered and requires authentication. It can be
restricted or disabled for HTTP callers with `Route.Allow`.

## Users and groups

| Capability | Method | Path | Purpose |
| --- | --- | --- | --- |
| `User` | `GET` | `/api/user` | List users with derived group memberships |
| `User` | `GET` | `/api/user/{name}` | Read one user and its groups |
| `User` | `PUT` | `/api/user/{name}` | Create a user or replace its password |
| `User` | `DELETE` | `/api/user/{name}` | Delete a user |
| `Group` | `GET` | `/api/group` | List groups and members |
| `Group` | `GET` | `/api/group/{name}` | Read one group |
| `Group` | `PATCH` | `/api/group/{name}` | Atomically replace its complete users array |
| `Group` | `DELETE` | `/api/group/{name}` | Delete a group |

User responses never include password hashes. Group membership is stored as
usernames in `[Groups]`; deleting a user currently does not cascade into those
arrays.

## Pages and access

| Capability | Method | Path | Purpose |
| --- | --- | --- | --- |
| `Pages` | `GET` | `/api/page` | List configured status pages |
| `Pages` | `PUT` | `/api/page/{status}` | Set a local page path |
| `Pages` | `DELETE` | `/api/page/{status}` | Delete a page mapping |
| `Access` | `GET` | `/api/access` | List route access rules |
| `Access` | `PATCH` | `/api/access` | Create or replace a rule's complete groups array |
| `Access` | `DELETE` | `/api/access` | Delete a rule |

Access write requests carry `method`, `path`, and `groups` in the body. Method
is optional and omission means every method. An empty groups array explicitly
leaves the matching route open.

## Log and network

| Capability | Method | Path |
| --- | --- | --- |
| `Log` | `GET`, `PATCH` | `/api/log/level` |
| `Log` | `GET`, `PATCH` | `/api/log/format` |
| `Network` | `GET`, `PATCH` | `/api/network/listen` |
| `Network` | `GET`, `PATCH` | `/api/network/tls` |

Request and response properties use the resource name (`level`, `format`,
`listen`, or `tls`). GET and successful PATCH responses also contain
`requires_restart`.

## Services

Every endpoint in this section is controlled by the single `Service`
capability.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/service/discovered` | Scan and list discovered packages |
| `GET` | `/api/service/running` | List runtime instances |
| `GET` | `/api/service/detail/{name}` | Read one discovered service |
| `POST` | `/api/service/start/{name}` | Start a service |
| `POST` | `/api/service/stop/{name}` | Stop a service |
| `POST` | `/api/service/restart/{name}` | Restart a service |
| `GET`, `PATCH` | `/api/service/dir` | Read or change ServiceDir |
| `GET`, `PATCH` | `/api/service/temp-dir` | Read or change ServiceTempDir |
| `GET`, `PATCH`, `DELETE` | `/api/service/config/{name}` | Manage an explicit service override |
| `GET`, `PATCH` | `/api/service/params/{name}` | Read or patch explicit Params |

ServiceTempDir responses contain `requires_restart`, which is true when at
least one running service was loaded with a different directory. Params PATCH
accepts `set` and `remove`; the resulting conf operations are applied in one
transaction.
