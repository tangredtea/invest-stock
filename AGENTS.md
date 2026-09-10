# Repository agent instructions

## Production environment authority and deployment contract

These are required maintenance rules, not a claim that every existing release
entrypoint already implements them. Verify implementation before deployment.

- Business secrets belong in the server's canonical project .env (mode 0600).
  Preserve existing values and component-specific credential isolation. Keep
  migration, worker and mail-component env files separate when least privilege
  requires it; document their paths. Never copy all credentials into every service.
- 1Panel File Manager may edit the canonical file. Container Edit must not become
  a second configuration source. A restart does not reload container environment;
  use the reviewed deployment entrypoint to recreate affected services.
- Git tracks variable names, safe examples, validation rules and deployment code.
  Do not commit, bake into images, log, or transfer production business secrets
  through Actions. Deployment credentials and public build configuration are
  separate concerns; browser build variables must never contain secrets.
- Specify Compose project name, project directory, files and env paths explicitly.
  --env-file controls Compose interpolation; service env_file controls injection.
  Account for environment overrides and image defaults. Do not source .env as
  shell code or allow the caller's shell to override business configuration.
- Before changing configuration ownership, compare canonical env, rendered
  Compose and actual container env privately. Back up configuration with restricted
  permissions. Preserve runtime-only values; stop on conflicting values rather
  than choosing silently. Never fill missing credentials with fake values.
- Validate required and feature-conditional variables using the application's
  shared configuration validator. Disabled optional integrations may be empty.
  Missing production configuration must fail before migrations or replacement.
- Validate the candidate immutable image with a dedicated check-env command that
  exits without migrations, workers, schedulers, emails or payments. Do not run the
  ordinary application entrypoint as a supposedly read-only preflight. A stopped
  container/config render proves environment injection, not application validity.
- Build elsewhere; pull and resolve the release to its registry digest on the VPS.
  Keep a project deployment lock and coordinate with image cleanup using the same
  host lock. Hold protection across validation and replacement; recheck config
  identity immediately before applying to detect concurrent manual file edits.
- After recreation, verify environment, image digest and health against the
  validated release. Drift checks are read-only, return failure on differences,
  and report field names only, not values or reusable secret hashes. Compare image
  defaults separately so legitimate candidate-image changes are not mistaken for
  manual configuration edits. Do not auto-import drift into the canonical file.
- Keep image identity and a protected configuration snapshot per successful
  release. A rollback must account for configuration and migration compatibility.
  Host cleanup removes unused images, including rollback tags: ensure the registry
  retains previous digests and the rollback path can pull them again.
- Test missing required fields, disabled/enabled optional features, special-character
  values, Compose override precedence, runtime-only fields, failed preflight with
  the old container unchanged, and post-deploy drift. Report local tests, pushed
  code, installed server scripts and live rollout as separate completion evidence.

For future VPS deployment, declare the project directory explicitly under
`/home/debian/project/<project>/.env`; do not infer deployment from repository presence.
