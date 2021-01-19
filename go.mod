module github.com/go-vela/pkg-executor

go 1.15

replace github.com/go-vela/pkg-runtime => ../pkg-runtime

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/go-vela/compiler v0.7.0-rc1
	github.com/go-vela/mock v0.7.0-rc1
	github.com/go-vela/pkg-runtime v0.7.0-rc1.0.20210115210301-627230eab176
	github.com/go-vela/sdk-go v0.7.0-rc1
	github.com/go-vela/types v0.7.0-rc1.0.20210115155442-682d1037e16d
	github.com/google/go-cmp v0.5.4
	github.com/joho/godotenv v1.3.0
	github.com/sirupsen/logrus v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
)
