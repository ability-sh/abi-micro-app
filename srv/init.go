package srv

import (
	"github.com/ability-sh/abi-micro/micro"
)

func init() {
	micro.Reg("uv-app", func(name string, config interface{}) (micro.Service, error) {
		return newAppService(name, config), nil
	})
}
