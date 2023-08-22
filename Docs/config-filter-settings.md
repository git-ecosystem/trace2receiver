# Config Filter Settings

The `filter.yml` file controls how the `trace2receiver` component
translates the Trace2 data stream from Git commands into OTEL data
structures.  This filtering is content- and context-aware and is
independent of any statistical filtering performed by later stages in
the OTEL Collector pipeline.

The filter settings pathname is set in the
`receivers.trace2receiver.filter`
parameter in the main `config.yml` file.



## Smart Filtering using Detail Levels, Rulesets, and Repo Nicknames

The `trace2receiver` does "smart filtering" rather than just
"percentage filtering".  This allows it to control the verbosity of
the generated telemetry from the Trace2 data stream from Git commands.
For example, you might want very verbose output for your monorepo
while doing a performance study and only minimal output otherwise.  Or
you might want no telemetry at all for insignificant or personal
repos.

1. *Detail Levels:* There are four builtin detail levels. These vary
from no telemetry to very verbose telemetry.  These form the
foundation of the filtering system.

2. *Rulesets:* Rulesets build upon detail levels. They let you define
a meaningful name for a set of filtering patterns, such as dropping
telemetry for "uninteresting" commands and requesting verbose
telemetry for "interesting" ones. Rulesets can only refer to detail
levels. They cannot refer to other rulesets.

3. *Repo Nicknames:* Repo Nicknames are an aliasing technique built on
top of rulesets.  They serve two roles: (1) they select a detail level
or ruleset for content filtering, and (2) they help with data
aggregation from different repo instances across different machines.

As we'll see later, a Git command can refer to a detail level, a
ruleset, or a repo nickname to override the default filtering and
telemetry verbosity.

The following sections explain each of these concepts in more detail.
And later in this document we'll see how they can be used by Git
commands.



### Builtin Detail Levels

All detail level names begin with a `dl:` prefix to distinguish
them from ruleset names and repo nicknames.

```
<detail-level> ::= "dl:drop"
                 | "dl:summary"
                 | "dl:process"
                 | "dl:verbose"
```

1. `dl:drop` -- Drop or omit all telemetry for the command.

2. `dl:summary` -- Generate basic telemetry for the command; the
primary focus is the lifespan of the command, the arguments, and exit
code.

3. `dl:process` -- Adds process-level data events, process-level timer
and counter values, and child process (and hook) events to the
summary-level data.

4. `dl:verbose` -- Adds thread-level and region-level details to the
process-level data.



### User-defined Rulesets

All ruleset names have a `rs:` prefix to distinguish them from detail
levels and repo nicknames.

```
<ruleset-name> ::= "rs:<string>"
```

The content of a ruleset is defined in a
[ruleset file](./config-ruleset-definition.md).

A ruleset name is essentially an alias for the underlying ruleset
file.  Using a ruleset name avoids requiring users know how and where
the telemetry service is installed.

The `filter.yml` file contains a dictionary to map ruleset names to
pathnames:

```
rulesets:
  <ruleset-name-1>: <ruleset-pathname-1>
  <ruleset-name-2>: <ruleset-pathname-2>
  ...
```

Ruleset files will be loaded when the receiver starts up.

> [!NOTE]
> If you want to modify the list of rulesets or edit one of the
> ruleset files, you'll need to restart the telemetry service
> when you're finished.



### Repo Nicknames

A repo nickname is another level aliasing on top of rulesets.
Conceptually, this is a way to say that this repo is an instance
of project "foo" and that telemetry data from it can be aggregated
with Git command data from other instances of project "foo".

This avoids the need for the telemetry service or data store to try to
_guess_ how to aggregate data by parsing the `remote.origin.url` or
the basename of the repo root directory.  Users can simple say that
this repo is an instance of repo "foo" and aggregate or partition
data as they want.

Nicknames also let us say that all instances of repo "foo" should use
the ruleset "rs:bar".

A repo nickname is a simple string without either `dl:` or `rs:` prefix.

The `filter.yml` file contains a dictionary to map nicknames to detail
levels or rulesets:

```
nicknames:
  <nickname-1>: <ruleset-name> | <detail-level>
  <nickname-1>: <ruleset-name> | <detail-level>
  ...
```



## Telemetry Meta Data

