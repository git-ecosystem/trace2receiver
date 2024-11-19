# Config Ruleset Definition

The `filter.yml` file controls how the `trace2receiver` component
translates and/or filters the Git Trace2 telemetry stream into OTEL
telemetry data.

Conceptually, the `filter.yml` layer says that telemetry for all Git
commands from a repo that is an instance of project "foo" should use
ruleset "rs:bar".  Ruleset "rs:bar" points to a pathname containing
the ruleset file.

A ruleset definition allows you to specify which commands on that repo
are "interesting" and which a not.  For example, you may only care
about `git status`, `git fetch`, and `git push`, but not `git
rev-list` or `git rev-parse`.



## Git Command Qualifiers

The Trace2 telemetry stream contains the name of the executable
and, when present, the name and mode.  We combine these (when
present) to fully describe the command.

1. `<exe>`: This is usally the basename of `argv[0]`.  Usually this is
`git`, but other tools may also emit Trace2 telemetry, such the
[GCM](https://github.com/git-ecosystem/git-credential-manager),
so we do not assume it is `git`.

2. `<name>`: The
[name (or verb)](https://git-scm.com/docs/api-trace2#Documentation/technical/api-trace2.txt-codecmdnamecode)
of the command, such a `status` or `fetch`.  This value is is
optional, since some `<exe>`'s might not have a name/verb.

3. `<mode>`: The
[mode](https://git-scm.com/docs/api-trace2#Documentation/technical/api-trace2.txt-codecmdmodecode)
of the command.  Commands like `git checkout` have different modes,
such as switching branches vs restoring an individual file.  This
value is also optional.



## A Fully Qualified Name

The `trace2receiver` combines the above command part names into a
single fully qualified token:

```
<cmd-3>   ::= "<exe>:<name>#<mode>"  # if both name and mode are present
<cmd-2>   ::= "<exe>:<name>"         # if mode is not present
<cmd-1>   ::= "<exe>"                # if name is not present

<cmd-fqn> ::= <cmd-3> | <cmd-2> | <cmd-1>
```

For example, `git:checkout#branch` or `git:status`.



## Ruleset Command Pattern Matching

A ruleset lets you select a different detail levels for different
commands.  The ruleset `.yml` file contains an optional dictionary
to map command patterns to detail levels.

The receiver will first try to find an entry for the `<cmd-3>`
in the dictionary.  If not present, it will try `<cmd-2>` and
then `<cmd-1>` until it finds a match.

If no match is found, the ruleset default (if present) will be used.
If the ruleset does not have a default value, the containing
`filter.yml` default or the receiver builtin default will be used.



##  Ruleset Definition Syntax

```
commands:
  <cmd-*>: <detail-level>
  <cmd-*>: <detail-level>
  ...

defaults:
  detail: <detail-level>
```



## Example

In this ruleset:

```
commands:
  "git:rev-list":        "dl:drop"      # (1)
  "git:config":          "dl:drop"      # (1)
  "git:checkout#path":   "dl:drop"      # (1)

  "git:checkout#branch": "dl:verbose"   # (2)

  "git:checkout":        "dl:process"   # (3)

  "git":                 "dl:summary"   # (4)

defaults:
  detail: "dl:drop"                     # (5)
```

The receiver will:
* (1) drop telemetry for `git rev-list`, `git config`, and individual
file (path) checkouts,
* (2) emit verbose telemetry for branch changes,
* (3) emit process level telemetry other types of `git checkout` commands,
* (4) emit summary telemetry for all other `git` commands, and
* (5) drop telemetry from any non-git command.

Note that the `commands` array is a dictionary rather than a list, so
order does not matter.  Lookups will try `<cmd-3>` then `<cmd-2>` and
then `<cmd-1>` until a match is found.



