local relay = require("relay")

-- print(relay.version)

local tree1 = 'https://raw.githubusercontent.com/asafschers/goscore/master/fixtures/tree.pmml'
-- local tree1 = 'file://d:/Workspace/Go/src/github.com/kelindar/relay/src/tree.pmml'

local result = relay.tree(tree1, {
    f1 = "f1v3",
    f2 = "f2v1",
    f4 = 0.08
})

return result