package trace2receiver

import "fmt"

func debugDescribe(base string, lval string, rval string) string {
	if len(base) == 0 {
		return fmt.Sprintf("[%s -> %s]", lval, rval)
	} else {
		return fmt.Sprintf("%s/[%s -> %s]", base, lval, rval)
	}
}

// Try to lookup the name of the custom ruleset or detail level using
// value passed in the `def_param` for the `Ruleset Key`.
func (fs *FilterSettings) lookupRulesetNameByRulesetKey(params map[string]string, debug_in string) (rs_dl_name string, ok bool, debug_out string) {
	debug_out = debug_in

	rskey := fs.NamespaceKeys.RulesetKey
	if len(rskey) == 0 {
		return "", false, debug_out
	}

	rs_dl_name, ok = params[rskey]
	if !ok || len(rs_dl_name) == 0 {
		return "", false, debug_out
	}

	// Acknowledge that we saw the ruleset key in the request and will try to use it.
	debug_out = debugDescribe(debug_out, "rskey", rs_dl_name)

	return rs_dl_name, true, debug_out
}

// Lookup ruleset or detail level name based upon the nickname (if the
// key is defined in the filter settings and if the worktree sent
// a def_param for it).
func (fs *FilterSettings) lookupRulesetNameByNickname(params map[string]string, debug_in string) (rs_dl_name string, ok bool, debug_out string) {
	debug_out = debug_in

	nnkey := fs.NamespaceKeys.NicknameKey
	if len(nnkey) == 0 {
		return "", false, debug_out
	}

	nnvalue, ok := params[nnkey]
	if !ok || len(nnvalue) == 0 {
		return "", false, debug_out
	}

	// Acknowledge that we saw the nickname in the request.
	debug_out = debugDescribe(debug_out, "nickname", nnvalue)

	rs_dl_name, ok = fs.NicknameMap[nnvalue]
	if !ok || len(rs_dl_name) == 0 {
		// Acknowledge that the nickname was not valid.
		debug_out := debugDescribe(debug_out, nnvalue, "UNKNOWN")
		return "", false, debug_out
	}

	// Acknowledge that we will try to use the nickname.
	debug_out = debugDescribe(debug_out, nnvalue, rs_dl_name)

	return rs_dl_name, true, debug_out
}

// Lookup the name of the default ruleset or detail level from
// the global defaults section in the filter settings if it has one.
func (fs *FilterSettings) lookupDefaultRulesetName(debug_in string) (rs_dl_name string, ok bool, debug_out string) {
	debug_out = debug_in

	if len(fs.Defaults.RulesetName) == 0 {
		return "", false, debug_out
	}

	// Acknowledge that we will try to use the global default.
	debug_out = debugDescribe(debug_out, "default-ruleset", fs.Defaults.RulesetName)

	return fs.Defaults.RulesetName, true, debug_out
}

// Determine whether a ruleset or detail level was requested.
func (fs *FilterSettings) lookupRulesetName(params map[string]string, debug_in string) (rs_dl_name string, ok bool, debug_out string) {
	debug_out = debug_in

	// If the command sent a `def_param` with the "Ruleset Key" that
	// is known, use it.
	rs_dl_name, ok, debug_out = fs.lookupRulesetNameByRulesetKey(params, debug_out)
	if !ok {
		// Otherwise, if the command sent a `def_param` with the "Nickname Key"
		// that has a known mapping, use it.
		rs_dl_name, ok, debug_out = fs.lookupRulesetNameByNickname(params, debug_out)
		if !ok {
			// Otherwise, if the filter settings defined a global default
			// ruleset, use it.
			rs_dl_name, ok, debug_out = fs.lookupDefaultRulesetName(debug_out)
		}
	}

	return rs_dl_name, ok, debug_out
}

// Use the global builtin default detail level.
func useBuiltinDefaultDetailLevel(debug_in string) (dl FSDetailLevel, debug_out string) {
	dl, _ = getDetailLevel(FSDetailLevelDefaultName)
	// Acknowledge that we will use the builtin default.
	debug_out = debugDescribe(debug_in, "builtin-default", FSDetailLevelDefaultName)
	return dl, debug_out
}

