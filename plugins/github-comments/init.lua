local ttt = require("ttt")
local sys = require("ttt.system")
local json = require("ttt.json")

-- State
local pr_number = nil
local repo_slug = nil
local pr_comments = {}
local threads = {}
local loading = false
local error_msg = nil
local selected_thread_idx = nil
local last_panel = nil

-- Detect repo slug
local function detect_repo()
  local result = sys.exec("gh", { "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner" })
  if result.exit_code == 0 then
    repo_slug = result.stdout:match("^(.-)%s*$")
  end
end

-- Parse PR number from a URL or plain number string
local function parse_pr_input(input)
  if not input or #input == 0 then return nil end
  input = input:match("^%s*(.-)%s*$")
  -- Plain number
  local num = input:match("^(%d+)$")
  if num then return tonumber(num) end
  -- GitHub URL: https://github.com/owner/repo/pull/123
  num = input:match("/pull/(%d+)")
  if num then return tonumber(num) end
  return nil
end

-- Extract repo slug from a GitHub PR URL
local function parse_repo_from_url(input)
  if not input then return nil end
  local owner, repo = input:match("github%.com/([^/]+)/([^/]+)/pull/")
  if owner and repo then
    return owner .. "/" .. repo
  end
  return nil
end

-- Detect current PR from branch
local function detect_pr()
  local result = sys.exec("gh", { "pr", "view", "--json", "number", "-q", ".number" })
  if result.exit_code == 0 then
    local num = result.stdout:match("(%d+)")
    if num then
      pr_number = tonumber(num)
    end
  end
end

-- Resolve/unresolve a thread
local function resolve_thread(thread, panel)
  if not thread or type(thread.id) == "string" then return end
  local mutation = string.format([[
mutation {
  resolveReviewThread(input: {threadId: "%s"}) {
    thread { isResolved }
  }
}]], thread.id)
  sys.exec_async("gh", {
    "api", "graphql", "-f", "query=" .. mutation,
  }, function(result)
    if result.exit_code == 0 then
      thread.resolved = true
      ttt.log("Thread resolved")
      if panel then panel:redraw() end
    else
      ttt.log("error", "Failed to resolve: " .. (result.stderr or ""))
    end
  end)
end

local function unresolve_thread(thread, panel)
  if not thread or type(thread.id) == "string" then return end
  local mutation = string.format([[
mutation {
  unresolveReviewThread(input: {threadId: "%s"}) {
    thread { isResolved }
  }
}]], thread.id)
  sys.exec_async("gh", {
    "api", "graphql", "-f", "query=" .. mutation,
  }, function(result)
    if result.exit_code == 0 then
      thread.resolved = false
      ttt.log("Thread unresolved")
      if panel then panel:redraw() end
    else
      ttt.log("error", "Failed to unresolve: " .. (result.stderr or ""))
    end
  end)
end

local function parse_issue_comments(data)
  local result = {}
  if type(data) ~= "table" then return result end
  for _, item in ipairs(data) do
    if type(item) == "table" then
      table.insert(result, {
        id = item.id,
        body = item.body or "",
        user = (type(item.user) == "table" and item.user.login) or "unknown",
        created_at = item.created_at or "",
      })
    end
  end
  return result
end

local function short_date(iso)
  if not iso or #iso < 10 then return "" end
  return iso:sub(1, 10)
end

-- Build GraphQL query for review threads + issue comments
local function build_graphql_query()
  local owner, repo = repo_slug:match("^([^/]+)/(.+)$")
  if not owner then return nil end
  return string.format([[
query {
  repository(owner: "%s", name: "%s") {
    pullRequest(number: %d) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 50) {
            nodes {
              id
              body
              path
              line
              author { login }
              createdAt
            }
          }
        }
      }
    }
  }
}]], owner, repo, pr_number)
end

-- Parse GraphQL response into threads
local function parse_graphql_threads(data)
  threads = {}
  if not data or not data.data then return end
  local pr = data.data.repository and data.data.repository.pullRequest
  if not pr then return end

  local review_threads = pr.reviewThreads and pr.reviewThreads.nodes
  if not review_threads then return end

  for _, rt in ipairs(review_threads) do
    local nodes = rt.comments and rt.comments.nodes
    if nodes and #nodes > 0 then
      local first = nodes[1]
      local thread = {
        id = rt.id,
        path = first.path or "",
        line = first.line or 0,
        resolved = rt.isResolved or false,
        messages = {},
      }
      for _, c in ipairs(nodes) do
        table.insert(thread.messages, {
          id = c.id,
          author = (c.author and c.author.login) or "unknown",
          body = c.body or "",
          created_at = c.createdAt or "",
        })
      end
      table.insert(threads, thread)
    end
  end

  -- Sort by path then line
  table.sort(threads, function(a, b)
    if a.path == b.path then
      return (a.line or 0) < (b.line or 0)
    end
    return a.path < b.path
  end)
