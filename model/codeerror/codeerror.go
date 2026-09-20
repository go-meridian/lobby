package codeerror

import ce "github.com/SilentQianyi/codeerror"

// CodeError re-export，保持对外部包的透明
type CodeError = ce.CodeError

// New 创建新的 CodeError
var New = ce.New
