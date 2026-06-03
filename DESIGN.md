# hook2email - Design

For what the tool is and how to use it, see [README.md](README.md).
This document covers what is not visible from reading the code:
the shape the design takes and things deliberately not done.

## Design

hook2email implements: receive a webhook payload as a data
structure, build the email message as a data structure, submit it
to a local MTA. The Go code does exactly that.

Independence from any specific webhook source falls out of this
shape. The fields, enums, and identifiers a particular source
uses have nothing to do with HTTP, JSON, or SMTP delivery, so
they do not appear in the code. They live in the JSON Schema
(the input data structure) and the Go `text/template` (the
output data structure), both external files that travel with
the deployment.

The template renders the DATA section from `Subject:` onward.
hook2email prepends the envelope headers (`From`, `To`, `Date`,
`Message-ID`, `MIME-Version`, `Content-Type: text/plain;
charset=utf-8`) and normalises line endings to CRLF. Headers
that the operator wants beyond `Subject` — `Reply-To`,
`Auto-Submitted`, `X-*` — go in the template alongside `Subject`.
The envelope headers are code-owned and must not be redefined in
the template.

The MVP transport is plaintext SMTP on port 25 with no AUTH.
TLS, SMTP authentication, queueing, retry, bounce handling, and
DKIM/SPF signing are the local MTA's responsibility. The intended
deployment is `smtp.server: localhost:25` against a host-local
Postfix or sendmail that handles relay to the outside.

## Non-goals

These are intentional exclusions. Each is something hook2email
could reasonably do, but does not.

- **Multiple schemas or templates per process.** Run multiple
  processes if multiple are required.

- **HTML body, multipart, attachments.** The contract is one
  `text/plain; charset=utf-8` body.

- **Multiple recipients per endpoint.** One endpoint sends to one
  `to`. Fan-out to multiple recipients is expressed as multiple
  endpoints or as a mailing-list address on the MTA side.

- **Conditional logic in the configuration.** Conditions belong
  in the template (using `{{ if }}`) or in the upstream sender.

- **Filtering or muting at the adapter layer.** If a webhook
  arrives, it is rendered and submitted. Suppression belongs
  upstream.

- **Custom template functions beyond `json`.** Path accessors,
  string manipulation helpers, date formatters, and other
  helpers are not provided. `json` is included so JSON-derived
  strings can be embedded into header values with the quoting
  and escaping JSON syntax requires; other helpers would extend
  the template language with capabilities orthogonal to that.

- **Authentication on the HTTP API.** Put a reverse proxy in
  front for access control.
