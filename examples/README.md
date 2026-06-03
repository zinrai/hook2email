# examples

A sample schema, template, payload, and configuration showing the
four artifacts a hook2email deployment needs.

## Files

| File | What |
|---|---|
| `schema.json` | JSON Schema describing the expected payload shape (`title`, `body`, `level`, `url`). |
| `template.tmpl` | Go `text/template` rendering the DATA section: Subject, `Auto-Submitted`, blank line, body. |
| `payload.json` | Sample request body matching the schema. |
| `hook2email.yaml` | Configuration tying the above together, using `/etc/hook2email/...` as placeholder absolute paths. |

## Preview the rendered message

Without starting the server or sending mail:

```
hook2email render \
  --schema examples/schema.json \
  --template examples/template.tmpl \
  --payload examples/payload.json
```

Output is the DATA section (Subject + body) that hook2email would
hand to the MTA. The envelope headers (From, To, Date, Message-ID,
MIME-Version, Content-Type) are added at send time and not shown
by `render`.

## Run end-to-end

Deploy the four files to absolute paths the hook2email process
can read. A typical layout:

```
/etc/hook2email/hook2email.yaml
/etc/hook2email/schema.json
/etc/hook2email/template.tmpl
```

The local MTA on `localhost:25` handles TLS, AUTH, queueing, and
relay upstream. Start the server:

```
hook2email serve --config /etc/hook2email/hook2email.yaml
```

Send the sample payload from another terminal:

```
curl -X POST -H 'Content-Type: application/json' \
  --data @payload.json \
  http://localhost:8080/webhooks/ops
```

The mail arrives at the address bound to that endpoint.

## Verification scope

`go test ./...` exercises hook2email's internal flow only: config
loading, schema validation, template rendering, message
construction, and handler error propagation. It does not verify
that any particular MTA (Postfix, sendmail) will accept the
produced message — protocol-level acceptance, header folding,
8-bit handling, and queue admission are MTA behaviour that lives
outside the Go code.

The end-to-end run above against the MTA you intend to deploy
behind is the integration verification path. Run it after
changing the schema or template, or after changing the MTA
configuration.
