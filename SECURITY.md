# Security

LibreQuenda keeps residents' bookings, accounts and, while a council checks them,
their census data. If you find a vulnerability, please tell us privately first.

## Reporting a vulnerability

Use GitHub's private reporting: **Security → Report a vulnerability** on this
repository. Please include what is affected, how to reproduce it, and the impact
you see. Do not open a public issue, and do not test against an installation you
do not run yourself.

We answer within a week, and publish a fix and an advisory once installations can
update (`docker compose pull && docker compose up -d --wait`).

## Supported versions

Only the latest release, and `main`, receive security fixes.
