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
func lookupFilterByRulesetKey(fs *FilterSettings, params map[string]string, debug string) (string, bool, string) {
	rskey := fs.NamespaceKeys.RulesetKey
	if len(rskey) == 0 {
		return "", false, debug
	}

	rsdl_value, ok := params[rskey]
	if !ok || len(rsdl_value) == 0 {
		return "", false, debug
	}

	debug = debugDescribe(debug, "rskey", rsdl_value)

	return rsdl_value, true, debug
}

// Lookup ruleset or detail level name based upon the nickname (if the
// key is defined in the filter settings and if the worktree sent
// a def_param for it).
func lookupFilterByNickname(fs *FilterSettings, params map[string]string, debug string) (string, bool, string) {
	nnkey := fs.NamespaceKeys.NicknameKey
	if len(nnkey) == 0 {
		return "", false, debug
	}

	nnvalue, ok := params[nnkey]
	if !ok || len(nnvalue) == 0 {
		return "", false, debug
	}

	debug = debugDescribe(debug, "nickname", nnvalue)

	rsdl_value, ok := fs.NicknameMap[nnvalue]
	if !ok || len(rsdl_value) == 0 {
		debug := debugDescribe(debug, nnvalue, "UNKNOWN")
		return "", false, debug
	}

	debug = debugDescribe(debug, nnvalue, rsdl_value)

	return rsdl_value, true, debug
}

// Lookup the default ruleset or detail level from the global defaults
// section in the filter settings.
func lookupFilterDefaultRulesetName(fs *FilterSettings, debug string) (string, bool, string) {
	rsdl := fs.Defaults.RulesetName
	if len(rsdl) == 0 {
		return "", false, debug
	}

	debug = debugDescribe(debug, "default-ruleset", rsdl)

	return rsdl, true, debug
}

// Compute the net-net detail level that we should use for this Git command.
func computeDetailLevel(fs *FilterSettings, params map[string]string,
	qn QualifiedNames) (FSDetailLevel, string) {

	var debug string

	// If the command sent a `def_param` "Ruleset Key", use it.
	rsdl_value, ok, debug := lookupFilterByRulesetKey(fs, params, debug)
	if !ok {
		// Otherwise, if the command sent a `def_param` "Repo Id Key"
		// that has mapping, use it.
		rsdl_value, ok, debug = lookupFilterByNickname(fs, params, debug)
		if !ok {
			// Otherwise, if the filter settings defined a global default
			// ruleset, use it.
			rsdl_value, ok, debug = lookupFilterDefaultRulesetName(fs, debug)
			if !ok {
				// Otherwise, apply the builtin default detail level.
				rsdl_value = FSDetailLevelDefaultName
				debug = debugDescribe(debug, "builtin-default", rsdl_value)
			}
		}
	}

	// If the overall value was a valid detail level rather than a
	// named ruleset, then we use it as is (since we don't do
	// per-command filtering for them).
	dl, ok := getDetailLevel(rsdl_value)
	if ok {
		return dl, debug
	}

	// Try to look it up as a custom ruleset.
	rsdef, ok := fs.rulesetDefs[rsdl_value]
	if !ok {
		debug = debugDescribe(debug, rsdl_value, "INVALID")

		// We do not have a ruleset with that name.  Silently assume the builtin
		// default detail level.
		dl, _ := getDetailLevel(FSDetailLevelDefaultName)
		debug = debugDescribe(debug, "builtin-default", FSDetailLevelDefaultName)
		return dl, debug
	}

	// Use the requested ruleset.

	debug = debugDescribe(debug, "command", qn.qualifiedExeVerbModeName)

	// See if there is an entry in the CmdMap for this Git command.
	//
	// We try: `<exe>:<verb>#<mode>`, `<exe>:<verb>`, and `<exe>` until we find
	// a match.  Then fallback to the ruleset default.  We assume that the CmdMap
	// only has detail level values (and not links to other custom rulesets), so
	// we won't get lookup cycles.
	dl_name, ok := rsdef.CmdMap[qn.qualifiedExeVerbModeName]
	if ok {
		debug = debugDescribe(debug, qn.qualifiedExeVerbModeName, dl_name)
	} else {
		dl_name, ok = rsdef.CmdMap[qn.qualifiedExeVerbName]
		if ok {
			debug = debugDescribe(debug, qn.qualifiedExeVerbName, dl_name)
		} else {
			dl_name, ok = rsdef.CmdMap[qn.qualifiedExeBaseName]
			if ok {
				debug = debugDescribe(debug, qn.qualifiedExeBaseName, dl_name)
			} else {
				// Use the ruleset default detail level.  (This was set to the global
				// default detail level if it wasn't set in the ruleset YML.)
				dl_name = rsdef.Defaults.DetailLevelName
				debug = debugDescribe(debug, "ruleset-default", dl_name)
			}
		}
	}

	dl, ok = getDetailLevel(dl_name)
	if ok {
		return dl, debug
	}

	// We should not get here because we validated the spelling of all
	// of the CmdMap values and the default value when we validated the
	// `config.yml`.  But force a sane backstop.
	dl, _ = getDetailLevel(FSDetailLevelDefaultName)
	debug = debugDescribe(debug, "BACKSTOP", FSDetailLevelDefaultName)

	return dl, debug
}
