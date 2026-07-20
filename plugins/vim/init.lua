-- Vim compatibility layer for ttt.
--
-- Single-file by necessity: the plugin sandbox strips package.loaders down to
-- the preload loader (internal/plugin/sandbox.go), so a plugin cannot require
-- sibling .lua files. Sections below are delimited and appended phase by phase.
--
-- Phase 0: mode state machine (normal/insert), Esc handling, status indicator.
-- Phase 1: normal-mode motions with {count} prefixes.
-- Phase 2: insert-mode entry points and single-key edits.
-- Phase 3: operators, motion ranges and text objects.
-- Phase 4: visual, visual-line and visual-block modes.
-- Phase 5: registers, marks, macros and `.` repeat.
-- Phase 6: search, the ex command line, substitution, editor integration.

local ttt = require("ttt")
local events = require("ttt.events")
local raw_editor = require("ttt.editor")

-- ---------------------------------------------------------------------------
-- State
-- ---------------------------------------------------------------------------

local state = {
	enabled = true,
	mode = "normal", -- normal | insert | visual | visual_line | visual_block | replace
	pending = "", -- unconsumed keys, e.g. "g", "2d"
	count = nil, -- pending count prefix
	operator = nil, -- pending operator, { op = "d", count = n }
	textobj = nil, -- "i" or "a" awaiting its object key
	register = nil, -- pending "x register prefix
	await_register = false, -- `"` typed, waiting for the register name
	last_change = nil, -- replay payload for `.`
	registers = {}, -- name -> { text = s, kind = "char"|"line"|"block" }
	marks = {}, -- name -> { line = n, col = n }
	mark_pending = nil, -- "m" (set) | "`" (exact) | "'" (line) awaiting its name
	macros = {}, -- name -> array of canonical tokens
	macro = { recording = nil, keys = {}, playing = 0, last = nil },
	await_macro = nil, -- "q" (record) | "@" (replay) awaiting its register name
	insert_ctx = nil, -- per-keystroke log for the insert session in flight
	replace_stack = nil, -- overwritten characters, so `R` + backspace restores
	replaying = false, -- inside `.`, so the replay does not re-record itself
	probing = false, -- inside probe_motion, so jump marks are not disturbed
	find_pending = nil, -- "f" | "F" | "t" | "T" awaiting its target char
	last_find = nil, -- { op = "f", ch = "x" } for `;` and `,`
	goal = nil, -- sticky column for j/k
	replace_pending = nil, -- count for `r{char}` awaiting its replacement char
	last_insert = nil, -- { line, col } where insert mode was last left, for `gi`
	visual = nil, -- { al, ac, cl, cc, dollar } anchor + Vim cursor while visual
	last_visual = nil, -- the range `gv` reselects
	block_insert = false, -- blockwise I/A/c is holding multi-cursors open
	clipboard = false, -- vim.clipboard: sync the unnamed register to the system clipboard
}

-- Startup settings are read deferred, at the bottom of this file (see the
-- ttt.set_timeout call), not here: for an already-approved plugin, LoadAll calls
-- Init before the host wires the settings API, so a read at init time would
-- always see nil. state.enabled / state.clipboard keep their defaults until then.

-- ---------------------------------------------------------------------------
-- Editor shim
--
-- Marks have to survive edits, and there is no buffer-change event to hang that
-- off, so the three mutating entry points are wrapped. `editor` is a fresh table
-- that forwards everything else to the real module via __index -- the module
-- itself is left untouched, so no other plugin sees the override.
-- ---------------------------------------------------------------------------

local function count_newlines(s)
	local n = 0
	for _ in string.gmatch(s or "", "\n") do
		n = n + 1
	end
	return n
end

-- Only lines strictly below the edit move; a mark inside a deleted span
-- collapses onto the start of that span, which is what Vim does too.
local function shift_marks(after, delta, collapse_from, collapse_to, collapse_col)
	if delta == 0 and collapse_from == nil then
		return
	end
	for _, m in pairs(state.marks) do
		if m.line > after then
			m.line = m.line + delta
		elseif collapse_from and m.line > collapse_from and m.line <= collapse_to then
			m.line = collapse_from
			m.col = collapse_col
		end
	end
end

local editor = setmetatable({
	insert = function(l, c, text)
		state.marks["."] = { line = l, col = c }
		shift_marks(l, count_newlines(text))
		return raw_editor.insert(l, c, text)
	end,
	replace = function(sl, sc, el, ec, text)
		state.marks["."] = { line = sl, col = sc }
		shift_marks(el, count_newlines(text) - (el - sl), sl, el, sc)
		return raw_editor.replace(sl, sc, el, ec, text)
	end,
	set_line = function(l, text)
		state.marks["."] = { line = l, col = 1 }
		return raw_editor.set_line(l, text)
	end,
}, { __index = raw_editor })

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
		ttt.remove_status_item("macro")
		return
	end
	ttt.set_status_item("left", "mode", MODE_LABELS[state.mode] or state.mode, { priority = 10 })
	if state.macro.recording then
		ttt.set_status_item("left", "macro", "recording @" .. state.macro.recording, { priority = 11 })
	else
		ttt.remove_status_item("macro")
	end
end

local function set_mode(mode)
	state.mode = mode
	state.pending = ""
	state.count = nil
	state.operator = nil
	state.textobj = nil
	state.register = nil
	state.await_register = false
	state.await_macro = nil
	state.mark_pending = nil
	state.find_pending = nil
	state.replace_pending = nil
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
		or state.textobj ~= nil
		or state.register ~= nil
		or state.await_register
		or state.await_macro ~= nil
		or state.mark_pending ~= nil
		or state.find_pending ~= nil
		or state.replace_pending ~= nil
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
				-- A word run must stop at the line boundary: without the line
				-- check, "two" at the end of one line and "count" at the start
				-- of the next are walked over as a single word, so `w` (and
				-- therefore `dw`) skips a word whenever the next line starts on
				-- a non-blank.
				local from_line = w.l
				while class_of(w:ch(), big) == cls do
					if not w:fwd() then
						break
					end
					if w.l ~= from_line then
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
-- Marks
--
-- state.marks maps a name to { line, col }. Line numbers are kept correct
-- across edits by the editor shim at the top of this file. Three names are
-- special and never set by `m`: `'` is the position before the last jump (what
-- `''` and ``` `` ``` return to), `.` is the position of the last change (set by
-- the shim), and backtick reads the same slot as `'`.
-- ---------------------------------------------------------------------------

local function mark_slot(name)
	if name == "`" or name == "'" then
		return "'"
	end
	return name
end

local function set_jump_mark()
	if state.probing then
		return
	end
	local cur = editor.cursor()
	state.marks["'"] = { line = cur.line, col = cur.col }
end

local function mark_set(name, line, col)
	state.marks[mark_slot(name)] = { line = line, col = col }
end

-- Where a mark points, clamped onto the buffer as it is now. `linewise` is the
-- `'{mark}` form, which lands on the first non-blank of the line.
local function mark_target(name, linewise)
	local m = state.marks[mark_slot(name)]
	if not m then
		return nil
	end
	local l = clamp(m.line, 1, line_count())
	if linewise then
		return l, first_non_blank(l)
	end
	return l, clamp(m.col, 1, max_col(line_runes(l)))
end

local function goto_mark(name, linewise)
	local l, c = mark_target(name, linewise)
	if not l then
		return
	end
	set_jump_mark()
	move_to(l, c)
end

local MARK_NAME = "^[%a'`%.]$"

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
		set_jump_mark()
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
		set_jump_mark()
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

local PREFIX_KEYS = { g = true, z = true, ["["] = true, ["]"] = true }
local FIND_KEYS = { f = true, F = true, t = true, T = true }

-- ---------------------------------------------------------------------------
-- Edits
--
-- UNDO CONTRACT: every Vim operation is exactly one undo step. Each edit is
-- bracketed by begin_undo_group()/end_undo_group() so `3x` undoes as one `u`,
-- not three. Undo transactions do NOT nest -- BeginTransaction resets the
-- transaction start index (internal/core/undo/undo.go) -- so an operation must
-- call begin exactly once. Commands that edit *and then* enter insert mode open
-- the group here and let leave_insert() close it, which is what makes
-- `cwfoo<Esc>` a single undo the way Vim does it.
-- ---------------------------------------------------------------------------

-- Vim's 'shiftwidth'. Hardcoded until Phase 7 wires plugin settings; reading
-- editor.tabSize would need a `settings` permission the manifest does not ask
-- for, and re-prompting installed users for it is not worth it yet.
local SHIFTWIDTH = 4

local function line_len(l)
	return #line_runes(l)
end

-- ---------------------------------------------------------------------------
-- The change log
--
-- `.` replays a *resolved payload*, not keystrokes: the operator, its target,
-- the count, the register and (for anything that ends in insert mode) the text
-- that was typed. The typed text is the one part that cannot be known when the
-- command starts, so an insert session carries a per-keystroke log
-- (state.insert_ctx) that is folded into the payload when Esc is pressed. That
-- same log is what makes `3i` repeat its text and backspace in `R` restore the
-- character it overwrote.
-- ---------------------------------------------------------------------------

-- A nil payload means "this command is not repeatable"; the previous `.` target
-- is kept rather than cleared, so a stray yank or visual operator does not
-- silently disarm `.`.
local function record_change(payload)
	if state.replaying or payload == nil then
		return
	end
	state.last_change = payload
end

-- One typed key as the text it produced, or nil when it is not something that
-- can be replayed as plain text (arrows, Ctrl chords, ...).
local function token_text(t)
	if t == "enter" then
		return "\n"
	end
	if t == "tab" then
		return "\t"
	end
	local b = t:byte(1)
	if b == nil then
		return nil
	end
	if #t == 1 then
		if b >= 0x20 and b <= 0x7e then
			return t
		end
		return nil
	end
	if b >= 0xc0 then
		return t
	end
	return nil
end

-- Insert `text` at (l, c) and return the position just past it.
local function insert_and_advance(l, c, text)
	editor.insert(l, c, text)
	local k = count_newlines(text)
	if k == 0 then
		return l, c + #runes_of(text)
	end
	return l + k, #runes_of(text:match("[^\n]*$") or "") + 1
end

local function insert_text_of(ctx)
	if not ctx or ctx.dirty then
		return nil
	end
	return table.concat(ctx.keys)
end

-- `{count}i`, `{count}o`, `{count}a`: Vim types the text once and repeats it
-- count-1 more times when insert mode is left. `o`/`O` repeat onto fresh lines.
local function repeat_insert(ctx, text)
	local cur = editor.cursor()
	local l, c = cur.line, cur.col
	local open_line = (ctx.tok == "o" or ctx.tok == "O")
	for _ = 2, ctx.count do
		if open_line then
			l, c = insert_and_advance(l, line_len(l) + 1, "\n" .. text)
		else
			l, c = insert_and_advance(l, c, text)
		end
	end
	editor.set_cursor(l, c)
end

-- The caller must already have opened the undo group. `ctx` describes how the
-- session was entered so it can be counted and repeated; pass nil for sessions
-- that are neither (blockwise multi-cursor insert).
local function begin_insert(ctx)
	set_mode("insert")
	if ctx then
		ctx.keys = {}
		ctx.count = ctx.count or 1
		state.insert_ctx = ctx
	end
	state.replace_stack = {}
end

-- Leaving insert/replace mode. Vim steps the cursor one column left and drops
-- it back onto a character; the position *before* that step is what `gi`
-- resumes from.
local function leave_insert()
	local ctx = state.insert_ctx
	state.insert_ctx = nil
	state.replace_stack = nil

	-- Blockwise I/A/c parked a cursor on every row; collapse them before reading
	-- the cursor back, so the position recorded for `gi` is the primary one.
	if state.block_insert then
		state.block_insert = false
		editor.clear_cursors()
		ctx = nil
	end

	if ctx then
		local text = insert_text_of(ctx)
		if text and text ~= "" and ctx.count > 1 then
			repeat_insert(ctx, text)
		end
		if ctx.change then
			ctx.change.text = text
			record_change(text and ctx.change or nil)
		end
	end

	local cur = editor.cursor()
	state.last_insert = { line = cur.line, col = cur.col }
	set_mode("normal")
	if cur.col > 1 then
		editor.set_cursor(cur.line, cur.col - 1)
	end
	clamp_cursor()
	editor.end_undo_group()
end

-- ---------------------------------------------------------------------------
-- Text extraction and registers
--
-- A register entry is { text, kind }, kind being "char", "line" or "block".
-- Linewise text always carries its trailing newline so a paste can be replayed
-- verbatim; blockwise text is the rows joined by newlines, and only the kind
-- distinguishes it from a multi-line charwise yank.
--
-- Names: `"` unnamed, `"0` last yank, `"1`-`"9` the delete ring, `"-` small
-- delete, `"a`-`"z` named (uppercase appends), `"_` blackhole, `"+`/`"*` the
-- system clipboard. See the README for what the clipboard registers can and
-- cannot do -- there is no clipboard binding in the plugin Lua API, so they are
-- routed through the `editor.copy` / `editor.paste` commands.
-- ---------------------------------------------------------------------------

