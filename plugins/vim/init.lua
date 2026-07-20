-- Vim compatibility layer for ttt.
--
-- Single-file by necessity: the plugin sandbox strips package.loaders down to
-- the preload loader (internal/plugin/sandbox.go), so a plugin cannot require
-- sibling .lua files. Sections below are delimited and appended phase by phase.
--
-- Phase 0: mode state machine (normal/insert), Esc handling, status indicator.

local ttt = require("ttt")
local events = require("ttt.events")

-- ---------------------------------------------------------------------------
-- State
-- ---------------------------------------------------------------------------

local state = {
	enabled = true,
	mode = "normal", -- normal | insert | visual | visual_line | visual_block | replace
	pending = "", -- unconsumed keys, e.g. "g", "2d"
	count = nil, -- pending count prefix
	operator = nil, -- pending operator
	register = nil, -- pending "x register prefix
	last_change = nil, -- replay payload for `.`
	registers = {},
	marks = {},
	macro = { recording = nil, keys = {}, playing = false },
}

local MODE_LABELS = {
	normal = "-- NORMAL --",
	insert = "-- INSERT --",
	visual = "-- VISUAL --",
	visual_line = "-- VISUAL LINE --",
	visual_block = "-- VISUAL BLOCK --",
	replace = "-- REPLACE --",
}

-- ---------------------------------------------------------------------------
-- Status indicator
-- ---------------------------------------------------------------------------

local function render_status()
	if not state.enabled then
		ttt.remove_status_item("mode")
		return
	end
	ttt.set_status_item("left", "mode", MODE_LABELS[state.mode] or state.mode, { priority = 10 })
end

local function set_mode(mode)
	state.mode = mode
	state.pending = ""
	state.count = nil
	state.operator = nil
	state.register = nil
	render_status()
end

-- ---------------------------------------------------------------------------
-- Key normalization
--
-- key.press delivers {type, key, rune, mod}. `rune` is present only for
-- printable input; `mod` is nil when unmodified. Verified event shapes:
--
--   ctrl+d  -> {key="Ctrl-D", rune=nil, mod="ctrl"}   -- BOTH, not one or the other
--   esc     -> {key="Esc",    rune=nil, mod=nil}
--   j       -> {key="j",      rune="j", mod=nil}
--
-- tcell folds Ctrl into the key constant and *also* sets ModCtrl, so the key
-- name must win: normalizing off `mod` alone yields "ctrl+Ctrl-D".
-- See internal/plugin/event_convert.go and internal/app/keys.go:68-71.
--
-- Canonical tokens: runes keep their case ("g" vs "G"); named keys lowercase
-- ("esc", "enter"); control keys collapse to "ctrl-d"; alt prefixes as "alt-".
-- ---------------------------------------------------------------------------

local function has_mod(mod, name)
	return mod ~= nil and mod:find(name, 1, true) ~= nil
end

local function token_of(ev)
	local alt = has_mod(ev.mod, "alt")

	if ev.rune and ev.rune ~= "" then
		return alt and ("alt-" .. ev.rune) or ev.rune
	end

	local key = ev.key or ""
	local ctrl_letter = key:match("^Ctrl%-(.+)$")
	local tok = ctrl_letter and ("ctrl-" .. ctrl_letter:lower()) or key:lower()
	return alt and ("alt-" .. tok) or tok
end

-- gopher-lua is Lua 5.1, where the %g character class does not exist, so this
-- is a byte-range check rather than a pattern match.
local function is_printable(tok)
	if #tok ~= 1 then
		return false
	end
	local b = tok:byte(1)
	return b >= 0x20 and b <= 0x7e
end

-- Keys that would otherwise mutate the buffer, so normal mode must swallow
-- them even before the motions/operators that give them meaning exist.
local MUTATING_KEYS = {
	["enter"] = true,
	["backspace"] = true,
	["backspace2"] = true,
	["delete"] = true,
	["tab"] = true,
}

local ESCAPE_TOKENS = {
	["esc"] = true,
	["ctrl-["] = true,
}

-- True when Esc has something Vim-side to cancel. When it does not, Esc is
-- passed through so core EscapeDismissers (clear search highlight, close
-- panels) still run -- the interceptor now sits above them.
local function has_pending()
	return state.pending ~= "" or state.count ~= nil or state.operator ~= nil or state.register ~= nil
end

-- ---------------------------------------------------------------------------
-- Mode handlers
-- ---------------------------------------------------------------------------

local function handle_insert(tok)
	if ESCAPE_TOKENS[tok] then
		set_mode("normal")
		return true
	end
	return false
end

local function handle_normal(tok)
	if ESCAPE_TOKENS[tok] then
		if not has_pending() then
			return false
		end
		set_mode("normal")
		return true
	end

	if tok == "i" then
		set_mode("insert")
		return true
	end

	-- Unhandled printable keys are swallowed rather than inserted: normal mode
	-- must never type into the buffer. Modified and special keys fall through
	-- so ctrl+s, ctrl+p and friends keep working.
	if is_printable(tok) or MUTATING_KEYS[tok] then
		return true
	end

	return false
end

local HANDLERS = {
	normal = handle_normal,
	insert = handle_insert,
}

local function on_key(ev)
	if not state.enabled then
		return false
	end
	local handler = HANDLERS[state.mode]
	if not handler then
		return false
	end
	return handler(token_of(ev))
end

-- ---------------------------------------------------------------------------
-- Commands
-- ---------------------------------------------------------------------------

local function enable()
	state.enabled = true
	set_mode("normal")
end

local function disable()
	state.enabled = false
	render_status()
end

local function toggle()
	if state.enabled then
		disable()
	else
		enable()
	end
end

ttt.register({
	commands = {
		{ id = "vim.toggle", title = "Vim: Toggle Vim Mode", handler = toggle },
		{ id = "vim.enable", title = "Vim: Enable Vim Mode", handler = enable },
		{ id = "vim.disable", title = "Vim: Disable Vim Mode", handler = disable },
	},
})

events.on("key.press", on_key)

-- set_status_item is only wired after WirePlugin runs, which is after this file
-- executes, so the initial indicator is deferred by a tick.
ttt.set_timeout(0, render_status)
