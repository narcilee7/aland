package tribes

import "errors"

// ErrNotIdentity 注册到 Land 的对象必须实现 Identity 接口。
var ErrNotIdentity = errors.New("tribes: adapter must implement Identity")
