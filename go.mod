module github.com/IndependentImpact/ttl2d3

go 1.24

toolchain go1.25.8

require (
	github.com/deiu/gon3 v0.0.0-20241212124032-93153c038193
	github.com/piprate/json-gold v0.8.0
	github.com/spf13/cobra v1.10.2
)

replace github.com/deiu/gon3 => ./third_party/gon3

require (
	github.com/cayleygraph/quad v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pquerna/cachecontrol v0.2.0 // indirect
	github.com/rychipman/easylex v0.0.0-20160129204217-49ee7767142f // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)
