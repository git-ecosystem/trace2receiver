# Config Ruleset Definition

The `filter.yml` file controls how the `trace2receiver` component
translates and/or filters the Git Trace2 telemetry stream into OTEL
telemetry data.

Conceptually, the `filter.yml` layer says that telemetry for all Git
commands from a repo that is an instance of project "foo" should use
ruleset "rs:bar".  And ruleset "rs:bar" is defined in some pathname.

A ruleset definition allows you to specify which commands on that repo
are "interesting" and which a not.  For example, you may only care
about `git status`, `git fetch`, and `git push`, but not `git
rev-list` or `git rev-parse`.



## Git Command Qualifiers

The Trace2 telemetry stream contains the name of the executable
and, when present, the name and mode:

1. `<exe>`: This is usally the basename of `argv[0]`.  Usually this is
`git`, but other tools may also emit Trace2 telemetry, such the
[GCM](https://github.com/git-ecosystem/git-credential-manager),
so we do not assume it is `git`.

2. `<name>`: The
[name (or verb)](https://git-scm.com/docs/api-trace2#Documentation/technical/api-trace2.txt-codecmdnamecode)
of the command, such a `status` or `fetch`.  This value is is
optional, since not all Git commands have a name/verb.

3. `<mode>`: The
[mode](https://git-scm.com/docs/api-trace2#Documentation/technical/api-trace2.txt-codecmdmodecode)
of the command.  Commands like `git checkout` have different modes,
such as switching branches vs restoring an individual file.  This
value is optional, since not all Git commands have multiple modes.



## A Fully Qualified Name

The `trace2receiver` combines the above command part names into a
single fully qualified token:

```
<cmd-3>   ::= "<exe>:<name>#<mode>"  # if both name and mode are present
<cmd-2>   ::= "<exe>:<name>"         # if only name is present
<cmd-1>   ::= "<exe>"                # if neither are present

<cmd-fqn> ::= <cmd-3> | <cmd-2> | <cmd-1>
```

For example, `git:checkout#branch` or `git:status`.



## Ruleset Command Pattern Matching

A ruleset lets you select a different detail levels for different
commands.  The ruleset `.yml` file contains an optional dictionary
to map command patterns to detail levels.

The receiver will first try to find an entry for the `<cmd-fqn>`
in the dictionary.  If not present, it will try the shorter forms
in turn.



##  Ruleset Definition Syntax

```
commands:
  <cmd-*>: <detail-level>
  <cmd-*>: <detail-level>
  ...

defaults:
  detail: <detail-level>
```

The value of the `defaults.detail` parameter will be used if no
command pattern matches.

If there is not default, the containing `filter.yml` default
or the receiver builtin default will be used.



## Example

In this ruleset:

```
commands:
  "git:rev-list":        "dl:drop"
  "git:config":          "dl:drop"
  "git:checkout#path":   "dl:drop"
  "git:checkout#branch": "dl:verbose"
  "git:checkout":        "dl:process"
  "git":                 "dl:summary"

defaults:
  detail: "dl:drop"
```

The receiver will:
* drop telemetry for `git rev-list`, `git config`, and individual
file (path) checkouts,
* emit verbose telemetry for branch changes,
* emit process level telemetry other types of `git checkout` commands,
* emit summary telemetry for all other `git` commands, and
* drop telemetry from any non-git command.

Note that the `commands` array is a dictionary, so order
does not matter.



