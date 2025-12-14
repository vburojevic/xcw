package embedfiles

import _ "embed"

//go:embed docs/help.json
var HelpJSON []byte

//go:embed schemas/generated.schema.json
var SchemaJSON []byte
