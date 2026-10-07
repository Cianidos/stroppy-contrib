package main

import (
	workload "example.com/select1/workload"
	stroppy "github.com/stroppy-io/stroppy/v6"
)

func main() { stroppy.Main(workload.Test) }