end

-- Fetch comments via GraphQL + REST for issue comments
local function fetch_comments(panel)
  if not repo_slug or not pr_number then
    error_msg = "No PR detected on current branch"
    loading = false
    if panel then panel:redraw() end
    return
  end

  loading = true
  error_msg = nil
  if panel then panel:redraw() end

  local pending = 2

  local function check_done()
    pending = pending - 1
    if pending <= 0 then
      loading = false
      -- Prepend general discussion thread if there are issue comments
      if #pr_comments > 0 then
        local general = {
          id = "general",
          path = "",
          line = 0,
          resolved = false,
          messages = {},
        }
        for _, c in ipairs(pr_comments) do
          table.insert(general.messages, {
            id = c.id,
            author = c.user,
            body = c.body,
            created_at = c.created_at,
          })
        end
        table.insert(threads, 1, general)
      end
      if panel then panel:redraw() end
    end
  end

  -- Fetch review threads via GraphQL
  local query = build_graphql_query()
  if not query then
    error_msg = "Invalid repo slug"
    loading = false
    if panel then panel:redraw() end
    return
  end

  sys.exec_async("gh", {
    "api", "graphql", "-f", "query=" .. query,
  }, function(result)
    if result.exit_code == 0 and #result.stdout > 0 then
      local data = json.decode(result.stdout)
      if data then
        parse_graphql_threads(data)
      end
    else
      ttt.log("error", "Failed to fetch review threads: " .. (result.stderr or ""))
    end
    check_done()
  end)

  -- Fetch issue comments via REST
  sys.exec_async("gh", {
    "api", "repos/" .. repo_slug .. "/issues/" .. pr_number .. "/comments",
    "--paginate",
  }, function(result)
    if result.exit_code == 0 and #result.stdout > 0 then
      local data = json.decode(result.stdout)
      if data then
        pr_comments = parse_issue_comments(data)
      end
    end
    check_done()
  end)
end

-- Reply to a thread
local function reply_to_thread(thread, body, panel)
  if not repo_slug or not pr_number or not body or #body == 0 then return end

  if type(thread.id) == "string" then
    sys.exec_async("gh", {
      "api", "repos/" .. repo_slug .. "/issues/" .. pr_number .. "/comments",
      "-f", "body=" .. body, "-X", "POST",
    }, function(result)
      if result.exit_code == 0 then
        ttt.log("Reply posted")
        fetch_comments(panel)
      else
        ttt.log("error", "Failed to post reply: " .. (result.stderr or ""))
      end
    end)
  else
    sys.exec_async("gh", {
      "api", "repos/" .. repo_slug .. "/pulls/" .. pr_number .. "/comments/" .. thread.id .. "/replies",
      "-f", "body=" .. body, "-X", "POST",
    }, function(result)
      if result.exit_code == 0 then
        ttt.log("Reply posted")
        fetch_comments(panel)
      else
        ttt.log("error", "Failed to post reply: " .. (result.stderr or ""))
      end
    end)
  end
end

