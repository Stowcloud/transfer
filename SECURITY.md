# Security policy

Report suspected vulnerabilities privately through GitHub Security Advisories.
Do not publish exploit details in a public issue.

The transfer package treats destinations, content identities, policies, and
provider credentials as opaque caller-owned values. Callers must authorize
intents and validate destination capabilities before invoking strategies. The
package does not perform path confinement or credential handling.

Only the latest release and the current default branch receive fixes while the
module is pre-1.0.
