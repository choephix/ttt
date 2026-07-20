-- Vim compatibility layer for ttt.
--
-- Single-file by necessity: the plugin sandbox strips package.loaders down to
-- the preload loader (internal/plugin/sandbox.go), so a plugin cannot require
-- sibling .lua files. Sections below are delimited and appended phase by phase.
--
-- Phase 0: mode state machine (normal/insert), Esc handling, status indicator.
-- Phase 1: normal-mode motions with {count} prefixes.
-- Phase 2: insert-mode entry points and single-key edits.

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
	operator = nil, -- pending operator, { op = "d", count = n }
	textobj = nil, -- "i" or "a" awaiting its object key
	register = nil, -- pending "x register prefix
	last_change = nil, -- replay payload for `.`
	registers = {},
	marks = {},
	macro = { recording = nil, keys = {}, playing = false },
	find_pending = nil, -- "f" | "F" | "t" | "T" awaiting its target char
	last_find = nil, -- { op = "f", ch = "x" } for `;` and `,`
	goal = nil, -- sticky column for j/k
	replace_pending = nil, -- count for `r{char}` awaiting its replacement char
	last_insert = nil, -- { line, col } where insert mode was last left, for `gi`
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
	state.textobj = nil
	state.register = nil
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

-- The caller must already have opened the undo group.
local function begin_insert()
	set_mode("insert")
end

-- Leaving insert/replace mode. Vim steps the cursor one column left and drops
-- it back onto a character; the position *before* that step is what `gi`
-- resumes from.
local function leave_insert()
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
-- Text extraction and the unnamed register
--
-- Phase 3 keeps exactly one register: the unnamed `"`. Named registers, the
-- numbered ring and `"x` prefixes arrive in Phase 5. A register entry is
-- { text, linewise }; linewise text always carries its trailing newline so a
-- paste can be replayed verbatim.
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

local function set_register(text, linewise)
	state.registers['"'] = { text = text, linewise = linewise and true or false }
end

local function yank_charwise(sl, sc, el, ec)
	set_register(charwise_text(sl, sc, el, ec), false)
end

