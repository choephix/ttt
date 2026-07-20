-- Vim compatibility layer for ttt.
--
-- Single-file by necessity: the plugin sandbox strips package.loaders down to
-- the preload loader (internal/plugin/sandbox.go), so a plugin cannot require
-- sibling .lua files. Sections below are delimited and appended phase by phase.
--
-- Phase 0: mode state machine (normal/insert), Esc handling, status indicator.
-- Phase 1: normal-mode motions with {count} prefixes.

local ttt = require("ttt")
local events = require("ttt.events")
local editor = require("ttt.editor")

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
	find_pending = nil, -- "f" | "F" | "t" | "T" awaiting its target char
	last_find = nil, -- { op = "f", ch = "x" } for `;` and `,`
	goal = nil, -- sticky column for j/k
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
	state.find_pending = nil
	state.goal = nil
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
	return state.pending ~= ""
		or state.count ~= nil
		or state.operator ~= nil
		or state.register ~= nil
		or state.find_pending ~= nil
end

-- ---------------------------------------------------------------------------
-- Buffer primitives
--
-- The editor Lua API is 1-based for both line and col, and `col` is a visual
-- (rune) column, not a byte index. Everything below works on rune arrays so
-- multi-byte lines behave. get_line is used rather than buffer_lines: the
-- latter copies the whole buffer, which is unacceptable on a key-press path.
-- ---------------------------------------------------------------------------

