local ttt = require("ttt")
local fs = require("ttt.fs")
local json = require("ttt.json")
local system = require("ttt.system")

local function sanitize(value)
  return string.gsub(value, "[^%w._-]", "_")
end

local function instance_key()
  local name = system.env("TTT_REMOTE_NAME")
  if name ~= "" then
    return "name_" .. sanitize(name)
  end

  local session = system.env("HERDR_SESSION")
  local pane = system.env("HERDR_PANE_ID")
  if session ~= "" and pane ~= "" then
    return "herdr_" .. sanitize(session) .. "_" .. sanitize(pane)
  end

  return nil
end

local key = nil
local mailbox_dir = ttt.plugin_dir() .. "/mailboxes"
local prefix = nil

local function acknowledge(path)
  local ok, err = fs.write(path, "")
  if not ok then
    ttt.log("error", "Remote Open could not acknowledge command: " .. tostring(err))
  end
end

local function consume(path)
  local content, read_err = fs.read(path)
  if not content then
    ttt.log("error", "Remote Open could not read command: " .. tostring(read_err))
    return
  end
  if content == "" then
    return
  end

  local command, decode_err = json.decode(content)
  if not command then
    ttt.log("error", "Remote Open rejected malformed command: " .. tostring(decode_err))
    acknowledge(path)
    return
  end
  if type(command.files) ~= "table" then
    ttt.log("error", "Remote Open rejected command without files")
    acknowledge(path)
    return
  end

  for _, file in ipairs(command.files) do
    if type(file) == "table" and type(file.path) == "string" and file.path ~= "" then
      local line = tonumber(file.line) or 0
      ttt.open_file(file.path, line)
    else
      ttt.log("error", "Remote Open skipped an invalid file entry")
    end
  end

  acknowledge(path)
end

local function poll()
  if not prefix then
    key = instance_key()
    prefix = key and (key .. "--") or nil
  end
  if not prefix then
    return
  end

  local entries = fs.list(mailbox_dir)
  if not entries then
    return
  end

  local commands = {}
  for _, entry in ipairs(entries) do
    if not entry.is_dir
      and string.sub(entry.name, 1, #prefix) == prefix
      and string.sub(entry.name, -5) == ".json" then
      table.insert(commands, entry.name)
    end
  end
  table.sort(commands)

  for _, name in ipairs(commands) do
    consume(mailbox_dir .. "/" .. name)
  end
end

ttt.set_interval(100, poll)
