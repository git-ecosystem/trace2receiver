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
func (tr2 *trace2Dataset) lookupFilterByRulesetKey() (string, bool, string) {
	rskey := tr2.rcvr_base.RcvrConfig.FilterSettings.NamespaceKeys.RulesetKey
	if len(rskey) == 0 {
		return "", false, ""
	}

	rsdl_value, ok := tr2.process.paramSetValues[rskey]
	if !ok || len(rsdl_value) == 0 {
		return "", false, ""
	}

	debug := debugDescribe("", "rskey", rsdl_value)

	return rsdl_value, true, debug
}

// Lookup ruleset or detail level name based upon the nickname (if the
// key is defined in the filter settings and if the worktree sent
// a def_param for it).
func (tr2 *trace2Dataset) lookupFilterByNickname() (string, bool, string) {
	nnkey := tr2.rcvr_base.RcvrConfig.FilterSettings.NamespaceKeys.NicknameKey
	if len(nnkey) == 0 {
		return "", false, ""
	}

	nnvalue, ok := tr2.process.paramSetValues[nnkey]
	if !ok || len(nnvalue) == 0 {
		return "", false, ""
	}

	rsdl_value, ok := tr2.rcvr_base.RcvrConfig.FilterSettings.NicknameMap[nnvalue]
	if !ok || len(rsdl_value) == 0 {
		return "", false, ""
	}

	debug := debugDescribe("", "nickname", nnvalue)
	debug = debugDescribe(debug, nnvalue, rsdl_value)

	return rsdl_value, true, debug
}

// Lookup the default ruleset or detail level from the global defaults
// section in the filter settings.
func (tr2 *trace2Dataset) lookupFilterDefaultRulesetName() (string, bool, string) {
	rsdl := tr2.rcvr_base.RcvrConfig.FilterSettings.Defaults.RulesetName
	if len(rsdl) == 0 {
		return "", false, ""
	}

	debug := debugDescribe("", "defaults", rsdl)

	return rsdl, true, debug
}

// Compute the net-net detail level that we should use for this Git command.
func (tr2 *trace2Dataset) computeDetailLevel() (FSDetailLevel, string) {
	// If the command sent a `def_param` "Ruleset Key", use it.
	rsdl_value, ok, debug := tr2.lookupFilterByRulesetKey()
	if !ok {
		// Otherwise, if the command sent a `def_param` "Repo Id Key"
		// that has mapping, use it.
		rsdl_value, ok, debug = tr2.lookupFilterByNickname()
		if !ok {
			// Otherwise, if the filter settings defined a global default
			// ruleset, use it.
			rsdl_value, ok, debug = tr2.lookupFilterDefaultRulesetName()
			if !ok {
				// Otherwise, apply the builtin default detail level.
				rsdl_value = FSDetailLevelDefaultName
				debug = debugDescribe("", "builtin-default", rsdl_value)
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
	rsdef, ok := tr2.rcvr_base.RcvrConfig.FilterSettings.rulesetDefs[rsdl_value]
	if !ok {
		// We do not have a ruleset with that name.  Silently assume the builtin
		// default detail level.
		dl, _ := getDetailLevel(FSDetailLevelDefaultName)

		debug = debugDescribe(debug, rsdl_value, "INVALID")
		debug = debugDescribe(debug, "builtin-default", FSDetailLevelDefaultName)
		return dl, debug
	}

	// Use the requested ruleset.

	debug = debugDescribe(debug, "command", tr2.process.qualifiedExeVerbModeName)

	// See if there is an entry in the CmdMap for this Git command.
	//
	// We try: `<exe>:<verb>#<mode>`, `<exe>:<verb>`, and `<exe>` until we find
	// a match.  Then fallback to the ruleset default.  We assume that the CmdMap
	// only has detail level values (and not links to other custom rulesets), so
	// we won't get lookup cycles.
	dl_name, ok := rsdef.CmdMap[tr2.process.qualifiedExeVerbModeName]
	if ok {
		debug = debugDescribe(debug, tr2.process.qualifiedExeVerbModeName, dl_name)
	} else {
		dl_name, ok = rsdef.CmdMap[tr2.process.qualifiedExeVerbName]
		if ok {
			debug = debugDescribe(debug, tr2.process.qualifiedExeVerbName, dl_name)
		} else {
			dl_name, ok = rsdef.CmdMap[tr2.process.qualifiedExeBaseName]
			if ok {
				debug = debugDescribe(debug, tr2.process.qualifiedExeBaseName, dl_name)
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