-- Split a UTF-8 string into an array of rune strings. Lua 5.1 has no utf8
-- library, so the lead byte decides the sequence length.
local function runes_of(s)
	local out = {}
	local i, n = 1, #s
	while i <= n do
		local b = s:byte(i)
		local len = 1
		if b >= 0xf0 then
			len = 4
		elseif b >= 0xe0 then
			len = 3
		elseif b >= 0xc0 then
			len = 2
		end
		out[#out + 1] = s:sub(i, i + len - 1)
		i = i + len
	end
	return out
end

local function line_runes(n)
	return runes_of(editor.get_line(n) or "")
end

local function line_count()
	return editor.line_count()
end

local function clamp(v, lo, hi)
	if v < lo then
		return lo
	end
	if v > hi then
		return hi
	end
	return v
end

-- Vim's normal mode puts the cursor *on* a character, so the last valid column
-- is #line (not #line + 1 as in insert mode). Empty lines still allow col 1.
local function max_col(r)
	if #r < 1 then
		return 1
	end
	return #r
end

-- Move and clear the sticky j/k goal column. Vertical motions bypass this and
-- manage the goal themselves.
local function move_to(l, c)
	l = clamp(l, 1, line_count())
	local r = line_runes(l)
	editor.set_cursor(l, clamp(c, 1, max_col(r)))
	state.goal = nil
end

local function clamp_cursor()
	local cur = editor.cursor()
	local r = line_runes(cur.line)
	local m = max_col(r)
	if cur.col > m then
		editor.set_cursor(cur.line, m)
	end
end

local function is_blank(ch)
	return ch == " " or ch == "\t"
end

local function first_non_blank(l)
	local r = line_runes(l)
	for i = 1, #r do
		if not is_blank(r[i]) then
			return i
		end
	end
	return 1
end

local function last_non_blank(l)
	local r = line_runes(l)
	for i = #r, 1, -1 do
		if not is_blank(r[i]) then
			return i
		end
	end
	return 1
end

-- ---------------------------------------------------------------------------
-- Character classes for word motions
--
-- 0 = blank, 1 = word char ([A-Za-z0-9_] plus any multi-byte rune), 2 = other
-- punctuation. For WORD motions (W/B/E/gE) everything non-blank is class 1.
-- Lua 5.1 has no %g class, so these are explicit byte ranges.
-- ---------------------------------------------------------------------------

local function class_of(ch, big)
	if ch == nil then
		return nil
	end
	if is_blank(ch) then
		return 0
	end
	if big then
		return 1
	end
	local b = ch:byte(1)
	if b >= 0x80 then
		return 1
	end
	if (b >= 0x30 and b <= 0x39) or (b >= 0x41 and b <= 0x5a) or (b >= 0x61 and b <= 0x7a) or b == 0x5f then
		return 1
	end
	return 2
end

-- A cursor walker over the buffer, one rune at a time, wrapping across lines.
local function walker(l, c)
	local w = { l = l, c = c, r = line_runes(l), n = line_count() }

	function w:ch()
		return self.r[self.c]
	end

	function w:empty()
		return #self.r == 0
	end

	function w:fwd()
		if self.c < #self.r then
			self.c = self.c + 1
			return true
		end
		if self.l >= self.n then
			return false
		end
		self.l = self.l + 1
		self.r = line_runes(self.l)
		self.c = 1
		return true
	end

	function w:back()
		if self.c > 1 then
			self.c = self.c - 1
			return true
		end
		if self.l <= 1 then
			return false
		end
		self.l = self.l - 1
		self.r = line_runes(self.l)
		self.c = max_col(self.r)
		return true
	end

	-- "End of word": non-blank whose successor on the line is a different class
	-- or does not exist.
	function w:at_word_end(big)
		local cls = class_of(self:ch(), big)
		if cls == nil or cls == 0 then
			return false
		end
		return self.c >= #self.r or class_of(self.r[self.c + 1], big) ~= cls
	end

	return w
end

-- ---------------------------------------------------------------------------
-- Motions
-- ---------------------------------------------------------------------------

-- w / W: start of the next word. An empty line counts as a word, matching Vim.
local function word_forward(count, big)
	local cur = editor.cursor()
	local w = walker(cur.line, cur.col)
	for _ = 1, count do
		if w:empty() then
			-- Standing on an empty line: it is its own word, so just leave it.
			if not w:fwd() then
				break
			end
		else
			local cls = class_of(w:ch(), big)
			if cls ~= 0 then
				while class_of(w:ch(), big) == cls do
					if not w:fwd() then
						break
					end
				end
			end
		end
		while not w:empty() do
			local k = class_of(w:ch(), big)
			if k ~= nil and k ~= 0 then
				break
			end
			if not w:fwd() then
				break
			end
		end
	end
	move_to(w.l, w.c)
end

-- b / B: start of the current or previous word.
local function word_back(count, big)
	local cur = editor.cursor()
	local w = walker(cur.line, cur.col)
	for _ = 1, count do
		if not w:back() then
			break
		end
		while not w:empty() and (class_of(w:ch(), big) or 0) == 0 do
			if not w:back() then
				break
			end
		end
		local cls = class_of(w:ch(), big)
		if cls ~= nil and cls ~= 0 then
			while w.c > 1 and class_of(w.r[w.c - 1], big) == cls do
				w.c = w.c - 1
			end
		end
	end
	move_to(w.l, w.c)
end

-- e / E and ge / gE: scan for the next/previous end-of-word position.
local function word_end(count, big, backward)
	local cur = editor.cursor()
	local w = walker(cur.line, cur.col)
	for _ = 1, count do
		local step = backward and w.back or w.fwd
		if not step(w) then
			break
		end
		while not w:at_word_end(big) do
			if not step(w) then
				break
			end
		end
	end
	move_to(w.l, w.c)
end

-- j / k, honouring the sticky goal column so travelling through short lines
-- does not permanently shorten the column.
local function vertical(dir, count)
	local cur = editor.cursor()
	local goal = state.goal or cur.col
	local l = clamp(cur.line + dir * count, 1, line_count())
	local r = line_runes(l)
	editor.set_cursor(l, clamp(goal, 1, max_col(r)))
	state.goal = goal
end

local function goto_line(n)
	local l = clamp(n, 1, line_count())
	move_to(l, first_non_blank(l))
end

local function paragraph(dir, count)
	local n = line_count()
	local l = editor.cursor().line
	for _ = 1, count do
		local probe = l + dir
		while probe >= 1 and probe <= n do
			if editor.get_line(probe) == "" then
				break
			end
			probe = probe + dir
		end
		l = clamp(probe, 1, n)
	end
	move_to(l, 1)
end

-- f/F/t/T. `skip_adjacent` is used when repeating a till-motion so `;` after
-- `tx` advances instead of sitting still on the same character.
local function find_char(op, ch, count, skip_adjacent)
	local cur = editor.cursor()
	local r = line_runes(cur.line)
	local forward = (op == "f" or op == "t")
	local till = (op == "t" or op == "T")
	local dir = forward and 1 or -1

	local pos = cur.col + dir
	if till and skip_adjacent then
		pos = cur.col + dir * 2
	end

	local left = count
	while pos >= 1 and pos <= #r do
		if r[pos] == ch then
			left = left - 1
			if left == 0 then
				move_to(cur.line, till and (pos - dir) or pos)
				return
			end
		end
		pos = pos + dir
	end
end

local REVERSE_FIND = { f = "F", F = "f", t = "T", T = "t" }

local function repeat_find(count, reverse)
	local last = state.last_find
	if not last then
		return
	end
	local op = reverse and REVERSE_FIND[last.op] or last.op
	find_char(op, last.ch, count, op == "t" or op == "T")
end

-- ---------------------------------------------------------------------------
-- Screen position and scrolling
--
-- SetCursor calls EnsureCursorVisible on the Go side, so every routine here
-- moves the cursor FIRST and scrolls LAST; the reverse order gets undone.
-- ---------------------------------------------------------------------------

local function viewport()
	local v = editor.viewport()
	return v.top_line, v.bottom_line, v.height
end

local function screen_line(where, count)
	local top, bottom = viewport()
	local l
	if where == "H" then
		l = top + (count - 1)
	elseif where == "L" then
		l = bottom - (count - 1)
	else
		l = math.floor((top + bottom) / 2)
	end
	l = clamp(l, top, bottom)
	goto_line(l)
end

local function scroll_page(dir, count, half)
	local top, _, h = viewport()
	local n = line_count()
	local amount = half and math.floor(h / 2) or h
	amount = math.max(1, amount) * count

	local cur = editor.cursor()
	move_to(cur.line + dir * amount, cur.col)

	if n > h then
		editor.scroll_to(clamp(top + dir * amount, 1, math.max(1, n - h + 1)))
	end
end

-- Ctrl-E / Ctrl-Y: scroll the view, dragging the cursor along only when it
-- would otherwise fall off the screen.
local function scroll_lines(dir, count)
	local top, _, h = viewport()
	local n = line_count()
	if n <= h then
		return
	end
	local new_top = clamp(top + dir * count, 1, math.max(1, n - h + 1))
	local cur = editor.cursor()
	local wanted = clamp(cur.line, new_top, math.min(n, new_top + h - 1))
	if wanted ~= cur.line then
		move_to(wanted, cur.col)
	end
	editor.scroll_to(new_top)
end

local function reposition(kind)
	local _, _, h = viewport()
	local line = editor.cursor().line
	local top
	if kind == "t" then
		top = line
	elseif kind == "b" then
		top = line - h + 1
	else
		top = line - math.floor(h / 2)
	end
	editor.scroll_to(math.max(1, top))
end

-- ---------------------------------------------------------------------------
-- Motion table
-- ---------------------------------------------------------------------------

local MOTIONS = {
	["h"] = function(n)
		local cur = editor.cursor()
		move_to(cur.line, cur.col - n)
	end,
	["l"] = function(n)
		local cur = editor.cursor()
		move_to(cur.line, cur.col + n)
	end,
	["j"] = function(n)
		vertical(1, n)
	end,
	["k"] = function(n)
		vertical(-1, n)
	end,
	["+"] = function(n)
		goto_line(editor.cursor().line + n)
	end,
	["-"] = function(n)
		goto_line(editor.cursor().line - n)
	end,
	["0"] = function()
		move_to(editor.cursor().line, 1)
	end,
	["^"] = function()
		local l = editor.cursor().line
		move_to(l, first_non_blank(l))
	end,
	["$"] = function(n)
		local l = clamp(editor.cursor().line + n - 1, 1, line_count())
		local r = line_runes(l)
		editor.set_cursor(l, max_col(r))
		-- Vim keeps the cursor glued to the line end for subsequent j/k.
		state.goal = math.huge
	end,
	["w"] = function(n)
		word_forward(n, false)
	end,
	["W"] = function(n)
		word_forward(n, true)
	end,
	["b"] = function(n)
		word_back(n, false)
	end,
	["B"] = function(n)
		word_back(n, true)
	end,
	["e"] = function(n)
		word_end(n, false, false)
	end,
	["E"] = function(n)
		word_end(n, true, false)
	end,
	["G"] = function(n, had_count)
		goto_line(had_count and n or line_count())
	end,
	["{"] = function(n)
		paragraph(-1, n)
	end,
	["}"] = function(n)
		paragraph(1, n)
	end,
	["%"] = function()
		ttt.exec_command("editor.goToMatchingBracket")
	end,
	["H"] = function(n)
		screen_line("H", n)
	end,
	["M"] = function()
		screen_line("M", 1)
	end,
	["L"] = function(n)
		screen_line("L", n)
	end,
	[";"] = function(n)
		repeat_find(n, false)
	end,
	[","] = function(n)
		repeat_find(n, true)
	end,
	["ctrl-d"] = function(n)
		scroll_page(1, n, true)
	end,
	["ctrl-u"] = function(n)
		scroll_page(-1, n, true)
	end,
	["ctrl-f"] = function(n)
		scroll_page(1, n, false)
	end,
	-- Ctrl-B is registered by core as a *force key* for sidebar.toggle, and force
	-- keys outrank the plugin interceptor by design (internal/ui/root.go), so it
	-- never reaches us unless the user rebinds sidebar.toggle. PgUp/PgDn are
	-- provided as always-available equivalents of Ctrl-B/Ctrl-F.
	["ctrl-b"] = function(n)
		scroll_page(-1, n, false)
	end,
	["pgdn"] = function(n)
		scroll_page(1, n, false)
	end,
	["pgup"] = function(n)
		scroll_page(-1, n, false)
	end,
	["ctrl-e"] = function(n)
		scroll_lines(1, n)
	end,
	["ctrl-y"] = function(n)
		scroll_lines(-1, n)
	end,
}

-- Second key of a `g`-prefixed sequence.
local G_MOTIONS = {
	["g"] = function(n, had_count)
		goto_line(had_count and n or 1)
	end,
	["_"] = function(n)
		local l = clamp(editor.cursor().line + n - 1, 1, line_count())
		move_to(l, last_non_blank(l))
	end,
	["e"] = function(n)
		word_end(n, false, true)
	end,
	["E"] = function(n)
		word_end(n, true, true)
	end,
}

-- Second key of a `z`-prefixed sequence, passed straight to reposition().
local Z_COMMANDS = { z = true, t = true, b = true }

local PREFIX_KEYS = { g = true, z = true }
local FIND_KEYS = { f = true, F = true, t = true, T = true }

-- ---------------------------------------------------------------------------
-- Mode handlers
-- ---------------------------------------------------------------------------

local function handle_insert(tok)
	if ESCAPE_TOKENS[tok] then
		set_mode("normal")
		-- Insert mode allows one-past-the-end; normal mode does not.
		clamp_cursor()
		return true
	end
	return false
end

local function take_count()
	local n = state.count
	state.count = nil
	return n or 1, n ~= nil
end

local function handle_normal(tok)
	if ESCAPE_TOKENS[tok] then
		if not has_pending() then
			return false
		end
		set_mode("normal")
		return true
	end

	-- f/F/t/T consume the very next key as their target, whatever it is.
	if state.find_pending then
		local op = state.find_pending
		state.find_pending = nil
		local n = take_count()
		if is_printable(tok) then
			state.last_find = { op = op, ch = tok }
			find_char(op, tok, n, false)
		end
		return true
	end

	if state.pending == "g" then
		state.pending = ""
		local n, had = take_count()
		local fn = G_MOTIONS[tok]
		if fn then
			fn(n, had)
		end
		return true
	end

	if state.pending == "z" then
		state.pending = ""
		state.count = nil
		if Z_COMMANDS[tok] then
			reposition(tok)
		end
		return true
	end

	-- Count prefix. A leading `0` is the motion, not a count digit.
	if #tok == 1 and tok >= "0" and tok <= "9" and not (tok == "0" and state.count == nil) then
		state.count = (state.count or 0) * 10 + tonumber(tok)
		return true
	end

	if PREFIX_KEYS[tok] then
		state.pending = tok
		return true
	end

	if FIND_KEYS[tok] then
		state.find_pending = tok
		return true
	end

	local motion = MOTIONS[tok]
	if motion then
		local n, had = take_count()
		motion(n, had)
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
