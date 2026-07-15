vim.lsp.config("gopls", {
  settings = {
    gopls = {
      buildFlags = { "-tags=integration" },
    },
  },
})
