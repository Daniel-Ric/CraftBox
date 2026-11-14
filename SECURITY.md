# Security Policy

CraftBox is a desktop and command-line application for capturing and working with Minecraft Bedrock data. Security reports should go to the CraftBox maintainer, **SpindexGFX**, rather than being posted with exploit details in a public issue.

## Versions and reports

Include the exact release version or commit in every report. Where practical, check the latest published release or current `master` to see whether the issue is still present.

There is no published long-term support schedule or guaranteed backport window. Reports affecting older builds or modified copies are still useful, especially when the same problem may exist in current source. Include the modifications needed to reproduce it.

## Private reporting

Contact the maintainer on [Discord](https://discord.com/users/252138923952832523). This is the private contact used by the maintainer's related project as well.

If private contact is unavailable, open a minimal [GitHub issue](https://github.com/Daniel-Ric/CraftBox/issues) titled `SECURITY: private contact requested`, with a way to reach you. Do not include the affected account, credentials, payload, private capture or exploit details in that public issue. It is only a request to establish private contact.

A private report should include:

- The affected version, operating system and workspace or CLI command.
- The relevant input or entry point, such as a server packet, imported archive, world, project file or local bridge request.
- Reproduction steps and a small proof of concept using synthetic data where possible.
- The permissions and user actions required, including whether the input comes from a remote server or a local file.
- Expected behavior, actual behavior and the practical impact.
- Any relevant source paths, logs or suggested correction, without unrelated personal data.

Do not delay a report just because you do not have a complete fix or proof of concept. If you are unsure whether a problem is security-sensitive, ask privately first.

## Security-sensitive areas

Examples worth reporting include:

- Exposure of Microsoft/Xbox credentials, session tokens, resource-pack content keys or private capture data through files, logs, exports or bridge responses.
- An untrusted packet, resource pack, image, model, world or studio project causing unintended code execution, unauthorized file access or writes outside the selected destination.
- Archive path traversal, unsafe extraction or an import that overwrites unrelated user files.
- Unauthorized access to the desktop's local bridge, map-window service or native operations, including a missing authorization check on a reachable operation.
- Bypassing the explicit controls around packet intervention, sending or native command execution.
- Remote-controlled URLs or metadata causing unauthorized requests to local services or disclosure of credentials.
- Update verification failures that allow an untrusted executable to be accepted or installed.
- A reasonably sized untrusted input causing a repeatable crash, unbounded allocation or prolonged resource exhaustion.

These are reporting examples, not claims that each control has been independently audited. Include the actual reachable path and impact; source patterns alone may not establish a vulnerability.

## Intended behavior and ordinary bugs

The native terminal runs commands with the current user's permissions. Capture scripts are trusted input and can change capture behavior. Executing a command or script that the user deliberately supplied is not by itself a security bug. A path that lets untrusted content trigger that execution without the intended user action is worth reporting.

Rendering artifacts, missing textures, unsupported packets and connection failures normally belong in [GitHub Issues](https://github.com/Daniel-Ric/CraftBox/issues). Use private reporting if the problem also exposes sensitive data, crosses an access boundary or enables realistic denial of service.

A dependency report should explain how the vulnerable code is reached through CraftBox. An advisory alone does not establish that every use of the dependency is affected. Issues in bundled snapshots can still matter here, including CraftBox's local changes.

## Credentials and recordings

Account files, packet archives, world data and logs can contain sensitive information. Do not attach them to a public issue or pull request without reviewing and removing that information. Use a minimal example instead of an entire private session.

If a secret has already been exposed, report its location privately without repeating the value. The account or service owner should revoke the affected token or rotate the credential where possible. Removing a file or editing a Git commit does not invalidate a credential that someone may already have copied.

Do not test a discovered credential by accessing another person's account or data. Maintainers can help investigate CraftBox's handling of the secret, but cannot revoke credentials owned by unrelated users or external services.

## Testing and disclosure

Use your own local environment, accounts and fixtures, or systems whose owner has explicitly authorized the test. Avoid collecting other players' data or disrupting public Minecraft servers, Microsoft/Xbox services or third-party catalogs. A minimal demonstration is sufficient; causing real damage is not required.

Keep sensitive details private while the issue is investigated and a correction is prepared. Coordinate public disclosure with the maintainer so users have an opportunity to update. Say whether you would like credit or prefer to remain anonymous.

There is no guaranteed response time, release date or bounty program stated by this policy. Updates and disclosure timing depend on the issue and available maintainer time.

## Related policies

- [Contributing](CONTRIBUTING.md) covers ordinary reports, development and pull requests.
- [Code of Conduct](CODE_OF_CONDUCT.md) covers participation and conduct reports.
- [README](Readme.md) describes supported workflows, compatibility and project attribution.
