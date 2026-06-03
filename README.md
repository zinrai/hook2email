# hook2email

A webhook receiver that validates incoming JSON against a JSON
Schema and renders it to an email message through a Go
`text/template`, then submits it to a local MTA on SMTP port 25.
The Go code knows only HTTP, JSON, JSON Schema, Go templates, and
SMTP; the shape of any specific webhook source lives in two
external files (a schema and a template).

For non-goals and design rationale, see [DESIGN.md](DESIGN.md).

## Subcommands

| Subcommand | Purpose |
|---|---|
| `serve --config <path>` | Run the HTTP server. |
| `check --config <path>` | Validate config, schema, and template without starting the server. Intended for CI. |
| `render --schema <path> --template <path> --payload <path>` | Render a sample payload through a given schema and template and print the DATA section (Subject + body) that would be submitted to the MTA. No network request. |
| `version` | Print build version. |

## Configuration

```yaml
listen: ":8080"
schema_file: /etc/hook2email/schema.json
template_file: /etc/hook2email/template.tmpl
endpoints:
  - path: /webhooks/ops
    smtp:
      server: localhost:25
      from: alerts@example.com
      to: oncall-ops@example.com
  - path: /webhooks/dev
    smtp:
      server: localhost:25
      from: alerts@example.com
      to: oncall-dev@example.com
```

| Field | Required | Notes |
|---|---|---|
| `listen` | yes | HTTP listen address. |
| `schema_file` | yes | Absolute path to the JSON Schema file (Draft 2020-12). |
| `template_file` | yes | Absolute path to the Go `text/template` file. |
| `endpoints` | yes | At least one endpoint. |
| `endpoints[].path` | yes | URL path. Must start with `/`. Cannot be `/-/healthy` or `/-/ready`. |
| `endpoints[].smtp.server` | yes | SMTP server address `host:port`. The intended target is a host-local MTA on `localhost:25`. |
| `endpoints[].smtp.from` | yes | Address used in `MAIL FROM` and the `From:` header. |
| `endpoints[].smtp.to` | yes | Address used in `RCPT TO` and the `To:` header. One recipient per endpoint. |

Configuration changes require a process restart; there is no hot
reload.

## Schema file

A standard [JSON Schema](https://json-schema.org/) document
(Draft 2020-12) describing the expected shape of the JSON the
sender will POST. Every request is validated before rendering;
failures return HTTP 400.

A sample is at [examples/schema.json](examples/schema.json).

## Template file

A Go [`text/template`](https://pkg.go.dev/text/template) that
renders the DATA section of the email from `Subject:` onward.
hook2email prepends the envelope headers (`From`, `To`, `Date`,
`Message-ID`, `MIME-Version`, `Content-Type: text/plain;
charset=utf-8`) and normalises line endings to CRLF.

The template owns every header from `Subject:` onward. Operators
may add any further headers alongside `Subject` — common cases
are `Reply-To`, `Auto-Submitted: auto-generated` (RFC 3834,
suppresses vacation responders), and `X-*` headers for
receiver-side filtering.

Templates have the standard Go template built-ins (`index`,
`range`, `if`, `with`, comparison operators) and one custom
function:

| Function | Purpose |
|---|---|
| `json` | Encodes its argument as a JSON value, performing the quoting and escaping JSON syntax requires. Use to embed JSON-derived strings into header values: `{{ .x \| json }}`. |

A sample is at [examples/template.tmpl](examples/template.tmpl).

## License

This project is licensed under the [MIT License](./LICENSE).