-- table.concat errors when j runs past the array, so clamp both ends here
-- rather than at every call site.
local function sub_runes(r, i, j)
	i = math.max(1, i)
	j = math.min(j, #r)
	if i > j then
		return ""
	end
	return table.concat(r, "", i, j)
end

-- Charwise text for [sl,sc] .. [el,ec) -- ec is an exclusive end column, the
-- same convention editor.replace uses.
local function charwise_text(sl, sc, el, ec)
	if sl == el then
		return sub_runes(line_runes(sl), sc, ec - 1)
	end
	local parts = { sub_runes(line_runes(sl), sc, math.huge) }
	for l = sl + 1, el - 1 do
		parts[#parts + 1] = editor.get_line(l) or ""
	end
	parts[#parts + 1] = sub_runes(line_runes(el), 1, ec - 1)
	return table.concat(parts, "\n")
end

local function linewise_text(sl, el)
	local parts = {}
	for l = sl, el do
		parts[#parts + 1] = editor.get_line(l) or ""
	end
	return table.concat(parts, "\n") .. "\n"
end

local CLIPBOARD_REGISTERS = { ["+"] = true, ["*"] = true }

local function take_register()
	local r = state.register
	state.register = nil
	return r
end

-- There is no clipboard binding in the plugin Lua API, so `"+y` borrows
-- editor.copy: select the range, copy, drop the selection, put the cursor back.
-- set_selection parks the cursor at the end of the range, which is exactly what
-- Selection.Text reads, so `ec` stays the usual exclusive end column.
local function clipboard_copy(sl, sc, el, ec)
	local cur = editor.cursor()
	local sel = editor.selection()
	editor.set_selection(sl, sc, el, ec)
	ttt.exec_command("editor.copy")
	editor.clear_selection()
	editor.set_cursor(cur.line, cur.col)
	if sel and sel.active then
		editor.set_selection(sel.start_line, sel.start_col, cur.line, cur.col)
	end
end

local function reg_store(name, text, kind)
	if name == "_" then
		return
	end
	local upper = name:match("^(%u)$")
	if upper then
		local lower = upper:lower()
		local prev = state.registers[lower]
		if prev then
			local joined
			if prev.kind == "line" then
				joined = prev.text .. text
				if kind ~= "line" then
					joined = joined .. "\n"
				end
			elseif kind == "line" then
				joined = prev.text .. "\n" .. text
			else
				joined = prev.text .. text
			end
			state.registers[lower] = { text = joined, kind = prev.kind == "line" and "line" or kind }
		else
			state.registers[lower] = { text = text, kind = kind }
		end
		state.registers['"'] = state.registers[lower]
		return
	end
	state.registers[name] = { text = text, kind = kind }
	state.registers['"'] = state.registers[name]
end

-- Shift "1-"9 down one slot and drop the new text into "1.
local function shift_delete_ring(text, kind)
	for i = 9, 2, -1 do
		state.registers[tostring(i)] = state.registers[tostring(i - 1)]
	end
	state.registers["1"] = { text = text, kind = kind }
end

-- The single write path for every yank and delete. `range` is optional and only
-- used by the clipboard registers, which need buffer coordinates rather than
-- text. Vim's defaults: a yank fills "0, a multi-line delete rotates the ring,
-- a small delete fills "-, and everything also lands in the unnamed register.
local function set_register(text, kind, is_delete, range)
	local name = take_register()

	if name and CLIPBOARD_REGISTERS[name] then
		state.registers[name] = { text = text, kind = kind }
		state.registers['"'] = state.registers[name]
		if range then
			clipboard_copy(range[1], range[2], range[3], range[4])
		end
		return
	end

	if name then
		reg_store(name, text, kind)
		return
	end

	-- vim.clipboard: an unnamed yank/delete also lands on the system clipboard,
	-- reusing the same editor.copy path as the "+/"* registers.
	if state.clipboard and range then
		clipboard_copy(range[1], range[2], range[3], range[4])
	end

	if is_delete then
		if kind == "line" or text:find("\n", 1, true) then
			shift_delete_ring(text, kind)
			state.registers['"'] = state.registers["1"]
		else
			state.registers["-"] = { text = text, kind = kind }
			state.registers['"'] = state.registers["-"]
		end
		return
	end

	state.registers["0"] = { text = text, kind = kind }
	state.registers['"'] = state.registers["0"]
end

-- Resolve a register for reading. Uppercase reads the lowercase slot.
local function get_register(name)
	name = name or '"'
	local upper = name:match("^(%u)$")
	if upper then
		name = upper:lower()
	end
	return state.registers[name]
end

local function yank_charwise(sl, sc, el, ec, is_delete)
	set_register(charwise_text(sl, sc, el, ec), "char", is_delete, { sl, sc, el, ec })
end

local function yank_linewise(sl, el, is_delete)
	local n = line_count()
	local ec_line = math.min(el, n)
	local range
	if ec_line < n then
		range = { sl, 1, ec_line + 1, 1 }
	else
		range = { sl, 1, ec_line, line_len(ec_line) + 1 }
	end
	set_register(linewise_text(sl, el), "line", is_delete, range)
end

-- Delete whole lines sl..el. Three cases, because the newline that has to go
-- with the text lives *after* the block normally, but *before* it when the
-- block runs to the end of the buffer -- and deleting every line has to leave
-- one empty line behind rather than an empty buffer.
local function delete_lines(sl, el)
	local n = line_count()
	sl = clamp(sl, 1, n)
	el = clamp(el, sl, n)
	if sl <= 1 and el >= n then
		editor.replace(1, 1, n, line_len(n) + 1, "")
	elseif el >= n then
		editor.replace(sl - 1, line_len(sl - 1) + 1, el, line_len(el) + 1, "")
	else
		editor.replace(sl, 1, el + 1, 1, "")
	end
end

-- Linewise change (`cc`, `S`, `c` with a linewise motion): collapse lines
-- sl..el to a single line that keeps sl's indent, then enter insert mode. The
-- caller has already opened the undo group; leave_insert() closes it.
local function change_lines(sl, el, ctx)
	local n = line_count()
	sl = clamp(sl, 1, n)
	el = clamp(el, sl, n)
	local indent = (editor.get_line(sl) or ""):match("^[ \t]*") or ""
	yank_linewise(sl, el, true)
	editor.replace(sl, 1, el, line_len(el) + 1, indent)
	editor.set_cursor(sl, #runes_of(indent) + 1)
	begin_insert(ctx)
end

-- Delete from the cursor to the end of line (cur.line + n - 1), joining the
-- lines in between. `2D` on "aaa"/"bbb" leaves a single empty line, as in Vim.
local function delete_to_end(n)
	local cur = editor.cursor()
	local last = clamp(cur.line + n - 1, 1, line_count())
	yank_charwise(cur.line, cur.col, last, line_len(last) + 1, true)
	editor.replace(cur.line, cur.col, last, line_len(last) + 1, "")
end

-- p / P. A linewise register lands on a new line below/above; a charwise one
-- lands after/before the cursor. Multi-line charwise text leaves the cursor at
-- the first pasted character, single-line text on its last, both as in Vim.
local function split_lines(s)
	local out = {}
	local from = 1
	while true do
		local at = s:find("\n", from, true)
		if not at then
			out[#out + 1] = s:sub(from)
			break
		end
		out[#out + 1] = s:sub(from, at - 1)
		from = at + 1
	end
	return out
end

-- Blockwise paste re-inserts a rectangle: row i of the register goes onto line
-- cursor+i-1 at the paste column, padding short lines with spaces and appending
-- new lines when the block runs past the end of the buffer. `count` repeats each
-- row horizontally, as in Vim.
local function paste_block(reg, count, after)
	local cur = editor.cursor()
	local rows = split_lines(reg.text)
	local col = cur.col
	if after and line_len(cur.line) > 0 then
		col = math.min(cur.col + 1, line_len(cur.line) + 1)
	end
	editor.begin_undo_group()
	for i = 1, #rows do
		local l = cur.line + i - 1
		while line_count() < l do
			local n = line_count()
			editor.insert(n, line_len(n) + 1, "\n")
		end
		local chunk = rows[i]:rep(count)
		local len = line_len(l)
		if len < col - 1 then
			editor.insert(l, len + 1, string.rep(" ", col - 1 - len) .. chunk)
		else
			editor.insert(l, col, chunk)
		end
	end
	editor.end_undo_group()
	move_to(cur.line, col)
end

-- `"+p` / `"*p`. The clipboard cannot be read from Lua, so this positions the
-- cursor and delegates to the core paste command. Consequences: the register
-- kind is whatever core makes of the text (there is no linewise flag to carry),
-- and each repetition is its own undo step because core opens its own
-- transaction and undo groups do not nest.
local function paste_clipboard(count, after)
	local cur = editor.cursor()
	local col = cur.col
	if after and line_len(cur.line) > 0 then
		col = math.min(cur.col + 1, line_len(cur.line) + 1)
	end
	editor.set_cursor(cur.line, col)
	for _ = 1, count do
		ttt.exec_command("editor.paste")
	end
	clamp_cursor()
end

local function paste(count, after)
	local name = take_register()
	if name and CLIPBOARD_REGISTERS[name] then
		paste_clipboard(count, after)
		return
	end
	local reg = get_register(name)
	if not reg or reg.text == "" then
		return
	end
	if reg.kind == "block" then
		paste_block(reg, count, after)
		return
	end
	local cur = editor.cursor()
	editor.begin_undo_group()
	if reg.kind == "line" then
		local body = (reg.text:gsub("\n$", ""))
		local chunk = body
		for _ = 2, count do
			chunk = chunk .. "\n" .. body
		end
		if after then
			editor.insert(cur.line, line_len(cur.line) + 1, "\n" .. chunk)
			move_to(cur.line + 1, first_non_blank(cur.line + 1))
		else
			editor.insert(cur.line, 1, chunk .. "\n")
			move_to(cur.line, first_non_blank(cur.line))
		end
	else
		local chunk = reg.text:rep(count)
		local col = cur.col
		if after and line_len(cur.line) > 0 then
			col = math.min(cur.col + 1, line_len(cur.line) + 1)
		end
		editor.insert(cur.line, col, chunk)
		if chunk:find("\n", 1, true) then
			editor.set_cursor(cur.line, col)
		else
			editor.set_cursor(cur.line, col + #runes_of(chunk) - 1)
		end
		clamp_cursor()
	end
	editor.end_undo_group()
end

local function swap_case(ch)
	local b = ch:byte(1)
	if b == nil then
		return ch
	end
	if b >= 0x61 and b <= 0x7a then
		return string.char(b - 32)
	end
	if b >= 0x41 and b <= 0x5a then
		return string.char(b + 32)
	end
	return ch
end

-- J / gJ. `count` is the number of lines to end up joined, so it performs
-- count-1 joins (and `J` alone, count 1, still performs one).
local function join(count, with_space)
	local l = editor.cursor().line
	local times = math.max(1, count - 1)
	local col = 1
	editor.begin_undo_group()
	for _ = 1, times do
		if l >= line_count() then
			break
		end
		local cur_r = line_runes(l)
		local nxt_r = line_runes(l + 1)
		local end_col, sep = 1, ""
		if with_space then
			local fnb = 1
			while fnb <= #nxt_r and is_blank(nxt_r[fnb]) do
				fnb = fnb + 1
			end
			end_col = fnb
			-- Vim adds no space when the joined-to line already ends in
			-- whitespace, when it is empty, or when the next line starts with
			-- a closing paren.
			if fnb <= #nxt_r and #cur_r > 0 and not is_blank(cur_r[#cur_r]) and nxt_r[fnb] ~= ")" then
				sep = " "
			end
		end
		editor.replace(l, #cur_r + 1, l + 1, end_col, sep)
		col = #cur_r + 1
	end
	editor.end_undo_group()
	editor.set_cursor(l, col)
	clamp_cursor()
end

-- >> and <<. Blank lines are left alone, as in Vim. The caller owns the undo
-- group so `>` as an operator and `>>` as a doubled key share one code path.
local function indent_range(first, last)
	return clamp(first, 1, line_count()), clamp(last, 1, line_count())
end

local function indent_lines_range(sl, el, dir)
	local first, last = indent_range(sl, el)
	for l = first, last do
		local r = line_runes(l)
		if #r > 0 then
			if dir > 0 then
				editor.insert(l, 1, string.rep(" ", SHIFTWIDTH))
			else
				local k = 0
				while k < SHIFTWIDTH do
					local ch = r[k + 1]
					if ch == "\t" then
						k = k + 1
						break
					elseif ch == " " then
						k = k + 1
					else
						break
					end
				end
				if k > 0 then
					editor.replace(l, 1, l, k + 1, "")
				end
			end
		end
	end
end

-- == reindent. ttt has no indent engine, so this is a deliberate heuristic:
-- copy the previous non-blank line's indent, add one shiftwidth if that line
-- opens a block, and remove one if this line closes one. Spaces only.
local function reindent_range(sl, el)
	local first, last = indent_range(sl, el)
	for l = first, last do
		local body = (editor.get_line(l) or ""):match("^[ \t]*(.-)[ \t]*$") or ""
		if body ~= "" then
			local width = 0
			local p = l - 1
			while p >= 1 do
				local prev = editor.get_line(p) or ""
				local pbody = prev:match("^[ \t]*(.-)[ \t]*$") or ""
				if pbody ~= "" then
					width = #(prev:match("^ *") or "")
					if pbody:match("[%{%(%[]$") or pbody:match(":$") then
						width = width + SHIFTWIDTH
					end
					break
				end
				p = p - 1
			end
			if body:match("^[%}%)%]]") then
				width = math.max(0, width - SHIFTWIDTH)
			end
			editor.set_line(l, string.rep(" ", width) .. body)
		end
	end
end

-- Ctrl-A / Ctrl-X. Finds the first digit run ending at or after the cursor,
-- takes an immediately preceding "-" as a sign, and preserves zero padding.
-- byte_to_col/col_to_byte bridge Lua's byte-oriented patterns to the editor's
-- rune columns.
local function bump(count, dir)
	local cur = editor.cursor()
	local text = editor.get_line(cur.line) or ""
	local bcur = editor.col_to_byte(text, cur.col)

	local s, e
	local from = 1
	while true do
		local a, b = text:find("%d+", from)
		if not a then
			break
		end
		if b >= bcur then
			s, e = a, b
			break
		end
		from = b + 1
	end
	if not s then
		return
	end

	local digits = text:sub(s, e)
	local start, neg = s, false
	if s > 1 and text:sub(s - 1, s - 1) == "-" then
		start, neg = s - 1, true
	end

	local val = tonumber(digits)
	if neg then
		val = -val
	end
	val = val + dir * count

	local body = string.format("%d", math.abs(val))
	if digits:sub(1, 1) == "0" and #digits > 1 and #body < #digits then
		body = string.rep("0", #digits - #body) .. body
	end
	local out = (val < 0 and "-" or "") .. body

	local scol = editor.byte_to_col(text, start)
	editor.begin_undo_group()
	editor.replace(cur.line, scol, cur.line, editor.byte_to_col(text, e + 1), out)
	editor.end_undo_group()
	editor.set_cursor(cur.line, scol + #out - 1)
end

-- Context for an insert session opened by a normal-mode command. `text_count`
-- is how many times the typed text is repeated on Esc (`3i`), `cmd_count` is
-- the count `.` should re-run the command with (`3s` deletes three characters
-- but types its text once).
local function entry_ctx(tok, cmd_count, text_count)
	return {
		tok = tok,
		count = text_count or 1,
		change = { kind = "insert", tok = tok, count = cmd_count or 1 },
	}
end

-- Single-key normal-mode edits. Called as fn(count, had_count).
local EDITS = {
	["i"] = function(n)
		editor.begin_undo_group()
		begin_insert(entry_ctx("i", n, n))
	end,
	["I"] = function(n)
		local l = editor.cursor().line
		move_to(l, first_non_blank(l))
		editor.begin_undo_group()
		begin_insert(entry_ctx("I", n, n))
	end,
	["a"] = function(n)
		local cur = editor.cursor()
		editor.set_cursor(cur.line, math.min(cur.col + 1, line_len(cur.line) + 1))
		editor.begin_undo_group()
		begin_insert(entry_ctx("a", n, n))
	end,
	["A"] = function(n)
		local cur = editor.cursor()
		editor.set_cursor(cur.line, line_len(cur.line) + 1)
		editor.begin_undo_group()
		begin_insert(entry_ctx("A", n, n))
	end,
	-- Insert rejects line >= #Lines (internal/app/plugin_api.go), so `o` on the
	-- last line appends the newline to the *end* of that line rather than
	-- addressing the line after it.
	["o"] = function(n)
		local l = editor.cursor().line
		editor.begin_undo_group()
		editor.insert(l, line_len(l) + 1, "\n")
		editor.set_cursor(l + 1, 1)
		begin_insert(entry_ctx("o", n, n))
	end,
	["O"] = function(n)
		local l = editor.cursor().line
		editor.begin_undo_group()
		editor.insert(l, 1, "\n")
		editor.set_cursor(l, 1)
		begin_insert(entry_ctx("O", n, n))
	end,

	["x"] = function(n)
		local cur = editor.cursor()
		local last = math.min(cur.col + n, line_len(cur.line) + 1)
		if last <= cur.col then
			return
		end
		editor.begin_undo_group()
		yank_charwise(cur.line, cur.col, cur.line, last, true)
		editor.replace(cur.line, cur.col, cur.line, last, "")
		editor.end_undo_group()
		clamp_cursor()
	end,
	["X"] = function(n)
		local cur = editor.cursor()
		local start = math.max(1, cur.col - n)
		if start >= cur.col then
			return
		end
		editor.begin_undo_group()
		yank_charwise(cur.line, start, cur.line, cur.col, true)
		editor.replace(cur.line, start, cur.line, cur.col, "")
		editor.end_undo_group()
		editor.set_cursor(cur.line, start)
	end,
	["D"] = function(n)
		editor.begin_undo_group()
		delete_to_end(n)
		editor.end_undo_group()
		clamp_cursor()
	end,
	["C"] = function(n)
		editor.begin_undo_group()
		delete_to_end(n)
		begin_insert(entry_ctx("C", n, 1))
	end,
	["s"] = function(n)
		local cur = editor.cursor()
		local last = math.min(cur.col + n, line_len(cur.line) + 1)
		editor.begin_undo_group()
		if last > cur.col then
			yank_charwise(cur.line, cur.col, cur.line, last, true)
			editor.replace(cur.line, cur.col, cur.line, last, "")
		end
		begin_insert(entry_ctx("s", n, 1))
	end,
	["S"] = function(n)
		local cur = editor.cursor()
		editor.begin_undo_group()
		change_lines(cur.line, cur.line + n - 1, entry_ctx("S", n, 1))
	end,
	-- Vim's `Y` is a synonym for `yy`; ttt follows Neovim's default of `y$`,
	-- which is what the other shorthands (`D`, `C`) do.
	["Y"] = function(n)
		local cur = editor.cursor()
		local last = clamp(cur.line + n - 1, 1, line_count())
		yank_charwise(cur.line, cur.col, last, line_len(last) + 1)
	end,
	-- r consumes the next key; the count rides along in replace_pending.
	["r"] = function(n)
		state.replace_pending = n
	end,
	["R"] = function()
		editor.begin_undo_group()
		begin_insert({ tok = "R", count = 1, change = { kind = "replace_mode" } })
		set_mode("replace")
	end,

	["~"] = function(n)
		local cur = editor.cursor()
		local r = line_runes(cur.line)
		if #r == 0 then
			return
		end
		local last = math.min(cur.col + n - 1, #r)
		local out = {}
		for i = cur.col, last do
			out[#out + 1] = swap_case(r[i])
		end
		editor.begin_undo_group()
		editor.replace(cur.line, cur.col, cur.line, last + 1, table.concat(out))
		editor.end_undo_group()
		editor.set_cursor(cur.line, math.min(last + 1, max_col(r)))
	end,
	["J"] = function(n)
		join(n, true)
	end,
	["p"] = function(n)
		paste(n, true)
	end,
	["P"] = function(n)
		paste(n, false)
	end,

	["u"] = function(n)
		for _ = 1, n do
			ttt.exec_command("editor.undo")
		end
		clamp_cursor()
	end,
	["ctrl-r"] = function(n)
		for _ = 1, n do
			ttt.exec_command("editor.redo")
		end
		clamp_cursor()
	end,

	["ctrl-a"] = function(n)
		bump(n, 1)
	end,
	["ctrl-x"] = function(n)
		bump(n, -1)
	end,
}

-- ---------------------------------------------------------------------------
-- Operators
--
-- The full grammar is {count}{operator}{count}{motion|textobject}, with the two
-- counts multiplying (`2d3w` deletes six words). Every operator ends up calling
-- apply_operator() with a normalized range plus a linewise flag, so the doubled
-- forms (`dd`, `>>`, `gugu`) and the motion forms share one code path.
--
-- Motion targets are resolved by *running the Phase 1 motion* and reading the
-- cursor it lands on, then putting the cursor back. That keeps one
-- implementation of every motion instead of a parallel "where would this go"
-- table that would inevitably drift.
-- ---------------------------------------------------------------------------

-- Exclusive vs inclusive is the single most common source of off-by-one bugs in
-- Vim emulation: `dw` must not delete the character it lands on, `de` must.
-- Anything not listed here is exclusive, which is Vim's default.
local MOTION_KIND = {
	["j"] = "linewise",
	["k"] = "linewise",
	["+"] = "linewise",
	["-"] = "linewise",
	["G"] = "linewise",
	["H"] = "linewise",
	["M"] = "linewise",
	["L"] = "linewise",
	["ctrl-d"] = "linewise",
	["ctrl-u"] = "linewise",
	["ctrl-f"] = "linewise",
	["ctrl-b"] = "linewise",
	["pgup"] = "linewise",
	["pgdn"] = "linewise",
	["e"] = "inclusive",
	["E"] = "inclusive",
	["$"] = "inclusive",
	["%"] = "inclusive",
}

local G_MOTION_KIND = {
	["g"] = "linewise",
	["_"] = "inclusive",
	["e"] = "inclusive",
	["E"] = "inclusive",
}

local OPERATORS = { d = true, c = true, y = true, [">"] = true, ["<"] = true, ["="] = true }
local G_OPERATORS = { u = "gu", U = "gU", ["~"] = "g~" }
local CASE_OPERATORS = { gu = "lower", gU = "upper", ["g~"] = "swap" }

for _, op in pairs(G_OPERATORS) do
	OPERATORS[op] = true
end

-- The token that, typed straight after the operator, makes it linewise. Vim
-- accepts both `gugu` and the shorthand `guu`; the `g` of `gugu` is consumed by
-- the normal g-prefix path and lands here as plain `u`, so one entry covers it.
local OP_DOUBLE = {
	["d"] = { ["d"] = true },
	["c"] = { ["c"] = true },
	["y"] = { ["y"] = true },
	[">"] = { [">"] = true },
	["<"] = { ["<"] = true },
	["="] = { ["="] = true },
	["gu"] = { ["u"] = true },
	["gU"] = { ["U"] = true },
	["g~"] = { ["~"] = true },
}

local function transform_case(text, how)
	local out = {}
	for _, ch in ipairs(runes_of(text)) do
		if how == "upper" then
			out[#out + 1] = ch:upper()
		elseif how == "lower" then
			out[#out + 1] = ch:lower()
		else
			out[#out + 1] = swap_case(ch)
		end
	end
	return table.concat(out)
end

-- The heart of Phase 3. `ec` is an exclusive end column, matching
-- editor.replace. Only `c` leaves the undo group open, because it hands off to
-- insert mode and leave_insert() closes it -- that is what makes `cwfoo<Esc>`
-- one undo step.
local function apply_operator(op, sl, sc, el, ec, linewise, payload)
	local n = line_count()
	sl = clamp(sl, 1, n)
	el = clamp(el, 1, n)

	-- `c` hands off to insert mode, so its payload is only complete once the
	-- typed text is known; leave_insert() records it. Everything else is done
	-- editing by the time this returns.
	if op ~= "c" then
		record_change(payload)
	end

	if op == "y" then
		if linewise then
			local col = editor.cursor().col
			yank_linewise(sl, el)
			move_to(sl, col)
		else
			yank_charwise(sl, sc, el, ec)
			move_to(sl, sc)
		end
		return
	end

	editor.begin_undo_group()

	if op == "d" then
		if linewise then
			yank_linewise(sl, el, true)
			delete_lines(sl, el)
			local l = math.min(sl, line_count())
			move_to(l, first_non_blank(l))
		else
			yank_charwise(sl, sc, el, ec, true)
			editor.replace(sl, sc, el, ec, "")
			editor.set_cursor(sl, sc)
			clamp_cursor()
		end
		editor.end_undo_group()
		return
	end

	if op == "c" then
		local ctx = { tok = "c", count = 1, change = payload }
		if linewise then
			change_lines(sl, el, ctx)
		else
			yank_charwise(sl, sc, el, ec, true)
			editor.replace(sl, sc, el, ec, "")
			editor.set_cursor(sl, sc)
			begin_insert(ctx)
		end
		-- Undo group deliberately left open for leave_insert().
		return
	end

	if op == ">" or op == "<" then
		indent_lines_range(sl, el, op == ">" and 1 or -1)
		editor.end_undo_group()
		move_to(sl, first_non_blank(sl))
		return
	end

	if op == "=" then
		reindent_range(sl, el)
		editor.end_undo_group()
		move_to(sl, first_non_blank(sl))
		return
	end

	local how = CASE_OPERATORS[op]
	if how then
		if linewise then
			sc, ec = 1, line_len(el) + 1
		end
		local text = charwise_text(sl, sc, el, ec)
		if text ~= "" then
			editor.replace(sl, sc, el, ec, transform_case(text, how))
		end
		editor.end_undo_group()
		move_to(sl, sc)
		return
	end

	editor.end_undo_group()
end

-- ---------------------------------------------------------------------------
-- Motion targets
-- ---------------------------------------------------------------------------

-- Run a motion for its cursor position and put the cursor back. state.goal is
-- saved too: `$` sets it to infinity, which would otherwise leak into the next
-- j/k after the operator.
local function probe_motion(fn, n, had_count)
	local start = editor.cursor()
	local saved_goal = state.goal
	state.probing = true
	fn(n, had_count)
	state.probing = false
	local dest = editor.cursor()
	editor.set_cursor(start.line, start.col)
	state.goal = saved_goal
	return start, dest
end

-- Vim: "When using the `w` motion with an operator and the last word moved over
-- is at the end of a line, the end of that word becomes the end of the operated
-- text." Without this, `dw` on the last word of a line eats the newline and the
-- next line's indent.
local function clip_word_motion(start, dest)
	if dest.line <= start.line then
		return dest
	end
	local w = walker(dest.line, dest.col)
	while w:back() do
		local ch = w:ch()
		if ch ~= nil and not is_blank(ch) then
			break
		end
	end
	if w.l < dest.line and (w.l > start.line or (w.l == start.line and w.c >= start.col)) then
		return { line = w.l, col = line_len(w.l) + 1 }
	end
	return dest
end

-- Turn a motion result into an operator range. Returns sl, sc, el, ec, linewise
-- or nil when the motion produced nothing to operate on.
local function motion_to_range(start, dest, kind)
	-- NOTE: gopher-lua does NOT evaluate multiple assignment simultaneously when
	-- a target also appears on the right -- `a, b = b, a` yields `b, b`. Every
	-- swap below therefore goes through explicit temporaries. See README.
	if kind == "linewise" then
		local a, b = start.line, dest.line
		if a > b then
			local t = a
			a = b
			b = t
		end
		return a, 1, b, 1, true
	end

	local sl, sc, el, ec = start.line, start.col, dest.line, dest.col
	local backward = (el < sl) or (el == sl and ec < sc)
	if backward then
		local tl, tc = sl, sc
		sl = el
		sc = ec
		el = tl
		ec = tc
	end

	if kind == "inclusive" then
		ec = ec + 1
	else
		-- Vim's two exclusive-motion adjustments, both keyed off the *original*
		-- end landing in column 1.
		if not backward and el > sl and ec == 1 then
			local prev = el - 1
			if sc <= first_non_blank(sl) and prev >= sl then
				return sl, 1, prev, 1, true
			end
			el, ec = prev, line_len(prev) + 1
		end
	end

	if sl == el and ec <= sc then
		return nil
	end
	return sl, sc, el, ec, false
end

-- ---------------------------------------------------------------------------
-- Text objects
--
-- Each returns sl, sc, el, ec (exclusive end column), linewise -- the same
-- shape apply_operator consumes -- or nil when there is no object here.
-- ---------------------------------------------------------------------------

-- Flatten lines sl..el into a rune array plus an index -> {line, col} map, so
-- objects that span lines (tags, sentences) can be found with flat scanning.
local function flatten(sl, el)
	local runes, map = {}, {}
	for l = sl, el do
		local r = line_runes(l)
		for i = 1, #r do
			runes[#runes + 1] = r[i]
			map[#map + 1] = { l, i }
		end
		if l < el then
			runes[#runes + 1] = "\n"
			map[#map + 1] = { l, #r + 1 }
		end
	end
	return runes, map
end

local function flat_offset(map, line, col)
	for i = 1, #map do
		if map[i][1] == line and map[i][2] >= col then
			return i
		end
		if map[i][1] > line then
			return i
		end
	end
	return #map
end

-- iw/aw/iW/aW. Vim counts *chunks* for `iw` (a whitespace run is a chunk of its
-- own) and whole words-with-their-whitespace for `aw`.
local function textobj_word(inner, big, count)
	local cur = editor.cursor()
	local r = line_runes(cur.line)
	if #r == 0 then
		return cur.line, 1, cur.line, 1, false
	end
	local i = math.min(cur.col, #r)
	local cls = class_of(r[i], big)
	local s, e = i, i
	while s > 1 and class_of(r[s - 1], big) == cls do
		s = s - 1
	end
	while e < #r and class_of(r[e + 1], big) == cls do
		e = e + 1
	end

	local function extend_chunk(from)
		if from >= #r then
			return from
		end
		local c2 = class_of(r[from + 1], big)
		local j = from + 1
		while j < #r and class_of(r[j + 1], big) == c2 do
			j = j + 1
		end
		return j
	end

	if inner then
		for _ = 2, count do
			e = extend_chunk(e)
		end
		return cur.line, s, cur.line, e + 1, false
	end

	for _ = 2, count do
		while e < #r and is_blank(r[e + 1]) do
			e = e + 1
		end
		e = extend_chunk(e)
	end

	local had_trailing = false
	while e < #r and is_blank(r[e + 1]) do
		e = e + 1
		had_trailing = true
	end
	if not had_trailing then
		while s > 1 and is_blank(r[s - 1]) do
			s = s - 1
		end
	end
	return cur.line, s, cur.line, e + 1, false
end

-- i"/a" and friends. Vim pairs quotes from the start of the line rather than
-- looking for the nearest one, and searches forward when the cursor sits before
-- a pair -- both reproduced here. Quotes are line-scoped in Vim too.
local function textobj_quote(inner, q)
	local cur = editor.cursor()
	local r = line_runes(cur.line)
	local qs = {}
	for i = 1, #r do
		if r[i] == q and (i == 1 or r[i - 1] ~= "\\") then
			qs[#qs + 1] = i
		end
	end
	for k = 1, #qs - 1, 2 do
		local a, b = qs[k], qs[k + 1]
		if cur.col <= b then
			if inner then
				return cur.line, a + 1, cur.line, b, false
			end
			local s, e, had = a, b, false
			while e < #r and is_blank(r[e + 1]) do
				e = e + 1
				had = true
			end
			if not had then
				while s > 1 and is_blank(r[s - 1]) do
					s = s - 1
				end
			end
			return cur.line, s, cur.line, e + 1, false
		end
	end
	return nil
end

-- Scan outward for the unmatched open/close of a bracket pair, honouring
-- nesting. Both walk the buffer, so blocks may span any number of lines.
local function find_unmatched(open, close, l, c, backward)
	local w = walker(l, c)
	local target = backward and open or close
	local other = backward and close or open
	if w:ch() == target then
		return w.l, w.c
	end
	local depth = 0
	while true do
		-- Not `backward and w:back() or w:fwd()`: when back() returns false at
		-- the top of the buffer that idiom falls through to fwd() and the search
		-- silently reverses direction.
		local moved
		if backward then
			moved = w:back()
		else
			moved = w:fwd()
		end
		if not moved then
			break
		end
		local ch = w:ch()
		if ch == other then
			depth = depth + 1
		elseif ch == target then
			if depth == 0 then
				return w.l, w.c
			end
			depth = depth - 1
		end
	end
	return nil
end

local function textobj_block(inner, open, close, count)
	local cur = editor.cursor()
	local ol, oc = cur.line, cur.col
	local cl, cc
	local from_l, from_c = ol, oc
	for i = 1, count do
		local nol, noc = find_unmatched(open, close, from_l, from_c, true)
		if not nol then
			return nil
		end
		ol, oc = nol, noc
		local ncl, ncc = find_unmatched(open, close, ol, oc, false)
		if not ncl then
			return nil
		end
		cl, cc = ncl, ncc
		-- Only step outside this pair when another level is still wanted;
		-- stepping unconditionally would leave `oc` one short of the real open
		-- bracket for the common count-of-one case.
		if i < count then
			if oc > 1 then
				from_l, from_c = ol, oc - 1
			elseif ol > 1 then
				from_l = ol - 1
				from_c = math.max(1, line_len(from_l))
			else
				break
			end
		end
	end
	if not cl then
		return nil
	end

	if not inner then
		return ol, oc, cl, cc + 1, false
	end

	-- Vim: an inner block whose open brace ends a line and whose close brace
	-- starts one becomes linewise, which is what makes `di{` clear a code block.
	if oc >= line_len(ol) and cl > ol + 1 and cc <= first_non_blank(cl) then
		return ol + 1, 1, cl - 1, 1, true
	end
	if ol == cl and cc <= oc + 1 then
		return ol, oc + 1, cl, oc + 1, false
	end
	return ol, oc + 1, cl, cc, false
end

-- it/at. Tags are found by flat-scanning a window around the cursor; a whole
-- buffer scan on a key-press path is not acceptable, and 500 lines covers any
-- realistic nesting.
local TAG_WINDOW = 250

local function scan_tags(runes)
	local tags = {}
	local i = 1
	local n = #runes
	while i <= n do
		if runes[i] == "<" then
			local j = i + 1
			local closing = false
			if runes[j] == "/" then
				closing = true
				j = j + 1
			end
			local name = {}
			while j <= n do
				local ch = runes[j]
				if ch:match("^[%w_:%-%.]$") then
					name[#name + 1] = ch
					j = j + 1
				else
					break
				end
			end
			if #name > 0 then
				local self_closing = false
				local k = j
				while k <= n and runes[k] ~= ">" do
					if runes[k] == "/" and runes[k + 1] == ">" then
						self_closing = true
					end
					k = k + 1
				end
				if k <= n then
					if not self_closing then
						tags[#tags + 1] = {
							name = table.concat(name),
							closing = closing,
							s = i,
							e = k,
						}
					end
					i = k
				end
			end
		end
		i = i + 1
	end
	return tags
end

local function textobj_tag(inner)
	local cur = editor.cursor()
	local n = line_count()
	local sl = math.max(1, cur.line - TAG_WINDOW)
	local el = math.min(n, cur.line + TAG_WINDOW)
	local runes, map = flatten(sl, el)
	local off = flat_offset(map, cur.line, cur.col)

	local tags = scan_tags(runes)
	local stack = {}
	local best = nil
	for _, t in ipairs(tags) do
		if t.closing then
			for k = #stack, 1, -1 do
				if stack[k].name == t.name then
					local o = stack[k]
					for _ = k, #stack do
						table.remove(stack)
					end
					if o.s <= off and off <= t.e and best == nil then
						best = { o = o, c = t }
					end
					break
				end
			end
		else
			stack[#stack + 1] = t
		end
	end
	if not best then
		return nil
	end

	local function pos(idx)
		local m = map[idx]
		if not m then
			return el, line_len(el) + 1
		end
		return m[1], m[2]
	end

	if inner then
		if best.o.e + 1 > best.c.s - 1 then
			local l, c = pos(best.c.s)
			return l, c, l, c, false
		end
		local a, b = pos(best.o.e + 1)
		local cl, cc = pos(best.c.s)
		return a, b, cl, cc, false
	end
	local a, b = pos(best.o.s)
	local cl, cc = pos(best.c.e)
	return a, b, cl, cc + 1, false
end

-- ip/ap. Linewise. A run of blank lines is itself a paragraph, so `dap` on a
-- blank line removes the gap.
local function textobj_paragraph(inner, count)
	local n = line_count()
	local l = editor.cursor().line
	local function blank(i)
		return (editor.get_line(i) or "") == ""
	end
	local kind = blank(l)
	local s, e = l, l
	while s > 1 and blank(s - 1) == kind do
		s = s - 1
	end
	while e < n and blank(e + 1) == kind do
		e = e + 1
	end

	-- Each extra step swallows the next block, whose blank-ness alternates.
	local cur_kind = kind
	local steps = inner and (count - 1) or count
	for step = 1, steps do
		local want = not cur_kind
		local grew = false
		while e < n and blank(e + 1) == want do
			e = e + 1
			grew = true
		end
		if not grew and not inner and step == 1 then
			-- `ap` with nothing after it takes the preceding block instead.
			while s > 1 and blank(s - 1) == want do
				s = s - 1
			end
		end
		cur_kind = want
	end
	return s, 1, e, 1, true
end

-- is/as. Sentences end at . ! or ? followed by closing quotes/brackets and then
-- whitespace or end of line. Scoped to the paragraph around the cursor.
local function textobj_sentence(inner, count)
	local n = line_count()
	local l = editor.cursor().line
	local function blank(i)
		return (editor.get_line(i) or "") == ""
	end
	if blank(l) then
		return l, 1, l, 1, false
	end
	local ps, pe = l, l
	while ps > 1 and not blank(ps - 1) do
		ps = ps - 1
	end
	while pe < n and not blank(pe + 1) do
		pe = pe + 1
	end

	local runes, map = flatten(ps, pe)
	local off = flat_offset(map, editor.cursor().line, editor.cursor().col)

	local starts = { 1 }
	local i = 1
	while i <= #runes do
		local ch = runes[i]
		if ch == "." or ch == "!" or ch == "?" then
			local j = i + 1
			while j <= #runes and (runes[j] == '"' or runes[j] == "'" or runes[j] == ")" or runes[j] == "]") do
				j = j + 1
			end
			if j > #runes then
				break
			end
			if runes[j] == " " or runes[j] == "\t" or runes[j] == "\n" then
				while j <= #runes and (runes[j] == " " or runes[j] == "\t" or runes[j] == "\n") do
					j = j + 1
				end
				if j <= #runes then
					starts[#starts + 1] = j
				end
				i = j
			else
				i = i + 1
			end
		else
			i = i + 1
		end
	end
	starts[#starts + 1] = #runes + 1

	local idx = 1
	for k = 1, #starts - 1 do
		if off >= starts[k] and off < starts[k + 1] then
			idx = k
			break
		end
	end
	local from = starts[idx]
	local to = starts[math.min(idx + count, #starts)] - 1

	if inner then
		while to > from and (runes[to] == " " or runes[to] == "\t" or runes[to] == "\n") do
			to = to - 1
		end
	end

	local function pos(k)
		local m = map[k]
		if not m then
			return pe, line_len(pe) + 1
		end
		return m[1], m[2]
	end
	local sl, sc = pos(from)
	local el, ec = pos(to)
	return sl, sc, el, ec + 1, false
end

-- Second key after `i`/`a` in operator-pending mode.
local TEXT_OBJECTS = {
	["w"] = function(inner, count)
		return textobj_word(inner, false, count)
	end,
	["W"] = function(inner, count)
		return textobj_word(inner, true, count)
	end,
	['"'] = function(inner)
		return textobj_quote(inner, '"')
	end,
	["'"] = function(inner)
		return textobj_quote(inner, "'")
	end,
	["`"] = function(inner)
		return textobj_quote(inner, "`")
	end,
	["("] = function(inner, count)
		return textobj_block(inner, "(", ")", count)
	end,
	[")"] = function(inner, count)
		return textobj_block(inner, "(", ")", count)
	end,
	["b"] = function(inner, count)
		return textobj_block(inner, "(", ")", count)
	end,
	["["] = function(inner, count)
		return textobj_block(inner, "[", "]", count)
	end,
	["]"] = function(inner, count)
		return textobj_block(inner, "[", "]", count)
	end,
	["{"] = function(inner, count)
		return textobj_block(inner, "{", "}", count)
	end,
	["}"] = function(inner, count)
		return textobj_block(inner, "{", "}", count)
	end,
	["B"] = function(inner, count)
		return textobj_block(inner, "{", "}", count)
	end,
	["<"] = function(inner, count)
		return textobj_block(inner, "<", ">", count)
	end,
	[">"] = function(inner, count)
		return textobj_block(inner, "<", ">", count)
	end,
	["t"] = function(inner)
		return textobj_tag(inner)
	end,
	["p"] = function(inner, count)
		return textobj_paragraph(inner, count)
	end,
	["s"] = function(inner, count)
		return textobj_sentence(inner, count)
	end,
}

-- ---------------------------------------------------------------------------
-- Operator-pending dispatch
-- ---------------------------------------------------------------------------

local function clear_operator()
	state.operator = nil
	state.textobj = nil
	state.count = nil
end

local function start_operator(op)
	state.operator = { op = op, count = state.count }
	state.count = nil
end

-- Both counts multiply, and either one satisfies "had a count" for the
-- line-number motions (`d3G`).
local function operator_count()
	local opc = state.operator and state.operator.count or nil
	local mc = state.count
	local total = (opc or 1) * (mc or 1)
	return total, (opc ~= nil or mc ~= nil)
end

-- The `.` payload for an operator. A yank changes nothing, so it is not
-- repeatable. state.register is still pending here -- the yank inside
-- apply_operator is what consumes it -- so the payload can capture it.
local function op_payload(op, n, target)
	if op == "y" then
		return nil
	end
	return { kind = "operator", op = op, count = n, register = state.register, target = target }
end

-- Doubled key: `dd`, `>>`, `guu`. Operates on `count` whole lines from the
-- cursor down.
local function run_linewise_operator()
	local op = state.operator.op
	local n = operator_count()
	local payload = op_payload(op, n, { t = "linewise" })
	local sl = editor.cursor().line
	local el = clamp(sl + n - 1, 1, line_count())
	clear_operator()
	apply_operator(op, sl, 1, el, 1, true, payload)
end

local function run_motion_operator(tok, gprefix)
	local op = state.operator.op
	local n, had = operator_count()
	local payload = op_payload(op, n, { t = "motion", tok = tok, gprefix = gprefix })
	local fn, kind
	if gprefix then
		kind = G_MOTION_KIND[tok]
		fn = kind and G_MOTIONS[tok] or nil
	else
		fn = MOTIONS[tok]
		kind = MOTION_KIND[tok] or "exclusive"
	end
	if not fn then
		clear_operator()
		return
	end

	-- `cw` on a non-blank behaves like `ce`: Vim's own documented special case.
	if op == "c" and not gprefix and (tok == "w" or tok == "W") then
		local r = line_runes(editor.cursor().line)
		local ch = r[editor.cursor().col]
		if ch ~= nil and not is_blank(ch) then
			fn = MOTIONS[tok == "w" and "e" or "E"]
			kind = "inclusive"
		end
	end

	local start, dest = probe_motion(fn, n, had)
	if not gprefix and (tok == "w" or tok == "W") then
		dest = clip_word_motion(start, dest)
	end
	clear_operator()

	local sl, sc, el, ec, linewise = motion_to_range(start, dest, kind)
	if sl == nil then
		return
	end
	apply_operator(op, sl, sc, el, ec, linewise, payload)
end

local function run_find_operator(op_char, ch)
	local op = state.operator.op
	local n = operator_count()
	local payload = op_payload(op, n, { t = "find", op = op_char, ch = ch })
	local start = editor.cursor()
	local saved_goal = state.goal
	find_char(op_char, ch, n, false)
	local dest = editor.cursor()
	editor.set_cursor(start.line, start.col)
	state.goal = saved_goal
	clear_operator()

	if dest.line == start.line and dest.col == start.col then
		return
	end
	-- Forward f/t include the character they land on; backward F/T do not
	-- include the one the cursor started on.
	local kind = (dest.col > start.col) and "inclusive" or "exclusive"
	local sl, sc, el, ec, linewise = motion_to_range(start, dest, kind)
	if sl == nil then
		return
	end
	apply_operator(op, sl, sc, el, ec, linewise, payload)
end

local function run_textobject(inner, tok)
	local op = state.operator.op
	local n = operator_count()
	local payload = op_payload(op, n, { t = "textobj", inner = inner, tok = tok })
	local fn = TEXT_OBJECTS[tok]
	clear_operator()
	if not fn then
		return
	end
	local sl, sc, el, ec, linewise = fn(inner, n)
	if sl == nil then
		return
	end
	if not linewise and sl == el and ec <= sc then
		return
	end
	apply_operator(op, sl, sc, el, ec, linewise, payload)
end

-- `d'a` / ``d`a``. Backtick is an exclusive charwise motion, `'` a linewise one.
local function run_mark_operator(name, linewise_mark)
	local op = state.operator.op
	local n = operator_count()
	local payload = op_payload(op, n, { t = "mark", name = name, linewise = linewise_mark })
	local start = editor.cursor()
	local ml, mc = mark_target(name, linewise_mark)
	clear_operator()
	if not ml then
		return
	end
	local kind = linewise_mark and "linewise" or "exclusive"
	local sl, sc, el, ec, lw = motion_to_range(start, { line = ml, col = mc }, kind)
	if sl == nil then
		return
	end
	apply_operator(op, sl, sc, el, ec, lw, payload)
end

-- g-prefixed edits, appended to the Phase 1 motion table.
G_MOTIONS["i"] = function()
	local li = state.last_insert
	editor.begin_undo_group()
	if li then
		local l = clamp(li.line, 1, line_count())
		editor.set_cursor(l, math.min(li.col, line_len(l) + 1))
	end
	begin_insert()
end

G_MOTIONS["J"] = function(n)
	join(n, false)
end

-- ---------------------------------------------------------------------------
-- Visual modes
--
-- state.visual holds the anchor (al, ac) and the *Vim* cursor (cl, cc). The
-- real editor cursor is not always the Vim cursor: ttt renders a selection as
-- Selection.Start .. live-cursor (internal/core/selection), so the only free
-- variable is the anchor, and linewise/blockwise need the real cursor parked
-- somewhere else to draw the range correctly. Keeping (cl, cc) in Lua means
-- motions still see and produce true Vim positions -- visual_sync() puts the
-- real cursor back before a motion runs, visual_capture() reads it after.
--
-- Blockwise has no native backing at all: it is drawn with add_cursor() per
-- line and every edit loops the lines by hand.
-- ---------------------------------------------------------------------------

local VISUAL_KEY_MODE = { v = "visual", V = "visual_line", ["ctrl-v"] = "visual_block" }

local function take_count()
	local n = state.count
	state.count = nil
	return n or 1, n ~= nil
end

local function visual_sync()
	local v = state.visual
	if not v then
		return
	end
	local l = clamp(v.cl, 1, line_count())
	editor.set_cursor(l, clamp(v.cc, 1, max_col(line_runes(l))))
end

local function visual_capture()
	local v = state.visual
	if not v then
		return
	end
	local c = editor.cursor()
	v.cl = c.line
	v.cc = c.col
end

-- Normalized block corners. `dollar` is the ragged right edge set by `$`.
local function block_rect()
	local v = state.visual
	local top, bot = v.al, v.cl
	if top > bot then
		local t = top
		top = bot
		bot = t
	end
	local left, right = v.ac, v.cc
	if left > right then
		local t = left
		left = right
		right = t
	end
	return top, bot, left, right
end

local function visual_render()
	local v = state.visual
	if not v then
		return
	end

	if state.mode == "visual_block" then
		editor.clear_cursors()
		editor.clear_selection()
		local top, bot, _, right = block_rect()
		local cl = clamp(v.cl, 1, line_count())
		editor.set_cursor(cl, clamp(v.cc, 1, max_col(line_runes(cl))))
		for l = top, bot do
			if l ~= cl then
				local col = v.dollar and max_col(line_runes(l)) or math.min(right, max_col(line_runes(l)))
				editor.add_cursor(l, col)
			end
		end
		return
	end

	if state.mode == "visual_line" then
		local top, bot = v.al, v.cl
		if top > bot then
			local t = top
			top = bot
			bot = t
		end
		if v.cl >= v.al then
			editor.set_selection(top, 1, v.cl, line_len(v.cl) + 1)
		else
			editor.set_selection(bot, line_len(bot) + 1, v.cl, 1)
		end
		return
	end

	-- Charwise. Forward: the anchor is the start and the cursor is the
	-- (exclusive) end, so the character *under* the cursor is not painted --
	-- the cursor block itself stands in for it. Backward: the anchor is shifted
	-- one column right so the character it sits on stays inside the range, which
	-- is exactly Vim's inclusive behaviour on that side.
	local backward = (v.cl < v.al) or (v.cl == v.al and v.cc < v.ac)
	if backward then
		editor.set_selection(v.al, math.min(v.ac + 1, line_len(v.al) + 1), v.cl, v.cc)
	else
		editor.set_selection(v.al, v.ac, v.cl, v.cc)
	end
end

-- The selection as an operator range: sl, sc, el, ec (exclusive end column),
-- linewise. Charwise visual is inclusive of the character under the cursor.
local function visual_range()
	local v = state.visual
	local sl, sc, el, ec = v.al, v.ac, v.cl, v.cc
	if (v.cl < v.al) or (v.cl == v.al and v.cc < v.ac) then
		sl = v.cl
		sc = v.cc
		el = v.al
		ec = v.ac
	end
	if state.mode == "visual_line" then
		return sl, 1, el, 1, true
	end
	return sl, sc, el, math.min(ec + 1, line_len(el) + 1), false
end

local function save_last_visual()
	local v = state.visual
	if not v then
		return
	end
	state.last_visual = { mode = state.mode, al = v.al, ac = v.ac, cl = v.cl, cc = v.cc, dollar = v.dollar }
end

-- Tear down the visual chrome and return to normal mode *without* touching the
-- cursor. Operators call this before they edit; Esc uses exit_visual().
local function end_visual()
	-- set_mode clears the pending `"x` prefix, but a visual operator types its
	-- register *before* the operator key, so it has to survive this teardown.
	local reg = state.register
	save_last_visual()
	if state.mode == "visual_block" then
		editor.clear_cursors()
	end
	editor.clear_selection()
	state.visual = nil
	set_mode("normal")
	state.register = reg
end

local function exit_visual()
	local v = state.visual
	local l, c = v and v.cl or nil, v and v.cc or nil
	end_visual()
	if l then
		move_to(l, c)
	end
end

local function enter_visual(mode)
	local cur = editor.cursor()
	state.visual = { al = cur.line, ac = cur.col, cl = cur.line, cc = cur.col, dollar = false }
	set_mode(mode)
	visual_render()
end

local function switch_visual(mode)
	if state.mode == "visual_block" and mode ~= "visual_block" then
		editor.clear_cursors()
	end
	if mode ~= "visual_block" then
		state.visual.dollar = false
	end
	set_mode(mode)
	visual_render()
end

local function reselect_visual()
	local lv = state.last_visual
	if not lv then
		return
	end
	local n = line_count()
	state.visual = {
		al = clamp(lv.al, 1, n),
		ac = lv.ac,
		cl = clamp(lv.cl, 1, n),
		cc = lv.cc,
		dollar = lv.dollar,
	}
	set_mode(lv.mode)
	visual_render()
end

-- `o`: swap the anchor and the cursor. gopher-lua does not swap in a multiple
-- assignment when a target appears on the right, so this goes via temporaries.
local function visual_swap_ends()
	local v = state.visual
	local al, ac = v.al, v.ac
	v.al = v.cl
	v.ac = v.cc
	v.cl = al
	v.cc = ac
end

-- `O` in blockwise: swap the two corners horizontally, leaving the rows alone.
local function visual_swap_corners()
	local v = state.visual
	local ac = v.ac
	v.ac = v.cc
	v.cc = ac
end

-- Run a Phase 1 motion with the real cursor parked at the Vim cursor, then take
-- the result back as the new Vim cursor and redraw.
local function visual_motion(fn, n, had)
	visual_sync()
	fn(n, had)
	visual_capture()
	visual_render()
end

-- ---------------------------------------------------------------------------
-- Blockwise edits
--
-- Every one of these loops the rows bottom-to-top: editor.replace shifts the
-- content of later lines only when it removes a newline, but iterating upward
-- keeps line indices stable no matter what, and it is the same discipline the
-- multi-cursor code uses.
-- ---------------------------------------------------------------------------

-- Column span [lo, hi) for row `l`, clamped onto the line. Returns nil when the
-- row is too short to contain any of the block.
local function block_span(l, left, right, dollar)
	local len = line_len(l)
	local lo = math.min(left, len + 1)
	local hi = dollar and (len + 1) or math.min(right + 1, len + 1)
	if hi <= lo then
		return nil
	end
	return lo, hi
end

local function block_text(top, bot, left, right, dollar)
	local parts = {}
	for l = top, bot do
		local lo, hi = block_span(l, left, right, dollar)
		parts[#parts + 1] = lo and sub_runes(line_runes(l), lo, hi - 1) or ""
	end
	return table.concat(parts, "\n")
end

local function block_delete(top, bot, left, right, dollar)
	for l = bot, top, -1 do
		local lo, hi = block_span(l, left, right, dollar)
		if lo then
			editor.replace(l, lo, l, hi, "")
		end
	end
end

-- I / A / c: park a cursor on every row and hand typing to the editor's native
-- multi-cursor path, which already coalesces into the open undo group.
local function block_insert_at(top, bot, col_for)
	editor.clear_cursors()
	editor.clear_selection()
	local primary = col_for(top)
	editor.set_cursor(top, primary)
	for l = top + 1, bot do
		editor.add_cursor(l, col_for(l))
	end
	state.block_insert = true
	begin_insert()
end

-- ---------------------------------------------------------------------------
-- Visual-mode operators
-- ---------------------------------------------------------------------------

local function visual_case(how)
	local sl, sc, el, ec, linewise = visual_range()
	local op = (how == "lower" and "gu") or (how == "upper" and "gU") or "g~"
	if state.mode == "visual_block" then
		local top, bot, left, right = block_rect()
		local dollar = state.visual.dollar
		end_visual()
		editor.begin_undo_group()
		for l = bot, top, -1 do
			local lo, hi = block_span(l, left, right, dollar)
			if lo then
				local text = sub_runes(line_runes(l), lo, hi - 1)
				editor.replace(l, lo, l, hi, transform_case(text, how))
			end
		end
		editor.end_undo_group()
		move_to(top, math.min(left, max_col(line_runes(top))))
		return
	end
	end_visual()
	apply_operator(op, sl, sc, el, ec, linewise)
end

local function visual_replace_char(ch)
	if state.mode == "visual_block" then
		local top, bot, left, right = block_rect()
		local dollar = state.visual.dollar
		end_visual()
		editor.begin_undo_group()
		for l = bot, top, -1 do
			local lo, hi = block_span(l, left, right, dollar)
			if lo then
				editor.replace(l, lo, l, hi, ch:rep(hi - lo))
			end
		end
		editor.end_undo_group()
		move_to(top, math.min(left, max_col(line_runes(top))))
		return
	end

	local sl, sc, el, ec, linewise = visual_range()
	if linewise then
		sc = 1
		ec = line_len(el) + 1
	end
	end_visual()
	editor.begin_undo_group()
	for l = el, sl, -1 do
		local lo = (l == sl) and sc or 1
		local hi = (l == el) and ec or (line_len(l) + 1)
		hi = math.min(hi, line_len(l) + 1)
		if hi > lo then
			editor.replace(l, lo, l, hi, ch:rep(hi - lo))
		end
	end
	editor.end_undo_group()
	move_to(sl, sc)
end

-- Visual `p`: the selection is replaced by the register, and the text that was
-- there becomes the new unnamed register, as in Vim.
local function visual_paste()
	local reg = get_register(take_register())
	local sl, sc, el, ec, linewise = visual_range()
	if state.mode == "visual_block" or not reg or reg.text == "" then
		return
	end
	local text = reg.text
	local body = (text:gsub("\n$", ""))
	end_visual()

	local a, b = sc, ec
	if linewise then
		a = 1
		b = line_len(el) + 1
	end

	editor.begin_undo_group()
	if linewise then
		yank_linewise(sl, el, true)
	else
		yank_charwise(sl, sc, el, ec, true)
	end

	local repl
	if linewise then
		repl = (reg.kind == "line") and body or text
	else
		repl = (reg.kind == "line") and ("\n" .. body .. "\n") or text
	end
	editor.replace(sl, a, el, b, repl)
	editor.end_undo_group()
	move_to(sl, linewise and first_non_blank(sl) or a)
end

local function visual_join()
	local top, bot = state.visual.al, state.visual.cl
	if top > bot then
		local t = top
		top = bot
		bot = t
	end
	end_visual()
	move_to(top, 1)
	join(math.max(2, bot - top + 1), true)
end

local function visual_indent(op, count)
	local sl, _, el = visual_range()
	end_visual()
	editor.begin_undo_group()
	if op == "=" then
		reindent_range(sl, el)
	else
		for _ = 1, math.max(1, count) do
			indent_lines_range(sl, el, op == ">" and 1 or -1)
		end
	end
	editor.end_undo_group()
	move_to(sl, first_non_blank(sl))
end

-- d / x / c / s / y over the current selection, in whichever visual mode.
local function visual_operator(op, force_linewise)
	if state.mode == "visual_block" then
		local top, bot, left, right = block_rect()
		local dollar = state.visual.dollar
		if op == "y" then
			set_register(block_text(top, bot, left, right, dollar), "block", false)
			end_visual()
			move_to(top, math.min(left, max_col(line_runes(top))))
			return
		end
		set_register(block_text(top, bot, left, right, dollar), "block", true)
		end_visual()
		editor.begin_undo_group()
		block_delete(top, bot, left, right, dollar)
		if op == "c" then
			block_insert_at(top, bot, function(l)
				return math.min(left, line_len(l) + 1)
			end)
			-- Undo group stays open; leave_insert() closes it.
			return
		end
		editor.end_undo_group()
		move_to(top, math.min(left, max_col(line_runes(top))))
		return
	end

	local sl, sc, el, ec, linewise = visual_range()
	if force_linewise then
		sc, ec, linewise = 1, 1, true
	end
	end_visual()
	apply_operator(op, sl, sc, el, ec, linewise)
end

-- ---------------------------------------------------------------------------
-- Visual-mode key dispatch
-- ---------------------------------------------------------------------------

-- Text objects in visual mode *set* the selection to the object rather than
-- running an operator, so `viw` selects the word under the cursor.
local function visual_textobject(inner, tok)
	local fn = TEXT_OBJECTS[tok]
	local n = state.count or 1
	state.count = nil
	if not fn then
		return
	end
	visual_sync()
	local sl, sc, el, ec, linewise = fn(inner, n)
	if sl == nil then
		return
	end
	local v = state.visual
	if linewise then
		v.al, v.ac = sl, 1
		v.cl, v.cc = el, 1
		if state.mode ~= "visual_line" then
			switch_visual("visual_line")
			return
		end
	else
		v.al, v.ac = sl, sc
		v.cl = el
		v.cc = math.max(1, ec - 1)
	end
	visual_render()
end

-- `$` in blockwise means "ragged right edge"; everywhere else it is the motion.
local function visual_dollar(n)
	if state.mode == "visual_block" then
		state.visual.dollar = true
	end
	visual_motion(MOTIONS["$"], n, false)
end

local VISUAL_LINEWISE_OPS = { X = "d", D = "d", R = "c", S = "c", C = "c", Y = "y" }

local function handle_visual(tok)
	if ESCAPE_TOKENS[tok] then
		exit_visual()
		return true
	end

	if state.await_register then
		state.await_register = false
		if is_printable(tok) then
			state.register = tok
		end
		return true
	end

	if state.mark_pending then
		local what = state.mark_pending
		state.mark_pending = nil
		if tok:match(MARK_NAME) then
			if what == "m" then
				visual_sync()
				local cur = editor.cursor()
				mark_set(tok, cur.line, cur.col)
			else
				local ml, mc = mark_target(tok, what == "'")
				if ml then
					visual_motion(function()
						move_to(ml, mc)
					end, 1, false)
				end
			end
		end
		return true
	end

	if state.replace_pending then
		state.replace_pending = nil
		if is_printable(tok) then
			visual_replace_char(tok)
		end
		return true
	end

	if state.find_pending then
		local op = state.find_pending
		state.find_pending = nil
		if is_printable(tok) then
			state.last_find = { op = op, ch = tok }
			visual_motion(function(n)
				find_char(op, tok, n, false)
			end, take_count(), false)
		end
		return true
	end

	if state.textobj then
		local inner = state.textobj == "i"
		state.textobj = nil
		visual_textobject(inner, tok)
		return true
	end

	if state.pending == "g" then
		state.pending = ""
		if tok == "v" then
			-- `gv` inside visual mode is a no-op in Vim; swallow it.
			state.count = nil
			return true
		end
		local gop = G_OPERATORS[tok]
		if gop then
			state.count = nil
			visual_case(CASE_OPERATORS[gop])
			return true
		end
		if tok == "J" then
			state.count = nil
			visual_join()
			return true
		end
		local fn = G_MOTIONS[tok]
		if fn and G_MOTION_KIND[tok] then
			local n, had = take_count()
			visual_motion(fn, n, had)
		else
			state.count = nil
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

	if #tok == 1 and tok >= "0" and tok <= "9" and not (tok == "0" and state.count == nil) then
		state.count = (state.count or 0) * 10 + tonumber(tok)
		return true
	end

	-- Mode keys: the same key exits, a different one switches.
	local vm = VISUAL_KEY_MODE[tok]
	if vm then
		state.count = nil
		if state.mode == vm then
			exit_visual()
		else
			switch_visual(vm)
		end
		return true
	end

	if tok == '"' then
		state.await_register = true
		return true
	end

	if tok == "m" or tok == "`" or tok == "'" then
		state.mark_pending = tok
		return true
	end

	if tok == "i" or tok == "a" then
		state.textobj = tok
		return true
	end

	if tok == "g" or tok == "z" then
		state.pending = tok
		return true
	end

	if FIND_KEYS[tok] then
		state.find_pending = tok
		return true
	end

	if tok == "o" then
		state.count = nil
		visual_swap_ends()
		visual_render()
		return true
	end

	if tok == "O" then
		state.count = nil
		if state.mode == "visual_block" then
			visual_swap_corners()
		else
			visual_swap_ends()
		end
		visual_render()
		return true
	end

	if tok == "$" then
		local n = take_count()
		visual_dollar(n)
		return true
	end

	if tok == "r" then
		state.replace_pending = 1
		return true
	end

	if tok == "J" then
		state.count = nil
		visual_join()
		return true
	end

	if tok == "u" or tok == "U" or tok == "~" then
		state.count = nil
		visual_case((tok == "u" and "lower") or (tok == "U" and "upper") or "swap")
		return true
	end

	if tok == ">" or tok == "<" or tok == "=" then
		local n = take_count()
		visual_indent(tok, n)
		return true
	end

	if tok == "p" or tok == "P" then
		state.count = nil
		visual_paste()
		return true
	end

	-- Blockwise insert at the left / right edge of every row.
	if (tok == "I" or tok == "A") and state.mode == "visual_block" then
		state.count = nil
		local top, bot, left, right = block_rect()
		local dollar = state.visual.dollar
		end_visual()
		editor.begin_undo_group()
		if tok == "I" then
			block_insert_at(top, bot, function(l)
				return math.min(left, line_len(l) + 1)
			end)
		else
			block_insert_at(top, bot, function(l)
				return dollar and (line_len(l) + 1) or math.min(right + 1, line_len(l) + 1)
			end)
		end
		return true
	end

	if tok == "d" or tok == "x" then
		state.count = nil
		visual_operator("d", false)
		return true
	end
	if tok == "c" or tok == "s" then
		state.count = nil
		visual_operator("c", false)
		return true
	end
	if tok == "y" then
		state.count = nil
		visual_operator("y", false)
		return true
	end

	local lw = VISUAL_LINEWISE_OPS[tok]
	if lw then
		state.count = nil
		visual_operator(lw, true)
		return true
	end

	local motion = MOTIONS[tok]
	if motion then
		local n, had = take_count()
		visual_motion(motion, n, had)
		return true
	end

	if is_printable(tok) or MUTATING_KEYS[tok] then
		return true
	end

	return false
end

-- ---------------------------------------------------------------------------
-- Dot repeat
--
-- `.` replays the resolved payload recorded by the last buffer-changing
-- command, not the keystrokes that produced it. Everything below re-enters the
-- very same code paths the original command took, which is what keeps `.` a
-- single undo step: each of those paths opens exactly one undo group.
-- ---------------------------------------------------------------------------

local function do_replace_char(n, ch)
	local cur = editor.cursor()
	-- Vim refuses the whole operation when the count runs past the end of the
	-- line rather than replacing fewer characters.
	if cur.col + n - 1 > line_len(cur.line) then
		return
	end
	editor.begin_undo_group()
	editor.replace(cur.line, cur.col, cur.line, cur.col + n, ch:rep(n))
	editor.end_undo_group()
	editor.set_cursor(cur.line, cur.col + n - 1)
end

-- Overtype `text` from the cursor, the way `R` does.
local function overtype(text)
	for _, ch in ipairs(runes_of(text)) do
		local cur = editor.cursor()
		if ch == "\n" then
			editor.insert(cur.line, cur.col, "\n")
			editor.set_cursor(cur.line + 1, 1)
		elseif cur.col <= line_len(cur.line) then
			editor.replace(cur.line, cur.col, cur.line, cur.col + 1, ch)
			editor.set_cursor(cur.line, cur.col + 1)
		else
			editor.insert(cur.line, cur.col, ch)
			editor.set_cursor(cur.line, cur.col + 1)
		end
	end
end

-- Finish an insert session that a replay opened: the recorded text has to be
-- typed by hand, because there is no editor to type it.
local function replay_insert(tok, text, count)
	local ctx = state.insert_ctx
	if ctx then
		-- The repetition is done here, so leave_insert() must not do it again.
		ctx.count = 1
	end
	if text and text ~= "" then
		local cur = editor.cursor()
		local l, c = cur.line, cur.col
		local open_line = (tok == "o" or tok == "O")
		for i = 1, math.max(1, count or 1) do
			if open_line and i > 1 then
				l, c = insert_and_advance(l, line_len(l) + 1, "\n" .. text)
			else
				l, c = insert_and_advance(l, c, text)
			end
		end
		editor.set_cursor(l, c)
	end
	leave_insert()
end

-- The entry commands whose count multiplies the typed text rather than the
-- amount of buffer they consume (`3i` types three copies, `3s` does not).
local REPEAT_TEXT = { i = true, I = true, a = true, A = true, o = true, O = true }

-- ---------------------------------------------------------------------------
-- Phase 6: search, ex command line, substitution, editor integration
--
-- The command line is a modal overlay, handled *above* the plugin key
-- interceptor: while it is open `key.press` delivers nothing here at all. So
-- there is no cmdline mode and no focus flag -- `/`, `?` and `:` call
-- ttt.command_line.show() and do every bit of their work in the callbacks.
--
-- on_change is a preview only. It moves the cursor (which scrolls the view) and
-- never touches the buffer; a buffer edit from an incremental preview would be
-- unundoable and would invalidate the very offsets it is searching.
-- ---------------------------------------------------------------------------

-- Phase 6 lives inside a `do` block on purpose. The main chunk is within a
-- handful of slots of Lua's 200-locals-per-function ceiling, and a block scope
-- releases its registers at `end`; without it the whole file fails to compile
-- with "too many local variables". Everything the dispatcher needs is re-exported
-- through the single `vim6` table below.
local vim6 = {}

do
	local fs_ok, fs = pcall(require, "ttt.fs")
	if not fs_ok then
		fs = nil
	end

	-- ---------------------------------------------------------------------------
	-- Byte <-> rune column conversion
	--
	-- The editor API speaks rune columns; Lua's pattern matcher speaks bytes. Every
	-- match crosses that boundary exactly twice, here.
	-- ---------------------------------------------------------------------------

	local function rune_len(b)
		if b >= 0xf0 then
			return 4
		elseif b >= 0xe0 then
			return 3
		elseif b >= 0xc0 then
			return 2
		end
		return 1
	end

	-- 1-based byte index at which rune column `col` starts. A col past the end of
	-- the line yields #s + 1.
	local function byte_of_col(s, col)
		local i, k, n = 1, 1, #s
		while i <= n and k < col do
			i = i + rune_len(s:byte(i))
			k = k + 1
		end
		return i
	end

	-- 1-based rune column containing byte index `bi`.
	local function col_of_byte(s, bi)
		local i, k, n = 1, 1, #s
		while i <= n and i < bi do
			i = i + rune_len(s:byte(i))
			k = k + 1
		end
		return k
	end

	local function is_word_byte(b)
		if b == nil then
			return false
		end
		return (b >= 0x30 and b <= 0x39)
			or (b >= 0x41 and b <= 0x5a)
			or (b >= 0x61 and b <= 0x7a)
			or b == 0x5f
			or b >= 0x80
	end

	-- ---------------------------------------------------------------------------
	-- Vim regex -> Lua pattern
	--
	-- Lua 5.1 patterns are NOT a regex engine: there is no alternation, no
	-- grouping-with-quantifier, no counted repeat and no lookaround, and gopher-lua
	-- additionally lacks the %f frontier pattern. Rather than silently
	-- misinterpreting a pattern, anything that cannot be represented is rejected
	-- with a message. The supported subset is documented in README.md.
	--
	-- Translated:
	--   .  *  ^  $  [...]        pass through (same meaning in both)
	--   \+ \? \=                 -> +  ?  ?
	--   \( \)                    -> capture group (Lua groups cannot be quantified)
	--   \d \D \s \S \a \l \u \x  -> %d %D %s %S %a %l %u %x  (and their complements)
	--   \w \W                    -> [%w_] / [^%w_]  (Vim's \w includes underscore)
	--   \n \t \e                 -> the literal control character
	--   \c                       -> anywhere in the pattern: ignore case
	--   \< \>                    -> word boundary, but only at the very start/end
	--                               of the pattern (there is no %f to compile to),
	--                               checked against the neighbouring byte instead
	--   ( ) + ? { } | ~ &        literal, as in Vim's default "magic" mode
	-- Rejected: \| \{n,m} \@ \% \zs \ze and any other \-escape with a regex meaning.
	-- ---------------------------------------------------------------------------

	local VIM_CLASS = {
		d = "%d",
		D = "%D",
		s = "%s",
		S = "%S",
		w = "[%w_]",
		W = "[^%w_]",
		a = "%a",
		A = "%A",
		l = "%l",
		L = "%L",
		u = "%u",
		U = "%U",
		x = "%x",
		X = "%X",
		o = "[0-7]",
		n = "\n",
		t = "\t",
		e = "\27",
	}

	-- \-escapes that mean something in Vim which Lua patterns cannot express.
	local VIM_UNSUPPORTED = {
		["|"] = "alternation (\\|)",
		["{"] = "counted repeats (\\{n,m})",
		["}"] = "counted repeats (\\{n,m})",
		["@"] = "lookaround (\\@=, \\@!)",
		["%"] = "\\%( groups",
		["z"] = "\\zs / \\ze",
	}

	local LUA_MAGIC = "^$*+?.([%])-"

	local function lua_escape(ch)
		if #ch == 1 and LUA_MAGIC:find(ch, 1, true) then
			return "%" .. ch
		end
		return ch
	end

	-- A literal character, doubled into a set when the match must ignore case.
	local function emit_literal(out, ch, ic)
		if ic and ch:match("^%a$") then
			out[#out + 1] = "[" .. ch:lower() .. ch:upper() .. "]"
		else
			out[#out + 1] = lua_escape(ch)
		end
	end

	-- Copy a [...] class through, translating \d-style escapes and escaping the
	-- Lua-only metacharacter `%`. Returns the class text and the index just past
	-- the closing bracket, or nil when the class is unterminated.
	local function convert_class(pat, i, ic)
		local n = #pat
		local cls = { "[" }
		local j = i + 1
		if pat:sub(j, j) == "^" then
			cls[#cls + 1] = "^"
			j = j + 1
		end
		if pat:sub(j, j) == "]" then
			cls[#cls + 1] = "%]"
			j = j + 1
		end
		local closed = false
		while j <= n do
			local c = pat:sub(j, j)
			if c == "]" then
				closed = true
				j = j + 1
				break
			elseif c == "\\" then
				local nx = pat:sub(j + 1, j + 1)
				local m = VIM_CLASS[nx]
				-- Only single %-classes and control characters compose inside a set.
				if m and (#m == 2 or #m == 1) then
					cls[#cls + 1] = m
				elseif nx == "" then
					cls[#cls + 1] = "%\\"
				else
					cls[#cls + 1] = lua_escape(nx)
				end
				j = j + 2
			elseif c == "%" then
				cls[#cls + 1] = "%%"
				j = j + 1
			else
				if ic and c:match("^%a$") then
					cls[#cls + 1] = c:lower() .. c:upper()
				else
					cls[#cls + 1] = c
				end
				j = j + 1
			end
		end
		if not closed then
			return nil
		end
		cls[#cls + 1] = "]"
		return table.concat(cls), j
	end

	-- Returns a compiled pattern { lua, word_start, word_end, ic } or nil + message.
	local function compile_vim_pattern(pat, ignorecase)
		local ic = ignorecase and true or false
		if pat:find("\\c", 1, true) then
			ic = true
			pat = pat:gsub("\\c", "")
		end
		if pat:find("\\C", 1, true) then
			ic = false
			pat = pat:gsub("\\C", "")
		end

		local out = {}
		local ws, we = false, false
		local i, n = 1, #pat

		while i <= n do
			local c = pat:sub(i, i)
			if c == "\\" then
				local nx = pat:sub(i + 1, i + 1)
				i = i + 2
				if nx == "" then
					out[#out + 1] = "%\\"
				elseif nx == "<" then
					if #out > 0 then
						return nil, "\\< is only supported at the start of a pattern"
					end
					ws = true
				elseif nx == ">" then
					if i <= n then
						return nil, "\\> is only supported at the end of a pattern"
					end
					we = true
				elseif nx == "+" then
					out[#out + 1] = "+"
				elseif nx == "?" or nx == "=" then
					out[#out + 1] = "?"
				elseif nx == "(" then
					out[#out + 1] = "("
				elseif nx == ")" then
					out[#out + 1] = ")"
				elseif VIM_UNSUPPORTED[nx] then
					return nil, VIM_UNSUPPORTED[nx] .. " is not supported"
				elseif VIM_CLASS[nx] then
					out[#out + 1] = VIM_CLASS[nx]
				else
					emit_literal(out, nx, ic)
				end
			elseif c == "[" then
				local cls, j = convert_class(pat, i, ic)
				if not cls then
					return nil, "unterminated [ ] in pattern"
				end
				out[#out + 1] = cls
				i = j
			elseif c == "." or c == "*" or c == "^" or c == "$" then
				out[#out + 1] = c
				i = i + 1
			elseif c == "%" then
				out[#out + 1] = "%%"
				i = i + 1
			else
				local b = c:byte(1)
				if b >= 0x80 then
					local len = rune_len(b)
					out[#out + 1] = pat:sub(i, i + len - 1)
					i = i + len
				else
					emit_literal(out, c, ic)
					i = i + 1
				end
			end
		end

		return { lua = table.concat(out), word_start = ws, word_end = we, ic = ic }
	end

	-- ---------------------------------------------------------------------------
	-- Replacement text
	--
	-- Vim's `&` and `\0`-`\9` become Lua's `%0`-`%9`, which `expand()` below
	-- substitutes by hand -- string.gsub is not used, because the word-boundary
	-- flags cannot be expressed as a pattern and have to be checked per match.
	-- ---------------------------------------------------------------------------

	local function compile_replacement(rep)
		local out = {}
		local i, n = 1, #rep
		while i <= n do
			local c = rep:sub(i, i)
			if c == "\\" then
				local nx = rep:sub(i + 1, i + 1)
				i = i + 2
				if nx == "" then
					out[#out + 1] = "\\"
				elseif nx == "r" or nx == "n" then
					return nil, "a replacement cannot contain a line break"
				elseif nx:match("^%d$") then
					out[#out + 1] = "%" .. nx
				elseif nx == "t" then
					out[#out + 1] = "\t"
				elseif nx == "%" then
					out[#out + 1] = "%%"
				else
					out[#out + 1] = nx
				end
			elseif c == "&" then
				out[#out + 1] = "%0"
				i = i + 1
			elseif c == "%" then
				out[#out + 1] = "%%"
				i = i + 1
			else
				out[#out + 1] = c
				i = i + 1
			end
		end
		return table.concat(out)
	end

	local function expand(rep, whole, caps)
		local out = {}
		local i, n = 1, #rep
		while i <= n do
			local c = rep:sub(i, i)
			if c == "%" then
				local d = rep:sub(i + 1, i + 1)
				if d == "0" then
					out[#out + 1] = whole
				elseif d:match("^%d$") then
					out[#out + 1] = tostring(caps[tonumber(d)] or "")
				elseif d == "%" then
					out[#out + 1] = "%"
				else
					out[#out + 1] = d
				end
				i = i + 2
			else
				out[#out + 1] = c
				i = i + 1
			end
		end
		return table.concat(out)
	end

	-- ---------------------------------------------------------------------------
	-- Matching
	-- ---------------------------------------------------------------------------

	-- Every match of `c` in one line, as { sb, eb, text } with inclusive byte
	-- bounds and the expanded replacement. Empty matches are skipped: Vim would
	-- accept them, but they are useless for `n` and dangerous for `:s`.
	local function line_matches(text, c, rep)
		local out = {}
		local anchored = c.lua:sub(1, 1) == "^"
		local init = 1
		while init <= #text + 1 do
			local res = { string.find(text, c.lua, init) }
			local sb, eb = res[1], res[2]
			if not sb then
				break
			end
			if eb < sb then
				init = sb + 1
			else
				local ok = true
				if c.word_start and is_word_byte(text:byte(sb - 1)) then
					ok = false
				end
				if c.word_end and is_word_byte(text:byte(eb + 1)) then
					ok = false
				end
				if ok then
					local caps = {}
					for k = 3, #res do
						caps[k - 2] = res[k]
					end
					out[#out + 1] = { sb = sb, eb = eb, text = expand(rep or "", text:sub(sb, eb), caps) }
				end
				init = eb + 1
			end
			if anchored then
				break
			end
		end
		return out
	end

	-- The whole-buffer scan. Deliberately one pass per search rather than per
	-- keystroke of a motion. Returns line, start col, end col (exclusive), wrapped.
	local function search_from(c, line, col, dir)
		local n = line_count()
		for step = 0, n do
			local l = line + dir * step
			local wrapped = false
			while l > n do
				l = l - n
				wrapped = true
			end
			while l < 1 do
				l = l + n
				wrapped = true
			end
			local text = editor.get_line(l) or ""
			local ms = line_matches(text, c)
			if #ms > 0 then
				if step == 0 then
					local cb = byte_of_col(text, col)
					if dir > 0 then
						for _, m in ipairs(ms) do
							if m.sb > cb then
								return l, col_of_byte(text, m.sb), col_of_byte(text, m.eb + 1), false
							end
						end
					else
						for k = #ms, 1, -1 do
							if ms[k].sb < cb then
								return l, col_of_byte(text, ms[k].sb), col_of_byte(text, ms[k].eb + 1), false
							end
						end
					end
				else
					local m = ms[1]
					if dir < 0 then
						m = ms[#ms]
					end
					return l, col_of_byte(text, m.sb), col_of_byte(text, m.eb + 1), wrapped
				end
			end
		end
		return nil
	end

	-- ---------------------------------------------------------------------------
	-- Search state and the `/`, `?`, `n`, `N`, `*`, `#` commands
	-- ---------------------------------------------------------------------------

	local search = { pat = nil, compiled = nil, dir = 1 }

	local function set_search(pat, compiled, dir)
		search.pat = pat
		search.compiled = compiled
		search.dir = dir
	end

	local function report_wrap(dir)
		if dir > 0 then
			ttt.notify("search hit BOTTOM, continuing at TOP")
		else
			ttt.notify("search hit TOP, continuing at BOTTOM")
		end
	end

	local function not_found(pat)
		ttt.notify("E486: Pattern not found: " .. (pat or ""), "error")
	end

	-- Jump to the next match of the stored pattern. `dir` is already resolved
	-- against `n` vs `N`.
	local function search_step(dir, count)
		if not search.compiled then
			ttt.notify("E35: No previous regular expression", "error")
			return
		end
		local wrapped_any = false
		for _ = 1, count do
			local cur = editor.cursor()
			local l, c1, _, wrapped = search_from(search.compiled, cur.line, cur.col, dir)
			if not l then
				not_found(search.pat)
				return
			end
			if wrapped then
				wrapped_any = true
			end
			set_jump_mark()
			move_to(l, c1)
		end
		if wrapped_any then
			report_wrap(dir)
		end
	end

	-- The word under (or after) the cursor, for `*` and `#`.
	local function word_under_cursor()
		local cur = editor.cursor()
		local r = line_runes(cur.line)
		local i = cur.col
		while i <= #r and class_of(r[i], false) ~= 1 do
			i = i + 1
		end
		if i > #r then
			return nil
		end
		local s = i
		while s > 1 and class_of(r[s - 1], false) == 1 do
			s = s - 1
		end
		local e = i
		while e < #r and class_of(r[e + 1], false) == 1 do
			e = e + 1
		end
		return table.concat(r, "", s, e)
	end

	-- `*` / `#`: search for the word under the cursor, whole-word.
	local function search_word(dir)
		local word = word_under_cursor()
		if not word then
			ttt.notify("E348: No string under cursor", "error")
			return
		end
		-- The word is matched literally; only the boundaries are pattern-ish, and
		-- those ride on the word_start/word_end flags rather than on \< and \>,
		-- which gopher-lua has no %f to compile to.
		local c = compile_vim_pattern(word)
		if not c then
			return
		end
		c.word_start = true
		c.word_end = true
		set_search("\\<" .. word .. "\\>", c, dir)
		local cur = editor.cursor()
		-- Vim starts `*` from the start of the word, so the current one is skipped.
		local r = line_runes(cur.line)
		local s = cur.col
		while s > 1 and class_of(r[s - 1], false) == 1 do
			s = s - 1
		end
		local l, c1, _, wrapped = search_from(c, cur.line, s, dir)
		if not l then
			not_found(search.pat)
			return
		end
		set_jump_mark()
		move_to(l, c1)
		if wrapped then
			report_wrap(dir)
		end
	end

	-- `d/foo<CR>`: the match position is an exclusive charwise motion target.
	local function run_search_operator(start, l, c)
		local op = state.operator.op
		local payload = op_payload(op, 1, { t = "search", pat = search.pat, dir = search.dir })
		clear_operator()
		local sl, sc, el, ec, lw = motion_to_range(start, { line = l, col = c }, "exclusive")
		if sl == nil then
			return
		end
		apply_operator(op, sl, sc, el, ec, lw, payload)
	end

	local function open_search(dir)
		local origin = editor.cursor()
		local had_operator = state.operator ~= nil
		local prefix = "/"
		if dir < 0 then
			prefix = "?"
		end

		local function restore()
			editor.set_cursor(origin.line, origin.col)
		end

		ttt.command_line.show({
			prefix = prefix,
			-- Preview only: cursor and scroll, never the buffer.
			on_change = function(text)
				if text == "" then
					restore()
					return
				end
				local c = compile_vim_pattern(text)
				if not c then
					restore()
					return
				end
				local l, c1 = search_from(c, origin.line, origin.col, dir)
				if l then
					editor.set_cursor(l, c1)
				else
					restore()
				end
			end,
			on_submit = function(text)
				restore()
				if text ~= "" then
					local c, err = compile_vim_pattern(text)
					if not c then
						ttt.notify("vim: " .. (err or "bad pattern"), "error")
						clear_operator()
						return
					end
					set_search(text, c, dir)
				end
				if not search.compiled then
					ttt.notify("E35: No previous regular expression", "error")
					clear_operator()
					return
				end
				search.dir = dir
				local l, c1, _, wrapped = search_from(search.compiled, origin.line, origin.col, dir)
				if not l then
					not_found(search.pat)
					clear_operator()
					return
				end
				if had_operator and state.operator then
					run_search_operator(origin, l, c1)
				else
					set_jump_mark()
					move_to(l, c1)
				end
				if wrapped then
					report_wrap(dir)
				end
			end,
			on_cancel = function()
				restore()
				clamp_cursor()
				clear_operator()
			end,
		})
	end

	-- ---------------------------------------------------------------------------
	-- Substitution
	-- ---------------------------------------------------------------------------

	-- Rebuild a line from the accepted match spans.
	local function apply_spans(text, spans)
		local out, prev = {}, 1
		for _, s in ipairs(spans) do
			out[#out + 1] = text:sub(prev, s.sb - 1)
			out[#out + 1] = s.text
			prev = s.eb + 1
		end
		out[#out + 1] = text:sub(prev)
		return table.concat(out)
	end

	-- One undo group for the whole run, however many lines it touched: `:%s/a/b/g`
	-- across the file is a single `u`.
	local function apply_work(work)
		local count, lines, last_line = 0, 0, nil
		editor.begin_undo_group()
		for _, w in ipairs(work) do
			if #w.spans > 0 then
				local new = apply_spans(w.text, w.spans)
				if new ~= w.text then
					editor.set_line(w.line, new)
					count = count + #w.spans
					lines = lines + 1
					last_line = w.line
				end
			end
		end
		editor.end_undo_group()

		if last_line then
			move_to(last_line, first_non_blank(last_line))
		end
		if count > 0 then
			local s1 = "s"
			if count == 1 then
				s1 = ""
			end
			local s2 = "s"
			if lines == 1 then
				s2 = ""
			end
			ttt.notify(count .. " substitution" .. s1 .. " on " .. lines .. " line" .. s2)
		end
	end

	-- The `c` flag. Every match is located up front and nothing is edited until the
	-- last answer is in, so the positions stay valid and the whole run is still one
	-- undo step. The prompt is the command line itself, and on_change fires on the
	-- first keystroke -- which is how a single-key y/n/a/q answer works.
	local confirm_walk

	confirm_walk = function(work, flat, i)
		if i > #flat then
			for _, w in ipairs(work) do
				local keep = {}
				for _, s in ipairs(w.spans) do
					if not s.skip then
						keep[#keep + 1] = s
					end
				end
				w.spans = keep
			end
			apply_work(work)
			return
		end

		local item = flat[i]
		editor.set_cursor(item.line, col_of_byte(item.text, item.span.sb))

		local function skip_rest()
			for k = i, #flat do
				flat[k].span.skip = true
			end
			ttt.set_timeout(0, function()
				confirm_walk(work, flat, #flat + 1)
			end)
		end

		local answered = false
		ttt.command_line.show({
			prefix = "replace with " .. item.span.text .. " (y/n/a/q)? ",
			on_change = function(text)
				if answered or text == "" then
					return
				end
				answered = true
				local ch = text:sub(1, 1):lower()
				ttt.command_line.hide()
				if ch == "q" then
					skip_rest()
					return
				end
				if ch == "a" then
					ttt.set_timeout(0, function()
						confirm_walk(work, flat, #flat + 1)
					end)
					return
				end
				if ch ~= "y" then
					item.span.skip = true
				end
				ttt.set_timeout(0, function()
					confirm_walk(work, flat, i + 1)
				end)
			end,
			on_cancel = skip_rest,
		})
	end

	-- Split `/pat/rep/flags` on its delimiter, honouring `\` escapes. The delimiter
	-- is whatever character follows the `s`, so `:s#a#b#` works too.
	local function split_spec(spec)
		local delim = spec:sub(1, 1)
		local parts, cur = {}, {}
		local i, n = 2, #spec
		while i <= n do
			local c = spec:sub(i, i)
			if c == "\\" then
				local nx = spec:sub(i + 1, i + 1)
				if nx == delim then
					cur[#cur + 1] = delim
				else
					cur[#cur + 1] = spec:sub(i, i + 1)
				end
				i = i + 2
			elseif c == delim then
				parts[#parts + 1] = table.concat(cur)
				cur = {}
				i = i + 1
			else
				cur[#cur + 1] = c
				i = i + 1
			end
		end
		parts[#parts + 1] = table.concat(cur)
		return parts
	end

	local function do_substitute(first, last, spec)
		local parts = split_spec(spec)
		local pat = parts[1] or ""
		local rep = parts[2] or ""
		local flags = parts[3] or ""

		if pat == "" then
			if not search.pat then
				ttt.notify("E35: No previous regular expression", "error")
				return
			end
			pat = search.pat
		end

		local all = flags:find("g", 1, true) ~= nil
		local ic = flags:find("i", 1, true) ~= nil
		local confirm = flags:find("c", 1, true) ~= nil

		local c, perr = compile_vim_pattern(pat, ic)
		if not c then
			ttt.notify("vim: " .. (perr or "bad pattern"), "error")
			return
		end
		-- `*` and `#` store their pattern with \< \>, which compile_vim_pattern only
		-- accepts at the edges; re-applying the flags keeps `:s//x/` after a `*`.
		if search.pat == pat and search.compiled then
			c.word_start = search.compiled.word_start
			c.word_end = search.compiled.word_end
		end
		local lrep, rerr = compile_replacement(rep)
		if lrep == nil then
			ttt.notify("vim: " .. (rerr or "bad replacement"), "error")
			return
		end
		set_search(pat, c, 1)

		local n = line_count()
		first = clamp(first, 1, n)
		last = clamp(last, 1, n)
		if first > last then
			local t = first
			first = last
			last = t
		end

		local work = {}
		for l = first, last do
			local text = editor.get_line(l) or ""
			local spans = line_matches(text, c, lrep)
			if #spans > 0 then
				if not all then
					while #spans > 1 do
						table.remove(spans)
					end
				end
				work[#work + 1] = { line = l, text = text, spans = spans }
			end
		end

		if #work == 0 then
			not_found(pat)
			return
		end

		if not confirm then
			apply_work(work)
			return
		end

		local flat = {}
		for _, w in ipairs(work) do
			for _, s in ipairs(w.spans) do
				flat[#flat + 1] = { line = w.line, text = w.text, span = s }
			end
		end
		-- Deferred by a tick: this runs inside the ex command line's own on_submit,
		-- and show() is a no-op while another overlay is still on screen.
		ttt.set_timeout(0, function()
			confirm_walk(work, flat, 1)
		end)
	end

	-- ---------------------------------------------------------------------------
	-- Ex commands
	-- ---------------------------------------------------------------------------

	-- One address: {n}, `.`, `$`. Returns the line and the unconsumed rest, or nil.
	local function ex_address(s)
		local d = s:match("^%d+")
		if d then
			return tonumber(d), s:sub(#d + 1)
		end
		local c = s:sub(1, 1)
		if c == "$" then
			return line_count(), s:sub(2)
		end
		if c == "." then
			return editor.cursor().line, s:sub(2)
		end
		return nil, s
	end

	-- A leading range: `%`, `{n}`, `{n},{m}`, `.,$`, ... Returns first, last, rest.
	-- first is nil when the command had no range of its own.
	local function parse_range(s)
		if s:sub(1, 1) == "%" then
			return 1, line_count(), s:sub(2)
		end
		local a, rest = ex_address(s)
		if not a then
			return nil, nil, s
		end
		if rest:sub(1, 1) == "," then
			local b, rest2 = ex_address(rest:sub(2))
			if not b then
				b = editor.cursor().line
			end
			return a, b, rest2
		end
		return a, a, rest
	end

	-- Relative paths are anchored to the directory of the file being edited, not
	-- to the process cwd: ttt is normally launched from somewhere else entirely,
	-- and the plugin filesystem API is scoped to the workspace roots anyway.
	local function resolve_path(path)
		if path:sub(1, 1) == "/" then
			return path
		end
		local here = editor.file_path()
		local dir = nil
		if here then
			dir = here:match("^(.*)/[^/]*$")
		end
		if dir then
			return dir .. "/" .. path
		end
		return path
	end

	local function ex_write(args)
		if args == "" then
			if not ttt.exec_command("file.save") then
				ttt.notify("vim: file.save is not available", "error")
			end
			return
		end
		if not fs or not fs.write then
			ttt.notify("vim: :w {file} needs the fs.write permission", "error")
			return
		end
		local path = resolve_path(args)
		local ok, err = fs.write(path, editor.buffer_text() or "")
		if ok then
			ttt.notify('"' .. path .. '" written')
		else
			ttt.notify("vim: " .. tostring(err or "write failed"), "error")
		end
	end

	local function ex_registers()
		local entries = {}
		local function add(name)
			local r = state.registers[name]
			if not r then
				return
			end
			local text = (r.text or ""):gsub("\n", "\\n")
			entries[#entries + 1] = { key = '"' .. name, value = (r.kind or "char") .. "  " .. text }
		end
		add('"')
		for i = 0, 9 do
			add(tostring(i))
		end
		add("-")
		local named = {}
		for name in pairs(state.registers) do
			if name:match("^%l$") then
				named[#named + 1] = name
			end
		end
		table.sort(named)
		for _, name in ipairs(named) do
			add(name)
		end
		if #entries == 0 then
			entries[#entries + 1] = { key = "", value = "no registers set" }
		end
		ttt.show_info("Registers", entries)
	end

	local function ex_marks()
		local names = {}
		for name in pairs(state.marks) do
			names[#names + 1] = name
		end
		table.sort(names)
		local entries = {}
		for _, name in ipairs(names) do
			local m = state.marks[name]
			local text = editor.get_line(clamp(m.line, 1, line_count())) or ""
			entries[#entries + 1] = { key = name, value = m.line .. "," .. m.col .. "  " .. text }
		end
		if #entries == 0 then
			entries[#entries + 1] = { key = "", value = "no marks set" }
		end
		ttt.show_info("Marks", entries)
	end

	local function exec_or_warn(id)
		if not ttt.exec_command(id) then
			ttt.notify("vim: " .. id .. " is not available", "error")
		end
	end

	local function run_ex(text)
		local cmd = text:gsub("^%s+", ""):gsub("%s+$", "")
		if cmd == "" then
			return
		end

		local first, last, rest = parse_range(cmd)
		rest = rest:gsub("^%s+", "")

		-- A bare range is a jump: `:12`, `:$`, `:1,5` lands on the last address.
		if rest == "" then
			if first then
				set_jump_mark()
				goto_line(last)
			end
			return
		end

		-- Substitution has to be recognised before the command-name match, because
		-- its delimiter can be any punctuation: `s/a/b/`, `s#a#b#`.
		if rest:sub(1, 1) == "s" then
			local delim = rest:sub(2, 2)
			if delim ~= "" and not delim:match("[%w%s]") then
				local cur = editor.cursor().line
				return do_substitute(first or cur, last or cur, rest:sub(2))
			end
		end

		local name, args = rest:match("^(%a+!?)%s*(.*)$")
		if not name then
			name = rest
			args = ""
		end
		local bang = false
		if name:sub(-1) == "!" then
			bang = true
			name = name:sub(1, -2)
		end

		if name == "w" or name == "write" then
			ex_write(args)
		elseif name == "q" or name == "quit" then
			if bang then
				ttt.quit()
			else
				exec_or_warn("editor.quit")
			end
		elseif name == "wq" or name == "x" or name == "xit" then
			ex_write(args)
			if bang then
				ttt.quit()
			else
				exec_or_warn("editor.quit")
			end
		elseif name == "e" or name == "edit" then
			if args == "" then
				ttt.notify("vim: :e needs a file name", "error")
			else
				ttt.open_file(resolve_path(args))
			end
		elseif name == "noh" or name == "nohl" or name == "nohlsearch" then
			exec_or_warn("search.clearFind")
		elseif name == "reg" or name == "registers" or name == "display" then
			ex_registers()
		elseif name == "marks" then
			ex_marks()
		else
			ttt.notify("E492: Not an editor command: " .. name, "error")
		end
	end

	local function open_ex()
		ttt.command_line.show({
			prefix = ":",
			on_submit = run_ex,
		})
	end

	-- ---------------------------------------------------------------------------
	-- Editor integration
	--
	-- Everything here delegates to a core command. Keys with no core equivalent
	-- (]c, [c, ]f, [f, zo, zc, Ctrl-W splits) are deliberately absent rather than
	-- approximated -- see README.md.
	-- ---------------------------------------------------------------------------

	local Z_FOLDS = {
		["a"] = "fold.toggle",
		["R"] = "fold.expandAll",
		["M"] = "fold.collapseAll",
	}

	local CTRL_W_COMMANDS = {
		["w"] = "focus.nextGroup",
		["W"] = "focus.prevGroup",
		["ctrl-w"] = "focus.nextGroup",
	}

	-- `]`/`[` bracket-command targets. `]c`/`[c` jump between changed hunks in the
	-- current file; `]f`/`[f` cycle through the files in the Changes panel. These
	-- delegate to core commands; a false return (command absent) is a silent
	-- no-op. Assigned straight onto vim6 (no locals) to stay under Lua 5.1's
	-- per-function local cap, which this do-block is already close to.
	vim6.BRACKET_NEXT = {
		["c"] = "diff.nextHunk",
		["f"] = "changes.nextFile",
	}
	vim6.BRACKET_PREV = {
		["c"] = "diff.prevHunk",
		["f"] = "changes.prevFile",
	}

	G_MOTIONS["t"] = function(n)
		for _ = 1, n do
			exec_or_warn("tab.next")
		end
	end

	G_MOTIONS["T"] = function(n)
		for _ = 1, n do
			exec_or_warn("tab.prev")
		end
	end

	vim6.search = search
	vim6.search_from = search_from
	vim6.search_step = search_step
	vim6.search_word = search_word
	vim6.open_search = open_search
	vim6.run_search_operator = run_search_operator
	vim6.open_ex = open_ex
	vim6.exec_or_warn = exec_or_warn
	vim6.Z_FOLDS = Z_FOLDS
	vim6.CTRL_W_COMMANDS = CTRL_W_COMMANDS
end

local function run_operator_payload(p, count)
	local n = count or p.count or 1
	state.operator = { op = p.op, count = n }
	state.count = nil
	state.register = p.register
	local t = p.target or {}
	if t.t == "linewise" then
		run_linewise_operator()
	elseif t.t == "motion" then
		run_motion_operator(t.tok, t.gprefix)
	elseif t.t == "textobj" then
		run_textobject(t.inner, t.tok)
	elseif t.t == "find" then
		run_find_operator(t.op, t.ch)
	elseif t.t == "mark" then
		run_mark_operator(t.name, t.linewise)
	elseif t.t == "search" then
		-- `.` after `d/foo` re-runs the search from wherever the cursor is now,
		-- which is what Vim does too.
		local start = editor.cursor()
		local l, c1
		if vim6.search.compiled then
			l, c1 = vim6.search_from(vim6.search.compiled, start.line, start.col, t.dir or 1)
		end
		if l then
			vim6.run_search_operator(start, l, c1)
		else
			clear_operator()
		end
	else
		clear_operator()
	end
	if p.op == "c" and state.mode == "insert" then
		replay_insert("c", p.text, 1)
	end
	state.register = nil
end

local function run_change(p, count)
	if not p then
		return
	end
	state.replaying = true
	pcall(function()
		if p.kind == "edit" then
			state.register = p.register
			local fn = EDITS[p.tok]
			if fn then
				fn(count or p.count or 1, true)
			end
			state.register = nil
		elseif p.kind == "replace_char" then
			do_replace_char(count or p.count or 1, p.ch)
		elseif p.kind == "replace_mode" then
			EDITS["R"](1)
			overtype(p.text or "")
			leave_insert()
		elseif p.kind == "insert" then
			local n = count or p.count or 1
			local fn = EDITS[p.tok]
			if fn then
				fn(n, true)
				replay_insert(p.tok, p.text, REPEAT_TEXT[p.tok] and n or 1)
			end
		elseif p.kind == "operator" then
			run_operator_payload(p, count)
		end
	end)
	state.replaying = false
end

-- ---------------------------------------------------------------------------
-- Macros
--
-- Recording captures canonical tokens, not raw events, so a replay is literally
-- "feed the tokens back through dispatch()". state.macro.playing is a depth
-- counter: it stops a replay from being recorded into the macro being recorded,
-- and caps `@a` calling itself. A key budget caps the other runaway shape, a
-- macro that loops forever without recursing.
-- ---------------------------------------------------------------------------

local MACRO_MAX_DEPTH = 10
local MACRO_MAX_KEYS = 20000

local HANDLERS -- assigned once the mode handlers exist
local dispatch -- forward declaration; play_macro re-enters it

local function start_recording(name)
	state.macro.recording = name
	state.macro.keys = {}
	render_status()
end

local function stop_recording()
	local name = state.macro.recording
	if not name then
		return
	end
	local slot = name:lower()
	if name:match("^%u$") and state.macros[slot] then
		local dst = state.macros[slot]
		for _, t in ipairs(state.macro.keys) do
			dst[#dst + 1] = t
		end
	else
		state.macros[slot] = state.macro.keys
	end
	state.macro.recording = nil
	state.macro.keys = {}
	render_status()
end

local function play_macro(name, count)
	local keys = state.macros[name:lower()]
	if not keys or #keys == 0 then
		return
	end
	if state.macro.playing >= MACRO_MAX_DEPTH then
		return
	end
	state.macro.last = name:lower()
	if state.macro.playing == 0 then
		state.macro.budget = MACRO_MAX_KEYS
	end
	state.macro.playing = state.macro.playing + 1
	-- A Lua error mid-replay would otherwise be swallowed by the key.press
	-- listener and leave the depth counter stuck.
	pcall(function()
		for _ = 1, count do
			for _, tok in ipairs(keys) do
				state.macro.budget = state.macro.budget - 1
				if state.macro.budget <= 0 then
					error("vim: macro key budget exhausted")
				end
				dispatch(tok)
			end
		end
	end)
	state.macro.playing = state.macro.playing - 1
end

-- ---------------------------------------------------------------------------
-- Mode handlers
-- ---------------------------------------------------------------------------

local function handle_insert(tok)
	if ESCAPE_TOKENS[tok] then
		leave_insert()
		return true
	end
	return false
end

-- R: overtype until Esc. state.replace_stack remembers what each keystroke
-- overwrote, so backspace restores the original character the way Vim does --
-- and, past the start of the session, just walks left.
local function handle_replace(tok)
	if ESCAPE_TOKENS[tok] then
		leave_insert()
		return true
	end
	if tok == "backspace" or tok == "backspace2" then
		local stack = state.replace_stack
		local top = stack and stack[#stack] or nil
		if top then
			table.remove(stack)
			if top.orig then
				editor.replace(top.line, top.col, top.line, top.col + 1, top.orig)
			else
				editor.replace(top.line, top.col, top.line, top.col + 1, "")
			end
			editor.set_cursor(top.line, top.col)
			return true
		end
		local cur = editor.cursor()
		if cur.col > 1 then
			editor.set_cursor(cur.line, cur.col - 1)
		end
		return true
	end
	if is_printable(tok) then
		local cur = editor.cursor()
		local stack = state.replace_stack
		if cur.col <= line_len(cur.line) then
			if stack then
				stack[#stack + 1] = { line = cur.line, col = cur.col, orig = line_runes(cur.line)[cur.col] }
			end
			editor.replace(cur.line, cur.col, cur.line, cur.col + 1, tok)
		else
			if stack then
				stack[#stack + 1] = { line = cur.line, col = cur.col, orig = nil }
			end
			editor.insert(cur.line, cur.col, tok)
		end
		editor.set_cursor(cur.line, cur.col + 1)
		return true
	end
	return false
end

-- Single-key edits that change the buffer without entering insert mode, so
-- their `.` payload is complete the moment they run.
local SIMPLE_CHANGES = {
	["x"] = true,
	["X"] = true,
	["D"] = true,
	["~"] = true,
	["J"] = true,
	["p"] = true,
	["P"] = true,
	["ctrl-a"] = true,
	["ctrl-x"] = true,
}

local function handle_normal(tok)
	if ESCAPE_TOKENS[tok] then
		if not has_pending() then
			return false
		end
		set_mode("normal")
		return true
	end

	-- r{char} consumes the next key.
	if state.replace_pending then
		local n = state.replace_pending
		state.replace_pending = nil
		if is_printable(tok) then
			record_change({ kind = "replace_char", count = n, ch = tok })
			do_replace_char(n, tok)
		end
		return true
	end

	if state.await_register then
		state.await_register = false
		if is_printable(tok) then
			state.register = tok
		end
		return true
	end

	-- q{a-z} / @{a-z} / @@ consume the next key as a register name.
	if state.await_macro then
		local what = state.await_macro
		state.await_macro = nil
		local n = take_count()
		if what == "q" then
			if tok:match("^%a$") then
				start_recording(tok)
			end
		elseif tok == "@" then
			if state.macro.last then
				play_macro(state.macro.last, n)
			end
		elseif tok:match("^%a$") then
			play_macro(tok, n)
		end
		return true
	end

	-- m{name} sets a mark; `{name} and '{name} jump to one, and are valid
	-- operator targets.
	if state.mark_pending then
		local what = state.mark_pending
		state.mark_pending = nil
		if not tok:match(MARK_NAME) then
			clear_operator()
			return true
		end
		if what == "m" then
			local cur = editor.cursor()
			mark_set(tok, cur.line, cur.col)
		elseif state.operator then
			run_mark_operator(tok, what == "'")
		else
			state.count = nil
			goto_mark(tok, what == "'")
		end
		return true
	end

	-- f/F/t/T consume the very next key as their target, whatever it is.
	if state.find_pending then
		local op = state.find_pending
		state.find_pending = nil
		if not is_printable(tok) then
			clear_operator()
			return true
		end
		state.last_find = { op = op, ch = tok }
		if state.operator then
			run_find_operator(op, tok)
		else
			find_char(op, tok, take_count(), false)
		end
		return true
	end

	-- `i`/`a` after an operator select a text object rather than entering
	-- insert mode.
	if state.textobj then
		local inner = state.textobj == "i"
		state.textobj = nil
		if state.operator then
			run_textobject(inner, tok)
		end
		return true
	end

	if state.pending == "g" then
		state.pending = ""
		if state.operator then
			if OP_DOUBLE[state.operator.op][tok] then
				run_linewise_operator()
			else
				run_motion_operator(tok, true)
			end
			return true
		end
		local gop = G_OPERATORS[tok]
		if gop then
			start_operator(gop)
			return true
		end
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
		elseif vim6.Z_FOLDS[tok] then
			vim6.exec_or_warn(vim6.Z_FOLDS[tok])
		end
		return true
	end

	-- Ctrl-W is a prefix in normal mode, so core's `tab.close` binding for it is
	-- overridden -- documented in the README alongside Ctrl-V and friends.
	if state.pending == "ctrl-w" then
		state.pending = ""
		state.count = nil
		if vim6.CTRL_W_COMMANDS[tok] then
			vim6.exec_or_warn(vim6.CTRL_W_COMMANDS[tok])
		end
		return true
	end

	-- `]`/`[` bracket commands (]c/[c hunks, ]f/[f changed files). A missing core
	-- command returns false and is a silent no-op. No new locals: handle_normal is
	-- large and near Lua 5.1's per-function local cap.
	if state.pending == "]" then
		state.pending = ""
		state.count = nil
		if vim6.BRACKET_NEXT[tok] then
			ttt.exec_command(vim6.BRACKET_NEXT[tok])
		end
		return true
	end
	if state.pending == "[" then
		state.pending = ""
		state.count = nil
		if vim6.BRACKET_PREV[tok] then
			ttt.exec_command(vim6.BRACKET_PREV[tok])
		end
		return true
	end

	-- Count prefix. A leading `0` is the motion, not a count digit.
	if #tok == 1 and tok >= "0" and tok <= "9" and not (tok == "0" and state.count == nil) then
		state.count = (state.count or 0) * 10 + tonumber(tok)
		return true
	end

	if tok == '"' then
		state.await_register = true
		return true
	end

	if tok == "`" or tok == "'" then
		state.mark_pending = tok
		return true
	end

	if not state.operator then
		if tok == "m" then
			state.mark_pending = "m"
			return true
		end
		if tok == "q" then
			if state.macro.recording then
				stop_recording()
			else
				state.await_macro = "q"
			end
			return true
		end
		if tok == "@" then
			state.await_macro = "@"
			return true
		end
		if tok == "." then
			local n, had = take_count()
			run_change(state.last_change, had and n or nil)
			return true
		end
		if tok == "n" or tok == "N" then
			local n = take_count()
			local dir = vim6.search.dir
			if tok == "N" then
				dir = -dir
			end
			vim6.search_step(dir, n)
			return true
		end
		if tok == "*" or tok == "#" then
			state.count = nil
			local dir = 1
			if tok == "#" then
				dir = -1
			end
			vim6.search_word(dir)
			return true
		end
		if tok == ":" then
			state.count = nil
			vim6.open_ex()
			return true
		end
		if tok == "ctrl-w" then
			state.pending = "ctrl-w"
			return true
		end
	end

	-- `/` and `?` are motions, so they are also operator targets: `d/foo<CR>`
	-- deletes up to the match. The operator stays pending across the modal
	-- command line -- no keys reach the plugin while it is open.
	if tok == "/" or tok == "?" then
		local dir = 1
		if tok == "?" then
			dir = -1
		end
		vim6.open_search(dir)
		return true
	end

	-- Operator-pending. This has to precede the motion and edit tables: `d` then
	-- `i` is "inner text object", not "enter insert mode".
	if state.operator then
		if OP_DOUBLE[state.operator.op][tok] then
			run_linewise_operator()
			return true
		end
		if tok == "i" or tok == "a" then
			state.textobj = tok
			return true
		end
		if tok == "g" then
			state.pending = "g"
			return true
		end
		if FIND_KEYS[tok] then
			state.find_pending = tok
			return true
		end
		run_motion_operator(tok, false)
		return true
	end

	if OPERATORS[tok] then
		start_operator(tok)
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

	local edit = EDITS[tok]
	if edit then
		local reg = state.register
		local n, had = take_count()
		-- Insert-entry edits record their payload in leave_insert() instead,
		-- once the typed text is known.
		if SIMPLE_CHANGES[tok] then
			record_change({ kind = "edit", tok = tok, count = n, register = reg })
		end
		edit(n, had)
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

-- Visual-mode entry points, appended to the normal-mode tables. Core binds
-- Ctrl-V to editor.paste; normal mode overrides it, as documented in the README.
EDITS["v"] = function()
	enter_visual("visual")
end
EDITS["V"] = function()
	enter_visual("visual_line")
end
EDITS["ctrl-v"] = function()
	enter_visual("visual_block")
end

G_MOTIONS["v"] = function()
	reselect_visual()
end

HANDLERS = {
	normal = handle_normal,
	insert = handle_insert,
	replace = handle_replace,
	visual = handle_visual,
	visual_line = handle_visual,
	visual_block = handle_visual,
}

-- The single entry point for a canonical token, whether it came from a real
-- keystroke or from a macro replay. Everything that needs to observe keys --
-- macro recording, the insert-mode change log -- hangs off here rather than off
-- the individual handlers.
dispatch = function(tok)
	local handler = HANDLERS[state.mode]
	if not handler then
		return false
	end

	-- `q` stops recording, and must not be recorded as part of the macro.
	if state.macro.recording and state.mode == "normal" and tok == "q" and not has_pending() then
		stop_recording()
		return true
	end

	local ctx = state.insert_ctx
	local typing = (state.mode == "insert" or state.mode == "replace")
	-- Sampled before the handler runs: the `a` of `qa` starts the recording and
	-- must not become the macro's first key.
	local was_recording = state.macro.recording

	local handled = handler(tok)

	-- Insert mode passes printable keys through to the editor, which types them.
	-- During a macro replay there is no editor doing that, so this does it.
	if not handled and typing and state.macro.playing > 0 then
		local txt = token_text(tok)
		if txt then
			local cur = editor.cursor()
			local l, c = insert_and_advance(cur.line, cur.col, txt)
			editor.set_cursor(l, c)
			handled = true
		end
	end

	-- Log the typed text for `.` and for `{count}i`. A key that cannot be
	-- rendered as text makes the session non-repeatable rather than wrong.
	if ctx and typing and not ESCAPE_TOKENS[tok] then
		if tok == "backspace" or tok == "backspace2" then
			if #ctx.keys > 0 then
				table.remove(ctx.keys)
			else
				ctx.dirty = true
			end
		else
			local txt = token_text(tok)
			if txt then
				ctx.keys[#ctx.keys + 1] = txt
			else
				ctx.dirty = true
			end
		end
	end

	if was_recording and state.macro.recording and state.macro.playing == 0 and (handled or typing) then
		local keys = state.macro.keys
		keys[#keys + 1] = tok
	end

	return handled
end

local function on_key(ev)
	if not state.enabled then
		return false
	end
	return dispatch(token_of(ev))
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

-- Deferred startup: read settings, then draw the initial status indicator. Both
-- the settings API and set_status_item are only wired after WirePlugin runs
-- (which is after this file executes), so this is delayed by a tick. get_setting
-- lives inside the closure so it consumes no permanent top-level local (Lua 5.1
-- caps the main chunk at 200). Reads are defensive: ttt.settings may be
-- unavailable or return nil, in which case the default stands, and a denied key
-- raises a Lua error, so each call is pcall'd. vim.enabled=false starts with Vim
-- mode off (re-enable from the command palette); vim.clipboard=true mirrors the
-- unnamed register to the system clipboard on every yank/delete.
ttt.set_timeout(0, function()
	local function get_setting(key, default)
		local ok, mod = pcall(require, "ttt.settings")
		if not ok or type(mod) ~= "table" or type(mod.get) ~= "function" then
			return default
		end
		local ok2, val = pcall(mod.get, key)
		if not ok2 or val == nil then
			return default
		end
		return val
	end

	if not get_setting("vim.enabled", true) then
		disable()
	end
	state.clipboard = get_setting("vim.clipboard", false) and true or false
	render_status()
end)
