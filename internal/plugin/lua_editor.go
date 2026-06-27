package plugin

import (
	lua "github.com/yuin/gopher-lua"
)

func setupEditorModule(L *lua.LState, p *Plugin) {
	loader := func(L *lua.LState) int {
		mod := L.NewTable()

		hasRead := p.Granted.Check("editor.read") == nil
		hasWrite := p.Granted.Check("editor.write") == nil

		if hasRead {
			L.SetField(mod, "buffer_text", L.NewFunction(editorBufferText(p)))
			L.SetField(mod, "buffer_lines", L.NewFunction(editorBufferLines(p)))
			L.SetField(mod, "current_line", L.NewFunction(editorCurrentLine(p)))
			L.SetField(mod, "cursor", L.NewFunction(editorCursor(p)))
			L.SetField(mod, "selection", L.NewFunction(editorSelection(p)))
			L.SetField(mod, "selection_text", L.NewFunction(editorSelectionText(p)))
			L.SetField(mod, "file_path", L.NewFunction(editorFilePath(p)))
			L.SetField(mod, "file_name", L.NewFunction(editorFileName(p)))
			L.SetField(mod, "language", L.NewFunction(editorLanguage(p)))
		}

		if hasWrite {
			L.SetField(mod, "insert", L.NewFunction(editorInsert(p)))
			L.SetField(mod, "replace", L.NewFunction(editorReplace(p)))
			L.SetField(mod, "set_cursor", L.NewFunction(editorSetCursor(p)))
			L.SetField(mod, "set_selection", L.NewFunction(editorSetSelection(p)))
			L.SetField(mod, "clear_selection", L.NewFunction(editorClearSelection(p)))
		}

		L.Push(mod)
		return 1
	}

	L.PreloadModule("ttt.editor", loader)
}

func editorBufferText(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(p.Editor.BufferText()))
		return 1
	}
}

func editorBufferLines(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(L.NewTable())
			return 1
		}
		lines := p.Editor.BufferLines()
		tbl := L.NewTable()
		for _, line := range lines {
			tbl.Append(lua.LString(line))
		}
		L.Push(tbl)
		return 1
	}
}

func editorCurrentLine(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(p.Editor.CurrentLine()))
		return 1
	}
}

func editorCursor(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			tbl := L.NewTable()
			L.SetField(tbl, "line", lua.LNumber(1))
			L.SetField(tbl, "col", lua.LNumber(1))
			L.Push(tbl)
			return 1
		}
		line, col := p.Editor.CursorPos()
		tbl := L.NewTable()
		L.SetField(tbl, "line", lua.LNumber(line+1))
		L.SetField(tbl, "col", lua.LNumber(col+1))
		L.Push(tbl)
		return 1
	}
}

func editorSelection(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		tbl := L.NewTable()
		if p.Editor == nil {
			L.SetField(tbl, "active", lua.LFalse)
			L.Push(tbl)
			return 1
		}
		active, sl, sc, el, ec := p.Editor.Selection()
		L.SetField(tbl, "active", lua.LBool(active))
		L.SetField(tbl, "start_line", lua.LNumber(sl+1))
		L.SetField(tbl, "start_col", lua.LNumber(sc+1))
		L.SetField(tbl, "end_line", lua.LNumber(el+1))
		L.SetField(tbl, "end_col", lua.LNumber(ec+1))
		L.Push(tbl)
		return 1
	}
}

func editorSelectionText(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(p.Editor.SelectionText()))
		return 1
	}
}

func editorFilePath(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(p.Editor.FilePath()))
		return 1
	}
}

func editorFileName(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(p.Editor.FileName()))
		return 1
	}
}

func editorLanguage(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(p.Editor.Language()))
		return 1
	}
}

func editorInsert(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			return 0
		}
		line := int(L.CheckNumber(1)) - 1
		col := int(L.CheckNumber(2)) - 1
		text := L.CheckString(3)
		p.Editor.Insert(line, col, text)
		return 0
	}
}

func editorReplace(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			return 0
		}
		sl := int(L.CheckNumber(1)) - 1
		sc := int(L.CheckNumber(2)) - 1
		el := int(L.CheckNumber(3)) - 1
		ec := int(L.CheckNumber(4)) - 1
		text := L.CheckString(5)
		p.Editor.Replace(sl, sc, el, ec, text)
		return 0
	}
}

func editorSetCursor(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			return 0
		}
		line := int(L.CheckNumber(1)) - 1
		col := int(L.CheckNumber(2)) - 1
		p.Editor.SetCursor(line, col)
		return 0
	}
}

func editorSetSelection(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			return 0
		}
		sl := int(L.CheckNumber(1)) - 1
		sc := int(L.CheckNumber(2)) - 1
		el := int(L.CheckNumber(3)) - 1
		ec := int(L.CheckNumber(4)) - 1
		p.Editor.SetSelection(sl, sc, el, ec)
		return 0
	}
}

func editorClearSelection(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.Editor == nil {
			return 0
		}
		p.Editor.ClearSelection()
		return 0
	}
}