Git can be told to send additional Git config key/value pairs in the
Trace2 telemetry string using the
[`trace2.configparams`](https://git-scm.com/docs/api-trace2#Documentation/technical/api-trace2.txt-ConfigdefparamEvents)
config setting.  We can use that mechanism to have Git send extra meta
data to help `trace2receiver` decide how to generate or filter OTEL
data.

_In the examples here we have chosen to use the `otel.trace2.*`
namespace for all of these special config settings, but you can use
any prefix you want._

To tell Git to always send these config settings, we must add this
namespace to the `trace2.configparams` config setting at the `global`
or `system` level.

```
$ git config --system trace2.configparams "otel.trace2.*"
```

The `filter.yml` contains a dictionary to define the spelling of
these keys:

```
keynames:
  nickname_key: "otel.trace2.nickname"
  ruleset_key:  "otel.trace2.ruleset"
```



### Using the Repo Nickname Config Setting

We can set repo nicknames on our repos using the Git config
setting named in the `nickname_key` parameter.  Thereafter, Git will
silently send the nickname on every Git command in those repos.

The nickname should be local to the individual repo.


```
$ cd /path/to/my/repo1
$ git config --local otel.trace2.nickname "monorepo"
$
$ cd /path/to/my/repo2
$ git config --local otel.trace2.nickname "monorepo"
$
$ cd /path/to/my/repo3
$ git config --local otel.trace2.nickname "personal"
```

Or you can set it for a single command:

```
$ cd /path/to/my/repo4
$ git -c otel.trace2.nickname=personal status
```

If no nickname is defined or the given repo nickname is not defined in
the `filter.yml` file, the receiver will fall back to the default
filter settings.

_In the above example, I've suggested "monorepo" and "personal" as
nicknames, but you might use the base name of the repo, such as
`git.git` or `chromium.git` or just `chromium`.  Or you might use a
project codename (and further hide the origin URL)._

_You might use different nicknames for desktop users versus build
servers on instances of the same repo to help partition the data in
the data store by use cases or machine classes.  For example, you
might want to see the P80 fetch times for interactive users and not
have to sift thru fetches from build machines._



### Using the Ruleset Config Setting

The repo nickname helps identify/classify the data and lets you set an
expected ruleset.  However, there are times when you might want to
maintain the above classification, but use different verbosity for
some commands or for some repo instances.

The `ruleset_key` parameter lets you explicitly select a ruleset and
override the ruleset associated with the nickname.


```
$ cd /path/to/my/repo1
$ git config --local otel.trace2.ruleset "rs:production"
$
$ cd /path/to/my/repo2
$ git config --local otel.trace2.ruleset "rs:test"
$
$ cd /path/to/my/repo3
$ git config --local otel.trace2.ruleset "dl:drop"
```

Or set it for a single command:

```
$ cd /path/to/my/repo4
$ git -c otel.trace2.ruleset="dl:summary" status
```

If the named ruleset or detail level is not defined in the `filter.yml`
file, the receiver will fall back to the default filter settings.

If a Git command sends both a `ruleset_key` and `nickname_key`, the
`ruleset_key` wins.  (Both key values will be included in the OTEL
telemetry, but the telemetry data will be filtered using the value of
the `ruleset_key`.)



## Filter Settings Syntax

Now that all of the concepts have been introduced, we can describe
the complete syntax of the `filter.yml` file.  All sections and rows
are optional.

```
keynames:
  nickname_key: <git-config-key>
  ruleset_key:  <git-config-key>

nicknames:
  <nickname-1>: <ruleset-name> | <detail-level>
  <nickname-1>: <ruleset-name> | <detail-level>
  ...

rulesets:
  <ruleset-name-1>: <ruleset-pathname-1>
  <ruleset-name-2>: <ruleset-pathname-2>
  ...

defaults:
  ruleset: <ruleset-name> | <detail-level>
```

The value of the `defaults.ruleset` parameter will be used when a Git
command does not specify a repo nickname or ruleset.

If there is no default, the builtin default of `dl:summary` will be
used.



## Example

In this filter:

```
keynames:
  nickname_key: "otel.trace2.nickname"
  ruleset_key:  "otel.trace2.ruleset"

nicknames:
  monorepo: "dl:verbose"
  personal: "dl:drop"

rulesets:
  "rs:status": "./rulesets/rs-status.yml"

defaults:
  ruleset: "dl:summary"
```

The receiver will watch for the `otel.trace2.nickname` and
`otel.trace2.ruleset` Git config key/values pairs in the Trace2
telemetry stream to override the builtin filtering defaults.

Commands that send `otel.trace2.ruleset = rs:status` will
use the command-level filtering described in the `rs-status.yml`
ruleset file.

Commands that send `otel.trace2.nickname = monorepo` will
use `dl:verbose` and emit very verbose telemetry.

Commands that send `otel.trace2.nickname = personal` will
use `dl:drop` and not emit any telemetry.

All other commands will use the default `dl:summary` and
emit command overview telemetry.