local function yank_linewise(sl, el)
	set_register(linewise_text(sl, el), true)
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
local function change_lines(sl, el)
	local n = line_count()
	sl = clamp(sl, 1, n)
	el = clamp(el, sl, n)
	local indent = (editor.get_line(sl) or ""):match("^[ \t]*") or ""
	yank_linewise(sl, el)
	editor.replace(sl, 1, el, line_len(el) + 1, indent)
	editor.set_cursor(sl, #runes_of(indent) + 1)
	begin_insert()
end

-- Delete from the cursor to the end of line (cur.line + n - 1), joining the
-- lines in between. `2D` on "aaa"/"bbb" leaves a single empty line, as in Vim.
local function delete_to_end(n)
	local cur = editor.cursor()
	local last = clamp(cur.line + n - 1, 1, line_count())
	yank_charwise(cur.line, cur.col, last, line_len(last) + 1)
	editor.replace(cur.line, cur.col, last, line_len(last) + 1, "")
end

-- p / P. A linewise register lands on a new line below/above; a charwise one
-- lands after/before the cursor. Multi-line charwise text leaves the cursor at
-- the first pasted character, single-line text on its last, both as in Vim.
local function paste(count, after)
	local reg = state.registers['"']
	if not reg or reg.text == "" then
		return
	end
	local cur = editor.cursor()
	editor.begin_undo_group()
	if reg.linewise then
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

-- Single-key normal-mode edits. Called as fn(count, had_count).
local EDITS = {
	["i"] = function()
		editor.begin_undo_group()
		begin_insert()
	end,
	["I"] = function()
		local l = editor.cursor().line
		move_to(l, first_non_blank(l))
		editor.begin_undo_group()
		begin_insert()
	end,
	["a"] = function()
		local cur = editor.cursor()
		editor.set_cursor(cur.line, math.min(cur.col + 1, line_len(cur.line) + 1))
		editor.begin_undo_group()
		begin_insert()
	end,
	["A"] = function()
		local cur = editor.cursor()
		editor.set_cursor(cur.line, line_len(cur.line) + 1)
		editor.begin_undo_group()
		begin_insert()
	end,
	-- Insert rejects line >= #Lines (internal/app/plugin_api.go), so `o` on the
	-- last line appends the newline to the *end* of that line rather than
	-- addressing the line after it.
	["o"] = function()
		local l = editor.cursor().line
		editor.begin_undo_group()
		editor.insert(l, line_len(l) + 1, "\n")
		editor.set_cursor(l + 1, 1)
		begin_insert()
	end,
	["O"] = function()
		local l = editor.cursor().line
		editor.begin_undo_group()
		editor.insert(l, 1, "\n")
		editor.set_cursor(l, 1)
		begin_insert()
	end,

	["x"] = function(n)
		local cur = editor.cursor()
		local last = math.min(cur.col + n, line_len(cur.line) + 1)
		if last <= cur.col then
			return
		end
		editor.begin_undo_group()
		yank_charwise(cur.line, cur.col, cur.line, last)
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
		yank_charwise(cur.line, start, cur.line, cur.col)
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
		begin_insert()
	end,
	["s"] = function(n)
		local cur = editor.cursor()
		local last = math.min(cur.col + n, line_len(cur.line) + 1)
		editor.begin_undo_group()
		if last > cur.col then
			yank_charwise(cur.line, cur.col, cur.line, last)
			editor.replace(cur.line, cur.col, cur.line, last, "")
		end
		begin_insert()
	end,
	["S"] = function(n)
		local cur = editor.cursor()
		editor.begin_undo_group()
		change_lines(cur.line, cur.line + n - 1)
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
local function apply_operator(op, sl, sc, el, ec, linewise)
	local n = line_count()
	sl = clamp(sl, 1, n)
	el = clamp(el, 1, n)

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
			yank_linewise(sl, el)
			delete_lines(sl, el)
			local l = math.min(sl, line_count())
			move_to(l, first_non_blank(l))
		else
			yank_charwise(sl, sc, el, ec)
			editor.replace(sl, sc, el, ec, "")
			editor.set_cursor(sl, sc)
			clamp_cursor()
		end
		editor.end_undo_group()
		return
	end

	if op == "c" then
		if linewise then
			change_lines(sl, el)
		else
			yank_charwise(sl, sc, el, ec)
			editor.replace(sl, sc, el, ec, "")
			editor.set_cursor(sl, sc)
			begin_insert()
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
	fn(n, had_count)
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

-- Doubled key: `dd`, `>>`, `guu`. Operates on `count` whole lines from the
-- cursor down.
local function run_linewise_operator()
	local op = state.operator.op
	local n = operator_count()
	local sl = editor.cursor().line
	local el = clamp(sl + n - 1, 1, line_count())
	clear_operator()
	apply_operator(op, sl, 1, el, 1, true)
end

local function run_motion_operator(tok, gprefix)
	local op = state.operator.op
	local n, had = operator_count()
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
	apply_operator(op, sl, sc, el, ec, linewise)
end

local function run_find_operator(op_char, ch)
	local op = state.operator.op
	local n = operator_count()
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
	apply_operator(op, sl, sc, el, ec, linewise)
end

local function run_textobject(inner, tok)
	local op = state.operator.op
	local n = operator_count()
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
	apply_operator(op, sl, sc, el, ec, linewise)
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
-- Mode handlers
-- ---------------------------------------------------------------------------

local function handle_insert(tok)
	if ESCAPE_TOKENS[tok] then
		leave_insert()
		return true
	end
	return false
end

-- R: overtype until Esc. Backspace only walks left -- Vim restores the
-- overwritten characters, which needs the per-keystroke change log that arrives
-- with `.` repeat in Phase 5.
local function handle_replace(tok)
	if ESCAPE_TOKENS[tok] then
		leave_insert()
		return true
	end
	if tok == "backspace" or tok == "backspace2" then
		local cur = editor.cursor()
		if cur.col > 1 then
			editor.set_cursor(cur.line, cur.col - 1)
		end
		return true
	end
	if is_printable(tok) then
		local cur = editor.cursor()
		if cur.col <= line_len(cur.line) then
			editor.replace(cur.line, cur.col, cur.line, cur.col + 1, tok)
		else
			editor.insert(cur.line, cur.col, tok)
		end
		editor.set_cursor(cur.line, cur.col + 1)
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

	-- r{char} consumes the next key. Vim refuses the whole operation when the
	-- count runs past the end of the line rather than replacing fewer chars.
	if state.replace_pending then
		local n = state.replace_pending
		state.replace_pending = nil
		if is_printable(tok) then
			local cur = editor.cursor()
			if cur.col + n - 1 <= line_len(cur.line) then
				editor.begin_undo_group()
				editor.replace(cur.line, cur.col, cur.line, cur.col + n, tok:rep(n))
				editor.end_undo_group()
				editor.set_cursor(cur.line, cur.col + n - 1)
			end
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
		end
		return true
	end

	-- Count prefix. A leading `0` is the motion, not a count digit.
	if #tok == 1 and tok >= "0" and tok <= "9" and not (tok == "0" and state.count == nil) then
		state.count = (state.count or 0) * 10 + tonumber(tok)
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
		local n, had = take_count()
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

local HANDLERS = {
	normal = handle_normal,
	insert = handle_insert,
	replace = handle_replace,
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