-- Build list items
local function thread_items()
  local items = {}
  for i, thread in ipairs(threads) do
    local label
    local badge = ""
    local icon = thread.resolved and "●" or "○"

    if thread.path == "" and type(thread.id) == "string" then
      label = "General discussion"
      icon = "≡"
    else
      local filename = thread.path:match("([^/]+)$") or thread.path
      if thread.line > 0 then
        label = filename .. ":" .. thread.line
      else
        label = filename
      end
    end

    if #thread.messages > 1 then
      badge = tostring(#thread.messages)
    end

    table.insert(items, {
      id = tostring(i),
      label = label,
      badge = badge,
      icon = icon,
      muted = thread.resolved,
    })
  end
  return items
end

-- Render the drawer content
local function render_drawer(p)
  last_panel = p

  if loading then
    p:label({ text = "Loading comments...", style = "muted", padding_left = 1, padding_top = 1 })
    return
  end

  if error_msg then
    p:label({ text = error_msg, style = "danger", padding_left = 1, padding_top = 1 })
    return
  end

  if not pr_number then
    p:label({ text = "No PR on current branch", style = "muted", padding_left = 1, padding_top = 1 })
    p:label({ text = "Enter a PR number or URL:", style = "muted", padding_left = 1, padding_top = 1 })
    p:hstack({
      height = 1,
      render = function(h)
        h:input({
          placeholder = "218 or https://github.com/.../pull/218",
          prefix = "PR: ",
          clear_on_submit = true,
          on_submit = function(text)
            local num = parse_pr_input(text)
            if num then
              local url_repo = parse_repo_from_url(text)
              if url_repo then
                repo_slug = url_repo
              end
              pr_number = num
              fetch_comments(p)
            end
          end,
        })
      end,
    })
    return
  end

  -- Thread detail view
  if selected_thread_idx then
    local thread = threads[selected_thread_idx]
    if not thread then
      selected_thread_idx = nil
      p:redraw()
      return
    end

    -- Back button + resolve toggle
    p:hstack({
      height = 1,
      render = function(h)
        h:button({
          label = "← Back",
          on_click = function()
            selected_thread_idx = nil
            p:redraw()
          end,
        })
        if type(thread.id) ~= "string" then
          if thread.resolved then
            h:button({
              label = "Unresolve",
              on_click = function() unresolve_thread(thread, p) end,
            })
          else
            h:button({
              label = "✓ Resolve",
              on_click = function() resolve_thread(thread, p) end,
            })
          end
        end
      end,
    })

    -- Thread header
    if thread.path ~= "" then
      p:label({
        text = thread.path,
        style = "bold",
        padding_left = 1,
      })
      if thread.line > 0 then
        p:label({
          text = "Line " .. thread.line,
          style = "muted",
          padding_left = 1,
        })
      end
    else
      p:label({
        text = "General Discussion",
        style = "bold",
        padding_left = 1,
      })
    end

    p:divider()

    -- Messages
    for _, msg in ipairs(thread.messages) do
      local header = "@" .. msg.author
      local date = short_date(msg.created_at)
      if #date > 0 then
        header = header .. "  " .. date
      end
      p:label({
        text = header,
        style = "syntax_function",
        padding_left = 1,
        padding_top = 1,
      })
      p:markdown({
        text = msg.body,
        padding_left = 1,
        padding_right = 1,
      })
      p:divider()
    end

    -- Reply input
    p:hstack({
      height = 1,
      render = function(h)
        h:input({
          placeholder = "Reply...",
          prefix = "> ",
          clear_on_submit = true,
          on_submit = function(text)
            if #text > 0 then
              reply_to_thread(thread, text, p)
            end
          end,
        })
      end,
    })

    return
  end

  -- Inbox list view
  p:title({
    text = "PR #" .. pr_number,
    style = "bold",
    badge = tostring(#threads) .. " threads",
    padding_left = 1,
    padded = true,
    menu = {
      { label = "Go to PR", command = "goto_pr" },
      { label = "Refresh", command = "refresh" },
      { separator = true },
      { label = "Close", command = "close" },
    },
    on_menu = function(command)
      if command == "goto_pr" then
        sys.exec("xdg-open", { "https://github.com/" .. repo_slug .. "/pull/" .. pr_number })
      elseif command == "refresh" then
        fetch_comments(p)
      elseif command == "close" then
        ttt.close_drawer()
      end
    end,
  })

  p:divider()

  if #threads == 0 then
    p:label({ text = "No comments found.", style = "muted", padding_left = 1 })
    return
  end

  p:list({
    padding_left = 1,
    items = thread_items(),
    on_select = function(node)
      local idx = tonumber(node.id)
      if not idx then return end
      local thread = threads[idx]
      if thread and thread.path ~= "" then
        ttt.open_file(thread.path, thread.line)
      end
    end,
    on_command = function(command, node)
      local idx = tonumber(node.id)
      if not idx then return end
      if command == "open" then
        selected_thread_idx = idx
        p:redraw()
      elseif command == "goto" then
        local thread = threads[idx]
        if thread and thread.path ~= "" then
          ttt.open_file(thread.path, thread.line)
        end
      elseif command == "resolve" then
        local thread = threads[idx]
        if thread then resolve_thread(thread, p) end
      elseif command == "unresolve" then
        local thread = threads[idx]
        if thread then unresolve_thread(thread, p) end
      end
    end,
    node_menu = {
      { label = "Open Thread", command = "open" },
      { label = "Go to File", command = "goto" },
      { separator = true },
      { label = "Resolve", command = "resolve" },
      { label = "Unresolve", command = "unresolve" },
    },
    key_commands = {
      o = "open",
      g = "goto",
      r = "resolve",
      u = "unresolve",
    },
  })
end

-- Open the drawer
local function open_comments_drawer()
  selected_thread_idx = nil

  if not repo_slug then
    detect_repo()
  end

  ttt.open_drawer({
    width = 60,
    min_width = 30,
    side = "right",
    render = render_drawer,
  })

  if pr_number then
    fetch_comments(last_panel)
  end
end

ttt.register({
  commands = {
    { id = "github.openComments", title = "GitHub: PR Comments", handler = open_comments_drawer },
  },
  keybindings = {
    { key = "ctrl+k g", command = "github.openComments" },
  },
})
