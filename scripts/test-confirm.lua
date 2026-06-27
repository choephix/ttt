local ttt = require("ttt")

ttt.register({
  commands = {
    {
      id = "test.confirmDialog",
      title = "Test: Confirm Dialog",
      handler = function()
        ttt.confirm("Do you want to continue?", function()
          ttt.log("info", "User confirmed: YES")
        end)
      end,
    },
  },
})
