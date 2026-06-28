package plugin

import (
	lua "github.com/yuin/gopher-lua"
)

func setupSystemModule(L *lua.LState, p *Plugin) {
	loader := func(L *lua.LState) int {
		mod := L.NewTable()

		hasExec := len(p.Granted.SystemExec) > 0
		hasEnv := p.Granted.Check("system.env") == nil

		if hasExec {
			L.SetField(mod, "exec", L.NewFunction(sysExec(p)))
			L.SetField(mod, "exec_async", L.NewFunction(sysExecAsync(p)))
		}

		if hasEnv {
			L.SetField(mod, "env", L.NewFunction(sysEnv(p)))
		}

		L.Push(mod)
		return 1
	}

	L.PreloadModule("ttt.system", loader)
}

func sysExec(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.System == nil {
			L.ArgError(1, "system API not available")
			return 0
		}
		binary := L.CheckString(1)
		if err := p.Granted.CheckExec(binary); err != nil {
			L.ArgError(1, err.Error())
			return 0
		}

		var args []string
		if argsTbl, ok := L.Get(2).(*lua.LTable); ok {
			argsTbl.ForEach(func(_, v lua.LValue) {
				args = append(args, v.String())
			})
		}

		stdout, stderr, exitCode, err := p.System.Exec(binary, args)
		if err != nil {
			tbl := L.NewTable()
			L.SetField(tbl, "stdout", lua.LString(""))
			L.SetField(tbl, "stderr", lua.LString(err.Error()))
			L.SetField(tbl, "exit_code", lua.LNumber(-1))
			L.Push(tbl)
			return 1
		}

		tbl := L.NewTable()
		L.SetField(tbl, "stdout", lua.LString(stdout))
		L.SetField(tbl, "stderr", lua.LString(stderr))
		L.SetField(tbl, "exit_code", lua.LNumber(exitCode))
		L.Push(tbl)
		return 1
	}
}

func sysExecAsync(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.System == nil {
			L.ArgError(1, "system API not available")
			return 0
		}
		binary := L.CheckString(1)
		if err := p.Granted.CheckExec(binary); err != nil {
			L.ArgError(1, err.Error())
			return 0
		}

		var args []string
		var callback *lua.LFunction
		if fn, ok := L.Get(2).(*lua.LFunction); ok {
			callback = fn
		} else {
			if argsTbl, ok := L.Get(2).(*lua.LTable); ok {
				argsTbl.ForEach(func(_, v lua.LValue) {
					args = append(args, v.String())
				})
			}
			callback = L.CheckFunction(3)
		}

		go func() {
			stdout, stderr, exitCode, err := p.System.Exec(binary, args)
			resultFn := func() {
				if p.State == nil {
					return
				}
				tbl := p.State.NewTable()
				if err != nil {
					p.State.SetField(tbl, "stdout", lua.LString(""))
					p.State.SetField(tbl, "stderr", lua.LString(err.Error()))
					p.State.SetField(tbl, "exit_code", lua.LNumber(-1))
				} else {
					p.State.SetField(tbl, "stdout", lua.LString(stdout))
					p.State.SetField(tbl, "stderr", lua.LString(stderr))
					p.State.SetField(tbl, "exit_code", lua.LNumber(exitCode))
				}
				if callErr := p.CallLuaFunc(callback, tbl); callErr != nil {
					p.logError("async exec callback", callErr)
				}
			}
			p.SafePostAsync(&PluginAsyncResult{Plugin: p, Callback: resultFn})
		}()

		return 0
	}
}

func sysEnv(p *Plugin) lua.LGFunction {
	return func(L *lua.LState) int {
		if p.System == nil {
			L.Push(lua.LString(""))
			return 1
		}
		name := L.CheckString(1)
		L.Push(lua.LString(p.System.Env(name)))
		return 1
	}
}