// Use the ruleset default detail level.  (This was set to the global
// builtin default detail level if it wasn't set in the ruleset YML.)
func (rsdef *RSDefinition) useRulesetDefaultDetailLevel(debug_in string) (dl FSDetailLevel, debug_out string) {
	dl, _ = getDetailLevel(rsdef.Defaults.DetailLevelName)
	// Acknowledge that we will use the ruleset default for this command.
	debug_out = debugDescribe(debug_in, "ruleset-default", rsdef.Defaults.DetailLevelName)
	return dl, debug_out
}

// Lookup the detail level for a command using the CmdMap in this ruleset.
//
// We try: `<exe>:<verb>#<mode>`, `<exe>:<verb>`, and `<exe>` until we find
// a match.  Then fallback to the ruleset default.  We assume that the CmdMap
// only has detail level values (and not links to other custom rulesets), so
// we won't get lookup cycles.
func (rsdef *RSDefinition) lookupCommandDetailLevelName(qn QualifiedNames, debug_in string) (string, bool, string) {
	// See if there is an entry in the CmdMap for this Git command.
	dl_name, ok := rsdef.CmdMap[qn.qualifiedExeVerbModeName]
	if ok {
		return dl_name, true, debugDescribe(debug_in, qn.qualifiedExeVerbModeName, dl_name)
	}

	dl_name, ok = rsdef.CmdMap[qn.qualifiedExeVerbName]
	if ok {
		return dl_name, true, debugDescribe(debug_in, qn.qualifiedExeVerbName, dl_name)
	}

	dl_name, ok = rsdef.CmdMap[qn.qualifiedExeBaseName]
	if ok {
		return dl_name, true, debugDescribe(debug_in, qn.qualifiedExeBaseName, dl_name)
	}

	return "", false, debug_in
}

// Compute the net-net detail level that we should use for this Git command.
func computeDetailLevel(fs *FilterSettings, params map[string]string,
	qn QualifiedNames) (FSDetailLevel, string) {

	if fs == nil {
		// No filter-spec, assume global builtin default detail level.
		return useBuiltinDefaultDetailLevel("")
	}

	rs_dl_name, ok, debug := fs.lookupRulesetName(params, "")
	if !ok {
		// No ruleset or detail level, assume global builtin default detail level.
		return useBuiltinDefaultDetailLevel(debug)
	}

	// If the name is a detail level rather than a named ruleset, then we use it
	// as is (since we don't do per-command filtering for detail levels).
	dl, err := getDetailLevel(rs_dl_name)
	if err == nil {
		return dl, debug
	}

	// Try to look it up as a custom ruleset.
	rsdef, ok := fs.rulesetDefs[rs_dl_name]
	if !ok {
		// Acknowledge that the ruleset name is not valid/unknown.
		debug = debugDescribe(debug, rs_dl_name, "INVALID")

		// We do not have a ruleset with that name.  Silently assume the builtin
		// default detail level.
		return useBuiltinDefaultDetailLevel(debug)
	}

	// Acknowledge that we are trying command-level filtering starting with
	// the full expression.
	debug = debugDescribe(debug, "command", qn.qualifiedExeVerbModeName)

	// Use the requested ruleset and see if this command has a
	// command-specific filtering.
	dl_name, ok, debug := rsdef.lookupCommandDetailLevelName(qn, debug)
	if !ok {
		return rsdef.useRulesetDefaultDetailLevel(debug)
	}

	dl, err = getDetailLevel(dl_name)
	if err == nil {
		return dl, debug
	}

	// We should not get here because we validated the spelling of all
	// of the CmdMap values and the default value when we validated the
	// `config.yml`.  But force a sane backstop.
	dl, _ = getDetailLevel(FSDetailLevelDefaultName)
	debug = debugDescribe(debug, "BACKSTOP", FSDetailLevelDefaultName)

	return dl, debug
}
