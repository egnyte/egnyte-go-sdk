# Changelog

## v2.0.0 — Unreleased

Initial release — a ground-up implementation covering the CFS Public API
surface (~150 endpoints, 30 services) with zero external dependencies,
validated end to end against a live Egnyte domain (several endpoint
contracts were corrected from live behavior where the published specs
disagree).

- Client core: domain-derived base URLs (https-only, incl. Gov Cloud via
  full host), functional options, impersonation headers, thread-safe
  token rotation, credential redaction in `fmt` output, typed `APIError`
  covering all six observed Egnyte error envelopes, `Retry-After`
  capture, per-token rate limit parsing on every response
- Request paths are contained to their API route family: dot segments
  are rejected so a caller-supplied path or ID can never redirect an
  authenticated call to another endpoint
- Successful responses are size-capped (64 MiB default, configurable
  with `WithMaxResponseBytes`) and reject trailing JSON; streaming
  downloads are unaffected
- OAuth: password, authorization-code and refresh grants; authorization
  URL builder; token introspection and revocation
- Services: AI (Q&A, assistant, knowledge bases, hybrid search), Agents,
  Audit (v1 reports + v2 stream), Bookmarks, Comments, Controlled Docs,
  Document Portal, eTMF, Events, File System (metadata, actions, by-ID
  lookups, folder stats, lock/unlock, folder options), File System
  Content (streaming + chunked uploads), Groups, Insights, Links,
  Metadata, MSP, Navigate, Permissions, Procore, Project Custom Fields,
  Project Folders, Salesforce, Search, Sign, Tokens, Trash, Upload
  Requests, Users, Webhooks, Workflows
- 200+ httptest-based unit tests (no credentials required) and an opt-in
  real-domain integration suite (`-tags integration`): full scenario,
  per-family read-only sweep, file lifecycle, live-contract checks,
  chunked upload
- ChunkedFileChecksum helper producing Egnyte's composite whole-file
  checksum format for chunked uploads
